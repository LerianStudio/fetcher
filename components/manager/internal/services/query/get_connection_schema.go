package query

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// GetConnectionSchema retrieves the database schema for a connection.
type GetConnectionSchema struct {
	connRepo          connRepo.Repository
	cryptor           crypto.Cryptor
	dataSourceFactory datasource.DataSourceFactory
}

// NewGetConnectionSchema creates a new GetConnectionSchema service.
func NewGetConnectionSchema(
	connectionRepo connRepo.Repository,
	cryptor crypto.Cryptor,
	factory datasource.DataSourceFactory,
) *GetConnectionSchema {
	return &GetConnectionSchema{
		connRepo:          connectionRepo,
		cryptor:           cryptor,
		dataSourceFactory: factory,
	}
}

// Execute retrieves the database schema for the specified connection.
func (s *GetConnectionSchema) Execute(ctx context.Context, organizationID, connectionID uuid.UUID) (*model.ConnectionSchemaResponse, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.get_connection_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	// Find connection by ID
	conn, err := s.connRepo.FindByID(ctx, connectionID, organizationID)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to find connection", err)
		return nil, fmt.Errorf("failed to find connection by id: %w", err)
	}

	if conn == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	// Create datasource
	ds, err := s.dataSourceFactory(ctx, conn, s.cryptor)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to create datasource", err)
		logger.Errorf("failed to create datasource for connection %s: %v", connectionID, err)

		return nil, pkg.ResponseError{
			Code:    http.StatusInternalServerError,
			Title:   "Database Connection Error",
			Message: "Failed to establish connection to the database. Check credentials and network access.",
		}
	}

	defer func() {
		if closeErr := ds.Close(ctx); closeErr != nil {
			logger.Warnf("failed to close datasource for connection %s: %v", connectionID, closeErr)
		}
	}()

	// Get schema info
	schema, err := ds.GetSchemaInfo(ctx, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to get schema info", err)
		logger.Errorf("failed to get schema info for connection %s: %v", connectionID, err)

		return nil, pkg.ResponseError{
			Code:    http.StatusInternalServerError,
			Title:   "Schema Retrieval Error",
			Message: "Failed to retrieve database schema information.",
		}
	}

	// Handle nil schema (empty database)
	if schema == nil || schema.Tables == nil {
		return model.NewConnectionSchemaFrom(conn, []model.TableDetails{}), nil
	}

	// Convert schema to response DTO, filtering system tables
	tables := make([]model.TableDetails, 0)

	for tableName, tableSchema := range schema.Tables {
		if isSystemTable(conn.Type, tableName) {
			continue
		}

		fields := tableSchema.GetColumnsList()
		sort.Strings(fields) // Sort for consistent output

		tables = append(tables, model.TableDetails{
			Name:   tableName,
			Fields: fields,
		})
	}

	// Sort tables by name for consistent output
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})

	span.SetAttributes(attribute.Int("app.schema.table_count", len(tables)))

	return model.NewConnectionSchemaFrom(conn, tables), nil
}

// isSystemTable checks if a table/collection is a system table that should be filtered out.
func isSystemTable(dbType model.DBType, tableName string) bool {
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
