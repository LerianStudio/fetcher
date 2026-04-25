package readyz

import (
	"context"
	"time"
)

// RedisPinger is the narrow surface the RedisClientChecker needs from its
// underlying client. *redis.Client's Ping(ctx) returns a *redis.StatusCmd —
// callers adapt that to (error) via the small closure in NewRedisClientChecker.
type RedisPinger interface {
	PingErr(ctx context.Context) error
}

// redisPingerFunc is a function-typed adapter implementing RedisPinger. It
// keeps the bootstrap code free of a wrapper type just to satisfy the
// interface.
type redisPingerFunc func(ctx context.Context) error

func (f redisPingerFunc) PingErr(ctx context.Context) error { return f(ctx) }

// RedisClientChecker probes a Redis instance with PING and reports latency.
// The same checker type serves two roles in fetcher:
//
//   - name="redis" for the schema-cache Redis (always global, single-tenant or
//     multi-tenant).
//   - name="multi_tenant_redis" for the event-discovery Redis used by the
//     multi-tenant middleware (present only when MULTI_TENANT_ENABLED=true
//     and MULTI_TENANT_REDIS_HOST is configured).
//
// The dep name is caller-supplied so the /readyz response distinguishes them
// for dashboarding. Both report TLS via the Gate 3 detector against the
// supplied URL.
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

// NewRedisClientCheckerFromFn constructs a checker from a plain function. This
// is the expected bootstrap form because *redis.Client's Ping method returns
// a *redis.StatusCmd, so callers write:
//
//	readyz.NewRedisClientCheckerFromFn("redis",
//	    func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
//	    redisURL)
func NewRedisClientCheckerFromFn(name string, ping func(ctx context.Context) error, rawURL string) *RedisClientChecker {
	return NewRedisClientChecker(name, redisPingerFunc(ping), rawURL)
}

// Name returns the caller-supplied dep identifier.
func (c *RedisClientChecker) Name() string { return c.name }

// Check executes PING under the caller-supplied deadline. See classifyErr for
// the error-vocabulary contract.
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
