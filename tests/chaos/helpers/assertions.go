package helpers

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ChaosType defines the type of chaos to inject.
type ChaosType string

const (
	ChaosTypeLatency   ChaosType = "latency"
	ChaosTypeTimeout   ChaosType = "timeout"
	ChaosTypeBandwidth ChaosType = "bandwidth"
	ChaosTypeSlowClose ChaosType = "slow_close"
	ChaosTypeResetPeer ChaosType = "reset_peer"
)

// ChaosDirection defines the direction of chaos injection.
type ChaosDirection string

const (
	ChaosDirectionDownstream ChaosDirection = "downstream"
	ChaosDirectionUpstream   ChaosDirection = "upstream"
)

// ChaosConfig defines configuration for chaos injection.
type ChaosConfig struct {
	Type       ChaosType
	Direction  ChaosDirection
	Toxicity   float32 // 0.0 to 1.0, probability of toxic being applied
	Attributes map[string]interface{}
}

// ChaosAssertions provides assertion helpers for chaos tests.
type ChaosAssertions struct {
	t       *testing.T
	metrics *ChaosMetrics
}

// NewChaosAssertions creates a new ChaosAssertions instance.
func NewChaosAssertions(t *testing.T, metrics *ChaosMetrics) *ChaosAssertions {
	return &ChaosAssertions{
		t:       t,
		metrics: metrics,
	}
}

// AssertSuccessRateAbove asserts that success rate is above threshold.
func (a *ChaosAssertions) AssertSuccessRateAbove(threshold float64) {
	a.t.Helper()
	rate := a.metrics.SuccessRate()
	assert.Greater(a.t, rate, threshold,
		"Success rate %.2f%% should be above %.2f%%", rate, threshold)
}

// AssertSuccessRateEquals asserts that success rate equals expected value.
func (a *ChaosAssertions) AssertSuccessRateEquals(expected float64, tolerance float64) {
	a.t.Helper()
	rate := a.metrics.SuccessRate()
	assert.InDelta(a.t, expected, rate, tolerance,
		"Success rate %.2f%% should be approximately %.2f%% (±%.2f%%)", rate, expected, tolerance)
}

// AssertRecoveryWithin asserts that system recovered within timeout.
func (a *ChaosAssertions) AssertRecoveryWithin(timeout time.Duration) {
	a.t.Helper()
	recoveryTime := a.metrics.GetRecoveryTime()
	assert.LessOrEqual(a.t, recoveryTime, timeout,
		"Recovery time %v should be within %v", recoveryTime, timeout)
}

// AssertNoFailures asserts that there were no failures.
func (a *ChaosAssertions) AssertNoFailures() {
	a.t.Helper()
	failed := a.metrics.GetFailedRequests()
	assert.Equal(a.t, 0, failed,
		"Expected no failures, got %d", failed)
}

// AssertFailuresOccurred asserts that failures occurred (expected during chaos).
func (a *ChaosAssertions) AssertFailuresOccurred() {
	a.t.Helper()
	assert.Greater(a.t, a.metrics.GetFailedRequests(), 0,
		"Expected failures during chaos, got none")
}

// AssertTimeoutsOccurred asserts that timeouts occurred (expected during chaos).
func (a *ChaosAssertions) AssertTimeoutsOccurred() {
	a.t.Helper()
	assert.Greater(a.t, a.metrics.GetTimeoutRequests(), 0,
		"Expected timeouts during chaos, got none")
}

// AssertLatencyIncreased asserts that latency increased compared to baseline.
func (a *ChaosAssertions) AssertLatencyIncreased(baseline time.Duration, minIncrease time.Duration) {
	a.t.Helper()
	avgLatency := a.metrics.AverageLatency()
	increase := avgLatency - baseline
	assert.GreaterOrEqual(a.t, increase, minIncrease,
		"Latency increase %v should be at least %v (baseline: %v, actual: %v)",
		increase, minIncrease, baseline, avgLatency)
}

// AssertLatencyWithin asserts that average latency is within bounds.
func (a *ChaosAssertions) AssertLatencyWithin(maxLatency time.Duration) {
	a.t.Helper()
	avgLatency := a.metrics.AverageLatency()
	assert.LessOrEqual(a.t, avgLatency, maxLatency,
		"Average latency %v should be within %v", avgLatency, maxLatency)
}

// AssertP50Within asserts that the 50th percentile latency is within bounds.
func (a *ChaosAssertions) AssertP50Within(maxLatency time.Duration) {
	a.t.Helper()
	p50 := a.metrics.P50()
	assert.LessOrEqual(a.t, p50, maxLatency,
		"P50 latency %v should be within %v", p50, maxLatency)
}

