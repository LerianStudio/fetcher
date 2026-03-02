package job

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	sharedMongo "github.com/LerianStudio/fetcher/pkg/mongodb"
	"github.com/LerianStudio/lib-commons/v3/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

const (
	indexCreateTimeout = 60 * time.Second
	indexDropTimeout   = 30 * time.Second
	remediationTimeout = 120 * time.Second
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

	logger.Infof("Creating indexes for %s collection", constant.MongoCollectionJob)

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
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

	uniqueActiveHashIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "organization_id", Value: 1},
			{Key: "request_hash", Value: 1},
		},
		Options: options.Index().
			SetName("uniq_job_org_hash_active").
			SetUnique(true).
			SetPartialFilterExpression(bson.D{
				{Key: "status", Value: bson.D{
					{Key: "$in", Value: bson.A{model.JobStatusPending, model.JobStatusProcessing}},
				}},
			}),
	}

	span.SetAttributes(
		attribute.Int("app.request.index_count", len(indexes)),
	)

	ctx, cancel := context.WithTimeout(ctx, indexCreateTimeout)
	defer cancel()

	logger.Infof("Attempting to create %d indexes for %s collection", len(indexes), constant.MongoCollectionJob)

	indexNames, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		if sharedMongo.IsIndexConflictError(err) {
			logger.Infof("Indexes for %s already exist", constant.MongoCollectionJob)
		} else {
			libOpentelemetry.HandleSpanError(&span, "Failed to create job indexes", err)
			logger.Errorf("Failed to create indexes for %s: %v", constant.MongoCollectionJob, err)

			return err
		}
	}

	logger.Infof("Successfully created %d indexes for %s collection: %v", len(indexNames), constant.MongoCollectionJob, indexNames)

	if _, err := coll.Indexes().CreateOne(ctx, uniqueActiveHashIndex); err != nil {
		if sharedMongo.IsIndexConflictError(err) {
			logger.Infof("Unique active hash index already exists or has equivalent options")
			return nil
		}

		if mongo.IsDuplicateKeyError(err) {
			logger.Warnf("Duplicate active jobs detected while creating unique active hash index; attempting remediation")

			remediationCtx, remediationCancel := context.WithTimeout(context.Background(), remediationTimeout)
			defer remediationCancel()

			resolvedCount, remediationErr := jr.resolveDuplicateActiveJobs(remediationCtx, coll)
			if remediationErr != nil {
				libOpentelemetry.HandleSpanError(&span, "Failed to remediate duplicate active jobs", remediationErr)
				return fmt.Errorf("remediate duplicate active jobs: %w", remediationErr)
			}

			logger.Warnf("Remediated %d duplicate active jobs; retrying unique index creation", resolvedCount)

			if _, retryErr := coll.Indexes().CreateOne(ctx, uniqueActiveHashIndex); retryErr != nil {
				if sharedMongo.IsIndexConflictError(retryErr) {
					logger.Infof("Unique active hash index already exists after remediation")
					return nil
				}

				libOpentelemetry.HandleSpanError(&span, "Failed to create unique active hash index after remediation", retryErr)

				return fmt.Errorf("create unique active hash index after remediation: %w", retryErr)
			}

			logger.Infof("Successfully created unique active hash index after remediation")

			return nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to create unique active hash index", err)

		return fmt.Errorf("create unique active hash index: %w", err)
	}

	logger.Infof("Successfully ensured unique active hash index for %s collection", constant.MongoCollectionJob)

	return nil
}

