package model

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"

	"github.com/google/uuid"
)

// ##############################################################################################################################################################################
// NewProduct

// TestNewProduct tests the NewProduct constructor function.
func TestNewProduct(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name        string
		orgID       uuid.UUID
		code        string
		productName string
		description string
		metadata    *map[string]any
		expectError bool
		checkFields func(t *testing.T, p *Product)
	}{
		{
			name:        "valid creation with all fields",
			orgID:       orgID,
			code:        "my-product",
			productName: "My Product",
			description: "A valid product description",
			metadata:    &map[string]any{"env": "prod"},
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.ID == uuid.Nil {
					t.Fatal("expected non-nil UUID for ID")
				}
				if p.OrganizationID != orgID {
					t.Fatalf("expected OrganizationID %s, got %s", orgID, p.OrganizationID)
				}
				if p.Code != "my-product" {
					t.Fatalf("expected Code 'my-product', got %s", p.Code)
				}
				if p.Name != "My Product" {
					t.Fatalf("expected Name 'My Product', got %s", p.Name)
				}
				if p.Description != "A valid product description" {
					t.Fatalf("expected Description 'A valid product description', got %s", p.Description)
				}
				if p.Metadata == nil {
					t.Fatal("expected Metadata to be set")
				}
				if p.CreatedAt.IsZero() {
					t.Fatal("expected CreatedAt to be set")
				}
				if p.UpdatedAt.IsZero() {
					t.Fatal("expected UpdatedAt to be set")
				}
			},
		},
		{
			name:        "valid creation with nil metadata",
			orgID:       orgID,
			code:        "reporter",
			productName: "Reporter",
			description: "",
			metadata:    nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Metadata != nil {
					t.Fatalf("expected Metadata nil, got %v", p.Metadata)
				}
			},
		},
		{
			name:        "valid creation with minimal fields",
			orgID:       orgID,
			code:        "ab",
			productName: "AB",
			description: "",
			metadata:    nil,
			expectError: false,
		},
		{
			name:        "empty code returns error",
			orgID:       orgID,
			code:        "",
			productName: "My Product",
			description: "",
			metadata:    nil,
			expectError: true,
		},
		{
			name:        "empty name returns error",
			orgID:       orgID,
			code:        "my-product",
			productName: "",
			description: "",
			metadata:    nil,
			expectError: true,
		},
		{
			// normalizeFields lowercases code before validation, so "MyProduct" -> "myproduct" which is valid.
			// Uppercase-only codes are normalized before validation in NewProduct/IsValid.
			name:        "uppercase code is normalized to lowercase - no error",
			orgID:       orgID,
			code:        "MyProduct",
			productName: "My Product",
			description: "",
			metadata:    nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Code != "myproduct" {
					t.Fatalf("expected Code 'myproduct' after normalization, got '%s'", p.Code)
				}
			},
		},
		{
			name:        "invalid code format - spaces",
			orgID:       orgID,
			code:        "my product",
			productName: "My Product",
			description: "",
			metadata:    nil,
			expectError: true,
		},
		{
			name:        "code too short - single character",
			orgID:       orgID,
			code:        "a",
			productName: "My Product",
			description: "",
			metadata:    nil,
			expectError: true,
		},
		{
			name:        "code too long - 51 characters",
			orgID:       orgID,
			code:        strings.Repeat("a", 51),
			productName: "My Product",
			description: "",
			metadata:    nil,
			expectError: true,
		},
		{
			name:        "name too long - 101 characters",
			orgID:       orgID,
			code:        "my-product",
			productName: strings.Repeat("a", 101),
			description: "",
			metadata:    nil,
			expectError: true,
		},
		{
			name:        "description too long - 501 characters",
			orgID:       orgID,
			code:        "my-product",
			productName: "My Product",
			description: strings.Repeat("a", 501),
			metadata:    nil,
			expectError: true,
		},
		{
			name:        "code normalization - trims leading/trailing whitespace",
			orgID:       orgID,
			code:        "  my-product  ",
			productName: "My Product",
			description: "",
			metadata:    nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Code != "my-product" {
					t.Fatalf("expected Code 'my-product' after trimming, got '%s'", p.Code)
				}
			},
		},
		{
			name:        "code normalization - lowercases code",
			orgID:       orgID,
			code:        "MY-PRODUCT",
			productName: "My Product",
			description: "",
			metadata:    nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Code != "my-product" {
					t.Fatalf("expected Code 'my-product' after lowercasing, got '%s'", p.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProduct(tt.orgID, tt.code, tt.productName, tt.description, tt.metadata)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if p == nil {
				t.Fatal("expected non-nil product")
			}

			if tt.checkFields != nil {
				tt.checkFields(t, p)
			}
		})
	}
}

