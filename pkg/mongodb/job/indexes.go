package job

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	sharedMongo "github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

const (
	indexCreateTimeout = 60 * time.Second
	indexDropTimeout   = 30 * time.Second
)

// EnsureIndexes creates MongoDB indexes tailored for the jobs collection workload.
func (jr *JobMongoDBRepository) EnsureIndexes(ctx context.Context) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.ensure_job_indexes")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.collection", constant.MongoCollectionJob),
	)

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Creating indexes for %s collection", constant.MongoCollectionJob))

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return err
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "organization_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().
				SetName("idx_job_org_status"),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_job_status_created").
				SetPartialFilterExpression(bson.D{{Key: "status", Value: model.JobStatusProcessing}}),
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().
				SetName("idx_job_created"),
		},
		{
			Keys: bson.D{{Key: "completed_at", Value: -1}},
			Options: options.Index().
				SetName("idx_job_completed"),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "completed_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_job_status_completed").
				SetPartialFilterExpression(bson.D{{Key: "status", Value: model.JobStatusCompleted}}),
		},
		{
			Keys: bson.D{
				{Key: "organization_id", Value: 1},
				{Key: "request_hash", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_job_org_hash_created"),
		},
	}

	span.SetAttributes(
		attribute.Int("app.request.index_count", len(indexes)),
	)

	ctx, cancel := context.WithTimeout(ctx, indexCreateTimeout)
	defer cancel()

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Attempting to create %d indexes for %s collection", len(indexes), constant.MongoCollectionJob))

	indexNames, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		if sharedMongo.IsIndexConflictError(err) {
			logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Indexes for %s already exist", constant.MongoCollectionJob))
			return nil
		}

		libOpentelemetry.HandleSpanError(span, "Failed to create job indexes", err)
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to create indexes for %s: %v", constant.MongoCollectionJob, err))

		return err
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Successfully created %d indexes for %s collection: %v", len(indexNames), constant.MongoCollectionJob, indexNames))

	return nil
}

// DropIndexes removes custom indexes from the jobs collection.
func (jr *JobMongoDBRepository) DropIndexes(ctx context.Context) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.drop_job_indexes")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.collection", constant.MongoCollectionJob),
	)

	logger.Log(ctx, libLog.LevelWarn, fmt.Sprintf("Dropping all custom indexes for %s collection", constant.MongoCollectionJob))

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return err
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))

	ctx, cancel := context.WithTimeout(ctx, indexDropTimeout)
	defer cancel()

	droppedIndexes, err := coll.Indexes().DropAll(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to drop job indexes", err)
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to drop indexes for %s: %v", constant.MongoCollectionJob, err))

		return err
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Dropped indexes: %v", droppedIndexes))
	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Successfully dropped all custom indexes for %s collection", constant.MongoCollectionJob))

	return nil
}
