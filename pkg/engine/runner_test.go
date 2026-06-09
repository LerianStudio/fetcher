// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
	"go.uber.org/goleak"
)

// runnerHarness wires an Engine over the in-memory doubles for the runner tests.
// It seeds a ConnectionStore and a ConnectorRegistry so a plan's steps resolve to
// a descriptor and a fault-injectable connector per datasource type.
type runnerHarness struct {
	engine    *engine.Engine
	store     *memory.ConnectionStore
	registry  *memory.ConnectorRegistry
	factories map[string]*memory.ConnectorFactory
}

const runnerTenant = "tenant-runner"

// newRunnerHarness builds an Engine with the in-memory connection store and
// connector registry. It does NOT seed any connection or factory; the per-test
// setup seeds exactly the datasources it needs.
func newRunnerHarness(t *testing.T) *runnerHarness {
	t.Helper()

	store := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()

	eng, err := engine.New(
		engine.WithConnectionStore(store),
		engine.WithConnectorRegistry(registry),
	)
	if err != nil {
		t.Fatalf("engine.New: unexpected error: %v", err)
	}

	return &runnerHarness{
		engine:    eng,
		store:     store,
		registry:  registry,
		factories: make(map[string]*memory.ConnectorFactory),
	}
}

// seedSource persists a connection of the given config name and datasource type
// and registers a connector factory for that type whose connector returns the
// supplied rows. It derives the connector's DiscoverSchema snapshot from schema
// (qualified table -> field names) so PlanExtraction validates the request the
// runner will later execute. It returns the connector so the test can assert on
// its lifecycle (CloseCount, QueryCalls) and inject failures.
func (h *runnerHarness) seedSource(
	t *testing.T,
	configName, dsType string,
	schema map[string][]string,
	rows map[string][]map[string]any,
) *memory.Connector {
	t.Helper()

	tenant := engine.TenantContext{TenantID: runnerTenant}

	input := engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName: configName,
		Type:       dsType,
		Host:       "localhost",
		Port:       5432,
	})

	if _, err := h.engine.CreateConnection(context.Background(), tenant, input); err != nil {
		t.Fatalf("seed connection %q: unexpected error: %v", configName, err)
	}

	tables := make([]engine.TableSnapshot, 0, len(schema))
	for table, fields := range schema {
		tables = append(tables, engine.TableSnapshot{Name: table, Fields: fields})
	}

	conn := memory.NewTemplateConnector(memory.ConnectorBehavior{
		Rows:   rows,
		Schema: engine.SchemaSnapshot{ConfigName: configName, Tables: tables},
	})
	factory := memory.NewConnectorFactory(conn)
	h.registry.Register(dsType, factory)
	h.factories[dsType] = factory

	return conn
}

// planFor builds a deterministic ExtractionPlan over the seeded datasources by
// reusing the Engine planner, so the runner test executes a real plan rather
// than a hand-assembled one.
func (h *runnerHarness) planFor(t *testing.T, mappedFields map[string]engine.FieldSelection) engine.ExtractionPlan {
	t.Helper()

	tenant := engine.TenantContext{TenantID: runnerTenant, RequestID: "req-1"}

	plan, err := h.engine.PlanExtraction(context.Background(), tenant, engine.ExtractionRequest{
		MappedFields: mappedFields,
	})
	if err != nil {
		t.Fatalf("PlanExtraction: unexpected error: %v", err)
	}

	// PlanExtraction builds + closes a DISTINCT connector per datasource to discover
	// schema; the runner later builds its OWN distinct connector per step. Because
	// the factory builds a fresh connector per call (build-per-step), the post-plan
	// lifecycle assertions — which read the template's accessors that delegate to
	// the factory's MOST-RECENTLY-built connector (the runner's) — already measure
	// only the runner's connector. No lifecycle reset is needed: the planner's and
	// the runner's connectors are different instances with independent counters.
	return plan
}

// ndjsonStoreLine mirrors the engine's NDJSON wire envelope for store-mode
// results: one {"config","table","row"} object per line, no enclosing array.
type ndjsonStoreLine struct {
	Config string         `json:"config"`
	Table  string         `json:"table"`
	Row    map[string]any `json:"row"`
}

// decodeNDJSON parses store-mode NDJSON bytes into one ndjsonStoreLine per
// non-empty line, failing the test on any malformed line.
func decodeNDJSON(t *testing.T, data []byte) []ndjsonStoreLine {
	t.Helper()

	var out []ndjsonStoreLine

	for _, raw := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(raw) == "" {
			continue
		}

		var line ndjsonStoreLine
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			t.Fatalf("NDJSON line is not valid JSON: %q: %v", raw, err)
		}

		out = append(out, line)
	}

	return out
}

func TestExecuteExtraction_SingleSource(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}, {"id": 2}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Direct == nil {
		t.Fatalf("ExecuteExtraction: expected a direct result, got nil")
	}

	if result.Direct.RowCount != 2 {
		t.Fatalf("RowCount = %d, want 2", result.Direct.RowCount)
	}

	if conn.QueryCalls() != 1 {
		t.Fatalf("Query calls = %d, want 1", conn.QueryCalls())
	}

	// Close-on-success: every opened connector is closed.
	if conn.CloseCount() != 1 {
		t.Fatalf("CloseCount = %d, want 1 (connector must close on success)", conn.CloseCount())
	}
}

