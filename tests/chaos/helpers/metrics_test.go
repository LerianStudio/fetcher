package helpers

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPercentile_EmptyLatencies(t *testing.T) {
	m := NewChaosMetrics()

	assert.Equal(t, time.Duration(0), m.P50())
	assert.Equal(t, time.Duration(0), m.P99())
}

func TestPercentile_SingleLatency(t *testing.T) {
	m := NewChaosMetrics()
	m.RecordRequest(true, false, 100*time.Millisecond)

	assert.Equal(t, 100*time.Millisecond, m.P50())
	assert.Equal(t, 100*time.Millisecond, m.P99())
}

func TestPercentile_MultipleLatencies(t *testing.T) {
	m := NewChaosMetrics()

	for i := 1; i <= 100; i++ {
		m.RecordRequest(true, false, time.Duration(i)*time.Millisecond)
	}

	p50 := m.P50()
	assert.GreaterOrEqual(t, p50, 49*time.Millisecond)
	assert.LessOrEqual(t, p50, 51*time.Millisecond)

	p90 := m.P90()
	assert.GreaterOrEqual(t, p90, 89*time.Millisecond)
	assert.LessOrEqual(t, p90, 91*time.Millisecond)

	p99 := m.P99()
	assert.GreaterOrEqual(t, p99, 98*time.Millisecond)
	assert.LessOrEqual(t, p99, 100*time.Millisecond)
}

func TestPercentile_CacheInvalidation(t *testing.T) {
	m := NewChaosMetrics()

	for i := 1; i <= 10; i++ {
		m.RecordRequest(true, false, time.Duration(i)*time.Millisecond)
	}

	p50First := m.P50()

	for i := 100; i <= 110; i++ {
		m.RecordRequest(true, false, time.Duration(i)*time.Millisecond)
	}

	p50Second := m.P50()

	assert.NotEqual(t, p50First, p50Second)
}

func TestPercentile_P999(t *testing.T) {
	m := NewChaosMetrics()

	for i := 1; i <= 1000; i++ {
		m.RecordRequest(true, false, time.Duration(i)*time.Millisecond)
	}

	p999 := m.P999()
	assert.GreaterOrEqual(t, p999, 990*time.Millisecond)
	assert.LessOrEqual(t, p999, 1000*time.Millisecond)
}

func TestPercentile_SnapshotPreservesPercentiles(t *testing.T) {
	m := NewChaosMetrics()

	for i := 1; i <= 100; i++ {
		m.RecordRequest(true, false, time.Duration(i)*time.Millisecond)
	}

	originalP50 := m.P50()
	snapshot := m.Snapshot()
	snapshotP50 := snapshot.P50()
	assert.Equal(t, originalP50, snapshotP50)
}

func TestPercentile_CacheConsistencyAfterPartialRecalc(t *testing.T) {
	// Regression test: ensures all percentiles are recalculated after data changes
	// Previously, querying P50 after new data would clear dirty flag, causing
	// subsequent P99 query to return stale cached value.
	m := NewChaosMetrics()

	// Baseline data - low latencies
	for i := 1; i <= 100; i++ {
		m.RecordRequest(true, false, time.Duration(i)*time.Millisecond)
	}

	// Populate cache for both P50 and P99
	p50Initial := m.P50()
	p99Initial := m.P99()

	assert.GreaterOrEqual(t, p50Initial, 49*time.Millisecond)
	assert.LessOrEqual(t, p99Initial, 100*time.Millisecond)

	// Add HIGH latency data (chaos scenario simulation)
	for i := 0; i < 100; i++ {
		m.RecordRequest(true, false, 1000*time.Millisecond)
	}

	// Query P50 first - this was triggering the bug by clearing dirty flag
	p50After := m.P50()

	// Query P99 - this should return FRESH value, not stale cached
	p99After := m.P99()

	// P99 should have increased significantly due to high-latency requests
	assert.Greater(t, p99After, p99Initial,
		"P99 should increase after high-latency requests")
	assert.Greater(t, p99After, 500*time.Millisecond,
		"P99 should reflect high-latency chaos requests, got %v", p99After)

	// P50 should also have changed
	assert.Greater(t, p50After, p50Initial,
		"P50 should increase after high-latency requests")
}

