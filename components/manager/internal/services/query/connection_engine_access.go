package query

import (
	"context"
	"errors"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	netHTTP "github.com/LerianStudio/fetcher/pkg/net/http"

	"github.com/google/uuid"
)

// authorizeConnectionAccess routes the per-request tenant-scope authority
// decision for connection READS (get/list) through the Engine. The Manager
// keeps its own identity model — get-by-UUID (with internal-datasource registry
// resolution) and paginated, resolver-merged list — because the Engine's
// connection ops are config-name-keyed and pagination-free; routing the full
// read through them would drop pagination/registry behavior (contract drift) or
// force a redundant scope pass. Routing only the SCOPE decision makes the Engine
// the single authority for "which tenant may read a connection" without dragging
// the host's UUID or pagination identity into the Engine.
//
// The tenant is derived fresh from this request's context (single-tenant falls
// back to connectioncompat.SingleTenantID), never ambient or cached. A nil
// engine (test-only construction) is a no-op so the read proceeds.
func authorizeConnectionAccess(ctx context.Context, eng *engine.Engine) error {
	if eng == nil {
		return nil
	}

	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		return err
	}

	return eng.AuthorizeConnectionAccess(ctx, tenant)
}

// getConnectionByIDViaEngine routes the EXTERNAL connection read through the
// Engine's ID-addressed connection op: the Engine validates the per-request
// tenant scope and resolves through the connectioncompat ConnectionStore adapter
// (FindByID -> repo.FindByID), returning the rich record packed in the opaque
// host payload. The Manager keeps its internal-datasource registry resolution on
// the hot path BEFORE this call; only the external persistence round-trip now
// flows through the Engine.
//
// It returns (nil, nil) when the connection is not found so the caller maps it to
// the Manager's existing not-found business error, preserving the byte-identical
// public response. A nil engine (test-only construction) is unreachable in the
// assembled Manager; callers pass a real engine.
func getConnectionByIDViaEngine(ctx context.Context, eng *engine.Engine, connectionID uuid.UUID) (*model.Connection, error) {
	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	descriptor, err := eng.GetConnectionByID(ctx, tenant, connectionID.String())
	if err != nil {
		var engErr *engine.EngineError
		if errors.As(err, &engErr) && engErr.Category == engine.CategoryNotFound {
			return nil, nil
		}

		return nil, err
	}

	return connectioncompat.ConnectionFromDescriptor(descriptor), nil
}

// listConnectionsViaEngine routes the paginated, filtered connection list through
// the Engine's ID-addressed ListConnectionsPaged op: the Engine validates the
// per-request tenant scope and delegates to the connectioncompat ConnectionStore
// adapter (ListPaged -> repo.List) carrying the host's net/http.QueryHeader as
// OPAQUE params the Engine never interprets. The adapter reproduces the Manager's
// exact pagination behavior; this function unpacks the page back into rich
// records and the repo total so the caller's resolver-merge + total math stays
// byte-identical.
func listConnectionsViaEngine(ctx context.Context, eng *engine.Engine, filters netHTTP.QueryHeader) ([]*model.Connection, int64, error) {
	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		return nil, 0, err
	}

	page, err := eng.ListConnectionsPaged(ctx, tenant, engine.ConnectionListParams{Filter: filters})
	if err != nil {
		return nil, 0, err
	}

	conns := make([]*model.Connection, 0, len(page.Items))
	for _, item := range page.Items {
		conns = append(conns, connectioncompat.ConnectionFromDescriptor(item))
	}

	return conns, page.Total, nil
}
