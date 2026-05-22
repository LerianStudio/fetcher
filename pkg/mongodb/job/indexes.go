package job

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	sharedMongo "github.com/LerianStudio/fetcher/pkg/mongodb"
	observability "github.com/LerianStudio/lib-observability"
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

// EnsureIndexes creates MongoDB indexes tailored for the jobs collection workload.
func (jr *JobMongoDBRepository) EnsureIndexes(ctx context.Context) error {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

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
				{Key: "status", Value: 1},
			},
			Options: options.Index().
				SetName("idx_job_status"),
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
				{Key: "request_hash", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_job_hash_created"),
		},
		{
			Keys: bson.D{
				{Key: "metadata.terminalEventPending", Value: 1},
				{Key: "status", Value: 1},
				{Key: "completed_at", Value: 1},
				{Key: "created_at", Value: 1},
			},
			Options: options.Index().
				SetName("idx_job_terminal_event_repair").
				SetPartialFilterExpression(bson.D{{Key: "metadata.terminalEventPending", Value: true}}),
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

	if err := ensureUniqueActiveHashIndex(ctx, coll, logger); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to ensure unique active hash index", err)
		return err
	}

	return nil
}

// ensureUniqueActiveHashIndex creates (or migrates) the uniq_job_hash_active
// index. The index enforces the invariant: no two jobs with the same
// request_hash may coexist while either is pending or processing.
//
// Historically the index used a partialFilterExpression with $in on status,
// which MongoDB accepts but AWS DocumentDB rejects as "unsupported expression
// in partial index". The new form filters by equality on the derived field
// dedup_active, which works in both engines.
//
// This function is idempotent and safe to run on every bootstrap:
//   - If the index does not exist yet, it is created.
//   - If it exists in the new form, nothing changes.
//   - If it exists in the legacy form, it is dropped, existing jobs are
//     backfilled with dedup_active, and the new form is created.
func ensureUniqueActiveHashIndex(ctx context.Context, coll *mongo.Collection, logger libLog.Logger) error {
	state, err := inspectUniqJobHashActive(ctx, coll)
	if err != nil {
		return fmt.Errorf("inspect uniq_job_hash_active: %w", err)
	}

	if state == uniqIndexCurrent {
		logger.Log(ctx, libLog.LevelInfo, "uniq_job_hash_active already uses dedup_active filter; skipping")
		return nil
	}

	if state == uniqIndexLegacy {
		logger.Log(ctx, libLog.LevelInfo, "Dropping legacy uniq_job_hash_active index for migration to dedup_active filter")

		if _, dropErr := coll.Indexes().DropOne(ctx, "uniq_job_hash_active"); dropErr != nil {
			return fmt.Errorf("drop legacy uniq_job_hash_active: %w", dropErr)
		}
	}

	if err := backfillDedupActive(ctx, coll, logger); err != nil {
		return fmt.Errorf("backfill dedup_active: %w", err)
	}

	newIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "request_hash", Value: 1}},
		Options: options.Index().
			SetName("uniq_job_hash_active").
			SetUnique(true).
			SetPartialFilterExpression(bson.D{{Key: "dedup_active", Value: true}}),
	}

	logger.Log(ctx, libLog.LevelInfo, "Creating uniq_job_hash_active with dedup_active filter")

	if _, err := coll.Indexes().CreateOne(ctx, newIndex); err != nil {
		if sharedMongo.IsIndexConflictError(err) {
			logger.Log(ctx, libLog.LevelInfo, "uniq_job_hash_active already exists with compatible options")
			return nil
		}

		if mongo.IsDuplicateKeyError(err) {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf(
				"Cannot create uniq_job_hash_active for %s: existing duplicate active jobs prevent index creation. "+
					"Manual cleanup required — deduplicate jobs with same request_hash where dedup_active=true. Error: %v",
				constant.MongoCollectionJob, err,
			))

			return err
		}

		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to create uniq_job_hash_active for %s: %v", constant.MongoCollectionJob, err))

		return err
	}

	logger.Log(ctx, libLog.LevelInfo, "uniq_job_hash_active created successfully")

	return nil
}

