package query

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/model"
	productMock "github.com/LerianStudio/fetcher/pkg/mongodb/product"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// newExistingProduct creates a valid existing Product for testing.
func newExistingProduct(orgID, productID uuid.UUID) *model.Product {
	return &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
		Description:    "A test product for unit tests",
		CreatedAt:      time.Now().UTC().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().UTC().Add(-1 * time.Hour),
	}
}

// TestGetProduct_Execute_Success tests successful product retrieval.
func TestGetProduct_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewGetProduct(mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()
	existingProduct := newExistingProduct(orgID, productID)

	// Mock: product found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	result, err := svc.Execute(ctx, orgID, productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.ID != productID {
		t.Fatalf("expected ID %s, got %s", productID, result.ID)
	}

	if result.OrganizationID != orgID {
		t.Fatalf("expected OrganizationID %s, got %s", orgID, result.OrganizationID)
	}

	if result.Code != existingProduct.Code {
		t.Fatalf("expected Code %s, got %s", existingProduct.Code, result.Code)
	}

	if result.Name != existingProduct.Name {
		t.Fatalf("expected Name %s, got %s", existingProduct.Name, result.Name)
	}

	if result.Description != existingProduct.Description {
		t.Fatalf("expected Description %s, got %s", existingProduct.Description, result.Description)
	}
}

// TestGetProduct_Execute_NotFoundError tests that non-existent product returns not found error.
func TestGetProduct_Execute_NotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewGetProduct(mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	// Mock: product not found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, productID)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for non-existent product, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}

// TestGetProduct_Execute_RepositoryError tests repository error during FindByID.
func TestGetProduct_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewGetProduct(mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	dbError := errors.New("database connection failed")

	// Mock: database error during lookup
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, productID)

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

// TestNewGetProduct verifies the constructor.
func TestNewGetProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewGetProduct(mockProductRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.productRepo == nil {
		t.Fatal("expected productRepo to be set")
	}
}

// TestGetProduct_Execute_OrganizationIsolation tests that products are isolated by organization.
func TestGetProduct_Execute_OrganizationIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewGetProduct(mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	differentOrgID := uuid.New()
	productID := uuid.New()

	// Product belongs to a different organization but we query with orgID
	// The repository should return nil because it filters by organizationID
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, productID)

	if result != nil {
		t.Fatalf("expected nil result for product in different organization, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for product in different organization, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)

	// Verify the existing product is not returned (it belongs to a different org)
	_ = differentOrgID // Unused in mock but demonstrates the test scenario
}

// TestGetProduct_Execute_TableDriven uses table-driven tests for various scenarios.
func TestGetProduct_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*productMock.MockRepository, uuid.UUID, uuid.UUID, *model.Product)
		wantErr        bool
		wantStatusCode int // 0 means generic error (no status code check)
	}{
		{
			name: "successful retrieval",
			setupMocks: func(prodMock *productMock.MockRepository, orgID, productID uuid.UUID, existing *model.Product) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(existing, nil)
			},
			wantErr: false,
		},
		{
			name: "product not found",
			setupMocks: func(prodMock *productMock.MockRepository, orgID, productID uuid.UUID, existing *model.Product) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "FindByID database error",
			setupMocks: func(prodMock *productMock.MockRepository, orgID, productID uuid.UUID, existing *model.Product) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr:        true,
			wantStatusCode: 0, // generic error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProductRepo := productMock.NewMockRepository(ctrl)

			ctx := testContext()
			orgID := uuid.New()
			productID := uuid.New()
			existingProduct := newExistingProduct(orgID, productID)

			tt.setupMocks(mockProductRepo, orgID, productID, existingProduct)

			svc := NewGetProduct(mockProductRepo)

			result, err := svc.Execute(ctx, orgID, productID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.wantStatusCode != 0 {
					var respErr pkg.ResponseErrorWithStatusCode
					if !errors.As(err, &respErr) {
						t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
					}
					assert.Equal(t, tt.wantStatusCode, respErr.StatusCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestGetProduct_Execute_ProductWithAllFields tests retrieval of product with all fields populated.
func TestGetProduct_Execute_ProductWithAllFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewGetProduct(mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	metadata := map[string]any{"env": "production", "tier": "premium"}
	existingProduct := &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           "full-product",
		Name:           "Full Product",
		Description:    "A product with all fields",
		Metadata:       &metadata,
		CreatedAt:      time.Now().UTC().Add(-48 * time.Hour),
		UpdatedAt:      time.Now().UTC().Add(-30 * time.Minute),
	}

	// Mock: product found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	result, err := svc.Execute(ctx, orgID, productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify all fields
	if result.ID != productID {
		t.Fatalf("expected ID %s, got %s", productID, result.ID)
	}
	if result.OrganizationID != orgID {
		t.Fatalf("expected OrganizationID %s, got %s", orgID, result.OrganizationID)
	}
	if result.Code != "full-product" {
		t.Fatalf("expected Code 'full-product', got %s", result.Code)
	}
	if result.Name != "Full Product" {
		t.Fatalf("expected Name 'Full Product', got %s", result.Name)
	}
	if result.Description != "A product with all fields" {
		t.Fatalf("expected Description 'A product with all fields', got %s", result.Description)
	}
	if result.Metadata == nil {
		t.Fatal("expected Metadata to be present")
	}
	if (*result.Metadata)["env"] != "production" {
		t.Fatalf("expected Metadata env 'production', got %v", (*result.Metadata)["env"])
	}
	if (*result.Metadata)["tier"] != "premium" {
		t.Fatalf("expected Metadata tier 'premium', got %v", (*result.Metadata)["tier"])
	}
}

// TestGetProduct_Execute_MultipleRepositoryErrors tests various repository error scenarios.
func TestGetProduct_Execute_MultipleRepositoryErrors(t *testing.T) {
	tests := []struct {
		name    string
		dbError error
	}{
		{
			name:    "connection timeout",
			dbError: errors.New("connection timeout"),
		},
		{
			name:    "network error",
			dbError: errors.New("network error: no route to host"),
		},
		{
			name:    "authentication failed",
			dbError: errors.New("authentication failed"),
		},
		{
			name:    "permission denied",
			dbError: errors.New("permission denied"),
		},
		{
			name:    "database unavailable",
			dbError: errors.New("database unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProductRepo := productMock.NewMockRepository(ctrl)

			svc := NewGetProduct(mockProductRepo)

			ctx := testContext()
			orgID := uuid.New()
			productID := uuid.New()

			// Mock: database error
			mockProductRepo.EXPECT().
				FindByID(gomock.Any(), productID, orgID).
				Return(nil, tt.dbError)

			result, err := svc.Execute(ctx, orgID, productID)

			if result != nil {
				t.Fatalf("expected nil result for %s, got %+v", tt.name, result)
			}

			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}

			if !errors.Is(err, tt.dbError) {
				t.Fatalf("expected error to wrap %v, got %v", tt.dbError, err)
			}
		})
	}
}

// TestGetProduct_Execute_EmptyUUIDs tests behavior with edge case UUIDs.
func TestGetProduct_Execute_EmptyUUIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewGetProduct(mockProductRepo)

	ctx := testContext()
	orgID := uuid.Nil
	productID := uuid.Nil

	// Mock: product not found with nil UUIDs
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, productID)

	if result != nil {
		t.Fatalf("expected nil result with nil UUIDs, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error with nil UUIDs, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}
