package command

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/ports/job"
	"github.com/LerianStudio/fetcher/pkg/ports/messaging"

	"github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// DeduplicationWindowMinutes is the time window for request deduplication.
	DeduplicationWindowMinutes = 5

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
	rabbitMQ         messaging.MessagePublisher
	connectionTester ConnectionTester
	dsFactory        datasource.DataSourceFactory
	queueName        string
}

// NewCreateFetcherJob creates a new CreateFetcherJob service.
// The queueName parameter specifies the RabbitMQ queue for publishing jobs.
// If empty or whitespace-only, defaults to "fetcher.extract-external-data.queue" for backwards compatibility.
func NewCreateFetcherJob(
	connectionRepo connRepo.Repository,
	jobRepository jobRepo.Repository,
	cryptor crypto.Cryptor,
	rabbitMQ messaging.MessagePublisher,
	queueName string,
	factory datasource.DataSourceFactory,
) *CreateFetcherJob {
	svc := &CreateFetcherJob{
		connRepo:  connectionRepo,
		jobRepo:   jobRepository,
		cryptor:   cryptor,
		rabbitMQ:  rabbitMQ,
		queueName: queueName,
		dsFactory: factory,
	}
	// Use default connection tester that tests real connections
	svc.connectionTester = svc

	return svc
}

// NewCreateFetcherJobWithTester creates a new CreateFetcherJob service with a custom connection tester.
// This is useful for testing where you want to mock connection testing.
// The queueName parameter specifies the RabbitMQ queue for publishing jobs.
// If empty or whitespace-only, defaults to "fetcher.extract-external-data.queue" for backwards compatibility.
func NewCreateFetcherJobWithTester(
	connectionRepo connRepo.Repository,
	jobRepository jobRepo.Repository,
	cryptor crypto.Cryptor,
	rabbitMQ messaging.MessagePublisher,
	tester ConnectionTester,
	queueName string,
	factory datasource.DataSourceFactory,
) *CreateFetcherJob {
	svc := &CreateFetcherJob{
		connRepo:         connectionRepo,
		jobRepo:          jobRepository,
		cryptor:          cryptor,
		rabbitMQ:         rabbitMQ,
		connectionTester: tester,
		queueName:        queueName,
		dsFactory:        factory,
	}
	if tester == nil {
		svc.connectionTester = svc
	}

	return svc
}

// Execute creates a new fetcher job or returns an existing duplicate.
func (s *CreateFetcherJob) Execute(ctx context.Context, organizationID uuid.UUID, request model.FetcherRequest) (*CreateFetcherJobResult, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.create_fetcher_job")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
	)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", request, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert fetcher request to JSON string", err)
	}

	newJob, err := s.validateAndBuildJob(span, organizationID, request)
	if err != nil {
		return nil, err
	}

	dupResult, err := s.checkDuplicateJob(ctx, span, logger, newJob)
	if err != nil {
		return nil, err
	}

	if dupResult != nil {
		return dupResult, nil
	}

	connections, err := s.resolveAndValidateConnections(ctx, span, organizationID, newJob)
	if err != nil {
		return nil, err
	}

	if source, ok := request.Metadata["source"].(string); ok && strings.TrimSpace(source) != "" {
		if err := s.validateProductOwnership(ctx, span, source, organizationID, connections); err != nil {
			return nil, err
		}
	}

	if err := s.testConnections(ctx, span, connections); err != nil {
		return nil, err
	}

	createdJob, err := s.createJobWithConflictResolution(ctx, span, logger, newJob)
	if err != nil {
		return nil, err
	}

	if createdJob.duplicate {
		return createdJob.asResult(), nil
	}

	span.SetAttributes(attribute.String("app.request.created_job_id", createdJob.job.ID.String()))

	logger.Log(ctx, libLog.LevelInfo, "created fetcher job",
		libLog.String("job_id", createdJob.job.ID.String()),
		libLog.String("organization_id", organizationID.String()),
	)

	if err := s.publishAndHandleFailure(ctx, span, logger, createdJob.job); err != nil {
		return nil, err
	}

	return &CreateFetcherJobResult{
		Job:          createdJob.job,
		IsDuplicate:  false,
		IsNewCreated: true,
	}, nil
}

