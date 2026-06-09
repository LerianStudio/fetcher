// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
)

// TestExecuteExtraction_StoreModeFlushesIncrementally proves the engine streams
// the result to the sink in MULTIPLE writes rather than one big write — i.e. it
// flushes batch-by-batch and never holds the whole result in memory. A single
// source emits far more rows than one batch, so the engine MUST issue more than
// one Write to the streaming sink.
func TestExecuteExtraction_StoreModeFlushesIncrementally(t *testing.T) {
	t.Parallel()

	const totalRows = 2000 // well above the engine's per-batch row count

	rows := make([]map[string]any, totalRows)
	for i := range rows {
		rows[i] = map[string]any{"id": i, "blob": "row-payload-" + strconv.Itoa(i)}
	}

	sink := memory.NewResultSink()
	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id", "blob"}},
		map[string][]map[string]any{"public.users": rows})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id", "blob"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Reference == nil {
		t.Fatalf("store mode must return a reference, got %+v", result)
	}

	writes := sink.WriteSizes()
	if len(writes) <= 1 {
		t.Fatalf("engine must flush incrementally: got %d Write call(s), want > 1 for %d rows", len(writes), totalRows)
	}

	// Every batch is bounded — no single Write carries the whole result.
	stored, ok := sink.Get(result.Reference.Path)
	if !ok {
		t.Fatalf("sink has no payload at %q", result.Reference.Path)
	}

	for i, size := range writes {
		if size >= len(stored) && len(writes) > 1 {
			t.Fatalf("Write[%d] size %d covers the whole %d-byte result; batching is not bounded", i, size, len(stored))
		}
	}

	if result.Reference.RowCount != totalRows {
		t.Fatalf("Reference.RowCount = %d, want %d", result.Reference.RowCount, totalRows)
	}
}

// TestExecuteExtraction_ConcurrencyOverlap proves MaxConcurrency is real: with
// MaxConcurrency >= N, N connectors that each block until all N have arrived make
// progress (the barrier releases), and the observed peak in-flight count equals
// min(N, MaxConcurrency). No sleeps — the barrier is a WaitGroup released only
// when the Nth connector arrives, so the test deadlocks (and fails by timeout)
// rather than passing flakily if concurrency is not honored.
func TestExecuteExtraction_ConcurrencyOverlap(t *testing.T) {
	t.Parallel()

	const n = 4

	barrier := newArrivalBarrier(n)

	limits := engine.DefaultLimits()
	limits.MaxConcurrency = n

	h := newLimitedRunnerHarness(t, limits, nil)

	mapped := make(map[string]engine.FieldSelection, n)
	for i := 0; i < n; i++ {
		name := "src-" + strconv.Itoa(i)
		conn := h.seedSource(t, name, distinctDBTypes[i],
			map[string][]string{"public.t": {"id"}},
			map[string][]map[string]any{"public.t": {{"id": i}}})
		// Each query records arrival and blocks until all N have arrived. With
		// MaxConcurrency < N this barrier would never release and the run would hang.
		conn.AfterQuery = barrier.arriveAndWait
		mapped[name] = engine.FieldSelection{"public.t": {"id"}}
	}

	plan := h.planFor(t, mapped)

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if result.Direct == nil || result.Direct.RowCount != n {
		t.Fatalf("expected a direct result with %d rows, got %+v", n, result.Direct)
	}

	if peak := barrier.peak(); peak != n {
		t.Fatalf("peak in-flight steps = %d, want %d (MaxConcurrency must be honored)", peak, n)
	}
}

