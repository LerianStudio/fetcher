// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/engine/memory"
)

// containsSecret marshals the descriptor and reports whether the raw secret
// leaks into its serialized form. ConnectionDescriptor is the secret-free output
// contract, so this must always be false for a correct implementation.
func containsSecret(t *testing.T, desc engine.ConnectionDescriptor, secret string) bool {
	t.Helper()

	raw, err := json.Marshal(desc)
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}

	return strings.Contains(string(raw), secret)
}

// engineWithStore builds an Engine wired with the in-memory connection store and
// connector registry. It fails the test on construction error so callers can use
// the returned Engine directly.
func engineWithStore(t *testing.T) (*engine.Engine, *memory.ConnectionStore) {
	t.Helper()

	store := memory.NewConnectionStore()

	eng, err := engine.New(
		engine.WithConnectorRegistry(memory.NewConnectorRegistry()),
		engine.WithConnectionStore(store),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	return eng, store
}

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

func newInput(configName string) engine.ConnectionInput {
	return engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName:   configName,
		Type:         "postgres",
		Host:         "db.internal",
		Port:         5432,
		DatabaseName: "ledger",
		Username:     "svc",
		Password:     "s3cr3t",
		SSLMode:      "require",
	})
}

func TestEngine_CreateConnection_StoresScopedAndRedacts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, store := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	desc, err := eng.CreateConnection(ctx, tenant, newInput("pg-main"))
	if err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	// Output is the secret-free descriptor.
	if desc.ConfigName != "pg-main" {
		t.Fatalf("CreateConnection: ConfigName = %q, want %q", desc.ConfigName, "pg-main")
	}
	if desc.Host != "db.internal" || desc.Port != 5432 || desc.Username != "svc" {
		t.Fatalf("CreateConnection: descriptor fields not projected: %+v", desc)
	}

	// The returned descriptor must carry no secret material. ConnectionDescriptor
	// has no password field, so a defensive scan over its JSON confirms redaction.
	if containsSecret(t, desc, "s3cr3t") {
		t.Fatalf("CreateConnection: descriptor leaked secret material: %+v", desc)
	}

	// Stored under the tenant scope and retrievable through the port.
	stored, found, err := store.FindConnection(ctx, tenant, "pg-main")
	if err != nil {
		t.Fatalf("FindConnection: unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("FindConnection: expected stored connection to be found")
	}
	if stored.ConfigName != "pg-main" {
		t.Fatalf("FindConnection: ConfigName = %q, want %q", stored.ConfigName, "pg-main")
	}
}

func TestEngine_CreateConnection_DuplicateWithinTenantConflicts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection first: unexpected error: %v", err)
	}

	_, err := eng.CreateConnection(ctx, tenant, newInput("pg-main"))
	if err == nil {
		t.Fatalf("CreateConnection duplicate: expected conflict error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("CreateConnection duplicate: error type = %T, want *engine.EngineError", err)
	}
	if engErr.Category != engine.CategoryConflict {
		t.Fatalf("CreateConnection duplicate: category = %q, want %q", engErr.Category, engine.CategoryConflict)
	}
}

