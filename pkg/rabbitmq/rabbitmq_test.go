package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libConstants "github.com/LerianStudio/lib-commons/v2/commons/constants"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// -----------------------------------------------------------------------------
// Unit Tests: NewRabbitMQAdapter
// -----------------------------------------------------------------------------

func TestNewRabbitMQAdapter_DefaultOptions_CreatesAdapter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupAdapter   func() (*RabbitMQAdapter, *testRabbitConnection)
		wantChannel    bool
		wantShutdown   bool
		wantCircuitBrk bool
	}{
		{
			name: "creates adapter with channel",
			setupAdapter: func() (*RabbitMQAdapter, *testRabbitConnection) {
				channel := newTestAMQPChannel()
				conn := &testRabbitConnection{channel: channel}
				adapter := newTestAdapterWithChannel(conn, channel)
				return adapter, conn
			},
			wantChannel:    true,
			wantShutdown:   false,
			wantCircuitBrk: true,
		},
		{
			name: "creates adapter without channel",
			setupAdapter: func() (*RabbitMQAdapter, *testRabbitConnection) {
				conn := &testRabbitConnection{err: errors.New("connection failed")}
				adapter := newTestAdapter(conn)
				return adapter, conn
			},
			wantChannel:    false,
			wantShutdown:   false,
			wantCircuitBrk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adapter, _ := tt.setupAdapter()

			require.NotNil(t, adapter, "adapter should not be nil")
			assert.Equal(t, tt.wantShutdown, adapter.shutdown.Load(), "shutdown flag mismatch")
			assert.NotNil(t, adapter.circuitBreaker, "circuit breaker should be initialized")

			if tt.wantChannel {
				assert.NotNil(t, adapter.channel, "channel should be set")
			} else {
				assert.Nil(t, adapter.channel, "channel should not be set")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Unit Tests: IsHealthy
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_IsHealthy_ReturnsTrue_WhenChannelOpen(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	channel.closed = false
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	require.True(t, adapter.IsHealthy(), "should return true when channel is open")
}

func TestRabbitMQAdapter_IsHealthy_ReturnsFalse_WhenChannelClosed(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	channel.closed = true
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	require.False(t, adapter.IsHealthy(), "should return false when channel is closed")
}

func TestRabbitMQAdapter_IsHealthy_ReturnsFalse_WhenShutdown(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	channel.closed = false
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)
	adapter.shutdown.Store(true)

	require.False(t, adapter.IsHealthy(), "should return false when shutdown is true")
}

func TestRabbitMQAdapter_IsHealthy_ReturnsFalse_WhenChannelNil(t *testing.T) {
	t.Parallel()

	conn := &testRabbitConnection{}
	adapter := newTestAdapter(conn)
	adapter.channel = nil

	require.False(t, adapter.IsHealthy(), "should return false when channel is nil")
}

// -----------------------------------------------------------------------------
// Unit Tests: CircuitBreaker
// -----------------------------------------------------------------------------

func TestCircuitBreaker_OpensAfterThresholdFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		threshold     int
		failures      int
		expectedState CircuitState
		canExecute    bool
	}{
		{
			name:          "stays closed below threshold",
			threshold:     5,
			failures:      4,
			expectedState: CircuitClosed,
			canExecute:    true,
		},
		{
			name:          "opens at threshold",
			threshold:     5,
			failures:      5,
			expectedState: CircuitOpen,
			canExecute:    false,
		},
		{
			name:          "opens above threshold",
			threshold:     3,
			failures:      5,
			expectedState: CircuitOpen,
			canExecute:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cb := newCircuitBreaker(tt.threshold, time.Minute)

			for i := 0; i < tt.failures; i++ {
				cb.recordFailure()
			}

			assert.Equal(t, tt.expectedState, cb.State(), "circuit state mismatch")
			assert.Equal(t, tt.canExecute, cb.canExecute(), "canExecute mismatch")
		})
	}
}

