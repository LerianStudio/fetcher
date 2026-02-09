package command

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/model"
	productMock "github.com/LerianStudio/fetcher/pkg/mongodb/product"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newValidProductInput creates a valid ProductInput for testing.
func newValidProductInput() model.ProductInput {
	return model.ProductInput{
		Code:        "my-product",
		Name:        "My Product",
		Description: "A test product for unit testing",
	}
}

// TestNewCreateProduct verifies the constructor creates a valid service instance.
func TestNewCreateProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewCreateProduct(mockRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.productRepo == nil {
		t.Fatal("expected productRepo to be set")
	}
}

// TestCreateProduct_Execute_Success tests successful product creation.
func TestCreateProduct_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewCreateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	input := newValidProductInput()

	// Mock: no existing product found (no duplicate)
	mockRepo.EXPECT().
		FindByCode(gomock.Any(), input.Code, orgID).
		Return(nil, nil)

	// Mock: create returns the product
	mockRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, product *model.Product) (*model.Product, error) {
			return product, nil
		})

	result, err := svc.Execute(ctx, orgID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Code != input.Code {
		t.Fatalf("expected Code %s, got %s", input.Code, result.Code)
	}

	if result.Name != input.Name {
		t.Fatalf("expected Name %s, got %s", input.Name, result.Name)
	}

	if result.Description != input.Description {
		t.Fatalf("expected Description %s, got %s", input.Description, result.Description)
	}

	if result.OrganizationID != orgID {
		t.Fatalf("expected OrganizationID %s, got %s", orgID, result.OrganizationID)
	}

	if result.ID == uuid.Nil {
		t.Fatal("expected non-nil product ID")
	}

	if result.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	if result.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
}

// TestCreateProduct_Execute_DuplicateCode tests that a duplicate product code returns a conflict error.
func TestCreateProduct_Execute_DuplicateCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewCreateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	input := newValidProductInput()

	existingProduct := &model.Product{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Code:           input.Code,
		Name:           "Existing Product",
	}

	// Mock: existing product found
	mockRepo.EXPECT().
		FindByCode(gomock.Any(), input.Code, orgID).
		Return(existingProduct, nil)

	result, err := svc.Execute(ctx, orgID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for duplicate product code, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}

	assert.Equal(t, http.StatusConflict, respErr.StatusCode)
}

// TestCreateProduct_Execute_FindByCodeRepoError tests repository error during code lookup.
func TestCreateProduct_Execute_FindByCodeRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewCreateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	input := newValidProductInput()

	dbError := errors.New("database connection failed")

	// Mock: database error during lookup
	mockRepo.EXPECT().
		FindByCode(gomock.Any(), input.Code, orgID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to wrap dbError, got %v", err)
	}
}

// TestCreateProduct_Execute_CreateRepoError tests repository error during creation.
func TestCreateProduct_Execute_CreateRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewCreateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()
	input := newValidProductInput()

	dbError := errors.New("failed to insert into database")

	// Mock: no existing product found
	mockRepo.EXPECT().
		FindByCode(gomock.Any(), input.Code, orgID).
		Return(nil, nil)

	// Mock: create returns error
	mockRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to wrap dbError, got %v", err)
	}
}

// TestCreateProduct_Execute_ValidationErrors tests validation failures using table-driven tests.
func TestCreateProduct_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		input        model.ProductInput
		wantErrField string
		wantErrMsg   string
	}{
		{
			name: "empty code",
			input: model.ProductInput{
				Code: "",
				Name: "Valid Name",
			},
			wantErrField: "code",
			wantErrMsg:   "code is required",
		},
		{
			name: "whitespace-only code (normalized to empty)",
			input: model.ProductInput{
				Code: "   ",
				Name: "Valid Name",
			},
			wantErrField: "code",
			wantErrMsg:   "code is required",
		},
		{
			name: "invalid slug format - special characters",
			input: model.ProductInput{
				Code: "my-product!",
				Name: "Valid Name",
			},
			wantErrField: "code",
		},
		{
			name: "invalid slug format - underscore",
			input: model.ProductInput{
				Code: "my_product",
				Name: "Valid Name",
			},
			wantErrField: "code",
		},
		{
			name: "invalid slug format - leading hyphen",
			input: model.ProductInput{
				Code: "-my-product",
				Name: "Valid Name",
			},
			wantErrField: "code",
		},
		{
			name: "invalid slug format - trailing hyphen",
			input: model.ProductInput{
				Code: "my-product-",
				Name: "Valid Name",
			},
			wantErrField: "code",
		},
		{
			name: "invalid slug format - consecutive hyphens",
			input: model.ProductInput{
				Code: "my--product",
				Name: "Valid Name",
			},
			wantErrField: "code",
		},
		{
			name: "invalid slug format - spaces",
			input: model.ProductInput{
				Code: "my product",
				Name: "Valid Name",
			},
			wantErrField: "code",
		},
		{
			name: "code too short (1 char)",
			input: model.ProductInput{
				Code: "a",
				Name: "Valid Name",
			},
			wantErrField: "code",
			wantErrMsg:   "code must be between 2 and 50 characters",
		},
		{
			name: "code too long (>50 chars)",
			input: model.ProductInput{
				Code: strings.Repeat("a", 51),
				Name: "Valid Name",
			},
			wantErrField: "code",
			wantErrMsg:   "code must be between 2 and 50 characters",
		},
		{
			name: "empty name",
			input: model.ProductInput{
				Code: "valid-code",
				Name: "",
			},
			wantErrField: "name",
			wantErrMsg:   "name is required",
		},
		{
			name: "name too long (>100 chars)",
			input: model.ProductInput{
				Code: "valid-code",
				Name: strings.Repeat("n", 101),
			},
			wantErrField: "name",
			wantErrMsg:   "name must be at most 100 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := productMock.NewMockRepository(ctrl)
			svc := NewCreateProduct(mockRepo)

			ctx := testContext()
			orgID := uuid.New()

			result, err := svc.Execute(ctx, orgID, tt.input)

			if result != nil {
				t.Fatalf("expected nil result for invalid request, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error for invalid request, got nil")
			}

			var knownFieldsErr pkg.ValidationKnownFieldsError
			if !errors.As(err, &knownFieldsErr) {
				t.Fatalf("expected ValidationKnownFieldsError, got %T: %v", err, err)
			}

			if _, exists := knownFieldsErr.Fields[tt.wantErrField]; !exists {
				t.Fatalf("expected field %s in error fields, got %v", tt.wantErrField, knownFieldsErr.Fields)
			}

			if tt.wantErrMsg != "" {
				if knownFieldsErr.Fields[tt.wantErrField] != tt.wantErrMsg {
					t.Fatalf("expected field message %q, got %q", tt.wantErrMsg, knownFieldsErr.Fields[tt.wantErrField])
				}
			}
		})
	}
}

