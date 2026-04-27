package readyz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisClientChecker_Up(t *testing.T) {
	called := 0

	c := NewRedisClientCheckerFromFn("redis",
		func(_ context.Context) error {
			called++
			return nil
		},
		"redis://host:6379",
	)

	assert.Equal(t, "redis", c.Name())

	res := c.Check(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	assert.Empty(t, res.Error)
	if assert.NotNil(t, res.TLS) {
		assert.False(t, *res.TLS)
	}

	assert.Equal(t, 1, called)
}

func TestRedisClientChecker_Down(t *testing.T) {
	c := NewRedisClientCheckerFromFn("redis",
		func(_ context.Context) error { return errors.New("dial tcp: connection refused") },
		"rediss://host:6380",
	)

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "connection refused")
	if assert.NotNil(t, res.TLS) {
		assert.True(t, *res.TLS, "rediss:// implies TLS=true")
	}
}

func TestRedisClientChecker_Timeout(t *testing.T) {
	c := NewRedisClientCheckerFromFn("redis",
		func(ctx context.Context) error {
			select {
			case <-time.After(100 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		"redis://host:6379",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	res := c.Check(ctx)
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "timeout", res.Error)
}

func TestRedisClientChecker_NilPinger(t *testing.T) {
	c := NewRedisClientChecker("multi_tenant_redis", nil, "redis://host:6379")

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "redis client not initialized", res.Error)
}

func TestRedisClientChecker_DepName(t *testing.T) {
	c := NewRedisClientCheckerFromFn("multi_tenant_redis",
		func(_ context.Context) error { return nil },
		"redis://host:6379",
	)
	assert.Equal(t, "multi_tenant_redis", c.Name())
}

// TestRedisClientCheckerFromFn_NilPing_DoesNotPanic is a regression for the
// nil-interface trap: passing a nil func to NewRedisClientCheckerFromFn used
// to wrap it as redisPingerFunc(nil), producing a non-nil interface holding
// a nil function value. The "ping == nil" guard inside Check() did not fire
// for that shape, so the next call to PingErr would panic invoking the nil
// func. The constructor now forwards a typed-nil interface so the guard
// triggers and the checker reports "down" cleanly.
func TestRedisClientCheckerFromFn_NilPing_DoesNotPanic(t *testing.T) {
	c := NewRedisClientCheckerFromFn("redis", nil, "redis://localhost:6379")

	require.NotPanics(t, func() {
		res := c.Check(context.Background())
		assert.Equal(t, StatusDown, res.Status)
		assert.Equal(t, "redis client not initialized", res.Error)
	})
}

func TestRedisClientChecker_RedactsPasswordInError(t *testing.T) {
	c := NewRedisClientCheckerFromFn("redis",
		func(_ context.Context) error {
			return errors.New("authentication failed for redis://user:s3cr3t@host:6379")
		},
		"redis://host:6379",
	)

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.NotContains(t, res.Error, "s3cr3t")
}
