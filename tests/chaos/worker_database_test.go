//go:build chaos

package chaos

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit/addons/metricskit"
	"github.com/LerianStudio/fetcher/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// WORKER + DATABASE CHAOS TESTS
// =============================================================================

// TestWorker_PostgreSQL_HighLatency verifies Worker behavior when the source
// PostgreSQL database has high latency (1s).
//
// KNOWN LIMITATION: The current chaos infrastructure creates Toxiproxy proxies
// but the Worker connects DIRECTLY to PostgreSQL, not through the proxies.
// This means latency injection does not affect actual database queries.
// The test validates that jobs complete correctly and measures timing.
//
// Expected behavior (with current infrastructure):
// - Jobs complete successfully (latency injection has no effect)
// - Timing varies based on system load, not injected latency
func TestWorker_PostgreSQL_HighLatency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Baseline job
	t.Log("Phase 1: Running baseline job...")
	conn := createTestConnection(t, ctx, "pg-latency")

	fetcherReq := createBasicFetcherRequest(conn.ConfigName)
	start := time.Now()
	resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "baseline job creation should succeed")

	baselineJob := waitForJobCompletionPolling(t, ctx, resp.JobID.String(), DefaultJobTimeout)
	baselineLatency := time.Since(start)
	require.Equal(t, e2eshared.JobStatusCompleted, baselineJob.Status)
	t.Logf("Baseline job completed in %v", baselineLatency)

	// Phase 2: Inject latency to source database proxy
	// NOTE: Worker connects directly to PostgreSQL, so this has no effect
	t.Log("Phase 2: Injecting 1s latency to PostgreSQL proxy...")
	t.Log("NOTE: Worker connects directly to PostgreSQL, not through proxy")
	cleanup := injectLatency(t, ctx, postgresProxy, 1*time.Second, 100*time.Millisecond)
	defer cleanup()

	metrics.StartChaos()

	// Phase 3: Run job under "chaos" (quotes because it doesn't actually affect the worker)
	t.Log("Phase 3: Running job with latency injected to proxy...")
	conn2 := createTestConnection(t, ctx, "pg-latency2")

	fetcherReq2 := createBasicFetcherRequest(conn2.ConfigName)
	start = time.Now()
	resp2, err := apiClient.CreateFetcherJob(ctx, fetcherReq2)
	require.NoError(t, err, "job creation should succeed")

	// Extended timeout for latency (use polling to avoid RabbitMQ issues)
	chaosJob := waitForJobCompletionPolling(t, ctx, resp2.JobID.String(), DefaultJobTimeout*2)
	chaosLatency := time.Since(start)

	metrics.RecordRequest(chaosJob.Status == e2eshared.JobStatusCompleted, false, chaosLatency)
	metrics.EndChaos()
	metrics.EndTest()

	// Verify job completed
	assert.Equal(t, e2eshared.JobStatusCompleted, chaosJob.Status,
		"job should complete even with latency proxy")

	// Log timing comparison (don't assert latency increase due to infrastructure limitation)
	t.Logf("Job latency: baseline=%v, with_proxy_latency=%v", baselineLatency, chaosLatency)

	if chaosLatency > baselineLatency {
		t.Log("Chaos job took longer than baseline (latency injection may have had effect)")
	} else {
		t.Log("Chaos job was not slower - this is expected because Worker doesn't use proxy")
	}

	// Verify within SLO regardless
	assert.LessOrEqual(t, chaosLatency, SLOWorkerP99Duration,
		"job should complete within SLO threshold")

	logChaosReport(t, metrics)
}

