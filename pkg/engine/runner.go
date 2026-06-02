// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"encoding/json"
	"time"
)

// This file implements the Engine ExecuteExtraction operation: SYNCHRONOUS,
// DIRECT-mode execution of an already-built ExtractionPlan (T-006) through the
// host-provided connector contracts. It is the execute half of the planner: the
// plan decides WHAT to read deterministically, the runner READS it.
//
// What the runner DOES:
//   - thread the tenant scope FROM the plan (tenantId + optional requestId only —
//     never an organization or product concept);
//   - resolve each planned datasource's connection descriptor via ConnectionStore;
//   - build a connector per datasource via the ConnectorRegistry factory (I/O-free
//     build), TestConnection (the single connect step), Query, then Close;
//   - aggregate rows into the canonical DirectResult (serialized payload, total
//     row count, plaintext size, completion time).
//
// What the runner DELIBERATELY does NOT do (these belong to other layers, and
// reaching into them here would be a scope violation):
//   - store-mode persistence via ResultSink and ExecutionStore state transitions
//     (ST-T007-03) — direct mode only here, with a clean seam left at the return;
//   - cancellation, timeout, and result size-limit enforcement (ST-T007-04);
//   - result-byte ENCRYPTION, HMAC signing of stored bytes, job-status
//     persistence, and RabbitMQ/streaming notifications — those are WORKER (host)
//     concerns (T-010). The runner returns plaintext bytes inline; the host
//     protects and persists them behind its own seams.
//
// CONNECTOR LIFECYCLE INVARIANT (the load-bearing guarantee): every connector
// that is successfully built MUST be closed, on BOTH the success path and EVERY
// failure path (TestConnection error, Query error, partial multi-source
// failure). Each step opens, drives, and closes its own connector within
// executeStep, whose deferred Close runs before the step returns — so a failure
// in step N closes step N's connector, and steps 1..N-1 are already closed
// because their executeStep calls returned earlier. No connector is shared across
// goroutines: execution is single-goroutine per call, honoring the connector
// contract's single-flight ownership.
//
// FAILURE POLICY: fail-fast. The legacy Worker (queryExternalData ->
// queryDatabase) aborts on the first datasource error, and the runner preserves
// that behavior: the first failing step stops execution and returns a safe,
// redacted EngineError. A partial result is never returned. This keeps the
// direct-mode contract simple (a DirectResult is whole or absent) and matches the
// host's existing job-failure semantics; a future collect-all-errors policy would
// be an additive change, not a silent behavior shift.

// directResultFormat is the serialization format of the inline direct-mode
// payload. It mirrors the Worker's "json" result format so downstream consumers
// (Reporter) read an identical shape during the strangler migration.
const directResultFormat = "json"

// ExecuteExtraction runs an ExtractionPlan synchronously in DIRECT mode and
// returns the extracted rows inline as a DirectResult. It threads the tenant
// scope from the plan, executes each planned datasource step through the
// connector contract, closes every connector it opens, and aggregates the rows
// into the canonical result model.
//
// Order of operations (each gate runs before the next acquires more):
//  1. validate the plan's tenant scope BEFORE any resource access;
//  2. for each step, in the plan's already-sorted order, resolve the scoped
//     connection, build + test + query a connector, and close it;
//  3. aggregate rows into a deterministic result map, serialize it, and return a
//     DirectResult with the total row count and plaintext size.
//
// Errors crossing the boundary are safe, redacted EngineErrors: a connector's
// raw error may embed a DSN, credential, or driver internals, so the underlying
// detail is DELIBERATELY discarded, mirroring TestConnection and DiscoverSchema.
func (e *Engine) ExecuteExtraction(ctx context.Context, plan ExtractionPlan) (ExtractionResult, error) {
	ctx, end := e.startSpan(ctx, "engine.extraction.execute")
	defer end()

	store, err := e.requireConnectionStore()
	if err != nil {
		return ExtractionResult{}, err
	}

	// The plan is tenant-scoped from T-006; thread that scope (tenantId + optional
	// requestId) and validate it BEFORE any resource access. A plan with an empty
	// or malformed tenant id is rejected here, not at first connector use.
	tenant := TenantContext{TenantID: plan.TenantID, RequestID: plan.RequestID}
	if err := validateTenantScope(tenant); err != nil {
		return ExtractionResult{}, err
	}

	// Aggregate rows keyed by config name then qualified table, mirroring the
	// Worker's result shape (map[database]map[table][]rows) so the serialized
	// payload is byte-compatible during the strangler migration. Steps run in the
	// plan's already-sorted order, so iteration order is deterministic.
	aggregated := make(map[string]map[string][]map[string]any, len(plan.Steps))

	var rowCount int64

	for _, step := range plan.Steps {
		stepRows, stepCount, err := e.executeStep(ctx, tenant, store, step)
		if err != nil {
			// Fail-fast: the first failing step aborts execution. Its connector is
			// already closed by executeStep's deferred Close, and every earlier
			// step's connector closed when its executeStep call returned.
			return ExtractionResult{}, err
		}

		aggregated[step.ConfigName] = stepRows
		rowCount += stepCount
	}

	payload, err := json.Marshal(aggregated)
	if err != nil {
		// The aggregated map holds only extracted data and identities; a marshal
		// failure is an unexpected internal condition. The error is not echoed.
		return ExtractionResult{}, NewEngineError(CategoryInternal, "failed to serialize extraction result")
	}

	completedAt := time.Now().UTC()

	return ExtractionResult{
		State: ExecutionState{
			Status:      StatusCompleted,
			CompletedAt: &completedAt,
		},
		Direct: &DirectResult{
			Data:          payload,
			Format:        directResultFormat,
			RowCount:      rowCount,
			PlaintextSize: int64(len(payload)),
			CompletedAt:   completedAt,
		},
	}, nil
}

