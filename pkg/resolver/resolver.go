package resolver

import (
	"context"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
)

// ConnectionResolver resolves datasource connections for a job.
// Internal datasources (Lerian products/plugins) are resolved automatically.
// External datasources are looked up from the connection repository (MongoDB).
//
// Tenant context flows through ctx (set by lib-commons middleware).
// In multi-tenant mode, tmcore.GetTenantIDContext(ctx) provides the tenant ID.
// In single-tenant mode, ctx has no tenant -- resolvers use env vars or default DB.
//
//go:generate mockgen --destination=resolver.mock.go --package=resolver . ConnectionResolver
type ConnectionResolver interface {
	// ResolveConnections returns connections for the given configNames.
	// Internal datasources are resolved via env vars (ST) or tenant-manager (MT).
	// External datasources are resolved from the Connection repository.
	// Tenant context comes from ctx (not an explicit parameter).
	ResolveConnections(ctx context.Context, configNames []string) ([]*model.Connection, error)

	// IsInternalDatasource returns true if the configName is a Lerian internal datasource.
	IsInternalDatasource(configName string) bool

	// ListInternalConnections returns in-memory Connection objects for all known
	// internal datasources available to the current tenant.
	// In multi-tenant mode, connections are resolved via tenant-manager.
	// In single-tenant mode, connections come from environment variables.
	ListInternalConnections(ctx context.Context) ([]*model.Connection, error)

	// ResolveInternalByConfigName resolves a single internal datasource by configName.
	// Returns nil, nil if configName is not an internal datasource.
	ResolveInternalByConfigName(ctx context.Context, configName string) (*model.Connection, error)
}