// AssertP90Within asserts that the 90th percentile latency is within bounds.
func (a *ChaosAssertions) AssertP90Within(maxLatency time.Duration) {
	a.t.Helper()
	p90 := a.metrics.P90()
	assert.LessOrEqual(a.t, p90, maxLatency,
		"P90 latency %v should be within %v", p90, maxLatency)
}

// AssertP95Within asserts that the 95th percentile latency is within bounds.
func (a *ChaosAssertions) AssertP95Within(maxLatency time.Duration) {
	a.t.Helper()
	p95 := a.metrics.P95()
	assert.LessOrEqual(a.t, p95, maxLatency,
		"P95 latency %v should be within %v", p95, maxLatency)
}

// AssertP99Within asserts that the 99th percentile latency is within bounds.
func (a *ChaosAssertions) AssertP99Within(maxLatency time.Duration) {
	a.t.Helper()
	p99 := a.metrics.P99()
	assert.LessOrEqual(a.t, p99, maxLatency,
		"P99 latency %v should be within %v", p99, maxLatency)
}

// AssertP999Within asserts that the 99.9th percentile latency is within bounds.
func (a *ChaosAssertions) AssertP999Within(maxLatency time.Duration) {
	a.t.Helper()
	p999 := a.metrics.P999()
	assert.LessOrEqual(a.t, p999, maxLatency,
		"P99.9 latency %v should be within %v", p999, maxLatency)
}

// AssertPercentileWithin asserts that a specific percentile latency is within bounds.
func (a *ChaosAssertions) AssertPercentileWithin(percentile float64, maxLatency time.Duration) {
	a.t.Helper()
	p := a.metrics.Percentile(percentile)
	assert.LessOrEqual(a.t, p, maxLatency,
		"P%.1f latency %v should be within %v", percentile, p, maxLatency)
}

// AssertRequestsProcessed asserts that at least minRequests were processed.
func (a *ChaosAssertions) AssertRequestsProcessed(minRequests int) {
	a.t.Helper()
	total := a.metrics.GetTotalRequests()
	assert.GreaterOrEqual(a.t, total, minRequests,
		"Expected at least %d requests, got %d", minRequests, total)
}

// SteadyStateResult holds the result of steady state measurement.
type SteadyStateResult struct {
	SuccessRate    float64
	AverageLatency time.Duration
	RequestCount   int
}

// MeasureSteadyState captures baseline metrics before chaos injection.
// Uses Snapshot() for atomic capture of all metrics.
func MeasureSteadyState(metrics *ChaosMetrics) SteadyStateResult {
	snapshot := metrics.Snapshot()
	return SteadyStateResult{
		SuccessRate:    snapshot.SuccessRate(),
		AverageLatency: snapshot.AverageLatency(),
		RequestCount:   snapshot.TotalRequests,
	}
}

// AssertSteadyStateRestored asserts that system returned to steady state.
func (a *ChaosAssertions) AssertSteadyStateRestored(baseline SteadyStateResult, tolerance float64) {
	a.t.Helper()
	currentRate := a.metrics.SuccessRate()
	assert.InDelta(a.t, baseline.SuccessRate, currentRate, tolerance,
		"Success rate should return to steady state (baseline: %.2f%%, current: %.2f%%)",
		baseline.SuccessRate, currentRate)
}

// DocumentHypothesis logs the test hypothesis for documentation.
func DocumentHypothesis(t *testing.T, hypothesis string) {
	t.Helper()
	t.Logf("HYPOTHESIS: %s", hypothesis)
}

// DocumentResult logs the test result with metrics.
func DocumentResult(t *testing.T, metrics *ChaosMetrics, result string) {
	t.Helper()
	snapshot := metrics.Snapshot()
	t.Logf("RESULT: %s", result)
	t.Logf("  Total Requests: %d", snapshot.TotalRequests)
	t.Logf("  Success Rate: %.2f%%", snapshot.SuccessRate())
	t.Logf("  Failed Requests: %d", snapshot.FailedRequests)
	t.Logf("  Timeout Requests: %d", snapshot.TimeoutRequests)
	t.Logf("  Average Latency: %v", snapshot.AverageLatency())
	t.Logf("  P50 Latency: %v", snapshot.P50())
	t.Logf("  P95 Latency: %v", snapshot.P95())
	t.Logf("  P99 Latency: %v", snapshot.P99())
	t.Logf("  Throughput: %.2f RPS", snapshot.ThroughputRPS())
	t.Logf("  Chaos Duration: %v", snapshot.ChaosDuration())
	t.Logf("  Recovery Time: %v", snapshot.RecoveryTime)
}

