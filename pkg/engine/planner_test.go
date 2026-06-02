// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
)

// errInjectedSecret is a host-side error embedding sensitive-looking material so
// tests can prove the Engine never relays raw connector/driver internals across
// its safe error boundary.
var errInjectedSecret = errors.New("dial tcp: super-secret-dsn password=p@ssw0rd")

// assertEngineCategory fails the test unless err is an *engine.EngineError of the
// expected category.
func assertEngineCategory(t *testing.T, err error, want engine.ErrorCategory) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected %s error, got nil", want)
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != want {
		t.Fatalf("expected %s EngineError, got %v", want, err)
	}
}

// plannerSnapshot returns a deterministic schema snapshot whose tables/fields
// cover the mappings exercised by the planner tests. Field order is deliberately
// UNSORTED so a planner that copies snapshot order would produce nondeterministic
// output; the planner must impose its own sort.
func plannerSnapshot(configName string) engine.SchemaSnapshot {
	return engine.SchemaSnapshot{
		ConfigName: configName,
		Tables: []engine.TableSnapshot{
			{Name: "public.orders", Fields: []string{"amount", "id", "status"}},
			{Name: "public.customers", Fields: []string{"name", "id", "email"}},
		},
	}
}

// engineForPlan wires an Engine with the in-memory store, a registry holding the
// supplied factory under "postgres", and an OPTIONAL schema cache. A persisted
// connection named configName is created so the planner can resolve it.
func engineForPlan(
	t *testing.T,
	factory engine.ConnectorFactory,
	cache engine.SchemaCache,
	tenant engine.TenantContext,
	configName string,
) *engine.Engine {
	t.Helper()

	store := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()

	if factory != nil {
		registry.Register("postgres", factory)
	}

	descriptor := engine.ConnectionDescriptor{
		ConfigName: configName,
		Type:       "postgres",
		Host:       "db.internal",
		Port:       5432,
	}
	if err := store.Create(context.Background(), tenant, descriptor, nil); err != nil {
		t.Fatalf("store.Create() unexpected error: %v", err)
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

	return eng
}

func plannerTenant(t *testing.T) engine.TenantContext {
	t.Helper()

	tenant, err := engine.NewTenantContext("tenant-plan")
	if err != nil {
		t.Fatalf("NewTenantContext() unexpected error: %v", err)
	}

	return tenant.WithRequestID("req-plan-1")
}

func plannerRequest() engine.ExtractionRequest {
	return engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {
				"public.orders":    {"status", "id", "amount"},
				"public.customers": {"name", "id"},
			},
		},
		Filters: map[string]any{
			"orders-db": map[string]any{
				"public.orders": map[string]any{
					"status": "open",
				},
			},
		},
		Metadata: map[string]any{
			"source": "plugin_crm",
		},
	}
}

// TestEngine_PlanExtraction_DeterministicPlanFromMappedFields proves equivalent
// requests yield deeply-equal plans regardless of map-iteration order.
func TestEngine_PlanExtraction_DeterministicPlanFromMappedFields(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	first, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() first call error: %v", err)
	}

	factory2 := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng2 := engineForPlan(t, factory2, memory.NewSchemaCache(), tenant, "orders-db")

	second, err := eng2.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() second call error: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("equivalent requests produced non-equal plans:\n first=%#v\nsecond=%#v", first, second)
	}
}

// TestEngine_PlanExtraction_StableOrdering proves datasource, table, and field
// keys are sorted, not emitted in map-iteration order.
func TestEngine_PlanExtraction_StableOrdering(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	plan, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}

	step := plan.Steps[0]
	if step.ConfigName != "orders-db" {
		t.Fatalf("expected step config orders-db, got %q", step.ConfigName)
	}

	wantTables := []string{"public.customers", "public.orders"}
	if !reflect.DeepEqual(step.Tables, wantTables) {
		t.Fatalf("tables not sorted: got %v want %v", step.Tables, wantTables)
	}

	ordersFields := step.Fields["public.orders"]
	wantFields := []string{"amount", "id", "status"}
	if !reflect.DeepEqual(ordersFields, wantFields) {
		t.Fatalf("fields not sorted: got %v want %v", ordersFields, wantFields)
	}
}

