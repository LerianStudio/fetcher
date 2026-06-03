// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"sort"
	"strings"
)

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

	return e.resolveSchema(ctx, tenant, descriptor, false)
}

// DiscoverSchemaFresh is the ALWAYS-FRESH variant of DiscoverSchema: it resolves
// the scoped connection and discovers the schema LIVE through the connector on
// every call, NEVER consulting and NEVER populating the optional SchemaCache.
//
// It exists to preserve the pre-embedded-Engine GET-schema contract, where the
// Manager's GET .../schema endpoint neither read nor wrote the schema cache and
// therefore always reflected the live datasource. Cache-first DiscoverSchema (and
// the cache-first ValidateSchema path) are unchanged; only this method bypasses
// the cache entirely, so the two freshness contracts coexist without a global
// cache toggle. Tenant scope is still validated BEFORE any resource access — the
// bypass is a CACHE decision, never an authorization one.
func (e *Engine) DiscoverSchemaFresh(
	ctx context.Context,
	tenant TenantContext,
	configName string,
) (SchemaSnapshot, error) {
	ctx, end := e.startSpan(ctx, "engine.schema.discover_fresh")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return SchemaSnapshot{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return SchemaSnapshot{}, err
	}

	descriptor, found, err := store.FindConnection(ctx, tenant, configName)
	if err != nil {
		return SchemaSnapshot{}, err
	}

	if !found {
		return SchemaSnapshot{}, NewEngineError(CategoryNotFound, "connection not found")
	}

	return e.resolveSchema(ctx, tenant, descriptor, true)
}

