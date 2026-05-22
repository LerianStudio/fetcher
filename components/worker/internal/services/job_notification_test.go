package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	libOutbox "github.com/LerianStudio/lib-commons/v5/commons/outbox"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libLog "github.com/LerianStudio/lib-observability/log"
	streaming "github.com/LerianStudio/lib-streaming"
	"github.com/LerianStudio/lib-streaming/streamingtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func publishJobNotificationForTest(t *testing.T, uc *UseCase, ctx context.Context, message ExtractExternalDataMessage, status string, errorMetadata map[string]any, opts *JobNotificationOptions, logger libLog.Logger) error {
	t.Helper()

	payload, err := buildJobNotificationPayload(message, status, errorMetadata, opts)
	if err != nil {
		return fmt.Errorf("marshalling job notification: %w", err)
	}

	return uc.publishJobNotificationPayload(ctx, status, message.JobID.String(), payload, logger)
}

// TestPublishJobNotification_Success tests successful job notification publishing.
func TestPublishJobNotification_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test-service"},
	}

	// Expect Publish to be called with correct parameters
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		DoAndReturn(func(_ context.Context, exchange, routingKey string, body []byte) error {
			// Verify the message body
			var notification JobNotificationMessage
			if err := json.Unmarshal(body, &notification); err != nil {
				t.Errorf("failed to unmarshal notification: %v", err)
			}
			if notification.JobID != jobID {
				t.Errorf("expected jobID %s, got %s", jobID, notification.JobID)
			}
			if notification.Status != "completed" {
				t.Errorf("expected status 'completed', got %s", notification.Status)
			}
			return nil
		})

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestEmitJobNotificationEvent_WithTenantContext_SetsOutboxTenantContext(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-123"
	emitter := &capturingEmitter{}
	uc := &UseCase{
		JobEventEmitter:                emitter,
		JobEventStreamingEnabled:       true,
		JobEventStreamingRequireTenant: true,
	}

	ctx := tmcore.ContextWithTenantID(testContext(), tenantID)
	err := uc.emitJobNotificationEvent(ctx, "completed", uuid.NewString(), []byte(`{"ok":true}`))
	require.NoError(t, err)
	require.Equal(t, tenantID, emitter.outboxTenantID)
	require.Equal(t, tenantID, emitter.requestTenantID)
}

type capturingEmitter struct {
	outboxTenantID  string
	requestTenantID string
}

func (e *capturingEmitter) Emit(ctx context.Context, request streaming.EmitRequest) error {
	tenantID, _ := libOutbox.TenantIDFromContext(ctx)
	e.outboxTenantID = tenantID
	e.requestTenantID = request.TenantID

	return nil
}

func (e *capturingEmitter) Close() error { return nil }

func (e *capturingEmitter) Healthy(context.Context) error { return nil }

// TestPublishJobNotification_WithErrorMetadata tests publishing failed job notifications.
func TestPublishJobNotification_WithErrorMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test-service"},
	}

	errorMetadata := map[string]any{
		"code":    "CONNECTION_FAILED",
		"message": "Failed to connect to database",
	}

	// Expect Publish to be called with error routing key
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
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

	err := publishJobNotificationForTest(t, uc, ctx, message, "failed", errorMetadata, nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestPublishJobNotification_EmitsLibStreamingEventWhenTenantPresent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	emitter := streamingtest.NewMockEmitter()
	uc.JobEventEmitter = emitter

	ctx := tmcore.ContextWithTenantID(testContext(), "tenant-job-events")
	jobID := newTestJobID()
	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test-service"},
	}

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, testLogger())
	require.NoError(t, err)

	streamingtest.AssertEventEmitted(t, emitter, "job.completed")
	streamingtest.AssertTenantID(t, emitter, "tenant-job-events")
	requests := emitter.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, jobID.String(), requests[0].Subject)
	var notification JobNotificationMessage
	require.NoError(t, json.Unmarshal(requests[0].Payload, &notification))
	assert.Equal(t, "completed", notification.Status)
}

func TestPublishJobNotification_StreamingFailureFailsWithoutLegacyPublish(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	emitter := streamingtest.NewMockEmitter()
	emitter.SetError(errors.New("streaming broker unavailable"))
	uc.JobEventEmitter = emitter

	ctx := tmcore.ContextWithTenantID(testContext(), "tenant-job-events")
	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test-service"},
	}

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, testLogger())
	require.Error(t, err)
	assert.Empty(t, emitter.Requests())
}

func TestPublishJobNotification_StreamingOnlyInfraFailureFails(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	emitter := streamingtest.NewMockEmitter()
	emitter.SetError(errors.New("streaming route unavailable"))
	uc.JobEventEmitter = emitter
	uc.JobEventStreamingEnabled = true

	ctx := tmcore.ContextWithTenantID(testContext(), "tenant-job-events")
	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test-service"},
	}

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, testLogger())
	require.Error(t, err)
}

