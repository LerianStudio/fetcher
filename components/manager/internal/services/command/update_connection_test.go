package command

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// newExistingConnection creates a valid existing Connection for testing updates.
func newExistingConnection(orgID, connID uuid.UUID) *model.Connection {
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

// newUpdateConnectionInput creates a valid ConnectionInput for testing updates.
func newUpdateConnectionInput() model.ConnectionInput {
	return model.ConnectionInput{
		ConfigName:   "updated-connection",
		Type:         "POSTGRESQL",
		Host:         "new-host.example.com",
		Port:         5433,
		DatabaseName: "newdb",
		Username:     "newuser",
		Password:     "newpassword",
	}
}

// TestUpdateConnection_Execute_Success tests successful connection update.
func TestUpdateConnection_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnection(orgID, connID)
	input := newUpdateConnectionInput()

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs for this connection
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	// Mock: update returns the updated connection
	mockConnRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
			return conn, nil
		})

	result, err := svc.Execute(ctx, orgID, connID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.ConfigName != input.ConfigName {
		t.Fatalf("expected ConfigName %s, got %s", input.ConfigName, result.ConfigName)
	}

	if result.Host != input.Host {
		t.Fatalf("expected Host %s, got %s", input.Host, result.Host)
	}

	if result.Port != input.Port {
		t.Fatalf("expected Port %d, got %d", input.Port, result.Port)
	}
}

// TestUpdateConnection_Execute_NotFoundError tests that non-existent connection returns not found error.
func TestUpdateConnection_Execute_NotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	input := newUpdateConnectionInput()

	// Mock: connection not found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID, input)

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

// TestUpdateConnection_Execute_FindByIDError tests repository error during FindByID.
func TestUpdateConnection_Execute_FindByIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	input := newUpdateConnectionInput()

	dbError := errors.New("database connection failed")

	// Mock: database error during lookup
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, connID, input)

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

// TestUpdateConnection_Execute_ActiveJobError tests that update fails when there are active jobs.
func TestUpdateConnection_Execute_ActiveJobError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnection(orgID, connID)
	input := newUpdateConnectionInput()

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: active jobs exist for this connection
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(true, nil)

	result, err := svc.Execute(ctx, orgID, connID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for active jobs, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusConflict, respErr.StatusCode)
}

// TestUpdateConnection_Execute_ExistsRunningJobError tests repository error during job check.
func TestUpdateConnection_Execute_ExistsRunningJobError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnection(orgID, connID)
	input := newUpdateConnectionInput()

	dbError := errors.New("database connection failed")

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: error during job check
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, dbError)

	result, err := svc.Execute(ctx, orgID, connID, input)

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

// TestUpdateConnection_Execute_UpdateError tests repository error during Update.
func TestUpdateConnection_Execute_UpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnection(orgID, connID)
	input := newUpdateConnectionInput()

	dbError := errors.New("failed to update in database")

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	// Mock: update returns error
	mockConnRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, connID, input)

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

// TestUpdateConnection_Execute_UpdateReturnsNil tests that update returning nil triggers not found error.
func TestUpdateConnection_Execute_UpdateReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnection(orgID, connID)
	input := newUpdateConnectionInput()

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	// Mock: update returns nil (connection not found during update)
	mockConnRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, connID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for nil update result, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}

