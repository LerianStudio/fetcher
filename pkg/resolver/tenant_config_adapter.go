package resolver

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
)

// TenantManagerAdapter implements TenantConfigProvider by wrapping the
// lib-commons tenant-manager client. It fetches per-tenant service connections
// from the tenant-manager API via GetTenantConfig.
type TenantManagerAdapter struct {
	client *tmclient.Client
}

// NewTenantManagerAdapter creates a TenantConfigProvider backed by the tenant-manager HTTP client.
func NewTenantManagerAdapter(client *tmclient.Client) *TenantManagerAdapter {
	return &TenantManagerAdapter{client: client}
}

// GetServiceConnection fetches connection details for a service/module belonging to a tenant.
// It calls the tenant-manager GET /v1/tenants/{tenantID}/services/{service}/connections endpoint
// and extracts the database connection for the specified module.
//
// The TenantConfig.Databases map is keyed by module name directly (flat format).
func (a *TenantManagerAdapter) GetServiceConnection(ctx context.Context, tenantID, service, module string) (*ServiceConnectionConfig, error) {
	tenantConfig, err := a.client.GetTenantConfig(ctx, tenantID, service)
	if err != nil {
		return nil, fmt.Errorf("get tenant config for %s/%s: %w", tenantID, service, err)
	}

	if tenantConfig == nil {
		return nil, fmt.Errorf("nil tenant config returned for tenant %s, service %s", tenantID, service)
	}

	if tenantConfig.Databases == nil {
		return nil, fmt.Errorf("no databases configured for tenant %s, service %s", tenantID, service)
	}

	// The Databases map is keyed by module name directly (flat format from tenant-manager)
	dbConfig, ok := tenantConfig.Databases[module]
	if !ok {
		return nil, fmt.Errorf("module '%s' not found in service '%s' for tenant %s (available: %v)",
			module, service, tenantID, availableModules(tenantConfig.Databases))
	}

	// Extract connection details based on database type
	if dbConfig.PostgreSQL != nil {
		return &ServiceConnectionConfig{
			Host:     dbConfig.PostgreSQL.Host,
			Port:     dbConfig.PostgreSQL.Port,
			Database: dbConfig.PostgreSQL.Database,
			Username: dbConfig.PostgreSQL.Username,
			Password: dbConfig.PostgreSQL.Password,
			SSLMode:  dbConfig.PostgreSQL.SSLMode,
		}, nil
	}

	if dbConfig.MongoDB != nil {
		return buildMongoServiceConfig(dbConfig.MongoDB), nil
	}

	return nil, fmt.Errorf("no database configuration found for module '%s' in service '%s' for tenant %s",
		module, service, tenantID)
}

// buildMongoServiceConfig builds a ServiceConnectionConfig from a MongoDBConfig.
// The tenant-manager may provide TLS settings either as explicit boolean fields
// (TLS, TLSSkipVerify) or embedded in a raw URI string. This function handles
// both cases: explicit fields take precedence, falling back to URI parsing.
func buildMongoServiceConfig(cfg *tmcore.MongoDBConfig) *ServiceConnectionConfig {
	sslMode := ""
	directConn := cfg.DirectConnection
	authSource := cfg.AuthSource

	// Explicit boolean fields take precedence.
	if cfg.TLS {
		if cfg.TLSSkipVerify {
			sslMode = "insecure"
		} else {
			sslMode = "enable"
		}
	}

	// When a raw URI is provided and explicit TLS fields are not set,
	// parse the URI to extract connection options.
	if cfg.URI != "" && !cfg.TLS {
		parsed, err := url.Parse(cfg.URI)
		if err == nil {
			q := parsed.Query()

			if strings.EqualFold(q.Get("tls"), "true") || strings.EqualFold(q.Get("ssl"), "true") {
				if strings.EqualFold(q.Get("tlsInsecure"), "true") {
					sslMode = "insecure"
				} else {
					sslMode = "enable"
				}
			}

			if strings.EqualFold(q.Get("directConnection"), "true") {
				directConn = true
			}

			if as := q.Get("authSource"); as != "" && authSource == "" {
				authSource = as
			}
		}
	}

	return &ServiceConnectionConfig{
		Host:             cfg.Host,
		Port:             cfg.Port,
		Database:         cfg.Database,
		Username:         cfg.Username,
		Password:         cfg.Password,
		SSLMode:          sslMode,
		DirectConnection: directConn,
		AuthSource:       authSource,
	}
}

// availableModules extracts keys from the databases map for error reporting.
func availableModules[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}
