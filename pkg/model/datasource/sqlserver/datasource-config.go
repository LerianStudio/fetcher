package sqlserver

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/sqlserver"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
)

// DataSourceConfigSQLServer represents a SQL Server-specific data source configuration.
// It embeds DataSourceConfig and adds SQL Server-specific fields and repository.
type DataSourceConfigSQLServer struct {
	datasource.DataSourceConfig

	SQLServerConnection *sqlserver.Connection
	SQLServerRepository *sqlserver.ExternalDataSource
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
func (ds *DataSourceConfigSQLServer) Connect(ctx context.Context, logger log.Logger) error {
	ds.Status = libConstant.DataSourceStatusAvailable
	logger.Infof("SQL Server connection ready for %s", ds.ConfigName)

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
func (ds *DataSourceConfigSQLServer) Query(ctx context.Context, tables map[string][]string, filters map[string]map[string]job.FilterCondition, logger log.Logger) (map[string][]map[string]any, error) {
	result := make(map[string][]map[string]any)

	schemaResult, err := ds.SQLServerRepository.GetDatabaseSchema(ctx)
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
			queryResult, errQuery = ds.SQLServerRepository.QueryWithAdvancedFilters(ctx, schemaResult, table, fields, tableFilters)
		} else {
			queryResult, errQuery = ds.SQLServerRepository.Query(ctx, schemaResult, table, fields, nil)
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
