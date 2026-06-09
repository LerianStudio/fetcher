// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/engine/memory"
)

func TestEngine_GetConnectionByID_FoundScopedAndNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, store := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	desc := seedDescriptor(t, ctx, store, tenant, "pg-main", "id-123")

	got, err := eng.GetConnectionByID(ctx, tenant, "id-123")
	if err != nil {
		t.Fatalf("GetConnectionByID: unexpected error: %v", err)
	}
	if got.ID != "id-123" || got.ConfigName != desc.ConfigName {
		t.Fatalf("GetConnectionByID: got %+v, want id-123/%s", got, desc.ConfigName)
	}

	_, err = eng.GetConnectionByID(ctx, tenant, "missing-id")
	assertNotFound(t, err)
}

// TestEngine_GetConnectionByID_CrossTenantIsolationAndOracle proves tenant A
// cannot read tenant B's connection by ID, and that an unknown ID and a
// cross-tenant ID produce a BYTE-IDENTICAL not-found error (the existence
// oracle).
func TestEngine_GetConnectionByID_CrossTenantIsolationAndOracle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, store := engineWithStore(t)
	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	seedDescriptor(t, ctx, store, tenantB, "prod-db", "id-b")

	_, crossErr := eng.GetConnectionByID(ctx, tenantA, "id-b")
	_, unknownErr := eng.GetConnectionByID(ctx, tenantA, "totally-unknown")

	assertNotFound(t, crossErr)
	assertNotFound(t, unknownErr)

	if crossErr.Error() != unknownErr.Error() {
		t.Fatalf("existence oracle broken: cross-tenant error %q != unknown error %q", crossErr.Error(), unknownErr.Error())
	}
}

func TestEngine_UpdateConnectionByID_PersistsAndScopes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, store := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	current := seedDescriptor(t, ctx, store, tenant, "pg-main", "id-1")

	// Host applies its patch out-of-band; the Engine persists the patched record.
	current.Host = "db.replica"
	updated, err := eng.UpdateConnectionByID(ctx, tenant, "id-1", current, engine.ConnectionPatch{})
	if err != nil {
		t.Fatalf("UpdateConnectionByID: unexpected error: %v", err)
	}
	if updated.Host != "db.replica" {
		t.Fatalf("UpdateConnectionByID: Host = %q, want db.replica", updated.Host)
	}

	got, _, _ := store.FindByID(ctx, tenant, "id-1")
	if got.Host != "db.replica" {
		t.Fatalf("UpdateConnectionByID: store not updated: %+v", got)
	}
}

func TestEngine_UpdateConnectionByID_MissingIsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	_, err := eng.UpdateConnectionByID(ctx, tenant, "ghost", engine.ConnectionDescriptor{ID: "ghost", ConfigName: "x"}, engine.ConnectionPatch{})
	assertNotFound(t, err)
}

// TestEngine_UpdateConnectionByID_CrossTenantCannotMutate proves tenant A cannot
// update tenant B's connection by ID.
func TestEngine_UpdateConnectionByID_CrossTenantCannotMutate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, store := engineWithStore(t)
	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	desc := seedDescriptor(t, ctx, store, tenantB, "prod-db", "id-b")

	desc.Host = "evil.host"
	_, err := eng.UpdateConnectionByID(ctx, tenantA, "id-b", desc, engine.ConnectionPatch{})
	assertNotFound(t, err)

	// Tenant B's record is untouched.
	got, _, _ := store.FindByID(ctx, tenantB, "id-b")
	if got.Host == "evil.host" {
		t.Fatalf("cross-tenant update mutated tenant-b's record: %+v", got)
	}
}

