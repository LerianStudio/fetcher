package in

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/v2/pkg/net/http"
	"github.com/LerianStudio/lib-commons/v6/commons/net/http/problem"
	observability "github.com/LerianStudio/lib-observability/v2"
	libLog "github.com/LerianStudio/lib-observability/v2/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/v2/tracing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
)

func bindConnectionHandlers(handlers *OperationHandlers, connectionHandler *ConnectionHandler) {
	handlers.CreateConnection = createConnectionCallback(connectionHandler)
	handlers.ListConnections = listConnectionsCallback(connectionHandler)
	handlers.ValidateSchema = validateSchemaCallback(connectionHandler)
	handlers.GetConnection = getConnectionCallback(connectionHandler)
	handlers.TestConnection = testConnectionCallback(connectionHandler)
	handlers.GetConnectionSchema = getConnectionSchemaCallback(connectionHandler)
	handlers.UpdateConnection = updateConnectionCallback(connectionHandler)
	handlers.DeleteConnection = deleteConnectionCallback(connectionHandler)
}

func createConnectionCallback(connectionHandler *ConnectionHandler) func(context.Context, *rawJSONInput) (*createConnectionOutput, error) {
	return func(ctx context.Context, input *rawJSONInput) (*createConnectionOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.Context()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.create_connection")
		defer span.End()

		fiberCtx.SetContext(ctx)

		productName, err := httpUtils.GetRequiredProductName(fiberCtx)
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "missing or invalid product name", err)

			return nil, mapServiceError(err)
		}

		span.SetAttributes(
			attribute.String("app.request.request_id", reqID),
			attribute.String("app.request.product_name", productName),
		)

		request, err := decodeJSON[model.ConnectionInput](input.RawBody, "connection")
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "failed to parse payload", err)

			return nil, err
		}

		if request.IsEmpty() {
			err = pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "empty request body",
			}
			libOpentelemetry.HandleSpanError(span, "empty request body", err)

			return nil, mapServiceError(err)
		}

		if err = validateStruct(&request); err != nil {
			libOpentelemetry.HandleSpanError(span, "request validation failed", err)

			return nil, err
		}

		connection, err := connectionHandler.CreateCmd.Execute(ctx, request, productName)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute create connection command, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to create connection", err)

			return nil, mapServiceError(err)
		}

		response := model.NewConnectionResponseFrom(connection)
		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection created id=%s", response.ID))

		return &createConnectionOutput{Body: *response}, nil
	}
}

func listConnectionsCallback(connectionHandler *ConnectionHandler) func(context.Context, *emptyInput) (*listConnectionsOutput, error) {
	return func(ctx context.Context, _ *emptyInput) (*listConnectionsOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.Context()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.list_connection")
		defer span.End()

		fiberCtx.SetContext(ctx)

		productName, err := httpUtils.GetProductName(fiberCtx)
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "invalid product name", err)

			return nil, mapServiceError(err)
		}

		span.SetAttributes(attribute.String("app.request.request_id", reqID))

		if productName != "" {
			span.SetAttributes(attribute.String("app.request.product_name", productName))
		}

		headerParams, err := httpUtils.ValidateParameters(fiberCtx.Queries())
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to validate query parameters", err)
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to validate query parameters, Error: %s", err.Error()))

			return nil, mapServiceError(err)
		}

		pagination, err := connectionHandler.ListQuery.Execute(ctx, productName, *headerParams)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute list connections query, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to list connections", err)

			return nil, mapServiceError(err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connections listed count=%d", pagination.Total))

		return &listConnectionsOutput{Body: connectionPageFrom(pagination)}, nil
	}
}

