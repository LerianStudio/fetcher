// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory

import (
	"context"
	"sync"

	"github.com/LerianStudio/fetcher/pkg/engine"
)

// schemaKey identifies a cached schema snapshot within its tenant scope.
type schemaKey struct {
	scope      tenantScope
	configName string
}

// SchemaCache is an in-memory engine.SchemaCache. It caches schema snapshots in
// a mutex-protected map keyed by tenant scope and config name. There is no TTL
// or eviction: this is a deterministic test/embedded cache, not a production one.
type SchemaCache struct {
	mu        sync.RWMutex
	snapshots map[schemaKey]engine.SchemaSnapshot
}

// NewSchemaCache returns an empty in-memory schema cache.
func NewSchemaCache() *SchemaCache {
	return &SchemaCache{
		snapshots: make(map[schemaKey]engine.SchemaSnapshot),
	}
}

// GetSchema implements engine.SchemaCache. It returns the cached snapshot for
// the named datasource within the tenant scope and whether it was present.
func (c *SchemaCache) GetSchema(
	_ context.Context,
	tenant engine.TenantContext,
	configName string,
) (engine.SchemaSnapshot, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot, ok := c.snapshots[schemaKey{scope: scopeOf(tenant), configName: configName}]

	return snapshot, ok, nil
}

// PutSchema implements engine.SchemaCache by storing the snapshot for the tenant
// under its config name.
func (c *SchemaCache) PutSchema(
	_ context.Context,
	tenant engine.TenantContext,
	snapshot engine.SchemaSnapshot,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.snapshots[schemaKey{scope: scopeOf(tenant), configName: snapshot.ConfigName}] = snapshot

	return nil
}
