package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	productRepo "github.com/LerianStudio/fetcher/pkg/mongodb/product"
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

func (s *ListConnections) Execute(ctx context.Context, organizationID uuid.UUID, productID *uuid.UUID, filters http.QueryHeader) ([]*model.Connection, int64, error) {
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
			return nil, 0, err
		}

		if prod == nil {
			return nil, 0, pkg.ValidateBusinessError(constant.ErrEntityNotFound, "product")
		}

		filters.ProductID = productID
	}

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", filters)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert fetcher input to JSON string", err)
	}

	list, totalCount, err := s.connRepo.List(ctx, organizationID, filters)
	if err != nil {
		return nil, 0, err
	}

	if list == nil {
		return []*model.Connection{}, totalCount, nil
	}

	return list, totalCount, nil
}