func validateSchemaCallback(connectionHandler *ConnectionHandler) func(context.Context, *rawJSONInput) (*validateSchemaOutput, error) {
	return func(ctx context.Context, input *rawJSONInput) (*validateSchemaOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.Context()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.validate_schema")
		defer span.End()

		fiberCtx.SetContext(ctx)
		span.SetAttributes(attribute.String("app.request.request_id", reqID))

		request, err := decodeJSON[model.SchemaValidationRequest](input.RawBody, "schema")
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "failed to parse payload", err)

			return nil, err
		}

		if err = validateStruct(&request); err != nil {
			libOpentelemetry.HandleSpanError(span, "request validation failed", err)

			return nil, err
		}

		response, err := connectionHandler.ValidateSchemaQuery.Execute(ctx, request)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute validate schema query, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to validate schema", err)

			return nil, mapServiceError(err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("schema validation completed status=%s", response.Status))

		if response.Status == model.StatusFailure {
			mapped := mapServiceError(pkg.UnprocessableOperationError{
				Code:    constant.ErrSchemaValidationFailed.Error(),
				Title:   "Schema validation failed",
				Message: response.Message,
			})

			detail, ok := mapped.(*problem.Detail)
			if !ok {
				return nil, mapped
			}

			for _, schemaError := range response.Errors {
				detail.Errors = append(detail.Errors, &huma.ErrorDetail{
					Location: schemaErrorLocation(schemaError),
					Message:  schemaError.Type,
					Value: map[string]string{
						"dataSourceId": schemaError.DataSourceID,
						"table":        schemaError.Table,
						"field":        schemaError.Field,
					},
				})
			}

			return nil, detail
		}

		return &validateSchemaOutput{Body: *response}, nil
	}
}

func getConnectionCallback(connectionHandler *ConnectionHandler) func(context.Context, *idInput) (*connectionOutput, error) {
	return func(ctx context.Context, input *idInput) (*connectionOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.Context()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.get_connection")
		defer span.End()

		fiberCtx.SetContext(ctx)
		span.SetAttributes(attribute.String("app.request.request_id", reqID))

		id, err := parseID(input.ID, "connection")
		if err != nil {
			return nil, err
		}

		span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

		connection, err := connectionHandler.GetQuery.Execute(ctx, id)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute get connection query, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to get connection", err)

			return nil, mapServiceError(err)
		}

		response := model.NewConnectionResponseFrom(connection)

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection retrieved id=%s", id))

		return &connectionOutput{Body: *response}, nil
	}
}

func testConnectionCallback(connectionHandler *ConnectionHandler) func(context.Context, *idInput) (*testConnectionOutput, error) {
	return func(ctx context.Context, input *idInput) (*testConnectionOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.Context()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.test_connection")
		defer span.End()

		fiberCtx.SetContext(ctx)
		span.SetAttributes(attribute.String("app.request.request_id", reqID))

		id, err := parseID(input.ID, "connection")
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "invalid connection id parameter", err)

			return nil, err
		}

		span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

		response, err := connectionHandler.TestQuery.Execute(ctx, id)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute test connection query, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to test connection", err)

			return nil, mapServiceError(err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection test successful id=%s latency_ms=%d", id, response.LatencyMs))

		return &testConnectionOutput{Body: *response}, nil
	}
}

func getConnectionSchemaCallback(connectionHandler *ConnectionHandler) func(context.Context, *idInput) (*connectionSchemaOutput, error) {
	return func(ctx context.Context, input *idInput) (*connectionSchemaOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.Context()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.get_connection_schema")
		defer span.End()

		fiberCtx.SetContext(ctx)
		span.SetAttributes(attribute.String("app.request.request_id", reqID))

		id, err := parseID(input.ID, "connection")
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "invalid connection id parameter", err)

			return nil, err
		}

		span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

		response, err := connectionHandler.GetSchemaQuery.Execute(ctx, id)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute get connection schema query, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to get connection schema", err)

			return nil, mapServiceError(err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection schema retrieved id=%s tables=%d", id, len(response.Tables)))

		return &connectionSchemaOutput{Body: *response}, nil
	}
}

