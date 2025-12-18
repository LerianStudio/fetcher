package connection

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

const (
	indexCreateTimeout = 60 * time.Second
	indexDropTimeout   = 30 * time.Second
)

// isIndexConflictError checks if the error is a MongoDB index conflict error.
// IndexOptionsConflict is code 85, IndexKeySpecsConflict is code 86.
func isIndexConflictError(err error) bool {
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		return cmdErr.Code == 85 || cmdErr.Code == 86
	}

	return false
}

// EnsureIndexes creates MongoDB indexes tailored for the connections collection workload.
func (cr *ConnectionMongoDBRepository) EnsureIndexes(ctx context.Context) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.ensure_connection_indexes")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.collection", constant.MongoCollectionConnection),
	)

	logger.Infof("Creating indexes for %s collection", constant.MongoCollectionConnection)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "organization_id", Value: 1},
				{Key: "config_name", Value: 1},
			},
			Options: options.Index().
				SetName("idx_connection_org_config_name").
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "deleted_at", Value: nil}}),
		},
		{
			Keys: bson.D{
				{Key: "organization_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_connection_org_created").
				SetPartialFilterExpression(bson.D{{Key: "deleted_at", Value: nil}}),
		},
		{
			Keys: bson.D{
				{Key: "organization_id", Value: 1},
				{Key: "database_name", Value: 1},
			},
			Options: options.Index().
				SetName("idx_connection_org_database_name").
				SetPartialFilterExpression(bson.D{{Key: "deleted_at", Value: nil}}),
		},
	}

	span.SetAttributes(
		attribute.Int("app.request.index_count", len(indexes)),
	)

	ctx, cancel := context.WithTimeout(ctx, indexCreateTimeout)
	defer cancel()

	logger.Infof("Attempting to create %d indexes for %s collection", len(indexes), constant.MongoCollectionConnection)

	indexNames, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		if isIndexConflictError(err) {
			logger.Infof("Indexes for %s already exist", constant.MongoCollectionConnection)
			return nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to create connection indexes", err)
		logger.Errorf("Failed to create indexes for %s: %v", constant.MongoCollectionConnection, err)

		return err
	}

	logger.Infof("Successfully created %d indexes for %s collection: %v", len(indexNames), constant.MongoCollectionConnection, indexNames)

	return nil
}

// DropIndexes removes custom indexes from the connections collection.
func (cr *ConnectionMongoDBRepository) DropIndexes(ctx context.Context) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.drop_connection_indexes")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.collection", constant.MongoCollectionConnection),
	)

	logger.Warnf("Dropping all custom indexes for %s collection", constant.MongoCollectionConnection)

	db, err := cr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return err
	}

	coll := db.Database(strings.ToLower(cr.Database)).Collection(strings.ToLower(constant.MongoCollectionConnection))

	ctx, cancel := context.WithTimeout(ctx, indexDropTimeout)
	defer cancel()

	droppedIndexes, err := coll.Indexes().DropAll(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to drop connection indexes", err)
		logger.Errorf("Failed to drop indexes for %s: %v", constant.MongoCollectionConnection, err)

		return err
	}

	logger.Infof("Dropped indexes: %v", droppedIndexes)
	logger.Infof("Successfully dropped all custom indexes for %s collection", constant.MongoCollectionConnection)

	return nil
}
