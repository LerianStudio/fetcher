// Package rabbitmq provides a resilient RabbitMQ adapter for producing and consuming messages,
// handling connection and channel lifecycle, retries, circuit breaker, and graceful shutdown.
package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	observability "github.com/LerianStudio/lib-observability"

	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libBackoff "github.com/LerianStudio/lib-commons/v5/commons/backoff"
	libCircuitBreaker "github.com/LerianStudio/lib-commons/v5/commons/circuitbreaker"
	libConstants "github.com/LerianStudio/lib-commons/v5/commons/constants"
	libRabbitmq "github.com/LerianStudio/lib-commons/v5/commons/rabbitmq"
	obsConstants "github.com/LerianStudio/lib-observability/constants"
	libLog "github.com/LerianStudio/lib-observability/log"
	obsMetrics "github.com/LerianStudio/lib-observability/metrics"
	obsRuntime "github.com/LerianStudio/lib-observability/runtime"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

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

	// DefaultSignatureFutureSkew is the maximum producer/consumer clock skew
	// accepted for signed RabbitMQ messages. Larger future timestamps are rejected
	// because they can extend the replay window.
	DefaultSignatureFutureSkew = 30 * time.Second

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
type CircuitState string

const (
	// CircuitClosed indicates the circuit is closed and requests flow normally.
	CircuitClosed CircuitState = "closed"
	// CircuitOpen indicates the circuit is open and requests are rejected.
	CircuitOpen CircuitState = "open"
	// CircuitHalfOpen indicates the circuit is half-open and testing recovery.
	CircuitHalfOpen CircuitState = "half-open"
	// CircuitUnknown indicates lib-commons could not map the breaker state.
	CircuitUnknown CircuitState = "unknown"
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
	case CircuitUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

const (
	rabbitMQCircuitBreakerName = "fetcher.rabbitmq.adapter"

	attrExchange                       = "exchange"
	attrRoutingKey                     = "routing_key"
	attrReason                         = "reason"
	attrAttempt                        = "attempt"
	attrQueue                          = "queue"
	attrAppRequestRequestID            = obsConstants.AttrPrefixAppRequest + "request_id"
	attrAppRequestExchange             = obsConstants.AttrPrefixAppRequest + "exchange"
	attrAppRequestKey                  = obsConstants.AttrPrefixAppRequest + "key"
	attrAppRequestQueue                = obsConstants.AttrPrefixAppRequest + "queue"
	attrMessagingSystem                = "messaging.system"
	attrMessagingDestinationName       = "messaging.destination.name"
	attrMessagingOperation             = "messaging.operation"
	attrMessagingMessageBodySize       = "messaging.message.body.size"
	attrMessagingSignatureVersion      = "messaging.signature.version"
	attrMessagingSignatureTimestamp    = "messaging.signature.timestamp"
	attrReasonCircuitBreakerOpen       = "circuit_breaker_open"
	attrReasonMaxRetriesExceeded       = "max_retries_exceeded"
	attrReasonSignatureVerifierMissing = "signature_verifier_not_configured"
	attrReasonSignatureFailed          = "signature_verification_failed"
	attrReasonPanic                    = "panic"
	attrReasonHandlerError             = "handler_error"
)

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

	// AllowLegacyBodyOnlySignatureFallback gates the rolling-drain compatibility
	// path for body-only signatures. Default: false.
	AllowLegacyBodyOnlySignatureFallback bool
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
	Confirm(noWait bool) error
	NotifyPublish(receiver chan amqp.Confirmation) chan amqp.Confirmation
	NotifyReturn(receiver chan amqp.Return) chan amqp.Return
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

// metrics holds operational metrics for the RabbitMQ adapter.
// Adapter defines the interface for RabbitMQ operations.
//
//go:generate mockgen --destination=rabbitmq.mock.go --package=rabbitmq . Adapter
type metrics struct {
	factory *obsMetrics.MetricsFactory
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

// ErrSignatureFromFuture is returned when the message signature timestamp is in the future.
var ErrSignatureFromFuture = errors.New("message signature timestamp is in the future")

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

	// circuitBreaker delegates failure accounting and state transitions to
	// lib-commons. Fetcher keeps only RabbitMQ-specific retry/ACK semantics here.
	circuitBreaker libCircuitBreaker.Manager

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
	breakerManager := newRabbitMQCircuitBreakerManager(c.Logger, opts)

	prmq := &RabbitMQAdapter{
		conn:           adapter,
		options:        opts,
		circuitBreaker: breakerManager,
	}

	// Initialize metrics if meter provider is available
	if opts.MeterProvider != nil {
		prmq.initMetrics(opts.MeterProvider, c.Logger)
	}

	ch, err := adapter.EnsureChannel()
	if err != nil {
		c.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to connect to RabbitMQ during initialization: %v", err))
		c.Logger.Log(context.Background(), libLog.LevelWarn, "RabbitMQ connection will be retried on first message publish")
	} else {
		prmq.channel = ch
		prmq.startChannelWatcher(c.Logger, ch)
		c.Logger.Log(context.Background(), libLog.LevelInfo, "RabbitMQ producer connected successfully")
	}

	return prmq
}

func newRabbitMQCircuitBreakerManager(logger libLog.Logger, opts AdapterOptions) libCircuitBreaker.Manager {
	if logger == nil {
		logger = libLog.NewNop()
	}

	manager, err := libCircuitBreaker.NewManager(logger)
	if err != nil {
		return nil
	}

	threshold := opts.CircuitBreakerThreshold
	if threshold <= 0 {
		threshold = DefaultCircuitBreakerThreshold
	}

	if threshold > math.MaxUint32 {
		logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("RabbitMQ circuit breaker threshold %d exceeds uint32 max; using default %d", threshold, DefaultCircuitBreakerThreshold))
		threshold = DefaultCircuitBreakerThreshold
	}

	cooldown := opts.CircuitBreakerCooldown
	if cooldown <= 0 {
		cooldown = DefaultCircuitBreakerCooldown
	}

	_, err = manager.GetOrCreate(rabbitMQCircuitBreakerName, libCircuitBreaker.Config{
		MaxRequests:         1,
		Timeout:             cooldown,
		ConsecutiveFailures: uint32(threshold), // #nosec G115 -- threshold is validated above.
	})
	if err != nil {
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("failed to initialize RabbitMQ circuit breaker: %v", err))

		return nil
	}

	return manager
}