func TestCircuitBreaker_HalfOpensAfterCooldown(t *testing.T) {
	t.Parallel()

	cooldown := 10 * time.Millisecond
	cb := newCircuitBreaker(1, cooldown)

	// Record failure to open the circuit
	cb.recordFailure()
	require.Equal(t, CircuitOpen, cb.State(), "circuit should be open after failure")

	// Wait for cooldown
	time.Sleep(cooldown + 5*time.Millisecond)

	// canExecute should transition to half-open
	canExec := cb.canExecute()
	require.True(t, canExec, "should allow execution after cooldown")
	assert.Equal(t, CircuitHalfOpen, cb.State(), "circuit should be half-open")
}

func TestCircuitBreaker_ClosesOnSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		initialState  func(cb *circuitBreaker)
		expectedState CircuitState
	}{
		{
			name: "closes from half-open on success",
			initialState: func(cb *circuitBreaker) {
				cb.state.Store(int32(CircuitHalfOpen))
			},
			expectedState: CircuitClosed,
		},
		{
			name: "stays closed on success",
			initialState: func(cb *circuitBreaker) {
				cb.state.Store(int32(CircuitClosed))
			},
			expectedState: CircuitClosed,
		},
		{
			name: "closes from open state on success",
			initialState: func(cb *circuitBreaker) {
				cb.state.Store(int32(CircuitOpen))
			},
			expectedState: CircuitClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cb := newCircuitBreaker(5, time.Minute)
			tt.initialState(cb)

			cb.recordSuccess()

			assert.Equal(t, tt.expectedState, cb.State(), "circuit state mismatch after success")
			assert.Equal(t, int32(0), cb.consecutiveErrors.Load(), "consecutive errors should be reset")
		})
	}
}

func TestCircuitState_String_ReturnsCorrectValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		state    CircuitState
		expected string
	}{
		{name: "closed state", state: CircuitClosed, expected: "closed"},
		{name: "open state", state: CircuitOpen, expected: "open"},
		{name: "half-open state", state: CircuitHalfOpen, expected: "half-open"},
		{name: "unknown state", state: CircuitState(99), expected: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

// -----------------------------------------------------------------------------
// Unit Tests: CalculateBackoff
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_CalculateBackoff_ExponentialGrowth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		attempt     int
		baseDelay   time.Duration
		maxDelay    time.Duration
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "first attempt returns base delay with jitter",
			attempt:     1,
			baseDelay:   100 * time.Millisecond,
			maxDelay:    2 * time.Second,
			minExpected: 100 * time.Millisecond,
			maxExpected: 125 * time.Millisecond, // base + 25% jitter
		},
		{
			name:        "second attempt doubles delay",
			attempt:     2,
			baseDelay:   100 * time.Millisecond,
			maxDelay:    2 * time.Second,
			minExpected: 200 * time.Millisecond,
			maxExpected: 250 * time.Millisecond,
		},
		{
			name:        "third attempt quadruples delay",
			attempt:     3,
			baseDelay:   100 * time.Millisecond,
			maxDelay:    2 * time.Second,
			minExpected: 400 * time.Millisecond,
			maxExpected: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := DefaultOptions()
			opts.BaseRetryDelay = tt.baseDelay
			opts.MaxRetryDelay = tt.maxDelay

			adapter := &RabbitMQAdapter{options: opts}

			backoff := adapter.calculateBackoff(tt.attempt)

			assert.GreaterOrEqual(t, backoff, tt.minExpected, "backoff should be at least base delay")
			assert.LessOrEqual(t, backoff, tt.maxExpected, "backoff should not exceed expected max with jitter")
		})
	}
}

