//go:build e2e

package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	e2eshared "github.com/LerianStudio/fetcher/v2/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateConnection_WithProductName_Success verifies that a connection can be
// created with a valid product name header and all fields are returned correctly.
func TestCreateConnection_WithProductName_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-create-product-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"test": "create-with-product",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.NotEmpty(t, conn.ID, "connection ID should be set")
	assert.Equal(t, uniqueName, conn.ConfigName, "config name should match")
	assert.Equal(t, e2eshared.DBTypePostgreSQL, conn.Type, "type should match")
	assert.Equal(t, pgHost, conn.Host, "host should match")
	assert.Equal(t, pgPort, conn.Port, "port should match")
	assert.Equal(t, "testdb", conn.DatabaseName, "database name should match")
	assert.Equal(t, "testuser", conn.Username, "username should match")
	assert.Equal(t, productName, conn.ProductName, "product name should match")
	assert.NotEmpty(t, conn.CreatedAt, "created_at should be set")

	e2eshared.AssertValidUUID(t, conn.ID)
	t.Logf("Created connection with product: id=%s, productName=%s, name=%s", conn.ID, conn.ProductName, conn.ConfigName)
}

// TestCreateConnection_WithoutProductName_BadRequest verifies that creating a connection
// without the X-Product-Name header returns a 400 Bad Request error.
func TestCreateConnection_WithoutProductName_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	uniqueName := fmt.Sprintf("e2e-no-product-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, "", connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should return 400 Bad Request for missing product name")
	t.Logf("Missing product name correctly rejected with status %d", resp.StatusCode())
}

// TestCreateConnection_PostgreSQL_Success verifies that a PostgreSQL connection
// can be created successfully with valid credentials.
func TestCreateConnection_PostgreSQL_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-pg-create-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"test": "create-postgresql",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.NotEmpty(t, conn.ID, "connection ID should be set")
	assert.Equal(t, uniqueName, conn.ConfigName, "config name should match")
	assert.Equal(t, e2eshared.DBTypePostgreSQL, conn.Type, "type should match")
	assert.Equal(t, pgHost, conn.Host, "host should match")
	assert.Equal(t, pgPort, conn.Port, "port should match")
	assert.Equal(t, "testdb", conn.DatabaseName, "database name should match")
	assert.Equal(t, "testuser", conn.Username, "username should match")
	assert.NotEmpty(t, conn.CreatedAt, "created_at should be set")

	e2eshared.AssertValidUUID(t, conn.ID)
	t.Logf("Created PostgreSQL connection: id=%s, name=%s", conn.ID, conn.ConfigName)
}