// ##############################################################################################################################################################################
// IsValid

// TestProduct_IsValid tests the Product.IsValid method.
func TestProduct_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		product     Product
		expectError bool
		errorField  string
	}{
		{
			name: "valid product",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "My Product",
				Description:    "A valid description",
			},
			expectError: false,
		},
		{
			name: "valid product with metadata",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "reporter",
				Name:           "Reporter",
				Metadata:       &map[string]any{"key": "value"},
			},
			expectError: false,
		},
		{
			name: "valid product with minimal code length - 2 chars",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "ab",
				Name:           "AB",
			},
			expectError: false,
		},
		{
			name: "valid product with max code length - 50 chars",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           strings.Repeat("a", 50),
				Name:           "Name",
			},
			expectError: false,
		},
		{
			name: "missing product ID",
			product: Product{
				ID:             uuid.Nil,
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "My Product",
			},
			expectError: true,
			errorField:  "id",
		},
		{
			name: "missing organization ID",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.Nil,
				Code:           "my-product",
				Name:           "My Product",
			},
			expectError: true,
			errorField:  "organization_id",
		},
		{
			name: "missing code",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "",
				Name:           "My Product",
			},
			expectError: true,
			errorField:  "code",
		},
		{
			name: "missing name",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "",
			},
			expectError: true,
			errorField:  "name",
		},
		{
			// normalizeFields lowercases code before validation, so "MYPRODUCT" -> "myproduct" which is valid.
			// IsValid calls normalizeFields first, so all-uppercase codes do not fail validation.
			name: "uppercase code is normalized to lowercase by IsValid - no error",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "MYPRODUCT",
				Name:           "My Product",
			},
			expectError: false,
		},
		{
			name: "invalid code format - with spaces",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my product",
				Name:           "My Product",
			},
			expectError: true,
			errorField:  "code",
		},
		{
			name: "code too long",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           strings.Repeat("a", 51),
				Name:           "My Product",
			},
			expectError: true,
			errorField:  "code",
		},
		{
			name: "name too long",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           strings.Repeat("a", 101),
			},
			expectError: true,
			errorField:  "name",
		},
		{
			name: "description too long",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "My Product",
				Description:    strings.Repeat("a", 501),
			},
			expectError: true,
			errorField:  "description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.product.IsValid()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				var knownFieldsErr pkg.ValidationKnownFieldsError
				if errors.As(err, &knownFieldsErr) {
					if tt.errorField != "" {
						if _, exists := knownFieldsErr.Fields[tt.errorField]; !exists {
							t.Fatalf("expected error field %s, got fields %v", tt.errorField, knownFieldsErr.Fields)
						}
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ##############################################################################################################################################################################
// normalizeFields

// TestProduct_normalizeFields tests the Product.normalizeFields method.
func TestProduct_normalizeFields(t *testing.T) {
	tests := []struct {
		name         string
		product      Product
		expectedCode string
		expectedName string
		expectedDesc string
	}{
		{
			name: "trims whitespace from all string fields",
			product: Product{
				Code:        "  my-product  ",
				Name:        "  My Product  ",
				Description: "  A description  ",
			},
			expectedCode: "my-product",
			expectedName: "My Product",
			expectedDesc: "A description",
		},
		{
			name: "lowercases code",
			product: Product{
				Code:        "MY-PRODUCT",
				Name:        "My Product",
				Description: "description",
			},
			expectedCode: "my-product",
			expectedName: "My Product",
			expectedDesc: "description",
		},
		{
			name: "lowercases code and trims whitespace together",
			product: Product{
				Code:        "  MY-PRODUCT  ",
				Name:        "  Name  ",
				Description: "  Desc  ",
			},
			expectedCode: "my-product",
			expectedName: "Name",
			expectedDesc: "Desc",
		},
		{
			name: "empty fields remain empty",
			product: Product{
				Code:        "",
				Name:        "",
				Description: "",
			},
			expectedCode: "",
			expectedName: "",
			expectedDesc: "",
		},
		{
			name: "whitespace-only code becomes empty",
			product: Product{
				Code:        "   ",
				Name:        "Name",
				Description: "",
			},
			expectedCode: "",
			expectedName: "Name",
			expectedDesc: "",
		},
		{
			name: "name is not lowercased",
			product: Product{
				Code:        "code",
				Name:        "MY NAME",
				Description: "MY DESCRIPTION",
			},
			expectedCode: "code",
			expectedName: "MY NAME",
			expectedDesc: "MY DESCRIPTION",
		},
		{
			name: "mixed case code is fully lowercased",
			product: Product{
				Code:        "MyMixedProduct",
				Name:        "Name",
				Description: "",
			},
			expectedCode: "mymixedproduct",
			expectedName: "Name",
			expectedDesc: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.product.normalizeFields()

			if tt.product.Code != tt.expectedCode {
				t.Fatalf("expected Code '%s', got '%s'", tt.expectedCode, tt.product.Code)
			}
			if tt.product.Name != tt.expectedName {
				t.Fatalf("expected Name '%s', got '%s'", tt.expectedName, tt.product.Name)
			}
			if tt.product.Description != tt.expectedDesc {
				t.Fatalf("expected Description '%s', got '%s'", tt.expectedDesc, tt.product.Description)
			}
		})
	}
}

// ##############################################################################################################################################################################
// validateRequiredFields

// TestProduct_validateRequiredFields tests the Product.validateRequiredFields method.
func TestProduct_validateRequiredFields(t *testing.T) {
	tests := []struct {
		name           string
		product        Product
		expectFields   []string
		expectNoFields []string
	}{
		{
			name: "all required fields present",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "My Product",
			},
			expectNoFields: []string{"id", "organization_id", "code", "name"},
		},
		{
			name: "missing organization ID",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.Nil,
				Code:           "my-product",
				Name:           "My Product",
			},
			expectFields:   []string{"organization_id"},
			expectNoFields: []string{"id", "code", "name"},
		},
		{
			name: "missing product ID",
			product: Product{
				ID:             uuid.Nil,
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "My Product",
			},
			expectFields:   []string{"id"},
			expectNoFields: []string{"organization_id", "code", "name"},
		},
		{
			name: "missing code",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "",
				Name:           "My Product",
			},
			expectFields:   []string{"code"},
			expectNoFields: []string{"id", "organization_id", "name"},
		},
		{
			name: "missing name",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "",
			},
			expectFields:   []string{"name"},
			expectNoFields: []string{"id", "organization_id", "code"},
		},
		{
			name: "all required fields missing",
			product: Product{
				ID:             uuid.Nil,
				OrganizationID: uuid.Nil,
				Code:           "",
				Name:           "",
			},
			expectFields: []string{"id", "organization_id", "code", "name"},
		},
		{
			name: "description is not a required field",
			product: Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "My Product",
				Description:    "",
			},
			expectNoFields: []string{"description"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.product.validateRequiredFields()

			for _, field := range tt.expectFields {
				if _, exists := result[field]; !exists {
					t.Fatalf("expected field '%s' to be in required fields map, got %v", field, result)
				}
			}

			for _, field := range tt.expectNoFields {
				if _, exists := result[field]; exists {
					t.Fatalf("did not expect field '%s' in required fields map, got %v", field, result)
				}
			}
		})
	}
}

