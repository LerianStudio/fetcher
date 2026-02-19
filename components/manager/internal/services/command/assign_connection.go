package command

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"

	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	productRepo "github.com/LerianStudio/fetcher/pkg/ports/product"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type AssignConnection struct {
	connRepo    connRepo.Repository
	productRepo productRepo.Repository
}

func NewAssignConnection(connectionRepo connRepo.Repository, prodRepo productRepo.Repository) *AssignConnection {
	return &AssignConnection{connRepo: connectionRepo, productRepo: prodRepo}
}

func (s *AssignConnection) Execute(ctx context.Context, organizationID, connectionID, productID uuid.UUID) (*model.Connection, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.assign_connection_to_product")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.String("app.request.connection_id", connectionID.String()),
		attribute.String("app.request.product_id", productID.String()),
	)

	// Validate product exists
	prod, err := s.productRepo.FindByID(ctx, productID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find product by id: %w", err)
	}

	if prod == nil {
		return nil, pkg.ValidateBusinessError(constant.ErrEntityNotFound, "product")
	}

	// Validate connection exists
	conn, err := s.connRepo.FindByID(ctx, connectionID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find connection by id: %w", err)
	}

	if conn == nil {
		return nil, pkg.ValidateBusinessError(constant.ErrEntityNotFound, "connection")
	}

	// Persist the assignment (atomic guard: repo uses product_id: {$eq: nil} filter)
	updated, err := s.connRepo.AssignProduct(ctx, connectionID, organizationID, productID)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to assign product to connection", err)
		return nil, fmt.Errorf("failed to assign product to connection: %w", err)
	}

	if updated == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(&span, "Connection already assigned to a product", constant.ErrConnectionAlreadyAssigned)
		return nil, pkg.ValidateBusinessError(constant.ErrConnectionAlreadyAssigned, "connection")
	}

	logger.Infof("connection assigned to product connection_id=%s product_id=%s org=%s", connectionID, productID, organizationID)

	return updated, nil
}
