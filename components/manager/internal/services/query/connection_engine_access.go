package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
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
