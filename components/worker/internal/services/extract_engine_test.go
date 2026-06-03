// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package services

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
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

func TestMapJobToExtractionRequest_NormalizesTableKeysPerType(t *testing.T) {
	t.Parallel()

	message := ExtractExternalDataMessage{
		MappedFields: map[string]map[string][]string{
			"pg":  {"public.users": {"id"}},
			"ora": {"accounts": {"balance"}}, // oracle: case-folded to UPPERCASE (FIX-2)
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
	// Oracle table AND fields fold to UPPERCASE (FIX-2: legacy case-insensitive parity).
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

	err := uc.extractViaEngine(testContext(), message, connections, result)
	require.Error(t, err)
	require.ErrorContains(t, err, "engine extraction")
}

func TestDecodeDirectResult(t *testing.T) {
	t.Parallel()

	completedAt := engine.ExecutionState{}
	_ = completedAt

	tests := []struct {
		name      string
		res       engine.ExtractionResult
		wantErr   string
		wantRows  bool
		wantEmpty bool
	}{
		{
			name: "direct payload decodes into result",
			res: engine.ExtractionResult{Direct: &engine.DirectResult{
				Data: []byte(`{"pg":{"users":[{"id":1}]}}`),
			}},
			wantRows: true,
		},
		{
			name:      "empty direct payload is a valid empty extraction",
			res:       engine.ExtractionResult{Direct: &engine.DirectResult{Data: nil}},
			wantEmpty: true,
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
		{
			name:    "malformed direct payload errors",
			res:     engine.ExtractionResult{Direct: &engine.DirectResult{Data: []byte(`{not json`)}},
			wantErr: "decode direct engine result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := make(map[string]map[string][]map[string]any)
			err := decodeDirectResult(testContext(), tt.res, result, testLogger())

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
