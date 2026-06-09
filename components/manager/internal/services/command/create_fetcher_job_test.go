package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/v2/pkg/model/job"
	jobRepo "github.com/LerianStudio/fetcher/v2/pkg/mongodb/job"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/v2/pkg/ports/messaging"
	pkgRabbitMQ "github.com/LerianStudio/fetcher/v2/pkg/rabbitmq"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/mock/gomock"
)

// newValidFetcherRequest creates a valid FetcherRequest for testing.
func newValidFetcherRequest() model.FetcherRequest {
	return model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"datasource1": {
					"table1": {"field1", "field2"},
				},
			},
		},
		Metadata: map[string]any{"source": "test-product", "key": "value"},
	}
}

// newConnectionWithEncryption creates a connection with encryption fields set for testing.
func newConnectionWithEncryption(id uuid.UUID, configName string, dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID:                   id,
		ProductName:          "test-product",
		ConfigName:           configName,
		Type:                 dbType,
		Host:                 "localhost",
		Port:                 5432,
		DatabaseName:         "testdb",
		Username:             "testuser",
		PasswordEncrypted:    "encrypted-password",
		EncryptionKeyVersion: "v1",
	}
}

// TestCreateFetcherJob_Execute_ValidationError tests that invalid requests return validation errors.
func TestCreateFetcherJob_Execute_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

	tests := []struct {
		name    string
		request model.FetcherRequest
		wantErr string
	}{
		{
			name: "nil mappedFields",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: nil,
				},
			},
			wantErr: "mappedFields is required",
		},
		{
			name: "empty mappedFields",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{},
				},
			},
			wantErr: "mappedFields cannot be empty",
		},
		{
			name: "datasource with empty fields",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						"datasource1": {
							"table1": {},
						},
					},
				},
			},
			wantErr: "datasource must have at least one table with fields",
		},
		{
			name: "nil metadata",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						"ds1": {"table1": {"field1"}},
					},
				},
				Metadata: nil,
			},
			wantErr: "metadata is required and must contain 'source' field",
		},
		{
			name: "metadata without source",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						"ds1": {"table1": {"field1"}},
					},
				},
				Metadata: map[string]any{"key": "value"},
			},
			wantErr: "metadata.source is required for job notification routing",
		},
		{
			name: "metadata with empty source",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						"ds1": {"table1": {"field1"}},
					},
				},
				Metadata: map[string]any{"source": ""},
			},
			wantErr: "metadata.source must be a non-empty string",
		},
		{
			name: "metadata with whitespace-only source",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						"ds1": {"table1": {"field1"}},
					},
				},
				Metadata: map[string]any{"source": "   \t\n  "},
			},
			wantErr: "metadata.source must be a non-empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext()

			result, err := svc.Execute(ctx, tt.request)

			if result != nil {
				t.Fatalf("expected nil result for invalid request, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error for invalid request, got nil")
			}

			var validationErr pkg.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}

			if validationErr.Message != tt.wantErr {
				t.Fatalf("expected error message %q, got %q", tt.wantErr, validationErr.Message)
			}
		})
	}
}

// TestCreateFetcherJob_Execute_DuplicateWithinWindow tests that duplicate requests return existing job.
func TestCreateFetcherJob_Execute_DuplicateWithinWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

	ctx := testContext()
	request := newValidFetcherRequest()

	existingJobID := uuid.New()
	existingJob := &model.Job{
		ID:        existingJobID,
		Status:    model.JobStatusPending,
		CreatedAt: time.Now().UTC().Add(-2 * time.Minute),
	}

	// Mock: find existing job within window
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(existingJob, nil)

	result, err := svc.Execute(ctx, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.IsDuplicate {
		t.Fatal("expected IsDuplicate to be true")
	}

	if result.IsNewCreated {
		t.Fatal("expected IsNewCreated to be false")
	}

	if result.Job.ID != existingJobID {
		t.Fatalf("expected job ID %s, got %s", existingJobID, result.Job.ID)
	}
}

// TestCreateFetcherJob_Execute_NoConnectionsFound tests that when no connections are found
// for any of the requested datasources, a validation error is returned.
func TestCreateFetcherJob_Execute_NoConnectionsFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

	ctx := testContext()
	request := newValidFetcherRequest()

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: no connections found at all
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{}, nil)

	result, err := svc.Execute(ctx, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for no connections found, got nil")
	}

	var validationErr pkg.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	// When NO connections are found at all, production returns ErrMissingDataSource
	if validationErr.Code != constant.ErrMissingDataSource.Error() {
		t.Fatalf("expected error code %s, got %s", constant.ErrMissingDataSource.Error(), validationErr.Code)
	}
}

// TestCreateFetcherJob_Execute_TooManyDatasources tests that too many datasources returns error.
func TestCreateFetcherJob_Execute_TooManyDatasources(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

	ctx := testContext()

	// Create request with 11 datasources (exceeds MaxDatasourcesPerJob = 10)
	mappedFields := make(map[string]map[string][]string)
	for i := 0; i < 11; i++ {
		dsName := fmt.Sprintf("datasource_%d", i)
		mappedFields[dsName] = map[string][]string{
			"table1": {"field1"},
		}
	}

	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: mappedFields,
		},
	}

	// No mock expectations needed - validation fails before any repository calls

	result, err := svc.Execute(ctx, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for too many datasources, got nil")
	}

	var validationErr pkg.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	if validationErr.Code != constant.ErrInvalidDataRequest.Error() {
		t.Fatalf("expected error code %s, got %s", constant.ErrInvalidDataRequest.Error(), validationErr.Code)
	}
}