func updateConnectionCallback(connectionHandler *ConnectionHandler) func(context.Context, *rawJSONIDInput) (*connectionOutput, error) {
	return func(ctx context.Context, input *rawJSONIDInput) (*connectionOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.Context()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.update_connection")
		defer span.End()

		fiberCtx.SetContext(ctx)
		span.SetAttributes(attribute.String("app.request.request_id", reqID))

		id, err := parseID(input.ID, "connection")
		if err != nil {
			return nil, err
		}

		span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

		request, err := decodeJSON[model.ConnectionUpdateInput](input.RawBody, "connection")
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "failed to parse payload", err)

			return nil, err
		}

		if request.IsEmpty() {
			err = pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "empty request body",
			}
			libOpentelemetry.HandleSpanError(span, "empty request body", err)

			return nil, mapServiceError(err)
		}

		if err = validateStruct(&request); err != nil {
			libOpentelemetry.HandleSpanError(span, "request validation failed", err)

			return nil, err
		}

		connection, err := connectionHandler.UpdateCmd.Execute(ctx, id, request)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute update connection command, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to update connection", err)

			return nil, mapServiceError(err)
		}

		response := model.NewConnectionResponseFrom(connection)

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection updated id=%s", id))

		return &connectionOutput{Body: *response}, nil
	}
}

