// Package rabbitmq provides a wrapper around RabbitMQAdapter for publishing messages to topic exchanges.
package rabbitmq

import (
	"context"
	"fmt"
	"sync"

	observability "github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	portPublisher "github.com/LerianStudio/fetcher/v2/pkg/ports/publisher"
	"github.com/LerianStudio/fetcher/v2/pkg/rabbitmq"
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
	multiTenantPublishRationale = "tenant-manager exposes tenant-scoped AMQP channels; this adapter enables publisher confirms before terminal event success"
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
	Confirm(noWait bool) error
	NotifyPublish(receiver chan amqp.Confirmation) chan amqp.Confirmation
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
}

type confirmablePublisher interface {
	Publish(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
}

// PublisherRoutes wraps RabbitMQAdapter to support publishing messages to topic exchanges.
// In multi-tenant mode, it uses tmrabbitmq.Manager for per-tenant vhost isolation.
type PublisherRoutes struct {
	adapter         rabbitmq.Adapter // Used in single-tenant mode
	conn            *libRabbitmq.RabbitMQConnection
	rabbitMQManager RabbitMQManagerInterface // Used in multi-tenant mode (nil in single-tenant)
	multiTenantMode bool
	signer          crypto.Signer
	publishMu       sync.Mutex
	publishers      map[RabbitMQChannel]confirmablePublisher
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

	pr := NewPublisherRoutesWithAdapter(adapter, logger, telemetry)
	pr.conn = conn
	pr.signer = signer

	return pr
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

// PublishStreamingTarget publishes for lib-streaming without routing through
// Fetcher's RabbitMQAdapter circuit breaker. Lib-streaming owns target circuit
// breaker semantics; this method only builds the secure envelope and waits for
// broker confirmation.
func (pr *PublisherRoutes) PublishStreamingTarget(ctx context.Context, exchange, routingKey, contentType string, body []byte, callerHeaders map[string]any) error {
	if pr.multiTenantMode {
		return pr.PublishWithHeaders(ctx, exchange, routingKey, contentType, body, callerHeaders)
	}

	if pr.conn == nil {
		return pr.PublishWithHeaders(ctx, exchange, routingKey, contentType, body, callerHeaders)
	}

	ctxLogger, ctxTracer, reqID, metricFactory := observability.NewTrackingFromContext(ctx)
	_ = ctxLogger
	_ = ctxTracer
	_ = metricFactory

	if err := pr.conn.EnsureChannel(); err != nil {
		return fmt.Errorf("rabbitmq ensure streaming publish channel: %w", err)
	}

	if pr.conn.Channel == nil {
		return fmt.Errorf("rabbitmq streaming publish channel is nil")
	}

	var headers map[string]any
	if len(callerHeaders) > 0 {
		headers = make(map[string]any, len(callerHeaders)+1)
		for k, v := range callerHeaders {
			headers[k] = v
		}
	}

	if tenantID := tmcore.GetTenantIDContext(ctx); tenantID != "" {
		if headers == nil {
			headers = map[string]any{}
		}

		headers[rabbitmq.HeaderTenantID] = tenantID
	}

	msg := rabbitmq.BuildSecurePublishing(ctx, reqID, exchange, routingKey, body, headers, pr.signer, true)
	if contentType != "" {
		msg.ContentType = contentType
	}

	if err := pr.publishWithConfirm(ctx, pr.conn.Channel, exchange, routingKey, msg); err != nil {
		return fmt.Errorf("rabbitmq streaming publish: %w", err)
	}

	return nil
}

// Ping reports whether the broker is reachable for streaming publishes. It is
// the cheapest meaningful probe for the current publish mode and backs
// lib-streaming's Adapter.Healthy via the streaming RabbitMQ target.
//
//   - Multi-tenant mode resolves a fresh channel per tenant at publish time and
//     has no ambient tenant to probe, so the manager owns per-tenant lifecycle
//     and Ping reports healthy here.
//   - Single-tenant mode ensures the shared channel against the live connection
//     (a real broker round-trip on reconnect).
//   - The adapter-only path (no connection) falls back to the adapter's own
//     health signal.
func (pr *PublisherRoutes) Ping(ctx context.Context) error {
	if pr.multiTenantMode {
		if pr.rabbitMQManager == nil {
			return fmt.Errorf("multi-tenant RabbitMQ manager is not configured")
		}

		return nil
	}

	if pr.conn != nil {
		if err := pr.conn.EnsureChannelContext(ctx); err != nil {
			return fmt.Errorf("rabbitmq ensure streaming health channel: %w", err)
		}

		return nil
	}

	if pr.adapter == nil {
		return fmt.Errorf("no RabbitMQ adapter configured; cannot probe broker health")
	}

	if !pr.adapter.IsHealthy() {
		return fmt.Errorf("rabbitmq adapter reports unhealthy broker connection")
	}

	return nil
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
	if pr.multiTenantMode {
		if pr.rabbitMQManager == nil {
			err := fmt.Errorf("multi-tenant RabbitMQ manager is not configured")
			opentelemetry.HandleSpanError(span, "RabbitMQ manager is nil", err)

			return err
		}

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

		if ch == nil {
			err := fmt.Errorf("tenant RabbitMQ manager returned nil channel for tenant %s", tenantID)
			opentelemetry.HandleSpanError(span, "Tenant channel is nil", err)

			return err
		}

		// Declare exchange on tenant vhost (idempotent)
		if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			if closeErr := ch.Close(); closeErr != nil {
				err = fmt.Errorf("%w; close tenant channel: %v", err, closeErr)
			}

			opentelemetry.HandleSpanError(span, "Failed to declare exchange on tenant vhost", err)
			pr.Log(ctx, libLog.LevelError, fmt.Sprintf("Error declaring exchange %s on tenant %s vhost", exchange, tenantID), libLog.Err(err))

			return fmt.Errorf("failed to declare exchange %s on tenant %s vhost: %w", exchange, tenantID, err)
		}

		headers := make(map[string]any, len(callerHeaders)+1)
		for key, value := range callerHeaders {
			headers[key] = value
		}

		headers[rabbitmq.HeaderTenantID] = tenantID

		msg := rabbitmq.BuildSecurePublishing(ctx, reqID, exchange, routingKey, body, headers, pr.signer, true)
		if contentType != "" {
			msg.ContentType = contentType
		}

		// This direct AMQP path uses tenant-manager for tenant-vhost channel
		// resolution and publisher confirms here so mandatory terminal events only
		// succeed after broker acceptance.
		span.SetAttributes(attribute.String(attrPublishRationale, multiTenantPublishRationale))

		if err := pr.publishWithTenantConfirmAndClose(ctx, ch, exchange, routingKey, msg); err != nil {
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
	var headersValue map[string]any
	if len(callerHeaders) > 0 {
		headersValue = make(map[string]any, len(callerHeaders)+1)
		for k, v := range callerHeaders {
			headersValue[k] = v
		}
	}

	if tenantID := tmcore.GetTenantIDContext(ctx); tenantID != "" {
		if headersValue == nil {
			headersValue = map[string]any{}
		}

		headersValue[rabbitmq.HeaderTenantID] = tenantID
	}

	var headers *map[string]any
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

func (pr *PublisherRoutes) publishWithConfirm(ctx context.Context, ch RabbitMQChannel, exchange, routingKey string, msg amqp.Publishing) error {
	publisher, err := pr.confirmablePublisherForChannel(ch)
	if err != nil {
		return err
	}

	if err := publisher.Publish(ctx, exchange, routingKey, true, false, msg); err != nil {
		return fmt.Errorf("rabbitmq publish confirmed message: %w", err)
	}

	return nil
}

func (pr *PublisherRoutes) publishWithTenantConfirmAndClose(ctx context.Context, ch RabbitMQChannel, exchange, routingKey string, msg amqp.Publishing) error {
	publisher, err := libRabbitmq.NewConfirmablePublisherFromChannel(ch,
		libRabbitmq.WithLogger(pr.Logger),
		libRabbitmq.WithConfirmTimeout(libRabbitmq.DefaultConfirmTimeout),
	)
	if err != nil {
		if closeErr := ch.Close(); closeErr != nil {
			return fmt.Errorf("create rabbitmq confirmable publisher: %w; close tenant channel: %v", err, closeErr)
		}

		return fmt.Errorf("create rabbitmq confirmable publisher: %w", err)
	}
	defer func() { _ = publisher.Close() }()

	if err := publisher.Publish(ctx, exchange, routingKey, true, false, msg); err != nil {
		return fmt.Errorf("rabbitmq publish confirmed message: %w", err)
	}

	return nil
}

func (pr *PublisherRoutes) confirmablePublisherForChannel(ch RabbitMQChannel) (confirmablePublisher, error) {
	pr.publishMu.Lock()
	defer pr.publishMu.Unlock()

	if pr.publishers == nil {
		pr.publishers = make(map[RabbitMQChannel]confirmablePublisher)
	}

	if publisher := pr.publishers[ch]; publisher != nil {
		return publisher, nil
	}

	publisher, err := libRabbitmq.NewConfirmablePublisherFromChannel(ch,
		libRabbitmq.WithLogger(pr.Logger),
		libRabbitmq.WithConfirmTimeout(libRabbitmq.DefaultConfirmTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("create rabbitmq confirmable publisher: %w", err)
	}

	pr.publishers[ch] = publisher

	return publisher, nil
}

func (pr *PublisherRoutes) closeConfirmablePublishers() {
	pr.publishMu.Lock()
	defer pr.publishMu.Unlock()

	for _, publisher := range pr.publishers {
		_ = publisher.Close()
	}

	pr.publishers = nil
}

// Shutdown gracefully shuts down the RabbitMQ adapter.
func (pr *PublisherRoutes) Shutdown(ctx context.Context) error {
	pr.Log(ctx, libLog.LevelInfo, "shutting down publisher routes")
	pr.closeConfirmablePublishers()

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
