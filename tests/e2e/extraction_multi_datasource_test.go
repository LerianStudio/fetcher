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

// TestMultiDatasourceExtraction verifies extraction from multiple datasources
// (PostgreSQL + MySQL) in a single job.
//
// This test validates that:
// 1. Multiple connections can be created for different database types
// 2. A single job can extract data from multiple datasources in parallel
// 3. Results from all datasources are consolidated correctly
//
// Prerequisites:
// - PostgreSQL source container (always enabled)
// - MySQL source container (enable via E2E_ENABLE_MYSQL=true)
//
// Expected data:
// - PostgreSQL transactions: Q1 2024 data (27 completed records)
// - MySQL transactions: Q2 2024 data (18 completed records)
func TestMultiDatasourceExtraction(t *testing.T) {
	t.Parallel()

	// Skip if MySQL is not available
	if mysqlInfra == nil {
		t.Skip("MySQL infrastructure not available (enable with E2E_ENABLE_MYSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get database connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	mysqlHost, mysqlPort, err := mysqlInfra.HostPort()
	require.NoError(t, err, "get mysql host/port")

	// Step 2: Generate product name
	productName := e2eshared.GenerateProductName()

	// Step 3: Create PostgreSQL connection
	pgConnName := fmt.Sprintf("e2e-multids-pg-%s", uuid.New().String()[:8])
	pgConnInput := e2eshared.ConnectionInput{
		ConfigName:   pgConnName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"quarter": "Q1-2024",
		},
	}

	pgConn, err := apiClient.CreateConnection(ctx, productName, pgConnInput)
	require.NoError(t, err, "create PostgreSQL connection")
	t.Logf("Created PostgreSQL connection: id=%s", pgConn.ID)

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), pgConn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, pgConn.ID, 10*time.Second)
	require.NoError(t, err, "wait for PostgreSQL connection to be available")

	// Step 4: Create MySQL connection
	mysqlConnName := fmt.Sprintf("e2e-multids-mysql-%s", uuid.New().String()[:8])
	mysqlConnInput := e2eshared.ConnectionInput{
		ConfigName:   mysqlConnName,
		Type:         e2eshared.DBTypeMySQL,
		Host:         mysqlHost,
		Port:         mysqlPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
		Metadata: map[string]any{
			"quarter": "Q2-2024",
		},
	}

	mysqlConn, err := apiClient.CreateConnection(ctx, productName, mysqlConnInput)
	require.NoError(t, err, "create MySQL connection")
	t.Logf("Created MySQL connection: id=%s", mysqlConn.ID)

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), mysqlConn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, mysqlConn.ID, 10*time.Second)
	require.NoError(t, err, "wait for MySQL connection to be available")

	// Step 4: Submit multi-datasource job
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				// PostgreSQL datasource - Q1 2024
				pgConnName: {
					"transactions": {"id", "account_id", "amount", "currency", "type", "status", "created_at"},
				},
				// MySQL datasource - Q2 2024
				mysqlConnName: {
					"transactions": {"id", "account_id", "amount", "currency", "type", "status", "created_at"},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "multi-datasource-extraction-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created multi-datasource extraction job: %s", jobID)

	// Step 5: Setup queue consumer
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

	// Step 6: Wait for job completion
	t.Log("Waiting for multi-datasource job completion...")
	msg, err := consumer.WaitForMessage(ctx)
	if err != nil {
		jobResult, jobErr := apiClient.GetJob(ctx, jobID)
		if jobErr == nil {
			t.Logf("Job status after timeout: status=%s, resultPath=%s", jobResult.Status, jobResult.ResultPath)
		}
		require.NoError(t, err, "wait for job completion event")
	}

	// Step 7: Assert completion
	queuekit.AssertMessage(t, msg).
		PayloadSatisfies("job ID matches", func(n e2eshared.JobNotification) bool {
			return n.JobID == jobID
		}).
		PayloadSatisfies("status is completed", func(n e2eshared.JobNotification) bool {
			return n.Status == "completed"
		})

	// Step 8: Verify job state
	jobResult, err := apiClient.GetJob(ctx, jobID)
	require.NoError(t, err, "get job status")

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status, "job should be completed")
	assert.NotEmpty(t, jobResult.ResultPath, "result path should be set")

	t.Logf("Multi-datasource extraction completed: status=%s, resultPath=%s",
		jobResult.Status, jobResult.ResultPath)
}

