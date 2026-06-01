// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// secretSentinel is a value that must never appear in any normal output
// (String, fmt %v/%+v, JSON) of a credential-bearing contract type.
const secretSentinel = "sup3r-s3cret-passw0rd"

func TestDefaultLimits_AreStable(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()

	tests := []struct {
		name string
		got  int
		want int
	}{
		{"max datasources", limits.MaxDatasources, DefaultMaxDatasources},
		{"max tables per datasource", limits.MaxTablesPerDatasource, DefaultMaxTablesPerDatasource},
		{"max fields per table", limits.MaxFieldsPerTable, DefaultMaxFieldsPerTable},
		{"max concurrency", limits.MaxConcurrency, DefaultMaxConcurrency},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Fatalf("limit %q = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}

	if limits.Timeout != DefaultTimeout {
		t.Fatalf("default timeout = %s, want %s", limits.Timeout, DefaultTimeout)
	}
	if limits.MaxResultBytes != DefaultMaxResultBytes {
		t.Fatalf("default max result bytes = %d, want %d", limits.MaxResultBytes, DefaultMaxResultBytes)
	}
}

func TestDefaultLimits_StableValues(t *testing.T) {
	t.Parallel()

	// These pin the public contract. Changing them is a breaking change and
	// must be a deliberate edit to this test, not an accidental drift.
	if DefaultMaxDatasources != 10 {
		t.Fatalf("DefaultMaxDatasources = %d, want 10", DefaultMaxDatasources)
	}
	if DefaultMaxTablesPerDatasource != 20 {
		t.Fatalf("DefaultMaxTablesPerDatasource = %d, want 20", DefaultMaxTablesPerDatasource)
	}
	if DefaultMaxFieldsPerTable != 50 {
		t.Fatalf("DefaultMaxFieldsPerTable = %d, want 50", DefaultMaxFieldsPerTable)
	}
	if DefaultMaxConcurrency != 4 {
		t.Fatalf("DefaultMaxConcurrency = %d, want 4", DefaultMaxConcurrency)
	}
	if DefaultTimeout != 5*time.Minute {
		t.Fatalf("DefaultTimeout = %s, want 5m", DefaultTimeout)
	}
	if DefaultMaxResultBytes != 256*1024*1024 {
		t.Fatalf("DefaultMaxResultBytes = %d, want 256MiB", DefaultMaxResultBytes)
	}
}

func TestLimits_ZeroValueIsSafe(t *testing.T) {
	t.Parallel()

	var limits Limits

	// Zero-value formatting must not panic.
	_ = fmt.Sprintf("%v %+v", limits, &limits)

	if limits.IsZero() != true {
		t.Fatalf("zero Limits.IsZero() = false, want true")
	}

	// DefaultLimits must not be the zero value.
	if DefaultLimits().IsZero() {
		t.Fatalf("DefaultLimits().IsZero() = true, want false")
	}
}

func TestExecutionStatus_ConstantsAreStableAndValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status ExecutionStatus
		value  string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusCanceled, "canceled"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			if string(tt.status) != tt.value {
				t.Fatalf("status %q = %q, want %q", tt.value, string(tt.status), tt.value)
			}
			if !tt.status.IsValid() {
				t.Fatalf("status %q must be valid", tt.value)
			}
			if !tt.status.IsTerminal() && (tt.status == StatusCompleted || tt.status == StatusFailed || tt.status == StatusCanceled) {
				t.Fatalf("status %q must be terminal", tt.value)
			}
			if tt.status.IsTerminal() && (tt.status == StatusPending || tt.status == StatusRunning) {
				t.Fatalf("status %q must not be terminal", tt.value)
			}
		})
	}
}

func TestExecutionStatus_InvalidIsRejected(t *testing.T) {
	t.Parallel()

	for _, s := range []ExecutionStatus{"", "PENDING", "done", "unknown"} {
		s := s
		t.Run(string(s), func(t *testing.T) {
			t.Parallel()

			if s.IsValid() {
				t.Fatalf("status %q must be invalid", string(s))
			}
		})
	}
}

