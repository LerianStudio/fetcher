// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "context"

// This file implements the Engine TestConnection operation: an explicit,
// host-controlled connectivity check that keeps connector CONSTRUCTION separate
// from CONNECTIVITY (connector.go's load-bearing invariant). It runs entirely
// through PORTS — ConnectionStore and ConnectorRegistry — so pkg/engine stays
// infrastructure-free: it imports no driver, no enginecompat adapter, and no
// concrete datasource. The host wraps its drivers behind the Connector seam.
//
// Tenant scope is the SOLE isolation boundary (TenantContext.TenantID); the
// Engine carries no organization or product concept. Tenant scope is validated
// BEFORE any resource access, mirroring the other connection operations.
//
// The credential is NOT decrypted inside Engine core. The descriptor that
// reaches the connector factory is the secret-free ConnectionDescriptor; the
// host's connector resolves the credential at connect time (via its injected
// resolver, which may call CredentialProtector.Reveal). Keeping ciphertext out
// of Engine core is why ConnectionStore returns no secret material and why this
// operation never touches the protector directly.

// ConnectionTestResult is the secret-free outcome of a connectivity check. It
// carries only the success flag and the config name that was tested; it never
// carries credentials, DSNs, latencies that could fingerprint internals, or any
// driver text. A failed test returns a zero-value (Success=false) result
// alongside a safe *EngineError.
type ConnectionTestResult struct {
	// Success reports whether the connector's explicit connectivity check passed.
	Success bool `json:"success"`
	// ConfigName echoes the connection that was tested within the tenant scope.
	ConfigName string `json:"configName"`
}

// TestConnection performs an explicit connectivity check for a stored connection
// within the tenant scope. It is the Engine-level operation that mirrors the
// Manager's connection test minus rate limiting (rate limiting stays a Manager
// transport concern and is deliberately NOT added to the Engine).
//
// Order of operations (each gate runs before the next acquires more):
//  1. validate tenant scope BEFORE any resource access;
//  2. resolve the scoped connection via ConnectionStore (unknown / wrong-tenant
//     connections fail here, BEFORE any connector construction);
//  3. resolve the ConnectorFactory by datasource type (unknown type → the stable
//     UnknownConnectorTypeError);
//  4. build the connector (I/O-free) and ALWAYS close it via defer;
//  5. run the connector's explicit TestConnection connectivity step.
//
// Connector construction and connectivity errors are mapped to safe
// CategoryUnavailable EngineErrors: the underlying error MAY embed a DSN,
// credential, or driver internals, so it is DELIBERATELY discarded from the
// returned message, mirroring protectSecret and guardActiveExecutions.
func (e *Engine) TestConnection(
	ctx context.Context,
	tenant TenantContext,
	configName string,
) (ConnectionTestResult, error) {
	ctx, end := e.startSpan(ctx, "engine.connection.test")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ConnectionTestResult{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return ConnectionTestResult{}, err
	}

	// Resolve the connection within the tenant scope. A missing connection — or a
	// connection owned by a different tenant, which is invisible under this scope —
	// fails as not-found BEFORE any connector is constructed.
	descriptor, found, err := store.FindConnection(ctx, tenant, configName)
	if err != nil {
		return ConnectionTestResult{}, err
	}

	if !found {
		return ConnectionTestResult{}, NewEngineError(CategoryNotFound, "connection not found")
	}

	// Resolve the connector factory by datasource type. An unregistered type yields
	// the stable unknown-type error, identical everywhere a registry resolves by
	// type, and still BEFORE any connector construction.
	factory, err := e.requireConnectorFactory(descriptor.Type)
	if err != nil {
		return ConnectionTestResult{}, err
	}

	// Build is I/O-free per the connector contract; connectivity happens only in
	// the connector's TestConnection below. A build failure is a safe, redacted
	// error: the factory's raw error may embed driver internals.
	connector, err := factory.Build(ctx, descriptor)
	if err != nil {
		return ConnectionTestResult{}, NewEngineError(CategoryUnavailable, "failed to build connector for connection")
	}

	// ALWAYS close the connector, on BOTH the success and failure paths. Close is
	// contractually safe after a failed TestConnection and idempotent enough that a
	// double close does not panic. A close failure must not mask the primary
	// outcome, so its error is intentionally not surfaced here.
	defer func() { _ = connector.Close(ctx) }()

	if err := connector.TestConnection(ctx); err != nil {
		return ConnectionTestResult{}, NewEngineError(CategoryUnavailable, "failed to connect to datasource")
	}

	return ConnectionTestResult{Success: true, ConfigName: descriptor.ConfigName}, nil
}

// requireConnectorFactory resolves the ConnectorFactory registered for the
// datasource type through the REQUIRED ConnectorRegistry port. An unregistered
// type yields the stable UnknownConnectorTypeError (CategoryNotFound). New
// guarantees the registry is present, so this never dereferences a nil registry.
func (e *Engine) requireConnectorFactory(datasourceType string) (ConnectorFactory, error) {
	factory, ok := e.options.connectorRegistry.Connector(datasourceType)
	if !ok {
		return nil, UnknownConnectorTypeError(datasourceType)
	}

	return factory, nil
}