// uniqIndexState classifies the live state of uniq_job_hash_active.
type uniqIndexState int

const (
	uniqIndexAbsent  uniqIndexState = iota // index does not exist
	uniqIndexLegacy                        // index exists with $in partial filter (incompatible with DocumentDB)
	uniqIndexCurrent                       // index exists with dedup_active equality filter
)

// inspectUniqJobHashActive returns the current state of the uniq_job_hash_active index.
func inspectUniqJobHashActive(ctx context.Context, coll *mongo.Collection) (uniqIndexState, error) {
	cur, err := coll.Indexes().List(ctx)
	if err != nil {
		return uniqIndexAbsent, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var spec bson.M
		if err := cur.Decode(&spec); err != nil {
			return uniqIndexAbsent, err
		}

		name, _ := spec["name"].(string)
		if name != "uniq_job_hash_active" {
			continue
		}

		if isCurrentUniqJobHashActive(spec) {
			return uniqIndexCurrent, nil
		}

		return uniqIndexLegacy, nil
	}

	if err := cur.Err(); err != nil {
		return uniqIndexAbsent, err
	}

	return uniqIndexAbsent, nil
}

// isCurrentUniqJobHashActive reports whether a decoded index spec matches
// the canonical shape of the new uniq_job_hash_active index. Every structural
// trait is validated rather than relying on the presence of a single field:
//
//   - partialFilterExpression must be EXACTLY {dedup_active: true} — extra
//     predicates would narrow the indexed set and could let duplicate active
//     request_hash values escape the uniqueness invariant.
//   - unique flag must be set.
//   - key must be a single ascending entry on request_hash.
//
// A drifted or hand-crafted index that matches only by name is intentionally
// classified as legacy so the migration path repairs it.
func isCurrentUniqJobHashActive(spec bson.M) bool {
	pfe, ok := spec["partialFilterExpression"].(bson.M)
	if !ok || len(pfe) != 1 {
		return false
	}

	dedupActive, _ := pfe["dedup_active"].(bool)
	if !dedupActive {
		return false
	}

	unique, _ := spec["unique"].(bool)
	if !unique {
		return false
	}

	key, ok := spec["key"].(bson.M)
	if !ok || len(key) != 1 {
		return false
	}

	// BSON decodes numeric index direction as int32. Accept int as well to
	// tolerate driver versions or test fixtures that hand-roll the spec.
	switch v := key["request_hash"].(type) {
	case int32:
		return v == 1
	case int64:
		return v == 1
	case int:
		return v == 1
	}

	return false
}

// backfillDedupActive ensures every existing job document carries a
// dedup_active boolean consistent with its status. Idempotent — only updates
// rows whose dedup_active is missing or incorrect for the current status.
func backfillDedupActive(ctx context.Context, coll *mongo.Collection, logger libLog.Logger) error {
	activeRes, err := coll.UpdateMany(ctx,
		bson.M{
			"status":       bson.M{"$in": bson.A{model.JobStatusPending, model.JobStatusProcessing}},
			"dedup_active": bson.M{"$ne": true},
		},
		bson.M{"$set": bson.M{"dedup_active": true}},
	)
	if err != nil {
		return fmt.Errorf("set dedup_active=true on active jobs: %w", err)
	}

	terminalRes, err := coll.UpdateMany(ctx,
		bson.M{
			"status":       bson.M{"$in": bson.A{model.JobStatusCompleted, model.JobStatusFailed}},
			"dedup_active": bson.M{"$ne": false},
		},
		bson.M{"$set": bson.M{"dedup_active": false}},
	)
	if err != nil {
		return fmt.Errorf("set dedup_active=false on terminal jobs: %w", err)
	}

	if activeRes.ModifiedCount > 0 || terminalRes.ModifiedCount > 0 {
		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf(
			"Backfilled dedup_active: %d active, %d terminal",
			activeRes.ModifiedCount, terminalRes.ModifiedCount,
		))
	}

	return nil
}

// DropIndexes removes custom indexes from the jobs collection.
func (jr *JobMongoDBRepository) DropIndexes(ctx context.Context) error {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

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