// TestEngine_PlanExtraction_FilterAttachment proves filters attach only to the
// matching datasource/table/field path.
func TestEngine_PlanExtraction_FilterAttachment(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	plan, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	step := plan.Steps[0]
	if step.Filters == nil {
		t.Fatalf("expected filters attached to step")
	}

	ordersFilters, ok := step.Filters["public.orders"]
	if !ok {
		t.Fatalf("expected orders table filters, got %#v", step.Filters)
	}

	if _, ok := ordersFilters["status"]; !ok {
		t.Fatalf("expected status filter on public.orders, got %#v", ordersFilters)
	}

	if _, ok := step.Filters["public.customers"]; ok {
		t.Fatalf("did not expect filters on public.customers")
	}
}

// TestEngine_PlanExtraction_MetadataPreserved proves request metadata, including
// metadata.source, survives into the plan.
func TestEngine_PlanExtraction_MetadataPreserved(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	plan, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	source, ok := plan.Metadata["source"].(string)
	if !ok || source != "plugin_crm" {
		t.Fatalf("expected metadata.source plugin_crm preserved, got %#v", plan.Metadata)
	}

	if plan.TenantID != tenant.TenantID {
		t.Fatalf("expected plan tenant %q, got %q", tenant.TenantID, plan.TenantID)
	}

	if plan.RequestID != tenant.RequestID {
		t.Fatalf("expected plan request id %q, got %q", tenant.RequestID, plan.RequestID)
	}
}

// TestEngine_PlanExtraction_EmptyMappedFields_Validation proves an empty request
// fails with a validation error before any resolution.
func TestEngine_PlanExtraction_EmptyMappedFields_Validation(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	_, err := eng.PlanExtraction(context.Background(), tenant, engine.ExtractionRequest{})
	assertEngineCategory(t, err, engine.CategoryValidation)
}

// TestEngine_PlanExtraction_UnknownConfigName_NotFound proves an unknown config
// name fails as not-found before execution.
func TestEngine_PlanExtraction_UnknownConfigName_NotFound(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"ghost-db": {"public.orders": {"id"}},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryNotFound)

	if factory.record.has("build") {
		t.Fatalf("connector must not be built for an unknown config name")
	}
}

// TestEngine_PlanExtraction_SourceFailure_SafeError proves a connector/source
// failure returns a safe unavailable error and never leaks driver internals.
func TestEngine_PlanExtraction_SourceFailure_SafeError(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, buildErr: errInjectedSecret}
	eng := engineForPlan(t, factory, nil, tenant, "orders-db")

	_, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	assertEngineCategory(t, err, engine.CategoryUnavailable)

	if strings.Contains(err.Error(), "super-secret-dsn") {
		t.Fatalf("error leaked raw driver internals: %v", err)
	}
}

// TestEngine_PlanExtraction_SchemaCacheHit_NoDiscovery proves a cache hit avoids
// connector schema discovery.
func TestEngine_PlanExtraction_SchemaCacheHit_NoDiscovery(t *testing.T) {
	tenant := plannerTenant(t)
	cache := memory.NewSchemaCache()
	if err := cache.PutSchema(context.Background(), tenant, plannerSnapshot("orders-db")); err != nil {
		t.Fatalf("cache.PutSchema() error: %v", err)
	}

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, cache, tenant, "orders-db")

	_, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if factory.record.has("discover") {
		t.Fatalf("schema cache hit must not trigger connector discovery, calls=%v", factory.record.snapshot())
	}
}

// TestEngine_PlanExtraction_SchemaCacheMiss_Discovers proves a cache miss uses
// connector discovery via the registry.
func TestEngine_PlanExtraction_SchemaCacheMiss_Discovers(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	_, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if !factory.record.has("discover") {
		t.Fatalf("schema cache miss must trigger connector discovery, calls=%v", factory.record.snapshot())
	}
}

// TestEngine_PlanExtraction_InvalidTableReference_FailsBeforeExecution proves an
// invalid table reference fails before any executable plan is produced.
func TestEngine_PlanExtraction_InvalidTableReference_FailsBeforeExecution(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {"public.ghost": {"id"}},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)
}

// TestEngine_PlanExtraction_InvalidFieldReference_FailsBeforeExecution proves an
// invalid field reference fails before any executable plan is produced.
func TestEngine_PlanExtraction_InvalidFieldReference_FailsBeforeExecution(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {"public.orders": {"nonexistent_field"}},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)
}

// TestEngine_PlanExtraction_InvalidFilterReference_FailsBeforeExecution proves an
// invalid filter field reference fails before any executable plan is produced.
func TestEngine_PlanExtraction_InvalidFilterReference_FailsBeforeExecution(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {"public.orders": {"id"}},
		},
		Filters: map[string]any{
			"orders-db": map[string]any{
				"public.orders": map[string]any{
					"ghost_field": "x",
				},
			},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)
}

