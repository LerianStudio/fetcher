package job

import (
	"context"
	"errors"
	"regexp"
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
	"go.opentelemetry.io/otel/trace"
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
	UpdateStatus(ctx context.Context, id, organizationID uuid.UUID, status JobStatus, metadata map[string]any) error
	FindByID(ctx context.Context, id, organizationID uuid.UUID) (*Job, error)
	List(ctx context.Context, filters *ListFilter) ([]*Job, error)
	ExistsRunningByMappedFieldKey(ctx context.Context, organizationID uuid.UUID, keyPattern string) (bool, error)
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
func NewJobMongoDBRepository(mc *libMongo.MongoConnection) (*JobMongoDBRepository, error) {
	repo := &JobMongoDBRepository{
		connection: mc,
		Database:   mc.Database,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

// UpdateStatus updates only the status and metadata of a job, automatically managing CompletedAt.
func (jr *JobMongoDBRepository) UpdateStatus(ctx context.Context, id, organizationID uuid.UUID, status JobStatus, metadata map[string]any) error {
	_, tracer, reqId, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.update_job_status")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.job_id", id.String()),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.status", string(status)),
	}
	span.SetAttributes(attributes...)

	if !status.IsValid() {
		err := errors.New("invalid job status")
		libOpentelemetry.HandleSpanError(&span, "Invalid status", err)

		return err
	}

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return err
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))

	filter := bson.M{
		"_id":             id,
		"organization_id": organizationID,
	}

	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	// Automatically set CompletedAt for terminal statuses
	if status == JobStatusCompleted || status == JobStatusFailed {
		now := time.Now().UTC()
		update["$set"].(bson.M)["completed_at"] = now
	} else {
		// Clear CompletedAt for non-terminal statuses
		update["$unset"] = bson.M{
			"completed_at": "",
		}
	}

	// Update metadata if provided
	if metadata != nil {
		update["$set"].(bson.M)["metadata"] = metadata
	}

	ctx, spanUpdate := tracer.Start(ctx, "mongodb.update_job_status.update_one")
	defer spanUpdate.End()

	spanUpdate.SetAttributes(attributes...)

	if err := setSpanAttributesFromStruct(&spanUpdate, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(&spanUpdate, "Failed to convert filter to JSON", err)
	}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		libOpentelemetry.HandleSpanError(&spanUpdate, "Failed to update job status", err)
		return err
	}

	if result.MatchedCount == 0 {
		err := errors.New("job not found")
		libOpentelemetry.HandleSpanError(&spanUpdate, "Job not found", err)

		return err
	}

	return nil
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

// ExistsRunningByMappedFieldKey reports whether there is any running job (pending or processing)
// that contains the specified key in its mapped_fields document for the given organization.
func (jr *JobMongoDBRepository) ExistsRunningByMappedFieldKey(ctx context.Context, organizationID uuid.UUID, keyPattern string) (bool, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.exists_running_job_by_mapped_field_key")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.key_pattern", keyPattern),
	}
	span.SetAttributes(attributes...)

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return false, pkg.ValidateInternalError(err, "job")
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))

	// Validate keyPattern to prevent injection attacks
	configNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !configNameRegex.MatchString(keyPattern) {
		errInvalidKey := errors.New("invalid key pattern format")
		libOpentelemetry.HandleSpanError(&span, "Key pattern validation failed", errInvalidKey)
		return false, pkg.ValidateInternalError(errInvalidKey, "job")
	}

	// Build a filter that checks if the key exists in mapped_fields
	// MongoDB uses dot notation to check for key existence in nested documents
	mappedFieldKey := "mapped_fields." + keyPattern

	filter := bson.M{
		"organization_id": organizationID,
		"status": bson.M{
			"$in": bson.A{JobStatusPending, JobStatusProcessing},
		},
		mappedFieldKey: bson.M{
			"$exists": true,
		},
	}

	if errSpan := setSpanAttributesFromStruct(&span, "app.request.repository_filter", filter); errSpan != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filter to JSON", errSpan)
	}

	count, err := coll.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to count running jobs by mapped field key", err)
		return false, pkg.ValidateInternalError(err, "job")
	}

	return count > 0, nil
}

