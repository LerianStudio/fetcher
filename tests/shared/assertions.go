package shared

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertJobCompleted polls the job status until completion or timeout.
// It waits for the job to reach JobStatusCompleted status, failing the test if:
//   - The timeout is reached before completion
//   - The job transitions to JobStatusFailed
//
// On success, returns the final job response with the result path.
// Polls every 500ms to balance responsiveness with API load.
func AssertJobCompleted(t *testing.T, client *ManagerClient, jobID string, timeout time.Duration) *model.JobResponse {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var (
		lastJob *model.JobResponse
		lastErr error
	)

	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				t.Fatalf("job %s did not complete within %v: last error: %v", jobID, timeout, lastErr)
			}

			if lastJob != nil {
				t.Fatalf("job %s did not complete within %v: last status: %s", jobID, timeout, lastJob.Status)
			}

			t.Fatalf("job %s did not complete within %v", jobID, timeout)

			return nil

		case <-ticker.C:
			job, err := client.GetJob(ctx, jobID)
			if err != nil {
				lastErr = err
				continue
			}

			lastJob = job
			lastErr = nil

			switch job.Status {
			case JobStatusCompleted:
				assert.NotEmpty(t, job.ResultPath, "completed job should have result path")
				return job
			case JobStatusFailed:
				t.Fatalf("job %s failed: %v", jobID, job.Metadata)
				return nil
			}
		}
	}
}

// AssertJobFailed polls the job status until failure or timeout.
// It waits for the job to reach JobStatusFailed status, failing the test if:
//   - The timeout is reached before failure
//   - The job transitions to JobStatusCompleted (unexpected success)
//
// On success, returns the final job response.
// Polls every 500ms to balance responsiveness with API load.
func AssertJobFailed(t *testing.T, client *ManagerClient, jobID string, timeout time.Duration) *model.JobResponse {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var (
		lastJob *model.JobResponse
		lastErr error
	)

	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				t.Fatalf("job %s did not fail within %v: last error: %v", jobID, timeout, lastErr)
			}

			if lastJob != nil {
				t.Fatalf("job %s did not fail within %v: last status: %s", jobID, timeout, lastJob.Status)
			}

			t.Fatalf("job %s did not fail within %v", jobID, timeout)

			return nil

		case <-ticker.C:
			job, err := client.GetJob(ctx, jobID)
			if err != nil {
				lastErr = err
				continue
			}

			lastJob = job
			lastErr = nil

			switch job.Status {
			case JobStatusFailed:
				return job
			case JobStatusCompleted:
				t.Fatalf("job %s completed unexpectedly (expected failure): %v", jobID, job.Metadata)
				return nil
			}
		}
	}
}

// AssertAPIError validates that an HTTP response contains the expected error.
// It checks:
//   - The response is not nil
//   - The status code matches expectedCode
//   - The response body contains expectedMessageContains (if provided)
//
// Use this with the *Raw methods to test error scenarios.
func AssertAPIError(t *testing.T, resp *resty.Response, expectedCode int, expectedMessageContains string) {
	t.Helper()

	require.NotNil(t, resp, "response should not be nil")
	assert.Equal(t, expectedCode, resp.StatusCode(), "unexpected status code")

	if expectedMessageContains != "" {
		body := string(resp.Body())
		assert.True(t, strings.Contains(body, expectedMessageContains),
			"response body should contain %q, got: %s", expectedMessageContains, body)
	}
}

// AssertConnectionExists verifies that a connection exists in the database and returns it.
// Fails the test if:
//   - The API request fails
//   - The connection is not found (404)
//   - The returned ID doesn't match the requested ID
func AssertConnectionExists(t *testing.T, client *ManagerClient, connectionID string) *ConnectionResponse {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectTimeout)
	defer cancel()

	conn, err := client.GetConnection(ctx, connectionID)
	require.NoError(t, err, "connection should exist")
	require.NotNil(t, conn, "connection should not be nil")
	assert.Equal(t, connectionID, conn.ID, "connection ID should match")

	return conn
}

// AssertConnectionNotFound verifies that a connection does not exist (was deleted or never created).
// Expects the API to return 404 Not Found.
func AssertConnectionNotFound(t *testing.T, client *ManagerClient, connectionID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectTimeout)
	defer cancel()

	resp, err := client.GetConnectionRaw(ctx, connectionID)
	require.NoError(t, err, "request should succeed")
	assert.Equal(t, 404, resp.StatusCode(), "connection should not exist")
}

// AssertJobNotFound verifies that a job does not exist.
// Expects the API to return 404 Not Found.
func AssertJobNotFound(t *testing.T, client *ManagerClient, jobID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectTimeout)
	defer cancel()

	resp, err := client.GetJobRaw(ctx, jobID)
	require.NoError(t, err, "request should succeed")
	assert.Equal(t, 404, resp.StatusCode(), "job should not exist")
}

// AssertValidUUID validates that a string is a valid UUID v4 format.
// Expected format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (lowercase hex with dashes).
func AssertValidUUID(t *testing.T, value string) {
	t.Helper()
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, value, "should be valid UUID")
}

// RequireNoError is a convenience wrapper around require.NoError that properly marks itself as a helper.
// It fails the test immediately if err is not nil.
func RequireNoError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}
