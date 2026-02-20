//go:build e2e

package extraction

import (
	"context"
	"encoding/json"
	"fmt"
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

	// Create product A and a connection assigned to it
	productA := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-assign-conflict-%s", uuid.New().String()[:8])
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
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Create product B
	productB := e2eshared.CreateTestProduct(t, apiClient, ctx)

	// Attempt to assign the connection (already assigned to product A) to product B
	resp, err := apiClient.AssignConnectionRaw(ctx, conn.ID, e2eshared.AssignConnectionInput{
		ProductID: productB.ID,
	})
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 409, resp.StatusCode(), "should return 409 Conflict for already-assigned connection")
	t.Logf("Already-assigned connection correctly rejected with status %d", resp.StatusCode())
}

// TestAssignConnection_ProductNotFound_404 verifies that assigning a connection
// to a non-existent product returns a 404 Not Found error.
func TestAssignConnection_ProductNotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-assign-no-product-%s", uuid.New().String()[:8])
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

	// Attempt to assign to a non-existent product
	resp, err := apiClient.AssignConnectionRaw(ctx, conn.ID, e2eshared.AssignConnectionInput{
		ProductID: uuid.New().String(),
	})
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 404, resp.StatusCode(), "should return 404 for non-existent product")
	t.Logf("Non-existent product correctly returned status %d", resp.StatusCode())
}

// TestAssignConnection_ConnectionNotFound_404 verifies that assigning a non-existent
// connection returns a 404 Not Found error.
func TestAssignConnection_ConnectionNotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	// Attempt to assign a non-existent connection
	resp, err := apiClient.AssignConnectionRaw(ctx, uuid.New().String(), e2eshared.AssignConnectionInput{
		ProductID: product.ID,
	})
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 404, resp.StatusCode(), "should return 404 for non-existent connection")
	t.Logf("Non-existent connection correctly returned status %d", resp.StatusCode())
}

// TestAssignConnection_InvalidProductID_BadRequest verifies that assigning a connection
// with an invalid (non-UUID) product ID returns a 400 Bad Request error.
func TestAssignConnection_InvalidProductID_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-assign-invalid-%s", uuid.New().String()[:8])
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

	// Attempt to assign with invalid product ID
	resp, err := apiClient.AssignConnectionRaw(ctx, conn.ID, e2eshared.AssignConnectionInput{
		ProductID: "not-a-uuid",
	})
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(), "should return 400 Bad Request for invalid product ID")
	t.Logf("Invalid product ID correctly rejected with status %d", resp.StatusCode())
}