func TestExecuteExtraction_MultiSource(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)
	connA := h.seedSource(t, "pg-a", "postgres",
		map[string][]string{"public.a": {"id"}},
		map[string][]map[string]any{
			"public.a": {{"id": 1}, {"id": 2}, {"id": 3}},
		})
	connB := h.seedSource(t, "mongo-b", "mongodb",
		map[string][]string{"coll.b": {"id"}},
		map[string][]map[string]any{
			"coll.b": {{"id": 10}, {"id": 11}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-a":    {"public.a": {"id"}},
		"mongo-b": {"coll.b": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Direct == nil {
		t.Fatalf("ExecuteExtraction: expected a direct result, got nil")
	}

	// Row count equals the SUM of every fake connector's rows.
	if result.Direct.RowCount != 5 {
		t.Fatalf("RowCount = %d, want 5 (sum across sources)", result.Direct.RowCount)
	}

	// Both opened connectors are closed on the success path.
	if connA.CloseCount() != 1 {
		t.Fatalf("connA CloseCount = %d, want 1", connA.CloseCount())
	}

	if connB.CloseCount() != 1 {
		t.Fatalf("connB CloseCount = %d, want 1", connB.CloseCount())
	}

	// The serialized payload is valid JSON keyed by config name then table.
	var decoded map[string]map[string][]map[string]any
	if err := json.Unmarshal(result.Direct.Data, &decoded); err != nil {
		t.Fatalf("result data is not valid JSON: %v", err)
	}

	if len(decoded["pg-a"]["public.a"]) != 3 {
		t.Fatalf("decoded pg-a rows = %d, want 3", len(decoded["pg-a"]["public.a"]))
	}

	if len(decoded["mongo-b"]["coll.b"]) != 2 {
		t.Fatalf("decoded mongo-b rows = %d, want 2", len(decoded["mongo-b"]["coll.b"]))
	}
}

func TestExecuteExtraction_SingleSourceWithFilters(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id", "status"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1, "status": "active"}},
		})

	tenant := engine.TenantContext{TenantID: runnerTenant, RequestID: "req-filter"}

	// A request carrying a filter exercises the runner's filter projection back
	// onto the connector ExtractionRequest contract.
	plan, err := h.engine.PlanExtraction(context.Background(), tenant, engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"pg-main": {"public.users": {"id"}},
		},
		Filters: map[string]any{
			"pg-main": map[string]any{
				"public.users": map[string]any{
					"status": "active",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("PlanExtraction: unexpected error: %v", err)
	}

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Direct == nil || result.Direct.RowCount != 1 {
		t.Fatalf("expected a direct result with RowCount 1, got %+v", result.Direct)
	}

	// conn is the TEMPLATE; CloseCount delegates to the runner's freshly-built
	// connector (the last one the factory built), proving the runner closed the
	// connector IT opened — not the planner's setup connector.
	if conn.CloseCount() != 1 {
		t.Fatalf("CloseCount = %d, want 1", conn.CloseCount())
	}
}

func TestExecuteExtraction_QueryFailureReturnsSafeError(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}}, nil)
	// Inject a raw driver error that embeds a DSN-like secret. The runner MUST
	// NOT echo it across the boundary.
	conn.QueryErr = errors.New("pq: connection refused host=db.internal password=hunter2")

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("ExecuteExtraction: expected a query error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryUnavailable)
	}

	// Safe error: the raw driver message (and its embedded secret) must not leak.
	if strings.Contains(engErr.Message, "hunter2") || strings.Contains(engErr.Message, "db.internal") {
		t.Fatalf("error message leaks raw driver detail: %q", engErr.Message)
	}

	// Close-on-failure: the connector that was opened is still closed.
	if conn.CloseCount() != 1 {
		t.Fatalf("CloseCount = %d, want 1 (connector must close even when Query fails)", conn.CloseCount())
	}
}

func TestExecuteExtraction_MultiSourcePartialFailureClosesAllOpened(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)
	// Steps execute in sorted config-name order: "pg-a" succeeds, then "pg-z"
	// fails. The first connector is already opened-and-closed; the failing one
	// must also be closed despite the query error.
	connA := h.seedSource(t, "pg-a", "postgres",
		map[string][]string{"public.a": {"id"}},
		map[string][]map[string]any{
			"public.a": {{"id": 1}},
		})
	connZ := h.seedSource(t, "pg-z", "mysql",
		map[string][]string{"public.z": {"id"}}, nil)
	connZ.QueryErr = errors.New("boom")

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-a": {"public.a": {"id"}},
		"pg-z": {"public.z": {"id"}},
	})

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("ExecuteExtraction: expected a partial-failure error, got nil")
	}

	// Every connector that was OPENED is closed, on the partial-failure path.
	if connA.CloseCount() != 1 {
		t.Fatalf("connA CloseCount = %d, want 1 (already-opened connector must close)", connA.CloseCount())
	}

	if connZ.CloseCount() != 1 {
		t.Fatalf("connZ CloseCount = %d, want 1 (failing connector must close)", connZ.CloseCount())
	}
}

func TestExecuteExtraction_TestConnectionFailureClosesConnector(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}}, nil)
	conn.TestErr = errors.New("dial tcp: connection refused")

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("ExecuteExtraction: expected a connectivity error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryUnavailable)
	}

	// A connector built and tested but never queried is still closed.
	if conn.CloseCount() != 1 {
		t.Fatalf("CloseCount = %d, want 1 (connector must close when TestConnection fails)", conn.CloseCount())
	}

	if conn.QueryCalls() != 0 {
		t.Fatalf("Query calls = %d, want 0 (Query must not run after a failed TestConnection)", conn.QueryCalls())
	}
}

func TestExecuteExtraction_RejectsUnscopedTenant(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)

	// A plan with no tenant id is rejected before any connector access.
	_, err := h.engine.ExecuteExtraction(context.Background(), engine.ExtractionPlan{
		Steps: []engine.PlanStep{{ConfigName: "pg-main"}},
	})
	if err == nil {
		t.Fatalf("ExecuteExtraction: expected a validation error for an unscoped tenant, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryValidation {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryValidation)
	}
}

// newStoreRunnerHarness builds an Engine wired with a ResultSink and a recording
// ExecutionStore on top of the standard runner harness, so store-mode and
// execution-state transition tests exercise the same plan/connector path.
func newStoreRunnerHarness(t *testing.T, sink engine.ResultSink, store engine.ExecutionStore) *runnerHarness {
	t.Helper()

	connStore := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()

	opts := []engine.Option{
		engine.WithConnectionStore(connStore),
		engine.WithConnectorRegistry(registry),
	}
	if sink != nil {
		opts = append(opts, engine.WithResultSink(sink))
	}
	if store != nil {
		opts = append(opts, engine.WithExecutionStore(store))
	}

	eng, err := engine.New(opts...)
	if err != nil {
		t.Fatalf("engine.New: unexpected error: %v", err)
	}

	return &runnerHarness{
		engine:    eng,
		store:     connStore,
		registry:  registry,
		factories: make(map[string]*memory.ConnectorFactory),
	}
}