// TestEngine_CreateConnection_DuplicateWithProtectorIsRejectedBeforeReEncrypt
// covers B4 finding #2: a duplicate create (same tenant+configName) while a
// CredentialProtector is enabled must be rejected by the uniqueness check
// BEFORE any re-encryption or second persistence. The first record's ciphertext
// must be untouched and no second record may be created.
func TestEngine_CreateConnection_DuplicateWithProtectorIsRejectedBeforeReEncrypt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	protector := newFakeProtector()
	eng, store := engineWithProtector(t, protector)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection first: unexpected error: %v", err)
	}
	if protector.calls() != 1 {
		t.Fatalf("after first create: protector calls = %d, want 1", protector.calls())
	}

	original, found := store.ProtectedCredential(tenant, "pg-main")
	if !found {
		t.Fatalf("expected stored credential after first create")
	}

	// A duplicate create with a DIFFERENT password must be rejected. The Engine
	// checks existence FIRST, so the protector is NOT invoked a second time: only
	// the original create encrypted. The stored ciphertext and record count must
	// not change.
	dup := engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName:   "pg-main",
		Type:         "postgres",
		Host:         "db.internal",
		Port:         5432,
		DatabaseName: "ledger",
		Username:     "svc",
		Password:     "different-secret",
		SSLMode:      "require",
	})

	_, err := eng.CreateConnection(ctx, tenant, dup)
	if err == nil {
		t.Fatalf("CreateConnection duplicate: expected conflict error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("CreateConnection duplicate: error type = %T, want *engine.EngineError", err)
	}
	if engErr.Category != engine.CategoryConflict {
		t.Fatalf("CreateConnection duplicate: category = %q, want %q", engErr.Category, engine.CategoryConflict)
	}

	// The duplicate was rejected by the existence pre-check BEFORE protect: the
	// protector must still show exactly one call (the original create). This is
	// the reject-before-encrypt ordering the test name asserts.
	if protector.calls() != 1 {
		t.Fatalf("duplicate create: protector calls = %d, want 1 (reject-before-encrypt: duplicate must NOT re-encrypt)", protector.calls())
	}

	// The existing record's ciphertext must be untouched.
	after, found := store.ProtectedCredential(tenant, "pg-main")
	if !found {
		t.Fatalf("duplicate create: original credential disappeared")
	}
	if string(after.Ciphertext) != string(original.Ciphertext) {
		t.Fatalf("duplicate create: stored ciphertext mutated: got %q, want %q", after.Ciphertext, original.Ciphertext)
	}

	// No second record may have been created.
	list, err := eng.ListConnections(ctx, tenant)
	if err != nil {
		t.Fatalf("ListConnections: unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("duplicate create: connection count = %d, want 1 (no second record)", len(list))
	}
}

// TestEngine_Connection_TenantIsolationThroughEngine covers B4 finding #3: with
// tenantID as the sole boundary, two tenants may each own a connection with the
// SAME config name without collision, and neither tenant's Get/List ever sees
// the other's connection.
func TestEngine_Connection_TenantIsolationThroughEngine(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)

	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	if _, err := eng.CreateConnection(ctx, tenantA, newInput("prod-db")); err != nil {
		t.Fatalf("CreateConnection tenant-a: unexpected error: %v", err)
	}

	// Same config name under a DIFFERENT tenant must succeed (no collision).
	if _, err := eng.CreateConnection(ctx, tenantB, newInput("prod-db")); err != nil {
		t.Fatalf("CreateConnection tenant-b: expected isolation success, got error: %v", err)
	}

	// Give tenant B a second, distinct connection so list scoping is observable.
	if _, err := eng.CreateConnection(ctx, tenantB, newInput("analytics-db")); err != nil {
		t.Fatalf("CreateConnection tenant-b second: unexpected error: %v", err)
	}

	// Get is tenant-scoped: tenant A sees only its own prod-db.
	gotA, err := eng.GetConnection(ctx, tenantA, "prod-db")
	if err != nil {
		t.Fatalf("GetConnection tenant-a: unexpected error: %v", err)
	}
	if gotA.ConfigName != "prod-db" {
		t.Fatalf("GetConnection tenant-a: ConfigName = %q, want prod-db", gotA.ConfigName)
	}
	if _, err := eng.GetConnection(ctx, tenantA, "analytics-db"); err == nil {
		t.Fatalf("GetConnection tenant-a: must not see tenant-b's analytics-db")
	}

	// List is tenant-scoped in both directions.
	listA, err := eng.ListConnections(ctx, tenantA)
	if err != nil {
		t.Fatalf("ListConnections tenant-a: unexpected error: %v", err)
	}
	if len(listA) != 1 || listA[0].ConfigName != "prod-db" {
		t.Fatalf("ListConnections tenant-a = %+v, want exactly [prod-db]", listA)
	}

	listB, err := eng.ListConnections(ctx, tenantB)
	if err != nil {
		t.Fatalf("ListConnections tenant-b: unexpected error: %v", err)
	}
	if len(listB) != 2 {
		t.Fatalf("ListConnections tenant-b: len = %d, want 2 (tenant-a leaked?)", len(listB))
	}
}

