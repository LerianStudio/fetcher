package command

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	productRepo "github.com/LerianStudio/fetcher/pkg/mongodb/product"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestDeleteProduct_Execute_Success tests successful product deletion when no connections are assigned.
func TestDeleteProduct_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	existingProduct := &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
	}

	// Mock: product exists
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: no connections assigned to this product
	mockConnRepo.EXPECT().
		CountByProduct(gomock.Any(), orgID, productID).
		Return(int64(0), nil)

	// Mock: delete succeeds
	mockProductRepo.EXPECT().
		Delete(gomock.Any(), productID, orgID, gomock.Any()).
		Return(nil)

	err := svc.Execute(ctx, orgID, productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeleteProduct_Execute_NotFoundError tests that a non-existent product returns not found error.
func TestDeleteProduct_Execute_NotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	// Mock: product not found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, nil)

	err := svc.Execute(ctx, orgID, productID)

	if err == nil {
		t.Fatal("expected error for non-existent product, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}

// TestDeleteProduct_Execute_ProductHasConnections tests that deletion fails when product has assigned connections.
func TestDeleteProduct_Execute_ProductHasConnections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	existingProduct := &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
	}

	// Mock: product exists
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: product has assigned connections
	mockConnRepo.EXPECT().
		CountByProduct(gomock.Any(), orgID, productID).
		Return(int64(3), nil)

	err := svc.Execute(ctx, orgID, productID)

	if err == nil {
		t.Fatal("expected error for product with connections, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusConflict, respErr.StatusCode)
}

// TestDeleteProduct_Execute_CountByProductError tests repository error during CountByProduct.
func TestDeleteProduct_Execute_CountByProductError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	dbError := errors.New("database connection failed")

	existingProduct := &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
	}

	// Mock: product exists
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: error during connection count
	mockConnRepo.EXPECT().
		CountByProduct(gomock.Any(), orgID, productID).
		Return(int64(0), dbError)

	err := svc.Execute(ctx, orgID, productID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to wrap dbError, got %v", err)
	}
}

// TestDeleteProduct_Execute_DeleteError tests repository error during Delete.
func TestDeleteProduct_Execute_DeleteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	dbError := errors.New("failed to delete from database")

	existingProduct := &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
	}

	// Mock: product exists
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: no connections assigned
	mockConnRepo.EXPECT().
		CountByProduct(gomock.Any(), orgID, productID).
		Return(int64(0), nil)

	// Mock: delete returns error
	mockProductRepo.EXPECT().
		Delete(gomock.Any(), productID, orgID, gomock.Any()).
		Return(dbError)

	err := svc.Execute(ctx, orgID, productID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to wrap dbError, got %v", err)
	}
}

// TestNewDeleteProduct verifies the constructor.
func TestNewDeleteProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.productRepo == nil {
		t.Fatal("expected productRepo to be set")
	}

	if svc.connRepo == nil {
		t.Fatal("expected connRepo to be set")
	}
}