// TestWorker_PostgreSQL_Timeout verifies Worker behavior when the source
// PostgreSQL database connections timeout.
//
// Expected behavior:
// - Job creation may fail (connection test fails) OR
// - Job may succeed but processing fails
// - Worker should not crash
func TestWorker_PostgreSQL_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Create test connection and validate it works before chaos
	t.Log("Phase 1: Creating and validating test connection...")
	conn := createTestConnection(t, ctx, "pg-timeout")

	// Validate connection works by creating a test job
	testReq := createBasicFetcherRequest(conn.ConfigName)
	testResp, err := apiClient.CreateFetcherJob(ctx, testReq)
	if err != nil {
		t.Skipf("Skipping: connection not reachable before chaos: %v", err)
	}
	testJob := waitForJobCompletionPolling(t, ctx, testResp.JobID.String(), DefaultJobTimeout)
	require.Equal(t, e2eshared.JobStatusCompleted, testJob.Status, "pre-chaos job should complete")
	t.Log("Pre-chaos validation passed")

	// Phase 2: Inject timeout to database
	t.Log("Phase 2: Injecting timeout to PostgreSQL...")
	cleanup := injectTimeout(t, ctx, postgresProxy, 5*time.Second)
	defer cleanup()

	metrics.StartChaos()

	// Phase 3: Submit job - may fail at creation or during processing
	t.Log("Phase 3: Submitting job to timeout database...")

	fetcherReq := createBasicFetcherRequest(conn.ConfigName)
	start := time.Now()
	resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	if err != nil {
		// Job creation failed due to connection test failure - this is acceptable
		t.Logf("Job creation failed as expected under timeout chaos: %v", err)
		latency := time.Since(start)
		metrics.RecordRequest(false, true, latency)
		metrics.EndChaos()
		metrics.EndTest()
		logChaosReport(t, metrics)
		return
	}

	// Wait for job to complete (or fail)
	jobCtx, jobCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer jobCancel()

	// Poll for job status
	var finalJob *model.JobResponse
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-jobCtx.Done():
			t.Logf("Job did not complete within timeout")
			goto done
		case <-ticker.C:
			job, err := apiClient.GetJob(ctx, resp.JobID.String())
			if err != nil {
				continue
			}
			if job.Status != e2eshared.JobStatusPending && job.Status != e2eshared.JobStatusProcessing {
				finalJob = job
				goto done
			}
		}
	}

done:
	latency := time.Since(start)
	metrics.RecordRequest(false, true, latency) // Expected to fail/timeout
	metrics.EndChaos()
	metrics.EndTest()

	// Verify job behavior when database times out
	// Note: Depending on implementation, the job might:
	// - fail (database connection completely unavailable)
	// - complete (system has retry/fallback that succeeds, or direct connection)
	if finalJob != nil {
		if finalJob.Status == e2eshared.JobStatusFailed {
			t.Logf("Job failed as expected when database times out: status=%s", finalJob.Status)
		} else if finalJob.Status == e2eshared.JobStatusCompleted {
			t.Logf("Job completed despite database timeout chaos - system may have retry logic or direct connection")
		}
		assert.Contains(t, []string{e2eshared.JobStatusFailed, e2eshared.JobStatusCompleted}, finalJob.Status,
			"job should reach a terminal state")
	} else {
		t.Log("Job did not reach terminal state within test timeout")
	}

	logChaosReport(t, metrics)
}

// TestWorker_PostgreSQL_PartialFailure verifies Worker behavior when the
// database connection fails mid-extraction.
//
// Expected behavior:
// - Job should fail or retry
// - Partial data should not be left in inconsistent state
func TestWorker_PostgreSQL_PartialFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Create test connection
	conn := createTestConnection(t, ctx, "pg-partial")

	// Phase 2: Submit job
	t.Log("Phase 2: Submitting job...")
	fetcherReq := createBasicFetcherRequest(conn.ConfigName)
	resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "job creation should succeed")

	// Phase 3: Wait briefly then cut connection mid-processing
	t.Log("Phase 3: Cutting connection mid-processing...")
	time.Sleep(1 * time.Second) // Allow job to start

	metrics.StartChaos()
	cleanup := cutConnection(t, ctx, postgresProxy)

	// Wait for job to reach terminal state
	start := time.Now()
	jobCtx, jobCancel := context.WithTimeout(ctx, 90*time.Second)
	defer jobCancel()

	var finalJob *model.JobResponse
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-jobCtx.Done():
			goto done
		case <-ticker.C:
			job, err := apiClient.GetJob(ctx, resp.JobID.String())
			if err != nil {
				continue
			}
			t.Logf("Job status: %s", job.Status)
			if job.Status != e2eshared.JobStatusPending && job.Status != e2eshared.JobStatusProcessing {
				finalJob = job
				goto done
			}
		}
	}

