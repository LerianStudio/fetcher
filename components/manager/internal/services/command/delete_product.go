package command

import (
	"context"
	"fmt"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"

	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	productRepo "github.com/LerianStudio/fetcher/pkg/ports/product"
	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

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
		libOpentelemetry.HandleSpanError(&span, "Failed to find product by ID", err)
		return fmt.Errorf("failed to find product by id: %w", err)
	}

	if current == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(&span, "Product not found", constant.ErrEntityNotFound)

		return pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"product",
		)
	}

	count, err := s.connRepo.CountByProduct(ctx, organizationID, productID)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to count connections by product", err)
		return fmt.Errorf("failed to count connections by product: %w", err)
	}

	if count > 0 {
		logger.Infof("Delete blocked: product id=%s has %d active connections", productID, count)
		libOpentelemetry.HandleSpanBusinessErrorEvent(&span, "Product has active connections", constant.ErrProductHasConnections)

		return pkg.ValidateBusinessError(
			constant.ErrProductHasConnections,
			"product",
		)
	}

	if err := s.productRepo.Delete(ctx, productID, organizationID, time.Now().UTC()); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to delete product", err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	logger.Infof("Deleted product id=%s org=%s", productID, organizationID)

	return nil
}