// ChaosTestCase represents a structured chaos test case.
type ChaosTestCase struct {
	Name        string
	Hypothesis  string
	ChaosConfig ChaosConfig
	Validation  func(t *testing.T, metrics *ChaosMetrics)
}

// FormatHypothesis creates a standardized hypothesis string.
func FormatHypothesis(expectedBehavior, whenCondition string) string {
	return fmt.Sprintf("The system should %s when %s", expectedBehavior, whenCondition)
}

// ValidateSLA validates all metrics against SLA thresholds and returns detailed results.
type SLAValidationResult struct {
	Passed   bool
	Failures []string
	Warnings []string
}

// ValidateAgainstSLA checks metrics against SLA thresholds.
func (a *ChaosAssertions) ValidateAgainstSLA(thresholds SLAThresholds) SLAValidationResult {
	a.t.Helper()
	result := SLAValidationResult{Passed: true}

	snapshot := a.metrics.Snapshot()

	// Success rate validation
	if snapshot.TotalRequests > 0 {
		successRate := snapshot.SuccessRate()
		if successRate < thresholds.MinSuccessRate {
			result.Passed = false
			result.Failures = append(result.Failures,
				fmt.Sprintf("Success rate %.2f%% below threshold %.2f%%", successRate, thresholds.MinSuccessRate))
		}
	}

	// Latency validations (only if we have latency data)
	if len(snapshot.Latencies) > 0 {
		if p50 := snapshot.P50(); p50 > thresholds.MaxP50Latency {
			result.Passed = false
			result.Failures = append(result.Failures,
				fmt.Sprintf("P50 latency %v exceeds threshold %v", p50, thresholds.MaxP50Latency))
		}

		if p95 := snapshot.P95(); p95 > thresholds.MaxP95Latency {
			result.Passed = false
			result.Failures = append(result.Failures,
				fmt.Sprintf("P95 latency %v exceeds threshold %v", p95, thresholds.MaxP95Latency))
		}

		if p99 := snapshot.P99(); p99 > thresholds.MaxP99Latency {
			result.Passed = false
			result.Failures = append(result.Failures,
				fmt.Sprintf("P99 latency %v exceeds threshold %v", p99, thresholds.MaxP99Latency))
		}

		// P99.9 is treated as warning rather than failure because:
		// - P99.9 requires 1000+ samples for statistical significance
		// - Chaos tests may have insufficient samples for reliable P99.9
		// - This provides visibility into tail latency without failing on edge cases
		if p999 := snapshot.P999(); p999 > thresholds.MaxP999Latency {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("P99.9 latency %v exceeds threshold %v", p999, thresholds.MaxP999Latency))
		}
	}

	// Recovery time validation
	if recoveryTime := snapshot.RecoveryTime; recoveryTime > 0 && recoveryTime > thresholds.MaxRecoveryTime {
		result.Passed = false
		result.Failures = append(result.Failures,
			fmt.Sprintf("Recovery time %v exceeds threshold %v", recoveryTime, thresholds.MaxRecoveryTime))
	}

	// Throughput validation (Phase 5)
	if thresholds.MinThroughputRPS > 0 && snapshot.TotalRequests > 0 {
		throughput := snapshot.ThroughputRPS()
		if throughput < thresholds.MinThroughputRPS {
			result.Passed = false
			result.Failures = append(result.Failures,
				fmt.Sprintf("Throughput %.2f RPS below threshold %.2f RPS",
					throughput, thresholds.MinThroughputRPS))
		}
	}

	return result
}

// AssertSLAMet asserts that all SLA thresholds are met.
func (a *ChaosAssertions) AssertSLAMet(thresholds SLAThresholds) {
	a.t.Helper()
	result := a.ValidateAgainstSLA(thresholds)

	// Log warnings
	for _, warning := range result.Warnings {
		a.t.Logf("WARNING: %s", warning)
	}

	// Assert no failures
	if !result.Passed {
		for _, failure := range result.Failures {
			a.t.Errorf("SLA VIOLATION: %s", failure)
		}
	}
}

// AssertMinSuccessRate asserts minimum success rate during chaos.
func (a *ChaosAssertions) AssertMinSuccessRate(minRate float64) {
	a.t.Helper()
	rate := a.metrics.SuccessRate()
	assert.GreaterOrEqual(a.t, rate, minRate,
		"Success rate %.2f%% should be at least %.2f%%", rate, minRate)
}

// AssertRecoverySuccessRate asserts success rate meets recovery threshold.
func (a *ChaosAssertions) AssertRecoverySuccessRate(minRate float64) {
	a.t.Helper()
	rate := a.metrics.SuccessRate()
	assert.GreaterOrEqual(a.t, rate, minRate,
		"Recovery success rate %.2f%% should be at least %.2f%%", rate, minRate)
}

