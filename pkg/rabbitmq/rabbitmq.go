// Package rabbitmq provides a resilient RabbitMQ adapter for producing and consuming messages,
// handling connection and channel lifecycle, retries, circuit breaker, and graceful shutdown.
package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v3/commons"
	libConstants "github.com/LerianStudio/lib-commons/v3/commons/constants"
	libLog "github.com/LerianStudio/lib-commons/v3/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	libRabbitmq "github.com/LerianStudio/lib-commons/v3/commons/rabbitmq"

	"github.com/LerianStudio/fetcher/pkg/crypto"

	amqp "github.com/rabbitmq/amqp091-go"
	errgroup "golang.org/x/sync/errgroup"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Named constants for operational parameters.
const (
	// DefaultMaxRetryAttempts is the default number of retry attempts for channel establishment.
	DefaultMaxRetryAttempts = 3

	// DefaultMaxPublishAttempts is the default number of retry attempts for publishing a message.
	DefaultMaxPublishAttempts = 2

	// DefaultBaseRetryDelay is the default base delay for exponential backoff.
	DefaultBaseRetryDelay = 100 * time.Millisecond

	// DefaultMaxRetryDelay is the default maximum delay for exponential backoff.
	DefaultMaxRetryDelay = 2 * time.Second

	// DefaultConsumerReconnectDelay is the default delay between consumer reconnection attempts.
	// Deprecated: Use DefaultConsumerBaseRetryDelay instead.
	DefaultConsumerReconnectDelay = 500 * time.Millisecond

	// DefaultConsumerBaseRetryDelay is the base delay for consumer loop exponential backoff.
	DefaultConsumerBaseRetryDelay = 1 * time.Second

	// DefaultConsumerMaxRetryDelay is the maximum delay for consumer loop exponential backoff.
	DefaultConsumerMaxRetryDelay = 60 * time.Second

	// DefaultConsumerPermanentErrorDelay is the delay for permanent errors (e.g., 404, 403).
	// Uses a longer base to avoid hammering the broker for errors that won't resolve.
	DefaultConsumerPermanentErrorDelay = 30 * time.Second

	// DefaultCircuitBreakerThreshold is the default number of consecutive failures before opening the circuit.
	DefaultCircuitBreakerThreshold = 5

	// DefaultCircuitBreakerCooldown is the default cooldown period before transitioning to half-open state.
	DefaultCircuitBreakerCooldown = 30 * time.Second

	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second

	// DefaultSignatureTimestampTolerance is the maximum age of a message signature.
	// Messages older than this will be rejected to prevent replay attacks.
	DefaultSignatureTimestampTolerance = 5 * time.Minute

	// HeaderMessageSignature contains the HMAC-SHA256 signature of the message payload
	HeaderMessageSignature = "x-message-signature"

	// HeaderSignatureTimestamp contains the Unix timestamp when the signature was created
	HeaderSignatureTimestamp = "t"

	// HeaderSignatureVersion contains the version of the signature algorithm (e.g., "v1")
	HeaderSignatureVersion = "signature-version"
)

// AMQP error codes for error classification.
// Permanent errors indicate configuration or permission issues that
// will not resolve by retrying with the same parameters.
const (
	amqpNotFound           = 404 // Queue/exchange does not exist
	amqpAccessRefused      = 403 // Authentication or authorization failure
	amqpResourceLocked     = 405 // Resource locked by another connection
	amqpPreconditionFailed = 406 // Queue/exchange properties mismatch
)

// CircuitState represents the state of the circuit breaker.
type CircuitState int32

