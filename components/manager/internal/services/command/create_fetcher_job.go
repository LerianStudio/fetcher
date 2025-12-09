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
	"github.com/LerianStudio/lib-commons/v2/commons/log"
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

// CreateFetcherJob is the command service for creating fetcher jobs.
type CreateFetcherJob struct {
	connRepo connRepo.Repository
	jobRepo  jobRepo.Repository
	cryptor  crypto.Cryptor
	rabbitMQ *rabbitmq.RabbitMQAdapter
}

// NewCreateFetcherJob creates a new CreateFetcherJob service.
func NewCreateFetcherJob(
	connRepo connRepo.Repository,
	jobRepo jobRepo.Repository,
	cryptor crypto.Cryptor,
	rabbitMQ *rabbitmq.RabbitMQAdapter,
) *CreateFetcherJob {
	return &CreateFetcherJob{
		connRepo: connRepo,
		jobRepo:  jobRepo,
		cryptor:  cryptor,
		rabbitMQ: rabbitMQ,
	}
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

	job, err := model.NewJob(
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
	if errValidation := job.IsValid(); errValidation != nil {
		libOpentelemetry.HandleSpanError(&span, "Invalid request payload", errValidation)
		return nil, errValidation
	}

	// Check for duplicate within deduplication window
	existingJob, err := s.jobRepo.FindByRequestHashWithinWindow(ctx, job.OrganizationID, job.RequestHash, DeduplicationWindowMinutes)
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
	connections, err := s.connRepo.FindByConfigNames(ctx, organizationID, job.GetDatasourceNames())
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to find connections", err)
		return nil, pkg.ValidateInternalError(err, "fetcher")
	}

	// Check that all datasources have corresponding connections
	connMap := make(map[string]*model.Connection, len(connections))
	for _, conn := range connections {
		connMap[conn.ConfigName] = conn
	}

	for _, dsName := range job.GetDatasourceNames() {
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
		if err := s.testConnection(ctx, conn, logger); err != nil {
			libOpentelemetry.HandleSpanError(&span, fmt.Sprintf("Connection test failed for %s", conn.ConfigName), err)

			return nil, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrConnectionDown.Error(),
				Title:      "Connection Down",
				Message:    fmt.Sprintf("Connection '%s' is not available: %s", conn.ConfigName, err.Error()),
			}
		}
	}

	createdJob, err := s.jobRepo.Create(ctx, job)
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

// testConnection tests if a connection is available.
func (s *CreateFetcherJob) testConnection(ctx context.Context, conn *model.Connection, logger log.Logger) error {
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
func (s *CreateFetcherJob) publishToQueue(ctx context.Context, job *model.Job) error {
	message := map[string]any{
		"jobId":          job.ID.String(),
		"organizationId": job.OrganizationID.String(),
		"mappedFields":   job.MappedFields,
		"filters":        job.Filters,
		"metadata":       job.Metadata,
		"createdAt":      job.CreatedAt,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal job message: %w", err)
	}

	header := map[string]any{
		"jobId":          job.ID.String(),
		"organizationId": job.OrganizationID.String(),
	}

	return s.rabbitMQ.ProducerDefault(ctx, "", ExtractExternalDataQueue, messageBytes, &header)
}