// jobCreationResult holds the outcome of a job creation attempt,
// distinguishing between a newly created job and a duplicate found during conflict resolution.
type jobCreationResult struct {
	job       *model.Job
	duplicate bool
}

// asResult converts a jobCreationResult into a CreateFetcherJobResult.
func (r *jobCreationResult) asResult() *CreateFetcherJobResult {
	return &CreateFetcherJobResult{
		Job:          r.job,
		IsDuplicate:  r.duplicate,
		IsNewCreated: !r.duplicate,
	}
}

// validateAndBuildJob computes the request hash, creates the job entity,
// validates it, and validates filter references.
func (s *CreateFetcherJob) validateAndBuildJob(span trace.Span, organizationID uuid.UUID, request model.FetcherRequest) (*model.Job, error) {
	requestHash, err := request.ComputeRequestHash()
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to compute request hash", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	span.SetAttributes(attribute.String("app.request.request_hash", requestHash))

	newJob, err := model.NewJob(
		organizationID,
		request.Metadata,
		request.DataRequest.MappedFields,
		request.DataRequest.Filters,
		model.JobStatusPending,
		"",
		requestHash,
		time.Now().UTC(),
		nil,
	)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to create job entity", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	if errValidation := newJob.IsValid(); errValidation != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Invalid request payload", errValidation)
		return nil, errValidation
	}

	// Validate required metadata.source field
	if err := validateMetadataSource(span, request.Metadata); err != nil {
		return nil, err
	}

	if err := s.validateFilterReferences(span, newJob); err != nil {
		return nil, err
	}

	return newJob, nil
}

// validateFilterReferences checks that any filters in the job reference valid
// datasources and fields from mappedFields.
func (s *CreateFetcherJob) validateFilterReferences(span trace.Span, newJob *model.Job) error {
	if len(newJob.Filters) == 0 {
		return nil
	}

	errFilters := model.ValidateFilterReferences(newJob.Filters, newJob.MappedFields)
	if errFilters == nil {
		return nil
	}

	libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Invalid filter references", errFilters)

	return pkg.ValidationError{
		EntityType: "fetcher",
		Code:       constant.ErrInvalidDataRequest.Error(),
		Title:      "Invalid Filter References",
		Message:    errFilters.Error(),
		Err:        errFilters,
	}
}

// checkDuplicateJob checks whether a duplicate job already exists within the
// deduplication window. Returns a non-nil result if a non-failed duplicate is
// found. Returns (nil, nil) if no duplicate exists or the existing job failed
// (allowing a retry). Returns (nil, error) on lookup failure.
func (s *CreateFetcherJob) checkDuplicateJob(ctx context.Context, span trace.Span, logger libLog.Logger, newJob *model.Job) (*CreateFetcherJobResult, error) {
	existingJob, err := s.jobRepo.FindByRequestHashWithinWindow(ctx, newJob.OrganizationID, newJob.RequestHash, DeduplicationWindowMinutes)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to check for duplicate job", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	if existingJob == nil {
		return nil, nil
	}

	if existingJob.Status == model.JobStatusFailed {
		logger.Log(ctx, libLog.LevelInfo, "existing failed job found, allowing retry",
			libLog.String("job_id", existingJob.ID.String()),
		)

		return nil, nil
	}

	logger.Log(ctx, libLog.LevelInfo, "duplicate request detected, returning existing job",
		libLog.String("job_id", existingJob.ID.String()),
	)
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

