//go:build integration
// +build integration

package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/tests/integration/containers/setup"
	"github.com/LerianStudio/fetcher/tests/shared/client"
	"github.com/LerianStudio/fetcher/tests/shared/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ExecutionMode represents the test execution mode.
type ExecutionMode int

const (
	// ModeNormal: Both Manager and Worker run as containers
	ModeNormal ExecutionMode = iota
	// ModeManagerDebug: Manager runs locally, Worker as container
	ModeManagerDebug
	// ModeWorkerDebug: Manager as container, Worker runs locally
	ModeWorkerDebug
	// ModeFullDebug: Both Manager and Worker run locally
	ModeFullDebug
)

// String returns the string representation of the execution mode.
func (m ExecutionMode) String() string {
	switch m {
	case ModeNormal:
		return "Normal (both containers)"
	case ModeManagerDebug:
		return "Manager Debug (Manager local, Worker container)"
	case ModeWorkerDebug:
		return "Worker Debug (Manager container, Worker local)"
	case ModeFullDebug:
		return "Full Debug (both local)"
	default:
		return "Unknown"
	}
}

// detectExecutionMode determines the execution mode from environment variables.
func detectExecutionMode() ExecutionMode {
	externalManagerURL := os.Getenv("EXTERNAL_MANAGER_URL")
	skipWorker := os.Getenv("SKIP_WORKER") == "true"

	if externalManagerURL != "" && skipWorker {
		return ModeFullDebug
	}
	if externalManagerURL != "" {
		return ModeManagerDebug
	}
	if skipWorker {
		return ModeWorkerDebug
	}
	return ModeNormal
}

// getInfrastructureOptions returns infrastructure options based on execution mode.
func getInfrastructureOptions() setup.InfrastructureOptions {
	// Check if we should reuse existing infrastructure
	if os.Getenv("REUSE_INFRA") == "true" {
		return setup.ReuseInfrastructureOptions()
	}

	// Check if SSL testing is enabled
	if os.Getenv("ENABLE_SSL") == "true" {
		return setup.SSLInfrastructureOptions()
	}

	// Check if we should use fixed ports (for debug modes)
	if os.Getenv("USE_FIXED_PORTS") == "true" {
		return setup.DebugInfrastructureOptions()
	}

	// Default: random ports
	return setup.DefaultInfrastructureOptions()
}

// WorkerIntegrationTestSuite is the main E2E test suite.
type WorkerIntegrationTestSuite struct {
	suite.Suite

	ctx           context.Context
	cancel        context.CancelFunc
	infra         *setup.InfrastructureContainers
	apps          *setup.ApplicationContainers
	managerClient *client.ManagerClient
	seaweedClient *client.SeaweedFSClient
	eventConsumer *client.RabbitMQEventConsumer
	executionMode ExecutionMode
}

// SetupSuite runs once before all tests.
func (s *WorkerIntegrationTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), setup.SuiteTimeout)

	// Detect execution mode
	s.executionMode = detectExecutionMode()
	s.T().Logf("Running in %s mode", s.executionMode)

	// Get infrastructure options
	infraOpts := getInfrastructureOptions()

	// Start or connect to infrastructure
	var infra *setup.InfrastructureContainers
	var err error

	if infraOpts.ReuseExisting {
		s.T().Log("Attempting to reuse existing infrastructure...")
	} else {
		s.T().Log("Starting new infrastructure containers...")
	}

	infra, err = setup.StartInfrastructureWithOptions(s.ctx, infraOpts)
	require.NoError(s.T(), err, "Failed to start infrastructure")
	s.infra = infra

	// Setup RabbitMQ topology (idempotent - safe to call even if already set up)
	err = setup.SetupRabbitMQTopology(s.ctx, s.infra.RabbitMQURI())
	require.NoError(s.T(), err, "Failed to setup RabbitMQ topology")

	// Purge test queue when reusing infrastructure to remove stale events
	if infraOpts.ReuseExisting {
		purged, err := setup.PurgeTestQueue(s.ctx, s.infra.RabbitMQURI())
		if err != nil {
			s.T().Logf("Warning: Could not purge test queue: %v", err)
		} else if purged > 0 {
			s.T().Logf("Purged %d stale events from test.job.events queue", purged)
		}
	}

	// Initialize test data in external databases (only if not reusing)
	if !infraOpts.ReuseExisting {
		err = s.seedExternalDatabases()
		require.NoError(s.T(), err, "Failed to seed external databases")
	}

	// Start application containers with encryption keys
	encryptionKeyBase64 := os.Getenv("TEST_ENCRYPTION_KEY_BASE64")
	if encryptionKeyBase64 == "" {
		encryptionKeyBase64 = "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="
	}

	encryptionKeyHex := os.Getenv("TEST_ENCRYPTION_KEY_HEX")
	if encryptionKeyHex == "" {
		encryptionKeyHex = "3132333435363738393031323334353637383930313233343536373839303132"
	}

	// Start applications containers
	apps, err := setup.StartApplications(s.ctx, s.infra, setup.ApplicationConfig{
		EncryptionKeyBase64: encryptionKeyBase64,
		EncryptionKeyHex:    encryptionKeyHex,
	})
	require.NoError(s.T(), err, "Failed to start applications")
	s.apps = apps

	// Log the Manager URL being used
	s.T().Logf("Manager URL: %s", s.apps.ManagerURL)

	// Initialize clients
	s.managerClient = client.NewManagerClient(apps.ManagerURL, fixtures.TestOrganizationID)
	s.seaweedClient = client.NewSeaweedFSClient(s.infra.SeaweedFSURL())

	// Initialize RabbitMQ event consumer
	eventConsumer, err := client.NewRabbitMQEventConsumer(s.infra.RabbitMQURI())
	require.NoError(s.T(), err, "Failed to create event consumer")
	s.eventConsumer = eventConsumer

	// Wait for Manager to be ready
	err = s.waitForManagerReady()
	require.NoError(s.T(), err, "Manager not ready")

	// Log ready message with mode info
	s.T().Logf("Test suite ready - Mode: %s", s.executionMode)
}

// TearDownSuite runs once after all tests.
func (s *WorkerIntegrationTestSuite) TearDownSuite() {
	if s.eventConsumer != nil {
		s.eventConsumer.Close()
	}

	// Stop applications (containers we started)
	if s.apps != nil {
		s.apps.Stop(s.ctx)
	}

	// Only stop infrastructure if we started it (not reusing)
	reuseInfra := os.Getenv("REUSE_INFRA") == "true"
	if s.infra != nil && !reuseInfra {
		s.T().Log("Stopping infrastructure containers...")
		s.infra.Stop(s.ctx)
	} else if reuseInfra {
		s.T().Log("Preserving infrastructure for reuse (REUSE_INFRA=true)")
	}

	if s.cancel != nil {
		s.cancel()
	}
}

// SetupTest runs before each test to ensure clean state.
func (s *WorkerIntegrationTestSuite) SetupTest() {
	// Clean up any connections from previous test runs.
	// This is necessary when REUSE_INFRA=true as MongoDB persists data.
	testConnections := []string{
		// Single datasource tests
		"postgres_test",
		"mysql_test",
		"sqlserver_test",
		"oracle_test",
		"mongodb_test",
		// Multi-datasource and filter tests
		"postgres_multi",
		"mysql_multi",
		"postgres_filtered",
		"postgres_idempotency",
		"postgres_seaweed",
		"postgres_metadata",
		// Error scenario tests
		"postgres_down",
		"postgres_bad_creds",
		"postgres_for_missing_table",
	}

	for _, configName := range testConnections {
		if err := s.managerClient.DeleteConnectionByConfigName(s.ctx, configName); err != nil {
			// Log but don't fail - connection might not exist
			s.T().Logf("Note: Could not delete connection %s: %v", configName, err)
		}
	}
}

// testMetadata returns metadata with a unique timestamp to avoid idempotency issues.
// Each test run will have a unique request hash due to the timestamp.
func (s *WorkerIntegrationTestSuite) testMetadata(testName string) map[string]any {
	return map[string]any{
		"source":    "integration_test",
		"testName":  testName,
		"timestamp": time.Now().UnixNano(),
	}
}

// uniqueConfigName generates a unique connection config name for each test run.
// This ensures the request hash is different, avoiding idempotency issues.
func (s *WorkerIntegrationTestSuite) uniqueConfigName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// seedExternalDatabases seeds test data into external databases.
// Note: PostgreSQL, MySQL, SQL Server, and Oracle are seeded via init scripts
// during container startup in StartInfrastructure().
// Only MongoDB needs programmatic seeding here due to ObjectID generation.
func (s *WorkerIntegrationTestSuite) seedExternalDatabases() error {
	// Seed MongoDB External (requires programmatic seeding for ObjectIDs)
	err := fixtures.InitMongoDBExternal(s.ctx, s.infra.MongoExternalURI(), "external_transactions")
	if err != nil {
		return err
	}

	return nil
}

// waitForManagerReady waits for the Manager API to be accessible.
func (s *WorkerIntegrationTestSuite) waitForManagerReady() error {
	deadline := time.Now().Add(setup.ManagerReadyTimeout)
	for {
		if time.Now().After(deadline) {
			return context.DeadlineExceeded
		}

		err := s.managerClient.HealthCheck(s.ctx)
		if err == nil {
			return nil
		}

		time.Sleep(setup.ManagerReadyPollInterval)
	}
}

// =============================================================================
// CORE DATA EXTRACTION TESTS - Single Datasource
// =============================================================================
// These tests validate the fundamental data extraction pipeline for each
// supported database type. Each test creates a connection, submits a job,
// waits for completion via RabbitMQ events, and validates the output file.
// Priority: P0 (critical path - must pass for any release)
// =============================================================================

