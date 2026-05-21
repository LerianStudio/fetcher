package services

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// TestExtractExternalData_ParseError_UpdatesJobAndPublishesNotification tests the full error flow
// when message parsing fails but jobID can be extracted from headers.
func TestExtractExternalData_ParseError_UpdatesJobAndPublishesNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	invalidBody := []byte(`{"invalid": json}`)
	headers := map[string]any{
		"jobId": jobID.String(),
	}

	// Expect job status to be updated to failed
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	// Expect failure notification to be published
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		DoAndReturn(func(_ interface{}, exchange, routingKey string, body []byte) error {
			var notification JobNotificationMessage
			if err := json.Unmarshal(body, &notification); err != nil {
				t.Errorf("failed to unmarshal notification: %v", err)
			}
			if notification.Status != "failed" {
				t.Errorf("expected status 'failed', got %s", notification.Status)
			}
			if notification.JobID != jobID {
				t.Errorf("expected jobID %s, got %s", jobID, notification.JobID)
			}
			return nil
		})

	err := uc.ExtractExternalData(ctx, invalidBody, headers)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestExtractExternalData_SkipsCompletedJob tests that completed jobs are skipped.
func TestExtractExternalData_SkipsCompletedJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"datasource1": {"table1": {"field1"}},
		},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	// Job is already completed - should be skipped
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{
			ID:     jobID,
			Status: model.JobStatusCompleted,
		}, nil)

	// No other mocks should be called since job is skipped

	err = uc.ExtractExternalData(ctx, body, nil)
	if err != nil {
		t.Fatalf("expected no error for skipped job, got: %v", err)
	}
}

// TestExtractExternalData_ProcessingJobDoesNotReprocessTerminalWork tests that
// processing jobs are not treated as terminal skip state; the handler still
// reloads the job and exits via the existing non-pending CAS guard.
func TestExtractExternalData_ProcessingJobDoesNotReprocessTerminalWork(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"datasource1": {"table1": {"field1"}},
		},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{
			ID:     jobID,
			Status: model.JobStatusProcessing,
		}, nil).Times(2)

	err = uc.ExtractExternalData(ctx, body, nil)
	if err != nil {
		t.Fatalf("expected no error for non-pending in-flight job redelivery, got: %v", err)
	}
}

// TestExtractExternalData_ConnectionNotFound tests error handling when connection is not found.
func TestExtractExternalData_ConnectionNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"users": {"id", "name"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	// Job is not completed - should continue processing
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{
			ID:     jobID,
			Status: model.JobStatusPending,
		}, nil)

	// Second call for job validation in handleErrorWithUpdate flow
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{
			ID:     jobID,
			Status: model.JobStatusPending,
		}, nil).AnyTimes()

	// Connection repository returns error
	mocks.connRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"postgres_db"}).
		Return(nil, errors.New("connection not found"))

	// Expect job transition to processing before extraction
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).
		Return(nil)

	// Expect job status to be updated to failed
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	// Expect failure notification to be published
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err = uc.ExtractExternalData(ctx, body, nil)
	if err == nil {
		t.Fatal("expected error when connection not found, got nil")
	}
}

// TestExtractExternalData_JobNotFound tests behavior when job is not found in repository.
func TestExtractExternalData_JobNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"datasource1": {"table1": {"field1"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	// First check - job is not found (nil status means process)
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(nil, nil)

	// Second check - job not found returns error
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(nil, errors.New("job not found"))

	// Expect job status update to failed
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	// Expect failure notification
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err = uc.ExtractExternalData(ctx, body, nil)
	if err == nil {
		t.Fatal("expected error when job not found, got nil")
	}
}

// TestExtractExternalDataMessage_JSON tests JSON serialization of the message.
func TestExtractExternalDataMessage_JSON(t *testing.T) {
	jobID := uuid.New()

	msg := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"db1": {"table1": {"field1", "field2"}},
			"db2": {"table2": {"field3"}},
		},
		Metadata: map[string]any{"source": "test", "version": "1.0"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ExtractExternalDataMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.JobID != jobID {
		t.Errorf("expected jobID %s, got %s", jobID, decoded.JobID)
	}

	if len(decoded.MappedFields) != 2 {
		t.Errorf("expected 2 mapped fields, got %d", len(decoded.MappedFields))
	}

	if decoded.Metadata["source"] != "test" {
		t.Errorf("expected source 'test', got %v", decoded.Metadata["source"])
	}
}
