package job

import (
	"context"
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

// EnsureIndexes creates MongoDB indexes tailored for the jobs collection workload.
func (jr *JobMongoDBRepository) EnsureIndexes(ctx context.Context) error {
	logger, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.ensure_job_indexes")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.collection", constant.MongoCollectionJob),
	)

	logger.Infof("Creating indexes for %s collection", constant.MongoCollectionJob)

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return err
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))

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
				SetPartialFilterExpression(bson.D{{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{JobStatusProcessing, JobStatus("Processing")}}}}}),
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
				SetPartialFilterExpression(bson.D{{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{JobStatusCompleted, JobStatus("completed"), JobStatus("finished"), JobStatus("Finished")}}}}}),
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	logger.Infof("Attempting to create %d indexes for %s collection", len(indexes), constant.MongoCollectionJob)

	indexNames, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		if strings.Contains(err.Error(), "IndexOptionsConflict") || strings.Contains(err.Error(), "already exists") {
			logger.Infof("Indexes for %s already exist", constant.MongoCollectionJob)
			return nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to create job indexes", err)
		logger.Errorf("Failed to create indexes for %s: %v", constant.MongoCollectionJob, err)
		return err
	}

	logger.Infof("Successfully created %d indexes for %s collection: %v", len(indexNames), constant.MongoCollectionJob, indexNames)

	return nil
}

// DropIndexes removes custom indexes from the jobs collection.
func (jr *JobMongoDBRepository) DropIndexes(ctx context.Context) error {
	logger, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.drop_job_indexes")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.collection", constant.MongoCollectionJob),
	)

	logger.Warnf("Dropping all custom indexes for %s collection", constant.MongoCollectionJob)

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return err
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if _, err := coll.Indexes().DropAll(ctx); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to drop job indexes", err)
		logger.Errorf("Failed to drop indexes for %s: %v", constant.MongoCollectionJob, err)
		return err
	}

	logger.Infof("Successfully dropped all custom indexes for %s collection", constant.MongoCollectionJob)

	return nil
}
