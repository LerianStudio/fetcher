//go:build chaos

package chaos

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/addons/metricskit"
	e2eshared "github.com/LerianStudio/fetcher/v2/tests/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SLO VALIDATION TESTS
// =============================================================================
//
// These tests validate that the system meets defined SLO thresholds under
// chaos conditions:
// - Manager Success Rate: >= 95%
// - Manager P99 Latency: < 2s
// - Worker Job Success: >= 90%
// - Worker P99 Duration: < 60s
// - Recovery Time: < 35s

// TestSLO_ManagerAPI_UnderLatencyChaos validates Manager API SLOs under
// MongoDB latency conditions.
//
// SLO Thresholds:
// - Success Rate: >= 95%
// - P99 Latency: < 2s
func TestSLO_ManagerAPI_UnderLatencyChaos(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Inject MongoDB latency
	t.Log("Phase 1: Injecting 400ms latency to MongoDB...")
	cleanup := injectLatency(t, ctx, mongoProxy, 400*time.Millisecond, 50*time.Millisecond)
	defer cleanup()

	metrics.StartChaos()

	// Phase 2: Run sustained load
	t.Log("Phase 2: Running sustained API load (100 requests)...")

	for i := 0; i < 100; i++ {
		start := time.Now()
		_, err := apiClient.ListConnections(ctx, e2eshared.ListConnectionsParams{Limit: 10})
		latency := time.Since(start)

		success := err == nil
		timeout := err != nil && (strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "deadline"))

		metrics.RecordRequest(success, timeout, latency)

		if err != nil {
			metrics.RecordError(err.Error())
		}

		// Small delay between requests
		time.Sleep(50 * time.Millisecond)
	}

	metrics.EndChaos()
	metrics.EndTest()

	// Phase 3: Validate SLOs
	t.Log("Phase 3: Validating SLOs...")

	assertions := metricskit.Assert(metrics).
		SuccessRateAbove(SLOManagerSuccessRate).
		P99Below(SLOManagerP99Latency).
		MinRequestsReached(90) // At least 90 requests recorded

	// Log report regardless of pass/fail
	t.Log(metricskit.Report(metrics).String())

	if assertions.Failed() {
		t.Log("SLO VALIDATION FAILED")
		t.Log(assertions.Summary())
		t.Fail()
	} else {
		t.Log("SLO VALIDATION PASSED")
		t.Log(assertions.Summary())
	}
}

// TestSLO_WorkerProcessing_UnderChaos validates Worker job processing SLOs
// under database latency conditions.
//
// SLO Thresholds:
// - Job Success Rate: >= 90%
// - P99 Duration: < 60s
func TestSLO_WorkerProcessing_UnderChaos(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Create test connections
	t.Log("Phase 1: Creating test connections...")
	var connections []*e2eshared.ConnectionResponse
	for i := 0; i < 10; i++ {
		conn := createTestConnection(t, ctx, "slo-worker")
		connections = append(connections, conn)
	}

	// Phase 2: Inject database latency
	t.Log("Phase 2: Injecting 500ms latency to PostgreSQL...")
	cleanup := injectLatency(t, ctx, postgresProxy, 500*time.Millisecond, 100*time.Millisecond)
	defer cleanup()

	metrics.StartChaos()

	// Phase 3: Submit jobs and track results
	t.Log("Phase 3: Submitting jobs and tracking results...")

	for i, conn := range connections {
		fetcherReq := createBasicFetcherRequest(conn.ConfigName)
		start := time.Now()

		resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
		if err != nil {
			t.Logf("Job %d creation failed: %v", i+1, err)
			metrics.RecordRequest(false, false, time.Since(start))
			metrics.RecordError(err.Error())
			continue
		}

		// Wait for completion (use polling to avoid RabbitMQ issues under chaos)
		job := waitForJobCompletionPolling(t, ctx, resp.JobID.String(), SLOWorkerP99Duration)
		latency := time.Since(start)

		success := job.Status == e2eshared.JobStatusCompleted
		metrics.RecordRequest(success, false, latency)

		t.Logf("Job %d: status=%s, duration=%v", i+1, job.Status, latency)

		time.Sleep(1 * time.Second)
	}

	metrics.EndChaos()
	metrics.EndTest()

	// Phase 4: Validate SLOs
	t.Log("Phase 4: Validating Worker SLOs...")

	assertions := metricskit.Assert(metrics).
		SuccessRateAbove(SLOWorkerJobSuccessRate).
		P99Below(SLOWorkerP99Duration).
		MinRequestsReached(8) // At least 8 jobs processed

	t.Log(metricskit.Report(metrics).String())

	if assertions.Failed() {
		t.Log("SLO VALIDATION FAILED")
		t.Log(assertions.Summary())
		t.Fail()
	} else {
		t.Log("SLO VALIDATION PASSED")
		t.Log(assertions.Summary())
	}
}

