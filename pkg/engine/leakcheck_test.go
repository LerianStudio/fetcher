// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"runtime"
	"testing"
	"time"
)

// verifyNoGoroutineLeak is a stdlib stand-in for go.uber.org/goleak.VerifyNone.
//
// The engine module's go.mod require block must stay empty so embedding consumers
// (e.g. the midaz reporter) inherit ZERO third-party dependencies — and a require is
// the consumer-facing contract regardless of whether the dep is test-only. goleak,
// as a direct require, would land in every consumer's go.sum, so the leak guard is
// reimplemented here against runtime alone.
//
// Usage: call at the top of a NON-parallel test and defer the returned cleanup:
//
//	defer verifyNoGoroutineLeak(t)()
//
// It captures the goroutine count baseline at call time, then polls at cleanup until
// the count settles back to (or below) the baseline, tolerating scheduler/GC churn.
// If goroutines are still outstanding after the settle window, it fails with a full
// stack dump. Tests using it must NOT run in parallel: a quiet baseline is required
// so sibling goroutines don't produce false positives (same constraint goleak has).
func verifyNoGoroutineLeak(t *testing.T) func() {
	t.Helper()

	baseline := runtime.NumGoroutine()

	return func() {
		const settleWindow = 2 * time.Second

		deadline := time.Now().Add(settleWindow)
		for {
			runtime.Gosched()

			current := runtime.NumGoroutine()
			if current <= baseline {
				return
			}

			if time.Now().After(deadline) {
				buf := make([]byte, 1<<20)
				buf = buf[:runtime.Stack(buf, true)]
				t.Fatalf(
					"goroutine leak: baseline %d, still %d running after %s settle window\n%s",
					baseline, current, settleWindow, buf,
				)
			}

			time.Sleep(10 * time.Millisecond)
		}
	}
}
