//go:build e2e

package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/job"
	e2eshared "github.com/LerianStudio/fetcher/v2/tests/shared"
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
			"source": "any-product",
			"test":   "invalid-connection-name",
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
		t.Logf("Job created, polling for failure during processing")

		var fetcherResp model.FetcherResponse
		err = json.Unmarshal(resp.Body(), &fetcherResp)
		require.NoError(t, err, "parse response")

		jobID := fetcherResp.JobID.String()

		job := e2eshared.AssertJobFailed(t, apiClient, jobID, e2eshared.DefaultJobTimeout)
		assert.Equal(t, e2eshared.JobStatusFailed, job.Status, "job should fail for invalid connection name")
		t.Logf("Job correctly failed for invalid connection: %v", job.Metadata)
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
	productName := e2eshared.GenerateProductName()

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

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

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
			"source": productName,
			"test":   "invalid-table-name",
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
		t.Logf("Job created: %s, polling for failure", jobID)

		job := e2eshared.AssertJobFailed(t, apiClient, jobID, e2eshared.DefaultJobTimeout)
		assert.Equal(t, e2eshared.JobStatusFailed, job.Status, "job should fail for invalid table name")
		t.Logf("Job correctly failed for invalid table: %v", job.Metadata)
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
			"source": "any-product",
			"test":   "empty-mapped-fields",
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

	productName := e2eshared.GenerateProductName()

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

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

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
			"source": productName,
			"test":   "missing-fields",
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

	// maxDatasourcesLimit is the maximum number of datasources allowed per request.
	const maxDatasourcesLimit = 10

	// Create mapped fields exceeding the maximum allowed datasources
	mappedFields := make(map[string]map[string][]string)
	for i := 0; i < maxDatasourcesLimit+2; i++ {
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
			"source": "any-product",
			"test":   "too-many-datasources",
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
	productName := e2eshared.GenerateProductName()

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

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
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
			"source": productName,
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

		job := e2eshared.AssertJobFailed(t, apiClient, jobID, e2eshared.DefaultJobTimeout)
		assert.Equal(t, e2eshared.JobStatusFailed, job.Status, "job should fail with wrong credentials")
		t.Logf("Job with wrong credentials correctly failed: status=%s", job.Status)
		return
	}

	// Unexpected status code
	t.Errorf("Unexpected status code: %d", resp.StatusCode())
}

// TestCreateJob_InvalidMetadata_BadRequest verifies that creating a fetcher job
// with invalid or missing metadata fields is rejected with a 400 Bad Request error.
func TestCreateJob_InvalidMetadata_BadRequest(t *testing.T) {
	t.Parallel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	tests := []struct {
		name     string
		metadata map[string]any
	}{
		{
			name:     "missing_metadata",
			metadata: nil,
		},
		{
			name: "missing_source",
			metadata: map[string]any{
				"test": "value",
			},
		},
		{
			name: "empty_source",
			metadata: map[string]any{
				"source": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer cancel()

			productName := e2eshared.GenerateProductName()

			connInput := e2eshared.ConnectionInput{
				ConfigName:   fmt.Sprintf("e2e-meta-%s-%s", tt.name, uuid.New().String()[:8]),
				Type:         e2eshared.DBTypePostgreSQL,
				Host:         pgHost,
				Port:         pgPort,
				DatabaseName: "testdb",
				Username:     "testuser",
				Password:     "testpass",
			}

			conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, connInput)

			waitErr := apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
			require.NoError(t, waitErr, "wait for connection to be available")

			fetcherReq := model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						conn.ConfigName: {
							"transactions": {"id", "account_id", "amount"},
						},
					},
				},
				Metadata: tt.metadata,
			}

			resp, reqErr := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
			require.NoError(t, reqErr, "request should succeed")

			assert.Equal(t, 400, resp.StatusCode(),
				"invalid metadata %q should return 400 Bad Request, got %d", tt.name, resp.StatusCode())
			t.Logf("Invalid metadata %q correctly rejected with status %d", tt.name, resp.StatusCode())
		})
	}
}

// TestCreateFetcherJob_ProductMismatch_Rejected verifies that submitting a job
// with metadata.source pointing to product A but referencing a connection owned
// by product B is rejected with an appropriate error (product mismatch validation).
func TestCreateFetcherJob_ProductMismatch_Rejected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create product A name (the one we'll reference in metadata.source)
	productNameA := e2eshared.GenerateProductName()

	// Create product B with its own connection
	productNameB := e2eshared.GenerateProductName()

	connInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-mismatch-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	connB := e2eshared.CreateTestConnection(t, apiClient, ctx, productNameB, connInput)

	// Wait for connection B to be available
	err = apiClient.WaitForConnectionAvailable(ctx, connB.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection B to be available")

	// Submit job with metadata.source = productNameA but reference connB (owned by product B)
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				connB.ConfigName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productNameA,
			"test":   "product-mismatch-e2e",
		},
	}

	resp, err := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
	require.NoError(t, err, "request should succeed")

	// Should be rejected with 400 (ValidationError) for product mismatch
	assert.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 409,
		"product mismatch should return 400 or 409, got %d", resp.StatusCode())

	body := string(resp.Body())
	assert.True(t,
		assert.ObjectsAreEqual(true, containsAny(body, "Product Mismatch", "does not belong", "FET-1016")),
		"error should reference product mismatch: %s", body)

	t.Logf("Product mismatch correctly rejected with status %d", resp.StatusCode())
}

// containsAny checks if s contains any of the given substrings.
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}

	return false
}

// TestCreateFetcherJob_InvalidFilterReferences_BadRequest verifies that submitting
// a job with filters referencing a datasource not present in mappedFields is rejected.
func TestCreateFetcherJob_InvalidFilterReferences_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	connInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-badfilter-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, connInput)

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Submit job with filter referencing a datasource not in mappedFields
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				conn.ConfigName: {
					"transactions": {"id", "account_id", "amount", "status"},
				},
			},
			Filters: model.NestedFilters{
				// This datasource name does NOT exist in mappedFields
				"nonexistent-datasource": {
					"transactions": {
						"status": job.FilterCondition{
							Equals: []any{"completed"},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "invalid-filter-references-e2e",
		},
	}

	resp, err := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 400, resp.StatusCode(),
		"invalid filter reference should return 400 Bad Request, got %d", resp.StatusCode())

	body := string(resp.Body())
	assert.True(t, containsAny(body, "nonexistent-datasource", "filter", "unknown datasource"),
		"error should reference the invalid datasource: %s", body)

	t.Logf("Invalid filter reference correctly rejected with status %d", resp.StatusCode())
}
