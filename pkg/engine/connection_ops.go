// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "context"

// This file implements the Engine connection lifecycle operations. They run
// THROUGH the ConnectionStore port under a TenantContext scope and never import
// MongoDB, the Manager, or any host repository — that is what keeps pkg/engine
// importable without infrastructure (T-001 boundary).
//
// Semantics mirror develop's authoritative model:
//   - Config-name uniqueness is enforced WITHIN the tenant scope only; the same
//     config name under a different tenant is an isolated, valid record
//     (config_name is unique within the tenant — see pkg/mongodb/connection).
//   - Create returns the secret-free ConnectionDescriptor; the raw secret never
//     leaves the input.
//   - Update applies Manager-compatible partial-patch semantics.
//   - Delete makes a connection unavailable for get/list/extraction selection.
//
// Credential PROTECTION (encrypt-before-store) is deferred to ST-T003-02; this
// file preserves the store/port shape and the redaction-on-output guarantee so
// the protector can be wired in without reshaping the contract.

// CreateConnection persists a new connection for the tenant and returns the
// secret-free descriptor. It enforces config-name uniqueness within the tenant
// scope and rejects an unscoped tenant.
func (e *Engine) CreateConnection(
	ctx context.Context,
	tenant TenantContext,
	input ConnectionInput,
) (ConnectionDescriptor, error) {
	ctx, end := e.startSpan(ctx, "engine.connection.create")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ConnectionDescriptor{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return ConnectionDescriptor{}, err
	}

	descriptor := DescriptorFromInput(input)

	// Reject a duplicate (tenantID, configName) BEFORE protecting the secret. The
	// pre-check both fixes the ordering — a rejected create must never invoke the
	// protector (wasted KMS/crypto call, an avoidable cost vector) — and preserves
	// the redaction-on-output contract. It is an optimization plus
	// correctness-of-ordering, NOT the source of truth: store.Create below remains
	// the atomic uniqueness backstop so two concurrent creates still have exactly
	// one winner.
	if _, found, err := store.FindConnection(ctx, tenant, descriptor.ConfigName); err != nil {
		return ConnectionDescriptor{}, err
	} else if found {
		// A duplicate (tenantID, configName) is a CONFLICT with the current state
		// of the resource, not malformed input — CategoryConflict maps to HTTP 409,
		// which is the duplicate-create status hosts (e.g. the Manager) preserve.
		return ConnectionDescriptor{}, NewEngineError(CategoryConflict, "connection already exists")
	}

	// Protect the secret BEFORE the store write. On failure we return a safe
	// error and leave the store untouched (atomicity): the Create call below is
	// never reached. Protection runs only when encrypted persistence is enabled
	// and a password is actually supplied.
	var credential *ProtectedCredential

	if e.options.encryptedPersistence && input.HasPassword() {
		protected, err := e.protectSecret(ctx, tenant, input.Password())
		if err != nil {
			return ConnectionDescriptor{}, err
		}

		credential = protected
		descriptor.KeyVersion = protected.KeyVersion
	}

	// store.Create still enforces uniqueness atomically: the pre-check above is an
	// optimization, this is the race backstop and source of truth.
	if err := store.Create(ctx, tenant, descriptor, credential); err != nil {
		return ConnectionDescriptor{}, err
	}

	return descriptor, nil
}

// GetConnection returns the descriptor for the named connection within the
// tenant scope. A missing or deleted connection yields a CategoryNotFound error.
func (e *Engine) GetConnection(
	ctx context.Context,
	tenant TenantContext,
	configName string,
) (ConnectionDescriptor, error) {
	ctx, end := e.startSpan(ctx, "engine.connection.get")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ConnectionDescriptor{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return ConnectionDescriptor{}, err
	}

	descriptor, found, err := store.FindConnection(ctx, tenant, configName)
	if err != nil {
		return ConnectionDescriptor{}, err
	}

	if !found {
		return ConnectionDescriptor{}, NewEngineError(CategoryNotFound, "connection not found")
	}

	return descriptor, nil
}

// ListConnections returns the non-deleted connections owned by the tenant, in
// the store's deterministic order.
func (e *Engine) ListConnections(
	ctx context.Context,
	tenant TenantContext,
) ([]ConnectionDescriptor, error) {
	ctx, end := e.startSpan(ctx, "engine.connection.list")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return nil, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return nil, err
	}

	return store.List(ctx, tenant)
}

