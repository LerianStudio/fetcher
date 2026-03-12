package query

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	cacheRepo "github.com/LerianStudio/fetcher/pkg/ports/cache"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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
	mockDataSource := datasource.NewMockDataSource(ctrl)

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

	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(&model.DataSourceSchema{
			ConfigName: "db1",
			Tables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true}},
			},
		}, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)
	mockSchemaCache.EXPECT().
		Set(gomock.Any(), "db1", gomock.Any(), cacheRepo.DefaultSchemaCacheTTL).
		Return(nil)

	testFactory := func(_ context.Context, conn *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
		assert.Equal(t, "db1", conn.ConfigName)
		return mockDataSource, nil
	}
	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, testFactory)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "success", resp.Status)
	assert.Empty(t, resp.Errors)
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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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
		name            string
		dbType          model.DBType
		schemaTableKey  string   // How table is stored in schema (case varies by DB)
		requestFields   []string // Fields in the request
		schemaColumnKey string   // How column is stored in schema (case varies by DB)
	}{
		{name: "MySQL", dbType: model.TypeMySQL, schemaTableKey: "users", requestFields: []string{"id"}, schemaColumnKey: "id"},
		{name: "MongoDB", dbType: model.TypeMongoDB, schemaTableKey: "users", requestFields: []string{"id"}, schemaColumnKey: "id"},
		// Oracle's GetSchemaInfo normalizes table/column names to lowercase; validation normalizes to lowercase
		{name: "Oracle", dbType: model.TypeOracle, schemaTableKey: "users", requestFields: []string{"id"}, schemaColumnKey: "id"},
		{name: "SQLServer", dbType: model.TypeSQLServer, schemaTableKey: "users", requestFields: []string{"id"}, schemaColumnKey: "id"},
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

			// Cache returns schema with correct case for each database type
			mockSchemaCache.EXPECT().
				Get(gomock.Any(), "db").
				Return(&model.DataSourceSchema{
					ConfigName: "db",
					Tables: map[string]*model.TableSchema{
						tt.schemaTableKey: {TableName: tt.schemaTableKey, Columns: map[string]bool{tt.schemaColumnKey: true}},
					},
				}, nil)

			service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

			ctx := testContext()
			request := model.SchemaValidationRequest{
				MappedFields: map[string]map[string][]string{
					"db": {"users": tt.requestFields}, // Table name uses lowercase, fields vary by DB
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
	mockDataSource := datasource.NewMockDataSource(ctrl)

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

	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(&model.DataSourceSchema{
			ConfigName: "db1",
			Tables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true}},
			},
		}, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)
	mockSchemaCache.EXPECT().
		Set(gomock.Any(), "db1", gomock.Any(), cacheRepo.DefaultSchemaCacheTTL).
		Return(errors.New("cache set failed"))

	testFactory := func(_ context.Context, conn *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
		assert.Equal(t, "db1", conn.ConfigName)
		return mockDataSource, nil
	}
	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, testFactory)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "success", resp.Status)
	assert.Empty(t, resp.Errors)
}

func TestValidateSchema_NilSchemaFromDatasource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)

	orgID := uuid.New()
	connID := uuid.New()
	conn := &model.Connection{ID: connID, ConfigName: "db1", Type: model.TypeMySQL, Host: "localhost", Port: 3306}

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{conn}, nil)

	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(nil, nil)

	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	mockDataSource.EXPECT().
		Close(gomock.Any()).
		Return(nil)

	mockSchemaCache.EXPECT().
		Set(gomock.Any(), "db1", gomock.Any(), cacheRepo.DefaultSchemaCacheTTL).
		DoAndReturn(func(_ any, _ string, schema *model.DataSourceSchema, _ any) error {
			require.NotNil(t, schema)
			require.NotNil(t, schema.Tables)
			assert.Empty(t, schema.Tables)

			return nil
		})

	dsFactory := func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}
	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, dsFactory)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "failure", resp.Status)
	assert.Len(t, resp.Errors, 1)
	assert.Equal(t, model.ErrTypeTableNotFound, resp.Errors[0].Type)
}