// ##############################################################################################################################################################################
// validateFieldValues

// TestProduct_validateFieldValues tests the Product.validateFieldValues method.
func TestProduct_validateFieldValues(t *testing.T) {
	tests := []struct {
		name           string
		product        Product
		expectFields   []string
		expectNoFields []string
	}{
		{
			name: "valid slug code",
			product: Product{
				Code: "my-product",
				Name: "Name",
			},
			expectNoFields: []string{"code"},
		},
		{
			name: "valid code - alphanumeric only",
			product: Product{
				Code: "myproduct",
				Name: "Name",
			},
			expectNoFields: []string{"code"},
		},
		{
			name: "valid code - numbers and letters",
			product: Product{
				Code: "product123",
				Name: "Name",
			},
			expectNoFields: []string{"code"},
		},
		{
			name: "valid code - multiple hyphen segments",
			product: Product{
				Code: "my-super-product",
				Name: "Name",
			},
			expectNoFields: []string{"code"},
		},
		{
			name: "valid code - exactly 2 characters",
			product: Product{
				Code: "ab",
				Name: "Name",
			},
			expectNoFields: []string{"code"},
		},
		{
			name: "valid code - exactly 50 characters",
			product: Product{
				Code: strings.Repeat("a", 50),
				Name: "Name",
			},
			expectNoFields: []string{"code"},
		},
		{
			name: "invalid code - uppercase letters",
			product: Product{
				Code: "MyProduct",
				Name: "Name",
			},
			expectFields: []string{"code"},
		},
		{
			name: "invalid code - spaces",
			product: Product{
				Code: "my product",
				Name: "Name",
			},
			expectFields: []string{"code"},
		},
		{
			name: "invalid code - leading hyphen",
			product: Product{
				Code: "-my-product",
				Name: "Name",
			},
			expectFields: []string{"code"},
		},
		{
			name: "invalid code - trailing hyphen",
			product: Product{
				Code: "my-product-",
				Name: "Name",
			},
			expectFields: []string{"code"},
		},
		{
			name: "invalid code - consecutive hyphens",
			product: Product{
				Code: "my--product",
				Name: "Name",
			},
			expectFields: []string{"code"},
		},
		{
			name: "invalid code - special characters",
			product: Product{
				Code: "my_product",
				Name: "Name",
			},
			expectFields: []string{"code"},
		},
		{
			name: "invalid code - dots",
			product: Product{
				Code: "my.product",
				Name: "Name",
			},
			expectFields: []string{"code"},
		},
		{
			name: "invalid code - too short (1 char)",
			product: Product{
				Code: "a",
				Name: "Name",
			},
			expectFields: []string{"code"},
		},
		{
			name: "invalid code - too long (51 chars)",
			product: Product{
				Code: strings.Repeat("a", 51),
				Name: "Name",
			},
			expectFields: []string{"code"},
		},
		{
			name: "empty code skips code validation",
			product: Product{
				Code: "",
				Name: "Name",
			},
			expectNoFields: []string{"code"},
		},
		{
			name: "name exactly 100 characters is valid",
			product: Product{
				Code: "my-product",
				Name: strings.Repeat("a", 100),
			},
			expectNoFields: []string{"name"},
		},
		{
			name: "name 101 characters is invalid",
			product: Product{
				Code: "my-product",
				Name: strings.Repeat("a", 101),
			},
			expectFields: []string{"name"},
		},
		{
			name: "description exactly 500 characters is valid",
			product: Product{
				Code:        "my-product",
				Name:        "Name",
				Description: strings.Repeat("a", 500),
			},
			expectNoFields: []string{"description"},
		},
		{
			name: "description 501 characters is invalid",
			product: Product{
				Code:        "my-product",
				Name:        "Name",
				Description: strings.Repeat("a", 501),
			},
			expectFields: []string{"description"},
		},
		{
			name: "all fields valid",
			product: Product{
				Code:        "my-product",
				Name:        "My Product",
				Description: "A valid description",
			},
			expectNoFields: []string{"code", "name", "description"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.product.validateFieldValues()

			for _, field := range tt.expectFields {
				if _, exists := result[field]; !exists {
					t.Fatalf("expected field '%s' to be in invalid fields map, got %v", field, result)
				}
			}

			for _, field := range tt.expectNoFields {
				if _, exists := result[field]; exists {
					t.Fatalf("did not expect field '%s' in invalid fields map, got %v", field, result)
				}
			}
		})
	}
}

