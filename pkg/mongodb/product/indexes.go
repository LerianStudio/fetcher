package product

import (
	"context"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	sharedMongo "github.com/LerianStudio/fetcher/pkg/mongodb"
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

// EnsureIndexes creates MongoDB indexes tailored for the products collection workload.
func (pr *ProductMongoDBRepository) EnsureIndexes(ctx context.Context) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.ensure_product_indexes")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.collection", constant.MongoCollectionProduct),
	)

	logger.Infof("Creating indexes for %s collection", constant.MongoCollectionProduct)

	db, err := pr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return err
	}

	coll := db.Database(strings.ToLower(pr.Database)).Collection(strings.ToLower(constant.MongoCollectionProduct))

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "organization_id", Value: 1},
				{Key: "code", Value: 1},
			},
			Options: options.Index().
				SetName("idx_product_org_code").
				SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "deleted_at", Value: nil}}),
		},
		{
			Keys: bson.D{
				{Key: "organization_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_product_org_created").
				SetPartialFilterExpression(bson.D{{Key: "deleted_at", Value: nil}}),
		},
	}

	span.SetAttributes(
		attribute.Int("app.request.index_count", len(indexes)),
	)

	ctx, cancel := context.WithTimeout(ctx, indexCreateTimeout)
	defer cancel()

	logger.Infof("Attempting to create %d indexes for %s collection", len(indexes), constant.MongoCollectionProduct)

	indexNames, err := coll.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		if sharedMongo.IsIndexConflictError(err) {
			logger.Infof("Indexes for %s already exist", constant.MongoCollectionProduct)
			return nil
		}

		libOpentelemetry.HandleSpanError(&span, "Failed to create product indexes", err)
		logger.Errorf("Failed to create indexes for %s: %v", constant.MongoCollectionProduct, err)

		return err
	}

	logger.Infof("Successfully created %d indexes for %s collection: %v", len(indexNames), constant.MongoCollectionProduct, indexNames)

	return nil
}

// DropIndexes removes custom indexes from the products collection.
func (pr *ProductMongoDBRepository) DropIndexes(ctx context.Context) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.drop_product_indexes")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.collection", constant.MongoCollectionProduct),
	)

	logger.Warnf("Dropping all custom indexes for %s collection", constant.MongoCollectionProduct)

	db, err := pr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return err
	}

	coll := db.Database(strings.ToLower(pr.Database)).Collection(strings.ToLower(constant.MongoCollectionProduct))

	ctx, cancel := context.WithTimeout(ctx, indexDropTimeout)
	defer cancel()

	droppedIndexes, err := coll.Indexes().DropAll(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to drop product indexes", err)
		logger.Errorf("Failed to drop indexes for %s: %v", constant.MongoCollectionProduct, err)

		return err
	}

	logger.Infof("Dropped indexes: %v", droppedIndexes)
	logger.Infof("Successfully dropped all custom indexes for %s collection", constant.MongoCollectionProduct)

	return nil
}
