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

type MigrationHandler struct {
	AssignCmd         *command.AssignConnection
	ListUnassignedQry *query.ListUnassignedConnections
}

func NewMigrationHandler(
	assignCmd *command.AssignConnection,
	listUnassignedQry *query.ListUnassignedConnections,
) *MigrationHandler {
	return &MigrationHandler{
		AssignCmd:         assignCmd,
		ListUnassignedQry: listUnassignedQry,
	}
}

// ListUnassignedConnections lists connections that have no product assigned.
//
//	@Summary		List unassigned connections
//	@Description	List connections that have no product assigned, useful for migration purposes.
//	@Tags			Migration
//	@Produce		json
//	@Param			Authorization		header		string	false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string	true	"Organization ID"
//	@Param			page				query		int		false	"Page number (minimum 1)"	default(1)
//	@Param			limit				query		int		false	"Page size (default 50, max 1000)"	default(50)
//	@Param			sortOrder			query		string	false	"Sort order"	Enums(asc, desc)	default(desc)
//	@Success		200					{object}	model.Pagination{items=[]model.ConnectionResponse,page=int,limit=int,total=int}
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections/unassigned [get]
func (h *MigrationHandler) ListUnassignedConnections(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.list_unassigned_connections")
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

	conns, totalCount, err := h.ListUnassignedQry.Execute(ctx, orgID, *headerParams)
	if err != nil {
		logger.Errorf("Failed to execute list unassigned connections query, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to list unassigned connections", err)

		return httpUtils.WithError(c, err)
	}

	connResp := make([]*model.ConnectionResponse, 0, len(conns))
	for _, conn := range conns {
		connResp = append(connResp, model.NewConnectionResponseFrom(conn))
	}

	logger.Infof("unassigned connections listed org=%s count=%d", orgID, len(connResp))

	pagination.SetItems(connResp)
	pagination.SetTotal(int(totalCount))

	return httpUtils.OK(c, pagination)
}

// AssignConnectionToProduct assigns a legacy (unassigned) connection to a product.
//
//	@Summary		Assign connection to product
//	@Description	Associate an unassigned connection to a product. This is a one-time, irreversible operation for migration purposes. The productId must be provided in the request body.
//	@Tags			Migration
//	@Accept			json
//	@Produce		json
//	@Param			Authorization		header		string							false	"The authorization token in the 'Bearer access_token' format. Only required when auth plugin is enabled."
//	@Param			X-Organization-Id	header		string							true	"Organization ID"
//	@Param			id					path		string							true	"Connection ID"
//	@Param			body				model.AssignConnectionInput		true	"Assignment payload"
//	@Success		200					{object}	model.ConnectionResponse
//	@Failure		400					{object}	pkg.HTTPError
//	@Failure		404					{object}	pkg.HTTPError
//	@Failure		409					{object}	pkg.HTTPError
//	@Failure		500					{object}	pkg.HTTPError
//	@Router			/v1/management/connections/{id}/assign [post]
func (h *MigrationHandler) AssignConnectionToProduct(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "handler.assign_connection_to_product")
	defer span.End()

	c.SetUserContext(ctx)

	orgID, err := httpUtils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httpUtils.WithError(c, err)
	}

	connectionID, err := uuid.Parse(c.Params("id"))
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

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	var request model.AssignConnectionInput
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

	productID, err := uuid.Parse(request.ProductID)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "invalid product id in body", err)

		return httpUtils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "productId must be a valid UUID",
			Err:        err,
		})
	}

	span.SetAttributes(attribute.String("app.request.product_id", productID.String()))

	conn, err := h.AssignCmd.Execute(ctx, orgID, connectionID, productID)
	if err != nil {
		logger.Errorf("Failed to assign connection to product, Error: %s", err.Error())
		libOpentelemetry.HandleSpanError(&span, "failed to assign connection to product", err)

		return httpUtils.WithError(c, err)
	}

	resp := model.NewConnectionResponseFrom(conn)

	logger.Infof("connection assigned to product connection_id=%s product_id=%s org=%s", connectionID, productID, orgID)

	return httpUtils.OK(c, resp)
}