func TestExecuteExtraction_StoreModeReturnsReference(t *testing.T) {
	t.Parallel()

	sink := memory.NewResultSink()
	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}, {"id": 2}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	// Store mode returns a reference and NOT inline bytes.
	if result.Direct != nil {
		t.Fatalf("store mode must not return inline Direct bytes, got %+v", result.Direct)
	}

	if result.Reference == nil {
		t.Fatalf("store mode must return a non-nil ResultReference, got nil")
	}

	if result.Reference.Path == "" {
		t.Fatalf("store mode must return a ResultReference with a path, got %+v", result.Reference)
	}

	// The reference resolves to the persisted payload through the sink.
	stored, ok := sink.Get(result.Reference.Path)
	if !ok {
		t.Fatalf("sink has no payload at reference path %q", result.Reference.Path)
	}

	// Store mode writes NDJSON: one {"config","table","row"} object per line.
	lines := decodeNDJSON(t, stored)
	if len(lines) != 2 {
		t.Fatalf("stored NDJSON lines = %d, want 2", len(lines))
	}

	for _, line := range lines {
		if line.Config != "pg-main" || line.Table != "public.users" {
			t.Fatalf("unexpected NDJSON line config/table: %+v", line)
		}
	}

	if result.Reference.RowCount != 2 {
		t.Fatalf("Reference.RowCount = %d, want 2", result.Reference.RowCount)
	}

	if result.State.Status != engine.StatusCompleted {
		t.Fatalf("State.Status = %q, want %q", result.State.Status, engine.StatusCompleted)
	}
}

func TestExecuteExtraction_StoreModeWithoutSinkFailsClearly(t *testing.T) {
	t.Parallel()

	// No sink is configured, but the plan explicitly requests store mode.
	h := newStoreRunnerHarness(t, nil, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})
	plan.Mode = engine.ModeStore

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("store mode without a sink must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryValidation {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryValidation)
	}
}

func TestExecuteExtraction_DirectModePreservesIntegrityAndProtection(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Direct == nil {
		t.Fatalf("expected a direct result, got nil")
	}

	// Direct mode carries canonical integrity over the inline bytes: an algorithm
	// plus a digest (the Engine computes an unkeyed content digest in direct mode).
	if result.Direct.Integrity == nil {
		t.Fatalf("direct mode must carry integrity metadata, got nil")
	}

	if result.Direct.Integrity.Algorithm == "" {
		t.Fatalf("integrity.algorithm must be set")
	}

	if result.Direct.Integrity.Digest == "" && result.Direct.Integrity.Signature == "" {
		t.Fatalf("integrity must carry a digest or a signature")
	}

	// Direct mode applies no result-byte encryption: protection is present but
	// reports not-encrypted, attributed to the engine.
	if result.Direct.Protection == nil {
		t.Fatalf("direct mode must carry protection metadata, got nil")
	}

	if result.Direct.Protection.Encrypted {
		t.Fatalf("direct mode must not report encrypted result bytes")
	}

	if !result.Direct.Protection.AppliedBy.IsValid() {
		t.Fatalf("protection.appliedBy = %q is not a valid applier", result.Direct.Protection.AppliedBy)
	}

	if result.Direct.Protection.AppliedBy != engine.ProtectionAppliedByEngine {
		t.Fatalf("direct mode protection.appliedBy = %q, want %q", result.Direct.Protection.AppliedBy, engine.ProtectionAppliedByEngine)
	}
}

func TestExecuteExtraction_StoreModePreservesSinkIntegrityAndProtection(t *testing.T) {
	t.Parallel()

	// The sink reports its own canonical integrity + protection metadata; the
	// Engine MUST preserve exactly what the sink/adapter returned.
	sink := memory.NewResultSink()
	sink.ProtectionResult = &engine.ResultProtection{
		Encrypted:  true,
		KeyVersion: 7,
		Mode:       "AES-256-GCM",
		AppliedBy:  engine.ProtectionAppliedByAdapter,
	}

	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Reference.Integrity == nil || result.Reference.Integrity.Algorithm == "" {
		t.Fatalf("store mode must preserve sink integrity metadata, got %+v", result.Reference.Integrity)
	}

	if result.Reference.Integrity.Digest == "" && result.Reference.Integrity.Signature == "" {
		t.Fatalf("preserved integrity must carry a digest or a signature")
	}

	if result.Reference.Protection == nil {
		t.Fatalf("store mode must preserve sink protection metadata, got nil")
	}

	if !result.Reference.Protection.Encrypted {
		t.Fatalf("store mode must preserve the sink's encrypted=true protection")
	}

	if result.Reference.Protection.KeyVersion != 7 {
		t.Fatalf("protection.keyVersion = %d, want 7", result.Reference.Protection.KeyVersion)
	}

	if result.Reference.Protection.Mode != "AES-256-GCM" {
		t.Fatalf("protection.mode = %q, want AES-256-GCM", result.Reference.Protection.Mode)
	}

	if result.Reference.Protection.AppliedBy != engine.ProtectionAppliedByAdapter {
		t.Fatalf("protection.appliedBy = %q, want %q", result.Reference.Protection.AppliedBy, engine.ProtectionAppliedByAdapter)
	}
}

func TestExecuteExtraction_StoreModeRejectsInvalidProtectionAppliedBy(t *testing.T) {
	t.Parallel()

	// A sink that returns an out-of-enum appliedBy is a contract violation: the
	// Engine must reject it as a safe error rather than propagate a bad value.
	sink := memory.NewResultSink()
	sink.ProtectionResult = &engine.ResultProtection{
		Encrypted: true,
		AppliedBy: engine.ProtectionApplier("rogue"),
	}

	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("an invalid protection.appliedBy must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryValidation {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryValidation)
	}
}

func TestExecuteExtraction_StoreModeDoesNotReuseCredentialProtection(t *testing.T) {
	t.Parallel()

	// Result protection describes EXTRACTED RESULT bytes, never persisted
	// datasource credentials. ProtectedCredential is the credential sidecar; it
	// must NEVER appear as result protection metadata. We assert the result
	// protection type is engine.ResultProtection and carries no credential
	// ciphertext field — the types are structurally disjoint.
	sink := memory.NewResultSink()
	sink.ProtectionResult = &engine.ResultProtection{
		Encrypted: true,
		AppliedBy: engine.ProtectionAppliedByAdapter,
	}

	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	// The result protection is the canonical ResultProtection, NOT a
	// ProtectedCredential. A ProtectedCredential carries Ciphertext + KeyVersion;
	// ResultProtection carries protection STATE only and embeds no secret bytes.
	var _ engine.ResultProtection = *result.Reference.Protection

	// Round-trip the reference metadata to JSON and assert it carries no
	// credential ciphertext key, proving credential protection terminology is not
	// reused for result protection.
	encoded, err := json.Marshal(result.Reference)
	if err != nil {
		t.Fatalf("marshal reference: %v", err)
	}

	for _, leaked := range []string{"ciphertext", "Ciphertext", "ProtectedCredential"} {
		if strings.Contains(string(encoded), leaked) {
			t.Fatalf("result reference must not expose credential-protection field %q: %s", leaked, encoded)
		}
	}
}

