// Package rabbitmq provides a resilient RabbitMQ adapter for producing and consuming messages,
// handling connection and channel lifecycle, retries, and graceful shutdown.
package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libConstants "github.com/LerianStudio/lib-commons/v2/commons/constants"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v2/commons/rabbitmq"

	"github.com/LerianStudio/fetcher/pkg"

	amqp "github.com/rabbitmq/amqp091-go"
	errgroup "golang.org/x/sync/errgroup"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type rabbitConnection interface {
	EnsureChannel() (amqpChannel, error)
	Close() error
}

// amqpChannel defines the methods required from a RabbitMQ channel.
type amqpChannel interface {
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Cancel(consumer string, noWait bool) error
	Close() error
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	IsClosed() bool
	Qos(prefetchCount, prefetchSize int, global bool) error
}

// rabbitmqConnectionAdapter adapts the RabbitMQConnection to the rabbitConnection interface.
type rabbitmqConnectionAdapter struct {
	conn *libRabbitmq.RabbitMQConnection
}

// EnsureChannel ensures that a RabbitMQ channel is available.
func (a *rabbitmqConnectionAdapter) EnsureChannel() (amqpChannel, error) {
	if err := a.conn.EnsureChannel(); err != nil {
		return nil, err
	}

	return a.conn.Channel, nil
}

// Close gracefully closes the RabbitMQ connection and its channel.
func (a *rabbitmqConnectionAdapter) Close() error {
	if a.conn == nil {
		return nil
	}

	if a.conn.Channel != nil {
		if err := a.conn.Channel.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			return err
		}

		a.conn.Channel = nil
	}

	if a.conn.Connection != nil {
		if err := a.conn.Connection.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			return err
		}

		a.conn.Connection = nil
	}

	return nil
}

// RabbitMQAdapter provides resilient publish and consumer operations over RabbitMQ.
type RabbitMQAdapter struct {
	conn    rabbitConnection
	channel amqpChannel

	// mu protects access to the channel.
	mu sync.Mutex

	// publishMu ensures that message publishing is thread-safe.
	publishMu sync.Mutex

	// shutdown indicates whether the adapter is in the process of shutting down.
	shutdown atomic.Bool

	// consumerWg tracks active consumer goroutines to ensure graceful shutdown.
	consumerWg sync.WaitGroup
}

var errDeliveriesClosed = errors.New("rabbitmq deliveries channel closed")

// NewRabbitMQAdapter initializes a new RabbitMQAdapter with the provided RabbitMQ connection.
func NewRabbitMQAdapter(c *libRabbitmq.RabbitMQConnection) *RabbitMQAdapter {
	adapter := &rabbitmqConnectionAdapter{conn: c}

	prmq := &RabbitMQAdapter{
		conn: adapter,
	}

	ch, err := adapter.EnsureChannel()
	if err != nil {
		c.Logger.Errorf("⚠️  Failed to connect to RabbitMQ during initialization: %v", err)
		c.Logger.Warn("RabbitMQ connection will be retried on first message publish")
	} else {
		prmq.channel = ch
		prmq.startChannelWatcher(c.Logger, ch)
		c.Logger.Info("✅ RabbitMQ producer connected successfully")
	}

	return prmq
}

// invalidateChannel safely closes and nullifies the current RabbitMQ channel.
func (prmq *RabbitMQAdapter) invalidateChannel(logger libLog.Logger) {
	prmq.mu.Lock()
	channel := prmq.channel
	prmq.channel = nil
	prmq.mu.Unlock()

	if channel != nil && !channel.IsClosed() {
		if err := channel.Close(); err != nil && logger != nil {
			logger.Warnf("Failed to close RabbitMQ channel: %v", err)
		}
	}
}

// startChannelWatcher monitors the RabbitMQ channel for closure events.
func (prmq *RabbitMQAdapter) startChannelWatcher(logger libLog.Logger, channel amqpChannel) {
	if channel == nil {
		return
	}

	notifications := channel.NotifyClose(make(chan *amqp.Error, 1))

	go func() {
		if err, ok := <-notifications; ok && err != nil {
			logger.Warnf("RabbitMQ channel closed: %v", err)
		} else {
			logger.Warn("RabbitMQ channel closed")
		}

		prmq.mu.Lock()
		defer prmq.mu.Unlock()

		prmq.channel = nil
	}()
}

