package command

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	productMock "github.com/LerianStudio/fetcher/pkg/mongodb/product"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// newExistingConnectionForAssign creates a valid existing Connection without a ProductID (unassigned).
func newExistingConnectionForAssign(orgID, connID uuid.UUID) *model.Connection {
	return &model.Connection{
		ID:                   connID,
		OrganizationID:       orgID,
		ConfigName:           "existing-connection",
		Type:                 model.TypePostgreSQL,
		Host:                 "localhost",
		Port:                 5432,
		DatabaseName:         "testdb",
		Username:             "testuser",
		PasswordEncrypted:    "encrypted-password",
		EncryptionKeyVersion: "v1",
		CreatedAt:            time.Now().UTC().Add(-24 * time.Hour),
		UpdatedAt:            time.Now().UTC().Add(-1 * time.Hour),
	}
}

// newExistingProduct creates a valid existing Product for testing.
func newExistingProduct(orgID, productID uuid.UUID) *model.Product {
	return &model.Product{
		ID:             productID,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
		Description:    "A test product",
		CreatedAt:      time.Now().UTC().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().UTC().Add(-1 * time.Hour),
	}
}

// TestAssignConnection_Execute_Success tests successful connection assignment to a product.
func TestAssignConnection_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()

	existingProduct := newExistingProduct(orgID, productID)
	existingConn := newExistingConnectionForAssign(orgID, connID)

	updatedConn := newExistingConnectionForAssign(orgID, connID)
	updatedConn.ProductID = &productID
	updatedConn.UpdatedAt = time.Now().UTC()

	// Mock: product found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: connection found (unassigned)
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: assign product succeeds
	mockConnRepo.EXPECT().
		AssignProduct(gomock.Any(), connID, orgID, productID).
		Return(updatedConn, nil)

	result, err := svc.Execute(ctx, orgID, connID, productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.ProductID == nil {
		t.Fatal("expected ProductID to be set on returned connection")
	}

	assert.Equal(t, productID, *result.ProductID)
	assert.Equal(t, connID, result.ID)
}

// TestAssignConnection_Execute_ProductNotFound tests that a non-existent product returns not found error.
func TestAssignConnection_Execute_ProductNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()

	// Mock: product not found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID, productID)

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

// TestAssignConnection_Execute_ProductRepoError tests repository error during product lookup.
func TestAssignConnection_Execute_ProductRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()

	dbError := errors.New("database connection failed")

	// Mock: product repo returns error
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, connID, productID)

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

// TestAssignConnection_Execute_ConnectionNotFound tests that a non-existent connection returns not found error.
func TestAssignConnection_Execute_ConnectionNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()

	existingProduct := newExistingProduct(orgID, productID)

	// Mock: product found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: connection not found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID, productID)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for non-existent connection, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}

	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}

// TestAssignConnection_Execute_ConnectionRepoError tests repository error during connection lookup.
func TestAssignConnection_Execute_ConnectionRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()

	existingProduct := newExistingProduct(orgID, productID)
	dbError := errors.New("database connection failed")

	// Mock: product found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: connection repo returns error
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, connID, productID)

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

// TestAssignConnection_Execute_ConnectionAlreadyAssigned tests that assigning a connection
// that already has a ProductID returns a 409 conflict error.
func TestAssignConnection_Execute_ConnectionAlreadyAssigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()
	existingProductID := uuid.New()

	existingProduct := newExistingProduct(orgID, productID)
	existingConn := newExistingConnectionForAssign(orgID, connID)
	existingConn.ProductID = &existingProductID // Already assigned

	// Mock: product found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: connection found (already assigned)
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: AssignProduct returns nil (atomic guard rejects — product_id is not nil)
	mockConnRepo.EXPECT().
		AssignProduct(gomock.Any(), connID, orgID, productID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID, productID)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for already assigned connection, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}

	assert.Equal(t, http.StatusConflict, respErr.StatusCode)
}

// TestAssignConnection_Execute_RepoAssignProductReturnsNil tests the race condition
// where the repo AssignProduct returns (nil, nil), indicating the connection was assigned
// between the domain check and the persistence layer.
func TestAssignConnection_Execute_RepoAssignProductReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()

	existingProduct := newExistingProduct(orgID, productID)
	existingConn := newExistingConnectionForAssign(orgID, connID)

	// Mock: product found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: connection found (unassigned)
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: AssignProduct returns nil (race condition — assigned between check and persist)
	mockConnRepo.EXPECT().
		AssignProduct(gomock.Any(), connID, orgID, productID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID, productID)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for race condition, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}

	assert.Equal(t, http.StatusConflict, respErr.StatusCode)
}

