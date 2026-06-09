// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
)

func TestResultSink_RoundTripsExactBytesByLogicalReference(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenant := testTenant(t)
	sink := memory.NewResultSink()

	payload := []byte(`{"rows":[{"id":1},{"id":2}]}`)
	ref, err := sink.PersistResult(ctx, tenant, payload)
	if err != nil {
		t.Fatalf("PersistResult: unexpected error: %v", err)
	}

	if ref.Path == "" {
		t.Fatalf("PersistResult: expected non-empty logical reference path")
	}
	if ref.SizeBytes != int64(len(payload)) {
		t.Fatalf("PersistResult: SizeBytes = %d, want %d", ref.SizeBytes, len(payload))
	}

	// Canonical integrity: the in-memory double computes a content digest and
	// records it as a canonical integrity digest, NOT as a bare HMAC field.
	if ref.Integrity == nil {
		t.Fatalf("PersistResult: expected canonical integrity metadata")
	}
	if ref.Integrity.Algorithm == "" {
		t.Fatalf("PersistResult: integrity must name its algorithm")
	}
	if !ref.Integrity.IsPresent() {
		t.Fatalf("PersistResult: integrity must be present (digest or signature)")
	}

	stored, found := sink.Get(ref.Path)
	if !found {
		t.Fatalf("Get: expected stored payload to be retrievable by logical reference")
	}
	if string(stored) != string(payload) {
		t.Fatalf("Get: payload = %q, want exact bytes %q", stored, payload)
	}
}

func TestResultSink_PutError_ReturnsSafeStorageCategoryError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tenant := testTenant(t)
	sink := memory.NewResultSink()

	// Fault injection: force the next persist to fail, mirroring the cache/store
	// doubles' GetErr/PutErr idiom so runner tests can drive sink failures.
	sink.PutErr = errors.New("disk full: secret-bearing detail must not leak")

	_, err := sink.PersistResult(ctx, tenant, []byte("payload"))
	if err == nil {
		t.Fatalf("PersistResult: expected an error when PutErr is set")
	}

	var engineErr *engine.EngineError
	if !errors.As(err, &engineErr) {
		t.Fatalf("PersistResult: error = %T, want *engine.EngineError", err)
	}
	if engineErr.Category != engine.CategoryUnavailable {
		t.Fatalf("PersistResult: error category = %q, want %q", engineErr.Category, engine.CategoryUnavailable)
	}

	// The safe Engine error must NOT echo the injected (potentially sensitive)
	// underlying detail.
	if errors.Is(err, sink.PutErr) {
		t.Fatalf("PersistResult: safe Engine error must not wrap the raw sink error")
	}
}