// executeStep runs a single planned datasource step: resolve the scoped
// connection, build a connector (I/O-free), connect, query, and ALWAYS close.
// It returns the step's rows keyed by qualified table and the step's row count.
//
// The deferred Close is the per-step half of the close-every-opened-connector
// invariant: it runs on EVERY return path — connectivity failure, query failure,
// or success — so a connector built here is never leaked. Close is registered
// only AFTER the connector is successfully built, so the build-failure path never
// dereferences a nil connector. A Close error is swallowed: it must not mask the
// step's primary outcome.
func (e *Engine) executeStep(
	ctx context.Context,
	tenant TenantContext,
	store ConnectionStore,
	step PlanStep,
) (map[string][]map[string]any, int64, error) {
	// Resolve the connection within the tenant scope. A missing connection — or
	// one owned by another tenant, invisible under this scope — fails as not-found
	// BEFORE any connector construction.
	descriptor, found, err := store.FindConnection(ctx, tenant, step.ConfigName)
	if err != nil {
		return nil, 0, err
	}

	if !found {
		return nil, 0, NewEngineError(CategoryNotFound, "extraction references an unknown datasource connection")
	}

	factory, err := e.requireConnectorFactory(descriptor.Type)
	if err != nil {
		return nil, 0, err
	}

	// Build is I/O-free per the connector contract. A build failure is a safe,
	// redacted error: the factory's raw error may embed driver internals. A buggy
	// host Build may also return (nil, nil); treat that as a build failure too,
	// BEFORE registering the deferred Close, so the success path never closes a
	// nil connector.
	connector, err := factory.Build(ctx, descriptor)
	if err != nil || isNilPort(connector) {
		return nil, 0, NewEngineError(CategoryUnavailable, "failed to build connector for connection")
	}

	// ALWAYS close the connector, on every return path below. Close is
	// contractually safe and idempotent enough that a double close does not panic;
	// a close failure must not mask the step's primary outcome, so its error is
	// not surfaced.
	defer func() { _ = connector.Close(ctx) }()

	// TestConnection is the single connect step. A connectivity failure is mapped
	// to a safe error; the connector is still closed by the deferred Close above.
	if err := connector.TestConnection(ctx); err != nil {
		return nil, 0, NewEngineError(CategoryUnavailable, "failed to connect to datasource")
	}

	// Query executes the extraction. A query failure is mapped to a safe error:
	// the connector's raw error may embed a DSN, credential, or driver internals,
	// so it is DELIBERATELY discarded from the returned message.
	queryResult, err := connector.Query(ctx, extractionRequestForStep(step))
	if err != nil {
		return nil, 0, NewEngineError(CategoryUnavailable, "failed to query datasource")
	}

	var count int64
	for _, rows := range queryResult {
		count += int64(len(rows))
	}

	return queryResult, count, nil
}

// extractionRequestForStep projects a PlanStep back onto the connector's
// ExtractionRequest contract. The plan already carries the deterministic,
// secret-free per-datasource selection (sorted fields, attached filters); the
// runner hands it to the connector as a single-datasource request. Credentials
// are NOT carried: the host's connector resolves them at connect time behind the
// factory seam.
func extractionRequestForStep(step PlanStep) ExtractionRequest {
	selection := make(FieldSelection, len(step.Fields))
	for table, fields := range step.Fields {
		selection[table] = fields
	}

	request := ExtractionRequest{
		MappedFields: map[string]FieldSelection{step.ConfigName: selection},
	}

	if len(step.Filters) > 0 {
		request.Filters = map[string]any{step.ConfigName: filtersToAny(step.Filters)}
	}

	return request
}

// filtersToAny widens the plan's typed filter map onto the connector contract's
// opaque any-shaped filter value, preserving the table -> field -> value nesting
// the adapters interpret.
func filtersToAny(filters map[string]map[string]any) map[string]any {
	out := make(map[string]any, len(filters))
	for table, fields := range filters {
		out[table] = fields
	}

	return out
}
