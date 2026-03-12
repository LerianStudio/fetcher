//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMySQLExtraction_TransactionsTable_Success verifies the complete data extraction
// flow from a MySQL database.
//
// Prerequisites:
// - E2E_ENABLE_MYSQL=true environment variable set
// - MySQL container with transactions table (seeded via init script)
func TestMySQLExtraction_TransactionsTable_Success(t *testing.T) {
	t.Parallel()

	if mysqlInfra == nil {
		t.Skip("MySQL infrastructure not available (set E2E_ENABLE_MYSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get MySQL connection details
	mysqlHost, mysqlPort, err := mysqlInfra.HostPort()
	require.NoError(t, err, "get mysql host/port")

	// Step 2: Generate product name and create connection to source database
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-mysql-extract-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeMySQL,
		Host:         mysqlHost,
		Port:         mysqlPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")
	require.NotEmpty(t, conn.ID, "connection ID should be set")
	t.Logf("Created MySQL connection: id=%s, host=%s:%d", conn.ID, conn.Host, conn.Port)

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
					"transactions": {"id", "account_id", "amount", "currency", "type", "created_at"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "mysql-extraction-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")
	require.NotEmpty(t, fetcherResp.JobID, "job ID should be set")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created job: %s", jobID)

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
	t.Log("Waiting for job completion event...")
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

	t.Logf("MySQL job completed successfully: status=%s, resultPath=%s", jobResult.Status, jobResult.ResultPath)
}

// TestMySQLExtraction_WithFilters_Success verifies that MySQL extraction
// works correctly with filter conditions.
func TestMySQLExtraction_WithFilters_Success(t *testing.T) {
	t.Parallel()

	if mysqlInfra == nil {
		t.Skip("MySQL infrastructure not available (set E2E_ENABLE_MYSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Get MySQL connection details
	mysqlHost, mysqlPort, err := mysqlInfra.HostPort()
	require.NoError(t, err, "get mysql host/port")

	// Generate product name and create connection
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-mysql-filter-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeMySQL,
		Host:         mysqlHost,
		Port:         mysqlPort,
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

	// Submit fetcher job with filters
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"id", "account_id", "amount", "currency", "type"},
				},
			},
			Filters: map[string]map[string]map[string]job.FilterCondition{
				uniqueName: {
					"transactions": {
						"currency": {
							Equals: []any{"USD"},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "mysql-extraction-with-filters",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created filtered MySQL job: %s", jobID)

	// Wait for completion using polling
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("MySQL filtered extraction completed: resultPath=%s", jobResult.ResultPath)
}
