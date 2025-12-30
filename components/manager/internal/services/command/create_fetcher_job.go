package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	"github.com/LerianStudio/fetcher/pkg/rabbitmq"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// DeduplicationWindowMinutes is the time window for request deduplication.
	DeduplicationWindowMinutes = 5

	// ExtractExternalDataQueue is the RabbitMQ queue name for extraction jobs.
	ExtractExternalDataQueue = "extract-external-data-queue"

	// ConnectionTestTimeout is the timeout for testing connections.
	ConnectionTestTimeout = 10 * time.Second
)

// CreateFetcherJobResult contains the result of creating a fetcher job.
type CreateFetcherJobResult struct {
	Job          *model.Job
	IsDuplicate  bool
	IsNewCreated bool
}

// ConnectionTester defines the interface for testing database connections.
// This interface allows mocking connection tests in unit tests.
//
//go:generate mockgen --destination=connection_tester.mock.go --package=command . ConnectionTester
type ConnectionTester interface {
	TestConnection(ctx context.Context, conn *model.Connection) error
}

// CreateFetcherJob is the command service for creating fetcher jobs.
type CreateFetcherJob struct {
	connRepo         connRepo.Repository
	jobRepo          jobRepo.Repository
	cryptor          crypto.Cryptor
	rabbitMQ         *rabbitmq.RabbitMQAdapter
	connectionTester ConnectionTester
}

// NewCreateFetcherJob creates a new CreateFetcherJob service.
func NewCreateFetcherJob(
	connectionRepo connRepo.Repository,
	jobRepository jobRepo.Repository,
	cryptor crypto.Cryptor,
	rabbitMQ *rabbitmq.RabbitMQAdapter,
) *CreateFetcherJob {
	svc := &CreateFetcherJob{
		connRepo: connectionRepo,
		jobRepo:  jobRepository,
		cryptor:  cryptor,
		rabbitMQ: rabbitMQ,
	}
	// Use default connection tester that tests real connections
	svc.connectionTester = svc

	return svc
}

// NewCreateFetcherJobWithTester creates a new CreateFetcherJob service with a custom connection tester.
// This is useful for testing where you want to mock connection testing.
func NewCreateFetcherJobWithTester(
	connectionRepo connRepo.Repository,
	jobRepository jobRepo.Repository,
	cryptor crypto.Cryptor,
	rabbitMQ *rabbitmq.RabbitMQAdapter,
	tester ConnectionTester,
) *CreateFetcherJob {
	svc := &CreateFetcherJob{
		connRepo:         connectionRepo,
		jobRepo:          jobRepository,
		cryptor:          cryptor,
		rabbitMQ:         rabbitMQ,
		connectionTester: tester,
	}
	if tester == nil {
		svc.connectionTester = svc
	}

	return svc
}