// resolveSchema serves the schema snapshot for an already-resolved connection
// descriptor under the tenant scope, cache-first. It is the shared resolve →
// cache → discover → write-through core that both DiscoverSchema and
// ValidateSchema run through, so the cache-hit (no connector discovery) and
// cache-miss (live discovery) behavior is identical for both operations.
//
// A cache hit short-circuits live discovery; the lookup is tenant-scoped so a
// hit can only belong to this tenant, and a cache read error degrades to a
// fresh discovery (the cache is an optimization). On a miss it resolves the
// connector factory, builds the connector (I/O-free, always closed via defer),
// discovers live, normalizes the snapshot identity, and writes through. Build
// and discovery failures are mapped to safe CategoryUnavailable errors — the
// raw error may embed a DSN, credential, or driver internals, so it is
// DELIBERATELY discarded from the returned message.
//
// skipCache forces an ALWAYS-FRESH resolution: when true the cache is neither
// READ (no short-circuit) nor WRITTEN (no write-through), so the call reflects
// the live datasource and leaves the cache untouched. DiscoverSchemaFresh sets
// it to preserve the legacy GET-schema freshness; DiscoverSchema and the
// ValidateSchema path leave it false to stay cache-first.
func (e *Engine) resolveSchema(
	ctx context.Context,
	tenant TenantContext,
	descriptor ConnectionDescriptor,
	skipCache bool,
) (SchemaSnapshot, error) {
	if !skipCache {
		if snapshot, ok := e.cachedSchema(ctx, tenant, descriptor.ConfigName); ok {
			return snapshot, nil
		}
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

	// Write through to the cache when configured, UNLESS the caller requested an
	// always-fresh resolution (skipCache) — an always-fresh GET must leave the
	// cache untouched, neither reading nor poisoning it. A write failure is
	// otherwise tolerated: the discovery already succeeded, so failing to populate
	// an optimization cache must not fail the operation.
	if !skipCache {
		e.cacheSchema(ctx, tenant, snapshot)
	}

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

// ---------------------------------------------------------------------------
// ValidateSchema (ST-T005-02)
// ---------------------------------------------------------------------------

// ValidationFailureType is a stable, safe classification for a single schema
// validation failure carried in a ValidationReport. Values are stable string
// constants — hosts may map them to their own transport shapes — so changing
// any value is a breaking change. They are credential-free by construction.
type ValidationFailureType string

const (
	// ValidationDatasourceNotFound indicates a mapped datasource has no persisted
	// connection within the tenant scope.
	ValidationDatasourceNotFound ValidationFailureType = "datasource_not_found"
	// ValidationTableNotFound indicates a mapped table is absent from the
	// datasource's schema snapshot.
	ValidationTableNotFound ValidationFailureType = "table_not_found"
	// ValidationFieldNotFound indicates a mapped field is absent from its table in
	// the schema snapshot.
	ValidationFieldNotFound ValidationFailureType = "field_not_found"
	// ValidationInvalidFilter indicates a filter field reference does not resolve
	// to a known field of its table in the schema snapshot.
	ValidationInvalidFilter ValidationFailureType = "invalid_filter"
	// ValidationLimitExceeded indicates the mapping breached an effective Engine
	// limit (datasource, table, field, or filter count).
	ValidationLimitExceeded ValidationFailureType = "limit_exceeded"
)

// TableMapping is the per-table portion of a validation request: the selected
// fields to extract and the field references used by filters. Both are
// validated against the datasource's schema snapshot.
type TableMapping struct {
	// Name is the qualified table/collection name to validate.
	Name string
	// Fields are the selected field names to validate against the table.
	Fields []string
	// FilterFields are the field references used by filters, validated against
	// the same table. They are validated SEPARATELY from Fields so an invalid
	// filter reference is a distinct failure type.
	FilterFields []string
}

// DatasourceMapping maps one datasource (by config name) to the tables, fields,
// and filter references to validate against its schema.
type DatasourceMapping struct {
	// ConfigName identifies the persisted connection within the tenant scope.
	ConfigName string
	// Tables enumerates the per-table field and filter mappings to validate.
	Tables []TableMapping
}

// SchemaValidationRequest is the canonical Engine input for ValidateSchema. It
// is a pure, secret-free data contract: it carries only datasource/table/field
// identities and filter field references, never credentials or query data. The
// Manager (or any host) maps its transport request onto this shape; the Engine
// returns a canonical ValidationReport, never an HTTP response shape.
type SchemaValidationRequest struct {
	// Datasources enumerates the per-datasource mappings to validate.
	Datasources []DatasourceMapping
}

// ValidationFailure is a single, safe validation failure. Every field is
// credential-free and data-free by construction: Detail is a static,
// pre-redacted description, never a raw connector or driver error.
type ValidationFailure struct {
	// Type classifies the failure.
	Type ValidationFailureType `json:"type"`
	// ConfigName identifies the datasource the failure belongs to.
	ConfigName string `json:"configName"`
	// Table is the qualified table name, when the failure is table-scoped.
	Table string `json:"table,omitempty"`
	// Field is the field or filter-field name, when the failure is field-scoped.
	Field string `json:"field,omitempty"`
	// Detail is a safe, human-readable description. It MUST NOT contain
	// credentials, connection strings, or extracted data.
	Detail string `json:"detail,omitempty"`
}

// ValidationReport is the canonical outcome of ValidateSchema. Valid is true
// only when Failures is empty. The report is the structured outcome the host
// formats for its transport; it never carries credentials or extracted rows.
type ValidationReport struct {
	// Valid reports whether the mapping validated cleanly (no failures).
	Valid bool `json:"valid"`
	// Failures enumerates every distinct validation failure found.
	Failures []ValidationFailure `json:"failures,omitempty"`
}

// ValidateSchema validates a mapping of datasources, tables, fields, and filter
// field references against the schema snapshot for each persisted connection
// within the tenant scope. It returns a canonical ValidationReport describing
// the outcome. The Manager (or any host) formats the report for transport; the
// Engine never returns an HTTP shape.
//
// Order of operations (each gate runs before the next acquires more):
//  1. require a ConnectionStore and validate the tenant scope BEFORE any access;
//  2. reject a malformed request (no datasources) as a CategoryValidation error
//     — distinct from a well-formed mapping that simply fails to validate;
//  3. enforce effective Engine limits (datasource / table / field / filter
//     counts) BEFORE any connector access; a breach is a DISTINCT
//     ValidationLimitExceeded report failure, not a generic failure;
//  4. per datasource: resolve the scoped connection (a missing connection — or
//     one owned by another tenant — is a DISTINCT ValidationDatasourceNotFound
//     report failure, NOT an error, and reaches no connector);
//  5. resolve the schema cache-first via the shared resolveSchema flow (cache
//     hit → no connector discovery; cache miss → live discovery). A source-down
//     connector failure is a SEPARATE Engine error (CategoryUnavailable family),
//     NOT a report failure — it is not the caller's malformed request;
//  6. validate the mapped tables, fields, and filter references against the
//     snapshot, recording each missing table / field / invalid filter as its own
//     DISTINCT report failure.
//
// The connector-discovery timeout is bounded by the effective Limits.Timeout so
// a single validation never blocks unbounded on a slow datasource.
func (e *Engine) ValidateSchema(
	ctx context.Context,
	tenant TenantContext,
	request SchemaValidationRequest,
) (ValidationReport, error) {
	return e.validateSchemaWithLimits(ctx, tenant, request, e.options.limits)
}

// validateSchemaWithLimits is the shared validation core that enforces an
// EXPLICIT effective Limits, rather than always the Engine default. ValidateSchema
// calls it with the Engine default; the planner calls it with the per-request
// effective limits (default merged with allowed overrides) so count-limit
// enforcement honors overrides without the planner duplicating the resolve →
// cache → discover → validate flow. The count-limit checks themselves remain in
// limitFailures, the single source of count enforcement.
func (e *Engine) validateSchemaWithLimits(
	ctx context.Context,
	tenant TenantContext,
	request SchemaValidationRequest,
	limits Limits,
) (ValidationReport, error) {
	ctx, end := e.startSpan(ctx, "engine.schema.validate")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ValidationReport{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return ValidationReport{}, err
	}

	// A request with no datasources is malformed input, distinct from a
	// well-formed mapping that fails to validate. Reject it as a CategoryValidation
	// error rather than returning a vacuously-valid report.
	if len(request.Datasources) == 0 {
		return ValidationReport{}, NewEngineError(CategoryValidation, "schema validation request has no datasources")
	}

	// Reject empty/whitespace identities as a malformed request BEFORE any
	// snapshot membership. An empty config / table / field / filter name is not a
	// legitimate schema mismatch; categorizing it as a not-found report failure
	// would mask malformed input, so it is a CategoryValidation error — consistent
	// with the empty-datasources guard above.
	if err := validateRequestNames(request); err != nil {
		return ValidationReport{}, err
	}

	// Bound the connector-discovery work by the effective timeout so a single
	// validation never blocks unbounded on a slow datasource. A zero timeout
	// leaves the context unchanged.
	if limits.Timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, limits.Timeout)
		defer cancel()
	}

	// Enforce count limits BEFORE any connector access. A breach is a distinct
	// limit-exceeded failure, recorded per offending scope, and short-circuits the
	// expensive resolve/discover path.
	if failures := limitFailures(request, limits); len(failures) > 0 {
		return newValidationReport(failures), nil
	}

	var failures []ValidationFailure

	for _, datasource := range request.Datasources {
		dsFailures, dsErr := e.validateDatasource(ctx, tenant, store, datasource)
		if dsErr != nil {
			// A source-down (or any connector/build) failure is a SEPARATE Engine
			// error, not a malformed-request report failure. Surface it as-is: it is
			// already a safe, redacted EngineError.
			return ValidationReport{}, dsErr
		}

		failures = append(failures, dsFailures...)
	}

	return newValidationReport(failures), nil
}

