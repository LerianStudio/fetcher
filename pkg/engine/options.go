// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

// Options is the resolved configuration an Engine is constructed from. Hosts do
// not build it directly; they pass functional Option values to New, which
// applies them in order and then fills safe defaults. Fields are unexported so
// the only construction path is New + Option, keeping invariants enforced in
// one place (the constructor) rather than scattered across callers.
type Options struct {
	connectorRegistry      ConnectorRegistry
	credentialProtector    CredentialProtector
	connectionStore        ConnectionStore
	executionStore         ExecutionStore
	resultSink             ResultSink
	schemaCache            SchemaCache
	eventSink              EventSink
	tenantResolver         TenantResolver
	activeExecutionChecker ActiveExecutionChecker
	observability          Observability

	encryptedPersistence bool
	limits               Limits
}

// Option mutates Options during New. Options are applied in the order supplied;
// later options override earlier ones for the same field.
type Option func(*Options)

// WithConnectorRegistry sets the REQUIRED connector registry the Engine uses to
// resolve datasource connectors by type.
func WithConnectorRegistry(registry ConnectorRegistry) Option {
	return func(o *Options) {
		o.connectorRegistry = registry
	}
}

// WithCredentialProtector sets the credential protector. It is required only
// when encrypted persistence is enabled (see WithEncryptedPersistence).
func WithCredentialProtector(protector CredentialProtector) Option {
	return func(o *Options) {
		o.credentialProtector = protector
	}
}

// WithEncryptedPersistence toggles encrypted persistence of connection secrets.
// When true, New requires a non-nil CredentialProtector.
func WithEncryptedPersistence(enabled bool) Option {
	return func(o *Options) {
		o.encryptedPersistence = enabled
	}
}

// WithConnectionStore sets the optional connection store.
func WithConnectionStore(store ConnectionStore) Option {
	return func(o *Options) {
		o.connectionStore = store
	}
}

// WithExecutionStore sets the optional execution store.
func WithExecutionStore(store ExecutionStore) Option {
	return func(o *Options) {
		o.executionStore = store
	}
}

// WithResultSink sets the optional result sink.
func WithResultSink(sink ResultSink) Option {
	return func(o *Options) {
		o.resultSink = sink
	}
}

// WithSchemaCache sets the optional schema cache.
func WithSchemaCache(cache SchemaCache) Option {
	return func(o *Options) {
		o.schemaCache = cache
	}
}

// WithEventSink sets the optional event sink.
func WithEventSink(sink EventSink) Option {
	return func(o *Options) {
		o.eventSink = sink
	}
}

// WithTenantResolver sets the optional tenant resolver.
func WithTenantResolver(resolver TenantResolver) Option {
	return func(o *Options) {
		o.tenantResolver = resolver
	}
}

// WithActiveExecutionChecker sets the optional logical conflict checker the
// Engine consults before mutating a connection (update/delete). When omitted,
// the Engine performs no conflict gating and mutations proceed.
func WithActiveExecutionChecker(checker ActiveExecutionChecker) Option {
	return func(o *Options) {
		o.activeExecutionChecker = checker
	}
}

// WithObservability sets the optional observability hooks.
func WithObservability(obs Observability) Option {
	return func(o *Options) {
		o.observability = obs
	}
}

// WithLimits sets custom resource limits. A zero-value Limits is treated as
// "unset" by New, which substitutes DefaultLimits to guarantee the Engine never
// operates with unbounded resources.
func WithLimits(limits Limits) Option {
	return func(o *Options) {
		o.limits = limits
	}
}
