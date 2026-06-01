// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "context"

// The interfaces in this file are the host-provided capability ports of the
// embedded Engine. The Engine core depends only on these abstractions; concrete
// adapters (PostgreSQL, MongoDB, Redis, RabbitMQ, S3, SeaweedFS, Fiber) live in
// the host and are injected through options. Keeping them as interfaces is what
// allows pkg/engine to stay importable without any infrastructure package.
//
// Ports split into two groups:
//   - REQUIRED: the Engine cannot operate without them and validates their
//     presence at construction (New).
//   - OPTIONAL: the Engine degrades gracefully without them; New succeeds when
//     they are omitted.

// Connector is a host-provided datasource connector. It is intentionally opaque
// at the contract layer for ST-T002-02: later subtasks define the connect,
// discover, plan, and execute behavior. The Engine resolves connectors by
// type through the ConnectorRegistry.
type Connector any

// ConnectorRegistry resolves a datasource Connector by its type identifier.
// It is a REQUIRED capability: an Engine with no way to obtain connectors
// cannot perform any extraction, so New rejects a missing registry.
type ConnectorRegistry interface {
	// Connector returns the connector registered for the given datasource type
	// and reports whether one exists. It performs no I/O.
	Connector(datasourceType string) (Connector, bool)
}

// CredentialProtector encrypts and decrypts credential material on behalf of
// the host. It is CONDITIONALLY REQUIRED: when encrypted persistence is enabled
// (WithEncryptedPersistence(true)) the Engine refuses to construct without one,
// because persisting connection secrets without a protector would store
// plaintext credentials. When encrypted persistence is disabled it is optional.
type CredentialProtector interface {
	// Protect encrypts the given plaintext for the tenant and returns the
	// protected material together with the key version that protected it. The
	// HOST owns key management (derivation, rotation, storage); the Engine only
	// calls this seam and records the returned key version as secret-free
	// metadata. Implementations MUST NOT log or leak the plaintext, and MUST NOT
	// embed the plaintext or ciphertext in their returned error.
	Protect(ctx context.Context, tenant TenantContext, plaintext []byte) (protected []byte, keyVersion int, err error)
	// Reveal decrypts the given ciphertext for the tenant using the key version
	// that protected it, so the host can resolve a rotated key.
	Reveal(ctx context.Context, tenant TenantContext, ciphertext []byte, keyVersion int) (plaintext []byte, err error)
}

// ProtectedCredential is the secret-bearing sidecar the Engine hands to the
// ConnectionStore alongside the secret-free ConnectionDescriptor. It carries the
// already-protected (encrypted) credential material plus the key version that
// protected it — never plaintext. Keeping it OUT of ConnectionDescriptor
// preserves the descriptor's "freely loggable, secret-free" output contract:
// the descriptor is what leaves the Engine to callers, while ProtectedCredential
// only ever travels inward to the persistence seam.
type ProtectedCredential struct {
	// Ciphertext is the host-protected credential material. It is opaque to the
	// Engine and MUST NOT be logged or returned across the public boundary.
	Ciphertext []byte
	// KeyVersion records which key version protected Ciphertext, mirroring the
	// secret-free metadata on ConnectionDescriptor.
	KeyVersion int
}