// TestCreateFetcherJob_Execute_FindByRequestHashError tests internal error during hash lookup.
func TestCreateFetcherJob_Execute_FindByRequestHashError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

	ctx := testContext()
	request := newValidFetcherRequest()

	dbError := errors.New("database connection failed")

	// Mock: database error during duplicate check
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestCreateFetcherJob_Execute_FindByConfigNamesError tests internal error during connection lookup.
func TestCreateFetcherJob_Execute_FindByConfigNamesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

	ctx := testContext()
	request := newValidFetcherRequest()

	dbError := errors.New("database connection failed")

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: database error during connection lookup
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestCreateFetcherJob_Constants verifies the service constants are as expected.
func TestCreateFetcherJob_Constants(t *testing.T) {
	if DeduplicationWindowMinutes != 5 {
		t.Fatalf("expected DeduplicationWindowMinutes to be 5, got %d", DeduplicationWindowMinutes)
	}

	if model.MaxDatasourcesPerJob != 10 {
		t.Fatalf("expected MaxDatasourcesPerJob to be 10, got %d", model.MaxDatasourcesPerJob)
	}
}

// TestCreateFetcherJob_QueueNameConfiguration verifies queue name is properly configured.
func TestCreateFetcherJob_QueueNameConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	tests := []struct {
		name              string
		inputQueueName    string
		expectedQueueName string
	}{
		{
			name:              "custom queue name is used",
			inputQueueName:    "custom.queue.name",
			expectedQueueName: "custom.queue.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, tt.inputQueueName, nil, nil)

			if svc.queueName != tt.expectedQueueName {
				t.Fatalf("expected queueName %q, got %q", tt.expectedQueueName, svc.queueName)
			}
		})
	}
}

// TestNewCreateFetcherJob verifies the constructor creates a valid service instance.
func TestNewCreateFetcherJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

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

func TestCreateFetcherJob_publishToQueue_TypedNilAdapterIsIgnored(t *testing.T) {
	var typedNilAdapter *pkgRabbitMQ.RabbitMQAdapter

	svc := NewCreateFetcherJob(nil, nil, nil, typedNilAdapter, "", nil, nil)
	job := &model.Job{
		ID: uuid.New(),
		MappedFields: map[string]map[string][]string{
			"datasource1": {
				"users": {"id"},
			},
		},
		Metadata:  map[string]any{"source": "test"},
		CreatedAt: time.Now().UTC(),
	}

	err := svc.publishToQueue(testContext(), job)
	require.NoError(t, err)
}

// TestCreateFetcherJob_Execute_PartialConnectionsFound tests that when only some datasources
// have matching connections, a validation error is returned listing the missing ones.
func TestCreateFetcherJob_Execute_PartialConnectionsFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

	ctx := testContext()
	connID := uuid.New()

	// Request has 2 datasources but only 1 connection exists
	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id", "status"}},
				"missing_db":  {"orders": {"id", "total"}},
			},
		},
		Metadata: map[string]any{"source": "test-product"},
	}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: only one connection found (postgres_db exists, missing_db does not)
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "postgres_db", Type: model.TypePostgreSQL},
		}, nil)

	result, err := svc.Execute(ctx, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for missing datasource, got nil")
	}

	var validationErr pkg.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	// Should mention the missing datasource
	if !strings.Contains(validationErr.Message, "missing_db") {
		t.Fatalf("expected error to mention 'missing_db', got: %s", validationErr.Message)
	}
}

// TestCreateFetcherJob_Execute_JobCreateError tests that connection test failures
// are handled properly when the cryptor fails to decrypt the connection password.
// This results in a ValidationError with ErrConnectionDown code.
func TestCreateFetcherJob_Execute_JobCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	// Provide a factory that simulates decryption during connection creation
	testFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		_, err := cryptor.Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		return nil, fmt.Errorf("connection failed")
	}

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, mockCryptor, nil, "", testFactory, nil)

	ctx := testContext()
	connID := uuid.New()

	request := newValidFetcherRequest()

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found with encryption fields
	conn := newConnectionWithEncryption(connID, "datasource1", model.TypePostgreSQL)
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	// Mock: decryption fails during connection test
	// This simulates a scenario where the connection test cannot proceed
	mockCryptor.EXPECT().
		Decrypt(gomock.Any(), conn.PasswordEncrypted, conn.EncryptionKeyVersion).
		Return("", errors.New("decryption failed"))

	result, err := svc.Execute(ctx, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for connection test failure, got nil")
	}

	// When connection test fails, we get a ValidationError with ErrConnectionDown
	var validationErr pkg.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	if validationErr.Code != constant.ErrConnectionDown.Error() {
		t.Fatalf("expected error code %s, got %s", constant.ErrConnectionDown.Error(), validationErr.Code)
	}

	if !strings.Contains(validationErr.Message, "datasource1") {
		t.Fatalf("expected error message to mention datasource1, got: %s", validationErr.Message)
	}
}