// AssertStabilityMaintained asserts that the system remained stable during the stability check period.
func (a *ChaosAssertions) AssertStabilityMaintained(minPassRate float64, maxConsecutiveFailures int) {
	a.t.Helper()

	passRate := a.metrics.StabilityPassRate()
	assert.GreaterOrEqual(a.t, passRate, minPassRate,
		"Stability pass rate %.2f%% should be at least %.2f%%", passRate, minPassRate)

	maxFails := a.metrics.GetMaxConsecutiveFailures()
	assert.LessOrEqual(a.t, maxFails, maxConsecutiveFailures,
		"Max consecutive failures %d should not exceed %d", maxFails, maxConsecutiveFailures)
}

// AssertStabilityDuration asserts that stability was maintained for at least the given duration.
func (a *ChaosAssertions) AssertStabilityDuration(minDuration time.Duration) {
	a.t.Helper()
	duration := a.metrics.StabilityDuration()
	assert.GreaterOrEqual(a.t, duration, minDuration,
		"Stability duration %v should be at least %v", duration, minDuration)
}

// AssertRecoveryWithStability asserts recovery within time AND sustained stability.
func (a *ChaosAssertions) AssertRecoveryWithStability(maxRecoveryTime time.Duration, thresholds SLAThresholds) {
	a.t.Helper()

	// Assert recovery time
	recoveryTime := a.metrics.GetRecoveryTime()
	assert.LessOrEqual(a.t, recoveryTime, maxRecoveryTime,
		"Recovery time %v should be within %v", recoveryTime, maxRecoveryTime)

	// Assert stability maintained
	a.AssertStabilityMaintained(100.0-float64(thresholds.MaxConsecutiveFailures)*10, thresholds.MaxConsecutiveFailures)

	// Assert stability duration
	a.AssertStabilityDuration(thresholds.StabilityDuration)
}

// AssertNoTimeoutErrors asserts that no timeout errors occurred.
func (a *ChaosAssertions) AssertNoTimeoutErrors() {
	a.t.Helper()
	counts := a.metrics.GetErrorCounts()
	assert.Equal(a.t, 0, counts[ErrorCategoryTimeout],
		"Expected no timeout errors, got %d", counts[ErrorCategoryTimeout])
}

// AssertTimeoutErrorsExpected asserts that timeout errors occurred (expected during timeout chaos).
func (a *ChaosAssertions) AssertTimeoutErrorsExpected() {
	a.t.Helper()
	counts := a.metrics.GetErrorCounts()
	assert.Greater(a.t, counts[ErrorCategoryTimeout], 0,
		"Expected timeout errors during timeout chaos, got none")
}

// AssertConnectionErrorsExpected asserts that connection errors occurred.
func (a *ChaosAssertions) AssertConnectionErrorsExpected() {
	a.t.Helper()
	counts := a.metrics.GetErrorCounts()
	assert.Greater(a.t, counts[ErrorCategoryConnection], 0,
		"Expected connection errors during chaos, got none")
}

// AssertErrorCategoryCount asserts specific error category count.
func (a *ChaosAssertions) AssertErrorCategoryCount(category ErrorCategory, expectedMin, expectedMax int) {
	a.t.Helper()
	counts := a.metrics.GetErrorCounts()
	actual := counts[category]
	assert.GreaterOrEqual(a.t, actual, expectedMin,
		"Error category %s count %d should be at least %d", category, actual, expectedMin)
	assert.LessOrEqual(a.t, actual, expectedMax,
		"Error category %s count %d should be at most %d", category, actual, expectedMax)
}

// DocumentErrorBreakdown logs error classification details.
func DocumentErrorBreakdown(t *testing.T, metrics *ChaosMetrics) {
	t.Helper()

	counts := metrics.GetErrorCounts()

	t.Log("ERROR BREAKDOWN:")

	for category, count := range counts {
		if count > 0 {
			t.Logf("  %s: %d", category, count)
		}
	}
}

// AssertMinThroughput asserts minimum requests per second.
func (a *ChaosAssertions) AssertMinThroughput(minRPS float64) {
	a.t.Helper()

	rps := a.metrics.ThroughputRPS()
	assert.GreaterOrEqual(a.t, rps, minRPS,
		"Throughput %.2f RPS should be at least %.2f RPS", rps, minRPS)
}

// AssertSuccessfulThroughput asserts minimum successful requests per second.
func (a *ChaosAssertions) AssertSuccessfulThroughput(minRPS float64) {
	a.t.Helper()

	rps := a.metrics.SuccessfulThroughputRPS()
	assert.GreaterOrEqual(a.t, rps, minRPS,
		"Successful throughput %.2f RPS should be at least %.2f RPS", rps, minRPS)
}