// GetConnectionByID returns the descriptor addressed by the OPAQUE, host-owned
// id within the tenant scope. It is the ID-addressed read the host uses when it
// owns a UUID identity model (the Manager): the Engine validates the per-request
// tenant scope, then resolves through ConnectionStore.FindByID. A missing or
// soft-deleted connection yields a CategoryNotFound error — the same byte-safe
// error an unknown id and a cross-tenant id both produce, so the existence
// oracle never distinguishes "not yours" from "does not exist".
func (e *Engine) GetConnectionByID(
	ctx context.Context,
	tenant TenantContext,
	id string,
) (ConnectionDescriptor, error) {
	ctx, end := e.startSpan(ctx, "engine.connection.get_by_id")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ConnectionDescriptor{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return ConnectionDescriptor{}, err
	}

	descriptor, found, err := store.FindByID(ctx, tenant, id)
	if err != nil {
		return ConnectionDescriptor{}, err
	}

	if !found {
		return ConnectionDescriptor{}, NewEngineError(CategoryNotFound, "connection not found")
	}

	return descriptor, nil
}

// ListConnectionsPaged returns the tenant's connection page, carrying the host's
// OPAQUE list params through to ConnectionStore.ListPaged. The Engine enforces
// ONLY tenant scope; it neither reads nor interprets the params and returns the
// store's page verbatim. This is the read the Manager routes its paginated,
// filtered, resolver-merged list through: the Engine is the single tenant-scope
// authority, the host owns the pagination mechanics.
func (e *Engine) ListConnectionsPaged(
	ctx context.Context,
	tenant TenantContext,
	params ConnectionListParams,
) (ConnectionPage, error) {
	ctx, end := e.startSpan(ctx, "engine.connection.list_paged")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ConnectionPage{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return ConnectionPage{}, err
	}

	return store.ListPaged(ctx, tenant, params)
}

// UpdateConnectionByID persists the host's already-patched record addressed by
// the OPAQUE id within the tenant scope and returns the updated secret-free
// descriptor. It is the ID-addressed write the Manager routes through after
// applying its own domain patch (cryptor re-encryption stays host-side): the
// Engine validates the tenant scope, re-protects the secret only when the patch
// carries one (encrypted persistence enabled), and delegates to
// ConnectionStore.UpdateByID. The conflict gate is NOT re-run here — the caller
// runs the shared CheckActiveExecutions gate against the resolved config name
// before applying its patch, so the gate fires exactly once and keeps its
// host-mapped error contract.
//
// descriptor carries the patched rich record in HostAttributes and the same
// opaque id. A nil patch.Password leaves the stored secret intact.
func (e *Engine) UpdateConnectionByID(
	ctx context.Context,
	tenant TenantContext,
	id string,
	descriptor ConnectionDescriptor,
	patch ConnectionPatch,
) (ConnectionDescriptor, error) {
	ctx, end := e.startSpan(ctx, "engine.connection.update_by_id")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ConnectionDescriptor{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return ConnectionDescriptor{}, err
	}

	// Re-protect the secret ONLY when the patch carries a password change and
	// encrypted persistence is enabled. A nil credential means "no password
	// change": ConnectionStore.UpdateByID leaves the existing stored secret
	// intact. Protection runs BEFORE the store write so a failure leaves the store
	// untouched (atomicity), mirroring UpdateConnection.
	var credential *ProtectedCredential

	if e.options.encryptedPersistence && patch.HasPassword() {
		protected, err := e.protectSecret(ctx, tenant, *patch.password)
		if err != nil {
			return ConnectionDescriptor{}, err
		}

		credential = protected
		descriptor.KeyVersion = protected.KeyVersion
	}

	if err := store.UpdateByID(ctx, tenant, id, descriptor, credential); err != nil {
		return ConnectionDescriptor{}, err
	}

	return descriptor, nil
}

// DeleteConnectionByID removes the connection addressed by the OPAQUE id within
// the tenant scope, making it unavailable for get/list/extraction selection. A
// store that models soft-delete soft-deletes. A missing connection yields a
// CategoryNotFound error. As with UpdateConnectionByID the conflict gate is the
// caller's responsibility (run once via CheckActiveExecutions before delete), so
// this op performs scope validation + the ID-addressed delete only.
func (e *Engine) DeleteConnectionByID(
	ctx context.Context,
	tenant TenantContext,
	id string,
) error {
	ctx, end := e.startSpan(ctx, "engine.connection.delete_by_id")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return err
	}

	if err := validateTenantScope(tenant); err != nil {
		return err
	}

	return store.DeleteByID(ctx, tenant, id)
}

