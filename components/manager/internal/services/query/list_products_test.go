package query

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	productMock "github.com/LerianStudio/fetcher/pkg/mongodb/product"
	"github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// newListTestProduct creates a test Product with the given parameters for list tests.
func newListTestProduct(orgID, productID uuid.UUID, code, name string) *model.Product {
	now := time.Now().UTC()

	return &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           code,
		Name:           name,
		Description:    "test product description",
		CreatedAt:      now.Add(-24 * time.Hour),
		UpdatedAt:      now.Add(-1 * time.Hour),
	}
}

// TestListProducts_Execute_Success tests successful listing of products.
func TestListProducts_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)
	svc := NewListProducts(mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	product1 := newListTestProduct(orgID, uuid.New(), "reporter", "Reporter")
	product2 := newListTestProduct(orgID, uuid.New(), "ledger", "Ledger")
	expectedList := []*model.Product{product1, product2}

	mockProductRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(expectedList, int64(2), nil)

	result, err := svc.Execute(ctx, orgID, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	items := result.Items.([]*model.ProductResponse)

	if len(items) != len(expectedList) {
		t.Fatalf("expected %d products, got %d", len(expectedList), len(items))
	}

	if items[0].Code != "reporter" {
		t.Fatalf("expected first product code 'reporter', got %s", items[0].Code)
	}

	if items[1].Code != "ledger" {
		t.Fatalf("expected second product code 'ledger', got %s", items[1].Code)
	}
}

// TestListProducts_Execute_EmptyList tests that nil repo result returns empty slice, not nil.
func TestListProducts_Execute_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)
	svc := NewListProducts(mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Mock: list returns nil (no products found)
	mockProductRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(nil, int64(0), nil)

	result, err := svc.Execute(ctx, orgID, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Total != 0 {
		t.Fatalf("expected total 0, got %d", result.Total)
	}
}

// TestListProducts_Execute_RepositoryError tests repository error handling.
func TestListProducts_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)
	svc := NewListProducts(mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	dbError := errors.New("database connection failed")

	// Mock: list returns error
	mockProductRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(nil, int64(0), dbError)

	result, err := svc.Execute(ctx, orgID, filters)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to be dbError, got %v", err)
	}
}

// TestNewListProducts verifies the constructor sets the repo correctly.
func TestNewListProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)
	svc := NewListProducts(mockProductRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.productRepo == nil {
		t.Fatal("expected productRepo to be set")
	}
}

// TestListProducts_Execute_TableDriven uses table-driven tests for various scenarios.
func TestListProducts_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		filters         http.QueryHeader
		setupMocks      func(*productMock.MockRepository, uuid.UUID, http.QueryHeader)
		wantErr         bool
		wantResultCount int
	}{
		{
			name: "successful list with multiple products",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  1,
			},
			setupMocks: func(mock *productMock.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				products := []*model.Product{
					newListTestProduct(orgID, uuid.New(), "reporter", "Reporter"),
					newListTestProduct(orgID, uuid.New(), "ledger", "Ledger"),
					newListTestProduct(orgID, uuid.New(), "transaction", "Transaction"),
				}
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(products, int64(3), nil)
			},
			wantErr:         false,
			wantResultCount: 3,
		},
		{
			name: "empty list returns empty slice",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  1,
			},
			setupMocks: func(mock *productMock.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, int64(0), nil)
			},
			wantErr:         false,
			wantResultCount: 0,
		},
		{
			name: "repository error",
			filters: http.QueryHeader{
				Limit: 10,
				Page:  1,
			},
			setupMocks: func(mock *productMock.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, int64(0), errors.New("database error"))
			},
			wantErr:         true,
			wantResultCount: 0,
		},
		{
			name: "list with page 2",
			filters: http.QueryHeader{
				Limit: 5,
				Page:  2,
			},
			setupMocks: func(mock *productMock.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				products := []*model.Product{
					newListTestProduct(orgID, uuid.New(), "product-6", "Product 6"),
					newListTestProduct(orgID, uuid.New(), "product-7", "Product 7"),
				}
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(products, int64(2), nil)
			},
			wantErr:         false,
			wantResultCount: 2,
		},
		{
			name: "list with limit 1",
			filters: http.QueryHeader{
				Limit: 1,
				Page:  1,
			},
			setupMocks: func(mock *productMock.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				products := []*model.Product{
					newListTestProduct(orgID, uuid.New(), "single", "Single Product"),
				}
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(products, int64(1), nil)
			},
			wantErr:         false,
			wantResultCount: 1,
		},
		{
			name:    "list with empty filters",
			filters: http.QueryHeader{},
			setupMocks: func(mock *productMock.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				products := []*model.Product{
					newListTestProduct(orgID, uuid.New(), "default", "Default Product"),
				}
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(products, int64(1), nil)
			},
			wantErr:         false,
			wantResultCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProductRepo := productMock.NewMockRepository(ctrl)
			ctx := testContext()
			orgID := uuid.New()

			tt.setupMocks(mockProductRepo, orgID, tt.filters)

			svc := NewListProducts(mockProductRepo)

			result, err := svc.Execute(ctx, orgID, tt.filters)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil && tt.wantResultCount > 0 {
				t.Fatal("expected non-nil result")
			}

			items := result.Items.([]*model.ProductResponse)

			if len(items) != tt.wantResultCount {
				t.Fatalf("expected %d products, got %d", tt.wantResultCount, len(items))
			}
		})
	}
}

