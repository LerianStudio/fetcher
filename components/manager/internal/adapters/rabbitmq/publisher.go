// Package rabbitmq provides RabbitMQ message publishing adapters for the Manager component.
package rabbitmq

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/ports/messaging"
	libCommons "github.com/LerianStudio/lib-commons/v3/commons"
	libConstants "github.com/LerianStudio/lib-commons/v3/commons/constants"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	tmcore "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/core"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
)

// Ensure MultiTenantPublisher implements MessagePublisher at compile time.
var _ messaging.MessagePublisher = (*MultiTenantPublisher)(nil)

// MultiTenantPublisher implements messaging.MessagePublisher using tmrabbitmq.Manager
// for per-tenant vhost isolation (Layer 1 of multi-tenant RabbitMQ model).
//
// Per multi-tenant.md standards:
//   - Layer 1 (Vhost Isolation): Uses tmrabbitmq.Manager.GetChannel(ctx, tenantID)
//   - Layer 2 (X-Tenant-ID Header): Injected for audit/tracing
type MultiTenantPublisher struct {
	rmqManager *tmrabbitmq.Manager
	logger     log.Logger
	telemetry  *libOtel.Telemetry
}

// NewMultiTenantPublisher creates a new multi-tenant RabbitMQ publisher.
// This publisher uses tmrabbitmq.Manager to get per-tenant RabbitMQ channels.
func NewMultiTenantPublisher(
	rmqManager *tmrabbitmq.Manager,
	logger log.Logger,
	telemetry *libOtel.Telemetry,
) *MultiTenantPublisher {
	return &MultiTenantPublisher{
		rmqManager: rmqManager,
		logger:     logger,
		telemetry:  telemetry,
	}
}

// ProducerDefault publishes a message to the specified exchange with the given routing key
// using a tenant-specific RabbitMQ channel (vhost isolation).
// This method implements the messaging.MessagePublisher interface.
func (p *MultiTenantPublisher) ProducerDefault(ctx context.Context, exchange, key string, queueMessage []byte, header *map[string]any) error {
	_, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "adapter.rabbitmq.produce_multi_tenant")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("messaging.exchange", exchange),
		attribute.String("messaging.routing_key", key),
		attribute.Int("messaging.body_size", len(queueMessage)),
	)

	// Get tenant ID from context (required in multi-tenant mode)
	tenantID := tmcore.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		err := fmt.Errorf("tenant ID is required in multi-tenant mode")
		libOtel.HandleSpanError(&span, "Missing tenant ID", err)
		p.logger.Error("Tenant ID is required in multi-tenant mode")

		return err
	}

	span.SetAttributes(attribute.String("tenant.id", tenantID))

	p.logger.Debugf("Publishing message to exchange=%s, key=%s, tenant=%s", exchange, key, tenantID)

	// Get tenant-specific channel from the RabbitMQ manager (Layer 1: Vhost Isolation)
	channel, err := p.rmqManager.GetChannel(ctx, tenantID)
	if err != nil {
		libOtel.HandleSpanError(&span, "Failed to get tenant RabbitMQ channel", err)
		p.logger.Errorf("Failed to get RabbitMQ channel for tenant %s: %v", tenantID, err)

		return fmt.Errorf("get tenant RabbitMQ channel: %w", err)
	}

	defer func() {
		// Close the channel after publishing (per lib-commons docs: caller owns channel lifecycle)
		if closeErr := channel.Close(); closeErr != nil {
			p.logger.Warnf("Failed to close RabbitMQ channel for tenant %s: %v", tenantID, closeErr)
		}
	}()

	// Build AMQP headers with X-Tenant-ID (Layer 2: Header for audit/tracing)
	headers := amqp.Table{
		"X-Tenant-ID":         tenantID,
		libConstants.HeaderID: reqID,
	}

	// Copy existing headers if provided, but protect reserved headers from overwrite
	if header != nil {
		for k, v := range *header {
			if k == "X-Tenant-ID" || k == libConstants.HeaderID {
				continue
			}

			headers[k] = v
		}
	}

	// Inject trace context
	libOtel.InjectTraceHeadersIntoQueue(ctx, (*map[string]any)(&headers))

	// Publish the message
	err = channel.PublishWithContext(ctx, exchange, key, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
			Body:         queueMessage,
		})
	if err != nil {
		libOtel.HandleSpanError(&span, "Failed to publish message", err)
		p.logger.Errorf("Error publishing message to exchange %s with key %s: %v", exchange, key, err)

		return fmt.Errorf("failed to publish message to exchange %s: %w", exchange, err)
	}

	p.logger.Debugf("Successfully published message to exchange=%s, key=%s, tenant=%s", exchange, key, tenantID)

	return nil
}