func (jr *JobMongoDBRepository) resolveDuplicateActiveJobs(ctx context.Context, coll *mongo.Collection) (int64, error) {
	// The pipeline groups active jobs by (org, hash) and uses $first/$push with
	// a preceding $sort to deterministically identify the newest job as keeper.
	// $first is spec-guaranteed to take the first document per group after $sort,
	// while $push order after $sort is observed in practice but not guaranteed.
	// We use keeperID ($first) to exclude the keeper, rather than relying on
	// $push ordering.
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "status", Value: bson.D{{Key: "$in", Value: bson.A{model.JobStatusPending, model.JobStatusProcessing}}}},
			{Key: "request_hash", Value: bson.D{{Key: "$exists", Value: true}, {Key: "$ne", Value: ""}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "organization_id", Value: "$organization_id"},
				{Key: "request_hash", Value: "$request_hash"},
			}},
			{Key: "keeperId", Value: bson.D{{Key: "$first", Value: "$_id"}}},
			{Key: "allIds", Value: bson.D{{Key: "$push", Value: "$_id"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$match", Value: bson.D{{Key: "count", Value: bson.D{{Key: "$gt", Value: 1}}}}}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	type duplicateGroup struct {
		KeeperID any   `bson:"keeperId"`
		AllIDs   []any `bson:"allIds"`
	}

	now := time.Now().UTC()
	totalResolved := int64(0)

	for cursor.Next(ctx) {
		var group duplicateGroup
		if err := cursor.Decode(&group); err != nil {
			return totalResolved, err
		}

		if len(group.AllIDs) <= 1 {
			continue
		}

		// Build redundant list by excluding the keeper (spec-safe via $first).
		// MongoDB _id fields decoded into any may hold primitive.Binary (for UUIDs)
		// which contains []byte and is not comparable via ==. We use bsonIDEqual
		// for safe comparison across all BSON types.
		redundantIDs := make([]any, 0, len(group.AllIDs)-1)
		for _, id := range group.AllIDs {
			if !bsonIDEqual(id, group.KeeperID) {
				redundantIDs = append(redundantIDs, id)
			}
		}

		if len(redundantIDs) == 0 {
			continue
		}

		updateFilter := bson.M{
			"_id": bson.M{"$in": redundantIDs},
			"status": bson.M{
				"$in": bson.A{model.JobStatusPending, model.JobStatusProcessing},
			},
		}

		update := bson.M{
			"$set": bson.M{
				"status":                               model.JobStatusFailed,
				"completed_at":                         now,
				"metadata.error":                       "job auto-failed during unique index remediation",
				"metadata.index_remediation_reason":    "duplicate active request hash",
				"metadata.index_remediation_timestamp": now.Format(time.RFC3339Nano),
			},
		}

		result, err := coll.UpdateMany(ctx, updateFilter, update)
		if err != nil {
			return totalResolved, err
		}

		totalResolved += result.ModifiedCount
	}

	if err := cursor.Err(); err != nil {
		return totalResolved, err
	}

	return totalResolved, nil
}

// bsonIDEqual compares two BSON _id values that may be of any type.
// MongoDB _id fields decoded into any can hold primitive.ObjectID ([12]byte, comparable),
// primitive.Binary (contains []byte, NOT comparable via ==), or string.
// Using == on uncomparable types causes a runtime panic, so we handle
// primitive.Binary explicitly with bytes.Equal.
func bsonIDEqual(a, b any) bool {
	binA, okA := a.(primitive.Binary)
	binB, okB := b.(primitive.Binary)

	if okA && okB {
		return binA.Subtype == binB.Subtype && string(binA.Data) == string(binB.Data)
	}

	// For comparable types (ObjectID, string, etc.) direct comparison is safe.
	// If one is Binary and the other is not, they are not equal.
	if okA != okB {
		return false
	}

	// Both are non-Binary comparable types — safe to use ==.
	return a == b
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

	logger.Warnf("Dropping all custom indexes for %s collection", constant.MongoCollectionJob)

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return err
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))

	ctx, cancel := context.WithTimeout(ctx, indexDropTimeout)
	defer cancel()

	droppedIndexes, err := coll.Indexes().DropAll(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to drop job indexes", err)
		logger.Errorf("Failed to drop indexes for %s: %v", constant.MongoCollectionJob, err)

		return err
	}

	logger.Infof("Dropped indexes: %v", droppedIndexes)
	logger.Infof("Successfully dropped all custom indexes for %s collection", constant.MongoCollectionJob)

	return nil
}
