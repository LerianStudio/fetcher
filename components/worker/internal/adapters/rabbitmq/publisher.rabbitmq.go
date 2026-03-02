// Package rabbitmq provides a wrapper around RabbitMQAdapter for publishing messages to topic exchanges.
package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	portPublisher "github.com/LerianStudio/fetcher/pkg/ports/publisher"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	libCommons "github.com/LerianStudio/lib-commons/v3/commons"
	libConstants "github.com/LerianStudio/lib-commons/v3/commons/constants"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
	"github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v3/commons/rabbitmq"
	tmcore "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/core"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
)

// PublisherRepository is an alias for the port interface.
// The canonical definition lives in pkg/ports/publisher.
type PublisherRepository = portPublisher.Repository

// PublisherRoutes wraps RabbitMQAdapter to support publishing messages to topic exchanges.
type PublisherRoutes struct {
	adapter rabbitmq.Adapter
	log.Logger
	opentelemetry.Telemetry
}

// NewPublisherRoutes creates a new instance of PublisherRoutes using a RabbitMQ connection.
// The signer parameter is optional - pass nil to disable message signing.
func NewPublisherRoutes(conn *libRabbitmq.RabbitMQConnection, logger log.Logger, telemetry *opentelemetry.Telemetry, signer crypto.Signer) *PublisherRoutes {
	opts := rabbitmq.DefaultOptions()
	opts.Signer = signer
	adapter := rabbitmq.NewRabbitMQAdapterWithOptions(conn, opts)

	return NewPublisherRoutesWithAdapter(adapter, logger, telemetry)
}

// NewPublisherRoutesWithAdapter creates a new instance of PublisherRoutes using a specific RabbitMQ adapter.
func NewPublisherRoutesWithAdapter(adapter rabbitmq.Adapter, logger log.Logger, telemetry *opentelemetry.Telemetry) *PublisherRoutes {
	telemetryValue := opentelemetry.Telemetry{}
	if telemetry != nil {
		telemetryValue = *telemetry
	}

	pr := &PublisherRoutes{
		adapter:   adapter,
		Logger:    logger,
		Telemetry: telemetryValue,
	}

	return pr
}

// Publish sends a message to the specified exchange with the given routing key.
func (pr *PublisherRoutes) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	_, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "adapter.rabbitmq.publish")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("messaging.exchange", exchange),
		attribute.String("messaging.routing_key", routingKey),
		attribute.Int("messaging.body_size", len(body)),
	)

	pr.Debugf("Publishing message to exchange=%s, routingKey=%s", exchange, routingKey)

	// Forward tenant ID from context to AMQP headers for multi-tenant isolation.
	// When no tenant context is present (single-tenant mode), headers remain nil.
	var headers *map[string]any

	if tenantID := tmcore.GetTenantIDFromContext(ctx); tenantID != "" {
		h := map[string]any{"X-Tenant-ID": tenantID}
		headers = &h
	}

	if err := pr.adapter.ProducerDefault(ctx, exchange, routingKey, body, headers); err != nil {
		opentelemetry.HandleSpanError(&span, "Failed to publish message", err)
		pr.Errorf("Error publishing message to exchange %s with routing key %s: %v", exchange, routingKey, err)

		return fmt.Errorf("failed to publish message to exchange %s: %w", exchange, err)
	}

	pr.Debugf("Successfully published message to exchange=%s, routingKey=%s", exchange, routingKey)

	return nil
}

// Shutdown gracefully shuts down the RabbitMQ adapter.
func (pr *PublisherRoutes) Shutdown(ctx context.Context) error {
	pr.Info("Shutting down PublisherRoutes...")

	// Shutdown the RabbitMQ adapter
	if err := pr.adapter.Shutdown(ctx); err != nil {
		pr.Errorf("Error shutting down RabbitMQ adapter: %v", err)
		return fmt.Errorf("failed to shutdown RabbitMQ adapter: %w", err)
	}

	pr.Info("PublisherRoutes shutdown complete")

	return nil
}

// MultiTenantPublisherRoutes wraps tmrabbitmq.Manager for publishing messages to tenant-specific vhosts.
// This implements Layer 1 (Vhost Isolation) of the multi-tenant RabbitMQ model.
//
// Per multi-tenant.md standards:
//   - Layer 1 (Vhost Isolation): Uses tmrabbitmq.Manager.GetChannel(ctx, tenantID)
//   - Layer 2 (X-Tenant-ID Header): Injected for audit/tracing (already implemented)
type MultiTenantPublisherRoutes struct {
	rmqManager *tmrabbitmq.Manager
	signer     crypto.Signer
	log.Logger
	opentelemetry.Telemetry
}