// ##############################################################################################################################################################################
// ApplyPatch

// TestProduct_ApplyPatch tests the Product.ApplyPatch method.
func TestProduct_ApplyPatch(t *testing.T) {
	baseProduct := func() *Product {
		return &Product{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			Code:           "original-code",
			Name:           "Original Name",
			Description:    "Original description",
			Metadata:       &map[string]any{"original": "value"},
			CreatedAt:      time.Now().UTC().Add(-1 * time.Hour),
			UpdatedAt:      time.Now().UTC().Add(-1 * time.Hour),
		}
	}

	tests := []struct {
		name        string
		product     *Product
		patchName   *string
		patchDesc   *string
		patchMeta   *map[string]any
		expectError bool
		checkFields func(t *testing.T, p *Product)
	}{
		{
			name:        "all nil values not applied",
			product:     baseProduct(),
			patchName:   nil,
			patchDesc:   nil,
			patchMeta:   nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Name != "Original Name" {
					t.Fatalf("expected Name 'Original Name', got '%s'", p.Name)
				}
				if p.Description != "Original description" {
					t.Fatalf("expected Description 'Original description', got '%s'", p.Description)
				}
				if p.Metadata == nil {
					t.Fatal("expected Metadata to remain set")
				}
				if (*p.Metadata)["original"] != "value" {
					t.Fatalf("expected Metadata['original'] = 'value', got '%v'", (*p.Metadata)["original"])
				}
			},
		},
		{
			name:        "patch name applies non-nil value",
			product:     baseProduct(),
			patchName:   productStrPtr("Updated Name"),
			patchDesc:   nil,
			patchMeta:   nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Name != "Updated Name" {
					t.Fatalf("expected Name 'Updated Name', got '%s'", p.Name)
				}
				if p.Description != "Original description" {
					t.Fatalf("expected Description unchanged, got '%s'", p.Description)
				}
			},
		},
		{
			name:        "patch description applies non-nil value",
			product:     baseProduct(),
			patchName:   nil,
			patchDesc:   productStrPtr("Updated description"),
			patchMeta:   nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Description != "Updated description" {
					t.Fatalf("expected Description 'Updated description', got '%s'", p.Description)
				}
				if p.Name != "Original Name" {
					t.Fatalf("expected Name unchanged, got '%s'", p.Name)
				}
			},
		},
		{
			name:    "patch metadata replaces existing metadata",
			product: baseProduct(),
			patchMeta: &map[string]any{
				"new_key": "new_value",
				"env":     "staging",
			},
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Metadata == nil {
					t.Fatal("expected Metadata to be set")
				}
				if (*p.Metadata)["new_key"] != "new_value" {
					t.Fatalf("expected Metadata['new_key'] = 'new_value', got '%v'", (*p.Metadata)["new_key"])
				}
				if (*p.Metadata)["env"] != "staging" {
					t.Fatalf("expected Metadata['env'] = 'staging', got '%v'", (*p.Metadata)["env"])
				}
			},
		},
		{
			name:        "patch name nil does not clear existing name",
			product:     baseProduct(),
			patchName:   nil,
			patchDesc:   nil,
			patchMeta:   nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Name == "" {
					t.Fatal("expected Name to remain non-empty when nil patch is applied")
				}
			},
		},
		{
			name:        "patch metadata nil does not clear existing metadata",
			product:     baseProduct(),
			patchName:   nil,
			patchDesc:   nil,
			patchMeta:   nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Metadata == nil {
					t.Fatal("expected Metadata to remain set when nil patch is applied")
				}
			},
		},
		{
			name:        "patch all fields at once",
			product:     baseProduct(),
			patchName:   productStrPtr("New Name"),
			patchDesc:   productStrPtr("New description"),
			patchMeta:   &map[string]any{"updated": "yes"},
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Name != "New Name" {
					t.Fatalf("expected Name 'New Name', got '%s'", p.Name)
				}
				if p.Description != "New description" {
					t.Fatalf("expected Description 'New description', got '%s'", p.Description)
				}
				if (*p.Metadata)["updated"] != "yes" {
					t.Fatalf("expected Metadata['updated'] = 'yes', got '%v'", (*p.Metadata)["updated"])
				}
			},
		},
		{
			name:        "update_at is modified after patch",
			product:     baseProduct(),
			patchName:   productStrPtr("New Name"),
			patchDesc:   nil,
			patchMeta:   nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.UpdatedAt.Before(p.CreatedAt) {
					t.Fatalf("expected UpdatedAt to be after CreatedAt, got UpdatedAt=%v CreatedAt=%v", p.UpdatedAt, p.CreatedAt)
				}
			},
		},
		{
			name: "validation triggered after patch - empty name fails",
			product: &Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "Original Name",
				CreatedAt:      time.Now().UTC().Add(-1 * time.Hour),
				UpdatedAt:      time.Now().UTC().Add(-1 * time.Hour),
			},
			patchName:   productStrPtr(""),
			patchDesc:   nil,
			patchMeta:   nil,
			expectError: true,
		},
		{
			name: "validation triggered after patch - name too long fails",
			product: &Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "my-product",
				Name:           "Original Name",
				CreatedAt:      time.Now().UTC().Add(-1 * time.Hour),
				UpdatedAt:      time.Now().UTC().Add(-1 * time.Hour),
			},
			patchName:   productStrPtr(strings.Repeat("a", 101)),
			patchDesc:   nil,
			patchMeta:   nil,
			expectError: true,
		},
		{
			name: "code is immutable - not patched via ApplyPatch",
			product: &Product{
				ID:             uuid.New(),
				OrganizationID: uuid.New(),
				Code:           "original-code",
				Name:           "Name",
				CreatedAt:      time.Now().UTC().Add(-1 * time.Hour),
				UpdatedAt:      time.Now().UTC().Add(-1 * time.Hour),
			},
			patchName:   productStrPtr("New Name"),
			patchDesc:   nil,
			patchMeta:   nil,
			expectError: false,
			checkFields: func(t *testing.T, p *Product) {
				if p.Code != "original-code" {
					t.Fatalf("expected Code to remain 'original-code', got '%s'", p.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.product.ApplyPatch(tt.patchName, tt.patchDesc, tt.patchMeta)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkFields != nil {
				tt.checkFields(t, tt.product)
			}
		})
	}
}

