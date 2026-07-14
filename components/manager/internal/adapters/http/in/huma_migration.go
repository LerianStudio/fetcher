package in

import (
	"context"
	"fmt"
	"net/http"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/v2/pkg/net/http"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

func bindMigrationHandlers(handlers *OperationHandlers, handler *MigrationHandler) {
	if handlers == nil {
		return
	}

	var (
		listUnassigned func(context.Context, httpUtils.QueryHeader) (*model.Pagination, error)
		assign         func(context.Context, uuid.UUID, string) (*model.Connection, error)
	)

	if handler != nil {
		if handler.ListUnassignedQry != nil {
			listUnassigned = handler.ListUnassignedQry.Execute
		}

		if handler.AssignCmd != nil {
			assign = handler.AssignCmd.Execute
		}
	}

	bindMigrationExecutors(handlers, listUnassigned, assign)
}

func bindMigrationExecutors(
	handlers *OperationHandlers,
	listUnassigned func(context.Context, httpUtils.QueryHeader) (*model.Pagination, error),
	assign func(context.Context, uuid.UUID, string) (*model.Connection, error),
) {
	if handlers == nil {
		return
	}

	handlers.ListUnassignedConnections = func(
		ctx context.Context,
		_ *emptyInput,
	) (*listConnectionsOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.UserContext()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.list_unassigned_connections")
		defer span.End()

		fiberCtx.SetUserContext(ctx)
		span.SetAttributes(attribute.String("app.request.request_id", reqID))

		query, err := httpUtils.ValidateParameters(fiberCtx.Queries())
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to validate query parameters", err)
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to validate query parameters, Error: %s", err.Error()))

			return nil, httpUtils.MapError(err)
		}

		if listUnassigned == nil {
			err := migrationInternalError("list unassigned connections query is not configured")
			logger.Log(ctx, libLog.LevelError, err.Error())
			libOpentelemetry.HandleSpanError(span, "list unassigned connections query is not configured", err)

			return nil, mapServiceError(err)
		}

		pagination, err := listUnassigned(ctx, *query)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute list unassigned connections query, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to list unassigned connections", err)

			return nil, mapServiceError(err)
		}

		if pagination == nil {
			err := migrationInternalError("list unassigned connections query returned no result")
			logger.Log(ctx, libLog.LevelError, err.Error())
			libOpentelemetry.HandleSpanError(span, "list unassigned connections query returned no result", err)

			return nil, mapServiceError(err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("unassigned connections listed count=%d", pagination.Total))

		return &listConnectionsOutput{Body: connectionPageFrom(pagination)}, nil
	}

	handlers.AssignConnection = func(ctx context.Context, input *idInput) (*connectionOutput, error) {
		fiberCtx, err := fiberContext(ctx)
		if err != nil {
			return nil, err
		}

		ctx = fiberCtx.UserContext()
		logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

		ctx, span := tracer.Start(ctx, "handler.assign_connection_to_product")
		defer span.End()

		fiberCtx.SetUserContext(ctx)

		connectionID, err := parseID(input.ID, "connection")
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "invalid connection id parameter", err)

			return nil, err
		}

		span.SetAttributes(
			attribute.String("app.request.request_id", reqID),
			attribute.String("app.request.connection_id", connectionID.String()),
		)

		productName, err := httpUtils.GetRequiredProductName(fiberCtx)
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "missing or invalid product name", err)

			return nil, httpUtils.MapError(err)
		}

		span.SetAttributes(attribute.String("app.request.product_name", productName))

		if assign == nil {
			err := migrationInternalError("assign connection command is not configured")
			logger.Log(ctx, libLog.LevelError, err.Error())
			libOpentelemetry.HandleSpanError(span, "assign connection command is not configured", err)

			return nil, mapServiceError(err)
		}

		connection, err := assign(ctx, connectionID, productName)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to assign connection to product, Error: %s", err.Error()))
			libOpentelemetry.HandleSpanError(span, "failed to assign connection to product", err)

			return nil, mapServiceError(err)
		}

		if connection == nil {
			err := migrationInternalError("assign connection command returned no result")
			logger.Log(ctx, libLog.LevelError, err.Error())
			libOpentelemetry.HandleSpanError(span, "assign connection command returned no result", err)

			return nil, mapServiceError(err)
		}

		response := model.NewConnectionResponseFrom(connection)

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection assigned to product connection_id=%s product_name=%s", connectionID, productName))

		return &connectionOutput{Body: *response}, nil
	}
}

func migrationInternalError(message string) error {
	return pkg.InternalServerError{
		Code:    constant.ErrInternalServer.Error(),
		Message: message,
	}
}

func registerMigrationOperations(
	app *fiber.App,
	api huma.API,
	handlers OperationHandlers,
	middlewareFactory OperationMiddlewareFactory,
) {
	registerTypedOperation(
		app,
		api,
		huma.Operation{
			OperationID:        "list-unassigned-connections",
			Method:             http.MethodGet,
			Path:               "/v1/management/connections/unassigned",
			Summary:            "List unassigned connections",
			Description:        "List connections that have no product assigned, useful for migration purposes.",
			Tags:               []string{"Migration"},
			Parameters:         migrationListParameters(),
			Errors:             protectedErrors(http.StatusBadRequest, http.StatusInternalServerError),
			SkipValidateParams: true,
		},
		connectionsResource,
		"get",
		middlewareFactory,
		handlers.ListUnassignedConnections,
	)

	registerTypedOperation(
		app,
		api,
		huma.Operation{
			OperationID: "assign-connection-to-product",
			Method:      http.MethodPost,
			Path:        "/v1/management/connections/{id}/assign",
			Summary:     "Assign connection to product",
			Description: "Associate an unassigned connection to a product. This is a one-time, irreversible operation for migration purposes. The product name must be provided via the X-Product-Name header.",
			Tags:        []string{"Migration"},
			Parameters:  []*huma.Param{productNameParameter(true)},
			Errors: protectedErrors(
				http.StatusBadRequest,
				http.StatusNotFound,
				http.StatusConflict,
				http.StatusInternalServerError,
			),
			SkipValidateParams: true,
		},
		connectionsResource,
		"post",
		middlewareFactory,
		handlers.AssignConnection,
	)
}

func migrationListParameters() []*huma.Param {
	params := paginationParameters()

	return append(params,
		&huma.Param{
			Name:        "startDate",
			In:          "query",
			Description: "Inclusive creation date in YYYY-MM-DD format.",
			Schema:      &huma.Schema{Type: "string", Examples: []any{"2026-01-01"}},
		},
		&huma.Param{
			Name:        "endDate",
			In:          "query",
			Description: "Inclusive final creation date in YYYY-MM-DD format.",
			Schema:      &huma.Schema{Type: "string", Examples: []any{"2026-01-31"}},
		},
	)
}