// validateDatasource resolves one datasource's connection and schema within the
// tenant scope and validates its mapped tables, fields, and filter references.
// It returns report failures for a missing connection or any malformed mapping,
// and a SEPARATE Engine error only for a source-down / connector failure.
func (e *Engine) validateDatasource(
	ctx context.Context,
	tenant TenantContext,
	store ConnectionStore,
	datasource DatasourceMapping,
) ([]ValidationFailure, error) {
	// Resolve the connection within the tenant scope. A missing connection — or one
	// owned by another tenant, invisible under this scope — is a distinct
	// datasource-not-found report failure, NOT an error, and reaches no connector.
	descriptor, found, err := store.FindConnection(ctx, tenant, datasource.ConfigName)
	if err != nil {
		return nil, err
	}

	if !found {
		return []ValidationFailure{{
			Type:       ValidationDatasourceNotFound,
			ConfigName: datasource.ConfigName,
			Detail:     "no connection configured for the requested datasource",
		}}, nil
	}

	// Resolve the schema cache-first. A cache hit validates WITHOUT connector
	// discovery; a cache miss discovers live and validates from the discovered
	// snapshot. A source-down failure surfaces here as a safe EngineError.
	snapshot, err := e.resolveSchema(ctx, tenant, descriptor, false)
	if err != nil {
		return nil, err
	}

	return validateMappingAgainstSnapshot(datasource, snapshot), nil
}

