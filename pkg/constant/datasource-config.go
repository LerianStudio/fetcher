package constant

import "time"

// PostgreSQL Pool Configuration
const (
	PostgresConnMaxLifetime = 5 * time.Minute
	PostgresConnMaxIdleTime = 1 * time.Minute
	PostgresMaxOpenConns    = 25
	PostgresMaxIdleConns    = 10
)

// Oracle Pool Configuration
const (
	OracleConnMaxLifetime = 5 * time.Minute
	OracleConnMaxIdleTime = 1 * time.Minute
	OracleMaxOpenConns    = 25
	OracleMaxIdleConns    = 10
)

// MySQL Pool Configuration
const (
	MySQLConnMaxLifetime = 5 * time.Minute
	MySQLConnMaxIdleTime = 1 * time.Minute
	MySQLMaxOpenConns    = 25
	MySQLMaxIdleConns    = 10
)

// SQL Server Pool Configuration
const (
	SQLServerConnMaxLifetime = 5 * time.Minute
	SQLServerConnMaxIdleTime = 1 * time.Minute
	SQLServerMaxOpenConns    = 25
	SQLServerMaxIdleConns    = 10
)

// Query Timeout Configuration
const (
	QueryTimeoutMedium     = 10 * time.Second
	QueryTimeoutSlow       = 15 * time.Second
	SchemaDiscoveryTimeout = 30 * time.Second
	ConnectionTimeout      = 5 * time.Second
)

// MongoDB Pool Configuration
const (
	MongoDBMaxPoolSize     uint64 = 100
	MongoDBMinPoolSize     uint64 = 10
	MongoDBMaxConnIdleTime        = 1 * time.Minute
)