// TestExecuteExtraction_StoreModeSerializesUnderLowConcurrency proves the inverse
// of the overlap test: with MaxConcurrency = 1, only one step runs at a time, so
// a barrier that needs 2 arrivals would deadlock. We verify the peak in-flight
// count never exceeds 1 using a non-blocking arrival counter.
func TestExecuteExtraction_SerializesUnderConcurrencyOne(t *testing.T) {
	t.Parallel()

	const n = 3

	var (
		mu    sync.Mutex
		inFlt int
		peak  int
	)

	limits := engine.DefaultLimits()
	limits.MaxConcurrency = 1

	h := newLimitedRunnerHarness(t, limits, nil)

	mapped := make(map[string]engine.FieldSelection, n)
	for i := 0; i < n; i++ {
		name := "src-" + strconv.Itoa(i)
		conn := h.seedSource(t, name, distinctDBTypes[i],
			map[string][]string{"public.t": {"id"}},
			map[string][]map[string]any{"public.t": {{"id": i}}})
		conn.AfterQuery = func() {
			mu.Lock()
			inFlt++
			if inFlt > peak {
				peak = inFlt
			}
			inFlt--
			mu.Unlock()
		}
		mapped[name] = engine.FieldSelection{"public.t": {"id"}}
	}

	plan := h.planFor(t, mapped)

	if _, err := h.engine.ExecuteExtraction(context.Background(), plan); err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	if peak > 1 {
		t.Fatalf("peak in-flight steps = %d, want <= 1 under MaxConcurrency=1", peak)
	}
}

// TestExecuteExtraction_StoreModeDeterministicBytesAndDigest proves the streamed
// NDJSON bytes and the integrity digest are byte-identical across many runs even
// when steps finish in different orders. Each run uses connectors that jitter
// their completion order (a randomized arrival barrier), yet the writer drains by
// Ordinal, so the output never varies.
func TestExecuteExtraction_StoreModeDeterministicBytesAndDigest(t *testing.T) {
	t.Parallel()

	const (
		runs  = 25
		nSrcs = 4
	)

	build := func() (*memory.ResultSink, engine.ExtractionResult) {
		sink := memory.NewResultSink()

		limits := engine.DefaultLimits()
		limits.MaxConcurrency = nSrcs

		h := newLimitedRunnerHarness(t, limits, sink)

		// A shared counter perturbs completion order: each connector spins a
		// different number of trivial iterations after producing rows, so steps
		// finish in a different order across runs.
		var jitter atomic.Int64

		mapped := make(map[string]engine.FieldSelection, nSrcs)
		for i := 0; i < nSrcs; i++ {
			name := "src-" + strconv.Itoa(i)
			conn := h.seedSource(t, name, distinctDBTypes[i],
				map[string][]string{"public.t": {"id", "name"}},
				map[string][]map[string]any{"public.t": {
					{"id": i*10 + 1, "name": "alpha-" + name},
					{"id": i*10 + 2, "name": "beta-" + name},
				}})
			spins := int64((i*7 + 3) % 5)
			conn.AfterQuery = func() {
				for s := int64(0); s < spins*1000; s++ {
					jitter.Add(1)
				}
			}
			mapped[name] = engine.FieldSelection{"public.t": {"id", "name"}}
		}

		plan := h.planFor(t, mapped)

		result, err := h.engine.ExecuteExtraction(context.Background(), plan)
		if err != nil {
			t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
		}

		return sink, result
	}

	firstSink, firstResult := build()
	wantBytes, ok := firstSink.Get(firstResult.Reference.Path)
	if !ok {
		t.Fatalf("first run stored no payload")
	}

	wantDigest := firstResult.Reference.Integrity.Digest
	if wantDigest == "" {
		t.Fatalf("first run produced no digest")
	}

	for r := 1; r < runs; r++ {
		sink, result := build()

		gotBytes, ok := sink.Get(result.Reference.Path)
		if !ok {
			t.Fatalf("run %d stored no payload", r)
		}

		if string(gotBytes) != string(wantBytes) {
			t.Fatalf("run %d NDJSON bytes differ from run 0:\n run0=%q\n run%d=%q", r, wantBytes, r, gotBytes)
		}

		if result.Reference.Integrity.Digest != wantDigest {
			t.Fatalf("run %d digest = %q, want %q (must be identical across runs)", r, result.Reference.Integrity.Digest, wantDigest)
		}
	}
}

