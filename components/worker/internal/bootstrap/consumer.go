package bootstrap

import (
	"context"
	"errors"
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

	"github.com/LerianStudio/lib-commons/v3/commons"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
)

// MultiTenantConsumerInterface abstracts the tmconsumer.MultiTenantConsumer for testing.
type MultiTenantConsumerInterface interface {
	Register(queueName string, handler tmconsumer.HandlerFunc)
	Run(ctx context.Context) error
	Close() error
}

// MultiQueueConsumer represents a multi-queue consumer.
// It supports two modes:
//   - Single-tenant: Uses consumerRoutes with static RabbitMQ connection
//   - Multi-tenant: Uses mtConsumer (tmconsumer.MultiTenantConsumer) with per-tenant vhost isolation
type MultiQueueConsumer struct {
	consumerRoutes *rabbitmq.ConsumerRoutes
	mtConsumer     MultiTenantConsumerInterface // Multi-tenant consumer (nil in single-tenant mode)
	UseCase        *services.UseCase
	logger         log.Logger
	queueName      string           // Stored for multi-tenant handler registration
	mongoManager   *tmmongo.Manager // For per-tenant MongoDB resolution (nil in single-tenant mode)
}

// NewMultiQueueConsumer creates a new instance of MultiQueueConsumer for single-tenant mode.
func NewMultiQueueConsumer(routes *rabbitmq.ConsumerRoutes, useCase *services.UseCase, queueName string, logger log.Logger, mongoManager *tmmongo.Manager) *MultiQueueConsumer {
	consumer := &MultiQueueConsumer{
		consumerRoutes: routes,
		mtConsumer:     nil, // Single-tenant mode
		UseCase:        useCase,
		logger:         logger,
		queueName:      queueName,
		mongoManager:   mongoManager,
	}

	// Registry handlers for each queue
	if routes != nil {
		routes.Register(queueName, consumer.handlerGenerateReport)
	}

	return consumer
}

// NewMultiQueueConsumerMultiTenant creates a new instance of MultiQueueConsumer for multi-tenant mode.
// It uses tmconsumer.MultiTenantConsumer for per-tenant vhost isolation with lazy initialization.
// The handler is registered with the MultiTenantConsumer to process messages from per-tenant queues.
func NewMultiQueueConsumerMultiTenant(
	mtConsumer MultiTenantConsumerInterface,
	useCase *services.UseCase,
	queueName string,
	logger log.Logger,
	mongoManager *tmmongo.Manager,
) *MultiQueueConsumer {
	consumer := &MultiQueueConsumer{
		consumerRoutes: nil, // Multi-tenant mode uses mtConsumer
		mtConsumer:     mtConsumer,
		UseCase:        useCase,
		logger:         logger,
		queueName:      queueName,
		mongoManager:   mongoManager,
	}

	// Register handler with MultiTenantConsumer
	// The handler signature is tmconsumer.HandlerFunc: func(ctx, amqp.Delivery) error
	if mtConsumer != nil {
		mtConsumer.Register(queueName, consumer.handlerGenerateReportDelivery)
		logger.Infof("MultiTenantConsumer: handler registered for queue %s", queueName)
	}

	return consumer
}

// Run starts consumers for all registered queues.
// In multi-tenant mode, uses mtConsumer.Run() which discovers tenants from Redis
// and spawns consumer goroutines per-tenant vhost.
// In single-tenant mode, uses consumerRoutes.RunConsumers() with static connection.
func (mq *MultiQueueConsumer) Run(l *commons.Launcher) error {
	// Create initial context with logger from ConsumerRoutes or standalone logger
	requestID := commons.GenerateUUIDv7().String()

	var baseLogger log.Logger
	if mq.consumerRoutes != nil {
		baseLogger = mq.consumerRoutes.Logger
	} else {
		baseLogger = mq.logger
	}

	baseCtx := commons.ContextWithLogger(
		commons.ContextWithHeaderID(context.Background(), requestID),
		baseLogger,
	)

	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs

		if mq.logger != nil {
			mq.logger.Info("Received shutdown signal, starting graceful shutdown...")
		}

		cancel()
	}()

	// Multi-tenant mode: use tmconsumer.MultiTenantConsumer
	if mq.mtConsumer != nil {
		mq.logger.Info("MultiQueueConsumer: starting multi-tenant consumer with per-tenant vhost isolation")

		// Run starts tenant discovery and spawns consumer goroutines
		if err := mq.mtConsumer.Run(ctx); err != nil {
			return fmt.Errorf("failed to run multi-tenant consumer: %w", err)
		}

		// Block until context is canceled (shutdown signal)
		<-ctx.Done()

		mq.logger.Info("MultiQueueConsumer: shutting down multi-tenant consumer")

		// Close gracefully stops all tenant consumers
		if err := mq.mtConsumer.Close(); err != nil {
			mq.logger.Errorf("MultiQueueConsumer: error closing multi-tenant consumer: %v", err)
		}

		return nil
	}

	// Single-tenant mode: use ConsumerRoutes with static connection
	wg := &sync.WaitGroup{}

	if err := mq.consumerRoutes.RunConsumers(ctx, wg); err != nil {
		return fmt.Errorf("failed to run consumers: %w", err)
	}

	wg.Wait()

	// Shutdown ConsumerRoutes gracefully after all workers are done
	shutdownCtx, shutdownCancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer shutdownCancel()

	if err := mq.consumerRoutes.Shutdown(shutdownCtx); err != nil {
		if mq.logger != nil {
			mq.logger.Errorf("Error during ConsumerRoutes shutdown: %v", err)
		}

		return fmt.Errorf("failed to shutdown consumer routes: %w", err)
	}

	return nil
}

