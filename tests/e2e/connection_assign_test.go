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

// TestListUnassignedConnections_Success verifies that the unassigned connections endpoint
// returns a valid paginated response structure.
func TestListUnassignedConnections_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	result, err := apiClient.ListUnassignedConnections(ctx, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list unassigned connections should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.NotNil(t, result.Items, "items should not be nil")
	assert.GreaterOrEqual(t, result.Page, 1, "page should be present and >= 1")
	assert.Greater(t, result.Limit, 0, "limit should be present and > 0")
	assert.GreaterOrEqual(t, result.Total, 0, "total should be present and >= 0")

	t.Logf("Unassigned connections: %d items, page=%d, limit=%d, total=%d",
		len(result.Items), result.Page, result.Limit, result.Total)
}

// TestAssignConnection_AlreadyAssigned_Conflict verifies that assigning a connection
// that is already assigned to a product returns a 409 Conflict error.
func TestAssignConnection_AlreadyAssigned_Conflict(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create a connection assigned to product A
	productNameA := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-assign-conflict-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productNameA, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Create product B name
	productNameB := e2eshared.GenerateProductName()

	// Attempt to assign the connection (already assigned to product A) to product B
	resp, err := apiClient.AssignConnectionRaw(ctx, conn.ID, productNameB)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 409, resp.StatusCode(), "should return 409 Conflict for already-assigned connection")
	t.Logf("Already-assigned connection correctly rejected with status %d", resp.StatusCode())
}

// TestAssignConnection_ConnectionNotFound_404 verifies that assigning a non-existent
// connection returns a 404 Not Found error.
func TestAssignConnection_ConnectionNotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	// Attempt to assign a non-existent connection
	resp, err := apiClient.AssignConnectionRaw(ctx, uuid.New().String(), productName)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 404, resp.StatusCode(), "should return 404 for non-existent connection")
	t.Logf("Non-existent connection correctly returned status %d", resp.StatusCode())
}

// TestAssignConnection_EmptyProductName_BadRequest verifies that assigning a connection
// with an empty product name returns a 400 Bad Request error.
func TestAssignConnection_EmptyProductName_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-assign-invalid-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Attempt to assign with empty product name
	resp, err := apiClient.AssignConnectionRaw(ctx, conn.ID, "")
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should return 400 Bad Request for empty product name")
	t.Logf("Empty product name correctly rejected with status %d", resp.StatusCode())
}

// TestAssignConnection_InvalidProductName_BadRequest verifies that assigning a connection
// with an invalid product name (whitespace, special characters, too long) returns 400.
func TestAssignConnection_InvalidProductName_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()
	uniqueName := fmt.Sprintf("e2e-assign-badname-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	invalidNames := []struct {
		name        string
		productName string
	}{
		{"whitespace_only", "   "},
		{"special_characters", "product@name!"},
		{"too_long", strings.Repeat("a", 101)},
	}

	for _, tt := range invalidNames {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			subCtx, subCancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer subCancel()

			resp, err := apiClient.AssignConnectionRaw(subCtx, conn.ID, tt.productName)
			require.NoError(t, err, "request should succeed")

			assert.Equal(t, 400, resp.StatusCode(),
				"invalid product name %q should return 400", tt.name)
		})
	}
}
