// Package redis provides Redis caching infrastructure with graceful fallback to in-memory cache.
package redis

import (
	"fmt"
	"time"

	"github.com/LerianStudio/lib-commons/v2/commons/log"
)

// NewCacheWithFallback creates a cache with Redis primary and in-memory fallback.
// If Redis connection fails, it gracefully degrades to memory-only mode.
//
// This function NEVER returns an error for Redis connection failures.
// Instead, it logs a warning and returns a memory-only cache.
//
// Parameters:
//   - cfg: Redis configuration
//   - logger: Logger instance for logging connection status
//   - ttl: Default TTL for cache entries (uses DefaultCacheTTL if <= 0)
//   - keyPrefix: Prefix for all cache keys (e.g., "fetcher:schema:")
//
// IMPORTANT: Caller MUST call Close() on the returned cache when done.
func NewCacheWithFallback[T any](
	cfg RedisConfig,
	logger log.Logger,
	ttl time.Duration,
	keyPrefix string,
) (Cache[T], error) {
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	// Try to connect to Redis
	redisConn, err := NewRedisConnection(cfg, logger)
	if err != nil {
		// Graceful degradation: log warning and use memory-only cache
		logger.Warnf("Redis connection failed, using memory-only cache: %v", err)
		return NewInMemoryCache[T](ttl, logger), nil
	}

	// Redis connected successfully - create fallback cache
	redisCache, err := NewRedisCache[T](redisConn, ttl, keyPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis cache: %w", err)
	}

	return NewFallbackCache(redisCache, logger, ttl), nil
}

// MustNewCacheWithFallback is like NewCacheWithFallback but panics on unexpected errors.
// Note: Redis connection failures do NOT cause a panic - they result in graceful degradation.
func MustNewCacheWithFallback[T any](
	cfg RedisConfig,
	logger log.Logger,
	ttl time.Duration,
	keyPrefix string,
) Cache[T] {
	cache, err := NewCacheWithFallback[T](cfg, logger, ttl, keyPrefix)
	if err != nil {
		panic(err)
	}

	return cache
}
