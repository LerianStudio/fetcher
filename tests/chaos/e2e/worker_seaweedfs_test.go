//go:build chaos

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/tests/chaos/helpers"
	"github.com/LerianStudio/fetcher/tests/chaos/setup"
	"github.com/LerianStudio/fetcher/tests/shared/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SeaweedFS Chaos Tests - via Worker Jobs
// =============================================================================
// These tests validate system behavior when SeaweedFS experiences network issues
// during file operations (job results storage).

// TestSeaweedFSLatency_FileUploadSlowed tests that file uploads complete despite
// latency to the SeaweedFS storage backend.
func (s *ChaosTestSuite) TestSeaweedFSLatency_FileUploadSlowed() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"complete job with extended duration when SeaweedFS has network latency",
		"SeaweedFS has latency injected during file upload phase",
	))

	// Phase 1: Create connection for the job
	t.Log("Phase 1: Creating PostgreSQL connection...")
	configName := s.uniqueConfigName("chaos_seaweedfs_latency")
	pg := s.chaosInfra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Phase 2: Inject SeaweedFS latency
	t.Log("Phase 2: Injecting SeaweedFS latency...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetSeaweedFSProxy()
	require.NotNil(t, proxy, "SeaweedFS proxy should exist")

	// Use Low latency to slow down uploads without causing timeouts
	chaosConfig := helpers.DefaultLatencyConfig(
		helpers.LatencyMs(setup.ChaosLatencyValues.Low),
		helpers.LatencyMs(setup.ChaosLatencyValues.Jitter),
	)
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 3: Create and execute job under latency
	t.Log("Phase 3: Creating extraction job under latency...")
	start := time.Now()
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id", "amount"}},
			},
		},
		Metadata: s.testMetadata("TestSeaweedFSLatency"),
	})
	require.NoError(t, err)

	// Wait for completion - should succeed despite latency
	notification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		jobResp.JobID.String(),
		setup.JobCompletionTimeoutSlow, // Extended timeout for slow uploads
	)

	duration := time.Since(start)
	s.metrics.EndChaos()

	require.NoError(t, err, "Job should complete despite SeaweedFS latency")
	assert.Equal(t, "completed", notification.Status)
	s.metrics.RecordRequest(true, false, duration)

	// Phase 4: Verify file exists in SeaweedFS
	// Worker stores results at /external-data/{jobID}.json (see constant.ExternalDataBucketName)
	t.Log("Phase 4: Verifying result file exists...")
	filePath := fmt.Sprintf("/external-data/%s.json", jobResp.JobID.String())
	_, fileErr := s.seaweedClient.GetFile(s.ctx, filePath)
	assert.NoError(t, fileErr, "Result file should exist in SeaweedFS")

	// Cleanup
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	helpers.DocumentResult(t, s.metrics, "Job completed despite SeaweedFS latency")
	t.Logf("Job completed in %v (expected slower due to storage latency)", duration)
}

// TestSeaweedFSTimeout_FileOperationFails tests that the system handles SeaweedFS
// timeouts gracefully and can recover for subsequent operations.
func (s *ChaosTestSuite) TestSeaweedFSTimeout_FileOperationFails() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"fail job gracefully when SeaweedFS times out, then recover",
		"SeaweedFS has timeout chaos injected causing storage failures",
	))

	// Phase 1: Create connection for the job
	t.Log("Phase 1: Creating PostgreSQL connection...")
	configName := s.uniqueConfigName("chaos_seaweedfs_timeout")
	pg := s.chaosInfra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Phase 2: Inject SeaweedFS timeout
	t.Log("Phase 2: Injecting SeaweedFS timeout...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetSeaweedFSProxy()
	require.NotNil(t, proxy, "SeaweedFS proxy should exist")

	chaosConfig := helpers.DefaultTimeoutConfig(helpers.TimeoutMs(setup.ChaosTimeoutValues.Short))
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 3: Create job - may fail due to SeaweedFS timeout
	t.Log("Phase 3: Creating extraction job under timeout chaos...")
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestSeaweedFSTimeout"),
	})

	if err != nil {
		// Job creation might fail if SeaweedFS is involved early in the process
		t.Logf("Job creation failed under SeaweedFS timeout (acceptable): %v", err)
		s.metrics.EndChaos()
	} else {
		// If job was created, wait for result (may fail or timeout)
		notification, waitErr := s.eventConsumer.WaitForJobEvent(
			s.ctx,
			jobResp.JobID.String(),
			setup.JobCompletionTimeout,
		)

		s.metrics.EndChaos()

		if waitErr != nil {
			t.Logf("Job failed or timed out under chaos (expected): %v", waitErr)
		} else {
			// Job might complete if extraction finishes before upload timeout
			t.Logf("Job status under chaos: %s", notification.Status)
		}
	}

	// Phase 4: Remove chaos and verify recovery
	t.Log("Phase 4: Removing chaos, verifying recovery...")
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	s.metrics.StartRecovery()
	time.Sleep(setup.RecoveryObservationTime)

	// Create new connection and job - should succeed
	recoveryConfigName := s.uniqueConfigName("chaos_seaweedfs_recovery")
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   recoveryConfigName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	recoveryJob, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				recoveryConfigName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestSeaweedFSTimeout_Recovery"),
	})
	require.NoError(t, err, "Recovery job creation should succeed")

	recoveryNotification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		recoveryJob.JobID.String(),
		setup.JobCompletionTimeout,
	)
	require.NoError(t, err, "Recovery job should complete")
	assert.Equal(t, "completed", recoveryNotification.Status)

	s.metrics.EndRecovery()

	helpers.DocumentResult(t, s.metrics, "System recovered from SeaweedFS timeout")
}

