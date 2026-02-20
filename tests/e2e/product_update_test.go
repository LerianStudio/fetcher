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

// TestUpdateProduct_Name_Success verifies that a product's name can be updated
// while other fields remain unchanged.
func TestUpdateProduct_Name_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-upd-name-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Original Name %s", uniqueCode),
		Description: "Original description",
	}

	created, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	t.Cleanup(func() {
		_ = apiClient.DeleteProduct(context.Background(), created.ID)
	})

	// Update only the name
	newName := fmt.Sprintf("Updated Name %s", uuid.New().String()[:8])
	updateInput := e2eshared.ProductUpdateInput{
		Name: &newName,
	}

	updated, err := apiClient.UpdateProduct(ctx, created.ID, updateInput)
	require.NoError(t, err, "update product name")

	// Verify name changed
	assert.Equal(t, newName, updated.Name, "name should be updated")

	// Verify code is unchanged (immutable)
	assert.Equal(t, uniqueCode, updated.Code, "code should not change")

	// Verify updatedAt changed
	assert.NotEqual(t, created.UpdatedAt, updated.UpdatedAt, "updatedAt should change")

	t.Logf("Updated product name: id=%s, old=%s, new=%s", updated.ID, input.Name, updated.Name)
}

// TestUpdateProduct_Description_Success verifies that a product's description can be updated
// while other fields remain unchanged.
func TestUpdateProduct_Description_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-upd-desc-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Description Test %s", uniqueCode),
		Description: "Original description for update test",
	}

	created, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	t.Cleanup(func() {
		_ = apiClient.DeleteProduct(context.Background(), created.ID)
	})

	// Update only the description
	newDescription := "Updated description for E2E test"
	updateInput := e2eshared.ProductUpdateInput{
		Description: &newDescription,
	}

	updated, err := apiClient.UpdateProduct(ctx, created.ID, updateInput)
	require.NoError(t, err, "update product description")

	// Verify description changed
	assert.Equal(t, newDescription, updated.Description, "description should be updated")

	// Verify other fields are unchanged
	assert.Equal(t, uniqueCode, updated.Code, "code should not change")
	assert.Equal(t, input.Name, updated.Name, "name should not change")

	t.Logf("Updated product description: id=%s", updated.ID)
}

// TestUpdateProduct_Metadata_Success verifies that a product's metadata can be updated
// and that the new metadata replaces the old one.
func TestUpdateProduct_Metadata_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-upd-meta-%s", uuid.New().String()[:8])
	originalMetadata := map[string]any{
		"version": "1",
		"team":    "original",
	}
	input := e2eshared.ProductInput{
		Code:     uniqueCode,
		Name:     fmt.Sprintf("Metadata Test %s", uniqueCode),
		Metadata: &originalMetadata,
	}

	created, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	t.Cleanup(func() {
		_ = apiClient.DeleteProduct(context.Background(), created.ID)
	})

	// Update metadata
	newMetadata := map[string]any{
		"version": "2",
		"team":    "updated",
		"newKey":  "newValue",
	}
	updateInput := e2eshared.ProductUpdateInput{
		Metadata: &newMetadata,
	}

	updated, err := apiClient.UpdateProduct(ctx, created.ID, updateInput)
	require.NoError(t, err, "update product metadata")

	// Verify metadata is replaced
	require.NotNil(t, updated.Metadata, "metadata should be present")
	assert.Equal(t, "2", (*updated.Metadata)["version"], "metadata version should be updated")
	assert.Equal(t, "updated", (*updated.Metadata)["team"], "metadata team should be updated")
	assert.Equal(t, "newValue", (*updated.Metadata)["newKey"], "metadata newKey should be present")

	t.Logf("Updated product metadata: id=%s", updated.ID)
}

// TestUpdateProduct_MultipleFields_Success verifies that multiple fields
// (name, description, metadata) can be updated in a single PATCH request.
func TestUpdateProduct_MultipleFields_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-upd-multi-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Multi Update Test %s", uniqueCode),
		Description: "Original description",
	}

	created, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	t.Cleanup(func() {
		_ = apiClient.DeleteProduct(context.Background(), created.ID)
	})

	// Update name, description, and metadata in a single request
	newName := fmt.Sprintf("Multi Updated %s", uuid.New().String()[:8])
	newDescription := "Updated description for multi-field test"
	newMetadata := map[string]any{
		"updated": "true",
		"batch":   "multi-field",
	}
	updateInput := e2eshared.ProductUpdateInput{
		Name:        &newName,
		Description: &newDescription,
		Metadata:    &newMetadata,
	}

	updated, err := apiClient.UpdateProduct(ctx, created.ID, updateInput)
	require.NoError(t, err, "update product multiple fields")

	// Verify all fields changed
	assert.Equal(t, newName, updated.Name, "name should be updated")
	assert.Equal(t, newDescription, updated.Description, "description should be updated")
	require.NotNil(t, updated.Metadata, "metadata should be present")
	assert.Equal(t, "true", (*updated.Metadata)["updated"], "metadata updated key should match")
	assert.Equal(t, "multi-field", (*updated.Metadata)["batch"], "metadata batch key should match")

	// Verify code is unchanged (immutable)
	assert.Equal(t, uniqueCode, updated.Code, "code should not change")

	t.Logf("Updated product multiple fields: id=%s", updated.ID)
}

// TestUpdateProduct_NotFound_404 verifies that updating a non-existent
// product returns a 404 Not Found error.
func TestUpdateProduct_NotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentID := uuid.New().String()
	newName := "should-not-exist"
	updateInput := e2eshared.ProductUpdateInput{
		Name: &newName,
	}

	resp, err := apiClient.UpdateProductRaw(ctx, nonExistentID, updateInput)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 404, "")
	t.Logf("Update of non-existent product correctly returned 404")
}

// TestUpdateProduct_EmptyBody_BadRequest verifies that an empty update
// request is rejected with a 400 Bad Request error.
func TestUpdateProduct_EmptyBody_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Create a product to update
	uniqueCode := fmt.Sprintf("e2e-upd-empty-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code: uniqueCode,
		Name: fmt.Sprintf("Empty Update Test %s", uniqueCode),
	}

	created, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	t.Cleanup(func() {
		_ = apiClient.DeleteProduct(context.Background(), created.ID)
	})

	// Send empty update - should be rejected
	updateInput := e2eshared.ProductUpdateInput{}

	resp, err := apiClient.UpdateProductRaw(ctx, created.ID, updateInput)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 400, "")
	t.Logf("Empty update correctly rejected with status %d", resp.StatusCode())
}