// TestEngine_PlanExtraction_NoRawCredentialMaterial proves a valid plan carries
// no raw credential material anywhere in its structure.
func TestEngine_PlanExtraction_NoRawCredentialMaterial(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	plan, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	rendered := strings.ToLower(plannerRender(plan))
	for _, banned := range []string{"password", "secret", "p@ssw0rd", "ciphertext"} {
		if strings.Contains(rendered, banned) {
			t.Fatalf("plan leaked credential material %q: %s", banned, rendered)
		}
	}
}

// TestEngine_PlanExtraction_PlanCopiesAreImmutable proves mutating a returned
// plan slice/map does not corrupt a freshly planned, equivalent request.
func TestEngine_PlanExtraction_PlanCopiesAreImmutable(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	plan, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	// Mutate caller-visible structures.
	if len(plan.Steps) > 0 {
		plan.Steps[0].Tables[0] = "MUTATED"
		plan.Steps[0].Fields["public.orders"][0] = "MUTATED"
	}
	plan.Metadata["source"] = "MUTATED"

	factory2 := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng2 := engineForPlan(t, factory2, memory.NewSchemaCache(), tenant, "orders-db")

	fresh, err := eng2.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() fresh error: %v", err)
	}

	if fresh.Steps[0].Tables[0] == "MUTATED" {
		t.Fatalf("plan internal table state was shared/mutable across calls")
	}

	if fresh.Metadata["source"] == "MUTATED" {
		t.Fatalf("plan metadata was shared/mutable across calls")
	}
}

// TestEngine_PlanExtraction_NoConnectionStore_StableError proves the planner
// fails with a stable error when no ConnectionStore is configured, before any
// tenant or request inspection.
func TestEngine_PlanExtraction_NoConnectionStore_StableError(t *testing.T) {
	registry := memory.NewConnectorRegistry()

	eng, err := engine.New(engine.WithConnectorRegistry(registry))
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	_, err = eng.PlanExtraction(context.Background(), plannerTenant(t), plannerRequest())
	assertEngineCategory(t, err, engine.CategoryValidation)
}

// TestEngine_PlanExtraction_InvalidTenantScope_FailsBeforeResolution proves an
// invalid tenant scope fails before any connection resolution.
func TestEngine_PlanExtraction_InvalidTenantScope_FailsBeforeResolution(t *testing.T) {
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), plannerTenant(t), "orders-db")

	_, err := eng.PlanExtraction(context.Background(), engine.TenantContext{}, plannerRequest())
	assertEngineCategory(t, err, engine.CategoryValidation)

	if factory.record.has("build") {
		t.Fatalf("connector must not be built for an invalid tenant scope")
	}
}

// TestEngine_PlanExtraction_FiltersWithoutValues_NoFilterMap proves a datasource
// whose filter map is empty (or malformed at the table layer) produces a step
// with no filter map rather than an empty one, keeping the plan minimal.
func TestEngine_PlanExtraction_FiltersWithoutValues_NoFilterMap(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlan(t, factory, memory.NewSchemaCache(), tenant, "orders-db")

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {"public.orders": {"id"}},
		},
		// Filters present for the datasource but with no usable table-level map.
		Filters: map[string]any{
			"orders-db": map[string]any{
				"public.orders": "not-a-map",
			},
		},
	}

	plan, err := eng.PlanExtraction(context.Background(), tenant, req)
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if plan.Steps[0].Filters != nil {
		t.Fatalf("expected no filter map for malformed table filters, got %#v", plan.Steps[0].Filters)
	}
}

// ---------------------------------------------------------------------------
// ST-T006-02: effective-limit enforcement + per-request overrides
// ---------------------------------------------------------------------------

// limitSnapshot builds a wide schema snapshot with the requested number of
// tables, each carrying the requested number of fields, so count-limit tests
// can construct requests that sit exactly on or just over a boundary. Table and
// field names are deterministic.
func limitSnapshot(configName string, tables, fieldsPerTable int) engine.SchemaSnapshot {
	snap := engine.SchemaSnapshot{ConfigName: configName}

	for ti := 0; ti < tables; ti++ {
		table := engine.TableSnapshot{Name: fmt.Sprintf("public.t%02d", ti)}
		for fi := 0; fi < fieldsPerTable; fi++ {
			table.Fields = append(table.Fields, fmt.Sprintf("f%02d", fi))
		}

		snap.Tables = append(snap.Tables, table)
	}

	return snap
}