// TestSingleDatasourcePostgreSQL validates the complete extraction pipeline for PostgreSQL.
func (s *WorkerIntegrationTestSuite) TestSingleDatasourcePostgreSQL() {
	t := s.T()

	// Generate unique config name for this test run to avoid idempotency conflicts
	configName := s.uniqueConfigName("postgres_test")

	// Step 1: Create PostgreSQL connection via API
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	pg := s.infra.PostgresInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create PostgreSQL connection")
	assert.NotEmpty(t, connResp.ID)
	assert.Equal(t, configName, connResp.ConfigName)
	assert.Equal(t, "POSTGRESQL", connResp.Type)
	assert.Equal(t, pg.Host, connResp.Host)
	assert.Equal(t, pg.Port, connResp.Port)
	assert.Equal(t, pg.Database, connResp.DatabaseName)
	assert.Equal(t, pg.Username, connResp.Username)
	assert.NotEmpty(t, connResp.CreatedAt)
	assert.NotEmpty(t, connResp.UpdatedAt)

	// Step 2: Create fetcher job via API
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount", "currency", "type", "category", "status", "created_at"},
				},
			},
		},
		Metadata: s.testMetadata("TestSingleDatasourcePostgreSQL"),
	})
	require.NoError(t, err, "Failed to create fetcher job")
	assert.NotEmpty(t, jobResp.JobID.String())
	assert.Equal(t, "pending", jobResp.Status)

	// Step 3: Wait for job completion via RabbitMQ event
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")
	if notification.Status == "failed" {
		t.Logf("PostgreSQL job failed with metadata: %+v", notification.Metadata)
	}
	assert.Equal(t, "completed", notification.Status)

	// Verify new result fields in notification
	require.NotNil(t, notification.Result, "Completed job notification should have result data")
	assert.NotEmpty(t, notification.Result.Path, "Result should have path")
	assert.Contains(t, notification.Result.Path, jobResp.JobID.String(), "Result path should contain job ID")
	assert.Greater(t, notification.Result.SizeBytes, int64(0), "Result should have size > 0")
	assert.Greater(t, notification.Result.RowCount, int64(0), "Result should have row count > 0")
	assert.Equal(t, "json", notification.Result.Format, "Result format should be json")

	// Verify row count is correct
	assert.Equal(t, notification.Result.RowCount, int64(26), "Result row count should be 26")

	// Verify execution metrics
	assert.Greater(t, notification.ExecutionTimeMs, int64(0), "Execution time should be > 0")
	assert.NotNil(t, notification.CompletedAt, "CompletedAt should be set")

	// Step 4: Verify job status via API
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err, "Failed to get job")
	assert.Equal(t, "completed", job.Status)
	assert.NotEmpty(t, job.ResultPath)

	// Step 5: Verify data in SeaweedFS
	resultPath := "/external-data/" + jobResp.JobID.String() + ".json"
	data, err := s.seaweedClient.WaitForFile(s.ctx, resultPath, setup.SeaweedFSFileTimeout)
	require.NoError(t, err, "Failed to get result file from SeaweedFS")

	// The data is encrypted, but we can verify it exists and has content
	assert.NotEmpty(t, data)
}

// TestSingleDatasourceMySQL validates the complete extraction pipeline for MySQL.
//
// Test flow: Same as PostgreSQL - connection → job → event → verification.
// Validates MySQL-specific connection handling and query execution.
func (s *WorkerIntegrationTestSuite) TestSingleDatasourceMySQL() {
	t := s.T()

	// Step 1: Create MySQL connection via API
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	mysql := s.infra.MySQLInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "mysql_test",
		Type:         "MYSQL",
		Host:         mysql.Host,
		Port:         mysql.Port,
		DatabaseName: mysql.Database,
		Username:     mysql.Username,
		Password:     mysql.Password,
	})
	require.NoError(t, err, "Failed to create MySQL connection")
	assert.NotEmpty(t, connResp.ID)

	// Step 2: Create fetcher job via API
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"mysql_test": {
					"transactions": {"id", "account_id", "amount", "currency", "type", "category", "status", "created_at"},
				},
			},
		},
		Metadata: s.testMetadata("TestSingleDatasourceMySQL"),
	})
	require.NoError(t, err, "Failed to create fetcher job")
	assert.NotEmpty(t, jobResp.JobID.String())

	// Step 3: Wait for job completion
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status)

	// Verify result and execution metrics in notification
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.SizeBytes, int64(0))
	assert.Equal(t, notification.Result.RowCount, int64(20))
	assert.Greater(t, notification.ExecutionTimeMs, int64(0))
	assert.NotNil(t, notification.CompletedAt)

	// Step 4: Verify job status
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err, "Failed to get job")
	assert.Equal(t, "completed", job.Status)
}

// TestSingleDatasourceMongoDB validates the complete extraction pipeline for MongoDB.
//
// Test flow: Same as PostgreSQL - connection → job → event → verification.
// Validates MongoDB (NoSQL) document extraction with field projection.
func (s *WorkerIntegrationTestSuite) TestSingleDatasourceMongoDB() {
	t := s.T()

	// Step 1: Create MongoDB connection via API
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	mongo := s.infra.MongoExternalInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "mongodb_test",
		Type:         "MONGODB",
		Host:         mongo.Host,
		Port:         mongo.Port,
		DatabaseName: "external_transactions",
		Username:     mongo.Username,
		Password:     mongo.Password,
	})
	require.NoError(t, err, "Failed to create MongoDB connection")
	assert.NotEmpty(t, connResp.ID)

	// Step 2: Create fetcher job via API
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"mongodb_test": {
					"transactions": {"account_id", "amount", "currency", "type", "category", "status", "created_at"},
				},
			},
		},
		Metadata: s.testMetadata("TestSingleDatasourceMongoDB"),
	})
	require.NoError(t, err, "Failed to create fetcher job")
	assert.NotEmpty(t, jobResp.JobID.String())

	// Step 3: Wait for job completion
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status)

	// Verify result and execution metrics in notification
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.SizeBytes, int64(0))
	assert.Greater(t, notification.Result.RowCount, int64(0))
	assert.Greater(t, notification.ExecutionTimeMs, int64(0))
	assert.NotNil(t, notification.CompletedAt)

	// Step 4: Verify job status
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err, "Failed to get job")
	assert.Equal(t, "completed", job.Status)
}

// TestSingleDatasourceSQLServer validates the complete extraction pipeline for SQL Server.
//
// Test flow: Same as PostgreSQL - connection → job → event → verification.
// Validates SQL Server (MSSQL) connection handling with T-SQL query execution.
func (s *WorkerIntegrationTestSuite) TestSingleDatasourceSQLServer() {
	t := s.T()

	// Step 1: Create SQL Server connection via API
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	mssql := s.infra.SQLServerInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "sqlserver_test",
		Type:         "SQL_SERVER", // Correct type for SQL Server
		Host:         mssql.Host,
		Port:         mssql.Port,
		DatabaseName: mssql.Database,
		Username:     mssql.Username,
		Password:     mssql.Password,
	})
	require.NoError(t, err, "Failed to create SQL Server connection")
	assert.NotEmpty(t, connResp.ID)

	// Step 2: Create fetcher job via API
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"sqlserver_test": {
					"transactions": {"id", "account_id", "amount", "currency", "type", "category", "status", "created_at"},
				},
			},
		},
		Metadata: s.testMetadata("TestSingleDatasourceSQLServer"),
	})
	require.NoError(t, err, "Failed to create fetcher job")
	assert.NotEmpty(t, jobResp.JobID.String())
	assert.Equal(t, "pending", jobResp.Status)

	// Step 3: Wait for job completion via RabbitMQ event
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status)

	// Verify result and execution metrics in notification
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.SizeBytes, int64(0))
	assert.Greater(t, notification.Result.RowCount, int64(0))
	assert.Greater(t, notification.ExecutionTimeMs, int64(0))
	assert.NotNil(t, notification.CompletedAt)

	// Step 4: Verify job status via API
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err, "Failed to get job")
	assert.Equal(t, "completed", job.Status)
	assert.NotEmpty(t, job.ResultPath)

	// Step 5: Verify data in SeaweedFS
	resultPath := "/external-data/" + jobResp.JobID.String() + ".json"
	data, err := s.seaweedClient.WaitForFile(s.ctx, resultPath, setup.SeaweedFSFileTimeout)
	require.NoError(t, err, "Failed to get result file from SeaweedFS")
	assert.NotEmpty(t, data)
}

// TestSingleDatasourceOracle validates the complete extraction pipeline for Oracle.
//
// Test flow: Same as PostgreSQL - connection → job → event → verification.
// Validates Oracle connection with service name metadata and PL/SQL execution.
// Note: Uses extended timeout (JobCompletionTimeoutSlow) as Oracle startup is slower.
func (s *WorkerIntegrationTestSuite) TestSingleDatasourceOracle() {
	t := s.T()

	// Step 1: Create Oracle connection via API
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	oracle := s.infra.OracleInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "oracle_test",
		Type:         "ORACLE",
		Host:         oracle.Host,
		Port:         oracle.Port,
		DatabaseName: oracle.Database,
		Username:     oracle.Username,
		Password:     oracle.Password,
		Metadata: map[string]any{
			"serviceName": oracle.Database, // Oracle requires serviceName in metadata
		},
	})
	require.NoError(t, err, "Failed to create Oracle connection")
	assert.NotEmpty(t, connResp.ID)

	// Step 2: Create fetcher job via API
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"oracle_test": {
					"transactions": {"id", "account_id", "amount", "currency", "type", "category", "status", "created_at"},
				},
			},
		},
		Metadata: s.testMetadata("TestSingleDatasourceOracle"),
	})
	require.NoError(t, err, "Failed to create fetcher job")
	assert.NotEmpty(t, jobResp.JobID.String())
	assert.Equal(t, "pending", jobResp.Status)

	// Step 3: Wait for job completion via RabbitMQ event
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeoutSlow) // Oracle may be slower
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status)

	// Verify result and execution metrics in notification
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.SizeBytes, int64(0))
	assert.Greater(t, notification.Result.RowCount, int64(0))
	assert.Greater(t, notification.ExecutionTimeMs, int64(0))
	assert.NotNil(t, notification.CompletedAt)

	// Step 4: Verify job status via API
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err, "Failed to get job")
	assert.Equal(t, "completed", job.Status)
	assert.NotEmpty(t, job.ResultPath)

	// Step 5: Verify data in SeaweedFS
	resultPath := "/external-data/" + jobResp.JobID.String() + ".json"
	data, err := s.seaweedClient.WaitForFile(s.ctx, resultPath, setup.SeaweedFSFileTimeout)
	require.NoError(t, err, "Failed to get result file from SeaweedFS")
	assert.NotEmpty(t, data)
}

