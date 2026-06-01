// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
)

// schemaConnRecord records connector lifecycle calls for the DiscoverSchema
// tests so a test can prove whether the Engine drove a live discovery
// (build -> discover -> close) or short-circuited on a cache hit (no calls).
type schemaConnRecord struct {
	mu    sync.Mutex
	calls []string
}

func (r *schemaConnRecord) note(step string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, step)
}

func (r *schemaConnRecord) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.calls...)
}

func (r *schemaConnRecord) has(step string) bool {
	for _, c := range r.snapshot() {
		if c == step {
			return true
		}
	}

	return false
}

// schemaConnConnector is a host-side connector double for the DiscoverSchema
// tests. discoverErr makes DiscoverSchema fail; snapshot is the schema returned
// on success. It records every lifecycle call.
type schemaConnConnector struct {
	record      *schemaConnRecord
	discoverErr error
	snapshot    engine.SchemaSnapshot
	closeErr    error
}

func (c *schemaConnConnector) TestConnection(_ context.Context) error {
	c.record.note("test")
	return nil
}

func (c *schemaConnConnector) DiscoverSchema(_ context.Context) (engine.SchemaSnapshot, error) {
	c.record.note("discover")
	if c.discoverErr != nil {
		return engine.SchemaSnapshot{}, c.discoverErr
	}

	return c.snapshot, nil
}

func (c *schemaConnConnector) Query(_ context.Context, _ engine.ExtractionRequest) (map[string][]map[string]any, error) {
	c.record.note("query")
	return nil, nil
}

func (c *schemaConnConnector) Close(_ context.Context) error {
	c.record.note("close")
	return c.closeErr
}

// schemaConnFactory builds schemaConnConnectors. Build is I/O-free and records
// the descriptor it received so a test can prove the SECRET-FREE descriptor is
// what reaches connector construction.
type schemaConnFactory struct {
	record      *schemaConnRecord
	buildErr    error
	discoverErr error
	snapshot    engine.SchemaSnapshot
	descrSeen   engine.ConnectionDescriptor
}

func (f *schemaConnFactory) Build(_ context.Context, descriptor engine.ConnectionDescriptor) (engine.Connector, error) {
	f.record.note("build")
	f.descrSeen = descriptor

	if f.buildErr != nil {
		return nil, f.buildErr
	}

	return &schemaConnConnector{record: f.record, discoverErr: f.discoverErr, snapshot: f.snapshot}, nil
}

var (
	_ engine.Connector        = (*schemaConnConnector)(nil)
	_ engine.ConnectorFactory = (*schemaConnFactory)(nil)
)

// engineForDiscoverSchema wires an Engine with the in-memory store, a registry
// holding the supplied factory under "postgres", and an OPTIONAL schema cache.
// A nil cache exercises the no-cache path (discover every time).
func engineForDiscoverSchema(
	t *testing.T,
	factory engine.ConnectorFactory,
	cache engine.SchemaCache,
) (*engine.Engine, *memory.ConnectionStore) {
	t.Helper()

	store := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()

	if factory != nil {
		registry.Register("postgres", factory)
	}

	opts := []engine.Option{
		engine.WithConnectorRegistry(registry),
		engine.WithConnectionStore(store),
	}
	if cache != nil {
		opts = append(opts, engine.WithSchemaCache(cache))
	}

	eng, err := engine.New(opts...)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	return eng, store
}

func sampleSnapshot(configName string) engine.SchemaSnapshot {
	return engine.SchemaSnapshot{
		ConfigName: configName,
		Tables: []engine.TableSnapshot{
			{Name: "public.orders", Fields: []string{"id", "amount"}},
			{Name: "public.customers", Fields: []string{"id", "name"}},
		},
		CapturedAt: time.Unix(1700000000, 0).UTC(),
	}
}

