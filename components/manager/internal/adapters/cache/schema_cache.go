// Package cache provides caching abstractions for the repository layer.
//
// SchemaCache wraps a generic cache implementation and adds schema-specific
// business logic such as TTL management and cache timestamp tracking.
// This separation keeps the generic cache infrastructure agnostic to domain rules
// while providing a clean interface for schema caching operations.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/redis"
)

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
		return nil, fmt.Errorf("failed to get schema from cache: %w", err)
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

	if err := s.cache.Set(ctx, configName, *schema, ttl); err != nil {
		return fmt.Errorf("failed to set schema in cache: %w", err)
	}

	return nil
}

// Delete removes a schema from the cache.
func (s *SchemaCache) Delete(ctx context.Context, configName string) error {
	if err := s.cache.Delete(ctx, configName); err != nil {
		return fmt.Errorf("failed to delete schema from cache: %w", err)
	}

	return nil
}

// Clear removes all schema cache entries.
func (s *SchemaCache) Clear(ctx context.Context) error {
	if err := s.cache.Clear(ctx); err != nil {
		return fmt.Errorf("failed to clear schema cache: %w", err)
	}

	return nil
}

// IsHealthy checks if the cache is operational.
func (s *SchemaCache) IsHealthy(ctx context.Context) bool {
	return s.cache.IsHealthy(ctx)
}

// Close closes the underlying cache if it implements Closeable.
func (s *SchemaCache) Close() error {
	if closeable, ok := s.cache.(redis.Closeable); ok {
		if err := closeable.Close(); err != nil {
			return fmt.Errorf("failed to close schema cache: %w", err)
		}
	}

	return nil
}