// validateMappingAgainstSnapshot validates one datasource's tables, fields, and
// filter references against its schema snapshot, recording each missing table /
// field / invalid filter as a distinct failure. It never touches the network,
// credentials, or extracted data — only the snapshot's table and field names.
//
// Field matching follows the legacy DataSourceSchema.HasField semantics: a field
// name matches when it is EITHER an exact column OR a PARENT object of a dotted
// field path (a request for "natural_person" is valid when the snapshot holds
// "natural_person.mother_name"). The Mongo/plugin_crm path flattens nested
// objects into dotted names, so parent-object references are a primary pattern.
// The "." delimiter is required so "natural" never falsely matches
// "natural_person.x". The same matcher applies to mapped fields and filters.
func validateMappingAgainstSnapshot(datasource DatasourceMapping, snapshot SchemaSnapshot) []ValidationFailure {
	var failures []ValidationFailure

	fieldsByTable := indexSnapshotFields(snapshot)

	for _, table := range datasource.Tables {
		index, tableExists := fieldsByTable[table.Name]
		if !tableExists {
			failures = append(failures, ValidationFailure{
				Type:       ValidationTableNotFound,
				ConfigName: datasource.ConfigName,
				Table:      table.Name,
				Detail:     "table not found in datasource schema",
			})

			continue
		}

		for _, field := range table.Fields {
			if !index.has(field) {
				failures = append(failures, ValidationFailure{
					Type:       ValidationFieldNotFound,
					ConfigName: datasource.ConfigName,
					Table:      table.Name,
					Field:      field,
					Detail:     "field not found in table schema",
				})
			}
		}

		for _, filterField := range table.FilterFields {
			if !index.has(filterField) {
				failures = append(failures, ValidationFailure{
					Type:       ValidationInvalidFilter,
					ConfigName: datasource.ConfigName,
					Table:      table.Name,
					Field:      filterField,
					Detail:     "filter references a field not found in table schema",
				})
			}
		}
	}

	return failures
}

// fieldIndex is a per-table field lookup that supports both exact membership and
// parent-object matching against dotted field paths. It keeps an exact-membership
// set for O(1) exact hits and a sorted name slice for a bounded prefix probe.
type fieldIndex struct {
	exact  map[string]struct{}
	sorted []string
}

// has reports whether a requested field name is present, matching the legacy
// HasField semantics: an exact column, OR a parent object of any dotted field
// path (name == col or col starts with name + "."). The "." delimiter prevents a
// bare string prefix ("natural") from matching a sibling path
// ("natural_person.x").
func (i fieldIndex) has(name string) bool {
	if _, ok := i.exact[name]; ok {
		return true
	}

	// Parent-object match: find the first sorted field >= name+"." and confirm it
	// still carries that prefix. Fields are bounded by Engine limits, so the
	// sort.Search probe stays cheap.
	prefix := name + "."

	pos := sort.SearchStrings(i.sorted, prefix)
	if pos < len(i.sorted) && strings.HasPrefix(i.sorted[pos], prefix) {
		return true
	}

	return false
}

// indexSnapshotFields builds a table -> fieldIndex map from a snapshot. Each
// index carries an exact-membership set and a sorted name slice so validation can
// do exact lookups and bounded parent-object prefix probes.
func indexSnapshotFields(snapshot SchemaSnapshot) map[string]fieldIndex {
	index := make(map[string]fieldIndex, len(snapshot.Tables))

	for _, table := range snapshot.Tables {
		exact := make(map[string]struct{}, len(table.Fields))
		sorted := make([]string, 0, len(table.Fields))

		for _, field := range table.Fields {
			exact[field] = struct{}{}
			sorted = append(sorted, field)
		}

		sort.Strings(sorted)

		index[table.Name] = fieldIndex{exact: exact, sorted: sorted}
	}

	return index
}

