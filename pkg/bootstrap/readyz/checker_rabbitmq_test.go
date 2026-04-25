package readyz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRabbitAdapter struct {
	state     BreakerState
	pingErr   error
	pingDelay time.Duration
	pingCalls int
}

func (f *fakeRabbitAdapter) State() BreakerState { return f.state }
func (f *fakeRabbitAdapter) Ping(ctx context.Context) error {
	f.pingCalls++

	if f.pingDelay > 0 {
		select {
		case <-time.After(f.pingDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return f.pingErr
}

func TestRabbitMQAdapterChecker_Closed_Up(t *testing.T) {
	adapter := &fakeRabbitAdapter{state: BreakerClosed, pingErr: nil}
	c := NewRabbitMQAdapterChecker(adapter, "amqp://host:5672")

	assert.Equal(t, "rabbitmq", c.Name())

	res := c.Check(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	assert.Equal(t, "closed", res.BreakerState)
	assert.Empty(t, res.Error)
	assert.Equal(t, 1, adapter.pingCalls, "closed state MUST invoke probe")
	if assert.NotNil(t, res.TLS) {
		assert.False(t, *res.TLS)
	}
}

func TestRabbitMQAdapterChecker_Closed_DownOnProbeError(t *testing.T) {
	adapter := &fakeRabbitAdapter{
		state:   BreakerClosed,
		pingErr: errors.New("dial tcp: connection refused"),
	}
	c := NewRabbitMQAdapterChecker(adapter, "amqps://host:5671")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "closed", res.BreakerState)
	assert.Contains(t, res.Error, "connection refused")
	if assert.NotNil(t, res.TLS) {
		assert.True(t, *res.TLS, "amqps:// implies TLS=true")
	}
}

func TestRabbitMQAdapterChecker_HalfOpen_SkipsProbeReturnsDegraded(t *testing.T) {
	adapter := &fakeRabbitAdapter{state: BreakerHalfOpen}
	c := NewRabbitMQAdapterChecker(adapter, "amqp://host:5672")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDegraded, res.Status)
	assert.Equal(t, "half-open", res.BreakerState)
	assert.Zero(t, adapter.pingCalls, "half-open MUST NOT call probe")
}

func TestRabbitMQAdapterChecker_Open_SkipsProbeReturnsDown(t *testing.T) {
	adapter := &fakeRabbitAdapter{state: BreakerOpen}
	c := NewRabbitMQAdapterChecker(adapter, "amqp://host:5672")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "open", res.BreakerState)
	assert.Equal(t, "circuit breaker open", res.Error)
	assert.Zero(t, adapter.pingCalls, "open MUST NOT call probe")
}

func TestRabbitMQAdapterChecker_Timeout(t *testing.T) {
	adapter := &fakeRabbitAdapter{
		state:     BreakerClosed,
		pingDelay: 50 * time.Millisecond,
		pingErr:   context.DeadlineExceeded,
	}
	c := NewRabbitMQAdapterChecker(adapter, "amqp://host:5672")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	res := c.Check(ctx)
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "timeout", res.Error)
}

func TestRabbitMQAdapterChecker_NilAdapter(t *testing.T) {
	c := NewRabbitMQAdapterChecker(nil, "amqp://host:5672")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "rabbitmq adapter not initialized", res.Error)
}

func TestRabbitMQAdapterChecker_SanitizesError(t *testing.T) {
	adapter := &fakeRabbitAdapter{
		state:   BreakerClosed,
		pingErr: errors.New("failed: amqp://guest:hunter2@rabbit:5672"),
	}
	c := NewRabbitMQAdapterChecker(adapter, "amqp://host:5672")

	res := c.Check(context.Background())
	require.Equal(t, StatusDown, res.Status)
	assert.NotContains(t, res.Error, "hunter2")
	assert.Contains(t, res.Error, "***@rabbit:5672")
}

func TestBreakerState_String(t *testing.T) {
	assert.Equal(t, "closed", BreakerClosed.String())
	assert.Equal(t, "open", BreakerOpen.String())
	assert.Equal(t, "half-open", BreakerHalfOpen.String())
	assert.Equal(t, "unknown", BreakerState(99).String())
}
