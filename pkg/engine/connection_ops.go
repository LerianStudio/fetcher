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

	// Protect the secret BEFORE any store write. On failure we return a safe
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

// validateTenantScope rejects a tenant that carries no tenantId. Config
// uniqueness and ownership are anchored to the tenant, so an unscoped tenant
// cannot own a connection. This is the operation-level guard that also catches a
// zero-value TenantContext that bypassed the NewTenantContext constructor.
func validateTenantScope(tenant TenantContext) error {
	if tenant.TenantID == "" {
		return NewEngineError(CategoryValidation, "tenant scope is required")
	}

	return nil
}