// validateRequestNames rejects empty or whitespace-only identities anywhere in
// the request — config name, table name, mapped field, or filter field — as a
// malformed-request CategoryValidation error. This is distinct from a well-formed
// mapping that fails snapshot membership: an empty name is never a legitimate
// schema mismatch.
func validateRequestNames(request SchemaValidationRequest) error {
	for _, datasource := range request.Datasources {
		if isBlank(datasource.ConfigName) {
			return NewEngineError(CategoryValidation, "schema validation request has a blank datasource config name")
		}

		for _, table := range datasource.Tables {
			if isBlank(table.Name) {
				return NewEngineError(CategoryValidation, "schema validation request has a blank table name")
			}

			for _, field := range table.Fields {
				if isBlank(field) {
					return NewEngineError(CategoryValidation, "schema validation request has a blank field name")
				}
			}

			for _, filterField := range table.FilterFields {
				if isBlank(filterField) {
					return NewEngineError(CategoryValidation, "schema validation request has a blank filter field name")
				}
			}
		}
	}

	return nil
}

// isBlank reports whether a name is empty or whitespace-only.
func isBlank(name string) bool {
	return strings.TrimSpace(name) == ""
}

// limitFailures enforces the effective Engine count limits against the request
// and returns a distinct limit-exceeded failure for each breached scope. A
// breach short-circuits validation BEFORE any connector access. The filter-count
// bound reuses MaxFieldsPerTable: filter references are field references, so they
// share the per-table field budget rather than introducing a separate limit.
func limitFailures(request SchemaValidationRequest, limits Limits) []ValidationFailure {
	var failures []ValidationFailure

	if limits.MaxDatasources > 0 && len(request.Datasources) > limits.MaxDatasources {
		failures = append(failures, ValidationFailure{
			Type:   ValidationLimitExceeded,
			Detail: "datasource count exceeds the configured limit",
		})
	}

	for _, datasource := range request.Datasources {
		if limits.MaxTablesPerDatasource > 0 && len(datasource.Tables) > limits.MaxTablesPerDatasource {
			failures = append(failures, ValidationFailure{
				Type:       ValidationLimitExceeded,
				ConfigName: datasource.ConfigName,
				Detail:     "table count exceeds the configured limit",
			})
		}

		for _, table := range datasource.Tables {
			if limits.MaxFieldsPerTable > 0 && len(table.Fields) > limits.MaxFieldsPerTable {
				failures = append(failures, ValidationFailure{
					Type:       ValidationLimitExceeded,
					ConfigName: datasource.ConfigName,
					Table:      table.Name,
					Detail:     "field count exceeds the configured limit",
				})
			}

			if limits.MaxFieldsPerTable > 0 && len(table.FilterFields) > limits.MaxFieldsPerTable {
				failures = append(failures, ValidationFailure{
					Type:       ValidationLimitExceeded,
					ConfigName: datasource.ConfigName,
					Table:      table.Name,
					Detail:     "filter count exceeds the configured limit",
				})
			}
		}
	}

	return failures
}

// newValidationReport assembles a ValidationReport from the accumulated
// failures, sorting them into a stable order so the report shape is
// deterministic regardless of map-iteration order during validation. Valid is
// true only when there are no failures.
func newValidationReport(failures []ValidationFailure) ValidationReport {
	if len(failures) == 0 {
		return ValidationReport{Valid: true}
	}

	sort.SliceStable(failures, func(i, j int) bool {
		left, right := failures[i], failures[j]

		if left.ConfigName != right.ConfigName {
			return left.ConfigName < right.ConfigName
		}

		if left.Table != right.Table {
			return left.Table < right.Table
		}

		if left.Type != right.Type {
			return left.Type < right.Type
		}

		return left.Field < right.Field
	})

	return ValidationReport{Valid: false, Failures: failures}
}
