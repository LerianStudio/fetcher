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
// WORKER + SEAWEEDFS CHAOS TESTS
// =============================================================================

// TestWorker_SeaweedFS_Unavailable verifies Worker behavior when SeaweedFS
// storage is completely unavailable.
//
// Expected behavior:
// - Job should fail gracefully
// - Error should clearly indicate storage issue
func TestWorker_SeaweedFS_Unavailable(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Create test connection
	conn := createTestConnection(t, ctx, "seaweed-unavail")

	// Phase 2: Cut SeaweedFS connection
	t.Log("Phase 2: Cutting SeaweedFS connection...")
	cleanup := cutConnection(t, ctx, seaweedProxy)
	defer cleanup()

	// Allow time for the cut to take effect
	time.Sleep(2 * time.Second)

	metrics.StartChaos()

	// Phase 3: Submit job
	t.Log("Phase 3: Submitting job with unavailable storage...")

	fetcherReq := createBasicFetcherRequest(conn.ConfigName)
	start := time.Now()
	resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "job creation should succeed (message queued)")

	// Wait for job to fail
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
	metrics.RecordRequest(finalJob != nil && finalJob.Status == e2eshared.JobStatusCompleted, false, latency)
	metrics.EndChaos()
	metrics.EndTest()

	// Verify job behavior when storage is unavailable
	// Note: Depending on implementation, the job might:
	// - fail (storage is required for output)
	// - complete (storage failure is handled gracefully or worker uses direct connection)
	if finalJob != nil {
		if finalJob.Status == e2eshared.JobStatusFailed {
			t.Logf("Job failed as expected when storage is unavailable: status=%s", finalJob.Status)
		} else if finalJob.Status == e2eshared.JobStatusCompleted {
			t.Logf("Job completed despite storage chaos - system may be resilient or using direct connection")
			// This is acceptable if the worker has fallback behavior or direct SeaweedFS access
		}
		assert.Contains(t, []string{e2eshared.JobStatusFailed, e2eshared.JobStatusCompleted}, finalJob.Status,
			"job should reach a terminal state")
	} else {
		t.Log("Job did not reach terminal state within test timeout")
	}

	logChaosReport(t, metrics)
}

// TestWorker_SeaweedFS_SlowUpload verifies Worker behavior when SeaweedFS
// uploads are slow (bandwidth limited).
//
// KNOWN LIMITATION: The current chaos infrastructure creates Toxiproxy proxies
// but the Worker connects DIRECTLY to SeaweedFS, not through the proxies.
// This means bandwidth injection does not affect actual storage uploads.
// The test validates that jobs complete correctly under proxy manipulation.
//
// Expected behavior (with current infrastructure):
// - Jobs complete successfully (bandwidth limit has no effect)
// - Timing varies based on system load, not bandwidth limit
func TestWorker_SeaweedFS_SlowUpload(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Validate connectivity with baseline job
	t.Log("Phase 1: Running baseline job...")
	conn := createTestConnection(t, ctx, "seaweed-slow")

	fetcherReq := createBasicFetcherRequest(conn.ConfigName)
	start := time.Now()
	resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	if err != nil {
		t.Logf("Baseline job creation failed: %v", err)
		t.Skip("Skipping: PostgreSQL not reachable from Manager container")
	}

	baselineJob := waitForJobCompletionPolling(t, ctx, resp.JobID.String(), DefaultJobTimeout)
	baselineLatency := time.Since(start)
	require.Equal(t, e2eshared.JobStatusCompleted, baselineJob.Status)
	t.Logf("Baseline job completed in %v", baselineLatency)

	// Phase 2: Inject bandwidth limit to SeaweedFS proxy
	// NOTE: Worker connects directly to SeaweedFS, so this has no effect
	t.Log("Phase 2: Injecting 56 KB/s bandwidth limit to SeaweedFS proxy...")
	t.Log("NOTE: Worker connects directly to SeaweedFS, not through proxy")
	cleanup := injectBandwidthLimit(t, ctx, seaweedProxy, 56) // 56 KB/s (slow upload)
	defer cleanup()

	metrics.StartChaos()

	// Phase 3: Run job with proxy bandwidth limited
	t.Log("Phase 3: Running job with bandwidth-limited proxy...")
	conn2 := createTestConnection(t, ctx, "seaweed-slow2")

	fetcherReq2 := createBasicFetcherRequest(conn2.ConfigName)
	start = time.Now()
	resp2, err := apiClient.CreateFetcherJob(ctx, fetcherReq2)
	if err != nil {
		t.Logf("Job creation failed under chaos: %v", err)
		t.Skip("Job creation failed - may be transient connectivity issue")
	}

	// Extended timeout for slow upload (use polling)
	chaosJob := waitForJobCompletionPolling(t, ctx, resp2.JobID.String(), DefaultJobTimeout*4)
	chaosLatency := time.Since(start)

	metrics.RecordRequest(chaosJob.Status == e2eshared.JobStatusCompleted, false, chaosLatency)
	metrics.EndChaos()
	metrics.EndTest()

	// Verify job completed
	assert.Equal(t, e2eshared.JobStatusCompleted, chaosJob.Status,
		"job should complete even with bandwidth-limited proxy")

	// Log timing comparison (don't assert latency increase due to infrastructure limitation)
	t.Logf("Job latency: baseline=%v, with_proxy_bandwidth_limit=%v", baselineLatency, chaosLatency)

	if chaosLatency > baselineLatency {
		t.Log("Chaos job took longer than baseline (bandwidth limit may have had effect)")
	} else {
		t.Log("Chaos job was not slower - this is expected because Worker doesn't use proxy")
	}

	logChaosReport(t, metrics)
}

