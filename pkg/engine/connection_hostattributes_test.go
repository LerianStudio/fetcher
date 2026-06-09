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
)

// hostAttrInput builds a ConnectionInput carrying an opaque host payload. The
// payload models the Manager's rich record (ProductName / SSL CA-Cert-Key /
// UUID / metadata / timestamps) that the Engine must CARRY but never interpret.
func hostAttrInput(configName string, host map[string]any) engine.ConnectionInput {
	return engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName:     configName,
		Type:           "postgres",
		Host:           "db.internal",
		Port:           5432,
		DatabaseName:   "ledger",
		Username:       "svc",
		Password:       "s3cr3t",
		SSLMode:        "require",
		HostAttributes: host,
	})
}

// TestEngine_HostAttributes_CarriedThroughCreate proves the opaque host payload
// supplied on the input survives onto the returned descriptor AND into the
// store, without the Engine reading any key from it. This is the seam that lets
// the Manager round-trip its rich model without contract drift.
func TestEngine_HostAttributes_CarriedThroughCreate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, store := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	host := map[string]any{
		"productName": "plugin_crm",
		"id":          "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b",
		"sslCA":       "ca-pem",
		"sslCert":     "cert-pem",
		"sslKey":      "key-pem",
		"metadata":    map[string]any{"source": "plugin_crm"},
	}

	desc, err := eng.CreateConnection(ctx, tenant, hostAttrInput("pg-main", host))
	if err != nil {
		t.Fatalf("CreateConnection: unexpected error: %v", err)
	}

	if desc.HostAttributes["productName"] != "plugin_crm" {
		t.Fatalf("host payload not carried onto returned descriptor: %#v", desc.HostAttributes)
	}

	stored, found, err := store.FindConnection(ctx, tenant, "pg-main")
	if err != nil || !found {
		t.Fatalf("FindConnection: found=%v err=%v", found, err)
	}

	if stored.HostAttributes["sslKey"] != "key-pem" {
		t.Fatalf("host payload not carried into the store: %#v", stored.HostAttributes)
	}
}

// TestEngine_HostAttributes_NotScopingDimension proves the Engine scopes ONLY by
// (tenantID, configName) and never by any host attribute. Two tenants may carry
// different productName host payloads under the SAME config name as isolated
// records, and one tenant's payload never leaks into another's read.
func TestEngine_HostAttributes_NotScopingDimension(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenantA := mustTenant(t, "tenant-a")
	tenantB := mustTenant(t, "tenant-b")

	if _, err := eng.CreateConnection(ctx, tenantA, hostAttrInput("pg-main", map[string]any{"productName": "crm"})); err != nil {
		t.Fatalf("create tenant-a: %v", err)
	}

	// Same config name, different tenant, different host payload: must be a
	// distinct, valid record — proving the Engine does not scope on productName.
	if _, err := eng.CreateConnection(ctx, tenantB, hostAttrInput("pg-main", map[string]any{"productName": "billing"})); err != nil {
		t.Fatalf("create tenant-b same config name must be isolated, got: %v", err)
	}

	got, err := eng.GetConnection(ctx, tenantB, "pg-main")
	if err != nil {
		t.Fatalf("get tenant-b: %v", err)
	}

	if got.HostAttributes["productName"] != "billing" {
		t.Fatalf("cross-tenant host payload leak: got %#v", got.HostAttributes)
	}
}

// TestEngine_HostAttributes_PreservedAcrossUpdate proves a patch that does not
// re-supply host attributes leaves the existing opaque payload intact, while a
// patch that DOES supply a fresh payload replaces it — both without the Engine
// interpreting a single key.
func TestEngine_HostAttributes_PreservedAcrossUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng, _ := engineWithStore(t)
	tenant := mustTenant(t, "tenant-a")

	if _, err := eng.CreateConnection(ctx, tenant, hostAttrInput("pg-main", map[string]any{"id": "uuid-1", "updatedAt": "t0"})); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Patch with NO host attributes: existing payload preserved.
	newPort := 6543
	desc, err := eng.UpdateConnection(ctx, tenant, "pg-main", engine.ConnectionPatch{Port: &newPort})
	if err != nil {
		t.Fatalf("update without host payload: %v", err)
	}

	if desc.HostAttributes["id"] != "uuid-1" {
		t.Fatalf("existing host payload not preserved on patch: %#v", desc.HostAttributes)
	}

	// Patch WITH a fresh host payload (host re-stamped updatedAt): replaced.
	fresh := map[string]any{"id": "uuid-1", "updatedAt": "t1"}
	desc2, err := eng.UpdateConnection(ctx, tenant, "pg-main", engine.ConnectionPatch{HostAttributes: fresh})
	if err != nil {
		t.Fatalf("update with host payload: %v", err)
	}

	if desc2.HostAttributes["updatedAt"] != "t1" {
		t.Fatalf("fresh host payload not applied on patch: %#v", desc2.HostAttributes)
	}
}

// TestEngine_AuthorizeConnectionAccess_ScopeRule proves the read-path scope
// authority rule: a present, well-formed tenant is authorized; a zero-value
// (unscoped) tenant is rejected with CategoryValidation. It performs no I/O and
// needs no ConnectionStore.
func TestEngine_AuthorizeConnectionAccess_ScopeRule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	eng, err := engine.New(engine.WithConnectorRegistry(engineConnectorRegistryStub{}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := eng.AuthorizeConnectionAccess(ctx, mustTenant(t, "tenant-a")); err != nil {
		t.Fatalf("authorize valid tenant: unexpected error: %v", err)
	}

	err = eng.AuthorizeConnectionAccess(ctx, engine.TenantContext{})
	if err == nil {
		t.Fatalf("authorize unscoped tenant: expected validation error, got nil")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryValidation {
		t.Fatalf("authorize unscoped tenant: want CategoryValidation, got %v", err)
	}
}

type engineConnectorRegistryStub struct{}

func (engineConnectorRegistryStub) Connector(string) (engine.ConnectorFactory, bool) {
	return nil, false
}

// TestEngine_HostAttributes_AbsentFromDescriptorJSON proves the opaque host
// payload is host-internal: it never appears in the descriptor's public JSON
// projection, so it cannot drift into the Engine's serialized output contract.
func TestEngine_HostAttributes_AbsentFromDescriptorJSON(t *testing.T) {
	t.Parallel()

	desc := engine.ConnectionDescriptor{
		ConfigName:     "pg-main",
		HostAttributes: map[string]any{"productName": "plugin_crm", "secretish": "do-not-serialize"},
	}

	raw, err := json.Marshal(desc)
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}

	if strings.Contains(string(raw), "productName") || strings.Contains(string(raw), "secretish") {
		t.Fatalf("host payload leaked into descriptor JSON: %s", raw)
	}
}
