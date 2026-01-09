package in

import (
	"github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/query"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

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
}

func NewConnectionHandler(
	createCmd *command.CreateConnection,
	updateCmd *command.UpdateConnection,
	deleteCmd *command.DeleteConnection,
	getQuery *query.GetConnection,
	listQuery *query.ListConnections,
	testQuery *query.TestConnection,
	validateSchemaQuery *query.ValidateSchema,
) *ConnectionHandler {
	return &ConnectionHandler{
		CreateCmd:           createCmd,
		UpdateCmd:           updateCmd,
		DeleteCmd:           deleteCmd,
		GetQuery:            getQuery,
		ListQuery:           listQuery,
		TestQuery:           testQuery,
		ValidateSchemaQuery: validateSchemaQuery,
	}
}

// CreateConnection is a method that creates a database connection.
//
//	@Summary		Create connection
//	@Description	Create a new database connection for the organization. ConfigName must be unique per organization; password is encrypted and a UUID is generated on creation.
//	@Tags			Connections
//	@Accept			json
//	@Produce		json
//	@Param			Authorization		header		string					false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string					true	"Organization ID"
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

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	var request model.ConnectionInput
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to parse payload", errParser)

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

		libOpentelemetry.HandleSpanError(&span, "empty request body", err)

		return httpUtils.WithError(c, err)
	}

	conn, err := h.CreateCmd.Execute(ctx, orgID, request)
	if err != nil {
		logger.Errorf("Failed to execute create connection command, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to create connection", err)

		return httpUtils.WithError(c, err)
	}

	resp := model.NewConnectionResponseFrom(conn)
	logger.Infof("connection created id=%s org=%s", resp.ID, orgID)

	return httpUtils.Created(c, resp)
}

// ListConnections is a method that retrieves connections with optional pagination and filters.
//
//	@Summary		List connections
//	@Description	List connections with pagination and filters.
//	@Tags			Connections
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string	true	"Organization ID"
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

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	headerParams, err := httpUtils.ValidateParameters(c.Queries())
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to validate query parameters", err)
		logger.Errorf("Failed to validate query parameters, Error: %s", err.Error())

		return httpUtils.WithError(c, err)
	}

	pagination := model.Pagination{
		Limit: headerParams.Limit,
		Page:  headerParams.Page,
	}

	conns, err := h.ListQuery.Execute(ctx, orgID, *headerParams)
	if err != nil {
		logger.Errorf("Failed to execute list connections query, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to list connections", err)

		return httpUtils.WithError(c, err)
	}

	connResp := make([]*model.ConnectionResponse, 0, len(conns))
	for _, conn := range conns {
		connResp = append(connResp, model.NewConnectionResponseFrom(conn))
	}

	logger.Infof("connections listed org=%s count=%d", orgID, len(connResp))

	pagination.SetItems(connResp)
	pagination.SetTotal(len(connResp))

	return httpUtils.OK(c, pagination)
}

// GetConnection is a method that retrieves a connection by ID.
//
//	@Summary		Get connection
//	@Description	Get connection details by ID for the given organization.
//	@Tags			Connections
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string	true	"Organization ID"
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

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
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

	conn, err := h.GetQuery.Execute(ctx, orgID, id)
	if err != nil {
		logger.Errorf("Failed to execute get connection query, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to get connection", err)

		return httpUtils.WithError(c, err)
	}

	resp := model.NewConnectionResponseFrom(conn)

	logger.Infof("connection retrieved id=%s org=%s", id, orgID)

	return httpUtils.OK(c, resp)
}

// TestConnection tests a database connection by attempting to connect and disconnect.
//
//	@Summary		Test connection
//	@Description	Test the configured connection by establishing and closing a connection.
//	@Tags			Connections
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string	true	"Organization ID"
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

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "invalid connection id parameter", err)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid connection id",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.connection_id", id.String()))

	resp, err := h.TestQuery.Execute(ctx, orgID, id)
	if err != nil {
		logger.Errorf("Failed to execute test connection query, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to test connection", err)

		return httpUtils.WithError(c, err)
	}

	logger.Infof("connection test successful id=%s org=%s latency_ms=%d", id, orgID, resp.LatencyMs)

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
//	@Param			X-Organization-Id	header		string						true	"Organization ID"
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

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
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
		libOpentelemetry.HandleSpanError(&span, "failed to parse payload", errParser)

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

		libOpentelemetry.HandleSpanError(&span, "empty request body", err)

		return httpUtils.WithError(c, err)
	}

	conn, err := h.UpdateCmd.Execute(ctx, orgID, id, request)
	if err != nil {
		logger.Errorf("Failed to execute update connection command, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to update connection", err)

		return httpUtils.WithError(c, err)
	}

	logger.Infof("connection updated id=%s org=%s", id, orgID)

	return httpUtils.OK(c, model.NewConnectionResponseFrom(conn))
}

// DeleteConnection is a method that performs a soft delete of a connection.
//
//	@Summary		Delete connection
//	@Description	Soft delete a connection when no active jobs are running for it. Returns 409 if there is any active job.
//	@Tags			Connections
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string	true	"Organization ID"
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

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
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

	if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
		logger.Errorf("Failed to execute delete connection command, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to delete connection", err)

		return httpUtils.WithError(c, err)
	}

	logger.Infof("connection deleted id=%s org=%s", id, orgID)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ValidateSchema validates schema references against configured datasources.
//
//	@Summary		Validate schema
//	@Description	Validate that tables and fields referenced in the request exist in the configured datasources.
//	@Tags			Connections
//	@Accept			json
//	@Produce		json
//	@Param			Authorization		header		string							false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string							true	"Organization ID"
//	@Param			request				body		model.SchemaValidationRequest	true	"Schema validation request"
//	@Success		200					{object}	model.SchemaValidationResponse
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections/validate-schema [post]
func (h *ConnectionHandler) ValidateSchema(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.validate_schema")
	defer span.End()

	c.SetUserContext(ctx)

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	var request model.SchemaValidationRequest
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to parse payload", errParser)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "schema",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        errParser,
		})
	}

	resp, err := h.ValidateSchemaQuery.Execute(ctx, orgID, request)
	if err != nil {
		logger.Errorf("Failed to execute validate schema query, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to validate schema", err)

		return httpUtils.WithError(c, err)
	}

	logger.Infof("schema validation completed org=%s status=%s", orgID, resp.Status)

	return httpUtils.OK(c, resp)
}
