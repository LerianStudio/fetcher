// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/LerianStudio/fetcher/pkg/engine"
)

// ResultSink is an in-memory engine.ResultSink. It stores serialized result
// payloads in a mutex-protected map keyed by a content-derived path and lets
// tests read them back through Get. It performs no I/O and persists nothing
// beyond the process.
type ResultSink struct {
	mu      sync.RWMutex
	results map[string][]byte
	seq     int
}

// NewResultSink returns an empty in-memory result sink.
func NewResultSink() *ResultSink {
	return &ResultSink{
		results: make(map[string][]byte),
	}
}

// PersistResult implements engine.ResultSink. It stores a defensive copy of the
// payload under a deterministic, collision-resistant path and returns a
// secret-free reference carrying the path, an integrity HMAC placeholder
// (content hash), and the byte size.
func (s *ResultSink) PersistResult(
	_ context.Context,
	tenant engine.TenantContext,
	payload []byte,
) (engine.ResultReference, error) {
	stored := make([]byte, len(payload))
	copy(stored, payload)

	sum := sha256.Sum256(stored)
	digest := hex.EncodeToString(sum[:])

	s.mu.Lock()
	defer s.mu.Unlock()

	// The path embeds the tenant scope and a monotonic sequence so distinct
	// writes never collide even when payloads are identical.
	s.seq++
	path := "memory://" + tenant.OrganizationID + "/" + tenant.ProductName + "/" + digest + "-" + itoaInternal(s.seq)
	s.results[path] = stored

	return engine.ResultReference{
		Path:      path,
		HMAC:      digest,
		SizeBytes: int64(len(stored)),
	}, nil
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

// itoaInternal converts a non-negative int to its decimal string without
// importing strconv, keeping the harness's stdlib surface minimal.
func itoaInternal(n int) string {
	if n == 0 {
		return "0"
	}

	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	return string(digits)
}
