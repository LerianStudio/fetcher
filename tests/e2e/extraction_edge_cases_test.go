//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtraction_EmptyResultSet verifies that a job completes successfully
// even when the filter conditions result in zero matching records.
//
// This is an important edge case because:
// 1. The system should not error when no data matches
// 2. The job should complete with status "completed" not "failed"
// 3. The result file should be created (even if empty or with headers only)
func TestExtraction_EmptyResultSet(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-empty-result-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Submit job with filter that matches no records
	// Using a non-existent account ID ensures zero results
	nonExistentAccountID := "99999999-9999-9999-9999-999999999999"

	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
			Filters: model.NestedFilters{
				uniqueName: {
					"transactions": {
						"account_id": job.FilterCondition{
							Equals: []any{nonExistentAccountID},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": "reporter",
			"test":   "empty-result-set-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created job expecting empty result: %s", jobID)

	// Wait for completion - should still complete, not fail
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	// Job should complete successfully even with no matching data
	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status,
		"job should complete successfully even with empty result set")
	assert.NotEmpty(t, jobResult.ResultPath,
		"result path should be set even for empty results")

	t.Logf("Empty result job completed: status=%s, resultPath=%s",
		jobResult.Status, jobResult.ResultPath)
}

// TestExtraction_EmptyResultSet_MultipleFilters verifies that a job completes
// when multiple filter conditions together result in zero matching records.
func TestExtraction_EmptyResultSet_MultipleFilters(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-empty-multi-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Submit job with conflicting filters that guarantee no results
	// Looking for credit transactions over 10000 (none exist in test data)
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount", "type"},
				},
			},
			Filters: model.NestedFilters{
				uniqueName: {
					"transactions": {
						"type": job.FilterCondition{
							Equals: []any{"credit"},
						},
						"amount": job.FilterCondition{
							GreaterThan: []any{10000}, // No credit > 10000 in test data
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": "reporter",
			"test":   "empty-result-multiple-filters-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created job with conflicting filters: %s", jobID)

	// Wait for completion
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status,
		"job should complete with conflicting filters")
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("Conflicting filters job completed: %s", jobResult.ResultPath)
}

// TestExtraction_LargeAmountFilter verifies extraction with very large numeric filters.
func TestExtraction_LargeAmountFilter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-largeamt-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Test with very large numbers (boundary testing)
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "amount"},
				},
			},
			Filters: model.NestedFilters{
				uniqueName: {
					"transactions": {
						"amount": job.FilterCondition{
							LessThan: []any{999999999.99}, // Very large number
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": "reporter",
			"test":   "large-amount-filter-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()

	// Wait for completion
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	t.Logf("Large amount filter job completed: %s", jobResult.ResultPath)
}

// TestExtraction_SpecialCharactersInFilter verifies that special characters
// in filter values are handled correctly.
func TestExtraction_SpecialCharactersInFilter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-special-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Filter with special characters that could be SQL injection attempts
	// The system should handle these safely
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "description"},
				},
			},
			Filters: model.NestedFilters{
				uniqueName: {
					"transactions": {
						"description": job.FilterCondition{
							// This should be safely escaped/parameterized
							Equals: []any{"Test's Value", "Value with \"quotes\""},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": "reporter",
			"test":   "special-characters-filter-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created job with special characters filter: %s", jobID)

	// Job should complete without error (though likely with empty results
	// since test data doesn't have these descriptions)
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status,
		"job should handle special characters safely")

	t.Logf("Special characters filter job completed safely: %s", jobResult.ResultPath)
}

// TestExtraction_AllFieldsFromTable verifies extraction when requesting
// all commonly used fields from a table.
func TestExtraction_AllFieldsFromTable(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create connection
	uniqueName := fmt.Sprintf("e2e-allfields-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Request all fields from transactions table
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {
						"id",
						"account_id",
						"amount",
						"currency",
						"type",
						"description",
						"category",
						"status",
						"created_at",
						"updated_at",
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": "reporter",
			"test":   "all-fields-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()

	// Wait for completion
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("All fields extraction completed: %s", jobResult.ResultPath)
}