// Execute creates a new fetcher job or returns an existing duplicate.
//
//nolint:gocyclo // High complexity is inherent to job creation orchestration with multiple validation steps
func (s *CreateFetcherJob) Execute(ctx context.Context, organizationID uuid.UUID, request model.FetcherRequest) (*CreateFetcherJobResult, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.create_fetcher_job")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
	)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", request)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert fetcher request to JSON string", err)
	}

	// Compute request hash for idempotency
	requestHash, err := request.ComputeRequestHash()
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to compute request hash", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	span.SetAttributes(attribute.String("app.request.request_hash", requestHash))

	newJob, err := model.NewJob(
		organizationID,
		request.Metadata,
		request.DataRequest.MappedFields,
		func() []model.Filter {
			filters := []model.Filter{}
			for _, f := range request.DataRequest.Filters {
				filters = append(filters, model.Filter(f))
			}

			return filters
		}(),
		model.JobStatusPending, // Initial status is PENDING
		"",                     // ResultPath is empty initially
		requestHash,
		time.Now().UTC(),
		nil, // CompletedAt is nil initially
	)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to create job entity", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	// Validate the job entity
	if errValidation := newJob.IsValid(); errValidation != nil {
		libOpentelemetry.HandleSpanError(&span, "Invalid request payload", errValidation)
		return nil, errValidation
	}

	// Validate filter references against mappedFields
	if len(newJob.Filters) > 0 {
		if errFilters := model.ValidateFilterReferences(newJob.Filters, newJob.MappedFields); errFilters != nil {
			libOpentelemetry.HandleSpanError(&span, "Invalid filter references", errFilters)

			return nil, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrInvalidDataRequest.Error(),
				Title:      "Invalid Filter References",
				Message:    errFilters.Error(),
				Err:        errFilters,
			}
		}
	}

	// Check for duplicate within deduplication window
	existingJob, err := s.jobRepo.FindByRequestHashWithinWindow(ctx, newJob.OrganizationID, newJob.RequestHash, DeduplicationWindowMinutes)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to check for duplicate job", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	if existingJob != nil {
		logger.Infof("Duplicate request detected, returning existing job id=%s", existingJob.ID)
		span.SetAttributes(
			attribute.Bool("app.request.is_duplicate", true),
			attribute.String("app.request.existing_job_id", existingJob.ID.String()),
		)

		return &CreateFetcherJobResult{
			Job:          existingJob,
			IsDuplicate:  true,
			IsNewCreated: false,
		}, nil
	}

	// Validate all referenced connections exist and are UP (test each connection)
	connections, err := s.connRepo.FindByConfigNames(ctx, organizationID, newJob.GetDatasourceNames())
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to find connections", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	// No connections found
	if len(connections) == 0 {
		libOpentelemetry.HandleSpanError(&span, "No connections found for the provided datasources", nil)

		return nil, pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrMissingDataSource.Error(),
			Title:      "No Connections Found",
			Message:    "No connections configured for the requested datasources",
		}
	}

	// Check that all datasources have corresponding connections
	connMap := make(map[string]*model.Connection, len(connections))
	for _, conn := range connections {
		connMap[conn.ConfigName] = conn
	}

	for _, dsName := range newJob.GetDatasourceNames() {
		if _, found := connMap[dsName]; !found {
			err := fmt.Errorf("connection not found for datasource: %s", dsName)
			libOpentelemetry.HandleSpanError(&span, "Connection not found", err)

			return nil, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrMissingDataSource.Error(),
				Title:      "Missing Data Source",
				Message:    fmt.Sprintf("No connection configured for datasource '%s'", dsName),
			}
		}
	}

	// Test each connection to verify they are UP
	for _, conn := range connections {
		if err := s.connectionTester.TestConnection(ctx, conn); err != nil {
			libOpentelemetry.HandleSpanError(&span, fmt.Sprintf("Connection test failed for %s", conn.ConfigName), err)

			return nil, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrConnectionDown.Error(),
				Title:      "Connection Down",
				Message:    fmt.Sprintf("Connection '%s' is not available: %s", conn.ConfigName, err.Error()),
			}
		}
	}

	createdJob, err := s.jobRepo.Create(ctx, newJob)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to create job", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	span.SetAttributes(attribute.String("app.request.created_job_id", createdJob.ID.String()))
	logger.Infof("Created fetcher job id=%s org=%s", createdJob.ID, organizationID)

	if err := s.publishToQueue(ctx, createdJob); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to publish job to queue", err)
		logger.Errorf("Failed to publish job to queue id=%s: %v", createdJob.ID, err)

		createdJob.SetFailedStatus("process failed: unable to publish")

		_, updateErr := s.jobRepo.Update(ctx, createdJob)
		if updateErr != nil {
			libOpentelemetry.HandleSpanError(&span, "Failed to update job status to FAILED", updateErr)
			logger.Errorf("Failed to update job status to FAILED for job id=%s: %v", createdJob.ID, updateErr)
		}

		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	return &CreateFetcherJobResult{
		Job:          createdJob,
		IsDuplicate:  false,
		IsNewCreated: true,
	}, nil
}

// TestConnection tests if a connection is available.
// This method implements the ConnectionTester interface.
func (s *CreateFetcherJob) TestConnection(ctx context.Context, conn *model.Connection) error {
	logger, _, _, _ := commons.NewTrackingFromContext(ctx)

	testCtx, cancel := context.WithTimeout(ctx, ConnectionTestTimeout)
	defer cancel()

	ds, err := datasource.NewDataSourceFromConnection(testCtx, conn, s.cryptor, logger)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	// Close the connection after test
	if err := ds.Close(testCtx); err != nil {
		logger.Warnf("Failed to close test connection for %s: %v", conn.ConfigName, err)
	}

	return nil
}