func TestPercentile_InvalidInputs(t *testing.T) {
	m := NewChaosMetrics()
	m.RecordRequest(true, false, 100*time.Millisecond)

	// Invalid percentiles should return 0
	assert.Equal(t, time.Duration(0), m.Percentile(-10),
		"Negative percentile should return 0")
	assert.Equal(t, time.Duration(0), m.Percentile(150),
		"Percentile > 100 should return 0")

	// Valid boundary values should work
	assert.Equal(t, 100*time.Millisecond, m.Percentile(0),
		"P0 should return min (only one sample)")
	assert.Equal(t, 100*time.Millisecond, m.Percentile(100),
		"P100 should return max (only one sample)")
}

func TestPercentile_Boundaries(t *testing.T) {
	m := NewChaosMetrics()
	for i := 1; i <= 100; i++ {
		m.RecordRequest(true, false, time.Duration(i)*time.Millisecond)
	}

	// P0 should be minimum
	p0 := m.Percentile(0)
	assert.Equal(t, 1*time.Millisecond, p0, "P0 should be minimum latency")

	// P100 should be maximum
	p100 := m.Percentile(100)
	assert.Equal(t, 100*time.Millisecond, p100, "P100 should be maximum latency")
}

func TestSLAThresholds_DefaultValues(t *testing.T) {
	thresholds := DefaultSLAThresholds()

	assert.Equal(t, 50.0, thresholds.MinSuccessRate)
	assert.Equal(t, 99.0, thresholds.RecoverySuccessRate)
	assert.Equal(t, 5*time.Second, thresholds.MaxP50Latency)
	assert.Equal(t, 30*time.Second, thresholds.MaxRecoveryTime)
}

func TestSLAThresholds_StrictValues(t *testing.T) {
	thresholds := StrictSLAThresholds()

	assert.Equal(t, 80.0, thresholds.MinSuccessRate)
	assert.Equal(t, 99.9, thresholds.RecoverySuccessRate)
	assert.Equal(t, 1*time.Second, thresholds.MaxP50Latency)
	assert.Equal(t, 15*time.Second, thresholds.MaxRecoveryTime)
}

func TestValidateAgainstSLA_PassingMetrics(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()

	// Add successful requests with low latency
	for i := 0; i < 100; i++ {
		m.RecordRequest(true, false, 100*time.Millisecond)
	}

	time.Sleep(10 * time.Millisecond)
	m.EndTest()

	assertions := NewChaosAssertions(t, m)
	result := assertions.ValidateAgainstSLA(DefaultSLAThresholds())

	assert.True(t, result.Passed)
	assert.Empty(t, result.Failures)
}

func TestValidateAgainstSLA_FailingSuccessRate(t *testing.T) {
	m := NewChaosMetrics()

	// Add 30 successful, 70 failed (30% success rate)
	for i := 0; i < 30; i++ {
		m.RecordRequest(true, false, 100*time.Millisecond)
	}
	for i := 0; i < 70; i++ {
		m.RecordRequest(false, false, 100*time.Millisecond)
	}

	// Use a test that won't fail
	mockT := &testing.T{}
	assertions := NewChaosAssertions(mockT, m)
	result := assertions.ValidateAgainstSLA(DefaultSLAThresholds())

	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.Failures)
	assert.Contains(t, result.Failures[0], "Success rate")
}

func TestValidateAgainstSLA_FailingLatency(t *testing.T) {
	m := NewChaosMetrics()

	// Add requests with high latency
	for i := 0; i < 100; i++ {
		m.RecordRequest(true, false, 10*time.Second)
	}

	mockT := &testing.T{}
	assertions := NewChaosAssertions(mockT, m)
	result := assertions.ValidateAgainstSLA(DefaultSLAThresholds())

	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.Failures)
}

func TestStabilityChecks_Recording(t *testing.T) {
	m := NewChaosMetrics()
	m.StartStabilityCheck()

	// Record some checks
	m.RecordStabilityCheck(true, 100.0, 10*time.Millisecond, "")
	m.RecordStabilityCheck(true, 99.0, 15*time.Millisecond, "")
	m.RecordStabilityCheck(false, 50.0, 100*time.Millisecond, "timeout")
	m.RecordStabilityCheck(true, 100.0, 10*time.Millisecond, "")

	m.EndStabilityCheck()

	checks := m.GetStabilityChecks()
	assert.Len(t, checks, 4)
	assert.Equal(t, 75.0, m.StabilityPassRate())
}