// ensureChannel checks and establishes a RabbitMQ channel if not already available.
func (prmq *RabbitMQAdapter) ensureChannel(span *trace.Span, logger libLog.Logger) (amqpChannel, error) {
	prmq.mu.Lock()
	channel := prmq.channel
	prmq.mu.Unlock()

	if channel != nil && !channel.IsClosed() {
		return channel, nil
	}

	prmq.mu.Lock()
	defer prmq.mu.Unlock()

	if prmq.channel != nil && !prmq.channel.IsClosed() {
		return prmq.channel, nil
	}

	logger.Warn("RabbitMQ channel not initialized - attempting to connect...")

	const maxAttempts = 3

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ch, err := prmq.conn.EnsureChannel()
		if err == nil {
			prmq.channel = ch
			prmq.startChannelWatcher(logger, ch)

			lastErr = nil

			break
		}

		lastErr = err

		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}

	if prmq.channel == nil {
		if span != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to establish RabbitMQ connection", lastErr)
		}

		logger.Errorf("Failed to establish RabbitMQ connection: %v", lastErr)

		return nil, lastErr
	}

	logger.Info("✅ RabbitMQ connection established on-demand")

	return prmq.channel, nil
}

// ProducerDefault sends a message to the specified exchange and routing key in RabbitMQ.
func (prmq *RabbitMQAdapter) ProducerDefault(ctx context.Context, exchange, key string, queueMessage []byte, header *map[string]interface{}) error {
	logger, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	logger.Infof("Init sent message")

	if prmq.shutdown.Load() {
		logger.Info("RabbitMQ adapter is shut down, cannot produce messages")
		return errors.New("rabbitmq adapter is shut down")
	}

	_, spanProducer := tracer.Start(ctx, "rabbitmq.producer.publish_message")
	defer spanProducer.End()

	spanProducer.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.exchange", exchange),
		attribute.String("app.request.key", key),
	)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&spanProducer, "app.request.rabbitmq.message", string(queueMessage))
	if err != nil {
		libOpentelemetry.HandleSpanError(&spanProducer, "Failed to convert queue message to JSON string", err)
	}

	headers := amqp.Table{
		libConstants.HeaderID: reqID,
		"x-retry-count":       0,
	}

	if header != nil {
		for k, v := range *header {
			headers[k] = v
		}

		err := libOpentelemetry.SetSpanAttributesFromStruct(&spanProducer, "app.request.rabbitmq.headers", *header)
		if err != nil {
			libOpentelemetry.HandleSpanError(&spanProducer, "Failed to convert headers to JSON string", err)
		}
	}

	libOpentelemetry.InjectTraceHeadersIntoQueue(ctx, (*map[string]any)(&headers))

	publish := func(ch amqpChannel) error {
		return ch.Publish(
			exchange,
			key,
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Headers:      headers,
				Body:         queueMessage,
			})
	}

	const maxPublishAttempts = 2

	var lastErr error

	prmq.publishMu.Lock()
	defer prmq.publishMu.Unlock()

	for attempt := 1; attempt <= maxPublishAttempts; attempt++ {
		channel, err := prmq.ensureChannel(&spanProducer, logger)
		if err != nil {
			return err
		}

		if err = publish(channel); err == nil {
			logger.Infoln("Messages sent successfully")
			return nil
		}

		lastErr = err

		prmq.invalidateChannel(logger)

		if attempt < maxPublishAttempts {
			logger.Warnf("Publish attempt %d/%d failed, retrying with new channel: %v", attempt, maxPublishAttempts, err)
		}
	}

	libOpentelemetry.HandleSpanError(&spanProducer, "Failed to publish message to queue", lastErr)
	logger.Errorf("Failed to publish message: %s", lastErr)

	return lastErr
}