// TestEngine_UpdateConnectionByID_ReProtectsOnPasswordChange proves the
// ID-addressed update re-protects the secret ONLY when the patch carries a
// password (encrypted persistence enabled), records the new key version on the
// descriptor, and never leaks the plaintext — preserving the credential-safety
// guarantee on the ID-addressed write path.
func TestEngine_UpdateConnectionByID_ReProtectsOnPasswordChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	protector := newFakeProtector()
	eng, store := engineWithProtector(t, protector)
	tenant := mustTenant(t, "tenant-a")

	// Seed a record carrying an opaque ID through the protector-backed create.
	input := engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName: "pg-main", Type: "postgres", Host: "db.internal", Port: 5432,
		DatabaseName: "ledger", Username: "svc", Password: "s3cr3t", SSLMode: "require",
	})
	desc := engine.DescriptorFromInput(input)
	desc.ID = "id-1"
	if err := store.Create(ctx, tenant, desc, nil); err != nil {
		t.Fatalf("seed create: %v", err)
	}

	// Host-only update (no password): protector must NOT be called.
	desc.Host = "db.replica"
	if _, err := eng.UpdateConnectionByID(ctx, tenant, "id-1", desc, engine.ConnectionPatch{}); err != nil {
		t.Fatalf("UpdateConnectionByID host-only: %v", err)
	}
	if protector.calls() != 0 {
		t.Fatalf("host-only update: protector calls = %d, want 0", protector.calls())
	}

	// Password update: protector MUST be called and the key version stamped.
	patch := engine.ConnectionPatch{}.WithPassword("rotated-secret")
	updated, err := eng.UpdateConnectionByID(ctx, tenant, "id-1", desc, patch)
	if err != nil {
		t.Fatalf("UpdateConnectionByID password: %v", err)
	}
	if protector.calls() != 1 {
		t.Fatalf("password update: protector calls = %d, want 1", protector.calls())
	}
	if updated.KeyVersion != protector.keyVersion {
		t.Fatalf("password update: KeyVersion = %d, want %d", updated.KeyVersion, protector.keyVersion)
	}
	if containsSecret(t, updated, "rotated-secret") {
		t.Fatalf("password update: descriptor leaked rotated secret: %+v", updated)
	}

	cred, found := store.ProtectedCredential(tenant, "pg-main")
	if !found || string(cred.Ciphertext) == "rotated-secret" {
		t.Fatalf("password update: stored credential not protected: found=%v cred=%q", found, cred.Ciphertext)
	}
}

// TestEngine_UpdateConnectionByID_ProtectorErrorIsSafeAndNoMutation proves a
// protection failure on the ID-addressed write returns a safe error (no
// plaintext leak) and leaves the store untouched (atomicity).
func TestEngine_UpdateConnectionByID_ProtectorErrorIsSafeAndNoMutation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	protector := newFakeProtector()
	eng, store := engineWithProtector(t, protector)
	tenant := mustTenant(t, "tenant-a")

	input := engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName: "pg-main", Type: "postgres", Host: "db.internal", Port: 5432,
		DatabaseName: "ledger", Username: "svc", Password: "s3cr3t", SSLMode: "require",
	})
	desc := engine.DescriptorFromInput(input)
	desc.ID = "id-1"
	if err := store.Create(ctx, tenant, desc, nil); err != nil {
		t.Fatalf("seed create: %v", err)
	}
	original, _ := store.ProtectedCredential(tenant, "pg-main")

	protector.mu.Lock()
	protector.errOnProtect = errors.New("kms down: rotated-secret")
	protector.mu.Unlock()

	desc.Host = "db.replica"
	_, err := eng.UpdateConnectionByID(ctx, tenant, "id-1", desc, engine.ConnectionPatch{}.WithPassword("rotated-secret"))
	if err == nil {
		t.Fatalf("UpdateConnectionByID: expected protection error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error type = %T, want *engine.EngineError", err)
	}
	if strings.Contains(engErr.Error(), "rotated-secret") {
		t.Fatalf("error leaked secret material: %q", engErr.Error())
	}

	// Atomicity: descriptor host and stored secret unchanged.
	current, _, _ := store.FindByID(ctx, tenant, "id-1")
	if current.Host != "db.internal" {
		t.Fatalf("descriptor mutated despite protection failure: host = %q", current.Host)
	}
	after, _ := store.ProtectedCredential(tenant, "pg-main")
	if string(after.Ciphertext) != string(original.Ciphertext) {
		t.Fatalf("stored secret mutated despite protection failure")
	}
}

func TestEngine_DeleteConnectionByID_RemovesAndScopes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, store := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	seedDescriptor(t, ctx, store, tenant, "pg-main", "id-1")

	if err := eng.DeleteConnectionByID(ctx, tenant, "id-1"); err != nil {
		t.Fatalf("DeleteConnectionByID: unexpected error: %v", err)
	}

	// Soft-delete semantics: routing by ID must not resurface a deleted record.
	if _, err := eng.GetConnectionByID(ctx, tenant, "id-1"); err == nil {
		t.Fatalf("DeleteConnectionByID: deleted connection still resolvable by ID")
	}
	if _, found, _ := store.FindByID(ctx, tenant, "id-1"); found {
		t.Fatalf("DeleteConnectionByID: deleted connection still found in store")
	}
}

func TestEngine_DeleteConnectionByID_MissingIsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	assertNotFound(t, eng.DeleteConnectionByID(ctx, tenant, "ghost"))
}