// handlerGenerateReportDelivery is the tmconsumer.HandlerFunc adapter for multi-tenant mode.
// It resolves per-tenant MongoDB if mongoManager is available, then delegates to handlerGenerateReport.
func (mq *MultiQueueConsumer) handlerGenerateReportDelivery(ctx context.Context, delivery amqp.Delivery) error {
	// Resolve per-tenant MongoDB connection if mongoManager is available.
	// The tenant ID is already in context from tmconsumer.MultiTenantConsumer.
	if mq.mongoManager != nil {
		tenantID := tmcore.GetTenantIDFromContext(ctx)
		if tenantID != "" {
			tenantDB, err := mq.mongoManager.GetDatabaseForTenant(ctx, tenantID)
			if err != nil {
				if isPermanentTenantError(err) {
					mq.logger.Warnf("Permanent tenant resolution failure for tenant %s (message will be dropped): %v", tenantID, err)

					return nil // Return nil so lib-commons Acks the message instead of requeuing
				}

				mq.logger.Errorf("Transient tenant resolution failure for tenant %s: %v", tenantID, err)

				return fmt.Errorf("resolve tenant mongo for tenant %s: %w", tenantID, err)
			}

			ctx = tmcore.ContextWithTenantMongo(ctx, tenantDB)
		}
	}

	return mq.handlerGenerateReport(ctx, delivery.Body, headersFromDelivery(delivery))
}

// headersFromDelivery converts amqp.Delivery.Headers (amqp.Table) to map[string]any.
// amqp.Table is already map[string]any, so this is a type assertion.
func headersFromDelivery(d amqp.Delivery) map[string]any {
	if d.Headers == nil {
		return nil
	}

	return map[string]any(d.Headers)
}

// handlerGenerateReport processes messages from the generate report queue.
func (mq *MultiQueueConsumer) handlerGenerateReport(ctx context.Context, body []byte, headers map[string]any) error {
	// Extract tenant ID from AMQP headers ONLY if not already set by the multi-tenant
	// consumer framework. In multi-tenant mode, the vhost-derived tenant ID is authoritative
	// and must not be overwritten by AMQP headers — publisher routing reads tenant ID from
	// context to determine which vhost to publish to.
	// In single-tenant mode (no tenant in context), the header extraction is the only source.
	if tmcore.GetTenantIDFromContext(ctx) == "" {
		ctx = extractTenantIDFromHeaders(ctx, headers)
	}

	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	spanCtx, span := tracer.Start(ctx, "consumer.handler_generate_report")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	// Resolve tenant-specific MongoDB database and inject into context.
	// In multi-tenant mode via handlerGenerateReportDelivery, the tenant DB is already
	// resolved and injected into context — resolveTenantMongo is a no-op because the
	// tenant ID is consumed. In single-tenant mode (mongoManager is nil), this is also a no-op.
	// For the single-tenant path that enters handlerGenerateReport directly (via ConsumerRoutes),
	// this call resolves tenant MongoDB when X-Tenant-ID is present in AMQP headers.
	spanCtx, err := resolveTenantMongo(spanCtx, mq.mongoManager)
	if err != nil {
		if isPermanentTenantError(err) {
			span.SetAttributes(attribute.String("app.tenant.error_class", "permanent"))
			opentelemetry.HandleSpanError(&span, "Tenant resolution failed permanently (will not retry).", err)
			logger.Warnf("Permanent tenant resolution failure (message will be dropped): %v", err)

			return nil // Return nil so lib-commons Acks the message instead of requeuing
		}

		span.SetAttributes(attribute.String("app.tenant.error_class", "transient"))
		opentelemetry.HandleSpanError(&span, "Tenant resolution failed (transient error).", err)
		logger.Errorf("Transient tenant resolution failure: %v", err)

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
// downstream repositories to retrieve the tenant DB via tmcore.GetMongoForTenant(ctx).
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

	// Skip resolution if tenant mongo is already set in context (e.g., by
	// handlerGenerateReportDelivery in multi-tenant mode). This avoids a
	// redundant GetDatabaseForTenant call.
	if tmcore.GetMongoFromContext(ctx) != nil {
		return ctx, nil
	}

	tenantDB, err := mongoManager.GetDatabaseForTenant(ctx, tenantID)
	if err != nil {
		// Propagate typed errors directly without re-wrapping, so callers can
		// use errors.Is / errors.As to classify permanent vs transient failures.
		return ctx, err
	}

	return tmcore.ContextWithTenantMongo(ctx, tenantDB), nil
}

// isPermanentTenantError returns true if the error indicates a permanent failure
// that will not resolve on retry. These errors should cause the message to be
// Nack'd without requeue and logged at WARN level (not ERROR).
func isPermanentTenantError(err error) bool {
	if err == nil {
		return false
	}

	// Tenant suspended or purged
	if tmcore.IsTenantSuspendedError(err) {
		return true
	}

	// Tenant does not exist in Tenant Manager
	if errors.Is(err, tmcore.ErrTenantNotFound) {
		return true
	}

	// Service not configured for this tenant
	if errors.Is(err, tmcore.ErrServiceNotConfigured) {
		return true
	}

	// Manager closed (shutdown in progress)
	if errors.Is(err, tmcore.ErrManagerClosed) {
		return true
	}

	return false
}
