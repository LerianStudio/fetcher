package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/datasource/hostsafety"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	schemacompat "github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/pkg/model"
)

// discoverSchemaViaEngine routes schema DISCOVERY for an already-resolved
// connection through the Engine's cache-first DiscoverSchema op. The HOST keeps
// connection resolution (internal via tenant-manager, external via the
// repository) on its own hot path and seeds the resolved connection — plus the
// per-request schema-name scope — into the request context; the Engine then
// resolves the connection through the schemacompat ConnectionStore WITHOUT
// re-resolving, consults the SchemaCache port (Redis, wired host-side), and on a
// miss builds the host datasource connector to discover live and write through
// the cache.
//
// The tenant is derived fresh from this request's context via the
// connectioncompat tmcore bridge (single-tenant falls back to the host default),
// never ambient. The returned Engine SchemaSnapshot is converted back into the
// host *model.DataSourceSchema so the Manager's existing validation, DB-type
// normalization, and plugin_crm policy operate on it unchanged.
//
// schemas is the optional schema-name scope discovery should fetch (preserving
// the legacy multi-schema behavior); an empty list lets the connector fall back
// to the connection's Schema field or the adapter default.
func discoverSchemaViaEngine(
	ctx context.Context,
	eng *engine.Engine,
	conn *model.Connection,
	schemas []string,
) (*model.DataSourceSchema, error) {
	// Run the host-safety (SSRF) guard BEFORE delegating discovery to the Engine.
	// The Engine deliberately REDACTS connector errors (a connect error may embed
	// a DSN/credential), so a typed host-safety rejection raised inside the Engine
	// connector would be flattened to a generic unavailable error — losing the
	// FET-0414 audit signal and its HTTP 400 mapping. Keeping the guard host-side
	// preserves that contract byte-identically and keeps SSRF policy out of the
	// Engine core. Internal datasources (EncryptionKeyVersion == "") are exempt by
	// construction, mirroring the datasource factory's own guard.
	if guardErr := hostsafety.ValidateHostForConnection(ctx, conn); guardErr != nil {
		return nil, guardErr
	}

	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	seeded := schemacompat.WithResolvedConnections(ctx, []*model.Connection{conn})
	seeded = schemacompat.WithSchemaScope(seeded, conn.ConfigName, schemas)

	snapshot, err := eng.DiscoverSchema(seeded, tenant, conn.ConfigName)
	if err != nil {
		return nil, err
	}

	return schemacompat.DataSourceSchemaFromSnapshot(snapshot), nil
}
