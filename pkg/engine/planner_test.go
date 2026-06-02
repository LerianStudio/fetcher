// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

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