func TestExecuteExtraction_ExecutionStoreTransitionsInOrder(t *testing.T) {
	t.Parallel()

	sink := memory.NewResultSink()
	execStore := memory.NewRecordingExecutionStore()
	h := newStoreRunnerHarness(t, sink, execStore)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	if _, err := h.engine.ExecuteExtraction(context.Background(), plan); err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	got := execStore.Statuses()
	want := []engine.ExecutionStatus{engine.StatusRunning, engine.StatusCompleted}
	if len(got) != len(want) {
		t.Fatalf("execution-store transitions = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("transition[%d] = %q, want %q (full order: %v)", i, got[i], want[i], got)
		}
	}
}

func TestExecuteExtraction_SinkWriteErrorMarksExecutionFailed(t *testing.T) {
	t.Parallel()

	sink := memory.NewResultSink()
	// Store mode streams to the sink: a failing streaming WRITE (not the legacy
	// whole-payload PersistResult) must fail the execution.
	sink.WriteErr = errors.New("sink down")
	execStore := memory.NewRecordingExecutionStore()
	h := newStoreRunnerHarness(t, sink, execStore)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("a sink write failure must fail the execution, got nil error")
	}

	got := execStore.Statuses()
	want := []engine.ExecutionStatus{engine.StatusRunning, engine.StatusFailed}
	if len(got) != len(want) {
		t.Fatalf("execution-store transitions = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("transition[%d] = %q, want %q (full order: %v)", i, got[i], want[i], got)
		}
	}
}

func TestExecuteExtraction_QueryFailureMarksExecutionFailed(t *testing.T) {
	t.Parallel()

	execStore := memory.NewRecordingExecutionStore()
	h := newStoreRunnerHarness(t, nil, execStore)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}}, nil)
	conn.QueryErr = errors.New("boom")

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	if _, err := h.engine.ExecuteExtraction(context.Background(), plan); err == nil {
		t.Fatalf("a query failure must fail the execution, got nil error")
	}

	got := execStore.Statuses()
	want := []engine.ExecutionStatus{engine.StatusRunning, engine.StatusFailed}
	if len(got) != len(want) {
		t.Fatalf("execution-store transitions = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("transition[%d] = %q, want %q (full order: %v)", i, got[i], want[i], got)
		}
	}
}

func TestExecuteExtraction_CancelledContextMarksCancelled(t *testing.T) {
	t.Parallel()

	execStore := memory.NewRecordingExecutionStore()
	h := newStoreRunnerHarness(t, nil, execStore)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := h.engine.ExecuteExtraction(ctx, plan)
	if err == nil {
		t.Fatalf("a cancelled context must fail the execution, got nil error")
	}

	got := execStore.Statuses()
	want := []engine.ExecutionStatus{engine.StatusRunning, engine.StatusCanceled}
	if len(got) != len(want) {
		t.Fatalf("execution-store transitions = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("transition[%d] = %q, want %q (full order: %v)", i, got[i], want[i], got)
		}
	}
}

func TestExecuteExtraction_StatusWriteFailureDoesNotCorruptResult(t *testing.T) {
	t.Parallel()

	// An ExecutionStore that fails its status writes must NOT corrupt the result
	// contract: status persistence is best-effort, the extraction still succeeds.
	execStore := memory.NewRecordingExecutionStore()
	execStore.SaveErr = errors.New("status store down")
	h := newStoreRunnerHarness(t, nil, execStore)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("a best-effort status-write failure must not fail the extraction: %v", err)
	}

	if result.Direct == nil || result.Direct.RowCount != 1 {
		t.Fatalf("extraction result must be intact despite status-write failure, got %+v", result.Direct)
	}
}

func TestExecuteExtraction_ModeDirectOverridesConfiguredSink(t *testing.T) {
	t.Parallel()

	// A sink IS configured (auto-mode would store), but the plan explicitly forces
	// direct mode: the result must be inline and the sink must be untouched.
	sink := memory.NewResultSink()
	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})
	plan.Mode = engine.ModeDirect

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Direct == nil || result.Direct.RowCount != 1 {
		t.Fatalf("forced direct mode must return inline rows, got %+v", result.Direct)
	}

	if result.Reference != nil {
		t.Fatalf("forced direct mode must not write a store reference, got %+v", result.Reference)
	}
}

func TestExecuteExtraction_DirectModeGreenWithoutExecutionStore(t *testing.T) {
	t.Parallel()

	// Direct mode must remain fully functional with NO execution store and NO
	// result sink configured — the baseline T-007-02 contract is unchanged.
	h := newRunnerHarness(t)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}, {"id": 2}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Direct == nil || result.Direct.RowCount != 2 {
		t.Fatalf("direct mode without an execution store must return inline rows, got %+v", result.Direct)
	}

	if result.Reference != nil {
		t.Fatalf("direct mode must not return a store reference, got %+v", result.Reference)
	}
}

// newLimitedRunnerHarness builds a runner harness whose Engine carries the given
// limits, so the runner reads the effective timeout / result-size bounds the
// planner stamps onto the plan. It optionally wires a result sink so size-limit
// tests can prove the failure happens BEFORE a sink write.
func newLimitedRunnerHarness(t *testing.T, limits engine.Limits, sink engine.ResultSink) *runnerHarness {
	t.Helper()

	connStore := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()

	opts := []engine.Option{
		engine.WithConnectionStore(connStore),
		engine.WithConnectorRegistry(registry),
		engine.WithLimits(limits),
	}
	if sink != nil {
		opts = append(opts, engine.WithResultSink(sink))
	}

	eng, err := engine.New(opts...)
	if err != nil {
		t.Fatalf("engine.New: unexpected error: %v", err)
	}

	return &runnerHarness{
		engine:    eng,
		store:     connStore,
		registry:  registry,
		factories: make(map[string]*memory.ConnectorFactory),
	}
}