// ConnectionStore is an OPTIONAL port for persisting and resolving connection
// descriptors owned by a tenant/product. When absent, the host is expected to
// supply connection inputs directly per request and the Engine's connection
// lifecycle operations are unavailable.
//
// The store is the only persistence seam the Engine connection operations use;
// the Engine never embeds MongoDB, SQL, or any host repository. Implementations
// MUST scope every record by the supplied TenantContext so connections owned by
// one tenant/product are never visible to another. They MUST NOT return secret
// material — ConnectionDescriptor carries none.
type ConnectionStore interface {
	// FindConnection returns the descriptor for the named connection and whether
	// it exists for the given tenant. It never returns secret material.
	FindConnection(ctx context.Context, tenant TenantContext, configName string) (ConnectionDescriptor, bool, error)
	// Create persists a new connection descriptor for the tenant. It MUST return
	// a CategoryValidation *EngineError when a connection with the same config
	// name already exists within the tenant scope.
	//
	// credential carries the already-protected secret material to persist
	// alongside the descriptor. It is nil when no secret accompanies the
	// connection (e.g. encrypted persistence disabled, or no password supplied);
	// implementations MUST persist the ciphertext as opaque bytes and MUST NOT
	// log it.
	Create(ctx context.Context, tenant TenantContext, descriptor ConnectionDescriptor, credential *ProtectedCredential) error
	// Update replaces an existing connection descriptor for the tenant. It MUST
	// return a CategoryNotFound *EngineError when no connection with that config
	// name exists within the tenant scope.
	//
	// credential is the newly-protected secret to persist. A nil credential means
	// the caller supplied NO password change: implementations MUST leave any
	// existing stored secret intact and MUST NOT wipe it.
	Update(ctx context.Context, tenant TenantContext, descriptor ConnectionDescriptor, credential *ProtectedCredential) error
	// Delete removes a connection descriptor for the tenant. It MUST return a
	// CategoryNotFound *EngineError when no connection with that config name
	// exists within the tenant scope. After deletion the connection MUST be
	// invisible to FindConnection and List.
	Delete(ctx context.Context, tenant TenantContext, configName string) error
	// List returns the connection descriptors owned by the tenant in a
	// deterministic order. Deleted connections MUST NOT appear.
	List(ctx context.Context, tenant TenantContext) ([]ConnectionDescriptor, error)
}

// ExecutionStore is an OPTIONAL port for persisting and reading execution
// lifecycle state. When absent, the Engine runs without durable execution
// tracking and the host owns state externally.
type ExecutionStore interface {
	// SaveExecution upserts the execution state for the tenant.
	SaveExecution(ctx context.Context, tenant TenantContext, state ExecutionState) error
	// FindExecution returns the execution state for the given job and whether it
	// exists for the tenant.
	FindExecution(ctx context.Context, tenant TenantContext, jobID string) (ExecutionState, bool, error)
}

// ResultSink is an OPTIONAL port for persisting extraction result payloads to
// host-managed storage. When absent, the Engine returns results inline and does
// not persist them.
type ResultSink interface {
	// PersistResult stores the serialized result for the tenant and returns a
	// secret-free reference (path, HMAC, size).
	PersistResult(ctx context.Context, tenant TenantContext, payload []byte) (ResultReference, error)
}

// SchemaCache is an OPTIONAL port for caching datasource schema snapshots.
// When absent, schema is always fetched fresh from the datasource.
type SchemaCache interface {
	// GetSchema returns the cached snapshot for the named datasource and whether
	// it was present for the tenant.
	GetSchema(ctx context.Context, tenant TenantContext, configName string) (SchemaSnapshot, bool, error)
	// PutSchema stores the snapshot for the tenant.
	PutSchema(ctx context.Context, tenant TenantContext, snapshot SchemaSnapshot) error
}

// EventSink is an OPTIONAL port for emitting past-tense execution lifecycle
// events to the host (e.g. job completed/failed). When absent, the Engine emits
// nothing.
type EventSink interface {
	// Emit delivers a lifecycle event derived from execution state for the
	// tenant.
	Emit(ctx context.Context, tenant TenantContext, state ExecutionState) error
}

// TenantResolver is an OPTIONAL port that lets the host enrich or validate the
// tenant context before the Engine resolves connections, reads schema, or
// extracts data. When absent, the Engine uses the caller-supplied tenant
// context as-is.
type TenantResolver interface {
	// Resolve returns the effective tenant context for the request.
	Resolve(ctx context.Context, tenant TenantContext) (TenantContext, error)
}

// Observability is an OPTIONAL port providing host tracing hooks. The contract
// is deliberately minimal so the Engine core never imports a tracing library;
// the host adapts its own tracer (e.g. lib-observability) behind this seam.
type Observability interface {
	// StartSpan starts a span for the named operation and returns the derived
	// context plus an end function the Engine defers.
	StartSpan(ctx context.Context, operation string) (context.Context, func())
}
