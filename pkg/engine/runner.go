// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
// What the runner ALSO does (ST-T007-03):
//   - selects the result delivery mode (direct vs store) and, in store mode,
//     writes the serialized payload to the host ResultSink and returns a
//     ResultReference instead of inline bytes;
//   - carries canonical integrity + protection metadata on BOTH modes: direct
//     mode stamps an unkeyed SHA-256 content digest applied BY the engine with no
//     result-byte encryption; store mode PRESERVES exactly the integrity +
//     protection the sink/adapter reported, after validating protection.appliedBy
//     against the closed {engine, adapter, host} enumeration;
//   - drives optional, synchronous ExecutionStore status transitions
//     (running -> completed on success; running -> failed on error, including a
//     sink WRITE failure; running -> cancelled when the host context is already
//     cancelled). Status writes are BEST-EFFORT: a status-write error never
//     corrupts the extraction result. A sink WRITE failure, by contrast, DOES
//     fail the execution — the two are deliberately distinct.
//
// What the runner DELIBERATELY does NOT do (these belong to other layers, and
// reaching into them here would be a scope violation):
//   - cancellation MECHANICS during connector calls, timeout/deadline
//     enforcement, and result size-limit accounting (ST-T007-04) — only the
//     cancelled STATE semantics live here, via a single pre-flight context check;
//   - durable scheduling, async retry policy, and at-least-once redelivery — all
//     host-owned; the Engine makes optional persistence CALLS synchronously inline
//     with no goroutines, queues, or retry loops;
//   - result-byte ENCRYPTION itself (the host adapter encrypts behind the
//     ResultSink seam and reports the protection state the Engine preserves) and
//     RabbitMQ/streaming notifications — those are WORKER (host) concerns (T-010).
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
//
// RESULT-SIZE LIMIT — TWO LAYERS. MaxResultBytes is enforced by two complementary
// guards so an over-limit extraction fails FAST rather than after fully
// materializing AND serializing the whole result:
//
//  1. Per-step lower-bound (fail-fast). After each step completes, the runner
//     compact-marshals that step's rows and adds their length to a running sum.
//     Compact JSON is a SOUND LOWER BOUND of the final INDENTED payload (indenting
//     only ADDS whitespace, never removes bytes), so the per-step sum is always <=
//     the final serialized size. The moment the running sum exceeds MaxResultBytes
//     the final payload is GUARANTEED to exceed it too, and the run aborts BEFORE
//     building or querying any further step's connector — bounding peak memory to
//     roughly one over-limit step rather than the whole result twice over. The
//     per-step compact bytes are MEASURED and discarded immediately; they are never
//     retained, so the guard adds no second full buffer.
//
//  2. Post-marshal authoritative check (source of truth). The lower bound can PASS
//     while the true indented size still exceeds the limit (whitespace pushes a
//     just-under-limit compact sum over once indented). The final check on the
//     actual indented payload therefore remains and is the authoritative gate; the
//     per-step guard never replaces it, only short-circuits the obvious cases early.
//
// Both layers share the SAME CategoryLimitExceeded error and message, and both are
// gated on MaxResultBytes > 0 (a zero/negative limit means unbounded).

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

	connStore, err := e.requireConnectionStore()
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

	// Resolve the delivery mode and validate it against the configured sink BEFORE
	// any resource access. Requesting store mode without a sink is a clear,
	// up-front validation error, not a late surprise after extraction work.
	mode := e.resolveExecutionMode(plan.Mode)
	if mode == ModeStore && isNilPort(e.options.resultSink) {
		return ExtractionResult{}, NewEngineError(CategoryValidation, "store mode requires a configured result sink")
	}

	// Derive the execution deadline from the plan's effective timeout (resolved by
	// the planner from the Engine default merged with allowed overrides). A
	// positive timeout bounds the whole extraction so a slow datasource cannot
	// block unbounded; a zero timeout leaves the host context unchanged. The host
	// context's own deadline still applies — WithTimeout keeps the earlier of the
	// two — so an already-deadlined host ctx is honored either way.
	if plan.Limits.Timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, plan.Limits.Timeout)
		defer cancel()
	}

	// Mark the execution running through the optional ExecutionStore. The call is
	// synchronous and BEST-EFFORT: durable scheduling and redelivery are
	// host-owned, and a status-write failure must never corrupt the result.
	e.recordExecutionStatus(ctx, tenant, plan.RequestID, StatusRunning, "")

	// STORE mode streams the result to the sink incrementally in constant memory:
	// extraction goroutines encode rows to NDJSON batches, a single writer drains
	// them in deterministic Ordinal order, and the integrity digest is computed
	// over exactly the bytes written. The whole result is never held in memory.
	if mode == ModeStore {
		return e.runStoreExtraction(ctx, tenant, connStore, plan)
	}

	// DIRECT mode aggregates every row into the canonical map and emits a single
	// indented payload, byte-identical to the historical shape. Extraction across
	// steps may run in parallel, but the output is order-independent because
	// json.MarshalIndent sorts map keys.
	result, err := e.runDirectExtraction(ctx, tenant, connStore, plan)
	if err != nil {
		// Distinguish a cancelled context from a genuine failure for the recorded
		// terminal state. Cancellation MECHANICS (per-step checks, timeouts) are
		// ST-T007-04; here we only honor an already-cancelled host context for the
		// terminal STATE semantics this subtask owns.
		e.recordTerminalFailure(ctx, tenant, plan.RequestID, err)
		return ExtractionResult{}, err
	}

	return e.completeDirectMode(ctx, tenant, plan.RequestID, result), nil
}