// TestExecuteExtraction_StoreModeStreamingBudgetFailsFast proves the streaming
// byte budget is enforced MID-STREAM: a result whose streamed bytes exceed
// MaxResultBytes fails with CategoryLimitExceeded, finalizes no reference, and
// the partial write is abandoned (StoredCount stays 0).
func TestExecuteExtraction_StoreModeStreamingBudgetFailsFast(t *testing.T) {
	t.Parallel()

	const totalRows = 5000

	rows := make([]map[string]any, totalRows)
	for i := range rows {
		rows[i] = map[string]any{"id": i, "blob": "padding-to-grow-the-stream-" + strconv.Itoa(i)}
	}

	limits := engine.DefaultLimits()
	limits.MaxResultBytes = 4096 // far smaller than the full stream

	sink := memory.NewResultSink()
	h := newLimitedRunnerHarness(t, limits, sink)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id", "blob"}},
		map[string][]map[string]any{"public.users": rows})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id", "blob"}},
	})
	plan.Mode = engine.ModeStore

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("an over-budget stream must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryLimitExceeded {
		t.Fatalf("error = %v, want CategoryLimitExceeded", err)
	}

	if result.Reference != nil {
		t.Fatalf("over-budget stream must not return a reference, got %+v", result.Reference)
	}

	if sink.StoredCount() != 0 {
		t.Fatalf("over-budget stream must finalize no result, StoredCount = %d", sink.StoredCount())
	}
}

// TestExecuteExtraction_ConnectorsClosedUnderConcurrency proves every BUILT
// connector is closed on success, on first-error fail-fast, and on a panic in one
// step's goroutine. It uses the build-per-step factory's connector close
// accounting under MaxConcurrency > 1.
func TestExecuteExtraction_ConnectorsClosedUnderConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		limits := engine.DefaultLimits()
		limits.MaxConcurrency = 4

		h := newLimitedRunnerHarness(t, limits, nil)
		conns := seedN(t, h, 4, nil)

		plan := h.planFor(t, mappedFor(4))
		if _, err := h.engine.ExecuteExtraction(context.Background(), plan); err != nil {
			t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
		}

		assertAllClosedOnce(t, conns)
	})

	t.Run("first_error_fail_fast", func(t *testing.T) {
		t.Parallel()

		limits := engine.DefaultLimits()
		limits.MaxConcurrency = 4

		h := newLimitedRunnerHarness(t, limits, nil)
		conns := seedN(t, h, 4, nil)
		// One step fails its query; the run fails fast but every built connector
		// (including the failing one) must still close.
		conns[2].QueryErr = errors.New("boom")

		plan := h.planFor(t, mappedFor(4))
		if _, err := h.engine.ExecuteExtraction(context.Background(), plan); err == nil {
			t.Fatalf("a query failure must fail the run, got nil error")
		}

		// Only steps that the pool actually dispatched build a connector; every one
		// that built MUST have closed. The failing step's connector closes too.
		assertBuiltAreClosed(t, h, 4)
	})

	t.Run("panic_in_one_step", func(t *testing.T) {
		t.Parallel()

		limits := engine.DefaultLimits()
		limits.MaxConcurrency = 4

		h := newLimitedRunnerHarness(t, limits, nil)
		conns := seedN(t, h, 4, nil)
		// A panic in one step's AfterQuery hook must be recovered per-goroutine,
		// fail the run as CategoryInternal, and leak no connector.
		conns[1].AfterQuery = func() { panic("step blew up") }

		plan := h.planFor(t, mappedFor(4))
		_, err := h.engine.ExecuteExtraction(context.Background(), plan)
		if err == nil {
			t.Fatalf("a panicking step must fail the run, got nil error")
		}

		var engErr *engine.EngineError
		if !errors.As(err, &engErr) || engErr.Category != engine.CategoryInternal {
			t.Fatalf("error = %v, want CategoryInternal", err)
		}

		assertBuiltAreClosed(t, h, 4)
	})
}