func TestRabbitMQAdapter_CalculateBackoff_CapsAtMaxDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		attempt   int
		baseDelay time.Duration
		maxDelay  time.Duration
	}{
		{
			name:      "high attempt capped at max delay",
			attempt:   10,
			baseDelay: 100 * time.Millisecond,
			maxDelay:  500 * time.Millisecond,
		},
		{
			name:      "very high attempt still capped",
			attempt:   20,
			baseDelay: 100 * time.Millisecond,
			maxDelay:  1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := DefaultOptions()
			opts.BaseRetryDelay = tt.baseDelay
			opts.MaxRetryDelay = tt.maxDelay

			adapter := &RabbitMQAdapter{options: opts}

			backoff := adapter.calculateBackoff(tt.attempt)

			// Max expected is maxDelay + 25% jitter
			maxExpected := tt.maxDelay + tt.maxDelay/4
			assert.LessOrEqual(t, backoff, maxExpected, "backoff should be capped at max delay plus jitter")
		})
	}
}

// -----------------------------------------------------------------------------
// Unit Tests: ProducerDefault
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_ProducerDefault_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exchange string
		key      string
		body     []byte
		headers  *map[string]any
		wantErr  bool
	}{
		{
			name:     "publishes message with no headers",
			exchange: "test-exchange",
			key:      "test-key",
			body:     []byte(`{"foo":"bar"}`),
			headers:  nil,
			wantErr:  false,
		},
		{
			name:     "publishes message with custom headers",
			exchange: "test-exchange",
			key:      "test-key",
			body:     []byte(`{"data":"value"}`),
			headers:  &map[string]any{"custom": "header"},
			wantErr:  false,
		},
		{
			name:     "publishes empty body",
			exchange: "another-exchange",
			key:      "another-key",
			body:     []byte(`{}`),
			headers:  nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			channel := newTestAMQPChannel()
			conn := &testRabbitConnection{channel: channel}
			adapter := newTestAdapter(conn)

			t.Cleanup(func() {
				_ = adapter.Shutdown(testContextWithHeader("cleanup"))
			})

			ctx := testContextWithHeader("req-123")
			err := adapter.ProducerDefault(ctx, tt.exchange, tt.key, tt.body, tt.headers)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, channel.published, 1, "expected one published message")

			record := channel.published[0]
			assert.Equal(t, tt.exchange, record.exchange)
			assert.Equal(t, tt.key, record.key)
			assert.Equal(t, string(tt.body), string(record.message.Body))
			assert.Equal(t, "application/json", record.message.ContentType)

			headerID, _ := record.message.Headers[libConstants.HeaderID].(string)
			assert.Equal(t, "req-123", headerID)
		})
	}
}

func TestRabbitMQAdapter_ProducerDefault_RetriesOnFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setupChannels     func() ([]amqpChannel, *testRabbitConnection)
		expectedConnCalls int
		expectedPublished int
		wantErr           bool
	}{
		{
			name: "retries with new channel on publish failure",
			setupChannels: func() ([]amqpChannel, *testRabbitConnection) {
				first := newTestAMQPChannel()
				first.publishErr = errors.New("publish failed")

				second := newTestAMQPChannel()

				conn := &testRabbitConnection{
					channels: []amqpChannel{first, second},
				}
				return []amqpChannel{first, second}, conn
			},
			expectedConnCalls: 2,
			expectedPublished: 1,
			wantErr:           false,
		},
		{
			name: "succeeds on first attempt",
			setupChannels: func() ([]amqpChannel, *testRabbitConnection) {
				channel := newTestAMQPChannel()
				conn := &testRabbitConnection{channel: channel}
				return []amqpChannel{channel}, conn
			},
			expectedConnCalls: 1,
			expectedPublished: 1,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			channels, conn := tt.setupChannels()
			adapter := newTestAdapter(conn)

			t.Cleanup(func() {
				_ = adapter.Shutdown(testContextWithHeader("cleanup"))
			})

			ctx := testContextWithHeader("req-retry")
			err := adapter.ProducerDefault(ctx, "ex", "rk", []byte(`{"hello":"world"}`), nil)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedConnCalls, conn.calls, "connection calls mismatch")

			// Count total published across all channels
			totalPublished := 0
			for _, ch := range channels {
				if tch, ok := ch.(*testAMQPChannel); ok {
					totalPublished += len(tch.published)
				}
			}
			assert.Equal(t, tt.expectedPublished, totalPublished, "published count mismatch")
		})
	}
}