func mapCircuitState(state libCircuitBreaker.State) CircuitState {
	switch state {
	case libCircuitBreaker.StateClosed:
		return CircuitClosed
	case libCircuitBreaker.StateOpen:
		return CircuitOpen
	case libCircuitBreaker.StateHalfOpen:
		return CircuitHalfOpen
	default:
		return CircuitUnknown
	}
}

func (prmq *RabbitMQAdapter) executeWithCircuitBreaker(fn func() error) error {
	if prmq.circuitBreaker == nil {
		return fn()
	}

	_, err := prmq.circuitBreaker.Execute(rabbitMQCircuitBreakerName, func() (any, error) {
		return nil, fn()
	})
	if errors.Is(err, libCircuitBreaker.ErrBreakerOpen) || errors.Is(err, libCircuitBreaker.ErrBreakerHalfOpenFull) {
		return ErrCircuitOpen
	}

	return err
}

// initMetrics initializes OpenTelemetry metrics for the adapter.
func (prmq *RabbitMQAdapter) initMetrics(provider metric.MeterProvider, Logger libLog.Logger) {
	meter := provider.Meter("github.com/LerianStudio/fetcher/pkg/rabbitmq")

	factory, err := obsMetrics.NewMetricsFactory(meter, Logger)
	if err != nil {
		Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to initialize RabbitMQ metrics factory: %v", err))
		return
	}

	prmq.metrics = &metrics{factory: factory}
}

// recordPublishAttempt safely records a publish attempt metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordPublishAttempt(ctx context.Context, attrs ...attribute.KeyValue) {
	prmq.recordCounter(ctx, "rabbitmq.publish.attempts", "Number of publish attempts", "{attempt}", attrs...)
}

