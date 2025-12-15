package mongodb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

// mockMongoClientProvider is a mock implementation of MongoClientProvider for testing
type mockMongoClientProvider struct {
	client *mongo.Client
	err    error
}

func (m *mockMongoClientProvider) GetDB(ctx context.Context) (*mongo.Client, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.client, nil
}

func TestValidateFieldsInSchemaMongo(t *testing.T) {
	tests := []struct {
		name           string
		expectedFields []string
		schema         CollectionSchema
		wantMissing    []string
		wantCount      int32
	}{
		{
			name:           "all fields exist",
			expectedFields: []string{"id", "name", "email"},
			schema: CollectionSchema{
				Fields: []FieldInformation{
					{Name: "id"},
					{Name: "name"},
					{Name: "email"},
				},
			},
			wantMissing: nil,
			wantCount:   3,
		},
		{
			name:           "some fields missing",
			expectedFields: []string{"id", "name", "phone"},
			schema: CollectionSchema{
				Fields: []FieldInformation{
					{Name: "id"},
					{Name: "name"},
				},
			},
			wantMissing: []string{"phone"},
			wantCount:   2,
		},
		{
			name:           "all fields missing",
			expectedFields: []string{"foo", "bar"},
			schema: CollectionSchema{
				Fields: []FieldInformation{
					{Name: "id"},
					{Name: "name"},
				},
			},
			wantMissing: []string{"foo", "bar"},
			wantCount:   0,
		},
		{
			name:           "case insensitive matching",
			expectedFields: []string{"ID", "NAME", "Email"},
			schema: CollectionSchema{
				Fields: []FieldInformation{
					{Name: "id"},
					{Name: "name"},
					{Name: "email"},
				},
			},
			wantMissing: nil,
			wantCount:   3,
		},
		{
			name:           "empty expected fields",
			expectedFields: []string{},
			schema: CollectionSchema{
				Fields: []FieldInformation{
					{Name: "id"},
				},
			},
			wantMissing: nil,
			wantCount:   0,
		},
		{
			name:           "empty schema fields",
			expectedFields: []string{"id", "name"},
			schema: CollectionSchema{
				Fields: []FieldInformation{},
			},
			wantMissing: []string{"id", "name"},
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int32
			missing := ValidateFieldsInSchemaMongo(tt.expectedFields, tt.schema, &count)

			assert.Equal(t, tt.wantMissing, missing)
			assert.Equal(t, tt.wantCount, count)
		})
	}
}