func TestPublishJobNotification_StreamingCallerErrorStillFails(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	emitter := streamingtest.NewMockEmitter()
	emitter.SetError(streaming.ErrMissingTenantID)
	uc.JobEventEmitter = emitter
	uc.JobEventStreamingEnabled = true

	ctx := tmcore.ContextWithTenantID(testContext(), "tenant-job-events")
	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test-service"},
	}

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, testLogger())
	require.Error(t, err)
	assert.ErrorIs(t, err, streaming.ErrMissingTenantID)
}

func TestPublishJobNotification_StreamingRequireTenantRejectsMissingContext(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	emitter := streamingtest.NewMockEmitter()
	uc.JobEventEmitter = emitter
	uc.JobEventStreamingRequireTenant = true

	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test-service"},
	}

	err := publishJobNotificationForTest(t, uc, testContext(), message, "completed", nil, nil, testLogger())
	require.Error(t, err)
	assert.ErrorIs(t, err, streaming.ErrMissingTenantID)
	assert.Empty(t, emitter.Requests())
}

func TestPublishJobNotification_SingleTenantUsesStableFallbackTenant(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	emitter := streamingtest.NewMockEmitter()
	uc.JobEventEmitter = emitter

	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test-service"},
	}

	require.NoError(t, publishJobNotificationForTest(t, uc, testContext(), message, "completed", nil, nil, testLogger()))
	streamingtest.AssertTenantID(t, emitter, "single-tenant")
}

// TestPublishJobNotification_PublisherNotConfigured tests that no error is returned when publisher is nil.
func TestPublishJobNotification_PublisherNotConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	// Create UseCase without publisher
	uc := &UseCase{
		ExternalDataStorage:      mocks.seaweedFS,
		JobRepository:            mocks.jobRepo,
		ConnectionRepository:     mocks.connRepo,
		Cryptor:                  mocks.cryptor,
		FileTTL:                  "1h",
		JobEventEmitter:          nil,
		JobEventStreamingEnabled: true,
	}

	ctx := testContext()
	logger := testLogger()

	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test"},
	}

	// Mandatory streaming must fail closed when no emitter is configured.
	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, logger)
	if err == nil {
		t.Fatal("expected error when mandatory job event emitter is not configured")
	}
}

func TestPublishJobNotification_NilLoggerFallsBackSafely(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test-service"},
	}

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		Return(nil)

	require.NotPanics(t, func() {
		err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, nil)
		require.NoError(t, err)
	})
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
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test-service"},
	}

	expectedErr := errors.New("connection refused")

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).Return(expectedErr)

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, logger)
	if err == nil {
		t.Fatal("expected error when publish fails, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Logf("error chain: %v", err)
	}
}

// TestPublishJobNotification_UnknownSource keeps source in payload metadata without changing the event key.
func TestPublishJobNotification_UnknownSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	// Message without source in metadata
	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: nil,
	}

	// Expect routing key with "unknown" source
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		Return(nil)

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestJobNotificationMessage_JSON tests JSON serialization of the notification message.
func TestJobNotificationMessage_JSON(t *testing.T) {
	jobID := uuid.New()

	msg := JobNotificationMessage{
		JobID:    jobID,
		Status:   "completed",
		Metadata: map[string]any{"source": "test", "key": "value"},
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

	if decoded.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", decoded.Status)
	}
}

// TestPublishJobNotification_WithResultData tests notification with result data.
func TestPublishJobNotification_WithResultData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test-service"},
	}

	resultData := &JobResultData{
		Path:      "/bucket/result.json",
		SizeBytes: 1024,
		RowCount:  50,
		Format:    "json",
	}

	opts := &JobNotificationOptions{
		Result:          resultData,
		ExecutionTimeMs: 5000,
	}

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		DoAndReturn(func(_ context.Context, exchange, routingKey string, body []byte) error {
			var notification JobNotificationMessage
			if err := json.Unmarshal(body, &notification); err != nil {
				t.Errorf("failed to unmarshal notification: %v", err)
			}
			if notification.Result == nil {
				t.Error("expected result data to be present")
			}
			if notification.Result.Path != "/bucket/result.json" {
				t.Errorf("expected path '/bucket/result.json', got %s", notification.Result.Path)
			}
			if notification.ExecutionTimeMs != 5000 {
				t.Errorf("expected executionTimeMs 5000, got %d", notification.ExecutionTimeMs)
			}
			return nil
		})

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, opts, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestPublishJobNotification_EmptyExchange tests when exchange is not configured.
func TestPublishJobNotification_EmptyExchange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	// Create UseCase with empty exchange
	uc := &UseCase{
		ExternalDataStorage:      mocks.seaweedFS,
		JobRepository:            mocks.jobRepo,
		ConnectionRepository:     mocks.connRepo,
		Cryptor:                  mocks.cryptor,
		FileTTL:                  "1h",
		JobEventEmitter:          nil,
		JobEventStreamingEnabled: true,
	}

	ctx := testContext()
	logger := testLogger()

	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test"},
	}

	// Mandatory streaming must fail closed when no emitter is configured.
	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, logger)
	if err == nil {
		t.Fatal("expected error when mandatory job event emitter is not configured")
	}
}

