// Package sslmode provides SSL/TLS mode validation for database connections.
//
// This package implements allowlist-based validation for SSL mode parameters
// to prevent injection attacks in connection strings. Each database driver
// has its own set of valid SSL modes, and this package provides validators
// for each supported database type.
//
// Security: All validators follow a fail-secure approach - unknown values
// are rejected rather than defaulted to a "safe" value.
//
// Supported databases:
//   - MySQL: Uses the go-sql-driver/mysql tls parameter
//   - Oracle: Uses the sijms/go-ora ssl and ssl_verify parameters
//   - PostgreSQL: Uses the lib/pq and pgx sslmode parameter
//   - MongoDB: Uses the mongo-go-driver tls and tlsInsecure parameters
//   - SQL Server: Uses the go-mssqldb encrypt parameter
//
// Usage:
//
//	if err := sslmode.ValidateMySQLMode(mode); err != nil {
//	    return err // Invalid mode, potential injection attempt
//	}
//	// Safe to use mode in connection string
package sslmode
