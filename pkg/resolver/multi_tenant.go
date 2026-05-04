package resolver

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/model"
	connPort "github.com/LerianStudio/fetcher/pkg/ports/connection"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
)

// ServiceConnectionConfig holds connection details returned by the tenant-manager.
type ServiceConnectionConfig struct {
	Host             string
	Port             int
	Database         string
	Username         string
	Password         string
	SSLMode          string
	DirectConnection bool
	AuthSource       string
}

// TenantConfigProvider abstracts the tenant-manager client for resolving
// per-tenant service connections.
//
//go:generate mockgen --destination=tenant_config_provider.mock.go --package=resolver . TenantConfigProvider
type TenantConfigProvider interface {
	// GetServiceConnection returns connection details for a specific service/module
	// belonging to the given tenant.
	GetServiceConnection(ctx context.Context, tenantID, service, module string) (*ServiceConnectionConfig, error)
}

// MultiTenantResolver resolves internal datasources from the tenant-manager
// and external datasources from the MongoDB connection repository.
// Used when MULTI_TENANT_ENABLED=true.
type MultiTenantResolver struct {
	connRepo       connPort.Repository
	registry       *InternalDatasourceRegistry
	tenantProvider TenantConfigProvider
}

// NewMultiTenantResolver creates a resolver for multi-tenant mode.
func NewMultiTenantResolver(
	connRepo connPort.Repository,
	registry *InternalDatasourceRegistry,
	tenantProvider TenantConfigProvider,
) *MultiTenantResolver {
	return &MultiTenantResolver{
		connRepo:       connRepo,
		registry:       registry,
		tenantProvider: tenantProvider,
	}
}

// ResolveConnections resolves connections for the given configNames.
// Internal datasources are resolved via tenant-manager GetServiceConnection.
// External datasources are resolved from MongoDB.
// Tenant context comes from ctx (set by lib-commons middleware).
func (r *MultiTenantResolver) ResolveConnections(ctx context.Context, configNames []string) ([]*model.Connection, error) {
	var (
		resolved      []*model.Connection
		externalNames []string
	)

	for _, name := range configNames {
		if r.registry.IsInternal(name) {
			conn, err := r.resolveInternalConnection(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("resolve internal datasource '%s': %w", name, err)
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

// resolveInternalConnection builds an in-memory Connection from tenant-manager data.
// Extracts tenantID from ctx (set by lib-commons TenantMiddleware).
func (r *MultiTenantResolver) resolveInternalConnection(ctx context.Context, configName string) (*model.Connection, error) {
	dsConfig, ok := r.registry.GetConfig(configName)
	if !ok {
		return nil, fmt.Errorf("unknown internal datasource: %s", configName)
	}

	// Extract tenant ID from context (set by lib-commons middleware)
	tenantID := tmcore.GetTenantIDContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID not found in context for internal datasource '%s'", configName)
	}

	svcConn, err := r.tenantProvider.GetServiceConnection(ctx, tenantID, dsConfig.Service, dsConfig.Module)
	if err != nil {
		return nil, fmt.Errorf("tenant-manager lookup for %s/%s: %w", dsConfig.Service, dsConfig.Module, err)
	}

	// Build an in-memory Connection with the same shape as MongoDB-stored connections.
	// This allows the downstream pipeline (factory -> connect -> query -> close) to work unchanged.
	conn := &model.Connection{
		ConfigName:   configName,
		Type:         dsConfig.DBType,
		Host:         svcConn.Host,
		Port:         svcConn.Port,
		DatabaseName: svcConn.Database,
		Username:     svcConn.Username,
		// EncryptionKeyVersion is intentionally left empty.
		// The datasource factory will skip decryption for connections without EncryptionKeyVersion.
	}

	// Set plaintext password (not encrypted, from tenant-manager)
	conn.SetPlaintextPassword(svcConn.Password)

	if svcConn.SSLMode != "" {
		conn.SSL = &model.SSLConfig{Mode: svcConn.SSLMode}
	}

	// Propagate MongoDB-specific connection options from tenant-manager config.
	// These are read by buildMongoDBOptions in the datasource factory.
	metadata := make(map[string]any)

	if svcConn.DirectConnection {
		metadata["directConnection"] = "true"
	}

	if svcConn.AuthSource != "" {
		metadata["authSource"] = svcConn.AuthSource
	}

	if len(metadata) > 0 {
		conn.Metadata = &metadata
	}

	return conn, nil
}

// IsInternalDatasource returns true if the configName is a known internal datasource.
func (r *MultiTenantResolver) IsInternalDatasource(configName string) bool {
	return r.registry.IsInternal(configName)
}

// ListInternalConnections resolves all known internal datasources for the current
// tenant via tenant-manager. Best-effort: if one datasource fails, it is skipped
// and the rest are returned.
func (r *MultiTenantResolver) ListInternalConnections(ctx context.Context) ([]*model.Connection, error) {
	tenantID := tmcore.GetTenantIDContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID not found in context for listing internal connections")
	}

	names := r.registry.ListInternal()
	conns := make([]*model.Connection, 0, len(names))

	for _, name := range names {
		conn, err := r.resolveInternalConnection(ctx, name)
		if err != nil {
			// Best-effort: skip datasources that fail to resolve (tenant-manager
			// may not have all services configured for every tenant).
			continue
		}

		conn.ID = r.registry.GetDeterministicID(tenantID, name)

		conns = append(conns, conn)
	}

	return conns, nil
}

// ResolveInternalByConfigName resolves a single internal datasource by configName
// for the current tenant. Returns nil, nil if configName is not internal.
func (r *MultiTenantResolver) ResolveInternalByConfigName(ctx context.Context, configName string) (*model.Connection, error) {
	if !r.registry.IsInternal(configName) {
		return nil, nil
	}

	tenantID := tmcore.GetTenantIDContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID not found in context for internal datasource '%s'", configName)
	}

	conn, err := r.resolveInternalConnection(ctx, configName)
	if err != nil {
		return nil, err
	}

	conn.ID = r.registry.GetDeterministicID(tenantID, configName)

	return conn, nil
}
