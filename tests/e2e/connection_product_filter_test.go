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
// filtered by product name returns only connections belonging to that product.
func TestListConnections_FilterByProduct_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create product A with 2 connections
	productNameA := e2eshared.GenerateProductName()
	productAConnIDs := make([]string, 0, 2)

	for i := 0; i < 2; i++ {
		uniqueName := fmt.Sprintf("e2e-filter-a-%s-%d", uuid.New().String()[:8], i)
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
		require.NoError(t, err, "create connection %d for product A", i)
		productAConnIDs = append(productAConnIDs, conn.ID)
	}

	t.Cleanup(func() {
		for _, id := range productAConnIDs {
			_ = apiClient.DeleteConnection(context.Background(), id)
		}
	})

	// Create product B with 1 connection
	productNameB := e2eshared.GenerateProductName()

	uniqueNameB := fmt.Sprintf("e2e-filter-b-%s", uuid.New().String()[:8])
	connB, err := apiClient.CreateConnection(ctx, productNameB, e2eshared.ConnectionInput{
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
	result, err := apiClient.ListConnectionsWithProductName(ctx, productNameA, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list connections with product A filter")

	assert.NotNil(t, result, "result should not be nil")
	assert.Equal(t, 2, len(result.Items), "should return exactly 2 connections for product A")

	// Verify all returned connections belong to product A
	for _, item := range result.Items {
		assert.Equal(t, productNameA, item.ProductName,
			"all returned connections should belong to product A")
	}

	// Verify product B's connection is NOT in the results
	for _, item := range result.Items {
		assert.NotEqual(t, connB.ID, item.ID,
			"product B's connection should not appear in product A's list")
	}

	t.Logf("Product filter returned %d connections for product A (expected 2)", len(result.Items))
}

// TestListConnections_FilterByProduct_NoResults verifies that listing connections
// for a product with no connections returns an empty items array.
func TestListConnections_FilterByProduct_NoResults(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Use a product name with no connections
	productName := e2eshared.GenerateProductName()

	result, err := apiClient.ListConnectionsWithProductName(ctx, productName, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list connections should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.Empty(t, result.Items, "items should be empty for product with no connections")

	t.Logf("Product with no connections returned %d items", len(result.Items))
}

// TestListConnections_FilterByProduct_NonexistentProduct_EmptyList verifies that listing
// connections with a non-existent product name returns 200 with an empty list.
// Since product names are just string labels (no Product entity), a non-existent name
// simply returns no matching connections.
func TestListConnections_FilterByProduct_NonexistentProduct_EmptyList(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentProductName := fmt.Sprintf("nonexistent-%s", uuid.New().String()[:8])

	result, err := apiClient.ListConnectionsWithProductName(ctx, nonExistentProductName, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list connections should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.Empty(t, result.Items, "items should be empty for non-existent product name")

	t.Logf("Non-existent product name returned %d items (expected 0)", len(result.Items))
}
