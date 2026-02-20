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

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	// Create connection
	uniqueName := fmt.Sprintf("e2e-update-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
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

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	// Create connection with all fields
	uniqueName := fmt.Sprintf("e2e-partial-%s", uuid.New().String()[:8])
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

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	// Create connection
	uniqueName := fmt.Sprintf("e2e-empty-update-%s", uuid.New().String()[:8])
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

	// Send empty update - should be rejected
	updateInput := e2eshared.ConnectionUpdateInput{}

	resp, err := apiClient.UpdateConnectionRaw(ctx, conn.ID, updateInput)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 400, "empty request body")
	t.Logf("Empty update correctly rejected with status %d", resp.StatusCode())
}

// TestUpdateConnection_InvalidValues_BadRequest verifies that updating a connection
// with invalid field values is rejected with a 400 Bad Request error.
func TestUpdateConnection_InvalidValues_BadRequest(t *testing.T) {
	t.Parallel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	tests := []struct {
		name   string
		update e2eshared.ConnectionUpdateInput
	}{
		{
			name: "empty_host",
			update: func() e2eshared.ConnectionUpdateInput {
				host := ""
				return e2eshared.ConnectionUpdateInput{Host: &host}
			}(),
		},
		{
			name: "port_zero",
			update: func() e2eshared.ConnectionUpdateInput {
				port := 0
				return e2eshared.ConnectionUpdateInput{Port: &port}
			}(),
		},
		{
			name: "port_too_high",
			update: func() e2eshared.ConnectionUpdateInput {
				port := 99999
				return e2eshared.ConnectionUpdateInput{Port: &port}
			}(),
		},
		{
			name: "empty_database",
			update: func() e2eshared.ConnectionUpdateInput {
				db := ""
				return e2eshared.ConnectionUpdateInput{DatabaseName: &db}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer cancel()

			connInput := e2eshared.ConnectionInput{
				ConfigName:   fmt.Sprintf("e2e-inv-update-%s", uuid.New().String()[:8]),
				Type:         e2eshared.DBTypePostgreSQL,
				Host:         pgHost,
				Port:         pgPort,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpass",
			}

			_, conn := e2eshared.CreateTestProductAndConnection(t, apiClient, ctx, connInput)

			resp, err := apiClient.UpdateConnectionRaw(ctx, conn.ID, tt.update)
			require.NoError(t, err, "request should succeed")

			assert.Equal(t, 400, resp.StatusCode(),
				"invalid update %q should return 400 Bad Request, got %d", tt.name, resp.StatusCode())
			t.Logf("Invalid update %q correctly rejected with status %d", tt.name, resp.StatusCode())
		})
	}
}

// TestUpdateConnection_Metadata_Success verifies that connection metadata
// can be updated via PATCH and that the new metadata replaces the old values.
func TestUpdateConnection_Metadata_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	// Create connection with initial metadata
	uniqueName := fmt.Sprintf("e2e-meta-update-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"environment": "staging",
			"team":        "platform",
			"version":     "1",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Verify initial metadata
	require.NotNil(t, conn.Metadata, "initial metadata should be present")
	assert.Equal(t, "staging", conn.Metadata["environment"], "initial environment")
	assert.Equal(t, "platform", conn.Metadata["team"], "initial team")

	// Update metadata
	newMetadata := map[string]any{
		"environment": "production",
		"team":        "infra",
		"version":     "2",
		"region":      "us-east-1",
	}
	updateInput := e2eshared.ConnectionUpdateInput{
		Metadata: &newMetadata,
	}

	updated, err := apiClient.UpdateConnection(ctx, conn.ID, updateInput)
	require.NoError(t, err, "update connection metadata")

	// Verify metadata was updated
	require.NotNil(t, updated.Metadata, "updated metadata should be present")
	assert.Equal(t, "production", updated.Metadata["environment"], "environment should be updated")
	assert.Equal(t, "infra", updated.Metadata["team"], "team should be updated")
	assert.Equal(t, "2", updated.Metadata["version"], "version should be updated")
	assert.Equal(t, "us-east-1", updated.Metadata["region"], "new field should be present")

	// Verify non-metadata fields unchanged
	assert.Equal(t, conn.ID, updated.ID, "ID should not change")
	assert.Equal(t, uniqueName, updated.ConfigName, "config name should not change")
	assert.Equal(t, pgHost, updated.Host, "host should not change")

	// Verify via GET to confirm persistence
	retrieved, err := apiClient.GetConnection(ctx, conn.ID)
	require.NoError(t, err, "get connection after update")
	require.NotNil(t, retrieved.Metadata, "persisted metadata should be present")
	assert.Equal(t, "production", retrieved.Metadata["environment"], "persisted environment")
	assert.Equal(t, "us-east-1", retrieved.Metadata["region"], "persisted region")

	t.Logf("Metadata update successful: %d fields in updated metadata", len(updated.Metadata))
}
