// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
)

// Compile-time assertions that the in-memory harness types satisfy the exact
// Engine ports. If a method set drifts from the port contract, the package
// fails to build here rather than at some distant call site.
var (
	_ engine.ConnectionStore   = (*memory.ConnectionStore)(nil)
	_ engine.ExecutionStore    = (*memory.ExecutionStore)(nil)
	_ engine.ResultSink        = (*memory.ResultSink)(nil)
	_ engine.SchemaCache       = (*memory.SchemaCache)(nil)
	_ engine.EventSink         = (*memory.EventSink)(nil)
	_ engine.ConnectorRegistry = (*memory.ConnectorRegistry)(nil)
)

// mustTenant builds a TenantContext for the given tenant ID, failing the test
// on a validation error. tenantID is the sole isolation boundary.
func mustTenant(t *testing.T, tenantID string) engine.TenantContext {
	t.Helper()

	tenant, err := engine.NewTenantContext(tenantID)
	if err != nil {
		t.Fatalf("NewTenantContext(%q): unexpected error: %v", tenantID, err)
	}

	return tenant
}

func testTenant(t *testing.T) engine.TenantContext {
	t.Helper()

	return mustTenant(t, "tenant-a")
}

func TestConnectionStore_CreateGetListUpdateDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenant := testTenant(t)
	store := memory.NewConnectionStore()

	// Create + Get.
	desc := engine.ConnectionDescriptor{ConfigName: "pg-main", Type: "postgres", Host: "h1"}
	if err := store.Create(ctx, tenant, desc, nil); err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	got, found, err := store.FindConnection(ctx, tenant, "pg-main")
	if err != nil {
		t.Fatalf("FindConnection: unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("FindConnection: expected found=true")
	}
	if got.Host != "h1" {
		t.Fatalf("FindConnection: host = %q, want %q", got.Host, "h1")
	}

	// Duplicate create must be a stable validation error.
	if err := store.Create(ctx, tenant, desc, nil); err == nil {
		t.Fatalf("Create duplicate: expected error, got nil")
	}

	// Miss for unknown config.
	if _, found, _ := store.FindConnection(ctx, tenant, "missing"); found {
		t.Fatalf("FindConnection missing: expected found=false")
	}

	// List with deterministic (sorted) ordering.
	for _, name := range []string{"zeta", "alpha", "mike"} {
		if err := store.Create(ctx, tenant, engine.ConnectionDescriptor{ConfigName: name, Type: "postgres"}, nil); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	list, err := store.List(ctx, tenant)
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	wantOrder := []string{"alpha", "mike", "pg-main", "zeta"}
	if len(list) != len(wantOrder) {
		t.Fatalf("List: len = %d, want %d", len(list), len(wantOrder))
	}
	for i, want := range wantOrder {
		if list[i].ConfigName != want {
			t.Fatalf("List[%d].ConfigName = %q, want %q (ordering must be deterministic)", i, list[i].ConfigName, want)
		}
	}

	// Update existing.
	if err := store.Update(ctx, tenant, engine.ConnectionDescriptor{ConfigName: "pg-main", Type: "postgres", Host: "h2"}, nil); err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}
	got, _, _ = store.FindConnection(ctx, tenant, "pg-main")
	if got.Host != "h2" {
		t.Fatalf("Update: host = %q, want %q", got.Host, "h2")
	}

	// Update unknown is a not-found error.
	if err := store.Update(ctx, tenant, engine.ConnectionDescriptor{ConfigName: "ghost"}, nil); err == nil {
		t.Fatalf("Update unknown: expected error, got nil")
	}

	// Delete.
	if err := store.Delete(ctx, tenant, "pg-main"); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}
	if _, found, _ := store.FindConnection(ctx, tenant, "pg-main"); found {
		t.Fatalf("Delete: expected pg-main to be gone")
	}

	// Delete unknown is a not-found error.
	if err := store.Delete(ctx, tenant, "pg-main"); err == nil {
		t.Fatalf("Delete unknown: expected error, got nil")
	}
}

