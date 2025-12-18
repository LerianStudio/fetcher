// Package cache provides caching abstractions for the repository layer.
//
// SchemaCache wraps a generic cache implementation and adds schema-specific
// business logic such as TTL management and cache timestamp tracking.
// This separation keeps the generic cache infrastructure agnostic to domain rules
// while providing a clean interface for schema caching operations.
package cache

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/redis"
)

// DefaultSchemaCacheTTL is the default TTL for cached schemas (5 minutes).
const DefaultSchemaCacheTTL = 5 * time.Minute

// SchemaCache wraps a generic cache and adds schema-specific business logic.
// This keeps the generic cache infrastructure agnostic to domain rules.
type SchemaCache struct {
	cache redis.Cache[model.DataSourceSchema]
	ttl   time.Duration
}

// NewSchemaCache creates a new schema cache wrapper.
func NewSchemaCache(cache redis.Cache[model.DataSourceSchema], ttl time.Duration) *SchemaCache {
	if ttl <= 0 {
		ttl = DefaultSchemaCacheTTL
	}

	return &SchemaCache{
		cache: cache,
		ttl:   ttl,
	}
}

// Get retrieves a cached schema by configName.
// Returns (nil, nil) on cache miss, (nil, error) on actual errors, (schema, nil) on hit.
// Business logic: Sets CachedAt timestamp when returning from cache.
func (s *SchemaCache) Get(ctx context.Context, configName string) (*model.DataSourceSchema, error) {
	schema, found, err := s.cache.Get(ctx, configName)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	// Return pointer to the schema
	return &schema, nil
}

// Set stores a schema in the cache with the specified TTL.
// Business logic: Sets CachedAt and ExpiresAt timestamps before caching.
func (s *SchemaCache) Set(ctx context.Context, configName string, schema *model.DataSourceSchema, ttl time.Duration) error {
	if schema == nil {
		return nil
	}

	if ttl <= 0 {
		ttl = s.ttl
	}

	schema.SetCacheTTL(ttl)

	return s.cache.Set(ctx, configName, *schema, ttl)
}

// Delete removes a schema from the cache.
func (s *SchemaCache) Delete(ctx context.Context, configName string) error {
	return s.cache.Delete(ctx, configName)
}

// Clear removes all schema cache entries.
func (s *SchemaCache) Clear(ctx context.Context) error {
	return s.cache.Clear(ctx)
}

// IsHealthy checks if the cache is operational.
func (s *SchemaCache) IsHealthy(ctx context.Context) bool {
	return s.cache.IsHealthy(ctx)
}

// Close closes the underlying cache if it implements Closeable.
func (s *SchemaCache) Close() error {
	if closeable, ok := s.cache.(redis.Closeable); ok {
		return closeable.Close()
	}
	return nil
}
