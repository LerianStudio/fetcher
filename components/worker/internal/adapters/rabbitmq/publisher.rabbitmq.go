// Package rabbitmq provides a wrapper around RabbitMQAdapter for publishing messages to topic exchanges.
package rabbitmq

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	portPublisher "github.com/LerianStudio/fetcher/pkg/ports/publisher"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v2/commons/rabbitmq"
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
	pr := &PublisherRoutes{
		adapter:   adapter,
		Logger:    logger,
		Telemetry: *telemetry,
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

	if err := pr.adapter.ProducerDefault(ctx, exchange, routingKey, body, nil); err != nil {
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
