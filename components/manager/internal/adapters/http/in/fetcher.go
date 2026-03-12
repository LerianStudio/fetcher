package in

import (
	"fmt"
	"strings"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/query"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// FetcherHandler handles HTTP requests for the fetcher API.
type FetcherHandler struct {
	CreateJobCmd *command.CreateFetcherJob
	GetJobQuery  *query.GetJob
}

// NewFetcherHandler creates a new FetcherHandler.
func NewFetcherHandler(createJobCmd *command.CreateFetcherJob, getJobQuery *query.GetJob) *FetcherHandler {
	return &FetcherHandler{
		CreateJobCmd: createJobCmd,
		GetJobQuery:  getJobQuery,
	}
}

// CreateJob handles POST /v1/fetcher requests to create a new data extraction job.
//
//	@Summary		Create fetcher job
//	@Description	Create a new data extraction job. The request will be validated, deduplicated within a 5-minute window, and all referenced connections will be tested before job creation. The metadata.source field is required for product isolation and datasource ownership validation.
//	@Tags			Fetcher
//	@Accept			json
//	@Produce		json
//	@Param			Authorization		header		string				false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string				true	"Organization ID"
//	@Param			request				body		model.FetcherRequest	true	"Fetcher request payload. metadata.source is required."
//	@Success		200					{object}	model.FetcherResponse	"Duplicate request - returning existing job"
//	@Success		202					{object}	model.FetcherResponse	"Job created and queued for processing"
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		409					{object}	pkg.HTTPError
//	@Failure		413					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/fetcher [post]
func (h *FetcherHandler) CreateJob(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.create_fetcher_job")
	defer span.End()

	c.SetUserContext(ctx)

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	var request model.FetcherRequest
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "failed to parse payload", errParser)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        errParser,
		})
	}

	// Validate required metadata.source field
	if request.Metadata == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "missing required metadata", nil)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrMissingFieldsInRequest.Error(),
			Title:      "Missing Required Field",
			Message:    "metadata is required and must contain 'source' field",
		})
	}

	source, hasSource := request.Metadata["source"]
	if !hasSource || source == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "missing required metadata.source", nil)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrMissingFieldsInRequest.Error(),
			Title:      "Missing Required Field",
			Message:    "metadata.source is required for job notification routing",
		})
	}

	// Validate source is a non-empty string
	sourceStr, ok := source.(string)
	if !ok || strings.TrimSpace(sourceStr) == "" {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "invalid metadata.source type or empty value", nil)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "fetcher",
			Code:       constant.ErrMissingFieldsInRequest.Error(),
			Title:      "Invalid Field Value",
			Message:    "metadata.source must be a non-empty string",
		})
	}

	result, err := h.CreateJobCmd.Execute(ctx, orgID, request)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute create fetcher job command, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to create fetcher job", err)

		return httpUtils.WithError(c, err)
	}

	response := model.FetcherResponse{
		JobID:     result.Job.ID,
		Status:    string(result.Job.Status),
		CreatedAt: result.Job.CreatedAt,
	}

	// Return 200 OK for duplicates, 202 Accepted for new jobs
	if result.IsDuplicate {
		response.Message = "Duplicate request detected - returning existing job"

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Duplicate fetcher job returned id=%s org=%s", result.Job.ID, orgID))

		return httpUtils.OK(c, response)
	}

	response.Message = "Job created and queued for processing"

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Fetcher job created id=%s org=%s", result.Job.ID, orgID))

	return httpUtils.Accepted(c, response)
}

// GetJob handles GET /v1/fetcher/:id requests to retrieve a job by ID.
//
//	@Summary		Get fetcher job
//	@Description	Retrieve detailed information about a specific data extraction job.
//	@Tags			Fetcher
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string	true	"Organization ID"
//	@Param			id					path		string	true	"Job ID"
//	@Success		200					{object}	model.JobResponse
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/fetcher/{id} [get]
func (h *FetcherHandler) GetJob(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.get_fetcher_job")
	defer span.End()

	c.SetUserContext(ctx)

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "invalid job id parameter", err)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "job",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid job id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.job_id", id.String()))

	job, err := h.GetJobQuery.Execute(ctx, orgID, id)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute get job query, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to get job", err)

		return httpUtils.WithError(c, err)
	}

	resp := model.NewJobResponseFrom(job)

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("job retrieved id=%s org=%s", id, orgID))

	return httpUtils.OK(c, resp)
}