// engineForPlanWithLimits wires an Engine like engineForPlan but with explicit
// Limits, so override and boundary behavior can be asserted against a known
// configuration.
func engineForPlanWithLimits(
	t *testing.T,
	factory engine.ConnectorFactory,
	cache engine.SchemaCache,
	tenant engine.TenantContext,
	configName string,
	limits engine.Limits,
) *engine.Engine {
	t.Helper()

	store := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()

	if factory != nil {
		registry.Register("postgres", factory)
	}

	descriptor := engine.ConnectionDescriptor{
		ConfigName: configName,
		Type:       "postgres",
		Host:       "db.internal",
		Port:       5432,
	}
	if err := store.Create(context.Background(), tenant, descriptor, nil); err != nil {
		t.Fatalf("store.Create() unexpected error: %v", err)
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

	return eng
}

// fieldSelection enumerates n deterministic field names matching limitSnapshot.
func fieldSelection(n int) []string {
	fields := make([]string, 0, n)
	for fi := 0; fi < n; fi++ {
		fields = append(fields, fmt.Sprintf("f%02d", fi))
	}

	return fields
}

// TestEngine_PlanExtraction_MaxDatasourceCount proves a request exceeding the
// effective datasource-count limit fails during planning, before execution.
func TestEngine_PlanExtraction_MaxDatasourceCount(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxDatasources = 1

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {"public.orders": {"id"}},
			"extra-db":  {"public.orders": {"id"}},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)

	if factory.record.has("discover") {
		t.Fatalf("datasource-count breach must short-circuit before connector discovery")
	}
}

// TestEngine_PlanExtraction_MaxTableCount proves a request exceeding the
// effective per-datasource table-count limit fails during planning.
func TestEngine_PlanExtraction_MaxTableCount(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxTablesPerDatasource = 1

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {
				"public.orders":    {"id"},
				"public.customers": {"id"},
			},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)
}

// TestEngine_PlanExtraction_MaxFieldCount proves a request exceeding the
// effective per-table field-count limit fails during planning.
func TestEngine_PlanExtraction_MaxFieldCount(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxFieldsPerTable = 2

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: limitSnapshot("orders-db", 1, 5)}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {"public.t00": fieldSelection(3)},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)
}

// TestEngine_PlanExtraction_MaxFilterCount proves a request whose filter-field
// references exceed the per-table field budget fails during planning.
func TestEngine_PlanExtraction_MaxFilterCount(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxFieldsPerTable = 2

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: limitSnapshot("orders-db", 1, 5)}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {"public.t00": {"f00"}},
		},
		Filters: map[string]any{
			"orders-db": map[string]any{
				"public.t00": map[string]any{
					"f00": "a", "f01": "b", "f02": "c",
				},
			},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)
}

// TestEngine_PlanExtraction_MaxConcurrencyOverrideRejected proves a per-request
// override that exceeds the Engine maximum concurrency is rejected as a
// validation error and never reaches connector discovery.
func TestEngine_PlanExtraction_MaxConcurrencyOverrideRejected(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxConcurrency = 4

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := plannerRequest()
	req.Overrides = &engine.Limits{MaxConcurrency: 999}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)

	if factory.record.has("discover") {
		t.Fatalf("an invalid override must be rejected before connector discovery")
	}
}

// TestEngine_PlanExtraction_TimeoutDefaulting proves that when a request carries
// no timeout override, the plan inherits the Engine's default timeout.
func TestEngine_PlanExtraction_TimeoutDefaulting(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	plan, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if plan.Limits.Timeout != limits.Timeout {
		t.Fatalf("expected plan timeout to default to %v, got %v", limits.Timeout, plan.Limits.Timeout)
	}
}

// TestEngine_PlanExtraction_ValidOverrideApplied proves a per-request override
// within the Engine maximums is applied to the produced plan's limits.
func TestEngine_PlanExtraction_ValidOverrideApplied(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxConcurrency = 8

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := plannerRequest()
	req.Overrides = &engine.Limits{MaxConcurrency: 2}

	plan, err := eng.PlanExtraction(context.Background(), tenant, req)
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if plan.Limits.MaxConcurrency != 2 {
		t.Fatalf("expected override concurrency 2 applied, got %d", plan.Limits.MaxConcurrency)
	}

	// Unspecified override fields must inherit the Engine default.
	if plan.Limits.MaxDatasources != limits.MaxDatasources {
		t.Fatalf("expected unspecified override field to inherit default %d, got %d",
			limits.MaxDatasources, plan.Limits.MaxDatasources)
	}
}

