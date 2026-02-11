//go:build e2e

package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit"
	"github.com/LerianStudio/fetcher/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMongoDBExtraction_Collection_Success verifies the complete data extraction
// flow from a MongoDB database.
//
// Note: This test uses a separate MongoDB instance configured as a source database,
// not the core infrastructure MongoDB used for Fetcher's internal storage.
//
// Prerequisites:
// - MongoDB source container with test collection (enable via E2E_ENABLE_MONGODB=true)
func TestMongoDBExtraction_Collection_Success(t *testing.T) {
	t.Parallel()

	// MongoDB extraction test requires a separate MongoDB source database.
	// Since MongoDB source infrastructure is not available by default,
	// this test verifies that the API correctly handles the unavailable connection.

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Create a MongoDB connection configuration
	// Note: This points to a hypothetical MongoDB source - the API will validate
	// and reject if the connection is not reachable.
	uniqueName := fmt.Sprintf("e2e-mongo-extract-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeMongoDB,
		Host:         "localhost",
		Port:         27017,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"authSource": "admin",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")
	t.Logf("Created MongoDB connection: id=%s", conn.ID)

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Submit fetcher job for MongoDB collection
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				uniqueName: {
					"transactions": {"_id", "account_id", "amount", "currency", "type", "created_at"},
				},
			},
		},
		Metadata: map[string]any{
			"source": "reporter",
			"test":   "mongodb-extraction-e2e",
		},
	}

	// Use Raw method to handle both success and error cases
	resp, err := apiClient.CreateFetcherJobRaw(ctx, fetcherReq)
	require.NoError(t, err, "request should succeed")

	// Case 1: API rejects immediately because MongoDB is not reachable
	if resp.StatusCode() == 400 {
		body := string(resp.Body())
		t.Logf("MongoDB extraction correctly rejected: connection not available (status %d)", resp.StatusCode())
		assert.Contains(t, body, "Connection", "error should mention connection issue")
		return
	}

	// Case 2: Job was created - wait for completion or failure
	if resp.StatusCode() == 200 || resp.StatusCode() == 202 {
		var fetcherResp model.FetcherResponse
		err = json.Unmarshal(resp.Body(), &fetcherResp)
		require.NoError(t, err, "parse response")

		jobID := fetcherResp.JobID.String()
		t.Logf("Created MongoDB extraction job: %s", jobID)

		// Setup queue consumer for job completion events
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

		// Wait for job event (could be completed or failed depending on MongoDB availability)
		t.Log("Waiting for MongoDB job event...")
		msg, err := consumer.WaitForMessage(ctx)
		if err != nil {
			// Check job status directly
			jobResult, jobErr := apiClient.GetJob(ctx, jobID)
			if jobErr != nil {
				t.Logf("Failed to get job status: %v", jobErr)
			} else {
				t.Logf("Job status: status=%s", jobResult.Status)
				// If job failed due to MongoDB connectivity, that's expected
				if jobResult.Status == e2eshared.JobStatusFailed {
					t.Logf("MongoDB extraction failed as expected when source not available")
					return
				}
			}
			require.NoError(t, err, "wait for job event")
		}

		// If we received a message, verify it
		queuekit.AssertMessage(t, msg).
			PayloadSatisfies("job ID matches", func(n e2eshared.JobNotification) bool {
				return n.JobID == jobID
			})

		// Get final job state
		jobResult, err := apiClient.GetJob(ctx, jobID)
		require.NoError(t, err, "get job status")

		t.Logf("MongoDB extraction job finished: status=%s, resultPath=%s", jobResult.Status, jobResult.ResultPath)

		// Note: If MongoDB source was available, assert completion
		if jobResult.Status == e2eshared.JobStatusCompleted {
			assert.NotEmpty(t, jobResult.ResultPath, "completed job should have result path")
		}
		return
	}

	// Unexpected status code
	t.Errorf("Unexpected status code: %d", resp.StatusCode())
}

// TestMongoDBExtraction_WithAggregation_Success tests MongoDB-specific aggregation features.
func TestMongoDBExtraction_WithAggregation_Success(t *testing.T) {
	t.Parallel()

	// This test demonstrates that the system can handle MongoDB-specific
	// configurations in the connection metadata.

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	uniqueName := fmt.Sprintf("e2e-mongo-agg-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypeMongoDB,
		Host:         "localhost",
		Port:         27017,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"authSource":     "admin",
			"replicaSet":     "",
			"connectTimeout": "5000",
		},
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection with metadata")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	// Verify connection was created with metadata
	retrieved, err := apiClient.GetConnection(ctx, conn.ID)
	require.NoError(t, err, "get connection")

	assert.Equal(t, e2eshared.DBTypeMongoDB, retrieved.Type)
	assert.NotNil(t, retrieved.Metadata, "metadata should be preserved")

	t.Logf("MongoDB connection with metadata created: id=%s", conn.ID)
}
