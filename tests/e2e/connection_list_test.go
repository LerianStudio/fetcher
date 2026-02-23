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

// TestListConnections_Empty_Success verifies that listing connections
// returns an empty result when no connections exist for the product.
func TestListConnections_Empty_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Use a product name with no connections — listing by product returns empty
	productName := e2eshared.GenerateProductName()

	result, err := apiClient.ListConnectionsWithProductName(ctx, productName, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list connections should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.Empty(t, result.Items, "items should be empty for product with no connections")

	t.Logf("List with empty product returned %d items", len(result.Items))
}

// TestListConnections_WithResults_Success verifies that listing connections
// returns the created connections with correct data.
func TestListConnections_WithResults_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	// Create multiple connections with unique prefix
	prefix := fmt.Sprintf("e2e-list-%s", uuid.New().String()[:8])
	createdIDs := make([]string, 0, 3)

	for i := 0; i < 3; i++ {
		connInput := e2eshared.ConnectionInput{
			ConfigName:   fmt.Sprintf("%s-%d", prefix, i),
			Type:         e2eshared.DBTypePostgreSQL,
			Host:         pgHost,
			Port:         pgPort,
			DatabaseName: "testdb",
			Username:     "testuser",
			Password:     "testpass",
		}

		conn, err := apiClient.CreateConnection(ctx, productName, connInput)
		require.NoError(t, err, "create connection %d", i)
		createdIDs = append(createdIDs, conn.ID)
	}

	t.Cleanup(func() {
		for _, id := range createdIDs {
			_ = apiClient.DeleteConnection(context.Background(), id)
		}
	})

	// List connections scoped to this test's product
	result, err := apiClient.ListConnectionsWithProductName(ctx, productName, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list connections should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.Equal(t, 3, len(result.Items), "should have exactly 3 connections for this product")

	// Verify our created connections are in the list
	foundCount := 0
	for _, item := range result.Items {
		for _, id := range createdIDs {
			if item.ID == id {
				foundCount++
				break
			}
		}
	}
	assert.Equal(t, 3, foundCount, "should find all 3 created connections")

	t.Logf("Listed %d connections, found %d created by this test", len(result.Items), foundCount)
}

// TestListConnections_FilterByType_Success verifies that listing connections
// scoped to a product returns connections with the expected type.
func TestListConnections_FilterByType_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	// Create a PostgreSQL connection
	uniqueName := fmt.Sprintf("e2e-filter-type-%s", uuid.New().String()[:8])
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

	// List connections scoped to product, then verify types client-side
	result, err := apiClient.ListConnectionsWithProductName(ctx, productName, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list connections should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.NotEmpty(t, result.Items, "should have at least one connection")

	// Verify all returned connections are PostgreSQL
	for _, item := range result.Items {
		assert.Equal(t, e2eshared.DBTypePostgreSQL, item.Type, "all items should be PostgreSQL")
	}

	t.Logf("Product-scoped list returned %d PostgreSQL connections", len(result.Items))
}

