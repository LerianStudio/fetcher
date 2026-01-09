package redis

import (
	"context"
	"time"
)

// Cache defines a generic caching interface that is agnostic to business logic.
// The type parameter T represents the cached value type.
//
// Note: mockgen does not support generic interfaces. For testing, create concrete
// mock implementations for specific type parameters (e.g., mockGenericCache in tests).
type Cache[T any] interface {
	// Get retrieves a cached value by key.
	// Returns:
	//   - (value, true, nil) on cache hit
	//   - (zero, false, nil) on cache miss
	//   - (zero, false, error) on actual errors (network, deserialization)
	Get(ctx context.Context, key string) (T, bool, error)

	// Set stores a value in the cache with the specified TTL.
	// If ttl <= 0, the implementation's default TTL is used.
	Set(ctx context.Context, key string, value T, ttl time.Duration) error

	// Delete removes a value from the cache.
	Delete(ctx context.Context, key string) error

	// Clear removes all entries with the configured key prefix.
	Clear(ctx context.Context) error

	// IsHealthy checks if the cache is operational.
	IsHealthy(ctx context.Context) bool
}

// Closeable is an optional interface for caches that need cleanup.
type Closeable interface {
	Close() error
}