// TestListProducts_Execute_OrganizationIsolation tests that products are isolated by organization.
func TestListProducts_Execute_OrganizationIsolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productMock.NewMockRepository(ctrl)
	svc := NewListProducts(mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	filters := http.QueryHeader{
		Limit: 10,
		Page:  1,
	}

	// Only products for the specified organization should be returned
	orgProducts := []*model.Product{
		newListTestProduct(orgID, uuid.New(), "org1-product-1", "Org1 Product 1"),
		newListTestProduct(orgID, uuid.New(), "org1-product-2", "Org1 Product 2"),
	}

	mockProductRepo.EXPECT().
		List(gomock.Any(), orgID, filters).
		Return(orgProducts, int64(2), nil)

	result, err := svc.Execute(ctx, orgID, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.Items.([]*model.ProductResponse)

	if len(items) != 2 {
		t.Fatalf("expected 2 products, got %d", len(items))
	}

	// Verify all returned products belong to the requested organization
	// Note: ProductResponse does not have OrganizationID, so we check the count matches
	// the expected org-scoped result from the repository.
	for _, product := range items {
		if product.Code != "org1-product-1" && product.Code != "org1-product-2" {
			t.Fatalf("unexpected product code: %s", product.Code)
		}
	}
}

// TestListProducts_Execute_ErrorScenarios tests various error scenarios.
func TestListProducts_Execute_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*productMock.MockRepository, uuid.UUID, http.QueryHeader)
		errorMsg  string
	}{
		{
			name: "database connection error",
			setupMock: func(mock *productMock.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, int64(0), errors.New("database connection failed"))
			},
			errorMsg: "database connection failed",
		},
		{
			name: "timeout error",
			setupMock: func(mock *productMock.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, int64(0), errors.New("context deadline exceeded"))
			},
			errorMsg: "context deadline exceeded",
		},
		{
			name: "permission denied error",
			setupMock: func(mock *productMock.MockRepository, orgID uuid.UUID, filters http.QueryHeader) {
				mock.EXPECT().
					List(gomock.Any(), orgID, filters).
					Return(nil, int64(0), errors.New("permission denied"))
			},
			errorMsg: "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProductRepo := productMock.NewMockRepository(ctrl)
			svc := NewListProducts(mockProductRepo)

			ctx := testContext()
			orgID := uuid.New()
			filters := http.QueryHeader{
				Limit: 10,
				Page:  1,
			}

			tt.setupMock(mockProductRepo, orgID, filters)

			result, err := svc.Execute(ctx, orgID, filters)

			if result != nil {
				t.Fatalf("expected nil result, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Fatalf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}