// TestAssignConnection_Success verifies that an unassigned connection can be assigned
// to a product. If the API requires productId at creation time and rejects connections
// without it, the test is skipped as unassigned connections cannot be created.
func TestAssignConnection_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Attempt to create connection WITHOUT productId (empty string)
	uniqueName := fmt.Sprintf("e2e-assign-success-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	// Check if the API rejects connection creation without productId
	resp, err := apiClient.CreateConnectionRaw(ctx, connInput)
	require.NoError(t, err, "request should succeed")

	if resp.StatusCode() == 400 {
		t.Skip("API requires productId, cannot create unassigned connections")
	}

	require.True(t, resp.StatusCode() == 200 || resp.StatusCode() == 201,
		"create connection should return 200 or 201, got %d: %s", resp.StatusCode(), resp.String())

	// API accepts connections without productId; create one using the typed method
	unassignedName := fmt.Sprintf("e2e-assign-unassigned-%s", uuid.New().String()[:8])
	unassignedInput := e2eshared.ConnectionInput{
		ConfigName:   unassignedName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, unassignedInput)
	require.NoError(t, err, "create unassigned connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Create a product to assign to
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	// Assign the connection to the product
	assigned, err := apiClient.AssignConnection(ctx, conn.ID, e2eshared.AssignConnectionInput{
		ProductID: product.ID,
	})
	require.NoError(t, err, "assign connection should succeed")

	assert.Equal(t, product.ID, assigned.ProductID, "assigned connection should have product ID set")
	assert.Equal(t, conn.ID, assigned.ID, "connection ID should not change")

	t.Logf("Successfully assigned connection %s to product %s", assigned.ID, assigned.ProductID)
}

// TestListUnassignedConnections_Pagination_Success verifies that pagination works correctly
// for the unassigned connections endpoint. If the API requires productId at creation time,
// the test is skipped as unassigned connections cannot be created.
func TestListUnassignedConnections_Pagination_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Check if the API allows unassigned connections
	probeInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-unassign-probe-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	probeResp, err := apiClient.CreateConnectionRaw(ctx, probeInput)
	require.NoError(t, err, "probe request should succeed")

	if probeResp.StatusCode() == 400 {
		t.Skip("API requires productId, cannot create unassigned connections for pagination test")
	}

	// Clean up probe connection
	if probeResp.StatusCode() == 200 || probeResp.StatusCode() == 201 {
		var probeConn e2eshared.ConnectionResponse
		if jsonErr := json.Unmarshal(probeResp.Body(), &probeConn); jsonErr == nil && probeConn.ID != "" {
			t.Cleanup(func() {
				_ = apiClient.DeleteConnection(context.Background(), probeConn.ID)
			})
		}
	}

	// Create 3 unassigned connections
	createdIDs := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		connInput := e2eshared.ConnectionInput{
			ConfigName:   fmt.Sprintf("e2e-unassign-page-%s-%d", uuid.New().String()[:8], i),
			Type:         e2eshared.DBTypePostgreSQL,
			Host:         pgHost,
			Port:         pgPort,
			DatabaseName: "testdb",
			Username:     "testuser",
			Password:     "testpass",
		}

		conn, createErr := apiClient.CreateConnection(ctx, connInput)
		require.NoError(t, createErr, "create unassigned connection %d", i)
		createdIDs = append(createdIDs, conn.ID)

		t.Cleanup(func() {
			_ = apiClient.DeleteConnection(context.Background(), conn.ID)
		})
	}

	// Paginate through unassigned connections with limit 2
	page1, err := apiClient.ListUnassignedConnections(ctx, e2eshared.ListConnectionsParams{
		Page:  1,
		Limit: 2,
	})
	require.NoError(t, err, "list unassigned connections page 1 should succeed")
	assert.NotNil(t, page1, "page 1 result should not be nil")
	assert.NotNil(t, page1.Items, "page 1 items should not be nil")
	assert.GreaterOrEqual(t, page1.Page, 1, "page should be present")
	assert.Greater(t, page1.Limit, 0, "limit should be present")

	page2, err := apiClient.ListUnassignedConnections(ctx, e2eshared.ListConnectionsParams{
		Page:  2,
		Limit: 2,
	})
	require.NoError(t, err, "list unassigned connections page 2 should succeed")
	assert.NotNil(t, page2, "page 2 result should not be nil")
	assert.NotNil(t, page2.Items, "page 2 items should not be nil")

	t.Logf("Unassigned pagination: page1=%d items, page2=%d items",
		len(page1.Items), len(page2.Items))
}

// TestListUnassignedConnections_AfterAssignment_Removed verifies that once a connection
// is assigned to a product, it no longer appears in the unassigned connections list.
// If the API requires productId at creation time, the test is skipped.
func TestListUnassignedConnections_AfterAssignment_Removed(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Check if the API allows unassigned connections
	probeInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-unassign-rm-probe-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	probeResp, err := apiClient.CreateConnectionRaw(ctx, probeInput)
	require.NoError(t, err, "probe request should succeed")

	if probeResp.StatusCode() == 400 {
		t.Skip("API requires productId, cannot create unassigned connections for assignment removal test")
	}

	// Clean up probe connection
	if probeResp.StatusCode() == 200 || probeResp.StatusCode() == 201 {
		var probeConn e2eshared.ConnectionResponse
		if jsonErr := json.Unmarshal(probeResp.Body(), &probeConn); jsonErr == nil && probeConn.ID != "" {
			t.Cleanup(func() {
				_ = apiClient.DeleteConnection(context.Background(), probeConn.ID)
			})
		}
	}

	// Create an unassigned connection
	connInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-unassign-rm-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create unassigned connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Verify the connection appears in the unassigned list
	unassignedBefore, err := apiClient.ListUnassignedConnections(ctx, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list unassigned connections should succeed")

	foundBefore := false
	for _, item := range unassignedBefore.Items {
		if item.ID == conn.ID {
			foundBefore = true

			break
		}
	}

	assert.True(t, foundBefore, "newly created unassigned connection should appear in unassigned list")

	// Assign the connection to a product
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	_, err = apiClient.AssignConnection(ctx, conn.ID, e2eshared.AssignConnectionInput{
		ProductID: product.ID,
	})
	require.NoError(t, err, "assign connection should succeed")

	// Verify the connection no longer appears in the unassigned list
	unassignedAfter, err := apiClient.ListUnassignedConnections(ctx, e2eshared.ListConnectionsParams{})
	require.NoError(t, err, "list unassigned connections after assignment should succeed")

	foundAfter := false
	for _, item := range unassignedAfter.Items {
		if item.ID == conn.ID {
			foundAfter = true

			break
		}
	}

	assert.False(t, foundAfter, "assigned connection should no longer appear in unassigned list")

	t.Logf("Connection %s removed from unassigned list after assignment to product %s", conn.ID, product.ID)
}
