// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package services

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// erroringEngineRunner returns a fixed error from RunExtraction, to exercise the
// engine-failure branch of extractViaEngine.
type erroringEngineRunner struct{ err error }

func (r erroringEngineRunner) RunExtraction(
	context.Context,
	engine.TenantContext,
	string,
	engine.ExtractionRequest,
) (engine.ExtractionResult, error) {
	return engine.ExtractionResult{}, r.err
}

// ctxCapturingEngineRunner records the context RunExtraction was called with so a
// test can read back the per-config schema scope the Worker seeded — the same ctx
// the Engine threads into the connector for DiscoverSchema, so asserting the scope
// here proves it reaches the connector. It returns a minimal valid Direct result so
// extractViaEngine completes its success path.
type ctxCapturingEngineRunner struct{ ctx context.Context }

func (r *ctxCapturingEngineRunner) RunExtraction(
	ctx context.Context,
	_ engine.TenantContext,
	_ string,
	_ engine.ExtractionRequest,
) (engine.ExtractionResult, error) {
	r.ctx = ctx
	return engine.ExtractionResult{Direct: &engine.DirectResult{Data: nil}}, nil
}

func TestMapJobToExtractionRequest_NormalizesTableKeysPerType(t *testing.T) {
	t.Parallel()

	message := ExtractExternalDataMessage{
		MappedFields: map[string]map[string][]string{
			"pg":  {"public.users": {"id"}},
			"ora": {"accounts": {"balance"}}, // oracle: case-folded to UPPERCASE-canonical
		},
		Filters: map[string]map[string]map[string]modelJob.FilterCondition{
			"pg": {"public.users": {"status": {Equals: []any{"a"}}}},
		},
		Metadata: map[string]any{"source": "plugin_crm"},
	}

	connections := []*model.Connection{
		{ConfigName: "pg", Type: model.TypePostgreSQL},
		{ConfigName: "ora", Type: model.TypeOracle},
	}

	req := mapJobToExtractionRequest(message, connections)

	// Postgres public-schema table canonicalizes to the unqualified name (case kept).
	if _, ok := req.MappedFields["pg"]["users"]; !ok {
		t.Fatalf("expected pg public.users to normalize to users, got %#v", req.MappedFields["pg"])
	}
	// Oracle table AND fields fold to UPPERCASE (matches the physical Oracle catalog
	// AND the extracted result keys; physical-case request resolution happens later in
	// pkg/oracle.ValidateTableAndFields).
	oraFields, ok := req.MappedFields["ora"]["ACCOUNTS"]
	if !ok {
		t.Fatalf("expected oracle accounts to fold to ACCOUNTS, got %#v", req.MappedFields["ora"])
	}
	if len(oraFields) != 1 || oraFields[0] != "BALANCE" {
		t.Fatalf("expected oracle field folded to BALANCE, got %v", oraFields)
	}

	// FIX-1: filters are emitted as fully-nested map[string]any (config -> table ->
	// field -> FilterCondition-as-any) so they survive the planner's map[string]any
	// assertion. Table key normalized in lockstep with mapped fields.
	pgFilters, ok := req.Filters["pg"].(map[string]any)
	require.True(t, ok, "expected nested map[string]any pg filters, got %T", req.Filters["pg"])
	usersFilters, ok := pgFilters["users"].(map[string]any)
	require.True(t, ok, "expected pg filter table normalized to users (nested any), got %#v", pgFilters)
	cond, ok := usersFilters["status"].(modelJob.FilterCondition)
	require.True(t, ok, "expected status FilterCondition leaf, got %T", usersFilters["status"])
	assert.Equal(t, "a", cond.Equals[0])

	// metadata.source is carried opaque.
	assert.Equal(t, "plugin_crm", req.Metadata["source"])
}

func TestMapJobToExtractionRequest_NilInputs(t *testing.T) {
	t.Parallel()

	req := mapJobToExtractionRequest(ExtractExternalDataMessage{}, nil)
	assert.Nil(t, req.MappedFields)
	assert.Nil(t, req.Filters)
}

func TestExtractViaEngine_RunnerErrorPropagates(t *testing.T) {
	t.Parallel()

	uc := &UseCase{EngineRunner: erroringEngineRunner{err: errors.New("plan rejected: table not found")}}

	message := ExtractExternalDataMessage{
		JobID: newTestJobID(),
		MappedFields: map[string]map[string][]string{
			"pg": {"users": {"id"}},
		},
	}
	connections := []*model.Connection{{ConfigName: "pg", Type: model.TypePostgreSQL}}
	result := make(map[string]map[string][]map[string]any)

	// mergeIntoResult is irrelevant here: the runner errors before any decode.
	_, err := uc.extractViaEngine(testContext(), message, connections, result, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "engine extraction")
}