const (
	// CircuitClosed indicates the circuit is closed and requests flow normally.
	CircuitClosed CircuitState = iota
	// CircuitOpen indicates the circuit is open and requests are rejected.
	CircuitOpen
	// CircuitHalfOpen indicates the circuit is half-open and testing recovery.
	CircuitHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// AdapterOptions contains configuration options for the RabbitMQ adapter.
type AdapterOptions struct {
	// MaxRetryAttempts is the maximum number of retry attempts for channel establishment.
	MaxRetryAttempts int

	// MaxPublishAttempts is the maximum number of retry attempts for publishing a message.
	MaxPublishAttempts int

	// BaseRetryDelay is the base delay for exponential backoff.
	BaseRetryDelay time.Duration

	// MaxRetryDelay is the maximum delay for exponential backoff.
	MaxRetryDelay time.Duration

	// ConsumerReconnectDelay is the delay between consumer reconnection attempts.
	// Deprecated: Use ConsumerBaseRetryDelay instead.
	ConsumerReconnectDelay time.Duration

	// ConsumerBaseRetryDelay is the base delay for consumer loop exponential backoff.
	ConsumerBaseRetryDelay time.Duration

	// ConsumerMaxRetryDelay is the maximum delay for consumer loop exponential backoff.
	ConsumerMaxRetryDelay time.Duration

	// ConsumerPermanentErrorDelay is the fixed delay applied when a permanent AMQP error
	// (404, 403) is detected. These errors won't resolve without configuration changes.
	ConsumerPermanentErrorDelay time.Duration

	// CircuitBreakerThreshold is the number of consecutive failures before opening the circuit.
	CircuitBreakerThreshold int

	// CircuitBreakerCooldown is the cooldown period before transitioning to half-open state.
	CircuitBreakerCooldown time.Duration

	// ShutdownTimeout is the timeout for graceful shutdown.
	ShutdownTimeout time.Duration

	// MeterProvider is the OpenTelemetry meter provider for metrics.
	MeterProvider metric.MeterProvider

	// Signer is used for message signing and verification.
	// If nil, message signing/verification is disabled.
	Signer crypto.Signer

	// EnableMessageSigning controls whether messages are signed when publishing.
	// Default: true (if Signer is provided)
	EnableMessageSigning bool

	// EnableSignatureVerification controls whether signatures are verified when consuming.
	// Default: true (if Signer is provided)
	EnableSignatureVerification bool

	// SignatureTimestampTolerance is the maximum age of a message signature.
	// Messages with timestamps older than this will be rejected to prevent replay attacks.
	// Default: 5 minutes
	SignatureTimestampTolerance time.Duration
}

// DefaultOptions returns the default adapter options.
func DefaultOptions() AdapterOptions {
	return AdapterOptions{
		MaxRetryAttempts:            DefaultMaxRetryAttempts,
		MaxPublishAttempts:          DefaultMaxPublishAttempts,
		BaseRetryDelay:              DefaultBaseRetryDelay,
		MaxRetryDelay:               DefaultMaxRetryDelay,
		ConsumerReconnectDelay:      DefaultConsumerReconnectDelay,
		ConsumerBaseRetryDelay:      DefaultConsumerBaseRetryDelay,
		ConsumerMaxRetryDelay:       DefaultConsumerMaxRetryDelay,
		ConsumerPermanentErrorDelay: DefaultConsumerPermanentErrorDelay,
		CircuitBreakerThreshold:     DefaultCircuitBreakerThreshold,
		CircuitBreakerCooldown:      DefaultCircuitBreakerCooldown,
		ShutdownTimeout:             DefaultShutdownTimeout,
		EnableMessageSigning:        true,
		EnableSignatureVerification: true,
		SignatureTimestampTolerance: DefaultSignatureTimestampTolerance,
	}
}

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
		return nil, fmt.Errorf("rabbitmq ensure channel: %w", err)
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
			return fmt.Errorf("rabbitmq close channel: %w", err)
		}

		a.conn.Channel = nil
	}

	if a.conn.Connection != nil {
		if err := a.conn.Connection.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			return fmt.Errorf("rabbitmq close connection: %w", err)
		}

		a.conn.Connection = nil
	}

	return nil
}

// circuitBreaker implements a circuit breaker pattern for RabbitMQ operations.
type circuitBreaker struct {
	state             atomic.Int32
	consecutiveErrors atomic.Int32
	lastErrorTime     atomic.Int64
	threshold         int
	cooldown          time.Duration
}

// newCircuitBreaker creates a new circuit breaker with the specified threshold and cooldown.
func newCircuitBreaker(threshold int, cooldown time.Duration) *circuitBreaker {
	cb := &circuitBreaker{
		threshold: threshold,
		cooldown:  cooldown,
	}
	cb.state.Store(int32(CircuitClosed))

	return cb
}

// State returns the current state of the circuit breaker.
func (cb *circuitBreaker) State() CircuitState {
	return CircuitState(cb.state.Load())
}

