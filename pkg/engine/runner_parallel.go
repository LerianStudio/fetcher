// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// This file implements the BOUNDED-PARALLEL execution of an ExtractionPlan and
// the STREAMING store path. It is the half of the runner that makes
// Limits.MaxConcurrency real and keeps store-mode peak memory bounded.
//
// CONCURRENCY MODEL. The unit of parallelism is the PlanStep. A buffered
// semaphore channel of capacity max(1, MaxConcurrency) caps how many steps run
// at once; each step runs in its own goroutine (build -> TestConnection ->
// QueryStream -> drive cursor -> Close). A shared cancelable child context fails
// the whole run fast: the FIRST step error (or a host cancel) cancels in-flight
// steps so the rest stop promptly. No partial result is ever returned.
//
// CONNECTOR LIFECYCLE UNDER CONCURRENCY. Every built connector is closed on every
// path — success, fail-fast, panic, ctx-cancel — because each step goroutine owns
// its connector and defers Close. A panic in one goroutine is recovered there,
// converted to a CategoryInternal engine error, and cancels the shared context so
// the others unwind and close their connectors too; it never leaks another step's
// connector.
//
// DETERMINISM. Steps carry a stable Ordinal (their index in the sorted plan).
// The DIRECT path merges per-step submaps and relies on json.MarshalIndent's
// key-sorting for byte-identity. The STORE path drains steps strictly by
// ascending Ordinal through a single writer, so the NDJSON bytes — and the
// integrity digest computed over exactly those bytes — are identical on every
// run regardless of which goroutine finished first.

// storeBatchRows is the NDJSON batch size: the runner flushes a step's encoded
// rows to its handoff channel every storeBatchRows rows (and once more for the
// remainder). It is a small constant so peak memory is roughly
// MaxConcurrency × (one batch) rather than the whole result. 256 rows balances
// per-row channel/lock overhead against the memory a single in-flight batch
// holds.
const storeBatchRows = 256

// stepResult is one DIRECT-path step's materialized output: the per-config rows
// and the step's row count, tagged with its config name for deterministic merge.
type stepResult struct {
	ordinal    int
	configName string
	rows       map[string][]map[string]any
	count      int64
}