// ##############################################################################################################################################################################
// ProductInput.IsEmpty

// TestProductInput_IsEmpty tests the ProductInput.IsEmpty method.
func TestProductInput_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    *ProductInput
		expected bool
	}{
		{
			name:     "nil input returns true",
			input:    nil,
			expected: true,
		},
		{
			name:     "all empty fields returns true",
			input:    &ProductInput{},
			expected: true,
		},
		{
			name: "has code returns false",
			input: &ProductInput{
				Code: "my-product",
			},
			expected: false,
		},
		{
			name: "has name returns false",
			input: &ProductInput{
				Name: "My Product",
			},
			expected: false,
		},
		{
			name: "has description returns false",
			input: &ProductInput{
				Description: "A description",
			},
			expected: false,
		},
		{
			name: "has metadata returns false",
			input: &ProductInput{
				Metadata: &map[string]any{"key": "value"},
			},
			expected: false,
		},
		{
			name: "all fields populated returns false",
			input: &ProductInput{
				Code:        "my-product",
				Name:        "My Product",
				Description: "A description",
				Metadata:    &map[string]any{"key": "value"},
			},
			expected: false,
		},
		{
			name: "empty strings and nil metadata returns true",
			input: &ProductInput{
				Code:        "",
				Name:        "",
				Description: "",
				Metadata:    nil,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.IsEmpty()
			if result != tt.expected {
				t.Fatalf("expected IsEmpty() = %v, got %v", tt.expected, result)
			}
		})
	}
}

