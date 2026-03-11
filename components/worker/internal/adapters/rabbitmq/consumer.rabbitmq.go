// Package rabbitmq provides a wrapper around RabbitMQAdapter to support multiple queues with headers.
package rabbitmq

import (
	"context"
	"fmt"
	"sync"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v4/commons/rabbitmq"
)

// ConsumerRepository provides an interface for Consumer related to rabbitmq.
//
//go:generate mockgen --destination=consumer.mock.go --package=rabbitmq . ConsumerRepository
type ConsumerRepository interface {
	Register(queueName string, handler QueueHandlerFunc)
	RunConsumers(ctx context.Context, wg *sync.WaitGroup) error
}

// QueueHandlerFunc is a function that processes a specific queue.
type QueueHandlerFunc func(ctx context.Context, body []byte, headers map[string]any) error

// ConsumerRoutes wraps RabbitMQAdapter to support multiple queues with headers.
type ConsumerRoutes struct {
	adapter    rabbitmq.Adapter
	routes     map[string]QueueHandlerFunc
	numWorkers int
	libLog.
		Logger
	opentelemetry.Telemetry
	shutdownWg sync.WaitGroup
}

// NewConsumerRoutes creates a new instance of ConsumerRoutes using a RabbitMQ connection.
// The signer parameter is optional - pass nil to disable signature verification.
func NewConsumerRoutes(conn *libRabbitmq.RabbitMQConnection, numWorkers int, logger libLog.Logger, telemetry *opentelemetry.Telemetry, signer crypto.Signer) *ConsumerRoutes {
	opts := rabbitmq.DefaultOptions()
	opts.Signer = signer
	adapter := rabbitmq.NewRabbitMQAdapterWithOptions(conn, opts)

	return NewConsumerRoutesWithAdapter(adapter, numWorkers, logger, telemetry)
}

// NewConsumerRoutesWithAdapter creates a new instance of ConsumerRoutes using a specific RabbitMQ adapter.
func NewConsumerRoutesWithAdapter(adapter rabbitmq.Adapter, numWorkers int, logger libLog.Logger, telemetry *opentelemetry.Telemetry) *ConsumerRoutes {
	if numWorkers <= 0 {
		numWorkers = 5
	}

	cr := &ConsumerRoutes{
		adapter:    adapter,
		routes:     make(map[string]QueueHandlerFunc),
		numWorkers: numWorkers,
		Logger:     logger,
		Telemetry:  *telemetry,
	}

	return cr
}

// Register adds a new queue handler.
func (cr *ConsumerRoutes) Register(queueName string, handler QueueHandlerFunc) {
	cr.routes[queueName] = handler
}

// RunConsumers starts consumers for all registered queues using RabbitMQAdapter.
func (cr *ConsumerRoutes) RunConsumers(ctx context.Context, wg *sync.WaitGroup) error {
	for queueName, handler := range cr.routes {
		cr.Log(context.Background(), libLog.LevelInfo, fmt.Sprint("Starting consumer for queue "+queueName))

		queueName := queueName
		handler := handler

		wg.Add(1)
		cr.shutdownWg.Add(1)

		go func() {
			defer wg.Done()
			defer cr.shutdownWg.Done()

			err := cr.adapter.ConsumerLoop(ctx, queueName, cr.numWorkers, handler)
			if err != nil && ctx.Err() == nil {
				cr.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Consumer loop for queue %s exited with error: %v", queueName, err))
			}
		}()
	}

	return nil
}

// Shutdown gracefully shuts down all consumers and the RabbitMQ adapter.
func (cr *ConsumerRoutes) Shutdown(ctx context.Context) error {
	cr.Log(context.Background(), libLog.LevelInfo, "Shutting down ConsumerRoutes...")

	cr.shutdownWg.Wait()

	// Shutdown the RabbitMQ adapter
	if err := cr.adapter.Shutdown(ctx); err != nil {
		cr.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error shutting down RabbitMQ adapter: %v", err))
		return err
	}

	cr.Log(context.Background(), libLog.LevelInfo, "ConsumerRoutes shutdown complete")

	return nil
}