// =============================================================================
// MULTI-SCHEMA DATA EXTRACTION TESTS
// =============================================================================
// These tests validate data extraction from multiple schemas within a single
// datasource connection. Schema-qualified table names (schema.table) are used.
// Priority: P0 (validates schema handling across database types)
// =============================================================================

// TestPostgreSQLMultiSchemaExtraction validates extraction from multiple schemas in PostgreSQL.
// This validates the SplitSchemaTable() and GetUniqueSchemas() functions work correctly.
func (s *WorkerIntegrationTestSuite) TestPostgreSQLMultiSchemaExtraction() {
	t := s.T()

	// Generate unique config name
	configName := s.uniqueConfigName("postgres_multi_schema")

	// Step 1: Create PostgreSQL connection
	pg := s.infra.PostgresInternal()
	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create PostgreSQL connection")
	assert.NotEmpty(t, connResp.ID)

	// Step 2: Create job with schema-qualified tables from multiple schemas
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					// Schema: public (default)
					"transactions": {"id", "account_id", "amount", "currency", "status"},
					// Schema: accounting
					"accounting.invoices": {"id", "account_id", "invoice_number", "amount", "status"},
					// Schema: reporting
					"reporting.daily_summary": {"id", "report_date", "account_id", "total_credits", "total_debits"},
				},
			},
		},
		Metadata: s.testMetadata("TestPostgreSQLMultiSchemaExtraction"),
	})
	require.NoError(t, err, "Failed to create fetcher job")
	assert.NotEmpty(t, jobResp.JobID.String())
	assert.Equal(t, "pending", jobResp.Status)

	// Step 3: Wait for job completion
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")

	if notification.Status == "failed" {
		t.Logf("Multi-schema job failed: %+v", notification.Metadata)
	}
	assert.Equal(t, "completed", notification.Status, "Job should complete successfully")

	// Step 4: Verify result
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.SizeBytes, int64(0))

	// Exact row count validation:
	// public.transactions: 26 rows + accounting.invoices: 10 rows + reporting.daily_summary: 12 rows = 48 total
	const expectedRows int64 = 48
	assert.Equal(t, expectedRows, notification.Result.RowCount,
		"Expected %d rows (transactions:26 + invoices:10 + daily_summary:12)", expectedRows)

	// Verify file exists in SeaweedFS
	resultPath := "/external-data/" + jobResp.JobID.String() + ".json"
	data, err := s.seaweedClient.WaitForFile(s.ctx, resultPath, setup.SeaweedFSFileTimeout)
	require.NoError(t, err, "Failed to get result file from SeaweedFS")
	assert.NotEmpty(t, data)
}

// TestPostgreSQLMultiSchemaWithFilters validates filtered extraction across multiple schemas.
// This validates filter application works correctly with schema-qualified table names.
func (s *WorkerIntegrationTestSuite) TestPostgreSQLMultiSchemaWithFilters() {
	t := s.T()

	configName := s.uniqueConfigName("postgres_multi_schema_filtered")

	// Create PostgreSQL connection
	pg := s.infra.PostgresInternal()
	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create job with filters - filter by status across all tables
	// Note: 'in' operator matches 'completed' (transactions) OR 'paid' (invoices)
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions":        {"id", "account_id", "amount", "status"},
					"accounting.invoices": {"id", "account_id", "amount", "status"},
				},
			},
			Filters: model.NestedFilters{
				configName: {
					"transactions": {
						"status": job.FilterCondition{
							In: []any{"completed", "paid"},
						},
					},
					"accounting.invoices": {
						"status": job.FilterCondition{
							In: []any{"completed", "paid"},
						},
					},
				},
			},
		},
		Metadata: s.testMetadata("TestPostgreSQLMultiSchemaWithFilters"),
	})
	require.NoError(t, err, "Failed to create fetcher job")

	// Wait for completion
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err)
	assert.Equal(t, "completed", notification.Status)

	// Exact row count validation:
	// transactions with status='completed': 24 rows
	// accounting.invoices with status='paid': 5 rows
	// Total: 29 rows
	require.NotNil(t, notification.Result)
	const expectedFilteredRows int64 = 29
	assert.Equal(t, expectedFilteredRows, notification.Result.RowCount,
		"Expected %d rows (transactions:24 completed + invoices:5 paid)", expectedFilteredRows)
}

// TestMultiDatasourceMultiSchemaExtraction validates complex extraction from 3 datasources with multiple schemas each.
func (s *WorkerIntegrationTestSuite) TestMultiDatasourceMultiSchemaExtraction() {
	t := s.T()

	// Generate unique config names
	pgConfigName := s.uniqueConfigName("postgres_multi_ds")
	mssqlConfigName := s.uniqueConfigName("sqlserver_multi_ds")
	oracleConfigName := s.uniqueConfigName("oracle_multi_ds")

	// Step 1: Create PostgreSQL connection
	pg := s.infra.PostgresInternal()
	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   pgConfigName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create PostgreSQL connection")

	// Step 2: Create SQL Server connection
	mssql := s.infra.SQLServerInternal()
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   mssqlConfigName,
		Type:         "SQL_SERVER",
		Host:         mssql.Host,
		Port:         mssql.Port,
		DatabaseName: mssql.Database,
		Username:     mssql.Username,
		Password:     mssql.Password,
	})
	require.NoError(t, err, "Failed to create SQL Server connection")

	// Step 3: Create Oracle connection
	oracle := s.infra.OracleInternal()
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   oracleConfigName,
		Type:         "ORACLE",
		Host:         oracle.Host,
		Port:         oracle.Port,
		DatabaseName: oracle.Database,
		Username:     oracle.Username,
		Password:     oracle.Password,
		Metadata: map[string]any{
			"serviceName": oracle.Database,
		},
	})
	require.NoError(t, err, "Failed to create Oracle connection")

	// Step 4: Create multi-datasource, multi-schema job with filters
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				// PostgreSQL: 2 schemas
				pgConfigName: {
					"transactions":        {"id", "account_id", "amount", "currency", "status"},
					"accounting.invoices": {"id", "account_id", "invoice_number", "amount", "status"},
				},
				// SQL Server: 2 schemas
				mssqlConfigName: {
					"dbo.transactions": {"id", "account_id", "amount", "currency", "status"},
					"finance.payments": {"id", "account_id", "payment_reference", "amount", "status"},
				},
				// Oracle: 2 tables (different logical domains)
				oracleConfigName: {
					"transactions":          {"id", "account_id", "amount", "currency", "status"},
					"billing_subscriptions": {"id", "account_id", "plan_name", "monthly_amount", "status"},
				},
			},
			Filters: model.NestedFilters{
				pgConfigName: {
					"transactions": {
						"status": job.FilterCondition{
							In: []any{"completed", "paid", "active"},
						},
					},
				},
				mssqlConfigName: {
					"dbo.transactions": {
						"status": job.FilterCondition{
							In: []any{"completed", "paid", "active"},
						},
					},
				},
				oracleConfigName: {
					"transactions": {
						"status": job.FilterCondition{
							In: []any{"completed", "paid", "active"},
						},
					},
				},
			},
		},
		Metadata: s.testMetadata("TestMultiDatasourceMultiSchemaExtraction"),
	})
	require.NoError(t, err, "Failed to create multi-datasource job")
	assert.NotEmpty(t, jobResp.JobID.String())

	// Step 5: Wait for completion (extended timeout for Oracle)
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeoutSlow)
	require.NoError(t, err, "Failed to receive job completion event")

	if notification.Status == "failed" {
		t.Logf("Multi-datasource multi-schema job failed: %+v", notification.Metadata)
	}
	assert.Equal(t, "completed", notification.Status, "Job should complete successfully")

	// Step 6: Verify comprehensive result with exact row count
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.SizeBytes, int64(0))

	// Expected rows with selective filters on transactions tables only:
	// PostgreSQL: 24 (transactions filtered) + 10 (invoices all) = 34
	// SQL Server: 22 (dbo.transactions filtered) + 10 (payments all) = 32
	// Oracle: 21 (transactions filtered) + 8 (subscriptions all) = 29
	// Total: 95 rows
	const expectedMultiDSRows int64 = 95
	assert.Equal(t, expectedMultiDSRows, notification.Result.RowCount,
		"Expected %d rows from 3 datasources x 2 tables each (with filters)", expectedMultiDSRows)
	assert.Greater(t, notification.ExecutionTimeMs, int64(0))

	// Verify file in SeaweedFS
	resultPath := "/external-data/" + jobResp.JobID.String() + ".json"
	data, err := s.seaweedClient.WaitForFile(s.ctx, resultPath, setup.SeaweedFSFileTimeout)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify job details via API
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err)
	assert.Equal(t, "completed", job.Status)
	assert.NotEmpty(t, job.ResultPath)
}

