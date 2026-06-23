package rabbitmq

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	libCircuitBreaker "github.com/LerianStudio/lib-commons/v5/commons/circuitbreaker"
	libConstants "github.com/LerianStudio/lib-commons/v5/commons/constants"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.uber.org/mock/gomock"
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

			adapter := newTestAdapter(&testRabbitConnection{})
			adapter.options.CircuitBreakerThreshold = tt.threshold
			adapter.circuitBreaker = newTestCircuitBreakerManager(adapter.options)

			for i := 0; i < tt.failures; i++ {
				err := adapter.executeWithCircuitBreaker(func() error { return errors.New("boom") })
				require.Error(t, err)
			}

			assert.Equal(t, tt.expectedState, adapter.CircuitBreakerState(), "circuit state mismatch")
			assert.Equal(t, tt.canExecute, adapter.CircuitBreakerState() != CircuitOpen, "execute eligibility mismatch")
		})
	}
}

func TestCircuitBreaker_HalfOpensAfterCooldown(t *testing.T) {
	t.Parallel()

	cooldown := 10 * time.Millisecond
	adapter := newTestAdapter(&testRabbitConnection{})
	adapter.options.CircuitBreakerThreshold = 1
	adapter.options.CircuitBreakerCooldown = cooldown
	adapter.circuitBreaker = newTestCircuitBreakerManager(adapter.options)

	err := adapter.executeWithCircuitBreaker(func() error { return errors.New("boom") })
	require.Error(t, err)
	require.Equal(t, CircuitOpen, adapter.CircuitBreakerState(), "circuit should be open after failure")

	time.Sleep(cooldown + 5*time.Millisecond)

	assert.Equal(t, CircuitHalfOpen, adapter.CircuitBreakerState(), "lib-commons breaker should half-open after cooldown")
}

func TestCircuitBreaker_ClosesOnSuccess(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter(&testRabbitConnection{})
	require.NoError(t, adapter.executeWithCircuitBreaker(func() error { return nil }))

	assert.Equal(t, CircuitClosed, adapter.CircuitBreakerState(), "lib-commons breaker should remain closed on success")
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
		{name: "unknown state", state: CircuitUnknown, expected: "unknown"},
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
		name      string
		attempt   int
		baseDelay time.Duration
		maxDelay  time.Duration
		maxBound  time.Duration
	}{
		{
			name:      "first attempt uses base delay as full-jitter ceiling",
			attempt:   1,
			baseDelay: 100 * time.Millisecond,
			maxDelay:  2 * time.Second,
			maxBound:  100 * time.Millisecond,
		},
		{
			name:      "second attempt doubles full-jitter ceiling",
			attempt:   2,
			baseDelay: 100 * time.Millisecond,
			maxDelay:  2 * time.Second,
			maxBound:  200 * time.Millisecond,
		},
		{
			name:      "third attempt quadruples full-jitter ceiling",
			attempt:   3,
			baseDelay: 100 * time.Millisecond,
			maxDelay:  2 * time.Second,
			maxBound:  400 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := DefaultOptions()
			opts.BaseRetryDelay = tt.baseDelay
			opts.MaxRetryDelay = tt.maxDelay

			backoff := calculateBackoff(tt.attempt, opts.BaseRetryDelay, opts.MaxRetryDelay)

			assert.GreaterOrEqual(t, backoff, time.Duration(0), "full jitter must never be negative")
			assert.Less(t, backoff, tt.maxBound, "lib-commons full jitter should stay below the exponential ceiling")
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

			backoff := calculateBackoff(tt.attempt, opts.BaseRetryDelay, opts.MaxRetryDelay)

			assert.GreaterOrEqual(t, backoff, time.Duration(0), "full jitter must never be negative")
			assert.Less(t, backoff, tt.maxDelay, "backoff should be capped by max delay before full jitter")
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
			assert.True(t, record.mandatory)
			assert.Equal(t, string(tt.body), string(record.message.Body))
			assert.Equal(t, "application/json", record.message.ContentType)

			headerID, _ := record.message.Headers[libConstants.HeaderID].(string)
			assert.Equal(t, "req-123", headerID)
		})
	}
}

