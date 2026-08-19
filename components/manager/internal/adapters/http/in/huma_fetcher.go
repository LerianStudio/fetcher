package in

import (
	"context"
	"fmt"
	"net/http"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	observability "github.com/LerianStudio/lib-observability/v2"
	libLog "github.com/LerianStudio/lib-observability/v2/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/v2/tracing"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
)

func bindFetcherHandlers(handlers *OperationHandlers, handler *FetcherHandler) {
	if handlers == nil || handler == nil {
		return
	}

	handlers.CreateJob = func(ctx context.Context, input *rawJSONInput) (*createJobOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		logger, tracer, requestID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.create_fetcher_job")
		defer span.End()

		fiberCtx.SetContext(ctx)
		span.SetAttributes(attribute.String("app.request.request_id", requestID))

		request, err := decodeJSON[model.FetcherRequest](input.RawBody, "fetcher")
		if err != nil {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "failed to parse payload", err)

			return nil, err
		}

		if err := validateStruct(&request); err != nil {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "request validation failed", err)

			return nil, err
		}

		result, err := handler.CreateJobCmd.Execute(ctx, request)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute create fetcher job command, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to create fetcher job", err)

			return nil, mapServiceError(err)
		}

		status := http.StatusAccepted
		message := "Job created and queued for processing"

		if result.IsDuplicate {
			status = http.StatusOK
			message = "Duplicate request detected - returning existing job"

			logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Duplicate fetcher job returned id=%s", result.Job.ID))
		} else {
			logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Fetcher job created id=%s", result.Job.ID))
		}

		return &createJobOutput{
			Status: status,
			Body: model.FetcherResponse{
				JobID:     result.Job.ID,
				Status:    string(result.Job.Status),
				CreatedAt: result.Job.CreatedAt,
				Message:   message,
			},
		}, nil
	}

	handlers.GetJob = func(ctx context.Context, input *idInput) (*getJobOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		logger, tracer, requestID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.get_fetcher_job")
		defer span.End()

		fiberCtx.SetContext(ctx)
		span.SetAttributes(attribute.String("app.request.request_id", requestID))

		id, err := parseID(input.ID, "job")
		if err != nil {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "invalid job id parameter", err)

			return nil, err
		}

		span.SetAttributes(attribute.String("app.request.job_id", id.String()))

		job, err := handler.GetJobQuery.Execute(ctx, id)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute get job query, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to get job", err)

			return nil, mapServiceError(err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("job retrieved id=%s", id))

		return &getJobOutput{Body: *model.NewJobResponseFrom(job)}, nil
	}
}

func registerFetcherOperations(
	app *fiber.App,
	api huma.API,
	handlers OperationHandlers,
	middlewareFactory OperationMiddlewareFactory,
) {
	requestSchema := documentedFetcherRequestSchema(api)

	registerTypedOperation(
		app,
		api,
		huma.Operation{
			OperationID:      "create-fetcher-job",
			Method:           http.MethodPost,
			Path:             "/v1/fetcher",
			DefaultStatus:    http.StatusAccepted,
			Summary:          "Create a data extraction job",
			Description:      "Creates and queues a data extraction job, or returns the existing job for an idempotent duplicate request.",
			Tags:             []string{"Fetcher"},
			SkipValidateBody: true,
			Errors: protectedErrors(
				http.StatusBadRequest,
				http.StatusConflict,
				http.StatusRequestEntityTooLarge,
				http.StatusInternalServerError,
			),
			Responses: map[string]*huma.Response{
				"200": {
					Description: "Duplicate request; the existing job is returned.",
					Content: map[string]*huma.MediaType{
						"application/json": {
							Schema: schemaFor[model.FetcherResponse](api, "FetcherResponse"),
						},
					},
				},
			},
		},
		fetcherResource,
		"post",
		middlewareFactory,
		handlers.CreateJob,
	)
	replaceJSONRequestSchema(
		api,
		http.MethodPost,
		"/v1/fetcher",
		"Fetcher request payload.",
		requestSchema,
	)

	registerTypedOperation(
		app,
		api,
		huma.Operation{
			OperationID:   "get-fetcher-job",
			Method:        http.MethodGet,
			Path:          "/v1/fetcher/{id}",
			DefaultStatus: http.StatusOK,
			Summary:       "Get a data extraction job",
			Description:   "Returns a data extraction job by identifier.",
			Tags:          []string{"Fetcher"},
			Errors: protectedErrors(
				http.StatusBadRequest,
				http.StatusNotFound,
				http.StatusInternalServerError,
			),
		},
		fetcherResource,
		"get",
		middlewareFactory,
		handlers.GetJob,
	)
}

func documentedFetcherRequestSchema(api huma.API) *huma.Schema {
	reference := schemaFor[model.FetcherRequest](api, "FetcherRequest")

	component := api.OpenAPI().Components.Schemas.SchemaFromRef(reference.Ref)
	if component == nil {
		return reference
	}

	metadata := component.Properties["metadata"]
	if metadata == nil {
		return reference
	}

	if metadata.Properties == nil {
		metadata.Properties = map[string]*huma.Schema{}
	}

	metadata.Properties["source"] = &huma.Schema{
		Type:        "string",
		Description: "Owning product used for notification routing and connection ownership validation.",
		Examples:    []any{"payments"},
	}
	metadata.Required = appendRequiredProperty(metadata.Required, "source")
	component.Required = appendRequiredProperty(component.Required, "metadata")

	return reference
}

func appendRequiredProperty(required []string, name string) []string {
	for _, existing := range required {
		if existing == name {
			return required
		}
	}

	return append(required, name)
}
