package product

import (
	"context"
	"errors"
	"log"
	nethttp "net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	http "github.com/LerianStudio/fetcher/pkg/net/http"

	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	libMongo "github.com/LerianStudio/lib-commons/v2/commons/mongo"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tryvium-travels/memongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

var (
	productTestMongoServer *memongo.Server
	productTestMongoConn   *libMongo.MongoConnection
)

const productTestDatabaseName = "fetcher_product_test"

func TestMain(m *testing.M) {
	server, err := memongo.Start("6.0.6")
	if err != nil {
		// memongo doesn't support all platforms (e.g., Fedora 42)
		// Skip tests gracefully instead of failing
		log.Printf("SKIP: memongo not available on this platform: %v", err)
		os.Exit(0)
	}
	productTestMongoServer = server
	productTestMongoConn = &libMongo.MongoConnection{
		ConnectionStringSource: server.URI(),
		Database:               productTestDatabaseName,
		Logger:                 &libLog.GoLogger{Level: libLog.ErrorLevel},
		MaxPoolSize:            5,
	}

	code := m.Run()

	server.Stop()
	os.Exit(code)
}

func newProductRepository(t *testing.T) *ProductMongoDBRepository {
	t.Helper()
	clearProductsCollection(t)
	repo, err := NewProductMongoDBRepository(context.Background(), productTestMongoConn)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("failed to ensure indexes: %v", err)
	}
	return repo
}

func clearProductsCollection(t *testing.T) {
	t.Helper()
	if productTestMongoConn == nil {
		t.Fatalf("mongo connection not initialized")
	}
	client, err := productTestMongoConn.GetDB(context.Background())
	if err != nil {
		t.Fatalf("failed to get db: %v", err)
	}
	coll := client.Database(strings.ToLower(productTestMongoConn.Database)).Collection(strings.ToLower(constant.MongoCollectionProduct))
	if err := coll.Drop(context.Background()); err != nil {
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Code == 26 {
			return
		}
		t.Fatalf("failed to drop collection: %v", err)
	}
}

func productFixture() *model.Product {
	now := time.Now().UTC()
	return &model.Product{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Code:           "test-product",
		Name:           "Test Product",
		Description:    "A test product for unit tests",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func productWithMetadataFixture() *model.Product {
	p := productFixture()
	meta := map[string]any{
		"env":     "testing",
		"version": "1.0",
	}
	p.Metadata = &meta
	return p
}

func createProduct(t *testing.T, repo *ProductMongoDBRepository, p *model.Product) *model.Product {
	t.Helper()
	created, err := repo.Create(context.Background(), p)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	return created
}

func stubProductSpanAttributes(t *testing.T, retErr error) {
	t.Helper()
	original := setSpanAttributesFromStruct
	setSpanAttributesFromStruct = func(span *trace.Span, key string, valueStruct any) error {
		return retErr
	}
	t.Cleanup(func() {
		setSpanAttributesFromStruct = original
	})
}

func TestProductMongoDBRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newProductRepository(t)
		p := productFixture()
		created, err := repo.Create(context.Background(), p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.ID != p.ID || created.Code != p.Code {
			t.Fatalf("expected returned product to match input")
		}
		if created.Name != p.Name {
			t.Fatalf("expected name %s, got %s", p.Name, created.Name)
		}
	})

	t.Run("success with metadata", func(t *testing.T) {
		repo := newProductRepository(t)
		p := productWithMetadataFixture()
		created, err := repo.Create(context.Background(), p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.Metadata == nil {
			t.Fatalf("expected metadata to be preserved")
		}
		meta := *created.Metadata
		if meta["env"] != "testing" {
			t.Fatalf("expected metadata env=testing, got %v", meta["env"])
		}
	})

	t.Run("nil payload returns error", func(t *testing.T) {
		repo := newProductRepository(t)
		if _, err := repo.Create(context.Background(), nil); err == nil {
			t.Fatalf("expected error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
		}
	})

	t.Run("duplicate code returns conflict", func(t *testing.T) {
		repo := newProductRepository(t)
		base := createProduct(t, repo, productFixture())

		dup := productFixture()
		dup.OrganizationID = base.OrganizationID
		dup.Code = base.Code

		if _, err := repo.Create(context.Background(), dup); err == nil {
			t.Fatalf("expected conflict error")
		} else {
			var respErr pkg.ResponseErrorWithStatusCode
			if !errors.As(err, &respErr) {
				t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
			}
			assert.Equal(t, nethttp.StatusConflict, respErr.StatusCode)
		}
	})

	t.Run("same code different organization succeeds", func(t *testing.T) {
		repo := newProductRepository(t)
		createProduct(t, repo, productFixture())

		other := productFixture()
		other.OrganizationID = uuid.New()
		// Same code, different org
		other.Code = "test-product"
		created, err := repo.Create(context.Background(), other)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.Code != "test-product" {
			t.Fatalf("expected code preserved")
		}
	})

	t.Run("span attribute errors are ignored", func(t *testing.T) {
		repo := newProductRepository(t)
		stubProductSpanAttributes(t, errors.New("span failure"))
		if _, err := repo.Create(context.Background(), productFixture()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			GetDB(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &ProductMongoDBRepository{
			connection: mockConn,
			Database:   productTestDatabaseName,
		}
		if _, err := repo.Create(context.Background(), productFixture()); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %T: %v", err, err)
			}
			assert.Equal(t, constant.ErrInternalServer.Error(), internal.Code)
		}
	})
}

func TestProductMongoDBRepository_FindByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productFixture())
		found, err := repo.FindByID(context.Background(), created.ID, created.OrganizationID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected matching id")
		}
		if found.Code != created.Code {
			t.Fatalf("expected matching code")
		}
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := newProductRepository(t)
		found, err := repo.FindByID(context.Background(), uuid.New(), uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil when product not found")
		}
	})

	t.Run("does not find soft-deleted product", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productFixture())

		deletedAt := time.Now().UTC()
		if err := repo.Delete(context.Background(), created.ID, created.OrganizationID, deletedAt); err != nil {
			t.Fatalf("failed to delete: %v", err)
		}

		found, err := repo.FindByID(context.Background(), created.ID, created.OrganizationID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil for soft-deleted product")
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			GetDB(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &ProductMongoDBRepository{
			connection: mockConn,
			Database:   productTestDatabaseName,
		}
		if _, err := repo.FindByID(context.Background(), uuid.New(), uuid.New()); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %T: %v", err, err)
			}
			assert.Equal(t, constant.ErrInternalServer.Error(), internal.Code)
		}
	})
}