// ConnectionPatch carries the partially-updatable fields of a connection. A nil
// pointer leaves the corresponding field unchanged, mirroring the Manager's
// patch semantics. The secret is supplied through the Password pointer so it is
// never stored in an exported, long-lived field; it is applied only when set.
type ConnectionPatch struct {
	Type         *string
	Host         *string
	Port         *int
	DatabaseName *string
	Schema       *string
	Username     *string
	SSLMode      *string

	// HostAttributes optionally re-supplies the OPAQUE host payload. A nil map
	// leaves the existing stored payload intact (mirroring the nil-pointer
	// patch semantics of the typed fields); a non-nil map REPLACES it wholesale.
	// The Engine never reads a key from it — it only forwards it onto the
	// updated descriptor so the host can re-stamp its rich record (e.g. a fresh
	// UpdatedAt) without the Engine interpreting host fields. See
	// ConnectionDescriptor.HostAttributes.
	HostAttributes map[string]any

	// password holds an optional new secret. It is unexported so default struct
	// formatting and JSON marshaling skip it. Build a patch with NewSecretPatch
	// when a credential change is intended.
	password    *string
	hasPassword bool
}

// WithPassword returns a copy of the patch carrying a new secret. ST-T003-02
// consumes the secret to re-protect the stored credential; this subtask only
// preserves the carrying shape and never persists it as plaintext metadata.
func (p ConnectionPatch) WithPassword(password string) ConnectionPatch {
	p.password = &password
	p.hasPassword = true

	return p
}

// HasPassword reports whether the patch carries a new secret without revealing
// it.
func (p ConnectionPatch) HasPassword() bool {
	return p.hasPassword && p.password != nil
}

// UpdateConnection applies a partial patch to an existing connection within the
// tenant scope and returns the updated secret-free descriptor. A missing
// connection yields a CategoryNotFound error.
func (e *Engine) UpdateConnection(
	ctx context.Context,
	tenant TenantContext,
	configName string,
	patch ConnectionPatch,
) (ConnectionDescriptor, error) {
	ctx, end := e.startSpan(ctx, "engine.connection.update")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ConnectionDescriptor{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return ConnectionDescriptor{}, err
	}

	current, found, err := store.FindConnection(ctx, tenant, configName)
	if err != nil {
		return ConnectionDescriptor{}, err
	}

	if !found {
		return ConnectionDescriptor{}, NewEngineError(CategoryNotFound, "connection not found")
	}

	// Block the mutation while active work references the connection, mirroring
	// the Manager's active-job guard. The gate runs BEFORE any protect/store write
	// so a conflict (or a checker failure) leaves the store untouched. It is a
	// no-op when no checker is configured.
	if err := e.guardActiveExecutions(ctx, tenant, configName); err != nil {
		return ConnectionDescriptor{}, err
	}

	updated := applyConnectionPatch(current, patch)

	// Re-protect the secret ONLY when a password change is supplied. A patch with
	// no password leaves the existing stored secret and its key version intact:
	// we pass a nil credential so the store does not wipe it, and preserve the
	// current descriptor's KeyVersion. Protection runs BEFORE the store write so a
	// failure leaves the store untouched (atomicity), and only when encrypted
	// persistence is enabled.
	var credential *ProtectedCredential

	if e.options.encryptedPersistence && patch.HasPassword() {
		protected, err := e.protectSecret(ctx, tenant, *patch.password)
		if err != nil {
			return ConnectionDescriptor{}, err
		}

		credential = protected
		updated.KeyVersion = protected.KeyVersion
	}

	if err := store.Update(ctx, tenant, updated, credential); err != nil {
		return ConnectionDescriptor{}, err
	}

	return updated, nil
}

// DeleteConnection removes a connection within the tenant scope, making it
// unavailable for get/list/extraction selection. A missing connection yields a
// CategoryNotFound error.
func (e *Engine) DeleteConnection(
	ctx context.Context,
	tenant TenantContext,
	configName string,
) error {
	ctx, end := e.startSpan(ctx, "engine.connection.delete")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return err
	}

	if err := validateTenantScope(tenant); err != nil {
		return err
	}

	// Symmetric with UpdateConnection: confirm the connection exists BEFORE
	// consulting the active-execution checker. A missing connection is a
	// not-found error and the guard must not run on a connection that does not
	// exist (B4 finding #1).
	if _, found, err := store.FindConnection(ctx, tenant, configName); err != nil {
		return err
	} else if !found {
		return NewEngineError(CategoryNotFound, "connection not found")
	}

	// Block deletion while active work references the connection, mirroring the
	// Manager's active-job guard. The gate runs BEFORE the store delete so a
	// conflict (or a checker failure) leaves the connection intact. It is a no-op
	// when no checker is configured.
	if err := e.guardActiveExecutions(ctx, tenant, configName); err != nil {
		return err
	}

	return store.Delete(ctx, tenant, configName)
}

