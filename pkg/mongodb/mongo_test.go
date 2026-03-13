package mongodb

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
	"go.uber.org/mock/gomock"
)

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
			name:       "server selection error",
			err:        topology.ServerSelectionError{Wrapped: errors.New("server selection failed")},
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
			name:       "command error shutdown",
			err:        mongo.CommandError{Code: 89},
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
			name:       "command error unhandled code",
			err:        mongo.CommandError{Code: 999},
			wantCode:   constant.ErrInternalServer.Error(),
			wantNotNil: true,
		},
		{
			name:       "connection error (treated as internal)",
			err:        topology.ConnectionError{Wrapped: errors.New("connection error")},
			wantCode:   constant.ErrInternalServer.Error(),
			wantNotNil: true,
		},
		{
			name:       "timeout error wrapped",
			err:        fmt.Errorf("timeout: %w", driver.ErrDeadlineWouldBeExceeded),
			wantCode:   constant.ErrServiceUnavailable.Error(),
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

	t.Run("returns error when client retrieval fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		provider := NewMockMongoClientProvider(ctrl)
		provider.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("connection failed"))

		err := PingMongo(context.Background(), provider, time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get database connection")
	})

	t.Run("uses default timeout when timeout is zero", func(t *testing.T) {
		// We can't easily test this without a real connection, but we can verify
		// the code path by checking it doesn't panic with zero timeout
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		provider := NewMockMongoClientProvider(ctrl)
		provider.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("connection failed"))

		err := PingMongo(context.Background(), provider, 0)
		assert.Error(t, err) // Will fail at client retrieval, but timeout code path is exercised
	})

	t.Run("uses default timeout when timeout is negative", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		provider := NewMockMongoClientProvider(ctrl)
		provider.EXPECT().
			Client(gomock.Any()).
			Return(nil, errors.New("connection failed"))

		err := PingMongo(context.Background(), provider, -1*time.Second)
		assert.Error(t, err)
	})
}

func TestDefaultPingTimeout(t *testing.T) {
	assert.Equal(t, 5*time.Second, DefaultPingTimeout)
}

func TestValidateFieldsInSchemaMongo_EdgeCases(t *testing.T) {
	t.Run("handles mixed case field names correctly", func(t *testing.T) {
		var count int32
		expectedFields := []string{"UserID", "userName", "EMAIL"}
		schema := CollectionSchema{
			Fields: []FieldInformation{
				{Name: "userid"},
				{Name: "USERNAME"},
				{Name: "email"},
			},
		}

		missing := ValidateFieldsInSchemaMongo(expectedFields, schema, &count)

		assert.Empty(t, missing, "all fields should match case-insensitively")
		assert.Equal(t, int32(3), count)
	})

	t.Run("handles special characters in field names", func(t *testing.T) {
		var count int32
		expectedFields := []string{"field_with_underscore", "field-with-dash", "field.with.dot"}
		schema := CollectionSchema{
			Fields: []FieldInformation{
				{Name: "field_with_underscore"},
				{Name: "field-with-dash"},
				{Name: "field.with.dot"},
			},
		}

		missing := ValidateFieldsInSchemaMongo(expectedFields, schema, &count)

		assert.Empty(t, missing)
		assert.Equal(t, int32(3), count)
	})

	t.Run("handles duplicate field names in expected fields", func(t *testing.T) {
		var count int32
		expectedFields := []string{"id", "id", "name", "name"}
		schema := CollectionSchema{
			Fields: []FieldInformation{
				{Name: "id"},
				{Name: "name"},
			},
		}

		missing := ValidateFieldsInSchemaMongo(expectedFields, schema, &count)

		assert.Empty(t, missing)
		assert.Equal(t, int32(4), count, "should count each occurrence")
	})

	t.Run("handles empty strings in field names", func(t *testing.T) {
		var count int32
		expectedFields := []string{"", "id"}
		schema := CollectionSchema{
			Fields: []FieldInformation{
				{Name: ""},
				{Name: "id"},
			},
		}

		missing := ValidateFieldsInSchemaMongo(expectedFields, schema, &count)

		assert.Empty(t, missing)
		assert.Equal(t, int32(2), count)
	})

	t.Run("partial match scenario", func(t *testing.T) {
		var count int32
		expectedFields := []string{"id", "name", "email", "phone", "address"}
		schema := CollectionSchema{
			Fields: []FieldInformation{
				{Name: "id"},
				{Name: "email"},
			},
		}

		missing := ValidateFieldsInSchemaMongo(expectedFields, schema, &count)

		assert.Equal(t, []string{"name", "phone", "address"}, missing)
		assert.Equal(t, int32(2), count)
	})
}

func TestCollectionSchema_Structure(t *testing.T) {
	t.Run("creates valid collection schema", func(t *testing.T) {
		schema := CollectionSchema{
			CollectionName: "users",
			Fields: []FieldInformation{
				{Name: "id", DataType: "objectId"},
				{Name: "name", DataType: "string"},
				{Name: "age", DataType: "number"},
				{Name: "created_at", DataType: "date"},
			},
		}

		assert.Equal(t, "users", schema.CollectionName)
		assert.Len(t, schema.Fields, 4)
		assert.Equal(t, "objectId", schema.Fields[0].DataType)
		assert.Equal(t, "string", schema.Fields[1].DataType)
		assert.Equal(t, "number", schema.Fields[2].DataType)
		assert.Equal(t, "date", schema.Fields[3].DataType)
	})

	t.Run("handles empty collection schema", func(t *testing.T) {
		schema := CollectionSchema{}

		assert.Empty(t, schema.CollectionName)
		assert.Nil(t, schema.Fields)
	})

	t.Run("handles schema with no fields", func(t *testing.T) {
		schema := CollectionSchema{
			CollectionName: "empty_collection",
			Fields:         []FieldInformation{},
		}

		assert.Equal(t, "empty_collection", schema.CollectionName)
		assert.Empty(t, schema.Fields)
	})
}

func TestFieldInformation_Structure(t *testing.T) {
	t.Run("creates valid field information", func(t *testing.T) {
		field := FieldInformation{
			Name:     "user_id",
			DataType: "objectId",
		}

		assert.Equal(t, "user_id", field.Name)
		assert.Equal(t, "objectId", field.DataType)
	})

	t.Run("handles empty field information", func(t *testing.T) {
		field := FieldInformation{}

		assert.Empty(t, field.Name)
		assert.Empty(t, field.DataType)
	})
}