// TestUpdateConnection_Execute_EncryptionError tests encryption failure during password update.
func TestUpdateConnection_Execute_EncryptionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	encryptionError := errors.New("encryption key invalid")
	mockCrypto := &mockCryptor{
		encryptFunc: func(ctx context.Context, plain string) (string, string, error) {
			return "", "", encryptionError
		},
	}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnection(orgID, connID)
	input := newUpdateConnectionInput()

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	result, err := svc.Execute(ctx, orgID, connID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for encryption failure, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestUpdateConnection_Execute_PartialUpdate tests that partial updates work correctly.
func TestUpdateConnection_Execute_PartialUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnection(orgID, connID)

	// Only update the host, keep other fields the same
	input := model.ConnectionInput{
		ConfigName:   existingConn.ConfigName,
		Type:         string(existingConn.Type),
		Host:         "new-host.example.com",
		Port:         existingConn.Port,
		DatabaseName: existingConn.DatabaseName,
		Username:     existingConn.Username,
		Password:     "newpassword",
	}

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	// Mock: update returns the updated connection
	mockConnRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
			// Verify only host was changed
			if conn.Host != "new-host.example.com" {
				t.Errorf("expected Host to be updated to 'new-host.example.com', got %s", conn.Host)
			}
			if conn.ConfigName != existingConn.ConfigName {
				t.Errorf("expected ConfigName to remain %s, got %s", existingConn.ConfigName, conn.ConfigName)
			}
			return conn, nil
		})

	result, err := svc.Execute(ctx, orgID, connID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Host != "new-host.example.com" {
		t.Fatalf("expected Host 'new-host.example.com', got %s", result.Host)
	}
}

// TestUpdateConnection_Execute_WithSSL tests connection update with SSL configuration.
func TestUpdateConnection_Execute_WithSSL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnection(orgID, connID)

	certValue := "-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----"
	keyValue := "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----"

	input := model.ConnectionInput{
		ConfigName:   "ssl-connection",
		Type:         "POSTGRESQL",
		Host:         "secure.example.com",
		Port:         5432,
		DatabaseName: "securedb",
		Username:     "secureuser",
		Password:     "securepassword",
		SSL: &model.SSLInput{
			Mode: "require",
			CA:   "-----BEGIN CERTIFICATE-----\ntest-ca\n-----END CERTIFICATE-----",
			Cert: &certValue,
			Key:  &keyValue,
		},
	}

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	// Mock: update returns the connection
	mockConnRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
			// Verify SSL was set correctly
			if conn.SSL == nil {
				t.Error("expected SSL to be set")
			} else {
				if conn.SSL.Mode != "require" {
					t.Errorf("expected SSL mode 'require', got %s", conn.SSL.Mode)
				}
			}
			return conn, nil
		})

	result, err := svc.Execute(ctx, orgID, connID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.SSL == nil {
		t.Fatal("expected SSL configuration to be present")
	}

	if result.SSL.Mode != "require" {
		t.Fatalf("expected SSL mode 'require', got %s", result.SSL.Mode)
	}
}

// TestUpdateConnection_Execute_InvalidTypeError tests that invalid database type returns error.
func TestUpdateConnection_Execute_InvalidTypeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnection(orgID, connID)

	input := model.ConnectionInput{
		ConfigName:   "test-connection",
		Type:         "INVALID_TYPE",
		Host:         "localhost",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpassword",
	}

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	result, err := svc.Execute(ctx, orgID, connID, input)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}

	var internalErr pkg.ValidationKnownFieldsError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestNewUpdateConnection verifies the constructor.
func TestNewUpdateConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCrypto := &mockCryptor{}

	svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.connRepo == nil {
		t.Fatal("expected connRepo to be set")
	}

	if svc.jobRepo == nil {
		t.Fatal("expected jobRepo to be set")
	}

	if svc.cryptor == nil {
		t.Fatal("expected cryptor to be set")
	}
}