// resolveAndValidateConnections looks up connections for the job's datasource names,
// ensures at least one connection exists, and verifies every requested datasource
// has a corresponding connection.
func (s *CreateFetcherJob) resolveAndValidateConnections(ctx context.Context, span trace.Span, organizationID uuid.UUID, newJob *model.Job) ([]*model.Connection, error) {
	connections, err := s.connRepo.FindByConfigNames(ctx, organizationID, newJob.GetDatasourceNames())
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find connections", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	if len(connections) == 0 {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "No connections found for the provided datasources", nil)

		return nil, pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrMissingDataSource.Error(),
			Title:      "No Connections Found",
			Message:    "No connections configured for the requested datasources",
		}
	}

	if err := s.ensureAllDatasourcesHaveConnections(span, newJob.GetDatasourceNames(), connections); err != nil {
		return nil, err
	}

	return connections, nil
}

// ensureAllDatasourcesHaveConnections verifies that every requested datasource name
// has a matching connection in the provided list.
func (s *CreateFetcherJob) ensureAllDatasourcesHaveConnections(span trace.Span, dsNames []string, connections []*model.Connection) error {
	connMap := make(map[string]*model.Connection, len(connections))
	for _, conn := range connections {
		if conn == nil {
			continue
		}

		connMap[conn.ConfigName] = conn
	}

	for _, dsName := range dsNames {
		if _, found := connMap[dsName]; !found {
			err := fmt.Errorf("connection not found for datasource: %s", dsName)
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection not found", err)

			return pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrMissingDataSource.Error(),
				Title:      "Missing Data Source",
				Message:    fmt.Sprintf("No connection configured for datasource '%s'", dsName),
			}
		}
	}

	return nil
}

// testConnections verifies that each connection is reachable.
func (s *CreateFetcherJob) testConnections(ctx context.Context, span trace.Span, connections []*model.Connection) error {
	for _, conn := range connections {
		if conn == nil {
			continue
		}

		if err := s.connectionTester.TestConnection(ctx, conn); err != nil {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, fmt.Sprintf("Connection test failed for %s", conn.ConfigName), err)

			return pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrConnectionDown.Error(),
				Title:      "Connection Down",
				Message:    fmt.Sprintf("Connection '%s' is not available", conn.ConfigName),
			}
		}
	}

	return nil
}