func TestProductMongoDBRepository_FindByCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productFixture())
		found, err := repo.FindByCode(context.Background(), created.Code, created.OrganizationID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.ID != created.ID {
			t.Fatalf("expected matching id")
		}
		if found.Code != created.Code {
			t.Fatalf("expected matching code")
		}
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		repo := newProductRepository(t)
		found, err := repo.FindByCode(context.Background(), "nonexistent-code", uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil when product not found by code")
		}
	})

	t.Run("does not find soft-deleted product", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productFixture())

		deletedAt := time.Now().UTC()
		if err := repo.Delete(context.Background(), created.ID, created.OrganizationID, deletedAt); err != nil {
			t.Fatalf("failed to delete: %v", err)
		}

		found, err := repo.FindByCode(context.Background(), created.Code, created.OrganizationID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil for soft-deleted product")
		}
	})

	t.Run("respects organization scope", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productFixture())

		// Search with a different organization
		found, err := repo.FindByCode(context.Background(), created.Code, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil for different organization")
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			GetDB(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &ProductMongoDBRepository{
			connection: mockConn,
			Database:   productTestDatabaseName,
		}
		if _, err := repo.FindByCode(context.Background(), "code", uuid.New()); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %T: %v", err, err)
			}
			assert.Equal(t, constant.ErrInternalServer.Error(), internal.Code)
		}
	})
}