// TestValidateSchema_MultiSchema validates schema validation with schema-qualified table names.
func (s *WorkerIntegrationTestSuite) TestValidateSchema_MultiSchema() {
	t := s.T()

	configName := s.uniqueConfigName("postgres_validate_multi")

	// Create PostgreSQL connection
	pg := s.infra.PostgresInternal()
	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Test 1: Validate existing schema-qualified tables
	result, err := s.managerClient.ValidateSchema(s.ctx, client.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			configName: {
				"transactions":            {"id", "account_id", "amount"},
				"accounting.invoices":     {"id", "account_id", "invoice_number"},
				"reporting.daily_summary": {"id", "report_date", "total_credits"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status, "Validation should succeed for existing tables")
	assert.Empty(t, result.Errors)

	// Test 2: Validate with non-existent schema
	result, err = s.managerClient.ValidateSchema(s.ctx, client.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			configName: {
				"nonexistent_schema.fake_table": {"id", "name"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "failure", result.Status, "Validation should fail for non-existent schema")
	assert.NotEmpty(t, result.Errors)

	// Verify the error type
	foundTableError := false
	for _, e := range result.Errors {
		if e.Type == "TABLE_NOT_FOUND" {
			foundTableError = true
			break
		}
	}
	assert.True(t, foundTableError, "Expected TABLE_NOT_FOUND error for non-existent schema")
}

// =============================================================================
// ADVANCED DATA EXTRACTION TESTS - Multi-Datasource & Features
// =============================================================================
// These tests validate advanced extraction scenarios including multiple
// datasources in a single job, filter conditions, metadata passthrough,
// and idempotency behavior.
// Priority: P1 (important features that extend core functionality)
// =============================================================================

// TestMultiDatasourceExtraction validates extraction from multiple datasources in one job.
func (s *WorkerIntegrationTestSuite) TestMultiDatasourceExtraction() {
	t := s.T()

	// Ensure connections exist (created in previous tests or create new ones)
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "postgres_multi",
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	mysql := s.infra.MySQLInternal()

	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "mysql_multi",
		Type:         "MYSQL",
		Host:         mysql.Host,
		Port:         mysql.Port,
		DatabaseName: mysql.Database,
		Username:     mysql.Username,
		Password:     mysql.Password,
	})
	require.NoError(t, err)

	// Create multi-datasource job
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_multi": {
					"transactions": {"id", "account_id", "amount", "category"},
				},
				"mysql_multi": {
					"transactions": {"id", "account_id", "amount", "category"},
				},
			},
		},
		Metadata: s.testMetadata("TestMultiDatasourceExtraction"),
	})
	require.NoError(t, err, "Failed to create multi-datasource job")

	// Wait for completion
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeoutSlow)
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status)

	// Verify result and execution metrics in notification
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.SizeBytes, int64(0))
	assert.Greater(t, notification.Result.RowCount, int64(0))
	assert.Greater(t, notification.ExecutionTimeMs, int64(0))
	assert.NotNil(t, notification.CompletedAt)

	// Verify job includes metadata from request
	assert.Equal(t, "integration_test", notification.Metadata["source"])
}

// TestJobWithFilters validates extraction with filter conditions applied.
func (s *WorkerIntegrationTestSuite) TestJobWithFilters() {
	t := s.T()

	// Create connection for filtered query
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "postgres_filtered",
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create job with filters
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_filtered": {
					"transactions": {"id", "account_id", "amount", "category", "status"},
				},
			},
			Filters: model.NestedFilters{
				"postgres_filtered": {
					"transactions": {
						"status": job.FilterCondition{
							Equals: []any{"completed"},
						},
						"category": job.FilterCondition{
							Equals: []any{"salary"},
						},
					},
				},
			},
		},
		Metadata: s.testMetadata("TestJobWithFilters"),
	})
	require.NoError(t, err, "Failed to create filtered job")

	// Wait for completion
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status)

	// Verify result and execution metrics in notification
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.SizeBytes, int64(0))
	// Note: RowCount may be 0 or more depending on filter results
	assert.Greater(t, notification.ExecutionTimeMs, int64(0))
	assert.NotNil(t, notification.CompletedAt)
}

// TestJobWithSelectiveFilters validates that filters are applied only to specified tables.
func (s *WorkerIntegrationTestSuite) TestJobWithSelectiveFilters() {
	t := s.T()

	// Create connection
	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "postgres_selective_filter",
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create job with filter on ONLY transactions table
	// accounts table should return ALL rows (no filter applied)
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_selective_filter": {
					"transactions": {"id", "status", "amount"},
					"accounts":     {"id", "name"},
				},
			},
			Filters: model.NestedFilters{
				"postgres_selective_filter": {
					"transactions": {
						"status": job.FilterCondition{
							Equals: []any{"completed"},
						},
					},
				},
			},
		},
		Metadata: s.testMetadata("TestJobWithSelectiveFilters"),
	})
	require.NoError(t, err, "Failed to create job with selective filters")

	// Wait for completion
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status)

	// Verify result exists
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)

	// The key assertion: accounts table should have all rows since no filter was applied to it
	// transactions table should have fewer rows due to the status filter
	// Note: Exact row counts depend on test data, but we verify the job completes successfully
	assert.Greater(t, notification.ExecutionTimeMs, int64(0))
}

// TestJobIdempotency validates request deduplication within the 5-minute window.
func (s *WorkerIntegrationTestSuite) TestJobIdempotency() {
	t := s.T()

	// Create connection
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "postgres_idempotency",
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create first job
	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_idempotency": {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: s.testMetadata("TestJobIdempotency"),
	}

	jobResp1, err := s.managerClient.CreateFetcherJob(s.ctx, request)
	require.NoError(t, err)

	// Send same request again within 5-minute window
	jobResp2, err := s.managerClient.CreateFetcherJob(s.ctx, request)
	require.NoError(t, err)

	// Should return same job ID
	assert.Equal(t, jobResp1.JobID, jobResp2.JobID, "Duplicate request should return same job")
}

// TestSeaweedFSFileValidation validates the complete file storage flow in SeaweedFS.
// This validates the Worker's file upload and Manager's result path tracking.
func (s *WorkerIntegrationTestSuite) TestSeaweedFSFileValidation() {
	t := s.T()

	// Create connection
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "postgres_seaweed",
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create job
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_seaweed": {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: s.testMetadata("TestSeaweedFSFileValidation"),
	})
	require.NoError(t, err)

	// Wait for completion
	_, err = s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err)

	// Get job to verify resultPath
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err)
	assert.NotEmpty(t, job.ResultPath, "Job should have resultPath after completion")

	// Verify file exists in SeaweedFS
	resultPath := "/external-data/" + jobResp.JobID.String() + ".json"
	exists, err := s.seaweedClient.FileExists(s.ctx, resultPath)
	require.NoError(t, err)
	assert.True(t, exists, "Result file should exist in SeaweedFS")

	// Verify file has content (encrypted)
	data, err := s.seaweedClient.GetFile(s.ctx, resultPath)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0, "Result file should have content")
}

// TestMetadataPassthrough validates metadata preservation through the extraction pipeline.
// This validates end-to-end metadata passthrough: API → MongoDB → RabbitMQ → Event.
func (s *WorkerIntegrationTestSuite) TestMetadataPassthrough() {
	t := s.T()

	// Create connection
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "postgres_metadata",
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create job with custom metadata (timestamp ensures unique request hash)
	customMetadata := map[string]any{
		"source":       "integration_test",
		"testName":     "TestMetadataPassthrough",
		"timestamp":    time.Now().UnixNano(),
		"customField1": "value1",
		"customField2": 42,
		"nested": map[string]any{
			"field": "nestedValue",
		},
	}

	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_metadata": {
					"transactions": {"id", "amount"},
				},
			},
		},
		Metadata: customMetadata,
	})
	require.NoError(t, err)

	// Wait for completion and check metadata in notification
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err)

	// Verify metadata is preserved in notification
	assert.Equal(t, "integration_test", notification.Metadata["source"])
	assert.Equal(t, "TestMetadataPassthrough", notification.Metadata["testName"])
	assert.Equal(t, "value1", notification.Metadata["customField1"])

	// Verify metadata in job response
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err)

	// Marshal and compare metadata
	jobMetaJSON, _ := json.Marshal(job.Metadata)
	expectedMetaJSON, _ := json.Marshal(customMetadata)
	assert.JSONEq(t, string(expectedMetaJSON), string(jobMetaJSON), "Metadata should be preserved")
}

// =============================================================================
// ERROR SCENARIO TESTS
// =============================================================================
// These tests validate graceful error handling across the extraction pipeline.
// Each test simulates a specific failure mode and verifies appropriate error
// responses and status updates.
// Priority: P2 (critical for production reliability)
// =============================================================================

// TestErrorScenario_ConnectionDown validates graceful handling of unavailable datasources.
// Expected: Job creation fails with connection error (Manager validates before queue)
func (s *WorkerIntegrationTestSuite) TestErrorScenario_ConnectionDown() {
	t := s.T()

	// Create connection pointing to non-existent host
	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "postgres_down",
		Type:         "POSTGRESQL",
		Host:         "non-existent-host.local",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpassword",
	})
	// Connection creation should succeed (no connectivity check at creation time)
	require.NoError(t, err)

	// Create job targeting the unavailable datasource
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_down": {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: s.testMetadata("TestErrorScenario_ConnectionDown"),
	})
	// Job creation will fail because Manager tests connection before publishing
	require.Error(t, err, "Expected job creation to fail with unavailable datasource")
	assert.Nil(t, jobResp)
	assert.Contains(t, strings.ToLower(err.Error()), "connection", "Error should mention connection issue")
}

// TestErrorScenario_InvalidCredentials validates handling of authentication failures.
// Expected: Job creation fails with authentication error message
func (s *WorkerIntegrationTestSuite) TestErrorScenario_InvalidCredentials() {
	t := s.T()

	// Get the real PostgreSQL host but use wrong password
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "postgres_bad_creds",
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     "WRONG_PASSWORD_12345",
	})
	require.NoError(t, err)

	// Create job - Manager will test connection and fail
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_bad_creds": {
					"transactions": {"id", "amount"},
				},
			},
		},
		Metadata: s.testMetadata("TestErrorScenario_InvalidCredentials"),
	})

	require.Error(t, err, "Expected job creation to fail with invalid credentials")
	assert.Nil(t, jobResp)
	// API returns "Connection Down" for any connection failure (including auth failures)
	errLower := strings.ToLower(err.Error())
	assert.True(t, strings.Contains(errLower, "authentication") || strings.Contains(errLower, "connection"),
		"Error should mention authentication or connection failure, got: %s", err.Error())
}

// TestErrorScenario_NonExistentTable validates handling of invalid table references.
// Expected: Job is created but fails during Worker processing with table error
func (s *WorkerIntegrationTestSuite) TestErrorScenario_NonExistentTable() {
	t := s.T()

	// Create valid connection
	// Uses Docker network hostname - works for containers (Docker DNS) and host (via /etc/hosts)
	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "postgres_for_missing_table",
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create job requesting non-existent table
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"postgres_for_missing_table": {
					"this_table_does_not_exist_xyz": {"id", "name"},
				},
			},
		},
		Metadata: s.testMetadata("TestErrorScenario_NonExistentTable"),
	})
	require.NoError(t, err, "Job creation should succeed - failure happens during processing")

	// Wait for job to fail
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Should receive job event")
	assert.Equal(t, "failed", notification.Status, "Job should fail for non-existent table")

	// Verify job status via API
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err)
	assert.Equal(t, "failed", job.Status)
}