// TestEngine_DeleteConnectionByID_CrossTenantCannotDelete proves tenant A cannot
// delete tenant B's connection by ID.
func TestEngine_DeleteConnectionByID_CrossTenantCannotDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, store := engineWithStore(t)
	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	seedDescriptor(t, ctx, store, tenantB, "prod-db", "id-b")

	assertNotFound(t, eng.DeleteConnectionByID(ctx, tenantA, "id-b"))

	if _, found, _ := store.FindByID(ctx, tenantB, "id-b"); !found {
		t.Fatalf("cross-tenant delete removed tenant-b's record")
	}
}

func TestEngine_ListConnectionsPaged_ScopedAndOpaqueParams(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, store := engineWithStore(t)
	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	seedDescriptor(t, ctx, store, tenantA, "alpha", "id-a1")
	seedDescriptor(t, ctx, store, tenantA, "beta", "id-a2")
	seedDescriptor(t, ctx, store, tenantB, "gamma", "id-b1")

	// Opaque params the Engine must NOT interpret.
	page, err := eng.ListConnectionsPaged(ctx, tenantA, engine.ConnectionListParams{Filter: struct{ Page int }{Page: 1}})
	if err != nil {
		t.Fatalf("ListConnectionsPaged: unexpected error: %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("ListConnectionsPaged: tenant-a page = %+v, want 2 items (tenant-b leaked?)", page)
	}
}

func TestEngine_ConnectionByIDOps_RequireStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, err := engine.New(engine.WithConnectorRegistry(memory.NewConnectorRegistry()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.GetConnectionByID(ctx, tenant, "x"); err == nil {
		t.Fatalf("GetConnectionByID without store: expected error")
	}
	if _, err := eng.ListConnectionsPaged(ctx, tenant, engine.ConnectionListParams{}); err == nil {
		t.Fatalf("ListConnectionsPaged without store: expected error")
	}
	if _, err := eng.UpdateConnectionByID(ctx, tenant, "x", engine.ConnectionDescriptor{}, engine.ConnectionPatch{}); err == nil {
		t.Fatalf("UpdateConnectionByID without store: expected error")
	}
	if err := eng.DeleteConnectionByID(ctx, tenant, "x"); err == nil {
		t.Fatalf("DeleteConnectionByID without store: expected error")
	}
}

func TestEngine_ConnectionByIDOps_RejectInvalidTenantScope(t *testing.T) {
	t.Parallel()

	tenants := map[string]engine.TenantContext{
		"empty":     {},
		"malformed": {TenantID: "has spaces"},
	}

	for name, tenant := range tenants {
		tenant := tenant
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			eng, _ := engineWithStore(t)

			if _, err := eng.GetConnectionByID(ctx, tenant, "x"); !isValidationErr(err) {
				t.Fatalf("GetConnectionByID: want validation error, got %v", err)
			}
			if _, err := eng.ListConnectionsPaged(ctx, tenant, engine.ConnectionListParams{}); !isValidationErr(err) {
				t.Fatalf("ListConnectionsPaged: want validation error, got %v", err)
			}
			if _, err := eng.UpdateConnectionByID(ctx, tenant, "x", engine.ConnectionDescriptor{}, engine.ConnectionPatch{}); !isValidationErr(err) {
				t.Fatalf("UpdateConnectionByID: want validation error, got %v", err)
			}
			if err := eng.DeleteConnectionByID(ctx, tenant, "x"); !isValidationErr(err) {
				t.Fatalf("DeleteConnectionByID: want validation error, got %v", err)
			}
		})
	}
}

// seedDescriptor stores a connection with an opaque host ID directly through the
// store and returns the seeded descriptor.
func seedDescriptor(t *testing.T, ctx context.Context, store *memory.ConnectionStore, tenant engine.TenantContext, configName, id string) engine.ConnectionDescriptor {
	t.Helper()

	desc := engine.DescriptorFromInput(engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName:   configName,
		Type:         "postgres",
		Host:         "db.internal",
		Port:         5432,
		DatabaseName: "ledger",
		Username:     "svc",
		SSLMode:      "require",
	}))
	desc.ID = id

	if err := store.Create(ctx, tenant, desc, nil); err != nil {
		t.Fatalf("seedDescriptor: %v", err)
	}

	return desc
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected not-found error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("error type = %T, want *engine.EngineError", err)
	}
	if engErr.Category != engine.CategoryNotFound {
		t.Fatalf("category = %q, want %q", engErr.Category, engine.CategoryNotFound)
	}
}

func isValidationErr(err error) bool {
	var engErr *engine.EngineError

	return errors.As(err, &engErr) && engErr.Category == engine.CategoryValidation
}