func TestEngine_GetConnection_FoundAndNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	got, err := eng.GetConnection(ctx, tenant, "pg-main")
	if err != nil {
		t.Fatalf("GetConnection: unexpected error: %v", err)
	}
	if got.ConfigName != "pg-main" {
		t.Fatalf("GetConnection: ConfigName = %q, want %q", got.ConfigName, "pg-main")
	}

	_, err = eng.GetConnection(ctx, tenant, "missing")
	if err == nil {
		t.Fatalf("GetConnection missing: expected not-found error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("GetConnection missing: error type = %T, want *engine.EngineError", err)
	}
	if engErr.Category != engine.CategoryNotFound {
		t.Fatalf("GetConnection missing: category = %q, want %q", engErr.Category, engine.CategoryNotFound)
	}
}

func TestEngine_GetConnection_DeletedIsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}
	if err := eng.DeleteConnection(ctx, tenant, "pg-main"); err != nil {
		t.Fatalf("DeleteConnection: unexpected error: %v", err)
	}

	_, err := eng.GetConnection(ctx, tenant, "pg-main")
	if err == nil {
		t.Fatalf("GetConnection deleted: expected not-found error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("GetConnection deleted: error type = %T, want *engine.EngineError", err)
	}
	if engErr.Category != engine.CategoryNotFound {
		t.Fatalf("GetConnection deleted: category = %q, want %q", engErr.Category, engine.CategoryNotFound)
	}
}

func TestEngine_ListConnections_ByTenantDeterministicAndScoped(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)

	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	for _, name := range []string{"zeta", "alpha", "mike"} {
		if _, err := eng.CreateConnection(ctx, tenantA, newInput(name)); err != nil {
			t.Fatalf("CreateConnection %s: unexpected error: %v", name, err)
		}
	}
	if _, err := eng.CreateConnection(ctx, tenantB, newInput("other")); err != nil {
		t.Fatalf("CreateConnection tenant-b: unexpected error: %v", err)
	}

	list, err := eng.ListConnections(ctx, tenantA)
	if err != nil {
		t.Fatalf("ListConnections: unexpected error: %v", err)
	}

	wantOrder := []string{"alpha", "mike", "zeta"}
	if len(list) != len(wantOrder) {
		t.Fatalf("ListConnections: len = %d, want %d (tenant-b leaked?)", len(list), len(wantOrder))
	}
	for i, want := range wantOrder {
		if list[i].ConfigName != want {
			t.Fatalf("ListConnections[%d].ConfigName = %q, want %q (order must be deterministic)", i, list[i].ConfigName, want)
		}
	}
}

func TestEngine_ListConnections_ExcludesDeleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	for _, name := range []string{"keep", "drop"} {
		if _, err := eng.CreateConnection(ctx, tenant, newInput(name)); err != nil {
			t.Fatalf("CreateConnection %s: unexpected error: %v", name, err)
		}
	}
	if err := eng.DeleteConnection(ctx, tenant, "drop"); err != nil {
		t.Fatalf("DeleteConnection: unexpected error: %v", err)
	}

	list, err := eng.ListConnections(ctx, tenant)
	if err != nil {
		t.Fatalf("ListConnections: unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ConfigName != "keep" {
		t.Fatalf("ListConnections: deleted connection not excluded: %+v", list)
	}
}

func TestEngine_UpdateConnection_PatchPartialFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	// Patch only the host; other fields must be preserved.
	newHost := "db.replica"
	patch := engine.ConnectionPatch{Host: &newHost}

	updated, err := eng.UpdateConnection(ctx, tenant, "pg-main", patch)
	if err != nil {
		t.Fatalf("UpdateConnection: unexpected error: %v", err)
	}

	if updated.Host != "db.replica" {
		t.Fatalf("UpdateConnection: Host = %q, want %q", updated.Host, "db.replica")
	}
	// Untouched fields preserved (Manager-compatible patch semantics).
	if updated.Port != 5432 || updated.DatabaseName != "ledger" || updated.Username != "svc" {
		t.Fatalf("UpdateConnection: untouched fields changed: %+v", updated)
	}
}

