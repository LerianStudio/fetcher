// Package rabbitmq provides a wrapper around RabbitMQAdapter for publishing messages to topic exchanges.
package rabbitmq

import (
	"context"
	"fmt"

	observability "github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	portPublisher "github.com/LerianStudio/fetcher/pkg/ports/publisher"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	libRabbitmq "github.com/LerianStudio/lib-commons/v5/commons/rabbitmq"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	obsConstants "github.com/LerianStudio/lib-observability/constants"
	libLog "github.com/LerianStudio/lib-observability/log"
	opentelemetry "github.com/LerianStudio/lib-observability/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
)

const (
	attrAppRequestRequestID     = obsConstants.AttrPrefixAppRequest + "request_id"
	attrAppTenantID             = obsConstants.AttrPrefixAppRequest + "tenant_id"
	attrMessagingExchange       = "messaging.exchange"
	attrMessagingRoutingKey     = "messaging.routing_key"
	attrMessagingBodySize       = "messaging.body_size"
	attrPublishRationale        = obsConstants.AttrPrefixAppRequest + "publish_rationale"
	multiTenantPublishRationale = "tenant-manager exposes tenant-scoped AMQP channels but not a confirmable publisher factory; single-tenant publishing uses pkg/rabbitmq's lib-commons-backed adapter"
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
	signer          crypto.Signer
	libLog.
		Logger
	opentelemetry.Telemetry
}

// NewPublisherRoutes creates a new instance of PublisherRoutes using a RabbitMQ connection.
// The signer parameter is optional - pass nil to disable message signing.
func NewPublisherRoutes(conn *libRabbitmq.RabbitMQConnection, logger libLog.Logger, telemetry *opentelemetry.Telemetry, signer crypto.Signer) *PublisherRoutes {
	opts := rabbitmq.DefaultOptions()
	opts.Signer = signer
	adapter := rabbitmq.NewRabbitMQAdapterWithOptions(conn, opts)

	return NewPublisherRoutesWithAdapter(adapter, logger, telemetry)
}

// NewPublisherRoutesWithAdapter creates a new instance of PublisherRoutes using a specific RabbitMQ adapter.
func NewPublisherRoutesWithAdapter(adapter rabbitmq.Adapter, logger libLog.Logger, telemetry *opentelemetry.Telemetry) *PublisherRoutes {
	telemetryValue := opentelemetry.Telemetry{}
	if telemetry != nil {
		telemetryValue = *telemetry
	}

	pr := &PublisherRoutes{
		adapter:   adapter,
		signer:    nil,
		Logger:    logger,
		Telemetry: telemetryValue,
	}

	return pr
}

// NewPublisherRoutesMultiTenant creates a new instance of PublisherRoutes configured for
// multi-tenant mode using tmrabbitmq.Manager for per-tenant vhost isolation.
func NewPublisherRoutesMultiTenant(manager RabbitMQManagerInterface, logger libLog.Logger, telemetry *opentelemetry.Telemetry, signer crypto.Signer) *PublisherRoutes {
	telemetryValue := opentelemetry.Telemetry{}
	if telemetry != nil {
		telemetryValue = *telemetry
	}

	return &PublisherRoutes{
		rabbitMQManager: manager,
		multiTenantMode: true,
		signer:          signer,
		Logger:          logger,
		Telemetry:       telemetryValue,
	}
}

// Publish sends a message to the specified exchange with the given routing key.
// In multi-tenant mode, publishes to the tenant-specific vhost via tmrabbitmq.Manager.
// In single-tenant mode, uses the static RabbitMQ adapter.
func (pr *PublisherRoutes) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	return pr.PublishWithHeaders(ctx, exchange, routingKey, "application/json", body, nil)
}

