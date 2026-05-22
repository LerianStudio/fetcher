package services

import (
	"context"
	"fmt"

	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	"go.opentelemetry.io/otel/trace"
)

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
		err := fmt.Errorf("mandatory lib-streaming job event emission is disabled")
		logger.Log(ctx, libLog.LevelError, "mandatory lib-streaming job event emission is disabled", libLog.Err(err))

		return err
	}

	var notificationTracer trace.Tracer
	if tracer != nil {
		notificationTracer = tracer
	} else {
		_, notificationTracer, _, _ = observability.NewTrackingFromContext(ctx)
	}

	ctx, notifySpan := notificationTracer.Start(ctx, "service.publish_job_notification")
	defer notifySpan.End()

	notificationJSON, err := buildJobNotificationPayload(message, status, errorMetadata, opts)
	if err != nil {
		libOtel.HandleSpanError(notifySpan, "Error marshalling job notification", err)
		logger.Log(ctx, libLog.LevelError, "error marshalling job notification", libLog.Err(err))

		return fmt.Errorf("marshalling job notification: %w", err)
	}

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
