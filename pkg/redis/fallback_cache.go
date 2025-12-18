package redis

import (
	"context"
	"sync"
	"time"

	"github.com/LerianStudio/lib-commons/v2/commons"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// HealthCheckInterval defines how often the fallback cache checks Redis health.
	HealthCheckInterval = 10 * time.Second
)

// FallbackCache wraps a Redis cache with in-memory fallback.
// It automatically monitors Redis health and switches between backends.
type FallbackCache[T any] struct {
	redis    *RedisCache[T]
	inMemory *InMemoryCache[T]
	logger   log.Logger
	useRedis bool
	mu       sync.RWMutex
	stopCh   chan struct{}
	closed   bool
}

// NewFallbackCache creates a new cache with Redis primary and in-memory fallback.
//
// IMPORTANT: Caller MUST call Close() when done to stop the health monitor goroutine.
// Failing to call Close() will result in a goroutine leak.
// Close() is idempotent and safe to call multiple times.
func NewFallbackCache[T any](redisCache *RedisCache[T], logger log.Logger, ttl time.Duration) *FallbackCache[T] {
	if ttl <= 0 {
		logger.Warnf("invalid TTL %v, using default %v", ttl, DefaultCacheTTL)
		ttl = DefaultCacheTTL
	}

	logger.Infof("creating fallback cache with TTL %v", ttl)

	fc := &FallbackCache[T]{
		redis:    redisCache,
		inMemory: NewInMemoryCache[T](ttl, logger),
		logger:   logger,
		useRedis: true,
		stopCh:   make(chan struct{}),
	}

	go fc.monitorRedisHealth()

	return fc
}

// Get retrieves a cached value, trying Redis first then in-memory.
func (c *FallbackCache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.fallback.get")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.cache.key", key),
		attribute.String("app.request.request_id", reqID),
	)

	c.mu.RLock()
	useRedis := c.useRedis
	c.mu.RUnlock()

	if useRedis {
		span.SetAttributes(attribute.String("app.cache.backend", "redis"))
		value, found, err := c.redis.Get(ctx, key)
		if err != nil {
			logger.Warnf("redis cache error for key %s, falling back to in-memory: %v", key, err)
		} else if found {
			return value, true, nil
		}
	} else {
		span.SetAttributes(attribute.String("app.cache.backend", "memory"))
	}

	return c.inMemory.Get(ctx, key)
}

// Set stores a value in both Redis (if available) and in-memory.
func (c *FallbackCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.fallback.set")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.cache.key", key),
		attribute.String("app.request.request_id", reqID),
	)

	// Always store in in-memory for fallback
	if err := c.inMemory.Set(ctx, key, value, ttl); err != nil {
		logger.Warnf("failed to set in in-memory cache: %v", err)
	}

	c.mu.RLock()
	useRedis := c.useRedis
	c.mu.RUnlock()

	if useRedis {
		span.SetAttributes(attribute.String("app.cache.backend", "redis+memory"))
		if err := c.redis.Set(ctx, key, value, ttl); err != nil {
			logger.Warnf("failed to set in redis, using in-memory only: %v", err)
		}
	} else {
		span.SetAttributes(attribute.String("app.cache.backend", "memory"))
	}

	return nil
}

// Delete removes a value from both caches.
func (c *FallbackCache[T]) Delete(ctx context.Context, key string) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.fallback.delete")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.cache.key", key),
		attribute.String("app.request.request_id", reqID),
	)

	_ = c.inMemory.Delete(ctx, key)

	c.mu.RLock()
	useRedis := c.useRedis
	c.mu.RUnlock()

	if useRedis {
		span.SetAttributes(attribute.String("app.cache.backend", "redis+memory"))
		if err := c.redis.Delete(ctx, key); err != nil {
			logger.Warnf("failed to delete from redis: %v", err)
		}
	} else {
		span.SetAttributes(attribute.String("app.cache.backend", "memory"))
	}

	return nil
}

// Clear removes all cached entries from both caches.
func (c *FallbackCache[T]) Clear(ctx context.Context) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.fallback.clear")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	_ = c.inMemory.Clear(ctx)

	c.mu.RLock()
	useRedis := c.useRedis
	c.mu.RUnlock()

	if useRedis {
		span.SetAttributes(attribute.String("app.cache.backend", "redis+memory"))
		if err := c.redis.Clear(ctx); err != nil {
			logger.Warnf("failed to clear redis cache: %v", err)
		}
	} else {
		span.SetAttributes(attribute.String("app.cache.backend", "memory"))
	}

	return nil
}

// IsHealthy returns true if at least one cache backend is operational.
// It checks both Redis and in-memory cache health status.
func (c *FallbackCache[T]) IsHealthy(ctx context.Context) bool {
	// In-memory cache is always healthy (it's local)
	inMemoryHealthy := c.inMemory.IsHealthy(ctx)

	// Check Redis health
	redisHealthy := c.redis.IsHealthy(ctx)

	// FallbackCache is healthy if at least one backend is operational
	return inMemoryHealthy || redisHealthy
}

// Close stops the health monitor and cleanup goroutines.
// Close is idempotent and safe to call multiple times.
func (c *FallbackCache[T]) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	close(c.stopCh)

	err := c.inMemory.Close()
	if err != nil {
		c.logger.Warnf("failed to close in-memory cache: %v", err)
	}

	err = c.redis.Close()
	if err != nil {
		c.logger.Warnf("failed to close redis cache: %v", err)
	}

	return err
}

// monitorRedisHealth periodically checks Redis health and updates useRedis flag.
func (c *FallbackCache[T]) monitorRedisHealth() {
	ticker := time.NewTicker(HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			healthy := c.redis.IsHealthy(ctx)
			cancel()

			c.mu.Lock()
			if healthy != c.useRedis {
				if healthy {
					c.logger.Info("redis connection restored, switching to redis cache")
				} else {
					c.logger.Warn("redis connection lost, falling back to in-memory cache")
				}
				c.useRedis = healthy
			}
			c.mu.Unlock()
		case <-c.stopCh:
			c.logger.Debug("stopping redis health monitor goroutine")
			return
		}
	}
}