func TestEngine_UpdateConnection_PatchAllFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	newType := "mysql"
	newHost := "db.replica"
	newPort := 3306
	newDB := "analytics"
	newSchema := "reporting"
	newUser := "reader"
	newSSL := "verify-full"

	patch := engine.ConnectionPatch{
		Type:         &newType,
		Host:         &newHost,
		Port:         &newPort,
		DatabaseName: &newDB,
		Schema:       &newSchema,
		Username:     &newUser,
		SSLMode:      &newSSL,
	}.WithPassword("rotated-secret")

	if !patch.HasPassword() {
		t.Fatalf("ConnectionPatch.HasPassword() = false, want true after WithPassword")
	}

	updated, err := eng.UpdateConnection(ctx, tenant, "pg-main", patch)
	if err != nil {
		t.Fatalf("UpdateConnection: unexpected error: %v", err)
	}

	if updated.Type != newType || updated.Host != newHost || updated.Port != newPort ||
		updated.DatabaseName != newDB || updated.Schema != newSchema ||
		updated.Username != newUser || updated.SSLMode != newSSL {
		t.Fatalf("UpdateConnection: not all fields patched: %+v", updated)
	}

	// Config name is immutable through a patch; identity stays anchored.
	if updated.ConfigName != "pg-main" {
		t.Fatalf("UpdateConnection: identity changed: %+v", updated)
	}

	// The rotated secret must not leak into the descriptor.
	if containsSecret(t, updated, "rotated-secret") {
		t.Fatalf("UpdateConnection: descriptor leaked rotated secret: %+v", updated)
	}
}

func TestEngine_ConnectionPatch_EmptyHasNoPassword(t *testing.T) {
	t.Parallel()

	if (engine.ConnectionPatch{}).HasPassword() {
		t.Fatalf("empty ConnectionPatch.HasPassword() = true, want false")
	}
}

func TestEngine_UpdateConnection_MissingIsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	newHost := "x"
	_, err := eng.UpdateConnection(ctx, tenant, "ghost", engine.ConnectionPatch{Host: &newHost})
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
}

func TestEngine_DeleteConnection_MissingIsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	err := eng.DeleteConnection(ctx, tenant, "ghost")
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
}

func TestEngine_ConnectionOps_RequireConnectionStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// An Engine with no connection store cannot perform connection ops; the
	// operations must report a stable error rather than panic on a nil port.
	eng, err := engine.New(engine.WithConnectorRegistry(memory.NewConnectorRegistry()))
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err == nil {
		t.Fatalf("CreateConnection without store: expected error, got nil")
	}
	if _, err := eng.GetConnection(ctx, tenant, "pg-main"); err == nil {
		t.Fatalf("GetConnection without store: expected error, got nil")
	}
	if _, err := eng.ListConnections(ctx, tenant); err == nil {
		t.Fatalf("ListConnections without store: expected error, got nil")
	}
}

// failingStore is a ConnectionStore whose read/list paths return a transport
// error, exercising the Engine's error-propagation branches without relying on
// the in-memory harness's happy path.
type failingStore struct{}

func (failingStore) FindConnection(context.Context, engine.TenantContext, string) (engine.ConnectionDescriptor, bool, error) {
	return engine.ConnectionDescriptor{}, false, engine.NewEngineError(engine.CategoryUnavailable, "store down")
}

func (failingStore) Create(context.Context, engine.TenantContext, engine.ConnectionDescriptor, *engine.ProtectedCredential) error {
	return engine.NewEngineError(engine.CategoryUnavailable, "store down")
}

func (failingStore) Update(context.Context, engine.TenantContext, engine.ConnectionDescriptor, *engine.ProtectedCredential) error {
	return engine.NewEngineError(engine.CategoryUnavailable, "store down")
}

