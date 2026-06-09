// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"sync"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
)

// compile-time proof the in-memory sink satisfies the streaming contract.
var _ engine.ResultSink = (*ResultSink)(nil)

// ResultSink is an in-memory engine.ResultSink. It stores serialized result
// payloads in a mutex-protected map keyed by a logical, content-derived path and
// lets tests read them back through Get. It performs no I/O and persists nothing
// beyond the process.
//
// PutErr is a test affordance mirroring the cache/store doubles' GetErr/PutErr
// idiom: when set, PersistResult fails so runner tests (ST-T007-02/03/04) can
// force a sink failure deterministically. It is inert (nil) by default.
type ResultSink struct {
	mu         sync.RWMutex
	results    map[string][]byte
	seq        int
	putCount   int
	openCount  int
	writeSizes []int

	// PutErr, when non-nil, makes PersistResult return a safe storage-category
	// Engine error. The injected error is NOT echoed across the boundary so no
	// sensitive underlying detail leaks.
	PutErr error

	// OpenErr, when non-nil, makes OpenResultStream fail so a store-mode test can
	// force the sink-open failure path deterministically. It is inert by default.
	OpenErr error
	// WriteErr, when non-nil, makes the streaming writer's Write fail (after the
	// first successful write count is recorded) so a test can drive the runner's
	// streaming write-failure path. It is inert by default.
	WriteErr error
	// CloseErr, when non-nil, makes the streaming writer's Close fail so a test can
	// drive the runner's finalize-failure path. It is inert by default.
	CloseErr error

	// WriteGate, when non-nil, makes the streaming writer's FIRST Write block until
	// the gate is closed (or a value is sent). It lets a test stall the single
	// writer goroutine so producers fill their one-batch-ahead channels and block
	// in flushBatch, exercising the producer-cancel path. It is inert (nil) by
	// default. Only the first Write blocks, so closing the gate lets the writer
	// drain the rest without further stalls.
	WriteGate chan struct{}
	gateOnce  sync.Once

	// ProtectionResult, when non-nil, is the canonical result protection metadata
	// the sink reports on the returned reference. It models an adapter that
	// encrypts the stored bytes and stamps its own protection state; the Engine
	// MUST preserve exactly this metadata (after validating appliedBy). It is inert
	// (nil) by default, modelling an unencrypted store.
	ProtectionResult *engine.ResultProtection

	// OmitIntegrity, when true, makes PersistResult return a reference with NIL
	// Integrity, modelling a sink that reports no integrity of its own. It drives
	// the Engine's canonical-fill fallback (stamp an unkeyed SHA-256 digest so a
	// stored result always carries verifiable integrity). It is false by default,
	// so the sink reports its own content digest.
	OmitIntegrity bool

	// OmitSizeBytes, when true, makes PersistResult return a reference with
	// SizeBytes == 0, modelling a sink that does not report the written size. It
	// drives the Engine's canonical-fill fallback (set SizeBytes from the payload
	// length). It is false by default, so the sink reports the stored byte size.
	OmitSizeBytes bool
}

// NewResultSink returns an empty in-memory result sink.
func NewResultSink() *ResultSink {
	return &ResultSink{
		results: make(map[string][]byte),
	}
}

// PersistResult implements engine.ResultSink. It stores a defensive copy of the
// payload under a deterministic, collision-resistant LOGICAL path and returns a
// secret-free reference carrying the path, canonical integrity (a SHA-256
// content digest — NOT a bare HMAC placeholder), and the byte size. When PutErr
// is set it returns a safe storage-category Engine error without echoing the
// injected detail.
func (s *ResultSink) PersistResult(
	_ context.Context,
	tenant engine.TenantContext,
	payload []byte,
) (engine.ResultReference, error) {
	s.mu.Lock()
	s.putCount++
	s.mu.Unlock()

	if s.PutErr != nil {
		// Safe storage-category error: the underlying PutErr (which may carry
		// sensitive detail) is intentionally NOT wrapped or echoed.
		return engine.ResultReference{}, engine.NewEngineError(engine.CategoryUnavailable, "result sink unavailable")
	}

	stored := make([]byte, len(payload))
	copy(stored, payload)

	sum := sha256.Sum256(stored)
	digest := hex.EncodeToString(sum[:])

	s.mu.Lock()
	defer s.mu.Unlock()

	// The LOGICAL path embeds the tenant scope and a monotonic sequence so
	// distinct writes never collide even when payloads are identical.
	//
	// NOTE: this in-memory double concatenates the tenant id into the logical
	// path verbatim, which is acceptable for an in-process map key. The REAL
	// S3/SeaweedFS adapter (T-010) MUST sanitize tenant ids before composing
	// physical object paths to prevent path traversal / key-injection.
	s.seq++
	path := "memory://" + tenant.TenantID + "/" + digest + "-" + strconv.Itoa(s.seq)
	s.results[path] = stored

	ref := engine.ResultReference{
		Path:      path,
		SizeBytes: int64(len(stored)),
		Integrity: &engine.ResultIntegrity{
			// HMAC is one possible integrity SIGNATURE; the in-memory double has
			// no key, so it records an unkeyed content DIGEST instead.
			Algorithm: "SHA-256",
			Digest:    digest,
		},
	}

	// Test affordances modelling a sink that under-reports: the Engine fills the
	// missing canonical fields so a stored result is always whole.
	if s.OmitIntegrity {
		ref.Integrity = nil
	}

	if s.OmitSizeBytes {
		ref.SizeBytes = 0
	}

	if s.ProtectionResult != nil {
		protection := *s.ProtectionResult
		ref.Protection = &protection
	}

	return ref, nil
}

