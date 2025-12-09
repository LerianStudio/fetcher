package command

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: "test-request-id",
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}

	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}

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

// TestCreateFetcherJob_Execute_ValidationError tests that invalid requests return validation errors.
func TestCreateFetcherJob_Execute_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	existingJobID := uuid.New()
	existingJob := &model.Job{
		ID:             existingJobID,
		OrganizationID: orgID,
		Status:         model.JobStatusPending,
		CreatedAt:      time.Now().Add(-2 * time.Minute),
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

// TestCreateFetcherJob_Execute_MissingDatasource tests that missing datasource returns error.
func TestCreateFetcherJob_Execute_MissingDatasource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: connection not found for datasource
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return([]*model.Connection{}, nil)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

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

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

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

	if ExtractExternalDataQueue != "extract-external-data-queue" {
		t.Fatalf("expected ExtractExternalDataQueue to be 'extract-external-data-queue', got %s", ExtractExternalDataQueue)
	}
}