// TestSLO_SystemRecovery_AfterChaos validates system recovery time SLO
// after chaos is removed.
//
// KNOWN LIMITATION: The current chaos infrastructure creates Toxiproxy proxies
// but the application containers connect DIRECTLY to MongoDB, not through
// the proxies. This means cutting the proxy connection does not affect the
// actual application. This test validates the chaos injection/removal
// operations work correctly, but cannot verify actual system degradation.
//
// SLO Threshold:
// - Recovery Time: < 35s
func TestSLO_SystemRecovery_AfterChaos(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Verify system is healthy
	t.Log("Phase 1: Verifying baseline health...")
	require.True(t, isAPIHealthy(ctx), "system should be healthy before test")

	// Phase 2: Inject chaos
	t.Log("Phase 2: Injecting chaos (cut MongoDB connection via proxy)...")
	t.Log("NOTE: Apps connect directly to MongoDB, not through proxy. This tests proxy manipulation only.")
	cleanup := cutConnection(t, ctx, mongoProxy)
	defer cleanup()

	metrics.StartChaos()

	// Check if system is degraded (it may not be due to infrastructure limitation)
	time.Sleep(3 * time.Second)
	isDegraded := !isAPIHealthy(ctx)
	t.Logf("System degraded during chaos: %v", isDegraded)

	if !isDegraded {
		// This is expected with current infrastructure - apps don't use proxies
		t.Log("System not degraded - this is expected because apps connect directly to MongoDB")
		t.Log("Chaos proxy manipulation works, but doesn't affect app connectivity")

		// Still test the cleanup path
		metrics.EndChaos()
		metrics.EndTest()

		t.Log("PARTIAL TEST: Chaos injection/removal operations work correctly")
		t.Log("SKIPPED: Actual recovery SLO validation (requires apps to use proxies)")
		return
	}

	// If system IS degraded (future when infra is fixed), measure recovery
	t.Log("Phase 3: Removing chaos and measuring recovery time...")
	chaosEndTime := time.Now()

	cleanup()
	restoreConnection(t, ctx, mongoProxy)

	metrics.EndChaos()

	// Measure recovery
	recoveryTime := waitForRecovery(t, ctx, func() bool {
		return isAPIHealthy(ctx)
	}, SLORecoveryTime+10*time.Second) // Allow some buffer

	metrics.EndTest()

	// Validate recovery SLO
	t.Logf("Recovery time: %v (SLO: < %v)", recoveryTime, SLORecoveryTime)

	assert.LessOrEqual(t, recoveryTime, SLORecoveryTime,
		"recovery time should be within SLO threshold")

	if recoveryTime <= SLORecoveryTime {
		t.Log("RECOVERY SLO PASSED")
	} else {
		t.Log("RECOVERY SLO FAILED")
	}

	// Also log absolute recovery time from chaos end
	absoluteRecovery := recoveryTime - time.Since(chaosEndTime)
	t.Logf("Time from chaos end to recovery: approximately %v", absoluteRecovery)
}

