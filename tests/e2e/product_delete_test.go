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

// TestDeleteProduct_Success verifies that an existing product
// can be deleted successfully and is no longer retrievable.
func TestDeleteProduct_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-del-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Delete Test %s", uniqueCode),
		Description: "E2E test product for delete success",
	}

	product, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	// Delete the product
	err = apiClient.DeleteProduct(ctx, product.ID)
	require.NoError(t, err, "delete product")

	// Verify the product is gone
	e2eshared.AssertProductNotFound(t, apiClient, product.ID)

	t.Logf("Successfully deleted product: id=%s", product.ID)
}

// TestDeleteProduct_NotFound_404 verifies that deleting a non-existent
// product returns a 404 Not Found error.
func TestDeleteProduct_NotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentID := uuid.New().String()

	resp, err := apiClient.DeleteProductRaw(ctx, nonExistentID)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 404, "")
	t.Logf("Delete of non-existent product correctly returned 404")
}

// TestDeleteProduct_Idempotent verifies that deleting the same product
// twice returns appropriate responses: first succeeds, second returns 404.
func TestDeleteProduct_Idempotent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-del-idem-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Idempotent Delete Test %s", uniqueCode),
		Description: "E2E test product for idempotent delete",
	}

	product, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	// First delete should succeed
	err = apiClient.DeleteProduct(ctx, product.ID)
	require.NoError(t, err, "first delete should succeed")

	// Second delete should return 404
	resp, err := apiClient.DeleteProductRaw(ctx, product.ID)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 404, resp.StatusCode(), "second delete should return 404")

	t.Logf("Double delete handled correctly: first succeeded, second returned 404")
}

// TestDeleteProduct_WithActiveConnections_Conflict verifies that deleting a product
// that has active connections returns a 409 Conflict error.
func TestDeleteProduct_WithActiveConnections_Conflict(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create a product
	uniqueCode := fmt.Sprintf("e2e-del-conn-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Delete With Connections %s", uniqueCode),
		Description: "E2E test product with active connections",
	}

	product, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	// Create a connection assigned to this product
	connName := fmt.Sprintf("e2e-conn-del-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   connName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	// Attempt to delete the product with an active connection - should fail with 409
	resp, err := apiClient.DeleteProductRaw(ctx, product.ID)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 409, "")
	t.Logf("Delete of product with active connections correctly returned %d", resp.StatusCode())

	// Cleanup: delete connection first, then product
	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
		_ = apiClient.DeleteProduct(context.Background(), product.ID)
	})
}
