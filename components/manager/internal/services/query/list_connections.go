package query

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/net/http"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type ListConnections struct {
	connRepo connRepo.Repository
}

func NewListConnections(connectionRepo connRepo.Repository) *ListConnections {
	return &ListConnections{connRepo: connectionRepo}
}

func (s *ListConnections) Execute(ctx context.Context, organizationID uuid.UUID, productName string, filters http.QueryHeader) (*model.Pagination, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.list_connections")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
	)

	if productName != "" {
		span.SetAttributes(attribute.String("app.request.product_name", productName))

		filters.ProductName = productName
	}

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", filters)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert fetcher input to JSON string", err)
	}

	list, totalCount, err := s.connRepo.List(ctx, organizationID, filters)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to list connections", err)
		return nil, fmt.Errorf("failed to list connections: %w", err)
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