func TestRabbitMQAdapter_ProducerDefault_ReusesConfirmablePublisherWithoutClosingSharedChannel(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	ctx := testContextWithHeader("req-confirm-reuse")
	require.NoError(t, adapter.ProducerDefault(ctx, "test-exchange", "job.completed", []byte(`{"n":1}`), nil))
	require.NoError(t, adapter.ProducerDefault(ctx, "test-exchange", "job.completed", []byte(`{"n":2}`), nil))

	require.Len(t, channel.published, 2)
	assert.Equal(t, 1, channel.confirmCalls)
	assert.Equal(t, 1, channel.notifyPublishCalls)
	assert.Equal(t, 0, channel.notifyReturnCalls)
	assert.Equal(t, 0, channel.closeCalls)
	assert.False(t, channel.closed)
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

	forceCircuitOpenForTest(t, adapter)

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	ctx := testContextWithHeader("req-circuit")
	err := adapter.ProducerDefault(ctx, "ex", "key", []byte(`{}`), nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestRabbitMQAdapter_ProducerDefault_ReturnsError_WhenPublisherConfirmFails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		channel *testAMQPChannel
		wantErr string
	}{
		{
			name:    "confirm select fails",
			channel: &testAMQPChannel{confirmErr: errors.New("confirm unavailable")},
			wantErr: "confirm mode",
		},
		{
			name:    "broker nacks publish",
			channel: &testAMQPChannel{confirmation: amqp.Confirmation{DeliveryTag: 7, Ack: false}},
			wantErr: "nacked by broker",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := &testRabbitConnection{channel: tt.channel}
			adapter := newTestAdapter(conn)

			err := adapter.ProducerDefault(testContextWithHeader("req-confirm"), "ex", "missing", []byte(`{}`), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
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

func TestNewRabbitMQCircuitBreakerManager_ThresholdOverflowFallsBackToDefault(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()
	opts.CircuitBreakerThreshold = int(^uint32(0)) + 1

	manager := newRabbitMQCircuitBreakerManager(libLog.NewNop(), opts)
	require.NotNil(t, manager)

	adapter := &RabbitMQAdapter{options: opts, circuitBreaker: manager}

	for i := 0; i < DefaultCircuitBreakerThreshold-1; i++ {
		err := adapter.executeWithCircuitBreaker(func() error { return errors.New("boom") })
		require.Error(t, err)
		assert.NotEqual(t, CircuitOpen, adapter.CircuitBreakerState())
	}

	err := adapter.executeWithCircuitBreaker(func() error { return errors.New("boom") })
	require.Error(t, err)
	assert.Equal(t, CircuitOpen, adapter.CircuitBreakerState())
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
	publishErr         error
	publishErrs        []error
	publishAttempts    int
	confirmErr         error
	confirmation       amqp.Confirmation
	returned           *amqp.Return
	confirmCalls       int
	notifyPublishCalls int
	notifyReturnCalls  int
	publishConfirmCh   chan amqp.Confirmation
	returnCh           chan amqp.Return
	consumeErr         error
	qosErr             error

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

	if t.returned != nil && t.returnCh != nil {
		t.returnCh <- *t.returned
	}
	if t.publishConfirmCh != nil {
		confirmation := t.confirmation
		if confirmation.DeliveryTag == 0 {
			confirmation = amqp.Confirmation{DeliveryTag: uint64(t.publishAttempts), Ack: true}
		}
		t.publishConfirmCh <- confirmation
	}

	return nil
}

func (t *testAMQPChannel) PublishWithContext(_ context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return t.Publish(exchange, key, mandatory, immediate, msg)
}

func (t *testAMQPChannel) Confirm(bool) error {
	t.confirmCalls++
	return t.confirmErr
}

func (t *testAMQPChannel) NotifyPublish(receiver chan amqp.Confirmation) chan amqp.Confirmation {
	t.notifyPublishCalls++
	t.publishConfirmCh = receiver
	return receiver
}

func (t *testAMQPChannel) NotifyReturn(receiver chan amqp.Return) chan amqp.Return {
	t.notifyReturnCalls++
	t.returnCh = receiver
	return receiver
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
	opts.EnableSignatureVerification = false
	opts.EnableMessageSigning = false
	return &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}
}

// newTestAdapterWithChannel creates a properly initialized RabbitMQAdapter with a channel for testing.
func newTestAdapterWithChannel(conn rabbitConnection, channel amqpChannel) *RabbitMQAdapter {
	opts := DefaultOptions()
	opts.EnableSignatureVerification = false
	opts.EnableMessageSigning = false
	return &RabbitMQAdapter{
		conn:           conn,
		channel:        channel,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}
}

func newTestCircuitBreakerManager(opts AdapterOptions) libCircuitBreaker.Manager {
	return newRabbitMQCircuitBreakerManager(libLog.NewNop(), opts)
}

func forceCircuitOpenForTest(t *testing.T, adapter *RabbitMQAdapter) {
	t.Helper()

	for adapter.CircuitBreakerState() != CircuitOpen {
		err := adapter.executeWithCircuitBreaker(func() error { return errors.New("forced circuit failure") })
		require.Error(t, err)
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
	logger := &libLog.GoLogger{Level: libLog.LevelDebug}
	ctx := observability.ContextWithHeaderID(context.Background(), header)
	ctx = observability.ContextWithLogger(ctx, logger)

	return observability.ContextWithTracer(ctx, otel.Tracer("test"))
}

// -----------------------------------------------------------------------------
// Unit Tests: CircuitBreakerState
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_CircuitBreakerState_ReturnsCorrectState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupState    func(t *testing.T, adapter *RabbitMQAdapter)
		expectedState CircuitState
	}{
		{
			name: "returns closed state",
			setupState: func(t *testing.T, adapter *RabbitMQAdapter) {
				t.Helper()
				require.NoError(t, adapter.executeWithCircuitBreaker(func() error { return nil }))
			},
			expectedState: CircuitClosed,
		},
		{
			name: "returns open state",
			setupState: func(t *testing.T, adapter *RabbitMQAdapter) {
				forceCircuitOpenForTest(t, adapter)
			},
			expectedState: CircuitOpen,
		},
		{
			name: "returns half-open state",
			setupState: func(t *testing.T, adapter *RabbitMQAdapter) {
				t.Helper()
				adapter.options.CircuitBreakerThreshold = 1
				adapter.options.CircuitBreakerCooldown = 10 * time.Millisecond
				adapter.circuitBreaker = newTestCircuitBreakerManager(adapter.options)
				forceCircuitOpenForTest(t, adapter)
				time.Sleep(15 * time.Millisecond)
			},
			expectedState: CircuitHalfOpen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := &testRabbitConnection{channel: newTestAMQPChannel()}
			adapter := newTestAdapter(conn)
			tt.setupState(t, adapter)

			state := adapter.CircuitBreakerState()

			assert.Equal(t, tt.expectedState, state)
		})
	}
}

// -----------------------------------------------------------------------------
// Unit Tests: invalidateChannel
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_InvalidateChannel_ClosesAndNullifiesChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		channelClosed         bool
		channelCloseErr       bool
		expectedCloseCalls    int
		expectChannelNilAfter bool
	}{
		{
			name:                  "closes open channel",
			channelClosed:         false,
			channelCloseErr:       false,
			expectedCloseCalls:    1,
			expectChannelNilAfter: true,
		},
		{
			name:                  "skips close on already closed channel",
			channelClosed:         true,
			channelCloseErr:       false,
			expectedCloseCalls:    0,
			expectChannelNilAfter: true,
		},
		{
			name:                  "handles close error gracefully",
			channelClosed:         false,
			channelCloseErr:       true,
			expectedCloseCalls:    1,
			expectChannelNilAfter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			channel := newTestAMQPChannel()
			channel.closed = tt.channelClosed
			channel.closeShouldFail = tt.channelCloseErr
			conn := &testRabbitConnection{channel: channel}
			adapter := newTestAdapterWithChannel(conn, channel)

			logger := &libLog.GoLogger{Level: libLog.LevelDebug}
			adapter.invalidateChannel(logger)

			assert.Equal(t, tt.expectedCloseCalls, channel.closeCalls)
			assert.Nil(t, adapter.channel, "channel should be nil after invalidation")
		})
	}
}

