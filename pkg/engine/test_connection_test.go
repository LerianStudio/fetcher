// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
)

// testConnRecord captures connector lifecycle calls for the TestConnection
// operation tests, so a test can prove the Engine drives build -> test -> close
// through the contract and ALWAYS closes the connector.
type testConnRecord struct {
	mu    sync.Mutex
	calls []string
}

func (r *testConnRecord) note(step string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, step)
}

func (r *testConnRecord) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.calls...)
}

func (r *testConnRecord) has(step string) bool {
	for _, c := range r.snapshot() {
		if c == step {
			return true
		}
	}

	return false
}

// testConnConnector is a host-side connector double for the Engine TestConnection
// tests. testErr makes the explicit connectivity check fail; closeErr makes Close
// fail. It records every lifecycle call so a test can assert close-always.
type testConnConnector struct {
	record   *testConnRecord
	testErr  error
	closeErr error
}

func (c *testConnConnector) TestConnection(_ context.Context) error {
	c.record.note("test")
	return c.testErr
}

func (c *testConnConnector) DiscoverSchema(_ context.Context) (engine.SchemaSnapshot, error) {
	c.record.note("discover")
	return engine.SchemaSnapshot{}, nil
}

func (c *testConnConnector) Query(_ context.Context, _ engine.ExtractionRequest) (map[string][]map[string]any, error) {
	c.record.note("query")
	return nil, nil
}

func (c *testConnConnector) Close(_ context.Context) error {
	c.record.note("close")
	return c.closeErr
}

// testConnFactory builds testConnConnectors. Build is I/O-free and records the
// descriptor it received so a test can prove the SECRET-FREE descriptor is what
// reaches connector construction.
type testConnFactory struct {
	record    *testConnRecord
	buildErr  error
	testErr   error
	closeErr  error
	descrSeen engine.ConnectionDescriptor
}

func (f *testConnFactory) Build(_ context.Context, descriptor engine.ConnectionDescriptor) (engine.Connector, error) {
	f.record.note("build")
	f.descrSeen = descriptor

	if f.buildErr != nil {
		return nil, f.buildErr
	}

	return &testConnConnector{record: f.record, testErr: f.testErr, closeErr: f.closeErr}, nil
}

var (
	_ engine.Connector        = (*testConnConnector)(nil)
	_ engine.ConnectorFactory = (*testConnFactory)(nil)
)