func TestRabbitMQAdapter_ProducerDefault_ReturnsError_WhenEnsureChannelFails(t *testing.T) {
	t.Parallel()

	conn := &testRabbitConnection{err: errors.New("ensure failed")}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	ctx := testContextWithHeader("req-err")
	err := adapter.ProducerDefault(ctx, "ex", "key", []byte(`{}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ensure channel")
}

func TestRabbitMQAdapter_ProducerDefault_ReturnsError_WhenShutdown(t *testing.T) {
	t.Parallel()

	conn := &testRabbitConnection{}
	adapter := newTestAdapter(conn)
	adapter.shutdown.Store(true)

	ctx := testContextWithHeader("req-shutdown")
	err := adapter.ProducerDefault(ctx, "ex", "key", []byte(`{}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "shut down")
}

func TestRabbitMQAdapter_ProducerDefault_ReturnsError_WhenCircuitOpen(t *testing.T) {
	t.Parallel()

	conn := &testRabbitConnection{channel: newTestAMQPChannel()}
	adapter := newTestAdapter(conn)

	// Open the circuit breaker by recording failures
	for i := 0; i < adapter.options.CircuitBreakerThreshold; i++ {
		adapter.circuitBreaker.recordFailure()
	}

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	ctx := testContextWithHeader("req-circuit")
	err := adapter.ProducerDefault(ctx, "ex", "key", []byte(`{}`), nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

// -----------------------------------------------------------------------------
// Unit Tests: ConsumerLoop
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_ConsumerLoop_AcksOnSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messageBody string
		handlerErr  error
		expectAcks  int
		expectNacks int
	}{
		{
			name:        "acks message on handler success",
			messageBody: `{"payload":"ok"}`,
			handlerErr:  nil,
			expectAcks:  1,
			expectNacks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(testContextWithHeader("req-ack"))
			defer cancel()

			channel := newTestAMQPChannel()
			ack := &testAcknowledger{}
			channel.deliveries <- amqp.Delivery{
				Body:         []byte(tt.messageBody),
				Acknowledger: ack,
			}

			conn := &testRabbitConnection{channel: channel}
			adapter := newTestAdapter(conn)

			t.Cleanup(func() {
				_ = adapter.Shutdown(testContextWithHeader("cleanup"))
			})

			handled := make(chan []byte, 1)
			handler := func(ctx context.Context, body []byte, headers map[string]any) error {
				handled <- body
				cancel()
				return tt.handlerErr
			}

			err := adapter.ConsumerLoop(ctx, "queue-ack", 1, handler)
			require.NoError(t, err)

			select {
			case body := <-handled:
				assert.Equal(t, tt.messageBody, string(body))
			case <-time.After(time.Second):
				t.Fatal("handler was not invoked")
			}

			require.Eventually(t, func() bool {
				return ack.acks == tt.expectAcks && ack.nacks == tt.expectNacks
			}, time.Second, 10*time.Millisecond)
		})
	}
}

func TestRabbitMQAdapter_ConsumerLoop_NacksOnHandlerError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(testContextWithHeader("req-nack"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body:         []byte(`{"payload":"fail"}`),
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	processed := make(chan struct{})
	handler := func(context.Context, []byte, map[string]any) error {
		close(processed)
		return errors.New("handler failed")
	}

	go func() {
		<-processed
		cancel()
	}()

	err := adapter.ConsumerLoop(ctx, "queue-nack", 1, handler)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return ack.nacks == 1 && ack.acks == 0
	}, time.Second, 10*time.Millisecond)
}

