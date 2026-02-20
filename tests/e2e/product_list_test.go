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

// TestListProducts_WithResults_Success verifies that listing products
// returns the created products with correct data.
func TestListProducts_WithResults_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Create 3 products with unique prefix
	prefix := fmt.Sprintf("e2e-list-%s", uuid.New().String()[:8])
	createdIDs := make([]string, 0, 3)

	for i := 0; i < 3; i++ {
		input := e2eshared.ProductInput{
			Code:        fmt.Sprintf("%s-%d", prefix, i),
			Name:        fmt.Sprintf("List Test Product %s-%d", prefix, i),
			Description: "E2E test product for list",
		}

		product, err := apiClient.CreateProduct(ctx, input)
		require.NoError(t, err, "create product %d", i)
		createdIDs = append(createdIDs, product.ID)
	}

	t.Cleanup(func() {
		for _, id := range createdIDs {
			_ = apiClient.DeleteProduct(context.Background(), id)
		}
	})

	// List all products with high limit to avoid missing items due to parallel tests
	result, err := apiClient.ListProducts(ctx, e2eshared.ListProductsParams{Limit: 100})
	require.NoError(t, err, "list products should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.GreaterOrEqual(t, len(result.Items), 3, "should have at least 3 products")

	// Verify all created products are in the list
	foundCount := 0
	for _, item := range result.Items {
		for _, id := range createdIDs {
			if item.ID == id {
				foundCount++

				break
			}
		}
	}

	assert.Equal(t, 3, foundCount, "should find all 3 created products")

	t.Logf("Listed %d products, found %d created by this test", len(result.Items), foundCount)
}

// TestListProducts_Pagination_Success verifies that pagination works correctly
// for listing products.
func TestListProducts_Pagination_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Create 5 products with unique prefix
	prefix := fmt.Sprintf("e2e-page-%s", uuid.New().String()[:8])
	createdIDs := make([]string, 0, 5)

	for i := 0; i < 5; i++ {
		input := e2eshared.ProductInput{
			Code:        fmt.Sprintf("%s-%d", prefix, i),
			Name:        fmt.Sprintf("Pagination Test Product %s-%d", prefix, i),
			Description: "E2E test product for pagination",
		}

		product, err := apiClient.CreateProduct(ctx, input)
		require.NoError(t, err, "create product %d", i)
		createdIDs = append(createdIDs, product.ID)
	}

	t.Cleanup(func() {
		for _, id := range createdIDs {
			_ = apiClient.DeleteProduct(context.Background(), id)
		}
	})

	// Verify pagination mechanics: page 1 and page 2 return items,
	// and the API respects limit parameter.
	// Note: Offset-based pagination can have duplicates/gaps when other parallel
	// tests create or delete products mid-iteration. We test pagination mechanics
	// (limit respected, pages return data) rather than strict cross-page uniqueness.
	page1, err := apiClient.ListProducts(ctx, e2eshared.ListProductsParams{
		Page:  1,
		Limit: 2,
	})
	require.NoError(t, err, "list page 1 should succeed")
	assert.NotNil(t, page1, "page 1 result should not be nil")
	assert.LessOrEqual(t, len(page1.Items), 2, "page 1 should have at most 2 items")
	assert.NotEmpty(t, page1.Items, "page 1 should have items")

	page2, err := apiClient.ListProducts(ctx, e2eshared.ListProductsParams{
		Page:  2,
		Limit: 2,
	})
	require.NoError(t, err, "list page 2 should succeed")
	assert.NotNil(t, page2, "page 2 result should not be nil")
	assert.NotEmpty(t, page2.Items, "page 2 should have items (we created 5 products)")

	// Verify our created products exist by fetching with high limit
	allResult, err := apiClient.ListProducts(ctx, e2eshared.ListProductsParams{Limit: 100})
	require.NoError(t, err, "list all products should succeed")

	createdIDSet := make(map[string]bool, len(createdIDs))
	for _, id := range createdIDs {
		createdIDSet[id] = true
	}

	foundCount := 0
	for _, item := range allResult.Items {
		if createdIDSet[item.ID] {
			foundCount++
		}
	}

	assert.Equal(t, 5, foundCount, "all 5 created products should exist")

	t.Logf("Pagination verified: page1=%d items, page2=%d items, total products found=%d/5",
		len(page1.Items), len(page2.Items), foundCount)
}

// TestListProducts_StructureValid_Success verifies that the list products response
// has the expected structure with Items array, Page, Limit, and Total fields.
func TestListProducts_StructureValid_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	result, err := apiClient.ListProducts(ctx, e2eshared.ListProductsParams{})
	require.NoError(t, err, "list products should succeed")

	assert.NotNil(t, result, "result should not be nil")
	assert.NotNil(t, result.Items, "items array should not be nil")
	assert.GreaterOrEqual(t, result.Total, 0, "total should be >= 0")
	assert.GreaterOrEqual(t, result.Page, 0, "page should be >= 0")
	assert.GreaterOrEqual(t, result.Limit, 0, "limit should be >= 0")

	t.Logf("List structure valid: items=%d, page=%d, limit=%d, total=%d",
		len(result.Items), result.Page, result.Limit, result.Total)
}
