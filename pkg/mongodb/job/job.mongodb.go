package job

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/mongodb"
	portsJob "github.com/LerianStudio/fetcher/pkg/ports/job"

	"github.com/LerianStudio/lib-commons/v4/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

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

	// DefaultInitTimeout is the default timeout for repository initialization
	DefaultInitTimeout = 10 * time.Second
)

// Repository is an alias for the domain port interface defined in pkg/ports/job.
type Repository = portsJob.Repository

//go:generate mockgen --destination=mock_db_provider_test.go --package=job . mongoDatabaseProvider
type mongoDatabaseProvider interface {
	Client(ctx context.Context) (*mongo.Client, error)
}

var setSpanAttributesFromValue = func(span trace.Span, key string, value any) error {
	return libOpentelemetry.SetSpanAttributesFromValue(span, key, value, nil)
}

// RepositoryConfig holds configuration options for the repository.
type RepositoryConfig struct {
	// InitTimeout is the timeout for repository initialization.
	// Default: DefaultInitTimeout (10s)
	InitTimeout time.Duration
}

// JobMongoDBRepository implements Repository backed by MongoDB.
// NOTE: Span names in this file use the pattern "mongodb.verb_entity" (e.g., "mongodb.create_job").
// The preferred convention is "mongodb.entity.operation" (e.g., "mongodb.job.create").
// This inconsistency is tracked for a future rename when dashboards and alerts can be updated.
type JobMongoDBRepository struct {
	connection mongoDatabaseProvider
	Database   string
	config     RepositoryConfig
}

// NewJobMongoDBRepository provisions a repository using the given client.
// Accepts an optional RepositoryConfig; if nil, defaults are used.
// The provider must implement Client(ctx) (*mongo.Client, error). When the provider
// also implements tmcore.MultiTenantChecker (IsMultiTenant() bool), ResolveMongo will
// return ErrTenantContextRequired instead of silently falling back to the default DB.
func NewJobMongoDBRepository(ctx context.Context, provider mongodb.MongoClientProvider, dbName string, cfg ...RepositoryConfig) (*JobMongoDBRepository, error) {
	config := RepositoryConfig{
		InitTimeout: DefaultInitTimeout,
	}
	if len(cfg) > 0 {
		config = cfg[0]
		if config.InitTimeout <= 0 {
			config.InitTimeout = DefaultInitTimeout
		}
	}

	repo := &JobMongoDBRepository{
		connection: provider,
		Database:   dbName,
		config:     config,
	}

	ctx, cancel := context.WithTimeout(ctx, config.InitTimeout)
	defer cancel()

	if _, err := repo.connection.Client(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return repo, nil
}

// getDatabase returns a *mongo.Database for the current request context.
// In multi-tenant mode, it retrieves the tenant-specific database from context
// via tmcore.GetMongoForTenant. In single-tenant mode (no tenant in context),
// it falls back to the static connection using jr.connection.Client.
func (jr *JobMongoDBRepository) getDatabase(ctx context.Context) (*mongo.Database, error) {
	return mongodb.ResolveDatabase(ctx, jr.connection, jr.Database)
}

// Create inserts a new job document.
func (jr *JobMongoDBRepository) Create(ctx context.Context, job *model.Job) (*model.Job, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.create_job")
	defer span.End()

	if job == nil {
		err := errors.New("job is required")
		libOpentelemetry.HandleSpanError(span, "Job payload is nil", err)

		return nil, err
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
	}

	if job.ID != uuid.Nil {
		attributes = append(attributes, attribute.String("app.request.job_id", job.ID.String()))
	}

	span.SetAttributes(attributes...)

	if err := setSpanAttributesFromValue(span, "app.request.payload", job); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert job payload to JSON", err)
	}

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))
	record := &JobMongoDBModel{}

	if err := record.FromEntity(job); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert entity to MongoDB model", err)
		return nil, fmt.Errorf("failed to convert job entity to model: %w", err)
	}

	ctx, spanInsert := tracer.Start(ctx, "mongodb.create_job.insert")
	defer spanInsert.End()

	spanInsert.SetAttributes(attributes...)
	spanInsert.SetAttributes(attribute.String("app.request.job_id", record.ID.String()))

	if err := setSpanAttributesFromValue(spanInsert, "app.request.repository_input", record); err != nil {
		libOpentelemetry.HandleSpanError(spanInsert, "Failed to convert record to JSON", err)
	}

	if _, err := coll.InsertOne(ctx, record); err != nil {
		libOpentelemetry.HandleSpanError(spanInsert, "Failed to insert job", err)
		return nil, fmt.Errorf("failed to insert job: %w", err)
	}

	job, err = record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert MongoDB model to entity", err)
		return nil, pkg.ValidateInternalError(err, "job")
	}

	return job, nil
}