// ConsumerLoop fetches messages from the queue, delegates processing to handler, and applies ACK/NACK.
// The handler always receives headers (may be empty map if message has no headers).
func (prmq *RabbitMQAdapter) ConsumerLoop(ctx context.Context, queue string, concurrency int, handler func(ctx context.Context, body []byte, headers map[string]any) error) error {
	logger, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)
	logger.Infof("Starting consumer loop for queue=%s", queue)

	if concurrency < 1 {
		concurrency = 1
	}

	for {
		if ctx.Err() != nil {
			logger.Warnf("Context canceled while running consumer loop: %v", ctx.Err())
			return ctx.Err()
		}

		if prmq.shutdown.Load() {
			logger.Info("RabbitMQ adapter is shut down, exiting consumer loop")
			return errors.New("rabbitmq adapter is shut down")
		}

		cycleErr := prmq.runConsumerCycle(ctx, tracer, logger, queue, reqID, concurrency, handler)
		if cycleErr == nil {
			continue
		}

		if errors.Is(cycleErr, context.Canceled) || errors.Is(cycleErr, context.DeadlineExceeded) {
			return nil
		}

		if errors.Is(cycleErr, errDeliveriesClosed) {
			logger.Warn("Deliveries channel closed, attempting to reconnect")
		} else {
			logger.Warnf("Consumer cycle finished with error: %v", cycleErr)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// runConsumerCycle establishes a consumer session and processes messages from the specified queue.
func (prmq *RabbitMQAdapter) runConsumerCycle(
	ctx context.Context,
	tracer trace.Tracer,
	logger libLog.Logger,
	queue, reqID string,
	concurrency int,
	handler func(ctx context.Context, body []byte, headers map[string]any) error,
) error {
	ctxSpan, span := tracer.Start(ctx, "rabbitmq.consumer.connection_cycle")

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.queue", queue),
	)
	defer span.End()

	channel, err := prmq.ensureChannel(&span, logger)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to ensure RabbitMQ channel", err)
		return err
	}

	// Set QoS for fair dispatch of messages among consumers based on concurrency
	if errCh := channel.Qos(concurrency, 0, false); errCh != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to set RabbitMQ QoS", errCh)
		logger.Errorf("Failed to set RabbitMQ QoS: %v", errCh)
		prmq.invalidateChannel(logger)

		return errCh
	}

	consumerTag := fmt.Sprintf("%s-%s", queue, reqID)

	deliveries, err := channel.Consume(
		queue,
		consumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to start RabbitMQ consumer", err)
		logger.Errorf("Failed to start RabbitMQ consumer: %v", err)
		prmq.invalidateChannel(logger)

		return err
	}

	sessionErr := prmq.dispatchDeliveries(ctxSpan, logger, consumerTag, channel, deliveries, concurrency, handler)
	if sessionErr == nil {
		return nil
	}

	if errors.Is(sessionErr, context.Canceled) || errors.Is(sessionErr, context.DeadlineExceeded) {
		return sessionErr
	}

	libOpentelemetry.HandleSpanError(&span, "RabbitMQ consumer session exited with error", sessionErr)

	if errors.Is(sessionErr, errDeliveriesClosed) {
		prmq.invalidateChannel(logger)
	}

	return sessionErr
}

// dispatchDeliveries processes incoming RabbitMQ deliveries with concurrency control.
func (prmq *RabbitMQAdapter) dispatchDeliveries(
	ctx context.Context,
	logger libLog.Logger,
	consumerTag string,
	channel amqpChannel,
	deliveries <-chan amqp.Delivery,
	concurrency int,
	handler func(ctx context.Context, body []byte, headers map[string]any) error,
) error {
	// Cancela o consumer apenas uma vez
	var cancelOnce sync.Once

	cancelConsumer := func() {
		cancelOnce.Do(func() {
			if cancelErr := channel.Cancel(consumerTag, false); cancelErr != nil && !errors.Is(cancelErr, amqp.ErrClosed) {
				logger.Warnf("Failed to cancel RabbitMQ consumer: %v", cancelErr)
			}
		})
	}
	defer cancelConsumer()

	// Use errgroup to manage concurrent processing of deliveries
	group, workerCtx := errgroup.WithContext(ctx)
	group.SetLimit(concurrency)

	for {
		select {
		case <-workerCtx.Done():
			cancelConsumer()

			if err := group.Wait(); err != nil {
				return err
			}

			return workerCtx.Err()

		case msg, ok := <-deliveries:
			if !ok {
				cancelConsumer()

				if err := group.Wait(); err != nil {
					return err
				}

				return errDeliveriesClosed
			}

			delivery := msg

			prmq.consumerWg.Add(1)
			group.Go(func() error {
				defer prmq.consumerWg.Done()

				prmq.processDelivery(logger, delivery, handler)

				return nil
			})
		}
	}
}

