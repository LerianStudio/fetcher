package query

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newSchemaConnectionFixture creates a valid Connection for testing GetConnectionSchema service.
func newSchemaConnectionFixture(orgID, connID uuid.UUID, dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID:                   connID,
		OrganizationID:       orgID,
		ConfigName:           "test-connection",
		Type:                 dbType,
		Host:                 "localhost",
		Port:                 5432,
		DatabaseName:         "testdb",
		Username:             "testuser",
		PasswordEncrypted:    "encrypted-password",
		EncryptionKeyVersion: "v1",
		CreatedAt:            time.Now().UTC().Add(-24 * time.Hour),
		UpdatedAt:            time.Now().UTC().Add(-1 * time.Hour),
	}
}

// TestGetConnectionSchema_Execute_Success tests successful schema retrieval.
func TestGetConnectionSchema_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)

	// Create mock factory
	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(orgID, connID, model.TypePostgreSQL)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: schema info returned with user tables and system tables
	schema := model.NewDataSourceSchema("test-connection")
	schema.AddTable("users", []string{"id", "name", "email"})
	schema.AddTable("orders", []string{"id", "user_id", "total"})
	schema.AddTable("pg_catalog", []string{"oid", "relname"}) // system table - should be filtered

	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(schema, nil)

	mockDataSource.EXPECT().
		Close(gomock.Any()).
		Return(nil)

	result, err := svc.Execute(ctx, orgID, connID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, connID.String(), result.ID)
	assert.Equal(t, "test-connection", result.ConfigName)
	assert.Equal(t, "testdb", result.DatabaseName)
	assert.Equal(t, string(model.TypePostgreSQL), result.Type)
	assert.Len(t, result.Tables, 2) // pg_catalog should be filtered out

	// Verify user tables are present
	tableNames := make([]string, 0, len(result.Tables))
	for _, t := range result.Tables {
		tableNames = append(tableNames, t.Name)
	}
	assert.Contains(t, tableNames, "users")
	assert.Contains(t, tableNames, "orders")
	assert.NotContains(t, tableNames, "pg_catalog")
}

// TestGetConnectionSchema_Execute_NotFound tests connection not found scenario.
func TestGetConnectionSchema_Execute_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)

	// Mock factory won't be called
	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		t.Fatal("factory should not be called when connection not found")
		return nil, nil
	}

	svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	// Mock: connection not found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID)

	assert.Nil(t, result)
	assert.Error(t, err)

	var respErr pkg.ResponseErrorWithStatusCode
	if assert.True(t, errors.As(err, &respErr)) {
		assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
	}
}

// TestGetConnectionSchema_Execute_RepositoryError tests repository error handling.
func TestGetConnectionSchema_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		t.Fatal("factory should not be called on repository error")
		return nil, nil
	}

	svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	dbError := errors.New("database connection failed")

	// Mock: repository error
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, connID)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, dbError))
}

// TestGetConnectionSchema_Execute_DataSourceFactoryError tests datasource creation error.
func TestGetConnectionSchema_Execute_DataSourceFactoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)

	factoryError := errors.New("failed to create datasource")
	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return nil, factoryError
	}

	svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(orgID, connID, model.TypePostgreSQL)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	result, err := svc.Execute(ctx, orgID, connID)

	assert.Nil(t, result)
	assert.Error(t, err)

	var respErr pkg.ResponseError
	if assert.True(t, errors.As(err, &respErr)) {
		assert.Equal(t, http.StatusInternalServerError, respErr.Code)
	}
}

// TestGetConnectionSchema_Execute_GetSchemaInfoError tests schema retrieval error.
func TestGetConnectionSchema_Execute_GetSchemaInfoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(orgID, connID, model.TypePostgreSQL)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	schemaError := errors.New("failed to get schema info")
	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(nil, schemaError)

	mockDataSource.EXPECT().
		Close(gomock.Any()).
		Return(nil)

	result, err := svc.Execute(ctx, orgID, connID)

	assert.Nil(t, result)
	assert.Error(t, err)

	var respErr pkg.ResponseError
	if assert.True(t, errors.As(err, &respErr)) {
		assert.Equal(t, http.StatusInternalServerError, respErr.Code)
	}
}