func TestEngine_DiscoverSchema_CacheMiss_DiscoversAndWritesThrough(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: sampleSnapshot("pg-main")}
	cache := memory.NewSchemaCache()
	eng, store := engineForDiscoverSchema(t, factory, cache)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	got, err := eng.DiscoverSchema(ctx, tenant, "pg-main")
	if err != nil {
		t.Fatalf("DiscoverSchema: unexpected error: %v", err)
	}

	if got.ConfigName != "pg-main" {
		t.Fatalf("DiscoverSchema: ConfigName = %q, want %q", got.ConfigName, "pg-main")
	}
	if len(got.Tables) != 2 {
		t.Fatalf("DiscoverSchema: expected 2 tables, got %d (%+v)", len(got.Tables), got.Tables)
	}

	// Cache miss must drive a live discovery through the connector contract.
	wantSeq := []string{"build", "discover", "close"}
	gotSeq := record.snapshot()
	if len(gotSeq) != len(wantSeq) {
		t.Fatalf("DiscoverSchema: lifecycle = %v, want %v", gotSeq, wantSeq)
	}
	for i, step := range wantSeq {
		if gotSeq[i] != step {
			t.Fatalf("DiscoverSchema: lifecycle[%d] = %q, want %q (%v)", i, gotSeq[i], step, gotSeq)
		}
	}

	// Write-through: the snapshot must now be cached under the tenant scope.
	cached, ok, cacheErr := cache.GetSchema(ctx, tenant, "pg-main")
	if cacheErr != nil {
		t.Fatalf("cache.GetSchema: unexpected error: %v", cacheErr)
	}
	if !ok {
		t.Fatalf("DiscoverSchema: expected snapshot written through to cache, got miss")
	}
	if cached.ConfigName != "pg-main" || len(cached.Tables) != 2 {
		t.Fatalf("DiscoverSchema: cached snapshot mismatch: %+v", cached)
	}

	// The factory must receive the SECRET-FREE descriptor.
	if factory.descrSeen.ConfigName != "pg-main" || factory.descrSeen.Type != "postgres" {
		t.Fatalf("DiscoverSchema: factory descriptor not propagated: %#v", factory.descrSeen)
	}
}

func TestEngine_DiscoverSchema_CacheHit_ReturnsWithoutDiscovery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: sampleSnapshot("pg-main")}
	cache := memory.NewSchemaCache()
	eng, store := engineForDiscoverSchema(t, factory, cache)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	// Pre-warm the cache for the tenant scope.
	preWarm := engine.SchemaSnapshot{
		ConfigName: "pg-main",
		Tables:     []engine.TableSnapshot{{Name: "public.cached_only", Fields: []string{"x"}}},
	}
	if err := cache.PutSchema(ctx, tenant, preWarm); err != nil {
		t.Fatalf("cache.PutSchema: unexpected error: %v", err)
	}

	got, err := eng.DiscoverSchema(ctx, tenant, "pg-main")
	if err != nil {
		t.Fatalf("DiscoverSchema: unexpected error: %v", err)
	}

	if !got.HasTable("public.cached_only") {
		t.Fatalf("DiscoverSchema: expected cached snapshot, got %+v", got)
	}

	// A cache hit MUST NOT touch the connector at all.
	if len(record.snapshot()) != 0 {
		t.Fatalf("DiscoverSchema: connector must NOT be built on a cache hit, got %v", record.snapshot())
	}
}

func TestEngine_DiscoverSchema_NoCache_DiscoversEveryTime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: sampleSnapshot("pg-main")}
	// No cache configured (nil optional port): must discover, no panic.
	eng, store := engineForDiscoverSchema(t, factory, nil)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	if _, err := eng.DiscoverSchema(ctx, tenant, "pg-main"); err != nil {
		t.Fatalf("DiscoverSchema #1: unexpected error: %v", err)
	}
	if _, err := eng.DiscoverSchema(ctx, tenant, "pg-main"); err != nil {
		t.Fatalf("DiscoverSchema #2: unexpected error: %v", err)
	}

	// Each call discovers afresh: two build+discover cycles.
	discoverCount := 0
	for _, c := range record.snapshot() {
		if c == "discover" {
			discoverCount++
		}
	}
	if discoverCount != 2 {
		t.Fatalf("DiscoverSchema: expected 2 discoveries with no cache, got %d (%v)", discoverCount, record.snapshot())
	}
}