func TestRabbitMQAdapter_InvalidateChannel_WithNilChannel(t *testing.T) {
	t.Parallel()

	conn := &testRabbitConnection{}
	adapter := newTestAdapter(conn)
	adapter.channel = nil

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	// Should not panic
	adapter.invalidateChannel(logger)

	assert.Nil(t, adapter.channel)
}

func TestRabbitMQAdapter_InvalidateChannel_WithNilLogger(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	channel.closeShouldFail = true
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)

	// Should not panic even with nil logger
	adapter.invalidateChannel(nil)

	assert.Nil(t, adapter.channel)
}

// -----------------------------------------------------------------------------
// Unit Tests: startChannelWatcher
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_StartChannelWatcher_HandlesNilChannel(t *testing.T) {
	t.Parallel()

	conn := &testRabbitConnection{}
	adapter := newTestAdapter(conn)
	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	// Should not panic with nil channel
	adapter.startChannelWatcher(logger, nil)

	assert.Nil(t, adapter.channel)
}

func TestRabbitMQAdapter_StartChannelWatcher_NullifiesChannelOnClose(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)
	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	adapter.startChannelWatcher(logger, channel)

	// Simulate channel close notification with error
	channel.notifyClose <- &amqp.Error{Code: 504, Reason: "channel closed"}

	// Wait for the goroutine to process
	require.Eventually(t, func() bool {
		adapter.mu.Lock()
		defer adapter.mu.Unlock()
		return adapter.channel == nil
	}, time.Second, 10*time.Millisecond)
}

func TestRabbitMQAdapter_StartChannelWatcher_HandlesNilError(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)
	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	adapter.startChannelWatcher(logger, channel)

	// Close the notification channel (simulates graceful close)
	close(channel.notifyClose)

	// Wait for the goroutine to process
	require.Eventually(t, func() bool {
		adapter.mu.Lock()
		defer adapter.mu.Unlock()
		return adapter.channel == nil
	}, time.Second, 10*time.Millisecond)
}

// -----------------------------------------------------------------------------
// Unit Tests: ConsumerLoop edge cases
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_ConsumerLoop_ReturnsNilOnContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(testContextWithHeader("req-cancel"))
	cancel() // Cancel immediately

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	handler := func(context.Context, []byte, map[string]any) error {
		return nil
	}

	err := adapter.ConsumerLoop(ctx, "queue", 1, handler)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestRabbitMQAdapter_ConsumerLoop_NormalizesConcurrency(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(testContextWithHeader("req-concurrency"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body:         []byte(`{"test":"data"}`),
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	handled := make(chan struct{})
	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		close(handled)
		cancel()
		return nil
	}

	// Pass concurrency < 1, should be normalized to 1
	err := adapter.ConsumerLoop(ctx, "queue", 0, handler)
	require.NoError(t, err)

	select {
	case <-handled:
		// Success
	case <-time.After(time.Second):
		t.Fatal("handler was not invoked")
	}
}

func TestRabbitMQAdapter_ConsumerLoop_ReturnsNilOnContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(testContextWithHeader("req-timeout"), 1*time.Millisecond)
	defer cancel()

	time.Sleep(5 * time.Millisecond) // Ensure deadline exceeded

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	handler := func(context.Context, []byte, map[string]any) error {
		return nil
	}

	err := adapter.ConsumerLoop(ctx, "queue", 1, handler)
	// ConsumerLoop may return either context.DeadlineExceeded (if caught at loop start)
	// or nil (if caught during runConsumerCycle). Both are valid graceful exits.
	if err != nil {
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	}
}