// canExecute checks if the circuit breaker allows execution.
func (cb *circuitBreaker) canExecute() bool {
	state := CircuitState(cb.state.Load())

	switch state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if cooldown has passed
		lastError := time.Unix(0, cb.lastErrorTime.Load())
		if time.Since(lastError) > cb.cooldown {
			// Transition to half-open
			cb.state.CompareAndSwap(int32(CircuitOpen), int32(CircuitHalfOpen))

			return true
		}

		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// recordSuccess records a successful operation and potentially closes the circuit.
func (cb *circuitBreaker) recordSuccess() {
	cb.consecutiveErrors.Store(0)
	cb.state.Store(int32(CircuitClosed))
}

// recordFailure records a failed operation and potentially opens the circuit.
func (cb *circuitBreaker) recordFailure() {
	cb.lastErrorTime.Store(time.Now().UTC().UnixNano())
	newCount := cb.consecutiveErrors.Add(1)

	if int(newCount) >= cb.threshold {
		cb.state.Store(int32(CircuitOpen))
	}
}

// metrics holds operational metrics for the RabbitMQ adapter.
// Adapter defines the interface for RabbitMQ operations.
//
//go:generate mockgen --destination=rabbitmq.mock.go --package=rabbitmq . Adapter
type metrics struct {
	publishAttempts     metric.Int64Counter
	publishSuccesses    metric.Int64Counter
	publishFailures     metric.Int64Counter
	publishLatency      metric.Float64Histogram
	consumeProcessed    metric.Int64Counter
	consumeFailed       metric.Int64Counter
	circuitBreakerGauge metric.Int64ObservableGauge
}

// Adapter defines the interface for RabbitMQ operations.
type Adapter interface {
	ProducerDefault(ctx context.Context, exchange, key string, queueMessage []byte, header *map[string]any) error
	ConsumerLoop(ctx context.Context, queue string, concurrency int, handler func(ctx context.Context, body []byte, headers map[string]any) error) error
	Shutdown(ctx context.Context) error
	IsHealthy() bool
	CircuitBreakerState() CircuitState
}

var errDeliveriesClosed = errors.New("rabbitmq deliveries channel closed")

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// ErrSignatureVerificationFailed is returned when message signature verification fails.
var ErrSignatureVerificationFailed = errors.New("message signature verification failed")

// ErrMissingSignatureHeaders is returned when required signature headers are missing.
var ErrMissingSignatureHeaders = errors.New("missing required signature headers")

// ErrSignatureVerifierNotConfigured is returned when verification is enabled without signer.
var ErrSignatureVerifierNotConfigured = errors.New("signature verification enabled but signer is not configured")

// ErrSignatureExpired is returned when the message signature timestamp is too old.
var ErrSignatureExpired = errors.New("message signature has expired")

// isPermanentAMQPError returns true if the error is a non-recoverable AMQP error
// such as queue not found (404) or access refused (403). These errors indicate
// configuration or permission issues that will not resolve by retrying.
func isPermanentAMQPError(err error) bool {
	var amqpErr *amqp.Error
	if !errors.As(err, &amqpErr) {
		return false
	}

	switch amqpErr.Code {
	case amqpNotFound, amqpAccessRefused, amqpResourceLocked, amqpPreconditionFailed:
		return true
	default:
		return false
	}
}

// RabbitMQAdapter provides resilient publish and consumer operations over RabbitMQ.
type RabbitMQAdapter struct {
	conn    rabbitConnection
	channel amqpChannel

	// options contains the adapter configuration.
	options AdapterOptions

	// mu protects access to the channel.
	mu sync.Mutex

	// publishMu ensures that message publishing is thread-safe.
	publishMu sync.Mutex

	// shutdown indicates whether the adapter is in the process of shutting down.
	shutdown atomic.Bool

	// consumerWg tracks active consumer goroutines to ensure graceful shutdown.
	consumerWg sync.WaitGroup

	// circuitBreaker prevents cascading failures.
	circuitBreaker *circuitBreaker

	// metrics holds operational metrics.
	metrics *metrics
}

// NewRabbitMQAdapter initializes a new RabbitMQAdapter with the provided RabbitMQ connection.
// It uses default options for configuration. Use NewRabbitMQAdapterWithOptions for custom configuration.
func NewRabbitMQAdapter(c *libRabbitmq.RabbitMQConnection) *RabbitMQAdapter {
	return NewRabbitMQAdapterWithOptions(c, DefaultOptions())
}

// NewRabbitMQAdapterWithOptions initializes a new RabbitMQAdapter with the provided RabbitMQ connection and options.
func NewRabbitMQAdapterWithOptions(c *libRabbitmq.RabbitMQConnection, opts AdapterOptions) *RabbitMQAdapter {
	adapter := &rabbitmqConnectionAdapter{conn: c}

	prmq := &RabbitMQAdapter{
		conn:           adapter,
		options:        opts,
		circuitBreaker: newCircuitBreaker(opts.CircuitBreakerThreshold, opts.CircuitBreakerCooldown),
	}

	// Initialize metrics if meter provider is available
	if opts.MeterProvider != nil {
		prmq.initMetrics(opts.MeterProvider, c.Logger)
	}

	ch, err := adapter.EnsureChannel()
	if err != nil {
		c.Logger.Errorf("Failed to connect to RabbitMQ during initialization: %v", err)
		c.Logger.Warn("RabbitMQ connection will be retried on first message publish")
	} else {
		prmq.channel = ch
		prmq.startChannelWatcher(c.Logger, ch)
		c.Logger.Info("RabbitMQ producer connected successfully")
	}

	return prmq
}

// initMetrics initializes OpenTelemetry metrics for the adapter.
func (prmq *RabbitMQAdapter) initMetrics(provider metric.MeterProvider, Logger libLog.Logger) {
	meter := provider.Meter("github.com/LerianStudio/fetcher/pkg/rabbitmq")

	m := &metrics{}

	var err error

	m.publishAttempts, err = meter.Int64Counter(
		"rabbitmq.publish.attempts",
		metric.WithDescription("Number of publish attempts"),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		Logger.Errorf("Failed to initialize RabbitMQ publish attempts metric: %v", err)
		return
	}

	m.publishSuccesses, err = meter.Int64Counter(
		"rabbitmq.publish.successes",
		metric.WithDescription("Number of successful publishes"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		Logger.Errorf("Failed to initialize RabbitMQ publish successes metric: %v", err)
		return
	}

	m.publishFailures, err = meter.Int64Counter(
		"rabbitmq.publish.failures",
		metric.WithDescription("Number of failed publishes"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		Logger.Errorf("Failed to initialize RabbitMQ publish failures metric: %v", err)
		return
	}

	m.publishLatency, err = meter.Float64Histogram(
		"rabbitmq.publish.latency",
		metric.WithDescription("Publish latency in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		Logger.Errorf("Failed to initialize RabbitMQ publish latency metric: %v", err)
		return
	}

	m.consumeProcessed, err = meter.Int64Counter(
		"rabbitmq.consume.processed",
		metric.WithDescription("Number of successfully processed messages"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		Logger.Errorf("Failed to initialize RabbitMQ consume processed metric: %v", err)
		return
	}

	m.consumeFailed, err = meter.Int64Counter(
		"rabbitmq.consume.failed",
		metric.WithDescription("Number of failed message processings"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		Logger.Errorf("Failed to initialize RabbitMQ consume failed metric: %v", err)
		return
	}

	// Register circuit breaker state as an observable gauge
	m.circuitBreakerGauge, _ = meter.Int64ObservableGauge(
		"rabbitmq.circuit_breaker.state",
		metric.WithDescription("Circuit breaker state (0=closed, 1=open, 2=half-open)"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			o.Observe(int64(prmq.circuitBreaker.State()))
			return nil
		}),
	)

	prmq.metrics = m
}

// recordPublishAttempt safely records a publish attempt metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordPublishAttempt(ctx context.Context, attrs ...attribute.KeyValue) {
	if prmq.metrics != nil && prmq.metrics.publishAttempts != nil {
		prmq.metrics.publishAttempts.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// recordPublishSuccess safely records a publish success metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordPublishSuccess(ctx context.Context, attrs ...attribute.KeyValue) {
	if prmq.metrics != nil && prmq.metrics.publishSuccesses != nil {
		prmq.metrics.publishSuccesses.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// recordPublishFailure safely records a publish failure metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordPublishFailure(ctx context.Context, attrs ...attribute.KeyValue) {
	if prmq.metrics != nil && prmq.metrics.publishFailures != nil {
		prmq.metrics.publishFailures.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// recordConsumeProcessed safely records a consume processed metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordConsumeProcessed(ctx context.Context, attrs ...attribute.KeyValue) {
	if prmq.metrics != nil && prmq.metrics.consumeProcessed != nil {
		prmq.metrics.consumeProcessed.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// recordConsumeFailed safely records a consume failed metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordConsumeFailed(ctx context.Context, attrs ...attribute.KeyValue) {
	if prmq.metrics != nil && prmq.metrics.consumeFailed != nil {
		prmq.metrics.consumeFailed.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// recordPublishLatency safely records a publish latency metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordPublishLatency(ctx context.Context, latencyMs float64, attrs ...attribute.KeyValue) {
	if prmq.metrics != nil && prmq.metrics.publishLatency != nil {
		prmq.metrics.publishLatency.Record(ctx, latencyMs, metric.WithAttributes(attrs...))
	}
}

// IsHealthy returns true if the RabbitMQ connection is healthy.
// This method is designed for Kubernetes liveness/readiness probes.
func (prmq *RabbitMQAdapter) IsHealthy() bool {
	prmq.mu.Lock()
	defer prmq.mu.Unlock()

	return prmq.channel != nil && !prmq.channel.IsClosed() && !prmq.shutdown.Load()
}

// CircuitBreakerState returns the current state of the circuit breaker.
func (prmq *RabbitMQAdapter) CircuitBreakerState() CircuitState {
	return prmq.circuitBreaker.State()
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

// calculateBackoff calculates exponential backoff with jitter.
// It returns a delay based on the attempt number, bounded by maxDelay,
// with random jitter (0-25%) added to prevent thundering herd.
func calculateBackoff(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	// Ensure non-negative shift amount (attempt starts at 1 in practice)
	shiftAmount := max(attempt-1, 0)
	// Calculate exponential backoff: baseDelay * 2^(attempt-1)
	delay := min(baseDelay*time.Duration(1<<uint(shiftAmount)), maxDelay) // #nosec G115 -- shiftAmount is clamped to >= 0

	// Add jitter (0-25%) to prevent thundering herd
	// Using math/rand is acceptable here as jitter is not security-sensitive
	jitter := time.Duration(rand.Int63n(int64(delay / 4))) // #nosec G404 -- jitter for backoff timing is not security-sensitive

	return delay + jitter
}

// FullJitter returns a random duration in [0, baseDelay) for use as a retry delay.
// This implements the "full jitter" strategy from the AWS Architecture Blog,
// which provides the best spread across retrying clients.
func FullJitter(baseDelay time.Duration) time.Duration {
	if baseDelay <= 0 {
		return 0
	}

	return time.Duration(rand.Int63n(int64(baseDelay))) // #nosec G404 -- jitter for backoff timing is not security-sensitive
}

// NextBackoff doubles the given delay, capping at DefaultMaxRetryDelay.
func NextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > DefaultMaxRetryDelay {
		return DefaultMaxRetryDelay
	}

	return next
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

	var lastErr error

	for attempt := 1; attempt <= prmq.options.MaxRetryAttempts; attempt++ {
		ch, err := prmq.conn.EnsureChannel()
		if err == nil {
			prmq.channel = ch
			prmq.startChannelWatcher(logger, ch)
			prmq.circuitBreaker.recordSuccess()

			lastErr = nil

			break
		}

		lastErr = err

		prmq.circuitBreaker.recordFailure()

		backoff := calculateBackoff(attempt, prmq.options.BaseRetryDelay, prmq.options.MaxRetryDelay)
		time.Sleep(backoff)
	}

	if prmq.channel == nil {
		if span != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to establish RabbitMQ connection", lastErr)
		}

		logger.Errorf("Failed to establish RabbitMQ connection: %v", lastErr)

		return nil, fmt.Errorf("rabbitmq establish connection after %d attempts: %w", prmq.options.MaxRetryAttempts, lastErr)
	}

	logger.Info("RabbitMQ channel re-established on existing connection")

	return prmq.channel, nil
}

// ProducerDefault sends a message to the specified exchange and routing key in RabbitMQ.
func (prmq *RabbitMQAdapter) ProducerDefault(ctx context.Context, exchange, key string, queueMessage []byte, header *map[string]any) error {
	logger, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)

	logger.Infof("Init sent message")

	if prmq.shutdown.Load() {
		logger.Info("RabbitMQ adapter is shut down, cannot produce messages")
		return errors.New("rabbitmq adapter is shut down")
	}

	// Check circuit breaker state
	if !prmq.circuitBreaker.canExecute() {
		logger.Warnf("Circuit breaker is open, rejecting publish request")
		prmq.recordPublishFailure(ctx,
			attribute.String("exchange", exchange),
			attribute.String("routing_key", key),
			attribute.String("reason", "circuit_breaker_open"),
		)

		return ErrCircuitOpen
	}

	ctx, spanProducer := tracer.Start(ctx, "adapter.rabbitmq.produce")
	defer spanProducer.End()

	spanProducer.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.exchange", exchange),
		attribute.String("app.request.key", key),
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.destination.name", exchange),
		attribute.String("messaging.operation", "publish"),
		attribute.Int64("messaging.message.body.size", int64(len(queueMessage))),
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
		maps.Copy(headers, *header)

		err := libOpentelemetry.SetSpanAttributesFromStruct(&spanProducer, "app.request.rabbitmq.headers", *header)
		if err != nil {
			libOpentelemetry.HandleSpanError(&spanProducer, "Failed to convert headers to JSON string", err)
		}
	}

	libOpentelemetry.InjectTraceHeadersIntoQueue(ctx, (*map[string]any)(&headers))

	// Sign message if signer is configured and signing is enabled
	if prmq.options.Signer != nil && prmq.options.EnableMessageSigning {
		timestamp := time.Now().UTC().Unix()
		payload := crypto.BuildSignaturePayload(timestamp, queueMessage)
		signature := prmq.options.Signer.Sign(payload)

		headers[HeaderMessageSignature] = signature
		headers[HeaderSignatureTimestamp] = strconv.FormatInt(timestamp, 10)
		headers[HeaderSignatureVersion] = prmq.options.Signer.SignatureVersion()

		spanProducer.SetAttributes(
			attribute.String("messaging.signature.version", prmq.options.Signer.SignatureVersion()),
			attribute.Int64("messaging.signature.timestamp", timestamp),
		)
	}

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

	var lastErr error

	prmq.publishMu.Lock()
	defer prmq.publishMu.Unlock()

	startTime := time.Now().UTC()

	for attempt := 1; attempt <= prmq.options.MaxPublishAttempts; attempt++ {
		prmq.recordPublishAttempt(ctx,
			attribute.String("exchange", exchange),
			attribute.String("routing_key", key),
			attribute.Int("attempt", attempt),
		)

		channel, err := prmq.ensureChannel(&spanProducer, logger)
		if err != nil {
			prmq.circuitBreaker.recordFailure()

			return fmt.Errorf("rabbitmq ensure channel for publish: %w", err)
		}

		if err = publish(channel); err == nil {
			latencyMs := float64(time.Since(startTime).Milliseconds())
			prmq.recordPublishLatency(ctx, latencyMs,
				attribute.String("exchange", exchange),
				attribute.String("routing_key", key),
			)
			prmq.recordPublishSuccess(ctx,
				attribute.String("exchange", exchange),
				attribute.String("routing_key", key),
			)
			prmq.circuitBreaker.recordSuccess()
			logger.Infoln("Messages sent successfully")

			return nil
		}

		lastErr = err

		prmq.circuitBreaker.recordFailure()

		prmq.invalidateChannel(logger)

		if attempt < prmq.options.MaxPublishAttempts {
			logger.Warnf("Publish attempt %d/%d failed, retrying with new channel: %v", attempt, prmq.options.MaxPublishAttempts, err)
		}
	}

	prmq.recordPublishFailure(ctx,
		attribute.String("exchange", exchange),
		attribute.String("routing_key", key),
		attribute.String("reason", "max_retries_exceeded"),
	)

	libOpentelemetry.HandleSpanError(&spanProducer, "Failed to publish message to queue", lastErr)
	logger.Errorf("Failed to publish message: %s", lastErr)

	return fmt.Errorf("rabbitmq publish message after %d attempts: %w", prmq.options.MaxPublishAttempts, lastErr)
}

// ConsumerLoop fetches messages from the queue, delegates processing to handler, and applies ACK/NACK.
// The handler always receives headers (may be empty map if message has no headers).
func (prmq *RabbitMQAdapter) ConsumerLoop(ctx context.Context, queue string, concurrency int, handler func(ctx context.Context, body []byte, headers map[string]any) error) error {
	logger, tracer, reqID, _ := libCommons.NewTrackingFromContext(ctx)
	logger.Infof("Starting consumer loop for queue=%s", queue)

	if concurrency < 1 {
		concurrency = 1
	}

	var consecutiveErrors int

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
			consecutiveErrors = 0
			continue
		}

		if errors.Is(cycleErr, context.Canceled) || errors.Is(cycleErr, context.DeadlineExceeded) {
			return nil
		}

		consecutiveErrors++

		if errors.Is(cycleErr, errDeliveriesClosed) {
			logger.Warn("Deliveries channel closed, attempting to reconnect")
		} else {
			logger.Warnf("Consumer cycle finished with error (attempt %d): %v", consecutiveErrors, cycleErr)
		}

		// Calculate backoff based on error type
		var backoff time.Duration

		switch {
		case errors.Is(cycleErr, ErrCircuitOpen):
			backoff = prmq.options.CircuitBreakerCooldown
			logger.Warnf("Circuit breaker is open, waiting %v before retry", backoff)
		case isPermanentAMQPError(cycleErr):
			backoff = prmq.options.ConsumerPermanentErrorDelay
			logger.Errorf("Permanent AMQP error detected (attempt %d), using extended backoff of %v: %v",
				consecutiveErrors, backoff, cycleErr)
		default:
			backoff = calculateBackoff(consecutiveErrors,
				prmq.options.ConsumerBaseRetryDelay,
				prmq.options.ConsumerMaxRetryDelay)
		}

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
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
	ctxSpan, span := tracer.Start(ctx, "adapter.rabbitmq.consumer_connection_cycle")

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.queue", queue),
	)
	defer span.End()

	// Check circuit breaker before attempting to connect
	if !prmq.circuitBreaker.canExecute() {
		logger.Warnf("Circuit breaker is open, skipping consumer cycle")
		return ErrCircuitOpen
	}

	channel, err := prmq.ensureChannel(&span, logger)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to ensure RabbitMQ channel", err)

		return fmt.Errorf("rabbitmq consumer ensure channel: %w", err)
	}

	// Set QoS for fair dispatch of messages among consumers based on concurrency
	if errCh := channel.Qos(concurrency, 0, false); errCh != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to set RabbitMQ QoS", errCh)
		logger.Errorf("Failed to set RabbitMQ QoS: %v", errCh)
		prmq.circuitBreaker.recordFailure()
		prmq.invalidateChannel(logger)

		return fmt.Errorf("rabbitmq set qos: %w", errCh)
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
		prmq.circuitBreaker.recordFailure()
		prmq.invalidateChannel(logger)

		return fmt.Errorf("rabbitmq start consumer: %w", err)
	}

	sessionErr := prmq.dispatchDeliveries(ctxSpan, logger, consumerTag, queue, channel, deliveries, concurrency, handler)
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
	queue string,
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

				prmq.processDelivery(workerCtx, logger, queue, delivery, handler)

				return nil
			})
		}
	}
}