func TestEngine_DiscoverSchema_UnknownConnection_ScopedNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: sampleSnapshot("pg-main")}
	eng, _ := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")

	_, err := eng.DiscoverSchema(ctx, tenant, "does-not-exist")
	if err == nil {
		t.Fatalf("DiscoverSchema: expected not-found error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryNotFound {
		t.Fatalf("DiscoverSchema: expected CategoryNotFound, got %v", err)
	}

	if len(record.snapshot()) != 0 {
		t.Fatalf("DiscoverSchema: connector must NOT be built for an unknown connection, got %v", record.snapshot())
	}
}

func TestEngine_DiscoverSchema_TenantIsolation_NotDiscoverableAcrossTenants(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: sampleSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())

	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	// Connection belongs to tenant A only.
	seedConnection(t, store, tenantA, "pg-main", "postgres")

	// Tenant B must NOT discover tenant A's connection: not-found, no connector.
	_, err := eng.DiscoverSchema(ctx, tenantB, "pg-main")
	if err == nil {
		t.Fatalf("DiscoverSchema: tenant B must not discover tenant A's connection")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryNotFound {
		t.Fatalf("DiscoverSchema: expected CategoryNotFound for wrong tenant, got %v", err)
	}

	if len(record.snapshot()) != 0 {
		t.Fatalf("DiscoverSchema: connector must NOT be built for a wrong-tenant connection, got %v", record.snapshot())
	}
}

func TestEngine_DiscoverSchema_CacheIsTenantScoped_NoCrossTenantBleed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: sampleSnapshot("pg-main")}
	cache := memory.NewSchemaCache()
	eng, store := engineForDiscoverSchema(t, factory, cache)

	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	// Same config name under BOTH tenants.
	seedConnection(t, store, tenantA, "pg-main", "postgres")
	seedConnection(t, store, tenantB, "pg-main", "postgres")

	// Tenant A discovers and write-throughs to cache.
	if _, err := eng.DiscoverSchema(ctx, tenantA, "pg-main"); err != nil {
		t.Fatalf("DiscoverSchema(tenantA): unexpected error: %v", err)
	}

	// Tenant B's cache lookup for the same config name MUST miss (tenant-scoped key),
	// forcing a fresh discovery rather than serving tenant A's cached schema.
	beforeB := 0
	for _, c := range record.snapshot() {
		if c == "discover" {
			beforeB++
		}
	}

	if _, err := eng.DiscoverSchema(ctx, tenantB, "pg-main"); err != nil {
		t.Fatalf("DiscoverSchema(tenantB): unexpected error: %v", err)
	}

	afterB := 0
	for _, c := range record.snapshot() {
		if c == "discover" {
			afterB++
		}
	}

	if afterB != beforeB+1 {
		t.Fatalf("DiscoverSchema: tenant B must not hit tenant A's cache; expected a fresh discovery, discovers before=%d after=%d", beforeB, afterB)
	}
}

func TestEngine_DiscoverSchema_DiscoveryFailure_IsSafeAndStillCloses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	// The connector's DiscoverSchema fails with a secret-bearing driver error; the
	// Engine must map it to a safe error, leak nothing, and STILL close.
	factory := &schemaConnFactory{
		record:      record,
		discoverErr: errors.New("query information_schema failed: password=s3cr3t host=10.0.0.1 user=svc"),
	}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	_, err := eng.DiscoverSchema(ctx, tenant, "pg-main")
	if err == nil {
		t.Fatalf("DiscoverSchema: expected discovery error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("DiscoverSchema: expected CategoryUnavailable, got %v", err)
	}

	for _, leak := range []string{"s3cr3t", "10.0.0.1", "password=", "information_schema", "svc"} {
		if strings.Contains(engErr.Error(), leak) {
			t.Fatalf("DiscoverSchema: error leaked %q: %q", leak, engErr.Error())
		}
	}

	// Close must be attempted even on the discovery-failure path.
	if !record.has("close") {
		t.Fatalf("DiscoverSchema: connector must be closed on failure, got %v", record.snapshot())
	}
}