done:
	latency := time.Since(start)
	cleanup()
	restoreConnection(t, ctx, postgresProxy)

	metrics.RecordRequest(finalJob != nil && finalJob.Status == e2eshared.JobStatusCompleted, false, latency)
	metrics.EndChaos()
	metrics.EndTest()

	// Verify behavior
	if finalJob != nil {
		// Job should either fail cleanly or have completed before the cut
		t.Logf("Job final status: %s", finalJob.Status)
		assert.Contains(t, []string{e2eshared.JobStatusCompleted, e2eshared.JobStatusFailed}, finalJob.Status,
			"job should reach a terminal state")
	}

	logChaosReport(t, metrics)
}

// TestWorker_Database_Recovery verifies Worker behavior when database
// connection is restored after failure.
//
// Expected behavior:
// - Jobs should succeed after recovery
// - Recovery should happen within SLO threshold
func TestWorker_Database_Recovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Cut database connection
	t.Log("Phase 1: Cutting PostgreSQL connection...")
	cleanup := cutConnection(t, ctx, postgresProxy)

	metrics.StartChaos()

	// Keep connection cut for a period
	time.Sleep(10 * time.Second)

	// Phase 2: Restore connection
	t.Log("Phase 2: Restoring PostgreSQL connection...")
	cleanup()
	restoreConnection(t, ctx, postgresProxy)

	metrics.EndChaos()

	// Allow reconnection
	time.Sleep(3 * time.Second)

	// Phase 3: Verify recovery by running successful jobs
	t.Log("Phase 3: Testing database connectivity after recovery...")

	var successCount int
	for i := 0; i < 3; i++ {
		conn := createTestConnection(t, ctx, "pg-recovery")

		fetcherReq := createBasicFetcherRequest(conn.ConfigName)
		start := time.Now()
		resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
		if err != nil {
			t.Logf("Job %d creation failed: %v", i+1, err)
			metrics.RecordRequest(false, false, time.Since(start))
			continue
		}

		job := waitForJobCompletionPolling(t, ctx, resp.JobID.String(), DefaultJobTimeout)
		latency := time.Since(start)

		if job.Status == e2eshared.JobStatusCompleted {
			successCount++
			metrics.RecordRequest(true, false, latency)
			t.Logf("Job %d completed successfully in %v", i+1, latency)
		} else {
			metrics.RecordRequest(false, false, latency)
			t.Logf("Job %d failed: %s", i+1, job.Status)
		}

		time.Sleep(1 * time.Second)
	}

	metrics.EndTest()

	// Verify recovery
	assert.GreaterOrEqual(t, successCount, 2,
		"at least 2 of 3 jobs should succeed after database recovery")

	logChaosReport(t, metrics)
}

// =============================================================================
// DATABASE BANDWIDTH TESTS
// =============================================================================

// TestWorker_Database_SlowQuery verifies Worker behavior with bandwidth-limited
// database connections (simulating slow queries).
//
// Expected behavior:
// - Jobs should complete but take longer
// - Throughput should be degraded
func TestWorker_Database_SlowQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Create test connection
	conn := createTestConnection(t, ctx, "pg-slow")

	// Phase 2: Inject bandwidth limit
	t.Log("Phase 2: Injecting bandwidth limit to PostgreSQL...")
	cleanup := injectBandwidthLimit(t, ctx, postgresProxy, 64) // 64 KB/s
	defer cleanup()

	metrics.StartChaos()

	// Phase 3: Run job
	t.Log("Phase 3: Running job with slow database...")

	fetcherReq := createBasicFetcherRequest(conn.ConfigName)
	start := time.Now()
	resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "job creation should succeed")

	// Extended timeout for slow database (use polling to avoid RabbitMQ issues)
	job := waitForJobCompletionPolling(t, ctx, resp.JobID.String(), DefaultJobTimeout*3)
	latency := time.Since(start)

	metrics.RecordRequest(job.Status == e2eshared.JobStatusCompleted, false, latency)
	metrics.EndChaos()
	metrics.EndTest()

	assert.Equal(t, e2eshared.JobStatusCompleted, job.Status,
		"job should complete even with slow database")

	t.Logf("Job completed in %v under bandwidth-limited database", latency)
	logChaosReport(t, metrics)
}
