// Package rabbitmq provides a wrapper around RabbitMQAdapter for publishing messages to topic exchanges.
package rabbitmq

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v4/commons/rabbitmq"
)

// PublisherRepository provides an interface for Publisher related to rabbitmq.
//
//go:generate mockgen --destination=publisher.mock.go --package=rabbitmq . PublisherRepository
type PublisherRepository interface {
	Publish(ctx context.Context, exchange, routingKey string, body []byte) error
	Shutdown(ctx context.Context) error
}

// PublisherRoutes wraps RabbitMQAdapter to support publishing messages to topic exchanges.
type PublisherRoutes struct {
	adapter rabbitmq.Adapter
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
	pr := &PublisherRoutes{
		adapter:   adapter,
		Logger:    logger,
		Telemetry: *telemetry,
	}

	return pr
}

// Publish sends a message to the specified exchange with the given routing key.
func (pr *PublisherRoutes) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	pr.Log(ctx, libLog.LevelDebug, "publishing message",
		libLog.String("exchange", exchange),
		libLog.String("routing_key", routingKey),
	)

	if err := pr.adapter.ProducerDefault(ctx, exchange, routingKey, body, nil); err != nil {
		pr.Log(ctx, libLog.LevelError, "error publishing message",
			libLog.String("exchange", exchange),
			libLog.String("routing_key", routingKey),
			libLog.Err(err),
		)

		return err
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

	if err := pr.adapter.Shutdown(ctx); err != nil {
		pr.Log(ctx, libLog.LevelError, "error shutting down RabbitMQ adapter", libLog.Err(err))
		return err
	}

	pr.Log(ctx, libLog.LevelInfo, "publisher routes shutdown complete")

	return nil
}
