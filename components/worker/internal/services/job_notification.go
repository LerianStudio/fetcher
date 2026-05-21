package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	observability "github.com/LerianStudio/lib-observability"
	streaming "github.com/LerianStudio/lib-streaming"

	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// JobResultData contains information about the extraction result.
// All fields use omitempty to only include data when provided.
type JobResultData struct {
	// Path is the SeaweedFS path where result data is stored.
	Path string `json:"path,omitempty"`

	// SizeBytes is the size of the result data in bytes (before encryption).
	SizeBytes int64 `json:"sizeBytes,omitempty"`

	// RowCount is the total number of records extracted across all tables.
	RowCount int64 `json:"rowCount,omitempty"`

	// Format is the output format (e.g., "json").
	Format string `json:"format,omitempty"`

	// HMAC is the HMAC-SHA256 signature of the result data (before encryption).
	// Consumers can use this to verify data integrity using the external HMAC key.
	HMAC string `json:"hmac,omitempty"`
}

// JobNotificationOptions contains optional data for job notifications.
type JobNotificationOptions struct {
	// Result contains extraction result data (path, size, rowCount, format).
	Result *JobResultData

	// ExecutionTimeMs is the total execution time in milliseconds.
	ExecutionTimeMs int64

	// CompletedAt is the timestamp when the job completed.
	CompletedAt *time.Time
}

// JobNotificationMessage represents the structure of a job event notification published to RabbitMQ.
type JobNotificationMessage struct {
	// JobID is the unique identifier of the job.
	JobID uuid.UUID `json:"jobId"`

	// Metadata contains additional metadata for the notification.
	// It should include "source" to identify which service requested the job.
	// For failed jobs, it may include "error" with error details.
	Metadata map[string]any `json:"metadata"`

	// Status indicates the job status: "completed" or "failed".
	Status string `json:"status"`

	// Result contains information about the extraction result (optional, only on success).
	Result *JobResultData `json:"result,omitempty"`

	// ExecutionTimeMs is the total execution time in milliseconds (optional).
	ExecutionTimeMs int64 `json:"executionTimeMs,omitempty"`

	// CompletedAt is the timestamp when the job completed (optional).
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// publishJobNotification publishes a job event notification through lib-streaming.
// Legacy direct RabbitMQ business-event routing is intentionally disabled: the
// event contract is the lib-streaming definition key (job.completed/job.failed),
// while source remains payload metadata instead of a routing-key segment.
func (uc *UseCase) publishJobNotification(
	ctx context.Context,
	tracer trace.Tracer,
	message ExtractExternalDataMessage,
	status string,
	errorMetadata map[string]any,
	opts *JobNotificationOptions,
	logger libLog.Logger,
) error {
	logger = normalizeJobNotificationLogger(ctx, logger)

	if !uc.JobEventStreamingEnabled {
		logger.Log(ctx, libLog.LevelDebug, "lib-streaming job event emission disabled, skipping job notification")

		return nil
	}

	var notificationTracer trace.Tracer
	if tracer != nil {
		notificationTracer = tracer
	} else {
		_, notificationTracer, _, _ = observability.NewTrackingFromContext(ctx)
	}

	ctx, notifySpan := notificationTracer.Start(ctx, "service.publish_job_notification")
	defer notifySpan.End()

	notification := JobNotificationMessage{
		JobID:    message.JobID,
		Status:   status,
		Metadata: make(map[string]any),
	}

	// Add optional result and execution data
	if opts != nil {
		notification.Result = opts.Result
		notification.ExecutionTimeMs = opts.ExecutionTimeMs
		notification.CompletedAt = opts.CompletedAt
	}

	if message.Metadata != nil {
		for k, v := range message.Metadata {
			notification.Metadata[k] = v
		}
	}

	if status == "failed" && errorMetadata != nil {
		notification.Metadata["error"] = errorMetadata
	}

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		libOtel.HandleSpanError(notifySpan, "Error marshalling job notification", err)
		logger.Log(ctx, libLog.LevelError, "error marshalling job notification", libLog.Err(err))

		return fmt.Errorf("marshalling job notification: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "publishing job notification",
		libLog.String("job_id", message.JobID.String()),
		libLog.String("status", status),
		libLog.String("event_key", fmt.Sprintf("job.%s", status)),
	)

	if err := uc.publishJobNotificationViaStreaming(ctx, notifySpan, status, message.JobID.String(), notificationJSON, logger); err != nil {
		return err
	}

	return nil
}

func (uc *UseCase) publishJobNotificationViaStreaming(ctx context.Context, span trace.Span, status, jobID string, payload []byte, logger libLog.Logger) error {
	if err := uc.emitJobNotificationEvent(ctx, status, jobID, payload); err != nil {
		libOtel.HandleSpanError(span, "Error emitting job notification with lib-streaming", err)
		logger.Log(ctx, libLog.LevelError, "error emitting job notification with lib-streaming", libLog.Err(err))

		return fmt.Errorf("emitting job notification event: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "published job notification successfully",
		libLog.String("job_id", jobID),
		libLog.String("status", status),
		libLog.String("event_key", fmt.Sprintf("job.%s", status)),
	)

	return nil
}

func (uc *UseCase) emitJobNotificationEvent(ctx context.Context, status, subject string, payload []byte) error {
	if uc.JobEventEmitter == nil {
		return nil
	}

	tenantID := core.GetTenantIDContext(ctx)
	if tenantID == "" {
		if uc.JobEventStreamingRequireTenant {
			return streaming.ErrMissingTenantID
		}

		tenantID = "single-tenant"
	}

	return uc.JobEventEmitter.Emit(ctx, streaming.EmitRequest{
		DefinitionKey: fmt.Sprintf("job.%s", status),
		TenantID:      tenantID,
		Subject:       subject,
		Payload:       payload,
	})
}

func normalizeJobNotificationLogger(ctx context.Context, logger libLog.Logger) libLog.Logger {
	if logger != nil {
		return logger
	}

	ctxLogger := observability.NewLoggerFromContext(ctx)
	if ctxLogger != nil {
		return ctxLogger
	}

	return &libLog.GoLogger{Level: libLog.LevelError}
}