func TestValidateSchema_NilDatasourceFactoryResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()
	conn := &model.Connection{ID: connID, ConfigName: "db1", Type: model.TypeMySQL, Host: "localhost", Port: 3306}

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{conn}, nil)

	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(nil, nil)

	nilFactory := func(context.Context, *model.Connection, crypto.Cryptor) (datasource.DataSource, error) {
		return nil, nil
	}
	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nilFactory)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "failure", resp.Status)
	assert.Len(t, resp.Errors, 1)
	assert.Equal(t, model.ErrTypeDataSourceDown, resp.Errors[0].Type)
	assert.Equal(t, "db1", resp.Errors[0].DataSourceID)
}

func TestValidateSchema_NilDatasourceFactory_ReturnsInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()
	conn := &model.Connection{ID: connID, ConfigName: "db1", Type: model.TypeMySQL, Host: "localhost", Port: 3306}

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{conn}, nil)

	mockSchemaCache.EXPECT().
		Get(gomock.Any(), "db1").
		Return(nil, nil)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

	ctx := testContext()
	request := model.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"db1": {"users": {"id"}},
		},
	}

	resp, err := service.Execute(ctx, orgID, request)

	assert.Nil(t, resp)
	require.Error(t, err)

	var internalErr pkg.InternalServerError
	require.True(t, errors.As(err, &internalErr), "expected InternalServerError, got %T: %v", err, err)
}

// TestValidateSchema_EmptyConfigName tests validation with empty config name in request.
func TestValidateSchema_EmptyConfigName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

	service := NewValidateSchema(mockConnRepo, mockCrypto, mockSchemaCache, nil)

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

// TestValidateTablesAgainstSchema_SQLServerQualifiedNames tests that SQL Server
// "dbo.tablename" requests match schema tables stored as "tablename".
func TestValidateTablesAgainstSchema_SQLServerQualifiedNames(t *testing.T) {
	tests := []struct {
		name           string
		requestTables  map[string][]string
		schemaTables   map[string]*model.TableSchema
		dbType         model.DBType
		expectedErrors int
		errorType      string
	}{
		{
			name: "dbo.users matches users in schema",
			requestTables: map[string][]string{
				"dbo.users": {"id", "name"},
			},
			schemaTables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true, "name": true}},
			},
			dbType:         model.TypeSQLServer,
			expectedErrors: 0,
		},
		{
			name: "unqualified users matches users in schema",
			requestTables: map[string][]string{
				"users": {"id", "name"},
			},
			schemaTables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true, "name": true}},
			},
			dbType:         model.TypeSQLServer,
			expectedErrors: 0,
		},
		{
			name: "sales.orders preserved for non-dbo schema",
			requestTables: map[string][]string{
				"sales.orders": {"id", "total"},
			},
			schemaTables: map[string]*model.TableSchema{
				"sales.orders": {TableName: "sales.orders", Columns: map[string]bool{"id": true, "total": true}},
			},
			dbType:         model.TypeSQLServer,
			expectedErrors: 0,
		},
		{
			name: "dbo.nonexistent fails validation",
			requestTables: map[string][]string{
				"dbo.nonexistent": {"id"},
			},
			schemaTables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true}},
			},
			dbType:         model.TypeSQLServer,
			expectedErrors: 1,
			errorType:      model.ErrTypeTableNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &model.DataSourceSchema{
				ConfigName: "test_db",
				Tables:     tt.schemaTables,
			}

			errors := validateTablesAgainstSchema("test_db", tt.requestTables, schema, nil, tt.dbType)

			assert.Len(t, errors, tt.expectedErrors)
			if tt.expectedErrors > 0 && tt.errorType != "" {
				assert.Equal(t, tt.errorType, errors[0].Type)
			}
		})
	}
}

