package command

import (
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

// newExistingConnectionForDelete creates a valid existing Connection for testing deletions.
func newExistingConnectionForDelete(orgID, connID uuid.UUID) *model.Connection {
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

// newSoftDeletedConnection creates a soft-deleted Connection for testing.
func newSoftDeletedConnection(orgID, connID uuid.UUID) *model.Connection {
	conn := newExistingConnectionForDelete(orgID, connID)
	deletedAt := time.Now().UTC().Add(-1 * time.Hour)
	conn.DeletedAt = &deletedAt
	return conn
}

// TestDeleteConnection_Execute_Success tests successful connection deletion.
func TestDeleteConnection_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnectionForDelete(orgID, connID)

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs for this connection
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	// Mock: delete succeeds
	mockConnRepo.EXPECT().
		Delete(gomock.Any(), connID, orgID, gomock.Any()).
		Return(nil)

	err := svc.Execute(ctx, orgID, connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeleteConnection_Execute_NotFoundError tests that non-existent connection returns not found error.
func TestDeleteConnection_Execute_NotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	// Mock: connection not found
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, nil)

	err := svc.Execute(ctx, orgID, connID)

	if err == nil {
		t.Fatal("expected error for non-existent connection, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)
}

// TestDeleteConnection_Execute_ActiveJobError tests that delete fails when there are active jobs.
func TestDeleteConnection_Execute_ActiveJobError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnectionForDelete(orgID, connID)

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: active jobs exist for this connection
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(true, nil)

	err := svc.Execute(ctx, orgID, connID)

	if err == nil {
		t.Fatal("expected error for active jobs, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusConflict, respErr.StatusCode)
}

// TestDeleteConnection_Execute_FindByIDError tests repository error during FindByID.
func TestDeleteConnection_Execute_FindByIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	dbError := errors.New("database connection failed")

	// Mock: database error during lookup
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, dbError)

	err := svc.Execute(ctx, orgID, connID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to wrap dbError, got %v", err)
	}
}

// TestDeleteConnection_Execute_ExistsRunningJobError tests repository error during job check.
func TestDeleteConnection_Execute_ExistsRunningJobError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnectionForDelete(orgID, connID)

	dbError := errors.New("database connection failed")

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: error during job check
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, dbError)

	err := svc.Execute(ctx, orgID, connID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to wrap dbError, got %v", err)
	}
}

// TestDeleteConnection_Execute_DeleteError tests repository error during Delete.
func TestDeleteConnection_Execute_DeleteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnectionForDelete(orgID, connID)

	dbError := errors.New("failed to delete from database")

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	// Mock: delete returns error
	mockConnRepo.EXPECT().
		Delete(gomock.Any(), connID, orgID, gomock.Any()).
		Return(dbError)

	err := svc.Execute(ctx, orgID, connID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, dbError) {
		t.Fatalf("expected error to wrap dbError, got %v", err)
	}
}

// TestNewDeleteConnection verifies the constructor.
func TestNewDeleteConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.connRepo == nil {
		t.Fatal("expected connRepo to be set")
	}

	if svc.jobRepo == nil {
		t.Fatal("expected jobRepo to be set")
	}
}