func TestErrorCategories_AreStableAndValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		category ErrorCategory
		value    string
	}{
		{CategoryValidation, "validation"},
		{CategoryNotFound, "not_found"},
		{CategoryUnauthorized, "unauthorized"},
		{CategoryForbidden, "forbidden"},
		{CategoryLimitExceeded, "limit_exceeded"},
		{CategoryUnavailable, "unavailable"},
		{CategoryTimeout, "timeout"},
		{CategoryInternal, "internal"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			if string(tt.category) != tt.value {
				t.Fatalf("category %q = %q, want %q", tt.value, string(tt.category), tt.value)
			}
			if !tt.category.IsValid() {
				t.Fatalf("category %q must be valid", tt.value)
			}
		})
	}
}

func TestErrorCategory_InvalidIsRejected(t *testing.T) {
	t.Parallel()

	for _, c := range []ErrorCategory{"", "VALIDATION", "boom"} {
		if c.IsValid() {
			t.Fatalf("category %q must be invalid", string(c))
		}
	}
}

func TestEngineError_ZeroValueSafeAndImplementsError(t *testing.T) {
	t.Parallel()

	var zero EngineError

	// Zero-value formatting and Error() must not panic.
	_ = fmt.Sprintf("%v %+v %s", zero, &zero, zero.Error())

	err := NewEngineError(CategoryValidation, "field is required")
	if err.Category != CategoryValidation {
		t.Fatalf("NewEngineError category = %q, want %q", err.Category, CategoryValidation)
	}
	if !strings.Contains(err.Error(), "field is required") {
		t.Fatalf("EngineError.Error() = %q, want message included", err.Error())
	}
	if !strings.Contains(err.Error(), string(CategoryValidation)) {
		t.Fatalf("EngineError.Error() = %q, want category included", err.Error())
	}
}

func TestEngineError_DoesNotLeakSecretInMessage(t *testing.T) {
	t.Parallel()

	// An error built from a connection input must never embed the secret.
	input := NewConnectionInput(ConnectionInputParams{
		ConfigName:   "prod-db",
		Type:         "POSTGRESQL",
		Host:         "db.internal",
		Port:         5432,
		DatabaseName: "ledger",
		Username:     "svc",
		Password:     secretSentinel,
	})

	err := NewEngineError(CategoryValidation, fmt.Sprintf("connection %s invalid: %s", input.ConfigName, input))
	assertNoSecret(t, "EngineError.Error", err.Error())
	assertNoSecret(t, "EngineError %v", fmt.Sprintf("%v", err))
	assertNoSecret(t, "EngineError %+v", fmt.Sprintf("%+v", err))
}

func TestTenantContext_ZeroValueSafe(t *testing.T) {
	t.Parallel()

	var tc TenantContext

	_ = fmt.Sprintf("%v %+v", tc, &tc)

	if tc.IsZero() != true {
		t.Fatalf("zero TenantContext.IsZero() = false, want true")
	}
	if tc.IsMultiTenant() {
		t.Fatalf("zero TenantContext.IsMultiTenant() = true, want false")
	}

	// tenantID is the sole isolation boundary. It is trimmed and validated.
	populated, err := NewTenantContext("  tenant-123  ")
	if err != nil {
		t.Fatalf("NewTenantContext(valid): unexpected error: %v", err)
	}
	if populated.IsZero() {
		t.Fatalf("populated TenantContext.IsZero() = true, want false")
	}
	if populated.TenantID != "tenant-123" {
		t.Fatalf("NewTenantContext TenantID = %q, want tenant-123 (trimmed)", populated.TenantID)
	}
	if !populated.IsMultiTenant() {
		t.Fatalf("populated TenantContext.IsMultiTenant() = false, want true (tenant present)")
	}

	withReq := populated.WithRequestID("  req-9  ")
	if withReq.RequestID != "req-9" {
		t.Fatalf("WithRequestID = %q, want req-9 (trimmed)", withReq.RequestID)
	}
	// Original must be unchanged (value semantics).
	if populated.RequestID != "" {
		t.Fatalf("WithRequestID mutated the receiver: %q", populated.RequestID)
	}
}

