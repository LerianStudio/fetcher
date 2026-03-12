package command

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	productMock "github.com/LerianStudio/fetcher/pkg/mongodb/product"
	rabbitmqMock "github.com/LerianStudio/fetcher/pkg/rabbitmq"

	"github.com/google/uuid"
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
		Metadata: map[string]any{"key": "value"},
	}
}

// newConnectionWithEncryption creates a connection with encryption fields set for testing.
func newConnectionWithEncryption(id uuid.UUID, configName string, dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID:                   id,
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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext()
			orgID := uuid.New()

			result, err := svc.Execute(ctx, orgID, tt.request)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	existingJobID := uuid.New()
	existingJob := &model.Job{
		ID:             existingJobID,
		OrganizationID: orgID,
		Status:         model.JobStatusPending,
		CreatedAt:      time.Now().UTC().Add(-2 * time.Minute),
	}

	// Mock: find existing job within window
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(existingJob, nil)

	result, err := svc.Execute(ctx, orgID, request)
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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: no connections found at all
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return([]*model.Connection{}, nil)

	result, err := svc.Execute(ctx, orgID, request)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

	ctx := testContext()
	orgID := uuid.New()

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

	result, err := svc.Execute(ctx, orgID, request)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	dbError := errors.New("database connection failed")

	// Mock: database error during duplicate check
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, request)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	dbError := errors.New("database connection failed")

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: database error during connection lookup
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, request)

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
			svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, tt.inputQueueName)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

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
	var typedNilAdapter *rabbitmqMock.RabbitMQAdapter

	svc := NewCreateFetcherJob(nil, nil, nil, nil, typedNilAdapter, "")
	job := &model.Job{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		MappedFields: map[string]map[string][]string{
			"datasource1": {
				"users": {"id"},
			},
		},
		Metadata:  map[string]any{"source": "test"},
		CreatedAt: time.Now().UTC(),
	}

	if err := svc.publishToQueue(testContext(), job); err != nil {
		t.Fatalf("expected typed-nil adapter to be ignored, got %v", err)
	}
}

// TestCreateFetcherJob_Execute_PartialConnectionsFound tests that when only some datasources
// have matching connections, a validation error is returned listing the missing ones.
func TestCreateFetcherJob_Execute_PartialConnectionsFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	// Request has 2 datasources but only 1 connection exists
	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id", "status"}},
				"missing_db":  {"orders": {"id", "total"}},
			},
		},
	}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: only one connection found (postgres_db exists, missing_db does not)
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
		Return([]*model.Connection{
			{ID: connID, ConfigName: "postgres_db", Type: model.TypePostgreSQL},
		}, nil)

	result, err := svc.Execute(ctx, orgID, request)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, mockCryptor, nil, "")

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	request := newValidFetcherRequest()

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found with encryption fields
	conn := newConnectionWithEncryption(connID, "datasource1", model.TypePostgreSQL)
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	// Mock: decryption fails during connection test
	// This simulates a scenario where the connection test cannot proceed
	mockCryptor.EXPECT().
		Decrypt(gomock.Any(), conn.PasswordEncrypted, conn.EncryptionKeyVersion).
		Return("", errors.New("decryption failed"))

	result, err := svc.Execute(ctx, orgID, request)

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
		{
			name:       "failed job is returned as duplicate",
			status:     model.JobStatusFailed,
			wantDup:    true,
			wantStatus: string(model.JobStatusFailed),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConnRepo := connRepo.NewMockRepository(ctrl)
			mockJobRepo := jobRepo.NewMockRepository(ctrl)

			svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

			ctx := testContext()
			orgID := uuid.New()
			request := newValidFetcherRequest()

			existingJobID := uuid.New()
			existingJob := &model.Job{
				ID:             existingJobID,
				OrganizationID: orgID,
				Status:         tt.status,
				CreatedAt:      time.Now().UTC().Add(-2 * time.Minute),
			}

			// Mock: find existing job within window
			mockJobRepo.EXPECT().
				FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
				Return(existingJob, nil)

			result, err := svc.Execute(ctx, orgID, request)
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

// Tests multiple connections success path without product repo (nil productRepo skips product validation)
func TestCreateFetcherJob_Execute_MultipleConnectionsSuccess_WithoutProductRepo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, nil, mockConnTester, "")

	ctx := testContext()
	orgID := uuid.New()
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
	conn1 := &model.Connection{ID: connID1, ConfigName: "postgres_db", Type: model.TypePostgreSQL}
	conn2 := &model.Connection{ID: connID2, ConfigName: "mysql_db", Type: model.TypeMySQL}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: both connections found
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, gomock.Any()).
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

	result, err := svc.Execute(ctx, orgID, request)
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
	mockRabbitMQ := rabbitmqMock.NewMockAdapter(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, mockRabbitMQ, mockConnTester, "fetcher.extract-external-data.queue")

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id", "status"}},
			},
		},
		Metadata: map[string]any{"source": "test"},
	}

	conn := &model.Connection{ID: connID, ConfigName: "postgres_db", Type: model.TypePostgreSQL}
	publishErr := errors.New("publish failed")

	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"postgres_db"}).
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
			if job.Status != model.JobStatusFailed {
				t.Fatalf("expected failed job status, got %s", job.Status)
			}

			if job.CompletedAt == nil {
				t.Fatal("expected completedAt to be set after publish failure")
			}

			if got := job.Metadata["error"]; got != "process failed: unable to publish" {
				t.Fatalf("expected failure metadata to be recorded, got %#v", got)
			}

			return job, nil
		})

	result, err := svc.Execute(ctx, orgID, request)
	if result != nil {
		t.Fatalf("expected nil result on publish failure, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected publish failure error, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected internal server error wrapper, got %T", err)
	}

	if internalErr.Err == nil || !strings.Contains(internalErr.Err.Error(), "publish failed") {
		t.Fatalf("expected wrapped publish failure, got %v", internalErr.Err)
	}
}

