package job

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/lib-commons/v2/commons"
	libMongo "github.com/LerianStudio/lib-commons/v2/commons/mongo"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

const (
	defaultJobPageLimit = 10
	maxJobPageLimit     = 50
)

// ListFilter controls pagination and filtering for job listings.
type ListFilter struct {
	OrganizationID uuid.UUID
	Status         JobStatus
	Statuses       []JobStatus
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	CompletedFrom  *time.Time
	CompletedTo    *time.Time
	Limit          int
	Page           int
	SortOrder      constant.Order
}

// Repository defines the MongoDB contract for the jobs collection.
//
//go:generate mockgen --destination=job.mongodb.mock.go --package=job . Repository
type Repository interface {
	Create(ctx context.Context, job *Job) (*Job, error)
	Update(ctx context.Context, job *Job) (*Job, error)
	FindByID(ctx context.Context, id, organizationID uuid.UUID) (*Job, error)
	List(ctx context.Context, filters *ListFilter) ([]*Job, error)
	ExistsRunningByConnection(ctx context.Context, organizationID, connectionID uuid.UUID) (bool, error)
}

type mongoDatabaseProvider interface {
	GetDB(ctx context.Context) (*mongo.Client, error)
}

var setSpanAttributesFromStruct = libOpentelemetry.SetSpanAttributesFromStruct

// JobMongoDBRepository implements Repository backed by MongoDB.
type JobMongoDBRepository struct {
	connection mongoDatabaseProvider
	Database   string
}

// NewJobMongoDBRepository provisions a repository using the given client.
func NewJobMongoDBRepository(ctx context.Context, mc *libMongo.MongoConnection) (*JobMongoDBRepository, error) {
	repo := &JobMongoDBRepository{
		connection: mc,
		Database:   mc.Database,
	}

	if _, err := repo.connection.GetDB(ctx); err != nil {
		return nil, err
	}

	return repo, nil
}

// Create inserts a new job document.
func (jr *JobMongoDBRepository) Create(ctx context.Context, job *Job) (*Job, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.create_job")
	defer span.End()

	if job == nil {
		err := errors.New("job is required")
		libOpentelemetry.HandleSpanError(&span, "Job payload is nil", err)
		return nil, err
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
	}

	if job.OrganizationID != uuid.Nil {
		attributes = append(attributes, attribute.String("app.request.organization_id", job.OrganizationID.String()))
	}
	if job.ID != uuid.Nil {
		attributes = append(attributes, attribute.String("app.request.job_id", job.ID.String()))
	}
	if job.ConnectionID != uuid.Nil {
		attributes = append(attributes, attribute.String("app.request.connection_id", job.ConnectionID.String()))
	}

	span.SetAttributes(attributes...)

	if err := job.ValidateForCreate(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Invalid job payload", err)
		return nil, err
	}

	if err := setSpanAttributesFromStruct(&span, "app.request.payload", job); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert job payload to JSON", err)
	}

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))
	record := &JobMongoDBModel{}

	if err := record.FromEntity(job); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert entity to MongoDB model", err)
		return nil, err
	}

	ctx, spanInsert := tracer.Start(ctx, "mongodb.create_job.insert")
	defer spanInsert.End()

	spanInsert.SetAttributes(attributes...)
	spanInsert.SetAttributes(attribute.String("app.request.job_id", record.ID.String()))

	if err := setSpanAttributesFromStruct(&spanInsert, "app.request.repository_input", NewJobTelemetryFromMongoDBModel(record)); err != nil {
		libOpentelemetry.HandleSpanError(&spanInsert, "Failed to convert record to JSON", err)
	}

	if _, err := coll.InsertOne(ctx, record); err != nil {
		libOpentelemetry.HandleSpanError(&spanInsert, "Failed to insert job", err)
		return nil, err
	}

	return record.ToEntity(), nil
}

// Update overwrites mutable fields of an existing job and returns the saved entity.
func (jr *JobMongoDBRepository) Update(ctx context.Context, job *Job) (*Job, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.update_job")
	defer span.End()

	if job == nil {
		err := errors.New("job is required")
		libOpentelemetry.HandleSpanError(&span, "Job payload is nil", err)
		return nil, err
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.job_id", job.ID.String()),
		attribute.String("app.request.organization_id", job.OrganizationID.String()),
		attribute.String("app.request.connection_id", job.ConnectionID.String()),
	}
	span.SetAttributes(attributes...)

	if err := job.ValidateForUpdate(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Invalid job payload", err)
		return nil, err
	}

	if job.Status == JobStatusCompleted || job.Status == JobStatusFailed {
		if job.CompletedAt == nil {
			now := time.Now().UTC()
			job.CompletedAt = &now
		}
	} else {
		job.CompletedAt = nil
	}

	if err := setSpanAttributesFromStruct(&span, "app.request.payload", job); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert job payload to JSON", err)
	}

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))
	filter := bson.M{
		"_id":             job.ID,
		"organization_id": job.OrganizationID,
		"connection_id":   job.ConnectionID,
	}

	update := bson.M{
		"$set": bson.M{
			"metadata":      job.Metadata,
			"mapped_fields": job.MappedFields,
			"filters":       job.Filters,
			"status":        job.Status,
			"result_path":   job.ResultPath,
			"completed_at":  job.CompletedAt,
		},
	}

	ctx, spanUpdate := tracer.Start(ctx, "mongodb.update_job.find_one_and_update")
	defer spanUpdate.End()

	spanUpdate.SetAttributes(attributes...)
	if err := setSpanAttributesFromStruct(&spanUpdate, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(&spanUpdate, "Failed to convert filter to JSON", err)
	}

	var record JobMongoDBModel
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	if err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&record); err != nil {
		libOpentelemetry.HandleSpanError(&spanUpdate, "Failed to update job", err)
		return nil, err
	}

	return record.ToEntity(), nil
}

