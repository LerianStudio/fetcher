package command

import (
	"context"
	"fmt"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"

	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type AssignConnection struct {
	connRepo connRepo.Repository
}

func NewAssignConnection(connectionRepo connRepo.Repository) *AssignConnection {
	return &AssignConnection{connRepo: connectionRepo}
}

func (s *AssignConnection) Execute(ctx context.Context, connectionID uuid.UUID, productName string) (*model.Connection, error) {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.assign_connection_to_product")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
		attribute.String("app.request.product_name", productName),
	)

	if productName == "" {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "empty product name", constant.ErrBadRequest)

		return nil, pkg.ValidateBadRequestFieldsError(
			map[string]string{"product_name": "product name is required"},
			nil,
			"connection",
			nil,
		)
	}

	// Validate connection exists
	conn, err := s.connRepo.FindByID(ctx, connectionID)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find connection by ID", err)
		return nil, fmt.Errorf("failed to find connection by id: %w", err)
	}

	if conn == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection not found", constant.ErrEntityNotFound)
		return nil, pkg.ValidateBusinessError(constant.ErrEntityNotFound, "connection")
	}

	// Persist the assignment (atomic guard: repo uses product_name: {$eq: ""} filter)
	updated, err := s.connRepo.AssignProductName(ctx, connectionID, productName)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to assign product to connection", err)
		return nil, fmt.Errorf("failed to assign product to connection: %w", err)
	}

	if updated == nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Connection already assigned to a product", constant.ErrConnectionAlreadyAssigned)
		return nil, pkg.ValidateBusinessError(constant.ErrConnectionAlreadyAssigned, "connection")
	}

	logger.Log(ctx, libLog.LevelInfo, "connection assigned to product",
		libLog.String("connection_id", connectionID.String()),
		libLog.String("product_name", productName),
	)

	return updated, nil
}