func TestConnectionStore_TenantIsolation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewConnectionStore()
	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	if err := store.Create(ctx, tenantA, engine.ConnectionDescriptor{ConfigName: "shared", Type: "postgres"}, nil); err != nil {
		t.Fatalf("Create tenantA: %v", err)
	}

	// Tenant B must not see tenant A's connection.
	if _, found, _ := store.FindConnection(ctx, tenantB, "shared"); found {
		t.Fatalf("tenant isolation breached: tenantB sees tenantA connection")
	}

	listB, err := store.List(ctx, tenantB)
	if err != nil {
		t.Fatalf("List tenantB: %v", err)
	}
	if len(listB) != 0 {
		t.Fatalf("tenant isolation breached: tenantB list len = %d, want 0", len(listB))
	}
}

func TestSchemaCache_GetSetHitMiss(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenant := testTenant(t)
	cache := memory.NewSchemaCache()

	// Miss.
	if _, found, err := cache.GetSchema(ctx, tenant, "pg-main"); err != nil || found {
		t.Fatalf("GetSchema miss: found=%v err=%v, want found=false err=nil", found, err)
	}

	snapshot := engine.SchemaSnapshot{
		ConfigName: "pg-main",
		Tables:     []engine.TableSnapshot{{Name: "public.users", Fields: []string{"id", "name"}}},
	}
	if err := cache.PutSchema(ctx, tenant, snapshot); err != nil {
		t.Fatalf("PutSchema: unexpected error: %v", err)
	}

	// Hit.
	got, found, err := cache.GetSchema(ctx, tenant, "pg-main")
	if err != nil {
		t.Fatalf("GetSchema hit: unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("GetSchema hit: expected found=true")
	}
	if !got.HasTable("public.users") {
		t.Fatalf("GetSchema hit: snapshot missing expected table")
	}
}

func TestResultSink_PutGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenant := testTenant(t)
	sink := memory.NewResultSink()

	payload := []byte(`{"rows":3}`)
	ref, err := sink.PersistResult(ctx, tenant, payload)
	if err != nil {
		t.Fatalf("PersistResult: unexpected error: %v", err)
	}
	if ref.Path == "" {
		t.Fatalf("PersistResult: expected non-empty path reference")
	}
	if ref.SizeBytes != int64(len(payload)) {
		t.Fatalf("PersistResult: SizeBytes = %d, want %d", ref.SizeBytes, len(payload))
	}

	stored, found := sink.Get(ref.Path)
	if !found {
		t.Fatalf("Get: expected stored payload to be found")
	}
	if string(stored) != string(payload) {
		t.Fatalf("Get: payload = %q, want %q", stored, payload)
	}

	// Stored copy must be independent of the caller's slice (no aliasing).
	payload[0] = 'X'
	stored, _ = sink.Get(ref.Path)
	if stored[0] == 'X' {
		t.Fatalf("Get: stored payload aliases caller slice; expected a defensive copy")
	}

	if _, found := sink.Get("missing"); found {
		t.Fatalf("Get missing: expected found=false")
	}
}

func TestEventSink_Emit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenant := testTenant(t)
	sink := memory.NewEventSink()

	state := engine.ExecutionState{JobID: "job-1", Status: engine.StatusCompleted}
	if err := sink.Emit(ctx, tenant, state); err != nil {
		t.Fatalf("Emit: unexpected error: %v", err)
	}

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("Events: len = %d, want 1", len(events))
	}
	if events[0].State.JobID != "job-1" || events[0].State.Status != engine.StatusCompleted {
		t.Fatalf("Events: unexpected event payload: %+v", events[0])
	}
	if events[0].Tenant.TenantID != "tenant-a" {
		t.Fatalf("Events: tenant not captured: %+v", events[0].Tenant)
	}
}

