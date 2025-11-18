package report

import (
	"context"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/lib-commons/v2/commons"
	libMongo "github.com/LerianStudio/lib-commons/v2/commons/mongo"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/attribute"
)

// Repository provides an interface for operations related to reports collection in MongoDB.
//
//go:generate mockgen --destination=report.mongodb.mock.go --package=report . Repository
type Repository interface {
	FindByID(ctx context.Context, id, organizationID uuid.UUID) (*ExternalConnection, error)
}

// ReportMongoDBRepository is a MongoDB-specific implementation of the ReportRepository.
type ExternalConnectionMongoDBRepository struct {
	connection *libMongo.MongoConnection
	Database   string
}

// NewExternalConnectionMongoDBRepository returns a new instance of ExternalConnectionMongoDBRepository using the given MongoDB connection.
func NewExternalConnectionMongoDBRepository(mc *libMongo.MongoConnection) *ExternalConnectionMongoDBRepository {
	r := &ExternalConnectionMongoDBRepository{
		connection: mc,
		Database:   mc.Database,
	}
	if _, err := r.connection.GetDB(context.Background()); err != nil {
		panic("Failed to connect mongo")
	}

	return r
}

// FindByID retrieves a report from the mongodb using the provided entity_id.
func (rm *ExternalConnectionMongoDBRepository) FindByID(ctx context.Context, id, organizationID uuid.UUID) (*ExternalConnection, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_by_entity")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.report_id", id.String()),
		attribute.String("app.request.organization_id", organizationID.String()),
	}

	span.SetAttributes(attributes...)

	db, err := rm.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)

		return nil, err
	}

	coll := db.Database(strings.ToLower(rm.Database)).Collection(strings.ToLower(constant.MongoCollectionExternalConnection))

	var record *ExternalConnectionMongoDBModel

	ctx, spanFindOne := tracer.Start(ctx, "mongodb.find_by_entity.find_one")

	spanFindOne.SetAttributes(attributes...)

	if err = coll.
		FindOne(ctx, bson.M{"_id": id, "organization_id": organizationID, "deleted_at": bson.D{{Key: "$eq", Value: nil}}}).
		Decode(&record); err != nil {
		libOpentelemetry.HandleSpanError(&spanFindOne, "Failed to find report by entity", err)
		return nil, err
	}

	if nil == record {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, mongo.ErrNoDocuments
	}

	spanFindOne.End()

	return record.ToEntityFindByID(), nil
}
