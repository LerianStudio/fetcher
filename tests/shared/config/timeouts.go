package config

import "time"

// =============================================================================
// TIMEOUT CONSTANTS
// =============================================================================
// All timeout values used across integration and chaos tests are centralized
// here for easy tuning and consistency.
// =============================================================================

// -----------------------------------------------------------------------------
// Test Suite Timeouts
// -----------------------------------------------------------------------------

const (
	// SuiteTimeout is the maximum duration for the entire test suite.
	SuiteTimeout = 15 * time.Minute

	// ManagerReadyTimeout is the maximum time to wait for the Manager API to become ready.
	ManagerReadyTimeout = 60 * time.Second

	// ManagerReadyPollInterval is the interval between health check retries.
	ManagerReadyPollInterval = 1 * time.Second
)

// -----------------------------------------------------------------------------
// Job Completion Timeouts
// -----------------------------------------------------------------------------

const (
	// JobCompletionTimeout is the default timeout for waiting for a job to complete.
	JobCompletionTimeout = 6000 * time.Second

	// JobCompletionTimeoutSlow is the timeout for slower datasources like Oracle
	// or multi-datasource jobs that require more processing time.
	JobCompletionTimeoutSlow = 9000 * time.Second
)

// -----------------------------------------------------------------------------
// File Storage Timeouts
// -----------------------------------------------------------------------------

const (
	// SeaweedFSFileTimeout is the timeout for waiting for a file to appear in SeaweedFS.
	SeaweedFSFileTimeout = 30 * time.Second
)

// -----------------------------------------------------------------------------
// Container Startup Timeouts
// -----------------------------------------------------------------------------

const (
	// RabbitMQStartupTimeout is the time to wait for RabbitMQ to become ready.
	RabbitMQStartupTimeout = 120 * time.Second

	// SeaweedFSStartupTimeout is the time to wait for SeaweedFS to become ready.
	SeaweedFSStartupTimeout = 60 * time.Second

	// PostgresStartupTimeout is the time to wait for PostgreSQL to become ready.
	PostgresStartupTimeout = 60 * time.Second

	// MySQLStartupTimeout is the time to wait for MySQL to become ready.
	MySQLStartupTimeout = 120 * time.Second

	// SQLServerStartupTimeout is the time to wait for SQL Server to become ready.
	SQLServerStartupTimeout = 180 * time.Second

	// SQLServerInitDelay is the additional delay after SQL Server reports ready
	// before executing init scripts. SQL Server needs time to fully configure
	// the sa user even after reporting "ready for client connections".
	SQLServerInitDelay = 5 * time.Second

	// OracleStartupTimeout is the time to wait for Oracle XE to become ready.
	// Oracle takes significantly longer to initialize.
	OracleStartupTimeout = 300 * time.Second

	// MongoDBStartupTimeout is the time to wait for MongoDB to become ready.
	MongoDBStartupTimeout = 60 * time.Second

	// RedisStartupTimeout is the time to wait for Redis/Valkey to become ready.
	RedisStartupTimeout = 30 * time.Second

	// ManagerStartupTimeout is the time to wait for the Manager container to become ready.
	ManagerStartupTimeout = 120 * time.Second

	// WorkerStartupTimeout is the time to wait for the Worker container to start consuming.
	WorkerStartupTimeout = 120 * time.Second

	// ToxiproxyStartupTimeout is the time to wait for Toxiproxy to become ready.
	ToxiproxyStartupTimeout = 30 * time.Second
)

// -----------------------------------------------------------------------------
// HTTP Client Timeouts
// -----------------------------------------------------------------------------

const (
	// HTTPClientTimeout is the default timeout for HTTP client requests.
	HTTPClientTimeout = 30 * time.Second

	// PollingInterval is the interval for polling operations (job status, file existence).
	PollingInterval = 500 * time.Millisecond
)