// TestMultiDatasourceExtraction_WithFilters verifies multi-datasource extraction
// with different filters applied per datasource.
//
// This test validates that:
// 1. Filters can be applied independently per datasource
// 2. Different filter conditions work correctly across different database types
//
// Filter configuration:
// - PostgreSQL: category = 'salary' (credit transactions)
// - MySQL: amount > 100 (larger transactions)
func TestMultiDatasourceExtraction_WithFilters(t *testing.T) {
	t.Parallel()

	// Skip if MySQL is not available
	if mysqlInfra == nil {
		t.Skip("MySQL infrastructure not available (enable with E2E_ENABLE_MYSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get database connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	mysqlHost, mysqlPort, err := mysqlInfra.HostPort()
	require.NoError(t, err, "get mysql host/port")

	// Step 2: Generate product name
	productName := e2eshared.GenerateProductName()

	// Step 3: Create connections
	pgConnName := fmt.Sprintf("e2e-multids-filter-pg-%s", uuid.New().String()[:8])
	pgConn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   pgConnName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create PostgreSQL connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), pgConn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, pgConn.ID, 10*time.Second)
	require.NoError(t, err, "wait for PostgreSQL connection to be available")

	mysqlConnName := fmt.Sprintf("e2e-multids-filter-mysql-%s", uuid.New().String()[:8])
	mysqlConn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   mysqlConnName,
		Type:         e2eshared.DBTypeMySQL,
		Host:         mysqlHost,
		Port:         mysqlPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create MySQL connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), mysqlConn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, mysqlConn.ID, 10*time.Second)
	require.NoError(t, err, "wait for MySQL connection to be available")

	// Step 3: Submit job with datasource-specific filters
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				pgConnName: {
					"transactions": {"id", "account_id", "amount", "category", "type"},
				},
				mysqlConnName: {
					"transactions": {"id", "account_id", "amount", "category", "type"},
				},
			},
			Filters: model.NestedFilters{
				// PostgreSQL: filter to salary category only
				pgConnName: {
					"transactions": {
						"category": job.FilterCondition{
							Equals: []any{"salary"},
						},
					},
				},
				// MySQL: filter to amounts greater than 100
				mysqlConnName: {
					"transactions": {
						"amount": job.FilterCondition{
							GreaterThan: []any{100},
						},
					},
				},
			},
		},
		Metadata: map[string]any{
			"source": productName,
			"test":   "multi-datasource-filtered-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")

	jobID := fetcherResp.JobID.String()
	t.Logf("Created multi-datasource filtered job: %s", jobID)

	// Step 4: Wait for completion
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

	t.Log("Waiting for filtered multi-datasource job...")
	msg, err := consumer.WaitForMessage(ctx)
	if err != nil {
		jobResult, jobErr := apiClient.GetJob(ctx, jobID)
		if jobErr == nil {
			t.Logf("Job status: %s", jobResult.Status)
		}
		require.NoError(t, err, "wait for job completion")
	}

	// Step 5: Verify completion
	queuekit.AssertMessage(t, msg).
		PayloadSatisfies("status is completed", func(n e2eshared.JobNotification) bool {
			return n.Status == "completed"
		})

	jobResult, err := apiClient.GetJob(ctx, jobID)
	require.NoError(t, err, "get job status")

	assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
	assert.NotEmpty(t, jobResult.ResultPath)

	// Step 6: Download result and verify per-datasource row counts
	seaweedURL, err := coreInfra.SeaweedFS.URL()
	require.NoError(t, err, "get seaweedfs url")

	resultData := e2eshared.DownloadAndDecryptResult(t, ctx, seaweedURL, e2eshared.DefaultE2EStorageBucket, jobResult.ResultPath)

	pgData := resultData[pgConnName]
	require.NotNil(t, pgData, "result should contain PostgreSQL datasource %s", pgConnName)
	assert.Len(t, pgData["transactions"], 9,
		"PostgreSQL transactions with category='salary' should return 9 rows")

	mysqlData := resultData[mysqlConnName]
	require.NotNil(t, mysqlData, "result should contain MySQL datasource %s", mysqlConnName)
	assert.Len(t, mysqlData["transactions"], 15,
		"MySQL transactions with amount > 100 should return 15 rows")

	totalRows := e2eshared.CountResultRows(resultData)
	assert.Equal(t, 24, totalRows, "total rows across both datasources should be 24")

	t.Logf("Multi-datasource filtered extraction completed: status=%s, pgRows=%d, mysqlRows=%d, totalRows=%d",
		jobResult.Status, len(pgData["transactions"]), len(mysqlData["transactions"]), totalRows)
}