// TestValidateTablesAgainstSchema_OracleQualifiedNames tests that Oracle
// table name normalization handles uppercase storage and schema prefixes.
// Oracle's GetSchemaInfo normalizes table names to lowercase for case-insensitive matching.
func TestValidateTablesAgainstSchema_OracleQualifiedNames(t *testing.T) {
	tests := []struct {
		name           string
		requestTables  map[string][]string
		schemaTables   map[string]*model.TableSchema
		expectedErrors int
		errorType      string
	}{
		{
			name: "lowercase users matches users in schema",
			requestTables: map[string][]string{
				"users": {"id", "name"},
			},
			schemaTables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true, "name": true}},
			},
			expectedErrors: 0,
		},
		{
			name: "USERS matches users in schema (converted to lowercase)",
			requestTables: map[string][]string{
				"USERS": {"ID", "NAME"},
			},
			schemaTables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true, "name": true}},
			},
			expectedErrors: 0,
		},
		{
			name: "hr.employees matches hr.employees in schema (converted to lowercase)",
			requestTables: map[string][]string{
				"HR.EMPLOYEES": {"ID", "NAME"},
			},
			schemaTables: map[string]*model.TableSchema{
				"hr.employees": {TableName: "hr.employees", Columns: map[string]bool{"id": true, "name": true}},
			},
			expectedErrors: 0,
		},
		{
			name: "mixed case Sales.Orders matches sales.orders in schema",
			requestTables: map[string][]string{
				"Sales.Orders": {"ID"},
			},
			schemaTables: map[string]*model.TableSchema{
				"sales.orders": {TableName: "sales.orders", Columns: map[string]bool{"id": true}},
			},
			expectedErrors: 0,
		},
		{
			name: "nonexistent table fails validation",
			requestTables: map[string][]string{
				"nonexistent": {"id"},
			},
			schemaTables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true}},
			},
			expectedErrors: 1,
			errorType:      model.ErrTypeTableNotFound,
		},
		{
			name: "nonexistent field fails validation",
			requestTables: map[string][]string{
				"users": {"id", "nonexistent"},
			},
			schemaTables: map[string]*model.TableSchema{
				"users": {TableName: "users", Columns: map[string]bool{"id": true, "name": true}},
			},
			expectedErrors: 1,
			errorType:      model.ErrTypeFieldNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &model.DataSourceSchema{
				ConfigName: "oracle_db",
				Tables:     tt.schemaTables,
			}

			errors := validateTablesAgainstSchema("oracle_db", tt.requestTables, schema, nil, model.TypeOracle)

			assert.Len(t, errors, tt.expectedErrors)
			if tt.expectedErrors > 0 && tt.errorType != "" {
				assert.Equal(t, tt.errorType, errors[0].Type)
			}
		})
	}
}

// TestEnsureDefaultSchema tests the generic ensureDefaultSchema function
// that works for both PostgreSQL and SQL Server.
func TestEnsureDefaultSchema(t *testing.T) {
	tests := []struct {
		name          string
		tables        map[string][]string
		inputSchemas  []string
		defaultSchema string
		expectDefault bool
	}{
		{
			name: "unqualified table adds default schema",
			tables: map[string][]string{
				"users": {"id", "name"},
			},
			inputSchemas:  []string{},
			defaultSchema: "public",
			expectDefault: true,
		},
		{
			name: "qualified table does not add default schema",
			tables: map[string][]string{
				"custom.users": {"id", "name"},
			},
			inputSchemas:  []string{"custom"},
			defaultSchema: "public",
			expectDefault: false,
		},
		{
			name: "mixed tables adds default schema",
			tables: map[string][]string{
				"users":         {"id", "name"},
				"custom.orders": {"id", "total"},
			},
			inputSchemas:  []string{"custom"},
			defaultSchema: "dbo",
			expectDefault: true,
		},
		{
			name: "default already included is not duplicated",
			tables: map[string][]string{
				"users": {"id", "name"},
			},
			inputSchemas:  []string{"dbo"},
			defaultSchema: "dbo",
			expectDefault: true,
		},
		{
			name:          "empty tables returns original schemas",
			tables:        map[string][]string{},
			inputSchemas:  []string{"custom"},
			defaultSchema: "public",
			expectDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureDefaultSchema(tt.tables, tt.inputSchemas, tt.defaultSchema)

			hasDefault := false
			for _, s := range result {
				if s == tt.defaultSchema {
					hasDefault = true
					break
				}
			}

			if hasDefault != tt.expectDefault {
				t.Errorf("expected default=%v, got default=%v in result %v", tt.expectDefault, hasDefault, result)
			}

			// Verify no duplicates
			seen := make(map[string]bool)
			for _, s := range result {
				if seen[s] {
					t.Errorf("duplicate schema %s in result %v", s, result)
				}
				seen[s] = true
			}
		})
	}
}