// ##############################################################################################################################################################################
// ProductUpdateInput.IsEmpty

// TestProductUpdateInput_IsEmpty tests the ProductUpdateInput.IsEmpty method.
func TestProductUpdateInput_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    *ProductUpdateInput
		expected bool
	}{
		{
			name:     "nil input returns true",
			input:    nil,
			expected: true,
		},
		{
			name:     "all nil fields returns true",
			input:    &ProductUpdateInput{},
			expected: true,
		},
		{
			name: "has name returns false",
			input: &ProductUpdateInput{
				Name: productStrPtr("My Product"),
			},
			expected: false,
		},
		{
			name: "has description returns false",
			input: &ProductUpdateInput{
				Description: productStrPtr("A description"),
			},
			expected: false,
		},
		{
			name: "has metadata returns false",
			input: &ProductUpdateInput{
				Metadata: &map[string]any{"key": "value"},
			},
			expected: false,
		},
		{
			name: "all fields set returns false",
			input: &ProductUpdateInput{
				Name:        productStrPtr("My Product"),
				Description: productStrPtr("A description"),
				Metadata:    &map[string]any{"key": "value"},
			},
			expected: false,
		},
		{
			name: "empty string pointer for name returns false (pointer is non-nil)",
			input: &ProductUpdateInput{
				Name: productStrPtr(""),
			},
			expected: false,
		},
		{
			name: "nil name nil description nil metadata returns true",
			input: &ProductUpdateInput{
				Name:        nil,
				Description: nil,
				Metadata:    nil,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.IsEmpty()
			if result != tt.expected {
				t.Fatalf("expected IsEmpty() = %v, got %v", tt.expected, result)
			}
		})
	}
}

