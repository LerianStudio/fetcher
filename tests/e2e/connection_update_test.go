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

// TestUpdateConnection_Success verifies that a connection can be updated
// with new values using PATCH semantics.
func TestUpdateConnection_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-update-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"version": "1",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Update connection
	newName := fmt.Sprintf("e2e-updated-%s", uuid.New().String()[:8])
	newPort := 5433
	updateInput := e2eshared.ConnectionUpdateInput{
		ConfigName: &newName,
		Port:       &newPort,
	}

	updated, err := apiClient.UpdateConnection(ctx, conn.ID, updateInput)
	require.NoError(t, err, "update connection")

	// Verify changes
	assert.Equal(t, conn.ID, updated.ID, "ID should not change")
	assert.Equal(t, newName, updated.ConfigName, "config name should be updated")
	assert.Equal(t, newPort, updated.Port, "port should be updated")

	// Verify unchanged fields
	assert.Equal(t, pgHost, updated.Host, "host should not change")
	assert.Equal(t, "testdb", updated.DatabaseName, "database name should not change")
	assert.Equal(t, "testuser", updated.Username, "username should not change")
	assert.Equal(t, e2eshared.DBTypePostgreSQL, updated.Type, "type should not change")

	t.Logf("Updated connection: id=%s, new name=%s, new port=%d", updated.ID, updated.ConfigName, updated.Port)
}

// TestUpdateConnection_PartialUpdate_Success verifies that PATCH allows
// updating only specific fields without affecting others.
func TestUpdateConnection_PartialUpdate_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection with all fields
	uniqueName := fmt.Sprintf("e2e-partial-%s", uuid.New().String()[:8])
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

	// Update only the database name
	newDBName := "updated_testdb"
	updateInput := e2eshared.ConnectionUpdateInput{
		DatabaseName: &newDBName,
	}

	updated, err := apiClient.UpdateConnection(ctx, conn.ID, updateInput)
	require.NoError(t, err, "update connection")

	// Verify only database name changed
	assert.Equal(t, newDBName, updated.DatabaseName, "database name should be updated")
	assert.Equal(t, uniqueName, updated.ConfigName, "config name should not change")
	assert.Equal(t, pgHost, updated.Host, "host should not change")
	assert.Equal(t, pgPort, updated.Port, "port should not change")
	assert.Equal(t, "testuser", updated.Username, "username should not change")

	t.Logf("Partial update successful: only database name changed to %s", updated.DatabaseName)
}

// TestUpdateConnection_NotFound_404 verifies that updating a non-existent
// connection returns a 404 Not Found error.
func TestUpdateConnection_NotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentID := uuid.New().String()
	newName := "should-not-exist"
	updateInput := e2eshared.ConnectionUpdateInput{
		ConfigName: &newName,
	}

	resp, err := apiClient.UpdateConnectionRaw(ctx, nonExistentID, updateInput)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 404, "")
	t.Logf("Update of non-existent connection correctly returned 404")
}

// TestUpdateConnection_EmptyUpdate_BadRequest verifies that an empty update
// request is rejected with a 400 Bad Request error.
func TestUpdateConnection_EmptyUpdate_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-empty-update-%s", uuid.New().String()[:8])
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

	// Send empty update - should be rejected
	updateInput := e2eshared.ConnectionUpdateInput{}

	resp, err := apiClient.UpdateConnectionRaw(ctx, conn.ID, updateInput)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 400, "empty request body")
	t.Logf("Empty update correctly rejected with status %d", resp.StatusCode())
}
