package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/net/http"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/resolver"
	observability "github.com/LerianStudio/lib-observability"

	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"go.opentelemetry.io/otel/attribute"
)

type ListConnections struct {
	connRepo connRepo.Repository
	resolver resolver.ConnectionResolver // nil-safe: if nil, no internal datasources
	engine   *engine.Engine              // nil-safe scope authority
}

func NewListConnections(connectionRepo connRepo.Repository, connResolver resolver.ConnectionResolver, eng *engine.Engine) *ListConnections {
	return &ListConnections{connRepo: connectionRepo, resolver: connResolver, engine: eng}
}

func (s *ListConnections) Execute(ctx context.Context, productName string, filters http.QueryHeader) (*model.Pagination, error) {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.list_connections")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
	)

	// Route the per-request tenant-scope authority through the Engine before the
	// read. The Manager keeps its paginated, resolver-merged list (a host
	// presentation concern the Engine's flat list does not model); the Engine
	// owns only the scope rule — the single authority for which tenant may read.
	if err := authorizeConnectionAccess(ctx, s.engine); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to authorize tenant scope", err)
		return nil, err
	}

	if productName != "" {
		span.SetAttributes(attribute.String("app.request.product_name", productName))

		filters.ProductName = productName
	}

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", filters, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert fetcher input to JSON string", err)
	}

	// Route the paginated, filtered external list through the Engine's
	// ID-addressed ListConnectionsPaged op: the Engine validates the tenant scope
	// and delegates to ConnectionStore.ListPaged -> repo.List, carrying the host's
	// QueryHeader as OPAQUE params. The adapter reproduces the Manager's exact
	// pagination; the resolver-merge + total math below stays Manager-side and
	// byte-identical.
	list, totalCount, err := listConnectionsViaEngine(ctx, s.engine, filters)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to list connections", err)
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}

	// Resolve internal connections (best-effort: log warning on failure)
	var internalConns []*model.Connection

	if s.resolver != nil && filters.Page <= 1 {
		internalConns, err = s.resolver.ListInternalConnections(ctx)
		if err != nil {
			logger.Log(ctx, libLog.LevelWarn, "failed to list internal connections, returning external only",
				libLog.Err(err),
			)
		}
	}

	// Filter internal connections by type if requested
	if filters.Type != "" && len(internalConns) > 0 {
		filtered := make([]*model.Connection, 0, len(internalConns))

		for _, conn := range internalConns {
			if strings.EqualFold(string(conn.Type), filters.Type) {
				filtered = append(filtered, conn)
			}
		}

		internalConns = filtered
	}

	// Build response: internal connections first, then external
	connResp := make([]*model.ConnectionResponse, 0, len(internalConns)+len(list))

	for _, conn := range internalConns {
		connResp = append(connResp, model.NewConnectionResponseFrom(conn))
	}

	for _, conn := range list {
		connResp = append(connResp, model.NewConnectionResponseFrom(conn))
	}

	pagination := &model.Pagination{
		Limit: filters.Limit,
		Page:  filters.Page,
	}
	pagination.SetItems(connResp)
	pagination.SetTotal(int(totalCount) + len(internalConns))

	return pagination, nil
}
