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

// TestListConnections_FilterByProduct_Success verifies that listing connections
// filtered by product ID returns only connections belonging to that product.
func TestListConnections_FilterByProduct_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create product A with 2 connections
	productA := e2eshared.CreateTestProduct(t, apiClient, ctx)
	productAConnIDs := make([]string, 0, 2)

	for i := 0; i < 2; i++ {
		uniqueName := fmt.Sprintf("e2e-filter-a-%s-%d", uuid.New().String()[:8], i)
		connInput := e2eshared.ConnectionInput{
			ProductID:    productA.ID,
			ConfigName:   uniqueName,
			Type:         e2eshared.DBTypePostgreSQL,
			Host:         pgHost,
			Port:         pgPort,
			DatabaseName: "testdb",
			Username:     "testuser",
			Password:     "testpass",
		}

		conn, err := apiClient.CreateConnection(ctx, connInput)
		require.NoError(t, err, "create connection %d for product A", i)
		productAConnIDs = append(productAConnIDs, conn.ID)
	}

	t.Cleanup(func() {
		for _, id := range productAConnIDs {
			_ = apiClient.DeleteConnection(context.Background(), id)
		}
	})

	// Create product B with 1 connection
	productB := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueNameB := fmt.Sprintf("e2e-filter-b-%s", uuid.New().String()[:8])
	connB, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
		ProductID:    productB.ID,
		ConfigName:   uniqueNameB,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create connection for product B")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), connB.ID)
	})

	// List connections filtered by product A
	result, err := apiClient.ListConnectionsWithProduct(ctx, productA.ID, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list connections with product A filter")

	assert.NotNil(t, result, "result should not be nil")
	assert.Equal(t, 2, len(result.Items), "should return exactly 2 connections for product A")

	// Verify all returned connections belong to product A
	for _, item := range result.Items {
		assert.Equal(t, productA.ID, item.ProductID,
			"all returned connections should belong to product A")
	}

	t.Logf("Product filter returned %d connections for product A (expected 2)", len(result.Items))
}

// TestListConnections_FilterByProduct_NoResults verifies that listing connections
// for a product with no connections returns an empty items array.
func TestListConnections_FilterByProduct_NoResults(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Create a product with no connections
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	result, err := apiClient.ListConnectionsWithProduct(ctx, product.ID, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list connections should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.Empty(t, result.Items, "items should be empty for product with no connections")

	t.Logf("Product with no connections returned %d items", len(result.Items))
}

// TestListConnections_FilterByProduct_InvalidProductID verifies that listing connections
// with an invalid (non-UUID) product ID in the X-Product-Id header returns a 400 Bad Request error.
func TestListConnections_FilterByProduct_InvalidProductID(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	resp, err := apiClient.ListConnectionsWithProductRaw(ctx, "not-a-uuid", e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should return 400 Bad Request for invalid product ID")
	t.Logf("Invalid product ID correctly rejected with status %d", resp.StatusCode())
}

// TestListConnections_FilterByProduct_NonexistentProduct_EmptyList verifies that listing
// connections with a valid UUID for a non-existent product returns 404.
// The service calls productRepo.FindByID which returns nil for non-existent products,
// then returns ErrEntityNotFound.
func TestListConnections_FilterByProduct_NonexistentProduct_EmptyList(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentProductID := uuid.New().String()

	resp, err := apiClient.ListConnectionsWithProductRaw(ctx, nonExistentProductID, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 404, resp.StatusCode(), "should return 404 for non-existent product")
	t.Logf("Non-existent product correctly returned status %d", resp.StatusCode())
}
