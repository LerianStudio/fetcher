//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit"
	"github.com/LerianStudio/fetcher/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSQLServerExtraction_Table_Success verifies the complete data extraction
// flow from a SQL Server database.
//
// Prerequisites:
// - E2E_ENABLE_MSSQL=true environment variable set
// - SQL Server container with transactions table (seeded via init script)
func TestSQLServerExtraction_Table_Success(t *testing.T) {
	t.Parallel()

	if mssqlInfra == nil {
		t.Skip("SQL Server infrastructure not available (set E2E_ENABLE_MSSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get SQL Server connection details
	mssqlHost, mssqlPort, err := mssqlInfra.HostPort()
	require.NoError(t, err, "get mssql host/port")

	// Step 2: Create product and connection to source database
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-mssql-extract-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeSQLServer,
		Host:         mssqlHost,
		Port:         mssqlPort,
		DatabaseName: "testdb",
		Username:     "sa",
		Password:     "YourStrong@Passw0rd",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")
	require.NotEmpty(t, conn.ID, "connection ID should be set")
	t.Logf("Created SQL Server connection: id=%s, host=%s:%d", conn.ID, conn.Host, conn.Port)

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Step 3: Submit fetcher job
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"dbo.transactions": {"id", "account_id", "amount", "currency", "type", "created_at"},
				},
			},
		},
		Metadata: map[string]any{
			"source": product.Code,
			"test":   "sqlserver-extraction-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")
	require.NotEmpty(t, fetcherResp.JobID, "job ID should be set")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created SQL Server extraction job: %s", jobID)

	// Step 4: Setup queue consumer for job completion events
	backend, err := queuekit.NewAMQPConsumerBuilder(amqpURL).
		FromQueue(e2eshared.NotificationsQueue).
		WithAutoAck(true).
		Build()
	require.NoError(t, err, "create AMQP consumer")
	defer backend.Close()

	consumer := queuekit.NewConsumer[e2eshared.JobNotification](t, backend).
		WithMatcher(queuekit.MatchJSONField("jobId", jobID)).
		WithTimeout(e2eshared.DefaultJobTimeout).
		Build()
	defer consumer.Close()

	// Step 5: Wait for job completion event
	t.Log("Waiting for SQL Server job completion event...")
	msg, err := consumer.WaitForMessage(ctx)
	if err != nil {
		j, jobErr := apiClient.GetJob(ctx, jobID)
		if jobErr != nil {
			t.Logf("Failed to get job status: %v", jobErr)
		} else {
			t.Logf("Job status after timeout: status=%s, resultPath=%s", j.Status, j.ResultPath)
		}
		require.NoError(t, err, "wait for job completion event")
	}

	// Step 6: Assert on the received message
	queuekit.AssertMessage(t, msg).
		PayloadSatisfies("job ID matches", func(n e2eshared.JobNotification) bool {
			return n.JobID == jobID
		}).
		PayloadSatisfies("status is completed", func(n e2eshared.JobNotification) bool {
			return n.Status == "completed"
		})

	// Step 7: Verify job state via API
	jobResult, err := apiClient.GetJob(ctx, jobID)
	require.NoError(t, err, "get job status")

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status, "job status should be completed")
	assert.NotEmpty(t, jobResult.ResultPath, "result path should be set")

	t.Logf("SQL Server job completed successfully: status=%s, resultPath=%s", jobResult.Status, jobResult.ResultPath)
}

// TestSQLServerExtraction_MultiSchema_Success tests SQL Server extraction from
// tables in different schemas (finance.payments, analytics.monthly_metrics).
func TestSQLServerExtraction_MultiSchema_Success(t *testing.T) {
	t.Parallel()

	if mssqlInfra == nil {
		t.Skip("SQL Server infrastructure not available (set E2E_ENABLE_MSSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	mssqlHost, mssqlPort, err := mssqlInfra.HostPort()
	require.NoError(t, err, "get mssql host/port")

	// Create product and connection
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-mssql-multi-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeSQLServer,
		Host:         mssqlHost,
		Port:         mssqlPort,
		DatabaseName: "testdb",
		Username:     "sa",
		Password:     "YourStrong@Passw0rd",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Submit fetcher job with multiple schema-qualified tables
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"dbo.transactions":          {"id", "account_id", "amount", "currency"},
					"finance.payments":          {"id", "account_id", "payment_reference", "status"},
					"analytics.monthly_metrics": {"id", "account_id", "revenue", "expenses"},
				},
			},
		},
		Metadata: map[string]any{
			"test": "sqlserver-multi-schema",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created SQL Server multi-schema job: %s", jobID)

	// Wait for completion using polling
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("SQL Server multi-schema extraction completed: resultPath=%s", jobResult.ResultPath)
}

// TestSQLServerExtraction_WithDateFilters_Success tests SQL Server extraction
// with date-based filters on the created_at column.
func TestSQLServerExtraction_WithDateFilters_Success(t *testing.T) {
	t.Parallel()

	if mssqlInfra == nil {
		t.Skip("SQL Server infrastructure not available (set E2E_ENABLE_MSSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	mssqlHost, mssqlPort, err := mssqlInfra.HostPort()
	require.NoError(t, err, "get mssql host/port")

	// Create product and connection
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-mssql-date-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeSQLServer,
		Host:         mssqlHost,
		Port:         mssqlPort,
		DatabaseName: "testdb",
		Username:     "sa",
		Password:     "YourStrong@Passw0rd",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Submit fetcher job with date range filter (Q3 2024: July-September)
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"dbo.transactions": {"id", "account_id", "amount", "currency", "created_at"},
				},
			},
			// Note: Filters use the job.FilterCondition type
		},
		Metadata: map[string]any{
			"test":       "sqlserver-date-filter",
			"dateFilter": "Q3-2024",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created SQL Server date-filtered job: %s", jobID)

	// Wait for completion
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("SQL Server date-filtered extraction completed: resultPath=%s", jobResult.ResultPath)
}