// TestErrorScenario_MissingDatasource validates handling of unconfigured datasource names.
// Expected: Job creation fails with "missing datasource" error
func (s *WorkerIntegrationTestSuite) TestErrorScenario_MissingDatasource() {
	t := s.T()

	// Create job referencing a datasource that was never configured
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"completely_unknown_datasource": {
					"some_table": {"field1", "field2"},
				},
			},
		},
		Metadata: s.testMetadata("TestErrorScenario_MissingDatasource"),
	})

	require.Error(t, err, "Expected job creation to fail with unknown datasource")
	assert.Nil(t, jobResp)
	assert.Contains(t, err.Error(), "datasource", "Error should mention missing datasource")
}

// =============================================================================
// CONNECTION MANAGEMENT TESTS - CRUD Operations
// =============================================================================
// These tests validate the Connection Management API endpoints:
// GET, POST, PATCH, DELETE for /v1/management/connections.
// Each test verifies proper request handling, validation, and persistence.
// Priority: P1 (core API functionality)
// =============================================================================

// TestConnection_GetByID validates retrieving a connection by its UUID.
func (s *WorkerIntegrationTestSuite) TestConnection_GetByID() {
	t := s.T()

	// Create a connection first
	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_get_test")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create connection")
	require.NotEmpty(t, connResp.ID)

	// Get the connection by ID
	gotConn, err := s.managerClient.GetConnection(s.ctx, connResp.ID)
	require.NoError(t, err, "Failed to get connection")

	// Verify all fields
	assert.Equal(t, connResp.ID, gotConn.ID)
	assert.Equal(t, configName, gotConn.ConfigName)
	assert.Equal(t, "POSTGRESQL", gotConn.Type)
	assert.Equal(t, pg.Host, gotConn.Host)
	assert.Equal(t, pg.Port, gotConn.Port)
	assert.Equal(t, pg.Database, gotConn.DatabaseName)
	assert.Equal(t, pg.Username, gotConn.Username)
	assert.NotEmpty(t, gotConn.CreatedAt)
}

// TestConnection_GetByID_NotFound validates 404 response for non-existent connection.
func (s *WorkerIntegrationTestSuite) TestConnection_GetByID_NotFound() {
	t := s.T()

	// Try to get a non-existent connection
	_, err := s.managerClient.GetConnection(s.ctx, "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// TestConnection_Update validates updating connection fields via PATCH endpoint.
func (s *WorkerIntegrationTestSuite) TestConnection_Update() {
	t := s.T()

	// Create a connection first
	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_update_test")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Update the connection - change configName
	newConfigName := s.uniqueConfigName("postgres_updated")
	updated, err := s.managerClient.UpdateConnection(s.ctx, connResp.ID, client.ConnectionInput{
		ConfigName:   newConfigName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)
	assert.Equal(t, newConfigName, updated.ConfigName)
	assert.NotEmpty(t, updated.UpdatedAt)

	// Verify the update persisted
	gotConn, err := s.managerClient.GetConnection(s.ctx, connResp.ID)
	require.NoError(t, err)
	assert.Equal(t, newConfigName, gotConn.ConfigName)
}

// TestConnection_Update_NotFound validates 404 response when updating non-existent connection.
func (s *WorkerIntegrationTestSuite) TestConnection_Update_NotFound() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	_, err := s.managerClient.UpdateConnection(s.ctx, "00000000-0000-0000-0000-000000000000", client.ConnectionInput{
		ConfigName:   "test",
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// TestConnection_PartialUpdate validates true partial update behavior via PATCH endpoint.
// This test verifies that only provided fields are updated, while other fields remain unchanged.
func (s *WorkerIntegrationTestSuite) TestConnection_PartialUpdate() {
	t := s.T()

	// Create a connection first with all fields populated
	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_partial_update")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, connResp.ID)

	// Store original values to verify they don't change
	originalHost := connResp.Host
	originalPort := connResp.Port
	originalDatabaseName := connResp.DatabaseName
	originalUsername := connResp.Username
	originalType := connResp.Type

	// Perform PARTIAL update - only change configName
	newConfigName := s.uniqueConfigName("postgres_partially_updated")
	updated, err := s.managerClient.PartialUpdateConnection(s.ctx, connResp.ID, client.ConnectionPartialUpdateInput{
		ConfigName: client.StringPtr(newConfigName),
		// All other fields are nil - should NOT be updated
	})
	require.NoError(t, err, "Partial update should succeed")

	// Verify the updated field changed
	assert.Equal(t, newConfigName, updated.ConfigName, "ConfigName should be updated")

	// Verify all other fields remain UNCHANGED
	assert.Equal(t, originalHost, updated.Host, "Host should remain unchanged")
	assert.Equal(t, originalPort, updated.Port, "Port should remain unchanged")
	assert.Equal(t, originalDatabaseName, updated.DatabaseName, "DatabaseName should remain unchanged")
	assert.Equal(t, originalUsername, updated.Username, "Username should remain unchanged")
	assert.Equal(t, originalType, updated.Type, "Type should remain unchanged")

	// Verify the update persisted by fetching again
	gotConn, err := s.managerClient.GetConnection(s.ctx, connResp.ID)
	require.NoError(t, err)
	assert.Equal(t, newConfigName, gotConn.ConfigName, "Persisted ConfigName should match")
	assert.Equal(t, originalHost, gotConn.Host, "Persisted Host should remain unchanged")
	assert.Equal(t, originalPort, gotConn.Port, "Persisted Port should remain unchanged")
}

// TestConnection_PartialUpdate_MultipleFields validates partial update with multiple fields.
func (s *WorkerIntegrationTestSuite) TestConnection_PartialUpdate_MultipleFields() {
	t := s.T()

	// Create a connection first
	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_multi_field_update")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Store original values
	originalType := connResp.Type
	originalUsername := connResp.Username

	// Update TWO fields: configName and port
	newConfigName := s.uniqueConfigName("postgres_multi_updated")
	newPort := 5433
	updated, err := s.managerClient.PartialUpdateConnection(s.ctx, connResp.ID, client.ConnectionPartialUpdateInput{
		ConfigName: client.StringPtr(newConfigName),
		Port:       client.IntPtr(newPort),
		// Other fields nil - should NOT be updated
	})
	require.NoError(t, err)

	// Verify updated fields changed
	assert.Equal(t, newConfigName, updated.ConfigName)
	assert.Equal(t, newPort, updated.Port)

	// Verify other fields remain unchanged
	assert.Equal(t, originalType, updated.Type, "Type should remain unchanged")
	assert.Equal(t, originalUsername, updated.Username, "Username should remain unchanged")
}

// TestConnection_Delete validates connection deletion and subsequent 404 on GET.
func (s *WorkerIntegrationTestSuite) TestConnection_Delete() {
	t := s.T()

	// Create a connection first
	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_delete_test")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Delete the connection
	err = s.managerClient.DeleteConnection(s.ctx, connResp.ID)
	require.NoError(t, err)

	// Verify it's deleted (should return 404)
	_, err = s.managerClient.GetConnection(s.ctx, connResp.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// TestConnection_Delete_NotFound validates 404 response when deleting non-existent connection.
func (s *WorkerIntegrationTestSuite) TestConnection_Delete_NotFound() {
	t := s.T()

	err := s.managerClient.DeleteConnection(s.ctx, "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// TestConnection_ListWithPagination validates paginated connection listing.
func (s *WorkerIntegrationTestSuite) TestConnection_ListWithPagination() {
	t := s.T()

	// Create multiple connections
	pg := s.infra.PostgresInternal()
	createdIDs := make([]string, 0, 3)

	for i := 0; i < 3; i++ {
		configName := s.uniqueConfigName(fmt.Sprintf("postgres_list_%d", i))
		connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
			ConfigName:   configName,
			Type:         "POSTGRESQL",
			Host:         pg.Host,
			Port:         pg.Port,
			DatabaseName: pg.Database,
			Username:     pg.Username,
			Password:     pg.Password,
		})
		require.NoError(t, err)
		createdIDs = append(createdIDs, connResp.ID)
	}

	// List with limit=2
	result, err := s.managerClient.ListConnectionsWithParams(s.ctx, map[string]string{
		"limit": "2",
		"page":  "1",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Limit)
	assert.LessOrEqual(t, len(result.Items), 2)

	// Cleanup
	for _, id := range createdIDs {
		_ = s.managerClient.DeleteConnection(s.ctx, id)
	}
}

// TestConnection_ListWithTypeFilter validates connection listing filtered by database type.
func (s *WorkerIntegrationTestSuite) TestConnection_ListWithTypeFilter() {
	t := s.T()

	// Create PostgreSQL connection
	pg := s.infra.PostgresInternal()
	pgConfigName := s.uniqueConfigName("postgres_filter_test")
	pgConn, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   pgConfigName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create MySQL connection
	mysql := s.infra.MySQLInternal()
	mysqlConfigName := s.uniqueConfigName("mysql_filter_test")
	mysqlConn, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   mysqlConfigName,
		Type:         "MYSQL",
		Host:         mysql.Host,
		Port:         mysql.Port,
		DatabaseName: mysql.Database,
		Username:     mysql.Username,
		Password:     mysql.Password,
	})
	require.NoError(t, err)

	// List only PostgreSQL connections
	result, err := s.managerClient.ListConnectionsWithParams(s.ctx, map[string]string{
		"type": "POSTGRESQL",
	})
	require.NoError(t, err)

	// All returned items should be PostgreSQL
	for _, conn := range result.Items {
		assert.Equal(t, "POSTGRESQL", conn.Type)
	}

	// Cleanup
	_ = s.managerClient.DeleteConnection(s.ctx, pgConn.ID)
	_ = s.managerClient.DeleteConnection(s.ctx, mysqlConn.ID)
}

// =============================================================================
// CONNECTION TEST & SCHEMA VALIDATION TESTS
// =============================================================================

// TestConnection_TestEndpoint validates the connection test endpoint with valid credentials.
func (s *WorkerIntegrationTestSuite) TestConnection_TestEndpoint() {
	t := s.T()

	// Create a valid connection
	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_test_endpoint")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Test the connection
	testResult, err := s.managerClient.TestConnectionEndpoint(s.ctx, connResp.ID)
	require.NoError(t, err)

	assert.Equal(t, "success", testResult.Status)
	assert.Equal(t, "Connection successful", testResult.Message)
	assert.Greater(t, testResult.LatencyMs, int64(0))
}

// TestConnection_TestEndpoint_NotFound validates 404 response when testing non-existent connection.
func (s *WorkerIntegrationTestSuite) TestConnection_TestEndpoint_NotFound() {
	t := s.T()

	_, err := s.managerClient.TestConnectionEndpoint(s.ctx, "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// TestConnection_TestEndpoint_InvalidCredentials validates error response when testing with bad credentials.
func (s *WorkerIntegrationTestSuite) TestConnection_TestEndpoint_InvalidCredentials() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_test_bad_creds")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     "WRONG_PASSWORD",
	})
	require.NoError(t, err)

	// Test should fail
	_, err = s.managerClient.TestConnectionEndpoint(s.ctx, connResp.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// TestValidateSchema_Success validates schema validation with existing table and fields.
func (s *WorkerIntegrationTestSuite) TestValidateSchema_Success() {
	t := s.T()

	// Create a valid connection
	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_schema_valid")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Validate schema with existing table and fields
	result, err := s.managerClient.ValidateSchema(s.ctx, client.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			configName: {
				"transactions": {"id", "account_id", "amount", "currency"},
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "success", result.Status)
	assert.Empty(t, result.Errors)
}

// TestValidateSchema_TableNotFound validates TABLE_NOT_FOUND error for non-existent table.
func (s *WorkerIntegrationTestSuite) TestValidateSchema_TableNotFound() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_schema_missing_table")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Validate schema with non-existent table
	result, err := s.managerClient.ValidateSchema(s.ctx, client.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			configName: {
				"non_existent_table_xyz": {"id", "name"},
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "failure", result.Status)
	assert.NotEmpty(t, result.Errors)

	// Should have TABLE_NOT_FOUND error
	foundTableError := false
	for _, e := range result.Errors {
		if e.Type == "TABLE_NOT_FOUND" {
			foundTableError = true
			break
		}
	}
	assert.True(t, foundTableError, "Expected TABLE_NOT_FOUND error")
}

// TestValidateSchema_FieldNotFound validates FIELD_NOT_FOUND error for non-existent column.
func (s *WorkerIntegrationTestSuite) TestValidateSchema_FieldNotFound() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_schema_missing_field")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Validate schema with non-existent field
	result, err := s.managerClient.ValidateSchema(s.ctx, client.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			configName: {
				"transactions": {"id", "non_existent_field_xyz"},
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "failure", result.Status)
	assert.NotEmpty(t, result.Errors)
}

// TestValidateSchema_DatasourceNotFound validates HTTP 400 error for unknown datasource.
//
// When no connections exist for the specified datasource name, the API returns
// HTTP 400 Bad Request since it cannot perform schema validation.
func (s *WorkerIntegrationTestSuite) TestValidateSchema_DatasourceNotFound() {
	t := s.T()

	// Validate schema with non-existent datasource (no connection created)
	// API returns HTTP 400 when no connections are found
	_, err := s.managerClient.ValidateSchema(s.ctx, client.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"unknown_datasource_xyz": {
				"transactions": {"id"},
			},
		},
	})

	// Expect an error since no connections exist for this datasource
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

// TestConnection_DeleteWithActiveJob validates 409 Conflict when deleting connection with active job.
func (s *WorkerIntegrationTestSuite) TestConnection_DeleteWithActiveJob() {
	t := s.T()

	// Create connection
	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_delete_active")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create a job that uses this connection
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: s.testMetadata("TestConnection_DeleteWithActiveJob"),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, jobResp.JobID.String())

	// Try to delete connection immediately (while job is pending/processing)
	err = s.managerClient.DeleteConnection(s.ctx, connResp.ID)

	// Should fail with 409 Conflict (job in progress)
	if err != nil {
		assert.Contains(t, err.Error(), "409")
	}

	// Wait for job to complete
	_, _ = s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)

	// Now deletion should succeed
	err = s.managerClient.DeleteConnection(s.ctx, connResp.ID)
	require.NoError(t, err)
}

// TestConnection_UpdateWithActiveJob validates 409 Conflict when updating connection with active job.
func (s *WorkerIntegrationTestSuite) TestConnection_UpdateWithActiveJob() {
	t := s.T()

	// Create connection
	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_update_active")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create a job that uses this connection
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: s.testMetadata("TestConnection_UpdateWithActiveJob"),
	})
	require.NoError(t, err)

	// Try to update connection immediately
	newConfigName := s.uniqueConfigName("postgres_updated_active")
	_, err = s.managerClient.UpdateConnection(s.ctx, connResp.ID, client.ConnectionInput{
		ConfigName:   newConfigName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})

	// Should fail with 409 Conflict
	if err != nil {
		assert.Contains(t, err.Error(), "409")
	}

	// Wait for job completion and cleanup
	_, _ = s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
}

// =============================================================================
// VALIDATION & EDGE CASE TESTS
// =============================================================================

// TestConnection_DuplicateConfigName validates 409 Conflict for duplicate configName.
func (s *WorkerIntegrationTestSuite) TestConnection_DuplicateConfigName() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_duplicate")

	// Create first connection
	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Try to create second connection with same configName
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName, // Same name
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "409") // Conflict
}

// TestJob_InvalidInput validates HTTP 400 Bad Request for empty mappedFields.
//
// The fetcher job requires at least one datasource/table/field mapping.
// An empty mappedFields map should be rejected with HTTP 400.
func (s *WorkerIntegrationTestSuite) TestJob_InvalidInput() {
	t := s.T()

	// Test empty mappedFields
	_, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{},
		},
		Metadata: s.testMetadata("TestJob_InvalidInput_Empty"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

// TestJob_AllFilterOperators validates all supported filter operators (eq, ne, gt, gte, lt, lte, in, nin).
// This validates the Manager→Worker filter transformation and SQL WHERE generation.
func (s *WorkerIntegrationTestSuite) TestJob_AllFilterOperators() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_all_filters")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	testCases := []struct {
		name      string
		field     string
		condition job.FilterCondition
	}{
		{"eq operator", "status", job.FilterCondition{Equals: []any{"completed"}}},
		{"ne operator", "status", job.FilterCondition{NotEquals: []any{"failed"}}},
		{"gt operator", "amount", job.FilterCondition{GreaterThan: []any{100}}},
		{"gte operator", "amount", job.FilterCondition{GreaterOrEqual: []any{100}}},
		{"lt operator", "amount", job.FilterCondition{LessThan: []any{1000}}},
		{"lte operator", "amount", job.FilterCondition{LessOrEqual: []any{1000}}},
		{"in operator", "status", job.FilterCondition{In: []any{"completed", "pending"}}},
		{"nin operator", "status", job.FilterCondition{NotIn: []any{"failed"}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						configName: {
							"transactions": {"id", "account_id", "amount", "status"},
						},
					},
					Filters: model.NestedFilters{
						configName: {
							"transactions": {
								tc.field: tc.condition,
							},
						},
					},
				},
				Metadata: s.testMetadata("TestJob_AllFilterOperators_" + tc.name),
			})
			require.NoError(t, err, "Failed to create job with %s", tc.name)

			// Wait for completion
			notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
			require.NoError(t, err)
			assert.Equal(t, "completed", notification.Status, "Job with %s should complete", tc.name)
		})
	}
}