func TestEngine_DiscoverSchema_BuildFailure_IsSafe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{
		record:   record,
		buildErr: errors.New("malformed dsn host=secret-host password=p@ss"),
	}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	_, err := eng.DiscoverSchema(ctx, tenant, "pg-main")
	if err == nil {
		t.Fatalf("DiscoverSchema: expected build error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("DiscoverSchema: expected CategoryUnavailable build failure, got %v", err)
	}

	for _, leak := range []string{"secret-host", "p@ss", "password=", "dsn"} {
		if strings.Contains(engErr.Error(), leak) {
			t.Fatalf("DiscoverSchema: build error leaked %q: %q", leak, engErr.Error())
		}
	}

	if record.has("discover") {
		t.Fatalf("DiscoverSchema: must not discover after a failed build, got %v", record.snapshot())
	}
}

func TestEngine_DiscoverSchema_UnknownDatasourceType_StableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: sampleSnapshot("mystery-db")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "mystery-db", "cassandra")

	_, err := eng.DiscoverSchema(ctx, tenant, "mystery-db")
	if err == nil {
		t.Fatalf("DiscoverSchema: expected unknown-type error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryNotFound {
		t.Fatalf("DiscoverSchema: expected CategoryNotFound for unknown type, got %v", err)
	}

	want := engine.UnknownConnectorTypeError("cassandra").Error()
	if engErr.Error() != want {
		t.Fatalf("DiscoverSchema: unknown-type error = %q, want stable %q", engErr.Error(), want)
	}

	if len(record.snapshot()) != 0 {
		t.Fatalf("DiscoverSchema: no connector access for an unknown type, got %v", record.snapshot())
	}
}

func TestEngine_DiscoverSchema_InvalidTenantScope_FailsBeforeStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record}
	eng, _ := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())

	_, err := eng.DiscoverSchema(ctx, engine.TenantContext{}, "pg-main")
	if err == nil {
		t.Fatalf("DiscoverSchema: expected validation error for unscoped tenant")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("DiscoverSchema: expected CategoryValidation, got %v", err)
	}

	if len(record.snapshot()) != 0 {
		t.Fatalf("DiscoverSchema: no connector access for an unscoped tenant, got %v", record.snapshot())
	}
}

func TestEngine_DiscoverSchema_NoConnectionStore_StableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := engine.New(engine.WithConnectorRegistry(memory.NewConnectorRegistry()))
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	_, err = eng.DiscoverSchema(ctx, mustTenant(t, "tenant-a"), "pg-main")
	if err == nil {
		t.Fatalf("DiscoverSchema: expected validation error when no store configured")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("DiscoverSchema: expected CategoryValidation, got %v", err)
	}
}

func TestEngine_DiscoverSchema_CacheReadError_FallsBackToDiscovery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: sampleSnapshot("pg-main")}
	// A cache whose read path fails must NOT break discovery: the cache is an
	// optimization, so a read failure degrades to a fresh discovery.
	cache := memory.NewSchemaCache()
	cache.GetErr = errors.New("redis: connection refused")
	eng, store := engineForDiscoverSchema(t, factory, cache)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	got, err := eng.DiscoverSchema(ctx, tenant, "pg-main")
	if err != nil {
		t.Fatalf("DiscoverSchema: cache read failure must not fail discovery, got %v", err)
	}
	if got.ConfigName != "pg-main" {
		t.Fatalf("DiscoverSchema: expected discovered schema, got %+v", got)
	}
	if !record.has("discover") {
		t.Fatalf("DiscoverSchema: expected fresh discovery after cache read error, got %v", record.snapshot())
	}
}

func TestEngine_DiscoverSchema_CacheWriteError_StillReturnsSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: sampleSnapshot("pg-main")}
	// A cache whose write path fails must NOT fail the operation: discovery already
	// succeeded, so a failed write-through is non-fatal.
	cache := memory.NewSchemaCache()
	cache.PutErr = errors.New("redis: write timeout")
	eng, store := engineForDiscoverSchema(t, factory, cache)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	got, err := eng.DiscoverSchema(ctx, tenant, "pg-main")
	if err != nil {
		t.Fatalf("DiscoverSchema: cache write failure must not fail discovery, got %v", err)
	}
	if got.ConfigName != "pg-main" || len(got.Tables) != 2 {
		t.Fatalf("DiscoverSchema: expected discovered schema despite cache write error, got %+v", got)
	}
}