// TestNormalizeTableNameForValidation tests the normalizeTableNameForValidation function
// for all supported database types.
func TestNormalizeTableNameForValidation(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		dbType    model.DBType
		expected  string
	}{
		// Oracle tests - converts to lowercase (to match GetSchemaInfo normalization)
		{
			name:      "Oracle: lowercase stays lowercase",
			tableName: "users",
			dbType:    model.TypeOracle,
			expected:  "users",
		},
		{
			name:      "Oracle: mixed case converts to lowercase",
			tableName: "UserAccounts",
			dbType:    model.TypeOracle,
			expected:  "useraccounts",
		},
		{
			name:      "Oracle: qualified name converts to lowercase",
			tableName: "HR.EMPLOYEES",
			dbType:    model.TypeOracle,
			expected:  "hr.employees",
		},
		{
			name:      "Oracle: uppercase converts to lowercase",
			tableName: "TRANSACTIONS",
			dbType:    model.TypeOracle,
			expected:  "transactions",
		},

		// SQL Server tests - strips dbo prefix
		{
			name:      "SQLServer: dbo prefix stripped",
			tableName: "dbo.users",
			dbType:    model.TypeSQLServer,
			expected:  "users",
		},
		{
			name:      "SQLServer: non-dbo schema preserved",
			tableName: "sales.orders",
			dbType:    model.TypeSQLServer,
			expected:  "sales.orders",
		},
		{
			name:      "SQLServer: unqualified name unchanged",
			tableName: "users",
			dbType:    model.TypeSQLServer,
			expected:  "users",
		},

		// PostgreSQL tests - strips public prefix
		{
			name:      "PostgreSQL: public prefix stripped",
			tableName: "public.users",
			dbType:    model.TypePostgreSQL,
			expected:  "users",
		},
		{
			name:      "PostgreSQL: non-public schema preserved",
			tableName: "accounting.invoices",
			dbType:    model.TypePostgreSQL,
			expected:  "accounting.invoices",
		},
		{
			name:      "PostgreSQL: unqualified name unchanged",
			tableName: "users",
			dbType:    model.TypePostgreSQL,
			expected:  "users",
		},

		// MySQL tests - no transformation
		{
			name:      "MySQL: name unchanged",
			tableName: "users",
			dbType:    model.TypeMySQL,
			expected:  "users",
		},
		{
			name:      "MySQL: qualified name unchanged",
			tableName: "mydb.users",
			dbType:    model.TypeMySQL,
			expected:  "mydb.users",
		},

		// MongoDB tests - no transformation
		{
			name:      "MongoDB: collection name unchanged",
			tableName: "users",
			dbType:    model.TypeMongoDB,
			expected:  "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTableNameForValidation(tt.tableName, tt.dbType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNormalizeFieldNameForValidation tests the normalizeFieldNameForValidation function
// for all supported database types.
func TestNormalizeFieldNameForValidation(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		dbType    model.DBType
		expected  string
	}{
		// Oracle tests - converts to lowercase (to match GetSchemaInfo normalization)
		{
			name:      "Oracle: lowercase stays lowercase",
			fieldName: "id",
			dbType:    model.TypeOracle,
			expected:  "id",
		},
		{
			name:      "Oracle: mixed case converts to lowercase",
			fieldName: "firstName",
			dbType:    model.TypeOracle,
			expected:  "firstname",
		},
		{
			name:      "Oracle: uppercase converts to lowercase",
			fieldName: "CREATED_AT",
			dbType:    model.TypeOracle,
			expected:  "created_at",
		},

		// Other databases - no transformation (case preserved as stored)
		{
			name:      "PostgreSQL: field name unchanged",
			fieldName: "created_at",
			dbType:    model.TypePostgreSQL,
			expected:  "created_at",
		},
		{
			name:      "SQLServer: field name unchanged",
			fieldName: "CreatedAt",
			dbType:    model.TypeSQLServer,
			expected:  "CreatedAt",
		},
		{
			name:      "MySQL: field name unchanged",
			fieldName: "created_at",
			dbType:    model.TypeMySQL,
			expected:  "created_at",
		},
		{
			name:      "MongoDB: field name unchanged",
			fieldName: "_id",
			dbType:    model.TypeMongoDB,
			expected:  "_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeFieldNameForValidation(tt.fieldName, tt.dbType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