// effectiveConcurrency resolves the worker-pool size from the plan's limits,
// honoring DefaultMaxConcurrency when unset and never returning less than one so
// a degenerate (zero/negative) configured concurrency still makes forward
// progress serially rather than dispatching zero workers.
func effectiveConcurrency(limits Limits) int {
	if limits.MaxConcurrency > 0 {
		return limits.MaxConcurrency
	}

	return 1
}

// extractionOutcome is the in-flight, mode-agnostic product of a successful run:
// the serialized payload, its row count, the per-config row counts, and the
// completion time. Both delivery modes are built from it.
type extractionOutcome struct {
	payload     []byte
	rowCount    int64
	rowCounts   map[string]int64
	completedAt time.Time
}

// runDirectExtraction performs the per-step extraction (build, test, query,
// close every connector) and serializes the aggregated rows. It is the DIRECT
// delivery path: it returns the plaintext bytes in-process. Steps may run
// CONCURRENTLY through a bounded worker pool, each producing its own per-config
// submap; the submaps are merged after the pool joins. The merged map is
// serialized with json.MarshalIndent, which sorts map keys, so the output bytes
// are IDENTICAL regardless of the order steps completed in.
func (e *Engine) runDirectExtraction(
	ctx context.Context,
	tenant TenantContext,
	connStore ConnectionStore,
	plan ExtractionPlan,
) (extractionOutcome, error) {
	// Pre-flight context check: when the context is ALREADY done (cancelled by the
	// host, or a deadline that elapsed before any work), exit before any connector
	// work — and map the context error to the right category (canceled vs timeout)
	// so the host sees which one fired. This is the BEFORE-first-query gate the
	// spec requires: connectors are never built or queried in this case.
	if engErr := contextError(ctx.Err()); engErr != nil {
		return extractionOutcome{}, engErr
	}

	results, err := e.runStepsParallel(ctx, tenant, connStore, plan)
	if err != nil {
		return extractionOutcome{}, err
	}

	// Aggregate rows keyed by config name then qualified table, mirroring the
	// Worker's result shape (map[database]map[table][]rows) so the serialized
	// payload is byte-compatible during the strangler migration. The submaps are
	// merged in Ordinal order; map-key sorting in MarshalIndent makes the final
	// bytes order-independent regardless.
	aggregated := make(map[string]map[string][]map[string]any, len(plan.Steps))
	rowCounts := make(map[string]int64, len(plan.Steps))

	var rowCount int64

	for _, res := range results {
		aggregated[res.configName] = res.rows
		rowCounts[res.configName] = res.count
		rowCount += res.count
	}

	// Context check DURING result assembly: after the row batches are gathered but
	// before serializing and returning them, a cancel or deadline that fired while
	// the last step ran still aborts the run rather than returning a result the
	// host has already abandoned.
	if engErr := contextError(ctx.Err()); engErr != nil {
		return extractionOutcome{}, engErr
	}

	// Serialize INDENTED (two-space) rather than compact. The serialized form is
	// part of the host-facing result contract, not an engine-private detail: an
	// in-process host (Worker/Reporter) persists these exact bytes as the stored
	// artifact and HMACs them for EXTERNAL verification. Emitting the indented shape
	// the artifact already uses lets such a host store DirectResult.Data directly,
	// eliminating a decode + re-marshal round-trip (and its transient ~MaxResultBytes
	// allocations each way) on the engine->host seam.
	//
	// DELIBERATE semantic shift: indented JSON is larger than compact, so this grows
	// PlaintextSize, changes the SHA-256 digest value, AND changes the byte count the
	// MaxResultBytes limit measures. That is intended — the limit now bounds the
	// serialized ARTIFACT size as it is actually stored, which is the size that
	// matters operationally. Both delivery modes share this payload (store mode
	// persists the same bytes via the sink), so the indented shape is consistent
	// across direct and store results.
	payload, err := json.MarshalIndent(aggregated, "", "  ")
	if err != nil {
		// The aggregated map holds only extracted data and identities; a marshal
		// failure is an unexpected internal condition. The error is not echoed.
		return extractionOutcome{}, NewEngineError(CategoryInternal, "failed to serialize extraction result")
	}

	// Enforce the effective result-size limit BEFORE the payload can be returned
	// inline or written to the sink. This is the AUTHORITATIVE post-marshal check
	// (layer 2 of the two-layer guard documented in the file header): it measures
	// the actual indented payload and remains the source of truth even when the
	// per-step lower bound passed. An over-limit result is a hard execution failure
	// (limit_exceeded): the bytes are NEVER returned or persisted. A zero limit
	// means unbounded. The check sits here, in the shared core, so it guards BOTH
	// delivery modes from a single place.
	if plan.Limits.MaxResultBytes > 0 && int64(len(payload)) > plan.Limits.MaxResultBytes {
		return extractionOutcome{}, NewEngineError(CategoryLimitExceeded, "extraction result exceeds the configured size limit")
	}

	return extractionOutcome{
		payload:     payload,
		rowCount:    rowCount,
		rowCounts:   rowCounts,
		completedAt: time.Now().UTC(),
	}, nil
}

