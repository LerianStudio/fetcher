// Package rabbitmq provides a wrapper around RabbitMQAdapter for publishing messages to topic exchanges.
package rabbitmq

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	portPublisher "github.com/LerianStudio/fetcher/pkg/ports/publisher"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	libCommons "github.com/LerianStudio/lib-commons/v3/commons"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
	"github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v3/commons/rabbitmq"
	tmcore "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/core"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
)

// PublisherRepository is an alias for the port interface.
// The canonical definition lives in pkg/ports/publisher.
type PublisherRepository = portPublisher.Repository

// RabbitMQManagerInterface abstracts tmrabbitmq.Manager for testing.
type RabbitMQManagerInterface interface {
	GetChannel(ctx context.Context, tenantID string) (RabbitMQChannel, error)
}

// RabbitMQChannel abstracts an AMQP channel for publishing.
type RabbitMQChannel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
}

// PublisherRoutes wraps RabbitMQAdapter to support publishing messages to topic exchanges.
// In multi-tenant mode, it uses tmrabbitmq.Manager for per-tenant vhost isolation.
type PublisherRoutes struct {
	adapter         rabbitmq.Adapter         // Used in single-tenant mode
	rabbitMQManager RabbitMQManagerInterface // Used in multi-tenant mode (nil in single-tenant)
	multiTenantMode bool
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

// NewPublisherRoutesMultiTenant creates a new instance of PublisherRoutes configured for
// multi-tenant mode using tmrabbitmq.Manager for per-tenant vhost isolation.
func NewPublisherRoutesMultiTenant(manager RabbitMQManagerInterface, logger log.Logger, telemetry *opentelemetry.Telemetry) *PublisherRoutes {
	telemetryValue := opentelemetry.Telemetry{}
	if telemetry != nil {
		telemetryValue = *telemetry
	}

	return &PublisherRoutes{
		rabbitMQManager: manager,
		multiTenantMode: true,
		Logger:          logger,
		Telemetry:       telemetryValue,
	}
}

// Publish sends a message to the specified exchange with the given routing key.
// In multi-tenant mode, publishes to the tenant-specific vhost via tmrabbitmq.Manager.
// In single-tenant mode, uses the static RabbitMQ adapter.
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

	// Multi-tenant mode: use tmrabbitmq.Manager for per-tenant vhost
	if pr.multiTenantMode && pr.rabbitMQManager != nil {
		tenantID := tmcore.GetTenantIDFromContext(ctx)
		if tenantID == "" {
			opentelemetry.HandleSpanError(&span, "No tenant ID in context for multi-tenant publish", fmt.Errorf("tenant ID required"))
			return fmt.Errorf("multi-tenant publish requires tenant ID in context")
		}

		span.SetAttributes(attribute.String("app.tenant.id", tenantID))

		ch, err := pr.rabbitMQManager.GetChannel(ctx, tenantID)
		if err != nil {
			opentelemetry.HandleSpanError(&span, "Failed to get tenant channel", err)
			pr.Errorf("Error getting channel for tenant %s: %v", tenantID, err)

			return fmt.Errorf("failed to get channel for tenant %s: %w", tenantID, err)
		}

		defer func() {
			if closeErr := ch.Close(); closeErr != nil {
				pr.Errorf("Error closing channel for tenant %s: %v", tenantID, closeErr)
			}
		}()

		// Declare exchange on tenant vhost (idempotent)
		if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			opentelemetry.HandleSpanError(&span, "Failed to declare exchange on tenant vhost", err)
			pr.Errorf("Error declaring exchange %s on tenant %s vhost: %v", exchange, tenantID, err)

			return fmt.Errorf("failed to declare exchange %s on tenant %s vhost: %w", exchange, tenantID, err)
		}

		msg := amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}

		if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
			opentelemetry.HandleSpanError(&span, "Failed to publish message to tenant vhost", err)
			pr.Errorf("Error publishing to exchange %s on tenant %s: %v", exchange, tenantID, err)

			return fmt.Errorf("failed to publish message to exchange %s on tenant %s: %w", exchange, tenantID, err)
		}

		pr.Debugf("Successfully published message to exchange=%s, routingKey=%s, tenant=%s", exchange, routingKey, tenantID)

		return nil
	}

	// Single-tenant mode: use static RabbitMQ adapter
	if pr.adapter == nil {
		err := fmt.Errorf("no RabbitMQ adapter configured; cannot publish in single-tenant mode")
		opentelemetry.HandleSpanError(&span, "RabbitMQ adapter is nil", err)

		return err
	}

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

	// Only shutdown the adapter in single-tenant mode
	if pr.adapter != nil {
		if err := pr.adapter.Shutdown(ctx); err != nil {
			pr.Errorf("Error shutting down RabbitMQ adapter: %v", err)
			return fmt.Errorf("failed to shutdown RabbitMQ adapter: %w", err)
		}
	}

	pr.Info("PublisherRoutes shutdown complete")

	return nil
}
