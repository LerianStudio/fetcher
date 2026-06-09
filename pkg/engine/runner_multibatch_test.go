// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/engine/memory"

	"go.uber.org/goleak"
)

// multiBatchRows builds a row slice large enough that the engine's store-path
// batching (storeBatchRows = 256) flushes batches batches for the step. With
// batches=3 the step emits >2 batches, so a NON-tail producer blocks on its
// second one-ahead send until the writer advances to it — the exact condition
// the ordinal-order writer deadlocked on before per-step channel self-close.
func multiBatchRows(seed, batches int) map[string][]map[string]any {
	const perBatch = 300 // > storeBatchRows (256), so each 300 rows is ≥2 batches

	n := batches * perBatch
	rows := make([]map[string]any, n)

	for i := 0; i < n; i++ {
		rows[i] = map[string]any{
			"id":   seed*1_000_000 + i,
			"blob": "multi-batch-payload-" + strconv.Itoa(seed) + "-" + strconv.Itoa(i),
		}
	}

	return map[string][]map[string]any{"public.t": rows}
}

// TestExecuteExtraction_StoreModeMultiStepMultiBatch is the deadlock regression:
// N >= 2 steps that EACH emit several batches. Before per-step channel self-close
// this HANGS — the ordinal-order writer parks on channel 0 (never closed until
// the whole pool joins) while step 1's producer blocks sending its second batch,
// a cyclic wait. It must now complete, produce deterministic NDJSON + digest
// across runs, and show incremental writes. Verified at MaxConcurrency = N AND 1.
func TestExecuteExtraction_StoreModeMultiStepMultiBatch(t *testing.T) {
	t.Parallel()

	const (
		nSteps  = 3
		batches = 3
	)

	for _, concurrency := range []int{nSteps, 1} {
		concurrency := concurrency
		t.Run("concurrency="+strconv.Itoa(concurrency), func(t *testing.T) {
			t.Parallel()

			run := func() (*memory.ResultSink, engine.ExtractionResult) {
				limits := engine.DefaultLimits()
				limits.MaxConcurrency = concurrency
				// Generous budget: this test is about completion + determinism, not
				// the size limit.
				limits.MaxResultBytes = 0

				sink := memory.NewResultSink()
				h := newLimitedRunnerHarness(t, limits, sink)

				mapped := make(map[string]engine.FieldSelection, nSteps)
				for i := 0; i < nSteps; i++ {
					name := "src-" + strconv.Itoa(i)
					h.seedSource(t, name, distinctDBTypes[i],
						map[string][]string{"public.t": {"id", "blob"}},
						multiBatchRows(i, batches))
					mapped[name] = engine.FieldSelection{"public.t": {"id", "blob"}}
				}

				plan := h.planFor(t, mapped)
				plan.Mode = engine.ModeStore

				result, err := h.engine.ExecuteExtraction(context.Background(), plan)
				if err != nil {
					t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
				}

				if result.Reference == nil {
					t.Fatalf("store mode must return a reference")
				}

				return sink, result
			}

			firstSink, firstResult := run()

			wantBytes, ok := firstSink.Get(firstResult.Reference.Path)
			if !ok {
				t.Fatalf("first run stored no payload")
			}

			wantDigest := firstResult.Reference.Integrity.Digest

			// Incremental flushing: many batches across many steps => many writes.
			if writes := firstSink.WriteSizes(); len(writes) < nSteps*batches {
				t.Fatalf("expected at least %d incremental writes, got %d", nSteps*batches, len(writes))
			}

			// Row total: each step emits batches*300 rows.
			wantRows := int64(nSteps * batches * 300)
			if firstResult.Reference.RowCount != wantRows {
				t.Fatalf("RowCount = %d, want %d", firstResult.Reference.RowCount, wantRows)
			}

			// Determinism: repeated runs produce byte-identical NDJSON + digest.
			for r := 0; r < 5; r++ {
				sink, result := run()

				gotBytes, ok := sink.Get(result.Reference.Path)
				if !ok {
					t.Fatalf("run %d stored no payload", r)
				}

				if string(gotBytes) != string(wantBytes) {
					t.Fatalf("run %d NDJSON bytes differ from run 0 (len %d vs %d)", r, len(gotBytes), len(wantBytes))
				}

				if result.Reference.Integrity.Digest != wantDigest {
					t.Fatalf("run %d digest = %q, want %q", r, result.Reference.Integrity.Digest, wantDigest)
				}
			}

			// NDJSON is ordered by Ordinal: the producing config name is
			// non-decreasing across the whole stream (all of src-0 before src-1, ...).
			lines := decodeNDJSON(t, wantBytes)
			if len(lines) != int(wantRows) {
				t.Fatalf("decoded %d NDJSON lines, want %d", len(lines), wantRows)
			}

			last := ""
			for i, line := range lines {
				if line.Config < last {
					t.Fatalf("line[%d] config %q precedes earlier config %q; ordinal order violated", i, line.Config, last)
				}

				last = line.Config
			}
		})
	}
}

