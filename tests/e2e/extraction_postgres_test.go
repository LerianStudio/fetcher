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

// TestPostgresExtraction_TransactionsTable verifies the complete data extraction flow:
// 1. Create a connection to the PostgreSQL source database
// 2. Submit a fetcher job requesting specific fields from the transactions table
// 3. Wait for the job completion event via RabbitMQ
// 4. Verify the job completed successfully with a valid result path
//
// Prerequisites:
// - PostgreSQL container with transactions table (seeded via init script)
// - Manager and Worker containers running
// - RabbitMQ with the notifications queue configured
func TestPostgresExtraction_TransactionsTable(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get PostgreSQL connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Step 2: Create product and connection to source database
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	connName := fmt.Sprintf("e2e-pg-extract-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   connName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")
	require.NotEmpty(t, conn.ID, "connection ID should be set")
	t.Logf("Created connection: id=%s, host=%s:%d", conn.ID, conn.Host, conn.Port)

	t.Cleanup(func() {
		if err := apiClient.DeleteConnection(context.Background(), conn.ID); err != nil {
			t.Logf("Warning: failed to delete connection %s: %v", conn.ID, err)
		}
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Step 3: Submit fetcher job
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				connName: {
					"transactions": {"id", "account_id", "amount", "currency", "type", "created_at"},
				},
			},
		},
		Metadata: map[string]any{
			"source": product.Code,
			"test":   "postgres-extraction-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")
	require.NotEmpty(t, fetcherResp.JobID, "job ID should be set")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created job: %s", jobID)

	// Step 4: Wait for job completion using API polling
	// This approach is more reliable than RabbitMQ message consumption when tests run in parallel
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, "completed", jobResult.Status, "job status should be completed")
	assert.NotEmpty(t, jobResult.ResultPath, "result path should be set")

	t.Logf("Job completed successfully: status=%s, resultPath=%s", jobResult.Status, jobResult.ResultPath)
}