// runStepsParallel runs every plan step through a bounded worker pool and returns
// the per-step results in ascending Ordinal order. It is the DIRECT path's
// concurrent extraction: each step materializes its own submap, the pool joins,
// and the caller merges. The first failing step (or a host cancel) fails the
// whole run fast; no partial results are returned. Every built connector is
// closed via executeStep's defers, including on panic and cancel.
func (e *Engine) runStepsParallel(
	ctx context.Context,
	tenant TenantContext,
	store ConnectionStore,
	plan ExtractionPlan,
) ([]stepResult, error) {
	results := make([]stepResult, len(plan.Steps))

	// runningSize accumulates a SOUND LOWER BOUND of the final serialized size: the
	// sum of each completed step's COMPACT-marshaled byte length. Compact <=
	// indented always, so once this sum crosses MaxResultBytes the final indented
	// payload is guaranteed over-limit too — letting the pool fail FAST (cancel
	// in-flight steps and skip not-yet-dispatched ones) before marshaling the whole
	// result. It is the parallel analogue of the historical per-step guard; the
	// authoritative post-marshal check in runDirectExtraction remains the source of
	// truth. A mutex guards it because steps complete concurrently.
	var (
		sizeMu      sync.Mutex
		runningSize int64
	)

	err := e.forEachStep(ctx, plan, func(runCtx context.Context, idx int, step PlanStep) error {
		rows, count, stepErr := e.executeStep(runCtx, tenant, store, step)
		if stepErr != nil {
			return stepErr
		}

		if plan.Limits.MaxResultBytes > 0 {
			compact, marshalErr := json.Marshal(rows)
			if marshalErr != nil {
				return NewEngineError(CategoryInternal, "failed to serialize extraction result")
			}

			sizeMu.Lock()

			runningSize += int64(len(compact))
			exceeded := runningSize > plan.Limits.MaxResultBytes

			sizeMu.Unlock()

			if exceeded {
				return NewEngineError(CategoryLimitExceeded, "extraction result exceeds the configured size limit")
			}
		}

		results[idx] = stepResult{
			ordinal:    step.Ordinal,
			configName: step.ConfigName,
			rows:       rows,
			count:      count,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// forEachStep dispatches one goroutine per plan step, bounded by a semaphore of
// capacity max(1, MaxConcurrency), and invokes fn for EVERY step with a SHARED
// cancelable context. It is the common worker-pool primitive both delivery paths
// use. The FIRST non-nil fn error (recorded once) cancels the shared context so
// in-flight steps stop, and is returned after every goroutine has joined — so the
// pool never returns while a step (and its connector) is still live. A panic in
// fn is recovered per-goroutine, converted to a CategoryInternal error, and also
// cancels the shared context.
//
// EVERY-STEP INVOCATION IS LOAD-BEARING for the store path. fn is invoked for
// every step even after a cancel: the semaphore is acquired INSIDE the goroutine
// (cancel-aware), and a cancelled run still calls fn, whose own top-of-function
// ctx guard returns immediately BEFORE building any connector. This preserves the
// "don't build connectors after cancel" optimization while guaranteeing each
// step's goroutine runs exactly once — which the store path relies on so every
// per-step result channel is closed exactly once (streamStep's defer close),
// avoiding a writer that parks forever on a never-closed channel. The concurrency
// bound still holds: only effectiveConcurrency goroutines hold the semaphore (do
// connector work) at a time; the rest are parked cheaply on the acquire or have
// already short-circuited on the cancel guard.
func (e *Engine) forEachStep(
	ctx context.Context,
	plan ExtractionPlan,
	fn func(ctx context.Context, idx int, step PlanStep) error,
) error {
	if len(plan.Steps) == 0 {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	semaphore := make(chan struct{}, effectiveConcurrency(plan.Limits))

	var (
		wg      sync.WaitGroup
		once    sync.Once
		firstWg sync.Mutex
		first   error
	)

	record := func(err error) {
		if err == nil {
			return
		}

		once.Do(func() {
			firstWg.Lock()

			first = err

			firstWg.Unlock()

			// Cancel in-flight steps so the rest fail fast and unwind their
			// connectors.
			cancel()
		})
	}

	for idx, step := range plan.Steps {
		wg.Add(1)

		go func(idx int, step PlanStep) {
			defer wg.Done()
			defer func() {
				// Recover per-goroutine so one step's panic neither crashes the process
				// nor leaks the OTHER steps' connectors: it becomes a CategoryInternal
				// error and cancels the shared context, and this goroutine's own
				// connector is already closed by executeStep's defers as the stack
				// unwinds through them.
				if r := recover(); r != nil {
					record(NewEngineError(CategoryInternal, fmt.Sprintf("extraction step panicked: %v", r)))
				}
			}()

			// Acquire a concurrency slot, cancel-aware. On cancel we STILL invoke fn
			// (below) so the step's own ctx guard runs and — critically for the store
			// path — its per-step channel is closed by fn's deferred close. A
			// cancelled goroutine does no connector work because fn returns at its
			// top-of-function ctx guard before building anything.
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-runCtx.Done():
				// Skip the slot entirely; fn still runs and short-circuits on the
				// cancelled context without acquiring the semaphore.
			}

			record(fn(runCtx, idx, step))
		}(idx, step)
	}

	wg.Wait()

	firstWg.Lock()
	defer firstWg.Unlock()

	if first != nil {
		return first
	}

	// A host cancel that fired with no step error still fails the run: a result
	// the host has abandoned must not be returned. Map it to the right category.
	if engErr := contextError(ctx.Err()); engErr != nil {
		return engErr
	}

	return nil
}

// streamSteps is the STORE path's ORDINAL-WINDOWED worker pool. Unlike the
// generic forEachStep semaphore, it guarantees the steps RUNNING at any moment
// are always the W lowest-ordinal incomplete steps (W = effectiveConcurrency).
// That invariant is what keeps the store path's strict-ordinal single writer
// deadlock-free: the writer's current target channel always has a producer that
// is running or finished, never one still parked waiting for a slot.
//
// MECHANISM: step idx is gated on the completion of step idx-W. A per-step "done"
// channel is closed when that step's goroutine returns (success, error, panic, or
// cancel), which releases step idx+W to start. Steps 0..W-1 start immediately.
// At most W goroutines are past their gate (doing connector work) at once.
//
// fn is invoked for EVERY step (even after a cancel) so each step's per-step
// result channel is closed exactly once by streamStep's deferred close; a
// cancelled step returns at fn's top-of-function ctx guard before building any
// connector. The FIRST error cancels the shared context (fail-fast), and a panic
// is recovered per-goroutine as a CategoryInternal error. The function returns
// only after every goroutine has joined, so no producer (or its connector) is
// still live on return.
func (e *Engine) streamSteps(
	ctx context.Context,
	plan ExtractionPlan,
	fn func(ctx context.Context, idx int, step PlanStep) error,
) error {
	if len(plan.Steps) == 0 {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	window := effectiveConcurrency(plan.Limits)

	// done[i] is closed when step i's goroutine returns. Step i waits for
	// done[i-window] before doing connector work, capping live steps at `window`
	// while keeping them the lowest-ordinal incomplete ones.
	done := make([]chan struct{}, len(plan.Steps))
	for i := range done {
		done[i] = make(chan struct{})
	}

	var (
		wg      sync.WaitGroup
		once    sync.Once
		firstWg sync.Mutex
		first   error
	)

	record := func(err error) {
		if err == nil {
			return
		}

		once.Do(func() {
			firstWg.Lock()

			first = err

			firstWg.Unlock()

			cancel()
		})
	}

	for idx, step := range plan.Steps {
		wg.Add(1)

		go func(idx int, step PlanStep) {
			defer wg.Done()
			// Signal completion LAST (after fn's defers, including streamStep's
			// channel close, have run) so the step idx+window is released only once
			// this step is fully torn down. Registered first => runs last (LIFO).
			defer close(done[idx])
			defer func() {
				if r := recover(); r != nil {
					record(NewEngineError(CategoryInternal, fmt.Sprintf("extraction step panicked: %v", r)))
				}
			}()

			// Gate on the prior-window step's completion (or a cancel). Even when
			// cancelled we still call fn so streamStep closes this step's channel;
			// fn short-circuits on the cancelled context before any connector work.
			if idx >= window {
				select {
				case <-done[idx-window]:
				case <-runCtx.Done():
				}
			}

			record(fn(runCtx, idx, step))
		}(idx, step)
	}

	wg.Wait()

	firstWg.Lock()
	defer firstWg.Unlock()

	if first != nil {
		return first
	}

	if engErr := contextError(ctx.Err()); engErr != nil {
		return engErr
	}

	return nil
}

// runStoreExtraction is the STORE delivery path: it streams the result to the
// host ResultSink incrementally, holding only ~MaxConcurrency in-flight batches
// in memory rather than the whole result. Extraction goroutines encode their
// rows to NDJSON batches and hand them, one batch ahead, to a single writer
// goroutine that drains steps in strict Ordinal order, writes them to the sink's
// ResultStreamWriter, advances a SHA-256 digest over exactly the bytes written,
// and enforces MaxResultBytes as a running counter. On any failure (write error,
// size-limit breach, cancel) the writer is ABANDONED (Close is not called) so a
// partial result is never finalized.
func (e *Engine) runStoreExtraction(
	ctx context.Context,
	tenant TenantContext,
	connStore ConnectionStore,
	plan ExtractionPlan,
) (ExtractionResult, error) {
	if engErr := contextError(ctx.Err()); engErr != nil {
		e.recordTerminalFailure(ctx, tenant, plan.RequestID, engErr)
		return ExtractionResult{}, engErr
	}

	writer, err := e.options.resultSink.OpenResultStream(ctx, tenant)
	if err != nil {
		e.recordExecutionStatus(ctx, tenant, plan.RequestID, StatusFailed, "failed to persist extraction result")
		return ExtractionResult{}, NewEngineError(CategoryUnavailable, "failed to persist extraction result")
	}

	result, runErr := e.streamPlanToSink(ctx, tenant, connStore, plan, writer)
	if runErr != nil {
		// Abandon the writer WITHOUT Close so no partial reference is finalized.
		e.recordTerminalFailure(ctx, tenant, plan.RequestID, runErr)
		return ExtractionResult{}, runErr
	}

	e.recordExecutionStatus(ctx, tenant, plan.RequestID, StatusCompleted, "")

	return result, nil
}

// streamPlanToSink runs the worker pool over the plan, ordering each step's
// NDJSON batches into the writer by Ordinal, and finalizes the writer on success.
// It owns the digest, the row-count accounting, and the running byte budget.
func (e *Engine) streamPlanToSink(
	ctx context.Context,
	tenant TenantContext,
	connStore ConnectionStore,
	plan ExtractionPlan,
	writer ResultStreamWriter,
) (ExtractionResult, error) {
	// Each step owns a size-1 (one-batch-ahead) handoff channel of NDJSON batches.
	// A step blocks after producing one batch ahead of the writer, which is the
	// backpressure that bounds peak memory to ~MaxConcurrency × one batch.
	channels := make([]chan ndjsonBatch, len(plan.Steps))
	for i := range channels {
		channels[i] = make(chan ndjsonBatch, 1)
	}

	writeCtx, cancelWrite := context.WithCancel(ctx)
	defer cancelWrite()

	// The single writer goroutine drains steps in Ordinal order, writing each
	// step's batches to the sink, advancing the digest, counting rows, and
	// enforcing the running byte budget. Its result (digest, totals, error) is
	// delivered once on writerDone.
	type writerOutcome struct {
		digest   string
		rowCount int64
		size     int64
		err      error
	}

	writerDone := make(chan writerOutcome, 1)

	go func() {
		hash := sha256.New()

		var (
			rowCount int64
			size     int64
		)

		for ordinal := 0; ordinal < len(channels); ordinal++ {
			for batch := range channels[ordinal] {
				size += int64(len(batch.bytes))

				if plan.Limits.MaxResultBytes > 0 && size > plan.Limits.MaxResultBytes {
					// Over-limit: cancel extraction so producers stop, and report the
					// limit error. The writer is abandoned by the caller (no Close).
					limitErr := NewEngineError(CategoryLimitExceeded, "extraction result exceeds the configured size limit")

					cancelWrite()

					writerDone <- writerOutcome{err: limitErr}

					return
				}

				if _, err := writer.Write(batch.bytes); err != nil {
					writeErr := NewEngineError(CategoryUnavailable, "failed to persist extraction result")

					cancelWrite()

					writerDone <- writerOutcome{err: writeErr}

					return
				}

				_, _ = hash.Write(batch.bytes)
				rowCount += batch.rows
			}
		}

		writerDone <- writerOutcome{
			digest:   hex.EncodeToString(hash.Sum(nil)),
			rowCount: rowCount,
			size:     size,
		}
	}()

	rowCounts := make([]int64, len(plan.Steps))

	// Drive the steps through an ORDINAL-WINDOWED pool (not the generic forEachStep
	// semaphore). The store path's single writer drains channels in strict ordinal
	// order, so it can only make progress if the LOWEST-ordinal incomplete steps
	// are the ones actually running. A plain semaphore lets a higher-ordinal step
	// grab the only slot and block on a send the writer won't service until earlier
	// (parked) steps run — a deadlock at low concurrency. The window gate fixes
	// that: step idx starts only after step idx-W finished (W = concurrency), so at
	// most W steps run at once AND they are always the W lowest-ordinal incomplete
	// steps. streamStep closes channels[idx] itself (defer close(out)) on every
	// return path, so the writer advances past a step the instant it is done.
	runErr := e.streamSteps(writeCtx, plan, func(runCtx context.Context, idx int, step PlanStep) error {
		count, err := e.streamStep(runCtx, tenant, connStore, step, channels[idx])
		if err != nil {
			return err
		}

		rowCounts[idx] = count

		return nil
	})

	// All step channels are already closed by their owning streamStep goroutines
	// (streamSteps invokes the fn for EVERY step exactly once, so every channel is
	// closed exactly once). The writer therefore reaches the end of every range
	// loop and reports on writerDone without any close site here.
	outcome := <-writerDone

	// The WRITER's error is the AUTHORITATIVE failure reason and takes precedence
	// over a producer error. When the writer trips the size budget (or a sink
	// Write fails) it cancels the shared context, which makes in-flight producers
	// return a generic "canceled" error; surfacing that cancel instead of the
	// writer's limit_exceeded would mislabel the failure. So the writer's reason is
	// checked first, and a producer error only surfaces when the writer succeeded.
	if outcome.err != nil {
		return ExtractionResult{}, outcome.err
	}

	if runErr != nil {
		return ExtractionResult{}, runErr
	}

	if engErr := contextError(ctx.Err()); engErr != nil {
		return ExtractionResult{}, engErr
	}

	return e.buildStoreReference(plan, writer, outcome.digest, outcome.rowCount, outcome.size, sumRowCounts(plan, rowCounts))
}

// streamStep opens the step's connector, drives its cursor, and emits the rows
// as NDJSON batches to the step's handoff channel in cursor order. It batches
// every storeBatchRows rows (plus a final partial batch) so memory stays bounded.
// It ALWAYS closes the connector and cursor. A send that races a cancel returns
// the context error so the producer stops promptly.
//
// CHANNEL-CLOSE OWNERSHIP IS LOAD-BEARING: streamStep closes its OWN out channel
// (the FIRST deferred action, so it runs on every return path — success, error,
// panic, or the cancel guard). Per-step self-close is what lets the single writer
// advance through channels in ordinal order: when step N's channel closes, the
// writer's `range` over it ends and it moves to step N+1, which unblocks step
// N+1's producer that was parked on its second (one-ahead) send. Without
// per-step close, the writer parks forever on the first never-closed channel and
// any non-tail step that produces a second batch deadlocks. forEachStep invokes
// this fn for EVERY step (even cancelled ones), so every channel closes exactly
// once. close() is registered LAST among defers here only in source order; Go
// runs defers LIFO, so it executes AFTER the connector/cursor closes below — the
// writer therefore never sees a closed channel while this step still holds a
// live cursor.
func (e *Engine) streamStep(
	ctx context.Context,
	tenant TenantContext,
	store ConnectionStore,
	step PlanStep,
	out chan<- ndjsonBatch,
) (int64, error) {
	// Close THIS step's channel on every return path. Registered first so (LIFO)
	// it runs last — after the cursor and connector are closed — so the writer
	// only advances past this step once the step is fully torn down.
	defer close(out)

	// Top-of-function cancel guard: a step whose run was already cancelled returns
	// HERE, before building any connector, while still closing its channel via the
	// defer above so the writer can advance.
	if engErr := contextError(ctx.Err()); engErr != nil {
		return 0, engErr
	}

	connector, err := e.openStepConnector(ctx, tenant, store, step)
	if err != nil {
		return 0, err
	}

	defer func() { _ = connector.Close(ctx) }()

	cursor, err := connector.QueryStream(ctx, extractionRequestForStep(step))
	if err != nil {
		return 0, safeConnectorError(ctx, err, "failed to query datasource")
	}

	defer func() { _ = cursor.Close(ctx) }()

	encoder := newNDJSONEncoder(step.ConfigName)

	var (
		count   int64
		pending int
	)

	for cursor.Next(ctx) {
		table, row := cursor.Row()
		if err := encoder.appendRow(table, row); err != nil {
			return 0, err
		}

		count++
		pending++

		if pending >= storeBatchRows {
			if err := flushBatch(ctx, out, encoder, int64(pending)); err != nil {
				return 0, err
			}

			pending = 0
		}
	}

	if err := cursor.Err(); err != nil {
		return 0, safeConnectorError(ctx, err, "failed to query datasource")
	}

	if engErr := contextError(ctx.Err()); engErr != nil {
		return 0, engErr
	}

	if pending > 0 {
		if err := flushBatch(ctx, out, encoder, int64(pending)); err != nil {
			return 0, err
		}
	}

	return count, nil
}

// flushBatch sends the encoder's accumulated bytes as one batch, then resets the
// encoder for the next batch. A send that races a cancel returns the context
// error so the producer stops promptly rather than blocking on a channel whose
// writer has stopped draining.
func flushBatch(ctx context.Context, out chan<- ndjsonBatch, encoder *ndjsonEncoder, rows int64) error {
	batch := ndjsonBatch{bytes: encoder.take(), rows: rows}

	select {
	case out <- batch:
		return nil
	case <-ctx.Done():
		if engErr := contextError(ctx.Err()); engErr != nil {
			return engErr
		}

		return NewEngineError(CategoryCanceled, "execution canceled")
	}
}

// buildStoreReference finalizes the writer and assembles the store-mode
// ExtractionResult. It Closes the writer exactly once (the success path), then
// PRESERVES the sink's reported integrity/protection (validating appliedBy) and
// fills the canonical fields the sink left unset — mirroring the historical
// completeStoreMode contract. When the sink reports no integrity, the Engine
// stamps the digest it computed over exactly the streamed bytes.
func (e *Engine) buildStoreReference(
	plan ExtractionPlan,
	writer ResultStreamWriter,
	digest string,
	rowCount int64,
	size int64,
	perConfig map[string]int64,
) (ExtractionResult, error) {
	reference, err := writer.Close()
	if err != nil {
		return ExtractionResult{}, NewEngineError(CategoryUnavailable, "failed to persist extraction result")
	}

	if reference.Protection != nil && !reference.Protection.AppliedBy.IsValid() {
		return ExtractionResult{}, NewEngineError(CategoryValidation, "result protection appliedBy is invalid")
	}

	if reference.Integrity == nil {
		reference.Integrity = &ResultIntegrity{Algorithm: "SHA-256", Digest: digest}
	}

	reference.Format = directResultFormat
	reference.RowCount = rowCount

	if reference.SizeBytes == 0 {
		reference.SizeBytes = size
	}

	completedAt := time.Now().UTC()
	reference.CompletedAt = completedAt

	return ExtractionResult{
		State: ExecutionState{
			JobID:       plan.RequestID,
			Status:      StatusCompleted,
			CompletedAt: &completedAt,
		},
		Reference: &reference,
		RowCounts: perConfig,
	}, nil
}

// sumRowCounts maps the per-step row counts back onto their config names so the
// host can attribute the total to each source, mirroring the direct path's
// RowCounts. The slice is index-aligned with plan.Steps.
func sumRowCounts(plan ExtractionPlan, counts []int64) map[string]int64 {
	out := make(map[string]int64, len(plan.Steps))
	for i, step := range plan.Steps {
		out[step.ConfigName] = counts[i]
	}

	return out
}

// ndjsonBatch is one flushed unit of NDJSON bytes plus the number of rows it
// encodes, handed from a producer step to the single writer goroutine.
type ndjsonBatch struct {
	bytes []byte
	rows  int64
}

// ndjsonEncoder accumulates NDJSON lines for one step. Each line is
//
//	{"config":"<configName>","table":"<qualifiedTable>","row":{<col>:<val>,...}}
//
// newline-terminated, with no enclosing array — the wire shape documented on
// ResultSink.OpenResultStream. It marshals each line with encoding/json so the
// row map's keys are SORTED, which is what makes the streamed bytes (and the
// digest over them) byte-identical across runs.
type ndjsonEncoder struct {
	configName string
	buf        []byte
}

func newNDJSONEncoder(configName string) *ndjsonEncoder {
	return &ndjsonEncoder{configName: configName}
}

// ndjsonLine is the per-row NDJSON envelope. json.Marshal sorts a map's keys, so
// encoding the row map here yields deterministic column order on every run.
type ndjsonLine struct {
	Config string         `json:"config"`
	Table  string         `json:"table"`
	Row    map[string]any `json:"row"`
}

// appendRow marshals one row into the buffer as a newline-terminated NDJSON line.
func (e *ndjsonEncoder) appendRow(table string, row map[string]any) error {
	line, err := json.Marshal(ndjsonLine{Config: e.configName, Table: table, Row: row})
	if err != nil {
		// The row holds only extracted data; a marshal failure is an unexpected
		// internal condition. The error is not echoed.
		return NewEngineError(CategoryInternal, "failed to serialize extraction result")
	}

	e.buf = append(e.buf, line...)
	e.buf = append(e.buf, '\n')

	return nil
}

// take returns the accumulated bytes and resets the buffer for the next batch.
// It hands ownership of a FRESH slice to the caller so a later append cannot
// alias the bytes already in flight on the channel.
func (e *ndjsonEncoder) take() []byte {
	out := e.buf
	e.buf = nil

	return out
}
