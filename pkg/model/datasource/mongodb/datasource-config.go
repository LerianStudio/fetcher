package mongodb

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	libConstant "github.com/LerianStudio/lib-commons/v2/commons/constants"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
)

// DataSourceConfigMongoDB represents a MongoDB-specific data source configuration.
// It embeds DataSourceConfig and adds MongoDB-specific fields and repository.
type DataSourceConfigMongoDB struct {
	datasource.DataSourceConfig

	MongoDBRepository *mongodb.ExternalDataSource
	MongoURI          string
	Options           string
}

// GetConfig returns the base DataSourceConfig.
func (ds *DataSourceConfigMongoDB) GetConfig() datasource.DataSourceConfig {
	return ds.DataSourceConfig
}

// GetType returns the database type.
func (ds *DataSourceConfigMongoDB) GetType() string {
	return ds.Type
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
		if err := ds.MongoDBRepository.CloseConnection(ctx); err != nil {
			return err
		}
	}

	ds.Status = libConstant.DataSourceStatusUnavailable

	return nil
}

// Query executes queries on multiple MongoDB collections.
func (ds *DataSourceConfigMongoDB) Query(ctx context.Context, collections map[string][]string, filters map[string]map[string]job.FilterCondition, logger log.Logger) (map[string][]map[string]any, error) {
	result := make(map[string][]map[string]any)

	for collection, fields := range collections {
		collectionFilters := getCollectionFilters(filters, collection)

		var (
			collectionResult []map[string]any
			errQuery         error
		)

		if len(collectionFilters) > 0 {
			collectionResult, errQuery = ds.MongoDBRepository.QueryWithAdvancedFilters(ctx, collection, fields, collectionFilters)
		} else {
			collectionResult, errQuery = ds.MongoDBRepository.Query(ctx, collection, fields, nil)
		}

		if errQuery != nil {
			logger.Errorf("Error querying collection %s: %s", collection, errQuery.Error())
			return nil, fmt.Errorf("error querying collection %s: %w", collection, errQuery)
		}

		result[collection] = collectionResult
	}

	return result, nil
}

// getCollectionFilters extracts filters for a specific collection.
func getCollectionFilters(databaseFilters map[string]map[string]job.FilterCondition, collectionName string) map[string]job.FilterCondition {
	if databaseFilters == nil {
		return nil
	}

	return databaseFilters[collectionName]
}