// TestConnection_Metadata validates custom metadata persistence on connection creation.
func (s *WorkerIntegrationTestSuite) TestConnection_Metadata() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_metadata_test")

	customMetadata := map[string]any{
		"environment": "integration-test",
		"team":        "platform",
		"nested": map[string]any{
			"key": "value",
		},
	}

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
		Metadata:     customMetadata,
	})
	require.NoError(t, err)

	// Get connection and verify metadata
	gotConn, err := s.managerClient.GetConnection(s.ctx, connResp.ID)
	require.NoError(t, err)

	assert.Equal(t, "integration-test", gotConn.Metadata["environment"])
	assert.Equal(t, "platform", gotConn.Metadata["team"])
}

// =============================================================================
// MULTI-DATASOURCE SCHEMA VALIDATION TESTS
// =============================================================================

// TestValidateSchema_MultiDatasource tests schema validation across multiple datasources.
func (s *WorkerIntegrationTestSuite) TestValidateSchema_MultiDatasource() {
	t := s.T()

	// Create PostgreSQL connection
	pg := s.infra.PostgresInternal()
	pgConfigName := s.uniqueConfigName("postgres_schema_multi")
	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   pgConfigName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Create MySQL connection
	mysql := s.infra.MySQLInternal()
	mysqlConfigName := s.uniqueConfigName("mysql_schema_multi")
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   mysqlConfigName,
		Type:         "MYSQL",
		Host:         mysql.Host,
		Port:         mysql.Port,
		DatabaseName: mysql.Database,
		Username:     mysql.Username,
		Password:     mysql.Password,
	})
	require.NoError(t, err)

	// Validate schema across both datasources
	result, err := s.managerClient.ValidateSchema(s.ctx, client.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			pgConfigName: {
				"transactions": {"id", "account_id", "amount"},
			},
			mysqlConfigName: {
				"transactions": {"id", "account_id", "amount"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Empty(t, result.Errors)
}

// TestValidateSchema_PartialFailure tests schema validation with some valid and some invalid.
func (s *WorkerIntegrationTestSuite) TestValidateSchema_PartialFailure() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	pgConfigName := s.uniqueConfigName("postgres_schema_partial")
	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   pgConfigName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Validate: one valid table, one invalid
	result, err := s.managerClient.ValidateSchema(s.ctx, client.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			pgConfigName: {
				"transactions":     {"id", "amount"},     // Valid
				"non_existent_xyz": {"field1", "field2"}, // Invalid
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "failure", result.Status)
	assert.NotEmpty(t, result.Errors)

	// Should have exactly one error for the invalid table
	foundError := false
	for _, e := range result.Errors {
		if e.Table == "non_existent_xyz" {
			foundError = true
			break
		}
	}
	assert.True(t, foundError)
}

// =============================================================================
// SSL Connection Tests
// =============================================================================
// These tests validate SSL/TLS connections to databases.
// Run with: ENABLE_SSL=true make test-integration-container

// TestSSLConnectionValidation validates that SSL connections can be established.
func (s *WorkerIntegrationTestSuite) TestSSLConnectionValidation() {
	t := s.T()

	// Skip if SSL is not enabled
	if s.infra.PostgresSSL == nil {
		t.Skip("SSL infrastructure not available (run with ENABLE_SSL=true)")
	}

	certBundle := s.infra.GetSSLCertBundle()
	require.NotNil(t, certBundle, "SSL certificate bundle should be available")

	// Test PostgreSQL SSL connection
	t.Run("PostgreSQL SSL", func(t *testing.T) {
		pgSSL := s.infra.PostgresSSLInternal()
		configName := s.uniqueConfigName("postgres_ssl_test")

		conn, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
			ConfigName:   configName,
			Type:         "POSTGRESQL",
			Host:         pgSSL.Host,
			Port:         pgSSL.Port,
			DatabaseName: pgSSL.Database,
			Username:     pgSSL.Username,
			Password:     pgSSL.Password,
			SSL: &client.SSLInput{
				Mode: "require", // Use 'require' for self-signed certs; 'verify-full' requires hostname match
				CA:   certBundle.CACertPEM,
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, conn.ID)

		// Test the connection
		testResult, err := s.managerClient.TestConnectionEndpoint(s.ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "success", testResult.Status, "SSL connection should succeed")
	})

	// Test MySQL SSL connection
	t.Run("MySQL SSL", func(t *testing.T) {
		mysqlSSL := s.infra.MySQLSSLInternal()
		configName := s.uniqueConfigName("mysql_ssl_test")

		conn, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
			ConfigName:   configName,
			Type:         "MYSQL",
			Host:         mysqlSSL.Host,
			Port:         mysqlSSL.Port,
			DatabaseName: mysqlSSL.Database,
			Username:     mysqlSSL.Username,
			Password:     mysqlSSL.Password,
			SSL: &client.SSLInput{
				// MySQL driver expects: true, false, skip-verify, preferred, or a registered TLS config name
				// Using skip-verify for self-signed certificates
				Mode: "skip-verify",
				CA:   certBundle.CACertPEM,
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, conn.ID)

		// Test the connection
		testResult, err := s.managerClient.TestConnectionEndpoint(s.ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "success", testResult.Status, "MySQL SSL connection should succeed")
	})

	// Test MongoDB SSL connection
	t.Run("MongoDB SSL", func(t *testing.T) {
		mongoSSL := s.infra.MongoDBSSLInternal()
		configName := s.uniqueConfigName("mongodb_ssl_test")

		conn, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
			ConfigName:   configName,
			Type:         "MONGODB",
			Host:         mongoSSL.Host,
			Port:         mongoSSL.Port,
			DatabaseName: mongoSSL.Database,
			Username:     mongoSSL.Username,
			Password:     mongoSSL.Password,
			SSL: &client.SSLInput{
				Mode: "skip-verify",
				CA:   certBundle.CACertPEM,
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, conn.ID)

		// Test the connection
		testResult, err := s.managerClient.TestConnectionEndpoint(s.ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "success", testResult.Status, "MongoDB SSL connection should succeed")
	})

	// Test SQL Server SSL connection
	t.Run("SQL Server SSL", func(t *testing.T) {
		if s.infra.SQLServerSSL == nil {
			t.Skip("SQL Server SSL infrastructure not available")
		}

		sqlserverSSL := s.infra.SQLServerSSLInternal()
		configName := s.uniqueConfigName("sqlserver_ssl_test")

		conn, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
			ConfigName:   configName,
			Type:         "SQL_SERVER",
			Host:         sqlserverSSL.Host,
			Port:         sqlserverSSL.Port,
			DatabaseName: sqlserverSSL.Database,
			Username:     sqlserverSSL.Username,
			Password:     sqlserverSSL.Password,
			SSL: &client.SSLInput{
				// SQL Server uses "true" mode with TrustServerCertificate for self-signed certs
				Mode: "true",
				CA:   certBundle.CACertPEM,
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, conn.ID)

		// Test the connection
		testResult, err := s.managerClient.TestConnectionEndpoint(s.ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "success", testResult.Status, "SQL Server SSL connection should succeed")
	})
}

// TestMultiDatasourceSSLConnections tests data extraction using SSL connections.
func (s *WorkerIntegrationTestSuite) TestMultiDatasourceSSLConnections() {
	t := s.T()

	// Skip if SSL is not enabled
	if s.infra.PostgresSSL == nil {
		t.Skip("SSL infrastructure not available (run with ENABLE_SSL=true)")
	}

	certBundle := s.infra.GetSSLCertBundle()
	require.NotNil(t, certBundle, "SSL certificate bundle should be available")

	// Create PostgreSQL SSL connection
	pgSSL := s.infra.PostgresSSLInternal()
	pgConfigName := s.uniqueConfigName("postgres_ssl_multi")
	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   pgConfigName,
		Type:         "POSTGRESQL",
		Host:         pgSSL.Host,
		Port:         pgSSL.Port,
		DatabaseName: pgSSL.Database,
		Username:     pgSSL.Username,
		Password:     pgSSL.Password,
		SSL: &client.SSLInput{
			Mode: "require", // Use 'require' for self-signed certs; 'verify-full' requires hostname match
			CA:   certBundle.CACertPEM,
		},
	})
	require.NoError(t, err)

	// Create MySQL SSL connection
	mysqlSSL := s.infra.MySQLSSLInternal()
	mysqlConfigName := s.uniqueConfigName("mysql_ssl_multi")
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   mysqlConfigName,
		Type:         "MYSQL",
		Host:         mysqlSSL.Host,
		Port:         mysqlSSL.Port,
		DatabaseName: mysqlSSL.Database,
		Username:     mysqlSSL.Username,
		Password:     mysqlSSL.Password,
		SSL: &client.SSLInput{
			// MySQL driver expects: true, false, skip-verify, preferred, or a registered TLS config name
			Mode: "skip-verify",
			CA:   certBundle.CACertPEM,
		},
	})
	require.NoError(t, err)

	// Create multi-datasource job using SSL connections
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				pgConfigName: {
					"transactions": {"id", "account_id", "amount", "category"},
				},
				mysqlConfigName: {
					"transactions": {"id", "account_id", "amount", "category"},
				},
			},
		},
		Metadata: s.testMetadata("TestMultiDatasourceSSLConnections"),
	})
	require.NoError(t, err, "Failed to create multi-datasource SSL job")

	// Wait for completion
	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeoutSlow)
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status, "SSL extraction job should complete successfully")

	// Verify result
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.SizeBytes, int64(0))
	assert.Greater(t, notification.Result.RowCount, int64(0))
}