// publishToQueue publishes a job to the RabbitMQ queue.
// If RabbitMQ is not configured (nil), this method does nothing and returns nil.
func (s *CreateFetcherJob) publishToQueue(ctx context.Context, j *model.Job) error {
	if s.rabbitMQ == nil {
		return nil
	}

	message := map[string]any{
		"jobId":          j.ID.String(),
		"organizationId": j.OrganizationID.String(),
		"mappedFields":   j.MappedFields,
		"metadata":       j.Metadata,
		"createdAt":      j.CreatedAt,
	}

	// Transform filters from array format to the nested map format expected by Worker:
	// map[datasource]map[table]map[field]FilterCondition
	if len(j.Filters) > 0 {
		transformedFilters := s.transformFiltersForWorker(j.Filters, j.MappedFields)
		if len(transformedFilters) > 0 {
			message["filters"] = transformedFilters
		}
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal job message: %w", err)
	}

	header := map[string]any{
		"jobId":          j.ID.String(),
		"organizationId": j.OrganizationID.String(),
	}

	return s.rabbitMQ.ProducerDefault(ctx, "", ExtractExternalDataQueue, messageBytes, &header)
}

// transformFiltersForWorker converts flat filter array to nested map structure.
// The Worker expects: map[datasource]map[table]map[field]FilterCondition
//
// Each filter's Field must be in qualified format:
// - "configName.tableName.fieldName" (3 parts)
// - "configName.schema.tableName.fieldName" (4 parts)
//
// Filters are applied ONLY to their specified datasource/table combination.
//
//nolint:gocyclo // High cyclomatic complexity is inherent to operator mapping with 9 distinct cases
func (s *CreateFetcherJob) transformFiltersForWorker(
	filters []model.Filter,
	mappedFields map[string]map[string][]string,
) map[string]map[string]map[string]job.FilterCondition {
	if len(filters) == 0 || len(mappedFields) == 0 {
		return nil
	}

	// Initialize result map with empty maps for all datasources and tables
	result := make(map[string]map[string]map[string]job.FilterCondition)
	for dsName, tables := range mappedFields {
		result[dsName] = make(map[string]map[string]job.FilterCondition)
		for table := range tables {
			result[dsName][table] = make(map[string]job.FilterCondition)
		}
	}

	// Process each filter and apply to its specific table
	for _, f := range filters {
		parsed, err := model.ParseFilterField(f.Field)
		if err != nil {
			// Skip invalid filters (validation should catch these earlier)
			continue
		}

		// Check if the datasource and table exist in mappedFields
		if _, dsExists := result[parsed.ConfigName]; !dsExists {
			continue
		}

		if _, tableExists := result[parsed.ConfigName][parsed.TableName]; !tableExists {
			continue
		}

		// Get or create the filter condition for this field
		condition, exists := result[parsed.ConfigName][parsed.TableName][parsed.FieldName]
		if !exists {
			condition = job.FilterCondition{}
		}

		// Map operator to appropriate filter condition field
		switch f.Operator {
		case "eq":
			condition.Equals = append(condition.Equals, f.Value...)
		case "gt":
			condition.GreaterThan = append(condition.GreaterThan, f.Value...)
		case "gte":
			condition.GreaterOrEqual = append(condition.GreaterOrEqual, f.Value...)
		case "lt":
			condition.LessThan = append(condition.LessThan, f.Value...)
		case "lte":
			condition.LessOrEqual = append(condition.LessOrEqual, f.Value...)
		case "ne":
			condition.NotEquals = append(condition.NotEquals, f.Value...)
		case "in":
			condition.In = append(condition.In, f.Value...)
		case "nin":
			condition.NotIn = append(condition.NotIn, f.Value...)
		case "like":
			condition.Like = append(condition.Like, f.Value...)
		case "between":
			condition.Between = append(condition.Between, f.Value...)
		}

		result[parsed.ConfigName][parsed.TableName][parsed.FieldName] = condition
	}

	// Clean up empty tables from result to avoid sending unnecessary data
	for dsName, tables := range result {
		for table, fields := range tables {
			if len(fields) == 0 {
				delete(result[dsName], table)
			}
		}

		if len(result[dsName]) == 0 {
			delete(result, dsName)
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}