// TestCreateProduct_Execute_WithMetadata tests successful product creation with metadata.
func TestCreateProduct_Execute_WithMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewCreateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()

	metadata := map[string]any{
		"team":        "engineering",
		"environment": "production",
		"version":     "1.0",
	}

	input := model.ProductInput{
		Code:        "product-with-meta",
		Name:        "Product With Metadata",
		Description: "A product that includes metadata",
		Metadata:    &metadata,
	}

	// Mock: no existing product found
	mockRepo.EXPECT().
		FindByCode(gomock.Any(), input.Code, orgID).
		Return(nil, nil)

	// Mock: create returns the product
	mockRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, product *model.Product) (*model.Product, error) {
			return product, nil
		})

	result, err := svc.Execute(ctx, orgID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Metadata == nil {
		t.Fatal("expected metadata to be set")
	}

	resultMeta := *result.Metadata
	if resultMeta["team"] != "engineering" {
		t.Fatalf("expected metadata team=engineering, got %v", resultMeta["team"])
	}

	if resultMeta["environment"] != "production" {
		t.Fatalf("expected metadata environment=production, got %v", resultMeta["environment"])
	}

	if resultMeta["version"] != "1.0" {
		t.Fatalf("expected metadata version=1.0, got %v", resultMeta["version"])
	}
}

// TestCreateProduct_Execute_CodeNormalization tests that codes with leading/trailing whitespace
// and uppercase letters are normalized before validation and storage.
func TestCreateProduct_Execute_CodeNormalization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productMock.NewMockRepository(ctrl)
	svc := NewCreateProduct(mockRepo)

	ctx := testContext()
	orgID := uuid.New()

	input := model.ProductInput{
		Code: "  my-product  ",
		Name: "My Product",
	}

	// After normalization, code becomes "my-product" (trimmed + lowered)
	mockRepo.EXPECT().
		FindByCode(gomock.Any(), "my-product", orgID).
		Return(nil, nil)

	mockRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, product *model.Product) (*model.Product, error) {
			if product.Code != "my-product" {
				t.Fatalf("expected normalized code 'my-product', got %s", product.Code)
			}
			return product, nil
		})

	result, err := svc.Execute(ctx, orgID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Code != "my-product" {
		t.Fatalf("expected code 'my-product', got %s", result.Code)
	}
}

// TestCreateProduct_Execute_ValidCodeEdgeCases tests valid code edge cases using table-driven tests.
func TestCreateProduct_Execute_ValidCodeEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		expectedCode string
	}{
		{
			name:         "exactly 2 characters",
			code:         "ab",
			expectedCode: "ab",
		},
		{
			name:         "exactly 50 characters",
			code:         strings.Repeat("a", 50),
			expectedCode: strings.Repeat("a", 50),
		},
		{
			name:         "slug with single hyphen",
			code:         "my-product",
			expectedCode: "my-product",
		},
		{
			name:         "slug with multiple hyphens",
			code:         "my-cool-product-name",
			expectedCode: "my-cool-product-name",
		},
		{
			name:         "numeric only",
			code:         "12345",
			expectedCode: "12345",
		},
		{
			name:         "alphanumeric with hyphens",
			code:         "product-v2-beta",
			expectedCode: "product-v2-beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := productMock.NewMockRepository(ctrl)
			svc := NewCreateProduct(mockRepo)

			ctx := testContext()
			orgID := uuid.New()

			input := model.ProductInput{
				Code: tt.code,
				Name: "Valid Product Name",
			}

			// Mock: no existing product found
			mockRepo.EXPECT().
				FindByCode(gomock.Any(), tt.expectedCode, orgID).
				Return(nil, nil)

			// Mock: create returns the product
			mockRepo.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, product *model.Product) (*model.Product, error) {
					return product, nil
				})

			result, err := svc.Execute(ctx, orgID, input)
			if err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Code != tt.expectedCode {
				t.Fatalf("expected code %s, got %s", tt.expectedCode, result.Code)
			}
		})
	}
}
