package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/LerianStudio/lib-commons/v4/commons"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
)

// DefaultCacheTTL is the default TTL for cached entries (5 minutes).
const DefaultCacheTTL = 5 * time.Minute

// RedisCache is a generic Redis implementation of Cache.
type RedisCache[T any] struct {
	client    *redis.Client
	ttl       time.Duration
	keyPrefix string
}

// NewRedisCache creates a new generic Redis cache.
// Parameters:
//   - conn: Redis connection
//   - ttl: Default TTL for cache entries (uses DefaultCacheTTL if <= 0)
//   - keyPrefix: Prefix for all cache keys (e.g., "fetcher:schema:")
func NewRedisCache[T any](conn *RedisConnection, ttl time.Duration, keyPrefix string) *RedisCache[T] {
	if conn == nil || conn.Client == nil {
		panic("redis connection and client must not be nil")
	}

	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	return &RedisCache[T]{
		client:    conn.Client,
		ttl:       ttl,
		keyPrefix: keyPrefix,
	}
}

// cacheKey generates the full Redis key.
func (c *RedisCache[T]) cacheKey(key string) string {
	return fmt.Sprintf("%s%s", c.keyPrefix, key)
}

// Get retrieves a cached value by key.
func (c *RedisCache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.redis.get")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.cache.key", key),
		attribute.String("app.request.request_id", reqID),
	)

	var zero T

	fullKey := c.cacheKey(key)

	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			span.SetAttributes(attribute.Bool("app.cache.hit", false))
			logger.Log(context.Background(), libLog.LevelDebug, fmt.Sprintf("cache miss for key %s", key))

			return zero, false, nil // Cache miss - not an error
		}

		span.SetAttributes(attribute.Bool("app.cache.hit", false))
		libOpentelemetry.HandleSpanError(span, "failed to get from cache", err)
		logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("error retrieving from cache key %s: %v", key, err))

		return zero, false, fmt.Errorf("failed to get from cache: %w", err)
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		span.SetAttributes(attribute.Bool("app.cache.hit", false))
		libOpentelemetry.HandleSpanError(span, "failed to unmarshal cached value", err)
		logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("error unmarshaling cache value for key %s: %v", key, err))

		return zero, false, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	span.SetAttributes(attribute.Bool("app.cache.hit", true))
	logger.Log(context.Background(), libLog.LevelDebug, fmt.Sprintf("cache hit for key %s", key))

	return value, true, nil
}

// Set stores a value in the cache.
func (c *RedisCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.redis.set")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.cache.key", key),
		attribute.Int64("app.cache.ttl_seconds", int64(ttl.Seconds())),
		attribute.String("app.request.request_id", reqID),
	)

	if ttl <= 0 {
		ttl = c.ttl
	}

	data, err := json.Marshal(value)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to marshal value", err)
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("error marshaling value for cache key %s: %v", key, err))

		return fmt.Errorf("failed to marshal value: %w", err)
	}

	fullKey := c.cacheKey(key)
	if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to set cache value", err)
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("error storing in cache key %s: %v", key, err))

		return fmt.Errorf("failed to store in cache: %w", err)
	}

	logger.Log(context.Background(), libLog.LevelDebug, fmt.Sprintf("cached key %s with TTL %v", key, ttl))

	return nil
}

// Delete removes a value from the cache.
func (c *RedisCache[T]) Delete(ctx context.Context, key string) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.redis.delete")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.cache.key", key),
		attribute.String("app.request.request_id", reqID),
	)

	fullKey := c.cacheKey(key)
	if err := c.client.Del(ctx, fullKey).Err(); err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to delete cache key", err)
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("error deleting cache key %s: %v", key, err))

		return fmt.Errorf("failed to delete from cache: %w", err)
	}

	logger.Log(context.Background(), libLog.LevelDebug, fmt.

		// Clear removes all cache entries with the configured prefix.
		Sprintf("deleted cache key %s", key))

	return nil
}

func (c *RedisCache[T]) Clear(ctx context.Context) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.redis.clear")
	defer span.End()

	pattern := fmt.Sprintf("%s*", c.keyPrefix)

	span.SetAttributes(
		attribute.String("app.cache.key_prefix", c.keyPrefix),
		attribute.String("app.cache.clear_pattern", pattern),
		attribute.String("app.request.request_id", reqID),
	)

	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("error deleting key %s: %v", iter.Val(), err))
		}
	}

	if err := iter.Err(); err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to clear cache keys", err)
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("error scanning keys for clear: %v", err))

		return fmt.Errorf("failed to clear cache: %w", err)
	}

	logger.Log(context.Background(), libLog.LevelInfo,

		// IsHealthy checks if Redis is operational.
		"cache cleared")

	return nil
}

func (c *RedisCache[T]) IsHealthy(ctx context.Context) bool {
	return c.client.Ping(ctx).Err() == nil
}

// Close closes the Redis connection.
func (c *RedisCache[T]) Close() error {
	return c.client.Close()
}