// processDelivery handles a single RabbitMQ message delivery, always extracting headers and creating proper context.
func (prmq *RabbitMQAdapter) processDelivery(
	ctx context.Context,
	logger libLog.Logger,
	queue string,
	d amqp.Delivery,
	handler func(ctx context.Context, body []byte, headers map[string]any) error,
) {
	// Extract headers from delivery
	headers := make(map[string]any)

	if d.Headers != nil {
		maps.Copy(headers, d.Headers)
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
	_, msgTracer, _, _ := libCommons.NewTrackingFromContext(msgCtx) //nolint:dogsled // NewTrackingFromContext returns 4 values, only tracer needed here

	hctx, hspan := msgTracer.Start(msgCtx, "rabbitmq.consumer.handle_message")
	defer hspan.End()

	// Add messaging semantic conventions (M12)
	hspan.SetAttributes(
		attribute.String("app.request.request_id", requestIDStr),
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.destination.name", queue),
		attribute.String("messaging.operation", "process"),
		attribute.Int64("messaging.message.body.size", int64(len(d.Body))),
	)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&hspan, "app.request.rabbitmq.consumer.message", d)
	if err != nil {
		libOpentelemetry.HandleSpanError(&hspan, "Failed to convert message to JSON string", err)
	}

	if prmq.options.EnableSignatureVerification {
		if prmq.options.Signer == nil {
			if nackErr := d.Nack(false, false); nackErr != nil {
				logWithFields.Warnf("Failed to Nack message when signer is not configured: %v", nackErr)
			}

			libOpentelemetry.HandleSpanError(&hspan, "Signature verification is enabled but signer is not configured", ErrSignatureVerifierNotConfigured)
			logWithFields.Errorf("Signature verification is enabled but signer is not configured")
			prmq.recordConsumeFailed(ctx,
				attribute.String("queue", queue),
				attribute.String("reason", "signature_verifier_not_configured"),
			)

			return
		}

		if err := prmq.verifyMessageSignature(d.Body, headers, logWithFields, &hspan); err != nil {
			if nackErr := d.Nack(false, false); nackErr != nil {
				logWithFields.Warnf("Failed to Nack message after signature verification failure: %v", nackErr)
			}

			libOpentelemetry.HandleSpanError(&hspan, "Message signature verification failed", err)
			logWithFields.Errorf("Message signature verification failed: %v", err)
			prmq.recordConsumeFailed(ctx,
				attribute.String("queue", queue),
				attribute.String("reason", "signature_verification_failed"),
			)

			return
		}
	}

	// Use the logger with fields created above (logWithFields) for all logging
	// Recover from panics during message processing
	defer func() {
		if r := recover(); r != nil {
			if nackErr := d.Nack(false, false); nackErr != nil {
				logWithFields.Warnf("Failed to Nack message after panic: %v", nackErr)
			}

			err := fmt.Errorf("%v", r)
			logWithFields.Errorf("Panic while processing message: %v", r)
			libOpentelemetry.HandleSpanError(&hspan, "Panic while processing message", err)
			prmq.recordConsumeFailed(ctx,
				attribute.String("queue", queue),
				attribute.String("reason", "panic"),
			)
		}
	}()

	if err := handler(hctx, d.Body, headers); err != nil {
		if nackErr := d.Nack(false, false); nackErr != nil {
			logWithFields.Warnf("Failed to Nack message after handler error: %v", nackErr)
		}

		libOpentelemetry.HandleSpanError(&hspan, "Handler failed to process consumed message", err)
		logWithFields.Errorf("Handler failed to process consumed message: %v", err)
		prmq.recordConsumeFailed(ctx,
			attribute.String("queue", queue),
			attribute.String("reason", "handler_error"),
		)

		return
	}

	if err := d.Ack(false); err != nil {
		libOpentelemetry.HandleSpanError(&hspan, "Failed to ACK consumed message", err)
		logWithFields.Errorf("Failed to ACK consumed message: %v", err)
	}

	prmq.recordConsumeProcessed(ctx,
		attribute.String("queue", queue),
	)
}

