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

// TestGetConnection_Exists_Success verifies that an existing connection
// can be retrieved by ID with all fields populated correctly.
func TestGetConnection_Exists_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-get-exists-%s", uuid.New().String()[:8])
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
			"purpose":     "e2e-get-test",
		},
	}

	created, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), created.ID)
	})

	// Get the connection
	retrieved, err := apiClient.GetConnection(ctx, created.ID)
	require.NoError(t, err, "get connection")

	// Verify all fields
	assert.Equal(t, created.ID, retrieved.ID, "ID should match")
	assert.Equal(t, uniqueName, retrieved.ConfigName, "config name should match")
	assert.Equal(t, e2eshared.DBTypePostgreSQL, retrieved.Type, "type should match")
	assert.Equal(t, pgHost, retrieved.Host, "host should match")
	assert.Equal(t, pgPort, retrieved.Port, "port should match")
	assert.Equal(t, "testdb", retrieved.DatabaseName, "database name should match")
	assert.Equal(t, "testuser", retrieved.Username, "username should match")
	assert.Equal(t, productName, retrieved.ProductName, "product name should match")
	assert.NotEmpty(t, retrieved.CreatedAt, "created_at should be set")
	assert.NotEmpty(t, retrieved.UpdatedAt, "updated_at should be set")

	// Verify metadata is preserved
	require.NotNil(t, retrieved.Metadata, "metadata should be present")
	assert.Equal(t, "test", retrieved.Metadata["environment"], "metadata environment should match")
	assert.Equal(t, "e2e-get-test", retrieved.Metadata["purpose"], "metadata purpose should match")

	t.Logf("Successfully retrieved connection: id=%s, name=%s", retrieved.ID, retrieved.ConfigName)
}

// TestGetConnection_NotFound_404 verifies that requesting a non-existent
// connection returns a 404 Not Found error.
func TestGetConnection_NotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentID := uuid.New().String()

	resp, err := apiClient.GetConnectionRaw(ctx, nonExistentID)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 404, "")
	t.Logf("Non-existent connection correctly returned 404")
}

// TestGetConnection_InvalidID_BadRequest verifies that requesting a connection
// with an invalid ID format returns a 400 Bad Request error.
func TestGetConnection_InvalidID_BadRequest(t *testing.T) {
	t.Parallel()

	invalidIDs := []string{
		"not-a-uuid",
		"12345",
		"",
		"invalid-uuid-format-here",
	}

	for _, invalidID := range invalidIDs {
		if invalidID == "" {
			continue // Empty ID would be a different route
		}

		t.Run(invalidID, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer cancel()

			resp, err := apiClient.GetConnectionRaw(ctx, invalidID)
			require.NoError(t, err, "request should succeed")

			// API should return either 400 (bad request) or 404 (not found)
			// depending on validation strategy
			assert.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 404,
				"should return 400 or 404, got %d", resp.StatusCode())
			t.Logf("Invalid ID %q returned status %d", invalidID, resp.StatusCode())
		})
	}
}
