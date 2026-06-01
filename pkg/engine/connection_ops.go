// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "context"

// This file implements the Engine connection lifecycle operations. They run
// THROUGH the ConnectionStore port under a TenantContext scope and never import
// MongoDB, the Manager, or any host repository — that is what keeps pkg/engine
// importable without infrastructure (T-001 boundary).
//
// Semantics mirror the Manager's connection services:
//   - Config-name uniqueness is enforced WITHIN the product scope only; the same
//     config name under a different product is an isolated, valid record.
//   - Create returns the secret-free ConnectionDescriptor; the raw secret never
//     leaves the input.
//   - Update applies Manager-compatible partial-patch semantics.
//   - Delete makes a connection unavailable for get/list/extraction selection.
//
// Credential PROTECTION (encrypt-before-store) is deferred to ST-T003-02; this
// file preserves the store/port shape and the redaction-on-output guarantee so
// the protector can be wired in without reshaping the contract.

// CreateConnection persists a new connection for the tenant and returns the
// secret-free descriptor. It enforces config-name uniqueness within the product
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

	if err := validateProductScope(tenant); err != nil {
		return ConnectionDescriptor{}, err
	}

	descriptor := DescriptorFromInput(input)
	descriptor.ProductName = tenant.ProductName

	if err := store.Create(ctx, tenant, descriptor); err != nil {
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

	if err := validateProductScope(tenant); err != nil {
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

// ListConnections returns the non-deleted connections owned by the tenant's
// product scope, in the store's deterministic order.
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

	if err := validateProductScope(tenant); err != nil {
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

	if err := validateProductScope(tenant); err != nil {
		return ConnectionDescriptor{}, err
	}

	current, found, err := store.FindConnection(ctx, tenant, configName)
	if err != nil {
		return ConnectionDescriptor{}, err
	}

	if !found {
		return ConnectionDescriptor{}, NewEngineError(CategoryNotFound, "connection not found")
	}

	updated := applyConnectionPatch(current, patch)

	if err := store.Update(ctx, tenant, updated); err != nil {
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

	if err := validateProductScope(tenant); err != nil {
		return err
	}

	return store.Delete(ctx, tenant, configName)
}

// applyConnectionPatch returns a copy of current with only the non-nil patch
// fields overwritten. The config name and product scope are immutable through a
// patch; identity stays anchored to the original descriptor.
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

// startSpan starts an optional host span for the operation and returns the
// derived context plus an end function the caller defers. When no Observability
// port is configured it is a no-op, so the Engine core never imports a tracer.
func (e *Engine) startSpan(ctx context.Context, operation string) (context.Context, func()) {
	if isNilPort(e.options.observability) {
		return ctx, func() {}
	}

	return e.options.observability.StartSpan(ctx, operation)
}

// validateProductScope rejects a tenant that carries no product scope. Config
// uniqueness and ownership are anchored to the product, so an unscoped tenant
// cannot own a connection.
func validateProductScope(tenant TenantContext) error {
	if tenant.ProductName == "" {
		return NewEngineError(CategoryValidation, "tenant product scope is required")
	}

	return nil
}