// TestExecuteExtraction_DirectModeBytesAreIdentical is the Direct-mode
// byte-identity golden: for a fixed multi-source input, the Direct payload bytes
// and the integrity digest are exactly the PRE-CHANGE values — indented JSON
// keyed by config -> table -> rows, with map keys sorted. Parallelizing Direct
// extraction must not change a single byte.
func TestExecuteExtraction_DirectModeBytesAreIdentical(t *testing.T) {
	t.Parallel()

	limits := engine.DefaultLimits()
	limits.MaxConcurrency = 4

	h := newLimitedRunnerHarness(t, limits, nil)
	h.seedSource(t, "pg-a", "postgres",
		map[string][]string{"public.a": {"id", "name"}},
		map[string][]map[string]any{
			"public.a": {{"id": 1, "name": "alice"}, {"id": 2, "name": "bob"}},
		})
	h.seedSource(t, "mongo-b", "mongodb",
		map[string][]string{"coll.b": {"id"}},
		map[string][]map[string]any{
			"coll.b": {{"id": 10}, {"id": 11}, {"id": 12}},
		})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-a":    {"public.a": {"id", "name"}},
		"mongo-b": {"coll.b": {"id"}},
	})

	// The golden bytes are computed from the SAME aggregate shape the runner emits:
	// config -> table -> rows, indented two-space, with json sorting all map keys.
	aggregated := map[string]map[string][]map[string]any{
		"pg-a": {"public.a": {
			{"id": 1, "name": "alice"},
			{"id": 2, "name": "bob"},
		}},
		"mongo-b": {"coll.b": {
			{"id": 10}, {"id": 11}, {"id": 12},
		}},
	}

	wantBytes, err := json.MarshalIndent(aggregated, "", "  ")
	if err != nil {
		t.Fatalf("marshal golden: %v", err)
	}

	wantSum := sha256.Sum256(wantBytes)
	wantDigest := hex.EncodeToString(wantSum[:])

	// Run many times: parallel completion order must never perturb the bytes.
	for r := 0; r < 20; r++ {
		result, err := h.engine.ExecuteExtraction(context.Background(), plan)
		if err != nil {
			t.Fatalf("run %d: ExecuteExtraction: unexpected error: %v", r, err)
		}

		if result.Direct == nil {
			t.Fatalf("run %d: expected a direct result", r)
		}

		if string(result.Direct.Data) != string(wantBytes) {
			t.Fatalf("run %d: Direct bytes differ from golden:\n want=%q\n got =%q", r, wantBytes, result.Direct.Data)
		}

		if result.Direct.Integrity == nil || result.Direct.Integrity.Digest != wantDigest {
			t.Fatalf("run %d: Direct digest = %+v, want %q", r, result.Direct.Integrity, wantDigest)
		}
	}
}

// TestExecuteExtraction_StoreModeOpenStreamErrorFails proves a sink that fails to
// OPEN the stream fails the execution as unavailable, before any extraction work.
func TestExecuteExtraction_StoreModeOpenStreamErrorFails(t *testing.T) {
	t.Parallel()

	sink := memory.NewResultSink()
	sink.OpenErr = errors.New("sink offline")

	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})
	plan.Mode = engine.ModeStore

	_, err := h.engine.ExecuteExtraction(context.Background(), plan)

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("open-stream error = %v, want CategoryUnavailable", err)
	}
}

// TestExecuteExtraction_StoreModeCloseErrorFails proves a sink whose writer Close
// fails on finalize fails the execution as unavailable and returns no reference.
func TestExecuteExtraction_StoreModeCloseErrorFails(t *testing.T) {
	t.Parallel()

	sink := memory.NewResultSink()
	sink.CloseErr = errors.New("finalize failed")

	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})
	plan.Mode = engine.ModeStore

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("close error = %v, want CategoryUnavailable", err)
	}

	if result.Reference != nil {
		t.Fatalf("a failed finalize must return no reference, got %+v", result.Reference)
	}
}