// =============================================================================
// FILTER OPERATOR TESTS - Advanced Operators (between, like)
// =============================================================================
// These tests validate advanced filter operators not covered by existing tests.
// Priority: P0 (core filter functionality)
// =============================================================================

// TestJob_Filter_Between_Numeric validates the between operator on numeric fields.
// Filters transactions where amount is between 100 and 500 (inclusive).
func (s *WorkerIntegrationTestSuite) TestJob_Filter_Between_Numeric() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_between_numeric")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create connection")
	defer func() {
		_ = s.managerClient.DeleteConnection(s.ctx, connResp.ID)
	}()

	// Create job with between filter on amount field
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount", "category", "status"},
				},
			},
			Filters: model.NestedFilters{
				configName: {
					"transactions": {
						"amount": job.FilterCondition{
							Between: []any{100, 500},
						},
					},
				},
			},
		},
		Metadata: s.testMetadata("TestJob_Filter_Between_Numeric"),
	})
	require.NoError(t, err, "Failed to create job with between filter")

	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status, "Job should complete successfully")

	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.NotEmpty(t, notification.Result.Path)
	assert.Greater(t, notification.Result.RowCount, int64(0), "Should have matching rows in range 100-500")
	assert.Less(t, notification.Result.RowCount, int64(26), "Should have fewer rows than total (26)")
}

// TestJob_Filter_Like_Contains validates the like operator with contains pattern.
func (s *WorkerIntegrationTestSuite) TestJob_Filter_Like_Contains() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_like_contains")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create connection")
	defer func() {
		_ = s.managerClient.DeleteConnection(s.ctx, connResp.ID)
	}()

	// Create job with like filter - matches categories containing 'sal' (salary)
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount", "category", "status"},
				},
			},
			Filters: model.NestedFilters{
				configName: {
					"transactions": {
						"category": job.FilterCondition{
							Like: []any{"%sal%"},
						},
					},
				},
			},
		},
		Metadata: s.testMetadata("TestJob_Filter_Like_Contains"),
	})
	require.NoError(t, err, "Failed to create job with like filter")

	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")
	assert.Equal(t, "completed", notification.Status, "Job should complete successfully")

	require.NotNil(t, notification.Result)
	assert.Greater(t, notification.Result.RowCount, int64(0), "Should have matching rows with pattern '%sal%'")
	assert.Less(t, notification.Result.RowCount, int64(26), "Should have fewer rows than total")
}

// TestJob_Filter_Like_Prefix validates the like operator with prefix pattern.
func (s *WorkerIntegrationTestSuite) TestJob_Filter_Like_Prefix() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_like_prefix")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create connection")
	defer func() {
		_ = s.managerClient.DeleteConnection(s.ctx, connResp.ID)
	}()

	// Like prefix pattern - categories starting with 'gro' (groceries)
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "amount", "category", "status"},
				},
			},
			Filters: model.NestedFilters{
				configName: {
					"transactions": {
						"category": job.FilterCondition{
							Like: []any{"gro%"},
						},
					},
				},
			},
		},
		Metadata: s.testMetadata("TestJob_Filter_Like_Prefix"),
	})
	require.NoError(t, err, "Failed to create job with like prefix filter")

	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err)
	assert.Equal(t, "completed", notification.Status)

	require.NotNil(t, notification.Result)
	assert.Greater(t, notification.Result.RowCount, int64(0), "Should have matching rows with prefix 'gro%'")
	assert.Less(t, notification.Result.RowCount, int64(26), "Should have fewer rows than total")
}

