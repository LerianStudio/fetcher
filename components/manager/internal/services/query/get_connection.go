package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"

	"github.com/LerianStudio/lib-commons/v2/commons"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type GetConnection struct {
	connRepo connRepo.Repository
}

func NewGetConnection(connRepo connRepo.Repository) *GetConnection {
	return &GetConnection{connRepo: connRepo}
}

func (s *GetConnection) Execute(ctx context.Context, organizationID, connectionID uuid.UUID) (*model.Connection, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)
	ctx, span := tracer.Start(ctx, "service.get_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	current, err := s.connRepo.FindByID(ctx, connectionID, organizationID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, pkg.EntityNotFoundError{
			EntityType: "connection",
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		}
	}

	return current, nil
}