func (pr *PublisherRoutes) PublishWithHeaders(ctx context.Context, exchange, routingKey, contentType string, body []byte, callerHeaders map[string]any) error {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "adapter.rabbitmq.publish")
	defer span.End()

	span.SetAttributes(
		attribute.String(attrAppRequestRequestID, reqID),
		attribute.String(attrMessagingExchange, exchange),
		attribute.String(attrMessagingRoutingKey, routingKey),
		attribute.Int(attrMessagingBodySize, len(body)),
	)

	pr.Log(ctx, libLog.LevelDebug, "publishing message",
		libLog.String("exchange", exchange),
		libLog.String("routing_key", routingKey),
	)

	// Multi-tenant mode: use tmrabbitmq.Manager for per-tenant vhost
	if pr.multiTenantMode && pr.rabbitMQManager != nil {
		tenantID := tmcore.GetTenantIDContext(ctx)
		if tenantID == "" {
			opentelemetry.HandleSpanError(span, "No tenant ID in context for multi-tenant publish", fmt.Errorf("tenant ID required"))
			return fmt.Errorf("multi-tenant publish requires tenant ID in context")
		}

		span.SetAttributes(attribute.String(attrAppTenantID, tenantID))

		ch, err := pr.rabbitMQManager.GetChannel(ctx, tenantID)
		if err != nil {
			opentelemetry.HandleSpanError(span, "Failed to get tenant channel", err)
			pr.Log(ctx, libLog.LevelError, fmt.Sprintf("Error getting channel for tenant %s", tenantID), libLog.Err(err))

			return fmt.Errorf("failed to get channel for tenant %s: %w", tenantID, err)
		}

		defer func() {
			if closeErr := ch.Close(); closeErr != nil {
				pr.Log(ctx, libLog.LevelError, fmt.Sprintf("Error closing channel for tenant %s", tenantID), libLog.Err(closeErr))
			}
		}()

		// Declare exchange on tenant vhost (idempotent)
		if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			opentelemetry.HandleSpanError(span, "Failed to declare exchange on tenant vhost", err)
			pr.Log(ctx, libLog.LevelError, fmt.Sprintf("Error declaring exchange %s on tenant %s vhost", exchange, tenantID), libLog.Err(err))

			return fmt.Errorf("failed to declare exchange %s on tenant %s vhost: %w", exchange, tenantID, err)
		}

		headers := map[string]any{rabbitmq.HeaderTenantID: tenantID}
		for key, value := range callerHeaders {
			headers[key] = value
		}

		msg := rabbitmq.BuildSecurePublishing(ctx, reqID, exchange, routingKey, body, headers, pr.signer, true)
		if contentType != "" {
			msg.ContentType = contentType
		}

		// This is the only remaining direct AMQP publish path: tenant-manager
		// owns tenant-vhost channel resolution and does not expose a confirmable
		// publisher factory yet. The single-tenant path below delegates to
		// pkg/rabbitmq, where lib-commons backoff/circuit-breaker primitives are
		// enforced. Do not add retries or circuit breakers here; add that to
		// tenant-manager/lib-commons first.
		span.SetAttributes(attribute.String(attrPublishRationale, multiTenantPublishRationale))

		if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
			opentelemetry.HandleSpanError(span, "Failed to publish message to tenant vhost", err)
			pr.Log(ctx, libLog.LevelError, fmt.Sprintf("Error publishing to exchange %s on tenant %s", exchange, tenantID), libLog.Err(err))

			return fmt.Errorf("failed to publish message to exchange %s on tenant %s: %w", exchange, tenantID, err)
		}

		pr.Log(ctx, libLog.LevelDebug, "message published successfully",
			libLog.String("exchange", exchange),
			libLog.String("routing_key", routingKey),
			libLog.String("tenant_id", tenantID),
		)

		return nil
	}

	// Single-tenant mode: use static RabbitMQ adapter
	if pr.adapter == nil {
		err := fmt.Errorf("no RabbitMQ adapter configured; cannot publish in single-tenant mode")
		opentelemetry.HandleSpanError(span, "RabbitMQ adapter is nil", err)

		return err
	}

	// Forward tenant ID from context to AMQP headers for multi-tenant isolation.
	// When no tenant context is present (single-tenant mode), headers remain nil.
	headersValue := callerHeaders

	var headers *map[string]any

	if tenantID := tmcore.GetTenantIDContext(ctx); tenantID != "" {
		if headersValue == nil {
			headersValue = map[string]any{}
		}

		headersValue[rabbitmq.HeaderTenantID] = tenantID
	}

	if headersValue != nil {
		headers = &headersValue
	}

	if err := pr.adapter.ProducerDefault(ctx, exchange, routingKey, body, headers); err != nil {
		opentelemetry.HandleSpanError(span, "Failed to publish message", err)
		pr.Log(ctx, libLog.LevelError, "error publishing message",
			libLog.String("exchange", exchange),
			libLog.String("routing_key", routingKey),
			libLog.Err(err),
		)

		return fmt.Errorf("failed to publish message to exchange %s: %w", exchange, err)
	}

	pr.Log(ctx, libLog.LevelDebug, "message published successfully",
		libLog.String("exchange", exchange),
		libLog.String("routing_key", routingKey),
	)

	return nil
}

// Shutdown gracefully shuts down the RabbitMQ adapter.
func (pr *PublisherRoutes) Shutdown(ctx context.Context) error {
	pr.Log(ctx, libLog.LevelInfo, "shutting down publisher routes")

	// Only shutdown the adapter in single-tenant mode
	if pr.adapter != nil {
		if err := pr.adapter.Shutdown(ctx); err != nil {
			pr.Log(ctx, libLog.LevelError, "error shutting down RabbitMQ adapter", libLog.Err(err))
			return fmt.Errorf("failed to shutdown RabbitMQ adapter: %w", err)
		}
	}

	pr.Log(ctx, libLog.LevelInfo, "publisher routes shutdown complete")

	return nil
}