// OpenResultStream implements engine.ResultSink. It returns an incremental
// writer that accumulates the streamed NDJSON bytes into a buffer and, on Close,
// stores them under a deterministic logical path — the same shape PersistResult
// produces, but written incrementally. ACCUMULATING into a buffer is acceptable
// for this in-memory test double: the bounded-memory guarantee lives on the
// ENGINE side (it flushes batch-by-batch), and the writer records every Write's
// size so a test can prove the engine flushed INCREMENTALLY (multiple Writes)
// rather than once.
func (s *ResultSink) OpenResultStream(_ context.Context, tenant engine.TenantContext) (engine.ResultStreamWriter, error) {
	s.mu.Lock()
	s.openCount++
	openErr := s.OpenErr
	s.mu.Unlock()

	if openErr != nil {
		return nil, engine.NewEngineError(engine.CategoryUnavailable, "result sink unavailable")
	}

	return &resultStreamWriter{sink: s, tenant: tenant}, nil
}

// resultStreamWriter is the in-memory engine.ResultStreamWriter. It buffers the
// NDJSON bytes the engine writes and finalizes them into the parent sink on
// Close, recording each Write's size on the sink so a test can assert the engine
// streamed incrementally.
type resultStreamWriter struct {
	sink   *ResultSink
	tenant engine.TenantContext
	buf    []byte
}

// Write records the batch size on the sink and appends the bytes to the buffer.
// When the sink's WriteErr is set it records the size first (so the test still
// sees the write was attempted) and then fails.
func (w *resultStreamWriter) Write(p []byte) (int, error) {
	w.sink.mu.Lock()
	w.sink.writeSizes = append(w.sink.writeSizes, len(p))
	writeErr := w.sink.WriteErr
	gate := w.sink.WriteGate
	w.sink.mu.Unlock()

	// Stall the FIRST Write on the gate so producers back up behind it, exercising
	// the producer flushBatch cancel path. Subsequent Writes do not block.
	if gate != nil {
		w.sink.gateOnce.Do(func() { <-gate })
	}

	if writeErr != nil {
		return 0, writeErr
	}

	w.buf = append(w.buf, p...)

	return len(p), nil
}

// Close finalizes the buffered bytes into the sink and returns a reference with
// the same canonical shape PersistResult produces (logical path, SHA-256 content
// digest, byte size), honoring the same OmitIntegrity/OmitSizeBytes/Protection
// affordances. When CloseErr is set it fails without storing.
func (w *resultStreamWriter) Close() (engine.ResultReference, error) {
	w.sink.mu.Lock()
	defer w.sink.mu.Unlock()

	if w.sink.CloseErr != nil {
		return engine.ResultReference{}, w.sink.CloseErr
	}

	stored := make([]byte, len(w.buf))
	copy(stored, w.buf)

	sum := sha256.Sum256(stored)
	digest := hex.EncodeToString(sum[:])

	w.sink.seq++
	path := "memory://" + w.tenant.TenantID + "/" + digest + "-" + strconv.Itoa(w.sink.seq)
	w.sink.results[path] = stored

	ref := engine.ResultReference{
		Path:      path,
		SizeBytes: int64(len(stored)),
		Integrity: &engine.ResultIntegrity{
			Algorithm: "SHA-256",
			Digest:    digest,
		},
	}

	if w.sink.OmitIntegrity {
		ref.Integrity = nil
	}

	if w.sink.OmitSizeBytes {
		ref.SizeBytes = 0
	}

	if w.sink.ProtectionResult != nil {
		protection := *w.sink.ProtectionResult
		ref.Protection = &protection
	}

	return ref, nil
}

// PutCount returns how many times PersistResult was invoked, including calls
// that failed via PutErr. It lets a size-limit test prove the runner failed an
// over-limit result BEFORE reaching the sink (PutCount == 0). It is a harness
// affordance for tests, not part of the Engine port.
func (s *ResultSink) PutCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.putCount
}

// OpenCount returns how many times OpenResultStream was invoked. It lets a
// store-mode test assert the streaming path was taken.
func (s *ResultSink) OpenCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.openCount
}

// StoredCount returns how many results were FINALIZED (a streaming writer's
// Close succeeded, or PersistResult stored bytes). An aborted store-mode run
// abandons its writer WITHOUT Close, so StoredCount stays 0 — a test uses this to
// prove an over-limit / failed run finalized no result.
func (s *ResultSink) StoredCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.results)
}

// WriteSizes returns a copy of the byte sizes of every streaming Write the
// engine issued, in order. A test asserts len(WriteSizes()) > 1 to prove the
// engine FLUSHED INCREMENTALLY rather than writing the whole result at once.
func (s *ResultSink) WriteSizes() []int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]int, len(s.writeSizes))
	copy(out, s.writeSizes)

	return out
}

// Get returns a defensive copy of the payload stored at path and whether it
// exists. It is a harness affordance for tests, not part of the Engine port.
func (s *ResultSink) Get(path string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, ok := s.results[path]
	if !ok {
		return nil, false
	}

	out := make([]byte, len(stored))
	copy(out, stored)

	return out, true
}