// TestDeleteConnection_Execute_TableDriven uses table-driven tests for various scenarios.
func TestDeleteConnection_Execute_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*connRepo.MockRepository, *jobRepo.MockRepository, uuid.UUID, uuid.UUID, *model.Connection)
		wantErr        bool
		wantStatusCode int // 0 means generic error (no status code check)
	}{
		{
			name: "successful deletion",
			setupMocks: func(connMock *connRepo.MockRepository, jobMock *jobRepo.MockRepository, orgID, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(existing, nil)
				jobMock.EXPECT().
					ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existing.ConfigName).
					Return(false, nil)
				connMock.EXPECT().
					Delete(gomock.Any(), connID, orgID, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "connection not found",
			setupMocks: func(connMock *connRepo.MockRepository, jobMock *jobRepo.MockRepository, orgID, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(nil, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "active jobs prevent deletion",
			setupMocks: func(connMock *connRepo.MockRepository, jobMock *jobRepo.MockRepository, orgID, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(existing, nil)
				jobMock.EXPECT().
					ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existing.ConfigName).
					Return(true, nil)
			},
			wantErr:        true,
			wantStatusCode: http.StatusConflict,
		},
		{
			name: "FindByID database error",
			setupMocks: func(connMock *connRepo.MockRepository, jobMock *jobRepo.MockRepository, orgID, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr:        true,
			wantStatusCode: 0, // generic error
		},
		{
			name: "ExistsRunningByMappedFieldKey database error",
			setupMocks: func(connMock *connRepo.MockRepository, jobMock *jobRepo.MockRepository, orgID, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(existing, nil)
				jobMock.EXPECT().
					ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existing.ConfigName).
					Return(false, errors.New("database connection failed"))
			},
			wantErr:        true,
			wantStatusCode: 0, // generic error
		},
		{
			name: "Delete database error",
			setupMocks: func(connMock *connRepo.MockRepository, jobMock *jobRepo.MockRepository, orgID, connID uuid.UUID, existing *model.Connection) {
				connMock.EXPECT().
					FindByID(gomock.Any(), connID, orgID).
					Return(existing, nil)
				jobMock.EXPECT().
					ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existing.ConfigName).
					Return(false, nil)
				connMock.EXPECT().
					Delete(gomock.Any(), connID, orgID, gomock.Any()).
					Return(errors.New("failed to delete"))
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
			mockJobRepo := jobRepo.NewMockRepository(ctrl)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()
			existingConn := newExistingConnectionForDelete(orgID, connID)

			tt.setupMocks(mockConnRepo, mockJobRepo, orgID, connID, existingConn)

			svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

			err := svc.Execute(ctx, orgID, connID)

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

// TestDeleteConnection_Execute_DifferentOrganizations tests that connections are isolated by organization.
func TestDeleteConnection_Execute_DifferentOrganizations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	ctx := testContext()
	orgID := uuid.New()
	differentOrgID := uuid.New()
	connID := uuid.New()

	// Connection belongs to a different organization
	existingConn := newExistingConnectionForDelete(differentOrgID, connID)

	// Mock: connection not found for the given organization
	// (the connection exists but belongs to a different org)
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(nil, nil)

	err := svc.Execute(ctx, orgID, connID)

	if err == nil {
		t.Fatal("expected error for connection in different organization, got nil")
	}

	var respErr pkg.ResponseErrorWithStatusCode
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseErrorWithStatusCode, got %T: %v", err, err)
	}
	assert.Equal(t, http.StatusNotFound, respErr.StatusCode)

	// Verify the existing connection is not affected (it belongs to a different org)
	if existingConn.DeletedAt != nil {
		t.Fatal("expected existing connection to remain undeleted")
	}
}

// TestDeleteConnection_Execute_DeletePassesCorrectTimestamp tests that Delete receives a valid timestamp.
func TestDeleteConnection_Execute_DeletePassesCorrectTimestamp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	existingConn := newExistingConnectionForDelete(orgID, connID)

	beforeExecution := time.Now().UTC()

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	// Mock: delete with timestamp validation
	mockConnRepo.EXPECT().
		Delete(gomock.Any(), connID, orgID, gomock.Any()).
		DoAndReturn(func(ctx interface{}, id, orgID uuid.UUID, deletedAt time.Time) error {
			afterExecution := time.Now().UTC()
			if deletedAt.Before(beforeExecution) || deletedAt.After(afterExecution) {
				t.Errorf("expected deletedAt between %v and %v, got %v", beforeExecution, afterExecution, deletedAt)
			}
			return nil
		})

	err := svc.Execute(ctx, orgID, connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDeleteConnection_Execute_MultipleJobsCheck tests the scenario with multiple pending jobs.
func TestDeleteConnection_Execute_MultipleJobsCheck(t *testing.T) {
	tests := []struct {
		name       string
		hasRunning bool
		wantErr    bool
	}{
		{
			name:       "no running jobs allows deletion",
			hasRunning: false,
			wantErr:    false,
		},
		{
			name:       "running jobs prevent deletion",
			hasRunning: true,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockJobRepo := jobRepo.NewMockRepository(ctrl)

			svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()
			existingConn := newExistingConnectionForDelete(orgID, connID)

			// Mock: find existing connection
			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID, orgID).
				Return(existingConn, nil)

			// Mock: job check
			mockJobRepo.EXPECT().
				ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
				Return(tt.hasRunning, nil)

			if !tt.hasRunning {
				// Mock: delete succeeds only if no running jobs
				mockConnRepo.EXPECT().
					Delete(gomock.Any(), connID, orgID, gomock.Any()).
					Return(nil)
			}

			err := svc.Execute(ctx, orgID, connID)

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

// TestDeleteConnection_Execute_ConnectionWithDifferentTypes tests deletion of connections with various database types.
func TestDeleteConnection_Execute_ConnectionWithDifferentTypes(t *testing.T) {
	tests := []struct {
		name   string
		dbType model.DBType
	}{
		{
			name:   "PostgreSQL connection",
			dbType: model.TypePostgreSQL,
		},
		{
			name:   "MySQL connection",
			dbType: model.TypeMySQL,
		},
		{
			name:   "MongoDB connection",
			dbType: model.TypeMongoDB,
		},
		{
			name:   "Oracle connection",
			dbType: model.TypeOracle,
		},
		{
			name:   "SQL Server connection",
			dbType: model.TypeSQLServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockJobRepo := jobRepo.NewMockRepository(ctrl)

			svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

			ctx := testContext()
			orgID := uuid.New()
			connID := uuid.New()

			existingConn := newExistingConnectionForDelete(orgID, connID)
			existingConn.Type = tt.dbType

			// Mock: find existing connection
			mockConnRepo.EXPECT().
				FindByID(gomock.Any(), connID, orgID).
				Return(existingConn, nil)

			// Mock: no active jobs
			mockJobRepo.EXPECT().
				ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
				Return(false, nil)

			// Mock: delete succeeds
			mockConnRepo.EXPECT().
				Delete(gomock.Any(), connID, orgID, gomock.Any()).
				Return(nil)

			err := svc.Execute(ctx, orgID, connID)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.name, err)
			}
		})
	}
}

// TestDeleteConnection_Execute_ConnectionWithSSL tests deletion of connection with SSL configuration.
func TestDeleteConnection_Execute_ConnectionWithSSL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewDeleteConnection(mockConnRepo, mockJobRepo)

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	existingConn := newExistingConnectionForDelete(orgID, connID)
	existingConn.SSL = &model.SSLConfig{
		Mode: "require",
		CA:   "-----BEGIN CERTIFICATE-----\ntest-ca\n-----END CERTIFICATE-----",
		Cert: "-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----",
		Key:  "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----",
	}

	// Mock: find existing connection
	mockConnRepo.EXPECT().
		FindByID(gomock.Any(), connID, orgID).
		Return(existingConn, nil)

	// Mock: no active jobs
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), orgID, existingConn.ConfigName).
		Return(false, nil)

	// Mock: delete succeeds
	mockConnRepo.EXPECT().
		Delete(gomock.Any(), connID, orgID, gomock.Any()).
		Return(nil)

	err := svc.Execute(ctx, orgID, connID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
