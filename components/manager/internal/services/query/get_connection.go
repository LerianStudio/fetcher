package query

import (
	"context"
	"fmt"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/resolver"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type GetConnection struct {
	connRepo connRepo.Repository
	resolver resolver.ConnectionResolver          // nil-safe
	registry *resolver.InternalDatasourceRegistry // nil-safe
	engine   *engine.Engine                       // nil-safe scope authority
}

func NewGetConnection(connectionRepo connRepo.Repository, connResolver resolver.ConnectionResolver, dsRegistry *resolver.InternalDatasourceRegistry, eng *engine.Engine) *GetConnection {
	return &GetConnection{connRepo: connectionRepo, resolver: connResolver, registry: dsRegistry, engine: eng}
}

func (s *GetConnection) Execute(ctx context.Context, connectionID uuid.UUID) (*model.Connection, error) {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.get_connection")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.connection_id", connectionID.String()),
	)

	// Route the per-request tenant-scope authority through the Engine before any
	// read. The Manager keeps its UUID identity + internal-datasource registry
	// resolution; the Engine owns only the scope rule.
	if err := authorizeConnectionAccess(ctx, s.engine); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to authorize tenant scope", err)
		return nil, err
	}

	// Check if this is an internal datasource (deterministic UUID per tenant)
	if s.registry != nil && s.resolver != nil {
		tenantID := tmcore.GetTenantIDContext(ctx)
		if configName, _, found := s.registry.FindConfigByID(connectionID, tenantID); found {
			resolved, resolveErr := s.resolver.ResolveInternalByConfigName(ctx, configName)
			if resolveErr != nil {
				libOpentelemetry.HandleSpanError(span, "failed to resolve internal datasource", resolveErr)
				return nil, fmt.Errorf("failed to resolve internal datasource '%s': %w", configName, resolveErr)
			}

			if resolved == nil {
				return nil, pkg.ValidateBusinessError(
					constant.ErrEntityNotFound,
					"connection",
					connectionID,
				)
			}

			return resolved, nil
		}
	}

	// Fallback to MongoDB lookup for external connections
	current, err := s.connRepo.FindByID(ctx, connectionID)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find connection by ID", err)
		return nil, fmt.Errorf("failed to find connection by id: %w", err)
	}

	if current == nil {
		return nil, pkg.ValidateBusinessError(
			constant.ErrEntityNotFound,
			"connection",
		)
	}

	return current, nil
}