func TestNewTenantContext_ValidatesTenantIDAsOpaqueValue(t *testing.T) {
	t.Parallel()

	// Valid values mirror lib-commons tenant-manager/core IsValidTenantID:
	// non-empty, <= MaxTenantIDLength, first char ASCII alphanumeric, rest
	// [a-zA-Z0-9_-].
	valid := []string{"t", "tenant-1", "Tenant_9", "0abc", strings.Repeat("a", maxTenantIDLength)}
	for _, id := range valid {
		id := id
		t.Run("valid/"+truncateLabel(id), func(t *testing.T) {
			t.Parallel()

			tc, err := NewTenantContext(id)
			if err != nil {
				t.Fatalf("NewTenantContext(%q): unexpected error: %v", truncateLabel(id), err)
			}
			if tc.TenantID != id {
				t.Fatalf("NewTenantContext(%q): TenantID = %q, want unchanged", truncateLabel(id), tc.TenantID)
			}
		})
	}

	invalid := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"leading hyphen", "-tenant"},
		{"leading underscore", "_tenant"},
		{"embedded space", "ten ant"},
		{"control char", "tenant\x00id"},
		{"tab", "tenant\tid"},
		{"slash", "tenant/id"},
		{"too long", strings.Repeat("a", maxTenantIDLength+1)},
	}
	for _, tt := range invalid {
		tt := tt
		t.Run("invalid/"+tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewTenantContext(tt.id)
			if err == nil {
				t.Fatalf("NewTenantContext(%q): expected validation error, got nil", truncateLabel(tt.id))
			}

			var engErr *EngineError
			if !errors.As(err, &engErr) {
				t.Fatalf("NewTenantContext(%q): error type = %T, want *EngineError", truncateLabel(tt.id), err)
			}
			if engErr.Category != CategoryValidation {
				t.Fatalf("NewTenantContext(%q): category = %q, want %q", truncateLabel(tt.id), engErr.Category, CategoryValidation)
			}
		})
	}
}

func truncateLabel(s string) string {
	if len(s) > 16 {
		return s[:16] + "..."
	}

	return s
}

func TestConnectionInput_DoesNotLeakSecret(t *testing.T) {
	t.Parallel()

	input := NewConnectionInput(ConnectionInputParams{
		ConfigName:   "prod-db",
		Type:         "POSTGRESQL",
		Host:         "db.internal",
		Port:         5432,
		DatabaseName: "ledger",
		Username:     "svc",
		Password:     secretSentinel,
	})

	// Password must be retrievable through the explicit accessor only.
	if input.Password() != secretSentinel {
		t.Fatalf("ConnectionInput.Password() = %q, want secret", input.Password())
	}

	assertNoSecret(t, "ConnectionInput.String", input.String())
	assertNoSecret(t, "ConnectionInput %v", fmt.Sprintf("%v", input))
	assertNoSecret(t, "ConnectionInput %+v", fmt.Sprintf("%+v", input))
	assertNoSecret(t, "ConnectionInput %#v", fmt.Sprintf("%#v", input))

	marshaled, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("json.Marshal(ConnectionInput): %v", err)
	}
	assertNoSecret(t, "ConnectionInput JSON", string(marshaled))

	// Pointer marshaling must redact too.
	marshaledPtr, err := json.Marshal(&input)
	if err != nil {
		t.Fatalf("json.Marshal(&ConnectionInput): %v", err)
	}
	assertNoSecret(t, "ConnectionInput pointer JSON", string(marshaledPtr))

	if !input.HasPassword() {
		t.Fatalf("ConnectionInput.HasPassword() = false, want true")
	}

	// The descriptor projection must carry the safe fields and no secret.
	desc := DescriptorFromInput(input)
	if desc.ConfigName != "prod-db" || desc.Host != "db.internal" || desc.Port != 5432 {
		t.Fatalf("DescriptorFromInput dropped safe fields: %+v", desc)
	}
	descJSON, err := json.Marshal(desc)
	if err != nil {
		t.Fatalf("json.Marshal(ConnectionDescriptor): %v", err)
	}
	assertNoSecret(t, "ConnectionDescriptor JSON", string(descJSON))
}

