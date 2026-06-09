package query

import (
	"context"
	"fmt"

	observability "github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/resolver"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type GetConnection struct {
	resolver resolver.ConnectionResolver          // nil-safe
	registry *resolver.InternalDatasourceRegistry // nil-safe
	engine   *engine.Engine                       // scope authority + ID-addressed read persistence
}

func NewGetConnection(connResolver resolver.ConnectionResolver, dsRegistry *resolver.InternalDatasourceRegistry, eng *engine.Engine) *GetConnection {
	return &GetConnection{resolver: connResolver, registry: dsRegistry, engine: eng}
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
	// read. This is the single scope gate for the internal-datasource branch
	// (which is a resolver concern, never persisted); the external read below
	// routes its PERSISTENCE through the Engine's ID-addressed op, which also
	// re-validates the scope as part of resolving through the store.
	if err := connectioncompat.AuthorizeAccess(ctx, s.engine); err != nil {
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

	// Fallback to the external connection read, routed through the Engine's
	// ID-addressed op (Engine tenant-scope + ConnectionStore.FindByID ->
	// repo.FindByID). The rich record round-trips through the opaque host payload.
	current, err := connectioncompat.FindByID(ctx, s.engine, connectionID.String())
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
