// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
)

// These tests pin the GRACEFUL-DEGRADATION contract for an unmarshalable row.
// A host connector that yields a row carrying a value encoding/json cannot
// marshal (a channel, a func, a cyclic structure) must NOT crash the engine and
// must NOT leak the driver/marshal error across the boundary. It must surface a
// safe *EngineError with Category == CategoryInternal and the documented message
// "failed to serialize extraction result".
//
// Both delivery paths reach a marshal of the row data and must behave the same:
//   - DIRECT mode marshals per step in runStepsParallel's running-size guard
//     (and finally in runDirectExtraction's MarshalIndent); and
//   - STORE mode marshals per row in ndjsonEncoder.appendRow.
//
// The unmarshalable value rides in the ROW VALUE (not a field name), so planning
// — which validates against field NAMES discovered from the schema — passes, and
// the failure surfaces only when the runner drives the cursor and marshals the
// row. That is exactly the seam under test.

const unmarshalableMessage = "failed to serialize extraction result"

// requireInternalSerializeError asserts err is a safe *EngineError of
// CategoryInternal carrying the documented message, and that it never leaks the
// sentinel driver text a real marshal error might embed.
func requireInternalSerializeError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected a graceful serialize error, got nil (the engine must not return a result)")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) {
		t.Fatalf("expected *engine.EngineError, got %T: %v", err, err)
	}

	if engErr.Category != engine.CategoryInternal {
		t.Fatalf("expected CategoryInternal, got %q (%v)", engErr.Category, err)
	}

	if engErr.Message != unmarshalableMessage {
		t.Fatalf("expected message %q, got %q", unmarshalableMessage, engErr.Message)
	}

	// The public Error() string is the safe boundary text only. A real
	// json.Marshal failure mentions the offending type (e.g. "chan int"); the
	// engine must not echo it.
	if strings.Contains(engErr.Error(), "chan") || strings.Contains(engErr.Error(), "json") {
		t.Fatalf("boundary error leaked driver/marshal detail: %q", engErr.Error())
	}
}

// unmarshalableRows is the canned cursor data carrying a channel value, which
// encoding/json cannot serialize.
func unmarshalableRows() map[string][]map[string]any {
	return map[string][]map[string]any{
		"public.users": {
			{"bad": make(chan int)},
		},
	}
}

func TestExecuteExtraction_DirectMode_UnmarshalableRow_GracefulInternal(t *testing.T) {
	t.Parallel()

	h := newRunnerHarness(t)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"bad"}},
		unmarshalableRows())

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"bad"}},
	})

	// Direct mode (no sink wired in newRunnerHarness).
	result, err := h.engine.ExecuteExtraction(context.Background(), plan)

	requireInternalSerializeError(t, err)

	if result.Direct != nil || result.Reference != nil {
		t.Fatalf("a failed serialize must return no result, got %+v", result)
	}
}

func TestExecuteExtraction_StoreMode_UnmarshalableRow_GracefulInternal(t *testing.T) {
	t.Parallel()

	sink := memory.NewResultSink()
	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"bad"}},
		unmarshalableRows())

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"bad"}},
	})
	plan.Mode = engine.ModeStore

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)

	requireInternalSerializeError(t, err)

	if result.Reference != nil || result.Direct != nil {
		t.Fatalf("a failed serialize must return no result, got %+v", result)
	}

	// The aborted store-mode run must finalize NOTHING: the writer is abandoned
	// without Close, so no result is stored.
	if sink.StoredCount() != 0 {
		t.Fatalf("aborted store run must finalize no result, got StoredCount=%d", sink.StoredCount())
	}
}

// TestExecuteExtraction_UnmarshalableRow_NoPanic proves the failure is a returned
// error, not a panic that the recover-per-goroutine machinery in forEachStep /
// streamSteps would have to catch. A panic recovered there would still produce a
// CategoryInternal error, so this test additionally guards that the marshal guard
// returns cleanly: it runs both modes and fails if either panics.
func TestExecuteExtraction_UnmarshalableRow_NoPanic(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, mode engine.ExecutionMode, withSink bool) {
		t.Helper()

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("engine panicked on an unmarshalable row (mode=%v): %v", mode, r)
			}
		}()

		var h *runnerHarness
		if withSink {
			h = newStoreRunnerHarness(t, memory.NewResultSink(), nil)
		} else {
			h = newRunnerHarness(t)
		}

		h.seedSource(t, "pg-main", "postgres",
			map[string][]string{"public.users": {"bad"}},
			unmarshalableRows())

		plan := h.planFor(t, map[string]engine.FieldSelection{
			"pg-main": {"public.users": {"bad"}},
		})
		plan.Mode = mode

		_, err := h.engine.ExecuteExtraction(context.Background(), plan)
		requireInternalSerializeError(t, err)
	}

	t.Run("direct", func(t *testing.T) { run(t, engine.ModeDirect, false) })
	t.Run("store", func(t *testing.T) { run(t, engine.ModeStore, true) })
}