// NewPublisherRoutesMultiTenant creates a new instance of MultiTenantPublisherRoutes.
// This publisher uses tmrabbitmq.Manager to get per-tenant RabbitMQ channels.
func NewPublisherRoutesMultiTenant(
	rmqManager *tmrabbitmq.Manager,
	logger log.Logger,
	telemetry *opentelemetry.Telemetry,
	signer crypto.Signer,
) *MultiTenantPublisherRoutes {
	telemetryValue := opentelemetry.Telemetry{}
	if telemetry != nil {
		telemetryValue = *telemetry
	}

	return &MultiTenantPublisherRoutes{
		rmqManager: rmqManager,
		signer:     signer,
		Logger:     logger,
		Telemetry:  telemetryValue,
	}
}

// Publish sends a message to the specified exchange with the given routing key
// using a tenant-specific RabbitMQ channel (vhost isolation).
func (pr *MultiTenantPublisherRoutes) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	if pr.rmqManager == nil {
		return fmt.Errorf("multi-tenant RabbitMQ manager is not initialized")
	}

	_, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "adapter.rabbitmq.publish_multi_tenant")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("messaging.exchange", exchange),
		attribute.String("messaging.routing_key", routingKey),
		attribute.Int("messaging.body_size", len(body)),
	)

	// Get tenant ID from context (required in multi-tenant mode)
	tenantID := tmcore.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		err := fmt.Errorf("tenant ID is required in multi-tenant mode")
		opentelemetry.HandleSpanError(&span, "Missing tenant ID", err)
		pr.Error("Tenant ID is required in multi-tenant mode")

		return err
	}

	span.SetAttributes(attribute.String("tenant.id", tenantID))

	pr.Debugf("Publishing message to exchange=%s, routingKey=%s, tenant=%s", exchange, routingKey, tenantID)

	// Build AMQP headers with X-Tenant-ID (Layer 2: Header for audit/tracing)
	headers := amqp.Table{
		"X-Tenant-ID":         tenantID,
		libConstants.HeaderID: reqID,
	}

	// Inject trace context
	opentelemetry.InjectTraceHeadersIntoQueue(ctx, (*map[string]any)(&headers))

	// Sign message if signer is configured (parity with single-tenant RabbitMQAdapter)
	if pr.signer != nil {
		timestamp := time.Now().UTC().Unix()
		payload := crypto.BuildSignaturePayload(timestamp, body)
		signature := pr.signer.Sign(payload)

		headers["x-message-signature"] = signature
		headers["t"] = fmt.Sprintf("%d", timestamp)
		headers["signature-version"] = pr.signer.SignatureVersion()

		span.SetAttributes(
			attribute.String("messaging.signature.version", pr.signer.SignatureVersion()),
		)
	}

	// Retry loop with exponential backoff and jitter (matches midaz pattern).
	// GetChannel returns a fresh channel each call, so we retry the full
	// get-channel + publish sequence on transient failures.
	const maxRetries = 3

	baseDelay := 200 * time.Millisecond

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := rabbitmq.FullJitter(baseDelay)
			pr.Infof("Retrying publish in %v (attempt %d/%d, tenant=%s)", delay, attempt+1, maxRetries+1, tenantID)

			span.SetAttributes(attribute.Int("messaging.retry_attempt", attempt))

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during publish retry: %w", ctx.Err())
			case <-time.After(delay):
			}

			baseDelay = rabbitmq.NextBackoff(baseDelay)
		}

		// Get tenant-specific channel from the RabbitMQ manager (Layer 1: Vhost Isolation)
		channel, chanErr := pr.rmqManager.GetChannel(ctx, tenantID)
		if chanErr != nil {
			lastErr = chanErr
			pr.Errorf("Failed to get RabbitMQ channel for tenant %s (attempt %d/%d): %v", tenantID, attempt+1, maxRetries+1, chanErr)

			continue
		}

		// Publish the message
		pubErr := channel.PublishWithContext(ctx, exchange, routingKey, false, false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Headers:      headers,
				Body:         body,
			})

		// Close the channel after publishing (per lib-commons docs: caller owns channel lifecycle)
		if closeErr := channel.Close(); closeErr != nil {
			pr.Warnf("Failed to close RabbitMQ channel for tenant %s: %v", tenantID, closeErr)
		}

		if pubErr != nil {
			lastErr = pubErr
			pr.Errorf("Publish failed for tenant %s (attempt %d/%d): %v", tenantID, attempt+1, maxRetries+1, pubErr)

			continue
		}

		// Success
		pr.Debugf("Successfully published message to exchange=%s, routingKey=%s, tenant=%s", exchange, routingKey, tenantID)

		return nil
	}

	opentelemetry.HandleSpanError(&span, "Failed to publish message after all retries", lastErr)
	pr.Errorf("Error publishing message to exchange %s with routing key %s after %d retries: %v", exchange, routingKey, maxRetries, lastErr)

	return fmt.Errorf("failed to publish message to exchange %s after %d retries: %w", exchange, maxRetries, lastErr)
}

// Shutdown is a no-op for multi-tenant publisher as the RabbitMQ manager
// handles connection lifecycle separately.
func (pr *MultiTenantPublisherRoutes) Shutdown(ctx context.Context) error {
	pr.Info("MultiTenantPublisherRoutes shutdown (no-op, manager handles connections)")
	return nil
}
