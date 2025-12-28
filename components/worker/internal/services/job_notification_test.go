package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// TestPublishJobNotification_Success tests successful job notification publishing.
func TestPublishJobNotification_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()
	orgID := newTestOrgID()

	message := ExtractExternalDataMessage{
		JobID:          jobID,
		OrganizationID: orgID,
		Metadata:       map[string]any{"source": "test-service"},
	}

	// Expect Publish to be called with correct parameters
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed.test-service", gomock.Any()).
		DoAndReturn(func(_ context.Context, exchange, routingKey string, body []byte) error {
			// Verify the message body
			var notification JobNotificationMessage
			if err := json.Unmarshal(body, &notification); err != nil {
				t.Errorf("failed to unmarshal notification: %v", err)
			}
			if notification.JobID != jobID {
				t.Errorf("expected jobID %s, got %s", jobID, notification.JobID)
			}
			if notification.OrganizationID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, notification.OrganizationID)
			}
			if notification.Status != "completed" {
				t.Errorf("expected status 'completed', got %s", notification.Status)
			}
			return nil
		})

	err := uc.publishJobNotification(ctx, nil, message, "completed", nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestPublishJobNotification_WithErrorMetadata tests publishing failed job notifications.
func TestPublishJobNotification_WithErrorMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()
	orgID := newTestOrgID()

	message := ExtractExternalDataMessage{
		JobID:          jobID,
		OrganizationID: orgID,
		Metadata:       map[string]any{"source": "test-service"},
	}

	errorMetadata := map[string]any{
		"code":    "CONNECTION_FAILED",
		"message": "Failed to connect to database",
	}

	// Expect Publish to be called with error routing key
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed.test-service", gomock.Any()).
		DoAndReturn(func(_ context.Context, exchange, routingKey string, body []byte) error {
			// Verify the message includes error metadata
			var notification JobNotificationMessage
			if err := json.Unmarshal(body, &notification); err != nil {
				t.Errorf("failed to unmarshal notification: %v", err)
			}
			if notification.Status != "failed" {
				t.Errorf("expected status 'failed', got %s", notification.Status)
			}
			if notification.Metadata["error"] == nil {
				t.Error("expected error metadata to be present")
			}
			return nil
		})

	err := uc.publishJobNotification(ctx, nil, message, "failed", errorMetadata, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestPublishJobNotification_PublisherNotConfigured tests that no error is returned when publisher is nil.
func TestPublishJobNotification_PublisherNotConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	// Create UseCase without publisher
	uc := &UseCase{
		ExternalDataSeaweedFS: mocks.seaweedFS,
		JobRepository:         mocks.jobRepo,
		ConnectionRepository:  mocks.connRepo,
		Cryptor:               mocks.cryptor,
		FileTTL:               "1h",
		RabbitMQPublisher:     nil, // No publisher
		JobEventsExchange:     "",
	}

	ctx := testContext()
	logger := testLogger()

	message := ExtractExternalDataMessage{
		JobID:          newTestJobID(),
		OrganizationID: newTestOrgID(),
		Metadata:       map[string]any{"source": "test"},
	}

	// Should return nil without calling publish
	err := uc.publishJobNotification(ctx, nil, message, "completed", nil, logger)
	if err != nil {
		t.Fatalf("expected no error when publisher not configured, got: %v", err)
	}
}

// TestPublishJobNotification_PublishError tests error handling when publish fails.
func TestPublishJobNotification_PublishError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	message := ExtractExternalDataMessage{
		JobID:          newTestJobID(),
		OrganizationID: newTestOrgID(),
		Metadata:       map[string]any{"source": "test-service"},
	}

	expectedErr := errors.New("connection refused")

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(expectedErr)

	err := uc.publishJobNotification(ctx, nil, message, "completed", nil, logger)
	if err == nil {
		t.Fatal("expected error when publish fails, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Logf("error chain: %v", err)
	}
}

// TestPublishJobNotification_UnknownSource tests routing key generation with unknown source.
func TestPublishJobNotification_UnknownSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	// Message without source in metadata
	message := ExtractExternalDataMessage{
		JobID:          newTestJobID(),
		OrganizationID: newTestOrgID(),
		Metadata:       nil,
	}

	// Expect routing key with "unknown" source
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed.unknown", gomock.Any()).
		Return(nil)

	err := uc.publishJobNotification(ctx, nil, message, "completed", nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestJobNotificationMessage_JSON tests JSON serialization of the notification message.
func TestJobNotificationMessage_JSON(t *testing.T) {
	jobID := uuid.New()
	orgID := uuid.New()

	msg := JobNotificationMessage{
		JobID:          jobID,
		OrganizationID: orgID,
		Status:         "completed",
		Metadata:       map[string]any{"source": "test", "key": "value"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded JobNotificationMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.JobID != jobID {
		t.Errorf("expected jobID %s, got %s", jobID, decoded.JobID)
	}

	if decoded.OrganizationID != orgID {
		t.Errorf("expected orgID %s, got %s", orgID, decoded.OrganizationID)
	}

	if decoded.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", decoded.Status)
	}
}
