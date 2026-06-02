// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"sort"
)

// This file implements the Engine PlanExtraction operation: a PURE planning
// step that turns an ExtractionRequest into a deterministic, secret-free
// ExtractionPlan. It performs NO connector execution (no Query) and NO job
// publishing — those are execute-time concerns (T-007). Planning runs entirely
// through PORTS — ConnectionStore, ConnectorRegistry, and the optional
// SchemaCache — so pkg/engine stays infrastructure-free.
//
// Tenant scope is the SOLE isolation boundary (TenantContext.TenantID), validated
// BEFORE any resource access. The produced plan carries tenantId (plus the
// optional request id) for downstream adapters and observability — NEVER an
// organization or product concept, which the embedded Engine does not model.
//
// Determinism is a HARD requirement: datasource (config name), table, and field
// keys are SORTED before the plan is produced, so equivalent requests yield
// byte-stable / deeply-equal plans regardless of Go map-iteration order. The
// failure mode this guards against is map-iteration nondeterminism leaking into
// the plan shape.
//
// No raw credential material reaches the plan. The credential-preparation stage
// works only through the secret-free ConnectionDescriptor and the connector
// seam; secrets are resolved at the host seam at execute time, never materialized
// in the plan. Schema validation REUSES the T-005 ValidateSchema/resolveSchema
// path (cache-first: ConnectionStore -> optional SchemaCache -> ConnectorFactory
// -> Connector.DiscoverSchema), so the planner never duplicates schema resolution.

// PlanExtraction builds a deterministic, secret-free ExtractionPlan from an
// ExtractionRequest within the tenant scope. It validates that every mapped
// table, field, and filter reference resolves against the (cache-first) schema
// of its persisted connection, and fails BEFORE producing any executable plan
// when the request is malformed, references an unknown connection, or fails
// schema validation.
//
// Order of operations (each gate runs before the next acquires more):
//  1. require a ConnectionStore and validate the tenant scope BEFORE any access;
//  2. normalize the request and reject empty mapped fields as a CategoryValidation
//     error (distinct from a well-formed request that fails to validate);
//  3. reuse ValidateSchema (T-005) to resolve each scoped connection cache-first
//     and validate the mapped tables, fields, and filter references against the
//     schema snapshot — a missing connection becomes a CategoryNotFound error and
//     any malformed reference becomes a CategoryValidation error, both BEFORE the
//     plan is produced; a source-down/connector failure surfaces as the safe
//     CategoryUnavailable error ValidateSchema already returns;
//  4. assemble deterministic per-datasource work items with sorted tables/fields
//     and filters attached on their owning table path, carrying the effective
//     limits and tenant identity.
//
// Full limit ENFORCEMENT is ST-T006-02 and the tenant-scope enforcement detail is
// ST-T006-03; this operation lays the normalization/resolution/validation seams
// those subtasks extend.
func (e *Engine) PlanExtraction(
	ctx context.Context,
	tenant TenantContext,
	request ExtractionRequest,
) (ExtractionPlan, error) {
	ctx, end := e.startSpan(ctx, "engine.extraction.plan")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ExtractionPlan{}, err
	}

	if err := validateTenantScope(tenant); err != nil {
		return ExtractionPlan{}, err
	}

	// Reject an empty request as malformed input, distinct from a well-formed
	// request that fails schema validation. An empty mapping can never produce a
	// meaningful plan.
	if len(request.MappedFields) == 0 {
		return ExtractionPlan{}, NewEngineError(CategoryValidation, "extraction request has no mapped fields")
	}

	// Enforce tenant-scope CONSISTENCY of every referenced connection BEFORE the
	// limit/schema-validation machinery and BEFORE any connector access. A config
	// name that does not resolve within this tenant's scope — whether it truly does
	// not exist or is owned by ANOTHER tenant — is rejected here with the same safe
	// scoped not-found error, so neither the error category nor the message reveals
	// cross-tenant existence (no existence oracle), and request shaping (e.g. a
	// concurrent limit breach) cannot defer or alter that decision. This is a
	// SCOPE-consistency check only: the Engine never authorizes the actor, it only
	// confirms the supplied context owns the connections it references.
	if err := e.requireScopedConnections(ctx, tenant, store, request); err != nil {
		return ExtractionPlan{}, err
	}

	// Resolve the effective limits for THIS request: the Engine default merged
	// with the allowed per-request overrides, with copy semantics. An override
	// that exceeds an Engine maximum (or is malformed) is rejected here as a safe
	// CategoryValidation error pointing at the violated limit field — BEFORE any
	// connection resolution or connector discovery — and the Engine's shared
	// default Limits is never mutated.
	limits, err := e.options.limits.Effective(request.Overrides)
	if err != nil {
		return ExtractionPlan{}, err
	}

	// Reuse the T-005 schema-validation machinery with the EFFECTIVE limits:
	// resolve each scoped connection cache-first and validate every mapped table,
	// field, and filter reference against the discovered/cached schema. Count
	// limits (datasource / table / field / filter) are enforced inside this flow
	// (limitFailures) before any connector access; the planner never duplicates
	// either the schema resolution or the count-limit checks.
	validationRequest := schemaValidationRequestFromExtraction(request)

	report, err := e.validateSchemaWithLimits(ctx, tenant, validationRequest, limits)
	if err != nil {
		// A source-down/connector failure is already a safe, redacted EngineError.
		return ExtractionPlan{}, err
	}

	if !report.Valid {
		return ExtractionPlan{}, planValidationError(report)
	}

	return e.buildExtractionPlan(tenant, request, limits), nil
}

