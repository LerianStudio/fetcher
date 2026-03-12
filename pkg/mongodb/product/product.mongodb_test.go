package product

import (
	"context"
	"errors"
	stdhttp "net/http"
	"testing"
	"time"

	pkgErrors "github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/model"
	customhttp "github.com/LerianStudio/fetcher/pkg/net/http"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type stubMongoDatabaseProvider struct {
	clientFunc func(ctx context.Context) (*mongo.Client, error)
}

func (s stubMongoDatabaseProvider) Client(ctx context.Context) (*mongo.Client, error) {
	if s.clientFunc == nil {
		return &mongo.Client{}, nil
	}

	return s.clientFunc(ctx)
}

func TestNewProductMongoDBRepository_NilClient(t *testing.T) {
	t.Parallel()

	repo, err := NewProductMongoDBRepository(nil, "testdb")

	require.Error(t, err)
	assert.EqualError(t, err, "mongo client is required")
	assert.Nil(t, repo)
}

func TestNewProductMongoDBRepository_ConfigAndInitialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		cfg                []RepositoryConfig
		providerErr        error
		wantInitTimeout    time.Duration
		wantDeadlineWindow time.Duration
		wantErr            string
	}{
		{
			name:               "uses default timeout when config omitted",
			wantInitTimeout:    DefaultInitTimeout,
			wantDeadlineWindow: time.Second,
		},
		{
			name:               "normalizes non positive timeout to default",
			cfg:                []RepositoryConfig{{InitTimeout: 0}},
			wantInitTimeout:    DefaultInitTimeout,
			wantDeadlineWindow: time.Second,
		},
		{
			name:               "preserves explicit timeout",
			cfg:                []RepositoryConfig{{InitTimeout: 25 * time.Millisecond}},
			wantInitTimeout:    25 * time.Millisecond,
			wantDeadlineWindow: 25 * time.Millisecond,
		},
		{
			name:        "propagates initialization errors",
			providerErr: errors.New("client unavailable"),
			wantErr:     "client unavailable",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			deadlineSeen := time.Time{}

			repo, err := newProductMongoDBRepository(stubMongoDatabaseProvider{
				clientFunc: func(ctx context.Context) (*mongo.Client, error) {
					if deadline, ok := ctx.Deadline(); ok {
						deadlineSeen = deadline
					}

					if tt.providerErr != nil {
						return nil, tt.providerErr
					}

					return &mongo.Client{}, nil
				},
			}, "catalog", tt.cfg...)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
				assert.Nil(t, repo)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, repo)
			assert.Equal(t, "catalog", repo.Database)
			assert.Equal(t, tt.wantInitTimeout, repo.config.InitTimeout)
			require.False(t, deadlineSeen.IsZero(), "expected constructor to pass a timeout context to the provider")

			gotDeadline := deadlineSeen.Sub(start)
			assert.InDelta(t, tt.wantInitTimeout, gotDeadline, float64(tt.wantDeadlineWindow))
		})
	}
}

func TestProductMongoDBRepository_Create_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		product       any
		providerErr   error
		assertOutcome func(t *testing.T, err error)
	}{
		{
			name:    "rejects nil product",
			product: nil,
			assertOutcome: func(t *testing.T, err error) {
				require.Error(t, err)

				var internalErr pkgErrors.InternalServerError
				require.ErrorAs(t, err, &internalErr)
				assert.Equal(t, "product", internalErr.EntityType)
				require.Error(t, internalErr.Err)
				assert.EqualError(t, internalErr.Err, "product is required")
			},
		},
		{
			name:        "maps provider failure to service unavailable response",
			product:     testProduct(),
			providerErr: context.DeadlineExceeded,
			assertOutcome: func(t *testing.T, err error) {
				require.Error(t, err)

				var responseErr pkgErrors.ResponseErrorWithStatusCode
				require.ErrorAs(t, err, &responseErr)
				assert.Equal(t, stdhttp.StatusServiceUnavailable, responseErr.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &ProductMongoDBRepository{
				connection: stubMongoDatabaseProvider{
					clientFunc: func(ctx context.Context) (*mongo.Client, error) {
						if tt.providerErr != nil {
							return nil, tt.providerErr
						}

						return &mongo.Client{}, nil
					},
				},
				Database: "catalog",
				config: RepositoryConfig{
					InitTimeout: DefaultInitTimeout,
				},
			}

			var productArg *model.Product
			if tt.product != nil {
				productArg = tt.product.(*model.Product)
			}

			_, err := repo.Create(context.Background(), productArg)
			tt.assertOutcome(t, err)
		})
	}
}

func TestProductMongoDBRepository_BuildQueryFilter(t *testing.T) {
	t.Parallel()

	organizationID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	metadata := bson.M{
		"metadata.country": "BR",
		"status":           "active",
	}

	tests := []struct {
		name      string
		filters   customhttp.QueryHeader
		assertion func(t *testing.T, filter bson.M)
	}{
		{
			name: "includes metadata prefixed filters when enabled",
			filters: customhttp.QueryHeader{
				Metadata:    &metadata,
				UseMetadata: true,
			},
			assertion: func(t *testing.T, filter bson.M) {
				assert.Equal(t, organizationID, filter["organization_id"])
				assert.Equal(t, "BR", filter["metadata.country"])
				assert.NotContains(t, filter, "status")
			},
		},
		{
			name: "ignores metadata filters when disabled",
			filters: customhttp.QueryHeader{
				Metadata:    &metadata,
				UseMetadata: false,
			},
			assertion: func(t *testing.T, filter bson.M) {
				assert.Equal(t, organizationID, filter["organization_id"])
				assert.NotContains(t, filter, "metadata.country")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &ProductMongoDBRepository{}
			filter := repo.buildQueryFilter(organizationID, tt.filters)

			require.Contains(t, filter, "deleted_at")
			tt.assertion(t, filter)
		})
	}
}

func testProduct() *model.Product {
	metadata := map[string]any{"region": "BR"}

	return &model.Product{
		ID:             uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		OrganizationID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		Code:           "catalog",
		Name:           "Catalog",
		Description:    "fetcher catalog",
		Metadata:       &metadata,
		CreatedAt:      time.Unix(1700000000, 0).UTC(),
		UpdatedAt:      time.Unix(1700000000, 0).UTC(),
	}
}