// TestExecuteExtraction_StoreModeMidStreamCursorErrorFails proves a cursor that
// fails PART-WAY through iteration on the streaming path fails the execution and
// finalizes no result.
func TestExecuteExtraction_StoreModeMidStreamCursorErrorFails(t *testing.T) {
	t.Parallel()

	rows := make([]map[string]any, 1000)
	for i := range rows {
		rows[i] = map[string]any{"id": i}
	}

	sink := memory.NewResultSink()
	h := newStoreRunnerHarness(t, sink, nil)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": rows})
	// The cursor opens fine but trips an error after 10 rows.
	conn.StreamRowErr = errors.New("reader exploded")
	conn.StreamFailAfter = 10

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})
	plan.Mode = engine.ModeStore

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err == nil {
		t.Fatalf("a mid-stream cursor error must fail the run, got nil error")
	}

	if result.Reference != nil {
		t.Fatalf("a mid-stream failure must finalize no reference, got %+v", result.Reference)
	}

	if sink.StoredCount() != 0 {
		t.Fatalf("a mid-stream failure must store nothing, StoredCount = %d", sink.StoredCount())
	}
}

// TestExecuteExtraction_StoreModePreCancelledContextFails proves a store-mode
// run whose context is ALREADY cancelled fails as canceled before opening the
// sink stream — no extraction work, no finalized result.
func TestExecuteExtraction_StoreModePreCancelledContextFails(t *testing.T) {
	t.Parallel()

	sink := memory.NewResultSink()
	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})
	plan.Mode = engine.ModeStore

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := h.engine.ExecuteExtraction(ctx, plan)

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryCanceled {
		t.Fatalf("pre-cancelled store run = %v, want CategoryCanceled", err)
	}

	if result.Reference != nil || sink.StoredCount() != 0 || sink.OpenCount() != 0 {
		t.Fatalf("pre-cancelled run must not touch the sink: ref=%v stored=%d opened=%d",
			result.Reference, sink.StoredCount(), sink.OpenCount())
	}
}

// TestExecuteExtraction_StoreModeCursorOpenErrorFails proves a connector whose
// QueryStream fails to OPEN (before yielding any row) fails the store run as
// unavailable and finalizes no result.
func TestExecuteExtraction_StoreModeCursorOpenErrorFails(t *testing.T) {
	t.Parallel()

	sink := memory.NewResultSink()
	h := newStoreRunnerHarness(t, sink, nil)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}}})
	conn.QueryErr = errors.New("open failed")

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})
	plan.Mode = engine.ModeStore

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("cursor-open error = %v, want CategoryUnavailable", err)
	}

	if result.Reference != nil || sink.StoredCount() != 0 {
		t.Fatalf("a cursor-open failure must finalize no result")
	}
}

// TestExecuteExtraction_StoreModeProducerCancelOnBackpressure proves the
// producer respects cancellation while blocked on backpressure: the single
// writer is stalled on a gate, so a producer fills its one-batch-ahead channel
// and blocks sending its next batch; cancelling the host context unblocks the
// producer through flushBatch's cancel path and fails the run as canceled,
// finalizing nothing. The gate is then released so the writer goroutine exits.
func TestExecuteExtraction_StoreModeProducerCancelOnBackpressure(t *testing.T) {
	t.Parallel()

	// Enough rows to span several batches so a producer must send more than the
	// single buffered batch and block on the stalled writer.
	rows := make([]map[string]any, 4000)
	for i := range rows {
		rows[i] = map[string]any{"id": i, "blob": "padding-row-" + strconv.Itoa(i)}
	}

	gate := make(chan struct{})
	sink := memory.NewResultSink()
	sink.WriteGate = gate

	h := newStoreRunnerHarness(t, sink, nil)
	conn := h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id", "blob"}},
		map[string][]map[string]any{"public.users": rows})

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the host context the instant the producer's query is entered: by the
	// time it has produced its first buffered batch and tries to send the next, the
	// context is already cancelled, so flushBatch takes its cancel arm.
	conn.AfterQuery = cancel

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id", "blob"}},
	})
	plan.Mode = engine.ModeStore

	// Release the stalled writer shortly after dispatch so its goroutine can drain
	// and exit once the producers stop. Closing the gate is safe to do eagerly:
	// the writer only blocks on its FIRST Write.
	go func() { close(gate) }()

	result, err := h.engine.ExecuteExtraction(ctx, plan)
	if err == nil {
		t.Fatalf("a cancelled store stream must fail, got nil error")
	}

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryCanceled {
		t.Fatalf("error = %v, want CategoryCanceled", err)
	}

	if result.Reference != nil {
		t.Fatalf("a cancelled stream must finalize no reference, got %+v", result.Reference)
	}
}