func TestExecuteExtraction_CancelledBeforeQuerySkipsConnectorWork(t *testing.T) {
	// NOT t.Parallel(): goleak.VerifyNone must observe a quiet goroutine baseline.
	// Under t.Parallel() it would see sibling parallel tests' goroutines and the
	// test runner's own parked goroutine, producing false positives. Running
	// serially lets goleak prove THIS test leaks nothing.
	defer goleak.VerifyNone(t)

	h := newRunnerHarness(t)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := h.engine.ExecuteExtraction(ctx, plan)
	if err == nil {
		t.Fatalf("a context cancelled before the first query must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	// A cancelled (not deadline-exceeded) context maps to the canceled category,
	// DISTINCT from timeout.
	if engErr.Category != engine.CategoryCanceled {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryCanceled)
	}

	// No connector query work runs when cancellation happens before the query.
	if conn.QueryCalls() != 0 {
		t.Fatalf("QueryCalls = %d, want 0 (cancelled-first must not query)", conn.QueryCalls())
	}
}

func TestExecuteExtraction_CancelledDuringQueryClosesConnector(t *testing.T) {
	// NOT t.Parallel(): goleak needs a quiet baseline (see the cancel-before-query
	// test). The cancel goroutine this test spawns joins before VerifyNone runs.
	defer goleak.VerifyNone(t)

	h := newRunnerHarness(t)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}}, nil)
	// The fake query blocks on ctx; cancelling mid-query must abort cleanly and
	// still close the opened connector (the ST-02 invariant survives cancellation).
	conn.BlockOnContext = true

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	defer cancel()

	_, err := h.engine.ExecuteExtraction(ctx, plan)
	if err == nil {
		t.Fatalf("a context cancelled during query must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryCanceled {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryCanceled)
	}

	// The blocking query was entered, then the connector was closed despite the
	// mid-flight cancellation.
	if conn.QueryCalls() != 1 {
		t.Fatalf("QueryCalls = %d, want 1 (the blocking query was entered)", conn.QueryCalls())
	}

	if conn.CloseCount() != 1 {
		t.Fatalf("CloseCount = %d, want 1 (connector must close on cancellation)", conn.CloseCount())
	}
}

func TestExecuteExtraction_TimeoutDuringQueryReturnsTimeoutError(t *testing.T) {
	// NOT t.Parallel(): goleak needs a quiet baseline. The runner blocks the
	// query SYNCHRONOUSLY on ctx (no spawned goroutine), so a clean VerifyNone
	// here proves the timeout path parks nothing.
	defer goleak.VerifyNone(t)

	// A short Engine timeout limit derives an execution deadline. The fake query
	// blocks on ctx, so the deadline fires deterministically and the runner maps
	// context.DeadlineExceeded to the timeout category.
	limits := engine.DefaultLimits()
	limits.Timeout = 30 * time.Millisecond

	h := newLimitedRunnerHarness(t, limits, nil)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}}, nil)
	conn.BlockOnContext = true

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("a query exceeding the timeout must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	// Deadline exceeded maps to timeout, DISTINCT from a plain cancellation.
	if engErr.Category != engine.CategoryTimeout {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryTimeout)
	}

	// The connector that blocked on the deadline is still closed.
	if conn.CloseCount() != 1 {
		t.Fatalf("CloseCount = %d, want 1 (connector must close on timeout)", conn.CloseCount())
	}
}

func TestExecuteExtraction_TestConnectionContextErrorMapsToCanceled(t *testing.T) {
	t.Parallel()

	// A connector that returns a context error from TestConnection (a connector
	// that observed the cancel during connect) must surface as canceled, NOT a
	// generic unavailable.
	h := newRunnerHarness(t)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}}, nil)
	conn.TestErr = context.Canceled

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("a context error from TestConnection must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryCanceled {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryCanceled)
	}

	if conn.QueryCalls() != 0 {
		t.Fatalf("QueryCalls = %d, want 0 (query must not run after a context error on connect)", conn.QueryCalls())
	}
}

func TestExecuteExtraction_QueryGenericErrorWhileContextDoneMapsToContext(t *testing.T) {
	t.Parallel()

	// A connector that swallows the context and returns its OWN generic error
	// while the context is already done must surface as the context category
	// (the live-ctx guard), not unavailable — the host abandoned the request.
	h := newRunnerHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}}, nil)
	// The connector cancels the context as the query runs, then returns a generic
	// non-context error. ctx.Err() is therefore set when the runner maps the
	// failure, so the live-ctx guard (not the generic-unavailable fallback) wins.
	conn.AfterQuery = cancel
	conn.QueryErr = errors.New("driver: generic failure host=db.internal password=hunter2")

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	_, err := h.engine.ExecuteExtraction(ctx, plan)
	if err == nil {
		t.Fatalf("a generic query error while the context is done must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryCanceled {
		t.Fatalf("error category = %q, want %q (live-ctx guard must win over generic error)", engErr.Category, engine.CategoryCanceled)
	}

	// The safe message must not leak the raw driver detail even on this path.
	if strings.Contains(engErr.Message, "hunter2") {
		t.Fatalf("error message leaks raw driver detail: %q", engErr.Message)
	}
}

func TestExecuteExtraction_CancelledDuringAssemblyAfterQuery(t *testing.T) {
	t.Parallel()

	// The connector cancels the context the instant its query succeeds, so the
	// rows are gathered but the runner's DURING-assembly context guard fires
	// before serializing — the run aborts as canceled rather than returning a
	// result the host has already abandoned.
	h := newRunnerHarness(t)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}},
		})

	ctx, cancel := context.WithCancel(context.Background())
	conn.AfterQuery = cancel
	defer cancel()

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(ctx, plan)
	if err == nil {
		t.Fatalf("a context cancelled during assembly must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryCanceled {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryCanceled)
	}

	// The query ran (rows were gathered) but no result crosses the boundary.
	if conn.QueryCalls() != 1 {
		t.Fatalf("QueryCalls = %d, want 1 (the query completed before assembly aborted)", conn.QueryCalls())
	}

	if result.Direct != nil {
		t.Fatalf("a run aborted during assembly must not return inline bytes, got %+v", result.Direct)
	}

	// The opened connector is still closed despite the mid-assembly cancellation.
	if conn.CloseCount() != 1 {
		t.Fatalf("CloseCount = %d, want 1", conn.CloseCount())
	}
}

