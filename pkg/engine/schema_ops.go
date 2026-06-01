// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "context"

// This file implements the Engine DiscoverSchema operation: a tenant-scoped read
// of a stored connection's schema snapshot, served from an OPTIONAL cache when
// available and otherwise discovered live through the connector contract. Like
// the other connection operations it runs entirely through PORTS —
// ConnectionStore, ConnectorRegistry, and the optional SchemaCache — so
// pkg/engine stays infrastructure-free: it imports no driver, no Redis client,
// and no enginecompat adapter. The host wraps its schema cache (e.g. Redis)
// behind the SchemaCache seam; the Engine core sees only the port.
//
// Tenant scope is the SOLE isolation boundary (TenantContext.TenantID). It is
// validated BEFORE any resource access, and it is the load-bearing component of
// the cache key: every cache read and write is scoped to (tenantID, configName)
// so one tenant's cached schema is never served to — or poisoned by — another in
// an N-tenant embedded runtime.
//
// The credential is NOT decrypted inside Engine core. The descriptor that
// reaches the connector factory is the secret-free ConnectionDescriptor; the
// host's connector resolves the credential at connect time (via its injected
// resolver, which may call CredentialProtector.Reveal). Keeping ciphertext out
// of Engine core is why the operation never touches the protector directly.
//
// System-table filtering (e.g. dropping pg_*, information_schema, Oracle SYS) is
// deliberately NOT performed here. It is datasource-type-specific knowledge that
// belongs to the host's connector, which produces the secret-free SchemaSnapshot
// already filtered before it crosses the Engine boundary. Teaching the core the
// system-table conventions of every driver would reintroduce exactly the
// infrastructure coupling this extraction removes. The Engine normalizes the
// snapshot's identity (config name) and otherwise returns it canonical and
// independent from any Manager HTTP response formatting.

// DiscoverSchema returns the schema snapshot for a stored connection within the
// tenant scope. It serves a cached snapshot when an optional SchemaCache is
// configured and holds one for (tenantID, configName); otherwise it discovers
// the schema live through the connector contract and writes it through the cache.
//
// Order of operations (each gate runs before the next acquires more):
//  1. validate tenant scope BEFORE any resource access;
//  2. resolve the scoped connection via ConnectionStore (unknown / wrong-tenant
//     connections fail here as not-found, BEFORE any cache or connector access);
//  3. consult the OPTIONAL cache under the tenant scope — a hit short-circuits and
//     returns WITHOUT building a connector;
//  4. resolve the ConnectorFactory by datasource type (unknown type → the stable
//     UnknownConnectorTypeError);
//  5. build the connector (I/O-free) and ALWAYS close it via defer;
//  6. discover the schema, normalize its identity, and write it through the cache.
//
// The cache is an OPTIMIZATION: a cache read or write failure NEVER fails the
// operation. A failed read degrades to a fresh discovery; a failed write leaves
// the discovered schema returned to the caller. Connector build and discovery
// errors are mapped to safe CategoryUnavailable EngineErrors — the underlying
// error MAY embed a DSN, credential, or driver internals, so it is DELIBERATELY
// discarded from the returned message, mirroring TestConnection.
func (e *Engine) DiscoverSchema(
	ctx context.Context,
	tenant TenantContext,
	configName string,
) (SchemaSnapshot, error) {
	ctx, end := e.startSpan(ctx, "engine.schema.discover")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return SchemaSnapshot{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return SchemaSnapshot{}, err
	}

	// Resolve the connection within the tenant scope. A missing connection — or a
	// connection owned by a different tenant, which is invisible under this scope —
	// fails as not-found BEFORE any cache lookup or connector construction.
	descriptor, found, err := store.FindConnection(ctx, tenant, configName)
	if err != nil {
		return SchemaSnapshot{}, err
	}

	if !found {
		return SchemaSnapshot{}, NewEngineError(CategoryNotFound, "connection not found")
	}

	// Cache hit short-circuits live discovery. The lookup is tenant-scoped, so a
	// hit can only belong to this tenant. A cache read error is tolerated: the
	// cache is an optimization, so we fall through to a fresh discovery rather than
	// failing the operation.
	if snapshot, ok := e.cachedSchema(ctx, tenant, descriptor.ConfigName); ok {
		return snapshot, nil
	}

	// Resolve the connector factory by datasource type. An unregistered type yields
	// the stable unknown-type error, identical everywhere a registry resolves by
	// type, still BEFORE any connector construction.
	factory, err := e.requireConnectorFactory(descriptor.Type)
	if err != nil {
		return SchemaSnapshot{}, err
	}

	// Build is I/O-free per the connector contract. A build failure is a safe,
	// redacted error: the factory's raw error may embed driver internals. A buggy
	// host Build may also return (nil, nil) — a nil/typed-nil connector with no
	// error; treat that as a build failure too, BEFORE registering the deferred
	// Close, so the success path never dereferences a nil connector.
	connector, err := factory.Build(ctx, descriptor)
	if err != nil || isNilPort(connector) {
		return SchemaSnapshot{}, NewEngineError(CategoryUnavailable, "failed to build connector for connection")
	}

	// ALWAYS close the connector, on BOTH the success and failure paths. Close is
	// contractually safe and idempotent enough that a double close does not panic.
	// A close failure must not mask the primary outcome, so its error is not
	// surfaced here.
	defer func() { _ = connector.Close(ctx) }()

	// Discover the schema live. A discovery failure is mapped to a safe error: the
	// connector's raw error may embed a DSN, credential, or driver internals.
	snapshot, err := connector.DiscoverSchema(ctx)
	if err != nil {
		return SchemaSnapshot{}, NewEngineError(CategoryUnavailable, "failed to discover datasource schema")
	}

	// Normalize the snapshot's identity to the requested config name so the
	// returned and cached schema is addressable under the same identity the caller
	// used, regardless of what the connector populated.
	snapshot.ConfigName = descriptor.ConfigName

	// Write through to the cache when configured. A write failure is tolerated: the
	// discovery already succeeded, so failing to populate an optimization cache must
	// not fail the operation.
	e.cacheSchema(ctx, tenant, snapshot)

	return snapshot, nil
}

// cachedSchema reads a snapshot from the optional SchemaCache under the tenant
// scope. It returns ok=false when no cache is configured, the snapshot is absent,
// or the cache read fails — a cache read error is DELIBERATELY swallowed so a
// failing optimization cache degrades to a fresh discovery rather than failing
// the operation. The Engine core has no logger; the host adapter owns cache-error
// logging behind the seam.
func (e *Engine) cachedSchema(ctx context.Context, tenant TenantContext, configName string) (SchemaSnapshot, bool) {
	if isNilPort(e.options.schemaCache) {
		return SchemaSnapshot{}, false
	}

	snapshot, ok, err := e.options.schemaCache.GetSchema(ctx, tenant, configName)
	if err != nil || !ok {
		return SchemaSnapshot{}, false
	}

	return snapshot, true
}

// cacheSchema writes a snapshot through the optional SchemaCache under the tenant
// scope. It is a no-op when no cache is configured, and a write failure is
// DELIBERATELY swallowed: the discovery already succeeded, so a failed
// write-through must not fail the operation. The host adapter owns cache-error
// logging behind the seam.
func (e *Engine) cacheSchema(ctx context.Context, tenant TenantContext, snapshot SchemaSnapshot) {
	if isNilPort(e.options.schemaCache) {
		return
	}

	_ = e.options.schemaCache.PutSchema(ctx, tenant, snapshot)
}
