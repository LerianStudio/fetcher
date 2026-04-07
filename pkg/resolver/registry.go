// Package resolver provides dual-resolution for internal and external datasources.
// Internal datasources (Lerian products/plugins) are resolved automatically via
// tenant-manager (multi-tenant) or environment variables (single-tenant).
// External datasources are resolved from the MongoDB connection repository.
package resolver

import (
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/google/uuid"
)

// InternalDatasourceNamespace is the UUID v5 namespace used to generate
// deterministic UUIDs for internal datasource connections.
// Formula: uuid.NewSHA1(namespace, []byte(tenantID + "/" + configName))
var InternalDatasourceNamespace = uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")

// InternalDSConfig describes a Lerian-internal datasource that can be resolved
// automatically without a user-registered Connection.
type InternalDSConfig struct {
	// Service is the tenant-manager service name (e.g., "midaz", "plugin-crm").
	Service string

	// Module is the tenant-manager module name (e.g., "onboarding", "transaction", "crm").
	Module string

	// DBType is the database type for the datasource factory.
	DBType model.DBType
}

// InternalDatasourceRegistry holds the known Lerian products/plugins that the
// Fetcher can connect to automatically. This is a hardcoded, finite set.
// Adding a new internal datasource requires a Fetcher deploy (per decision D1).
type InternalDatasourceRegistry struct {
	datasources map[string]InternalDSConfig
}

// NewInternalDatasourceRegistry creates a registry with the approved MVP datasources (D4).
func NewInternalDatasourceRegistry() *InternalDatasourceRegistry {
	return &InternalDatasourceRegistry{
		datasources: map[string]InternalDSConfig{
			"midaz_onboarding":  {Service: "ledger", Module: "onboarding", DBType: model.TypePostgreSQL},
			"midaz_transaction": {Service: "ledger", Module: "transaction", DBType: model.TypePostgreSQL},
			"plugin_crm":        {Service: "ledger", Module: "crm", DBType: model.TypeMongoDB},
		},
	}
}

// IsInternal returns true if the configName is a known Lerian internal datasource.
func (r *InternalDatasourceRegistry) IsInternal(configName string) bool {
	_, ok := r.datasources[configName]
	return ok
}

// GetConfig returns the internal datasource configuration for the given configName.
func (r *InternalDatasourceRegistry) GetConfig(configName string) (InternalDSConfig, bool) {
	cfg, ok := r.datasources[configName]
	return cfg, ok
}

// ListInternal returns all known internal datasource configNames.
func (r *InternalDatasourceRegistry) ListInternal() []string {
	names := make([]string, 0, len(r.datasources))
	for name := range r.datasources {
		names = append(names, name)
	}

	return names
}

// GetAllConfigs returns the full map of internal datasource configurations.
func (r *InternalDatasourceRegistry) GetAllConfigs() map[string]InternalDSConfig {
	return r.datasources
}

// GetDeterministicID generates a stable UUID v5 for an internal datasource
// scoped to a specific tenant. The same tenant+configName always produces the
// same UUID, and different tenants produce different UUIDs.
func (r *InternalDatasourceRegistry) GetDeterministicID(tenantID, configName string) uuid.UUID {
	return uuid.NewSHA1(InternalDatasourceNamespace, []byte(tenantID+"/"+configName))
}

// FindConfigByID performs a reverse lookup: given a UUID and tenantID, it checks
// whether the UUID matches any internal datasource for that tenant.
// Returns the configName, config, and true if found; empty values and false otherwise.
func (r *InternalDatasourceRegistry) FindConfigByID(id uuid.UUID, tenantID string) (string, InternalDSConfig, bool) {
	for name, cfg := range r.datasources {
		if uuid.NewSHA1(InternalDatasourceNamespace, []byte(tenantID+"/"+name)) == id {
			return name, cfg, true
		}
	}

	return "", InternalDSConfig{}, false
}