func (failingStore) Delete(context.Context, engine.TenantContext, string) error {
	return engine.NewEngineError(engine.CategoryUnavailable, "store down")
}

func (failingStore) List(context.Context, engine.TenantContext) ([]engine.ConnectionDescriptor, error) {
	return nil, engine.NewEngineError(engine.CategoryUnavailable, "store down")
}

func (failingStore) FindByID(context.Context, engine.TenantContext, string) (engine.ConnectionDescriptor, bool, error) {
	return engine.ConnectionDescriptor{}, false, engine.NewEngineError(engine.CategoryUnavailable, "store down")
}

func (failingStore) UpdateByID(context.Context, engine.TenantContext, string, engine.ConnectionDescriptor, *engine.ProtectedCredential) error {
	return engine.NewEngineError(engine.CategoryUnavailable, "store down")
}

func (failingStore) DeleteByID(context.Context, engine.TenantContext, string) error {
	return engine.NewEngineError(engine.CategoryUnavailable, "store down")
}

func (failingStore) ListPaged(context.Context, engine.TenantContext, engine.ConnectionListParams) (engine.ConnectionPage, error) {
	return engine.ConnectionPage{}, engine.NewEngineError(engine.CategoryUnavailable, "store down")
}

func TestEngine_ConnectionOps_PropagateStoreErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := engine.New(
		engine.WithConnectorRegistry(memory.NewConnectorRegistry()),
		engine.WithConnectionStore(failingStore{}),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("c")); err == nil {
		t.Fatalf("CreateConnection: expected store error, got nil")
	}
	if _, err := eng.GetConnection(ctx, tenant, "c"); err == nil {
		t.Fatalf("GetConnection: expected store error, got nil")
	}
	if _, err := eng.ListConnections(ctx, tenant); err == nil {
		t.Fatalf("ListConnections: expected store error, got nil")
	}
	if _, err := eng.UpdateConnection(ctx, tenant, "c", engine.ConnectionPatch{}); err == nil {
		t.Fatalf("UpdateConnection: expected store error, got nil")
	}
	if err := eng.DeleteConnection(ctx, tenant, "c"); err == nil {
		t.Fatalf("DeleteConnection: expected store error, got nil")
	}
}

// recordingObservability records the span operations the Engine starts so the
// optional Observability seam is exercised end-to-end.
type recordingObservability struct {
	mu    sync.Mutex
	spans []string
}

func (o *recordingObservability) StartSpan(ctx context.Context, operation string) (context.Context, func()) {
	o.mu.Lock()
	o.spans = append(o.spans, operation)
	o.mu.Unlock()

	return ctx, func() {}
}

func TestEngine_ConnectionOps_StartSpanWhenObservabilityConfigured(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	obs := &recordingObservability{}
	store := memory.NewConnectionStore()

	eng, err := engine.New(
		engine.WithConnectorRegistry(memory.NewConnectorRegistry()),
		engine.WithConnectionStore(store),
		engine.WithObservability(obs),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	obs.mu.Lock()
	defer obs.mu.Unlock()
	if len(obs.spans) != 1 || obs.spans[0] != "engine.connection.create" {
		t.Fatalf("Observability spans = %v, want [engine.connection.create]", obs.spans)
	}
}

// fakeProtector is a deterministic CredentialProtector test double. It does not
// encrypt for real; it wraps the plaintext with a constant prefix so tests can
// prove the stored material differs from the plaintext, and reports a fixed key
// version so descriptor metadata can be asserted. When errOnProtect is set every
// Protect call fails, exercising the protection-failure path. Calls are counted
// so tests can assert the protector is invoked only when a password is supplied.
type fakeProtector struct {
	mu           sync.Mutex
	prefix       string
	keyVersion   int
	errOnProtect error
	protectCalls int
	lastPlain    []byte
}

func newFakeProtector() *fakeProtector {
	return &fakeProtector{prefix: "enc:", keyVersion: 7}
}

func (f *fakeProtector) Protect(_ context.Context, _ engine.TenantContext, plaintext []byte) ([]byte, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.protectCalls++
	f.lastPlain = append([]byte(nil), plaintext...)

	if f.errOnProtect != nil {
		return nil, 0, f.errOnProtect
	}

	out := append([]byte(f.prefix), plaintext...)

	return out, f.keyVersion, nil
}

func (f *fakeProtector) Reveal(_ context.Context, _ engine.TenantContext, ciphertext []byte, _ int) ([]byte, error) {
	return ciphertext, nil
}

func (f *fakeProtector) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.protectCalls
}