// TestCreateFetcherJob_Execute_DuplicateWithDifferentStatuses tests deduplication with various job statuses.
func TestCreateFetcherJob_Execute_DuplicateWithDifferentStatuses(t *testing.T) {
	tests := []struct {
		name       string
		status     model.JobStatus
		wantDup    bool
		wantStatus string
	}{
		{
			name:       "pending job is returned as duplicate",
			status:     model.JobStatusPending,
			wantDup:    true,
			wantStatus: string(model.JobStatusPending),
		},
		{
			name:       "processing job is returned as duplicate",
			status:     model.JobStatusProcessing,
			wantDup:    true,
			wantStatus: string(model.JobStatusProcessing),
		},
		{
			name:       "completed job is returned as duplicate",
			status:     model.JobStatusCompleted,
			wantDup:    true,
			wantStatus: string(model.JobStatusCompleted),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockJobRepo := jobRepo.NewMockRepository(ctrl)

			svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

			ctx := testContext()
			request := newValidFetcherRequest()

			existingJobID := uuid.New()
			existingJob := &model.Job{
				ID:        existingJobID,
				Status:    tt.status,
				CreatedAt: time.Now().UTC().Add(-2 * time.Minute),
			}

			// Mock: find existing job within window
			mockJobRepo.EXPECT().
				FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
				Return(existingJob, nil)

			result, err := svc.Execute(ctx, request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.IsDuplicate != tt.wantDup {
				t.Fatalf("expected IsDuplicate=%v, got %v", tt.wantDup, result.IsDuplicate)
			}

			if string(result.Job.Status) != tt.wantStatus {
				t.Fatalf("expected status %s, got %s", tt.wantStatus, result.Job.Status)
			}
		})
	}
}

// TestCreateFetcherJob_Execute_FailedJobWithinWindow_AllowsRetry ensures failed jobs do not block valid retries.
func TestCreateFetcherJob_Execute_FailedJobWithinWindow_AllowsRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	request := newValidFetcherRequest()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}

	existingFailedJob := &model.Job{
		ID:        uuid.New(),
		Status:    model.JobStatusFailed,
		CreatedAt: time.Now().UTC().Add(-1 * time.Minute),
	}

	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(existingFailedJob, nil)

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn).
		Return(nil)

	mockJobRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, created *model.Job) (*model.Job, error) {
			return created, nil
		})

	result, err := svc.Execute(ctx, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.IsDuplicate {
		t.Fatal("expected IsDuplicate to be false for failed previous job")
	}

	if !result.IsNewCreated {
		t.Fatal("expected IsNewCreated to be true")
	}
}

// TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_ReturnsExistingJob validates race-safe fallback on unique key conflict.
func TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_ReturnsExistingJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	request := newValidFetcherRequest()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}

	existingJob := &model.Job{
		ID:        uuid.New(),
		Status:    model.JobStatusPending,
		CreatedAt: time.Now().UTC(),
	}

	gomock.InOrder(
		mockJobRepo.EXPECT().
			FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
			Return(nil, nil),
		mockConnRepo.EXPECT().
			FindByConfigNames(gomock.Any(), []string{"datasource1"}).
			Return([]*model.Connection{conn}, nil),
		mockConnTester.EXPECT().
			TestConnection(gomock.Any(), conn).
			Return(nil),
		mockJobRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil, mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000, Message: "duplicate key"}}}),
		mockJobRepo.EXPECT().
			FindActiveByRequestHash(gomock.Any(), gomock.Any()).
			Return(existingJob, nil),
	)

	result, err := svc.Execute(ctx, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.IsDuplicate {
		t.Fatal("expected IsDuplicate to be true")
	}

	if result.IsNewCreated {
		t.Fatal("expected IsNewCreated to be false")
	}

	if result.Job.ID != existingJob.ID {
		t.Fatalf("expected existing job ID %s, got %s", existingJob.ID, result.Job.ID)
	}
}

// TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_ActiveLookupMissAndRetryFailure validates deterministic
// error mapping when duplicate-key recovery cannot find an active job and retry create also fails.
func TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_ActiveLookupMissAndRetryFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	request := newValidFetcherRequest()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}

	gomock.InOrder(
		mockJobRepo.EXPECT().
			FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
			Return(nil, nil),
		mockConnRepo.EXPECT().
			FindByConfigNames(gomock.Any(), []string{"datasource1"}).
			Return([]*model.Connection{conn}, nil),
		mockConnTester.EXPECT().
			TestConnection(gomock.Any(), conn).
			Return(nil),
		mockJobRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil, mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000, Message: "duplicate key"}}}),
		mockJobRepo.EXPECT().
			FindActiveByRequestHash(gomock.Any(), gomock.Any()).
			Return(nil, nil),
		mockJobRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("create failed after duplicate recovery")),
	)

	result, err := svc.Execute(ctx, request)
	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_ActiveLookupMissAndRetrySuccess validates
// the bounded retry path when duplicate conflict disappears before retry create.
func TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_ActiveLookupMissAndRetrySuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	request := newValidFetcherRequest()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}

	gomock.InOrder(
		mockJobRepo.EXPECT().
			FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
			Return(nil, nil),
		mockConnRepo.EXPECT().
			FindByConfigNames(gomock.Any(), []string{"datasource1"}).
			Return([]*model.Connection{conn}, nil),
		mockConnTester.EXPECT().
			TestConnection(gomock.Any(), conn).
			Return(nil),
		mockJobRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil, mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000, Message: "duplicate key"}}}),
		mockJobRepo.EXPECT().
			FindActiveByRequestHash(gomock.Any(), gomock.Any()).
			Return(nil, nil),
		mockJobRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, created *model.Job) (*model.Job, error) {
				return created, nil
			}),
	)

	result, err := svc.Execute(ctx, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.IsDuplicate {
		t.Fatal("expected IsDuplicate to be false")
	}

	if !result.IsNewCreated {
		t.Fatal("expected IsNewCreated to be true")
	}
}