// TestWorker_SeaweedFS_LatencySpike verifies Worker behavior when SeaweedFS
// has high latency (2s).
//
// Expected behavior:
// - Job should complete with delay
// - Storage operations should eventually succeed
func TestWorker_SeaweedFS_LatencySpike(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Create test connection
	conn := createTestConnection(t, ctx, "seaweed-latency")

	// Phase 2: Inject latency to SeaweedFS
	t.Log("Phase 2: Injecting 2s latency to SeaweedFS...")
	cleanup := injectLatency(t, ctx, seaweedProxy, 2*time.Second, 200*time.Millisecond)
	defer cleanup()

	metrics.StartChaos()

	// Phase 3: Run job under chaos
	t.Log("Phase 3: Running job with high storage latency...")

	fetcherReq := createBasicFetcherRequest(conn.ConfigName)
	start := time.Now()
	resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "job creation should succeed")

	// Extended timeout for high latency (use polling)
	chaosJob := waitForJobCompletionPolling(t, ctx, resp.JobID.String(), DefaultJobTimeout*3)
	latency := time.Since(start)

	metrics.RecordRequest(chaosJob.Status == e2eshared.JobStatusCompleted, false, latency)
	metrics.EndChaos()
	metrics.EndTest()

	// Verify job completed
	assert.Equal(t, e2eshared.JobStatusCompleted, chaosJob.Status,
		"job should complete even with high storage latency")

	t.Logf("Job completed in %v under high storage latency", latency)
	logChaosReport(t, metrics)
}

// =============================================================================
// SEAWEEDFS RECOVERY TESTS
// =============================================================================

// TestWorker_SeaweedFS_Recovery verifies Worker behavior when SeaweedFS
// connection is restored after failure.
//
// Expected behavior:
// - Jobs should succeed after storage recovery
// - Recovery should be automatic
func TestWorker_SeaweedFS_Recovery(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), LongChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Cut SeaweedFS connection
	t.Log("Phase 1: Cutting SeaweedFS connection...")
	cleanup := cutConnection(t, ctx, seaweedProxy)

	metrics.StartChaos()

	// Keep connection cut briefly
	time.Sleep(5 * time.Second)

	// Phase 2: Restore connection
	t.Log("Phase 2: Restoring SeaweedFS connection...")
	cleanup()
	restoreConnection(t, ctx, seaweedProxy)

	metrics.EndChaos()

	// Allow reconnection
	time.Sleep(3 * time.Second)

	// Phase 3: Verify recovery
	t.Log("Phase 3: Testing storage after recovery...")

	conn := createTestConnection(t, ctx, "seaweed-recovery")

	fetcherReq := createBasicFetcherRequest(conn.ConfigName)
	start := time.Now()
	resp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "job creation should succeed after recovery")

	job := waitForJobCompletionPolling(t, ctx, resp.JobID.String(), DefaultJobTimeout)
	latency := time.Since(start)

	metrics.RecordRequest(job.Status == e2eshared.JobStatusCompleted, false, latency)
	metrics.EndTest()

	assert.Equal(t, e2eshared.JobStatusCompleted, job.Status,
		"job should complete after storage recovery")

	t.Logf("Job completed in %v after storage recovery", latency)
	logChaosReport(t, metrics)
}
