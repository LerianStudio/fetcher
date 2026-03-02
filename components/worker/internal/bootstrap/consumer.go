package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/components/worker/internal/services"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
	"github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	tmconsumer "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/consumer"
	tmcore "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/core"
	tmmongo "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/mongo"
	tmrabbitmq "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/LerianStudio/lib-commons/v3/commons"
	"go.opentelemetry.io/otel/attribute"
)

// MultiQueueConsumer represents a multi-queue consumer.
type MultiQueueConsumer struct {
	consumerRoutes *rabbitmq.ConsumerRoutes
	UseCase        *services.UseCase
	mongoManager   *tmmongo.Manager
}

// NewMultiQueueConsumer create a new instance of MultiQueueConsumer.
func NewMultiQueueConsumer(routes *rabbitmq.ConsumerRoutes, useCase *services.UseCase, queueName string, mongoManager *tmmongo.Manager) *MultiQueueConsumer {
	consumer := &MultiQueueConsumer{
		consumerRoutes: routes,
		UseCase:        useCase,
		mongoManager:   mongoManager,
	}

	// Registry handlers for each queue
	routes.Register(queueName, consumer.handlerGenerateReport)

	return consumer
}

// Run starts consumers for all registered queues.
func (mq *MultiQueueConsumer) Run(l *commons.Launcher) error {
	// Create initial context with logger from ConsumerRoutes
	requestID := commons.GenerateUUIDv7().String()
	baseCtx := commons.ContextWithLogger(
		commons.ContextWithHeaderID(context.Background(), requestID),
		mq.consumerRoutes.Logger,
	)

	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	wg := &sync.WaitGroup{}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		mq.consumerRoutes.Info("Received shutdown signal, starting graceful shutdown...")
		cancel()
	}()

	if err := mq.consumerRoutes.RunConsumers(ctx, wg); err != nil {
		return fmt.Errorf("failed to run consumers: %w", err)
	}

	wg.Wait()

	// Shutdown ConsumerRoutes gracefully after all workers are done
	shutdownCtx, shutdownCancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer shutdownCancel()

	if err := mq.consumerRoutes.Shutdown(shutdownCtx); err != nil {
		mq.consumerRoutes.Errorf("Error during ConsumerRoutes shutdown: %v", err)
		return fmt.Errorf("failed to shutdown consumer routes: %w", err)
	}

	return nil
}

// handlerGenerateReport processes messages from the generate report queue.
func (mq *MultiQueueConsumer) handlerGenerateReport(ctx context.Context, body []byte, headers map[string]any) error {
	// Extract tenant ID from AMQP headers and inject into context for multi-tenant isolation.
	// When no X-Tenant-ID header is present (single-tenant mode), context remains unchanged.
	ctx = extractTenantIDFromHeaders(ctx, headers)

	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	spanCtx, span := tracer.Start(ctx, "consumer.handler_generate_report")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	// Resolve tenant-specific MongoDB database and inject into context.
	// In single-tenant mode (mongoManager is nil), this is a no-op.
	spanCtx, err := resolveTenantMongo(spanCtx, mq.mongoManager)
	if err != nil {
		opentelemetry.HandleSpanError(&span, "Failed to resolve tenant MongoDB database.", err)
		logger.Errorf("Failed to resolve tenant MongoDB: %v", err)

		return fmt.Errorf("resolve tenant mongo: %w", err)
	}

	logger.Info("Processing message from generate report queue")

	err = mq.UseCase.ExtractExternalData(spanCtx, body, headers)
	if err != nil {
		opentelemetry.HandleSpanError(&span, "Error generating report.", err)

		logger.Errorf("Error generating report: %v", err)

		return fmt.Errorf("failed to generate report: %w", err)
	}

	return nil
}

// extractTenantIDFromHeaders extracts X-Tenant-ID from AMQP message headers
// and injects it into the context. If the header is absent, not a string,
// or empty, the context is returned unchanged (backward-compatible with single-tenant mode).
func extractTenantIDFromHeaders(ctx context.Context, headers map[string]any) context.Context {
	if headers == nil {
		return ctx
	}

	tenantID, ok := headers["X-Tenant-ID"].(string)
	if !ok || tenantID == "" {
		return ctx
	}

	return tmcore.SetTenantIDInContext(ctx, tenantID)
}

// resolveTenantMongo resolves a tenant-specific MongoDB database using the mongo manager
// and injects it into the context via tmcore.ContextWithTenantMongo. This enables
// downstream repositories to retrieve the tenant DB via tmcore.GetMongoFromContext(ctx).
//
// Per multi-tenant.md worker pattern:
//   - When mongoManager is nil (single-tenant mode), returns context unchanged (no-op)
//   - When tenant ID is absent from context, returns context unchanged (no-op)
//   - When both are present, resolves tenant DB and injects into context
func resolveTenantMongo(ctx context.Context, mongoManager *tmmongo.Manager) (context.Context, error) {
	if mongoManager == nil {
		return ctx, nil
	}

	tenantID := tmcore.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return ctx, nil
	}

	tenantDB, err := mongoManager.GetDatabaseForTenant(ctx, tenantID)
	if err != nil {
		return ctx, fmt.Errorf("get tenant mongo database for tenant %s: %w", tenantID, err)
	}

	return tmcore.ContextWithTenantMongo(ctx, tenantDB), nil
}