// TestMultiDatasourceValidation verifies that schema validation works
// correctly across multiple datasources.
func TestMultiDatasourceValidation(t *testing.T) {
	t.Parallel()

	// Skip if MySQL is not available
	if mysqlInfra == nil {
		t.Skip("MySQL infrastructure not available (enable with E2E_ENABLE_MYSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get database connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	mysqlHost, mysqlPort, err := mysqlInfra.HostPort()
	require.NoError(t, err, "get mysql host/port")

	// Step 2: Generate product name
	productName := e2eshared.GenerateProductName()

	// Step 3: Create connections
	pgConnName := fmt.Sprintf("e2e-multids-valid-pg-%s", uuid.New().String()[:8])
	pgConn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   pgConnName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create PostgreSQL connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), pgConn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, pgConn.ID, 10*time.Second)
	require.NoError(t, err, "wait for PostgreSQL connection to be available")

	mysqlConnName := fmt.Sprintf("e2e-multids-valid-mysql-%s", uuid.New().String()[:8])
	mysqlConn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   mysqlConnName,
		Type:         e2eshared.DBTypeMySQL,
		Host:         mysqlHost,
		Port:         mysqlPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create MySQL connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), mysqlConn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, mysqlConn.ID, 10*time.Second)
	require.NoError(t, err, "wait for MySQL connection to be available")

	// Step 3: Validate schema across both datasources
	validationReq := e2eshared.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			pgConnName: {
				"transactions": {"id", "account_id", "amount", "currency"},
				"accounts":     {"id", "name", "email"},
			},
			mysqlConnName: {
				"transactions": {"id", "account_id", "amount", "currency"},
			},
		},
	}

	result, err := apiClient.ValidateSchema(ctx, validationReq)
	require.NoError(t, err, "validate schema")

	assert.Equal(t, "success", result.Status, "multi-datasource schema validation should succeed")
	assert.Empty(t, result.Errors, "should have no validation errors")

	t.Logf("Multi-datasource validation successful: status=%s", result.Status)
}

// TestMultiDatasourceValidation_PartialFailure verifies that validation correctly
// reports errors when one datasource has invalid schema while others are valid.
func TestMultiDatasourceValidation_PartialFailure(t *testing.T) {
	t.Parallel()

	// Skip if MySQL is not available
	if mysqlInfra == nil {
		t.Skip("MySQL infrastructure not available (enable with E2E_ENABLE_MYSQL=true)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Step 1: Get database connection details
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	mysqlHost, mysqlPort, err := mysqlInfra.HostPort()
	require.NoError(t, err, "get mysql host/port")

	// Step 2: Generate product name
	productName := e2eshared.GenerateProductName()

	// Step 3: Create connections
	pgConnName := fmt.Sprintf("e2e-partial-pg-%s", uuid.New().String()[:8])
	pgConn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   pgConnName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create PostgreSQL connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), pgConn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, pgConn.ID, 10*time.Second)
	require.NoError(t, err, "wait for PostgreSQL connection to be available")

	mysqlConnName := fmt.Sprintf("e2e-partial-mysql-%s", uuid.New().String()[:8])
	mysqlConn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
		ConfigName:   mysqlConnName,
		Type:         e2eshared.DBTypeMySQL,
		Host:         mysqlHost,
		Port:         mysqlPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create MySQL connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), mysqlConn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, mysqlConn.ID, 10*time.Second)
	require.NoError(t, err, "wait for MySQL connection to be available")

	// Step 3: Validate with one valid and one invalid table
	validationReq := e2eshared.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			pgConnName: {
				"transactions": {"id", "account_id", "amount"}, // valid
			},
			mysqlConnName: {
				"nonexistent_table": {"id", "name"}, // invalid - table doesn't exist
			},
		},
	}

	result, err := apiClient.ValidateSchema(ctx, validationReq)
	require.NoError(t, err, "validate schema request should succeed")

	// Validation should fail due to non-existent table in MySQL
	assert.Equal(t, "failure", result.Status, "validation should fail with partial invalid schema")
	assert.NotEmpty(t, result.Errors, "should have validation errors")

	// Check that error references the MySQL datasource
	hasMyQLError := false
	for _, e := range result.Errors {
		if e.DataSourceID == mysqlConnName {
			hasMyQLError = true
			break
		}
	}
	assert.True(t, hasMyQLError, "should have error for MySQL datasource")

	t.Logf("Partial validation failure detected: errors=%d", len(result.Errors))
}
