package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/postgres"
	"github.com/LerianStudio/lib-commons/v4/commons"
	libConstant "github.com/LerianStudio/lib-commons/v4/commons/constants"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"go.opentelemetry.io/otel/attribute"
)

// DataSourceConfigPostgres represents a PostgreSQL-specific data source configuration.
// It embeds DataSourceConfig and adds PostgreSQL-specific fields and repository.
type DataSourceConfigPostgres struct {
	datasource.DataSourceConfig

	PostgresConnection *postgres.Connection
	PostgresRepository *postgres.ExternalDataSource
}

// GetConfig returns the base DataSourceConfig.
func (ds *DataSourceConfigPostgres) GetConfig() datasource.DataSourceConfig {
	return ds.DataSourceConfig
}

// GetType returns the database type.
func (ds *DataSourceConfigPostgres) GetType() string {
	return ds.Type
}

// Connect establishes a connection to PostgreSQL.
// This method is a no-op as the connection is established during factory creation.
func (ds *DataSourceConfigPostgres) Connect(ctx context.Context, logger libLog.Logger) error {
	ds.Status = libConstant.DataSourceStatusAvailable
	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("PostgreSQL connection ready for %s", ds.ConfigName))

	return nil
}

// Close closes the PostgreSQL connection.
func (ds *DataSourceConfigPostgres) Close(ctx context.Context) error {
	if ds.PostgresRepository != nil {
		if err := ds.PostgresRepository.CloseConnection(); err != nil {
			return err
		}
	}

	ds.Status = libConstant.DataSourceStatusUnavailable

	return nil
}

// Query executes queries on multiple PostgreSQL tables.
func (ds *DataSourceConfigPostgres) Query(ctx context.Context, tables map[string][]string, filters map[string]map[string]job.FilterCondition, logger libLog.Logger) (map[string][]map[string]any, error) {
	result := make(map[string][]map[string]any)

	// Extract unique schemas from table names
	schemas := datasource.GetUniqueSchemas(tables)

	// Check if any table is unqualified (no schema prefix) - these belong to the default "public" schema
	schemas = ensureDefaultSchemaIncluded(tables, schemas)

	schemaResult, err := ds.PostgresRepository.GetDatabaseSchema(ctx, schemas)
	if err != nil {
		logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error getting database schema: %s", err.Error()))
		return nil, err
	}

	for table, fields := range tables {
		tableFilters := getTableFilters(filters, table)

		var (
			queryResult any
			errQuery    error
		)

		if len(tableFilters) > 0 {
			queryResult, errQuery = ds.PostgresRepository.QueryWithAdvancedFilters(ctx, schemaResult, table, fields, tableFilters)
		} else {
			queryResult, errQuery = ds.PostgresRepository.Query(ctx, schemaResult, table, fields, nil)
		}

		if errQuery != nil {
			logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error querying table %s: %s", table, errQuery.Error()))
			return nil, errQuery
		}

		tableResult, ok := queryResult.([]map[string]any)
		if !ok {
			logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Unexpected query result type for table %s", table))
			return nil, fmt.Errorf("unexpected query result type for table %s", table)
		}

		result[table] = tableResult
	}

	return result, nil
}

// GetSchemaInfo returns the schema information for PostgreSQL.
func (ds *DataSourceConfigPostgres) GetSchemaInfo(ctx context.Context, schemas []string) (*model.DataSourceSchema, error) {
	_, tracer, _, _ := commons.NewTrackingFromContext(ctx) //nolint:dogsled // Only tracer needed for span creation

	ctx, span := tracer.Start(ctx, "datasource.postgres.get_schema_info")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.datasource.config_name", ds.ConfigName),
		attribute.String("app.datasource.type", "postgres"),
	)

	schemaResult, err := ds.PostgresRepository.GetDatabaseSchema(ctx, schemas)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to get database schema", err)
		return nil, fmt.Errorf("failed to get PostgreSQL schema: %w", err)
	}

	// Use factory function from domain entity
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

// getTableFilters extracts filters for a specific table.
// Supports matching with or without schema prefix for flexibility:
// - Exact match: filters["public.transactions"] for table "public.transactions"
// - Without schema: filters["transactions"] for table "public.transactions"
// - With default schema: filters["public.transactions"] for table "transactions"
func getTableFilters(databaseFilters map[string]map[string]job.FilterCondition, tableName string) map[string]job.FilterCondition {
	if databaseFilters == nil {
		return nil
	}

	// 1. Try exact match first
	if filters, exists := databaseFilters[tableName]; exists {
		return filters
	}

	// 2. If tableName has schema prefix (e.g., "public.transactions"),
	//    try without schema (e.g., "transactions")
	if strings.Contains(tableName, ".") {
		_, unqualifiedName := datasource.SplitSchemaTable(tableName)
		if filters, exists := databaseFilters[unqualifiedName]; exists {
			return filters
		}
	}

	// 3. If tableName has no schema prefix, try with default schema
	if !strings.Contains(tableName, ".") {
		qualifiedName := postgres.DefaultSchema + "." + tableName
		if filters, exists := databaseFilters[qualifiedName]; exists {
			return filters
		}
	}

	return nil
}

// ensureDefaultSchemaIncluded adds the default "public" schema to the schemas list
// if any table name is unqualified (has no schema prefix with a dot).
// This ensures tables in the public schema are discoverable when mixed with schema-qualified tables.
func ensureDefaultSchemaIncluded(tables map[string][]string, schemas []string) []string {
	// Check if any table has no dot (unqualified name)
	hasUnqualifiedTable := false

	for tableName := range tables {
		if !containsDot(tableName) {
			hasUnqualifiedTable = true
			break
		}
	}

	// If there are unqualified tables, ensure public schema is included
	if hasUnqualifiedTable {
		if !containsSchema(schemas, postgres.DefaultSchema) {
			schemas = append(schemas, postgres.DefaultSchema)
		}
	}

	return schemas
}

// containsDot checks if a string contains a dot character.
func containsDot(s string) bool {
	for _, c := range s {
		if c == '.' {
			return true
		}
	}

	return false
}

// containsSchema checks if a schema name is already in the list.
func containsSchema(schemas []string, target string) bool {
	for _, s := range schemas {
		if s == target {
			return true
		}
	}

	return false
}
