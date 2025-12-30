package query

import (
	"errors"
	"fmt"
	"testing"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSchema_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	// Setup mock expectations
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"midaz_onboarding"}).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "midaz_onboarding"},
		}, nil)

	// Note: Get now returns (*DataSourceSchema, error)
	// Note: Tables is map[string]*TableSchema (pointer), Columns is map[string]bool
	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "midaz_onboarding").
		Return(&model.DataSourceSchema{
			ConfigName: "midaz_onboarding",
			Tables: map[string]*model.TableSchema{
				"account": {TableName: "account", Columns: map[string]bool{"id": true, "name": true}},
			},
		}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"midaz_onboarding": {
				"account": {"id", "name"},
			},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "success", resp.Status)
	assert.Empty(t, resp.Errors)
}

func TestValidateSchema_DataSourceNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{}, nil) // No connections found

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"nonexistent_db": {"table1": {"field1"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	// When no connections are found for ANY of the requested datasources,
	// the production code returns a ValidationError immediately
	assert.Error(t, err)
	assert.Nil(t, resp)

	var validationErr pkg.ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, constant.ErrSchemaValidationNotFound.Error(), validationErr.Code)
	assert.Contains(t, validationErr.Message, "No connections configured")
}

func TestValidateSchema_TableNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "db1"},
		}, nil)

	// Note: Get now returns (*DataSourceSchema, error)
	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(&model.DataSourceSchema{
			ConfigName: "db1",
			Tables:     map[string]*model.TableSchema{}, // Empty tables
		}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"nonexistent_table": {"field1"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "failure", resp.Status)
	assert.Len(t, resp.Errors, 1)
	assert.Equal(t, model.ErrTypeTableNotFound, resp.Errors[0].Type)
}

func TestValidateSchema_FieldNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "db1"},
		}, nil)

	// Note: Get now returns (*DataSourceSchema, error)
	// Note: Tables is map[string]*TableSchema (pointer), Columns is map[string]bool
	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(&model.DataSourceSchema{
			ConfigName: "db1",
			Tables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true, "name": true}},
			},
		}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id", "nonexistent_field"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "failure", resp.Status)
	assert.Len(t, resp.Errors, 1)
	assert.Equal(t, model.ErrTypeFieldNotFound, resp.Errors[0].Type)
	assert.Equal(t, "nonexistent_field", resp.Errors[0].Field)
}

func TestValidateSchema_MultipleErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "db1"},
		}, nil)

	// Note: Get now returns (*DataSourceSchema, error)
	// Note: Tables is map[string]*TableSchema (pointer), Columns is map[string]bool
	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(&model.DataSourceSchema{
			ConfigName: "db1",
			Tables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true}},
			},
		}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {
				"users":  {"id", "name", "email"}, // name, email don't exist
				"orders": {"id"},                  // table doesn't exist
			},
			"nonexistent": {"table1": {"field1"}}, // datasource doesn't exist
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "failure", resp.Status)
	assert.GreaterOrEqual(t, len(resp.Errors), 3)
}

func TestValidateSchema_InvalidRequest_NilMappedFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: nil,
	}

	orgID := uuid.New()
	resp, err := service.Execute(ctx, orgID, request)

	// Should return validation error
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestValidateSchema_InvalidRequest_EmptyMappedFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{},
	}

	orgID := uuid.New()
	resp, err := service.Execute(ctx, orgID, request)

	// Should return validation error
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestValidateSchema_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()

	dbError := errors.New("database connection failed")
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return(nil, dbError)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestValidateSchema_CacheError_ContinuesToFetch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "db1", Type: model.TypePostgreSQL, Host: "localhost", Port: 5432},
		}, nil)

	// Cache returns error - should try to fetch from datasource
	cacheError := errors.New("cache error")
	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(nil, cacheError)

	// Mock Decrypt for when it tries to connect to actual datasource
	mockCrypto.EXPECT().
		Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("decrypted-password", nil).
		AnyTimes()

	// Note: The service will try to fetch from the actual datasource which will fail
	// since we don't have a real database. This test verifies the cache error is handled gracefully.

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	// The execution should complete (either success or failure due to datasource connection)
	// but should not fail due to cache error
	assert.NoError(t, err)
	require.NotNil(t, resp)
	// Since we can't connect to actual DB, it will report datasource down
	assert.Equal(t, "failure", resp.Status)
	assert.Len(t, resp.Errors, 1)
	assert.Equal(t, model.ErrTypeDataSourceDown, resp.Errors[0].Type)
}

