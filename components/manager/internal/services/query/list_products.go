package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/model"
	productRepo "github.com/LerianStudio/fetcher/pkg/mongodb/product"
	"github.com/LerianStudio/fetcher/pkg/net/http"

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

func (s *ListProducts) Execute(ctx context.Context, organizationID uuid.UUID, filters http.QueryHeader) ([]*model.Product, error) {
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

	list, err := s.productRepo.List(ctx, organizationID, filters)
	if err != nil {
		return nil, err
	}

	if list == nil {
		return []*model.Product{}, nil
	}

	return list, nil
}
