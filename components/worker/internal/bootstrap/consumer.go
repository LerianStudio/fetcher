package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/LerianStudio/fetcher/components/worker/internal/adapters/rabbitmq"
	"github.com/LerianStudio/fetcher/components/worker/internal/services"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/LerianStudio/lib-commons/v4/commons"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

var (
	notifySignals       = signal.Notify
	extractExternalData = func(uc *services.UseCase, ctx context.Context, body []byte, headers map[string]any) error {
		return uc.ExtractExternalData(ctx, body, headers)
	}
)

// MultiQueueConsumer represents a multi-queue consumer.
type MultiQueueConsumer struct {
	consumerRoutes *rabbitmq.ConsumerRoutes
	UseCase        *services.UseCase
}

// NewMultiQueueConsumer create a new instance of MultiQueueConsumer.
func NewMultiQueueConsumer(routes *rabbitmq.ConsumerRoutes, useCase *services.UseCase) *MultiQueueConsumer {
	consumer := &MultiQueueConsumer{
		consumerRoutes: routes,
		UseCase:        useCase,
	}

	// Registry handlers for each queue
	routes.Register(os.Getenv("RABBITMQ_FETCHER_WORK_QUEUE"), consumer.handlerGenerateReport)

	return consumer
}

// Run starts consumers for all registered queues.
func (mq *MultiQueueConsumer) Run(l *commons.Launcher) error {
	// Create initial context with logger from ConsumerRoutes
	requestIDValue, err := commons.GenerateUUIDv7()
	if err != nil {
		requestIDValue = uuid.New()
	}

	requestID := requestIDValue.String()
	baseCtx := commons.ContextWithLogger(
		commons.ContextWithHeaderID(context.Background(), requestID),
		mq.consumerRoutes.Logger,
	)

	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	wg := &sync.WaitGroup{}

	sigs := make(chan os.Signal, 1)

	notifySignals(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)

	go func() {
		<-sigs
		mq.consumerRoutes.Log(baseCtx, libLog.LevelInfo, "received shutdown signal, starting graceful shutdown")
		cancel()
	}()

	if err := mq.consumerRoutes.RunConsumers(ctx, wg); err != nil {
		return err
	}

	wg.Wait()

	// Shutdown ConsumerRoutes gracefully after all workers are done
	shutdownCtx, shutdownCancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer shutdownCancel()

	if err := mq.consumerRoutes.Shutdown(shutdownCtx); err != nil {
		mq.consumerRoutes.Log(shutdownCtx, libLog.LevelError, "error during consumer routes shutdown", libLog.Err(err))
		return err
	}

	return nil
}

// handlerGenerateReport processes messages from the generate report queue.
func (mq *MultiQueueConsumer) handlerGenerateReport(ctx context.Context, body []byte, headers map[string]any) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	spanCtx, span := tracer.Start(ctx, "consumer.handler_generate_report")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	logger.Log(spanCtx, libLog.LevelInfo, "processing message from generate report queue")

	err := extractExternalData(mq.UseCase, spanCtx, body, headers)
	if err != nil {
		opentelemetry.HandleSpanError(span, "Error generating report.", err)

		logger.Log(spanCtx, libLog.LevelError, "error generating report", libLog.Err(err))

		return err
	}

	return nil
}
