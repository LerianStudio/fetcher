package connection

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	sharedMongo "github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

const (
	indexCreateTimeout = 60 * time.Second
	indexDropTimeout   = 30 * time.Second
)

// EnsureIndexes creates MongoDB indexes tailored for the connections collection workload.
func (cr *ConnectionMongoDBRepository) EnsureIndexes(ctx context.Context) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.ensure_connection_indexes")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.collection", constant.MongoCollectionConnection),
	)

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Creating indexes for %s collection", constant.MongoCollectionConnection))

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return err
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))

	if err := dropOrphanProductIndexes(ctx, coll, logger); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to drop orphan product_id indexes", err)
		return err
	}

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "config_name", Value: 1},
			},
			Options: options.Index().
				SetName("idx_connection_config_name").
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "deleted_at", Value: nil}}),
		},
		{
			Keys: bson.D{
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_connection_created").
				SetPartialFilterExpression(bson.D{{Key: "deleted_at", Value: nil}}),
		},
		{
			Keys: bson.D{
				{Key: "database_name", Value: 1},
			},
			Options: options.Index().
				SetName("idx_connection_database_name").
				SetPartialFilterExpression(bson.D{{Key: "deleted_at", Value: nil}}),
		},
	}

	span.SetAttributes(
		attribute.Int("app.request.index_count", len(indexes)),
	)

	ctx, cancel := context.WithTimeout(ctx, indexCreateTimeout)
	defer cancel()

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Attempting to create %d indexes for %s collection", len(indexes), constant.MongoCollectionConnection))

	indexNames, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		if sharedMongo.IsIndexConflictError(err) {
			logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Indexes for %s already exist", constant.MongoCollectionConnection))
			return nil
		}

		libOpentelemetry.HandleSpanError(span, "Failed to create connection indexes", err)
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to create indexes for %s: %v", constant.MongoCollectionConnection, err))

		return err
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Successfully created %d indexes for %s collection: %v", len(indexNames), constant.MongoCollectionConnection, indexNames))

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

	logger.Log(ctx, libLog.LevelWarn, fmt.Sprintf("Dropping all custom indexes for %s collection", constant.MongoCollectionConnection))

	db, err := cr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return err
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionConnection))

	ctx, cancel := context.WithTimeout(ctx, indexDropTimeout)
	defer cancel()

	droppedIndexes, err := coll.Indexes().DropAll(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to drop connection indexes", err)
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to drop indexes for %s: %v", constant.MongoCollectionConnection, err))

		return err
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Dropped indexes: %v", droppedIndexes))
	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Successfully dropped all custom indexes for %s collection", constant.MongoCollectionConnection))

	return nil
}

// orphanProductIndexes lists indexes that referenced the legacy product_id
// field on the connections collection. The product entity was removed from
// the domain model (see commit a6b8339, "remove product entity and make
// connections standalone") but the indexes survived as dead code, indexing
// a field that no write path ever populates.
//
// One of them, idx_connection_product_config, additionally used a
// partialFilterExpression with $type: "binData", which DocumentDB rejects.
// Removing all three eliminates both the dead-code overhead and the
// DocumentDB incompatibility in one step.
var orphanProductIndexes = []string{
	"idx_connection_product_config",
	"idx_connection_product_created",
	"idx_connection_unassigned",
}

// dropOrphanProductIndexes removes legacy product_id-based indexes from
// environments where they were previously created. Safe to run on every
// bootstrap: IndexNotFound is tolerated and treated as already-cleaned.
func dropOrphanProductIndexes(ctx context.Context, coll *mongo.Collection, logger libLog.Logger) error {
	for _, name := range orphanProductIndexes {
		if _, err := coll.Indexes().DropOne(ctx, name); err != nil {
			if isIndexNotFoundError(err) {
				continue
			}

			return fmt.Errorf("drop orphan index %q: %w", name, err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Dropped orphan product_id index %q", name))
	}

	return nil
}

// isIndexNotFoundError reports whether err signals that the index simply does
// not exist on the collection. The Mongo driver does not expose a typed sentinel
// for IndexNotFound, so we match on the documented error text. Both MongoDB
// and DocumentDB return the same message for this case.
func isIndexNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "IndexNotFound") || strings.Contains(msg, "index not found")
}