// createJobWithConflictResolution persists the job to the database, handling
// duplicate key conflicts by looking up the active job or retrying once.
func (s *CreateFetcherJob) createJobWithConflictResolution(ctx context.Context, span trace.Span, logger libLog.Logger, newJob *model.Job) (*jobCreationResult, error) {
	createdJob, err := s.jobRepo.Create(ctx, newJob)
	if err == nil {
		return &jobCreationResult{job: createdJob}, nil
	}

	if !mongo.IsDuplicateKeyError(err) {
		libOpentelemetry.HandleSpanError(span, "Failed to create job", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	// Duplicate key conflict -- attempt to find the active job that caused it
	result, err := s.recoverFromDuplicateKey(ctx, span, logger, newJob)
	if err != nil {
		return nil, err
	}

	if result != nil {
		return result, nil
	}

	// Active job may have transitioned to terminal status between duplicate-key write
	// and read-after-conflict; retry create once for deterministic behavior.
	return s.retryCreateAfterConflict(ctx, span, logger, newJob)
}

// recoverFromDuplicateKey attempts to find the active duplicate job after a
// duplicate key error during creation.
func (s *CreateFetcherJob) recoverFromDuplicateKey(ctx context.Context, span trace.Span, logger libLog.Logger, newJob *model.Job) (*jobCreationResult, error) {
	existingActiveJob, findErr := s.findActiveDuplicateJob(ctx, newJob.OrganizationID, newJob.RequestHash)
	if findErr != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to load active job after duplicate create", findErr)
		return nil, pkg.ValidateInternalError(findErr, "fetcher")
	}

	if existingActiveJob == nil {
		return nil, nil
	}

	logger.Log(ctx, libLog.LevelInfo, "concurrent duplicate request detected, returning active job",
		libLog.String("job_id", existingActiveJob.ID.String()),
	)
	span.SetAttributes(
		attribute.Bool("app.request.is_duplicate", true),
		attribute.String("app.request.existing_job_id", existingActiveJob.ID.String()),
	)

	return &jobCreationResult{job: existingActiveJob, duplicate: true}, nil
}

// retryCreateAfterConflict retries job creation once after the initial duplicate
// key conflict and active-job lookup returned nothing. This is a bounded single
// retry -- it must never be extended into a loop.
func (s *CreateFetcherJob) retryCreateAfterConflict(ctx context.Context, span trace.Span, logger libLog.Logger, newJob *model.Job) (*jobCreationResult, error) {
	createdJob, err := s.jobRepo.Create(ctx, newJob)
	if err == nil {
		return &jobCreationResult{job: createdJob}, nil
	}

	if mongo.IsDuplicateKeyError(err) {
		result, recoverErr := s.recoverFromDuplicateKey(ctx, span, logger, newJob)
		if recoverErr != nil {
			return nil, recoverErr
		}

		if result != nil {
			return result, nil
		}
	}

	libOpentelemetry.HandleSpanError(span, "Failed to create job after conflict resolution retry", err)

	return nil, pkg.ValidateInternalError(
		fmt.Errorf("unable to create job after conflict resolution: concurrent active job prevents creation: %w", err),
		"fetcher",
	)
}

// publishAndHandleFailure publishes the created job to the queue. If publishing
// fails, it marks the job as failed and returns the publish error.
//
// Design limitation (2PC): If both the queue publish AND the subsequent DB
// status update fail, the job remains in PENDING with no queue message.
// The unique active hash index prevents re-creation for the same request hash.
// Mitigation: a TTL-based reaper for stale PENDING jobs can be added if this
// edge case manifests in production (requires simultaneous RabbitMQ + MongoDB failure).
func (s *CreateFetcherJob) publishAndHandleFailure(ctx context.Context, span trace.Span, logger libLog.Logger, createdJob *model.Job) error {
	err := s.publishToQueue(ctx, createdJob)
	if err == nil {
		return nil
	}

	libOpentelemetry.HandleSpanError(span, "Failed to publish job to queue", err)
	logger.Log(ctx, libLog.LevelError, "failed to publish job to queue",
		libLog.String("job_id", createdJob.ID.String()),
		libLog.Err(err),
	)

	createdJob.SetFailedStatus("process failed: unable to publish")

	_, updateErr := s.jobRepo.Update(ctx, createdJob)
	if updateErr != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to update job status to FAILED", updateErr)
		logger.Log(ctx, libLog.LevelError, "failed to update job status to failed",
			libLog.String("job_id", createdJob.ID.String()),
			libLog.Err(updateErr),
		)
	}

	return pkg.ValidateInternalError(err, "fetcher")
}

func (s *CreateFetcherJob) findActiveDuplicateJob(ctx context.Context, organizationID uuid.UUID, requestHash string) (*model.Job, error) {
	return s.jobRepo.FindActiveByRequestHash(ctx, organizationID, requestHash)
}

// validateMetadataSource validates that the request metadata contains a valid source field.
// Returns an error if metadata is nil, source is missing, or source is not a non-empty string.
func validateMetadataSource(span trace.Span, metadata map[string]any) error {
	if metadata == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Missing required metadata", nil)

		return pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrMissingFieldsInRequest.Error(),
			Title:      "Missing Required Field",
			Message:    "metadata is required and must contain 'source' field",
		}
	}

	source, hasSource := metadata["source"]
	if !hasSource || source == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Missing required metadata.source", nil)

		return pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrMissingFieldsInRequest.Error(),
			Title:      "Missing Required Field",
			Message:    "metadata.source is required for job notification routing",
		}
	}

	sourceStr, ok := source.(string)
	if !ok || strings.TrimSpace(sourceStr) == "" {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Invalid metadata.source type or empty value", nil)

		return pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrMissingFieldsInRequest.Error(),
			Title:      "Invalid Field Value",
			Message:    "metadata.source must be a non-empty string",
		}
	}

	return nil
}