// recordPublishSuccess safely records a publish success metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordPublishSuccess(ctx context.Context, attrs ...attribute.KeyValue) {
	prmq.recordCounter(ctx, "rabbitmq.publish.successes", "Number of successful publishes", "{message}", attrs...)
}

// recordPublishFailure safely records a publish failure metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordPublishFailure(ctx context.Context, attrs ...attribute.KeyValue) {
	prmq.recordCounter(ctx, "rabbitmq.publish.failures", "Number of failed publishes", "{message}", attrs...)
}

// recordConsumeProcessed safely records a consume processed metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordConsumeProcessed(ctx context.Context, attrs ...attribute.KeyValue) {
	prmq.recordCounter(ctx, "rabbitmq.consume.processed", "Number of successfully processed messages", "{message}", attrs...)
}

// recordConsumeFailed safely records a consume failed metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordConsumeFailed(ctx context.Context, attrs ...attribute.KeyValue) {
	prmq.recordCounter(ctx, "rabbitmq.consume.failed", "Number of failed message processings", "{message}", attrs...)
}

// recordPublishLatency safely records a publish latency metric if metrics are initialized.
func (prmq *RabbitMQAdapter) recordPublishLatency(ctx context.Context, latencyMs float64, attrs ...attribute.KeyValue) {
	if prmq.metrics == nil || prmq.metrics.factory == nil {
		return
	}

	histogram, err := prmq.metrics.factory.Histogram(obsMetrics.Metric{Name: "rabbitmq.publish.latency", Description: "Publish latency in milliseconds", Unit: "ms"})
	if err != nil {
		return
	}

	_ = histogram.WithAttributes(attrs...).Record(ctx, int64(latencyMs))
}

func (prmq *RabbitMQAdapter) recordCounter(ctx context.Context, name, description, unit string, attrs ...attribute.KeyValue) {
	if prmq.metrics == nil || prmq.metrics.factory == nil {
		return
	}

	counter, err := prmq.metrics.factory.Counter(obsMetrics.Metric{Name: name, Description: description, Unit: unit})
	if err != nil {
		return
	}

	_ = counter.WithAttributes(attrs...).AddOne(ctx)
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
	if prmq.circuitBreaker == nil {
		return CircuitUnknown
	}

	return mapCircuitState(prmq.circuitBreaker.GetState(rabbitMQCircuitBreakerName))
}

// invalidateChannel safely closes and nullifies the current RabbitMQ channel.
func (prmq *RabbitMQAdapter) invalidateChannel(logger libLog.Logger) {
	prmq.mu.Lock()
	channel := prmq.channel
	prmq.channel = nil
	prmq.mu.Unlock()

	if channel != nil && !channel.IsClosed() {
		if err := channel.Close(); err != nil && logger != nil {
			logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Failed to close RabbitMQ channel: %v", err))
		}
	}
}

// startChannelWatcher monitors the RabbitMQ channel for closure events.
func (prmq *RabbitMQAdapter) startChannelWatcher(logger libLog.Logger, channel amqpChannel) {
	if channel == nil {
		return
	}

	notifications := channel.NotifyClose(make(chan *amqp.Error, 1))

	obsRuntime.SafeGo(logger, "rabbitmq-channel-watcher", obsRuntime.KeepRunning, func() {
		if err, ok := <-notifications; ok && err != nil {
			logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("RabbitMQ channel closed: %v", err))
		} else {
			logger.Log(context.Background(), libLog.LevelWarn, "RabbitMQ channel closed")
		}

		prmq.mu.Lock()
		defer prmq.mu.Unlock()

		prmq.channel = nil
	})
}

// calculateBackoff adapts Fetcher's historical 1-based attempt API to
// lib-commons/backoff. Keeping this tiny wrapper avoids churn in public tests
// while removing the previous service-local exponential/jitter implementation.
func calculateBackoff(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	if maxDelay <= 0 {
		return 0
	}

	delay := libBackoff.Exponential(baseDelay, max(attempt-1, 0))
	if delay > maxDelay {
		delay = maxDelay
	}

	return libBackoff.FullJitter(delay)
}

