package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libMongo "github.com/LerianStudio/lib-commons/v2/commons/mongo"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

// Repository defines the domain port for products.
type Repository interface {
	Create(ctx context.Context, product *model.Product) (*model.Product, error)
	Update(ctx context.Context, product *model.Product) (*model.Product, error)
	Delete(ctx context.Context, id, organizationID uuid.UUID, deletedAt time.Time) error
	FindByID(ctx context.Context, id, organizationID uuid.UUID) (*model.Product, error)
	FindByCode(ctx context.Context, code string, organizationID uuid.UUID) (*model.Product, error)
	List(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) ([]*model.Product, error)
}

// mongoDatabaseProvider defines the interface for obtaining a MongoDB client.
type mongoDatabaseProvider interface {
	GetDB(ctx context.Context) (*mongo.Client, error)
}

const (
	// DefaultInitTimeout is the default timeout for repository initialization.
	DefaultInitTimeout = 10 * time.Second
)

// setSpanAttributesFromStruct is a helper to set span attributes from a struct.
var setSpanAttributesFromStruct = libOpentelemetry.SetSpanAttributesFromStruct

// RepositoryConfig holds configuration options for the repository.
type RepositoryConfig struct {
	InitTimeout time.Duration
}

// ProductMongoDBRepository implements Repository backed by MongoDB.
type ProductMongoDBRepository struct {
	connection mongoDatabaseProvider
	Database   string
	config     RepositoryConfig
}

// NewProductMongoDBRepository provisions a repository using the given client.
func NewProductMongoDBRepository(mc *libMongo.MongoConnection, cfg ...RepositoryConfig) (*ProductMongoDBRepository, error) {
	config := RepositoryConfig{
		InitTimeout: DefaultInitTimeout,
	}
	if len(cfg) > 0 {
		config = cfg[0]
		if config.InitTimeout <= 0 {
			config.InitTimeout = DefaultInitTimeout
		}
	}

	repo := &ProductMongoDBRepository{
		connection: mc,
		Database:   mc.Database,
		config:     config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.InitTimeout)
	defer cancel()

	if _, err := repo.connection.GetDB(ctx); err != nil {
		return nil, err
	}

	return repo, nil
}

// Create inserts a new product respecting the unique constraint per organization.
func (pr *ProductMongoDBRepository) Create(ctx context.Context, p *model.Product) (*model.Product, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.create_product")
	defer span.End()

	if p == nil {
		err := errors.New("product is required")
		libOpentelemetry.HandleSpanError(&span, "Product payload is nil", err)

		return nil, pkg.ValidateInternalError(err, "product")
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", p.OrganizationID.String()),
		attribute.String("app.request.product_code", p.Code),
		attribute.String("app.request.product_id", p.ID.String()),
	)

	db, err := pr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	record, err := NewProductMongoDBModelFromDomain(p)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert domain to MongoDB model", err)
		return nil, pkg.ValidateInternalError(err, "product")
	}

	if err := setSpanAttributesFromStruct(&span, "app.request.payload", record.ToMapWithMask()); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to JSON", err)
	}

	coll := db.Database(strings.ToLower(pr.Database)).Collection(strings.ToLower(constant.MongoCollectionProduct))
	if _, err := coll.InsertOne(ctx, record); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err := fmt.Errorf("product with code '%s' already exists for organization '%s'", p.Code, p.OrganizationID.String())
			libOpentelemetry.HandleSpanError(&span, "Duplicate product", err)

			return nil, pkg.ValidateBusinessError(
				constant.ErrEntityConflict,
				"product",
			)
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to insert product", err)

		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	product, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "product")
	}

	return product, nil
}

