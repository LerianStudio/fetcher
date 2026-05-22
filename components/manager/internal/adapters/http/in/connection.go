package in

import (
	"fmt"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/query"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type ConnectionHandler struct {
	CreateCmd           *command.CreateConnection
	UpdateCmd           *command.UpdateConnection
	DeleteCmd           *command.DeleteConnection
	GetQuery            *query.GetConnection
	ListQuery           *query.ListConnections
	TestQuery           *query.TestConnection
	ValidateSchemaQuery *query.ValidateSchema
	GetSchemaQuery      *query.GetConnectionSchema
}

func NewConnectionHandler(
	createCmd *command.CreateConnection,
	updateCmd *command.UpdateConnection,
	deleteCmd *command.DeleteConnection,
	getQuery *query.GetConnection,
	listQuery *query.ListConnections,
	testQuery *query.TestConnection,
	validateSchemaQuery *query.ValidateSchema,
	getSchemaQuery *query.GetConnectionSchema,
) *ConnectionHandler {
	return &ConnectionHandler{
		CreateCmd:           createCmd,
		UpdateCmd:           updateCmd,
		DeleteCmd:           deleteCmd,
		GetQuery:            getQuery,
		ListQuery:           listQuery,
		TestQuery:           testQuery,
		ValidateSchemaQuery: validateSchemaQuery,
		GetSchemaQuery:      getSchemaQuery,
	}
}

// CreateConnection is a method that creates a database connection.
//
//	@Summary		Create connection
//	@Description	Create a new database connection for the organization. The X-Product-Name header is required and identifies the product for this connection. ConfigName must be unique per organization (and per product when assigned); password is encrypted and a UUID is generated on creation.
//	@Tags			Connections
//	@Accept			json
//	@Produce		json
//	@Param			Authorization		header		string					false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Product-Name		header		string					true	"Product name (required, non-empty)"
//	@Param			connection			body		model.ConnectionInput	true	"Connection payload"
//	@Success		201					{object}	map[string]string		"Created connection identifier"
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		409					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections [post]
func (h *ConnectionHandler) CreateConnection(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.create_connection")
	defer span.End()

	c.SetUserContext(ctx)

	productName, err := httpUtils.GetRequiredProductName(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "missing or invalid product name", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.product_name", productName),
	)

	var request model.ConnectionInput
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanError(span, "failed to parse payload", errParser)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        errParser,
		})
	}

	if request.IsEmpty() {
		err := pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "empty request body",
		}

		libOpentelemetry.HandleSpanError(span, "empty request body", err)

		return httpUtils.WithError(c, err)
	}

	// Struct-tag validation (required, hostname|ip, safe_host, ...). Without
	// this call the `safe_host` tag is dead code — BodyParser does not invoke
	// the validator. See docs/PROJECT_RULES.md § "Defense-in-Depth: Two Layers".
	if err := httpUtils.ValidateStruct(&request); err != nil {
		libOpentelemetry.HandleSpanError(span, "request validation failed", err)

		return httpUtils.WithError(c, err)
	}

	conn, err := h.CreateCmd.Execute(ctx, request, productName)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute create connection command, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to create connection", err)

		return httpUtils.WithError(c, err)
	}

	resp := model.NewConnectionResponseFrom(conn)
	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection created id=%s", resp.ID))

	return httpUtils.Created(c, resp)
}

