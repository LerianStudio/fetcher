package query

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource/hostsafety"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	cacheRepo "github.com/LerianStudio/fetcher/pkg/ports/cache"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/resolver"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newGetConnectionSchemaSvc wires the GetConnectionSchema service the way the
// production bootstrap does: connection resolution flows through the connection
// authority Engine (over the connection repo), and schema DISCOVERY flows through
// the schema-discovery Engine (over the datasource factory). The connection
// repo's FindByID expectations are satisfied through the connection Engine; the
// datasource factory's GetSchemaInfo/Close expectations through the schema Engine.
func newGetConnectionSchemaSvc(
	t *testing.T,
	mockConnRepo connRepo.Repository,
	factory dsFactoryFunc,
	multiTenantEnabled bool,
) *GetConnectionSchema {
	t.Helper()

	connEng := scopeAuthorityEngine(t, mockConnRepo)
	schemaEng := schemaDiscoveryEngine(t, factory, nil, nil)

	return NewGetConnectionSchema(nil, nil, connEng, schemaEng, multiTenantEnabled)
}

// dsFactoryFunc mirrors datasource.DataSourceFactory for test factory closures.
type dsFactoryFunc = func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error)

// newSchemaConnectionFixture creates a valid Connection for testing GetConnectionSchema service.
func newSchemaConnectionFixture(connID uuid.UUID, dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID:                   connID,
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
	mockDataSource := datasource.NewMockDataSource(ctrl)

	// Create mock factory
	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(connID, model.TypePostgreSQL)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
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

	result, err := svc.Execute(ctx, connID)

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

// TestGetConnectionSchema_Execute_AlwaysFresh_BypassesCache proves the migration
// preserves the pre-embedded-Engine GET-schema freshness contract: the endpoint
// is ALWAYS-LIVE, so the schema cache is NEITHER read NOR written. The cache mock
// is wired with NO Get/Set expectations and the gomock controller fails the test
// if either is called; the datasource factory IS invoked (live discovery) on the
// call, proving the discovery actually ran. ValidateSchema's cache-first contract
// is asserted separately by the validate_schema cache tests.
func TestGetConnectionSchema_Execute_AlwaysFresh_BypassesCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)

	// Strict cache mock: any Get or Set is an unexpected call and fails the test,
	// proving GET schema never consults or populates the cache.
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	// Build the service with a CACHE-BACKED schema engine (unlike the nil-cache
	// helper) so the bypass is exercised against a live cache port.
	connEng := scopeAuthorityEngine(t, mockConnRepo)
	schemaEng := schemaDiscoveryEngine(t, mockFactory, nil, mockSchemaCache)
	svc := NewGetConnectionSchema(nil, nil, connEng, schemaEng, false)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(connID, model.TypePostgreSQL)

	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	// Live discovery MUST run on every call (cache bypassed): factory + close fire.
	schema := model.NewDataSourceSchema("test-connection")
	schema.AddTable("users", []string{"id", "name"})
	mockDataSource.EXPECT().GetSchemaInfo(gomock.Any(), gomock.Any()).Return(schema, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)

	result, err := svc.Execute(ctx, connID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, "users", result.Tables[0].Name)
}

// TestGetConnectionSchema_Execute_NotFound tests connection not found scenario.
func TestGetConnectionSchema_Execute_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	// Mock factory won't be called
	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		t.Fatal("factory should not be called when connection not found")
		return nil, nil
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

	ctx := testContext()
	connID := uuid.New()

	// Mock: connection not found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, connID)

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

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		t.Fatal("factory should not be called on repository error")
		return nil, nil
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

	ctx := testContext()
	connID := uuid.New()

	dbError := errors.New("database connection failed")

	// Mock: repository error
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, connID)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, dbError))
}

// TestGetConnectionSchema_Execute_DataSourceFactoryError tests datasource creation
// (CONNECT-stage) error handling. A factory failure is a connect-stage failure, so
// it must render the "Database Connection Error" title — the SAME title the /test
// endpoint returns — distinct from a discovery-read failure ("Schema Retrieval
// Error"). This is the branch-level regression this fix restores.
func TestGetConnectionSchema_Execute_DataSourceFactoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	factoryError := errors.New("failed to create datasource")
	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return nil, factoryError
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(connID, model.TypePostgreSQL)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	result, err := svc.Execute(ctx, connID)

	assert.Nil(t, result)
	assert.Error(t, err)

	var respErr pkg.ResponseError
	if assert.True(t, errors.As(err, &respErr)) {
		assert.Equal(t, http.StatusInternalServerError, respErr.Code)
		assert.Equal(t, "Database Connection Error", respErr.Title,
			"a connect-stage failure must render the Database Connection Error title, matching /test")
	}
}