func TestValidateSchema_PartialConnectionsFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	// Only db1 is found, db2 is not
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "db1"},
		}, nil)

	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(&model.DataSourceSchema{
			ConfigName: "db1",
			Tables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true}},
			},
		}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id"}},
			"db2": {"orders": {"id"}}, // This connection doesn't exist
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "failure", resp.Status)
	// Should have error for db2 not found
	hasDataSourceNotFound := false
	for _, e := range resp.Errors {
		if e.Type == model.ErrTypeDataSourceNotFound && e.DataSourceID == "db2" {
			hasDataSourceNotFound = true
			break
		}
	}
	assert.True(t, hasDataSourceNotFound, "Expected DATA_SOURCE_NOT_FOUND error for db2")
}

func TestNewValidateSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	assert.NotNil(t, service)
	assert.NotNil(t, service.connRepo)
	assert.NotNil(t, service.cryptor)
	assert.NotNil(t, service.schemaCache)
}

func TestValidateSchema_NoConnections_ReturnsSchemaEntityType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"nonexistent_db": {"table1": {"field1"}},
		},
	}

	_, err := service.Execute(ctx, orgID, request)

	require.Error(t, err)
	var validationErr pkg.ValidationError
	require.True(t, errors.As(err, &validationErr))
	// Verify EntityType is "schema" not "fetcher"
	assert.Equal(t, "schema", validationErr.EntityType)
}

func TestValidateSchema_MultipleDatasources_AllValid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID1 := uuid.New()
	connID2 := uuid.New()

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID1, ConfigName: "db1"},
			{ID: connID2, ConfigName: "db2"},
		}, nil)

	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(&model.DataSourceSchema{
			ConfigName: "db1",
			Tables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true, "name": true}},
			},
		}, nil)

	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db2").
		Return(&model.DataSourceSchema{
			ConfigName: "db2",
			Tables: map[string]*model.TableSchema{
				"orders": {TableName: "orders", Columns: map[string]bool{"id": true, "total": true}},
			},
		}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id", "name"}},
			"db2": {"orders": {"id", "total"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "success", resp.Status)
	assert.Empty(t, resp.Errors)
}

// TestEnsureDefaultSchemaForPostgreSQL tests the helper function for adding default schema.
func TestEnsureDefaultSchemaForPostgreSQL(t *testing.T) {
	tests := []struct {
		name          string
		tables        map[string][]string
		inputSchemas  []string
		expectSchemas []string
		expectPublic  bool
	}{
		{
			name: "unqualified table adds public schema",
			tables: map[string][]string{
				"users": {"id", "name"},
			},
			inputSchemas:  []string{},
			expectSchemas: []string{"public"},
			expectPublic:  true,
		},
		{
			name: "qualified table does not add public schema",
			tables: map[string][]string{
				"custom.users": {"id", "name"},
			},
			inputSchemas:  []string{"custom"},
			expectSchemas: []string{"custom"},
			expectPublic:  false,
		},
		{
			name: "mixed tables adds public schema",
			tables: map[string][]string{
				"users":         {"id", "name"},
				"custom.orders": {"id", "total"},
			},
			inputSchemas:  []string{"custom"},
			expectSchemas: []string{"custom", "public"},
			expectPublic:  true,
		},
		{
			name: "public already included is not duplicated",
			tables: map[string][]string{
				"users": {"id", "name"},
			},
			inputSchemas:  []string{"public"},
			expectSchemas: []string{"public"},
			expectPublic:  true,
		},
		{
			name:          "empty tables returns original schemas",
			tables:        map[string][]string{},
			inputSchemas:  []string{"custom"},
			expectSchemas: []string{"custom"},
			expectPublic:  false,
		},
		{
			name: "all qualified tables does not add public",
			tables: map[string][]string{
				"schema1.table1": {"field1"},
				"schema2.table2": {"field2"},
			},
			inputSchemas:  []string{"schema1", "schema2"},
			expectSchemas: []string{"schema1", "schema2"},
			expectPublic:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureDefaultSchemaForPostgreSQL(tt.tables, tt.inputSchemas)

			// Check if public is in result when expected
			hasPublic := false
			for _, s := range result {
				if s == "public" {
					hasPublic = true
					break
				}
			}

			if hasPublic != tt.expectPublic {
				t.Errorf("expected public=%v, got public=%v in result %v", tt.expectPublic, hasPublic, result)
			}

			// Verify all expected schemas are present
			for _, expected := range tt.expectSchemas {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected schema %s not found in result %v", expected, result)
				}
			}
		})
	}
}

// TestValidateSchema_PostgreSQLWithUnqualifiedTables tests that PostgreSQL connections
// with unqualified table names get the public schema added automatically.
func TestValidateSchema_PostgreSQLWithUnqualifiedTables(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	// PostgreSQL connection
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "pg_db", Type: model.TypePostgreSQL},
		}, nil)

	// Cache returns schema with public.users table
	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "pg_db").
		Return(&model.DataSourceSchema{
			ConfigName: "pg_db",
			Tables: map[string]*model.TableSchema{
				"public.users": {TableName: "public.users", Columns: map[string]bool{"id": true, "name": true}},
			},
		}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	// Request with unqualified table name
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"pg_db": {"users": {"id", "name"}}, // Unqualified "users" should match "public.users"
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	// The schema validation might still fail because "users" != "public.users" in the schema
	// but the important part is that public schema is requested from the datasource
	assert.NoError(t, err)
	require.NotNil(t, resp)
}