// TestEngine_PlanExtraction_InvalidOverrideDoesNotMutateDefaults proves a
// rejected override leaves the Engine's default limits untouched (copy
// semantics), so a later valid request sees the original defaults.
func TestEngine_PlanExtraction_InvalidOverrideDoesNotMutateDefaults(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxConcurrency = 4

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	bad := plannerRequest()
	bad.Overrides = &engine.Limits{MaxConcurrency: 999}

	if _, err := eng.PlanExtraction(context.Background(), tenant, bad); err == nil {
		t.Fatalf("expected invalid override to be rejected")
	}

	if got := eng.Limits().MaxConcurrency; got != 4 {
		t.Fatalf("invalid override mutated Engine default concurrency: got %d want 4", got)
	}

	// A subsequent valid request must see the untouched defaults.
	plan, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error after rejected override: %v", err)
	}

	if plan.Limits.MaxConcurrency != 4 {
		t.Fatalf("expected default concurrency 4 after rejected override, got %d", plan.Limits.MaxConcurrency)
	}
}

// TestEngine_PlanExtraction_CacheHitUnderLimits proves a request within
// effective limits served from cache avoids connector discovery.
func TestEngine_PlanExtraction_CacheHitUnderLimits(t *testing.T) {
	tenant := plannerTenant(t)
	cache := memory.NewSchemaCache()
	if err := cache.PutSchema(context.Background(), tenant, plannerSnapshot("orders-db")); err != nil {
		t.Fatalf("cache.PutSchema() error: %v", err)
	}

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, cache, tenant, "orders-db", engine.DefaultLimits())

	if _, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest()); err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if factory.record.has("discover") {
		t.Fatalf("cache hit under limits must not trigger connector discovery")
	}
}

// TestEngine_PlanExtraction_CacheMissUnderLimits proves a request within
// effective limits with no cache entry uses connector discovery.
func TestEngine_PlanExtraction_CacheMissUnderLimits(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", engine.DefaultLimits())

	if _, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest()); err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if !factory.record.has("discover") {
		t.Fatalf("cache miss under limits must trigger connector discovery")
	}
}

// TestEngine_PlanExtraction_UnknownConfigAfterLimits proves an unknown config
// name still fails as not-found after the request passes limit validation.
func TestEngine_PlanExtraction_UnknownConfigAfterLimits(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", engine.DefaultLimits())

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"ghost-db": {"public.orders": {"id"}},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryNotFound)
}

// TestEngine_PlanExtraction_InvalidRefsWithinLimits proves invalid table/field/
// filter references still fail before execution when the request sits within
// limit boundaries.
func TestEngine_PlanExtraction_InvalidRefsWithinLimits(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", engine.DefaultLimits())

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {"public.orders": {"nonexistent_field"}},
		},
	}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)
}

// TestEngine_PlanExtraction_NoCredentialMaterialAfterLimits proves a valid plan
// produced after limit checks carries no raw credential material.
func TestEngine_PlanExtraction_NoCredentialMaterialAfterLimits(t *testing.T) {
	tenant := plannerTenant(t)
	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", engine.DefaultLimits())

	plan, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	rendered := strings.ToLower(plannerRender(plan))
	for _, banned := range []string{"password", "secret", "p@ssw0rd", "ciphertext"} {
		if strings.Contains(rendered, banned) {
			t.Fatalf("plan leaked credential material %q: %s", banned, rendered)
		}
	}
}

// TestEngine_PlanExtraction_BoundaryRequestPlansSuccessfully proves a request
// exactly on the configured limit boundaries plans successfully.
func TestEngine_PlanExtraction_BoundaryRequestPlansSuccessfully(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxDatasources = 1
	limits.MaxTablesPerDatasource = 1
	limits.MaxFieldsPerTable = 3

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: limitSnapshot("orders-db", 1, 3)}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"orders-db": {"public.t00": fieldSelection(3)},
		},
	}

	plan, err := eng.PlanExtraction(context.Background(), tenant, req)
	if err != nil {
		t.Fatalf("boundary request must plan successfully, got error: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step at boundary, got %d", len(plan.Steps))
	}
}

// TestEngine_PlanExtraction_TimeoutOverrideRejected proves a timeout override
// above the Engine maximum is rejected before any connector discovery.
func TestEngine_PlanExtraction_TimeoutOverrideRejected(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.Timeout = time.Minute

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := plannerRequest()
	req.Overrides = &engine.Limits{Timeout: time.Hour}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)

	if factory.record.has("discover") {
		t.Fatalf("timeout override breach must short-circuit before connector discovery")
	}
}