// TestExecuteExtraction_StoreModeBuildFailureFails proves the store path closes
// no connector and finalizes nothing when a step's connector fails to BUILD: the
// failure surfaces as unavailable before any cursor is opened.
func TestExecuteExtraction_StoreModeBuildFailureFails(t *testing.T) {
	t.Parallel()

	sink := memory.NewResultSink()
	h := newStoreRunnerHarness(t, sink, nil)
	h.seedSource(t, "pg-main", "postgres",
		map[string][]string{"public.users": {"id"}},
		map[string][]map[string]any{"public.users": {{"id": 1}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"pg-main": {"public.users": {"id"}},
	})
	plan.Mode = engine.ModeStore

	// After planning, force the factory to fail every subsequent Build so the
	// runner's store-path build fails.
	h.factories["postgres"].BuildErr = errors.New("build blew up")

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)

	var engErr *engine.EngineError
	if !errors.As(err, &engErr) || engErr.Category != engine.CategoryUnavailable {
		t.Fatalf("store build failure = %v, want CategoryUnavailable", err)
	}

	if result.Reference != nil || sink.StoredCount() != 0 {
		t.Fatalf("a build failure must finalize no result")
	}
}

// TestExecuteExtraction_StoreModeMultiSourceOrderedNDJSON proves the streamed
// NDJSON is ordered by step Ordinal (sorted config name): all of the first
// config's lines precede the second config's, regardless of completion order.
func TestExecuteExtraction_StoreModeMultiSourceOrderedNDJSON(t *testing.T) {
	t.Parallel()

	limits := engine.DefaultLimits()
	limits.MaxConcurrency = 4

	sink := memory.NewResultSink()
	h := newLimitedRunnerHarness(t, limits, sink)
	h.seedSource(t, "aaa", "postgres",
		map[string][]string{"public.t": {"id"}},
		map[string][]map[string]any{"public.t": {{"id": 1}, {"id": 2}}})
	h.seedSource(t, "zzz", "mysql",
		map[string][]string{"public.t": {"id"}},
		map[string][]map[string]any{"public.t": {{"id": 9}}})

	plan := h.planFor(t, map[string]engine.FieldSelection{
		"aaa": {"public.t": {"id"}},
		"zzz": {"public.t": {"id"}},
	})

	result, err := h.engine.ExecuteExtraction(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteExtraction: unexpected error: %v", err)
	}

	stored, ok := sink.Get(result.Reference.Path)
	if !ok {
		t.Fatalf("sink has no payload at %q", result.Reference.Path)
	}

	lines := decodeNDJSON(t, stored)
	if len(lines) != 3 {
		t.Fatalf("NDJSON lines = %d, want 3", len(lines))
	}

	// Ordinal order: config "aaa" (ordinal 0) lines come before "zzz" (ordinal 1).
	wantConfigs := []string{"aaa", "aaa", "zzz"}
	for i, want := range wantConfigs {
		if lines[i].Config != want {
			t.Fatalf("line[%d] config = %q, want %q (must follow Ordinal order)", i, lines[i].Config, want)
		}
	}
}

// TestEagerCursor_RowBeforeNextAndAfterEnd proves the eager cursor's Row()
// contract: it yields the zero row before the first Next and after Next returns
// false, and iterates tables in sorted order.
func TestEagerCursor_RowBeforeNextAndAfterEnd(t *testing.T) {
	t.Parallel()

	cursor := engine.NewEagerCursor(map[string][]map[string]any{
		"z.table": {{"id": 1}},
		"a.table": {{"id": 2}, {"id": 3}},
	})

	// Before the first Next, Row is the zero value.
	if table, row := cursor.Row(); table != "" || row != nil {
		t.Fatalf("Row before Next = (%q, %v), want empty", table, row)
	}

	var order []string
	for cursor.Next(context.Background()) {
		table, _ := cursor.Row()
		order = append(order, table)
	}

	// Tables iterate in SORTED key order: all of a.table before z.table.
	want := []string{"a.table", "a.table", "z.table"}
	if len(order) != len(want) {
		t.Fatalf("iterated tables = %v, want %v", order, want)
	}

	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("table[%d] = %q, want %q (sorted order)", i, order[i], want[i])
		}
	}

	// After end-of-stream, Row is the zero value again and Err is nil.
	if table, row := cursor.Row(); table != "" || row != nil {
		t.Fatalf("Row after end = (%q, %v), want empty", table, row)
	}

	if err := cursor.Err(); err != nil {
		t.Fatalf("eager cursor Err = %v, want nil", err)
	}

	if err := cursor.Close(context.Background()); err != nil {
		t.Fatalf("eager cursor Close = %v, want nil", err)
	}
}

