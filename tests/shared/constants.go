// Package shared provides common utilities, constants, and helpers for E2E and integration tests.
// It includes infrastructure configuration, API clients, and test assertions that are shared
// across different test packages.
package shared

import "time"

// Database type constants represent the supported database types in the Fetcher API.
// These values must match the API's expected type strings.
const (
	// DBTypePostgreSQL represents a PostgreSQL database connection.
	DBTypePostgreSQL = "POSTGRESQL"
	// DBTypeMySQL represents a MySQL database connection.
	DBTypeMySQL = "MYSQL"
	// DBTypeMongoDB represents a MongoDB database connection.
	DBTypeMongoDB = "MONGODB"
	// DBTypeOracle represents an Oracle database connection.
	DBTypeOracle = "ORACLE"
	// DBTypeSQLServer represents a SQL Server database connection.
	DBTypeSQLServer = "SQL_SERVER"
)

// Core infrastructure credentials are used to connect to the shared test infrastructure
// components (MongoDB, RabbitMQ, Redis, SeaweedFS). These are the credentials configured
// in the test containers.
const (
	// CoreInfraUsername is the username for core infrastructure services.
	CoreInfraUsername = "plugin"
	// CoreInfraPassword is the password for core infrastructure services.
	CoreInfraPassword = "Lerian@123"
)

// Source database credentials are used for external databases that serve as data sources
// in E2E tests (PostgreSQL, MySQL, Oracle, etc.).
const (
	// SourceDBUsername is the default username for source databases.
	SourceDBUsername = "testuser"
	// SourceDBPassword is the default password for source databases.
	SourceDBPassword = "testpass"
)

// Job status constants represent the possible states of a Fetcher job.
const (
	// JobStatusPending indicates the job is queued and waiting to be processed.
	JobStatusPending = "pending"
	// JobStatusProcessing indicates the job is currently being executed.
	JobStatusProcessing = "processing"
	// JobStatusCompleted indicates the job finished successfully.
	JobStatusCompleted = "completed"
	// JobStatusFailed indicates the job encountered an error during execution.
	JobStatusFailed = "failed"
)

// Default test timeouts define the standard wait durations for various test operations.
// These values are tuned for CI environments and local development.
const (
	// DefaultJobTimeout is how long to wait for a job to complete.
	DefaultJobTimeout = 90 * time.Second

	// DefaultConnectTimeout is how long to wait for database connections.
	DefaultConnectTimeout = 30 * time.Second

	// DefaultTestTimeout is the maximum duration for the entire test execution.
	DefaultTestTimeout = 2 * time.Minute

	// DefaultHTTPTimeout is the timeout for HTTP client operations.
	DefaultHTTPTimeout = 30 * time.Second
)

// Application ports define the network ports used by Fetcher services.
const (
	// ManagerAPIPort is the HTTP port for the Manager API service.
	ManagerAPIPort = 4006
)

// Test account IDs are UUIDs used consistently across all database fixtures.
// These allow tests to reference specific accounts regardless of which database is being tested.
const (
	// TestAccount1ID is the UUID for the first test account.
	TestAccount1ID = "11111111-1111-1111-1111-111111111111"
	// TestAccount2ID is the UUID for the second test account.
	TestAccount2ID = "22222222-2222-2222-2222-222222222222"
	// TestAccount3ID is the UUID for the third test account.
	TestAccount3ID = "33333333-3333-3333-3333-333333333333"
)