// TestUpdateConnection_Execute_TableDriven uses table-driven tests for various scenarios.
func TestUpdateConnection_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*connRepo.MockRepository, *jobRepo.MockRepository, uuid.UUID, uuid.UUID, *model.Connection)
		input          model.ConnectionInput
		mockCrypto     *mockCryptor
		wantErr        bool
		wantStatusCode int // 0 means no status code check
		validateResult func(*testing.T, *model.Connection)
	}{
		{
			name: "successful update with all fields",
			setupMocks: func(connMock *connRepo.MockRepository, jobMock *jobRepo.MockRepository, orgID, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(existing, nil)
				jobMock.EXPECT().
					ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existing.ConfigName).
					Return(false, nil)
				connMock.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
						return conn, nil
					})
			},
			input: model.ConnectionInput{
				ConfigName:   "new-name",
				Type:         "MYSQL",
				Host:         "new-host",
				Port:         3306,
				DatabaseName: "newdb",
				Username:     "newuser",
				Password:     "newpassword",
			},
			mockCrypto: &mockCryptor{},
			wantErr:    false,
			validateResult: func(t *testing.T, result *model.Connection) {
				if result.ConfigName != "new-name" {
					t.Errorf("expected ConfigName 'new-name', got %s", result.ConfigName)
				}
				if result.Type != model.TypeMySQL {
					t.Errorf("expected Type MySQL, got %s", result.Type)
				}
			},
		},
		{
			name: "connection not found",
			setupMocks: func(connMock *connRepo.MockRepository, jobMock *jobRepo.MockRepository, orgID, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(nil, nil)
			},
			input:          newUpdateConnectionInput(),
			mockCrypto:     &mockCryptor{},
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "active jobs prevent update",
			setupMocks: func(connMock *connRepo.MockRepository, jobMock *jobRepo.MockRepository, orgID, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(existing, nil)
				jobMock.EXPECT().
					ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existing.ConfigName).
					Return(true, nil)
			},
			input:          newUpdateConnectionInput(),
			mockCrypto:     &mockCryptor{},
			wantErr:        true,
			wantStatusCode: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockJobRepo := jobRepo.NewMockRepository(ctrl)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()
			existingConn := newExistingConnection(orgID, connID)

			tt.setupMocks(mockConnRepo, mockJobRepo, orgID, connID, existingConn)

			svc := NewUpdateConnection(mockConnRepo, mockJobRepo, tt.mockCrypto)

			result, err := svc.Execute(ctx, orgID, connID, tt.input)

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

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

// TestUpdateConnection_Execute_DatabaseTypeChange tests changing database type.
func TestUpdateConnection_Execute_DatabaseTypeChange(t *testing.T) {
	tests := []struct {
		name         string
		newType      string
		expectedType model.DBType
	}{
		{
			name:         "Change to MySQL",
			newType:      "MYSQL",
			expectedType: model.TypeMySQL,
		},
		{
			name:         "Change to MongoDB",
			newType:      "MONGODB",
			expectedType: model.TypeMongoDB,
		},
		{
			name:         "Change to Oracle",
			newType:      "ORACLE",
			expectedType: model.TypeOracle,
		},
		{
			name:         "Change to SQL Server",
			newType:      "SQL_SERVER",
			expectedType: model.TypeSQLServer,
		},
		{
			name:         "Change to PostgreSQL (lowercase)",
			newType:      "postgresql",
			expectedType: model.TypePostgreSQL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockJobRepo := jobRepo.NewMockRepository(ctrl)
			mockCrypto := &mockCryptor{}

			svc := NewUpdateConnection(mockConnRepo, mockJobRepo, mockCrypto)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()
			existingConn := newExistingConnection(orgID, connID)

			input := model.ConnectionInput{
				ConfigName:   existingConn.ConfigName,
				Type:         tt.newType,
				Host:         existingConn.Host,
				Port:         existingConn.Port,
				DatabaseName: existingConn.DatabaseName,
				Username:     existingConn.Username,
				Password:     "newpassword",
			}

			// Mock: find existing connection
			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID, orgID).
				Return(existingConn, nil)

			// Mock: no active jobs
			mockJobRepo.EXPECT().
				ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
				Return(false, nil)

			// Mock: update returns the connection
			mockConnRepo.EXPECT().
				Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, conn *model.Connection) (*model.Connection, error) {
					return conn, nil
				})

			result, err := svc.Execute(ctx, orgID, connID, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Type != tt.expectedType {
				t.Fatalf("expected Type %s, got %s", tt.expectedType, result.Type)
			}
		})
	}
}