// TestExtractViaEngine_SeedsSchemaScopeForMultiSchemaRequest is the worker-side
// regression guard for the non-default-schema extraction defect: a generic
// PostgreSQL extraction requesting schema-qualified tables MUST seed the schemas
// those tables reference into the context the Engine receives, so discovery covers
// "accounting"/"reporting" instead of narrowing to "public" and failing schema
// validation. It also proves the mixed qualified+unqualified case still injects the
// "public" default, and that a non-narrowing type (Oracle) is left unseeded.
func TestExtractViaEngine_SeedsSchemaScopeForMultiSchemaRequest(t *testing.T) {
	t.Parallel()

	runner := &ctxCapturingEngineRunner{}
	uc := &UseCase{EngineRunner: runner}

	message := ExtractExternalDataMessage{
		JobID: newTestJobID(),
		MappedFields: map[string]map[string][]string{
			"pg":  {"accounting.invoices": {"id"}, "reporting.daily_summary": {"id"}, "audit_log": {"id"}},
			"ora": {"HR.EMPLOYEES": {"ID"}},
		},
	}
	connections := []*model.Connection{
		{ConfigName: "pg", Type: model.TypePostgreSQL},
		{ConfigName: "ora", Type: model.TypeOracle},
	}
	result := make(map[string]map[string][]map[string]any)

	_, err := uc.extractViaEngine(testContext(), message, connections, result, false)
	require.NoError(t, err)
	require.NotNil(t, runner.ctx, "runner must have been invoked")

	// The PG scope carries both qualified schemas PLUS the injected "public" default
	// (because "audit_log" is unqualified), mirroring the legacy ensureDefaultSchema.
	pgScope := schemacompat.SchemaScope(runner.ctx, "pg")
	sort.Strings(pgScope)
	assert.Equal(t, []string{"accounting", "public", "reporting"}, pgScope)

	// Oracle has no strippable default schema, so only the explicitly-qualified owner
	// scopes discovery (HR); no default is injected.
	oraScope := schemacompat.SchemaScope(runner.ctx, "ora")
	assert.Equal(t, []string{"HR"}, oraScope)
}

// TestValidateDirectResult covers the result-shape contract that gates the direct
// path: exactly one of Direct / Reference must be non-nil. A store-mode reference or
// a neither-arm result is a contract violation; a Direct arm (even with an empty
// payload) is valid and returned for the caller to consume.
func TestValidateDirectResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		res     engine.ExtractionResult
		wantErr string
	}{
		{
			name: "direct arm is valid",
			res:  engine.ExtractionResult{Direct: &engine.DirectResult{Data: []byte(`{"pg":{}}`)}},
		},
		{
			name: "empty direct arm is valid",
			res:  engine.ExtractionResult{Direct: &engine.DirectResult{Data: nil}},
		},
		{
			name:    "store-mode reference is a contract violation in direct path",
			res:     engine.ExtractionResult{Reference: &engine.ResultReference{Path: "tenant/x.json"}},
			wantErr: "store-mode reference",
		},
		{
			name:    "neither arm set is a contract violation",
			res:     engine.ExtractionResult{},
			wantErr: "no direct result payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			direct, err := validateDirectResult(tt.res)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, direct)
		})
	}
}

// TestDecodeDirectResult covers the mixed-job merge: a DirectResult payload is decoded
// and merged PER-TABLE into the result map (never a whole-map clobber). An empty
// payload leaves the map untouched; malformed JSON errors.
func TestDecodeDirectResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		direct    *engine.DirectResult
		wantErr   string
		wantRows  bool
		wantEmpty bool
	}{
		{
			name:     "direct payload decodes into result",
			direct:   &engine.DirectResult{Data: []byte(`{"pg":{"users":[{"id":1}]}}`)},
			wantRows: true,
		},
		{
			name:      "empty direct payload is a valid empty extraction",
			direct:    &engine.DirectResult{Data: nil},
			wantEmpty: true,
		},
		{
			name:    "malformed direct payload errors",
			direct:  &engine.DirectResult{Data: []byte(`{not json`)},
			wantErr: "decode direct engine result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := make(map[string]map[string][]map[string]any)
			err := decodeDirectResult(testContext(), tt.direct, result, testLogger())

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			if tt.wantRows {
				assert.Len(t, result["pg"]["users"], 1)
			}
			if tt.wantEmpty {
				assert.Empty(t, result)
			}
		})
	}
}

// TestDecodeDirectResult_PerTableMergePreservesExisting proves the bundled merge fix:
// decoding generic rows into a result map that the CRM path already populated keeps the
// existing config's tables and ADDS the new ones, rather than clobbering result[db]
// wholesale. This is the mixed-job invariant.
func TestDecodeDirectResult_PerTableMergePreservesExisting(t *testing.T) {
	t.Parallel()

	result := map[string]map[string][]map[string]any{
		"db": {
			"crm_table": {{"id": float64(1)}},
		},
	}

	// The engine payload carries a NEW table for the SAME config and a NEW config.
	direct := &engine.DirectResult{Data: []byte(`{"db":{"generic_table":[{"id":2}]},"other":{"t":[{"id":3}]}}`)}

	require.NoError(t, decodeDirectResult(testContext(), direct, result, testLogger()))

	// The pre-existing CRM table survives (no whole-map clobber).
	require.Len(t, result["db"]["crm_table"], 1, "existing CRM table must be preserved")
	// The generic table is merged into the same config.
	require.Len(t, result["db"]["generic_table"], 1, "generic table must be merged into the existing config")
	// A brand-new config is created.
	require.Len(t, result["other"]["t"], 1, "new config must be added")
}
