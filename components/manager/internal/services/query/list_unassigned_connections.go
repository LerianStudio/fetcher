package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	"github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/LerianStudio/lib-commons/v4/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type ListUnassignedConnections struct {
	connRepo connRepo.Repository
}

func NewListUnassignedConnections(connectionRepo connRepo.Repository) *ListUnassignedConnections {
	return &ListUnassignedConnections{connRepo: connectionRepo}
}

func (s *ListUnassignedConnections) Execute(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) ([]*model.Connection, int64, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.list_unassigned_connections")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
	)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", filters, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert fetcher input to JSON string", err)
	}

	list, totalCount, err := s.connRepo.ListUnassigned(ctx, organizationID, filters)
	if err != nil {
		return nil, 0, err
	}

	if list == nil {
		return []*model.Connection{}, totalCount, nil
	}

	return list, totalCount, nil
}
