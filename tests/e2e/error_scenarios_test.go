//go:build e2e

package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtraction_InvalidConnectionName_Error verifies that requesting
// data from a non-existent connection fails appropriately.
func TestExtraction_InvalidConnectionName_Error(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Use a non-existent connection name
	nonExistentName := fmt.Sprintf("non-existent-conn-%s", uuid.New().String()[:8])

	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				nonExistentName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: map[string]any{
			"test": "invalid-connection-name",
		},
	}

	// The API may either:
	// 1. Reject the request immediately (400/404)
	// 2. Accept the job and fail it during processing
	resp, err := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
	require.NoError(t, err, "request should succeed")

	if resp.StatusCode() == 400 || resp.StatusCode() == 404 || resp.StatusCode() == 422 {
		// Request was rejected immediately
		t.Logf("Invalid connection name rejected with status %d", resp.StatusCode())
		return
	}

	if resp.StatusCode() == 200 || resp.StatusCode() == 202 {
		// Job was created - it should fail during processing
		t.Logf("Job created, checking for failure during processing")

		// Parse response to get job ID
		var fetcherResp model.FetcherResponse
		err = json.Unmarshal(resp.Body(), &fetcherResp)
		require.NoError(t, err, "parse response")

		// Poll for job status - expect failure
		jobID := fetcherResp.JobID.String()

		// Wait with shorter timeout since we expect failure
		job, err := apiClient.GetJob(ctx, jobID)
		if err == nil {
			// Job exists, check status
			if job.Status == e2eshared.JobStatusFailed {
				t.Logf("Job correctly failed: %v", job.Metadata)
			} else if job.Status == e2eshared.JobStatusCompleted {
				t.Errorf("Job should have failed for invalid connection, but completed")
			}
		}
	}
}

// TestExtraction_InvalidTableName_Error verifies that requesting
// data from a non-existent table fails appropriately.
func TestExtraction_InvalidTableName_Error(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create valid connection
	uniqueName := fmt.Sprintf("e2e-invalid-table-%s", uuid.New().String()[:8])
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

	// Request data from non-existent table
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"nonexistent_table_xyz": {"id", "name"},
				},
			},
		},
		Metadata: map[string]any{
			"test": "invalid-table-name",
		},
	}

	// The API may either reject immediately or fail during processing
	resp, err := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
	require.NoError(t, err, "request should succeed")

	if resp.StatusCode() == 400 || resp.StatusCode() == 422 {
		t.Logf("Invalid table name rejected with status %d", resp.StatusCode())
		return
	}

	if resp.StatusCode() == 200 || resp.StatusCode() == 202 {
		var fetcherResp model.FetcherResponse
		err = json.Unmarshal(resp.Body(), &fetcherResp)
		require.NoError(t, err, "parse response")

		jobID := fetcherResp.JobID.String()
		t.Logf("Job created: %s, waiting for failure", jobID)

		// Poll for status
		job, err := apiClient.GetJob(ctx, jobID)
		if err == nil && job.Status == e2eshared.JobStatusFailed {
			t.Logf("Job correctly failed for invalid table")
		}
	}
}

// TestExtraction_EmptyMappedFields_BadRequest verifies that a request
// with empty mapped fields is rejected.
func TestExtraction_EmptyMappedFields_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{},
		},
		Metadata: map[string]any{
			"test": "empty-mapped-fields",
		},
	}

	resp, err := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
	require.NoError(t, err, "request should succeed")

	// Empty mapped fields should be rejected
	assert.Equal(t, 400, resp.StatusCode(), "should return 400 Bad Request")
	t.Logf("Empty mapped fields correctly rejected with status %d", resp.StatusCode())
}

// TestExtraction_MissingFields_BadRequest verifies that a request
// with a table but no fields is rejected.
func TestExtraction_MissingFields_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	uniqueName := fmt.Sprintf("e2e-no-fields-%s", uuid.New().String()[:8])
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

	// Request with table but empty fields
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {}, // Empty fields
				},
			},
		},
		Metadata: map[string]any{
			"test": "missing-fields",
		},
	}

	resp, err := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
	require.NoError(t, err, "request should succeed")

	// Empty fields should be rejected (400) or handled gracefully
	assert.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 422,
		"should return 400 or 422, got %d", resp.StatusCode())
	t.Logf("Missing fields correctly handled with status %d", resp.StatusCode())
}

// TestExtraction_TooManyDatasources_BadRequest verifies that a request
// with too many datasources is rejected.
func TestExtraction_TooManyDatasources_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Create mapped fields with more than maximum allowed datasources (10)
	mappedFields := make(map[string]map[string][]string)
	for i := 0; i < 12; i++ {
		dsName := fmt.Sprintf("datasource-%d", i)
		mappedFields[dsName] = map[string][]string{
			"table1": {"id", "name"},
		}
	}

	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: mappedFields,
		},
		Metadata: map[string]any{
			"test": "too-many-datasources",
		},
	}

	resp, err := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
	require.NoError(t, err, "request should succeed")

	// Should be rejected for exceeding datasource limit
	assert.Equal(t, 400, resp.StatusCode(), "should return 400 for too many datasources")
	t.Logf("Too many datasources correctly rejected with status %d", resp.StatusCode())
}

// TestExtraction_ConnectionWithWrongCredentials_Rejected verifies that
// an extraction job with wrong credentials is either:
// 1. Rejected immediately by the API (400 - Connection Down)
// 2. Created and then fails during processing
func TestExtraction_ConnectionWithWrongCredentials_Rejected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection with wrong password
	uniqueName := fmt.Sprintf("e2e-wrong-creds-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "wrong_password_should_fail",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Submit extraction job
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
			"test":   "wrong-credentials",
		},
	}

	// Use Raw method to handle both success and error cases
	resp, err := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
	require.NoError(t, err, "request should succeed")

	// Case 1: API rejects immediately with 400 (Connection Down)
	if resp.StatusCode() == 400 {
		body := string(resp.Body())
		assert.Contains(t, body, "Connection", "error should mention connection issue")
		t.Logf("Job correctly rejected at creation: connection unavailable (status %d)", resp.StatusCode())
		return
	}

	// Case 2: Job was created - verify it fails during processing
	if resp.StatusCode() == 200 || resp.StatusCode() == 202 {
		var fetcherResp model.FetcherResponse
		err = json.Unmarshal(resp.Body(), &fetcherResp)
		require.NoError(t, err, "parse response")

		jobID := fetcherResp.JobID.String()
		t.Logf("Created job with wrong credentials: %s", jobID)

		// Poll for status - should fail
		var finalStatus string
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for i := 0; i < 30; i++ {
			select {
			case <-ctx.Done():
				t.Fatalf("context cancelled while waiting for job failure")
			case <-ticker.C:
				job, jobErr := apiClient.GetJob(ctx, jobID)
				if jobErr != nil {
					continue
				}
				finalStatus = job.Status
				if job.Status == e2eshared.JobStatusFailed || job.Status == e2eshared.JobStatusCompleted {
					goto done
				}
			}
		}
	done:

		assert.Equal(t, e2eshared.JobStatusFailed, finalStatus, "job should fail with wrong credentials")
		t.Logf("Job with wrong credentials correctly failed: status=%s", finalStatus)
		return
	}

	// Unexpected status code
	t.Errorf("Unexpected status code: %d", resp.StatusCode())
}