// TestCreateFetcherJob_Execute_FiltersWithMultipleDatasources tests filter transformation with multiple datasources.
func TestCreateFetcherJob_Execute_FiltersWithMultipleDatasources(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, nil, nil, nil, mockConnTester, "")

	ctx := testContext()
	orgID := uuid.New()
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
	}

	// Create connection
	conn := &model.Connection{ID: connID, ConfigName: "postgres_db", Type: model.TypePostgreSQL}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"postgres_db"}).
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

	result, err := svc.Execute(ctx, orgID, request)
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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil, nil, "")

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
			},
			wantErr: "datasource 'unknown_db' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext()
			orgID := uuid.New()

			result, err := svc.Execute(ctx, orgID, tt.request)

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

// TestCreateFetcherJob_Execute_ProductNotFound tests that when metadata.source references
// a product code that does not exist, a ValidationError with ErrEntityNotFound is returned.
func TestCreateFetcherJob_Execute_ProductNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockProductRepo, nil, nil, mockConnTester, "")

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"datasource1": {"table1": {"field1"}},
			},
		},
		Metadata: map[string]any{"source": "unknown-product"},
	}

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found (must pass connection lookup before product validation)
	conn := &model.Connection{ID: connID, ConfigName: "datasource1", Type: model.TypePostgreSQL}
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	// Mock: product not found (nil, nil)
	mockProductRepo.EXPECT().
		FindByCode(gomock.Any(), "unknown-product", orgID).
		Return(nil, nil)

	result, err := svc.Execute(ctx, orgID, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for product not found, got nil")
	}

	var validationErr pkg.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	if validationErr.Code != constant.ErrEntityNotFound.Error() {
		t.Fatalf("expected error code %s, got %s", constant.ErrEntityNotFound.Error(), validationErr.Code)
	}
}

// TestCreateFetcherJob_Execute_ProductRepoError tests that when productRepo.FindByCode
// returns a database error, an InternalServerError is returned.
func TestCreateFetcherJob_Execute_ProductRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockProductRepo, nil, nil, mockConnTester, "")

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()

	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"datasource1": {"table1": {"field1"}},
			},
		},
		Metadata: map[string]any{"source": "test"},
	}

	dbError := errors.New("database connection failed")

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found
	conn := &model.Connection{ID: connID, ConfigName: "datasource1", Type: model.TypePostgreSQL}
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	// Mock: productRepo returns error
	mockProductRepo.EXPECT().
		FindByCode(gomock.Any(), "test", orgID).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, request)

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

// TestCreateFetcherJob_Execute_ConnectionNotAssigned tests that when a connection
// has no product assigned (ProductID == nil), a ValidationError with ErrConnectionNotAssigned is returned.
func TestCreateFetcherJob_Execute_ConnectionNotAssigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockProductRepo, nil, nil, mockConnTester, "")

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()

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
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found with ProductID = nil (unassigned)
	conn := &model.Connection{ID: connID, ConfigName: "datasource1", Type: model.TypePostgreSQL, ProductID: nil}
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	// Mock: product found
	product := &model.Product{ID: productID, OrganizationID: orgID, Code: "test", Name: "Test Product"}
	mockProductRepo.EXPECT().
		FindByCode(gomock.Any(), "test", orgID).
		Return(product, nil)

	result, err := svc.Execute(ctx, orgID, request)

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
	mockProductRepo := productMock.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockProductRepo, nil, nil, mockConnTester, "")

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productIDX := uuid.New()
	productIDY := uuid.New()

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
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found with ProductID = Y (different from product X)
	conn := &model.Connection{ID: connID, ConfigName: "datasource1", Type: model.TypePostgreSQL, ProductID: &productIDY}
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	// Mock: product found with ID = X
	product := &model.Product{ID: productIDX, OrganizationID: orgID, Code: "test", Name: "Test Product"}
	mockProductRepo.EXPECT().
		FindByCode(gomock.Any(), "test", orgID).
		Return(product, nil)

	result, err := svc.Execute(ctx, orgID, request)

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

// TestCreateFetcherJob_Execute_ProductValidationSuccess tests the full happy path when
// metadata.source identifies a valid product and all connections belong to that product.
func TestCreateFetcherJob_Execute_ProductValidationSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockProductRepo := productMock.NewMockRepository(ctrl)
	mockConnTester := NewMockConnectionTester(ctrl)

	svc := NewCreateFetcherJobWithTester(mockConnRepo, mockJobRepo, mockProductRepo, nil, nil, mockConnTester, "")

	ctx := testContext()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()

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
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection found with matching ProductID
	conn := &model.Connection{ID: connID, ConfigName: "datasource1", Type: model.TypePostgreSQL, ProductID: &productID}
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return([]*model.Connection{conn}, nil)

	// Mock: product found with matching ID
	product := &model.Product{ID: productID, OrganizationID: orgID, Code: "test", Name: "Test Product"}
	mockProductRepo.EXPECT().
		FindByCode(gomock.Any(), "test", orgID).
		Return(product, nil)

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

	result, err := svc.Execute(ctx, orgID, request)
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
