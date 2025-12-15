package redis

import (
	"context"
	"sync"
	"time"

	"github.com/LerianStudio/lib-commons/v2/commons"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"go.opentelemetry.io/otel/attribute"
)

// cacheEntry holds a cached value with its expiration time.
type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

// InMemoryCache is a generic in-memory implementation of Cache.
type InMemoryCache[T any] struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry[T]
	ttl     time.Duration
	logger  log.Logger
	stopCh  chan struct{}
	closed  bool
}

// NewInMemoryCache creates a new generic in-memory cache.
// The cache automatically cleans up expired entries every minute.
//
// IMPORTANT: Caller MUST call Close() when done to stop the cleanup goroutine.
// Failing to call Close() will result in a goroutine leak.
// Close() is idempotent and safe to call multiple times.
func NewInMemoryCache[T any](ttl time.Duration, logger log.Logger) *InMemoryCache[T] {
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	c := &InMemoryCache[T]{
		entries: make(map[string]*cacheEntry[T]),
		ttl:     ttl,
		logger:  logger,
		stopCh:  make(chan struct{}),
	}

	go c.cleanupExpired()

	return c
}

// Get retrieves a cached value by key.
func (c *InMemoryCache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.in_memory.get")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.cache.key", key),
		attribute.String("app.request.request_id", reqID),
	)

	var zero T

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		span.SetAttributes(attribute.Bool("app.cache.hit", false))
		logger.Debugf("in-memory cache miss for key %s", key)
		return zero, false, nil
	}

	if time.Now().UTC().After(entry.expiresAt) {
		span.SetAttributes(attribute.Bool("app.cache.hit", false))
		logger.Debugf("in-memory cache expired for key %s", key)
		return zero, false, nil
	}

	span.SetAttributes(attribute.Bool("app.cache.hit", true))
	logger.Debugf("in-memory cache hit for key %s", key)
	return entry.value, true, nil
}

// Set stores a value in the cache.
func (c *InMemoryCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "cache.in_memory.set")
	defer span.End()

	if ttl <= 0 {
		ttl = c.ttl
	}

	span.SetAttributes(
		attribute.String("app.cache.key", key),
		attribute.String("app.request.request_id", reqID),
		attribute.Int64("app.cache.ttl_seconds", int64(ttl.Seconds())),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry[T]{
		value:     value,
		expiresAt: time.Now().UTC().Add(ttl),
	}

	logger.Debugf("in-memory cached key %s with TTL %v", key, ttl)
	return nil
}

// Delete removes a value from the cache.
func (c *InMemoryCache[T]) Delete(ctx context.Context, key string) error {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "cache.in_memory.delete")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.cache.key", key),
		attribute.String("app.request.request_id", reqID),
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
	return nil
}

// Clear removes all cache entries.
func (c *InMemoryCache[T]) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry[T])
	return nil
}

// IsHealthy always returns true for in-memory cache.
func (c *InMemoryCache[T]) IsHealthy(ctx context.Context) bool {
	return true
}

// Close stops the cleanup goroutine.
// Close is idempotent and safe to call multiple times.
func (c *InMemoryCache[T]) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.stopCh)

	return nil
}

// cleanupExpired periodically removes expired entries.
func (c *InMemoryCache[T]) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now().UTC()
			for key, entry := range c.entries {
				if now.After(entry.expiresAt) {
					delete(c.entries, key)
					c.logger.Debugf("in-memory cache cleanup removed expired key %s", key)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}
