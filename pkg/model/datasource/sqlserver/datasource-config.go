package sqlserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/sqlserver"
	libConstant "github.com/LerianStudio/lib-commons/v5/commons/constants"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// DefaultSchema is the default SQL Server schema name
const DefaultSchema = "dbo"

// DataSourceConfigSQLServer represents a SQL Server-specific data source configuration.
// It embeds DataSourceConfig and adds SQL Server-specific fields and repository.
type DataSourceConfigSQLServer struct {
	datasource.DataSourceConfig

	SQLServerConnection *sqlserver.Connection
	SQLServerRepository sqlserver.Datasource
}

// GetConfig returns the base DataSourceConfig.
func (ds *DataSourceConfigSQLServer) GetConfig() datasource.DataSourceConfig {
	return ds.DataSourceConfig
}

// GetType returns the database type.
func (ds *DataSourceConfigSQLServer) GetType() string {
	return ds.Type
}

// Connect establishes a connection to SQL Server.
// This method is a no-op as the connection is established during factory creation.
func (ds *DataSourceConfigSQLServer) Connect(ctx context.Context, logger libLog.Logger) error {
	ds.Status = libConstant.DataSourceStatusAvailable
	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("SQL Server connection ready for %s", ds.ConfigName))

	return nil
}

// Close closes the SQL Server connection.
func (ds *DataSourceConfigSQLServer) Close(ctx context.Context) error {
	if ds.SQLServerRepository != nil {
		if err := ds.SQLServerRepository.CloseConnection(); err != nil {
			return err
		}
	}

	ds.Status = libConstant.DataSourceStatusUnavailable

	return nil
}

// Query executes queries on multiple SQL Server tables.
func (ds *DataSourceConfigSQLServer) Query(ctx context.Context, tables map[string][]string, filters map[string]map[string]job.FilterCondition, logger libLog.Logger) (map[string][]map[string]any, error) {
	result := make(map[string][]map[string]any)

	// Extract unique schemas from table names
	schemas := datasource.GetUniqueSchemas(tables)

	schemaResult, err := ds.SQLServerRepository.GetDatabaseSchema(ctx, schemas)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error getting database schema: %s", err.Error()))
		return nil, err
	}

	for table, fields := range tables {
		tableFilters := getTableFilters(filters, table)

		var (
			tableResult []map[string]any
			queryResult any
			errQuery    error
		)

		if len(tableFilters) > 0 {
			queryResult, errQuery = ds.SQLServerRepository.QueryWithAdvancedFilters(ctx, schemaResult, table, fields, tableFilters)
		} else {
			queryResult, errQuery = ds.SQLServerRepository.Query(ctx, schemaResult, table, fields, nil)
		}

		if errQuery != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error querying table %s: %s", table, errQuery.Error()))
			return nil, errQuery
		}

		tableResult = queryResult.([]map[string]any)
		result[table] = tableResult
	}

	return result, nil
}

// getTableFilters extracts filters for a specific table.
// Supports matching with or without schema prefix for flexibility.
func getTableFilters(databaseFilters map[string]map[string]job.FilterCondition, tableName string) map[string]job.FilterCondition {
	if databaseFilters == nil {
		return nil
	}

	// 1. Try exact match first
	if filters, exists := databaseFilters[tableName]; exists {
		return filters
	}

	// 2. If tableName has schema prefix, try without schema
	if strings.Contains(tableName, ".") {
		_, unqualifiedName := datasource.SplitSchemaTable(tableName)
		if filters, exists := databaseFilters[unqualifiedName]; exists {
			return filters
		}
	}

	// 3. If tableName has no schema prefix, try with default schema (dbo)
	if !strings.Contains(tableName, ".") {
		qualifiedName := DefaultSchema + "." + tableName
		if filters, exists := databaseFilters[qualifiedName]; exists {
			return filters
		}
	}

	return nil
}

// GetSchemaInfo returns the schema information for SQL Server.
func (ds *DataSourceConfigSQLServer) GetSchemaInfo(ctx context.Context, schemas []string) (*model.DataSourceSchema, error) {
	_, tracer, _, _ := observability.NewTrackingFromContext(ctx) //nolint:dogsled // Only tracer needed for span creation

	ctx, span := tracer.Start(ctx, "datasource.sqlserver.get_schema_info")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.datasource.config_name", ds.ConfigName),
		attribute.String("app.datasource.type", "sqlserver"),
	)

	schemaResult, err := ds.SQLServerRepository.GetDatabaseSchema(ctx, schemas)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to get database schema", err)
		return nil, fmt.Errorf("failed to get SQL Server schema: %w", err)
	}

	schema := model.NewDataSourceSchema(ds.ConfigName)

	for _, table := range schemaResult {
		columns := make([]string, len(table.Columns))
		for i, col := range table.Columns {
			columns[i] = col.Name
		}

		schema.AddTable(table.TableName, columns)
	}

	span.SetAttributes(attribute.Int("app.schema.tables_count", len(schema.Tables)))

	return schema, nil
}
