package command

import (
	"context"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"

	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	productRepo "github.com/LerianStudio/fetcher/pkg/mongodb/product"
	"github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type DeleteProduct struct {
	productRepo productRepo.Repository
	connRepo    connRepo.Repository
}

func NewDeleteProduct(repo productRepo.Repository, connectionRepo connRepo.Repository) *DeleteProduct {
	return &DeleteProduct{productRepo: repo, connRepo: connectionRepo}
}

func (s *DeleteProduct) Execute(ctx context.Context, organizationID, productID uuid.UUID) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.delete_product")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.product_id", productID.String()),
	)

	current, err := s.productRepo.FindByID(ctx, productID, organizationID)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find product by ID", err)
		return err
	}

	if current == nil {
		libOpentelemetry.HandleSpanError(span, "Product not found", constant.ErrEntityNotFound)

		return pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"product",
		)
	}

	count, err := s.connRepo.CountByProduct(ctx, organizationID, productID)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to count connections by product", err)
		return err
	}

	if count > 0 {
		logger.Log(ctx, libLog.LevelInfo, "delete product blocked by active connections",
			libLog.String("product_id", productID.String()),
			libLog.String("organization_id", organizationID.String()),
			libLog.Any("active_connections", count),
		)
		libOpentelemetry.HandleSpanError(span, "Product has active connections", constant.ErrProductHasConnections)

		return pkg.ValidateBusinessError(
			constant.ErrProductHasConnections,
			"product",
		)
	}

	if err := s.productRepo.Delete(ctx, productID, organizationID, time.Now().UTC()); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to delete product", err)
		return err
	}

	logger.Log(ctx, libLog.LevelInfo, "deleted product",
		libLog.String("product_id", productID.String()),
		libLog.String("organization_id", organizationID.String()),
	)

	return nil
}
