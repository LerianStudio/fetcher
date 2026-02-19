package query

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/net/http"
	productRepo "github.com/LerianStudio/fetcher/pkg/ports/product"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type ListProducts struct {
	productRepo productRepo.Repository
}

func NewListProducts(repo productRepo.Repository) *ListProducts {
	return &ListProducts{productRepo: repo}
}

func (s *ListProducts) Execute(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) (*model.Pagination, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.list_products")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
	)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", filters)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert filters to JSON string", err)
	}

	list, totalCount, err := s.productRepo.List(ctx, organizationID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	productResp := make([]*model.ProductResponse, 0, len(list))
	for _, p := range list {
		productResp = append(productResp, model.NewProductResponseFrom(p))
	}

	pagination := &model.Pagination{
		Limit: filters.Limit,
		Page:  filters.Page,
	}
	pagination.SetItems(productResp)
	pagination.SetTotal(int(totalCount))

	return pagination, nil
}
