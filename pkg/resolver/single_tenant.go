package resolver

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	connPort "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
)

// SingleTenantResolver resolves internal datasources from env vars (loaded at startup)
// and external datasources from the MongoDB connection repository.
// Used when MULTI_TENANT_ENABLED=false.
type SingleTenantResolver struct {
	connRepo       connPort.Repository
	registry       *InternalDatasourceRegistry
	envConnections map[string]*model.Connection
}

// NewSingleTenantResolver creates a resolver for single-tenant mode.
// envConnections maps configName -> Connection built from environment variables at startup.
func NewSingleTenantResolver(
	connRepo connPort.Repository,
	registry *InternalDatasourceRegistry,
	envConnections map[string]*model.Connection,
) *SingleTenantResolver {
	if envConnections == nil {
		envConnections = make(map[string]*model.Connection)
	}

	return &SingleTenantResolver{
		connRepo:       connRepo,
		registry:       registry,
		envConnections: envConnections,
	}
}

// ResolveConnections resolves connections for the given configNames.
// Internal datasources are resolved from pre-loaded env vars.
// External datasources are resolved from MongoDB.
func (r *SingleTenantResolver) ResolveConnections(ctx context.Context, configNames []string) ([]*model.Connection, error) {
	var (
		resolved      []*model.Connection
		externalNames []string
	)

	for _, name := range configNames {
		if r.registry.IsInternal(name) {
			conn, ok := r.envConnections[name]
			if !ok {
				return nil, fmt.Errorf("internal datasource '%s' not configured via environment variables", name)
			}

			resolved = append(resolved, conn)
		} else {
			externalNames = append(externalNames, name)
		}
	}

	if len(externalNames) > 0 {
		externalConns, err := r.connRepo.FindByConfigNames(ctx, externalNames)
		if err != nil {
			return nil, fmt.Errorf("resolve external connections: %w", err)
		}

		resolved = append(resolved, externalConns...)
	}

	return resolved, nil
}

// IsInternalDatasource returns true if the configName is a known internal datasource.
func (r *SingleTenantResolver) IsInternalDatasource(configName string) bool {
	return r.registry.IsInternal(configName)
}

// ListInternalConnections returns all pre-loaded internal connections from env vars.
// In single-tenant mode, if envConnections is empty (current default), this returns
// an empty list — no regression from existing behavior.
func (r *SingleTenantResolver) ListInternalConnections(_ context.Context) ([]*model.Connection, error) {
	conns := make([]*model.Connection, 0, len(r.envConnections))

	for name, conn := range r.envConnections {
		c := *conn // shallow copy to avoid mutating the original
		c.ID = r.registry.GetDeterministicID("", name)

		conns = append(conns, &c)
	}

	return conns, nil
}

// ResolveInternalByConfigName resolves a single internal datasource from env vars.
// Returns nil, nil if configName is not an internal datasource.
func (r *SingleTenantResolver) ResolveInternalByConfigName(_ context.Context, configName string) (*model.Connection, error) {
	if !r.registry.IsInternal(configName) {
		return nil, nil
	}

	conn, ok := r.envConnections[configName]
	if !ok {
		return nil, fmt.Errorf("internal datasource '%s' not configured via environment variables", configName)
	}

	c := *conn
	c.ID = r.registry.GetDeterministicID("", configName)

	return &c, nil
}