// CheckActiveExecutions runs the shared active-execution conflict gate for a
// connection within the tenant scope, independent of the Engine's own
// connection persistence. It is the gate that UpdateConnection and
// DeleteConnection apply internally, exposed as a standalone capability so a
// host that owns connection persistence itself (e.g. the Manager, which keeps
// its UUID-keyed MongoDB repository) can still delegate the SHARED policy
// decision — "block a mutation while active work references the connection" —
// to the Engine under a per-request tenant scope.
//
// It returns:
//   - nil when no checker is configured, or the checker reports no active work;
//   - a CategoryConflict *EngineError when active work references the connection;
//   - the checker's own error (WRAPPED, not replaced) when the checker fails, so
//     the host that owns the checker can preserve its existing error-mapping
//     contract (e.g. errors.Is on the underlying repository error). This differs
//     from guardActiveExecutions, which is used by the Engine's own persistence
//     ops and DISCARDS the raw error: there the error would cross to external
//     embedding callers, whereas CheckActiveExecutions returns in-process to the
//     same host that supplied the checker, so no boundary is crossed;
//   - a CategoryValidation *EngineError when the tenant scope is missing.
//
// connectionID is the connection's config name within the tenant scope.
func (e *Engine) CheckActiveExecutions(ctx context.Context, tenant TenantContext, connectionID string) error {
	ctx, end := e.startSpan(ctx, "engine.connection.check_active_executions")
	defer end()

	if err := validateTenantScope(tenant); err != nil {
		return err
	}

	if isNilPort(e.options.activeExecutionChecker) {
		return nil
	}

	active, err := e.options.activeExecutionChecker.HasActiveExecutions(ctx, tenant, connectionID)
	if err != nil {
		return err
	}

	if active {
		return NewEngineError(CategoryConflict, "connection has active executions")
	}

	return nil
}

// AuthorizeConnectionAccess applies the Engine's connection-read RULE: a
// tenant-owned connection read is permitted only under a present, well-formed
// tenant scope. It is the authority entrypoint a host uses for read paths
// (get/list) that keep their OWN identity model and persistence — the Manager,
// for example, reads by uuid.UUID and paginates on its repository, neither of
// which the Engine's config-name-keyed, flat ConnectionStore ops model. Routing
// the SCOPE decision through the Engine makes it the single authority for
// "which tenant may read a connection" without dragging the host's UUID or
// pagination identity into the Engine, and without a redundant persistence
// round-trip.
//
// It returns a CategoryValidation *EngineError when the tenant scope is missing
// or malformed, and nil otherwise. It performs no I/O and consults no store.
func (e *Engine) AuthorizeConnectionAccess(ctx context.Context, tenant TenantContext) error {
	_, end := e.startSpan(ctx, "engine.connection.authorize_access")
	defer end()

	return validateTenantScope(tenant)
}

// guardActiveExecutions consults the optional ActiveExecutionChecker before a
// connection mutation. It returns:
//   - nil when no checker is configured (conflict gating is opt-in);
//   - a CategoryConflict EngineError when the checker reports active work;
//   - a CategoryUnavailable EngineError when the checker itself fails — the raw
//     checker error is DELIBERATELY discarded from the returned message so a host
//     implementation can never leak internals (DSNs, job IDs) across the Engine
//     boundary, consistent with protectSecret's error handling.
//
// connectionID is the connection's config name within the tenant scope; the
// checker is always invoked under that scope so one tenant's active work never
// blocks another's mutation.
func (e *Engine) guardActiveExecutions(ctx context.Context, tenant TenantContext, connectionID string) error {
	if isNilPort(e.options.activeExecutionChecker) {
		return nil
	}

	active, err := e.options.activeExecutionChecker.HasActiveExecutions(ctx, tenant, connectionID)
	if err != nil {
		return NewEngineError(CategoryUnavailable, "failed to check for active executions")
	}

	if active {
		return NewEngineError(CategoryConflict, "connection has active executions")
	}

	return nil
}