// completeDirectMode finalizes an inline (direct) result: it stamps canonical
// integrity (an unkeyed SHA-256 content digest) and protection (not encrypted,
// applied by the engine) over the plaintext bytes, records the completed state,
// and returns the DirectResult.
func (e *Engine) completeDirectMode(
	ctx context.Context,
	tenant TenantContext,
	requestID string,
	outcome extractionOutcome,
) ExtractionResult {
	e.recordExecutionStatus(ctx, tenant, requestID, StatusCompleted, "")

	return ExtractionResult{
		State: ExecutionState{
			JobID:       requestID,
			Status:      StatusCompleted,
			CompletedAt: &outcome.completedAt,
		},
		Direct: &DirectResult{
			Data:          outcome.payload,
			Format:        directResultFormat,
			RowCount:      outcome.rowCount,
			PlaintextSize: int64(len(outcome.payload)),
			Integrity:     engineContentIntegrity(outcome.payload),
			Protection:    enginePlaintextProtection(),
			CompletedAt:   outcome.completedAt,
		},
		RowCounts: outcome.rowCounts,
	}
}

// resolveExecutionMode maps the requested mode onto an effective mode. ModeAuto
// (the zero value) resolves to store when a ResultSink is configured and direct
// otherwise, preserving the historical default for callers that set no mode.
func (e *Engine) resolveExecutionMode(requested ExecutionMode) ExecutionMode {
	switch requested {
	case ModeStore:
		return ModeStore
	case ModeDirect:
		return ModeDirect
	default:
		if !isNilPort(e.options.resultSink) {
			return ModeStore
		}

		return ModeDirect
	}
}

