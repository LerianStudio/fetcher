package command

import (
	"context"
	"fmt"

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

type CreateProduct struct {
	productRepo productRepo.Repository
}

func NewCreateProduct(repo productRepo.Repository) *CreateProduct {
	return &CreateProduct{productRepo: repo}
}

func (s *CreateProduct) Execute(ctx context.Context, organizationID uuid.UUID, input model.ProductInput) (*model.Product, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.create_product")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
	)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", input, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert product input to JSON string", err)
	}

	product, err := model.NewProduct(
		organizationID,
		input.Code,
		input.Name,
		input.Description,
		input.Metadata,
	)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to create product entity", err)
		return nil, err
	}

	existing, errRepo := s.productRepo.FindByCode(ctx, product.Code, organizationID)
	if errRepo != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to check existing product by code", errRepo)
		return nil, errRepo
	}

	if existing != nil {
		libOpentelemetry.HandleSpanError(span, "Product with this code already exists", constant.ErrEntityConflict)

		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityConflict,
			"product",
		)
	}

	created, err := s.productRepo.Create(ctx, product)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to create product in repository", err)
		return nil, err
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Created product id=%s code=%s org=%s", created.ID, created.Code, organizationID))

	return created, nil
}