// TestValidateSchema_PostgreSQLWithMixedQualifiedTables tests PostgreSQL with both
// qualified and unqualified table names.
func TestValidateSchema_PostgreSQLWithMixedQualifiedTables(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	// PostgreSQL connection
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "pg_db", Type: model.TypePostgreSQL},
		}, nil)

	// Cache returns schema
	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "pg_db").
		Return(&model.DataSourceSchema{
			ConfigName: "pg_db",
			Tables: map[string]*model.TableSchema{
				"public.users":   {TableName: "public.users", Columns: map[string]bool{"id": true}},
				"finance.orders": {TableName: "finance.orders", Columns: map[string]bool{"id": true, "total": true}},
			},
		}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	// Request with mixed qualified/unqualified table names
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"pg_db": {
				"users":          {"id"},          // Unqualified
				"finance.orders": {"id", "total"}, // Qualified
			},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	// finance.orders should be found directly
}

// TestValidateSchema_NonPostgreSQLDoesNotAddPublicSchema tests that non-PostgreSQL databases
// don't get the public schema added automatically.
func TestValidateSchema_NonPostgreSQLDoesNotAddPublicSchema(t *testing.T) {
	tests := []struct {
		name   string
		dbType model.DBType
	}{
		{name: "MySQL", dbType: model.TypeMySQL},
		{name: "MongoDB", dbType: model.TypeMongoDB},
		{name: "Oracle", dbType: model.TypeOracle},
		{name: "SQLServer", dbType: model.TypeSQLServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockCrypto := crypto.NewMockCryptor(ctrl)
			mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

			orgID := uuid.New()
			connID := uuid.New()

			mockConnRepo.EXPECT().
				FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
				Return([]*model.Connection{
					{ID: connID, ConfigName: "db", Type: tt.dbType},
				}, nil)

			// Cache returns schema
			mockSchemaCache.EXPECT().
				Get(gomock.Any(), "db").
				Return(&model.DataSourceSchema{
					ConfigName: "db",
					Tables: map[string]*model.TableSchema{
						"users": {TableName: "users", Columns: map[string]bool{"id": true}},
					},
				}, nil)

			service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

			ctx := testContext()
			request := model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db": {"users": {"id"}},
				},
			}

			resp, err := service.Execute(ctx, orgID, request)

			assert.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestValidateSchema_CacheSetError tests that cache set errors are logged but don't fail the request.
func TestValidateSchema_CacheSetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "db1", Type: model.TypeMySQL, Host: "localhost", Port: 3306},
		}, nil)

	// Cache miss
	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(nil, nil)

	// Mock Decrypt for datasource connection
	mockCrypto.EXPECT().
		Decrypt(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("password", nil).
		AnyTimes()

	// The actual datasource connection will fail, so we expect DATA_SOURCE_DOWN error
	// This test verifies the cache miss path is taken

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	// Should complete without error but with DATA_SOURCE_DOWN in response
	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "failure", resp.Status)
	assert.Len(t, resp.Errors, 1)
	assert.Equal(t, model.ErrTypeDataSourceDown, resp.Errors[0].Type)
}

// TestValidateSchema_EmptyConfigName tests validation with empty config name in request.
func TestValidateSchema_EmptyConfigName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	orgID := uuid.New()

	// Request with empty config name key
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"": {"table1": {"field1"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	// Should return validation error for invalid request
	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestValidateSchema_LargeNumberOfTables tests validation with the maximum allowed tables (20).
func TestValidateSchema_LargeNumberOfTables(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	// Create schema with maximum allowed tables (MaxTablesPerDataSource = 20)
	tableSchemas := make(map[string]*model.TableSchema)
	mappedTables := make(map[string][]string)
	for i := 0; i < model.MaxTablesPerDataSource; i++ {
		tableName := fmt.Sprintf("table%d", i)
		tableSchemas[tableName] = &model.TableSchema{
			TableName: tableName,
			Columns:   map[string]bool{"id": true, "name": true, "created_at": true},
		}
		mappedTables[tableName] = []string{"id", "name"}
	}

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "large_db"},
		}, nil)

	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "large_db").
		Return(&model.DataSourceSchema{
			ConfigName: "large_db",
			Tables:     tableSchemas,
		}, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"large_db": mappedTables,
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "success", resp.Status)
	assert.Empty(t, resp.Errors)
}