// Shutdown gracefully closes open channels and the underlying connection.
// It respects the context deadline or uses the configured shutdown timeout.
func (prmq *RabbitMQAdapter) Shutdown(ctx context.Context) error {
	logger := libCommons.NewLoggerFromContext(ctx)

	// Indicate shutdown in progress
	prmq.shutdown.Store(true)

	// Determine timeout: use context deadline if available, otherwise use configured timeout
	timeout := prmq.options.ShutdownTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	// Use a done channel to implement timeout on WaitGroup
	done := make(chan struct{})

	go func() {
		prmq.consumerWg.Wait()
		close(done)
	}()

	// Create a timeout context for the wait operation
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-done:
		// All consumers finished
		logger.Info("All consumers finished processing")
	case <-timeoutCtx.Done():
		logger.Warn("Shutdown timeout reached, some consumers may not have finished")
	case <-ctx.Done():
		logger.Warn("Shutdown context canceled, some consumers may not have finished")
	}

	// Invalidate and close the channel
	prmq.invalidateChannel(logger)

	if err := prmq.conn.Close(); err != nil {
		logger.Errorf("Failed to close RabbitMQ connection: %v", err)

		return fmt.Errorf("rabbitmq shutdown close connection: %w", err)
	}

	logger.Info("RabbitMQ repository shut down gracefully")

	return nil
}

