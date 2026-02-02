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
// returns an empty result when no connections exist for the organization.
func TestListConnections_Empty_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Use a filter that should return no results
	params := e2eshared.ListConnectionsParams{
		Host: fmt.Sprintf("nonexistent-host-%s", uuid.New().String()),
	}

	result, err := apiClient.ListConnections(ctx, params)
	require.NoError(t, err, "list connections should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.Empty(t, result.Items, "items should be empty for non-matching filter")

	t.Logf("List with non-matching filter returned %d items", len(result.Items))
}

// TestListConnections_WithResults_Success verifies that listing connections
// returns the created connections with correct data.
func TestListConnections_WithResults_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

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

		conn, err := apiClient.CreateConnection(ctx, connInput)
		require.NoError(t, err, "create connection %d", i)
		createdIDs = append(createdIDs, conn.ID)
	}

	t.Cleanup(func() {
		for _, id := range createdIDs {
			_ = apiClient.DeleteConnection(context.Background(), id)
		}
	})

	// List all connections
	result, err := apiClient.ListConnections(ctx, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list connections should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.GreaterOrEqual(t, len(result.Items), 3, "should have at least 3 connections")

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
// with a type filter returns only matching connections.
func TestListConnections_FilterByType_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

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

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Filter by PostgreSQL type
	params := e2eshared.ListConnectionsParams{
		Type: e2eshared.DBTypePostgreSQL,
	}

	result, err := apiClient.ListConnections(ctx, params)
	require.NoError(t, err, "list connections with type filter should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.NotEmpty(t, result.Items, "should have at least one PostgreSQL connection")

	// Verify all returned connections are PostgreSQL
	for _, item := range result.Items {
		assert.Equal(t, e2eshared.DBTypePostgreSQL, item.Type, "all items should be PostgreSQL")
	}

	t.Logf("Type filter returned %d PostgreSQL connections", len(result.Items))
}

// TestListConnections_Pagination_Success verifies that pagination works correctly
// for listing connections.
// This test uses a unique host filter to isolate its connections from other parallel tests,
// preventing race conditions that could cause flaky pagination results.
func TestListConnections_Pagination_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Use a unique host to isolate this test from other parallel tests.
	// This prevents race conditions where other tests create/delete connections
	// between fetching page 1 and page 2.
	uniqueHost := fmt.Sprintf("pagination-test-%s.local", uuid.New().String()[:8])

	// Create multiple connections with unique host
	prefix := fmt.Sprintf("e2e-page-%s", uuid.New().String()[:8])
	createdIDs := make([]string, 0, 5)

	for i := 0; i < 5; i++ {
		connInput := e2eshared.ConnectionInput{
			ConfigName:   fmt.Sprintf("%s-%d", prefix, i),
			Type:         e2eshared.DBTypePostgreSQL,
			Host:         uniqueHost,
			Port:         5432,
			DatabaseName: "testdb",
			Username:     "testuser",
			Password:     "testpass",
		}

		conn, err := apiClient.CreateConnection(ctx, connInput)
		require.NoError(t, err, "create connection %d", i)
		createdIDs = append(createdIDs, conn.ID)
	}

	t.Cleanup(func() {
		for _, id := range createdIDs {
			_ = apiClient.DeleteConnection(context.Background(), id)
		}
	})

	// Get first page with limit 2, filtered by unique host
	page1Params := e2eshared.ListConnectionsParams{
		Page:  1,
		Limit: 2,
		Host:  uniqueHost,
	}

	page1, err := apiClient.ListConnections(ctx, page1Params)
	require.NoError(t, err, "list page 1 should succeed")

	assert.NotNil(t, page1, "page 1 result should not be nil")
	assert.Equal(t, 2, len(page1.Items), "page 1 should have exactly 2 items")

	// Get second page
	page2Params := e2eshared.ListConnectionsParams{
		Page:  2,
		Limit: 2,
		Host:  uniqueHost,
	}

	page2, err := apiClient.ListConnections(ctx, page2Params)
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
	page3Params := e2eshared.ListConnectionsParams{
		Page:  3,
		Limit: 2,
		Host:  uniqueHost,
	}

	page3, err := apiClient.ListConnections(ctx, page3Params)
	require.NoError(t, err, "list page 3 should succeed")
	assert.Equal(t, 1, len(page3.Items), "page 3 should have exactly 1 item")

	t.Logf("Page 1: %d items, Page 2: %d items, Page 3: %d items",
		len(page1.Items), len(page2.Items), len(page3.Items))
}
