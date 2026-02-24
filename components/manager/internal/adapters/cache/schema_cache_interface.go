// Package cache provides caching adapter implementations for port interfaces.
package cache

import (
	portCache "github.com/LerianStudio/fetcher/pkg/ports/cache"
)

// SchemaCacheRepository is an alias for the port interface.
// This keeps backward compatibility while the canonical definition
// lives in the ports layer (pkg/ports/cache).
type SchemaCacheRepository = portCache.SchemaCacheRepository

// DefaultSchemaCacheTTL is re-exported from the ports layer for backward compatibility.
const DefaultSchemaCacheTTL = portCache.DefaultSchemaCacheTTL