// processDelivery handles a single RabbitMQ message delivery, always extracting headers and creating proper context.
func (prmq *RabbitMQAdapter) processDelivery(
	logger libLog.Logger,
	d amqp.Delivery,
	handler func(ctx context.Context, body []byte, headers map[string]any) error,
) {
	// Extract headers from delivery
	headers := make(map[string]any)

	if d.Headers != nil {
		for k, v := range d.Headers {
			headers[k] = v
		}
	}

	// Extract request ID from headers or generate new one
	requestID, found := headers[libConstants.HeaderID]
	if !found {
		requestID = libCommons.GenerateUUIDv7().String()
	}

	requestIDStr, ok := requestID.(string)
	if !ok {
		requestIDStr = libCommons.GenerateUUIDv7().String()
	}

	// Create context with request ID and logger
	logWithFields := logger.WithFields(
		libConstants.HeaderID, requestIDStr,
	).WithDefaultMessageTemplate(requestIDStr + libConstants.LoggerDefaultSeparator)

	msgCtx := libCommons.ContextWithLogger(
		libCommons.ContextWithHeaderID(context.Background(), requestIDStr),
		logWithFields,
	)

	// Extract trace context from message headers
	msgCtx = libOpentelemetry.ExtractTraceContextFromQueueHeaders(msgCtx, d.Headers)

	// Create tracer from context and start span
	msgTracer := pkg.NewTracerFromContext(msgCtx)

	hctx, hspan := msgTracer.Start(msgCtx, "rabbitmq.consumer.handle_message")
	defer hspan.End()

	hspan.SetAttributes(
		attribute.String("app.request.rabbitmq.consumer.request_id", requestIDStr),
	)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&hspan, "app.request.rabbitmq.consumer.message", d)
	if err != nil {
		libOpentelemetry.HandleSpanError(&hspan, "Failed to convert message to JSON string", err)
	}

	// Use the logger with fields created above (logWithFields) for all logging
	// Recover from panics during message processing
	defer func() {
		if r := recover(); r != nil {
			_ = d.Nack(false, false)
			err := fmt.Errorf("%v", r)
			logWithFields.Errorf("Panic while processing message: %v", r)
			libOpentelemetry.HandleSpanError(&hspan, "Panic while processing message", err)
		}
	}()

	if err := handler(hctx, d.Body, headers); err != nil {
		_ = d.Nack(false, false)

		libOpentelemetry.HandleSpanError(&hspan, "Handler failed to process consumed message", err)
		logWithFields.Errorf("Handler failed to process consumed message: %v", err)

		return
	}

	if err := d.Ack(false); err != nil {
		libOpentelemetry.HandleSpanError(&hspan, "Failed to ACK consumed message", err)
		logWithFields.Errorf("Failed to ACK consumed message: %v", err)
	}
}

// Shutdown gracefully closes open channels and the underlying connection.
func (prmq *RabbitMQAdapter) Shutdown(ctx context.Context) error {
	logger := libCommons.NewLoggerFromContext(ctx)

	// Indicate shutdown in progress
	prmq.shutdown.Store(true)

	// Wait for all consumers to finish processing
	prmq.consumerWg.Wait()

	// Invalidate and close the channel
	prmq.invalidateChannel(logger)

	if err := prmq.conn.Close(); err != nil {
		logger.Errorf("Failed to close RabbitMQ connection: %v", err)

		return err
	}

	logger.Info("RabbitMQ repository shut down gracefully")

	return nil
}