// engineWithProtector builds an Engine with the in-memory store, a connector
// registry, and encrypted persistence backed by the supplied protector.
func engineWithProtector(t *testing.T, protector engine.CredentialProtector) (*engine.Engine, *memory.ConnectionStore) {
	t.Helper()

	store := memory.NewConnectionStore()

	eng, err := engine.New(
		engine.WithConnectorRegistry(memory.NewConnectorRegistry()),
		engine.WithConnectionStore(store),
		engine.WithEncryptedPersistence(true),
		engine.WithCredentialProtector(protector),
	)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	return eng, store
}

func TestEngine_CreateConnection_EncryptsPasswordBeforeStorage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	protector := newFakeProtector()
	eng, store := engineWithProtector(t, protector)
	tenant := mustTenant(t, "tenant-a")

	desc, err := eng.CreateConnection(ctx, tenant, newInput("pg-main"))
	if err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	// The protector must have been called exactly once with the plaintext secret.
	if protector.calls() != 1 {
		t.Fatalf("CreateConnection: protector calls = %d, want 1", protector.calls())
	}

	// The returned descriptor carries the key-version metadata and no secret.
	if desc.KeyVersion != protector.keyVersion {
		t.Fatalf("CreateConnection: descriptor KeyVersion = %d, want %d", desc.KeyVersion, protector.keyVersion)
	}
	if containsSecret(t, desc, "s3cr3t") {
		t.Fatalf("CreateConnection: descriptor leaked secret material: %+v", desc)
	}

	// The stored protected material must differ from the plaintext password and
	// carry the protector's key version.
	cred, found := store.ProtectedCredential(tenant, "pg-main")
	if !found {
		t.Fatalf("CreateConnection: expected protected credential to be stored")
	}
	if len(cred.Ciphertext) == 0 {
		t.Fatalf("CreateConnection: stored ciphertext is empty")
	}
	if string(cred.Ciphertext) == "s3cr3t" {
		t.Fatalf("CreateConnection: stored credential equals plaintext (not protected)")
	}
	if strings.Contains(string(cred.Ciphertext), "s3cr3t") == false {
		// The fake protector wraps (does not strip) the plaintext, so the wrapped
		// form must still contain it; a real protector would not. This asserts the
		// fake actually saw the plaintext rather than an empty value.
		t.Fatalf("CreateConnection: fake protector did not receive plaintext, ciphertext = %q", cred.Ciphertext)
	}
	if cred.KeyVersion != protector.keyVersion {
		t.Fatalf("CreateConnection: stored KeyVersion = %d, want %d", cred.KeyVersion, protector.keyVersion)
	}
}

func TestEngine_CreateConnection_NoPasswordDoesNotProtect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	protector := newFakeProtector()
	eng, store := engineWithProtector(t, protector)
	tenant := mustTenant(t, "tenant-a")

	input := engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName: "no-secret",
		Type:       "postgres",
		Host:       "db.internal",
		Port:       5432,
		Username:   "svc",
	})

	desc, err := eng.CreateConnection(ctx, tenant, input)
	if err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	if protector.calls() != 0 {
		t.Fatalf("CreateConnection without password: protector calls = %d, want 0", protector.calls())
	}
	if desc.KeyVersion != 0 {
		t.Fatalf("CreateConnection without password: KeyVersion = %d, want 0", desc.KeyVersion)
	}
	if _, found := store.ProtectedCredential(tenant, "no-secret"); found {
		t.Fatalf("CreateConnection without password: no protected credential should be stored")
	}
}

