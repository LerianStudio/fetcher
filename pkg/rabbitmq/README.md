# RabbitMQ Adapter Package

Resilient RabbitMQ adapter for producing and consuming messages with built-in circuit breaker, retry mechanisms, message signing, and comprehensive observability.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Circuit Breaker](#circuit-breaker)
- [Message Signing](#message-signing)
- [Observability](#observability)
- [Error Handling](#error-handling)
- [Graceful Shutdown](#graceful-shutdown)
- [Testing](#testing)
- [Architecture](#architecture)
- [Best Practices](#best-practices)

## Overview

This package provides a production-ready RabbitMQ adapter that handles:

- **Resilience**: Circuit breaker pattern, exponential backoff with jitter, automatic reconnection
- **Security**: HMAC-SHA256 message signing with replay attack prevention
- **Observability**: OpenTelemetry metrics and distributed tracing
- **Concurrency**: Worker pool for parallel message processing
- **Reliability**: Proper ACK/NACK handling, persistent messages, graceful shutdown

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        RabbitMQAdapter                              │
├─────────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │   Producer   │  │   Consumer   │  │   Shutdown   │              │
│  │   Default    │  │    Loop      │  │   Handler    │              │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘              │
│         │                 │                                         │
│         ▼                 ▼                                         │
│  ┌──────────────────────────────────────┐                          │
│  │         Circuit Breaker              │                          │
│  │   (Closed → Open → Half-Open)        │                          │
│  └──────────────┬───────────────────────┘                          │
│                 │                                                   │
│  ┌──────────────▼───────────────────────┐                          │
│  │      Channel Manager (with mutex)     │                          │
│  │   - Lazy initialization               │                          │
│  │   - Auto-reconnection                 │                          │
│  │   - Exponential backoff               │                          │
│  └──────────────┬───────────────────────┘                          │
│                 │                                                   │
│  ┌──────────────▼───────────────────────┐                          │
│  │         Message Signing               │                          │
│  │   - HMAC-SHA256                       │                          │
│  │   - Timestamp validation              │                          │
│  │   - Version checking                  │                          │
│  └──────────────────────────────────────┘                          │
│                                                                     │
│  ┌──────────────────────────────────────┐                          │
│  │         OpenTelemetry                 │                          │
│  │   - Metrics (counters, histograms)    │                          │
│  │   - Distributed tracing               │                          │
│  │   - Context propagation               │                          │
│  └──────────────────────────────────────┘                          │
└─────────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Producer Example

```go
package main

import (
    "context"
    "encoding/json"

    libRabbitmq "github.com/LerianStudio/lib-commons/v2/commons/rabbitmq"
    "github.com/LerianStudio/fetcher/pkg/rabbitmq"
)

func main() {
    // Create RabbitMQ connection (from lib-commons)
    conn := &libRabbitmq.RabbitMQConnection{
        // Configure connection...
    }

    // Create adapter with default options
    adapter := rabbitmq.NewRabbitMQAdapter(conn)

    // Prepare message
    msg := map[string]string{"event": "user.created", "user_id": "123"}
    body, _ := json.Marshal(msg)

    // Optional custom headers
    headers := map[string]any{
        "x-priority": "high",
    }

    // Publish message
    ctx := context.Background()
    err := adapter.ProducerDefault(ctx, "exchange", "routing.key", body, &headers)
    if err != nil {
        // Handle error
    }
}
```

### Consumer Example

```go
package main

import (
    "context"
    "encoding/json"
    "sync"

    libRabbitmq "github.com/LerianStudio/lib-commons/v2/commons/rabbitmq"
    "github.com/LerianStudio/fetcher/pkg/rabbitmq"
)

func main() {
    conn := &libRabbitmq.RabbitMQConnection{
        // Configure connection...
    }

    adapter := rabbitmq.NewRabbitMQAdapter(conn)

    // Define message handler
    handler := func(ctx context.Context, body []byte, headers map[string]any) error {
        var msg map[string]string
        if err := json.Unmarshal(body, &msg); err != nil {
            return err // NACK - message will be requeued
        }

        // Process message...

        return nil // ACK - message processed successfully
    }

    // Start consumer loop with 5 concurrent workers
    ctx := context.Background()
    concurrency := 5
    err := adapter.ConsumerLoop(ctx, "queue-name", concurrency, handler)
    if err != nil {
        // Handle error
    }
}
```

### With Custom Options

```go
opts := rabbitmq.DefaultOptions()
opts.MaxRetryAttempts = 5
opts.CircuitBreakerThreshold = 10
opts.ShutdownTimeout = 60 * time.Second

// With message signing
opts.Signer = crypto.NewHMACSigner([]byte("your-32-byte-secret-key-here!!!"))
opts.EnableMessageSigning = true
opts.EnableSignatureVerification = true

adapter := rabbitmq.NewRabbitMQAdapterWithOptions(conn, opts)
```

## Configuration

### AdapterOptions

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `MaxRetryAttempts` | `int` | `3` | Max retry attempts for channel establishment |
| `MaxPublishAttempts` | `int` | `2` | Max retry attempts for publishing a message |
| `BaseRetryDelay` | `time.Duration` | `100ms` | Base delay for exponential backoff |
| `MaxRetryDelay` | `time.Duration` | `2s` | Maximum delay for exponential backoff |
| `ConsumerReconnectDelay` | `time.Duration` | `500ms` | Delay between consumer reconnection attempts |
| `CircuitBreakerThreshold` | `int` | `5` | Consecutive failures before opening circuit |
| `CircuitBreakerCooldown` | `time.Duration` | `30s` | Cooldown before transitioning to half-open |
| `ShutdownTimeout` | `time.Duration` | `30s` | Timeout for graceful shutdown |
| `MeterProvider` | `metric.MeterProvider` | `nil` | OpenTelemetry meter provider for metrics |
| `Signer` | `crypto.Signer` | `nil` | Signer for message signing/verification |
| `EnableMessageSigning` | `bool` | `true`* | Enable message signing on publish |
| `EnableSignatureVerification` | `bool` | `true`* | Enable signature verification on consume |
| `SignatureTimestampTolerance` | `time.Duration` | `5min` | Max age of message signature (replay protection) |

*\* Defaults to `true` only if `Signer` is provided*

### Default Options Function

```go
opts := rabbitmq.DefaultOptions()
```

Returns a fully configured `AdapterOptions` with all defaults set.

## Circuit Breaker

The circuit breaker prevents cascading failures by temporarily rejecting requests when the system is unhealthy.

### States

| State | Value | Behavior |
|-------|-------|----------|
| `CircuitClosed` | `0` | Normal operation, requests flow through |
| `CircuitOpen` | `1` | Requests immediately rejected with `ErrCircuitOpen` |
| `CircuitHalfOpen` | `2` | Testing recovery, allowing limited requests |

### State Transitions

```
                 ┌─────────────────┐
                 │                 │
    Success      │   CircuitClosed │◄──────────────┐
    ─────────────│                 │               │
                 └────────┬────────┘               │
                          │                        │
                          │ N consecutive          │ Success
                          │ failures               │
                          ▼                        │
                 ┌─────────────────┐               │
                 │                 │               │
                 │   CircuitOpen   │               │
                 │                 │               │
                 └────────┬────────┘               │
                          │                        │
                          │ After cooldown         │
                          │                        │
                          ▼                        │
                 ┌─────────────────┐               │
                 │                 │───────────────┘
                 │ CircuitHalfOpen │
                 │                 │───────────────┐
                 └─────────────────┘               │
                                                   │
                                          Failure  │
                                          ─────────▼
                                          Back to CircuitOpen
```

### Monitoring

```go
// Check current state
state := adapter.CircuitBreakerState()
fmt.Println(state.String()) // "closed", "open", or "half-open"

// Use in health checks
if !adapter.IsHealthy() {
    // Connection is unhealthy or shutting down
}
```

## Message Signing

HMAC-SHA256 message signing provides integrity verification and replay attack prevention.

### Headers

| Header | Constant | Description |
|--------|----------|-------------|
| `x-message-signature` | `constant.HeaderMessageSignature` | HMAC-SHA256 signature (hex encoded) |
| `t` | `constant.HeaderSignatureTimestamp` | Unix timestamp of signing |
| `signature-version` | `constant.HeaderSignatureVersion` | Signature algorithm version |

### Signature Payload Format

```
{timestamp}.{message_body}
```

### Producer Flow

```
1. Generate current Unix timestamp
2. Build payload: timestamp + "." + body
3. Calculate HMAC-SHA256 signature
4. Add headers to message:
   - x-message-signature: <hex_signature>
   - t: <timestamp>
   - signature-version: v1
5. Publish to RabbitMQ
```

### Consumer Flow

```
1. Extract signature headers from message
2. Validate all required headers present
3. Check timestamp freshness (reject if > tolerance)
4. Verify signature version matches
5. Rebuild payload and verify signature
6. If valid: call handler
   If invalid: NACK without calling handler
```

### Replay Attack Prevention

Messages older than `SignatureTimestampTolerance` (default: 5 minutes) are automatically rejected:

```go
opts.SignatureTimestampTolerance = 10 * time.Minute // Increase tolerance
opts.SignatureTimestampTolerance = 0                // Disable timestamp check
```

### Disabling Signing

```go
// Disable signing (producer)
opts.EnableMessageSigning = false

// Disable verification (consumer)
opts.EnableSignatureVerification = false

// Remove signer entirely
opts.Signer = nil
```

## Observability

### OpenTelemetry Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `rabbitmq.publish.attempts` | Counter | `{attempt}` | Number of publish attempts |
| `rabbitmq.publish.successes` | Counter | `{message}` | Number of successful publishes |
| `rabbitmq.publish.failures` | Counter | `{message}` | Number of failed publishes |
| `rabbitmq.publish.latency` | Histogram | `ms` | Publish latency in milliseconds |
| `rabbitmq.consume.processed` | Counter | `{message}` | Successfully processed messages |
| `rabbitmq.consume.failed` | Counter | `{message}` | Failed message processings |
| `rabbitmq.circuit_breaker.state` | ObservableGauge | - | Circuit breaker state (0/1/2) |

### Metric Attributes

**Publish Metrics:**
- `exchange`: Exchange name
- `routing_key`: Routing key
- `attempt`: Attempt number (1, 2, ...)
- `reason`: Failure reason (`circuit_breaker_open`, `max_retries_exceeded`)

**Consume Metrics:**
- `queue`: Queue name
- `reason`: Failure reason (`signature_verification_failed`, `handler_error`, `panic`)

### Enabling Metrics

```go
import "go.opentelemetry.io/otel/metric"

opts := rabbitmq.DefaultOptions()
opts.MeterProvider = otel.GetMeterProvider() // Or your custom provider

adapter := rabbitmq.NewRabbitMQAdapterWithOptions(conn, opts)
```

### Distributed Tracing

**Producer Spans:**
- `rabbitmq.producer.publish_message`
  - Attributes: request_id, exchange, key, messaging.system, messaging.destination.name, messaging.operation, messaging.message.body.size

**Consumer Spans:**
- `rabbitmq.consumer.connection_cycle` - Connection/channel lifecycle
- `rabbitmq.consumer.handle_message` - Individual message processing
  - Attributes: request_id, messaging.system, messaging.destination.name, messaging.operation, messaging.message.body.size

**Trace Context Propagation:**
- Producer injects trace context into message headers
- Consumer extracts trace context from headers
- Enables end-to-end distributed tracing across services

## Error Handling

### Error Types

| Error | Description |
|-------|-------------|
| `ErrCircuitOpen` | Circuit breaker is open, request rejected |
| `ErrSignatureVerificationFailed` | Message signature doesn't match |
| `ErrMissingSignatureHeaders` | Required signature headers not present |
| `ErrSignatureExpired` | Message timestamp exceeds tolerance |

### Handler Error Behavior

| Handler Result | Action | Message Fate |
|----------------|--------|--------------|
| `return nil` | ACK | Removed from queue |
| `return error` | NACK (no requeue) | Sent to DLQ if configured |
| `panic` | Recovered + NACK | Sent to DLQ if configured |

### Retry Behavior

**Channel Establishment:**
- Up to `MaxRetryAttempts` (default: 3)
- Exponential backoff with 0-25% jitter
- Circuit breaker tracks failures

**Message Publishing:**
- Up to `MaxPublishAttempts` (default: 2)
- Channel invalidated and recreated on failure
- Circuit breaker tracks failures

**Consumer Reconnection:**
- Continuous retry loop
- Fixed delay: `ConsumerReconnectDelay` (default: 500ms)
- Exits on context cancellation or shutdown

## Graceful Shutdown

### Shutdown Sequence

```
1. Set shutdown flag (rejects new publishes)
2. Wait for active consumer goroutines
   - Timeout: context deadline or ShutdownTimeout
3. Cancel consumer subscriptions
4. Close AMQP channel
5. Close AMQP connection
6. Return any errors
```

### Usage

```go
// With context timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := adapter.Shutdown(ctx); err != nil {
    log.Printf("Shutdown error: %v", err)
}
```

### Health Check

```go
// For Kubernetes probes
func healthHandler(w http.ResponseWriter, r *http.Request) {
    if adapter.IsHealthy() {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
}
```

## Testing

### Using the Mock

A mock is auto-generated via `mockgen`:

```go
//go:generate mockgen --destination=rabbitmq.mock.go --package=rabbitmq . Adapter
```

### Test Example

```go
import (
    "testing"
    "go.uber.org/mock/gomock"
    "github.com/LerianStudio/fetcher/pkg/rabbitmq"
)

func TestMyService(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockAdapter := rabbitmq.NewMockAdapter(ctrl)

    // Setup expectations
    mockAdapter.EXPECT().
        ProducerDefault(gomock.Any(), "exchange", "key", gomock.Any(), gomock.Any()).
        Return(nil)

    mockAdapter.EXPECT().
        IsHealthy().
        Return(true)

    // Use mock in tests
    myService := NewMyService(mockAdapter)
    // ...
}
```

### Running Package Tests

```bash
# Unit tests
go test ./pkg/rabbitmq/...

# With coverage
go test -cover ./pkg/rabbitmq/...

# Integration tests (requires RabbitMQ)
go test -tags=integration ./pkg/rabbitmq/...
```

## Architecture

### Internal Components

```go
// Main adapter struct
type RabbitMQAdapter struct {
    conn           rabbitConnection  // Connection wrapper
    channel        amqpChannel       // Current AMQP channel
    options        AdapterOptions    // Configuration
    mu             sync.Mutex        // Channel access mutex
    publishMu      sync.Mutex        // Publish serialization
    shutdown       atomic.Bool       // Shutdown flag
    consumerWg     sync.WaitGroup    // Active consumer tracking
    circuitBreaker *circuitBreaker   // Failure protection
    metrics        *metrics          // OpenTelemetry metrics
}
```

### Interfaces

```go
// Public adapter interface
type Adapter interface {
    ProducerDefault(ctx context.Context, exchange, key string, queueMessage []byte, header *map[string]any) error
    ConsumerLoop(ctx context.Context, queue string, concurrency int, handler func(ctx context.Context, body []byte, headers map[string]any) error) error
    Shutdown(ctx context.Context) error
    IsHealthy() bool
    CircuitBreakerState() CircuitState
}

// Internal connection interface (for testing)
type rabbitConnection interface {
    EnsureChannel() (amqpChannel, error)
    Close() error
}

// Internal channel interface (for testing)
type amqpChannel interface {
    Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
    Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
    Cancel(consumer string, noWait bool) error
    Close() error
    NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
    IsClosed() bool
    Qos(prefetchCount, prefetchSize int, global bool) error
}
```

### Thread Safety

| Component | Protection | Notes |
|-----------|------------|-------|
| Channel access | `mu sync.Mutex` | Read/write of channel reference |
| Publishing | `publishMu sync.Mutex` | Serializes publish operations |
| Shutdown flag | `atomic.Bool` | Lock-free read/write |
| Consumer tracking | `sync.WaitGroup` | Goroutine coordination |
| Circuit breaker | `atomic.Int32/Int64` | Lock-free state management |

## Best Practices

### Concurrency Tuning

```go
// Match concurrency to your workload
concurrency := runtime.NumCPU() * 2 // CPU-bound processing
concurrency := 10                    // I/O-bound processing (DB, API calls)
concurrency := 1                     // Ordered processing required
```

### Circuit Breaker Configuration

```go
// High-availability service (fail fast)
opts.CircuitBreakerThreshold = 3
opts.CircuitBreakerCooldown = 10 * time.Second

// Batch processing (tolerate more failures)
opts.CircuitBreakerThreshold = 10
opts.CircuitBreakerCooldown = 60 * time.Second
```

### Message Signing

- **Always enable** in production for inter-service communication
- Use a **secure key** (minimum 32 bytes for HMAC-SHA256)
- **Rotate keys** periodically using signature version
- Set appropriate **timestamp tolerance** based on clock drift

### Error Handling in Handlers

```go
handler := func(ctx context.Context, body []byte, headers map[string]any) error {
    // Transient error - will be retried
    if isTransientError(err) {
        return err
    }

    // Permanent error - log and acknowledge to prevent infinite retry
    if isPermanentError(err) {
        logger.Error("Permanent error, discarding message", "error", err)
        return nil // ACK to remove from queue
    }

    return nil
}
```

### Graceful Shutdown

```go
// Register shutdown handler
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-sigChan
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := adapter.Shutdown(ctx); err != nil {
        log.Printf("Shutdown error: %v", err)
    }
}()
```

### Observability

- **Always provide** `MeterProvider` in production
- Monitor `rabbitmq.circuit_breaker.state` for alerts
- Set up dashboards for publish/consume rates and latencies
- Configure alerts for high failure rates

## Related Files

| File | Description |
|------|-------------|
| [`rabbitmq.go`](./rabbitmq.go) | Main implementation |
| [`rabbitmq_test.go`](./rabbitmq_test.go) | Unit tests |
| [`rabbitmq_integration_test.go`](./rabbitmq_integration_test.go) | Integration tests |
| [`rabbitmq.mock.go`](./rabbitmq.mock.go) | Generated mock |
| [`../crypto/signer.go`](../crypto/signer.go) | Signer interface |
| [`../constant/rabbitmq.go`](../constant/rabbitmq.go) | Header constants |