// Tests multiple connections success path without product ownership validation (metadata.source not set)
func TestCreateFetcherJob_Execute_MultipleConnectionsSuccess_WithoutProductRepo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	connID1 := uuid.New()
	connID2 := uuid.New()

	// Request with multiple datasources
	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id", "status"}},
				"mysql_db":    {"orders": {"id", "total"}},
			},
		},
		Metadata: map[string]any{"source": "test"},
	}

	// Create connections
	conn1 := &model.Connection{ID: connID1, ProductName: "test", ConfigName: "postgres_db", Type: model.TypePostgreSQL}
	conn2 := &model.Connection{ID: connID2, ProductName: "test", ConfigName: "mysql_db", Type: model.TypeMySQL}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: both connections found
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), gomock.Any()).
		Return([]*model.Connection{conn1, conn2}, nil)

	// Mock: connection tests succeed
	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn1).
		Return(nil)
	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn2).
		Return(nil)

	// Mock: job creation succeeds (returns created job and nil error)
	mockJobRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, job *model.Job) (*model.Job, error) {
			return job, nil
		})

	result, err := svc.Execute(ctx, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.IsDuplicate {
		t.Fatal("expected IsDuplicate to be false")
	}

	if !result.IsNewCreated {
		t.Fatal("expected IsNewCreated to be true")
	}

	// Verify metadata is preserved
	if result.Job.Metadata == nil || result.Job.Metadata["source"] != "test" {
		t.Fatal("expected metadata to be preserved")
	}
}

func TestCreateFetcherJob_Execute_PublishFailureMarksJobFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, mockRabbitMQ, mockConnTester, "fetcher.extract-external-data.queue", nil)

	ctx := testContext()
	connID := uuid.New()
	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id", "status"}},
			},
		},
		Metadata: map[string]any{"source": "test"},
	}

	conn := &model.Connection{ID: connID, ConfigName: "postgres_db", Type: model.TypePostgreSQL, ProductName: "test"}
	publishErr := errors.New("publish failed")

	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"postgres_db"}).
		Return([]*model.Connection{conn}, nil)

	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn).
		Return(nil)

	mockJobRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, job *model.Job) (*model.Job, error) {
			return job, nil
		})

	mockRabbitMQ.EXPECT().
		ProducerDefault(gomock.Any(), "", "fetcher.extract-external-data.queue", gomock.Any(), gomock.Any()).
		Return(publishErr)

	mockJobRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, job *model.Job) (*model.Job, error) {
			assert.Equal(t, model.JobStatusFailed, job.Status)
			assert.NotNil(t, job.CompletedAt)
			assert.Equal(t, "process failed: unable to publish", job.Metadata["error"])

			return job, nil
		})

	result, err := svc.Execute(ctx, request)
	require.Nil(t, result)
	require.Error(t, err)

	var internalErr pkg.InternalServerError
	require.ErrorAs(t, err, &internalErr)
	require.ErrorContains(t, internalErr.Err, "publish failed")
}

// TestCreateFetcherJob_Execute_FiltersWithMultipleDatasources tests filter transformation with multiple datasources.
func TestCreateFetcherJob_Execute_FiltersWithMultipleDatasources(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	connID := uuid.New()

	// Request with filters referencing the correct datasource
	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id", "status", "amount"}},
			},
			Filters: model.NestedFilters{
				"postgres_db": {
					"transactions": {
						"status": job.FilterCondition{Equals: []any{"completed"}},
						"amount": job.FilterCondition{GreaterThan: []any{100}},
					},
				},
			},
		},
		Metadata: map[string]any{"source": "test-product"},
	}

	// Create connection
	conn := &model.Connection{ID: connID, ProductName: "test-product", ConfigName: "postgres_db", Type: model.TypePostgreSQL}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"postgres_db"}).
		Return([]*model.Connection{conn}, nil)

	// Mock: connection test succeeds
	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn).
		Return(nil)

	// Mock: job creation succeeds (returns created job and nil error)
	mockJobRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, job *model.Job) (*model.Job, error) {
			return job, nil
		})

	result, err := svc.Execute(ctx, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify filters are preserved in the job
	if result.Job.Filters == nil {
		t.Fatal("expected filters to be preserved")
	}

	// Check that postgres_db datasource has filters
	if _, ok := result.Job.Filters["postgres_db"]; !ok {
		t.Fatal("expected filters for postgres_db datasource")
	}

	// Check that transactions table has filters
	if _, ok := result.Job.Filters["postgres_db"]["transactions"]; !ok {
		t.Fatal("expected filters for transactions table")
	}

	// Verify both filter fields exist
	if _, ok := result.Job.Filters["postgres_db"]["transactions"]["status"]; !ok {
		t.Fatal("expected filter for status field")
	}
	if _, ok := result.Job.Filters["postgres_db"]["transactions"]["amount"]; !ok {
		t.Fatal("expected filter for amount field")
	}
}

// TestCreateFetcherJob_Execute_InvalidFilterReferences tests that invalid filter references return validation error.
// NOTE: Table name validation is intentionally NOT done at this stage - the DataSource adapter handles
// schema resolution with fallback logic (e.g., trying "public.table" if "table" not found).
// Therefore, only datasource names and filter format are validated here.
func TestCreateFetcherJob_Execute_InvalidFilterReferences(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, "", nil, nil)

	tests := []struct {
		name    string
		request model.FetcherRequest
		wantErr string
	}{
		{
			name: "filter references unknown datasource",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						"postgres_db": {"transactions": {"id", "status"}},
					},
					Filters: model.NestedFilters{
						"unknown_db": {
							"transactions": {
								"status": job.FilterCondition{Equals: []any{"completed"}},
							},
						},
					},
				},
				Metadata: map[string]any{"source": "test-product"},
			},
			wantErr: "datasource 'unknown_db' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext()

			result, err := svc.Execute(ctx, tt.request)

			if result != nil {
				t.Fatalf("expected nil result for invalid filter, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error for invalid filter, got nil")
			}

			var validationErr pkg.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}

			if !strings.Contains(validationErr.Message, tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, validationErr.Message)
			}
		})
	}
}

