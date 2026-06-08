package query

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/datasource/hostsafety"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/pkg/model"
)

// discoverSchemaViaEngine routes schema DISCOVERY for an already-resolved
// connection through the Engine. The HOST keeps connection resolution (internal
// via tenant-manager, external via the repository) on its own hot path and seeds
// the resolved connection — plus the per-request schema-name scope — into the
// request context; the Engine then resolves the connection through the
// schemacompat ConnectionStore WITHOUT re-resolving and builds the host
// datasource connector to discover live.
//
// forceRefresh selects the freshness contract, the one behavior delta the
// embedded-Engine migration must preserve:
//   - false (ValidateSchema): cache-first via Engine.DiscoverSchema — a cache hit
//     short-circuits discovery and a miss writes through, the cache behavior
//     ValidateSchema kept across the migration;
//   - true (GET .../schema): ALWAYS-FRESH via Engine.DiscoverSchemaFresh — the
//     cache is neither read nor written, so the call reflects the live datasource
//     on every request, byte-identical to the pre-migration GET-schema path that
//     never touched the schema cache.
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
	forceRefresh bool,
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

	var snapshot engine.SchemaSnapshot

	if forceRefresh {
		snapshot, err = eng.DiscoverSchemaFresh(seeded, tenant, conn.ConfigName)
	} else {
		snapshot, err = eng.DiscoverSchema(seeded, tenant, conn.ConfigName)
	}

	if err != nil {
		return nil, err
	}

	return schemacompat.DataSourceSchemaFromSnapshot(snapshot), nil
}
