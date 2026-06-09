// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/engine/memory"
)

// fakeActiveExecutionChecker is a logical ActiveExecutionChecker test double. It
// reports a fixed active/inactive verdict or a transport-style error, records the
// config names it was asked about, and never touches durable job storage —
// proving the conflict gate is reusable without making job persistence mandatory
// in the Engine core.
type fakeActiveExecutionChecker struct {
	mu        sync.Mutex
	active    bool
	err       error
	asked     []string
	tenants   []engine.TenantContext
	callCount int
}

func (f *fakeActiveExecutionChecker) HasActiveExecutions(
	_ context.Context,
	tenant engine.TenantContext,
	configName string,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.callCount++
	f.asked = append(f.asked, configName)
	f.tenants = append(f.tenants, tenant)

	if f.err != nil {
		return false, f.err
	}

	return f.active, nil
}

func (f *fakeActiveExecutionChecker) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.callCount
}

// engineWithChecker builds an Engine wired with the in-memory store, a connector
// registry, and the supplied active-execution checker.
func engineWithChecker(t *testing.T, checker engine.ActiveExecutionChecker) (*engine.Engine, *memory.ConnectionStore) {
	t.Helper()

	store := memory.NewConnectionStore()

	eng, err := engine.New(
		engine.WithConnectorRegistry(memory.NewConnectorRegistry()),
		engine.WithConnectionStore(store),
		engine.WithActiveExecutionChecker(checker),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	return eng, store
}

// TestEngine_CheckActiveExecutions_StandaloneGate exercises the standalone
// host-facing conflict gate that a host owning its own connection persistence
// delegates to. Unlike the persistence ops, it requires no ConnectionStore and
// WRAPS (does not discard) the checker's own error so the host can preserve its
// error mapping.
func TestEngine_CheckActiveExecutions_StandaloneGate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenant := mustTenant(t, "tenant-a")

	newEngine := func(t *testing.T, checker engine.ActiveExecutionChecker) *engine.Engine {
		t.Helper()

		opts := []engine.Option{engine.WithConnectorRegistry(memory.NewConnectorRegistry())}
		if checker != nil {
			opts = append(opts, engine.WithActiveExecutionChecker(checker))
		}

		eng, err := engine.New(opts...)
		if err != nil {
			t.Fatalf("New() unexpected error: %v", err)
		}

		return eng
	}

	t.Run("no checker configured is a no-op", func(t *testing.T) {
		t.Parallel()

		if err := newEngine(t, nil).CheckActiveExecutions(ctx, tenant, "pg-main"); err != nil {
			t.Fatalf("CheckActiveExecutions without checker: unexpected error: %v", err)
		}
	})

	t.Run("active executions yield a conflict", func(t *testing.T) {
		t.Parallel()

		err := newEngine(t, &fakeActiveExecutionChecker{active: true}).CheckActiveExecutions(ctx, tenant, "pg-main")

		var engErr *engine.EngineError
		if !errors.As(err, &engErr) || engErr.Category != engine.CategoryConflict {
			t.Fatalf("CheckActiveExecutions active: error = %v, want CategoryConflict", err)
		}
	})

	t.Run("checker error is wrapped not replaced", func(t *testing.T) {
		t.Parallel()

		checkerErr := errors.New("job store down")
		err := newEngine(t, &fakeActiveExecutionChecker{err: checkerErr}).CheckActiveExecutions(ctx, tenant, "pg-main")
		if !errors.Is(err, checkerErr) {
			t.Fatalf("CheckActiveExecutions checker error: want wrapped %v, got %v", checkerErr, err)
		}
	})

	t.Run("missing tenant scope is rejected", func(t *testing.T) {
		t.Parallel()

		err := newEngine(t, &fakeActiveExecutionChecker{}).CheckActiveExecutions(ctx, engine.TenantContext{}, "pg-main")

		var engErr *engine.EngineError
		if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
			t.Fatalf("CheckActiveExecutions no tenant: error = %v, want CategoryValidation", err)
		}
	})
}