// TestSeaweedFSBandwidth_SlowFileTransfer tests that file uploads complete
// despite severely limited bandwidth to SeaweedFS.
func (s *ChaosTestSuite) TestSeaweedFSBandwidth_SlowFileTransfer() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"complete job with very slow file transfer when SeaweedFS bandwidth is limited",
		"SeaweedFS has bandwidth throttling (10 KB/s)",
	))

	// Phase 1: Create connection for the job
	t.Log("Phase 1: Creating PostgreSQL connection...")
	configName := s.uniqueConfigName("chaos_seaweedfs_bandwidth")
	pg := s.chaosInfra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Phase 2: Inject bandwidth limiting
	t.Log("Phase 2: Injecting SeaweedFS bandwidth limit...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetSeaweedFSProxy()
	require.NotNil(t, proxy, "SeaweedFS proxy should exist")

	// Use Medium bandwidth limit (10 KB/s) - slow but should complete
	chaosConfig := helpers.DefaultBandwidthConfig(setup.ChaosBandwidthValues.Medium)
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 3: Create and execute job under bandwidth limit
	t.Log("Phase 3: Creating extraction job under bandwidth limit...")
	start := time.Now()
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestSeaweedFSBandwidth"),
	})
	require.NoError(t, err)

	// Wait for completion with extended timeout for slow transfers
	notification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		jobResp.JobID.String(),
		setup.JobCompletionTimeoutSlow, // Extended timeout for bandwidth-limited transfers
	)

	duration := time.Since(start)
	s.metrics.EndChaos()

	require.NoError(t, err, "Job should complete despite bandwidth limit")
	assert.Equal(t, "completed", notification.Status)
	s.metrics.RecordRequest(true, false, duration)

	// Cleanup
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	helpers.DocumentResult(t, s.metrics, "Job completed despite SeaweedFS bandwidth limit")
	t.Logf("Job completed in %v (expected slower due to 10 KB/s bandwidth limit)", duration)
}

// =============================================================================
// SeaweedFS Direct Client Tests
// =============================================================================
// These tests validate direct SeaweedFS client behavior under chaos conditions.

// TestSeaweedFSDirectUpload_WithLatency tests direct file upload and download
// operations with latency injected into SeaweedFS.
func (s *ChaosTestSuite) TestSeaweedFSDirectUpload_WithLatency() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"complete direct file upload and download despite SeaweedFS latency",
		"SeaweedFS has 500ms latency injected for HTTP operations",
	))

	// Phase 1: Baseline - upload file without chaos
	t.Log("Phase 1: Baseline file upload (no chaos)...")
	baselineFileName := fmt.Sprintf("/test/chaos/baseline_%d.txt", time.Now().UnixNano())
	baselineContent := []byte("Baseline test content for SeaweedFS chaos test")

	baselineStart := time.Now()
	err := uploadToSeaweedFS(s.ctx, s.chaosInfra.SeaweedFSProxyURL, baselineFileName, baselineContent)
	baselineDuration := time.Since(baselineStart)
	require.NoError(t, err, "Baseline upload should succeed")
	t.Logf("Baseline upload completed in %v", baselineDuration)

	// Verify baseline file exists
	downloadedBaseline, err := s.seaweedClient.GetFile(s.ctx, baselineFileName)
	require.NoError(t, err, "Baseline download should succeed")
	assert.Equal(t, baselineContent, downloadedBaseline, "Downloaded content should match")

	// Phase 2: Inject latency
	t.Log("Phase 2: Injecting SeaweedFS latency...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetSeaweedFSProxy()
	require.NotNil(t, proxy, "SeaweedFS proxy should exist")

	chaosConfig := helpers.DefaultLatencyConfig(
		helpers.LatencyMs(setup.ChaosLatencyValues.Low),
		helpers.LatencyMs(setup.ChaosLatencyValues.Jitter),
	)
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 3: Upload file under latency
	t.Log("Phase 3: Uploading file under latency...")
	chaosFileName := fmt.Sprintf("/test/chaos/latency_%d.txt", time.Now().UnixNano())
	chaosContent := []byte("Chaos test content - uploaded with latency injection")

	chaosStart := time.Now()
	err = uploadToSeaweedFS(s.ctx, s.chaosInfra.SeaweedFSProxyURL, chaosFileName, chaosContent)
	chaosDuration := time.Since(chaosStart)
	s.metrics.EndChaos()

	require.NoError(t, err, "Upload should succeed despite latency")
	s.metrics.RecordRequest(true, false, chaosDuration)
	t.Logf("Chaos upload completed in %v (expected longer than baseline %v)", chaosDuration, baselineDuration)

	// Verify file was uploaded correctly
	downloadedChaos, err := s.seaweedClient.GetFile(s.ctx, chaosFileName)
	require.NoError(t, err, "Download should succeed")
	assert.Equal(t, chaosContent, downloadedChaos, "Downloaded content should match uploaded content")

	// Cleanup
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	helpers.DocumentResult(t, s.metrics, "Direct SeaweedFS upload/download succeeded with latency")
}

