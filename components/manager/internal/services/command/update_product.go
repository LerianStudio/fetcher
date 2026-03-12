package command

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"

	productRepo "github.com/LerianStudio/fetcher/pkg/mongodb/product"
	"github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type UpdateProduct struct {
	productRepo productRepo.Repository
}

func NewUpdateProduct(repo productRepo.Repository) *UpdateProduct {
	return &UpdateProduct{productRepo: repo}
}

func (s *UpdateProduct) Execute(ctx context.Context, organizationID, productID uuid.UUID, input model.ProductUpdateInput) (*model.Product, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.update_product")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.product_id", productID.String()),
	)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", input, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert product input to JSON string", err)
	}

	current, err := s.productRepo.FindByID(ctx, productID, organizationID)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find product by ID", err)
		return nil, err
	}

	if current == nil {
		libOpentelemetry.HandleSpanError(span, "Product not found", constant.ErrEntityNotFound)

		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"product",
		)
	}

	if errPatch := current.ApplyPatch(
		input.Name,
		input.Description,
		input.Metadata,
	); errPatch != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to apply patch to product", errPatch)
		return nil, errPatch
	}

	updated, err := s.productRepo.Update(ctx, current)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to update product in repository", err)
		return nil, err
	}

	if updated == nil {
		libOpentelemetry.HandleSpanError(span, "Product not found after update (race condition)", constant.ErrEntityNotFound)

		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"product",
		)
	}

	logger.Log(ctx, libLog.LevelInfo, "updated product",
		libLog.String("product_id", productID.String()),
		libLog.String("organization_id", organizationID.String()),
	)

	return updated, nil
}
