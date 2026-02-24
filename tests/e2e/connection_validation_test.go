//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"strings"
	"testing"

	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CONFIG NAME BOUNDARY TESTS
// =============================================================================

// TestConnection_ConfigName_MinLength verifies that config names at minimum
// length (3 characters) are accepted.
func TestConnection_ConfigName_MinLength(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	// 3 character config name (minimum)
	connInput := e2eshared.ConnectionInput{
		ConfigName:   "abc",
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection with min length config name")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.NotEmpty(t, conn.ID)
	assert.Equal(t, "abc", conn.ConfigName)
	t.Logf("Min length (3 chars) config name accepted: %s", conn.ConfigName)
}

// TestConnection_ConfigName_BelowMin verifies that config names below minimum
// length (2 characters) are rejected.
func TestConnection_ConfigName_BelowMin(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	// 2 character config name (below minimum)
	connInput := e2eshared.ConnectionInput{
		ConfigName:   "ab",
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should reject config name below min length")
	t.Logf("Config name below min length correctly rejected: status %d", resp.StatusCode())
}

// TestConnection_ConfigName_MaxLength verifies that config names at maximum
// length (100 characters) are accepted.
func TestConnection_ConfigName_MaxLength(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	// 100 character config name (maximum)
	maxName := strings.Repeat("a", 100)
	connInput := e2eshared.ConnectionInput{
		ConfigName:   maxName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection with max length config name")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.NotEmpty(t, conn.ID)
	assert.Equal(t, maxName, conn.ConfigName)
	t.Logf("Max length (100 chars) config name accepted")
}

// TestConnection_ConfigName_AboveMax verifies that config names above maximum
// length (101 characters) are rejected.
func TestConnection_ConfigName_AboveMax(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	// 101 character config name (above maximum)
	tooLongName := strings.Repeat("a", 101)
	connInput := e2eshared.ConnectionInput{
		ConfigName:   tooLongName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should reject config name above max length")
	t.Logf("Config name above max length correctly rejected: status %d", resp.StatusCode())
}

// =============================================================================
// PORT BOUNDARY TESTS
// =============================================================================

// TestConnection_Port_MinBoundary verifies that port 1 (minimum valid port) is accepted.
func TestConnection_Port_MinBoundary(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	// Note: Port 1 won't actually connect, but the API should accept it
	uniqueName := fmt.Sprintf("e2e-port-min-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         "localhost",
		Port:         1, // Minimum valid port
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection with port 1")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.Equal(t, 1, conn.Port)
	t.Logf("Port 1 (min boundary) accepted")
}

// TestConnection_Port_BelowMin verifies that port 0 is rejected.
func TestConnection_Port_BelowMin(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-port-zero-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         "localhost",
		Port:         0, // Invalid - below minimum
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should reject port 0")
	t.Logf("Port 0 correctly rejected: status %d", resp.StatusCode())
}

// TestConnection_Port_MaxBoundary verifies that port 65535 (maximum valid port) is accepted.
func TestConnection_Port_MaxBoundary(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-port-max-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         "localhost",
		Port:         65535, // Maximum valid port
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection with port 65535")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.Equal(t, 65535, conn.Port)
	t.Logf("Port 65535 (max boundary) accepted")
}

// TestConnection_Port_AboveMax verifies that port 65536 is rejected.
func TestConnection_Port_AboveMax(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-port-overflow-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         "localhost",
		Port:         65536, // Invalid - above maximum
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should reject port 65536")
	t.Logf("Port 65536 correctly rejected: status %d", resp.StatusCode())
}

// TestConnection_Port_Negative verifies that negative ports are rejected.
func TestConnection_Port_Negative(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-port-neg-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         "localhost",
		Port:         -1, // Invalid - negative
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should reject negative port")
	t.Logf("Negative port correctly rejected: status %d", resp.StatusCode())
}

// =============================================================================
// ADDITIONAL REQUIRED FIELDS TESTS
// =============================================================================

// TestConnection_MissingPassword verifies that creating a connection
// without a password is rejected.
func TestConnection_MissingPassword(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-no-pass-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		// Password intentionally omitted
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should reject missing password")
	t.Logf("Missing password correctly rejected: status %d", resp.StatusCode())
}

// TestConnection_MissingDatabaseName verifies that creating a connection
// without a database name is rejected.
func TestConnection_MissingDatabaseName(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-no-db-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName: uniqueName,
		Type:       e2eshared.DBTypePostgreSQL,
		Host:       pgHost,
		Port:       pgPort,
		// DatabaseName intentionally omitted
		Username: "testuser",
		Password: "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should reject missing database name")
	t.Logf("Missing database name correctly rejected: status %d", resp.StatusCode())
}

// TestConnection_MissingUsername verifies that creating a connection
// without a username is rejected.
func TestConnection_MissingUsername(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-no-user-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		// Username intentionally omitted
		Password: "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should reject missing username")
	t.Logf("Missing username correctly rejected: status %d", resp.StatusCode())
}

// =============================================================================
// DATABASE TYPE VALIDATION TESTS
// =============================================================================

// TestConnection_InvalidDatabaseType verifies that an unsupported database
// type is rejected.
func TestConnection_InvalidDatabaseType(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-invalid-type-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         "UNSUPPORTED_DB",
		Host:         "localhost",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should reject invalid database type")
	t.Logf("Invalid database type correctly rejected: status %d", resp.StatusCode())
}

// TestConnection_MetadataPreserved verifies that custom metadata is preserved
// when creating and retrieving a connection.
func TestConnection_MetadataPreserved(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-metadata-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"environment": "test",
			"team":        "platform",
			"version":     "1.0",
			"nested": map[string]any{
				"key": "value",
			},
		},
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Retrieve connection and verify metadata
	retrieved, err := apiClient.GetConnection(ctx, conn.ID)
	require.NoError(t, err, "get connection")

	assert.NotNil(t, retrieved.Metadata, "metadata should be present")
	assert.Equal(t, "test", retrieved.Metadata["environment"])
	assert.Equal(t, "platform", retrieved.Metadata["team"])

	t.Logf("Metadata preserved correctly for connection %s", conn.ID)
}