// TestCreateConnection_MySQL_Success verifies that a MySQL connection
// can be created successfully with valid credentials.
func TestCreateConnection_MySQL_Success(t *testing.T) {
	t.Parallel()

	if mysqlInfra == nil {
		t.Skip("MySQL infrastructure not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	mysqlHost, mysqlPort, err := mysqlInfra.HostPort()
	require.NoError(t, err, "get mysql host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-mysql-create-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeMySQL,
		Host:         mysqlHost,
		Port:         mysqlPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"test": "create-mysql",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.NotEmpty(t, conn.ID, "connection ID should be set")
	assert.Equal(t, uniqueName, conn.ConfigName, "config name should match")
	assert.Equal(t, e2eshared.DBTypeMySQL, conn.Type, "type should match")

	t.Logf("Created MySQL connection: id=%s, name=%s", conn.ID, conn.ConfigName)
}

// TestCreateConnection_MongoDB_Success verifies that a MongoDB connection
// can be created successfully with valid credentials.
func TestCreateConnection_MongoDB_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	// MongoDB uses the core infrastructure for app storage, not source database
	// For E2E, we test creating a connection config pointing to a hypothetical MongoDB
	uniqueName := fmt.Sprintf("e2e-mongo-create-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeMongoDB,
		Host:         "localhost",
		Port:         27017,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"test": "create-mongodb",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.NotEmpty(t, conn.ID, "connection ID should be set")
	assert.Equal(t, uniqueName, conn.ConfigName, "config name should match")
	assert.Equal(t, e2eshared.DBTypeMongoDB, conn.Type, "type should match")

	t.Logf("Created MongoDB connection: id=%s, name=%s", conn.ID, conn.ConfigName)
}

// TestCreateConnection_Oracle_Success verifies that an Oracle connection
// can be created successfully with valid credentials.
func TestCreateConnection_Oracle_Success(t *testing.T) {
	t.Parallel()

	if oracleInfra == nil {
		t.Skip("Oracle infrastructure not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	oracleHost, oraclePort, err := oracleInfra.HostPort()
	require.NoError(t, err, "get oracle host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-oracle-create-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeOracle,
		Host:         oracleHost,
		Port:         oraclePort,
		DatabaseName: "XE",
		Username:     "system",
		Password:     "testpass",
		Metadata: map[string]any{
			"test": "create-oracle",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.NotEmpty(t, conn.ID, "connection ID should be set")
	assert.Equal(t, uniqueName, conn.ConfigName, "config name should match")
	assert.Equal(t, e2eshared.DBTypeOracle, conn.Type, "type should match")

	t.Logf("Created Oracle connection: id=%s, name=%s", conn.ID, conn.ConfigName)
}

// TestCreateConnection_SQLServer_Success verifies that a SQL Server connection
// can be created successfully with valid credentials.
func TestCreateConnection_SQLServer_Success(t *testing.T) {
	t.Parallel()

	if mssqlInfra == nil {
		t.Skip("SQL Server infrastructure not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	mssqlHost, mssqlPort, err := mssqlInfra.HostPort()
	require.NoError(t, err, "get mssql host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-mssql-create-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeSQLServer,
		Host:         mssqlHost,
		Port:         mssqlPort,
		DatabaseName: "testdb",
		Username:     "sa",
		Password:     "YourStrong@Passw0rd",
		Metadata: map[string]any{
			"test": "create-sqlserver",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.NotEmpty(t, conn.ID, "connection ID should be set")
	assert.Equal(t, uniqueName, conn.ConfigName, "config name should match")
	assert.Equal(t, e2eshared.DBTypeSQLServer, conn.Type, "type should match")

	t.Logf("Created SQL Server connection: id=%s, name=%s", conn.ID, conn.ConfigName)
}

// TestCreateConnection_DuplicateConfigName_Conflict verifies that creating
// a connection with a duplicate config name returns a conflict error.
func TestCreateConnection_DuplicateConfigName_Conflict(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-duplicate-%s", uuid.New().String()[:8])

	// Create first connection
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn1, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create first connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn1.ID)
	})

	// Try to create second connection with same name
	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 409, resp.StatusCode(), "should return 409 Conflict")
	t.Logf("Duplicate connection correctly rejected with status %d", resp.StatusCode())
}

// TestCreateConnection_ConcurrentDuplicateName verifies that when two parallel requests
// attempt to create connections with the same configName, exactly one succeeds and the
// other receives a 409 Conflict error.
func TestCreateConnection_ConcurrentDuplicateName(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	sharedName := fmt.Sprintf("e2e-concurrent-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   sharedName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	type result struct {
		statusCode int
		connID     string
	}

	results := make(chan result, 2)

	// Launch two concurrent creation requests
	for i := 0; i < 2; i++ {
		go func() {
			resp, reqErr := apiClient.CreateConnectionRaw(ctx, productName, connInput)
			if reqErr != nil {
				results <- result{statusCode: -1}
				return
			}

			r := result{statusCode: resp.StatusCode()}
			if resp.StatusCode() == 200 || resp.StatusCode() == 201 {
				var conn e2eshared.ConnectionResponse
				if jsonErr := json.Unmarshal(resp.Body(), &conn); jsonErr == nil {
					r.connID = conn.ID
				}
			}

			results <- r
		}()
	}

	// Collect results
	r1 := <-results
	r2 := <-results

	// Cleanup any created connection
	t.Cleanup(func() {
		for _, r := range []result{r1, r2} {
			if r.connID != "" {
				_ = apiClient.DeleteConnection(context.Background(), r.connID)
			}
		}
	})

	// Exactly one should succeed and one should conflict
	codes := []int{r1.statusCode, r2.statusCode}

	successCount := 0
	conflictCount := 0

	for _, code := range codes {
		if code == 200 || code == 201 {
			successCount++
		} else if code == 409 {
			conflictCount++
		}
	}

	assert.Equal(t, 1, successCount,
		"exactly one request should succeed, got status codes %v", codes)
	assert.Equal(t, 1, conflictCount,
		"exactly one request should get 409 Conflict, got status codes %v", codes)

	t.Logf("Concurrent creation: status codes %v (1 success, 1 conflict)", codes)
}

// TestCreateConnection_MissingRequiredFields_BadRequest verifies that creating
// a connection with missing required fields returns a bad request error.
func TestCreateConnection_MissingRequiredFields_BadRequest(t *testing.T) {
	t.Parallel()

	productName := e2eshared.GenerateProductName()

	testCases := []struct {
		name     string
		input    e2eshared.ConnectionInput
		contains string
	}{
		{
			name: "missing_config_name",
			input: e2eshared.ConnectionInput{
				Type:         e2eshared.DBTypePostgreSQL,
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpass",
			},
			contains: "configName",
		},
		{
			name: "missing_type",
			input: e2eshared.ConnectionInput{
				ConfigName:   fmt.Sprintf("e2e-missing-type-%s", uuid.New().String()[:8]),
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpass",
			},
			contains: "type",
		},
		{
			name: "missing_host",
			input: e2eshared.ConnectionInput{
				ConfigName:   fmt.Sprintf("e2e-missing-host-%s", uuid.New().String()[:8]),
				Type:         e2eshared.DBTypePostgreSQL,
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpass",
			},
			contains: "host",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Each subtest needs its own context to avoid parent context cancellation
			subCtx, subCancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer subCancel()

			resp, err := apiClient.CreateConnectionRaw(subCtx, productName, tc.input)
			require.NoError(t, err, "request should succeed")

			assert.Equal(t, 400, resp.StatusCode(), "should return 400 Bad Request")
			t.Logf("Missing field correctly rejected with status %d", resp.StatusCode())
		})
	}
}

// TestCreateConnection_InvalidProductNameHeader verifies that creating a connection
// with an invalid X-Product-Name header returns a 400 Bad Request error.
func TestCreateConnection_InvalidProductNameHeader(t *testing.T) {
	t.Parallel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	tests := []struct {
		name        string
		productName string
		expectCode  int
	}{
		{
			name:        "whitespace_only",
			productName: "   ",
			expectCode:  400,
		},
		{
			name:        "special_characters",
			productName: "product@name!",
			expectCode:  400,
		},
		{
			name:        "too_long_101_chars",
			productName: strings.Repeat("a", 101),
			expectCode:  400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer cancel()

			connInput := e2eshared.ConnectionInput{
				ConfigName:   fmt.Sprintf("e2e-header-val-%s", uuid.New().String()[:8]),
				Type:         e2eshared.DBTypePostgreSQL,
				Host:         pgHost,
				Port:         pgPort,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpass",
			}

			resp, err := apiClient.CreateConnectionRaw(ctx, tt.productName, connInput)
			require.NoError(t, err, "request should succeed")

			assert.Equal(t, tt.expectCode, resp.StatusCode(),
				"product name %q should return %d", tt.name, tt.expectCode)
		})
	}
}

// TestCreateConnection_ProductNameMaxLength_Success verifies that a product name
// at the maximum allowed length (100 characters) is accepted.
func TestCreateConnection_ProductNameMaxLength_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := strings.Repeat("a", 100)
	connInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-maxlen-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "100-char product name should be accepted")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.Equal(t, productName, conn.ProductName, "product name should match")
}
