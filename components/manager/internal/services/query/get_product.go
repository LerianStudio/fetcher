package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	productRepo "github.com/LerianStudio/fetcher/pkg/mongodb/product"

	"github.com/LerianStudio/lib-commons/v2/commons"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type GetProduct struct {
	productRepo productRepo.Repository
}

func NewGetProduct(repo productRepo.Repository) *GetProduct {
	return &GetProduct{productRepo: repo}
}

func (s *GetProduct) Execute(ctx context.Context, organizationID, productID uuid.UUID) (*model.Product, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.get_product")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.product_id", productID.String()),
	)

	current, err := s.productRepo.FindByID(ctx, productID, organizationID)
	if err != nil {
		return nil, err
	}

	if current == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"product",
		)
	}

	return current, nil
}