func deleteConnectionCallback(connectionHandler *ConnectionHandler) func(context.Context, *idInput) (*deleteConnectionOutput, error) {
	return func(ctx context.Context, input *idInput) (*deleteConnectionOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.Context()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.delete_connection")
		defer span.End()

		fiberCtx.SetContext(ctx)
		span.SetAttributes(attribute.String("app.request.request_id", reqID))

		id, err := parseID(input.ID, "connection")
		if err != nil {
			return nil, err
		}

		span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

		if err = connectionHandler.DeleteCmd.Execute(ctx, id); err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute delete connection command, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to delete connection", err)

			return nil, mapServiceError(err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection deleted id=%s", id))

		return &deleteConnectionOutput{}, nil
	}
}

func registerConnectionOperations(
	app *fiber.App,
	api huma.API,
	handlers OperationHandlers,
	middlewareFactory OperationMiddlewareFactory,
) {
	const basePath = "/v1/management/connections"

	registerTypedOperation(app, api, huma.Operation{
		OperationID:      "create-connection",
		Method:           http.MethodPost,
		Path:             basePath,
		Summary:          "Create connection",
		Description:      "Creates a database connection for the product named by the X-Product-Name header.",
		Tags:             []string{"Connections"},
		DefaultStatus:    http.StatusCreated,
		Errors:           protectedErrors(http.StatusBadRequest, http.StatusConflict, http.StatusInternalServerError),
		Parameters:       []*huma.Param{productNameParameter(true)},
		SkipValidateBody: true,
	}, connectionsResource, "post", middlewareFactory, handlers.CreateConnection)
	replaceJSONRequestSchema(
		api,
		http.MethodPost,
		basePath,
		"Connection settings and credentials.",
		schemaFor[model.ConnectionInput](api, "ConnectionInput"),
	)

	connectionParameters := connectionListParameters()
	listParameters := make([]*huma.Param, 0, 1+len(connectionParameters))
	listParameters = append(listParameters, productNameParameter(false))
	listParameters = append(listParameters, connectionParameters...)
	registerTypedOperation(app, api, huma.Operation{
		OperationID:   "list-connections",
		Method:        http.MethodGet,
		Path:          basePath,
		Summary:       "List connections",
		Description:   "Returns a filtered, paginated list of database connections visible to the optional product. Dynamic metadata filters use query parameters such as metadata.region=br.",
		Tags:          []string{"Connections"},
		DefaultStatus: http.StatusOK,
		Errors:        protectedErrors(http.StatusBadRequest, http.StatusInternalServerError),
		Parameters:    listParameters,
	}, connectionsResource, "get", middlewareFactory, handlers.ListConnections)

	validateSchemaPath := basePath + "/validate-schema"
	registerTypedOperation(app, api, huma.Operation{
		OperationID:      "validate-schema",
		Method:           http.MethodPost,
		Path:             validateSchemaPath,
		Summary:          "Validate a connection schema",
		Description:      "Validates mapped datasource, table, and field identifiers before a connection is used.",
		Tags:             []string{"Connections"},
		DefaultStatus:    http.StatusOK,
		Errors:           protectedErrors(http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusInternalServerError),
		SkipValidateBody: true,
	}, connectionsResource, "post", middlewareFactory, handlers.ValidateSchema)
	replaceJSONRequestSchema(
		api,
		http.MethodPost,
		validateSchemaPath,
		"Datasource, table, and field mappings to validate.",
		schemaFor[model.SchemaValidationRequest](api, "SchemaValidationRequest"),
	)

	connectionPath := basePath + "/{id}"
	registerTypedOperation(app, api, huma.Operation{
		OperationID:   "get-connection",
		Method:        http.MethodGet,
		Path:          connectionPath,
		Summary:       "Get a connection",
		Description:   "Returns one database connection by identifier.",
		Tags:          []string{"Connections"},
		DefaultStatus: http.StatusOK,
		Errors:        protectedErrors(http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError),
	}, connectionsResource, "get", middlewareFactory, handlers.GetConnection)

	registerTypedOperation(app, api, huma.Operation{
		OperationID:   "test-connection",
		Method:        http.MethodPost,
		Path:          connectionPath + "/test",
		Summary:       "Test a connection",
		Description:   "Attempts to connect to the configured datasource and reports the observed latency.",
		Tags:          []string{"Connections"},
		DefaultStatus: http.StatusOK,
		Errors: protectedErrors(
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
		),
	}, connectionsResource, "post", middlewareFactory, handlers.TestConnection)

	registerTypedOperation(app, api, huma.Operation{
		OperationID:   "get-connection-schema",
		Method:        http.MethodGet,
		Path:          connectionPath + "/schema",
		Summary:       "Discover a connection schema",
		Description:   "Returns tables and fields discovered from the configured datasource.",
		Tags:          []string{"Connections"},
		DefaultStatus: http.StatusOK,
		Errors:        protectedErrors(http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError),
	}, connectionsResource, "get", middlewareFactory, handlers.GetConnectionSchema)

	registerTypedOperation(app, api, huma.Operation{
		OperationID:   "update-connection",
		Method:        http.MethodPatch,
		Path:          connectionPath,
		Summary:       "Update a connection",
		Description:   "Applies a partial update to an existing database connection.",
		Tags:          []string{"Connections"},
		DefaultStatus: http.StatusOK,
		Errors: protectedErrors(
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
			http.StatusInternalServerError,
		),
		SkipValidateBody: true,
	}, connectionsResource, "patch", middlewareFactory, handlers.UpdateConnection)
	replaceJSONRequestSchema(
		api,
		http.MethodPatch,
		connectionPath,
		"Connection fields to update; omitted fields remain unchanged.",
		schemaFor[model.ConnectionUpdateInput](api, "ConnectionUpdateInput"),
	)

	registerTypedOperation(app, api, huma.Operation{
		OperationID:   "delete-connection",
		Method:        http.MethodDelete,
		Path:          connectionPath,
		Summary:       "Delete a connection",
		Description:   "Deletes a connection when it has no active jobs.",
		Tags:          []string{"Connections"},
		DefaultStatus: http.StatusNoContent,
		Errors: protectedErrors(
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
			http.StatusInternalServerError,
		),
	}, connectionsResource, "delete", middlewareFactory, handlers.DeleteConnection)
}

func schemaErrorLocation(schemaError model.SchemaValidationError) string {
	parts := []string{"body", "mappedFields"}
	if schemaError.DataSourceID != "" {
		parts = append(parts, schemaError.DataSourceID)
	}

	if schemaError.Table != "" {
		parts = append(parts, schemaError.Table)
	}

	if schemaError.Field != "" {
		parts = append(parts, schemaError.Field)
	}

	return strings.Join(parts, ".")
}