// TestGetConnectionSchema_Execute_GetSchemaInfoError tests schema retrieval error.
func TestGetConnectionSchema_Execute_GetSchemaInfoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(connID, model.TypePostgreSQL)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	schemaError := errors.New("failed to get schema info")
	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(nil, schemaError)

	mockDataSource.EXPECT().
		Close(gomock.Any()).
		Return(nil)

	result, err := svc.Execute(ctx, connID)

	assert.Nil(t, result)
	assert.Error(t, err)

	// A DISCOVERY-read failure (connected, but GetSchemaInfo failed) must keep the
	// "Schema Retrieval Error" title — distinct from a connect-stage failure. This
	// is the other half of the two-title contract.
	var respErr pkg.ResponseError
	if assert.True(t, errors.As(err, &respErr)) {
		assert.Equal(t, http.StatusInternalServerError, respErr.Code)
		assert.Equal(t, "Schema Retrieval Error", respErr.Title,
			"a discovery-read failure must stay the Schema Retrieval Error title")
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
			mockDataSource := datasource.NewMockDataSource(ctrl)

			mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
				return mockDataSource, nil
			}

			svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

			ctx := testContext()
			connID := uuid.New()
			existingConn := newSchemaConnectionFixture(connID, tt.dbType)

			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID).
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

			result, err := svc.Execute(ctx, connID)

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
	mockDataSource := datasource.NewMockDataSource(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(connID, model.TypePostgreSQL)

	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	// Nil schema (edge case - empty database)
	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	mockDataSource.EXPECT().
		Close(gomock.Any()).
		Return(nil)

	result, err := svc.Execute(ctx, connID)

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
	mockDataSource := datasource.NewMockDataSource(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

	ctx := testContext()
	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(connID, model.TypePostgreSQL)

	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	// Empty schema
	schema := model.NewDataSourceSchema("test-connection")

	mockDataSource.EXPECT().
		GetSchemaInfo(gomock.Any(), gomock.Any()).
		Return(schema, nil)

	mockDataSource.EXPECT().
		Close(gomock.Any()).
		Return(nil)

	result, err := svc.Execute(ctx, connID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Tables)
}

// TestGetConnectionSchema_Execute_OrganizationIsolation tests that connections are isolated by organization.
func TestGetConnectionSchema_Execute_OrganizationIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		t.Fatal("factory should not be called when connection not found")
		return nil, nil
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

	ctx := testContext()
	connID := uuid.New()

	// Repository returns nil because connection belongs to different organization
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, connID)

	assert.Nil(t, result)
	assert.Error(t, err)

	var respErr pkg.ResponseErrorWithStatusCode
	if assert.True(t, errors.As(err, &respErr)) {
		assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
	}
}