func TestExecuteExtraction_CancelledBetweenStepsSkipsRemaining(t *testing.T) {
	t.Parallel()

	// Two steps, MaxConcurrency 1: exactly ONE step holds the single slot, runs,
	// and cancels the context as it finishes; the OTHER step's top-of-function
	// cancel guard then fires, so it is never queried or built. The Direct pool
	// (forEachStep) is UNORDERED, so the test does not pin WHICH step wins the slot
	// — it asserts the order-independent invariant: exactly one step queried, the
	// other was skipped (never queried, never built by the runner), and the run
	// failed canceled with no inline bytes.
	limits := engine.DefaultLimits()
	limits.MaxConcurrency = 1

	h := newLimitedRunnerHarness(t, limits, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connA := h.seedSource(t, "pg-a", "postgres",
		map[string][]string{"public.a": {"id"}},
		map[string][]map[string]any{"public.a": {{"id": 1}}})
	connA.AfterQuery = cancel
	connZ := h.seedSource(t, "pg-z", "mysql",
		map[string][]string{"public.z": {"id"}},
		map[string][]map[string]any{"public.z": {{"id": 2}}})
	connZ.AfterQuery = cancel

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-a": {"public.a": {"id"}},
		"pg-z": {"public.z": {"id"}},
	})

	// Build counts after planning (the planner built one connector per datasource
	// for schema discovery). The SKIPPED step's count must not grow.
	pgBuildsAfterPlan := h.factories["postgres"].BuildCount()
	zBuildsAfterPlan := h.factories["mysql"].BuildCount()

	_, err := h.engine.ExecuteExtraction(ctx, plan)
	if err == nil {
		t.Fatalf("a context cancelled between steps must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryCanceled {
		t.Fatalf("error = %v, want canceled-category EngineError", err)
	}

	// Exactly one step queried (whichever won the single slot); the other was
	// skipped by the cancel guard. Their roles are interchangeable under the
	// unordered pool, so assert the invariant symmetrically.
	totalQueries := connA.QueryCalls() + connZ.QueryCalls()
	if totalQueries != 1 {
		t.Fatalf("total QueryCalls = %d (connA=%d connZ=%d), want exactly 1 (one ran, one skipped after cancel)",
			totalQueries, connA.QueryCalls(), connZ.QueryCalls())
	}

	// The step that ran closed its connector; the skipped step built nothing new.
	if connA.QueryCalls() == 1 {
		if connA.CloseCount() != 1 {
			t.Fatalf("the running step's connector must close: connA CloseCount = %d, want 1", connA.CloseCount())
		}

		if grew := h.factories["mysql"].BuildCount(); grew != zBuildsAfterPlan {
			t.Fatalf("skipped step pg-z was built: BuildCount = %d, want %d (unchanged)", grew, zBuildsAfterPlan)
		}
	} else {
		if connZ.CloseCount() != 1 {
			t.Fatalf("the running step's connector must close: connZ CloseCount = %d, want 1", connZ.CloseCount())
		}

		if grew := h.factories["postgres"].BuildCount(); grew != pgBuildsAfterPlan {
			t.Fatalf("skipped step pg-a was built: BuildCount = %d, want %d (unchanged)", grew, pgBuildsAfterPlan)
		}
	}
}

func TestExecuteExtraction_ResultSizeExceededFailsBeforeDirectReturn(t *testing.T) {
	t.Parallel()

	// A tiny result-size limit makes the (small) result oversized. The execution
	// must fail with a limit-exceeded error and return NO inline payload.
	limits := engine.DefaultLimits()
	limits.MaxResultBytes = 8

	h := newLimitedRunnerHarness(t, limits, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}, {"id": 2}, {"id": 3}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("an oversized result must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryLimitExceeded {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryLimitExceeded)
	}

	// No inline payload crosses the boundary for an over-limit result.
	if result.Direct != nil {
		t.Fatalf("oversized result must not return inline Direct bytes, got %+v", result.Direct)
	}
}