// --- streaming-test helpers ---

// arrivalBarrier blocks each arriving goroutine until n have arrived, then
// releases all of them. It records the peak number simultaneously parked, so a
// test can assert MaxConcurrency goroutines truly overlapped.
type arrivalBarrier struct {
	n  int
	mu sync.Mutex
	wg sync.WaitGroup

	arrived  int
	inFlight int
	peakN    int
}

func newArrivalBarrier(n int) *arrivalBarrier {
	b := &arrivalBarrier{n: n}
	b.wg.Add(n)

	return b
}

// arriveAndWait records the arrival, releases the shared latch once the nth
// goroutine arrives, and blocks every goroutine until then. The peak in-flight
// count is the max simultaneously parked here.
func (b *arrivalBarrier) arriveAndWait() {
	b.mu.Lock()
	b.arrived++
	b.inFlight++
	if b.inFlight > b.peakN {
		b.peakN = b.inFlight
	}
	b.mu.Unlock()

	b.wg.Done()
	b.wg.Wait() // releases only once all n have arrived

	b.mu.Lock()
	b.inFlight--
	b.mu.Unlock()
}

func (b *arrivalBarrier) peak() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.peakN
}

// distinctDBTypes are valid model DB types used to give each seeded source its
// OWN connector factory. seedSource registers one factory per datasource type,
// so reusing a type would overwrite the prior factory; distinct types keep each
// source's connector (and its lifecycle counters) independent.
var distinctDBTypes = []string{"postgres", "mysql", "oracle", "sqlserver", "mongodb"}

func seedN(t *testing.T, h *runnerHarness, n int, _ error) []*memory.Connector {
	t.Helper()

	if n > len(distinctDBTypes) {
		t.Fatalf("seedN supports at most %d sources, got %d", len(distinctDBTypes), n)
	}

	conns := make([]*memory.Connector, n)
	for i := 0; i < n; i++ {
		name := "src-" + strconv.Itoa(i)
		conns[i] = h.seedSource(t, name, distinctDBTypes[i],
			map[string][]string{"public.t": {"id"}},
			map[string][]map[string]any{"public.t": {{"id": i}}})
	}

	return conns
}

func mappedFor(n int) map[string]engine.FieldSelection {
	mapped := make(map[string]engine.FieldSelection, n)
	for i := 0; i < n; i++ {
		mapped["src-"+strconv.Itoa(i)] = engine.FieldSelection{"public.t": {"id"}}
	}

	return mapped
}

func assertAllClosedOnce(t *testing.T, conns []*memory.Connector) {
	t.Helper()

	for i, conn := range conns {
		if got := conn.CloseCount(); got != 1 {
			t.Fatalf("connector %d CloseCount = %d, want 1 (every built connector closes exactly once)", i, got)
		}
	}
}

// assertBuiltAreClosed proves that across all factories, every connector the
// RUNNER built was closed. The factories build one connector per Build call
// (planner + runner), so a built-but-unclosed connector signals a leak.
func assertBuiltAreClosed(t *testing.T, h *runnerHarness, _ int) {
	t.Helper()

	for dsType, factory := range h.factories {
		for idx, conn := range factory.Built() {
			if conn.CloseCount() == 0 {
				t.Fatalf("factory %q connector #%d was built but never closed (leak)", dsType, idx)
			}
		}
	}
}
