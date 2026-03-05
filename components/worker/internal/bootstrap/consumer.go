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
	"github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	tmcore "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/core"
	tmmongo "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/mongo"

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
		if isPermanentTenantError(err) {
			// Permanent errors: tenant is suspended, not found, or service not configured.
			// Log at WARN (not ERROR) because this is an expected business condition, not
			// an infrastructure failure. The message will be Nack'd without requeue.
			span.SetAttributes(attribute.String("app.tenant.error_class", "permanent"))
			opentelemetry.HandleSpanError(&span, "Tenant resolution failed permanently (will not retry).", err)
			logger.Warnf("Permanent tenant resolution failure (message will be dropped): %v", err)
		} else {
			// Transient errors: circuit breaker open, network failure, etc.
			// Log at ERROR because this indicates an infrastructure issue that should
			// be investigated. The message will be Nack'd without requeue.
			span.SetAttributes(attribute.String("app.tenant.error_class", "transient"))
			opentelemetry.HandleSpanError(&span, "Tenant resolution failed (transient error).", err)
			logger.Errorf("Transient tenant resolution failure: %v", err)
		}

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
