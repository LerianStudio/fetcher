package oracle

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/oracle"
	"github.com/LerianStudio/lib-commons/v2/commons"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	"go.opentelemetry.io/otel/attribute"
)

// DataSourceConfigOracle represents an Oracle-specific data source configuration.
// It embeds DataSourceConfig and adds Oracle-specific fields and repository.
type DataSourceConfigOracle struct {
	datasource.DataSourceConfig

	OracleConnection *oracle.Connection
	OracleRepository oracle.Repository
}

// GetConfig returns the base DataSourceConfig.
func (ds *DataSourceConfigOracle) GetConfig() datasource.DataSourceConfig {
	return ds.DataSourceConfig
}

// GetType returns the database type.
func (ds *DataSourceConfigOracle) GetType() string {
	return ds.Type
}

// Connect establishes a connection to Oracle.
// This method is a no-op as the connection is established during factory creation.
func (ds *DataSourceConfigOracle) Connect(ctx context.Context, logger log.Logger) error {
	ds.Status = libConstant.DataSourceStatusAvailable
	logger.Infof("Oracle connection ready for %s", ds.ConfigName)

	return nil
}

// Close closes the Oracle connection.
func (ds *DataSourceConfigOracle) Close(ctx context.Context) error {
	if ds.OracleRepository != nil {
		if err := ds.OracleRepository.CloseConnection(); err != nil {
			return err
		}
	}

	ds.Status = libConstant.DataSourceStatusUnavailable

	return nil
}

// Query executes queries on multiple Oracle tables.
func (ds *DataSourceConfigOracle) Query(ctx context.Context, tables map[string][]string, filters map[string]map[string]job.FilterCondition, logger log.Logger) (map[string][]map[string]any, error) {
	result := make(map[string][]map[string]any)

	// Extract unique schemas from table names
	schemas := datasource.GetUniqueSchemas(tables)

	schemaResult, err := ds.OracleRepository.GetDatabaseSchema(ctx, schemas)
	if err != nil {
		logger.Errorf("Error getting database schema: %s", err.Error())
		return nil, err
	}

	for table, fields := range tables {
		tableFilters := getTableFilters(filters, table)

		var (
			queryResult any
			errQuery    error
		)

		if len(tableFilters) > 0 {
			queryResult, errQuery = ds.OracleRepository.QueryWithAdvancedFilters(ctx, schemaResult, table, fields, tableFilters)
		} else {
			queryResult, errQuery = ds.OracleRepository.Query(ctx, schemaResult, table, fields, nil)
		}

		if errQuery != nil {
			logger.Errorf("Error querying table %s: %s", table, errQuery.Error())
			return nil, errQuery
		}

		tableResult, ok := queryResult.([]map[string]any)
		if !ok {
			logger.Errorf("Unexpected query result type for table %s", table)
			return nil, fmt.Errorf("unexpected query result type for table %s", table)
		}

		result[table] = tableResult
	}

	return result, nil
}

// getTableFilters extracts filters for a specific table.
func getTableFilters(databaseFilters map[string]map[string]job.FilterCondition, tableName string) map[string]job.FilterCondition {
	if databaseFilters == nil {
		return nil
	}

	return databaseFilters[tableName]
}

// GetSchemaInfo returns the schema information for Oracle.
func (ds *DataSourceConfigOracle) GetSchemaInfo(ctx context.Context, schemas []string) (*model.DataSourceSchema, error) {
	_, tracer, _, _ := commons.NewTrackingFromContext(ctx) //nolint:dogsled // Only tracer needed for span creation

	ctx, span := tracer.Start(ctx, "datasource.oracle.get_schema_info")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.datasource.config_name", ds.ConfigName),
		attribute.String("app.datasource.type", "oracle"),
	)

	schemaResult, err := ds.OracleRepository.GetDatabaseSchema(ctx, schemas)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to get database schema", err)
		return nil, fmt.Errorf("failed to get Oracle schema: %w", err)
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