// TestCreateFetcherJob_Execute_ConnectionNotAssigned tests that when a connection
// has no product assigned (ProductName == ""), a ValidationError with ErrConnectionNotAssigned is returned.
func TestCreateFetcherJob_Execute_ConnectionNotAssigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	connID := uuid.New()

	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"datasource1": {"table1": {"field1"}},
			},
		},
		Metadata: map[string]any{"source": "test"},
	}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found with ProductName = "" (unassigned)
	conn := &model.Connection{ID: connID, ConfigName: "datasource1", Type: model.TypePostgreSQL, ProductName: ""}
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	result, err := svc.Execute(ctx, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for unassigned connection, got nil")
	}

	var validationErr pkg.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	if validationErr.Code != constant.ErrConnectionNotAssigned.Error() {
		t.Fatalf("expected error code %s, got %s", constant.ErrConnectionNotAssigned.Error(), validationErr.Code)
	}

	if !strings.Contains(validationErr.Message, "datasource1") {
		t.Fatalf("expected error message to mention datasource1, got: %s", validationErr.Message)
	}
}

// TestCreateFetcherJob_Execute_ProductMismatch tests that when a connection belongs
// to a different product than the one identified by metadata.source, a ValidationError
// with ErrProductMismatch is returned.
func TestCreateFetcherJob_Execute_ProductMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	connID := uuid.New()

	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"datasource1": {"table1": {"field1"}},
			},
		},
		Metadata: map[string]any{"source": "test"},
	}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found with ProductName = "other-product" (different from source "test")
	conn := &model.Connection{ID: connID, ConfigName: "datasource1", Type: model.TypePostgreSQL, ProductName: "other-product"}
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	result, err := svc.Execute(ctx, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for product mismatch, got nil")
	}

	var validationErr pkg.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	if validationErr.Code != constant.ErrProductMismatch.Error() {
		t.Fatalf("expected error code %s, got %s", constant.ErrProductMismatch.Error(), validationErr.Code)
	}

	if !strings.Contains(validationErr.Message, "datasource1") {
		t.Fatalf("expected error message to mention datasource1, got: %s", validationErr.Message)
	}
}

// TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_FindActiveReturnsError validates error
// propagation when FindActiveByRequestHash returns a DB error during duplicate key recovery.
func TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_FindActiveReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	request := newValidFetcherRequest()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}

	gomock.InOrder(
		mockJobRepo.EXPECT().
			FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
			Return(nil, nil),
		mockConnRepo.EXPECT().
			FindByConfigNames(gomock.Any(), []string{"datasource1"}).
			Return([]*model.Connection{conn}, nil),
		mockConnTester.EXPECT().
			TestConnection(gomock.Any(), conn).
			Return(nil),
		mockJobRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil, mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000, Message: "duplicate key"}}}),
		mockJobRepo.EXPECT().
			FindActiveByRequestHash(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("mongodb connection lost")),
	)

	result, err := svc.Execute(ctx, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_RetryAlsoHitsDupKey_SecondRecoveryFindsJob validates
// the full retry chain: first create hits dup key, active lookup misses, retry create also hits dup key,
// and the second recovery finds the active job.
func TestCreateFetcherJob_Execute_DuplicateKeyOnCreate_RetryAlsoHitsDupKey_SecondRecoveryFindsJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	request := newValidFetcherRequest()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}

	existingJob := &model.Job{
		ID:        uuid.New(),
		Status:    model.JobStatusPending,
		CreatedAt: time.Now().UTC(),
	}

	gomock.InOrder(
		// 1. Dedup window check returns no match
		mockJobRepo.EXPECT().
			FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
			Return(nil, nil),
		// 2. Connection lookup
		mockConnRepo.EXPECT().
			FindByConfigNames(gomock.Any(), []string{"datasource1"}).
			Return([]*model.Connection{conn}, nil),
		// 3. Connection test
		mockConnTester.EXPECT().
			TestConnection(gomock.Any(), conn).
			Return(nil),
		// 4. First Create → duplicate key error
		mockJobRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil, mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000, Message: "duplicate key"}}}),
		// 5. First FindActiveByRequestHash → miss (nil, nil)
		mockJobRepo.EXPECT().
			FindActiveByRequestHash(gomock.Any(), gomock.Any()).
			Return(nil, nil),
		// 6. Retry Create → also duplicate key error
		mockJobRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil, mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000, Message: "duplicate key"}}}),
		// 7. Second FindActiveByRequestHash → finds the active job
		mockJobRepo.EXPECT().
			FindActiveByRequestHash(gomock.Any(), gomock.Any()).
			Return(existingJob, nil),
	)

	result, err := svc.Execute(ctx, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.IsDuplicate {
		t.Fatal("expected IsDuplicate to be true")
	}

	if result.IsNewCreated {
		t.Fatal("expected IsNewCreated to be false")
	}

	if result.Job.ID != existingJob.ID {
		t.Fatalf("expected existing job ID %s, got %s", existingJob.ID, result.Job.ID)
	}
}

// TestCreateFetcherJob_Execute_ProductValidationSuccess tests the full happy path when
// metadata.source identifies a valid product and all connections belong to that product.
func TestCreateFetcherJob_Execute_ProductValidationSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	connID := uuid.New()

	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"datasource1": {"table1": {"field1"}},
			},
		},
		Metadata: map[string]any{"source": "test"},
	}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found with matching ProductName
	conn := &model.Connection{ID: connID, ConfigName: "datasource1", Type: model.TypePostgreSQL, ProductName: "test"}
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	// Mock: connection test succeeds
	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn).
		Return(nil)

	// Mock: job creation succeeds
	mockJobRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, j *model.Job) (*model.Job, error) {
			return j, nil
		})

	result, err := svc.Execute(ctx, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.IsDuplicate {
		t.Fatal("expected IsDuplicate to be false")
	}

	if !result.IsNewCreated {
		t.Fatal("expected IsNewCreated to be true")
	}

	if result.Job.Metadata == nil || result.Job.Metadata["source"] != "test" {
		t.Fatal("expected metadata source to be preserved")
	}
}