func TestMapMongoErrorToResponse(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		err        error
		wantCode   string
		wantNotNil bool
	}{
		{
			name:       "context canceled",
			err:        context.Canceled,
			wantCode:   constant.ErrServiceUnavailable.Error(),
			wantNotNil: true,
		},
		{
			name:       "context deadline exceeded",
			err:        context.DeadlineExceeded,
			wantCode:   constant.ErrServiceUnavailable.Error(),
			wantNotNil: true,
		},
		{
			name:       "server selection timeout",
			err:        topology.ErrServerSelectionTimeout,
			wantCode:   constant.ErrServiceUnavailable.Error(),
			wantNotNil: true,
		},
		{
			name:       "no documents",
			err:        mongo.ErrNoDocuments,
			wantCode:   constant.ErrNotFound.Error(),
			wantNotNil: true,
		},
		{
			name:       "duplicate key error",
			err:        mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000}}},
			wantCode:   constant.ErrConflict.Error(),
			wantNotNil: true,
		},
		{
			name:       "duplicate key error code 11001",
			err:        mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11001}}},
			wantCode:   constant.ErrConflict.Error(),
			wantNotNil: true,
		},
		{
			name:       "write exception other error",
			err:        mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 12345}}},
			wantCode:   constant.ErrInternalServer.Error(),
			wantNotNil: true,
		},
		{
			name:       "command error unauthorized",
			err:        mongo.CommandError{Code: 13},
			wantCode:   constant.ErrInternalServer.Error(),
			wantNotNil: true,
		},
		{
			name:       "command error authentication failed",
			err:        mongo.CommandError{Code: 18},
			wantCode:   constant.ErrInternalServer.Error(),
			wantNotNil: true,
		},
		{
			name:       "command error exceeded time limit",
			err:        mongo.CommandError{Code: 50},
			wantCode:   constant.ErrServiceUnavailable.Error(),
			wantNotNil: true,
		},
		{
			name:       "command error host unreachable",
			err:        mongo.CommandError{Code: 6},
			wantCode:   constant.ErrServiceUnavailable.Error(),
			wantNotNil: true,
		},
		{
			name:       "command error host not found",
			err:        mongo.CommandError{Code: 7},
			wantCode:   constant.ErrServiceUnavailable.Error(),
			wantNotNil: true,
		},
		{
			name:       "command error shutdown in progress",
			err:        mongo.CommandError{Code: 91},
			wantCode:   constant.ErrServiceUnavailable.Error(),
			wantNotNil: true,
		},
		{
			name:       "command error failed to parse",
			err:        mongo.CommandError{Code: 9},
			wantCode:   constant.ErrInternalServer.Error(),
			wantNotNil: true,
		},
		{
			name:       "command error namespace not found",
			err:        mongo.CommandError{Code: 26},
			wantCode:   constant.ErrInternalServer.Error(),
			wantNotNil: true,
		},
		{
			name:       "decode error",
			err:        bsoncodec.ValueDecoderError{},
			wantCode:   constant.ErrInternalServer.Error(),
			wantNotNil: true,
		},
		{
			name:       "unknown error",
			err:        errors.New("some unknown error"),
			wantCode:   constant.ErrInternalServer.Error(),
			wantNotNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapMongoErrorToResponse(tt.err, ctx)

			assert.NotNil(t, result)
			// Check that the result error message or code contains the expected code
			// The error can be ResponseErrorWithStatusCode, InternalServerError, or ValidationError
			errMsg := result.Error()
			// For InternalServerError, the error message is the Message field
			// For ResponseErrorWithStatusCode, the error message is the Message field
			// We need to verify the error was mapped correctly by checking the type
			switch e := result.(type) {
			case pkg.ResponseErrorWithStatusCode:
				assert.Equal(t, tt.wantCode, e.Code, "unexpected error code")
			case pkg.InternalServerError:
				assert.Equal(t, tt.wantCode, e.Code, "unexpected error code")
			case pkg.ValidationError:
				assert.Equal(t, tt.wantCode, e.Code, "unexpected error code")
			default:
				t.Errorf("unexpected error type: %T, message: %s", result, errMsg)
			}
		})
	}
}

func TestPingMongo(t *testing.T) {
	t.Run("returns error when provider is nil", func(t *testing.T) {
		err := PingMongo(context.Background(), nil, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider is nil")
	})

	t.Run("returns error when GetDB fails", func(t *testing.T) {
		provider := &mockMongoClientProvider{
			err: errors.New("connection failed"),
		}
		err := PingMongo(context.Background(), provider, time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get database connection")
	})

	t.Run("uses default timeout when timeout is zero", func(t *testing.T) {
		// We can't easily test this without a real connection, but we can verify
		// the code path by checking it doesn't panic with zero timeout
		provider := &mockMongoClientProvider{
			err: errors.New("connection failed"),
		}
		err := PingMongo(context.Background(), provider, 0)
		assert.Error(t, err) // Will fail at GetDB, but timeout code path is exercised
	})

	t.Run("uses default timeout when timeout is negative", func(t *testing.T) {
		provider := &mockMongoClientProvider{
			err: errors.New("connection failed"),
		}
		err := PingMongo(context.Background(), provider, -1*time.Second)
		assert.Error(t, err)
	})
}

func TestDefaultPingTimeout(t *testing.T) {
	assert.Equal(t, 5*time.Second, DefaultPingTimeout)
}