func TestExecutionStore_StatusTransitions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenant := testTenant(t)
	store := memory.NewExecutionStore()

	// Miss before any save.
	if _, found, err := store.FindExecution(ctx, tenant, "job-1"); err != nil || found {
		t.Fatalf("FindExecution miss: found=%v err=%v", found, err)
	}

	transitions := []engine.ExecutionStatus{
		engine.StatusPending,
		engine.StatusRunning,
		engine.StatusCompleted,
	}
	for _, status := range transitions {
		if err := store.SaveExecution(ctx, tenant, engine.ExecutionState{JobID: "job-1", Status: status}); err != nil {
			t.Fatalf("SaveExecution %s: unexpected error: %v", status, err)
		}
		got, found, err := store.FindExecution(ctx, tenant, "job-1")
		if err != nil || !found {
			t.Fatalf("FindExecution after %s: found=%v err=%v", status, found, err)
		}
		if got.Status != status {
			t.Fatalf("FindExecution: status = %q, want %q", got.Status, status)
		}
	}
}

func TestConnectorRegistry_Lookup(t *testing.T) {
	t.Parallel()

	registry := memory.NewConnectorRegistry()

	type fakeConnector struct{ name string }
	registry.Register("postgres", fakeConnector{name: "pg"})

	// Found.
	conn, ok := registry.Connector("postgres")
	if !ok {
		t.Fatalf("Connector(postgres): expected ok=true")
	}
	if fc, isFake := conn.(fakeConnector); !isFake || fc.name != "pg" {
		t.Fatalf("Connector(postgres): unexpected connector %#v", conn)
	}

	// Unknown.
	if _, ok := registry.Connector("oracle"); ok {
		t.Fatalf("Connector(oracle): expected ok=false for unknown type")
	}
}

func TestConnectorRegistry_LookupOrError_StableEngineError(t *testing.T) {
	t.Parallel()

	registry := memory.NewConnectorRegistry()
	registry.Register("postgres", struct{}{})

	if _, err := registry.LookupOrError("postgres"); err != nil {
		t.Fatalf("LookupOrError known: unexpected error: %v", err)
	}

	_, err := registry.LookupOrError("oracle")
	if err == nil {
		t.Fatalf("LookupOrError unknown: expected error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("LookupOrError unknown: error is not *engine.EngineError: %T", err)
	}
	if engErr.Category != engine.CategoryNotFound {
		t.Fatalf("LookupOrError unknown: category = %q, want %q", engErr.Category, engine.CategoryNotFound)
	}
}

// TestConcurrentAccess hammers every shared map concurrently. It must be
// race-clean under `go test -race`.
func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenant := testTenant(t)

	connStore := memory.NewConnectionStore()
	execStore := memory.NewExecutionStore()
	cache := memory.NewSchemaCache()
	sink := memory.NewResultSink()
	events := memory.NewEventSink()
	registry := memory.NewConnectorRegistry()

	const workers = 32
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				name := configNameFor(id, i)

				_ = connStore.Create(ctx, tenant, engine.ConnectionDescriptor{ConfigName: name, Type: "postgres"}, nil)
				_, _, _ = connStore.FindConnection(ctx, tenant, name)
				_, _ = connStore.List(ctx, tenant)
				_ = connStore.Update(ctx, tenant, engine.ConnectionDescriptor{ConfigName: name, Type: "postgres", Host: "h"}, nil)
				_ = connStore.Delete(ctx, tenant, name)

				_ = execStore.SaveExecution(ctx, tenant, engine.ExecutionState{JobID: name, Status: engine.StatusRunning})
				_, _, _ = execStore.FindExecution(ctx, tenant, name)

				_ = cache.PutSchema(ctx, tenant, engine.SchemaSnapshot{ConfigName: name})
				_, _, _ = cache.GetSchema(ctx, tenant, name)

				ref, _ := sink.PersistResult(ctx, tenant, []byte(name))
				_, _ = sink.Get(ref.Path)

				_ = events.Emit(ctx, tenant, engine.ExecutionState{JobID: name, Status: engine.StatusCompleted})

				registry.Register(name, struct{}{})
				_, _ = registry.Connector(name)
			}
		}(w)
	}
	wg.Wait()
}

func configNameFor(worker, iteration int) string {
	return "conn-" + strconv.Itoa(worker) + "-" + strconv.Itoa(iteration)
}