// FullJitter returns a random duration in [0, baseDelay) for use as a retry delay.
// This implements the "full jitter" strategy from the AWS Architecture Blog,
// which provides the best spread across retrying clients.
func FullJitter(baseDelay time.Duration) time.Duration {
	return libBackoff.FullJitter(baseDelay)
}

// NextBackoff doubles the given delay, capping at DefaultMaxRetryDelay.
// Non-positive inputs are seeded with DefaultBaseRetryDelay to prevent zero-delay retry loops.
func NextBackoff(current time.Duration) time.Duration {
	if current <= 0 {
		return DefaultBaseRetryDelay
	}

	next := current * 2
	if next > DefaultMaxRetryDelay {
		return DefaultMaxRetryDelay
	}

	return next
}

// ensureChannel checks and establishes a RabbitMQ channel if not already available.
func (prmq *RabbitMQAdapter) ensureChannel(span trace.Span, logger libLog.Logger) (amqpChannel, error) {
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

	logger.Log(context.Background(), libLog.LevelWarn, "RabbitMQ channel not initialized - attempting to connect...")

	var lastErr error

	for attempt := 1; attempt <= prmq.options.MaxRetryAttempts; attempt++ {
		ch, err := prmq.conn.EnsureChannel()
		if err == nil {
			prmq.channel = ch
			prmq.startChannelWatcher(logger, ch)

			lastErr = nil

			break
		}

		lastErr = err

		backoff := calculateBackoff(attempt, prmq.options.BaseRetryDelay, prmq.options.MaxRetryDelay)
		time.Sleep(backoff)
	}

	if prmq.channel == nil {
		if span != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to establish RabbitMQ connection", lastErr)
		}

		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to establish RabbitMQ connection: %v", lastErr))

		return nil, fmt.Errorf("rabbitmq establish connection after %d attempts: %w", prmq.options.MaxRetryAttempts, lastErr)
	}

	logger.Log(context.Background(), libLog.LevelInfo, "RabbitMQ channel re-established on existing connection")

	return prmq.channel, nil
}

