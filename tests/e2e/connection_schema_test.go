//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// connectionAvailableTimeout is how long to wait for a connection to become available
// before fetching its schema.
const connectionAvailableTimeout = 30 * time.Second

// TestGetConnectionSchema_PostgreSQL_Success verifies that the schema endpoint returns
// the correct tables and fields for a PostgreSQL connection, filtering out system tables
// and returning sorted fields.
func TestGetConnectionSchema_PostgreSQL_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-schema-pg-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Wait for the connection to be available before fetching schema
	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, connectionAvailableTimeout)
	require.NoError(t, err, "wait for connection available")

	// Get schema
	schema, err := apiClient.GetConnectionSchema(ctx, conn.ID)
	require.NoError(t, err, "get connection schema")

	// Assert response has tables
	require.NotNil(t, schema, "schema should not be nil")
	require.NotEmpty(t, schema.Tables, "schema should have tables")

	// Assert "transactions" table is found (seeded via postgres_init.sql)
	var transactionsTable *e2eshared.ConnectionSchemaTable

	for i, table := range schema.Tables {
		if table.Name == "transactions" {
			transactionsTable = &schema.Tables[i]

			break
		}
	}

	require.NotNil(t, transactionsTable, "transactions table should be present in schema")
	assert.NotEmpty(t, transactionsTable.Fields, "transactions table should have fields")

	// Assert system tables are NOT present
	for _, table := range schema.Tables {
		assert.False(t, strings.HasPrefix(table.Name, "pg_"),
			"system table %q should not be present", table.Name)
		assert.NotEqual(t, "information_schema", table.Name,
			"information_schema should not be present as a table")
	}

	// Assert fields in each table are sorted
	for _, table := range schema.Tables {
		if len(table.Fields) > 1 {
			sorted := make([]string, len(table.Fields))
			copy(sorted, table.Fields)
			sort.Strings(sorted)

			assert.Equal(t, sorted, table.Fields,
				"fields in table %q should be sorted alphabetically", table.Name)
		}
	}

	t.Logf("PostgreSQL schema: %d tables, transactions table has %d fields",
		len(schema.Tables), len(transactionsTable.Fields))
}

// TestGetConnectionSchema_NotFound_404 verifies that requesting the schema for a
// non-existent connection returns a 404 Not Found error.
func TestGetConnectionSchema_NotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentID := uuid.New().String()

	resp, err := apiClient.GetConnectionSchemaRaw(ctx, nonExistentID)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 404, resp.StatusCode(), "should return 404 for non-existent connection")
	t.Logf("Non-existent connection schema correctly returned 404")
}