func TestProductMongoDBRepository_List(t *testing.T) {
	t.Run("returns results for organization", func(t *testing.T) {
		repo := newProductRepository(t)
		org := uuid.New()

		p1 := productFixture()
		p1.OrganizationID = org
		p1.Code = "product-alpha"
		p1.Name = "Alpha"
		createProduct(t, repo, p1)

		p2 := productFixture()
		p2.OrganizationID = org
		p2.Code = "product-beta"
		p2.Name = "Beta"
		createProduct(t, repo, p2)

		// Different org product should not appear
		otherOrg := productFixture()
		otherOrg.OrganizationID = uuid.New()
		otherOrg.Code = "product-gamma"
		createProduct(t, repo, otherOrg)

		filters := http.QueryHeader{Limit: 10, Page: 1}
		list, total, err := repo.List(context.Background(), org, filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 products, got %d", len(list))
		}
		if total != 2 {
			t.Fatalf("expected total count 2, got %d", total)
		}
	})

	t.Run("returns empty list when no products exist", func(t *testing.T) {
		repo := newProductRepository(t)
		filters := http.QueryHeader{Limit: 10, Page: 1}
		list, total, err := repo.List(context.Background(), uuid.New(), filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("expected empty list, got %d", len(list))
		}
		if total != 0 {
			t.Fatalf("expected total count 0, got %d", total)
		}
	})

	t.Run("respects pagination", func(t *testing.T) {
		repo := newProductRepository(t)
		org := uuid.New()

		for i := range 3 {
			p := productFixture()
			p.OrganizationID = org
			p.Code = "paginated-" + uuid.New().String()[:8]
			p.CreatedAt = time.Now().UTC().Add(time.Duration(i) * time.Hour)
			p.UpdatedAt = p.CreatedAt
			createProduct(t, repo, p)
		}

		filters := http.QueryHeader{Limit: 2, Page: 1}
		list, total, err := repo.List(context.Background(), org, filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 products on page 1, got %d", len(list))
		}
		if total != 3 {
			t.Fatalf("expected total count 3, got %d", total)
		}

		// Page 2
		filters.Page = 2
		list, total, err = repo.List(context.Background(), org, filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 product on page 2, got %d", len(list))
		}
		if total != 3 {
			t.Fatalf("expected total count 3, got %d", total)
		}
	})

	t.Run("excludes soft-deleted products", func(t *testing.T) {
		repo := newProductRepository(t)
		org := uuid.New()

		p := productFixture()
		p.OrganizationID = org
		p.Code = "to-be-deleted"
		created := createProduct(t, repo, p)

		if err := repo.Delete(context.Background(), created.ID, org, time.Now().UTC()); err != nil {
			t.Fatalf("failed to delete: %v", err)
		}

		filters := http.QueryHeader{Limit: 10, Page: 1}
		list, total, err := repo.List(context.Background(), org, filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("expected deleted product excluded, got %d", len(list))
		}
		if total != 0 {
			t.Fatalf("expected total count 0, got %d", total)
		}
	})

	t.Run("filters by created_at range", func(t *testing.T) {
		repo := newProductRepository(t)
		org := uuid.New()

		outOfRange := productFixture()
		outOfRange.OrganizationID = org
		outOfRange.Code = "too-old"
		outOfRange.CreatedAt = time.Now().UTC().Add(-48 * time.Hour)
		outOfRange.UpdatedAt = outOfRange.CreatedAt
		createProduct(t, repo, outOfRange)

		inRange := productFixture()
		inRange.OrganizationID = org
		inRange.Code = "in-range"
		inRange.CreatedAt = time.Now().UTC().Add(-1 * time.Hour)
		inRange.UpdatedAt = inRange.CreatedAt
		createProduct(t, repo, inRange)

		start := inRange.CreatedAt.Add(-30 * time.Minute)
		end := inRange.CreatedAt.Add(30 * time.Minute)
		filters := http.QueryHeader{
			Limit:     5,
			Page:      1,
			StartDate: start,
			EndDate:   end,
		}

		list, _, err := repo.List(context.Background(), org, filters)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 || list[0].Code != "in-range" {
			t.Fatalf("expected only in-range product")
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			GetDB(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &ProductMongoDBRepository{
			connection: mockConn,
			Database:   productTestDatabaseName,
		}
		if _, _, err := repo.List(context.Background(), uuid.New(), http.QueryHeader{}); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %T: %v", err, err)
			}
			assert.Equal(t, constant.ErrInternalServer.Error(), internal.Code)
		}
	})
}

func TestProductMongoDBRepository_Update(t *testing.T) {
	t.Run("updates mutable fields", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productFixture())

		created.Name = "Updated Name"
		created.Description = "Updated description"
		meta := map[string]any{"key": "value"}
		created.Metadata = &meta
		created.UpdatedAt = time.Now().UTC()

		updated, err := repo.Update(context.Background(), created)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != "Updated Name" {
			t.Fatalf("expected name updated, got %s", updated.Name)
		}
		if updated.Description != "Updated description" {
			t.Fatalf("expected description updated, got %s", updated.Description)
		}
		if updated.Metadata == nil {
			t.Fatalf("expected metadata set")
		}
	})

	t.Run("clears metadata when nil", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productWithMetadataFixture())

		created.Metadata = nil
		created.UpdatedAt = time.Now().UTC()
		updated, err := repo.Update(context.Background(), created)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Metadata != nil {
			t.Fatalf("expected metadata cleared")
		}
	})

	t.Run("nil payload returns error", func(t *testing.T) {
		repo := newProductRepository(t)
		if _, err := repo.Update(context.Background(), nil); err == nil {
			t.Fatalf("expected error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %v", err)
			}
		}
	})

	t.Run("returns not found for nonexistent product", func(t *testing.T) {
		repo := newProductRepository(t)
		p := productFixture()
		if _, err := repo.Update(context.Background(), p); err == nil {
			t.Fatalf("expected not found error")
		} else {
			var respErr pkg.ResponseErrorWithStatusCode
			if !errors.As(err, &respErr) {
				t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
			}
			assert.Equal(t, nethttp.StatusNotFound, respErr.StatusCode)
		}
	})

	t.Run("span attribute errors are ignored", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productFixture())
		stubProductSpanAttributes(t, errors.New("span failure"))
		created.Name = "another-name"
		created.UpdatedAt = time.Now().UTC()
		if _, err := repo.Update(context.Background(), created); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			GetDB(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &ProductMongoDBRepository{
			connection: mockConn,
			Database:   productTestDatabaseName,
		}
		p := productFixture()
		if _, err := repo.Update(context.Background(), p); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %T: %v", err, err)
			}
			assert.Equal(t, constant.ErrInternalServer.Error(), internal.Code)
		}
	})
}