func TestRabbitMQAdapter_ConsumerLoop_ReturnsError_WhenShutdown(t *testing.T) {
	t.Parallel()

	conn := &testRabbitConnection{channel: newTestAMQPChannel()}
	adapter := newTestAdapter(conn)
	adapter.shutdown.Store(true)

	ctx := testContextWithHeader("req-shutdown")
	handler := func(context.Context, []byte, map[string]any) error {
		return nil
	}

	err := adapter.ConsumerLoop(ctx, "queue", 1, handler)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "shut down")
}

// -----------------------------------------------------------------------------
// Unit Tests: Shutdown
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_Shutdown_ClosesResources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setupAdapter      func() (*RabbitMQAdapter, *testAMQPChannel, *testRabbitConnection)
		expectedShutdown  bool
		expectedConnClose int
		expectedChClose   int
	}{
		{
			name: "closes channel and connection",
			setupAdapter: func() (*RabbitMQAdapter, *testAMQPChannel, *testRabbitConnection) {
				channel := newTestAMQPChannel()
				conn := &testRabbitConnection{channel: channel}
				adapter := newTestAdapterWithChannel(conn, channel)
				return adapter, channel, conn
			},
			expectedShutdown:  true,
			expectedConnClose: 1,
			expectedChClose:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adapter, channel, conn := tt.setupAdapter()

			ctx := testContextWithHeader("req-shutdown")
			err := adapter.Shutdown(ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedShutdown, adapter.shutdown.Load(), "shutdown flag mismatch")
			assert.Nil(t, adapter.channel, "channel should be cleared")
			assert.Equal(t, tt.expectedConnClose, conn.closeCalls, "connection close calls mismatch")
			assert.Equal(t, tt.expectedChClose, channel.closeCalls, "channel close calls mismatch")
		})
	}
}

func TestRabbitMQAdapter_Shutdown_HandlesAlreadyShutdown(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)

	// First shutdown
	ctx := testContextWithHeader("req-shutdown-1")
	err := adapter.Shutdown(ctx)
	require.NoError(t, err)

	// Second shutdown should also succeed (idempotent)
	ctx2 := testContextWithHeader("req-shutdown-2")
	err = adapter.Shutdown(ctx2)
	require.NoError(t, err)
}

// -----------------------------------------------------------------------------
// Unit Tests: DefaultOptions
// -----------------------------------------------------------------------------

func TestDefaultOptions_ReturnsExpectedValues(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()

	assert.Equal(t, DefaultMaxRetryAttempts, opts.MaxRetryAttempts)
	assert.Equal(t, DefaultMaxPublishAttempts, opts.MaxPublishAttempts)
	assert.Equal(t, DefaultBaseRetryDelay, opts.BaseRetryDelay)
	assert.Equal(t, DefaultMaxRetryDelay, opts.MaxRetryDelay)
	assert.Equal(t, DefaultConsumerReconnectDelay, opts.ConsumerReconnectDelay)
	assert.Equal(t, DefaultCircuitBreakerThreshold, opts.CircuitBreakerThreshold)
	assert.Equal(t, DefaultCircuitBreakerCooldown, opts.CircuitBreakerCooldown)
	assert.Equal(t, DefaultShutdownTimeout, opts.ShutdownTimeout)
	assert.Nil(t, opts.MeterProvider)
}

// -----------------------------------------------------------------------------
// Helpers/Mocks
// -----------------------------------------------------------------------------

type testRabbitConnection struct {
	channel    amqpChannel
	channels   []amqpChannel
	err        error
	calls      int
	closeCalls int
	closeErr   error
}

func (t *testRabbitConnection) EnsureChannel() (amqpChannel, error) {
	t.calls++
	if t.err != nil {
		return nil, t.err
	}

	if len(t.channels) > 0 {
		idx := t.calls - 1
		if idx >= len(t.channels) {
			idx = len(t.channels) - 1
		}
		ch := t.channels[idx]
		if tch, ok := ch.(*testAMQPChannel); ok {
			tch.closed = false
		}
		t.channel = ch
		return ch, nil
	}

	if t.channel == nil {
		t.channel = newTestAMQPChannel()
	}

	if ch, ok := t.channel.(*testAMQPChannel); ok {
		ch.closed = false
	}

	return t.channel, nil
}

