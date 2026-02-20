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

// TestGetProduct_Exists_Success verifies that an existing product
// can be retrieved by ID with all fields populated correctly.
func TestGetProduct_Exists_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-get-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Get Test Product %s", uniqueCode),
		Description: "E2E test product for get success",
	}

	created, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	t.Cleanup(func() {
		_ = apiClient.DeleteProduct(context.Background(), created.ID)
	})

	// Get the product
	retrieved, err := apiClient.GetProduct(ctx, created.ID)
	require.NoError(t, err, "get product")

	// Verify all fields match
	assert.Equal(t, created.ID, retrieved.ID, "ID should match")
	assert.Equal(t, uniqueCode, retrieved.Code, "code should match")
	assert.Equal(t, input.Name, retrieved.Name, "name should match")
	assert.Equal(t, input.Description, retrieved.Description, "description should match")
	assert.NotEmpty(t, retrieved.CreatedAt, "createdAt should be set")
	assert.NotEmpty(t, retrieved.UpdatedAt, "updatedAt should be set")

	t.Logf("Successfully retrieved product: id=%s, code=%s", retrieved.ID, retrieved.Code)
}

// TestGetProduct_NotFound_404 verifies that requesting a non-existent
// product returns a 404 Not Found error.
func TestGetProduct_NotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentID := uuid.New().String()

	resp, err := apiClient.GetProductRaw(ctx, nonExistentID)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 404, "")
	t.Logf("Non-existent product correctly returned 404")
}

// TestGetProduct_InvalidID_BadRequest verifies that requesting a product
// with an invalid ID format returns a 400 Bad Request or 404 error.
func TestGetProduct_InvalidID_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	invalidID := "not-a-uuid"

	resp, err := apiClient.GetProductRaw(ctx, invalidID)
	require.NoError(t, err, "request should succeed")

	// API should return either 400 (bad request) or 404 (not found)
	// depending on validation strategy
	assert.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 404,
		"should return 400 or 404, got %d", resp.StatusCode())
	t.Logf("Invalid ID %q returned status %d", invalidID, resp.StatusCode())
}