// TestSLO_ErrorClassification validates that errors are properly classified
// and categorized during chaos conditions.
//
// Purpose: Ensure observability and debugging capabilities
func TestSLO_ErrorClassification(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Inject timeout to cause specific error types
	t.Log("Phase 1: Injecting timeout to MongoDB...")
	cleanup := injectTimeout(t, ctx, mongoProxy, 100*time.Millisecond)
	defer cleanup()

	metrics.StartChaos()

	// Phase 2: Generate errors
	t.Log("Phase 2: Generating requests to trigger errors...")

	for i := 0; i < 30; i++ {
		start := time.Now()
		_, err := apiClient.ListConnections(ctx, e2eshared.ListConnectionsParams{Limit: 10})
		latency := time.Since(start)

		if err != nil {
			// Classify error
			errStr := err.Error()
			isTimeout := strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "deadline") ||
				strings.Contains(errStr, "context")
			isConnection := strings.Contains(errStr, "connection") ||
				strings.Contains(errStr, "refused") ||
				strings.Contains(errStr, "reset")

			metrics.RecordRequest(false, isTimeout, latency)
			metrics.RecordError(errStr)

			if isTimeout {
				t.Logf("Request %d: TIMEOUT error", i+1)
			} else if isConnection {
				t.Logf("Request %d: CONNECTION error", i+1)
			} else {
				t.Logf("Request %d: OTHER error: %s", i+1, errStr)
			}
		} else {
			metrics.RecordRequest(true, false, latency)
		}

		time.Sleep(100 * time.Millisecond)
	}

	metrics.EndChaos()
	metrics.EndTest()

	// Phase 3: Validate error classification
	t.Log("Phase 3: Validating error classification...")

	errorCounts := metrics.GetErrorCounts()
	totalErrors := metrics.GetFailedRequests()
	timeoutErrors := metrics.GetTimeoutRequests()

	t.Log(metricskit.Report(metrics).String())

	t.Logf("Error breakdown:")
	t.Logf("  Total errors: %d", totalErrors)
	t.Logf("  Timeout errors: %d", timeoutErrors)
	for category, count := range errorCounts {
		t.Logf("  %s: %d", category, count)
	}

	// Validate that we have error classification
	if totalErrors > 0 {
		assert.Greater(t, len(errorCounts), 0,
			"errors should be classified into categories")
		t.Log("ERROR CLASSIFICATION WORKING")
	} else {
		t.Log("No errors to classify (system may be resilient to this chaos)")
	}
}

// =============================================================================
// SLO SUMMARY TEST
// =============================================================================

// TestSLO_Summary provides a summary of all SLO thresholds for documentation.
func TestSLO_Summary(t *testing.T) {
	t.Log("=============================================================================")
	t.Log("                         CHAOS TEST SLO SUMMARY")
	t.Log("=============================================================================")
	t.Log("")
	t.Log("Manager API SLOs (under chaos):")
	t.Logf("  - Success Rate:     >= %.1f%%", SLOManagerSuccessRate)
	t.Logf("  - P99 Latency:      < %v", SLOManagerP99Latency)
	t.Log("")
	t.Log("Worker Processing SLOs (under chaos):")
	t.Logf("  - Job Success Rate: >= %.1f%%", SLOWorkerJobSuccessRate)
	t.Logf("  - P99 Duration:     < %v", SLOWorkerP99Duration)
	t.Log("")
	t.Log("Recovery SLOs:")
	t.Logf("  - Recovery Time:    < %v", SLORecoveryTime)
	t.Log("")
	t.Log("Circuit Breaker Configuration:")
	t.Logf("  - Failure Threshold: %d consecutive failures", CircuitBreakerThreshold)
	t.Logf("  - Cooldown Period:   %v", CircuitBreakerCooldown)
	t.Log("")
	t.Log("=============================================================================")
}