// TestGetConnectionSchema_MySQL_Success verifies that the schema endpoint returns
// the correct tables and fields for a MySQL connection, filtering out system schemas.
func TestGetConnectionSchema_MySQL_Success(t *testing.T) {
	t.Parallel()

	if mysqlInfra == nil {
		t.Skip("MySQL infrastructure not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	mysqlHost, mysqlPort, err := mysqlInfra.HostPort()
	require.NoError(t, err, "get mysql host/port")

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-schema-mysql-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeMySQL,
		Host:         mysqlHost,
		Port:         mysqlPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Wait for the connection to be available before fetching schema
	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, connectionAvailableTimeout)
	require.NoError(t, err, "wait for connection available")

	// Get schema
	schema, err := apiClient.GetConnectionSchema(ctx, conn.ID)
	require.NoError(t, err, "get connection schema")

	require.NotNil(t, schema, "schema should not be nil")
	require.NotEmpty(t, schema.Tables, "schema should have tables")

	// Assert MySQL system schemas are filtered out
	mysqlSystemSchemas := []string{"information_schema", "mysql", "performance_schema", "sys"}
	for _, table := range schema.Tables {
		for _, sysSchema := range mysqlSystemSchemas {
			assert.NotEqual(t, sysSchema, table.Name,
				"system schema %q should not be present as a table", sysSchema)
		}
	}

	t.Logf("MySQL schema: %d tables returned", len(schema.Tables))
}

// TestGetConnectionSchema_Oracle_Success verifies that the schema endpoint returns
// tables and fields for an Oracle connection, filtering out system schemas.
func TestGetConnectionSchema_Oracle_Success(t *testing.T) {
	t.Parallel()

	if oracleInfra == nil {
		t.Skip("Oracle infrastructure not available (set E2E_ENABLE_ORACLE=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	oracleHost, oraclePort, err := oracleInfra.HostPort()
	require.NoError(t, err, "get oracle host/port")

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-schema-oracle-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeOracle,
		Host:         oracleHost,
		Port:         oraclePort,
		DatabaseName: "XEPDB1",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, connectionAvailableTimeout)
	require.NoError(t, err, "wait for connection available")

	schema, err := apiClient.GetConnectionSchema(ctx, conn.ID)
	require.NoError(t, err, "get connection schema")

	require.NotNil(t, schema, "schema should not be nil")
	require.NotEmpty(t, schema.Tables, "schema should have tables")

	// Assert Oracle system schemas are filtered out
	for _, table := range schema.Tables {
		assert.False(t, strings.HasPrefix(table.Name, "SYS."),
			"system table %q should not be present", table.Name)
		assert.False(t, strings.HasPrefix(table.Name, "SYSTEM."),
			"system table %q should not be present", table.Name)
		assert.False(t, strings.Contains(table.Name, "$"),
			"system table %q with $ should not be present", table.Name)
	}

	t.Logf("Oracle schema: %d tables returned", len(schema.Tables))
}

// TestGetConnectionSchema_SQLServer_Success verifies that the schema endpoint returns
// tables and fields for a SQL Server connection, filtering out system schemas.
func TestGetConnectionSchema_SQLServer_Success(t *testing.T) {
	t.Parallel()

	if mssqlInfra == nil {
		t.Skip("SQL Server infrastructure not available (set E2E_ENABLE_MSSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	mssqlHost, mssqlPort, err := mssqlInfra.HostPort()
	require.NoError(t, err, "get mssql host/port")

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-schema-mssql-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeSQLServer,
		Host:         mssqlHost,
		Port:         mssqlPort,
		DatabaseName: "testdb",
		Username:     "sa",
		Password:     "TestPass123!",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, connectionAvailableTimeout)
	require.NoError(t, err, "wait for connection available")

	schema, err := apiClient.GetConnectionSchema(ctx, conn.ID)
	require.NoError(t, err, "get connection schema")

	require.NotNil(t, schema, "schema should not be nil")
	require.NotEmpty(t, schema.Tables, "schema should have tables")

	// Assert SQL Server system schemas are filtered out
	for _, table := range schema.Tables {
		assert.False(t, strings.HasPrefix(table.Name, "sys."),
			"system table %q should not be present", table.Name)
		assert.False(t, strings.HasPrefix(table.Name, "information_schema."),
			"system table %q should not be present", table.Name)
	}

	t.Logf("SQL Server schema: %d tables returned", len(schema.Tables))
}

// TestGetConnectionSchema_MongoDB_Success verifies that the schema endpoint returns
// collections and fields for a MongoDB connection, filtering out system collections.
func TestGetConnectionSchema_MongoDB_Success(t *testing.T) {
	t.Parallel()

	if mongodbSourceInfra == nil {
		t.Skip("MongoDB source infrastructure not available (set E2E_ENABLE_MONGODB=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	mongoHost, mongoPort, err := mongodbSourceInfra.HostPort()
	require.NoError(t, err, "get mongodb host/port")

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-schema-mongo-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeMongoDB,
		Host:         mongoHost,
		Port:         mongoPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"authSource": "admin",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, connectionAvailableTimeout)
	require.NoError(t, err, "wait for connection available")

	schema, err := apiClient.GetConnectionSchema(ctx, conn.ID)
	require.NoError(t, err, "get connection schema")

	require.NotNil(t, schema, "schema should not be nil")
	require.NotEmpty(t, schema.Tables, "schema should have collections")

	// Assert MongoDB system collections are filtered out
	for _, table := range schema.Tables {
		assert.False(t, strings.HasPrefix(table.Name, "system."),
			"system collection %q should not be present", table.Name)
	}

	t.Logf("MongoDB schema: %d collections returned", len(schema.Tables))
}