// TestGetConnectionSchema_Execute_FiltersSystemTables tests that system tables are filtered.
func TestGetConnectionSchema_Execute_FiltersSystemTables(t *testing.T) {
	tests := []struct {
		name           string
		dbType         model.DBType
		tables         map[string][]string
		expectedTables []string
		filteredTables []string
	}{
		{
			name:   "PostgreSQL filters pg_catalog and information_schema",
			dbType: model.TypePostgreSQL,
			tables: map[string][]string{
				"users":              {"id", "name"},
				"pg_catalog":         {"oid"},
				"information_schema": {"table_name"},
				"pg_toast":           {"chunk_id"},
			},
			expectedTables: []string{"users"},
			filteredTables: []string{"pg_catalog", "information_schema", "pg_toast"},
		},
		{
			name:   "MySQL filters mysql and information_schema",
			dbType: model.TypeMySQL,
			tables: map[string][]string{
				"products":           {"id", "name"},
				"mysql":              {"user"},
				"information_schema": {"tables"},
				"performance_schema": {"events"},
				"sys":                {"version"},
			},
			expectedTables: []string{"products"},
			filteredTables: []string{"mysql", "information_schema", "performance_schema", "sys"},
		},
		{
			name:   "Oracle filters SYS and SYSTEM",
			dbType: model.TypeOracle,
			tables: map[string][]string{
				"EMPLOYEES": {"ID", "NAME"},
				"SYS":       {"DUAL"},
				"SYSTEM":    {"HELP"},
				"OUTLN":     {"OL"},
				"XDB":       {"RESOURCE"},
				"MDSYS":     {"SDO"},
				"CTXSYS":    {"DR"},
			},
			expectedTables: []string{"EMPLOYEES"},
			filteredTables: []string{"SYS", "SYSTEM", "OUTLN", "XDB", "MDSYS", "CTXSYS"},
		},
		{
			name:   "SQL Server filters sys and INFORMATION_SCHEMA",
			dbType: model.TypeSQLServer,
			tables: map[string][]string{
				"Customers":          {"Id", "Name"},
				"sys":                {"objects"},
				"INFORMATION_SCHEMA": {"TABLES"},
			},
			expectedTables: []string{"Customers"},
			filteredTables: []string{"sys", "INFORMATION_SCHEMA"},
		},
		{
			name:   "MongoDB filters admin and local",
			dbType: model.TypeMongoDB,
			tables: map[string][]string{
				"users":  {"_id", "name"},
				"admin":  {"system.users"},
				"local":  {"oplog.rs"},
				"config": {"chunks"},
			},
			expectedTables: []string{"users"},
			filteredTables: []string{"admin", "local", "config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockCrypto := crypto.NewMockCryptor(ctrl)
			mockDataSource := datasource.NewMockDataSource(ctrl)

			mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
				return mockDataSource, nil
			}

			svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()
			existingConn := newSchemaConnectionFixture(orgID, connID, tt.dbType)

			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID, orgID).
				Return(existingConn, nil)

			schema := model.NewDataSourceSchema("test-connection")
			for tableName, columns := range tt.tables {
				schema.AddTable(tableName, columns)
			}

			mockDataSource.EXPECT().
				GetSchemaInfo(gomock.Any(), gomock.Any()).
				Return(schema, nil)

			mockDataSource.EXPECT().
				Close(gomock.Any()).
				Return(nil)

			result, err := svc.Execute(ctx, orgID, connID)

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Get returned table names
			returnedTables := make([]string, 0, len(result.Tables))
			for _, tbl := range result.Tables {
				returnedTables = append(returnedTables, tbl.Name)
			}

			// Verify expected tables are present
			for _, expected := range tt.expectedTables {
				assert.Contains(t, returnedTables, expected, "expected table %s not found", expected)
			}

			// Verify filtered tables are NOT present
			for _, filtered := range tt.filteredTables {
				assert.NotContains(t, returnedTables, filtered, "system table %s should be filtered", filtered)
			}
		})
	}
}

// TestGetConnectionSchema_Execute_NilSchema tests handling of nil schema from GetSchemaInfo.
func TestGetConnectionSchema_Execute_NilSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(orgID, connID, model.TypePostgreSQL)

	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Nil schema (edge case - empty database)
	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	mockDataSource.EXPECT().
		Close(gomock.Any()).
		Return(nil)

	result, err := svc.Execute(ctx, orgID, connID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Tables)
	assert.Equal(t, connID.String(), result.ID)
	assert.Equal(t, "test-connection", result.ConfigName)
}

// TestGetConnectionSchema_Execute_EmptySchema tests handling of empty schema.
func TestGetConnectionSchema_Execute_EmptySchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(orgID, connID, model.TypePostgreSQL)

	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Empty schema
	schema := model.NewDataSourceSchema("test-connection")

	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(schema, nil)

	mockDataSource.EXPECT().
		Close(gomock.Any()).
		Return(nil)

	result, err := svc.Execute(ctx, orgID, connID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Tables)
}