// -----------------------------------------------------------------------------
// Unit Tests: runConsumerCycle error paths
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_RunConsumerCycle_FailsOnQosError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(testContextWithHeader("req-qos-err"), 500*time.Millisecond)
	defer cancel()

	channel := newTestAMQPChannel()
	channel.qosErr = errors.New("qos failed")
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	handler := func(context.Context, []byte, map[string]any) error {
		return nil
	}

	// The ConsumerLoop should eventually timeout because QoS keeps failing
	err := adapter.ConsumerLoop(ctx, "queue", 1, handler)

	// Context should be canceled due to deadline
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRabbitMQAdapter_RunConsumerCycle_FailsOnConsumeError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(testContextWithHeader("req-consume-err"), 500*time.Millisecond)
	defer cancel()

	channel := newTestAMQPChannel()
	channel.consumeErr = errors.New("consume failed")
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	handler := func(context.Context, []byte, map[string]any) error {
		return nil
	}

	err := adapter.ConsumerLoop(ctx, "queue", 1, handler)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRabbitMQAdapter_RunConsumerCycle_RecordsSetupFailuresInCircuitBreaker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(*testAMQPChannel)
	}{
		{
			name: "qos failure opens circuit",
			setup: func(ch *testAMQPChannel) {
				ch.qosErr = errors.New("qos failed")
			},
		},
		{
			name: "consume failure opens circuit",
			setup: func(ch *testAMQPChannel) {
				ch.consumeErr = errors.New("consume failed")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			channel := newTestAMQPChannel()
			tt.setup(channel)

			conn := &testRabbitConnection{channel: channel}
			adapter := newTestAdapter(conn)
			adapter.options.CircuitBreakerThreshold = 1
			adapter.circuitBreaker = newTestCircuitBreakerManager(adapter.options)

			err := adapter.runConsumerCycle(testContextWithHeader("req-setup"), otel.Tracer("test"), libLog.NewNop(), "queue", "req-setup", 1, func(context.Context, []byte, map[string]any) error {
				return nil
			})

			require.Error(t, err)
			assert.Equal(t, CircuitOpen, adapter.CircuitBreakerState())
		})
	}
}

// -----------------------------------------------------------------------------
// Unit Tests: processDelivery edge cases
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_ProcessDelivery_ExtractsHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		headers        amqp.Table
		expectedHeader string
	}{
		{
			name:           "extracts existing header ID",
			headers:        amqp.Table{libConstants.HeaderID: "existing-id"},
			expectedHeader: "existing-id",
		},
		{
			name:           "handles nil headers",
			headers:        nil,
			expectedHeader: "", // Will generate new UUID
		},
		{
			name:           "handles missing header ID",
			headers:        amqp.Table{"other": "value"},
			expectedHeader: "", // Will generate new UUID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(testContextWithHeader("req-headers"))
			defer cancel()

			channel := newTestAMQPChannel()
			ack := &testAcknowledger{}
			channel.deliveries <- amqp.Delivery{
				Body:         []byte(`{"test":"data"}`),
				Headers:      tt.headers,
				Acknowledger: ack,
			}

			conn := &testRabbitConnection{channel: channel}
			adapter := newTestAdapter(conn)

			t.Cleanup(func() {
				_ = adapter.Shutdown(testContextWithHeader("cleanup"))
			})

			var receivedHeaders map[string]any
			handled := make(chan struct{})
			handler := func(ctx context.Context, body []byte, headers map[string]any) error {
				receivedHeaders = headers
				close(handled)
				cancel()
				return nil
			}

			go func() {
				_ = adapter.ConsumerLoop(ctx, "queue", 1, handler)
			}()

			select {
			case <-handled:
				if tt.expectedHeader != "" {
					assert.Equal(t, tt.expectedHeader, receivedHeaders[libConstants.HeaderID])
				}
			case <-time.After(time.Second):
				t.Fatal("handler was not invoked")
			}
		})
	}
}

