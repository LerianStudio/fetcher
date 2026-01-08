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
	queueName        string
}

// NewCreateFetcherJob creates a new CreateFetcherJob service.
// The queueName parameter specifies the RabbitMQ queue for publishing jobs.
// If empty or whitespace-only, defaults to "fetcher.extract-external-data.queue" for backwards compatibility.
func NewCreateFetcherJob(
	connectionRepo connRepo.Repository,
	jobRepository jobRepo.Repository,
	cryptor crypto.Cryptor,
	rabbitMQ *rabbitmq.RabbitMQAdapter,
	queueName string,
) *CreateFetcherJob {
	svc := &CreateFetcherJob{
		connRepo:  connectionRepo,
		jobRepo:   jobRepository,
		cryptor:   cryptor,
		rabbitMQ:  rabbitMQ,
		queueName: queueName,
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
	rabbitMQ *rabbitmq.RabbitMQAdapter,
	tester ConnectionTester,
	queueName string,
) *CreateFetcherJob {
	svc := &CreateFetcherJob{
		connRepo:         connectionRepo,
		jobRepo:          jobRepository,
		cryptor:          cryptor,
		rabbitMQ:         rabbitMQ,
		connectionTester: tester,
		queueName:        queueName,
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
		request.DataRequest.Filters, // Filters already in nested format
		model.JobStatusPending,      // Initial status is PENDING
		"",                          // ResultPath is empty initially
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
				Message:    fmt.Sprintf("Connection '%s' is not available", conn.ConfigName),
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
	logger := commons.NewLoggerFromContext(ctx)

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

	return s.rabbitMQ.ProducerDefault(ctx, "", s.queueName, messageBytes, &header)
}
