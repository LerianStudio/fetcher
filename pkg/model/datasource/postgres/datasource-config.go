package postgres

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
)

// DataSourceConfigPostgres represents a PostgreSQL-specific data source configuration.
// It embeds DataSourceConfig and adds PostgreSQL-specific fields and repository.
// Note: PostgresConnection and PostgresRepository are stored as interface{} to avoid import cycles.
// The actual types are *postgres.Connection and *postgres.ExternalDataSource and should be type-asserted when used.
type DataSourceConfigPostgres struct {
	datasource.DataSourceConfig

	PostgresConnection any // *postgres.Connection - using any to avoid import cycle
	PostgresRepository any // *postgres.ExternalDataSource - using any to avoid import cycle
}

// GetConfig returns the base DataSourceConfig.
func (ds *DataSourceConfigPostgres) GetConfig() datasource.DataSourceConfig {
	return ds.DataSourceConfig
}

// GetType returns the database type.
func (ds *DataSourceConfigPostgres) GetType() string {
	return ds.DataSourceConfig.Type
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
		type closer interface {
			CloseConnection() error
		}
		if repo, ok := ds.PostgresRepository.(closer); ok {
			if err := repo.CloseConnection(); err != nil {
				return err
			}
		}
	}
	ds.Status = libConstant.DataSourceStatusUnavailable
	return nil
}
