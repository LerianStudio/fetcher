//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"strings"
	"testing"

	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateProduct_Success verifies that a product can be created successfully
// with a unique code, name, and description.
func TestCreateProduct_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-prod-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Product %s", uniqueCode),
		Description: "E2E test product for create success",
	}

	product, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product")

	t.Cleanup(func() {
		_ = apiClient.DeleteProduct(context.Background(), product.ID)
	})

	// Verify server-generated fields
	e2eshared.AssertValidUUID(t, product.ID)

	// Verify input fields are preserved
	assert.Equal(t, uniqueCode, product.Code, "code should match")
	assert.Equal(t, input.Name, product.Name, "name should match")
	assert.Equal(t, input.Description, product.Description, "description should match")

	// Verify timestamps are populated
	assert.NotEmpty(t, product.CreatedAt, "createdAt should not be empty")
	assert.NotEmpty(t, product.UpdatedAt, "updatedAt should not be empty")

	t.Logf("Created product: id=%s, code=%s", product.ID, product.Code)
}

// TestCreateProduct_WithMetadata_Success verifies that a product can be created
// with metadata and that the metadata is preserved in the response.
func TestCreateProduct_WithMetadata_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-meta-%s", uuid.New().String()[:8])
	metadata := map[string]any{
		"environment": "test",
		"team":        "platform",
	}
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Product With Metadata %s", uniqueCode),
		Description: "E2E test product with metadata",
		Metadata:    &metadata,
	}

	product, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create product with metadata")

	t.Cleanup(func() {
		_ = apiClient.DeleteProduct(context.Background(), product.ID)
	})

	// Verify metadata is preserved
	require.NotNil(t, product.Metadata, "metadata should be present")
	assert.Equal(t, "test", (*product.Metadata)["environment"], "metadata environment should match")
	assert.Equal(t, "platform", (*product.Metadata)["team"], "metadata team should match")

	t.Logf("Created product with metadata: id=%s, code=%s", product.ID, product.Code)
}

// TestCreateProduct_DuplicateCode_Conflict verifies that creating a product
// with a duplicate code returns a 409 Conflict error.
func TestCreateProduct_DuplicateCode_Conflict(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueCode := fmt.Sprintf("e2e-dup-%s", uuid.New().String()[:8])
	input := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Duplicate Test %s", uniqueCode),
		Description: "First product",
	}

	// Create the first product
	product, err := apiClient.CreateProduct(ctx, input)
	require.NoError(t, err, "create first product")

	t.Cleanup(func() {
		_ = apiClient.DeleteProduct(context.Background(), product.ID)
	})

	// Try to create a second product with the same code
	duplicateInput := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        "Different Name",
		Description: "Second product with same code",
	}

	resp, err := apiClient.CreateProductRaw(ctx, duplicateInput)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 409, "")
	t.Logf("Duplicate code correctly rejected with status %d", resp.StatusCode())
}

// TestCreateProduct_MissingRequiredFields_BadRequest verifies that creating a product
// with missing required fields returns a 400 Bad Request error.
func TestCreateProduct_MissingRequiredFields_BadRequest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input e2eshared.ProductInput
	}{
		{
			name: "missing_code",
			input: e2eshared.ProductInput{
				Code: "",
				Name: "Valid Name",
			},
		},
		{
			name: "missing_name",
			input: e2eshared.ProductInput{
				Code: fmt.Sprintf("e2e-noname-%s", uuid.New().String()[:8]),
				Name: "",
			},
		},
		{
			name: "empty_body",
			input: e2eshared.ProductInput{
				Code: "",
				Name: "",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			subCtx, subCancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer subCancel()

			resp, err := apiClient.CreateProductRaw(subCtx, tc.input)
			require.NoError(t, err, "request should succeed")

			e2eshared.AssertAPIError(t, resp, 400, "")
			t.Logf("Missing field correctly rejected with status %d", resp.StatusCode())
		})
	}
}

// TestCreateProduct_InvalidCode_BadRequest verifies that creating a product
// with an invalid code format returns a 400 Bad Request error.
// Code rules: ^[a-z0-9]+(-[a-z0-9]+)*$, 2-50 chars.
func TestCreateProduct_InvalidCode_BadRequest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		code string
	}{
		// Note: uppercase codes (e.g. "ABC-DEF") are NOT invalid — the API
		// normalizes them to lowercase via strings.ToLower() before validation.
		{
			name: "with_spaces",
			code: "my product",
		},
		{
			name: "special_chars",
			code: "my@product!",
		},
		{
			name: "too_short_1char",
			code: "a",
		},
		{
			name: "too_long_51chars",
			code: strings.Repeat("a", 51),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			subCtx, subCancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer subCancel()

			input := e2eshared.ProductInput{
				Code: tc.code,
				Name: "Valid Name",
			}

			resp, err := apiClient.CreateProductRaw(subCtx, input)
			require.NoError(t, err, "request should succeed")

			e2eshared.AssertAPIError(t, resp, 400, "")
			t.Logf("Invalid code %q correctly rejected with status %d", tc.code, resp.StatusCode())
		})
	}
}