// requireScopedConnections confirms every datasource config name referenced by
// the request resolves to a persisted connection WITHIN the tenant scope, before
// any limit enforcement, schema validation, or connector access. It is the
// planner's own first-class scope-consistency gate rather than a side effect of
// the downstream resolve path.
//
// A blank/whitespace config name is rejected as a malformed-request
// CategoryValidation error (an empty name can never identify a connection). Any
// reference that does not resolve under this tenant — a name that exists for no
// tenant, OR a name owned by a DIFFERENT tenant (invisible under this scope) —
// yields a SINGLE safe CategoryNotFound error with an identical, redacted
// message in both cases. That identity is deliberate: it denies a cross-tenant
// existence oracle, since the caller cannot distinguish "unknown to everyone"
// from "owned by someone else". References are probed in sorted order so the
// failing name is deterministic.
//
// A store error (infrastructure failure) is surfaced as-is: it is the host's
// already-safe error, not a scope decision.
func (e *Engine) requireScopedConnections(
	ctx context.Context,
	tenant TenantContext,
	store ConnectionStore,
	request ExtractionRequest,
) error {
	for _, configName := range sortedKeys(request.MappedFields) {
		if isBlank(configName) {
			return NewEngineError(CategoryValidation, "extraction request has a blank datasource config name")
		}

		_, found, err := store.FindConnection(ctx, tenant, configName)
		if err != nil {
			return err
		}

		if !found {
			return NewEngineError(CategoryNotFound, "extraction references an unknown datasource connection")
		}
	}

	return nil
}

// schemaValidationRequestFromExtraction maps an ExtractionRequest onto the
// canonical SchemaValidationRequest, extracting per-table field selections and
// filter field references. It performs no validation itself — ValidateSchema
// owns that — it only restructures the request into the validation contract.
func schemaValidationRequestFromExtraction(request ExtractionRequest) SchemaValidationRequest {
	datasources := make([]DatasourceMapping, 0, len(request.MappedFields))

	for _, configName := range sortedKeys(request.MappedFields) {
		selection := request.MappedFields[configName]
		filterFieldsByTable := filterFieldsByTable(request.Filters, configName)

		tables := make([]TableMapping, 0, len(selection))

		for _, tableName := range sortedKeys(selection) {
			tables = append(tables, TableMapping{
				Name:         tableName,
				Fields:       sortedCopy(selection[tableName]),
				FilterFields: sortedCopy(filterFieldsByTable[tableName]),
			})
		}

		datasources = append(datasources, DatasourceMapping{
			ConfigName: configName,
			Tables:     tables,
		})
	}

	return SchemaValidationRequest{Datasources: datasources}
}