// TestPublishToQueue_TenantIDHeaderPropagation tests that publishToQueue includes X-Tenant-ID
// in AMQP headers when tenant context is present, and omits it when absent.
func TestPublishToQueue_TenantIDHeaderPropagation(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		expectTenantID bool
	}{
		{
			name:           "includes X-Tenant-ID when tenant context is present",
			tenantID:       "tenant-abc-123",
			expectTenantID: true,
		},
		{
			name:           "omits X-Tenant-ID when tenant context is empty",
			tenantID:       "",
			expectTenantID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockPublisher := messaging.NewMockMessagePublisher(ctrl)

			svc := &CreateFetcherJob{
				rabbitMQ:  mockPublisher,
				queueName: "test-queue",
			}

			testJob := &model.Job{
				ID:           uuid.New(),
				MappedFields: map[string]map[string][]string{"ds1": {"t1": {"f1"}}},
				Metadata:     map[string]any{"source": "test-product"},
				CreatedAt:    time.Now().UTC(),
			}

			// Capture headers passed to ProducerDefault
			var capturedHeaders *map[string]any
			mockPublisher.EXPECT().
				ProducerDefault(gomock.Any(), "", "test-queue", gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ string, _ []byte, h *map[string]any) error {
					capturedHeaders = h
					return nil
				})

			ctx := testContext()
			if tt.tenantID != "" {
				ctx = tmcore.ContextWithTenantID(ctx, tt.tenantID)
			}

			err := svc.publishToQueue(ctx, testJob)
			require.NoError(t, err)
			require.NotNil(t, capturedHeaders)

			headers := *capturedHeaders
			if tt.expectTenantID {
				assert.Equal(t, tt.tenantID, headers["X-Tenant-ID"], "expected X-Tenant-ID header to match tenant ID from context")
			} else {
				_, exists := headers["X-Tenant-ID"]
				assert.False(t, exists, "expected no X-Tenant-ID header when tenant context is empty")
			}

			// Always verify standard headers are present
			assert.Equal(t, testJob.ID.String(), headers["jobId"])
		})
	}
}

// --- ST-T009-01: request -> engine.ExtractionRequest mapping (Option 2) ----
//
// Option 2 (owner decision): map the Manager FetcherRequest onto an
// engine.ExtractionRequest as a pure, host-side intermediate. NO PlanExtraction
// call, NO schema discovery, NO new validation — all current Manager validation
// stays byte-identical. The engine.ExtractionRequest is the LIVE intermediate
// from which the queue payload's mappedFields/metadata are built, so it is not
// orphan code. These tests pin:
//   (1) MappedFields, Filters, and metadata.source map correctly into the
//       engine.ExtractionRequest (metadata.source preserved OPAQUE);
//   (2) the queue payload built from the intermediate is byte-identical to the
//       legacy hand-built payload.
//
// The engine.TenantContext bridge is intentionally NOT part of this subtask: it
// has no non-orphan consumer on the create path until the planning/execution
// consumer lands (ST-03 / T-010). See extraction_request_mapper.go SCOPE NOTE.

// TestMapToExtractionRequest_MapsFieldsAndMetadata_DefersFilters asserts the pure
// mapper projects MappedFields and opaque Metadata (incl. metadata.source) onto
// engine.ExtractionRequest, and that Filters are intentionally NOT projected at
// create (deferred to the T-010 execution consumer) even when the request carries
// them — so the engine request never holds a filter shape unvalidated by a consumer.
func TestMapToExtractionRequest_MapsFieldsAndMetadata_DefersFilters(t *testing.T) {
	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id", "status", "amount"}},
			},
			Filters: model.NestedFilters{
				"postgres_db": {
					"transactions": {
						"status": job.FilterCondition{Equals: []any{"completed"}},
					},
				},
			},
		},
		Metadata: map[string]any{"source": "plugin_crm", "key": "value"},
	}

	req := mapToExtractionRequest(request)

	// MappedFields map correctly (FieldSelection is the same underlying type).
	require.Equal(t, []string{"id", "status", "amount"}, req.MappedFields["postgres_db"]["transactions"])

	// Metadata is carried opaque, including metadata.source (e.g. plugin_crm).
	require.Equal(t, "plugin_crm", req.Metadata["source"], "metadata.source must be preserved opaque")
	require.Equal(t, "value", req.Metadata["key"], "all metadata is carried opaque")

	// Filters are intentionally NOT projected at create (Option 2): they stay on the
	// typed request.DataRequest.Filters path for the persisted job and queue payload.
	// The engine-request filter projection is deferred to the T-010 execution
	// consumer, which pins it against the planner's nested filter shape.
	require.Nil(t, req.Filters, "filters are not projected into the engine request at create (deferred to T-010)")

	// B2: the engine request carries NO product/org concept and the mapper must
	// not invent Overrides (no per-request bounds in this subtask).
	require.Nil(t, req.Overrides, "no limit overrides are introduced at mapping time")
}

