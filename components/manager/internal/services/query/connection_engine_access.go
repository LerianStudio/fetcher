package query

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/model"
	netHTTP "github.com/LerianStudio/fetcher/pkg/net/http"
)

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
		conn := connectioncompat.ConnectionFromDescriptor(item)
		if conn == nil {
			return nil, 0, fmt.Errorf("convert connection descriptor: invalid host attributes for connection %s", item.ConfigName)
		}

		conns = append(conns, conn)
	}

	return conns, page.Total, nil
}