// recordTerminalFailure records the terminal state for a failed run. The state
// is chosen from the FAILURE CATEGORY, not from the live ctx.Err(): a host
// cancellation (canceled category) records the cancelled state, while everything
// else — including a deadline-exceeded TIMEOUT — records failed. Keying on the
// category, not ctx.Err(), is what keeps a timeout distinct from a cancel: both
// leave ctx.Err() non-nil, but only a host cancel is a "canceled" terminal state.
func (e *Engine) recordTerminalFailure(ctx context.Context, tenant TenantContext, requestID string, cause error) {
	message := "execution failed"

	var engErr *EngineError
	if asEngineError(cause, &engErr) {
		message = engErr.Message

		if engErr.Category == CategoryCanceled {
			e.recordExecutionStatus(ctx, tenant, requestID, StatusCanceled, message)
			return
		}
	}

	e.recordExecutionStatus(ctx, tenant, requestID, StatusFailed, message)
}

// safeConnectorError maps a connector-step failure to a safe, category-correct
// EngineError. A context cause takes precedence so a cancel/deadline surfaces as
// canceled/timeout rather than a generic unavailable: first the connector's own
// error (a well-behaved connector returns ctx.Err() when it observes the
// cancel/deadline), then the live context state (a connector that swallows the
// context but failed while the context was already done). Otherwise the supplied
// safe fallback message is used — the raw error may embed a DSN, credential, or
// driver internals, so it is DELIBERATELY discarded.
func safeConnectorError(ctx context.Context, cause error, fallbackMessage string) *EngineError {
	if engErr := contextError(cause); engErr != nil {
		return engErr
	}

	if engErr := contextError(ctx.Err()); engErr != nil {
		return engErr
	}

	return NewEngineError(CategoryUnavailable, fallbackMessage)
}

// asEngineError reports whether cause is (or wraps) a non-nil *EngineError,
// binding it to target. It centralizes the cast so the terminal-state logic
// reads against the error CATEGORY rather than the live context.
func asEngineError(cause error, target **EngineError) bool {
	return errors.As(cause, target) && *target != nil
}

// recordExecutionStatus makes a single, synchronous, BEST-EFFORT status write
// through the optional ExecutionStore. When no store is configured it is a no-op,
// so direct mode runs without durable execution tracking. A store error is
// DELIBERATELY swallowed: optional status persistence must never corrupt the
// extraction result contract (mirroring the legacy best-effort job-status writes
// and the T-003 ActiveExecutionChecker precedent that an optional-port error
// leaves the primary outcome intact). A sink WRITE failure is handled separately
// and DOES fail the execution.
func (e *Engine) recordExecutionStatus(
	ctx context.Context,
	tenant TenantContext,
	requestID string,
	status ExecutionStatus,
	failureMessage string,
) {
	if isNilPort(e.options.executionStore) {
		return
	}

	state := ExecutionState{
		JobID:          requestID,
		Status:         status,
		FailureMessage: failureMessage,
	}

	if status.IsTerminal() {
		completedAt := time.Now().UTC()
		state.CompletedAt = &completedAt
	} else {
		state.StartedAt = time.Now().UTC()
	}

	// Best-effort: discard the store error so it cannot corrupt the result.
	_ = e.options.executionStore.SaveExecution(ctx, tenant, state)
}

// engineContentIntegrity computes the canonical, engine-applied integrity over
// result bytes: an unkeyed SHA-256 content digest. HMAC is ONE possible keyed
// integrity signature; the Engine core holds no key, so it records a digest, not
// a signature. The host adapter may instead supply a keyed signature behind the
// ResultSink seam, which store mode preserves.
func engineContentIntegrity(payload []byte) *ResultIntegrity {
	sum := sha256.Sum256(payload)

	return &ResultIntegrity{
		Algorithm: "SHA-256",
		Digest:    hex.EncodeToString(sum[:]),
	}
}

// enginePlaintextProtection is the canonical protection state for direct-mode
// bytes the Engine returns in the clear: not encrypted, attributed to the engine.
// It is the RESULT protection model (ResultProtection) and is intentionally
// disjoint from the credential-protection sidecar (ProtectedCredential): it
// describes extracted RESULT bytes, never persisted datasource credentials.
func enginePlaintextProtection() *ResultProtection {
	return &ResultProtection{
		Encrypted: false,
		AppliedBy: ProtectionAppliedByEngine,
	}
}

