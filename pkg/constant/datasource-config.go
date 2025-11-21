package constant

import "time"

const (
	// PostgreSQLType represents PostgreSQL database type
	PostgreSQLType = "postgresql"

	// MongoDBType represents the MongoDB database type constant.
	MongoDBType = "mongodb"
)

// PostgreSQL Pool Configuration
const (
	PostgresConnMaxLifetime = 5 * time.Minute
	PostgresConnMaxIdleTime = 1 * time.Minute
	PostgresMaxOpenConns    = 25
	PostgresMaxIdleConns    = 10
)

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
