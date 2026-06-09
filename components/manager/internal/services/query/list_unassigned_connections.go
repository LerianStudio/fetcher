package query

import (
	"context"
	"fmt"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/net/http"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"

	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"go.opentelemetry.io/otel/attribute"
)

type ListUnassignedConnections struct {
	connRepo connRepo.Repository
}

func NewListUnassignedConnections(connectionRepo connRepo.Repository) *ListUnassignedConnections {
	return &ListUnassignedConnections{connRepo: connectionRepo}
}

func (s *ListUnassignedConnections) Execute(ctx context.Context, filters http.QueryHeader) (*model.Pagination, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.list_unassigned_connections")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", filters, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert fetcher input to JSON string", err)
	}

	list, totalCount, err := s.connRepo.ListUnassigned(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list unassigned connections: %w", err)
	}

	connResp := make([]*model.ConnectionResponse, 0, len(list))
	for _, conn := range list {
		connResp = append(connResp, model.NewConnectionResponseFrom(conn))
	}

	pagination := &model.Pagination{
		Limit: filters.Limit,
		Page:  filters.Page,
	}
	pagination.SetItems(connResp)
	pagination.SetTotal(int(totalCount))

	return pagination, nil
}
