package postgres

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/postgres"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
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
func (ds *DataSourceConfigPostgres) Connect(ctx context.Context, logger log.Logger) error {
	ds.Status = libConstant.DataSourceStatusAvailable
	logger.Infof("PostgreSQL connection ready for %s", ds.ConfigName)

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
func (ds *DataSourceConfigPostgres) Query(ctx context.Context, tables map[string][]string, filters map[string]map[string]job.FilterCondition, logger log.Logger) (map[string][]map[string]any, error) {
	result := make(map[string][]map[string]any)

	schemaResult, err := ds.PostgresRepository.GetDatabaseSchema(ctx)
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
			queryResult, errQuery = ds.PostgresRepository.QueryWithAdvancedFilters(ctx, schemaResult, table, fields, tableFilters)
		} else {
			queryResult, errQuery = ds.PostgresRepository.Query(ctx, schemaResult, table, fields, nil)
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