func TestExecuteExtraction_ResultSizeExceededFailsFastBetweenSteps(t *testing.T) {
	t.Parallel()

	// Direct-mode size guard. The Direct pool (forEachStep) is an UNORDERED
	// semaphore: it does not guarantee which step runs first, so a fail-fast test
	// cannot deterministically pin "step pg-a ran, pg-z skipped" to a NAMED step.
	// This test therefore uses a SINGLE oversized source — its rows alone overflow
	// the tiny MaxResultBytes — to deterministically prove the over-limit Direct
	// result fails with limit_exceeded, returns NO inline bytes, and closes the
	// connector it opened. The "skip remaining steps after fail-fast" property is
	// covered deterministically by the cancel-skip and store-mode fail-fast tests.
	limits := engine.DefaultLimits()
	limits.MaxResultBytes = 8

	h := newLimitedRunnerHarness(t, limits, nil)
	connA := h.seedSource(t, "pg-a", "postgres",
		map[string][]string{"public.a": {"id", "name"}},
		map[string][]map[string]any{
			"public.a": {
				{"id": 1, "name": "alice-with-a-long-enough-value-to-exceed-eight-bytes"},
				{"id": 2, "name": "bob-also-contributes-more-than-the-tiny-limit"},
			},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-a": {"public.a": {"id", "name"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("an oversized result must fail fast, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryLimitExceeded {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryLimitExceeded)
	}

	// No inline payload crosses the boundary for an over-limit result.
	if result.Direct != nil {
		t.Fatalf("oversized result must not return inline Direct bytes, got %+v", result.Direct)
	}

	// The step ran and its connector was closed (no leak on the over-limit path).
	if connA.QueryCalls() != 1 {
		t.Fatalf("connA QueryCalls = %d, want 1 (the source must run)", connA.QueryCalls())
	}

	if connA.CloseCount() != 1 {
		t.Fatalf("connA CloseCount = %d, want 1 (the connector must close)", connA.CloseCount())
	}
}

func TestExecuteExtraction_ResultSizeExceededOnlyByIndentation(t *testing.T) {
	t.Parallel()

	// Layer 2 (authoritative post-marshal check): the per-step COMPACT lower bound
	// passes, but the final INDENTED payload — larger by the whitespace indenting
	// adds — exceeds the limit. The early guard does NOT catch this; the final
	// post-marshal check must, proving the two layers are complementary and the
	// post-marshal check remains the source of truth.
	//
	// The limit is pinned BETWEEN the compact size and the indented size of a
	// single-step result, computed from the same shape the runner serializes.
	rows := map[string][]map[string]any{
		"public.users": {
			{"id": 1, "status": "active"},
			{"id": 2, "status": "inactive"},
			{"id": 3, "status": "active"},
		},
	}
	aggregated := map[string]map[string][]map[string]any{"pg-main": rows}

	compact, err := json.Marshal(aggregated)
	if err != nil {
		t.Fatalf("marshal compact: %v", err)
	}

	indented, err := json.MarshalIndent(aggregated, "", "  ")
	if err != nil {
		t.Fatalf("marshal indented: %v", err)
	}

	if len(indented) <= len(compact) {
		t.Fatalf("test invariant broken: indented (%d) must exceed compact (%d)", len(indented), len(compact))
	}

	// A limit strictly between the two sizes: the per-step compact sum (== len(compact)
	// for a single step) stays at or below it, but the indented payload overshoots.
	limit := int64(len(compact)+len(indented)) / 2
	if limit < int64(len(compact)) || limit >= int64(len(indented)) {
		// Fall back to a value provably between the two when the midpoint is degenerate.
		limit = int64(len(compact))
	}

	limits := engine.DefaultLimits()
	limits.MaxResultBytes = limit

	h := newLimitedRunnerHarness(t, limits, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id", "status"}}, rows)

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id", "status"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("a result that fits compact but exceeds indented must be rejected, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryLimitExceeded {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryLimitExceeded)
	}

	if result.Direct != nil {
		t.Fatalf("an over-limit (indented) result must not return inline bytes, got %+v", result.Direct)
	}
}

func TestExecuteExtraction_ResultSizeExceededFailsBeforeSinkWrite(t *testing.T) {
	t.Parallel()

	// In store mode, an oversized result must fail BEFORE the sink is written:
	// the engine never persists an over-limit payload. A recording sink proves
	// PersistResult was never called.
	limits := engine.DefaultLimits()
	limits.MaxResultBytes = 8

	sink := memory.NewResultSink()
	h := newLimitedRunnerHarness(t, limits, sink)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{
			"public.users": {{"id": 1}, {"id": 2}, {"id": 3}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})
	plan.Mode = engine.ModeStore

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("an oversized result in store mode must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryLimitExceeded {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryLimitExceeded)
	}

	// Store mode opens a streaming writer, but an over-limit run ABANDONS it
	// without Close, so NO result is finalized: the streamed bytes never become a
	// stored object, and no reference is returned.
	if sink.StoredCount() != 0 {
		t.Fatalf("sink StoredCount = %d, want 0 (over-limit result must not be finalized)", sink.StoredCount())
	}
}

func TestExecuteExtraction_TimeoutMarksExecutionFailed(t *testing.T) {
	// NOT t.Parallel(): goleak needs a quiet baseline (synchronous blocking query,
	// no spawned goroutine).
	defer goleak.VerifyNone(t)

	// A timeout records a terminal FAILED transition (deadline exceeded is a
	// failure, not a host-initiated cancel) through the execution store.
	limits := engine.DefaultLimits()
	limits.Timeout = 30 * time.Millisecond

	execStore := memory.NewRecordingExecutionStore()

	connStore := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()
	eng, err := engine.New(
		engine.WithConnectionStore(connStore),
		engine.WithConnectorRegistry(registry),
		engine.WithLimits(limits),
		engine.WithExecutionStore(execStore),
	)
	if err != nil {
		t.Fatalf("engine.New: unexpected error: %v", err)
	}
	h := &runnerHarness{engine: eng, store: connStore, registry: registry, factories: make(map[string]*memory.ConnectorFactory)}

	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}}, nil)
	conn.BlockOnContext = true

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	if _, err := h.engine.ExecuteExtraction(context.Background(), plan); err == nil {
		t.Fatalf("a timeout must fail the execution, got nil error")
	}

	got := execStore.Statuses()
	want := []engine.ExecutionStatus{engine.StatusRunning, engine.StatusFailed}
	if len(got) != len(want) {
		t.Fatalf("execution-store transitions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("transition[%d] = %q, want %q (full order: %v)", i, got[i], want[i], got)
		}
	}
}

func TestExecuteExtraction_ConnectorBuildFailureReturnsSafeError(t *testing.T) {
	t.Parallel()

	// FIX-1(a): the host factory's Build fails. The runner must surface a safe
	// EngineError, never panic, never query, and never close a connector it could
	// not build (there is nothing to close).
	h := newRunnerHarness(t)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	// Inject the build failure AFTER planning (planning needs a working build to
	// discover the schema) so only the runner's build is forced to fail.
	factory := h.factories["postgres"]
	factory.BuildErr = errors.New("driver: dsn parse failed host=db.internal password=hunter2")

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("a connector build failure must fail the execution, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryUnavailable)
	}

	// Safe error: the raw build error (and its embedded secret) must not leak.
	if strings.Contains(engErr.Message, "hunter2") || strings.Contains(engErr.Message, "db.internal") {
		t.Fatalf("build error leaks raw driver detail: %q", engErr.Message)
	}

	if result.Direct != nil {
		t.Fatalf("a build failure must not return inline bytes, got %+v", result.Direct)
	}

	// The runner's build failed, so it produced NO connector to drive or close.
	if last := factory.LastBuilt(); last != nil {
		// A connector may have been built only by the planner BEFORE the failure was
		// injected; if so it was closed by planning and never queried by the runner.
		if last.QueryCalls() != 0 {
			t.Fatalf("a failed-build step must never query: QueryCalls = %d", last.QueryCalls())
		}
	}
}

func TestExecuteExtraction_NilConnectorBuildIsGuarded(t *testing.T) {
	t.Parallel()

	// FIX-1(b): a buggy host Build returns (nil, nil). The runner's isNilPort guard
	// must catch it and return a safe error, NEVER panicking and NEVER calling
	// Query or Close on the nil connector.
	h := newRunnerHarness(t)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	factory := h.factories["postgres"]
	factory.NilConnector = true

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("a (nil, nil) build must fail the execution, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryUnavailable)
	}

	if result.Direct != nil {
		t.Fatalf("a nil-connector build must not return inline bytes, got %+v", result.Direct)
	}

	// NilConnector produced no connector to record, so nothing was queried/closed.
	if last := factory.LastBuilt(); last != nil && last.QueryCalls() != 0 {
		t.Fatalf("a nil-connector step must never query: QueryCalls = %d", last.QueryCalls())
	}
}

