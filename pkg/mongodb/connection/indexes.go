package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg/constant"
	sharedMongo "github.com/LerianStudio/fetcher/pkg/mongodb"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

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
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

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

	dropCtx, dropCancel := context.WithTimeout(ctx, indexDropTimeout)
	defer dropCancel()

	if err := dropOrphanProductIndexes(dropCtx, coll, logger); err != nil {
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
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

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
// bootstrap: both missing-collection (fresh DB) and missing-index are
// tolerated and treated as already-cleaned.
func dropOrphanProductIndexes(ctx context.Context, coll *mongo.Collection, logger libLog.Logger) error {
	for _, name := range orphanProductIndexes {
		if _, err := coll.Indexes().DropOne(ctx, name); err != nil {
			if isIgnorableDropIndexError(err) {
				continue
			}

			return fmt.Errorf("drop orphan index %q: %w", name, err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Dropped orphan product_id index %q", name))
	}

	return nil
}

// isIgnorableDropIndexError reports whether err can be safely ignored during
// the orphan-index cleanup. Two server errors are treated as "already gone":
//
//   - NamespaceNotFound (code 26): the collection does not exist yet
//     (fresh DB on first boot — EnsureIndexes will create it).
//   - IndexNotFound (code 27): the index has already been dropped or was
//     never present.
//
// Prefer the typed mongo.CommandError discriminator (matching the pattern
// of IsIndexConflictError in pkg/mongodb/errors.go); the string fallback
// covers driver wrappers or wire-protocol formats that do not surface the
// command error structure.
func isIgnorableDropIndexError(err error) bool {
	if err == nil {
		return false
	}

	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		return cmdErr.Code == 26 || cmdErr.Code == 27
	}

	msg := err.Error()

	return strings.Contains(msg, "IndexNotFound") ||
		strings.Contains(msg, "index not found") ||
		strings.Contains(msg, "NamespaceNotFound") ||
		strings.Contains(msg, "ns not found")
}