func TestEngine_UpdateConnection_BlockedByActiveExecutions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checker := &fakeActiveExecutionChecker{active: true}
	eng, store := engineWithChecker(t, checker)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	newHost := "db.replica"
	_, err := eng.UpdateConnection(ctx, tenant, "pg-main", engine.ConnectionPatch{Host: &newHost})
	if err == nil {
		t.Fatalf("UpdateConnection with active executions: expected conflict error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("UpdateConnection: error type = %T, want *engine.EngineError", err)
	}
	if engErr.Category != engine.CategoryConflict {
		t.Fatalf("UpdateConnection: category = %q, want %q", engErr.Category, engine.CategoryConflict)
	}

	// The checker must have been consulted with the connection's config name under
	// the tenant scope (logical, not durable).
	if checker.calls() != 1 {
		t.Fatalf("UpdateConnection: checker calls = %d, want 1", checker.calls())
	}
	checker.mu.Lock()
	gotName := checker.asked[0]
	gotTenant := checker.tenants[0]
	checker.mu.Unlock()
	if gotName != "pg-main" {
		t.Fatalf("UpdateConnection: checker asked about %q, want %q", gotName, "pg-main")
	}
	if gotTenant.TenantID != "tenant-a" {
		t.Fatalf("UpdateConnection: checker tenant = %+v, want tenant-a scope", gotTenant)
	}

	// The store must be unchanged: the host field must not have been patched.
	current, found, _ := store.FindConnection(ctx, tenant, "pg-main")
	if !found {
		t.Fatalf("UpdateConnection: connection vanished after blocked update")
	}
	if current.Host != "db.internal" {
		t.Fatalf("UpdateConnection: store mutated despite active-execution conflict: host = %q", current.Host)
	}
}

func TestEngine_DeleteConnection_BlockedByActiveExecutions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checker := &fakeActiveExecutionChecker{active: true}
	eng, store := engineWithChecker(t, checker)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	err := eng.DeleteConnection(ctx, tenant, "pg-main")
	if err == nil {
		t.Fatalf("DeleteConnection with active executions: expected conflict error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("DeleteConnection: error type = %T, want *engine.EngineError", err)
	}
	if engErr.Category != engine.CategoryConflict {
		t.Fatalf("DeleteConnection: category = %q, want %q", engErr.Category, engine.CategoryConflict)
	}

	if checker.calls() != 1 {
		t.Fatalf("DeleteConnection: checker calls = %d, want 1", checker.calls())
	}

	// The checker must have been consulted under the connection's tenant scope, so
	// one tenant's active work never blocks another tenant's mutation (symmetric
	// with the Update-blocked twin).
	checker.mu.Lock()
	gotName := checker.asked[0]
	gotTenant := checker.tenants[0]
	checker.mu.Unlock()
	if gotName != "pg-main" {
		t.Fatalf("DeleteConnection: checker asked about %q, want %q", gotName, "pg-main")
	}
	if gotTenant.TenantID != "tenant-a" {
		t.Fatalf("DeleteConnection: checker tenant = %+v, want tenant-a scope", gotTenant)
	}

	// The store must be unchanged: the connection must still be present.
	if _, found, _ := store.FindConnection(ctx, tenant, "pg-main"); !found {
		t.Fatalf("DeleteConnection: store mutated despite active-execution conflict")
	}
}