// List returns a paginated set of jobs for the given organization.
func (jr *JobMongoDBRepository) List(ctx context.Context, filters *ListFilter) ([]*Job, error) {
	if filters == nil {
		filters = &ListFilter{}
	}

	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.list_jobs")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", filters.OrganizationID.String()),
	}
	span.SetAttributes(attributes...)

	db, err := jr.connection.GetDB(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to get database", err)
		return nil, err
	}

	coll := db.Database(strings.ToLower(jr.Database)).Collection(strings.ToLower(constant.MongoCollectionJob))

	queryFilter := jr.buildQueryFilter(filters)
	opts := jr.buildPaginationOptions(filters)

	if err := setSpanAttributesFromStruct(&span, "app.request.repository_filter", queryFilter); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert list filter to JSON", err)
	}

	cur, err := coll.Find(ctx, queryFilter, &opts)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to list jobs", err)
		return nil, err
	}
	defer cur.Close(ctx)

	jobs, err := jr.scanJobs(ctx, cur, &span, int(*opts.Limit))
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

// buildQueryFilter builds the MongoDB query filter from filters
func (jr *JobMongoDBRepository) buildQueryFilter(filters *ListFilter) bson.M {
	queryFilter := bson.M{
		"organization_id": filters.OrganizationID,
	}

	jr.addStatusFilter(queryFilter, filters)
	jr.addDateRangeFilter(queryFilter, filters)

	return queryFilter
}

// addStatusFilter adds status filter to the query
func (jr *JobMongoDBRepository) addStatusFilter(queryFilter bson.M, filters *ListFilter) {
	if len(filters.Statuses) > 0 {
		queryFilter["status"] = bson.M{"$in": filters.Statuses}
	} else if filters.Status != "" {
		queryFilter["status"] = filters.Status
	}
}

// addDateRangeFilter adds date range filters to the query
func (jr *JobMongoDBRepository) addDateRangeFilter(queryFilter bson.M, filters *ListFilter) {
	createdRange := jr.buildDateRange(filters.CreatedFrom, filters.CreatedTo)
	if len(createdRange) > 0 {
		queryFilter["created_at"] = createdRange
	}

	completedRange := jr.buildDateRange(filters.CompletedFrom, filters.CompletedTo)
	if len(completedRange) > 0 {
		queryFilter["completed_at"] = completedRange
	}
}

// buildDateRange builds a date range filter
func (jr *JobMongoDBRepository) buildDateRange(from, to *time.Time) bson.M {
	dateRange := bson.M{}

	if from != nil {
		dateRange["$gte"] = *from
	}

	if to != nil {
		dateRange["$lte"] = *to
	}

	return dateRange
}

// buildPaginationOptions builds MongoDB pagination options
func (jr *JobMongoDBRepository) buildPaginationOptions(filters *ListFilter) options.FindOptions {
	limit := jr.calculateLimit(filters.Limit)
	page := jr.calculatePage(filters.Page)
	skip := int64((page - 1) * limit)
	limit64 := int64(limit)
	sortDirection := jr.calculateSortDirection(filters.SortOrder)

	return options.FindOptions{
		Limit: &limit64,
		Skip:  &skip,
		Sort:  bson.D{{Key: "created_at", Value: sortDirection}},
	}
}

// calculateLimit calculates and validates the limit
func (jr *JobMongoDBRepository) calculateLimit(limit int) int {
	if limit <= 0 {
		return defaultJobPageLimit
	}

	if limit > maxJobPageLimit {
		return maxJobPageLimit
	}

	return limit
}

// calculatePage calculates and validates the page number
func (jr *JobMongoDBRepository) calculatePage(page int) int {
	if page <= 0 {
		return 1
	}

	return page
}

// calculateSortDirection calculates the sort direction
func (jr *JobMongoDBRepository) calculateSortDirection(sortOrder constant.Order) int32 {
	if strings.EqualFold(string(sortOrder), string(constant.Asc)) {
		return 1
	}

	return -1
}

// scanJobs scans job records from the cursor
func (jr *JobMongoDBRepository) scanJobs(ctx context.Context, cur *mongo.Cursor, span *trace.Span, limit int) ([]*Job, error) {
	jobs := make([]*Job, 0, limit)

	for cur.Next(ctx) {
		var record JobMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to decode job record", err)
			return nil, err
		}

		jobs = append(jobs, record.ToEntity())
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to iterate over jobs", err)
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