// engineForTestConnection wires an Engine with the in-memory store and a registry
// holding the supplied factory under datasource type "postgres".
func engineForTestConnection(t *testing.T, factory engine.ConnectorFactory) (*engine.Engine, *memory.ConnectionStore) {
	t.Helper()

	store := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()

	if factory != nil {
		registry.Register("postgres", factory)
	}

	eng, err := engine.New(
		engine.WithConnectorRegistry(registry),
		engine.WithConnectionStore(store),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	return eng, store
}

// seedConnection creates a connection descriptor for the tenant directly through
// the store so the TestConnection tests start from a persisted connection.
func seedConnection(t *testing.T, store *memory.ConnectionStore, tenant engine.TenantContext, configName, dsType string) {
	t.Helper()

	descriptor := engine.ConnectionDescriptor{
		ConfigName:   configName,
		Type:         dsType,
		Host:         "db.internal",
		Port:         5432,
		DatabaseName: "ledger",
		Username:     "svc",
		SSLMode:      "require",
	}

	if err := store.Create(context.Background(), tenant, descriptor, nil); err != nil {
		t.Fatalf("seed Create: unexpected error: %v", err)
	}
}

func TestEngine_TestConnection_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &testConnRecord{}
	factory := &testConnFactory{record: record}
	eng, store := engineForTestConnection(t, factory)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	result, err := eng.TestConnection(ctx, tenant, "pg-main")
	if err != nil {
		t.Fatalf("TestConnection: unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("TestConnection: expected Success=true, got %+v", result)
	}
	if result.ConfigName != "pg-main" {
		t.Fatalf("TestConnection: ConfigName = %q, want %q", result.ConfigName, "pg-main")
	}

	// The connector lifecycle must be build -> test -> close IN THAT ORDER, proving
	// construction and connectivity are separate and the connector is closed AFTER
	// the connectivity check (a close-before-test regression must fail here).
	wantSeq := []string{"build", "test", "close"}
	gotSeq := record.snapshot()
	if len(gotSeq) != len(wantSeq) {
		t.Fatalf("TestConnection: lifecycle = %v, want %v", gotSeq, wantSeq)
	}
	for i, step := range wantSeq {
		if gotSeq[i] != step {
			t.Fatalf("TestConnection: lifecycle[%d] = %q, want %q (full sequence %v)", i, gotSeq[i], step, gotSeq)
		}
	}

	// The factory must receive the SECRET-FREE descriptor.
	if factory.descrSeen.ConfigName != "pg-main" || factory.descrSeen.Type != "postgres" {
		t.Fatalf("TestConnection: factory descriptor not propagated: %#v", factory.descrSeen)
	}
}

func TestEngine_TestConnection_UnknownConnection_FailsBeforeConnectorBuild(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &testConnRecord{}
	factory := &testConnFactory{record: record}
	eng, _ := engineForTestConnection(t, factory)
	tenant := mustTenant(t, "tenant-a")

	// No connection seeded: the lookup must fail BEFORE any connector construction.
	_, err := eng.TestConnection(ctx, tenant, "does-not-exist")
	if err == nil {
		t.Fatalf("TestConnection: expected not-found error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryNotFound {
		t.Fatalf("TestConnection: expected CategoryNotFound, got %v", err)
	}

	if len(record.snapshot()) != 0 {
		t.Fatalf("TestConnection: connector must NOT be built for an unknown connection, got %v", record.snapshot())
	}
}

func TestEngine_TestConnection_WrongTenant_FailsBeforeConnectorBuild(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &testConnRecord{}
	factory := &testConnFactory{record: record}
	eng, store := engineForTestConnection(t, factory)

	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	// Connection belongs to tenant A.
	seedConnection(t, store, tenantA, "pg-main", "postgres")

	// Tenant B must NOT see tenant A's connection: it fails as not-found BEFORE
	// any connector construction. Tenant scope is the sole isolation boundary.
	_, err := eng.TestConnection(ctx, tenantB, "pg-main")
	if err == nil {
		t.Fatalf("TestConnection: tenant B must not test tenant A's connection")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryNotFound {
		t.Fatalf("TestConnection: expected CategoryNotFound for wrong tenant, got %v", err)
	}

	if len(record.snapshot()) != 0 {
		t.Fatalf("TestConnection: connector must NOT be built for a wrong-tenant connection, got %v", record.snapshot())
	}
}

func TestEngine_TestConnection_InvalidTenantScope_FailsBeforeStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &testConnRecord{}
	factory := &testConnFactory{record: record}
	eng, _ := engineForTestConnection(t, factory)

	// A zero-value (unscoped) tenant is rejected by the tenant-scope guard BEFORE
	// any resource access.
	_, err := eng.TestConnection(ctx, engine.TenantContext{}, "pg-main")
	if err == nil {
		t.Fatalf("TestConnection: expected validation error for unscoped tenant")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("TestConnection: expected CategoryValidation, got %v", err)
	}

	if len(record.snapshot()) != 0 {
		t.Fatalf("TestConnection: no connector access for an unscoped tenant, got %v", record.snapshot())
	}
}

func TestEngine_TestConnection_ConnectivityFailure_IsSafeAndStillCloses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &testConnRecord{}
	// The connector's explicit connectivity check fails with a secret-bearing
	// driver error; the Engine must map it to a safe error and STILL close.
	factory := &testConnFactory{
		record:  record,
		testErr: errors.New("dial tcp 10.0.0.1:5432: password=s3cr3t auth failed for user svc"),
	}
	eng, store := engineForTestConnection(t, factory)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	result, err := eng.TestConnection(ctx, tenant, "pg-main")
	if err == nil {
		t.Fatalf("TestConnection: expected connectivity error, got result %+v", result)
	}
	if result.Success {
		t.Fatalf("TestConnection: expected Success=false on connectivity failure, got %+v", result)
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("TestConnection: expected CategoryUnavailable, got %v", err)
	}

	// The error MUST NOT leak secret/DSN/driver-internal material.
	for _, leak := range []string{"s3cr3t", "10.0.0.1", "password=", "dial tcp", "svc"} {
		if strings.Contains(engErr.Error(), leak) {
			t.Fatalf("TestConnection: error leaked %q: %q", leak, engErr.Error())
		}
	}

	// Close must be attempted even on the failure path.
	if !record.has("close") {
		t.Fatalf("TestConnection: connector must be closed on failure, got %v", record.snapshot())
	}
}

func TestEngine_TestConnection_BuildFailure_IsSafeAndStillCloses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &testConnRecord{}
	factory := &testConnFactory{
		record:   record,
		buildErr: errors.New("malformed dsn host=secret-host password=p@ss"),
	}
	eng, store := engineForTestConnection(t, factory)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	_, err := eng.TestConnection(ctx, tenant, "pg-main")
	if err == nil {
		t.Fatalf("TestConnection: expected build error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("TestConnection: expected *EngineError, got %v", err)
	}

	for _, leak := range []string{"secret-host", "p@ss", "password=", "dsn"} {
		if strings.Contains(engErr.Error(), leak) {
			t.Fatalf("TestConnection: build error leaked %q: %q", leak, engErr.Error())
		}
	}

	// Build failed before a connector value existed, so no test/close are expected;
	// but the build step must have been attempted.
	if !record.has("build") {
		t.Fatalf("TestConnection: expected build attempt, got %v", record.snapshot())
	}
	if record.has("test") {
		t.Fatalf("TestConnection: must not invoke test after a failed build, got %v", record.snapshot())
	}
}

// nilBuildFactory is a buggy host factory whose Build returns (nil, nil): no
// error, but no connector either. The Engine must treat this as a build failure
// and never dereference the nil connector for TestConnection/Close.
type nilBuildFactory struct {
	record *testConnRecord
}

func (f *nilBuildFactory) Build(_ context.Context, _ engine.ConnectionDescriptor) (engine.Connector, error) {
	f.record.note("build")
	return nil, nil //nolint:nilnil // deliberately models a buggy host Build for the nil guard test.
}

var _ engine.ConnectorFactory = (*nilBuildFactory)(nil)

func TestEngine_TestConnection_TypedNilFactory_StableErrorNoPanic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()

	// Host wiring bug: a TYPED-NIL ConnectorFactory is registered for the type.
	// The registry returns (factory, ok=true) where factory is a non-nil interface
	// wrapping a nil *testConnFactory. Without a guard, factory.Build panics.
	var typedNil *testConnFactory
	registry.Register("postgres", typedNil)

	eng, err := engine.New(
		engine.WithConnectorRegistry(registry),
		engine.WithConnectionStore(store),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	_, err = eng.TestConnection(ctx, tenant, "pg-main")
	if err == nil {
		t.Fatalf("TestConnection: expected stable error for typed-nil factory, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryNotFound {
		t.Fatalf("TestConnection: expected CategoryNotFound for typed-nil factory, got %v", err)
	}

	// It must be the SAME stable error an unregistered type produces.
	want := engine.UnknownConnectorTypeError("postgres").Error()
	if engErr.Error() != want {
		t.Fatalf("TestConnection: typed-nil factory error = %q, want stable %q", engErr.Error(), want)
	}
}

func TestEngine_TestConnection_BuildReturnsNilConnector_StableErrorNoPanic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &testConnRecord{}
	// Build returns (nil, nil): the Engine must treat the nil connector as a build
	// failure BEFORE registering the deferred Close, so neither TestConnection nor
	// Close is invoked on a nil interface.
	factory := &nilBuildFactory{record: record}
	eng, store := engineForTestConnection(t, factory)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	_, err := eng.TestConnection(ctx, tenant, "pg-main")
	if err == nil {
		t.Fatalf("TestConnection: expected build-failure error for nil connector, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("TestConnection: expected CategoryUnavailable build failure, got %v", err)
	}

	// Build was attempted, but no test/close may run against a nil connector.
	if !record.has("build") {
		t.Fatalf("TestConnection: expected build attempt, got %v", record.snapshot())
	}
	if record.has("test") || record.has("close") {
		t.Fatalf("TestConnection: must not test/close a nil connector, got %v", record.snapshot())
	}
}

func TestEngine_TestConnection_UnknownDatasourceType_StableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &testConnRecord{}
	// Factory registered under "postgres" only; seed a connection of an
	// unregistered type so the registry lookup yields the stable unknown-type error.
	factory := &testConnFactory{record: record}
	eng, store := engineForTestConnection(t, factory)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "mystery-db", "cassandra")

	_, err := eng.TestConnection(ctx, tenant, "mystery-db")
	if err == nil {
		t.Fatalf("TestConnection: expected unknown-type error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryNotFound {
		t.Fatalf("TestConnection: expected CategoryNotFound for unknown type, got %v", err)
	}

	want := engine.UnknownConnectorTypeError("cassandra").Error()
	if engErr.Error() != want {
		t.Fatalf("TestConnection: unknown-type error = %q, want stable %q", engErr.Error(), want)
	}

	if len(record.snapshot()) != 0 {
		t.Fatalf("TestConnection: no connector access for an unknown type, got %v", record.snapshot())
	}
}

func TestEngine_TestConnection_StoreError_Propagates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// A store whose read path fails must surface its error from TestConnection
	// without ever reaching connector construction. failingStore is defined in
	// connection_ops_test.go (same test package).
	eng, err := engine.New(
		engine.WithConnectorRegistry(memory.NewConnectorRegistry()),
		engine.WithConnectionStore(failingStore{}),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	_, err = eng.TestConnection(ctx, mustTenant(t, "tenant-a"), "pg-main")
	if err == nil {
		t.Fatalf("TestConnection: expected store error to propagate")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("TestConnection: expected CategoryUnavailable store error, got %v", err)
	}
}

func TestEngine_TestConnection_NoConnectionStore_StableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := engine.New(engine.WithConnectorRegistry(memory.NewConnectorRegistry()))
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	_, err = eng.TestConnection(ctx, mustTenant(t, "tenant-a"), "pg-main")
	if err == nil {
		t.Fatalf("TestConnection: expected validation error when no store configured")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("TestConnection: expected CategoryValidation, got %v", err)
	}
}