// TestEngine_DeleteConnection_MissingDoesNotConsultChecker proves the
// active-execution guard never runs for a connection that does not exist: Delete
// confirms existence FIRST, so a missing connection short-circuits to NotFound
// before the checker is consulted (B4 finding #1). Wiring an always-active
// checker makes the regression observable — if the guard ran, the call count
// would be 1 (and the error would be a conflict, not not-found).
func TestEngine_DeleteConnection_MissingDoesNotConsultChecker(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checker := &fakeActiveExecutionChecker{active: true}
	eng, _ := engineWithChecker(t, checker)
	tenant := mustTenant(t, "tenant-a")

	err := eng.DeleteConnection(ctx, tenant, "never-created")
	if err == nil {
		t.Fatalf("DeleteConnection missing: expected not-found error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("DeleteConnection missing: error type = %T, want *engine.EngineError", err)
	}
	if engErr.Category != engine.CategoryNotFound {
		t.Fatalf("DeleteConnection missing: category = %q, want %q", engErr.Category, engine.CategoryNotFound)
	}

	if checker.calls() != 0 {
		t.Fatalf("DeleteConnection missing: checker calls = %d, want 0 (guard must not run on a missing connection)", checker.calls())
	}
}

// TestEngine_UpdateConnection_MissingDoesNotConsultChecker is the symmetric proof
// for Update, which also Finds before guarding: a missing connection yields
// NotFound without consulting the active-execution checker (B4 finding #1).
func TestEngine_UpdateConnection_MissingDoesNotConsultChecker(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checker := &fakeActiveExecutionChecker{active: true}
	eng, _ := engineWithChecker(t, checker)
	tenant := mustTenant(t, "tenant-a")

	newHost := "db.replica"
	_, err := eng.UpdateConnection(ctx, tenant, "never-created", engine.ConnectionPatch{Host: &newHost})
	if err == nil {
		t.Fatalf("UpdateConnection missing: expected not-found error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("UpdateConnection missing: error type = %T, want *engine.EngineError", err)
	}
	if engErr.Category != engine.CategoryNotFound {
		t.Fatalf("UpdateConnection missing: category = %q, want %q", engErr.Category, engine.CategoryNotFound)
	}

	if checker.calls() != 0 {
		t.Fatalf("UpdateConnection missing: checker calls = %d, want 0 (guard must not run on a missing connection)", checker.calls())
	}
}

func TestEngine_UpdateAndDelete_ProceedWhenCheckerAbsent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// engineWithStore wires NO active-execution checker: the conflict gate is
	// optional, so mutations must proceed.
	eng, store := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	newHost := "db.replica"
	updated, err := eng.UpdateConnection(ctx, tenant, "pg-main", engine.ConnectionPatch{Host: &newHost})
	if err != nil {
		t.Fatalf("UpdateConnection without checker: unexpected error: %v", err)
	}
	if updated.Host != "db.replica" {
		t.Fatalf("UpdateConnection without checker: Host = %q, want %q", updated.Host, "db.replica")
	}

	if err := eng.DeleteConnection(ctx, tenant, "pg-main"); err != nil {
		t.Fatalf("DeleteConnection without checker: unexpected error: %v", err)
	}
	if _, found, _ := store.FindConnection(ctx, tenant, "pg-main"); found {
		t.Fatalf("DeleteConnection without checker: connection still present after delete")
	}
}

func TestEngine_UpdateAndDelete_ProceedWhenNoActiveExecutions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	checker := &fakeActiveExecutionChecker{active: false}
	eng, store := engineWithChecker(t, checker)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	newHost := "db.replica"
	updated, err := eng.UpdateConnection(ctx, tenant, "pg-main", engine.ConnectionPatch{Host: &newHost})
	if err != nil {
		t.Fatalf("UpdateConnection no-active: unexpected error: %v", err)
	}
	if updated.Host != "db.replica" {
		t.Fatalf("UpdateConnection no-active: Host = %q, want %q", updated.Host, "db.replica")
	}

	if err := eng.DeleteConnection(ctx, tenant, "pg-main"); err != nil {
		t.Fatalf("DeleteConnection no-active: unexpected error: %v", err)
	}
	if _, found, _ := store.FindConnection(ctx, tenant, "pg-main"); found {
		t.Fatalf("DeleteConnection no-active: connection still present after delete")
	}

	// The checker was consulted once per mutation (update + delete).
	if checker.calls() != 2 {
		t.Fatalf("checker calls = %d, want 2 (one per mutation)", checker.calls())
	}
}

func TestEngine_UpdateAndDelete_CheckerErrorIsSafeAndNoMutation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// The checker error carries secret-looking text to prove the Engine does not
	// surface the raw checker error across its boundary.
	checker := &fakeActiveExecutionChecker{err: errors.New("job store down: dsn=secret")}
	eng, store := engineWithChecker(t, checker)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	newHost := "db.replica"
	_, updErr := eng.UpdateConnection(ctx, tenant, "pg-main", engine.ConnectionPatch{Host: &newHost})
	if updErr == nil {
		t.Fatalf("UpdateConnection with checker error: expected error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(updErr, &engErr) {
		t.Fatalf("UpdateConnection: error type = %T, want *engine.EngineError", updErr)
	}
	if strings.Contains(engErr.Error(), "secret") {
		t.Fatalf("UpdateConnection: error leaked checker internals: %q", engErr.Error())
	}

	// Update must not have mutated the store on a checker failure.
	current, found, _ := store.FindConnection(ctx, tenant, "pg-main")
	if !found || current.Host != "db.internal" {
		t.Fatalf("UpdateConnection: store mutated despite checker failure: found=%v host=%q", found, current.Host)
	}

	delErr := eng.DeleteConnection(ctx, tenant, "pg-main")
	if delErr == nil {
		t.Fatalf("DeleteConnection with checker error: expected error, got nil")
	}
	if !errors.As(delErr, &engErr) {
		t.Fatalf("DeleteConnection: error type = %T, want *engine.EngineError", delErr)
	}

	// Delete must not have removed the connection on a checker failure.
	if _, found, _ := store.FindConnection(ctx, tenant, "pg-main"); !found {
		t.Fatalf("DeleteConnection: store mutated despite checker failure")
	}
}