// TestGetConnectionSchema_Execute_SchemaResolution_MultiTenantGating verifies that the
// username-as-schema heuristic only triggers for internal PostgreSQL connections when
// MULTI_TENANT_ENABLED is true. In single-tenant deployments, the adapter must receive
// nil so its "public" default applies.
func TestGetConnectionSchema_Execute_SchemaResolution_MultiTenantGating(t *testing.T) {
	tests := []struct {
		name               string
		multiTenantEnabled bool
		mutate             func(*model.Connection)
		expectedSchemas    []string
	}{
		{
			name:               "MT off + internal PG → nil (adapter applies public default)",
			multiTenantEnabled: false,
			mutate: func(c *model.Connection) {
				c.EncryptionKeyVersion = "" // internal connection marker
				c.Username = "midaz_onboarding"
			},
			expectedSchemas: nil,
		},
		{
			name:               "MT on + internal PG → username as schema",
			multiTenantEnabled: true,
			mutate: func(c *model.Connection) {
				c.EncryptionKeyVersion = ""
				c.Username = "midaz_onboarding"
			},
			expectedSchemas: []string{"midaz_onboarding"},
		},
		{
			name:               "MT on + external PG (EncryptionKeyVersion set) → nil",
			multiTenantEnabled: true,
			mutate: func(c *model.Connection) {
				c.EncryptionKeyVersion = "v1"
				c.Username = "any_user"
			},
			expectedSchemas: nil,
		},
		{
			name:               "explicit Schema field always wins regardless of MT",
			multiTenantEnabled: false,
			mutate: func(c *model.Connection) {
				c.EncryptionKeyVersion = ""
				c.Username = "midaz_onboarding"
				explicit := "accounting"
				c.Schema = &explicit
			},
			expectedSchemas: []string{"accounting"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockDataSource := datasource.NewMockDataSource(ctrl)

			mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
				return mockDataSource, nil
			}

			svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, tt.multiTenantEnabled)

			ctx := testContext()
			connID := uuid.New()
			existingConn := newSchemaConnectionFixture(connID, model.TypePostgreSQL)
			tt.mutate(existingConn)

			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID).
				Return(existingConn, nil)

			schema := model.NewDataSourceSchema("test-connection")
			mockDataSource.EXPECT().
				GetSchemaInfo(gomock.Any(), gomock.Eq(tt.expectedSchemas)).
				Return(schema, nil)

			mockDataSource.EXPECT().
				Close(gomock.Any()).
				Return(nil)

			result, err := svc.Execute(ctx, connID)

			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

// TestGetConnectionSchema_Execute_InternalDatasource verifies that an internal
// datasource (resolved via the registry + resolver, a host concern) is resolved
// on the host hot path and never routed through the Engine's connection store,
// then has its schema discovered through the schema Engine.
func TestGetConnectionSchema_Execute_InternalDatasource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockResolver := resolver.NewMockConnectionResolver(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)
	registry := resolver.NewInternalDatasourceRegistry()

	// The deterministic per-tenant UUID for the built-in internal "midaz_onboarding"
	// config in single-tenant mode (empty tenant id).
	connID := uuid.NewSHA1(resolver.InternalDatasourceNamespace, []byte("/midaz_onboarding"))

	internalConn := &model.Connection{
		ConfigName:   "midaz_onboarding",
		Type:         model.TypePostgreSQL,
		Host:         "internal-db",
		DatabaseName: "ledger",
		// EncryptionKeyVersion intentionally empty: internal/in-memory connection.
	}

	mockResolver.EXPECT().
		ResolveInternalByConfigName(gomock.Any(), "midaz_onboarding").
		Return(internalConn, nil)

	schema := model.NewDataSourceSchema("midaz_onboarding")
	schema.AddTable("account", []string{"id", "name"})
	mockDataSource.EXPECT().GetSchemaInfo(gomock.Any(), gomock.Any()).Return(schema, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)

	factory := func(context.Context, *model.Connection, crypto.Cryptor) (datasource.DataSource, error) {
		return mockDataSource, nil
	}

	svc := NewGetConnectionSchema(mockResolver, registry,
		scopeAuthorityEngine(t, mockConnRepo),
		schemaDiscoveryEngine(t, factory, nil, nil),
		false,
	)

	result, err := svc.Execute(testContext(), connID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "midaz_onboarding", result.ConfigName)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, "account", result.Tables[0].Name)
}

// TestNewGetConnectionSchema verifies the constructor.
func TestNewGetConnectionSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		return nil, nil
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, mockFactory, false)

	assert.NotNil(t, svc)
}

// TestGetConnectionSchema_Execute_HostSafetyRejectionPropagates verifies that a
// host-safety (SSRF / FET-0414) rejection is surfaced verbatim as a
// pkg.ValidationError rather than masked behind a generic 500. The guard runs
// HOST-side before discovery is delegated to the Engine: the Engine returns
// transport-neutral errors, but typed pkg.ValidationErrors are preserved verbatim
// and re-mapped host-side to 400. Running the SSRF guard host-side keeps the
// FET-0414 audit signal and its 400 mapping intact, so a tenant connection whose
// host is denylisted is rejected before any datasource call. The factory
// therefore must NOT be invoked.
func TestGetConnectionSchema_Execute_HostSafetyRejectionPropagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hostsafety.SetHostSafetyEnabled(true)
	defer hostsafety.SetHostSafetyEnabled(false)

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	connID := uuid.New()
	existingConn := newSchemaConnectionFixture(connID, model.TypePostgreSQL)
	existingConn.Host = "127.0.0.1" // denylisted by the SSRF guard for tenant connections

	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	stubFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		t.Fatal("factory must not be called when the host-safety guard rejects the connection")
		return nil, nil
	}

	svc := newGetConnectionSchemaSvc(t, mockConnRepo, stubFactory, false)

	_, err := svc.Execute(testContext(), connID)
	require.Error(t, err)

	var ve pkg.ValidationError
	require.True(t, errors.As(err, &ve),
		"GetConnectionSchema must propagate the host-safety pkg.ValidationError unchanged, got: %T %v", err, err)
	assert.Equal(t, "FET-0414", ve.Code)
}