// Update overwrites mutable fields of an existing job and returns the saved entity.
func (jr *JobMongoDBRepository) Update(ctx context.Context, job *model.Job) (*model.Job, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.update_job")
	defer span.End()

	if job == nil {
		err := errors.New("job is required")
		libOpentelemetry.HandleSpanError(span, "Job payload is nil", err)

		return nil, err
	}

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.job_id", job.ID.String()),
	}
	span.SetAttributes(attributes...)

	if err := setSpanAttributesFromValue(span, "app.request.payload", job); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert job payload to JSON", err)
	}

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))
	filter := bson.M{
		"_id": job.ID,
	}

	update := bson.M{
		"$set": bson.M{
			"metadata":      job.Metadata,
			"mapped_fields": job.MappedFields,
			"filters":       job.Filters,
			"status":        job.Status,
			"result_path":   job.ResultPath,
			"result_hmac":   job.ResultHMAC,
			"completed_at":  job.CompletedAt,
		},
	}

	ctx, spanUpdate := tracer.Start(ctx, "mongodb.update_job.find_one_and_update")
	defer spanUpdate.End()

	spanUpdate.SetAttributes(attributes...)

	if err := setSpanAttributesFromValue(spanUpdate, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(spanUpdate, "Failed to convert filter to JSON", err)
	}

	var record JobMongoDBModel

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	if err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&record); err != nil {
		libOpentelemetry.HandleSpanError(spanUpdate, "Failed to update job", err)
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	job, err = record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert MongoDB model to entity", err)
		return nil, pkg.ValidateInternalError(err, "job")
	}

	return job, nil
}

// UpdateStatus updates only the status, resultPath, resultHMAC and metadata of a job, automatically managing CompletedAt.
func (jr *JobMongoDBRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.update_job_status")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.job_id", id.String()),
		attribute.String("app.request.status", string(status)),
	}
	span.SetAttributes(attributes...)

	if !status.IsValid() {
		err := errors.New("invalid job status")
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Invalid status", err)

		return err
	}

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))

	filter := bson.M{
		"_id": id,
	}

	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	// Automatically set CompletedAt for terminal statuses
	if status == model.JobStatusCompleted || status == model.JobStatusFailed {
		now := time.Now().UTC()
		update["$set"].(bson.M)["completed_at"] = now
	} else {
		// Clear CompletedAt for non-terminal statuses
		update["$unset"] = bson.M{
			"completed_at": "",
		}
	}

	// Update resultPath if provided
	if resultPath != "" {
		update["$set"].(bson.M)["result_path"] = resultPath
	}

	// Update resultHMAC if provided
	if resultHMAC != "" {
		update["$set"].(bson.M)["result_hmac"] = resultHMAC
	}

	// Merge metadata fields using dot-notation so individual keys are set
	// without replacing the entire metadata document. This preserves existing
	// metadata fields (e.g. "source") when adding error information.
	for k, v := range metadata {
		if strings.Contains(k, ".") || strings.HasPrefix(k, "$") {
			err := fmt.Errorf("invalid metadata key %q: must not contain '.' or start with '$'", k)
			libOpentelemetry.HandleSpanError(span, "Invalid metadata key", err)

			return err
		}

		update["$set"].(bson.M)["metadata."+k] = v
	}

	ctx, spanUpdate := tracer.Start(ctx, "mongodb.update_job_status.update_one")
	defer spanUpdate.End()

	spanUpdate.SetAttributes(attributes...)

	if err := setSpanAttributesFromValue(spanUpdate, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(spanUpdate, "Failed to convert filter to JSON", err)
	}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		libOpentelemetry.HandleSpanError(spanUpdate, "Failed to update job status", err)
		return fmt.Errorf("failed to update job status: %w", err)
	}

	if result.MatchedCount == 0 {
		err := errors.New("job not found")
		libOpentelemetry.HandleSpanBusinessErrorEvent(spanUpdate, "Job not found", err)

		return err
	}

	return nil
}

// FindByID fetches a job by its ID scoped to an organization.
func (jr *JobMongoDBRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Job, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_job_by_id")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.job_id", id.String()),
	}
	span.SetAttributes(attributes...)

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))

	var record JobMongoDBModel

	filter := bson.M{
		"_id": id,
	}

	if err := coll.FindOne(ctx, filter).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(span, "Failed to find job", err)

		return nil, fmt.Errorf("failed to find job by id: %w", err)
	}

	job, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert MongoDB model to entity", err)
		return nil, pkg.ValidateInternalError(err, "job")
	}

	return job, nil
}

// FindByRequestHashWithinWindow finds the most recent job with the given request hash
// created within the specified time window (in minutes) for deduplication purposes.
// Returns nil without error if no matching job is found.
func (jr *JobMongoDBRepository) FindByRequestHashWithinWindow(ctx context.Context, requestHash string, windowMinutes int) (*model.Job, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_job_by_request_hash_within_window")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.request_hash", requestHash),
		attribute.Int("app.request.window_minutes", windowMinutes),
	}
	span.SetAttributes(attributes...)

	if requestHash == "" {
		return nil, nil
	}

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))

	windowStart := time.Now().UTC().Add(-time.Duration(windowMinutes) * time.Minute)

	filter := bson.M{
		"request_hash": requestHash,
		"created_at": bson.M{
			"$gte": windowStart,
		},
	}

	if err := setSpanAttributesFromValue(span, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert filter to JSON", err)
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})

	var record JobMongoDBModel
	if err := coll.FindOne(ctx, filter, opts).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(span, "Failed to find job by request hash", err)

		return nil, fmt.Errorf("failed to find job by request hash: %w", err)
	}

	job, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert MongoDB model to entity", err)
		return nil, pkg.ValidateInternalError(err, "job")
	}

	return job, nil
}