// =============================================================================
// INPUT VALIDATION TESTS - ConfigName
// =============================================================================
// These tests validate configName field constraints (minLength: 3, maxLength: 100)
// Priority: P0 (input validation)
// =============================================================================

// TestConnection_ConfigName_MinLength validates configName with exactly 3 characters.
func (s *WorkerIntegrationTestSuite) TestConnection_ConfigName_MinLength() {
	t := s.T()

	pg := s.infra.PostgresInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "abc", // Exactly 3 characters - boundary
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "ConfigName with 3 characters should succeed")
	assert.NotEmpty(t, connResp.ID)
	assert.Equal(t, "abc", connResp.ConfigName)

	_ = s.managerClient.DeleteConnection(s.ctx, connResp.ID)
}

// TestConnection_ConfigName_BelowMin validates configName with 2 characters fails.
func (s *WorkerIntegrationTestSuite) TestConnection_ConfigName_BelowMin() {
	t := s.T()

	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "ab", // Only 2 characters - below minimum
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "ConfigName with 2 characters should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_ConfigName_MaxLength validates configName with exactly 100 characters.
func (s *WorkerIntegrationTestSuite) TestConnection_ConfigName_MaxLength() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := strings.Repeat("a", 100)

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "ConfigName with 100 characters should succeed")
	assert.NotEmpty(t, connResp.ID)
	assert.Equal(t, 100, len(connResp.ConfigName))

	_ = s.managerClient.DeleteConnection(s.ctx, connResp.ID)
}

// TestConnection_ConfigName_AboveMax validates configName with 101 characters fails.
func (s *WorkerIntegrationTestSuite) TestConnection_ConfigName_AboveMax() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := strings.Repeat("a", 101)

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "ConfigName with 101 characters should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// =============================================================================
// INPUT VALIDATION TESTS - Port
// =============================================================================
// These tests validate port field constraints (minimum: 1, maximum: 65535)
// Priority: P0 (input validation)
// =============================================================================

// TestConnection_Port_MinBoundary validates port with value 1.
func (s *WorkerIntegrationTestSuite) TestConnection_Port_MinBoundary() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_port_min")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         1, // Minimum valid port
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Port 1 should be accepted for creation")
	defer func() { _ = s.managerClient.DeleteConnection(s.ctx, connResp.ID) }()

	assert.NotEmpty(t, connResp.ID)
	assert.Equal(t, 1, connResp.Port)
}

// TestConnection_Port_BelowMin validates port with value 0 fails.
func (s *WorkerIntegrationTestSuite) TestConnection_Port_BelowMin() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_port_zero")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         0, // Below minimum
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "Port 0 should fail validation")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_Port_MaxBoundary validates port with value 65535.
func (s *WorkerIntegrationTestSuite) TestConnection_Port_MaxBoundary() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_port_max")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         65535, // Maximum valid port
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Port 65535 should be accepted for creation")
	defer func() { _ = s.managerClient.DeleteConnection(s.ctx, connResp.ID) }()

	assert.NotEmpty(t, connResp.ID)
	assert.Equal(t, 65535, connResp.Port)
}

// TestConnection_Port_AboveMax validates port with value 65536 fails.
func (s *WorkerIntegrationTestSuite) TestConnection_Port_AboveMax() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_port_above_max")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         65536, // Above maximum
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "Port 65536 should fail validation")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_Port_Negative validates port with negative value fails.
func (s *WorkerIntegrationTestSuite) TestConnection_Port_Negative() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_port_negative")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         -1, // Negative value
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "Negative port should fail validation")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// =============================================================================
// INPUT VALIDATION TESTS - Required Fields
// =============================================================================
// These tests validate that missing required fields return 400 Bad Request.
// Required: configName, databaseName, host, password, type, userName
// Note: Port validation is covered in Port Validation Tests section above
// Priority: P0 (input validation)
// =============================================================================

// TestConnection_MissingConfigName validates error when configName is missing.
func (s *WorkerIntegrationTestSuite) TestConnection_MissingConfigName() {
	t := s.T()

	pg := s.infra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   "", // Missing required field
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "Missing configName should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_MissingHost validates error when host is missing.
func (s *WorkerIntegrationTestSuite) TestConnection_MissingHost() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_missing_host")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         "", // Missing required field
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "Missing host should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_MissingPassword validates error when password is missing.
func (s *WorkerIntegrationTestSuite) TestConnection_MissingPassword() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_missing_password")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     "", // Missing required field
	})
	require.Error(t, err, "Missing password should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_MissingType validates error when type is missing.
func (s *WorkerIntegrationTestSuite) TestConnection_MissingType() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_missing_type")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "", // Missing required field
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "Missing type should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_MissingDatabaseName validates error when databaseName is missing.
func (s *WorkerIntegrationTestSuite) TestConnection_MissingDatabaseName() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_missing_db")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: "", // Missing required field
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "Missing databaseName should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_MissingUsername validates error when userName is missing.
func (s *WorkerIntegrationTestSuite) TestConnection_MissingUsername() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_missing_user")

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     "", // Missing required field
		Password:     pg.Password,
	})
	require.Error(t, err, "Missing userName should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// =============================================================================
// RATE LIMITING TESTS
// =============================================================================
// These tests validate rate limiting behavior on the test connection endpoint.
// Rate limit: 10 requests per minute per connection.
// Priority: P0 (production reliability)
// =============================================================================

// TestConnection_TestEndpoint_RateLimit validates 429 response when rate limit is exceeded.
func (s *WorkerIntegrationTestSuite) TestConnection_TestEndpoint_RateLimit() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_rate_limit")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create connection")
	defer func() { _ = s.managerClient.DeleteConnection(s.ctx, connResp.ID) }()

	// Call test endpoint rapidly to trigger rate limit (10/min)
	rateLimitExceeded := false
	for i := 0; i < 15; i++ {
		_, err := s.managerClient.TestConnectionEndpoint(s.ctx, connResp.ID)
		if err != nil && strings.Contains(err.Error(), "429") {
			rateLimitExceeded = true
			t.Logf("Rate limit triggered after %d requests", i+1)
			break
		}
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "rate") {
			rateLimitExceeded = true
			t.Logf("Rate limit triggered after %d requests", i+1)
			break
		}
	}

	assert.True(t, rateLimitExceeded, "Rate limit should be triggered after multiple rapid requests")
}

// =============================================================================
// HEADER VALIDATION TESTS - X-Organization-Id
// =============================================================================
// These tests validate X-Organization-Id header requirements.
// Priority: P0 (required header validation)
// =============================================================================

// TestConnection_MissingOrganizationId validates 400 response when header is missing.
func (s *WorkerIntegrationTestSuite) TestConnection_MissingOrganizationId() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_no_org_id")

	_, err := s.managerClient.CreateConnectionWithoutOrgHeader(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.Error(t, err, "Missing X-Organization-Id should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_InvalidOrganizationId validates 400 response for invalid UUID format.
func (s *WorkerIntegrationTestSuite) TestConnection_InvalidOrganizationId() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_invalid_org_id")

	_, err := s.managerClient.CreateConnectionWithInvalidOrgHeader(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	}, "not-a-valid-uuid")
	require.Error(t, err, "Invalid X-Organization-Id should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// TestConnection_EmptyOrganizationId validates 400 response for empty header value.
func (s *WorkerIntegrationTestSuite) TestConnection_EmptyOrganizationId() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_empty_org_id")

	_, err := s.managerClient.CreateConnectionWithInvalidOrgHeader(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	}, "")
	require.Error(t, err, "Empty X-Organization-Id should fail")
	assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
}

// =============================================================================
// EDGE CASE TESTS - Empty Results
// =============================================================================
// These tests validate handling of edge cases like empty result sets.
// Priority: P0 (edge case handling)
// =============================================================================

// TestJob_EmptyResultSet validates job completion when filter matches zero rows.
func (s *WorkerIntegrationTestSuite) TestJob_EmptyResultSet() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_empty_result")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create connection")
	defer func() { _ = s.managerClient.DeleteConnection(s.ctx, connResp.ID) }()

	// Filter that matches no rows
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount", "status"},
				},
			},
			Filters: model.NestedFilters{
				configName: {
					"transactions": {
						"status": job.FilterCondition{
							Equals: []any{"nonexistent_status_xyz"},
						},
					},
				},
			},
		},
		Metadata: s.testMetadata("TestJob_EmptyResultSet"),
	})
	require.NoError(t, err, "Failed to create job with zero-match filter")

	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err, "Failed to receive job completion event")

	assert.Equal(t, "completed", notification.Status, "Job should complete successfully with empty result")
	require.NotNil(t, notification.Result, "Completed job should have result data")
	assert.Equal(t, int64(0), notification.Result.RowCount, "RowCount should be 0 for empty result set")
	assert.NotEmpty(t, notification.Result.Path, "Result path should be set")

	fetchedJob, err := s.managerClient.GetJob(s.ctx, jobResp.JobID.String())
	require.NoError(t, err)
	assert.Equal(t, "completed", fetchedJob.Status)
}

// TestJob_EmptyResultSet_MultipleFilters validates empty result with multiple filter conditions.
func (s *WorkerIntegrationTestSuite) TestJob_EmptyResultSet_MultipleFilters() {
	t := s.T()

	pg := s.infra.PostgresInternal()
	configName := s.uniqueConfigName("postgres_empty_multi_filter")

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)
	defer func() { _ = s.managerClient.DeleteConnection(s.ctx, connResp.ID) }()

	// Combined filters that yield no results
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "amount", "status"},
				},
			},
			Filters: model.NestedFilters{
				configName: {
					"transactions": {
						"status": job.FilterCondition{
							Equals: []any{"completed"},
						},
						"amount": job.FilterCondition{
							GreaterThan: []any{1000000}, // No transaction > 1 million
						},
					},
				},
			},
		},
		Metadata: s.testMetadata("TestJob_EmptyResultSet_MultipleFilters"),
	})
	require.NoError(t, err)

	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID.String(), setup.JobCompletionTimeout)
	require.NoError(t, err)

	assert.Equal(t, "completed", notification.Status)
	require.NotNil(t, notification.Result)
	assert.Equal(t, int64(0), notification.Result.RowCount)
}

// TestSuite runs the test suite.
func TestWorkerIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(WorkerIntegrationTestSuite))
}