// TestExecuteExtraction_StoreModeMultiStepMultiBatchNoLeak proves the multi-step
// multi-batch store path leaks no goroutine: the writer and every producer exit
// cleanly once the result is finalized. NOT parallel so goleak observes a quiet
// baseline.
func TestExecuteExtraction_StoreModeMultiStepMultiBatchNoLeak(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		nSteps  = 3
		batches = 3
	)

	limits := engine.DefaultLimits()
	limits.MaxConcurrency = nSteps
	limits.MaxResultBytes = 0

	sink := memory.NewResultSink()
	h := newLimitedRunnerHarness(t, limits, sink)

	mapped := make(map[string]engine.FieldSelection, nSteps)
	for i := 0; i < nSteps; i++ {
		name := "src-" + strconv.Itoa(i)
		h.seedSource(t, name, distinctDBTypes[i],
			map[string][]string{"public.t": {"id", "blob"}},
			multiBatchRows(i, batches))
		mapped[name] = engine.FieldSelection{"public.t": {"id", "blob"}}
	}

	plan := h.planFor(t, mapped)
	plan.Mode = engine.ModeStore

	if _, err := h.engine.ExecuteExtraction(context.Background(), plan); err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}
}

// TestExecuteExtraction_StoreModeMultiBatchBudgetFailFast proves a budget breach
// PARTWAY through a multi-step multi-batch run fails fast with
// CategoryLimitExceeded, returns no reference, finalizes nothing, and leaks no
// goroutine (all producers unwind on the writer's cancel, every channel closes).
func TestExecuteExtraction_StoreModeMultiBatchBudgetFailFast(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		nSteps  = 3
		batches = 4
	)

	limits := engine.DefaultLimits()
	limits.MaxConcurrency = nSteps
	// A budget far below the full stream, but above a few batches, so the breach
	// happens MID-stream rather than on the first write.
	limits.MaxResultBytes = 6000

	sink := memory.NewResultSink()
	h := newLimitedRunnerHarness(t, limits, sink)

	mapped := make(map[string]engine.FieldSelection, nSteps)
	for i := 0; i < nSteps; i++ {
		name := "src-" + strconv.Itoa(i)
		h.seedSource(t, name, distinctDBTypes[i],
			map[string][]string{"public.t": {"id", "blob"}},
			multiBatchRows(i, batches))
		mapped[name] = engine.FieldSelection{"public.t": {"id", "blob"}}
	}

	plan := h.planFor(t, mapped)
	plan.Mode = engine.ModeStore

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("an over-budget multi-batch stream must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryLimitExceeded {
		t.Fatalf("error = %v, want CategoryLimitExceeded", err)
	}

	if result.Reference != nil {
		t.Fatalf("over-budget run must return no reference, got %+v", result.Reference)
	}

	if sink.StoredCount() != 0 {
		t.Fatalf("over-budget run must finalize nothing, StoredCount = %d", sink.StoredCount())
	}

	assertBuiltAreClosed(t, h, nSteps)
}

// TestExecuteExtraction_StoreModeMultiBatchWriteErrorFailFast proves a sink Write
// error PARTWAY through a multi-step multi-batch run fails fast with
// CategoryUnavailable, returns no reference, and leaks no goroutine.
func TestExecuteExtraction_StoreModeMultiBatchWriteErrorFailFast(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		nSteps  = 3
		batches = 4
	)

	limits := engine.DefaultLimits()
	limits.MaxConcurrency = nSteps
	limits.MaxResultBytes = 0

	sink := memory.NewResultSink()
	// Fail the streaming write: the writer trips on its first Write, cancels the
	// run, and every producer must unwind through flushBatch's cancel arm.
	sink.WriteErr = errors.New("sink write exploded")

	h := newLimitedRunnerHarness(t, limits, sink)

	mapped := make(map[string]engine.FieldSelection, nSteps)
	for i := 0; i < nSteps; i++ {
		name := "src-" + strconv.Itoa(i)
		h.seedSource(t, name, distinctDBTypes[i],
			map[string][]string{"public.t": {"id", "blob"}},
			multiBatchRows(i, batches))
		mapped[name] = engine.FieldSelection{"public.t": {"id", "blob"}}
	}

	plan := h.planFor(t, mapped)
	plan.Mode = engine.ModeStore

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("a sink write error must fail the run, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("error = %v, want CategoryUnavailable", err)
	}

	if result.Reference != nil {
		t.Fatalf("a write-error run must return no reference, got %+v", result.Reference)
	}

	if sink.StoredCount() != 0 {
		t.Fatalf("a write-error run must finalize nothing, StoredCount = %d", sink.StoredCount())
	}

	assertBuiltAreClosed(t, h, nSteps)
}
