// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package enginecompatschema

import (
	"strings"

	"github.com/LerianStudio/fetcher/pkg/model"
)

// IsSystemTable reports whether a table/collection is a system object that the
// host excludes from a schema snapshot before it crosses the Engine boundary.
// The conventions are datasource-type-specific (pg_*, Oracle SYS, SQL Server
// db_*, Mongo system.*) and therefore live in the host adapter, NOT the Engine
// core, which validates literal snapshot names. This is the same predicate the
// legacy Manager applied in GetConnectionSchema, moved here unchanged so schema
// discovery through the Engine keeps its byte-identical filtering.
func IsSystemTable(dbType model.DBType, tableName string) bool {
	tableNameLower := strings.ToLower(tableName)

	switch dbType {
	case model.TypePostgreSQL:
		return isPostgreSQLSystemTable(tableNameLower)
	case model.TypeMySQL:
		return isMySQLSystemTable(tableNameLower)
	case model.TypeOracle:
		return isOracleSystemTable(tableName) // Oracle handles case normalization internally
	case model.TypeSQLServer:
		return isSQLServerSystemTable(tableNameLower)
	case model.TypeMongoDB:
		return isMongoDBSystemCollection(tableNameLower)
	default:
		return false
	}
}

// isPostgreSQLSystemTable checks if a table is a PostgreSQL system table.
func isPostgreSQLSystemTable(tableName string) bool {
	systemPrefixes := []string{
		"pg_",
		"information_schema",
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(tableName, prefix) || tableName == prefix {
			return true
		}
	}

	return false
}

// isMySQLSystemTable checks if a table is a MySQL system table.
func isMySQLSystemTable(tableName string) bool {
	systemSchemas := map[string]bool{
		"mysql":              true,
		"information_schema": true,
		"performance_schema": true,
		"sys":                true,
	}

	return systemSchemas[tableName]
}

// isOracleSystemTable checks if a table is an Oracle system table.
// Handles both uppercase (standard) and lowercase (driver-dependent) table names.
// Filters out:
// - System schema tables (SYS, SYSTEM, etc.)
// - Internal objects containing $ (e.g., ROLLING$EVENTS, MVIEW$_ADV_*)
// - LogMiner tables (LOGMNR*, LOGMNRGGC_*)
// - Recovery tables (REDO_LOG, etc.)
func isOracleSystemTable(tableName string) bool {
	// Normalize to uppercase for comparison (Oracle convention)
	tableNameUpper := strings.ToUpper(tableName)

	// Filter internal Oracle objects containing $ (system/internal tables)
	if strings.Contains(tableNameUpper, "$") {
		return true
	}

	// Filter LogMiner and recovery-related tables
	systemPrefixes := []string{
		"LOGMNR",   // LogMiner tables (LOGMNR*, LOGMNRC_*, LOGMNRGGC_*)
		"REDO_",    // Redo log tables
		"MVIEW$",   // Materialized view tables
		"AQ$",      // Advanced Queuing tables
		"DEF$",     // Deferred RPC tables
		"REPCAT$",  // Replication tables
		"SQLPLUS_", // SQL*Plus tables
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(tableNameUpper, prefix) {
			return true
		}
	}

	// Also filter exact matches for common system tables
	systemTables := map[string]bool{
		"REDO_LOG":   true,
		"REDO_DB":    true,
		"PLAN_TABLE": true,
	}

	if systemTables[tableNameUpper] {
		return true
	}

	systemSchemas := map[string]bool{
		"SYS":                true,
		"SYSTEM":             true,
		"OUTLN":              true,
		"XDB":                true,
		"MDSYS":              true,
		"CTXSYS":             true,
		"DBSNMP":             true,
		"WMSYS":              true,
		"EXFSYS":             true,
		"ORDSYS":             true,
		"ORDDATA":            true,
		"ORDPLUGINS":         true,
		"SI_INFORMTN_SCHEMA": true,
		"APEX_PUBLIC_USER":   true,
		"APEX_040000":        true,
		"APEX_030200":        true,
		"FLOWS_FILES":        true,
		"ANONYMOUS":          true,
	}

	return systemSchemas[tableNameUpper]
}

// isSQLServerSystemTable checks if a table is a SQL Server system table.
// Filters sys, information_schema, and db_ prefixed schemas.
func isSQLServerSystemTable(tableName string) bool {
	systemSchemas := map[string]bool{
		"sys":                true,
		"information_schema": true,
	}

	// Check exact match first
	if systemSchemas[tableName] {
		return true
	}

	// Check db_ prefix (e.g., db_owner, db_backup, db_accessadmin)
	if strings.HasPrefix(tableName, "db_") {
		return true
	}

	// Check if schema portion starts with db_ (e.g., "db_backup.audit_logs")
	if schema, _, hasDot := strings.Cut(tableName, "."); hasDot {
		if strings.HasPrefix(schema, "db_") {
			return true
		}
	}

	return false
}

// isMongoDBSystemCollection checks if a collection is a MongoDB system collection.
func isMongoDBSystemCollection(collectionName string) bool {
	systemDatabases := map[string]bool{
		"admin":  true,
		"local":  true,
		"config": true,
	}

	// Also filter system collections that start with "system."
	if strings.HasPrefix(collectionName, "system.") {
		return true
	}

	return systemDatabases[collectionName]
}
