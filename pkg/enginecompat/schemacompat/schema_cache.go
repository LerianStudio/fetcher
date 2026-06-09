// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

// Package schemacompat is the host-side compatibility seam that lets the
// Manager route schema DISCOVERY (and only discovery) through the embedded
// Engine while keeping connection resolution, schema VALIDATION, DB-type name
// normalization, and plugin_crm policy on the host. It adapts three host
// capabilities to the Engine's optional ports:
//
//   - SchemaCache wraps the Manager's Redis-backed schema cache behind the
//     engine.SchemaCache port, so the Engine sees only the port and the Redis
//     client never crosses the pkg/engine boundary;
//   - ConnectorFactory unpacks the rich *model.Connection the host already
//     resolved (carried verbatim through the descriptor's opaque host payload)
//     and builds it through the host's existing datasource factory, preserving
//     the legacy decrypt + connect behavior byte-for-byte;
//   - ConnectionStore resolves a config name to the connection the HOST already
//     resolved for this request (internal via tenant-manager or external via
//     MongoDB), seeded into the request context — so the Engine never re-resolves
//     and tenant-manager never has to be dragged into the Engine core.
//
// IMPORT DIRECTION IS ONE-WAY. This package MAY import pkg/engine, the host
// model, the host cache port, and connectioncompat (to reuse the rich-record
// round-trip). pkg/engine MUST NOT import this package; the dependency boundary
// test in pkg/engine keeps that invariant enforced.
package schemacompat

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	cachePort "github.com/LerianStudio/fetcher/v2/pkg/ports/cache"
)

// SchemaCache adapts the Manager's configName-keyed schema cache port to the
// Engine's tenant-scoped engine.SchemaCache port. The underlying Redis cache is
// keyed by config name (the Manager's existing, byte-identical cache behavior);
// the tenant argument is accepted to satisfy the port but the host cache key is
// preserved so cached entries written by the legacy path remain compatible.
//
// A nil cache yields a nil adapter so the caller can treat "no cache" as "no
// schema cache" (the Engine port is optional and degrades to live discovery).
type SchemaCache struct {
	cache cachePort.SchemaCacheRepository
	ttl   time.Duration
}

// NewSchemaCache builds the adapter over the Manager's schema cache repository.
// A nil cache yields a nil adapter.
func NewSchemaCache(cache cachePort.SchemaCacheRepository, ttl time.Duration) *SchemaCache {
	if cache == nil {
		return nil
	}

	if ttl <= 0 {
		ttl = cachePort.DefaultSchemaCacheTTL
	}

	return &SchemaCache{cache: cache, ttl: ttl}
}

// GetSchema implements engine.SchemaCache. A cache miss (the host port returns
// (nil, nil)) reports ok=false. A cache error is surfaced to the Engine, which
// DELIBERATELY degrades a read error to a fresh discovery, preserving the legacy
// "cache error → continue to fetch" behavior.
func (c *SchemaCache) GetSchema(
	ctx context.Context,
	_ engine.TenantContext,
	configName string,
) (engine.SchemaSnapshot, bool, error) {
	cached, err := c.cache.Get(ctx, configName)
	if err != nil {
		return engine.SchemaSnapshot{}, false, err
	}

	if cached == nil {
		return engine.SchemaSnapshot{}, false, nil
	}

	return SnapshotFromDataSourceSchema(cached), true, nil
}

// PutSchema implements engine.SchemaCache. It converts the Engine snapshot back
// into the host's DataSourceSchema and writes it through the host cache with the
// configured TTL. A write error is surfaced to the Engine, which tolerates it
// (the discovered snapshot is still returned to the caller).
func (c *SchemaCache) PutSchema(
	ctx context.Context,
	_ engine.TenantContext,
	snapshot engine.SchemaSnapshot,
) error {
	schema := DataSourceSchemaFromSnapshot(snapshot)

	return c.cache.Set(ctx, snapshot.ConfigName, schema, c.ttl)
}

// SnapshotFromDataSourceSchema converts a host *model.DataSourceSchema into a
// secret-free Engine SchemaSnapshot with NO filtering and NO normalization (the
// cache round-trip preserves identifiers verbatim). It is a thin wrapper over the
// single forward builder. A nil schema yields an empty snapshot.
func SnapshotFromDataSourceSchema(schema *model.DataSourceSchema) engine.SchemaSnapshot {
	configName := ""
	if schema != nil {
		configName = schema.ConfigName
	}

	return BuildSnapshot(configName, "", schema, SnapshotOptions{})
}

// DataSourceSchemaFromSnapshot converts an Engine SchemaSnapshot back into the
// host *model.DataSourceSchema the Manager's validation, normalization, and
// plugin_crm policy operate on. The result always carries a non-nil Tables map.
func DataSourceSchemaFromSnapshot(snapshot engine.SchemaSnapshot) *model.DataSourceSchema {
	schema := model.NewDataSourceSchema(snapshot.ConfigName)

	for _, table := range snapshot.Tables {
		schema.AddTable(table.Name, table.Fields)
	}

	return schema
}
