package topology

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SetupRabbitMQTopology creates the required exchanges and queues for Fetcher.
func SetupRabbitMQTopology(ctx context.Context, amqpURL string) error {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	// Declare exchanges
	exchanges := []struct {
		name string
		kind string
	}{
		{"fetcher.extract-external-data.exchange", "direct"},
		{"fetcher.job.events", "topic"},
		{"fetcher.dlx", "direct"},
	}

	for _, ex := range exchanges {
		err = ch.ExchangeDeclare(
			ex.name,
			ex.kind,
			true,  // durable
			false, // auto-delete
			false, // internal
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", ex.name, err)
		}
	}

	// Declare queues
	queues := []struct {
		name string
		args amqp.Table
	}{
		{
			"extract-external-data-queue",
			amqp.Table{
				"x-dead-letter-exchange":    "fetcher.dlx",
				"x-dead-letter-routing-key": "fetcher.dlq.key",
			},
		},
		{
			"fetcher.dlq",
			amqp.Table{
				"x-message-ttl": int32(604800000), // 7 days
				"x-max-length":  int32(10000),
			},
		},
		{
			// Test queue for capturing job completion/failure events
			"test.job.events",
			nil,
		},
	}

	for _, q := range queues {
		_, err = ch.QueueDeclare(
			q.name,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false, // no-wait
			q.args,
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", q.name, err)
		}
	}

	// Bind queues
	bindings := []struct {
		queue      string
		routingKey string
		exchange   string
	}{
		{"extract-external-data-queue", "fetcher.job.key", "fetcher.extract-external-data.exchange"},
		{"fetcher.dlq", "fetcher.dlq.key", "fetcher.dlx"},
		// Test queue bindings for job events (topic exchange with wildcards)
		{"test.job.events", "job.completed.*", "fetcher.job.events"},
		{"test.job.events", "job.failed.*", "fetcher.job.events"},
	}

	for _, b := range bindings {
		err = ch.QueueBind(
			b.queue,
			b.routingKey,
			b.exchange,
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", b.queue, err)
		}
	}

	return nil
}

// PurgeTestQueue purges the test.job.events queue to remove stale events.
// This is useful when reusing infrastructure between test runs.
func PurgeTestQueue(ctx context.Context, amqpURL string) (int, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return 0, fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	count, err := ch.QueuePurge("test.job.events", false)
	if err != nil {
		return 0, fmt.Errorf("failed to purge test.job.events queue: %w", err)
	}

	return count, nil
}
