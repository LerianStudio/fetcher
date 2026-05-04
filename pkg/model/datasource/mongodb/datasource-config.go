package mongodb

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/lib-commons/v5/commons"
	libConstant "github.com/LerianStudio/lib-commons/v5/commons/constants"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
	"go.opentelemetry.io/otel/attribute"
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
func (ds *DataSourceConfigMongoDB) Connect(ctx context.Context, logger libLog.Logger) error {
	ds.Status = libConstant.DataSourceStatusAvailable
	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("MongoDB connection ready for %s", ds.ConfigName))

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
func (ds *DataSourceConfigMongoDB) Query(ctx context.Context, collections map[string][]string, filters map[string]map[string]job.FilterCondition, logger libLog.Logger) (map[string][]map[string]any, error) {
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
			logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error querying collection %s: %s", collection, errQuery.Error()))
			return nil, fmt.Errorf("error querying collection %s: %w", collection, errQuery)
		}

		result[collection] = collectionResult
	}

	return result, nil
}

// QueryCollection queries a single MongoDB collection with the specified fields and optional filter.
// Implements portDS.CRMQueryable.
func (ds *DataSourceConfigMongoDB) QueryCollection(ctx context.Context, collection string, fields []string, filter map[string][]any) ([]map[string]any, error) {
	return ds.MongoDBRepository.Query(ctx, collection, fields, filter)
}

// QueryCollectionWithAdvancedFilters queries a single MongoDB collection using advanced FilterCondition filters.
// Implements portDS.CRMQueryable.
func (ds *DataSourceConfigMongoDB) QueryCollectionWithAdvancedFilters(ctx context.Context, collection string, fields []string, filters map[string]job.FilterCondition) ([]map[string]any, error) {
	return ds.MongoDBRepository.QueryWithAdvancedFilters(ctx, collection, fields, filters)
}

// ListCollectionNames returns all collection names in the MongoDB database.
// Implements portDS.CRMQueryable.
func (ds *DataSourceConfigMongoDB) ListCollectionNames(ctx context.Context) ([]string, error) {
	return ds.MongoDBRepository.ListCollectionNames(ctx)
}

// getCollectionFilters extracts filters for a specific collection.
func getCollectionFilters(databaseFilters map[string]map[string]job.FilterCondition, collectionName string) map[string]job.FilterCondition {
	if databaseFilters == nil {
		return nil
	}

	return databaseFilters[collectionName]
}

// GetSchemaInfo returns the schema information for MongoDB.
func (ds *DataSourceConfigMongoDB) GetSchemaInfo(ctx context.Context, schemas []string) (*model.DataSourceSchema, error) {
	_, tracer, _, _ := commons.NewTrackingFromContext(ctx) //nolint:dogsled // Only tracer needed for span creation

	ctx, span := tracer.Start(ctx, "datasource.mongodb.get_schema_info")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.datasource.config_name", ds.ConfigName),
		attribute.String("app.datasource.type", "mongodb"),
	)

	schemaResult, err := ds.MongoDBRepository.GetDatabaseSchema(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to get database schema", err)
		return nil, fmt.Errorf("failed to get MongoDB schema: %w", err)
	}

	schema := model.NewDataSourceSchema(ds.ConfigName)

	for _, collection := range schemaResult {
		columns := make([]string, len(collection.Fields))
		for i, field := range collection.Fields {
			columns[i] = field.Name
		}

		schema.AddTable(collection.CollectionName, columns)
	}

	span.SetAttributes(attribute.Int("app.schema.collections_count", len(schema.Tables)))

	return schema, nil
}