// buildExtractionPlan assembles the deterministic plan from an already-validated
// request. Datasource config names, table names, and field names are sorted so
// the plan shape is independent of map-iteration order. Filters are attached on
// their owning table path. The plan carries the tenant identity and effective
// limits and is secret-free by construction (it copies only identities and
// host-defined filter values, never credentials).
func (e *Engine) buildExtractionPlan(tenant TenantContext, request ExtractionRequest, limits Limits) ExtractionPlan {
	steps := make([]PlanStep, 0, len(request.MappedFields))

	for _, configName := range sortedKeys(request.MappedFields) {
		selection := request.MappedFields[configName]

		tables := sortedKeys(selection)

		fields := make(map[string][]string, len(selection))
		for _, table := range tables {
			fields[table] = sortedCopy(selection[table])
		}

		step := PlanStep{
			ConfigName: configName,
			Tables:     tables,
			Fields:     fields,
		}

		if filters := datasourceFilters(request.Filters, configName); len(filters) > 0 {
			step.Filters = filters
		}

		steps = append(steps, step)
	}

	return ExtractionPlan{
		TenantID:  tenant.TenantID,
		RequestID: tenant.RequestID,
		Steps:     steps,
		Metadata:  copyMetadata(request.Metadata),
		Limits:    limits,
	}
}

// datasourceFilters extracts the nested table -> field -> value filter map for a
// single datasource from the request's opaque filter map and returns a deep,
// independent copy so a caller mutating the plan cannot reach back into the
// request (or vice versa). It returns nil when the datasource has no filters.
func datasourceFilters(filters map[string]any, configName string) map[string]map[string]any {
	raw, ok := filters[configName]
	if !ok {
		return nil
	}

	tables, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	out := make(map[string]map[string]any, len(tables))

	for table, tableRaw := range tables {
		fields, ok := tableRaw.(map[string]any)
		if !ok {
			continue
		}

		copied := make(map[string]any, len(fields))
		for field, value := range fields {
			copied[field] = value
		}

		out[table] = copied
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// filterFieldsByTable returns, per table, the filter field references declared
// for a datasource. It is the validation-side view of datasourceFilters: it
// surfaces only the field NAMES (the references that must resolve against the
// schema), not their values.
func filterFieldsByTable(filters map[string]any, configName string) map[string][]string {
	tables := datasourceFilters(filters, configName)
	if len(tables) == 0 {
		return nil
	}

	out := make(map[string][]string, len(tables))

	for table, fields := range tables {
		names := make([]string, 0, len(fields))
		for field := range fields {
			names = append(names, field)
		}

		out[table] = names
	}

	return out
}

// planValidationError maps a non-valid ValidationReport onto a single safe
// CategoryValidation EngineError so the planner fails BEFORE producing a plan.
// The report's failures are already credential-free; a NotFound datasource is
// surfaced as CategoryNotFound because an unknown connection is a distinct
// resource-not-found condition the caller must fix before execution.
func planValidationError(report ValidationReport) error {
	for _, failure := range report.Failures {
		if failure.Type == ValidationDatasourceNotFound {
			return NewEngineError(CategoryNotFound, "extraction references an unknown datasource connection")
		}
	}

	return NewEngineError(CategoryValidation, "extraction request failed schema validation")
}

// copyMetadata returns an independent copy of the request metadata so the plan's
// map cannot be mutated through the request (or vice versa). It returns nil for
// empty metadata. Values are copied by reference: metadata is safe, non-secret
// data, and the keys/string values are what compatibility paths read.
func copyMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}

	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}

	return out
}

// sortedKeys returns the keys of a string-keyed map in sorted order. It is the
// single determinism primitive the planner uses to escape Go map-iteration
// nondeterminism.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

// sortedCopy returns a sorted, independent copy of a string slice so the plan's
// slices can be mutated by callers without corrupting the request or another
// plan. It returns nil for an empty input.
func sortedCopy(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, len(values))
	copy(out, values)
	sort.Strings(out)

	return out
}