// TestPublishToQueue_PayloadByteIdenticalFromIntermediate proves the queue bytes
// built via the engine.ExtractionRequest intermediate equal the legacy hand-built
// payload. RED: buildQueueMessage does not exist yet.
func TestPublishToQueue_PayloadByteIdenticalFromIntermediate(t *testing.T) {
	j := &model.Job{
		ID: uuid.New(),
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"transactions": {"id", "status", "amount"}},
		},
		Filters: model.NestedFilters{
			"postgres_db": {"transactions": {"status": job.FilterCondition{Equals: []any{"completed"}}}},
		},
		Metadata:  map[string]any{"source": "plugin_crm", "key": "value"},
		CreatedAt: time.Now().UTC(),
	}

	// Legacy hand-built payload (the exact shape publishToQueue emitted before).
	legacy := map[string]any{
		"jobId":        j.ID.String(),
		"mappedFields": j.MappedFields,
		"metadata":     j.Metadata,
		"createdAt":    j.CreatedAt,
		"filters":      j.Filters,
	}
	legacyBytes, err := json.Marshal(legacy)
	require.NoError(t, err)

	// Payload built from the engine.ExtractionRequest intermediate.
	gotBytes, err := json.Marshal(buildQueueMessage(j))
	require.NoError(t, err)

	require.JSONEq(t, string(legacyBytes), string(gotBytes), "queue payload must be byte-identical to the legacy shape")
}

// --- ST-T009-02: idempotency & persistence LOCK tests ----------------------
//
// These pin the existing idempotency/persistence behavior with the DISTINCT
// assertions the subtask asks for. Under Option 2 there is NO PlanExtraction at
// creation and the mapper is pure/infallible, so there is no new create-time
// failure mode — these LOCK the already-correct behavior, they do not change it.

// TestCreateFetcherJob_Execute_DuplicateWithinWindow_DoesNotPublish explicitly
// locks that a duplicate within the dedup window returns the existing job and
// publishes NO new message — asserted against a REAL publisher mock whose
// ProducerDefault is never expected (so any publish call fails the test). The
// pre-existing duplicate test used a nil publisher, which only proved this
// implicitly.
func TestCreateFetcherJob_Execute_DuplicateWithinWindow_DoesNotPublish(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, mockRabbitMQ, "fetcher.extract-external-data.queue", nil, nil)

	ctx := testContext()
	request := newValidFetcherRequest()

	existingJobID := uuid.New()
	existingJob := &model.Job{
		ID:        existingJobID,
		Status:    model.JobStatusPending,
		CreatedAt: time.Now().UTC().Add(-2 * time.Minute),
	}

	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(existingJob, nil)

	// Guard: NO mockJobRepo.Create and NO mockRabbitMQ.ProducerDefault expectations.
	// A duplicate must neither persist a new job nor publish a message.

	result, err := svc.Execute(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsDuplicate, "duplicate within window must be reported as a duplicate")
	require.False(t, result.IsNewCreated, "duplicate must not be a new creation")
	require.Equal(t, existingJobID, result.Job.ID, "the existing job must be returned unchanged")
}

// TestCreateFetcherJob_Execute_NewRequest_CreatesPendingAndPublishesExactlyOnce
// locks that a new valid request persists a PENDING job and publishes EXACTLY
// once (Times(1)) — pinning both the persisted status and the single-publish
// guarantee together against real mocks.
func TestCreateFetcherJob_Execute_NewRequest_CreatesPendingAndPublishesExactlyOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, mockRabbitMQ, mockConnTester, "fetcher.extract-external-data.queue", nil)

	ctx := testContext()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}
	request := newValidFetcherRequest()

	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)
	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn).
		Return(nil)

	// Persisted job must be PENDING at creation.
	mockJobRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, j *model.Job) (*model.Job, error) {
			assert.Equal(t, model.JobStatusPending, j.Status, "a newly created job must be persisted as PENDING")
			return j, nil
		}).
		Times(1)

	// Publish must happen EXACTLY once.
	mockRabbitMQ.EXPECT().
		ProducerDefault(gomock.Any(), "", "fetcher.extract-external-data.queue", gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	result, err := svc.Execute(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsNewCreated, "a new valid request must create a new job")
	require.False(t, result.IsDuplicate, "a new valid request must not be a duplicate")
	require.Equal(t, model.JobStatusPending, result.Job.Status, "the returned job must be PENDING")
}

// TestCreateFetcherJob_Execute_ValidationFailure_CreatesNoJobAndDoesNotPublish
// is the Option-2 replacement for the spec's STALE "Engine planning failure does
// not create or publish a job" assertion. Under Option 2 there is no planning at
// creation, so the equivalent guard is: a MANAGER validation failure (here, the
// metadata.source gate) creates no job and publishes nothing. Asserted against
// real Create/publish mocks with no expectations set.
func TestCreateFetcherJob_Execute_ValidationFailure_CreatesNoJobAndDoesNotPublish(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, mockRabbitMQ, "q", nil, nil)

	ctx := testContext()
	// Valid mappedFields but missing metadata.source -> Manager validation failure
	// BEFORE any persistence or publish.
	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"datasource1": {"table1": {"field1"}},
			},
		},
		Metadata: map[string]any{"key": "value"},
	}

	// Guard: validation must short-circuit BEFORE dedup lookup, Create, or publish.
	// No mock expectations are set, so any repository or publisher call fails.

	result, err := svc.Execute(ctx, request)
	require.Error(t, err, "a Manager validation failure must surface as an error")
	require.Nil(t, result, "no job result when validation fails")

	var ve pkg.ValidationError
	require.True(t, errors.As(err, &ve), "expected a ValidationError, got %T", err)
}