// ListConnections is a method that retrieves connections with optional pagination and filters.
//
//	@Summary		List connections
//	@Description	List connections with pagination and filters. When X-Product-Name is provided, returns only connections associated with that product.
//	@Tags			Connections
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Product-Name		header		string	false	"Product name. When provided, filters connections by product."
//	@Param			page				query		int		false	"Page number (minimum 1)"	default(1)
//	@Param			limit				query		int		false	"Page size (default 50, max 1000)"	default(50)
//	@Param			sortOrder			query		string	false	"Sort order"											Enums(asc, desc)	default(desc)
//	@Param			type				query		string	false	"Filter by database type"								Enums(ORACLE, SQL_SERVER, POSTGRESQL, MONGODB, MYSQL)
//	@Param			host				query		string	false	"Filter by host"
//	@Param			databaseName		query		string	false	"Filter by database name"
//	@Param			startDate			query		string	false	"Filter by start date (YYYY-MM-DD)"
//	@Param			endDate				query		string	false	"Filter by end date (YYYY-MM-DD)"
//	@Success		200					{object}	model.Pagination{items=[]model.ConnectionResponse,page=int,limit=int,total=int}
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections [get]
func (h *ConnectionHandler) ListConnections(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.list_connection")
	defer span.End()

	c.SetUserContext(ctx)

	productName, err := httpUtils.GetProductName(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "invalid product name", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	if productName != "" {
		span.SetAttributes(attribute.String("app.request.product_name", productName))
	}

	headerParams, err := httpUtils.ValidateParameters(c.Queries())
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to validate query parameters", err)
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to validate query parameters, Error: %s", err.Error()))

		return httpUtils.WithError(c, err)
	}

	pagination, err := h.ListQuery.Execute(ctx, productName, *headerParams)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute list connections query, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to list connections", err)

		return httpUtils.WithError(c, err)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connections listed count=%d", pagination.Total))

	return httpUtils.OK(c, pagination)
}

// GetConnection is a method that retrieves a connection by ID.
//
//	@Summary		Get connection
//	@Description	Get connection details by ID for the given organization.
//	@Tags			Connections
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			id					path		string	true	"Connection ID"
//	@Success		200					{object}	model.ConnectionResponse
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections/{id} [get]
func (h *ConnectionHandler) GetConnection(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.get_connection")
	defer span.End()

	c.SetUserContext(ctx)

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid connection id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

	conn, err := h.GetQuery.Execute(ctx, id)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute get connection query, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to get connection", err)

		return httpUtils.WithError(c, err)
	}

	resp := model.NewConnectionResponseFrom(conn)

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection retrieved id=%s", id))

	return httpUtils.OK(c, resp)
}

// TestConnection tests a database connection by attempting to connect and disconnect.
//
//	@Summary		Test connection
//	@Description	Test the configured connection by establishing and closing a connection.
//	@Tags			Connections
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			id					path		string	true	"Connection ID"
//	@Success		200					{object}	map[string]any	"Connection test result"
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		429					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections/{id}/test [post]
func (h *ConnectionHandler) TestConnection(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.test_connection")
	defer span.End()

	c.SetUserContext(ctx)

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "invalid connection id parameter", err)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid connection id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

	resp, err := h.TestQuery.Execute(ctx, id)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute test connection query, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to test connection", err)

		return httpUtils.WithError(c, err)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection test successful id=%s latency_ms=%d", id, resp.LatencyMs))

	return httpUtils.OK(c, resp)
}

// UpdateConnection is a method that partially updates a connection.
//
//	@Summary		Update connection
//	@Description	Apply a partial update to a connection. Only include fields you want to change. Returns 409 if there is any active job.
//	@Tags			Connections
//	@Accept			json
//	@Produce		json
//	@Param			Authorization		header		string						false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			id					path		string						true	"Connection ID"
//	@Param			connection			body		model.ConnectionUpdateInput	true	"Fields to update (only include fields you want to change)"
//	@Success		200					{object}	model.ConnectionResponse
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		409					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections/{id} [patch]
func (h *ConnectionHandler) UpdateConnection(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.update_connection")
	defer span.End()

	c.SetUserContext(ctx)

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid connection id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

	var request model.ConnectionUpdateInput
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanError(span, "failed to parse payload", errParser)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        errParser,
		})
	}

	if request.IsEmpty() {
		err := pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "empty request body",
		}

		libOpentelemetry.HandleSpanError(span, "empty request body", err)

		return httpUtils.WithError(c, err)
	}

	// Struct-tag validation (omitempty, hostname|ip, safe_host, ...). Required
	// so the `safe_host` tag in ConnectionUpdateInput is actually enforced.
	if err := httpUtils.ValidateStruct(&request); err != nil {
		libOpentelemetry.HandleSpanError(span, "request validation failed", err)

		return httpUtils.WithError(c, err)
	}

	conn, err := h.UpdateCmd.Execute(ctx, id, request)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute update connection command, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to update connection", err)

		return httpUtils.WithError(c, err)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection updated id=%s", id))

	return httpUtils.OK(c, model.NewConnectionResponseFrom(conn))
}

