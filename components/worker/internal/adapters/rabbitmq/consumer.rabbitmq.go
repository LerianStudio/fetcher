// Package rabbitmq provides a wrapper around RabbitMQAdapter to support multiple queues with headers.
package rabbitmq

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	"github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v5/commons/rabbitmq"
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
// The signer parameter is required in non-development environments.
// In dev/local/test, nil signer disables signature verification and message signing.
// The envName parameter should come from the bootstrap Config.EnvName field.
func NewConsumerRoutes(conn *libRabbitmq.RabbitMQConnection, numWorkers int, logger libLog.Logger, telemetry *opentelemetry.Telemetry, signer crypto.Signer, envName string) (*ConsumerRoutes, error) {
	opts := rabbitmq.DefaultOptions()
	opts.Signer = signer

	envName = strings.TrimSpace(envName)

	if signer == nil {
		if isNonDevelopmentEnvironment(envName) {
			return nil, fmt.Errorf("rabbitmq signature verification requires configured signer in env %q", envName)
		}

		logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("RabbitMQ signer not configured in env %q; disabling signature verification", envName))

		opts.EnableSignatureVerification = false
		opts.EnableMessageSigning = false
	}

	adapter := rabbitmq.NewRabbitMQAdapterWithOptions(conn, opts)

	return NewConsumerRoutesWithAdapter(adapter, numWorkers, logger, telemetry), nil
}

// NewConsumerRoutesWithAdapter creates a new instance of ConsumerRoutes using a specific RabbitMQ adapter.
func NewConsumerRoutesWithAdapter(adapter rabbitmq.Adapter, numWorkers int, logger libLog.Logger, telemetry *opentelemetry.Telemetry) *ConsumerRoutes {
	if numWorkers <= 0 {
		numWorkers = 5
	}

	telemetryValue := opentelemetry.Telemetry{}
	if telemetry != nil {
		telemetryValue = *telemetry
	}

	cr := &ConsumerRoutes{
		adapter:    adapter,
		routes:     make(map[string]QueueHandlerFunc),
		numWorkers: numWorkers,
		Logger:     logger,
		Telemetry:  telemetryValue,
	}

	return cr
}

// Adapter exposes the underlying adapter so /readyz can inspect the
// circuit-breaker state and liveness without reaching into unexported
// fields.
func (c *ConsumerRoutes) Adapter() rabbitmq.Adapter {
	if c == nil {
		return nil
	}

	return c.adapter
}

// isNonDevelopmentEnvironment returns true unless envName is an explicitly
// allowlisted development environment. Empty/unknown values default to true
// (fail-closed: unknown environment is treated as production).
// Caller is expected to TrimSpace before calling.
func isNonDevelopmentEnvironment(envName string) bool {
	switch strings.ToLower(envName) {
	case "dev", "development", "local", "test", "testing":
		return false
	default:
		return true
	}
}

// Register adds a new queue handler.
func (cr *ConsumerRoutes) Register(queueName string, handler QueueHandlerFunc) {
	cr.routes[queueName] = handler
}

// RunConsumers starts consumers for all registered queues using RabbitMQAdapter.
func (cr *ConsumerRoutes) RunConsumers(ctx context.Context, wg *sync.WaitGroup) error {
	for queueName, handler := range cr.routes {
		cr.Log(ctx, libLog.LevelInfo, "starting consumer for queue",
			libLog.String("queue", queueName),
		)

		queueName := queueName
		handler := handler

		wg.Add(1)
		cr.shutdownWg.Add(1)

		go func() {
			defer wg.Done()
			defer cr.shutdownWg.Done()

			err := cr.adapter.ConsumerLoop(ctx, queueName, cr.numWorkers, handler)
			if err != nil && ctx.Err() == nil {
				cr.Log(ctx, libLog.LevelError, "consumer loop exited with error",
					libLog.String("queue", queueName),
					libLog.Err(err),
				)
			}
		}()
	}

	return nil
}

// Shutdown gracefully shuts down all consumers and the RabbitMQ adapter.
func (cr *ConsumerRoutes) Shutdown(ctx context.Context) error {
	cr.Log(ctx, libLog.LevelInfo, "shutting down consumer routes")

	cr.shutdownWg.Wait()

	// Shutdown the RabbitMQ adapter
	if err := cr.adapter.Shutdown(ctx); err != nil {
		cr.Log(ctx, libLog.LevelError, "error shutting down RabbitMQ adapter", libLog.Err(err))
		return err
	}

	cr.Log(ctx, libLog.LevelInfo, "consumer routes shutdown complete")

	return nil
}