// --- ST-T009-03: RabbitMQ dispatch boundary LOCK tests ---------------------
//
// ST-02 already locked: publish-exactly-once-on-success, no-publish-on-duplicate,
// no-publish-on-validation-failure. ST-03 adds the remaining dispatch-boundary
// invariants WITHOUT duplicating those:
//   (a) a job PERSISTENCE failure (Create returns a non-duplicate error) emits NO
//       message — no orphan publish ahead of a job that was never stored;
//   (b) the dispatch routing/topology (exchange + routing key) is byte-identical:
//       publish targets the default exchange ("") with the configured queue name
//       as the routing key.
// The spec's "publish after successful Engine planning" is STALE under Option 2
// (no PlanExtraction at create); the real invariant is "publish only after
// successful validation + persistence", which (a) locks.

// TestCreateFetcherJob_Execute_PersistenceFailure_DoesNotPublish locks that when
// the job repository's Create fails with a non-duplicate error, Execute returns an
// internal error and NEVER publishes — so a failed persistence cannot leave an
// orphan queue message pointing at a job that was never stored. This is a DIFFERENT
// direction from PublishFailureMarksJobFailed (publish-attempted-then-fails): here
// persistence fails BEFORE publish is ever reachable.
func TestCreateFetcherJob_Execute_PersistenceFailure_DoesNotPublish(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, mockRabbitMQ, mockConnTester, "fetcher.extract-external-data.queue", nil)

	ctx := testContext()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}
	request := newValidFetcherRequest()

	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)
	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn).
		Return(nil)

	// Persistence fails with a non-duplicate error.
	mockJobRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("mongodb write failed"))

	// Guard: NO mockRabbitMQ.ProducerDefault expectation — publish must never be
	// reached when persistence fails (no orphan message). Likewise no jobRepo.Update,
	// since there is no created job to mark FAILED.

	result, err := svc.Execute(ctx, request)
	require.Nil(t, result, "no result when persistence fails")
	require.Error(t, err, "a persistence failure must surface as an error")

	var internalErr pkg.InternalServerError
	require.True(t, errors.As(err, &internalErr), "expected InternalServerError, got %T: %v", err, err)
}

// TestCreateFetcherJob_Execute_DispatchRoutingTopologyIsByteIdentical locks the
// dispatch routing/topology: a successful create publishes to the DEFAULT exchange
// ("") with the configured queue name as the routing key, capturing the exact
// exchange/key arguments rather than matching them loosely. A change to the
// exchange or routing key would break the un-migrated Worker's consumption and is
// the kind of drift this test guards against.
func TestCreateFetcherJob_Execute_DispatchRoutingTopologyIsByteIdentical(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)
	mockRabbitMQ := messaging.NewMockMessagePublisher(ctrl)

	const queueName = "fetcher.extract-external-data.queue"

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, mockRabbitMQ, mockConnTester, queueName, nil)

	ctx := testContext()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}
	request := newValidFetcherRequest()

	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)
	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn).
		Return(nil)
	mockJobRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, j *model.Job) (*model.Job, error) { return j, nil })

	var (
		gotExchange string
		gotKey      string
	)

	mockRabbitMQ.EXPECT().
		ProducerDefault(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, exchange, key string, _ []byte, _ *map[string]any) error {
			gotExchange = exchange
			gotKey = key
			return nil
		}).
		Times(1)

	_, err := svc.Execute(ctx, request)
	require.NoError(t, err)

	require.Equal(t, "", gotExchange, "dispatch must target the default exchange (empty string), unchanged")
	require.Equal(t, queueName, gotKey, "dispatch routing key must be the configured queue name, unchanged")
}

// TestCreateFetcherJob_Execute_PreservesFET0414FromConnectionTester verifies
// that when the injected ConnectionTester returns a pkg.ValidationError
// (FET-0414 host safety rejection from the factory), CreateFetcherJob
// propagates it verbatim instead of masking it behind ErrConnectionDown
// (FET-1040). Masking would (a) break the documented FET-0414 → HTTP 400
// contract for POST /v1/fetcher, and (b) lose the audit signal that a tenant
// tried to reach a denylisted host.
func TestCreateFetcherJob_Execute_PreservesFET0414FromConnectionTester(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockConnTester, "", nil)

	ctx := testContext()
	request := newValidFetcherRequest()
	conn := &model.Connection{ID: uuid.New(), ProductName: "test-product", ConfigName: "datasource1", Type: model.TypePostgreSQL}

	// No duplicate, connection lookup succeeds — fall through to the tester.
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	// Mimic what the default ConnectionTester emits when the datasource factory
	// rejects the host via hostsafety.ValidateHostForConnection: a typed
	// pkg.ValidationError with code FET-0414, wrapped through fmt.Errorf("%w").
	factoryRejection := pkg.ValidationError{
		EntityType: "connection",
		Code:       "FET-0414",
		Title:      "Forbidden Host",
		Message:    "Host is not a valid external database endpoint",
	}
	mockConnTester.EXPECT().
		TestConnection(gomock.Any(), conn).
		Return(fmt.Errorf("failed to create datasource: %w", factoryRejection))

	// Job creation must NOT be attempted — the propagated error stops the flow.
	// Absence of mockJobRepo.EXPECT().Create(...) here is the guard.

	result, err := svc.Execute(ctx, request)
	require.Error(t, err)
	require.Nil(t, result, "no job must be persisted when host safety rejects the connection")

	var ve pkg.ValidationError
	require.True(t, errors.As(err, &ve),
		"CreateFetcherJob must propagate factory's pkg.ValidationError unchanged, got: %T %v", err, err)
	assert.Equal(t, "FET-0414", ve.Code,
		"FET-0414 must survive the testConnections layer instead of being remapped to ErrConnectionDown (FET-1040)")
}