func publishConfirmed(ctx context.Context, ch amqpChannel, exchange, key string, msg amqp.Publishing) error {
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("enable rabbitmq publisher confirms: %w", err)
	}

	confirmations := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	returns := ch.NotifyReturn(make(chan amqp.Return, 1))

	if err := ch.Publish(exchange, key, true, false, msg); err != nil {
		return fmt.Errorf("rabbitmq publish mandatory message: %w", err)
	}

	select {
	case returned := <-returns:
		return fmt.Errorf("rabbitmq message unroutable: reply_code=%d reply_text=%s exchange=%s routing_key=%s", returned.ReplyCode, returned.ReplyText, returned.Exchange, returned.RoutingKey)
	case confirmation := <-confirmations:
		if !confirmation.Ack {
			return fmt.Errorf("rabbitmq publisher confirmation nack: delivery_tag=%d", confirmation.DeliveryTag)
		}

		select {
		case returned := <-returns:
			return fmt.Errorf("rabbitmq message unroutable after ack: reply_code=%d reply_text=%s exchange=%s routing_key=%s", returned.ReplyCode, returned.ReplyText, returned.Exchange, returned.RoutingKey)
		default:
		}

		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ProducerDefault sends a message to the specified exchange and routing key in RabbitMQ.
func (prmq *RabbitMQAdapter) ProducerDefault(ctx context.Context, exchange, key string, queueMessage []byte, header *map[string]any) error {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	logger.Log(context.Background(), libLog.LevelInfo, "Init sent message")

	if prmq.shutdown.Load() {
		logger.Log(context.Background(), libLog.LevelInfo, "RabbitMQ adapter is shut down, cannot produce messages")
		return errors.New("rabbitmq adapter is shut down")
	}

	if prmq.CircuitBreakerState() == CircuitOpen {
		logger.Log(context.Background(), libLog.LevelWarn, "Circuit breaker is open, rejecting publish request")
		prmq.recordPublishFailure(ctx,
			attribute.String(attrExchange, exchange),
			attribute.String(attrRoutingKey, key),
			attribute.String(attrReason, attrReasonCircuitBreakerOpen),
		)

		return ErrCircuitOpen
	}

	ctx, spanProducer := tracer.Start(ctx, "adapter.rabbitmq.produce")
	defer spanProducer.End()

	spanProducer.SetAttributes(
		attribute.String(attrAppRequestRequestID, reqID),
		attribute.String(attrAppRequestExchange, exchange),
		attribute.String(attrAppRequestKey, key),
		attribute.String(attrMessagingSystem, obsConstants.DBSystemRabbitMQ),
		attribute.String(attrMessagingDestinationName, exchange),
		attribute.String(attrMessagingOperation, "publish"),
		attribute.Int64(attrMessagingMessageBodySize, int64(len(queueMessage))),
	)

	headers := map[string]any(nil)

	if header != nil {
		headers = *header
	}

	if !isNilSigner(prmq.options.Signer) && prmq.options.EnableMessageSigning {
		timestamp := time.Now().UTC().Unix()

		spanProducer.SetAttributes(
			attribute.String(attrMessagingSignatureVersion, prmq.options.Signer.SignatureVersion()),
			attribute.Int64(attrMessagingSignatureTimestamp, timestamp),
		)
	}

	publish := func(ch amqpChannel) error {
		msg := BuildSecurePublishing(ctx, reqID, exchange, key, queueMessage, headers, prmq.options.Signer, prmq.options.EnableMessageSigning)
		return publishConfirmed(ctx, ch, exchange, key, msg)
	}

	var lastErr error

	prmq.publishMu.Lock()
	defer prmq.publishMu.Unlock()

	startTime := time.Now().UTC()

	for attempt := 1; attempt <= prmq.options.MaxPublishAttempts; attempt++ {
		prmq.recordPublishAttempt(ctx,
			attribute.String(attrExchange, exchange),
			attribute.String(attrRoutingKey, key),
			attribute.Int(attrAttempt, attempt),
		)

		err := prmq.executeWithCircuitBreaker(func() error {
			channel, err := prmq.ensureChannel(spanProducer, logger)
			if err != nil {
				return fmt.Errorf("rabbitmq ensure channel for publish: %w", err)
			}

			return publish(channel)
		})
		if err == nil {
			latencyMs := float64(time.Since(startTime).Milliseconds())
			prmq.recordPublishLatency(ctx, latencyMs,
				attribute.String(attrExchange, exchange),
				attribute.String(attrRoutingKey, key),
			)
			prmq.recordPublishSuccess(ctx,
				attribute.String(attrExchange, exchange),
				attribute.String(attrRoutingKey, key),
			)
			logger.Log(context.Background(), libLog.LevelInfo, "Messages sent successfully")

			return nil
		}

		lastErr = err

		prmq.invalidateChannel(logger)

		if attempt < prmq.options.MaxPublishAttempts {
			logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Publish attempt %d/%d failed, retrying with new channel: %v", attempt, prmq.options.MaxPublishAttempts, err))
		}
	}

	prmq.recordPublishFailure(ctx,
		attribute.String(attrExchange, exchange),
		attribute.String(attrRoutingKey, key),
		attribute.String(attrReason, attrReasonMaxRetriesExceeded),
	)

	libOpentelemetry.HandleSpanError(spanProducer, "Failed to publish message to queue", lastErr)
	logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to publish message: %s", lastErr))

	return fmt.Errorf("rabbitmq publish message after %d attempts: %w", prmq.options.MaxPublishAttempts, lastErr)
}

// ConsumerLoop fetches messages from the queue, delegates processing to handler, and applies ACK/NACK.
// The handler always receives headers (may be empty map if message has no headers).
func (prmq *RabbitMQAdapter) ConsumerLoop(ctx context.Context, queue string, concurrency int, handler func(ctx context.Context, body []byte, headers map[string]any) error) error {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)
	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Starting consumer loop for queue=%s", queue))

	if concurrency < 1 {
		concurrency = 1
	}

	var consecutiveErrors int

	for {
		if ctx.Err() != nil {
			logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Context canceled while running consumer loop: %v", ctx.Err()))
			return ctx.Err()
		}

		if prmq.shutdown.Load() {
			logger.Log(context.Background(), libLog.LevelInfo, "RabbitMQ adapter is shut down, exiting consumer loop")
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
			logger.Log(context.Background(), libLog.LevelWarn, "Deliveries channel closed, attempting to reconnect")
		} else {
			logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Consumer cycle finished with error (attempt %d): %v", consecutiveErrors, cycleErr))
		}

		// Calculate backoff based on error type
		var backoff time.Duration

		switch {
		case errors.Is(cycleErr, ErrCircuitOpen):
			backoff = prmq.options.CircuitBreakerCooldown
			logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Circuit breaker is open, waiting %v before retry", backoff))
		case isPermanentAMQPError(cycleErr):
			backoff = prmq.options.ConsumerPermanentErrorDelay
			logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Permanent AMQP error detected (attempt %d), using extended backoff of %v: %v", consecutiveErrors, backoff, cycleErr))
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
		attribute.String(attrAppRequestRequestID, reqID),
		attribute.String(attrAppRequestQueue, queue),
	)
	defer span.End()

	if prmq.CircuitBreakerState() == CircuitOpen {
		logger.Log(context.Background(), libLog.LevelWarn, "Circuit breaker is open, skipping consumer cycle")
		return ErrCircuitOpen
	}

	var (
		channel    amqpChannel
		deliveries <-chan amqp.Delivery
	)

	consumerTag := fmt.Sprintf("%s-%s", queue, reqID)

	err := prmq.executeWithCircuitBreaker(func() error {
		var err error

		channel, err = prmq.ensureChannel(span, logger)
		if err != nil {
			return err
		}

		// Set QoS for fair dispatch of messages among consumers based on concurrency.
		if errCh := channel.Qos(concurrency, 0, false); errCh != nil {
			return fmt.Errorf("rabbitmq set qos: %w", errCh)
		}

		deliveries, err = channel.Consume(
			queue,
			consumerTag,
			false,
			false,
			false,
			false,
			nil,
		)

		return err
	})
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to establish RabbitMQ consumer", err)
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to establish RabbitMQ consumer: %v", err))
		prmq.invalidateChannel(logger)

		return fmt.Errorf("rabbitmq consumer setup: %w", err)
	}

	sessionErr := prmq.dispatchDeliveries(ctxSpan, logger, consumerTag, queue, channel, deliveries, concurrency, handler)
	if sessionErr == nil {
		return nil
	}

	if errors.Is(sessionErr, context.Canceled) || errors.Is(sessionErr, context.DeadlineExceeded) {
		return sessionErr
	}

	libOpentelemetry.HandleSpanError(span, "RabbitMQ consumer session exited with error", sessionErr)

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
				logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Failed to cancel RabbitMQ consumer: %v", cancelErr))
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
		id, _ := libCommons.GenerateUUIDv7()
		requestID = id.String()
	}

	requestIDStr, ok := requestID.(string)
	if !ok {
		id2, _ := libCommons.GenerateUUIDv7()
		requestIDStr = id2.String()
	}

	// Create context with request ID and logger
	logWithFields := logger.With(libLog.Field{Key: libConstants.HeaderID, Value: requestIDStr})

	msgCtx := observability.ContextWithLogger(
		observability.ContextWithHeaderID(context.Background(), requestIDStr),
		logWithFields,
	)

	// Extract trace context from message headers
	msgCtx = libOpentelemetry.ExtractTraceContextFromQueueHeaders(msgCtx, d.Headers)

	// Create tracer from context and start span
	_, msgTracer, _, _ := observability.NewTrackingFromContext(msgCtx) //nolint:dogsled // NewTrackingFromContext returns 4 values, only tracer needed here

	hctx, hspan := msgTracer.Start(msgCtx, "rabbitmq.consumer.handle_message")
	defer hspan.End()

	// Add messaging semantic conventions (M12)
	hspan.SetAttributes(
		attribute.String(attrAppRequestRequestID, requestIDStr),
		attribute.String(attrMessagingSystem, obsConstants.DBSystemRabbitMQ),
		attribute.String(attrMessagingDestinationName, queue),
		attribute.String(attrMessagingOperation, "process"),
		attribute.Int64(attrMessagingMessageBodySize, int64(len(d.Body))),
	)

	if prmq.options.EnableSignatureVerification {
		if isNilSigner(prmq.options.Signer) {
			if nackErr := d.Nack(false, false); nackErr != nil {
				logWithFields.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Failed to Nack message when signer is not configured: %v", nackErr))
			}

			libOpentelemetry.HandleSpanError(hspan, "Signature verification is enabled but signer is not configured", ErrSignatureVerifierNotConfigured)
			logWithFields.Log(context.Background(), libLog.LevelError, "Signature verification is enabled but signer is not configured")
			prmq.recordConsumeFailed(ctx,
				attribute.String(attrQueue, queue),
				attribute.String(attrReason, attrReasonSignatureVerifierMissing),
			)

			return
		}

		if err := VerifyMessageSignature(d.Body, headers, d.Exchange, d.RoutingKey, prmq.options.Signer, prmq.options.SignatureTimestampTolerance, logWithFields, hspan, prmq.options.AllowLegacyBodyOnlySignatureFallback); err != nil {
			if nackErr := d.Nack(false, false); nackErr != nil {
				logWithFields.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Failed to Nack message after signature verification failure: %v", nackErr))
			}

			libOpentelemetry.HandleSpanError(hspan, "Message signature verification failed", err)
			logWithFields.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Message signature verification failed: %v", err))
			prmq.recordConsumeFailed(ctx,
				attribute.String(attrQueue, queue),
				attribute.String(attrReason, attrReasonSignatureFailed),
			)

			return
		}
	}

	// Use the logger with fields created above (logWithFields) for all logging.
	// Nack on panic, then re-panic for lib-observability/runtime to recover and report.
	defer obsRuntime.RecoverWithPolicyAndContext(hctx, logWithFields, "rabbitmq", "rabbitmq-consumer-handle-message", obsRuntime.KeepRunning)
	defer func() {
		if r := recover(); r != nil {
			if nackErr := d.Nack(false, false); nackErr != nil {
				logWithFields.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Failed to Nack message after panic: %v", nackErr))
			}

			err := fmt.Errorf("%v", r)
			logWithFields.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Panic while processing message: %v", r))
			libOpentelemetry.HandleSpanError(hspan, "Panic while processing message", err)
			prmq.recordConsumeFailed(ctx,
				attribute.String(attrQueue, queue),
				attribute.String(attrReason, attrReasonPanic),
			)

			panic(r)
		}
	}()

	if err := handler(hctx, d.Body, headers); err != nil {
		if nackErr := d.Nack(false, false); nackErr != nil {
			logWithFields.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Failed to Nack message after handler error: %v", nackErr))
		}

		libOpentelemetry.HandleSpanError(hspan, "Handler failed to process consumed message", err)
		logWithFields.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Handler failed to process consumed message: %v", err))
		prmq.recordConsumeFailed(ctx,
			attribute.String(attrQueue, queue),
			attribute.String(attrReason, attrReasonHandlerError),
		)

		return
	}

	if err := d.Ack(false); err != nil {
		libOpentelemetry.HandleSpanError(hspan, "Failed to ACK consumed message", err)
		logWithFields.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to ACK consumed message: %v", err))
	}

	prmq.recordConsumeProcessed(ctx,
		attribute.String(attrQueue, queue),
	)
}

