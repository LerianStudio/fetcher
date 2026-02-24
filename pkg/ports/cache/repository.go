// Package cache defines the domain port interface for schema caching.
package cache

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
)

// DefaultSchemaCacheTTL is the default TTL for cached schemas (5 minutes).
const DefaultSchemaCacheTTL = 5 * time.Minute

// SchemaCacheRepository defines the contract for schema caching.
// This interface is used by services that need to cache schemas.
//
//go:generate mockgen --destination=repository.mock.go --package=cache . SchemaCacheRepository
type SchemaCacheRepository interface {
	// Get retrieves a cached schema by configName.
	// Returns (nil, nil) on cache miss.
	// Returns (nil, error) on actual errors.
	// Returns (schema, nil) on cache hit.
	Get(ctx context.Context, configName string) (*model.DataSourceSchema, error)

	// Set stores a schema in the cache with the specified TTL.
	Set(ctx context.Context, configName string, schema *model.DataSourceSchema, ttl time.Duration) error

	// Delete removes a schema from the cache.
	Delete(ctx context.Context, configName string) error

	// Clear removes all schema cache entries.
	Clear(ctx context.Context) error

	// IsHealthy checks if the cache is operational.
	IsHealthy(ctx context.Context) bool

	// Close closes the cache.
	Close() error
}