// verifyMessageSignature verifies the HMAC signature of a message.
// It checks for required signature headers, validates the signature version,
// and verifies the signature against the message body.
func (prmq *RabbitMQAdapter) verifyMessageSignature(
	body []byte,
	headers map[string]any,
	logger libLog.Logger,
	span *trace.Span,
) error {
	// Extract signature from headers
	signatureRaw, ok := headers[HeaderMessageSignature]
	if !ok {
		return fmt.Errorf("%w: %s", ErrMissingSignatureHeaders, HeaderMessageSignature)
	}

	signature, ok := signatureRaw.(string)
	if !ok {
		return fmt.Errorf("%w: %s must be a string", ErrMissingSignatureHeaders, HeaderMessageSignature)
	}

	// Extract timestamp from headers
	timestampRaw, ok := headers[HeaderSignatureTimestamp]
	if !ok {
		return fmt.Errorf("%w: %s", ErrMissingSignatureHeaders, HeaderSignatureTimestamp)
	}

	var timestamp int64

	switch v := timestampRaw.(type) {
	case string:
		var err error

		timestamp, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: invalid timestamp format: %v", ErrSignatureVerificationFailed, err)
		}
	case int64:
		timestamp = v
	case int:
		timestamp = int64(v)
	default:
		return fmt.Errorf("%w: %s must be a string or int64", ErrMissingSignatureHeaders, HeaderSignatureTimestamp)
	}

	// Check timestamp freshness to prevent replay attacks
	if prmq.options.SignatureTimestampTolerance > 0 {
		messageTime := time.Unix(timestamp, 0)

		age := time.Since(messageTime)
		if age > prmq.options.SignatureTimestampTolerance {
			return fmt.Errorf("%w: message is %v old, tolerance is %v",
				ErrSignatureExpired, age.Round(time.Second), prmq.options.SignatureTimestampTolerance)
		}
	}

	// Extract and validate signature version
	versionRaw, ok := headers[HeaderSignatureVersion]
	if !ok {
		return fmt.Errorf("%w: %s", ErrMissingSignatureHeaders, HeaderSignatureVersion)
	}

	version, ok := versionRaw.(string)
	if !ok {
		return fmt.Errorf("%w: %s must be a string", ErrMissingSignatureHeaders, HeaderSignatureVersion)
	}

	// Check if the signature version matches the signer's version
	if version != prmq.options.Signer.SignatureVersion() {
		return fmt.Errorf("%w: expected %s, got %s",
			crypto.ErrUnsupportedSignatureVersion,
			prmq.options.Signer.SignatureVersion(),
			version,
		)
	}

	// Add signature attributes to span
	if span != nil {
		(*span).SetAttributes(
			attribute.String("messaging.signature.version", version),
			attribute.Int64("messaging.signature.timestamp", timestamp),
		)
	}

	// Build the signature payload and verify
	payload := crypto.BuildSignaturePayload(timestamp, body)
	if err := prmq.options.Signer.Verify(payload, signature); err != nil {
		return fmt.Errorf("%w: %v", ErrSignatureVerificationFailed, err)
	}

	logger.Debug("Message signature verified successfully")

	return nil
}
