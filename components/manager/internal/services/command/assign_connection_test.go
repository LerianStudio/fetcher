package command

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newExistingConnectionForAssign creates a valid existing Connection without a ProductName (unassigned).
func newExistingConnectionForAssign(connID uuid.UUID) *model.Connection {
	return &model.Connection{
		ID:                   connID,
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

// TestAssignConnection_Execute_Success tests successful connection assignment to a product.
func TestAssignConnection_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo)

	ctx := testContext()
	connID := uuid.New()
	productName := "reporter"

	existingConn := newExistingConnectionForAssign(connID)

	updatedConn := newExistingConnectionForAssign(connID)
	updatedConn.ProductName = productName
	updatedConn.UpdatedAt = time.Now().UTC()

	// Mock: connection found (unassigned)
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	// Mock: assign product name succeeds
	mockConnRepo.EXPECT().
		AssignProductName(gomock.Any(), connID, productName).
		Return(updatedConn, nil)

	result, err := svc.Execute(ctx, connID, productName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	assert.Equal(t, productName, result.ProductName)
	assert.Equal(t, connID, result.ID)
}

// TestAssignConnection_Execute_ConnectionNotFound tests that a non-existent connection returns not found error.
func TestAssignConnection_Execute_ConnectionNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo)

	ctx := testContext()
	connID := uuid.New()

	// Mock: connection not found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, connID, "reporter")

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

	svc := NewAssignConnection(mockConnRepo)

	ctx := testContext()
	connID := uuid.New()

	dbError := errors.New("database connection failed")

	// Mock: connection repo returns error
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, connID, "reporter")

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
// that already has a ProductName returns a 409 conflict error.
func TestAssignConnection_Execute_ConnectionAlreadyAssigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo)

	ctx := testContext()
	connID := uuid.New()

	existingConn := newExistingConnectionForAssign(connID)
	existingConn.ProductName = "existing-product" // Already assigned

	// Mock: connection found (already assigned)
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	// Mock: AssignProductName returns nil (atomic guard rejects -- product_name is not empty)
	mockConnRepo.EXPECT().
		AssignProductName(gomock.Any(), connID, "reporter").
		Return(nil, nil)

	result, err := svc.Execute(ctx, connID, "reporter")

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
// where the repo AssignProductName returns (nil, nil), indicating the connection was assigned
// between the domain check and the persistence layer.
func TestAssignConnection_Execute_RepoAssignProductReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo)

	ctx := testContext()
	connID := uuid.New()

	existingConn := newExistingConnectionForAssign(connID)

	// Mock: connection found (unassigned)
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	// Mock: AssignProductName returns nil (race condition -- assigned between check and persist)
	mockConnRepo.EXPECT().
		AssignProductName(gomock.Any(), connID, "reporter").
		Return(nil, nil)

	result, err := svc.Execute(ctx, connID, "reporter")

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

// TestAssignConnection_Execute_RepoAssignProductError tests repository error during AssignProductName.
func TestAssignConnection_Execute_RepoAssignProductError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo)

	ctx := testContext()
	connID := uuid.New()

	existingConn := newExistingConnectionForAssign(connID)

	dbError := errors.New("failed to update in database")

	// Mock: connection found (unassigned)
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID).
		Return(existingConn, nil)

	// Mock: AssignProductName returns error
	mockConnRepo.EXPECT().
		AssignProductName(gomock.Any(), connID, "reporter").
		Return(nil, dbError)

	result, err := svc.Execute(ctx, connID, "reporter")

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

	svc := NewAssignConnection(mockConnRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.connRepo == nil {
		t.Fatal("expected connRepo to be set")
	}
}

// TestAssignConnection_Execute_TableDriven uses table-driven tests for various scenarios.
func TestAssignConnection_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*connRepo.MockRepository, uuid.UUID, string)
		wantErr        bool
		wantStatusCode int // 0 means generic error (no status code check)
	}{
		{
			name: "successful assignment",
			setupMocks: func(connMock *connRepo.MockRepository, connID uuid.UUID, productName string) {
				conn := newExistingConnectionForAssign(connID)
				connMock.EXPECT().
					FindByID(gomock.Any(), connID).
					Return(conn, nil)
				updatedConn := newExistingConnectionForAssign(connID)
				updatedConn.ProductName = productName
				connMock.EXPECT().
					AssignProductName(gomock.Any(), connID, productName).
					Return(updatedConn, nil)
			},
			wantErr: false,
		},
		{
			name: "connection not found",
			setupMocks: func(connMock *connRepo.MockRepository, connID uuid.UUID, productName string) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "connection already assigned",
			setupMocks: func(connMock *connRepo.MockRepository, connID uuid.UUID, productName string) {
				conn := newExistingConnectionForAssign(connID)
				conn.ProductName = "existing-product"
				connMock.EXPECT().
					FindByID(gomock.Any(), connID).
					Return(conn, nil)
				// Atomic guard rejects -- product_name is not empty
				connMock.EXPECT().
					AssignProductName(gomock.Any(), connID, productName).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusConflict,
		},
		{
			name: "connection repo database error",
			setupMocks: func(connMock *connRepo.MockRepository, connID uuid.UUID, productName string) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr:        true,
			wantStatusCode: 0, // generic error
		},
		{
			name: "assign product repo returns nil (race condition)",
			setupMocks: func(connMock *connRepo.MockRepository, connID uuid.UUID, productName string) {
				conn := newExistingConnectionForAssign(connID)
				connMock.EXPECT().
					FindByID(gomock.Any(), connID).
					Return(conn, nil)
				connMock.EXPECT().
					AssignProductName(gomock.Any(), connID, productName).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusConflict,
		},
		{
			name: "assign product repo error",
			setupMocks: func(connMock *connRepo.MockRepository, connID uuid.UUID, productName string) {
				conn := newExistingConnectionForAssign(connID)
				connMock.EXPECT().
					FindByID(gomock.Any(), connID).
					Return(conn, nil)
				connMock.EXPECT().
					AssignProductName(gomock.Any(), connID, productName).
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

			ctx := testContext()
			connID := uuid.New()
			productName := "reporter"

			tt.setupMocks(mockConnRepo, connID, productName)

			svc := NewAssignConnection(mockConnRepo)

			result, err := svc.Execute(ctx, connID, productName)

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

// TestAssignConnection_Execute_EmptyProductName tests that an empty product name
// returns a validation error before any repository calls are made.
func TestAssignConnection_Execute_EmptyProductName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	svc := NewAssignConnection(mockConnRepo)

	ctx := testContext()
	connID := uuid.New()

	// No mock expectations: validation returns before any repository calls
	result, err := svc.Execute(ctx, connID, "")

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for empty product name, got nil")
	}

	var knownFieldsErr pkg.ValidationKnownFieldsError
	if !errors.As(err, &knownFieldsErr) {
		t.Fatalf("expected ValidationKnownFieldsError, got %T: %v", err, err)
	}

	if _, exists := knownFieldsErr.Fields["product_name"]; !exists {
		t.Fatalf("expected 'product_name' in error fields, got %v", knownFieldsErr.Fields)
	}

	assert.Equal(t, "product name is required", knownFieldsErr.Fields["product_name"])
}
