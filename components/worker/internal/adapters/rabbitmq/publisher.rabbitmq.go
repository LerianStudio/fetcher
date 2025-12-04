// Package rabbitmq provides a wrapper around RabbitMQAdapter for publishing messages to topic exchanges.
package rabbitmq

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v2/commons/rabbitmq"
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
	adapter *rabbitmq.RabbitMQAdapter
	log.Logger
	opentelemetry.Telemetry
}

// NewPublisherRoutes creates a new instance of PublisherRoutes using RabbitMQAdapter.
func NewPublisherRoutes(conn *libRabbitmq.RabbitMQConnection, logger log.Logger, telemetry *opentelemetry.Telemetry) *PublisherRoutes {
	adapter := rabbitmq.NewRabbitMQAdapter(conn)

	pr := &PublisherRoutes{
		adapter:   adapter,
		Logger:    logger,
		Telemetry: *telemetry,
	}

	return pr
}

// Publish sends a message to the specified exchange with the given routing key.
func (pr *PublisherRoutes) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	pr.Debugf("Publishing message to exchange=%s, routingKey=%s", exchange, routingKey)

	if err := pr.adapter.ProducerDefault(ctx, exchange, routingKey, body); err != nil {
		pr.Errorf("Error publishing message to exchange %s with routing key %s: %v", exchange, routingKey, err)
		return err
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
		return err
	}

	pr.Info("PublisherRoutes shutdown complete")

	return nil
}
