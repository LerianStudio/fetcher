// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package schemacompat

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"
)

// resolvedConnectionsKey is the context key under which the host seeds the
// connections it ALREADY resolved for this request. Keeping it a private type
// prevents collision with any other context value.
type resolvedConnectionsKey struct{}

// schemaScopeKey is the context key under which the host seeds the schema-name
// scope for discovery, keyed by config name. The connector reads it to pass the
// exact schema list the request references to the underlying datasource —
// preserving the legacy multi-schema discovery (e.g. mixed qualified +
// unqualified PostgreSQL tables fetching both "custom" and "public").
type schemaScopeKey struct{}

// WithSchemaScope seeds the request context with the schema-name list for a
// config name. An empty list means "no explicit scope" (the connector lets the
// datasource adapter apply its default).
func WithSchemaScope(ctx context.Context, configName string, schemas []string) context.Context {
	existing, _ := ctx.Value(schemaScopeKey{}).(map[string][]string)

	byConfig := make(map[string][]string, len(existing)+1)
	for k, v := range existing {
		byConfig[k] = v
	}

	byConfig[configName] = schemas

	return context.WithValue(ctx, schemaScopeKey{}, byConfig)
}

// SchemaScope reads the seeded schema-name list for a config name. It is the
// PUBLIC reader of the same scope seeded by WithSchemaScope, so a sibling host
// adapter (e.g. the extraction datasource connector) can honor the request-scoped
// discovery scope WITHOUT a parallel scope-plumbing of its own. Returns nil when
// no scope was seeded for the config name.
func SchemaScope(ctx context.Context, configName string) []string {
	byConfig, ok := ctx.Value(schemaScopeKey{}).(map[string][]string)
	if !ok {
		return nil
	}

	return byConfig[configName]
}

// WithResolvedConnections seeds the request context with the connections the
// host resolved (internal via tenant-manager, external via the repository). The
// schema Engine's ConnectionStore reads from this seed so it never re-resolves —
// avoiding double tenant-manager / MongoDB calls and keeping internal-datasource
// resolution a host concern (the Engine core never imports tenant-manager).
//
// Connections are keyed by config name; a duplicate config name keeps the first
// seeded connection. A nil or empty slice yields an empty seed (every lookup
// reports not-found).
func WithResolvedConnections(ctx context.Context, connections []*model.Connection) context.Context {
	byConfig := make(map[string]*model.Connection, len(connections))

	for _, conn := range connections {
		if conn == nil {
			continue
		}

		if _, exists := byConfig[conn.ConfigName]; !exists {
			byConfig[conn.ConfigName] = conn
		}
	}

	return context.WithValue(ctx, resolvedConnectionsKey{}, byConfig)
}

// resolvedConnection reads a seeded connection by config name from the context.
func resolvedConnection(ctx context.Context, configName string) (*model.Connection, bool) {
	byConfig, ok := ctx.Value(resolvedConnectionsKey{}).(map[string]*model.Connection)
	if !ok {
		return nil, false
	}

	conn, found := byConfig[configName]

	return conn, found
}

// ConnectionStore is the schema Engine's engine.ConnectionStore. It resolves a
// config name to the rich connection the HOST already resolved for this request
// (seeded via WithResolvedConnections) and packs that record into the descriptor's
// opaque host payload using the connectioncompat round-trip, so the schema
// ConnectorFactory can rebuild the exact connection — internal or external —
// without the Engine ever re-resolving or learning about tenant-manager.
//
// Only FindConnection is meaningful: the schema Engine performs discovery only,
// never connection lifecycle writes. The remaining engine.ConnectionStore methods
// are present for port-completeness and return a stable validation error if ever
// called, which they are not on the schema path.
type ConnectionStore struct{}

// NewConnectionStore builds the request-scoped schema connection store. It holds
// no state; the per-request connections travel in the context seed.
func NewConnectionStore() *ConnectionStore {
	return &ConnectionStore{}
}

// FindConnection implements engine.ConnectionStore by returning the host-resolved
// connection seeded into the context for this request. A missing connection
// reports found=false so the Engine maps it to its scoped not-found rule.
func (s *ConnectionStore) FindConnection(
	ctx context.Context,
	_ engine.TenantContext,
	configName string,
) (engine.ConnectionDescriptor, bool, error) {
	conn, found := resolvedConnection(ctx, configName)
	if !found || conn == nil {
		return engine.ConnectionDescriptor{}, false, nil
	}

	return connectioncompat.DescriptorFromConnection(conn), true, nil
}

// unsupported is the stable error returned by the lifecycle methods the schema
// Engine never calls. Discovery is the only operation routed through this store.
func unsupported() error {
	return engine.NewEngineError(engine.CategoryValidation, "schema connection store supports discovery only")
}

// Create is unsupported on the schema path.
func (s *ConnectionStore) Create(context.Context, engine.TenantContext, engine.ConnectionDescriptor, *engine.ProtectedCredential) error {
	return unsupported()
}

// Update is unsupported on the schema path.
func (s *ConnectionStore) Update(context.Context, engine.TenantContext, engine.ConnectionDescriptor, *engine.ProtectedCredential) error {
	return unsupported()
}

// Delete is unsupported on the schema path.
func (s *ConnectionStore) Delete(context.Context, engine.TenantContext, string) error {
	return unsupported()
}

// List is unsupported on the schema path.
func (s *ConnectionStore) List(context.Context, engine.TenantContext) ([]engine.ConnectionDescriptor, error) {
	return nil, unsupported()
}

// FindByID is unsupported on the schema path (discovery addresses by config name).
func (s *ConnectionStore) FindByID(context.Context, engine.TenantContext, string) (engine.ConnectionDescriptor, bool, error) {
	return engine.ConnectionDescriptor{}, false, unsupported()
}

// UpdateByID is unsupported on the schema path.
func (s *ConnectionStore) UpdateByID(context.Context, engine.TenantContext, string, engine.ConnectionDescriptor, *engine.ProtectedCredential) error {
	return unsupported()
}

// DeleteByID is unsupported on the schema path.
func (s *ConnectionStore) DeleteByID(context.Context, engine.TenantContext, string) error {
	return unsupported()
}

// ListPaged is unsupported on the schema path.
func (s *ConnectionStore) ListPaged(context.Context, engine.TenantContext, engine.ConnectionListParams) (engine.ConnectionPage, error) {
	return engine.ConnectionPage{}, unsupported()
}