func TestProductMongoDBRepository_Delete(t *testing.T) {
	t.Run("soft deletes product", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productFixture())
		deletedAt := time.Now().UTC()
		if err := repo.Delete(context.Background(), created.ID, created.OrganizationID, deletedAt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the product is no longer findable via normal queries
		found, err := repo.FindByID(context.Background(), created.ID, created.OrganizationID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != nil {
			t.Fatalf("expected nil after soft delete")
		}
	})

	t.Run("not found returns entity not found", func(t *testing.T) {
		repo := newProductRepository(t)
		if err := repo.Delete(context.Background(), uuid.New(), uuid.New(), time.Now()); err == nil {
			t.Fatalf("expected not found")
		} else {
			var respErr pkg.ResponseErrorWithStatusCode
			if !errors.As(err, &respErr) {
				t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
			}
			assert.Equal(t, nethttp.StatusNotFound, respErr.StatusCode)
		}
	})

	t.Run("double delete returns not found", func(t *testing.T) {
		repo := newProductRepository(t)
		created := createProduct(t, repo, productFixture())
		deletedAt := time.Now().UTC()

		if err := repo.Delete(context.Background(), created.ID, created.OrganizationID, deletedAt); err != nil {
			t.Fatalf("unexpected error on first delete: %v", err)
		}

		// Second delete should return not found since deleted_at is already set
		if err := repo.Delete(context.Background(), created.ID, created.OrganizationID, deletedAt); err == nil {
			t.Fatalf("expected not found on second delete")
		} else {
			var respErr pkg.ResponseErrorWithStatusCode
			if !errors.As(err, &respErr) {
				t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
			}
			assert.Equal(t, nethttp.StatusNotFound, respErr.StatusCode)
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			GetDB(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &ProductMongoDBRepository{
			connection: mockConn,
			Database:   productTestDatabaseName,
		}
		if err := repo.Delete(context.Background(), uuid.New(), uuid.New(), time.Now()); err == nil {
			t.Fatalf("expected db error")
		} else {
			var internal pkg.InternalServerError
			if !errors.As(err, &internal) {
				t.Fatalf("expected internal server error, got %T: %v", err, err)
			}
			assert.Equal(t, constant.ErrInternalServer.Error(), internal.Code)
		}
	})
}

func TestProductMongoDBRepository_EnsureIndexes(t *testing.T) {
	t.Run("creates indexes successfully", func(t *testing.T) {
		repo := newProductRepository(t)
		// newProductRepository already calls EnsureIndexes, but let's call it again
		if err := repo.EnsureIndexes(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		repo := newProductRepository(t)
		// Call multiple times - should not error
		for i := range 3 {
			if err := repo.EnsureIndexes(context.Background()); err != nil {
				t.Fatalf("unexpected error on iteration %d: %v", i, err)
			}
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			GetDB(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &ProductMongoDBRepository{
			connection: mockConn,
			Database:   productTestDatabaseName,
		}
		if err := repo.EnsureIndexes(context.Background()); err == nil {
			t.Fatalf("expected db error")
		}
	})
}

func TestProductMongoDBRepository_DropIndexes(t *testing.T) {
	t.Run("drops indexes successfully", func(t *testing.T) {
		repo := newProductRepository(t)
		if err := repo.EnsureIndexes(context.Background()); err != nil {
			t.Fatalf("failed to create indexes: %v", err)
		}
		if err := repo.DropIndexes(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("database error surfaces", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := mongodb.NewMockMongoClientProvider(ctrl)
		mockConn.EXPECT().
			GetDB(gomock.Any()).
			Return(nil, errors.New("db down"))

		repo := &ProductMongoDBRepository{
			connection: mockConn,
			Database:   productTestDatabaseName,
		}
		if err := repo.DropIndexes(context.Background()); err == nil {
			t.Fatalf("expected db error")
		}
	})
}

func TestNewProductMongoDBRepository(t *testing.T) {
	t.Run("creates repository successfully", func(t *testing.T) {
		repo, err := NewProductMongoDBRepository(context.Background(), productTestMongoConn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo == nil {
			t.Fatalf("expected repository instance")
		}
		if repo.Database != productTestDatabaseName {
			t.Fatalf("expected database %s, got %s", productTestDatabaseName, repo.Database)
		}
	})
}