// TestSeaweedFSDirectUpload_WithResetPeer tests that the client handles
// connection reset scenarios gracefully.
func (s *ChaosTestSuite) TestSeaweedFSDirectUpload_WithResetPeer() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"handle connection reset gracefully during SeaweedFS operations",
		"SeaweedFS connections are reset after short timeout",
	))

	// Phase 1: Inject reset_peer chaos
	t.Log("Phase 1: Injecting connection reset chaos...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetSeaweedFSProxy()
	require.NotNil(t, proxy, "SeaweedFS proxy should exist")

	// Reset connections after 100ms - aggressive but should cause failures
	chaosConfig := helpers.DefaultResetPeerConfig(100)
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 2: Attempt file operations - expect failures
	t.Log("Phase 2: Attempting file operations under reset chaos...")
	resetFileName := fmt.Sprintf("/test/chaos/reset_%d.txt", time.Now().UnixNano())
	resetContent := []byte("Content that may fail to upload due to connection reset")

	uploadErr := uploadToSeaweedFS(s.ctx, s.chaosInfra.SeaweedFSProxyURL, resetFileName, resetContent)
	s.metrics.EndChaos()

	// Connection reset should cause upload failure
	if uploadErr != nil {
		t.Logf("Upload failed as expected under reset chaos: %v", uploadErr)
		s.metrics.RecordRequest(false, false, 0)
	} else {
		t.Log("Upload unexpectedly succeeded (connection may have completed before reset)")
		s.metrics.RecordRequest(true, false, 0)
	}

	// Phase 3: Remove chaos and verify recovery
	t.Log("Phase 3: Removing chaos, verifying recovery...")
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	s.metrics.StartRecovery()
	time.Sleep(setup.RecoveryObservationTime)

	// Upload should succeed after chaos is removed
	recoveryFileName := fmt.Sprintf("/test/chaos/recovery_%d.txt", time.Now().UnixNano())
	recoveryContent := []byte("Recovery test content - should upload successfully")

	recoveryStart := time.Now()
	err = uploadToSeaweedFS(s.ctx, s.chaosInfra.SeaweedFSProxyURL, recoveryFileName, recoveryContent)
	recoveryDuration := time.Since(recoveryStart)

	require.NoError(t, err, "Recovery upload should succeed")
	s.metrics.RecordRequest(true, true, recoveryDuration)

	// Verify recovery file
	downloadedRecovery, err := s.seaweedClient.GetFile(s.ctx, recoveryFileName)
	require.NoError(t, err, "Recovery download should succeed")
	assert.Equal(t, recoveryContent, downloadedRecovery, "Downloaded content should match")

	s.metrics.EndRecovery()

	helpers.DocumentResult(t, s.metrics, "System recovered from SeaweedFS connection resets")
}

// =============================================================================
// Helper Functions
// =============================================================================

// uploadToSeaweedFS uploads content to SeaweedFS using multipart/form-data.
// SeaweedFS filer requires multipart encoding for file uploads.
func uploadToSeaweedFS(ctx context.Context, baseURL, path string, content []byte) error {
	url := fmt.Sprintf("%s%s", baseURL, path)

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Use the filename from the path
	filename := filepath.Base(path)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	httpClient := &http.Client{Timeout: setup.SeaweedFSFileTimeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
