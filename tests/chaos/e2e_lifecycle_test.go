//go:build chaos

package chaos

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit/addons/metricskit"
	"github.com/LerianStudio/fetcher/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// E2E LIFECYCLE CHAOS TESTS
// =============================================================================

// TestE2E_FullLifecycle_IntermittentChaos verifies system behavior under
// intermittent chaos conditions (10s on, 10s off pattern).
//
// Expected behavior:
// - 90%+ jobs should complete successfully
// - System should handle intermittent failures gracefully
func TestE2E_FullLifecycle_IntermittentChaos(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Create test connections
	t.Log("Phase 1: Creating test connections...")
	var connections []*e2eshared.ConnectionResponse
	for i := 0; i < 10; i++ {
		conn := createTestConnection(t, ctx, "e2e-intermittent")
		connections = append(connections, conn)
	}

	// Phase 2: Start intermittent chaos pattern
	t.Log("Phase 2: Starting intermittent chaos pattern...")

	chaosActive := false
	chaosMu := sync.Mutex{}

	// Chaos toggle goroutine
	chaosCtx, chaosCancel := context.WithCancel(ctx)
	defer chaosCancel()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-chaosCtx.Done():
				return
			case <-ticker.C:
				chaosMu.Lock()
				if chaosActive {
					// Remove chaos
					restoreConnection(t, chaosCtx, postgresProxy)
					chaosActive = false
					t.Log("Chaos OFF")
				} else {
					// Add chaos
					suite.Chaos().AddLatency(chaosCtx, postgresProxy, 500*time.Millisecond, 50*time.Millisecond)
					chaosActive = true
					t.Log("Chaos ON (500ms latency)")
				}
				chaosMu.Unlock()
			}
		}
	}()

	metrics.StartChaos()

	// Phase 3: Submit jobs and track results
	t.Log("Phase 3: Submitting jobs under intermittent chaos...")

	var wg sync.WaitGroup
	results := make(chan bool, len(connections))

	for i, conn := range connections {
		wg.Add(1)
		go func(idx int, c *e2eshared.ConnectionResponse) {
			defer wg.Done()

			fetcherReq := createBasicFetcherRequest(c.ConfigName)
			start := time.Now()

			resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
			if err != nil {
				t.Logf("Job %d creation failed: %v", idx+1, err)
				metrics.RecordRequest(false, false, time.Since(start))
				results <- false
				return
			}

			// Wait for completion with extended timeout
			jobCtx, jobCancel := context.WithTimeout(ctx, DefaultJobTimeout*2)
			defer jobCancel()

			job, err := waitForJobCompletionWithContext(t, jobCtx, resp.JobID.String())
			latency := time.Since(start)

			success := err == nil && job != nil && job.Status == e2eshared.JobStatusCompleted
			metrics.RecordRequest(success, false, latency)
			results <- success

			if success {
				t.Logf("Job %d completed in %v", idx+1, latency)
			} else {
				status := "unknown"
				if job != nil {
					status = job.Status
				}
				t.Logf("Job %d failed: status=%s, err=%v", idx+1, status, err)
			}
		}(i, conn)

		// Stagger job submissions
		time.Sleep(2 * time.Second)
	}

	// Wait for all jobs
	wg.Wait()
	close(results)

	chaosCancel() // Stop chaos toggle

	metrics.EndChaos()
	metrics.EndTest()

	// Phase 4: Calculate results
	var successCount int
	for success := range results {
		if success {
			successCount++
		}
	}

	successRate := float64(successCount) / float64(len(connections)) * 100

	// Verify SLO
	assert.GreaterOrEqual(t, successRate, 90.0,
		"at least 90%% of jobs should complete under intermittent chaos")

	t.Logf("Intermittent chaos results: %d/%d jobs completed (%.2f%%)",
		successCount, len(connections), successRate)

	logChaosReport(t, metrics)
}

