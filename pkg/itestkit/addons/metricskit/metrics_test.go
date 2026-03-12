package metricskit

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func deterministicMetrics() *ChaosMetrics {
	m := NewChaosMetrics()
	m.TestStartTime = time.Unix(1700000000, 0)
	m.TestEndTime = m.TestStartTime.Add(10 * time.Second)
	m.ChaosStartTime = m.TestStartTime.Add(2 * time.Second)
	m.ChaosEndTime = m.ChaosStartTime.Add(4 * time.Second)

	m.RecordRequest(true, false, 10*time.Millisecond)
	m.RecordRequest(true, false, 20*time.Millisecond)
	m.RecordRequest(false, true, 50*time.Millisecond)
	m.RecordRequest(false, false, 100*time.Millisecond)
	m.RecordError("context deadline exceeded")
	m.RecordError("connection refused by upstream")

	return m
}

func TestChaosMetricsAndReporting(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "metrics record deterministic rates durations and percentiles",
			run: func(t *testing.T) {
				t.Parallel()

				metrics := deterministicMetrics()

				if got := metrics.GetTotalRequests(); got != 4 {
					t.Fatalf("expected total requests 4, got %d", got)
				}

				if got := metrics.GetFailedRequests(); got != 2 {
					t.Fatalf("expected failed requests 2, got %d", got)
				}

				if got := metrics.GetTimeoutRequests(); got != 1 {
					t.Fatalf("expected timeout requests 1, got %d", got)
				}

				if got := metrics.SuccessRate(); got != 50 {
					t.Fatalf("expected success rate 50, got %.2f", got)
				}

				if got := metrics.AverageLatency(); got != 45*time.Millisecond {
					t.Fatalf("expected average latency 45ms, got %v", got)
				}

				if got := metrics.GetMinLatency(); got != 10*time.Millisecond {
					t.Fatalf("expected min latency 10ms, got %v", got)
				}

				if got := metrics.P50(); got != 20*time.Millisecond {
					t.Fatalf("expected p50 20ms, got %v", got)
				}

				if got := metrics.P95(); got != 50*time.Millisecond {
					t.Fatalf("expected p95 50ms, got %v", got)
				}

				if got := metrics.P99(); got != 50*time.Millisecond {
					t.Fatalf("expected p99 50ms, got %v", got)
				}

				if got := metrics.P999(); got != 50*time.Millisecond {
					t.Fatalf("expected p999 50ms, got %v", got)
				}

				if got := metrics.ThroughputRPS(); got != 0.4 {
					t.Fatalf("expected throughput 0.4, got %.4f", got)
				}

				if got := metrics.SuccessfulThroughputRPS(); got != 0.2 {
					t.Fatalf("expected successful throughput 0.2, got %.4f", got)
				}

				if got := metrics.ChaosThroughputRPS(); got != 1 {
					t.Fatalf("expected chaos throughput 1, got %.4f", got)
				}

				if got := metrics.TestDuration(); got != 10*time.Second {
					t.Fatalf("expected test duration 10s, got %v", got)
				}

				if got := metrics.ChaosDuration(); got != 4*time.Second {
					t.Fatalf("expected chaos duration 4s, got %v", got)
				}

				if got := metrics.Percentile(-1); got != 0 {
					t.Fatalf("expected invalid percentile to return zero, got %v", got)
				}

				counts := metrics.GetErrorCounts()
				if counts[ErrorCategoryTimeout] != 1 || counts[ErrorCategoryRefused] != 1 {
					t.Fatalf("unexpected error counts: %#v", counts)
				}
			},
		},
		{
			name: "snapshot clone isolates mutable internals and reset clears state",
			run: func(t *testing.T) {
				t.Parallel()

				metrics := deterministicMetrics()
				_ = metrics.Percentile(50)

				snapshot, ok := metrics.Snapshot().(*ChaosMetrics)
				if !ok {
					t.Fatalf("expected snapshot to be a ChaosMetrics clone")
				}

				snapshot.Latencies[0] = time.Second
				snapshot.ErrorClassifier.RecordError("tls handshake failed")

				if metrics.Latencies[0] == time.Second {
					t.Fatalf("expected snapshot latencies to be isolated from original")
				}

				if metrics.GetErrorCounts()[ErrorCategoryTLS] != 0 {
					t.Fatalf("expected snapshot classifier to be isolated from original")
				}

				metrics.Reset()
				if metrics.GetTotalRequests() != 0 || metrics.GetFailedRequests() != 0 || metrics.GetTimeoutRequests() != 0 {
					t.Fatalf("expected reset to clear counters")
				}

				if metrics.GetMinLatency() != 0 || metrics.P50() != 0 {
					t.Fatalf("expected reset to clear latency state")
				}

				if len(metrics.GetErrorCounts()) != 0 {
					t.Fatalf("expected reset to clear error counts")
				}
			},
		},
		{
			name: "error classifier prioritizes categories and supports custom patterns",
			run: func(t *testing.T) {
				t.Parallel()

				classifier := NewErrorClassifier()
				classifier.RecordError("context deadline exceeded while calling upstream")
				classifier.RecordError("TLS handshake failed")
				classifier.AddPattern(ErrorCategoryUnknown, "mystery")
				classifier.RecordError("mystery failure")

				counts := classifier.GetCategoryCounts()
				if counts[ErrorCategoryTimeout] != 1 || counts[ErrorCategoryTLS] != 1 || counts[ErrorCategoryUnknown] != 1 {
					t.Fatalf("unexpected classifier counts: %#v", counts)
				}

				clone := classifier.Clone()
				clone.RecordError("connection reset by peer")
				if classifier.GetCategoryCounts()[ErrorCategoryReset] != 0 {
					t.Fatalf("expected clone mutations not to leak back to original")
				}

				classifier.Reset()
				if len(classifier.GetCategoryCounts()) != 0 {
					t.Fatalf("expected classifier reset to clear counts")
				}
			},
		},
		{
			name: "assertions and reporter summarize snapshot health",
			run: func(t *testing.T) {
				t.Parallel()

				metrics := deterministicMetrics()
				assertions := Assert(metrics).
					SuccessRateAbove(40).
					P99Below(60 * time.Millisecond).
					P95Below(60 * time.Millisecond).
					P50Below(25 * time.Millisecond).
					AverageLatencyBelow(50 * time.Millisecond).
					ThroughputAbove(0.3).
					TimeoutsBelow(1).
					FailuresBelow(2).
					MinRequestsReached(4)

				if !assertions.Passed() || assertions.Failed() {
					t.Fatalf("expected assertions to pass, got %#v", assertions.Results())
				}

				failing := NewAssertions(metrics.Snapshot()).SuccessRateAbove(80).FailuresBelow(1)
				if !failing.Failed() || len(failing.FailedResults()) != 2 {
					t.Fatalf("expected failing assertions to be tracked, got %#v", failing.FailedResults())
				}

				summary := failing.Summary()
				for _, fragment := range []string{"Assertions: 0/2 passed", "SuccessRateAbove", "FailuresBelow"} {
					if !strings.Contains(summary, fragment) {
						t.Fatalf("expected assertion summary to contain %q, got %q", fragment, summary)
					}
				}

				reporter := Report(metrics)
				var buf bytes.Buffer
				if err := reporter.WriteReport(&buf); err != nil {
					t.Fatalf("write report: %v", err)
				}

				report := buf.String()
				for _, fragment := range []string{"CHAOS TEST REPORT", "Total Requests:      4", "ERROR BREAKDOWN", "timeout:"} {
					if !strings.Contains(strings.ToLower(report), strings.ToLower(fragment)) {
						t.Fatalf("expected report to contain %q, got %q", fragment, report)
					}
				}

				if reporter.String() != report {
					t.Fatalf("expected String to reuse report output")
				}

				compact := reporter.CompactSummary()
				for _, fragment := range []string{"requests=4", "success_rate=50.00%", "p99=50ms", "chaos_duration=4s"} {
					if !strings.Contains(compact, fragment) {
						t.Fatalf("expected compact summary to contain %q, got %q", fragment, compact)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
