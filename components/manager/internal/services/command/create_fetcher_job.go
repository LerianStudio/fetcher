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
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/ports/job"
	"github.com/LerianStudio/fetcher/pkg/ports/messaging"
	productRepo "github.com/LerianStudio/fetcher/pkg/ports/product"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
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
	productRepo      productRepo.Repository
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
	productRepository productRepo.Repository,
	cryptor crypto.Cryptor,
	rabbitMQ messaging.MessagePublisher,
	queueName string,
	factory datasource.DataSourceFactory,
) *CreateFetcherJob {
	svc := &CreateFetcherJob{
		connRepo:    connectionRepo,
		jobRepo:     jobRepository,
		productRepo: productRepository,
		cryptor:     cryptor,
		rabbitMQ:    rabbitMQ,
		queueName:   queueName,
		dsFactory:   factory,
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
	productRepository productRepo.Repository,
	cryptor crypto.Cryptor,
	rabbitMQ messaging.MessagePublisher,
	tester ConnectionTester,
	queueName string,
	factory datasource.DataSourceFactory,
) *CreateFetcherJob {
	svc := &CreateFetcherJob{
		connRepo:         connectionRepo,
		jobRepo:          jobRepository,
		productRepo:      productRepository,
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
//
//nolint:gocyclo // Complexity reduced by extracting validateProductOwnership; remaining complexity is inherent to job creation orchestration
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
		libOpentelemetry.HandleSpanBusinessErrorEvent(&span, "Invalid request payload", errValidation)
		return nil, errValidation
	}

	// Validate required metadata.source field
	if err := validateMetadataSource(&span, request.Metadata); err != nil {
		return nil, err
	}

	// Validate filter references against mappedFields
	if len(newJob.Filters) > 0 {
		if errFilters := model.ValidateFilterReferences(newJob.Filters, newJob.MappedFields); errFilters != nil {
			libOpentelemetry.HandleSpanBusinessErrorEvent(&span, "Invalid filter references", errFilters)

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
		libOpentelemetry.HandleSpanBusinessErrorEvent(&span, "No connections found for the provided datasources", nil)

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
			libOpentelemetry.HandleSpanBusinessErrorEvent(&span, "Connection not found", err)

			return nil, pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrMissingDataSource.Error(),
				Title:      "Missing Data Source",
				Message:    fmt.Sprintf("No connection configured for datasource '%s'", dsName),
			}
		}
	}

	// Validate product ownership when metadata.source is provided and productRepo is available
	if source, ok := request.Metadata["source"].(string); ok && source != "" && s.productRepo != nil {
		if err := s.validateProductOwnership(ctx, &span, source, organizationID, connections); err != nil {
			return nil, fmt.Errorf("failed to validate product ownership: %w", err)
		}
	}

	// Test each connection to verify they are UP
	for _, conn := range connections {
		if err := s.connectionTester.TestConnection(ctx, conn); err != nil {
			libOpentelemetry.HandleSpanBusinessErrorEvent(&span, fmt.Sprintf("Connection test failed for %s", conn.ConfigName), err)

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

// validateMetadataSource validates that the request metadata contains a valid source field.
// Returns an error if metadata is nil, source is missing, or source is not a non-empty string.
func validateMetadataSource(span *trace.Span, metadata map[string]any) error {
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
	if !ok || sourceStr == "" {
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
// identified by the given source code. It resolves the product by code, then checks
// that every connection is assigned to that product.
func (s *CreateFetcherJob) validateProductOwnership(ctx context.Context, span *trace.Span, source string, organizationID uuid.UUID, connections []*model.Connection) error {
	product, errProd := s.productRepo.FindByCode(ctx, source, organizationID)
	if errProd != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to resolve product by code", errProd)
		return pkg.ValidateInternalError(errProd, "fetcher")
	}

	if product == nil {
		err := fmt.Errorf("product not found for source code: %s", source)
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Product not found", err)

		return pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Product Not Found",
			Message:    fmt.Sprintf("No product found with code '%s'", source),
		}
	}

	(*span).SetAttributes(attribute.String("app.request.resolved_product_id", product.ID.String()))

	for _, conn := range connections {
		if conn.ProductID == nil {
			err := fmt.Errorf("connection '%s' has no product assigned", conn.ConfigName)
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Unassigned connection", err)

			return pkg.ValidationError{
				EntityType: "fetcher",
				Code:       constant.ErrConnectionNotAssigned.Error(),
				Title:      "Connection Not Assigned",
				Message:    fmt.Sprintf("Connection '%s' has no product assigned. Use the migration endpoint to assign it first.", conn.ConfigName),
			}
		}

		if *conn.ProductID != product.ID {
			err := fmt.Errorf("connection '%s' belongs to product '%s', not '%s'", conn.ConfigName, conn.ProductID, product.ID)
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
		libOpentelemetry.HandleSpanError(&span, "Connection test failed", err)
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