// TestE2E_FullLifecycle_MultipleFailures verifies system behavior when
// multiple components are degraded simultaneously.
//
// Expected behavior:
// - System should degrade gracefully
// - Some jobs should still complete
func TestE2E_FullLifecycle_MultipleFailures(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Create test connection
	conn := createTestConnection(t, ctx, "e2e-multiple")

	// Phase 2: Inject multiple chaos conditions
	t.Log("Phase 2: Injecting multiple chaos conditions...")

	// MongoDB latency
	mongoCleanup := injectLatency(t, ctx, mongoProxy, 300*time.Millisecond, 50*time.Millisecond)
	defer mongoCleanup()

	// PostgreSQL latency
	pgCleanup := injectLatency(t, ctx, postgresProxy, 500*time.Millisecond, 50*time.Millisecond)
	defer pgCleanup()

	// SeaweedFS bandwidth limit
	seaweedCleanup := injectBandwidthLimit(t, ctx, seaweedProxy, 128) // 128 KB/s
	defer seaweedCleanup()

	metrics.StartChaos()

	// Phase 3: Run jobs under multiple failures
	t.Log("Phase 3: Running jobs under multiple degradations...")

	var successCount int
	for i := 0; i < 5; i++ {
		fetcherReq := createBasicFetcherRequest(conn.ConfigName)
		start := time.Now()

		resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
		if err != nil {
			t.Logf("Job %d creation failed: %v", i+1, err)
			metrics.RecordRequest(false, false, time.Since(start))
			continue
		}

		// Wait with extended timeout for degraded conditions (use polling)
		job := waitForJobCompletionPolling(t, ctx, resp.JobID.String(), DefaultJobTimeout*3)
		latency := time.Since(start)

		if job.Status == e2eshared.JobStatusCompleted {
			successCount++
			metrics.RecordRequest(true, false, latency)
			t.Logf("Job %d completed in %v", i+1, latency)
		} else {
			metrics.RecordRequest(false, false, latency)
			t.Logf("Job %d failed: %s", i+1, job.Status)
		}

		time.Sleep(2 * time.Second)
	}

	metrics.EndChaos()
	metrics.EndTest()

	// Under multiple failures, expect degraded but functional system
	assert.GreaterOrEqual(t, successCount, 2,
		"at least 2 of 5 jobs should complete under multiple degradations")

	logChaosReport(t, metrics)
}

// TestE2E_FullLifecycle_RecoveryTime verifies system recovery time after
// chaos is removed.
//
// Expected behavior:
// - System should recover within SLO threshold (35s)
// - First successful request marks recovery
func TestE2E_FullLifecycle_RecoveryTime(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Verify baseline
	t.Log("Phase 1: Verifying baseline...")
	conn := createTestConnection(t, ctx, "e2e-recovery")

	fetcherReq := createBasicFetcherRequest(conn.ConfigName)
	resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "baseline job creation should succeed")

	baselineJob := waitForJobCompletionPolling(t, ctx, resp.JobID.String(), DefaultJobTimeout)
	require.Equal(t, e2eshared.JobStatusCompleted, baselineJob.Status)

	// Phase 2: Inject severe chaos
	t.Log("Phase 2: Injecting severe chaos...")
	mongoCleanup := cutConnection(t, ctx, mongoProxy)
	pgCleanup := cutConnection(t, ctx, postgresProxy)

	metrics.StartChaos()

	// Keep chaos for a while
	time.Sleep(10 * time.Second)

	// Phase 3: Remove chaos and measure recovery
	t.Log("Phase 3: Removing chaos and measuring recovery time...")
	chaosEndTime := time.Now()

	mongoCleanup()
	pgCleanup()
	restoreConnection(t, ctx, mongoProxy)
	restoreConnection(t, ctx, postgresProxy)

	metrics.EndChaos()

	// Phase 4: Measure recovery time
	recoveryStart := time.Now()
	var recoveryTime time.Duration

	// Try to complete a job - first success marks recovery
	for i := 0; i < 10; i++ {
		conn2 := createTestConnection(t, ctx, "e2e-recovery-probe")

		fetcherReq2 := createBasicFetcherRequest(conn2.ConfigName)
		resp2, err := apiClient.CreateFetcherJob(ctx, fetcherReq2)

		if err != nil {
			t.Logf("Probe %d: job creation failed - still recovering", i+1)
			time.Sleep(3 * time.Second)
			continue
		}

		// Short timeout for probes
		probeCtx, probeCancel := context.WithTimeout(ctx, 30*time.Second)
		job, _ := waitForJobCompletionWithContext(t, probeCtx, resp2.JobID.String())
		probeCancel()

		if job != nil && job.Status == e2eshared.JobStatusCompleted {
			recoveryTime = time.Since(chaosEndTime)
			t.Logf("System recovered in %v (probe %d)", recoveryTime, i+1)
			break
		}

		t.Logf("Probe %d: job not completed - still recovering", i+1)
		time.Sleep(3 * time.Second)
	}

	if recoveryTime == 0 {
		recoveryTime = time.Since(recoveryStart)
		t.Logf("Recovery not confirmed within test window (%v elapsed)", recoveryTime)
	}

	metrics.EndTest()

	// Verify recovery SLO
	assert.LessOrEqual(t, recoveryTime, SLORecoveryTime,
		"system should recover within %v", SLORecoveryTime)

	logChaosReport(t, metrics)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// waitForJobCompletionWithContext waits for job completion with a custom context.
func waitForJobCompletionWithContext(t *testing.T, ctx context.Context, jobID string) (*model.JobResponse, error) {
	t.Helper()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			job, err := apiClient.GetJob(ctx, jobID)
			if err != nil {
				continue
			}
			if job.Status != e2eshared.JobStatusPending && job.Status != e2eshared.JobStatusProcessing {
				return job, nil
			}
		}
	}
}