func TestExecuteExtraction_StoreModeFillsMissingIntegrity(t *testing.T) {
	t.Parallel()

	// FIX-2: a sink that reports NO integrity drives the Engine's canonical-fill
	// fallback — a stored result always carries verifiable integrity, an
	// engine-stamped unkeyed SHA-256 digest over the persisted payload.
	sink := memory.NewResultSink()
	sink.OmitIntegrity = true

	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}, {"id": 2}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Reference == nil || result.Reference.Integrity == nil {
		t.Fatalf("the Engine must fill missing integrity, got %+v", result.Reference)
	}

	if result.Reference.Integrity.Algorithm != "SHA-256" {
		t.Fatalf("filled integrity algorithm = %q, want SHA-256", result.Reference.Integrity.Algorithm)
	}

	// The filled digest is the unkeyed SHA-256 of the persisted payload, not a
	// signature: the Engine holds no key.
	stored, ok := sink.Get(result.Reference.Path)
	if !ok {
		t.Fatalf("sink has no payload at reference path %q", result.Reference.Path)
	}

	want := sha256.Sum256(stored)
	if result.Reference.Integrity.Digest != hex.EncodeToString(want[:]) {
		t.Fatalf("filled digest = %q, want the SHA-256 of the payload", result.Reference.Integrity.Digest)
	}

	if result.Reference.Integrity.Signature != "" {
		t.Fatalf("engine-filled integrity must be a digest, not a signature")
	}
}

func TestExecuteExtraction_StoreModeFillsMissingSizeBytes(t *testing.T) {
	t.Parallel()

	// FIX-3: a sink that reports SizeBytes == 0 drives the Engine's canonical-fill
	// fallback — SizeBytes is set from the persisted payload length.
	sink := memory.NewResultSink()
	sink.OmitSizeBytes = true

	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}, {"id": 2}, {"id": 3}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Reference == nil {
		t.Fatalf("store mode must return a non-nil reference")
	}

	stored, ok := sink.Get(result.Reference.Path)
	if !ok {
		t.Fatalf("sink has no payload at reference path %q", result.Reference.Path)
	}

	if result.Reference.SizeBytes != int64(len(stored)) {
		t.Fatalf("filled SizeBytes = %d, want %d (the payload length)", result.Reference.SizeBytes, len(stored))
	}
}

func TestExecuteExtraction_RunnerClosesItsOwnFreshlyBuiltConnector(t *testing.T) {
	t.Parallel()

	// FIX-4: the factory builds a DISTINCT connector per Build() call. The planner
	// builds one (for schema discovery) and the runner builds its own. This test
	// makes the distinctness load-bearing: the runner must close the connector IT
	// built, identified by the factory's LAST-built instance — not the planner's.
	h := newRunnerHarness(t)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}, {"id": 2}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	factory := h.factories["postgres"]
	// Planning built and closed connector #1; the runner has not built yet.
	planBuilds := factory.BuildCount()
	if planBuilds < 1 {
		t.Fatalf("planning must have built at least one connector, got %d", planBuilds)
	}

	plannerConn := factory.LastBuilt()

	if _, err := h.engine.ExecuteExtraction(context.Background(), plan); err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	runnerConn := factory.LastBuilt()

	// The runner built its OWN connector: a new build, a distinct pointer.
	if factory.BuildCount() != planBuilds+1 {
		t.Fatalf("runner must build its own connector: BuildCount = %d, want %d", factory.BuildCount(), planBuilds+1)
	}

	if runnerConn == plannerConn {
		t.Fatalf("runner reused the planner's connector instance; build-per-step is not honest")
	}

	// The runner's freshly-built connector was queried and closed exactly once;
	// the planner's connector was never queried by the runner.
	if runnerConn.QueryCalls() != 1 {
		t.Fatalf("runner connector QueryCalls = %d, want 1", runnerConn.QueryCalls())
	}

	if runnerConn.CloseCount() != 1 {
		t.Fatalf("runner connector CloseCount = %d, want 1 (runner must close the connector it opened)", runnerConn.CloseCount())
	}

	if plannerConn.QueryCalls() != 0 {
		t.Fatalf("planner connector must never be queried by the runner: QueryCalls = %d", plannerConn.QueryCalls())
	}
}

func TestExecuteExtraction_PopulatesRowCounts(t *testing.T) {
	t.Parallel()

	// FIX-5: RowCounts is keyed by datasource config name; each value is that
	// source's aggregate row count and the values sum to the total.
	h := newRunnerHarness(t)
	h.seedSource(t, "pg-a", "postgres",
		map[string][]string{"public.a": {"id"}},
		map[string][]map[string]any{"public.a": {{"id": 1}, {"id": 2}, {"id": 3}}})
	h.seedSource(t, "mongo-b", "mongodb",
		map[string][]string{"coll.b": {"id"}},
		map[string][]map[string]any{"coll.b": {{"id": 10}, {"id": 11}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-a":    {"public.a": {"id"}},
		"mongo-b": {"coll.b": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.RowCounts == nil {
		t.Fatalf("RowCounts must be populated, got nil")
	}

	if result.RowCounts["pg-a"] != 3 {
		t.Fatalf("RowCounts[pg-a] = %d, want 3", result.RowCounts["pg-a"])
	}

	if result.RowCounts["mongo-b"] != 2 {
		t.Fatalf("RowCounts[mongo-b] = %d, want 2", result.RowCounts["mongo-b"])
	}

	var sum int64
	for _, n := range result.RowCounts {
		sum += n
	}

	if sum != result.Direct.RowCount {
		t.Fatalf("sum of RowCounts = %d, want %d (the total row count)", sum, result.Direct.RowCount)
	}
}

func TestExecuteExtraction_DirectModeOmitsReferenceInJSON(t *testing.T) {
	t.Parallel()

	// FIX-6: the result union is symmetric. In direct mode Reference is nil and the
	// marshaled JSON must omit the "reference" key entirely (no empty object), and
	// must carry the "direct" arm.
	h := newRunnerHarness(t)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Reference != nil {
		t.Fatalf("direct mode must leave Reference nil, got %+v", result.Reference)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	if strings.Contains(string(encoded), `"reference"`) {
		t.Fatalf("direct-mode JSON must omit the reference arm, got %s", encoded)
	}

	if !strings.Contains(string(encoded), `"direct"`) {
		t.Fatalf("direct-mode JSON must carry the direct arm, got %s", encoded)
	}

	// Symmetry check: store mode must likewise omit the empty "direct" arm.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &probe); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if _, ok := probe["reference"]; ok {
		t.Fatalf("direct-mode JSON must not contain a reference key at all")
	}
}

func TestExecuteExtraction_UnknownConnectionIsNotFound(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)

	plan := engine.ExtractionPlan{
		TenantID: runnerTenant,
		Steps:    []engine.PlanStep{{ConfigName: "ghost", Tables: []string{"public.t"}}},
	}

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("ExecuteExtraction: expected a not-found error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error is not *engine.EngineError: %T (%v)", err, err)
	}

	if engErr.Category != engine.CategoryNotFound {
		t.Fatalf("error category = %q, want %q", engErr.Category, engine.CategoryNotFound)
	}
}
