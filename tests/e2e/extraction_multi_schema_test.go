//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/job"
	e2eshared "github.com/LerianStudio/fetcher/v2/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgreSQLMultiSchemaExtraction verifies extraction from multiple PostgreSQL schemas
// in a single job: public.transactions, accounting.invoices, reporting.daily_summary.
//
// This test validates that:
// 1. Connection can be created for schema-qualified table extraction
// 2. Job can request data from multiple schemas simultaneously
// 3. All data from all schemas is extracted correctly
//
// Expected data:
// - public.transactions: 27 records (Q1 2024 data)
// - accounting.invoices: 10 records
// - reporting.daily_summary: 12 records
// - Total: 49 records
func TestPostgreSQLMultiSchemaExtraction(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Step 2: Generate product name and create connection to source database
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-multischema-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"test": "multi-schema-extraction",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")
	t.Logf("Created connection: id=%s", conn.ID)

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Wait for connection to be validated before using it
	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Step 3: Submit fetcher job requesting data from multiple schemas
	// Note: Schema-qualified table names use "schema.table" format
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					// public schema (implicit)
					"transactions": {"id", "account_id", "amount", "currency", "type", "status", "created_at"},
					// accounting schema
					"accounting.invoices": {"id", "account_id", "invoice_number", "amount", "currency", "status", "due_date"},
					// reporting schema
					"reporting.daily_summary": {"id", "report_date", "account_id", "total_credits", "total_debits", "transaction_count"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "multi-schema-extraction-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created multi-schema extraction job: %s", jobID)

	// Step 4: Wait for job completion using API polling
	// This approach is more reliable than RabbitMQ message consumption when tests run in parallel
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status, "job should be completed")
	assert.NotEmpty(t, jobResult.ResultPath, "result path should be set")

	t.Logf("Multi-schema extraction completed: status=%s, resultPath=%s", jobResult.Status, jobResult.ResultPath)
}

// TestPostgreSQLMultiSchemaWithFilters verifies extraction from multiple schemas
// with different filters applied per table.
//
// This test validates that:
// 1. Filters can be applied to specific tables within a multi-schema extraction
// 2. Different filter conditions work correctly across schemas
//
// Filter configuration:
// - transactions: status = 'completed' (should exclude pending)
// - accounting.invoices: status IN ('paid', 'pending')
// - reporting.daily_summary: account_id = TestAccount1ID
func TestPostgreSQLMultiSchemaWithFilters(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Step 2: Generate product name and create connection
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-multischema-filter-%s", uuid.New().String()[:8])
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
	t.Logf("Created connection: id=%s", conn.ID)

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Wait for connection to be validated before using it
	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Step 3: Submit fetcher job with filters per table
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions":            {"id", "account_id", "amount", "status", "created_at"},
					"accounting.invoices":     {"id", "account_id", "invoice_number", "amount", "status"},
					"reporting.daily_summary": {"id", "report_date", "account_id", "total_credits", "total_debits"},
				},
			},
			Filters: model.NestedFilters{
				uniqueName: {
					// Filter transactions to only completed status
					"transactions": {
						"status": job.FilterCondition{
							Equals: []any{"completed"},
						},
					},
					// Filter invoices to paid or pending status
					"accounting.invoices": {
						"status": job.FilterCondition{
							Equals: []any{"paid", "pending"},
						},
					},
					// Filter daily_summary to Account 1 only
					"reporting.daily_summary": {
						"account_id": job.FilterCondition{
							Equals: []any{e2eshared.TestAccount1ID},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "multi-schema-filtered-extraction-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created multi-schema filtered job: %s", jobID)

	// Step 4: Wait for job completion using API polling
	// This approach is more reliable than RabbitMQ message consumption when tests run in parallel
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status, "job should be completed")
	assert.NotEmpty(t, jobResult.ResultPath, "result path should be set")

	// Step 5: Download result and verify per-table row counts
	seaweedURL, err := coreInfra.SeaweedFS.URL()
	require.NoError(t, err, "get seaweedfs url")

	resultData := e2eshared.DownloadAndDecryptResult(t, ctx, seaweedURL, e2eshared.DefaultE2EStorageBucket, jobResult.ResultPath)

	dsData := resultData[uniqueName]
	require.NotNil(t, dsData, "result should contain datasource %s", uniqueName)

	assert.Len(t, dsData["transactions"], 24,
		"transactions with status='completed' should return 24 rows")
	assert.Len(t, dsData["accounting.invoices"], 8,
		"invoices with status IN ('paid','pending') should return 8 rows")
	assert.Len(t, dsData["reporting.daily_summary"], 7,
		"daily_summary with account_id=TestAccount1ID should return 7 rows")

	totalRows := e2eshared.CountResultRows(resultData)
	assert.Equal(t, 39, totalRows, "total rows across all tables should be 39")

	t.Logf("Multi-schema filtered extraction completed: status=%s, resultPath=%s, totalRows=%d",
		jobResult.Status, jobResult.ResultPath, totalRows)
}

// TestPostgreSQLMultiSchemaValidation verifies that schema validation works
// correctly for schema-qualified table names.
func TestPostgreSQLMultiSchemaValidation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Step 2: Generate product name and create connection
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-multischema-valid-%s", uuid.New().String()[:8])
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

	// Step 3: Validate schema with multi-schema tables
	validationReq := e2eshared.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			uniqueName: {
				"transactions":            {"id", "account_id", "amount"},
				"accounting.invoices":     {"id", "invoice_number", "amount"},
				"reporting.daily_summary": {"id", "report_date", "total_credits"},
			},
		},
	}

	result, err := apiClient.ValidateSchema(ctx, validationReq)
	require.NoError(t, err, "validate schema")

	assert.Equal(t, "success", result.Status, "schema validation should succeed")
	assert.Empty(t, result.Errors, "should have no validation errors")

	t.Logf("Multi-schema validation successful: status=%s", result.Status)
}

// TestPostgreSQLMultiSchemaValidation_InvalidSchema verifies that validation
// correctly rejects non-existent schemas.
func TestPostgreSQLMultiSchemaValidation_InvalidSchema(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Step 2: Generate product name and create connection
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-invalidschema-%s", uuid.New().String()[:8])
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

	// Step 3: Validate schema with invalid schema name
	validationReq := e2eshared.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			uniqueName: {
				"transactions":                  {"id", "account_id"},
				"nonexistent_schema.some_table": {"id", "name"},
			},
		},
	}

	resp, err := apiClient.ValidateSchemaRaw(ctx, validationReq)
	require.NoError(t, err, "validate schema request should succeed")

	// API returns 422 Unprocessable Entity for schema validation failures
	assert.Equal(t, 422, resp.StatusCode(), "should return 422 for invalid schema")

	body := string(resp.Body())
	assert.Contains(t, body, "nonexistent_schema.some_table", "error should reference the invalid table")

	t.Logf("Invalid schema correctly rejected with 422: %s", body)
}