// TestGetConnectionSchema_Execute_OrganizationIsolation tests that connections are isolated by organization.
func TestGetConnectionSchema_Execute_OrganizationIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		t.Fatal("factory should not be called when connection not found")
		return nil, nil
	}

	svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

	ctx := testContext()
	requestingOrgID := uuid.New()
	connID := uuid.New()

	// Repository returns nil because connection belongs to different organization
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, requestingOrgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, requestingOrgID, connID)

	assert.Nil(t, result)
	assert.Error(t, err)

	var respErr pkg.ResponseErrorWithStatusCode
	if assert.True(t, errors.As(err, &respErr)) {
		assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
	}
}

// TestNewGetConnectionSchema verifies the constructor.
func TestNewGetConnectionSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCrypto := crypto.NewMockCryptor(ctrl)
	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return nil, nil
	}

	svc := NewGetConnectionSchema(mockConnRepo, mockCrypto, mockFactory)

	assert.NotNil(t, svc)
}

// TestIsSystemTable tests the isSystemTable function for all database types.
func TestIsSystemTable(t *testing.T) {
	tests := []struct {
		name      string
		dbType    model.DBType
		tableName string
		expected  bool
	}{
		// PostgreSQL
		{"PostgreSQL pg_catalog", model.TypePostgreSQL, "pg_catalog", true},
		{"PostgreSQL information_schema", model.TypePostgreSQL, "information_schema", true},
		{"PostgreSQL pg_toast", model.TypePostgreSQL, "pg_toast", true},
		{"PostgreSQL pg_temp_1", model.TypePostgreSQL, "pg_temp_1", true},
		{"PostgreSQL user table", model.TypePostgreSQL, "users", false},

		// MySQL
		{"MySQL mysql", model.TypeMySQL, "mysql", true},
		{"MySQL information_schema", model.TypeMySQL, "information_schema", true},
		{"MySQL performance_schema", model.TypeMySQL, "performance_schema", true},
		{"MySQL sys", model.TypeMySQL, "sys", true},
		{"MySQL user table", model.TypeMySQL, "products", false},

		// Oracle - uppercase (standard)
		{"Oracle SYS", model.TypeOracle, "SYS", true},
		{"Oracle SYSTEM", model.TypeOracle, "SYSTEM", true},
		{"Oracle OUTLN", model.TypeOracle, "OUTLN", true},
		{"Oracle XDB", model.TypeOracle, "XDB", true},
		{"Oracle MDSYS", model.TypeOracle, "MDSYS", true},
		{"Oracle CTXSYS", model.TypeOracle, "CTXSYS", true},
		{"Oracle DBSNMP", model.TypeOracle, "DBSNMP", true},
		{"Oracle user table", model.TypeOracle, "EMPLOYEES", false},
		// Oracle - lowercase (driver-dependent)
		{"Oracle sys lowercase", model.TypeOracle, "sys", true},
		{"Oracle system lowercase", model.TypeOracle, "system", true},
		{"Oracle mixed case Sys", model.TypeOracle, "Sys", true},
		{"Oracle mixed case System", model.TypeOracle, "System", true},

		// SQL Server - exact matches
		{"SQLServer sys", model.TypeSQLServer, "sys", true},
		{"SQLServer INFORMATION_SCHEMA", model.TypeSQLServer, "INFORMATION_SCHEMA", true},
		{"SQLServer user table", model.TypeSQLServer, "Customers", false},
		// SQL Server - db_ prefix (per business requirement)
		{"SQLServer db_owner", model.TypeSQLServer, "db_owner", true},
		{"SQLServer db_backup", model.TypeSQLServer, "db_backup", true},
		{"SQLServer db_accessadmin", model.TypeSQLServer, "db_accessadmin", true},
		{"SQLServer db_backup.audit_logs", model.TypeSQLServer, "db_backup.audit_logs", true},
		{"SQLServer dbo.users", model.TypeSQLServer, "dbo.users", false},
		{"SQLServer sales.orders", model.TypeSQLServer, "sales.orders", false},

		// MongoDB - databases
		{"MongoDB admin", model.TypeMongoDB, "admin", true},
		{"MongoDB local", model.TypeMongoDB, "local", true},
		{"MongoDB config", model.TypeMongoDB, "config", true},
		{"MongoDB user collection", model.TypeMongoDB, "users", false},
		// MongoDB - system.* prefix (per business requirement)
		{"MongoDB system.indexes", model.TypeMongoDB, "system.indexes", true},
		{"MongoDB system.users", model.TypeMongoDB, "system.users", true},
		{"MongoDB system.profile", model.TypeMongoDB, "system.profile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSystemTable(tt.dbType, tt.tableName)
			assert.Equal(t, tt.expected, result, "isSystemTable(%s, %s)", tt.dbType, tt.tableName)
		})
	}
}