func TestStabilityChecks_ConsecutiveFailures(t *testing.T) {
	m := NewChaosMetrics()
	m.StartStabilityCheck()

	// Record pattern: pass, fail, fail, fail, pass
	m.RecordStabilityCheck(true, 100.0, 10*time.Millisecond, "")
	m.RecordStabilityCheck(false, 0.0, 0, "error1")
	m.RecordStabilityCheck(false, 0.0, 0, "error2")
	m.RecordStabilityCheck(false, 0.0, 0, "error3")
	m.RecordStabilityCheck(true, 100.0, 10*time.Millisecond, "")

	m.EndStabilityCheck()

	assert.Equal(t, 3, m.GetMaxConsecutiveFailures())
}

func TestStabilityChecks_Duration(t *testing.T) {
	m := NewChaosMetrics()
	m.StartStabilityCheck()

	// Simulate some time passing
	time.Sleep(50 * time.Millisecond)
	m.RecordStabilityCheck(true, 100.0, 10*time.Millisecond, "")

	time.Sleep(50 * time.Millisecond)
	m.EndStabilityCheck()

	duration := m.StabilityDuration()
	// Allow 10ms tolerance for timing variations in CI
	assert.GreaterOrEqual(t, duration, 90*time.Millisecond,
		"Stability duration should be at least ~100ms (with tolerance)")
}

func TestThroughputRPS_Calculation(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()

	// Record 100 requests
	for i := 0; i < 100; i++ {
		m.RecordRequest(true, false, 10*time.Millisecond)
	}

	// Wait a bit to have measurable duration
	time.Sleep(100 * time.Millisecond)
	m.EndTest()

	rps := m.ThroughputRPS()
	// Should be around 1000 RPS (100 requests / 0.1 seconds)
	// But actual time may vary, so just check it's positive
	assert.Greater(t, rps, 0.0)
}

