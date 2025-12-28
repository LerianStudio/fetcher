package services

import (
	"context"
	"encoding/json"
	"fmt"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

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
}

// publishJobNotification publishes a job event notification to RabbitMQ topic exchange.
func (uc *UseCase) publishJobNotification(
	ctx context.Context,
	tracer trace.Tracer,
	message ExtractExternalDataMessage,
	status string,
	errorMetadata map[string]any,
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

	if err := uc.RabbitMQPublisher.Publish(ctx, uc.JobEventsExchange, routingKey, notificationJSON); err != nil {
		libOtel.HandleSpanError(&notifySpan, "Error publishing job notification to RabbitMQ", err)
		logger.Errorf("Error publishing job notification to RabbitMQ: %s", err.Error())

		return fmt.Errorf("publishing job notification: %w", err)
	}

	logger.Infof("Successfully published job notification: jobId=%s, status=%s, routingKey=%s", message.JobID, status, routingKey)

	return nil
}
