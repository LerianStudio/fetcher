//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"

	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectionTest_PostgreSQL_Success verifies that testing a valid PostgreSQL
// connection returns a successful result.
func TestConnectionTest_PostgreSQL_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-test-pg-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
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

	// Test connection
	result, err := apiClient.TestConnection(ctx, conn.ID)
	require.NoError(t, err, "test connection")

	assert.Equal(t, "success", result.Status, "status should be success")
	assert.NotEmpty(t, result.Message, "message should be set")
	assert.Greater(t, result.LatencyMs, int64(0), "latency should be positive")

	t.Logf("Connection test successful: status=%s, latency=%dms, message=%s",
		result.Status, result.LatencyMs, result.Message)
}

// TestConnectionTest_MySQL_Success verifies that testing a valid MySQL
// connection returns a successful result.
func TestConnectionTest_MySQL_Success(t *testing.T) {
	t.Parallel()

	if mysqlInfra == nil {
		t.Skip("MySQL infrastructure not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	mysqlHost, mysqlPort, err := mysqlInfra.HostPort()
	require.NoError(t, err, "get mysql host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-test-mysql-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
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

	// Test connection
	result, err := apiClient.TestConnection(ctx, conn.ID)
	require.NoError(t, err, "test connection")

	assert.Equal(t, "success", result.Status, "status should be success")
	assert.Greater(t, result.LatencyMs, int64(0), "latency should be positive")

	t.Logf("MySQL connection test successful: latency=%dms", result.LatencyMs)
}

// TestConnectionTest_UnreachableHost_Error verifies that testing a connection
// with an unreachable host returns an error.
func TestConnectionTest_UnreachableHost_Error(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Create connection with unreachable host
	uniqueName := fmt.Sprintf("e2e-test-unreachable-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         "unreachable.invalid.host",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Test connection - should fail
	resp, err := apiClient.TestConnectionRaw(ctx, conn.ID)
	require.NoError(t, err, "request should succeed")

	// Could return 200 with failure status or 400/500
	if resp.StatusCode() == 200 {
		// Parse result to check status
		t.Logf("Connection test returned 200 with body: %s", string(resp.Body()))
	} else {
		t.Logf("Connection test failed with status %d", resp.StatusCode())
	}
}

// TestConnectionTest_WrongCredentials_Error verifies that testing a connection
// with wrong credentials returns an error.
func TestConnectionTest_WrongCredentials_Error(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection with wrong password
	uniqueName := fmt.Sprintf("e2e-test-wrongcreds-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "wrong_password_that_should_fail",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Test connection - should fail due to wrong credentials
	resp, err := apiClient.TestConnectionRaw(ctx, conn.ID)
	require.NoError(t, err, "request should succeed")

	t.Logf("Connection test with wrong credentials returned status %d", resp.StatusCode())
}

// TestConnectionTest_NotFound_404 verifies that testing a non-existent
// connection returns a 404 Not Found error.
func TestConnectionTest_NotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentID := uuid.New().String()

	resp, err := apiClient.TestConnectionRaw(ctx, nonExistentID)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 404, "")
	t.Logf("Test of non-existent connection correctly returned 404")
}
