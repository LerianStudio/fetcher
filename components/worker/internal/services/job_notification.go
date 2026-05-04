package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

var invalidRoutingSourceChars = regexp.MustCompile(`[^a-z0-9_-]+`)

const maxRoutingSourceSegmentLength = 64

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

// publishJobNotification publishes a job event notification to RabbitMQ topic exchange.
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

	// Skip if publisher is not configured
	if uc.RabbitMQPublisher == nil || uc.JobEventsExchange == "" {
		logger.Log(ctx, libLog.LevelDebug, "rabbitmq publisher not configured, skipping job notification")

		return nil
	}

	var notificationTracer trace.Tracer
	if tracer != nil {
		notificationTracer = tracer
	} else {
		_, notificationTracer, _, _ = libCommons.NewTrackingFromContext(ctx)
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

	source := "unknown"

	if notification.Metadata != nil {
		if src, ok := notification.Metadata["source"].(string); ok && src != "" {
			source = sanitizeRoutingSourceSegment(src)
		}
	}

	routingKey := fmt.Sprintf("job.%s.%s", status, source)

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		libOtel.HandleSpanError(notifySpan, "Error marshalling job notification", err)
		logger.Log(ctx, libLog.LevelError, "error marshalling job notification", libLog.Err(err))

		return fmt.Errorf("marshalling job notification: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "publishing job notification",
		libLog.String("job_id", message.JobID.String()),
		libLog.String("status", status),
		libLog.String("routing_key", routingKey),
		libLog.String("exchange", uc.JobEventsExchange),
	)

	if err := uc.RabbitMQPublisher.Publish(ctx, uc.JobEventsExchange, routingKey, notificationJSON); err != nil {
		libOtel.HandleSpanError(notifySpan, "Error publishing job notification to RabbitMQ", err)
		logger.Log(ctx, libLog.LevelError, "error publishing job notification to RabbitMQ", libLog.Err(err))

		return fmt.Errorf("publishing job notification: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "published job notification successfully",
		libLog.String("job_id", message.JobID.String()),
		libLog.String("status", status),
		libLog.String("routing_key", routingKey),
	)

	return nil
}

func normalizeJobNotificationLogger(ctx context.Context, logger libLog.Logger) libLog.Logger {
	if logger != nil {
		return logger
	}

	ctxLogger := libCommons.NewLoggerFromContext(ctx)
	if ctxLogger != nil {
		return ctxLogger
	}

	return &libLog.GoLogger{Level: libLog.LevelError}
}

func sanitizeRoutingSourceSegment(source string) string {
	normalized := strings.ToLower(strings.TrimSpace(source))
	normalized = invalidRoutingSourceChars.ReplaceAllString(normalized, "-")

	normalized = strings.Trim(normalized, "-_")
	if len(normalized) > maxRoutingSourceSegmentLength {
		normalized = normalized[:maxRoutingSourceSegmentLength]
		normalized = strings.Trim(normalized, "-_")
	}

	if normalized == "" {
		return "unknown"
	}

	return normalized
}