func TestSuccessfulThroughputRPS(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()

	// Record 50 successful, 50 failed
	for i := 0; i < 50; i++ {
		m.RecordRequest(true, false, 10*time.Millisecond)
	}

	for i := 0; i < 50; i++ {
		m.RecordRequest(false, false, 10*time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	m.EndTest()

	totalRPS := m.ThroughputRPS()
	successRPS := m.SuccessfulThroughputRPS()

	// Successful should be about half of total
	assert.Greater(t, totalRPS, 0.0)
	assert.Greater(t, successRPS, 0.0)
	assert.Less(t, successRPS, totalRPS)
}

func TestChaosThroughputRPS(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()
	m.StartChaos()

	// Record requests
	for i := 0; i < 50; i++ {
		m.RecordRequest(true, false, 10*time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	m.EndChaos()
	m.EndTest()

	chaosRPS := m.ChaosThroughputRPS()
	assert.Greater(t, chaosRPS, 0.0)
}

func TestThroughputRPS_NoRequests(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()
	time.Sleep(10 * time.Millisecond)
	m.EndTest()

	// Should return 0 when no requests
	rps := m.ThroughputRPS()
	assert.Equal(t, 0.0, rps)
}

func TestThroughputRPS_ZeroDuration(t *testing.T) {
	m := NewChaosMetrics()
	// Don't start test - TestStartTime is zero

	rps := m.ThroughputRPS()
	assert.Equal(t, 0.0, rps)
}

func TestChaosThroughputRPS_NoChaosStarted(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()

	for i := 0; i < 50; i++ {
		m.RecordRequest(true, false, 10*time.Millisecond)
	}

	m.EndTest()
	// Note: No StartChaos/EndChaos called

	chaosRPS := m.ChaosThroughputRPS()
	assert.Equal(t, 0.0, chaosRPS)
}

func TestChaosThroughputRPS_DuringActiveChaos(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()
	m.StartChaos()

	for i := 0; i < 50; i++ {
		m.RecordRequest(true, false, 10*time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)
	// Chaos still active, EndChaos not called

	chaosRPS := m.ChaosThroughputRPS()
	// Should return in-progress calculation since ChaosStartTime is set
	assert.Greater(t, chaosRPS, 0.0,
		"Should calculate in-progress chaos RPS")
}

func TestSuccessfulThroughputRPS_ZeroDuration(t *testing.T) {
	m := NewChaosMetrics()
	// Don't start test - TestStartTime is zero

	rps := m.SuccessfulThroughputRPS()
	assert.Equal(t, 0.0, rps)
}

func TestSuccessfulThroughputRPS_NoSuccessfulRequests(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()

	// Only failed requests
	for i := 0; i < 10; i++ {
		m.RecordRequest(false, false, 10*time.Millisecond)
	}

	time.Sleep(10 * time.Millisecond)
	m.EndTest()

	rps := m.SuccessfulThroughputRPS()
	assert.Equal(t, 0.0, rps)
}

func TestValidateAgainstSLA_ThroughputValidation(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()

	// Very few requests over measurable time
	for i := 0; i < 5; i++ {
		m.RecordRequest(true, false, 10*time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	m.EndTest()

	// Create thresholds requiring high throughput
	thresholds := DefaultSLAThresholds()
	thresholds.MinThroughputRPS = 1000.0 // Require 1000 RPS

	mockT := &testing.T{}
	assertions := NewChaosAssertions(mockT, m)
	result := assertions.ValidateAgainstSLA(thresholds)

	// Should fail due to low throughput
	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.Failures)

	// Find throughput failure
	foundThroughputFailure := false
	for _, failure := range result.Failures {
		if strings.Contains(failure, "Throughput") {
			foundThroughputFailure = true
			break
		}
	}

	assert.True(t, foundThroughputFailure, "Should have throughput failure")
}

func TestConcurrentRecordRequest(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()

	const numGoroutines = 10
	const requestsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				success := j%2 == 0
				timeout := j%5 == 0
				latency := time.Duration(10+id) * time.Millisecond
				m.RecordRequest(success, timeout, latency)
			}
		}(i)
	}

	wg.Wait()
	m.EndTest()

	// Verify total requests
	expectedTotal := numGoroutines * requestsPerGoroutine
	assert.Equal(t, expectedTotal, m.GetTotalRequests(),
		"Total requests should match expected count after concurrent access")

	// Verify no data corruption - metrics should be consistent
	assert.GreaterOrEqual(t, m.SuccessRate(), 0.0)
	assert.LessOrEqual(t, m.SuccessRate(), 100.0)
}

func TestConcurrentPercentileAccess(t *testing.T) {
	m := NewChaosMetrics()

	// Add some baseline data
	for i := 1; i <= 100; i++ {
		m.RecordRequest(true, false, time.Duration(i)*time.Millisecond)
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrently access percentiles while also recording new requests
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				// Mix reads and writes
				if j%2 == 0 {
					_ = m.P50()
					_ = m.P99()
				} else {
					m.RecordRequest(true, false, time.Duration(50+id)*time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify percentiles are still valid
	p50 := m.P50()
	p99 := m.P99()

	assert.Greater(t, p50, time.Duration(0), "P50 should be positive")
	assert.Greater(t, p99, time.Duration(0), "P99 should be positive")
	assert.GreaterOrEqual(t, p99, p50, "P99 should be >= P50")
}

func TestConcurrentStabilityChecks(t *testing.T) {
	m := NewChaosMetrics()
	m.StartStabilityCheck()

	const numGoroutines = 5
	const checksPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < checksPerGoroutine; j++ {
				success := j%3 != 0
				m.RecordStabilityCheck(success, 100.0, 10*time.Millisecond, "")
			}
		}(i)
	}

	wg.Wait()
	m.EndStabilityCheck()

	// Verify all stability checks were recorded
	checks := m.GetStabilityChecks()
	expectedChecks := numGoroutines * checksPerGoroutine
	assert.Equal(t, expectedChecks, len(checks),
		"All stability checks should be recorded")

	// Verify pass rate is valid
	passRate := m.StabilityPassRate()
	assert.GreaterOrEqual(t, passRate, 0.0)
	assert.LessOrEqual(t, passRate, 100.0)
}

func TestConcurrentSnapshot(t *testing.T) {
	m := NewChaosMetrics()
	m.StartTest()

	const numGoroutines = 5
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Half writers, half readers

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				m.RecordRequest(true, false, time.Duration(10+id)*time.Millisecond)
			}
		}(i)
	}

	// Readers (snapshot)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				snapshot := m.Snapshot()
				// Verify snapshot is consistent
				assert.GreaterOrEqual(t, snapshot.TotalRequests, 0)
				assert.GreaterOrEqual(t, snapshot.SuccessfulRequests, 0)
				assert.LessOrEqual(t, snapshot.SuccessfulRequests, snapshot.TotalRequests)
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()
	m.EndTest()
}