// TestListConnections_CombinedFilters_Success verifies that connections created under
// a single product with different types and hosts can be distinguished client-side.
func TestListConnections_CombinedFilters_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Use a unique host to distinguish connections within the product
	uniqueHost := fmt.Sprintf("combined-filter-%s.local", uuid.New().String()[:8])

	productName := e2eshared.GenerateProductName()

	// Create 2 PostgreSQL connections with the unique host
	pgConnIDs := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		connInput := e2eshared.ConnectionInput{
			ConfigName:   fmt.Sprintf("e2e-combined-pg-%s-%d", uuid.New().String()[:8], i),
			Type:         e2eshared.DBTypePostgreSQL,
			Host:         uniqueHost,
			Port:         pgPort,
			DatabaseName: "testdb",
			Username:     "testuser",
			Password:     "testpass",
		}

		conn, createErr := apiClient.CreateConnection(ctx, productName, connInput)
		require.NoError(t, createErr, "create PG connection %d", i)
		pgConnIDs = append(pgConnIDs, conn.ID)
	}

	// Create 1 MongoDB connection with the same unique host (different type)
	mongoConn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-combined-mongo-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypeMongoDB,
		Host:         uniqueHost,
		Port:         27017,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create MongoDB connection")

	// Create 1 PostgreSQL connection with a different host (same type)
	otherHostConn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-combined-other-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create other-host connection")

	t.Cleanup(func() {
		for _, id := range pgConnIDs {
			_ = apiClient.DeleteConnection(context.Background(), id)
		}
		_ = apiClient.DeleteConnection(context.Background(), mongoConn.ID)
		_ = apiClient.DeleteConnection(context.Background(), otherHostConn.ID)
	})

	// List all connections for this product
	allResult, err := apiClient.ListConnectionsWithProductName(ctx, productName, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list all connections for product")

	assert.Equal(t, 4, len(allResult.Items),
		"product should have 4 total connections")

	// Client-side filter: count PostgreSQL connections with uniqueHost
	pgUniqueHostCount := 0
	for _, item := range allResult.Items {
		if item.Type == e2eshared.DBTypePostgreSQL && item.Host == uniqueHost {
			pgUniqueHostCount++
		}
	}
	assert.Equal(t, 2, pgUniqueHostCount,
		"should have exactly 2 PostgreSQL connections with unique host")

	// Verify MongoDB connection is also present
	mongoCount := 0
	for _, item := range allResult.Items {
		if item.Type == e2eshared.DBTypeMongoDB {
			mongoCount++
		}
	}
	assert.Equal(t, 1, mongoCount, "should have exactly 1 MongoDB connection")

	t.Logf("Product has %d total connections: %d PG/uniqueHost, %d MongoDB",
		len(allResult.Items), pgUniqueHostCount, mongoCount)
}

// TestListConnections_Pagination_Success verifies that pagination works correctly
// for listing connections.
// This test uses product-based isolation to ensure deterministic pagination results
// independent of other parallel tests.
func TestListConnections_Pagination_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	// Create 5 connections under this product
	prefix := fmt.Sprintf("e2e-page-%s", uuid.New().String()[:8])
	createdIDs := make([]string, 0, 5)

	for i := 0; i < 5; i++ {
		connInput := e2eshared.ConnectionInput{
			ConfigName:   fmt.Sprintf("%s-%d", prefix, i),
			Type:         e2eshared.DBTypePostgreSQL,
			Host:         "pagination-test.local",
			Port:         5432,
			DatabaseName: "testdb",
			Username:     "testuser",
			Password:     "testpass",
		}

		conn, err := apiClient.CreateConnection(ctx, productName, connInput)
		require.NoError(t, err, "create connection %d", i)
		createdIDs = append(createdIDs, conn.ID)
	}

	t.Cleanup(func() {
		for _, id := range createdIDs {
			_ = apiClient.DeleteConnection(context.Background(), id)
		}
	})

	// Get first page with limit 2, scoped by product
	page1, err := apiClient.ListConnectionsWithProductName(ctx, productName, e2eshared.ListConnectionsParams{
		Page:  1,
		Limit: 2,
	})
	require.NoError(t, err, "list page 1 should succeed")

	assert.NotNil(t, page1, "page 1 result should not be nil")
	assert.Equal(t, 2, len(page1.Items), "page 1 should have exactly 2 items")

	// Get second page
	page2, err := apiClient.ListConnectionsWithProductName(ctx, productName, e2eshared.ListConnectionsParams{
		Page:  2,
		Limit: 2,
	})
	require.NoError(t, err, "list page 2 should succeed")

	assert.NotNil(t, page2, "page 2 result should not be nil")
	assert.Equal(t, 2, len(page2.Items), "page 2 should have exactly 2 items")

	// Verify no overlap between pages
	page1IDs := make(map[string]bool)
	for _, item := range page1.Items {
		page1IDs[item.ID] = true
	}

	for _, item := range page2.Items {
		assert.False(t, page1IDs[item.ID], "page 2 should not contain items from page 1")
	}

	// Get third page (should have 1 item)
	page3, err := apiClient.ListConnectionsWithProductName(ctx, productName, e2eshared.ListConnectionsParams{
		Page:  3,
		Limit: 2,
	})
	require.NoError(t, err, "list page 3 should succeed")
	assert.Equal(t, 1, len(page3.Items), "page 3 should have exactly 1 item")

	t.Logf("Page 1: %d items, Page 2: %d items, Page 3: %d items",
		len(page1.Items), len(page2.Items), len(page3.Items))
}
