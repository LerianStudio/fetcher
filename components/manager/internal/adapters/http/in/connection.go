package in

import (
	"time"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/connection"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/connection/command"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/connection/query"
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	domainConn "github.com/LerianStudio/fetcher/pkg/domain"
	httputils "github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type ConnectionHandler struct {
	CreateCmd *command.CreateConnection
	UpdateCmd *command.UpdateConnection
	DeleteCmd *command.DeleteConnection
	GetQuery  *query.GetConnection
	ListQuery *query.ListConnections
}

func NewConnectionHandler(
	createCmd *command.CreateConnection,
	updateCmd *command.UpdateConnection,
	deleteCmd *command.DeleteConnection,
	getQuery *query.GetConnection,
	listQuery *query.ListConnections,
) *ConnectionHandler {
	return &ConnectionHandler{
		CreateCmd: createCmd,
		UpdateCmd: updateCmd,
		DeleteCmd: deleteCmd,
		GetQuery:  getQuery,
		ListQuery: listQuery,
	}
}

func (h *ConnectionHandler) CreateConnection(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)
	ctx, span := tracer.Start(ctx, "handler.create_connection")
	defer span.End()
	c.SetUserContext(ctx)

	orgID, err := httputils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httputils.WithError(c, err)
	}

	var request ConnectionRequest
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to parse payload", errParser)
		return httputils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        errParser,
		})
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	conn, err := h.CreateCmd.Execute(ctx, orgID, request.ToCreateConnectionInput())

	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to create connection", err)
		return httputils.WithError(c, err)
	}

	resp := NewConnectionResponseFromDomain(conn)

	logger.Infof("connection created id=%s org=%s", resp.ID, orgID)
	return httputils.Created(c, fiber.Map{"id": resp.ID})
}

func (h *ConnectionHandler) ListConnections(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)
	ctx, span := tracer.Start(ctx, "handler.list_connections")
	defer span.End()
	c.SetUserContext(ctx)

	orgID, err := httputils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httputils.WithError(c, err)
	}

	input := connection.ListConnectionsInput{
		Page:         httputils.ClampNonNegative(httputils.ParseIntDefault(c.Query("page"), 0)),
		Limit:        httputils.ClampLimit(httputils.ParseIntDefault(c.Query("limit"), 50), 50, 1000),
		SortOrder:    c.Query("sortOrder", "desc"),
		Type:         c.Query("type"),
		ConfigName:   c.Query("configName"),
		Host:         c.Query("host"),
		DatabaseName: c.Query("databaseName"),
		CreatedAt:    c.Query("createdAt"),
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
	)

	conns, err := h.ListQuery.Execute(ctx, orgID, input)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to list connections", err)
		return httputils.WithError(c, err)
	}

	connResp := make([]*ConnectionResponse, 0, len(conns))
	for _, conn := range conns {
		connResp = append(connResp, NewConnectionResponseFromDomain(conn))
	}

	logger.Infof("connections listed org=%s count=%d", orgID, len(connResp))
	return httputils.OK(c, connResp)
}

func (h *ConnectionHandler) GetConnection(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)
	ctx, span := tracer.Start(ctx, "handler.get_connection")
	defer span.End()
	c.SetUserContext(ctx)

	orgID, err := httputils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httputils.WithError(c, err)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputils.WithError(c, pkg.ValidationError{
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
		attribute.String("app.request.connection_id", id.String()),
	)

	conn, err := h.GetQuery.Execute(ctx, orgID, id)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to get connection", err)
		return httputils.WithError(c, err)
	}

	resp := NewConnectionResponseFromDomain(conn)

	logger.Infof("connection retrieved id=%s org=%s", id, orgID)
	return httputils.OK(c, resp)
}

