// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package resolver

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
)

// TenantManagerAdapter implements TenantConfigProvider by wrapping one or more
// lib-commons tenant-manager clients. Each service may authenticate with its own
// API key, so the adapter holds a per-service client map (keyed by the normalized
// service token, see normalizeServiceToken) plus a default client used when no
// per-service entry exists.
type TenantManagerAdapter struct {
	clients       map[string]*tmclient.Client // keyed by normalizeServiceToken(service)
	defaultClient *tmclient.Client
}

// NewTenantManagerAdapterWithClients creates a TenantConfigProvider backed by a
// per-service client map and a default client. The map keys must be normalized
// service tokens (normalizeServiceToken). GetServiceConnection selects the
// client for a service by token, falling back to defaultClient when the service
// has no dedicated entry. Either argument may be nil; if both resolve to nil for
// a given service, GetServiceConnection returns an explicit error.
func NewTenantManagerAdapterWithClients(clients map[string]*tmclient.Client, defaultClient *tmclient.Client) *TenantManagerAdapter {
	return &TenantManagerAdapter{clients: clients, defaultClient: defaultClient}
}

// pickClient selects the tenant-manager client for a service: the per-service
// entry (matched by normalized token) when present, otherwise the default
// client. Returns nil when neither is configured; callers must turn that into
// an explicit error rather than dereferencing it.
func (a *TenantManagerAdapter) pickClient(service string) *tmclient.Client {
	if c := a.clients[normalizeServiceToken(service)]; c != nil {
		return c
	}

	return a.defaultClient
}

// GetServiceConnection fetches connection details for a service/module belonging to a tenant.
// It calls the tenant-manager GET /v1/tenants/{tenantID}/services/{service}/connections endpoint
// and extracts the database connection for the specified module.
//
// The TenantConfig.Databases map is keyed by module name directly (flat format).
func (a *TenantManagerAdapter) GetServiceConnection(ctx context.Context, tenantID, service, module string) (*ServiceConnectionConfig, error) {
	c := a.pickClient(service)
	if c == nil {
		return nil, fmt.Errorf("no tenant-manager client configured for service %q and no default", service)
	}

	tenantConfig, err := c.GetTenantConfig(ctx, tenantID, service)
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
