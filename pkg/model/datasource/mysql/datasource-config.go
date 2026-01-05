package mysql

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/mysql"
	"github.com/LerianStudio/lib-commons/v2/commons"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	"go.opentelemetry.io/otel/attribute"
)

// DataSourceConfigMySQL represents a MySQL-specific data source configuration.
// It embeds DataSourceConfig and adds MySQL-specific fields and repository.
type DataSourceConfigMySQL struct {
	datasource.DataSourceConfig

	MySQLConnection *mysql.Connection
	MySQLRepository *mysql.ExternalDataSource
}

// GetConfig returns the base DataSourceConfig.
func (ds *DataSourceConfigMySQL) GetConfig() datasource.DataSourceConfig {
	return ds.DataSourceConfig
}

// GetType returns the database type.
func (ds *DataSourceConfigMySQL) GetType() string {
	return ds.Type
}

// Connect establishes a connection to MySQL.
// This method is a no-op as the connection is established during factory creation.
func (ds *DataSourceConfigMySQL) Connect(ctx context.Context, logger log.Logger) error {
	ds.Status = libConstant.DataSourceStatusAvailable
	logger.Infof("MySQL connection ready for %s", ds.ConfigName)

	return nil
}

// Close closes the MySQL connection.
func (ds *DataSourceConfigMySQL) Close(ctx context.Context) error {
	if ds.MySQLRepository != nil {
		if err := ds.MySQLRepository.CloseConnection(); err != nil {
			return err
		}
	}

	ds.Status = libConstant.DataSourceStatusUnavailable

	return nil
}

// Query executes queries on multiple MySQL tables.
func (ds *DataSourceConfigMySQL) Query(ctx context.Context, tables map[string][]string, filters map[string]map[string]job.FilterCondition, logger log.Logger) (map[string][]map[string]any, error) {
	result := make(map[string][]map[string]any)

	schemaResult, err := ds.MySQLRepository.GetDatabaseSchema(ctx)
	if err != nil {
		logger.Errorf("Error getting database schema: %s", err.Error())
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
			queryResult, errQuery = ds.MySQLRepository.QueryWithAdvancedFilters(ctx, schemaResult, table, fields, tableFilters)
		} else {
			queryResult, errQuery = ds.MySQLRepository.Query(ctx, schemaResult, table, fields, nil)
		}

		if errQuery != nil {
			logger.Errorf("Error querying table %s: %s", table, errQuery.Error())
			return nil, errQuery
		}

		tableResult = queryResult.([]map[string]any)
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

// GetSchemaInfo returns the schema information for MySQL.
func (ds *DataSourceConfigMySQL) GetSchemaInfo(ctx context.Context, schemas []string) (*model.DataSourceSchema, error) {
	_, tracer, _, _ := commons.NewTrackingFromContext(ctx) //nolint:dogsled // Only tracer needed for span creation

	ctx, span := tracer.Start(ctx, "datasource.mysql.get_schema_info")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.datasource.config_name", ds.ConfigName),
		attribute.String("app.datasource.type", "mysql"),
	)

	schemaResult, err := ds.MySQLRepository.GetDatabaseSchema(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to get database schema", err)
		return nil, fmt.Errorf("failed to get MySQL schema: %w", err)
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