func TestRabbitMQAdapter_ProcessDelivery_HandlesNonStringHeaderID(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(testContextWithHeader("req-non-string"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body:         []byte(`{"test":"data"}`),
		Headers:      amqp.Table{libConstants.HeaderID: 12345}, // Non-string
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	handled := make(chan struct{})
	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		close(handled)
		cancel()
		return nil
	}

	go func() {
		_ = adapter.ConsumerLoop(ctx, "queue", 1, handler)
	}()

	select {
	case <-handled:
		// Success - should handle non-string gracefully
	case <-time.After(time.Second):
		t.Fatal("handler was not invoked")
	}
}

func TestRabbitMQAdapter_ProcessDelivery_RecoverFromPanic(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(testContextWithHeader("req-panic"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body:         []byte(`{"panic":"test"}`),
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	panicked := make(chan struct{})
	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		close(panicked)
		panic("simulated panic")
	}

	go func() {
		<-panicked
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := adapter.ConsumerLoop(ctx, "queue", 1, handler)

	require.NoError(t, err)

	// Message should be nacked on panic
	require.Eventually(t, func() bool {
		return ack.nacks == 1
	}, time.Second, 10*time.Millisecond)
}

func TestRabbitMQAdapter_ProcessDelivery_NackError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(testContextWithHeader("req-nack-err"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledgerWithError{nackErr: errors.New("nack failed")}
	channel.deliveries <- amqp.Delivery{
		Body:         []byte(`{"data":"test"}`),
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	handled := make(chan struct{})
	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		close(handled)
		return errors.New("handler error")
	}

	go func() {
		<-handled
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := adapter.ConsumerLoop(ctx, "queue", 1, handler)

	require.NoError(t, err)
}

func TestRabbitMQAdapter_ProcessDelivery_AckError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(testContextWithHeader("req-ack-err"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledgerWithError{ackErr: errors.New("ack failed")}
	channel.deliveries <- amqp.Delivery{
		Body:         []byte(`{"data":"test"}`),
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	handled := make(chan struct{})
	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		close(handled)
		cancel()
		return nil
	}

	err := adapter.ConsumerLoop(ctx, "queue", 1, handler)

	require.NoError(t, err)

	select {
	case <-handled:
		// Success - ack error is logged but doesn't fail processing
	case <-time.After(time.Second):
		t.Fatal("handler was not invoked")
	}
}

// -----------------------------------------------------------------------------
// Unit Tests: Shutdown edge cases
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_Shutdown_RespectsContextDeadline(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)

	// Create context with very short deadline
	ctx, cancel := context.WithTimeout(testContextWithHeader("req-deadline"), 10*time.Millisecond)
	defer cancel()

	err := adapter.Shutdown(ctx)

	require.NoError(t, err)
	assert.True(t, adapter.shutdown.Load())
}

func TestRabbitMQAdapter_Shutdown_WaitsForConsumers(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)

	// Simulate an active consumer
	adapter.consumerWg.Add(1)

	done := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		adapter.consumerWg.Done()
		close(done)
	}()

	ctx := testContextWithHeader("req-wait-consumers")
	err := adapter.Shutdown(ctx)

	require.NoError(t, err)
	assert.True(t, adapter.shutdown.Load())

	<-done // Ensure goroutine completed
}

func TestRabbitMQAdapter_Shutdown_HandlesConnectionCloseError(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{
		channel:  channel,
		closeErr: errors.New("connection close error"),
	}
	adapter := newTestAdapterWithChannel(conn, channel)

	ctx := testContextWithHeader("req-close-err")
	err := adapter.Shutdown(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "close connection")
	assert.True(t, adapter.shutdown.Load())
}

func TestRabbitMQAdapter_Shutdown_TimeoutWithActiveConsumers(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}

	opts := DefaultOptions()
	opts.ShutdownTimeout = 50 * time.Millisecond
	adapter := &RabbitMQAdapter{
		conn:           conn,
		channel:        channel,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	// Simulate a stuck consumer
	adapter.consumerWg.Add(1)

	ctx := testContextWithHeader("req-timeout")
	err := adapter.Shutdown(ctx)

	require.NoError(t, err)
	assert.True(t, adapter.shutdown.Load())

	// Clean up the WaitGroup
	adapter.consumerWg.Done()
}

func TestRabbitMQAdapter_Shutdown_ContextCanceledDuringWait(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)

	// Simulate an active consumer
	adapter.consumerWg.Add(1)

	ctx, cancel := context.WithCancel(testContextWithHeader("req-cancel-during"))
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := adapter.Shutdown(ctx)

	require.NoError(t, err)
	assert.True(t, adapter.shutdown.Load())

	// Clean up the WaitGroup
	adapter.consumerWg.Done()
}

// -----------------------------------------------------------------------------
// Unit Tests: ProducerDefault additional paths
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_ProducerDefault_AllRetriesFail(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	channel.publishErr = errors.New("persistent publish failure")
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapter(conn)

	t.Cleanup(func() {
		adapter.shutdown.Store(true)
	})

	ctx := testContextWithHeader("req-all-fail")
	err := adapter.ProducerDefault(ctx, "ex", "key", []byte(`{}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish message after")
}

// -----------------------------------------------------------------------------
// Unit Tests: ensureChannel
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_EnsureChannel_ReturnsExistingChannel(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}
	adapter := newTestAdapterWithChannel(conn, channel)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}
	ch, err := adapter.ensureChannel(nil, logger)

	require.NoError(t, err)
	assert.Equal(t, channel, ch)
	assert.Equal(t, 0, conn.calls, "should not call EnsureChannel when channel exists")
}

func TestRabbitMQAdapter_EnsureChannel_ReconnectsWhenChannelClosed(t *testing.T) {
	t.Parallel()

	closedChannel := newTestAMQPChannel()
	closedChannel.closed = true

	newChannel := newTestAMQPChannel()
	conn := &testRabbitConnection{
		channels: []amqpChannel{newChannel},
	}

	adapter := newTestAdapterWithChannel(conn, closedChannel)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}
	ch, err := adapter.ensureChannel(nil, logger)

	require.NoError(t, err)
	assert.NotNil(t, ch)
	assert.Equal(t, 1, conn.calls, "should call EnsureChannel once")
}

func TestRabbitMQAdapter_EnsureChannel_RetriesOnFailure(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()
	opts.MaxRetryAttempts = 2
	opts.BaseRetryDelay = 1 * time.Millisecond

	failingConn := &testRabbitConnectionWithAttempts{
		failAttempts: 1,
		channel:      newTestAMQPChannel(),
	}

	adapter := &RabbitMQAdapter{
		conn:           failingConn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}
	ch, err := adapter.ensureChannel(nil, logger)

	require.NoError(t, err)
	assert.NotNil(t, ch)
	assert.Equal(t, 2, failingConn.calls, "should retry after first failure")
}

// -----------------------------------------------------------------------------
// Unit Tests: CalculateBackoff edge cases
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_CalculateBackoff_ZeroAttempt(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()

	backoff := calculateBackoff(0, opts.BaseRetryDelay, opts.MaxRetryDelay)

	assert.GreaterOrEqual(t, backoff, time.Duration(0))
	assert.Less(t, backoff, opts.BaseRetryDelay)
}

func TestRabbitMQAdapter_CalculateBackoff_NegativeAttempt(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()

	backoff := calculateBackoff(-1, opts.BaseRetryDelay, opts.MaxRetryDelay)

	assert.GreaterOrEqual(t, backoff, time.Duration(0))
	assert.Less(t, backoff, opts.BaseRetryDelay)
}

// -----------------------------------------------------------------------------
// Additional Test Helpers
// -----------------------------------------------------------------------------

// testAcknowledgerWithError extends testAcknowledger to support error returns
type testAcknowledgerWithError struct {
	acks    int
	nacks   int
	ackErr  error
	nackErr error
}

func (t *testAcknowledgerWithError) Ack(uint64, bool) error {
	t.acks++
	return t.ackErr
}

func (t *testAcknowledgerWithError) Nack(uint64, bool, bool) error {
	t.nacks++
	return t.nackErr
}

func (t *testAcknowledgerWithError) Reject(uint64, bool) error {
	t.nacks++
	return t.nackErr
}

// testRabbitConnectionWithAttempts allows controlling failures by attempt count
type testRabbitConnectionWithAttempts struct {
	channel      amqpChannel
	failAttempts int
	calls        int
	closeCalls   int
}

func (t *testRabbitConnectionWithAttempts) EnsureChannel() (amqpChannel, error) {
	t.calls++
	if t.calls <= t.failAttempts {
		return nil, errors.New("connection failed")
	}

	if ch, ok := t.channel.(*testAMQPChannel); ok {
		ch.closed = false
	}

	return t.channel, nil
}

func (t *testRabbitConnectionWithAttempts) Close() error {
	t.closeCalls++
	return nil
}

// -----------------------------------------------------------------------------
// Unit Tests: Message Signing
// -----------------------------------------------------------------------------

func TestRabbitMQAdapter_ProducerDefault_SignsMessage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)
	mockSigner.EXPECT().Sign(gomock.Any()).Return("test-signature")
	mockSigner.EXPECT().SignatureVersion().Return("v1").Times(2)

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}

	opts := DefaultOptions()
	opts.Signer = mockSigner
	opts.EnableMessageSigning = true

	adapter := &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	ctx := testContextWithHeader("req-sign")
	err := adapter.ProducerDefault(ctx, "test-exchange", "test-key", []byte(`{"data":"test"}`), nil)

	require.NoError(t, err)
	require.Len(t, channel.published, 1)

	headers := channel.published[0].message.Headers
	assert.Equal(t, "test-signature", headers[HeaderMessageSignature])
	assert.Equal(t, "v1", headers[HeaderSignatureVersion])

	// Verify timestamp is a valid number
	timestampStr, ok := headers[HeaderSignatureTimestamp].(string)
	require.True(t, ok, "timestamp should be a string")
	_, err = strconv.ParseInt(timestampStr, 10, 64)
	assert.NoError(t, err, "timestamp should be a valid int64")
}

func TestRabbitMQAdapter_ProducerDefault_SkipsSigningWhenDisabled(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)
	// No expectations - signer should not be called

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}

	opts := DefaultOptions()
	opts.Signer = mockSigner
	opts.EnableMessageSigning = false

	adapter := &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	ctx := testContextWithHeader("req-no-sign")
	err := adapter.ProducerDefault(ctx, "test-exchange", "test-key", []byte(`{"data":"test"}`), nil)

	require.NoError(t, err)
	require.Len(t, channel.published, 1)

	headers := channel.published[0].message.Headers
	assert.Nil(t, headers[HeaderMessageSignature], "signature header should not be present")
	assert.Nil(t, headers[HeaderSignatureTimestamp], "timestamp header should not be present")
	assert.Nil(t, headers[HeaderSignatureVersion], "version header should not be present")
}

func TestRabbitMQAdapter_ProducerDefault_SkipsSigningWhenNoSigner(t *testing.T) {
	t.Parallel()

	channel := newTestAMQPChannel()
	conn := &testRabbitConnection{channel: channel}

	opts := DefaultOptions()
	opts.Signer = nil
	opts.EnableMessageSigning = true

	adapter := &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	ctx := testContextWithHeader("req-no-signer")
	err := adapter.ProducerDefault(ctx, "test-exchange", "test-key", []byte(`{"data":"test"}`), nil)

	require.NoError(t, err)
	require.Len(t, channel.published, 1)

	headers := channel.published[0].message.Headers
	assert.Nil(t, headers[HeaderMessageSignature])
}

func TestRabbitMQAdapter_ConsumerLoop_VerifiesSignatureSuccessfully(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)
	mockSigner.EXPECT().SignatureVersion().Return("v1")
	mockSigner.EXPECT().Verify(gomock.Any(), "valid-signature").Return(nil)

	ctx, cancel := context.WithCancel(testContextWithHeader("req-verify-ok"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body: []byte(`{"data":"test"}`),
		Headers: amqp.Table{
			HeaderMessageSignature:   "valid-signature",
			HeaderSignatureTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
			HeaderSignatureVersion:   "v1",
		},
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}

	opts := DefaultOptions()
	opts.Signer = mockSigner
	opts.EnableSignatureVerification = true

	adapter := &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	handled := make(chan struct{})
	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		close(handled)
		cancel()
		return nil
	}

	err := adapter.ConsumerLoop(ctx, "queue-verify", 1, handler)
	require.NoError(t, err)

	select {
	case <-handled:
		// Success - handler was called
	case <-time.After(time.Second):
		t.Fatal("handler was not invoked")
	}

	require.Eventually(t, func() bool {
		return ack.acks == 1 && ack.nacks == 0
	}, time.Second, 10*time.Millisecond)
}

func TestRabbitMQAdapter_ConsumerLoop_NacksOnMissingSignature(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)
	// No expectations - verification should fail before calling signer

	ctx, cancel := context.WithCancel(testContextWithHeader("req-missing-sig"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body:    []byte(`{"data":"test"}`),
		Headers: amqp.Table{
			// Missing signature headers
		},
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}

	opts := DefaultOptions()
	opts.Signer = mockSigner
	opts.EnableSignatureVerification = true

	adapter := &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		t.Fatal("handler should not be called when signature is missing")
		return nil
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := adapter.ConsumerLoop(ctx, "queue-missing-sig", 1, handler)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return ack.nacks == 1 && ack.acks == 0
	}, time.Second, 10*time.Millisecond)
}

func TestRabbitMQAdapter_ConsumerLoop_NacksOnInvalidSignature(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)
	mockSigner.EXPECT().SignatureVersion().Return("v1")
	mockSigner.EXPECT().Verify(gomock.Any(), "invalid-signature").Return(crypto.ErrInvalidSignature)

	ctx, cancel := context.WithCancel(testContextWithHeader("req-invalid-sig"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body: []byte(`{"data":"test"}`),
		Headers: amqp.Table{
			HeaderMessageSignature:   "invalid-signature",
			HeaderSignatureTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
			HeaderSignatureVersion:   "v1",
		},
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}

	opts := DefaultOptions()
	opts.Signer = mockSigner
	opts.EnableSignatureVerification = true

	adapter := &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		t.Fatal("handler should not be called when signature is invalid")
		return nil
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := adapter.ConsumerLoop(ctx, "queue-invalid-sig", 1, handler)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return ack.nacks == 1 && ack.acks == 0
	}, time.Second, 10*time.Millisecond)
}

func TestRabbitMQAdapter_ConsumerLoop_NacksOnVersionMismatch(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)
	mockSigner.EXPECT().SignatureVersion().Return("v2").AnyTimes() // Signer expects v2

	ctx, cancel := context.WithCancel(testContextWithHeader("req-version-mismatch"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body: []byte(`{"data":"test"}`),
		Headers: amqp.Table{
			HeaderMessageSignature:   "some-signature",
			HeaderSignatureTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
			HeaderSignatureVersion:   "v1", // Message has v1
		},
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}

	opts := DefaultOptions()
	opts.Signer = mockSigner
	opts.EnableSignatureVerification = true

	adapter := &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		t.Fatal("handler should not be called when version mismatches")
		return nil
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := adapter.ConsumerLoop(ctx, "queue-version-mismatch", 1, handler)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return ack.nacks == 1 && ack.acks == 0
	}, time.Second, 10*time.Millisecond)
}

func TestRabbitMQAdapter_ConsumerLoop_SkipsVerificationWhenDisabled(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)
	// No expectations - signer should not be called

	ctx, cancel := context.WithCancel(testContextWithHeader("req-skip-verify"))
	defer cancel()

	channel := newTestAMQPChannel()
	ack := &testAcknowledger{}
	channel.deliveries <- amqp.Delivery{
		Body:    []byte(`{"data":"test"}`),
		Headers: amqp.Table{
			// No signature headers - should still work when verification disabled
		},
		Acknowledger: ack,
	}

	conn := &testRabbitConnection{channel: channel}

	opts := DefaultOptions()
	opts.Signer = mockSigner
	opts.EnableSignatureVerification = false

	adapter := &RabbitMQAdapter{
		conn:           conn,
		options:        opts,
		circuitBreaker: newTestCircuitBreakerManager(opts),
	}

	t.Cleanup(func() {
		_ = adapter.Shutdown(testContextWithHeader("cleanup"))
	})

	handled := make(chan struct{})
	handler := func(ctx context.Context, body []byte, headers map[string]any) error {
		close(handled)
		cancel()
		return nil
	}

	err := adapter.ConsumerLoop(ctx, "queue-skip-verify", 1, handler)
	require.NoError(t, err)

	select {
	case <-handled:
		// Success - handler was called without verification
	case <-time.After(time.Second):
		t.Fatal("handler was not invoked")
	}

	require.Eventually(t, func() bool {
		return ack.acks == 1 && ack.nacks == 0
	}, time.Second, 10*time.Millisecond)
}

// -----------------------------------------------------------------------------
// Unit Tests: verifyMessageSignature
// -----------------------------------------------------------------------------

func TestVerifyMessageSignature_MissingSignatureHeader(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	headers := map[string]any{
		HeaderSignatureTimestamp: "1704067200",
		HeaderSignatureVersion:   "v1",
	}

	err := VerifyMessageSignature([]byte(`{}`), headers, "", "", mockSigner, DefaultSignatureTimestampTolerance, logger, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingSignatureHeaders)
	assert.Contains(t, err.Error(), HeaderMessageSignature)
}

func TestVerifyMessageSignature_MissingTimestampHeader(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	headers := map[string]any{
		HeaderMessageSignature: "some-signature",
		HeaderSignatureVersion: "v1",
	}

	err := VerifyMessageSignature([]byte(`{}`), headers, "", "", mockSigner, DefaultSignatureTimestampTolerance, logger, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingSignatureHeaders)
	assert.Contains(t, err.Error(), HeaderSignatureTimestamp)
}

func TestVerifyMessageSignature_MissingVersionHeader(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	headers := map[string]any{
		HeaderMessageSignature:   "some-signature",
		HeaderSignatureTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
	}

	err := VerifyMessageSignature([]byte(`{}`), headers, "", "", mockSigner, DefaultSignatureTimestampTolerance, logger, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingSignatureHeaders)
	assert.Contains(t, err.Error(), HeaderSignatureVersion)
}

func TestVerifyMessageSignature_InvalidTimestampFormat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	headers := map[string]any{
		HeaderMessageSignature:   "some-signature",
		HeaderSignatureTimestamp: "not-a-number",
		HeaderSignatureVersion:   "v1",
	}

	err := VerifyMessageSignature([]byte(`{}`), headers, "", "", mockSigner, DefaultSignatureTimestampTolerance, logger, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureVerificationFailed)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestVerifyMessageSignature_TimestampAsInt64(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)
	mockSigner.EXPECT().SignatureVersion().Return("v1")
	mockSigner.EXPECT().Verify(gomock.Any(), "valid-signature").Return(nil)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	headers := map[string]any{
		HeaderMessageSignature:   "valid-signature",
		HeaderSignatureTimestamp: time.Now().Unix(),
		HeaderSignatureVersion:   "v1",
	}

	err := VerifyMessageSignature([]byte(`{}`), headers, "", "", mockSigner, DefaultSignatureTimestampTolerance, logger, nil)

	require.NoError(t, err)
}

func TestVerifyMessageSignature_TimestampAsInt(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)
	mockSigner.EXPECT().SignatureVersion().Return("v1")
	mockSigner.EXPECT().Verify(gomock.Any(), "valid-signature").Return(nil)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	headers := map[string]any{
		HeaderMessageSignature:   "valid-signature",
		HeaderSignatureTimestamp: int(time.Now().Unix()),
		HeaderSignatureVersion:   "v1",
	}

	err := VerifyMessageSignature([]byte(`{}`), headers, "", "", mockSigner, DefaultSignatureTimestampTolerance, logger, nil)

	require.NoError(t, err)
}

func TestVerifyMessageSignature_NonStringSignature(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	headers := map[string]any{
		HeaderMessageSignature:   12345, // Non-string
		HeaderSignatureTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
		HeaderSignatureVersion:   "v1",
	}

	err := VerifyMessageSignature([]byte(`{}`), headers, "", "", mockSigner, DefaultSignatureTimestampTolerance, logger, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingSignatureHeaders)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestVerifyMessageSignature_NonStringVersion(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	headers := map[string]any{
		HeaderMessageSignature:   "some-signature",
		HeaderSignatureTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
		HeaderSignatureVersion:   123, // Non-string
	}

	err := VerifyMessageSignature([]byte(`{}`), headers, "", "", mockSigner, DefaultSignatureTimestampTolerance, logger, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingSignatureHeaders)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestVerifyMessageSignature_UnsupportedTimestampType(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSigner := crypto.NewMockSigner(ctrl)

	logger := &libLog.GoLogger{Level: libLog.LevelDebug}

	headers := map[string]any{
		HeaderMessageSignature:   "some-signature",
		HeaderSignatureTimestamp: []byte("timestamp"), // Unsupported type
		HeaderSignatureVersion:   "v1",
	}

	err := VerifyMessageSignature([]byte(`{}`), headers, "", "", mockSigner, DefaultSignatureTimestampTolerance, logger, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingSignatureHeaders)
}

// -----------------------------------------------------------------------------
// Unit Tests: DefaultOptions with signing
// -----------------------------------------------------------------------------

func TestDefaultOptions_IncludesSigningDefaults(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()

	assert.True(t, opts.EnableMessageSigning, "EnableMessageSigning should default to true")
	assert.True(t, opts.EnableSignatureVerification, "EnableSignatureVerification should default to true")
	assert.Nil(t, opts.Signer, "Signer should be nil by default")
}