// openStepConnector resolves the scoped connection, builds a connector
// (I/O-free), and runs the single explicit connect step (TestConnection),
// returning the connected connector. The CALLER owns the connector's Close —
// openStepConnector does NOT close on success, because the cursor it opens lives
// past this call. On EVERY failure path after a successful build, openStepConnector
// closes the connector itself, so a connector is never leaked even when connect
// fails. A build failure (or a buggy (nil,nil) build) returns before any Close is
// owed, so the failure path never dereferences a nil connector.
func (e *Engine) openStepConnector(
	ctx context.Context,
	tenant TenantContext,
	store ConnectionStore,
	step PlanStep,
) (Connector, error) {
	// Resolve the connection within the tenant scope. A missing connection — or
	// one owned by another tenant, invisible under this scope — fails as not-found
	// BEFORE any connector construction.
	descriptor, found, err := store.FindConnection(ctx, tenant, step.ConfigName)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, NewEngineError(CategoryNotFound, "extraction references an unknown datasource connection")
	}

	factory, err := e.requireConnectorFactory(descriptor.Type)
	if err != nil {
		return nil, err
	}

	// Build is I/O-free per the connector contract. A build failure is a safe,
	// redacted error: the factory's raw error may embed driver internals. A buggy
	// host Build may also return (nil, nil); treat that as a build failure too, so
	// the success path never returns a nil connector to drive.
	connector, err := factory.Build(ctx, descriptor)
	if err != nil || isNilPort(connector) {
		return nil, NewEngineError(CategoryUnavailable, "failed to build connector for connection")
	}

	// TestConnection is the single connect step. On failure the connector this
	// function built is closed HERE (the caller never receives it, so it cannot
	// close it), preserving the close-every-built-connector invariant. A context
	// error (cancel/deadline) takes precedence so the host sees the real cause.
	if err := connector.TestConnection(ctx); err != nil {
		_ = connector.Close(ctx)
		return nil, safeConnectorError(ctx, err, "failed to connect to datasource")
	}

	return connector, nil
}

// executeStep runs a single planned datasource step for the DIRECT path: open
// the connector, drive its cursor to completion, and ALWAYS close the connector.
// It returns the step's rows keyed by qualified table and the step's row count.
//
// The deferred Close is the per-step half of the close-every-opened-connector
// invariant: once openStepConnector returns a connected connector, this function
// owns and always closes it, on the cursor-error path and the success path alike.
// A Close error is swallowed: it must not mask the step's primary outcome.
func (e *Engine) executeStep(
	ctx context.Context,
	tenant TenantContext,
	store ConnectionStore,
	step PlanStep,
) (map[string][]map[string]any, int64, error) {
	// Top-of-function cancel guard: forEachStep now invokes the step fn for EVERY
	// step (so each goroutine — and, on the store path, each per-step channel — is
	// accounted for exactly once). A step whose run was already cancelled returns
	// HERE, before building any connector, preserving the "don't build connectors
	// after cancel" optimization the old dispatch-time skip provided.
	if engErr := contextError(ctx.Err()); engErr != nil {
		return nil, 0, engErr
	}

	connector, err := e.openStepConnector(ctx, tenant, store, step)
	if err != nil {
		return nil, 0, err
	}

	// ALWAYS close the connector now that we own it. Close is contractually safe
	// and idempotent enough that a double close does not panic; a close failure
	// must not mask the step's primary outcome, so its error is not surfaced.
	defer func() { _ = connector.Close(ctx) }()

	// Context check immediately BEFORE opening the cursor: a cancel/deadline
	// between connect and query stops the step without issuing the query.
	if engErr := contextError(ctx.Err()); engErr != nil {
		return nil, 0, engErr
	}

	cursor, err := connector.QueryStream(ctx, extractionRequestForStep(step))
	if err != nil {
		return nil, 0, safeConnectorError(ctx, err, "failed to query datasource")
	}

	defer func() { _ = cursor.Close(ctx) }()

	rows := make(map[string][]map[string]any)

	var count int64

	for cursor.Next(ctx) {
		table, row := cursor.Row()
		rows[table] = append(rows[table], row)
		count++
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, safeConnectorError(ctx, err, "failed to query datasource")
	}

	// A context error observed AFTER the cursor drained (a cursor that swallowed
	// the cancel and reported no Err, but the context is now done) still aborts the
	// step so a half-read result is never aggregated.
	if engErr := contextError(ctx.Err()); engErr != nil {
		return nil, 0, engErr
	}

	return rows, count, nil
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