// TestPublishJobNotification_MetadataPreservation tests that metadata is preserved.
func TestPublishJobNotification_MetadataPreservation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	message := ExtractExternalDataMessage{
		JobID: newTestJobID(),
		Metadata: map[string]any{
			"source":      "test-service",
			"requestId":   "req-123",
			"customField": "custom-value",
		},
	}

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		DoAndReturn(func(_ context.Context, exchange, routingKey string, body []byte) error {
			var notification JobNotificationMessage
			if err := json.Unmarshal(body, &notification); err != nil {
				t.Errorf("failed to unmarshal notification: %v", err)
			}
			if notification.Metadata["source"] != "test-service" {
				t.Errorf("expected source 'test-service', got %v", notification.Metadata["source"])
			}
			if notification.Metadata["requestId"] != "req-123" {
				t.Errorf("expected requestId 'req-123', got %v", notification.Metadata["requestId"])
			}
			if notification.Metadata["customField"] != "custom-value" {
				t.Errorf("expected customField 'custom-value', got %v", notification.Metadata["customField"])
			}
			return nil
		})

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestPublishJobNotification_EventKeyGeneration tests stable lib-streaming event key selection.
func TestPublishJobNotification_EventKeyGeneration(t *testing.T) {
	tests := []struct {
		name             string
		status           string
		metadata         map[string]any
		expectedEventKey string
	}{
		{
			name:             "completed with known source",
			status:           "completed",
			metadata:         map[string]any{"source": "api-gateway"},
			expectedEventKey: "job.completed",
		},
		{
			name:             "failed with known source",
			status:           "failed",
			metadata:         map[string]any{"source": "scheduler"},
			expectedEventKey: "job.failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := newTestMocks(ctrl)
			uc := newTestUseCase(mocks)

			ctx := testContext()
			logger := testLogger()

			message := ExtractExternalDataMessage{
				JobID:    newTestJobID(),
				Metadata: tt.metadata,
			}

			mocks.rabbitPublisher.EXPECT().
				Publish(gomock.Any(), "test-exchange", tt.expectedEventKey, gomock.Any()).
				Return(nil)

			err := publishJobNotificationForTest(t, uc, ctx, message, tt.status, nil, nil, logger)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

// TestPublishJobNotification_WithCompletedAt tests notification with completion timestamp.
func TestPublishJobNotification_WithCompletedAt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	completedTime := time.Now()
	opts := &JobNotificationOptions{
		CompletedAt: &completedTime,
	}

	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test"},
	}

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		DoAndReturn(func(_ context.Context, exchange, routingKey string, body []byte) error {
			var notification JobNotificationMessage
			if err := json.Unmarshal(body, &notification); err != nil {
				t.Errorf("failed to unmarshal notification: %v", err)
			}
			if notification.CompletedAt == nil {
				t.Error("expected completedAt to be present")
			}
			return nil
		})

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, opts, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestPublishJobNotification_WithAllOptions tests notification with all optional fields.
func TestPublishJobNotification_WithAllOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	completedTime := time.Now()
	opts := &JobNotificationOptions{
		Result: &JobResultData{
			Path:      "/path/to/result",
			SizeBytes: 2048,
			RowCount:  100,
			Format:    "json",
		},
		ExecutionTimeMs: 1500,
		CompletedAt:     &completedTime,
	}

	message := ExtractExternalDataMessage{
		JobID:    newTestJobID(),
		Metadata: map[string]any{"source": "test", "custom": "value"},
	}

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		DoAndReturn(func(_ context.Context, exchange, routingKey string, body []byte) error {
			var notification JobNotificationMessage
			if err := json.Unmarshal(body, &notification); err != nil {
				t.Errorf("failed to unmarshal notification: %v", err)
			}
			if notification.Result == nil {
				t.Error("expected result to be present")
			}
			if notification.ExecutionTimeMs != 1500 {
				t.Errorf("expected executionTimeMs 1500, got %d", notification.ExecutionTimeMs)
			}
			if notification.CompletedAt == nil {
				t.Error("expected completedAt to be present")
			}
			if notification.Metadata["custom"] != "value" {
				t.Error("expected custom metadata to be preserved")
			}
			return nil
		})

	err := publishJobNotificationForTest(t, uc, ctx, message, "completed", nil, opts, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