// TestDeleteProduct_Execute_TableDriven uses table-driven tests for various scenarios.
func TestDeleteProduct_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*productRepo.MockRepository, *connRepo.MockRepository, uuid.UUID, uuid.UUID)
		wantErr        bool
		wantStatusCode int // 0 means generic error (no status code check)
	}{
		{
			name: "successful deletion",
			setupMocks: func(prodMock *productRepo.MockRepository, connMock *connRepo.MockRepository, orgID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(&model.Product{ID: productID, OrganizationID: orgID, Code: "test", Name: "Test"}, nil)
				connMock.EXPECT().
					CountByProduct(gomock.Any(), orgID, productID).
					Return(int64(0), nil)
				prodMock.EXPECT().
					Delete(gomock.Any(), productID, orgID, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "product not found via FindByID",
			setupMocks: func(prodMock *productRepo.MockRepository, connMock *connRepo.MockRepository, orgID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "product has connections",
			setupMocks: func(prodMock *productRepo.MockRepository, connMock *connRepo.MockRepository, orgID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(&model.Product{ID: productID, OrganizationID: orgID, Code: "test", Name: "Test"}, nil)
				connMock.EXPECT().
					CountByProduct(gomock.Any(), orgID, productID).
					Return(int64(5), nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusConflict,
		},
		{
			name: "CountByProduct database error",
			setupMocks: func(prodMock *productRepo.MockRepository, connMock *connRepo.MockRepository, orgID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(&model.Product{ID: productID, OrganizationID: orgID, Code: "test", Name: "Test"}, nil)
				connMock.EXPECT().
					CountByProduct(gomock.Any(), orgID, productID).
					Return(int64(0), errors.New("database connection failed"))
			},
			wantErr:        true,
			wantStatusCode: 0,
		},
		{
			name: "Delete database error",
			setupMocks: func(prodMock *productRepo.MockRepository, connMock *connRepo.MockRepository, orgID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(&model.Product{ID: productID, OrganizationID: orgID, Code: "test", Name: "Test"}, nil)
				connMock.EXPECT().
					CountByProduct(gomock.Any(), orgID, productID).
					Return(int64(0), nil)
				prodMock.EXPECT().
					Delete(gomock.Any(), productID, orgID, gomock.Any()).
					Return(errors.New("failed to delete"))
			},
			wantErr:        true,
			wantStatusCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProductRepo := productRepo.NewMockRepository(ctrl)
			mockConnRepo := connRepo.NewMockRepository(ctrl)

			ctx := testContext()
			orgID := uuid.New()
			productID := uuid.New()

			tt.setupMocks(mockProductRepo, mockConnRepo, orgID, productID)

			svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

			err := svc.Execute(ctx, orgID, productID)

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
		})
	}
}

// TestDeleteProduct_Execute_DifferentOrganizations tests that products are isolated by organization.
func TestDeleteProduct_Execute_DifferentOrganizations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	// Mock: product not found (belongs to a different org)
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, nil)

	err := svc.Execute(ctx, orgID, productID)

	if err == nil {
		t.Fatal("expected error for product in different organization, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}

// TestDeleteProduct_Execute_DeletePassesCorrectTimestamp tests that Delete receives a valid timestamp.
func TestDeleteProduct_Execute_DeletePassesCorrectTimestamp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

	ctx := testContext()
	orgID := uuid.New()
	productID := uuid.New()

	beforeExecution := time.Now().UTC()

	existingProduct := &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
	}

	// Mock: product exists
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: no connections assigned
	mockConnRepo.EXPECT().
		CountByProduct(gomock.Any(), orgID, productID).
		Return(int64(0), nil)

	// Mock: delete with timestamp validation
	mockProductRepo.EXPECT().
		Delete(gomock.Any(), productID, orgID, gomock.Any()).
		DoAndReturn(func(ctx interface{}, id, orgID uuid.UUID, deletedAt time.Time) error {
			afterExecution := time.Now().UTC()
			if deletedAt.Before(beforeExecution) || deletedAt.After(afterExecution) {
				t.Errorf("expected deletedAt between %v and %v, got %v", beforeExecution, afterExecution, deletedAt)
			}
			return nil
		})

	err := svc.Execute(ctx, orgID, productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeleteProduct_Execute_ConnectionCountBoundary tests boundary values for connection count.
func TestDeleteProduct_Execute_ConnectionCountBoundary(t *testing.T) {
	tests := []struct {
		name    string
		count   int64
		wantErr bool
	}{
		{
			name:    "zero connections allows deletion",
			count:   0,
			wantErr: false,
		},
		{
			name:    "one connection prevents deletion",
			count:   1,
			wantErr: true,
		},
		{
			name:    "many connections prevent deletion",
			count:   100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProductRepo := productRepo.NewMockRepository(ctrl)
			mockConnRepo := connRepo.NewMockRepository(ctrl)

			svc := NewDeleteProduct(mockProductRepo, mockConnRepo)

			ctx := testContext()
			orgID := uuid.New()
			productID := uuid.New()

			existingProduct := &model.Product{
				ID:             productID,
				OrganizationID: orgID,
				Code:           "test-product",
				Name:           "Test Product",
			}

			// Mock: product exists
			mockProductRepo.EXPECT().
				FindByID(gomock.Any(), productID, orgID).
				Return(existingProduct, nil)

			// Mock: connection count
			mockConnRepo.EXPECT().
				CountByProduct(gomock.Any(), orgID, productID).
				Return(tt.count, nil)

			if !tt.wantErr {
				// Mock: delete succeeds only if no connections
				mockProductRepo.EXPECT().
					Delete(gomock.Any(), productID, orgID, gomock.Any()).
					Return(nil)
			}

			err := svc.Execute(ctx, orgID, productID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var respErr pkg.ResponseErrorWithStatusCode
				if !errors.As(err, &respErr) {
					t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
				}
				assert.Equal(t, http.StatusConflict, respErr.StatusCode)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