// applyConnectionPatch returns a copy of current with only the non-nil patch
// fields overwritten. The config name is immutable through a patch; identity
// stays anchored to the original descriptor under its tenant scope.
func applyConnectionPatch(current ConnectionDescriptor, patch ConnectionPatch) ConnectionDescriptor {
	if patch.Type != nil {
		current.Type = *patch.Type
	}

	if patch.Host != nil {
		current.Host = *patch.Host
	}

	if patch.Port != nil {
		current.Port = *patch.Port
	}

	if patch.DatabaseName != nil {
		current.DatabaseName = *patch.DatabaseName
	}

	if patch.Schema != nil {
		current.Schema = *patch.Schema
	}

	if patch.Username != nil {
		current.Username = *patch.Username
	}

	if patch.SSLMode != nil {
		current.SSLMode = *patch.SSLMode
	}

	// Replace the opaque host payload only when the patch supplies a fresh one;
	// a nil map preserves the existing payload (the current descriptor already
	// carries it). The Engine forwards the payload verbatim and reads no key.
	if patch.HostAttributes != nil {
		current.HostAttributes = patch.HostAttributes
	}

	return current
}

// requireConnectionStore returns the configured connection store or a stable
// error when the Engine was constructed without one. It guards every connection
// operation so a missing optional port surfaces as a clear EngineError instead
// of a nil-pointer panic.
func (e *Engine) requireConnectionStore() (ConnectionStore, error) {
	if isNilPort(e.options.connectionStore) {
		return nil, NewEngineError(CategoryValidation, "connection store is not configured")
	}

	return e.options.connectionStore, nil
}

// protectSecret encrypts a plaintext credential through the host-provided
// CredentialProtector and returns the protected sidecar to persist. It is the
// single choke point through which every secret passes before storage.
//
// It returns a safe *EngineError on failure. The protector's raw error is
// DELIBERATELY discarded from the returned message: a host implementation could
// embed the plaintext or ciphertext in its error, and the Engine boundary must
// never surface secret material. The category is CategoryUnavailable because a
// protection failure is an infrastructure-dependency failure, not bad input.
func (e *Engine) protectSecret(ctx context.Context, tenant TenantContext, plaintext string) (*ProtectedCredential, error) {
	protector, err := e.requireCredentialProtector()
	if err != nil {
		return nil, err
	}

	ciphertext, keyVersion, protectErr := protector.Protect(ctx, tenant, []byte(plaintext))
	if protectErr != nil {
		return nil, NewEngineError(CategoryUnavailable, "failed to protect connection credential")
	}

	return &ProtectedCredential{Ciphertext: ciphertext, KeyVersion: keyVersion}, nil
}

// requireCredentialProtector returns the configured protector or a stable error.
// New already rejects a missing protector when encrypted persistence is enabled
// (T-002), so this is a defensive guard ensuring no credential write ever
// proceeds to plaintext persistence if that invariant were somehow bypassed.
func (e *Engine) requireCredentialProtector() (CredentialProtector, error) {
	if isNilPort(e.options.credentialProtector) {
		return nil, NewEngineError(CategoryValidation, "credential protector is not configured")
	}

	return e.options.credentialProtector, nil
}

// startSpan starts an optional host span for the operation and returns the
// derived context plus an end function the caller defers. When no Observability
// port is configured it is a no-op, so the Engine core never imports a tracer.
func (e *Engine) startSpan(ctx context.Context, operation string) (context.Context, func()) {
	if isNilPort(e.options.observability) {
		return ctx, func() {}
	}

	return e.options.observability.StartSpan(ctx, operation)
}

// validateTenantScope rejects a tenant whose TenantID is empty OR malformed.
// Config uniqueness and ownership are anchored to the tenant, so an unscoped or
// ill-shaped tenant cannot own a connection. Because TenantContext.TenantID is an
// exported field, a host can build a TenantContext directly and bypass
// NewTenantContext's validation; this operation-level guard applies the SAME
// isValidTenantID shape check on every operation so the tenant-id contract holds
// regardless of how the context was constructed. isValidTenantID already rejects
// the empty string, so a zero-value TenantContext is caught here too.
func validateTenantScope(tenant TenantContext) error {
	if !isValidTenantID(tenant.TenantID) {
		return NewEngineError(CategoryValidation, "tenant scope is required")
	}

	return nil
}
