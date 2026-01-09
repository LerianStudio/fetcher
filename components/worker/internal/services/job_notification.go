package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

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

	// OrganizationID is the unique identifier of the organization.
	OrganizationID uuid.UUID `json:"organizationId"`

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
	logger log.Logger,
) error {
	// Skip if publisher is not configured
	if uc.RabbitMQPublisher == nil || uc.JobEventsExchange == "" {
		if logger != nil {
			logger.Debug("RabbitMQ publisher not configured, skipping job notification")
		}

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
		JobID:          message.JobID,
		OrganizationID: message.OrganizationID,
		Status:         status,
		Metadata:       make(map[string]any),
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
			source = src
		}
	}

	routingKey := fmt.Sprintf("job.%s.%s", status, source)

	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		libOtel.HandleSpanError(&notifySpan, "Error marshalling job notification", err)
		logger.Errorf("Error marshalling job notification: %s", err.Error())

		return fmt.Errorf("marshalling job notification: %w", err)
	}

	logger.Infof("Publishing job notification: jobId=%s, status=%s, routingKey=%s, exchange=%s",
		message.JobID, status, routingKey, uc.JobEventsExchange)

	if err := uc.RabbitMQPublisher.Publish(ctx, uc.JobEventsExchange, routingKey, notificationJSON); err != nil {
		libOtel.HandleSpanError(&notifySpan, "Error publishing job notification to RabbitMQ", err)
		logger.Errorf("Error publishing job notification to RabbitMQ: %s", err.Error())

		return fmt.Errorf("publishing job notification: %w", err)
	}

	logger.Infof("Successfully published job notification: jobId=%s, status=%s, routingKey=%s", message.JobID, status, routingKey)

	return nil
}
