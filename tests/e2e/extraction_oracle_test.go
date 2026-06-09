//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/addons/queuekit"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/v2/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOracleExtraction_Table_Success verifies the complete data extraction
// flow from an Oracle database.
//
// Prerequisites:
// - E2E_ENABLE_ORACLE=true environment variable set
// - Oracle XE container with transactions table (seeded via init script)
func TestOracleExtraction_Table_Success(t *testing.T) {
	t.Parallel()

	if oracleInfra == nil {
		t.Skip("Oracle infrastructure not available (set E2E_ENABLE_ORACLE=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get Oracle connection details
	oracleHost, oraclePort, err := oracleInfra.HostPort()
	require.NoError(t, err, "get oracle host/port")

	// Step 2: Generate product name and create connection to source database
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-oracle-extract-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeOracle,
		Host:         oracleHost,
		Port:         oraclePort,
		DatabaseName: "XE", // Oracle SID
		Username:     "system",
		Password:     "testpass",
		Metadata: map[string]any{
			"serviceName": "XEPDB1", // Oracle service name
		},
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")
	require.NotEmpty(t, conn.ID, "connection ID should be set")
	t.Logf("Created Oracle connection: id=%s, host=%s:%d", conn.ID, conn.Host, conn.Port)

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
					"TRANSACTIONS": {"ID", "ACCOUNT_ID", "AMOUNT", "CURRENCY", "TYPE", "CREATED_AT"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "oracle-extraction-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")
	require.NotEmpty(t, fetcherResp.JobID, "job ID should be set")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created Oracle extraction job: %s", jobID)

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
	t.Log("Waiting for Oracle job completion event...")
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

	t.Logf("Oracle job completed successfully: status=%s, resultPath=%s", jobResult.Status, jobResult.ResultPath)
}

// TestOracleExtraction_MultiSchema_Success tests Oracle extraction from
// tables in different schemas (e.g., billing_subscriptions, audit_events).
func TestOracleExtraction_MultiSchema_Success(t *testing.T) {
	t.Parallel()

	if oracleInfra == nil {
		t.Skip("Oracle infrastructure not available (set E2E_ENABLE_ORACLE=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	oracleHost, oraclePort, err := oracleInfra.HostPort()
	require.NoError(t, err, "get oracle host/port")

	// Generate product name and create connection
	productName := e2eshared.GenerateProductName()

	uniqueName := fmt.Sprintf("e2e-oracle-multi-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeOracle,
		Host:         oracleHost,
		Port:         oraclePort,
		DatabaseName: "XE",
		Username:     "system",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Submit fetcher job with multiple tables including schema-prefixed tables
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"TRANSACTIONS":          {"ID", "ACCOUNT_ID", "AMOUNT", "CURRENCY"},
					"BILLING_SUBSCRIPTIONS": {"ID", "ACCOUNT_ID", "PLAN_NAME", "STATUS"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "oracle-multi-schema",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created Oracle multi-table job: %s", jobID)

	// Wait for completion using polling
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	t.Logf("Oracle multi-schema extraction completed: resultPath=%s", jobResult.ResultPath)
}