// FindActiveByRequestHash finds the most recent active job (pending or processing)
// for a request hash in an organization. Returns nil without error when no active
// matching job exists.
func (jr *JobMongoDBRepository) FindActiveByRequestHash(ctx context.Context, requestHash string) (*model.Job, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.find_active_job_by_request_hash")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.request_hash", requestHash),
	}
	span.SetAttributes(attributes...)

	if requestHash == "" {
		return nil, nil
	}

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))

	filter := bson.M{
		"request_hash": requestHash,
		"status": bson.M{
			"$in": bson.A{model.JobStatusPending, model.JobStatusProcessing},
		},
	}

	if err := setSpanAttributesFromValue(span, "app.request.repository_filter", filter); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert filter to JSON", err)
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})

	var record JobMongoDBModel
	if err := coll.FindOne(ctx, filter, opts).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		libOpentelemetry.HandleSpanError(span, "Failed to find active job by request hash", err)

		return nil, fmt.Errorf("failed to find active job by request hash: %w", err)
	}

	job, err := record.ToEntity()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert MongoDB model to entity", err)
		return nil, pkg.ValidateInternalError(err, "job")
	}

	return job, nil
}

// ExistsRunningByMappedFieldKey reports whether there is any running job (pending or processing)
// that contains the specified key in its mapped_fields document for the given organization.
func (jr *JobMongoDBRepository) ExistsRunningByMappedFieldKey(ctx context.Context, keyPattern string) (bool, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.exists_running_job_by_mapped_field_key")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.key_pattern", keyPattern),
	}
	span.SetAttributes(attributes...)

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return false, pkg.ValidateInternalError(err, "job")
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))

	// Validate keyPattern to prevent injection attacks
	configNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !configNameRegex.MatchString(keyPattern) {
		errInvalidKey := errors.New("invalid key pattern format")
		libOpentelemetry.HandleSpanError(span, "Key pattern validation failed", errInvalidKey)

		return false, pkg.ValidateInternalError(errInvalidKey, "job")
	}

	// Build a filter that checks if the key exists in mapped_fields
	// MongoDB uses dot notation to check for key existence in nested documents
	mappedFieldKey := "mapped_fields." + keyPattern

	filter := bson.M{
		"status": bson.M{
			"$in": bson.A{model.JobStatusPending, model.JobStatusProcessing},
		},
		mappedFieldKey: bson.M{
			"$exists": true,
		},
	}

	if errSpan := setSpanAttributesFromValue(span, "app.request.repository_filter", filter); errSpan != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert filter to JSON", errSpan)
	}

	count, err := coll.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to count running jobs by mapped field key", err)
		return false, pkg.ValidateInternalError(err, "job")
	}

	return count > 0, nil
}

// List returns a paginated set of jobs for the given organization.
func (jr *JobMongoDBRepository) List(ctx context.Context, filters *ListFilter) ([]*model.Job, error) {
	if filters == nil {
		filters = &ListFilter{}
	}

	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "mongodb.list_jobs")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.String("app.request.request_id", reqID),
	}
	span.SetAttributes(attributes...)

	db, err := jr.getDatabase(ctx)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to get database", err)
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	coll := db.Collection(strings.ToLower(constant.MongoCollectionJob))

	queryFilter := jr.buildQueryFilter(filters)
	opts := jr.buildPaginationOptions(filters)

	if err := setSpanAttributesFromValue(span, "app.request.repository_filter", queryFilter); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert list filter to JSON", err)
	}

	cur, err := coll.Find(ctx, queryFilter, &opts)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to list jobs", err)
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer cur.Close(ctx)

	jobs, err := jr.scanJobs(ctx, cur, span, int(*opts.Limit))
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

// buildQueryFilter builds the MongoDB query filter from filters
func (jr *JobMongoDBRepository) buildQueryFilter(filters *ListFilter) bson.M {
	queryFilter := bson.M{}

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
func (jr *JobMongoDBRepository) scanJobs(ctx context.Context, cur *mongo.Cursor, span trace.Span, limit int) ([]*model.Job, error) {
	jobs := make([]*model.Job, 0, limit)

	for cur.Next(ctx) {
		var record JobMongoDBModel
		if err := cur.Decode(&record); err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to decode job record", err)
			return nil, fmt.Errorf("failed to decode job record: %w", err)
		}

		job, err := record.ToEntity()
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to convert MongoDB model to entity", err)
			return nil, pkg.ValidateInternalError(err, "job")
		}

		jobs = append(jobs, job)
	}

	if err := cur.Err(); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to iterate over jobs", err)
		return nil, fmt.Errorf("failed to iterate over jobs: %w", err)
	}

	return jobs, nil
}