// TestEngine_PlanExtraction_TimeoutOverrideApplied proves a timeout override
// within the Engine maximum is applied to the plan.
func TestEngine_PlanExtraction_TimeoutOverrideApplied(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.Timeout = time.Hour

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := plannerRequest()
	req.Overrides = &engine.Limits{Timeout: time.Minute}

	plan, err := eng.PlanExtraction(context.Background(), tenant, req)
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if plan.Limits.Timeout != time.Minute {
		t.Fatalf("expected timeout override 1m applied, got %v", plan.Limits.Timeout)
	}
}

// TestEngine_PlanExtraction_ResultBytesOverrideRejected proves a result-size
// override above the Engine maximum is rejected before connector discovery.
func TestEngine_PlanExtraction_ResultBytesOverrideRejected(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxResultBytes = 1024

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := plannerRequest()
	req.Overrides = &engine.Limits{MaxResultBytes: 1 << 30}

	_, err := eng.PlanExtraction(context.Background(), tenant, req)
	assertEngineCategory(t, err, engine.CategoryValidation)

	if factory.record.has("discover") {
		t.Fatalf("result-size override breach must short-circuit before connector discovery")
	}
}

// TestEngine_PlanExtraction_ResultBytesOverrideApplied proves a result-size
// override within the Engine maximum is applied to the plan.
func TestEngine_PlanExtraction_ResultBytesOverrideApplied(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.MaxResultBytes = 1 << 30

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	req := plannerRequest()
	req.Overrides = &engine.Limits{MaxResultBytes: 1024}

	plan, err := eng.PlanExtraction(context.Background(), tenant, req)
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if plan.Limits.MaxResultBytes != 1024 {
		t.Fatalf("expected result-size override 1024 applied, got %d", plan.Limits.MaxResultBytes)
	}
}

// TestEngine_PlanExtraction_ConnectorHardLimitsCopiedAndIsolated proves the
// connector-specific hard limits configured on the Engine are carried into the
// plan as an INDEPENDENT copy: mutating the plan's map must not reach back into
// the Engine's shared default limits.
func TestEngine_PlanExtraction_ConnectorHardLimitsCopiedAndIsolated(t *testing.T) {
	tenant := plannerTenant(t)
	limits := engine.DefaultLimits()
	limits.ConnectorHardLimits = map[string]int{"postgres": 1000}

	factory := &schemaConnFactory{record: &schemaConnRecord{}, snapshot: plannerSnapshot("orders-db")}
	eng := engineForPlanWithLimits(t, factory, memory.NewSchemaCache(), tenant, "orders-db", limits)

	plan, err := eng.PlanExtraction(context.Background(), tenant, plannerRequest())
	if err != nil {
		t.Fatalf("PlanExtraction() error: %v", err)
	}

	if plan.Limits.ConnectorHardLimits["postgres"] != 1000 {
		t.Fatalf("expected connector hard limit carried into plan, got %#v", plan.Limits.ConnectorHardLimits)
	}

	// Mutate the plan's copy and confirm the Engine default is untouched.
	plan.Limits.ConnectorHardLimits["postgres"] = 1
	if eng.Limits().ConnectorHardLimits["postgres"] != 1000 {
		t.Fatalf("plan mutation leaked into Engine default connector hard limits")
	}
}

// plannerRender produces a flat string view of every identity in the plan so a
// credential-leak assertion can scan it without depending on JSON tags.
func plannerRender(plan engine.ExtractionPlan) string {
	var b strings.Builder

	b.WriteString(plan.TenantID)
	b.WriteString(plan.RequestID)

	for k, v := range plan.Metadata {
		b.WriteString(k)
		b.WriteString(strings.ToLower(toStr(v)))
	}

	for _, step := range plan.Steps {
		b.WriteString(step.ConfigName)
		b.WriteString(strings.Join(step.Tables, ","))

		for table, fields := range step.Fields {
			b.WriteString(table)
			b.WriteString(strings.Join(fields, ","))
		}

		for table, fields := range step.Filters {
			b.WriteString(table)

			for f, v := range fields {
				b.WriteString(f)
				b.WriteString(toStr(v))
			}
		}
	}

	return b.String()
}

func toStr(v any) string {
	s, ok := v.(string)
	if ok {
		return s
	}

	return ""
}