// Shutdown gracefully closes open channels and the underlying connection.
// It respects the context deadline or uses the configured shutdown timeout.
func (prmq *RabbitMQAdapter) Shutdown(ctx context.Context) error {
	logger := observability.NewLoggerFromContext(ctx)

	// Indicate shutdown in progress
	prmq.shutdown.Store(true)

	// Determine timeout: use context deadline if available, otherwise use configured timeout
	timeout := prmq.options.ShutdownTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	// Use a done channel to implement timeout on WaitGroup
	done := make(chan struct{})

	obsRuntime.SafeGoWithContext(ctx, logger, "rabbitmq-shutdown-wait", obsRuntime.KeepRunning, func(context.Context) {
		prmq.consumerWg.Wait()
		close(done)
	})

	// Create a timeout context for the wait operation
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-done:
		// All consumers finished
		logger.Log(context.Background(), libLog.LevelInfo, "All consumers finished processing")
	case <-timeoutCtx.Done():
		logger.Log(context.Background(), libLog.LevelWarn, "Shutdown timeout reached, some consumers may not have finished")
	case <-ctx.Done():
		logger.Log(context.Background(), libLog.LevelWarn, "Shutdown context canceled, some consumers may not have finished")
	}

	// Invalidate and close the channel
	prmq.invalidateChannel(logger)

	if err := prmq.conn.Close(); err != nil {
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to close RabbitMQ connection: %v", err))

		return fmt.Errorf("rabbitmq shutdown close connection: %w", err)
	}

	logger.Log(context.Background(), libLog.LevelInfo, "RabbitMQ repository shut down gracefully")

	return nil
}

