// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"sync"

	"github.com/LerianStudio/fetcher/pkg/engine"
)

// ResultSink is an in-memory engine.ResultSink. It stores serialized result
// payloads in a mutex-protected map keyed by a logical, content-derived path and
// lets tests read them back through Get. It performs no I/O and persists nothing
// beyond the process.
//
// PutErr is a test affordance mirroring the cache/store doubles' GetErr/PutErr
// idiom: when set, PersistResult fails so runner tests (ST-T007-02/03/04) can
// force a sink failure deterministically. It is inert (nil) by default.
type ResultSink struct {
	mu      sync.RWMutex
	results map[string][]byte
	seq     int

	// PutErr, when non-nil, makes PersistResult return a safe storage-category
	// Engine error. The injected error is NOT echoed across the boundary so no
	// sensitive underlying detail leaks.
	PutErr error

	// ProtectionResult, when non-nil, is the canonical result protection metadata
	// the sink reports on the returned reference. It models an adapter that
	// encrypts the stored bytes and stamps its own protection state; the Engine
	// MUST preserve exactly this metadata (after validating appliedBy). It is inert
	// (nil) by default, modelling an unencrypted store.
	ProtectionResult *engine.ResultProtection
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

	if s.ProtectionResult != nil {
		protection := *s.ProtectionResult
		ref.Protection = &protection
	}

	return ref, nil
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

// RecordedEvent captures one emitted lifecycle event together with the tenant
// it was emitted for, so tests can assert on emission.
type RecordedEvent struct {
	Tenant engine.TenantContext
	State  engine.ExecutionState
}

// EventSink is an in-memory engine.EventSink. It appends every emitted event to
// a mutex-protected slice that tests inspect through Events.
type EventSink struct {
	mu     sync.RWMutex
	events []RecordedEvent
}

// NewEventSink returns an empty in-memory event sink.
func NewEventSink() *EventSink {
	return &EventSink{}
}

// Emit implements engine.EventSink by recording the lifecycle event.
func (s *EventSink) Emit(
	_ context.Context,
	tenant engine.TenantContext,
	state engine.ExecutionState,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, RecordedEvent{Tenant: tenant, State: state})

	return nil
}

// Events returns a copy of the recorded events in emission order. It is a
// harness affordance for tests, not part of the Engine port.
func (s *EventSink) Events() []RecordedEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]RecordedEvent, len(s.events))
	copy(out, s.events)

	return out
}