// validateProductOwnership validates that all connections belong to the product
// identified by the given source name. It checks that every connection's ProductName
// matches the expected source string.
func (s *CreateFetcherJob) validateProductOwnership(_ context.Context, span trace.Span, source string, _ uuid.UUID, connections []*model.Connection) error {
	for _, conn := range connections {
		if conn == nil {
			continue
		}

		if conn.ProductName == "" {
			err := fmt.Errorf("connection '%s' has no product assigned", conn.ConfigName)
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Unassigned connection", err)

			return pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrConnectionNotAssigned.Error(),
				Title:      "Connection Not Assigned",
				Message:    fmt.Sprintf("Connection '%s' has no product assigned. Use the migration endpoint to assign it first.", conn.ConfigName),
			}
		}

		if conn.ProductName != source {
			err := fmt.Errorf("connection '%s' belongs to product '%s', not '%s'", conn.ConfigName, conn.ProductName, source)
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Product mismatch", err)

			return pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrProductMismatch.Error(),
				Title:      "Product Mismatch",
				Message:    fmt.Sprintf("Connection '%s' does not belong to the product identified by source '%s'", conn.ConfigName, source),
			}
		}
	}

	return nil
}

// TestConnection tests if a connection is available.
// This method implements the ConnectionTester interface.
func (s *CreateFetcherJob) TestConnection(ctx context.Context, conn *model.Connection) error {
	if s.dsFactory == nil {
		return nil
	}

	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.test_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.config_name", conn.ConfigName),
		attribute.String("app.request.connection_type", string(conn.Type)),
	)

	testCtx, cancel := context.WithTimeout(ctx, ConnectionTestTimeout)
	defer cancel()

	ds, err := s.dsFactory(testCtx, conn, s.cryptor)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Connection test failed", err)
		return fmt.Errorf("connection test failed: %w", err)
	}

	// Close the connection after test
	if err := ds.Close(testCtx); err != nil {
		logger.Log(testCtx, libLog.LevelWarn, "failed to close test connection",
			libLog.String("config_name", conn.ConfigName),
			libLog.Err(err),
		)
	}

	return nil
}

// publishToQueue publishes a job to the RabbitMQ queue.
// If RabbitMQ is not configured (nil), this method does nothing and returns nil.
func (s *CreateFetcherJob) publishToQueue(ctx context.Context, j *model.Job) error {
	if isNilRabbitMQAdapter(s.rabbitMQ) {
		return nil
	}

	message := map[string]any{
		"jobId":          j.ID.String(),
		"organizationId": j.OrganizationID.String(),
		"mappedFields":   j.MappedFields,
		"metadata":       j.Metadata,
		"createdAt":      j.CreatedAt,
	}

	// Filters are already in the nested format expected by Worker
	if len(j.Filters) > 0 {
		message["filters"] = j.Filters
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal job message: %w", err)
	}

	header := map[string]any{
		"jobId":          j.ID.String(),
		"organizationId": j.OrganizationID.String(),
	}

	// Propagate tenant ID to worker via AMQP header for multi-tenant isolation.
	// When MULTI_TENANT_ENABLED=false, tenant context is empty and no header is added.
	if tenantID := tmcore.GetTenantIDFromContext(ctx); tenantID != "" {
		header["X-Tenant-ID"] = tenantID
	}

	return s.rabbitMQ.ProducerDefault(ctx, "", s.queueName, messageBytes, &header)
}

func isNilRabbitMQAdapter(adapter messaging.MessagePublisher) bool {
	if adapter == nil {
		return true
	}

	return isNilReferenceValue(adapter)
}

func isNilReferenceValue(value any) bool {
	rv := reflect.ValueOf(value)

	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