// Update overwrites mutable fields of an existing product and returns the saved entity.
func (pr *ProductMongoDBRepository) Update(ctx context.Context, p *model.Product) (*model.Product, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.update_product")
	defer span.End()

	if p == nil {
		err := errors.New("product is required")
		libOpentelemetry.HandleSpanError(&span, "Product payload is nil", err)

		return nil, pkg.ValidateInternalError(err, "product")
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.product_id", p.ID.String()),
		attribute.String("app.request.organization_id", p.OrganizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := pr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	coll := db.Database(strings.ToLower(pr.Database)).Collection(strings.ToLower(constant.MongoCollectionProduct))
	filter := bson.M{
		"_id":             p.ID,
		"organization_id": p.OrganizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	update := bson.M{
		"$set": bson.M{
			"name":        p.Name,
			"description": p.Description,
			"metadata":    p.Metadata,
			"updated_at":  p.UpdatedAt,
		},
	}

	if err := setSpanAttributesFromStruct(&span, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filter to JSON", err)
	}

	var record ProductMongoDBModel

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	if err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to update product", err)

		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	product, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "product")
	}

	return product, nil
}

// Delete performs a soft delete by stamping deleted_at and updated_at fields.
func (pr *ProductMongoDBRepository) Delete(ctx context.Context, productID, organizationID uuid.UUID, deletedAt time.Time) error {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.delete_product")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.product_id", productID.String()),
		attribute.String("app.request.organization_id", organizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := pr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return mongodb.MapMongoErrorToResponse(err, ctx)
	}

	coll := db.Database(strings.ToLower(pr.Database)).Collection(strings.ToLower(constant.MongoCollectionProduct))
	filter := bson.M{
		"_id":             productID,
		"organization_id": organizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": deletedAt,
			"updated_at": deletedAt,
		},
	}

	res, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to soft delete product", err)
		return mongodb.MapMongoErrorToResponse(err, ctx)
	}

	if res.MatchedCount == 0 {
		libOpentelemetry.HandleSpanError(&span, "Product not found for delete", mongo.ErrNoDocuments)

		return pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"product",
		)
	}

	return nil
}

// FindByID fetches a product by its ID scoped to an organization.
func (pr *ProductMongoDBRepository) FindByID(ctx context.Context, productID, organizationID uuid.UUID) (*model.Product, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_product_by_id")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.product_id", productID.String()),
		attribute.String("app.request.organization_id", organizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := pr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, pkg.ValidateInternalError(err, "product")
	}

	var record ProductMongoDBModel

	filter := bson.M{
		"_id":             productID,
		"organization_id": organizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	coll := db.Database(strings.ToLower(pr.Database)).Collection(strings.ToLower(constant.MongoCollectionProduct))
	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to find product", err)

		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	product, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "product")
	}

	return product, nil
}

// FindByCode retrieves a product by code scoped to an organization.
func (pr *ProductMongoDBRepository) FindByCode(ctx context.Context, code string, organizationID uuid.UUID) (*model.Product, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_product_by_code")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.product_code", code),
	}
	span.SetAttributes(attributes...)

	db, err := pr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	var record ProductMongoDBModel

	filter := bson.M{
		"organization_id": organizationID,
		"code":            code,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	coll := db.Database(strings.ToLower(pr.Database)).Collection(strings.ToLower(constant.MongoCollectionProduct))
	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to find product by code", err)

		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	product, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
		return nil, pkg.ValidateInternalError(err, "product")
	}

	return product, nil
}

// List returns a paginated set of products for the given organization.
func (pr *ProductMongoDBRepository) List(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) ([]*model.Product, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.list_products")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", filters)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filters to JSON string", err)
	}

	db, err := pr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, pkg.ValidateInternalError(err, "product")
	}

	queryFilter := pr.buildQueryFilter(organizationID, filters)
	opts := mongodb.BuildPaginationOptions(filters)

	err = libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.repository_filter", queryFilter)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filters to JSON string", err)
	}

	coll := db.Database(strings.ToLower(pr.Database)).Collection(strings.ToLower(constant.MongoCollectionProduct))

	cur, err := coll.Find(ctx, queryFilter, &opts)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to list products", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}
	defer cur.Close(ctx)

	if opts.Limit == nil {
		limit := int64(50)
		opts.Limit = &limit
	}

	products := make([]*model.Product, 0, int(*opts.Limit))

	for cur.Next(ctx) {
		var record ProductMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to decode product record", err)
			return nil, mongodb.MapMongoErrorToResponse(err, ctx)
		}

		product, err := record.ToEntity()
		if err != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to convert record to domain", err)
			return nil, pkg.ValidateInternalError(err, "product")
		}

		products = append(products, product)
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to iterate over products", err)
		return nil, mongodb.MapMongoErrorToResponse(err, ctx)
	}

	return products, nil
}

// buildQueryFilter builds the MongoDB query filter from filters.
func (pr *ProductMongoDBRepository) buildQueryFilter(organizationID uuid.UUID, filters http.QueryHeader) bson.M {
	queryFilter := bson.M{
		"organization_id": organizationID,
		"deleted_at":      bson.D{{Key: "$eq", Value: nil}},
	}

	if filters.Metadata != nil && filters.UseMetadata {
		for key, value := range *filters.Metadata {
			queryFilter[key] = value
		}
	}

	mongodb.AddDateRangeFilter(queryFilter, filters)

	return queryFilter
}