// FindByID fetches a job by its ID scoped to an organization.
func (jr *JobMongoDBRepository) FindByID(ctx context.Context, id, organizationID uuid.UUID) (*Job, error) {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_job_by_id")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.job_id", id.String()),
		attribute.String("app.request.organization_id", organizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))

	var record JobMongoDBModel
	filter := bson.M{
		"_id":             id,
		"organization_id": organizationID,
	}

	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to find job", err)
		return nil, err
	}

	return record.ToEntity(), nil
}

// ExistsRunningByConnection reports whether there is any running job for a connection (pending or processing).
func (jr *JobMongoDBRepository) ExistsRunningByConnection(ctx context.Context, organizationID, connectionID uuid.UUID) (bool, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.exists_running_job_by_connection")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.connection_id", connectionID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return false, pkg.ValidateInternalError(err, "connection")
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))

	filter := bson.M{
		"organization_id": organizationID,
		"connection_id":   connectionID,
		"status": bson.M{
			"$in": bson.A{JobStatusPending, JobStatusProcessing},
		},
	}

	if errSpan := setSpanAttributesFromStruct(&span, "app.request.repository_filter", filter); errSpan != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filter to JSON", errSpan)
	}

	count, err := coll.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to count running jobs", err)
		return false, pkg.ValidateInternalError(err, "connection")
	}

	return count > 0, nil
}

// List returns a paginated set of jobs for the given organization.
func (jr *JobMongoDBRepository) List(ctx context.Context, filters *ListFilter) ([]*Job, error) {
	if filters == nil {
		filters = &ListFilter{}
	}

	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.list_jobs")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.organization_id", filters.OrganizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))

	queryFilter := bson.M{
		"organization_id": filters.OrganizationID,
	}

	if len(filters.Statuses) > 0 {
		queryFilter["status"] = bson.M{"$in": filters.Statuses}
	} else if filters.Status != "" {
		queryFilter["status"] = filters.Status
	}

	createdRange := bson.M{}
	if filters.CreatedFrom != nil {
		createdRange["$gte"] = *filters.CreatedFrom
	}
	if filters.CreatedTo != nil {
		createdRange["$lte"] = *filters.CreatedTo
	}
	if len(createdRange) > 0 {
		queryFilter["created_at"] = createdRange
	}

	completedRange := bson.M{}
	if filters.CompletedFrom != nil {
		completedRange["$gte"] = *filters.CompletedFrom
	}
	if filters.CompletedTo != nil {
		completedRange["$lte"] = *filters.CompletedTo
	}
	if len(completedRange) > 0 {
		queryFilter["completed_at"] = completedRange
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = defaultJobPageLimit
	}
	if limit > maxJobPageLimit {
		limit = maxJobPageLimit
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}

	skip := int64((page - 1) * limit)
	limit64 := int64(limit)
	sortDirection := int32(-1)
	if strings.EqualFold(string(filters.SortOrder), string(constant.Asc)) {
		sortDirection = 1
	}

	opts := options.FindOptions{
		Limit: &limit64,
		Skip:  &skip,
		Sort:  bson.D{{Key: "created_at", Value: sortDirection}},
	}

	if err := setSpanAttributesFromStruct(&span, "app.request.repository_filter", queryFilter); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert list filter to JSON", err)
	}

	cur, err := coll.Find(ctx, queryFilter, &opts)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to list jobs", err)
		return nil, err
	}
	defer cur.Close(ctx)

	jobs := make([]*Job, 0, limit)
	for cur.Next(ctx) {
		var record JobMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to decode job record", err)
			return nil, err
		}
		jobs = append(jobs, record.ToEntity())
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to iterate over jobs", err)
		return nil, err
	}

	return jobs, nil
}

// JobTelemetry models a job for telemetry without large nested data.
type JobTelemetry struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	Status         JobStatus `json:"status"`
	ResultPath     string    `json:"resultPath,omitempty"`
}

// NewJobTelemetryFromMongoDBModel creates a telemetry-friendly struct from the MongoDB model.
func NewJobTelemetryFromMongoDBModel(job *JobMongoDBModel) *JobTelemetry {
	if job == nil {
		return nil
	}

	return &JobTelemetry{
		ID:             job.ID,
		OrganizationID: job.OrganizationID,
		Status:         job.Status,
		ResultPath:     job.ResultPath,
	}
}
