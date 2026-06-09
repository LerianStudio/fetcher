// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
)

// TestEagerCursor_CleanDrain_ReportsNoError verifies the baseline contract: a
// cursor driven to natural end-of-stream surfaces nil from Err, so a caller can
// treat the drained result as a completed extraction.
func TestEagerCursor_CleanDrain_ReportsNoError(t *testing.T) {
	t.Parallel()

	cursor := engine.NewEagerCursor(map[string][]map[string]any{
		"public.t": {{"id": 1}, {"id": 2}},
	})

	ctx := context.Background()

	var rows int
	for cursor.Next(ctx) {
		rows++
	}

	if rows != 2 {
		t.Fatalf("expected to drain 2 rows, got %d", rows)
	}

	if err := cursor.Err(); err != nil {
		t.Fatalf("clean drain must report no error, got %v", err)
	}
}

// TestEagerCursor_Cancellation_SurfacesViaErr is the regression guard for the
// load-bearing invariant: a context cancelled mid-iteration must be observable
// through Err, NOT silently indistinguishable from a clean end-of-stream. An
// external embedder that drives the cursor without the runner's own ctx.Err()
// belt-and-suspenders relies on Err to tell an aborted stream from a complete one.
func TestEagerCursor_Cancellation_SurfacesViaErr(t *testing.T) {
	t.Parallel()

	cursor := engine.NewEagerCursor(map[string][]map[string]any{
		"public.t": {{"id": 1}, {"id": 2}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if cursor.Next(ctx) {
		t.Fatal("Next must return false once the context is cancelled")
	}

	err := cursor.Err()
	if err == nil {
		t.Fatal("cancellation must surface via Err, not be swallowed as EOF")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Err must report the context cancellation, got %v", err)
	}
}

// TestEagerCursor_Close_ResetsError confirms Close clears the recorded error so a
// pooled/reused cursor value does not leak a prior cancellation into a fresh run.
func TestEagerCursor_Close_ResetsError(t *testing.T) {
	t.Parallel()

	cursor := engine.NewEagerCursor(map[string][]map[string]any{
		"public.t": {{"id": 1}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cursor.Next(ctx)

	if cursor.Err() == nil {
		t.Fatal("precondition: cancellation should have been recorded")
	}

	if err := cursor.Close(context.Background()); err != nil {
		t.Fatalf("Close must not error, got %v", err)
	}

	if err := cursor.Err(); err != nil {
		t.Fatalf("Close must reset the recorded error, got %v", err)
	}
}
