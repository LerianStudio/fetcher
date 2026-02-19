package query

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	productRepo "github.com/LerianStudio/fetcher/pkg/ports/product"
	"github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type ListConnections struct {
	connRepo    connRepo.Repository
	productRepo productRepo.Repository
}

func NewListConnections(connectionRepo connRepo.Repository, prodRepo productRepo.Repository) *ListConnections {
	return &ListConnections{connRepo: connectionRepo, productRepo: prodRepo}
}

func (s *ListConnections) Execute(ctx context.Context, organizationID uuid.UUID, productID *uuid.UUID, filters http.QueryHeader) (*model.Pagination, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.list_connections")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
	)

	if productID != nil {
		span.SetAttributes(attribute.String("app.request.product_id", productID.String()))

		prod, err := s.productRepo.FindByID(ctx, *productID, organizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to find product by id: %w", err)
		}

		if prod == nil {
			return nil, pkg.ValidateBusinessError(constant.ErrEntityNotFound, "product")
		}

		filters.ProductID = productID
	}

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", filters)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert fetcher input to JSON string", err)
	}

	list, totalCount, err := s.connRepo.List(ctx, organizationID, filters)
	if err != nil {
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