func TestConnectionInput_NoPasswordRedactsToEmpty(t *testing.T) {
	t.Parallel()

	input := NewConnectionInput(ConnectionInputParams{ConfigName: "no-secret", Type: "MYSQL"})

	if input.HasPassword() {
		t.Fatalf("ConnectionInput.HasPassword() = true, want false")
	}
	if strings.Contains(input.String(), redactedMarker) {
		t.Fatalf("empty password must not emit redaction marker: %q", input.String())
	}

	marshaled, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("json.Marshal(ConnectionInput): %v", err)
	}
	if strings.Contains(string(marshaled), redactedMarker) {
		t.Fatalf("empty password must not emit redaction marker in JSON: %q", marshaled)
	}
}

func TestEngineError_NilReceiverIsSafe(t *testing.T) {
	t.Parallel()

	var err *EngineError
	if err.Error() != "" {
		t.Fatalf("nil EngineError.Error() = %q, want empty", err.Error())
	}

	// Missing category must default to internal without panicking.
	got := (&EngineError{Message: "boom"}).Error()
	if !strings.Contains(got, string(CategoryInternal)) {
		t.Fatalf("EngineError with empty category = %q, want internal default", got)
	}
}

func TestSchemaSnapshot_PopulatedBehavior(t *testing.T) {
	t.Parallel()

	snap := SchemaSnapshot{
		ConfigName: "ledger",
		Tables: []TableSnapshot{
			{Name: "public.accounts", Fields: []string{"id", "balance"}},
			{Name: "public.transactions"},
		},
	}

	if !snap.HasTable("public.accounts") {
		t.Fatalf("HasTable(public.accounts) = false, want true")
	}
	if snap.HasTable("public.missing") {
		t.Fatalf("HasTable(public.missing) = true, want false")
	}

	got := snap.TableNames()
	want := []string{"public.accounts", "public.transactions"}
	if len(got) != len(want) {
		t.Fatalf("TableNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TableNames()[%d] = %q, want %q (sorted)", i, got[i], want[i])
		}
	}
}

func TestConnectionInput_ZeroValueSafe(t *testing.T) {
	t.Parallel()

	var input ConnectionInput

	_ = input.String()
	_ = fmt.Sprintf("%v %+v", input, &input)

	if input.Password() != "" {
		t.Fatalf("zero ConnectionInput.Password() = %q, want empty", input.Password())
	}

	marshaled, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("json.Marshal(zero ConnectionInput): %v", err)
	}
	if len(marshaled) == 0 {
		t.Fatalf("json.Marshal(zero ConnectionInput) produced no output")
	}
}

func TestConnectionDescriptor_ZeroValueSafeAndNoSecretField(t *testing.T) {
	t.Parallel()

	var desc ConnectionDescriptor

	out := fmt.Sprintf("%+v", desc)
	if strings.Contains(strings.ToLower(out), "password") {
		t.Fatalf("ConnectionDescriptor must not carry a password field, got %q", out)
	}
}

func TestSchemaSnapshot_ZeroValueSafe(t *testing.T) {
	t.Parallel()

	var snap SchemaSnapshot

	_ = fmt.Sprintf("%v %+v", snap, &snap)

	if snap.HasTable("anything") {
		t.Fatalf("zero SchemaSnapshot must not report tables")
	}
	if got := snap.TableNames(); len(got) != 0 {
		t.Fatalf("zero SchemaSnapshot.TableNames() = %v, want empty", got)
	}
}

func TestExtractionContracts_ZeroValueSafe(t *testing.T) {
	t.Parallel()

	var (
		req   ExtractionRequest
		plan  ExtractionPlan
		state ExecutionState
		res   ExtractionResult
		ref   ResultReference
	)

	// None of the contract types may panic when formatted in zero state.
	_ = fmt.Sprintf("%v %v %v %v %v", req, plan, state, res, ref)
	_ = fmt.Sprintf("%+v %+v %+v %+v %+v", &req, &plan, &state, &res, &ref)

	if state.Status != "" {
		t.Fatalf("zero ExecutionState.Status = %q, want empty", state.Status)
	}
	if state.Status.IsValid() {
		t.Fatalf("zero ExecutionState.Status must be invalid")
	}
}

func assertNoSecret(t *testing.T, label, output string) {
	t.Helper()

	if strings.Contains(output, secretSentinel) {
		t.Fatalf("%s leaked secret: %q", label, output)
	}
}