// MultiTenantConsumerAdapter wraps tmconsumer.MultiTenantConsumer to implement the Consumer interface.
// It adapts the lib-commons multi-tenant consumer for use with the fetcher worker's UseCase.
//
// Per multi-tenant.md standards:
//   - Uses tmconsumer.MultiTenantConsumer for lazy consumer initialization
//   - Consumers are spawned on-demand per tenant vhost (not at startup)
//   - Tenant ID is set in context via tmcore.SetTenantIDInContext by the lib-commons consumer
type MultiTenantConsumerAdapter struct {
	consumer     *tmconsumer.MultiTenantConsumer
	useCase      *services.UseCase
	queueName    string
	mongoManager *tmmongo.Manager
	rmqManager   *tmrabbitmq.Manager
	logger       log.Logger
}

// NewMultiTenantConsumerAdapter creates a new adapter that wraps the lib-commons MultiTenantConsumer.
func NewMultiTenantConsumerAdapter(
	consumer *tmconsumer.MultiTenantConsumer,
	useCase *services.UseCase,
	queueName string,
	mongoManager *tmmongo.Manager,
	rmqManager *tmrabbitmq.Manager,
	logger log.Logger,
) *MultiTenantConsumerAdapter {
	adapter := &MultiTenantConsumerAdapter{
		consumer:     consumer,
		useCase:      useCase,
		queueName:    queueName,
		mongoManager: mongoManager,
		rmqManager:   rmqManager,
		logger:       logger,
	}

	// Register the handler for the queue.
	// The handler signature is tmconsumer.HandlerFunc: func(ctx context.Context, delivery amqp.Delivery) error
	// The tenant ID is already set in context by the lib-commons consumer.
	consumer.Register(queueName, adapter.handleDelivery)

	return adapter
}

// Run starts the multi-tenant consumer in lazy mode.
// It discovers tenants without starting consumers (non-blocking) and starts background polling.
func (m *MultiTenantConsumerAdapter) Run(l *commons.Launcher) error {
	requestID := commons.GenerateUUIDv7().String()
	baseCtx := commons.ContextWithLogger(
		commons.ContextWithHeaderID(context.Background(), requestID),
		m.logger,
	)

	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		m.logger.Info("Received shutdown signal, starting graceful shutdown...")
		cancel()
	}()

	// Start the multi-tenant consumer (lazy mode - non-blocking)
	if err := m.consumer.Run(ctx); err != nil {
		m.logger.Errorf("Failed to start multi-tenant consumer: %v", err)
		return fmt.Errorf("start multi-tenant consumer: %w", err)
	}

	m.logger.Info("Multi-tenant consumer started in lazy mode")

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	// Graceful shutdown
	m.logger.Info("Shutting down multi-tenant consumer...")

	if err := m.consumer.Close(); err != nil {
		m.logger.Errorf("Error closing multi-tenant consumer: %v", err)
		return fmt.Errorf("close multi-tenant consumer: %w", err)
	}

	// Close RabbitMQ manager connections
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := m.rmqManager.Close(shutdownCtx); err != nil {
		m.logger.Errorf("Error closing RabbitMQ manager: %v", err)
	}

	m.logger.Info("Multi-tenant consumer shutdown complete")

	return nil
}

// handleDelivery is the handler function for the multi-tenant consumer.
// It adapts the amqp.Delivery to the format expected by the UseCase.
// The tenant ID is already set in context by the lib-commons consumer via tmcore.SetTenantIDInContext.
func (m *MultiTenantConsumerAdapter) handleDelivery(ctx context.Context, delivery amqp.Delivery) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	spanCtx, span := tracer.Start(ctx, "consumer.handler_generate_report")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	// Resolve tenant-specific MongoDB database and inject into context.
	// The tenant ID is already in context from the lib-commons consumer.
	spanCtx, err := resolveTenantMongo(spanCtx, m.mongoManager)
	if err != nil {
		opentelemetry.HandleSpanError(&span, "Failed to resolve tenant MongoDB database.", err)
		logger.Errorf("Failed to resolve tenant MongoDB: %v", err)

		return fmt.Errorf("resolve tenant mongo: %w", err)
	}

	logger.Info("Processing message from generate report queue (multi-tenant)")

	// Convert AMQP headers to map[string]any
	headers := make(map[string]any)

	if delivery.Headers != nil {
		for k, v := range delivery.Headers {
			headers[k] = v
		}
	}

	// Call the UseCase with the message body and headers
	err = m.useCase.ExtractExternalData(spanCtx, delivery.Body, headers)
	if err != nil {
		opentelemetry.HandleSpanError(&span, "Error generating report.", err)
		logger.Errorf("Error generating report: %v", err)

		return fmt.Errorf("failed to generate report: %w", err)
	}

	return nil
}
