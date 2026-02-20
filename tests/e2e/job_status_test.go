//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetJob_AfterCreation_Success verifies that a newly created job
// can be retrieved via GET and has valid fields.
// Note: We do not assert on a specific status (pending/processing/completed)
// because the worker may process jobs faster than the test can observe
// transient states. Instead, we verify the job exists and has valid structure.
func TestGetJob_AfterCreation_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create product and connection
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-job-get-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: map[string]any{
			"source": product.Code,
			"test":   "job-get-after-creation",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")
	require.NotEmpty(t, fetcherResp.JobID, "job ID should be set in creation response")

	jobID := fetcherResp.JobID.String()

	// Retrieve job via GET endpoint
	job, err := apiClient.GetJob(ctx, jobID)
	require.NoError(t, err, "get job should succeed")

	// Assert structural fields are present (not transient status)
	assert.Equal(t, jobID, job.ID.String(), "job ID should match creation response")
	assert.NotEmpty(t, job.CreatedAt, "created_at should be set")
	assert.NotEmpty(t, job.Status, "status should not be empty")

	t.Logf("Job retrieved: id=%s, status=%s", job.ID, job.Status)
}

// TestGetJob_Completed_Success verifies that a completed job
// shows the correct status and has a result path.
func TestGetJob_Completed_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create product and connection
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-job-complete-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Create job
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount", "currency"},
				},
			},
		},
		Metadata: map[string]any{
			"source": product.Code,
			"test":   "job-completed-status",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created job: %s", jobID)

	// Wait for job to complete
	job := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	// Verify completed state
	assert.Equal(t, e2eshared.JobStatusCompleted, job.Status, "job should be completed")
	assert.NotEmpty(t, job.ResultPath, "completed job should have result path")
	assert.NotNil(t, job.CompletedAt, "completed job should have completedAt timestamp")

	t.Logf("Job completed: status=%s, resultPath=%s, completedAt=%v",
		job.Status, job.ResultPath, job.CompletedAt)
}

// TestGetJob_NotFound_404 verifies that requesting a non-existent
// job returns a 404 Not Found error.
func TestGetJob_NotFound_404(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	nonExistentID := uuid.New().String()

	resp, err := apiClient.GetJobRaw(ctx, nonExistentID)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 404, "")
	t.Logf("Non-existent job correctly returned 404")
}

// TestCreateJob_DuplicateRequest_ReturnsExisting verifies that creating
// a job with the same request hash returns the existing job (idempotency).
func TestCreateJob_DuplicateRequest_ReturnsExisting(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create product and connection
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-job-dup-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Create a request with fixed metadata (no unique requestId) for idempotency testing
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: map[string]any{
			"source": product.Code,
			"test":   "job-duplicate-same",
		},
	}

	// Create first job
	resp2, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create first job")

	t.Logf("First job created: %s", resp2.JobID)

	// Create duplicate job with exact same request
	resp3, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create duplicate job")

	// The API deduplicates requests with the same hash within a time window.
	// Both requests have identical DataRequest and Metadata, so the second should
	// return the same job as the first (HTTP 200 instead of 202).
	assert.Equal(t, resp2.JobID, resp3.JobID,
		"duplicate request should return the same job ID (idempotency)")

	t.Logf("Idempotency verified: both requests returned job %s", resp2.JobID)
}

// TestGetJob_InvalidID_BadRequest verifies that requesting a job
// with an invalid ID format returns an appropriate error.
func TestGetJob_InvalidID_BadRequest(t *testing.T) {
	t.Parallel()

	invalidIDs := []string{
		"not-a-uuid",
		"12345",
		"invalid-uuid-format",
	}

	for _, invalidID := range invalidIDs {
		t.Run(invalidID, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer cancel()

			resp, err := apiClient.GetJobRaw(ctx, invalidID)
			require.NoError(t, err, "request should succeed")

			// API should return either 400 (bad request) or 404 (not found)
			assert.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 404,
				"should return 400 or 404, got %d", resp.StatusCode())
			t.Logf("Invalid ID %q returned status %d", invalidID, resp.StatusCode())
		})
	}
}
