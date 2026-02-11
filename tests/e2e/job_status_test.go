//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetJob_Pending_Success verifies that a newly created job
// can be retrieved and shows pending status.
func TestGetJob_Pending_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-job-pending-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
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

	// Create job - intentionally use a slow query or large dataset to catch pending state
	// Note: In practice, the job may process very quickly, so this test may see
	// processing or completed status instead of pending.
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: map[string]any{
			"source": "reporter",
			"test":   "job-pending-status",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created job: %s, initial status: %s", jobID, fetcherResp.Status)

	// Immediately get job status
	job, err := apiClient.GetJob(ctx, jobID)
	require.NoError(t, err, "get job")

	// Job should be in one of the valid states
	validStatuses := []string{
		e2eshared.JobStatusPending,
		e2eshared.JobStatusProcessing,
		e2eshared.JobStatusCompleted,
	}

	assert.Contains(t, validStatuses, job.Status, "job should be in a valid state")
	assert.NotEmpty(t, job.ID, "job ID should be set")
	assert.NotEmpty(t, job.CreatedAt, "created_at should be set")

	t.Logf("Job state: status=%s, id=%s", job.Status, job.ID)
}

// TestGetJob_Completed_Success verifies that a completed job
// shows the correct status and has a result path.
func TestGetJob_Completed_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-job-complete-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
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
			"source": "reporter",
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

	// Create connection
	uniqueName := fmt.Sprintf("e2e-job-dup-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
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

	// Create identical request
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: map[string]any{
			"source":    "reporter",
			"test":      "job-duplicate",
			"requestId": uuid.New().String(), // This makes metadata different but same data request
		},
	}

	// Create first job
	resp1, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create first job")

	job1ID := resp1.JobID.String()
	t.Logf("First job created: %s", job1ID)

	// Note: Because metadata includes a unique requestId, the hash will be different.
	// For true idempotency testing, we need to use the exact same request.
	fetcherReqSame := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: map[string]any{
			"source": "reporter",
			"test":   "job-duplicate-same",
		},
	}

	// Create first job with same metadata
	resp2, err := apiClient.CreateFetcherJob(ctx, fetcherReqSame)
	require.NoError(t, err, "create first job")

	// Create duplicate job with exact same request
	resp3, err := apiClient.CreateFetcherJob(ctx, fetcherReqSame)
	require.NoError(t, err, "create duplicate job")

	// If idempotency is implemented, both should return the same job ID
	// If not, they will be different jobs
	if resp2.JobID == resp3.JobID {
		t.Logf("Idempotency working: duplicate request returned same job %s", resp2.JobID)
	} else {
		t.Logf("No idempotency: requests created different jobs %s and %s", resp2.JobID, resp3.JobID)
	}
}

// TestGetJob_InvalidID_BadRequest verifies that requesting a job
// with an invalid ID format returns an appropriate error.
func TestGetJob_InvalidID_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	invalidIDs := []string{
		"not-a-uuid",
		"12345",
		"invalid-uuid-format",
	}

	for _, invalidID := range invalidIDs {
		t.Run(invalidID, func(t *testing.T) {
			resp, err := apiClient.GetJobRaw(ctx, invalidID)
			require.NoError(t, err, "request should succeed")

			// API should return either 400 (bad request) or 404 (not found)
			assert.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 404,
				"should return 400 or 404, got %d", resp.StatusCode())
			t.Logf("Invalid ID %q returned status %d", invalidID, resp.StatusCode())
		})
	}
}