func TestEngine_CreateConnection_ProtectorErrorIsSafeAndNoMutation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	protector := newFakeProtector()
	protector.errOnProtect = errors.New("kms unreachable: secret=s3cr3t")
	eng, store := engineWithProtector(t, protector)
	tenant := mustTenant(t, "tenant-a")

	_, err := eng.CreateConnection(ctx, tenant, newInput("pg-main"))
	if err == nil {
		t.Fatalf("CreateConnection: expected protection error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("CreateConnection: error type = %T, want *engine.EngineError", err)
	}
	// The safe error must not leak the plaintext password nor the raw protector
	// error (which itself carries the secret in this test).
	if strings.Contains(engErr.Error(), "s3cr3t") {
		t.Fatalf("CreateConnection: error leaked secret material: %q", engErr.Error())
	}

	// Atomicity: nothing must have been written to the store on protection failure.
	if _, found, _ := store.FindConnection(ctx, tenant, "pg-main"); found {
		t.Fatalf("CreateConnection: store mutated despite protection failure")
	}
	if _, found := store.ProtectedCredential(tenant, "pg-main"); found {
		t.Fatalf("CreateConnection: protected credential stored despite protection failure")
	}
}

func TestEngine_UpdateConnection_EncryptsOnlyWhenPasswordChanges(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	protector := newFakeProtector()
	eng, store := engineWithProtector(t, protector)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}
	// One protect call so far (the create).
	if protector.calls() != 1 {
		t.Fatalf("after create: protector calls = %d, want 1", protector.calls())
	}

	original, found := store.ProtectedCredential(tenant, "pg-main")
	if !found {
		t.Fatalf("expected stored credential after create")
	}

	// Patch a non-secret field only: the protector MUST NOT be called and the
	// stored secret MUST be left intact.
	newHost := "db.replica"
	updated, err := eng.UpdateConnection(ctx, tenant, "pg-main", engine.ConnectionPatch{Host: &newHost})
	if err != nil {
		t.Fatalf("UpdateConnection host-only: unexpected error: %v", err)
	}
	if protector.calls() != 1 {
		t.Fatalf("host-only update: protector calls = %d, want 1 (no re-encrypt)", protector.calls())
	}
	if updated.KeyVersion != protector.keyVersion {
		t.Fatalf("host-only update: KeyVersion = %d, want preserved %d", updated.KeyVersion, protector.keyVersion)
	}

	afterHost, found := store.ProtectedCredential(tenant, "pg-main")
	if !found {
		t.Fatalf("host-only update: stored credential disappeared")
	}
	if string(afterHost.Ciphertext) != string(original.Ciphertext) {
		t.Fatalf("host-only update: stored secret changed: got %q, want %q", afterHost.Ciphertext, original.Ciphertext)
	}

	// Now supply a new password: the protector MUST be called and the stored
	// ciphertext MUST change.
	patch := engine.ConnectionPatch{}.WithPassword("rotated-secret")
	rotated, err := eng.UpdateConnection(ctx, tenant, "pg-main", patch)
	if err != nil {
		t.Fatalf("UpdateConnection password: unexpected error: %v", err)
	}
	if protector.calls() != 2 {
		t.Fatalf("password update: protector calls = %d, want 2", protector.calls())
	}
	if containsSecret(t, rotated, "rotated-secret") {
		t.Fatalf("password update: descriptor leaked rotated secret: %+v", rotated)
	}

	afterRotate, found := store.ProtectedCredential(tenant, "pg-main")
	if !found {
		t.Fatalf("password update: stored credential disappeared")
	}
	if string(afterRotate.Ciphertext) == string(original.Ciphertext) {
		t.Fatalf("password update: stored ciphertext unchanged after rotation")
	}
	if string(afterRotate.Ciphertext) == "rotated-secret" {
		t.Fatalf("password update: stored credential equals plaintext (not protected)")
	}
}

