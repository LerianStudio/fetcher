package readyz

import (
	"context"
	"time"
)

// RedisPinger is the narrow client surface required by the checker. The
// go-redis Ping returns *redis.StatusCmd, so callers adapt it to (error) via
// the closure form in NewRedisClientCheckerFromFn.
type RedisPinger interface {
	PingErr(ctx context.Context) error
}

type redisPingerFunc func(ctx context.Context) error

func (f redisPingerFunc) PingErr(ctx context.Context) error { return f(ctx) }

// RedisClientChecker probes a Redis instance with PING and reports latency.
// The dep name is caller-supplied so the same type can register under
// distinct names (e.g. "redis" vs "multi_tenant_redis") for dashboarding.
type RedisClientChecker struct {
	name   string
	ping   RedisPinger
	rawURL string
}

// NewRedisClientChecker constructs a checker under the given dep name. rawURL
// is used only for TLS posture detection (scheme "rediss" → TLS=true).
func NewRedisClientChecker(name string, ping RedisPinger, rawURL string) *RedisClientChecker {
	return &RedisClientChecker{name: name, ping: ping, rawURL: rawURL}
}

// NewRedisClientCheckerFromFn adapts a plain ping function (e.g.
// rdb.Ping(ctx).Err()) into the checker's interface.
func NewRedisClientCheckerFromFn(name string, ping func(ctx context.Context) error, rawURL string) *RedisClientChecker {
	return NewRedisClientChecker(name, redisPingerFunc(ping), rawURL)
}

func (c *RedisClientChecker) Name() string { return c.name }

func (c *RedisClientChecker) Check(ctx context.Context) DependencyCheck {
	if c.ping == nil {
		return DependencyCheck{
			Status: StatusDown,
			TLS:    TLSPtr(tlsOrFalse(detectRedisTLS(c.rawURL))),
			Error:  "redis client not initialized",
		}
	}

	start := time.Now()
	err := c.ping.PingErr(ctx)
	elapsed := time.Since(start)

	tlsOn := tlsOrFalse(detectRedisTLS(c.rawURL))

	if err == nil {
		return DependencyCheck{
			Status:    StatusUp,
			LatencyMs: elapsed.Milliseconds(),
			TLS:       TLSPtr(tlsOn),
		}
	}

	return DependencyCheck{
		Status:    StatusDown,
		LatencyMs: elapsed.Milliseconds(),
		TLS:       TLSPtr(tlsOn),
		Error:     classifyErr(ctx, err),
	}
}
