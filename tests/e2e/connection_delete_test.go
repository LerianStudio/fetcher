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

// TestDeleteConnection_Success verifies that an existing connection
// can be deleted successfully.
func TestDeleteConnection_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-delete-%s", uuid.New().String()[:8])
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

	// Delete connection
	err = apiClient.DeleteConnection(ctx, conn.ID)
	require.NoError(t, err, "delete connection")

	// Verify connection is gone
	e2eshared.AssertConnectionNotFound(t, apiClient, conn.ID)

	t.Logf("Successfully deleted connection: id=%s", conn.ID)
}

// TestDeleteConnection_NotFound_404 verifies that deleting a non-existent
// connection returns a 404 Not Found error.
func TestDeleteConnection_NotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentID := uuid.New().String()

	resp, err := apiClient.DeleteConnectionRaw(ctx, nonExistentID)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 404, "")
	t.Logf("Delete of non-existent connection correctly returned 404")
}

// TestDeleteConnection_Idempotent verifies that deleting the same connection
// twice returns appropriate responses.
func TestDeleteConnection_Idempotent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-delete-idempotent-%s", uuid.New().String()[:8])
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

	// First delete should succeed
	err = apiClient.DeleteConnection(ctx, conn.ID)
	require.NoError(t, err, "first delete should succeed")

	// Second delete should return 404
	resp, err := apiClient.DeleteConnectionRaw(ctx, conn.ID)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 404, resp.StatusCode(), "second delete should return 404")

	t.Logf("Double delete handled correctly: first succeeded, second returned 404")
}

// TestDeleteConnection_InvalidID_BadRequest verifies that deleting with
// an invalid ID format returns an appropriate error.
func TestDeleteConnection_InvalidID_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	invalidID := "not-a-valid-uuid"

	resp, err := apiClient.DeleteConnectionRaw(ctx, invalidID)
	require.NoError(t, err, "request should succeed")

	// API should return either 400 (bad request) or 404 (not found)
	assert.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 404,
		"should return 400 or 404, got %d", resp.StatusCode())

	t.Logf("Delete with invalid ID returned status %d", resp.StatusCode())
}