func (t *testRabbitConnection) Close() error {
	t.closeCalls++
	return t.closeErr
}

type publishedRecord struct {
	exchange  string
	key       string
	mandatory bool
	immediate bool
	message   amqp.Publishing
}

type testAMQPChannel struct {
	publishErr      error
	publishErrs     []error
	publishAttempts int
	consumeErr      error
	qosErr          error

	deliveries chan amqp.Delivery
	published  []publishedRecord

	consumeQueue   string
	consumeAutoAck bool

	cancelCalls     int
	cancelConsumer  string
	cancelNoWait    bool
	closeCalls      int
	closeShouldFail bool

	notifyClose chan *amqp.Error
	closed      bool
}

func newTestAMQPChannel() *testAMQPChannel {
	return &testAMQPChannel{
		deliveries: make(chan amqp.Delivery, 1),
	}
}

func (t *testAMQPChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	t.publishAttempts++

	var err error
	if len(t.publishErrs) > 0 {
		err = t.publishErrs[0]
		t.publishErrs = t.publishErrs[1:]
	} else {
		err = t.publishErr
	}

	if err != nil {
		return err
	}

	t.published = append(t.published, publishedRecord{
		exchange:  exchange,
		key:       key,
		mandatory: mandatory,
		immediate: immediate,
		message:   msg,
	})

	return nil
}

func (t *testAMQPChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if t.consumeErr != nil {
		return nil, t.consumeErr
	}

	t.consumeQueue = queue
	t.consumeAutoAck = autoAck
	t.cancelConsumer = consumer

	if t.deliveries == nil {
		t.deliveries = make(chan amqp.Delivery, 1)
	}

	return t.deliveries, nil
}

func (t *testAMQPChannel) IsClosed() bool {
	return t.closed
}

func (t *testAMQPChannel) Cancel(consumer string, noWait bool) error {
	t.cancelCalls++
	t.cancelConsumer = consumer
	t.cancelNoWait = noWait
	return nil
}

func (t *testAMQPChannel) Close() error {
	t.closeCalls++
	if t.closeShouldFail {
		return errors.New("close fail")
	}

	t.closed = true
	return nil
}

func (t *testAMQPChannel) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	t.notifyClose = receiver
	return receiver
}

func (t *testAMQPChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	if t.qosErr != nil {
		return t.qosErr
	}

	return nil
}

type testAcknowledger struct {
	acks  int
	nacks int
}

// newTestAdapter creates a properly initialized RabbitMQAdapter for testing.
func newTestAdapter(conn rabbitConnection) *RabbitMQAdapter {
	opts := DefaultOptions()
	return &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newCircuitBreaker(opts.CircuitBreakerThreshold, opts.CircuitBreakerCooldown),
	}
}

// newTestAdapterWithChannel creates a properly initialized RabbitMQAdapter with a channel for testing.
func newTestAdapterWithChannel(conn rabbitConnection, channel amqpChannel) *RabbitMQAdapter {
	opts := DefaultOptions()
	return &RabbitMQAdapter{
		conn:           conn,
		channel:        channel,
		options:        opts,
		circuitBreaker: newCircuitBreaker(opts.CircuitBreakerThreshold, opts.CircuitBreakerCooldown),
	}
}

func (t *testAcknowledger) Ack(uint64, bool) error {
	t.acks++
	return nil
}

func (t *testAcknowledger) Nack(uint64, bool, bool) error {
	t.nacks++
	return nil
}

func (t *testAcknowledger) Reject(uint64, bool) error {
	t.nacks++
	return nil
}

func testContextWithHeader(header string) context.Context {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: header,
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}

	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}