// TestAssignConnection_Execute_RepoAssignProductError tests repository error during AssignProduct.
func TestAssignConnection_Execute_RepoAssignProductError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo, mockProductRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()

	existingProduct := newExistingProduct(orgID, productID)
	existingConn := newExistingConnectionForAssign(orgID, connID)

	dbError := errors.New("failed to update in database")

	// Mock: product found
	mockProductRepo.EXPECT().
		FindByID(gomock.Any(), productID, orgID).
		Return(existingProduct, nil)

	// Mock: connection found (unassigned)
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: AssignProduct returns error
	mockConnRepo.EXPECT().
		AssignProduct(gomock.Any(), connID, orgID, productID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, connID, productID)

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

// TestNewAssignConnection verifies the constructor.
func TestNewAssignConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo, mockProductRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.connRepo == nil {
		t.Fatal("expected connRepo to be set")
	}

	if svc.productRepo == nil {
		t.Fatal("expected productRepo to be set")
	}
}

// TestAssignConnection_Execute_TableDriven uses table-driven tests for various scenarios.
func TestAssignConnection_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*connRepo.MockRepository, *productMock.MockRepository, uuid.UUID, uuid.UUID, uuid.UUID)
		wantErr        bool
		wantStatusCode int // 0 means generic error (no status code check)
	}{
		{
			name: "successful assignment",
			setupMocks: func(connMock *connRepo.MockRepository, prodMock *productMock.MockRepository, orgID, connID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(newExistingProduct(orgID, productID), nil)
				conn := newExistingConnectionForAssign(orgID, connID)
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(conn, nil)
				updatedConn := newExistingConnectionForAssign(orgID, connID)
				updatedConn.ProductID = &productID
				connMock.EXPECT().
					AssignProduct(gomock.Any(), connID, orgID, productID).
					Return(updatedConn, nil)
			},
			wantErr: false,
		},
		{
			name: "product not found",
			setupMocks: func(connMock *connRepo.MockRepository, prodMock *productMock.MockRepository, orgID, connID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "connection not found",
			setupMocks: func(connMock *connRepo.MockRepository, prodMock *productMock.MockRepository, orgID, connID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(newExistingProduct(orgID, productID), nil)
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "connection already assigned",
			setupMocks: func(connMock *connRepo.MockRepository, prodMock *productMock.MockRepository, orgID, connID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(newExistingProduct(orgID, productID), nil)
				conn := newExistingConnectionForAssign(orgID, connID)
				existingPID := uuid.New()
				conn.ProductID = &existingPID
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(conn, nil)
				// Atomic guard rejects — product_id is not nil
				connMock.EXPECT().
					AssignProduct(gomock.Any(), connID, orgID, productID).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusConflict,
		},
		{
			name: "product repo database error",
			setupMocks: func(connMock *connRepo.MockRepository, prodMock *productMock.MockRepository, orgID, connID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr:        true,
			wantStatusCode: 0, // generic error
		},
		{
			name: "connection repo database error",
			setupMocks: func(connMock *connRepo.MockRepository, prodMock *productMock.MockRepository, orgID, connID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(newExistingProduct(orgID, productID), nil)
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr:        true,
			wantStatusCode: 0, // generic error
		},
		{
			name: "assign product repo returns nil (race condition)",
			setupMocks: func(connMock *connRepo.MockRepository, prodMock *productMock.MockRepository, orgID, connID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(newExistingProduct(orgID, productID), nil)
				conn := newExistingConnectionForAssign(orgID, connID)
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(conn, nil)
				connMock.EXPECT().
					AssignProduct(gomock.Any(), connID, orgID, productID).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusConflict,
		},
		{
			name: "assign product repo error",
			setupMocks: func(connMock *connRepo.MockRepository, prodMock *productMock.MockRepository, orgID, connID, productID uuid.UUID) {
				prodMock.EXPECT().
					FindByID(gomock.Any(), productID, orgID).
					Return(newExistingProduct(orgID, productID), nil)
				conn := newExistingConnectionForAssign(orgID, connID)
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(conn, nil)
				connMock.EXPECT().
					AssignProduct(gomock.Any(), connID, orgID, productID).
					Return(nil, errors.New("failed to update in database"))
			},
			wantErr:        true,
			wantStatusCode: 0, // generic error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockProductRepo := productMock.NewMockRepository(ctrl)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()
			productID := uuid.New()

			tt.setupMocks(mockConnRepo, mockProductRepo, orgID, connID, productID)

			svc := NewAssignConnection(mockConnRepo, mockProductRepo)

			result, err := svc.Execute(ctx, orgID, connID, productID)

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
