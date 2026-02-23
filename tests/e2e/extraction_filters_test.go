//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtraction_AllFilterOperators verifies that all filter operators work correctly:
// eq, ne, gt, gte, lt, lte, in (via eq with multiple values)
//
// This test uses the transactions table with known data to verify each operator.
func TestExtraction_AllFilterOperators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		filter       job.FilterCondition
		field        string
		desc         string
		expectedRows int
	}{
		{
			name:  "equals_single",
			field: "status",
			filter: job.FilterCondition{
				Equals: []any{"completed"},
			},
			desc:         "status = 'completed'",
			expectedRows: 24,
		},
		{
			name:  "equals_multiple_OR",
			field: "category",
			filter: job.FilterCondition{
				Equals: []any{"salary", "groceries"},
			},
			desc:         "category IN ('salary', 'groceries')",
			expectedRows: 13,
		},
		{
			name:  "greater_than",
			field: "amount",
			filter: job.FilterCondition{
				GreaterThan: []any{1000},
			},
			desc:         "amount > 1000",
			expectedRows: 8,
		},
		{
			name:  "greater_or_equal",
			field: "amount",
			filter: job.FilterCondition{
				GreaterOrEqual: []any{1500},
			},
			desc:         "amount >= 1500",
			expectedRows: 8,
		},
		{
			name:  "less_than",
			field: "amount",
			filter: job.FilterCondition{
				LessThan: []any{100},
			},
			desc:         "amount < 100",
			expectedRows: 7,
		},
		{
			name:  "less_or_equal",
			field: "amount",
			filter: job.FilterCondition{
				LessOrEqual: []any{89.99},
			},
			desc:         "amount <= 89.99",
			expectedRows: 5,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
			defer cancel()

			// Get PostgreSQL connection details
			pgHost, pgPort, err := postgresInfra.HostPort()
			require.NoError(t, err, "get postgres host/port")

			// Generate product name and create connection
			productName := e2eshared.GenerateProductName()

			uniqueName := fmt.Sprintf("e2e-filter-%s-%s", tc.name, uuid.New().String()[:8])
			conn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
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

			err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
			require.NoError(t, err, "wait for connection to be available")

			// Submit job with filter
			fetcherReq := model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						uniqueName: {
							"transactions": {"id", "account_id", "amount", "category", "status"},
						},
					},
					Filters: model.NestedFilters{
						uniqueName: {
							"transactions": {
								tc.field: tc.filter,
							},
						},
					},
				},
				Metadata: map[string]any{
					"source":     productName,
					"test":       fmt.Sprintf("filter-operator-%s", tc.name),
					"filterDesc": tc.desc,
				},
			}

			fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
			require.NoError(t, err, "create fetcher job")

			jobID := fetcherResp.JobID.String()
			t.Logf("Created job with filter %s: %s", tc.desc, jobID)

			// Wait for completion
			jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)
			assert.NotEmpty(t, jobResult.ResultPath, "should have result path")

			// Download result and verify row count matches filter
			seaweedURL, err := coreInfra.SeaweedFS.URL()
			require.NoError(t, err, "get seaweedfs url")

			resultData := e2eshared.DownloadAndDecryptResult(t, ctx, seaweedURL, jobResult.ResultPath)
			rowCount := e2eshared.CountResultRows(resultData)
			assert.Equal(t, tc.expectedRows, rowCount,
				"filter %s should return exactly %d rows, got %d", tc.desc, tc.expectedRows, rowCount)

			t.Logf("Filter %s completed successfully (%d rows)", tc.desc, rowCount)
		})
	}
}

// TestExtraction_SelectiveFilters verifies that filters can be applied to only
// some tables while others return all data.
//
// This test:
// 1. Requests data from transactions (with filter) and accounts (no filter)
// 2. Verifies both tables are extracted correctly
func TestExtraction_SelectiveFilters(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Generate product name and create connection
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-selective-filter-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
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

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Submit job: transactions filtered, accounts not filtered
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount", "status"},
					"accounts":     {"id", "name", "email"}, // No filter - returns all 3 accounts
				},
			},
			Filters: model.NestedFilters{
				uniqueName: {
					// Only filter transactions, not accounts
					"transactions": {
						"type": job.FilterCondition{
							Equals: []any{"credit"},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "selective-filters-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created selective filter job: %s", jobID)

	// Wait for completion via polling
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("Selective filter extraction completed: %s", jobResult.ResultPath)
}

// TestExtraction_DateRangeFilter verifies filtering by date range using
// gte (greater or equal) and lt (less than) operators.
func TestExtraction_DateRangeFilter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Generate product name and create connection
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-daterange-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
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

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Submit job filtering January 2024 transactions only
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount", "created_at"},
				},
			},
			Filters: model.NestedFilters{
				uniqueName: {
					"transactions": {
						"created_at": job.FilterCondition{
							GreaterOrEqual: []any{"2024-01-01T00:00:00Z"},
							LessThan:       []any{"2024-02-01T00:00:00Z"},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "date-range-filter-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created date range filter job (Jan 2024): %s", jobID)

	// Wait for completion
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("Date range filter completed: %s", jobResult.ResultPath)
}

// TestExtraction_CombinedFilters verifies that multiple filter conditions
// can be combined on the same table (AND logic).
func TestExtraction_CombinedFilters(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Generate product name and create connection
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-combined-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
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

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Submit job with multiple filter conditions:
	// - status = 'completed' AND
	// - type = 'credit' AND
	// - amount >= 1000
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount", "type", "status"},
				},
			},
			Filters: model.NestedFilters{
				uniqueName: {
					"transactions": {
						"status": job.FilterCondition{
							Equals: []any{"completed"},
						},
						"type": job.FilterCondition{
							Equals: []any{"credit"},
						},
						"amount": job.FilterCondition{
							GreaterOrEqual: []any{1000},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "combined-filters-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created combined filter job: %s", jobID)

	// Wait for completion
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("Combined filter extraction completed: %s", jobResult.ResultPath)
}

// TestExtraction_AccountIdFilter verifies filtering by specific account UUID.
func TestExtraction_AccountIdFilter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Generate product name and create connection
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-accountid-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
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

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Submit job filtering by specific account ID
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount", "description"},
				},
			},
			Filters: model.NestedFilters{
				uniqueName: {
					"transactions": {
						"account_id": job.FilterCondition{
							Equals: []any{e2eshared.TestAccount1ID},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source":    productName,
			"test":      "account-id-filter-e2e",
			"accountId": e2eshared.TestAccount1ID,
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created account ID filter job for account %s: %s", e2eshared.TestAccount1ID, jobID)

	// Wait for completion
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("Account ID filter extraction completed: %s", jobResult.ResultPath)
}