// DeleteConnection is a method that performs a soft delete of a connection.
//
//	@Summary		Delete connection
//	@Description	Soft delete a connection when no active jobs are running for it. Returns 409 if there is any active job.
//	@Tags			Connections
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			id					path		string	true	"Connection ID"
//	@Success		204					"No Content"
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		409					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections/{id} [delete]
func (h *ConnectionHandler) DeleteConnection(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.delete_connection")
	defer span.End()

	c.SetUserContext(ctx)

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid connection id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

	if err := h.DeleteCmd.Execute(ctx, id); err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute delete connection command, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to delete connection", err)

		return httpUtils.WithError(c, err)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection deleted id=%s", id))

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ValidateSchema validates schema references against configured datasources.
//
// Returns 200 OK with SchemaValidationResponse when all tables and fields exist.
// Returns 422 Unprocessable Entity with SchemaValidationErrorResponse when any
// datasource is not found, table doesn't exist, field is missing, or datasource is unreachable.
//
//	@Summary		Validate schema
//	@Description	Validate that tables and fields referenced in the request exist in the configured datasources. Returns 200 when validation passes, 422 when validation fails with detailed error information.
//	@Tags			Connections
//	@Accept			json
//	@Produce		json
//	@Param			Authorization		header		string							false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			request				body		model.SchemaValidationRequest	true	"Schema validation request"
//	@Success		200					{object}	model.SchemaValidationResponse			"Validation successful - all tables and fields exist"
//	@Failure		400					{object}	pkg.HTTPError							"Invalid request payload or missing headers"
//	@Failure		422					{object}	model.SchemaValidationErrorResponse		"Validation failed - schema errors found (missing tables, fields, or unreachable datasources)"
//	@Failure		500					{object}	pkg.HTTPError							"Internal server error"
//	@Router			/v1/management/connections/validate-schema [post]
func (h *ConnectionHandler) ValidateSchema(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.validate_schema")
	defer span.End()

	c.SetUserContext(ctx)

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	var request model.SchemaValidationRequest
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanError(span, "failed to parse payload", errParser)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "schema",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        errParser,
		})
	}

	// Struct-tag validation — enforces `required` on MappedFields consistently
	// with the sibling connection handlers.
	if err := httpUtils.ValidateStruct(&request); err != nil {
		libOpentelemetry.HandleSpanError(span, "request validation failed", err)

		return httpUtils.WithError(c, err)
	}

	resp, err := h.ValidateSchemaQuery.Execute(ctx, request)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute validate schema query, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to validate schema", err)

		return httpUtils.WithError(c, err)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("schema validation completed status=%s", resp.Status))

	if resp.Status == model.StatusFailure {
		return httpUtils.JSONResponse(c, fiber.StatusUnprocessableEntity, model.SchemaValidationErrorResponse{
			Title:   "Schema validation failed",
			Code:    constant.ErrSchemaValidationFailed.Error(),
			Message: resp.Message,
			Errors:  resp.Errors,
		})
	}

	return httpUtils.OK(c, resp)
}

// GetConnectionSchema retrieves the database schema for a connection.
//
//	@Summary		Get connection schema
//	@Description	Get the database schema (tables and fields) for a connection.
//	@Tags			Connections
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			id					path		string	true	"Connection ID"
//	@Success		200					{object}	model.ConnectionSchemaResponse
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections/{id}/schema [get]
func (h *ConnectionHandler) GetConnectionSchema(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.get_connection_schema")
	defer span.End()

	c.SetUserContext(ctx)

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "invalid connection id parameter", err)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid connection id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

	resp, err := h.GetSchemaQuery.Execute(ctx, id)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Failed to execute get connection schema query, Error: %s", err.Error()))
		libOpentelemetry.HandleSpanError(span, "failed to get connection schema", err)

		return httpUtils.WithError(c, err)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("connection schema retrieved id=%s tables=%d", id, len(resp.Tables)))

	return httpUtils.OK(c, resp)
}