func (h *ConnectionHandler) UpdateConnection(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)
	ctx, span := tracer.Start(ctx, "handler.update_connection")
	defer span.End()
	c.SetUserContext(ctx)

	orgID, err := httputils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httputils.WithError(c, err)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    "invalid connection id",
			Err:        err,
		})
	}

	var request ConnectionRequest
	if errParser := c.BodyParser(&request); errParser != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to parse payload", errParser)
		return httputils.WithError(c, pkg.ValidationError{
			EntityType: "connection",
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Invalid payload",
			Message:    "unable to parse request body",
			Err:        errParser,
		})
	}

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", orgID.String()),
		attribute.String("app.request.connection_id", id.String()),
	)

	conn, err := h.UpdateCmd.Execute(ctx, orgID, id, request.ToCreateConnectionInput())
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to update connection", err)
		return httputils.WithError(c, err)
	}

	logger.Infof("connection updated id=%s org=%s", id, orgID)
	return httputils.OK(c, NewConnectionResponseFromDomain(conn))
}

func (h *ConnectionHandler) DeleteConnection(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)
	ctx, span := tracer.Start(ctx, "handler.delete_connection")
	defer span.End()
	c.SetUserContext(ctx)

	orgID, err := httputils.GetOrganizationID(c)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "missing or invalid org id", err)
		return httputils.WithError(c, err)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httputils.WithError(c, pkg.ValidationError{
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
		attribute.String("app.request.connection_id", id.String()),
	)

	if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to delete connection", err)
		return httputils.WithError(c, err)
	}

	logger.Infof("connection deleted id=%s org=%s", id, orgID)
	return httputils.OK(c, fiber.Map{"id": id})
}

type ConnectionResponse struct {
	ID           uuid.UUID    `json:"id"`
	ConfigName   string       `json:"configName"`
	Type         string       `json:"type"`
	Host         string       `json:"host"`
	Port         int          `json:"port"`
	DatabaseName string       `json:"databaseName"`
	Username     string       `json:"username"`
	SSL          *SSLResponse `json:"ssl,omitempty"`
	KeyVersion   string       `json:"keyVersion"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type SSLResponse struct {
	Mode string `json:"mode,omitempty"`
}

// NewConnectionResponseFromDomain maps a Connection domain entity to a ConnectionResponse DTO.
func NewConnectionResponseFromDomain(conn *domainConn.Connection) *ConnectionResponse {
	if conn == nil {
		return nil
	}
	resp := &ConnectionResponse{
		ID:           conn.ID,
		ConfigName:   conn.ConfigName,
		Type:         string(conn.Type),
		Host:         conn.Host,
		Port:         conn.Port,
		DatabaseName: conn.DatabaseName,
		Username:     conn.Username,
		KeyVersion:   conn.KeyVersion,
		CreatedAt:    conn.CreatedAt,
		UpdatedAt:    conn.UpdatedAt,
	}
	if conn.SSL != nil {
		resp.SSL = &SSLResponse{
			Mode: conn.SSL.Mode,
		}
	}
	return resp
}

type ConnectionRequest struct {
	ConfigName   string      `json:"configName"`
	Type         string      `json:"type"`
	Host         string      `json:"host"`
	Port         int         `json:"port"`
	DatabaseName string      `json:"databaseName"`
	Username     string      `json:"username"`
	Password     string      `json:"password"`
	SSL          *SSLRequest `json:"ssl,omitempty"`
}

type SSLRequest struct {
	Mode string  `json:"mode"`
	CA   *string `json:"ca"`
	Cert *string `json:"cert"`
	Key  *string `json:"key"`
}

func (c *ConnectionRequest) ToCreateConnectionInput() connection.ConnectionInput {
	var ssl *connection.SSLInput
	if c.SSL != nil {
		ssl = &connection.SSLInput{
			Mode: c.SSL.Mode,
			CA:   c.SSL.CA,
			Cert: c.SSL.Cert,
			Key:  c.SSL.Key,
		}
	}

	return connection.ConnectionInput{
		ConfigName:   c.ConfigName,
		Type:         c.Type,
		Host:         c.Host,
		Port:         c.Port,
		DatabaseName: c.DatabaseName,
		Username:     c.Username,
		Password:     c.Password,
		SSL:          ssl,
	}
}
