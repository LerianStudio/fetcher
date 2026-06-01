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
	// nilConnector, when true, makes Build return (nil, nil): a buggy host that
	// reports success but yields no connector. The Engine must treat this as a
	// build failure and never dereference (or Close) the nil connector.
	nilConnector bool
}

func (f *schemaConnFactory) Build(_ context.Context, descriptor engine.ConnectionDescriptor) (engine.Connector, error) {
	f.record.note("build")
	f.descrSeen = descriptor

	if f.buildErr != nil {
		return nil, f.buildErr
	}

	if f.nilConnector {
		return nil, nil
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

func TestEngine_DiscoverSchema_NilConnectorBuild_IsSafeAndDoesNotClose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	// A buggy host Build returns (nil, nil): success with no connector. The Engine
	// must treat it as a build failure (CategoryUnavailable) and must NOT attempt
	// to discover or close a nil connector.
	factory := &schemaConnFactory{record: record, nilConnector: true}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	_, err := eng.DiscoverSchema(ctx, tenant, "pg-main")
	if err == nil {
		t.Fatalf("DiscoverSchema: expected build failure for nil connector, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("DiscoverSchema: expected CategoryUnavailable for nil connector, got %v", err)
	}

	if record.has("discover") {
		t.Fatalf("DiscoverSchema: must not discover with a nil connector, got %v", record.snapshot())
	}
	if record.has("close") {
		t.Fatalf("DiscoverSchema: must not close a nil connector, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_NilConnectorBuild_IsSafeAndDoesNotClose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, nilConnector: true}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	_, err := eng.ValidateSchema(ctx, tenant, validRequest())
	if err == nil {
		t.Fatalf("ValidateSchema: expected build failure for nil connector, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("ValidateSchema: expected CategoryUnavailable for nil connector, got %v", err)
	}

	if record.has("discover") || record.has("close") {
		t.Fatalf("ValidateSchema: must not discover/close a nil connector, got %v", record.snapshot())
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

// ---------------------------------------------------------------------------
// ValidateSchema (ST-T005-02)
//
// ValidateSchema validates a mapping (datasources -> tables -> fields, plus
// filter field references) against the schema snapshot for each persisted
// connection within the tenant scope. It reuses the DiscoverSchema resolve ->
// cache -> discover flow: a cache hit validates WITHOUT connector discovery; a
// cache miss discovers live and validates from the discovered snapshot.
//
// Distinctions the report must preserve:
//   - missing datasource / missing table / missing field / invalid filter are
//     each a DISTINCT, typed validation failure carried in the report;
//   - a limit breach (datasource/table/field/filter count) is a DISTINCT
//     validation failure, not a generic failure;
//   - a connector/source-down failure is a SEPARATE Engine error (the
//     CategoryUnavailable family), NOT a report entry — it is not the caller's
//     malformed request;
//   - no error or report field leaks credentials or extracted data.
// ---------------------------------------------------------------------------

// validationSnapshot builds a snapshot with explicit tables and fields for the
// ValidateSchema tests.
func validationSnapshot(configName string) engine.SchemaSnapshot {
	return engine.SchemaSnapshot{
		ConfigName: configName,
		Tables: []engine.TableSnapshot{
			{Name: "public.orders", Fields: []string{"id", "amount", "created_at"}},
			{Name: "public.customers", Fields: []string{"id", "name", "email"}},
		},
		CapturedAt: time.Unix(1700000000, 0).UTC(),
	}
}

// mappedRequest builds a SchemaValidationRequest mapping one datasource to a set
// of tables/fields, with optional filter field references per table.
func validRequest() engine.SchemaValidationRequest {
	return engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables: []engine.TableMapping{
					{
						Name:         "public.orders",
						Fields:       []string{"id", "amount"},
						FilterFields: []string{"created_at"},
					},
					{
						Name:   "public.customers",
						Fields: []string{"id", "name"},
					},
				},
			},
		},
	}
}

func TestEngine_ValidateSchema_ValidMapping_SuccessReport(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	report, err := eng.ValidateSchema(ctx, tenant, validRequest())
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}

	if !report.Valid {
		t.Fatalf("ValidateSchema: expected valid report, got failures: %+v", report.Failures)
	}
	if len(report.Failures) != 0 {
		t.Fatalf("ValidateSchema: expected zero failures, got %+v", report.Failures)
	}
}

func TestEngine_ValidateSchema_CacheHit_ValidatesWithoutDiscovery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	cache := memory.NewSchemaCache()
	eng, store := engineForDiscoverSchema(t, factory, cache)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	// Pre-warm the cache for the tenant scope so validation runs against the
	// cached snapshot and never touches the connector.
	if err := cache.PutSchema(ctx, tenant, validationSnapshot("pg-main")); err != nil {
		t.Fatalf("cache.PutSchema: unexpected error: %v", err)
	}

	report, err := eng.ValidateSchema(ctx, tenant, validRequest())
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if !report.Valid {
		t.Fatalf("ValidateSchema: expected valid report from cache, got %+v", report.Failures)
	}

	// A cache hit MUST NOT build or query the connector.
	if len(record.snapshot()) != 0 {
		t.Fatalf("ValidateSchema: connector must NOT be built on a cache hit, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_CacheMiss_DiscoversThenValidates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	report, err := eng.ValidateSchema(ctx, tenant, validRequest())
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if !report.Valid {
		t.Fatalf("ValidateSchema: expected valid report, got %+v", report.Failures)
	}

	// A cache miss must drive a live discovery through the connector contract.
	if !record.has("discover") {
		t.Fatalf("ValidateSchema: expected live discovery on cache miss, got %v", record.snapshot())
	}
	if !record.has("close") {
		t.Fatalf("ValidateSchema: connector must be closed after discovery, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_MissingDatasource_DistinctFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, _ := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	// No connection seeded: the mapped datasource does not exist for the tenant.

	report, err := eng.ValidateSchema(ctx, tenant, validRequest())
	if err != nil {
		t.Fatalf("ValidateSchema: missing datasource must be a report failure, not an Engine error: %v", err)
	}
	if report.Valid {
		t.Fatalf("ValidateSchema: expected invalid report for missing datasource")
	}

	if !hasFailure(report, engine.ValidationDatasourceNotFound, "pg-main", "", "") {
		t.Fatalf("ValidateSchema: expected datasource-not-found failure, got %+v", report.Failures)
	}

	// A missing datasource must not reach the connector.
	if len(record.snapshot()) != 0 {
		t.Fatalf("ValidateSchema: connector must NOT be built for a missing datasource, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_MissingTable_DistinctFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables: []engine.TableMapping{
					{Name: "public.ghost", Fields: []string{"id"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: missing table must be a report failure, not an error: %v", err)
	}
	if report.Valid {
		t.Fatalf("ValidateSchema: expected invalid report for missing table")
	}
	if !hasFailure(report, engine.ValidationTableNotFound, "pg-main", "public.ghost", "") {
		t.Fatalf("ValidateSchema: expected table-not-found failure, got %+v", report.Failures)
	}
}

func TestEngine_ValidateSchema_MissingField_DistinctFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables: []engine.TableMapping{
					{Name: "public.orders", Fields: []string{"id", "nonexistent"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: missing field must be a report failure, not an error: %v", err)
	}
	if report.Valid {
		t.Fatalf("ValidateSchema: expected invalid report for missing field")
	}
	if !hasFailure(report, engine.ValidationFieldNotFound, "pg-main", "public.orders", "nonexistent") {
		t.Fatalf("ValidateSchema: expected field-not-found failure, got %+v", report.Failures)
	}
}

func TestEngine_ValidateSchema_InvalidFilterField_DistinctFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables: []engine.TableMapping{
					{
						Name:         "public.orders",
						Fields:       []string{"id"},
						FilterFields: []string{"no_such_filter_col"},
					},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: invalid filter must be a report failure, not an error: %v", err)
	}
	if report.Valid {
		t.Fatalf("ValidateSchema: expected invalid report for invalid filter field")
	}
	if !hasFailure(report, engine.ValidationInvalidFilter, "pg-main", "public.orders", "no_such_filter_col") {
		t.Fatalf("ValidateSchema: expected invalid-filter failure, got %+v", report.Failures)
	}
}

func TestEngine_ValidateSchema_SourceDown_SeparateFromMalformedRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	// The connector's DiscoverSchema fails with a secret-bearing driver error.
	// Source-down must surface as a SEPARATE Engine error (CategoryUnavailable),
	// NOT a validation report failure, and must leak nothing.
	factory := &schemaConnFactory{
		record:      record,
		discoverErr: errors.New("dial tcp 10.0.0.1:5432: password=s3cr3t user=svc connection refused"),
	}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	_, err := eng.ValidateSchema(ctx, tenant, validRequest())
	if err == nil {
		t.Fatalf("ValidateSchema: source-down must surface as an Engine error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("ValidateSchema: expected CategoryUnavailable for source-down, got %v", err)
	}

	for _, leak := range []string{"s3cr3t", "10.0.0.1", "password=", "svc", "connection refused"} {
		if strings.Contains(engErr.Error(), leak) {
			t.Fatalf("ValidateSchema: source-down error leaked %q: %q", leak, engErr.Error())
		}
	}

	// Even on a source-down failure, the connector must be closed.
	if !record.has("close") {
		t.Fatalf("ValidateSchema: connector must be closed on source-down, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_LimitViolation_DatasourceCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}

	// Tighten the datasource limit so two mapped datasources breach it.
	limits := engine.DefaultLimits()
	limits.MaxDatasources = 1
	eng, store := engineForDiscoverSchemaWithLimits(t, factory, memory.NewSchemaCache(), limits)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")
	seedConnection(t, store, tenant, "pg-second", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{ConfigName: "pg-main", Tables: []engine.TableMapping{{Name: "public.orders", Fields: []string{"id"}}}},
			{ConfigName: "pg-second", Tables: []engine.TableMapping{{Name: "public.orders", Fields: []string{"id"}}}},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: limit breach must be a report failure, not an error: %v", err)
	}
	if report.Valid {
		t.Fatalf("ValidateSchema: expected invalid report for datasource-count limit breach")
	}
	if !hasFailureType(report, engine.ValidationLimitExceeded) {
		t.Fatalf("ValidateSchema: expected limit-exceeded failure, got %+v", report.Failures)
	}

	// A limit breach is rejected BEFORE any connector access.
	if len(record.snapshot()) != 0 {
		t.Fatalf("ValidateSchema: connector must NOT be built on a limit breach, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_LimitViolation_TableCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}

	limits := engine.DefaultLimits()
	limits.MaxTablesPerDatasource = 1
	eng, store := engineForDiscoverSchemaWithLimits(t, factory, memory.NewSchemaCache(), limits)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables: []engine.TableMapping{
					{Name: "public.orders", Fields: []string{"id"}},
					{Name: "public.customers", Fields: []string{"id"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: table-count limit breach must be a report failure: %v", err)
	}
	if report.Valid || !hasFailureType(report, engine.ValidationLimitExceeded) {
		t.Fatalf("ValidateSchema: expected limit-exceeded failure, got %+v", report.Failures)
	}
	if len(record.snapshot()) != 0 {
		t.Fatalf("ValidateSchema: connector must NOT be built on a table-count limit breach, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_LimitViolation_FieldCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}

	limits := engine.DefaultLimits()
	limits.MaxFieldsPerTable = 1
	eng, store := engineForDiscoverSchemaWithLimits(t, factory, memory.NewSchemaCache(), limits)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables: []engine.TableMapping{
					{Name: "public.orders", Fields: []string{"id", "amount"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: field-count limit breach must be a report failure: %v", err)
	}
	if report.Valid || !hasFailureType(report, engine.ValidationLimitExceeded) {
		t.Fatalf("ValidateSchema: expected limit-exceeded failure, got %+v", report.Failures)
	}
	// The field-count breach must carry the field-count Detail, distinct from the
	// filter-count Detail, so a swap of the two strings is detectable.
	if !hasFailureDetail(report, engine.ValidationLimitExceeded, "field count") {
		t.Fatalf("ValidateSchema: expected field-count limit Detail, got %+v", report.Failures)
	}
}

func TestEngine_ValidateSchema_LimitViolation_FilterCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}

	limits := engine.DefaultLimits()
	limits.MaxFieldsPerTable = 1
	eng, store := engineForDiscoverSchemaWithLimits(t, factory, memory.NewSchemaCache(), limits)
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables: []engine.TableMapping{
					{Name: "public.orders", Fields: []string{"id"}, FilterFields: []string{"id", "amount"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: filter-count limit breach must be a report failure: %v", err)
	}
	if report.Valid || !hasFailureType(report, engine.ValidationLimitExceeded) {
		t.Fatalf("ValidateSchema: expected limit-exceeded failure, got %+v", report.Failures)
	}
	// The filter-count breach must carry the filter-count Detail, distinct from
	// the field-count Detail.
	if !hasFailureDetail(report, engine.ValidationLimitExceeded, "filter count") {
		t.Fatalf("ValidateSchema: expected filter-count limit Detail, got %+v", report.Failures)
	}
}

// ---------------------------------------------------------------------------
// FIX 1: nested / parent-object field matching (legacy HasField semantics)
// ---------------------------------------------------------------------------

// nestedSnapshot models the Mongo/plugin_crm flattening where nested objects
// become dotted field names. The parent object "natural_person" is NOT a column
// itself; only its dotted leaves exist.
func nestedSnapshot(configName string) engine.SchemaSnapshot {
	return engine.SchemaSnapshot{
		ConfigName: configName,
		Tables: []engine.TableSnapshot{
			{Name: "holders", Fields: []string{
				"id",
				"natural_person.mother_name",
				"natural_person.birth_date",
				"address.city",
			}},
		},
	}
}

func TestEngine_ValidateSchema_ParentObjectField_IsValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: nestedSnapshot("mongo-crm")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "mongo-crm", "postgres")

	// "natural_person" is a parent of "natural_person.mother_name": valid, even
	// though it is not itself a flattened column.
	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "mongo-crm",
				Tables: []engine.TableMapping{
					{Name: "holders", Fields: []string{"id", "natural_person"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if !report.Valid {
		t.Fatalf("ValidateSchema: parent-object field must be valid, got %+v", report.Failures)
	}
}

func TestEngine_ValidateSchema_ParentObjectFilterField_IsValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: nestedSnapshot("mongo-crm")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "mongo-crm", "postgres")

	// A filter referencing the parent object "address" (a parent of "address.city")
	// must validate, identically to mapped fields.
	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "mongo-crm",
				Tables: []engine.TableMapping{
					{Name: "holders", Fields: []string{"id"}, FilterFields: []string{"address"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if !report.Valid {
		t.Fatalf("ValidateSchema: parent-object filter field must be valid, got %+v", report.Failures)
	}
}

func TestEngine_ValidateSchema_PartialPrefix_IsNotAFalseMatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: nestedSnapshot("mongo-crm")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "mongo-crm", "postgres")

	// "natural" is a string prefix of "natural_person.mother_name" but NOT a parent
	// object (the delimiter is "." not "_"): it must remain field-not-found.
	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "mongo-crm",
				Tables: []engine.TableMapping{
					{Name: "holders", Fields: []string{"natural"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if report.Valid {
		t.Fatalf("ValidateSchema: 'natural' must not falsely match 'natural_person.*'")
	}
	if !hasFailure(report, engine.ValidationFieldNotFound, "mongo-crm", "holders", "natural") {
		t.Fatalf("ValidateSchema: expected field-not-found for 'natural', got %+v", report.Failures)
	}
}

func TestEngine_ValidateSchema_ExactDottedLeafField_IsValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: nestedSnapshot("mongo-crm")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "mongo-crm", "postgres")

	// The exact dotted leaf still validates via exact membership.
	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "mongo-crm",
				Tables: []engine.TableMapping{
					{Name: "holders", Fields: []string{"natural_person.mother_name"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if !report.Valid {
		t.Fatalf("ValidateSchema: exact dotted leaf must be valid, got %+v", report.Failures)
	}
}

// ---------------------------------------------------------------------------
// FIX 3: empty / whitespace names are malformed requests, not schema mismatches
// ---------------------------------------------------------------------------

func TestEngine_ValidateSchema_EmptyTableName_IsMalformedRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables:     []engine.TableMapping{{Name: "   ", Fields: []string{"id"}}},
			},
		},
	}

	_, err := eng.ValidateSchema(ctx, tenant, req)
	if err == nil {
		t.Fatalf("ValidateSchema: whitespace table name must be a malformed-request error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("ValidateSchema: expected CategoryValidation for whitespace table name, got %v", err)
	}
}

func TestEngine_ValidateSchema_EmptyFieldName_IsMalformedRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables:     []engine.TableMapping{{Name: "public.orders", Fields: []string{""}}},
			},
		},
	}

	_, err := eng.ValidateSchema(ctx, tenant, req)
	if err == nil {
		t.Fatalf("ValidateSchema: empty field name must be a malformed-request error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("ValidateSchema: expected CategoryValidation for empty field name, got %v", err)
	}
}

func TestEngine_ValidateSchema_EmptyConfigName_IsMalformedRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, _ := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{ConfigName: "  ", Tables: []engine.TableMapping{{Name: "public.orders", Fields: []string{"id"}}}},
		},
	}

	_, err := eng.ValidateSchema(ctx, tenant, req)
	if err == nil {
		t.Fatalf("ValidateSchema: whitespace config name must be a malformed-request error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("ValidateSchema: expected CategoryValidation for whitespace config name, got %v", err)
	}
	if len(record.snapshot()) != 0 {
		t.Fatalf("ValidateSchema: no connector access for a malformed config name, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_InvalidTenantScope_FailsBeforeStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record}
	eng, _ := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())

	_, err := eng.ValidateSchema(ctx, engine.TenantContext{}, validRequest())
	if err == nil {
		t.Fatalf("ValidateSchema: expected validation error for unscoped tenant")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("ValidateSchema: expected CategoryValidation, got %v", err)
	}
	if len(record.snapshot()) != 0 {
		t.Fatalf("ValidateSchema: no connector access for an unscoped tenant, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_EmptyRequest_MalformedValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, _ := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")

	_, err := eng.ValidateSchema(ctx, tenant, engine.SchemaValidationRequest{})
	if err == nil {
		t.Fatalf("ValidateSchema: expected validation error for empty request")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("ValidateSchema: expected CategoryValidation for empty request, got %v", err)
	}
	if len(record.snapshot()) != 0 {
		t.Fatalf("ValidateSchema: no connector access for an empty request, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_TenantIsolation_NotValidatableAcrossTenants(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())

	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")
	seedConnection(t, store, tenantA, "pg-main", "postgres")

	// Tenant B does not own pg-main: it must surface as a missing-datasource
	// report failure (the connection is invisible under B's scope), never tenant
	// A's schema.
	report, err := eng.ValidateSchema(ctx, tenantB, validRequest())
	if err != nil {
		t.Fatalf("ValidateSchema: cross-tenant must be a report failure, got %v", err)
	}
	if report.Valid {
		t.Fatalf("ValidateSchema: tenant B must not validate tenant A's connection")
	}
	if !hasFailure(report, engine.ValidationDatasourceNotFound, "pg-main", "", "") {
		t.Fatalf("ValidateSchema: expected datasource-not-found for wrong tenant, got %+v", report.Failures)
	}
	if len(record.snapshot()) != 0 {
		t.Fatalf("ValidateSchema: connector must NOT be built for a wrong-tenant connection, got %v", record.snapshot())
	}
}

func TestEngine_ValidateSchema_MultipleFailures_StableReportShape(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	record := &schemaConnRecord{}
	factory := &schemaConnFactory{record: record, snapshot: validationSnapshot("pg-main")}
	eng, store := engineForDiscoverSchema(t, factory, memory.NewSchemaCache())
	tenant := mustTenant(t, "tenant-a")
	seedConnection(t, store, tenant, "pg-main", "postgres")

	req := engine.SchemaValidationRequest{
		Datasources: []engine.DatasourceMapping{
			{
				ConfigName: "pg-main",
				Tables: []engine.TableMapping{
					{Name: "public.ghost", Fields: []string{"id"}},
					{Name: "public.orders", Fields: []string{"missing_col"}, FilterFields: []string{"bad_filter"}},
				},
			},
		},
	}

	report, err := eng.ValidateSchema(ctx, tenant, req)
	if err != nil {
		t.Fatalf("ValidateSchema: unexpected error: %v", err)
	}
	if report.Valid {
		t.Fatalf("ValidateSchema: expected invalid report with multiple failures")
	}

	// Each distinct failure must be present and addressable.
	if !hasFailure(report, engine.ValidationTableNotFound, "pg-main", "public.ghost", "") {
		t.Fatalf("ValidateSchema: missing table-not-found, got %+v", report.Failures)
	}
	if !hasFailure(report, engine.ValidationFieldNotFound, "pg-main", "public.orders", "missing_col") {
		t.Fatalf("ValidateSchema: missing field-not-found, got %+v", report.Failures)
	}
	if !hasFailure(report, engine.ValidationInvalidFilter, "pg-main", "public.orders", "bad_filter") {
		t.Fatalf("ValidateSchema: missing invalid-filter, got %+v", report.Failures)
	}

	// Report shape is stable: every failure carries a typed, safe reason and no
	// failure leaks credentials or extracted data.
	for _, f := range report.Failures {
		if f.Type == "" || f.ConfigName == "" {
			t.Fatalf("ValidateSchema: failure missing type/configName: %+v", f)
		}
		for _, leak := range []string{"password", "s3cr3t", "svc", "10.0.0.1"} {
			if strings.Contains(f.Detail, leak) {
				t.Fatalf("ValidateSchema: failure detail leaked %q: %+v", leak, f)
			}
		}
	}
}

func TestEngine_ValidateSchema_NoConnectionStore_StableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := engine.New(engine.WithConnectorRegistry(memory.NewConnectorRegistry()))
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	_, err = eng.ValidateSchema(ctx, mustTenant(t, "tenant-a"), validRequest())
	if err == nil {
		t.Fatalf("ValidateSchema: expected validation error when no store configured")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("ValidateSchema: expected CategoryValidation, got %v", err)
	}
}

// engineForDiscoverSchemaWithLimits wires an Engine like engineForDiscoverSchema
// but with explicit Engine limits so the limit-violation tests can tighten a
// single bound.
func engineForDiscoverSchemaWithLimits(
	t *testing.T,
	factory engine.ConnectorFactory,
	cache engine.SchemaCache,
	limits engine.Limits,
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
		engine.WithLimits(limits),
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

// hasFailure reports whether the report carries a failure matching the given
// type/configName/table/field. Empty table/field act as wildcards.
func hasFailure(report engine.ValidationReport, failureType engine.ValidationFailureType, configName, table, field string) bool {
	for _, f := range report.Failures {
		if f.Type != failureType || f.ConfigName != configName {
			continue
		}
		if table != "" && f.Table != table {
			continue
		}
		if field != "" && f.Field != field {
			continue
		}

		return true
	}

	return false
}

// hasFailureType reports whether any failure of the given type is present.
func hasFailureType(report engine.ValidationReport, failureType engine.ValidationFailureType) bool {
	for _, f := range report.Failures {
		if f.Type == failureType {
			return true
		}
	}

	return false
}

// hasFailureDetail reports whether any failure of the given type carries a Detail
// containing the given substring, so distinct sub-types (e.g. field-count vs
// filter-count limit breaches) are individually assertable.
func hasFailureDetail(report engine.ValidationReport, failureType engine.ValidationFailureType, detailSubstr string) bool {
	for _, f := range report.Failures {
		if f.Type == failureType && strings.Contains(f.Detail, detailSubstr) {
			return true
		}
	}

	return false
}