// VerifyMessageSignature verifies the canonical Fetcher RabbitMQ security envelope.
// It accepts the current route/tenant-bound payload and a rolling body-only legacy
// payload so already-queued messages can drain without being dropped.
//
//nolint:gocyclo // header parsing plus rolling legacy compatibility are deliberately kept in one verifier.
func VerifyMessageSignature(
	body []byte,
	headers map[string]any,
	exchange string,
	routingKey string,
	signer crypto.Signer,
	tolerance time.Duration,
	logger libLog.Logger,
	span trace.Span,
	allowLegacyBodyOnlyFallback ...bool,
) error {
	if isNilSigner(signer) {
		return ErrSignatureVerifierNotConfigured
	}

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

	// Check timestamp freshness to prevent replay attacks. Stale messages use the
	// configured replay tolerance; future messages get only a small explicit clock
	// skew window so minor producer/consumer drift does not break rolling deploys.
	messageTime := time.Unix(timestamp, 0)

	age := time.Since(messageTime)
	if age < -DefaultSignatureFutureSkew {
		return fmt.Errorf("%w: message timestamp is %v in the future",
			ErrSignatureFromFuture, (-age).Round(time.Second))
	}

	if tolerance > 0 {
		if age > tolerance {
			return fmt.Errorf("%w: message is %v old, tolerance is %v",
				ErrSignatureExpired, age.Round(time.Second), tolerance)
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
	if version != signer.SignatureVersion() {
		return fmt.Errorf("%w: expected %s, got %s",
			crypto.ErrUnsupportedSignatureVersion,
			signer.SignatureVersion(),
			version,
		)
	}

	// Add signature attributes to span
	if span != nil {
		span.SetAttributes(
			attribute.String(attrMessagingSignatureVersion, version),
			attribute.Int64(attrMessagingSignatureTimestamp, timestamp),
		)
	}

	tenantID, _ := headers[HeaderTenantID].(string)
	jobID := extractJobID(body)

	// Build the signature payload and verify.
	payload := BuildMessageSignaturePayload(timestamp, version, tenantID, jobID, exchange, routingKey, body)
	if err := signer.Verify(payload, signature); err != nil {
		allowLegacy := len(allowLegacyBodyOnlyFallback) > 0 && allowLegacyBodyOnlyFallback[0]
		if !allowLegacy {
			return fmt.Errorf("%w: %v", ErrSignatureVerificationFailed, err)
		}

		legacyPayload := crypto.BuildSignaturePayload(timestamp, body)
		if legacyErr := signer.Verify(legacyPayload, signature); legacyErr == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("app.request.signature_legacy_fallback", true))
			}

			if logger != nil {
				logger.Log(context.Background(), libLog.LevelWarn, "accepted legacy body-only RabbitMQ message signature during rolling drain")
			}

			return nil
		}

		return fmt.Errorf("%w: %v", ErrSignatureVerificationFailed, err)
	}

	if logger != nil {
		logger.Log(context.Background(), libLog.LevelDebug, "Message signature verified successfully")
	}

	return nil
}
