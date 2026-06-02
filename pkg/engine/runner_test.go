// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
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

	conn := &memory.Connector{
		Rows:   rows,
		Schema: engine.SchemaSnapshot{ConfigName: configName, Tables: tables},
	}
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

	// PlanExtraction builds + closes a connector per datasource to discover schema.
	// Discard that setup lifecycle so the post-plan assertions measure ONLY the
	// runner's connector lifecycle (the close-every-opened-connector invariant).
	for _, factory := range h.factories {
		if factory.Conn != nil {
			factory.Conn.ResetLifecycle()
		}
	}

	return plan
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

	conn.ResetLifecycle()

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Direct == nil || result.Direct.RowCount != 1 {
		t.Fatalf("expected a direct result with RowCount 1, got %+v", result.Direct)
	}

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

	if result.Reference.Path == "" {
		t.Fatalf("store mode must return a ResultReference with a path, got %+v", result.Reference)
	}

	// The reference resolves to the persisted payload through the sink.
	stored, ok := sink.Get(result.Reference.Path)
	if !ok {
		t.Fatalf("sink has no payload at reference path %q", result.Reference.Path)
	}

	var decoded map[string]map[string][]map[string]any
	if err := json.Unmarshal(stored, &decoded); err != nil {
		t.Fatalf("stored payload is not valid JSON: %v", err)
	}

	if len(decoded["pg-main"]["public.users"]) != 2 {
		t.Fatalf("stored pg-main rows = %d, want 2", len(decoded["pg-main"]["public.users"]))
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
	sink.PutErr = errors.New("sink down")
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

	if result.Reference.Path != "" {
		t.Fatalf("forced direct mode must not write a store reference, got %q", result.Reference.Path)
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

	if result.Reference.Path != "" {
		t.Fatalf("direct mode must not return a store reference, got %q", result.Reference.Path)
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
