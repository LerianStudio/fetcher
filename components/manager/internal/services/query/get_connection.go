package query

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type GetConnection struct {
	connRepo connRepo.Repository
}

func NewGetConnection(connectionRepo connRepo.Repository) *GetConnection {
	return &GetConnection{connRepo: connectionRepo}
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
		libOpentelemetry.HandleSpanError(&span, "Failed to find connection by ID", err)
		return nil, fmt.Errorf("failed to find connection by id: %w", err)
	}

	if current == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	return current, nil
}