// ##############################################################################################################################################################################
// NewProductResponseFrom

// TestNewProductResponseFrom tests the NewProductResponseFrom function.
func TestNewProductResponseFrom(t *testing.T) {
	t.Run("nil product returns nil", func(t *testing.T) {
		result := NewProductResponseFrom(nil)
		if result != nil {
			t.Fatalf("expected nil, got %+v", result)
		}
	})

	t.Run("valid product maps all fields correctly", func(t *testing.T) {
		productID := uuid.New()
		orgID := uuid.New()
		now := time.Now().UTC()
		meta := &map[string]any{"key": "value"}

		p := &Product{
			ID:             productID,
			OrganizationID: orgID,
			Code:           "my-product",
			Name:           "My Product",
			Description:    "A valid description",
			Metadata:       meta,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		result := NewProductResponseFrom(p)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.ID != productID {
			t.Fatalf("expected ID %v, got %v", productID, result.ID)
		}
		if result.Code != "my-product" {
			t.Fatalf("expected Code 'my-product', got '%s'", result.Code)
		}
		if result.Name != "My Product" {
			t.Fatalf("expected Name 'My Product', got '%s'", result.Name)
		}
		if result.Description != "A valid description" {
			t.Fatalf("expected Description 'A valid description', got '%s'", result.Description)
		}
		if result.Metadata == nil {
			t.Fatal("expected Metadata to be set")
		}
		if (*result.Metadata)["key"] != "value" {
			t.Fatalf("expected Metadata['key'] = 'value', got '%v'", (*result.Metadata)["key"])
		}
		if !result.CreatedAt.Equal(now) {
			t.Fatalf("expected CreatedAt %v, got %v", now, result.CreatedAt)
		}
		if !result.UpdatedAt.Equal(now) {
			t.Fatalf("expected UpdatedAt %v, got %v", now, result.UpdatedAt)
		}
	})

	t.Run("product without description maps correctly", func(t *testing.T) {
		p := &Product{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			Code:           "reporter",
			Name:           "Reporter",
			Description:    "",
			Metadata:       nil,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}

		result := NewProductResponseFrom(p)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Description != "" {
			t.Fatalf("expected empty Description, got '%s'", result.Description)
		}
		if result.Metadata != nil {
			t.Fatalf("expected Metadata nil, got %v", result.Metadata)
		}
	})

	t.Run("organization ID is not included in response", func(t *testing.T) {
		orgID := uuid.New()
		p := &Product{
			ID:             uuid.New(),
			OrganizationID: orgID,
			Code:           "my-product",
			Name:           "My Product",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}

		result := NewProductResponseFrom(p)

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// ProductResponse does not expose OrganizationID
		// Verify that we can only access ID, not OrganizationID from the response
		if result.ID == uuid.Nil {
			t.Fatal("expected result ID to be set")
		}
	})

	t.Run("response ID matches product ID", func(t *testing.T) {
		productID := uuid.New()
		p := &Product{
			ID:             productID,
			OrganizationID: uuid.New(),
			Code:           "my-product",
			Name:           "My Product",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}

		result := NewProductResponseFrom(p)

		if result.ID != productID {
			t.Fatalf("expected response ID %v to match product ID %v", result.ID, productID)
		}
	})
}

// ##############################################################################################################################################################################
// ToMapWithMask

// TestProduct_ToMapWithMask tests the Product.ToMapWithMask method.
func TestProduct_ToMapWithMask(t *testing.T) {
	t.Run("all fields present in map", func(t *testing.T) {
		productID := uuid.New()
		orgID := uuid.New()
		now := time.Now().UTC()
		meta := &map[string]any{"key": "value"}
		deleteTime := now.Add(1 * time.Hour)

		p := &Product{
			ID:             productID,
			OrganizationID: orgID,
			Code:           "my-product",
			Name:           "My Product",
			Description:    "A description",
			Metadata:       meta,
			CreatedAt:      now,
			UpdatedAt:      now,
			DeletedAt:      &deleteTime,
		}

		result := p.ToMapWithMask()

		if result["id"] != productID {
			t.Fatalf("expected id %v, got %v", productID, result["id"])
		}
		if result["organization_id"] != orgID {
			t.Fatalf("expected organization_id %v, got %v", orgID, result["organization_id"])
		}
		if result["code"] != "my-product" {
			t.Fatalf("expected code 'my-product', got %v", result["code"])
		}
		if result["name"] != "My Product" {
			t.Fatalf("expected name 'My Product', got %v", result["name"])
		}
		if result["description"] != "A description" {
			t.Fatalf("expected description 'A description', got %v", result["description"])
		}
		if result["metadata"] != meta {
			t.Fatalf("expected metadata %v, got %v", meta, result["metadata"])
		}
		if result["created_at"] != now {
			t.Fatalf("expected created_at %v, got %v", now, result["created_at"])
		}
		if result["updated_at"] != now {
			t.Fatalf("expected updated_at %v, got %v", now, result["updated_at"])
		}
		if result["deleted_at"] != &deleteTime {
			t.Fatalf("expected deleted_at %v, got %v", &deleteTime, result["deleted_at"])
		}
	})

	t.Run("product without deleted_at has nil deleted_at in map", func(t *testing.T) {
		p := &Product{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			Code:           "my-product",
			Name:           "My Product",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
			DeletedAt:      nil,
		}

		result := p.ToMapWithMask()

		if result["deleted_at"] != (*time.Time)(nil) {
			t.Fatalf("expected deleted_at nil, got %v", result["deleted_at"])
		}
	})

	t.Run("map contains all expected keys", func(t *testing.T) {
		p := &Product{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			Code:           "my-product",
			Name:           "My Product",
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}

		result := p.ToMapWithMask()

		expectedKeys := []string{
			"id",
			"organization_id",
			"code",
			"name",
			"description",
			"metadata",
			"created_at",
			"updated_at",
			"deleted_at",
		}

		for _, key := range expectedKeys {
			if _, exists := result[key]; !exists {
				t.Fatalf("expected key '%s' to be present in map, got keys: %v", key, mapKeys(result))
			}
		}
	})

	t.Run("product with nil metadata shows nil in map", func(t *testing.T) {
		p := &Product{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			Code:           "my-product",
			Name:           "My Product",
			Metadata:       nil,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}

		result := p.ToMapWithMask()

		if result["metadata"] != (*map[string]any)(nil) {
			t.Fatalf("expected metadata nil, got %v", result["metadata"])
		}
	})
}

// ##############################################################################################################################################################################
// Helper functions

// productStrPtr is a local helper to get a pointer to a string value.
// (Note: strPtr and intPtr are already defined in connection_test.go, which is in the same package.
// productStrPtr avoids re-declaration conflicts and is semantically scoped to product tests.)
func productStrPtr(s string) *string {
	return &s
}

// mapKeys returns the keys of a map for diagnostic messages.
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