func TestEngine_UpdateConnection_ProtectorErrorIsSafeAndNoMutation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	protector := newFakeProtector()
	eng, store := engineWithProtector(t, protector)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, newInput("pg-main")); err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}
	original, _ := store.ProtectedCredential(tenant, "pg-main")

	// Arm the protector to fail on the next call, then attempt a password update.
	protector.mu.Lock()
	protector.errOnProtect = errors.New("kms down: rotated-secret")
	protector.mu.Unlock()

	patch := engine.ConnectionPatch{Host: strPtr("db.replica")}.WithPassword("rotated-secret")
	_, err := eng.UpdateConnection(ctx, tenant, "pg-main", patch)
	if err == nil {
		t.Fatalf("UpdateConnection: expected protection error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("UpdateConnection: error type = %T, want *engine.EngineError", err)
	}
	if strings.Contains(engErr.Error(), "rotated-secret") {
		t.Fatalf("UpdateConnection: error leaked secret material: %q", engErr.Error())
	}

	// Atomicity: neither the descriptor (host) nor the stored secret may change.
	current, found, _ := store.FindConnection(ctx, tenant, "pg-main")
	if !found {
		t.Fatalf("UpdateConnection: connection vanished after failed update")
	}
	if current.Host != "db.internal" {
		t.Fatalf("UpdateConnection: descriptor mutated despite protection failure: host = %q", current.Host)
	}
	afterFail, _ := store.ProtectedCredential(tenant, "pg-main")
	if string(afterFail.Ciphertext) != string(original.Ciphertext) {
		t.Fatalf("UpdateConnection: stored secret mutated despite protection failure")
	}
}

func strPtr(s string) *string { return &s }

// TestEngine_ConnectionOps_RejectInvalidTenantScope asserts that EVERY connection
// operation rejects an invalid tenant scope with CategoryValidation, regardless
// of how the TenantContext was built. Because TenantContext.TenantID is an
// exported field, a host can bypass NewTenantContext and hand the Engine either a
// zero-value context (empty TenantID) OR a malformed one (e.g. spaces). The
// operation-level guard must catch both shapes for Create, Get, List, Update, and
// Delete — not just Create.
func TestEngine_ConnectionOps_RejectInvalidTenantScope(t *testing.T) {
	t.Parallel()

	tenants := map[string]engine.TenantContext{
		"empty":     {},
		"malformed": {TenantID: "has spaces"},
	}

	ops := map[string]func(*engine.Engine, context.Context, engine.TenantContext) error{
		"Create": func(e *engine.Engine, ctx context.Context, tn engine.TenantContext) error {
			_, err := e.CreateConnection(ctx, tn, newInput("pg-main"))
			return err
		},
		"Get": func(e *engine.Engine, ctx context.Context, tn engine.TenantContext) error {
			_, err := e.GetConnection(ctx, tn, "pg-main")
			return err
		},
		"List": func(e *engine.Engine, ctx context.Context, tn engine.TenantContext) error {
			_, err := e.ListConnections(ctx, tn)
			return err
		},
		"Update": func(e *engine.Engine, ctx context.Context, tn engine.TenantContext) error {
			host := "x"
			_, err := e.UpdateConnection(ctx, tn, "pg-main", engine.ConnectionPatch{Host: &host})
			return err
		},
		"Delete": func(e *engine.Engine, ctx context.Context, tn engine.TenantContext) error {
			return e.DeleteConnection(ctx, tn, "pg-main")
		},
	}

	for tenantName, tenant := range tenants {
		for opName, op := range ops {
			t.Run(tenantName+"/"+opName, func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				eng, _ := engineWithStore(t)

				err := op(eng, ctx, tenant)
				if err == nil {
					t.Fatalf("%s with %s tenant: expected validation error, got nil", opName, tenantName)
				}

				var engErr *engine.EngineError
				if !errors.As(err, &engErr) {
					t.Fatalf("%s with %s tenant: error type = %T, want *engine.EngineError", opName, tenantName, err)
				}
				if engErr.Category != engine.CategoryValidation {
					t.Fatalf("%s with %s tenant: category = %q, want %q", opName, tenantName, engErr.Category, engine.CategoryValidation)
				}
			})
		}
	}
}
