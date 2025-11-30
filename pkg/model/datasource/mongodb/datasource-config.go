package mongodb

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
)

// DataSourceConfigMongoDB represents a MongoDB-specific data source configuration.
// It embeds DataSourceConfig and adds MongoDB-specific fields and repository.
// Note: MongoDBRepository is stored as interface{} to avoid import cycles.
// The actual type is *mongodb.ExternalDataSource and should be type-asserted when used.
type DataSourceConfigMongoDB struct {
	datasource.DataSourceConfig

	MongoDBRepository any
	MongoURI          string
	Options           string
}

// GetConfig returns the base DataSourceConfig.
func (ds *DataSourceConfigMongoDB) GetConfig() datasource.DataSourceConfig {
	return ds.DataSourceConfig
}

// GetType returns the database type.
func (ds *DataSourceConfigMongoDB) GetType() string {
	return ds.DataSourceConfig.Type
}

// Connect establishes a connection to MongoDB.
// This method is a no-op as the connection is established during factory creation.
func (ds *DataSourceConfigMongoDB) Connect(ctx context.Context, logger log.Logger) error {
	ds.Status = libConstant.DataSourceStatusAvailable
	logger.Infof("MongoDB connection ready for %s", ds.ConfigName)
	return nil
}

// Close closes the MongoDB connection.
func (ds *DataSourceConfigMongoDB) Close(ctx context.Context) error {
	if ds.MongoDBRepository != nil {
		type closer interface {
			CloseConnection(ctx context.Context) error
		}
		if repo, ok := ds.MongoDBRepository.(closer); ok {
			if err := repo.CloseConnection(ctx); err != nil {
				return err
			}
		}
	}
	ds.Status = libConstant.DataSourceStatusUnavailable
	return nil
}
