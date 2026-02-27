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
		return ctx, fmt.Errorf("get tenant mongo database for tenant %s: %w", tenantID, err)
	}

	return tmcore.ContextWithTenantMongo(ctx, tenantDB), nil
}
