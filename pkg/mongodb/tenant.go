package mongodb

import (
	"context"
	"errors"
	"strings"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type multiTenantChecker interface {
	IsMultiTenant() bool
}

// ResolveDatabase returns the tenant-scoped Mongo database when present.
// If a tenant ID exists in context but no tenant database has been injected,
// it fails closed with ErrTenantContextRequired instead of silently falling
// back to the shared static connection.
//
// When no tenant ID is present, the static single-tenant connection is used.
func ResolveDatabase(ctx context.Context, conn MongoClientProvider, dbName string) (*mongo.Database, error) {
	if conn == nil {
		return nil, errors.New("mongo client provider is nil")
	}

	// Check if there is a tenant-specific database in context
	if db := tmcore.GetMBContext(ctx); db != nil {
		return db, nil
	}

	// Multi-tenant mode must never fall back to the shared database. The
	// provider-level flag catches both missing tenant IDs and missing tenant DBs.
	if checker, ok := conn.(multiTenantChecker); ok && checker.IsMultiTenant() {
		return nil, tmcore.ErrTenantContextRequired
	}

	// If a tenant ID exists in context but no tenant DB was injected, fail closed
	// even when the provider does not expose an explicit multi-tenant flag.
	if tmcore.GetTenantIDContext(ctx) != "" {
		return nil, tmcore.ErrTenantContextRequired
	}

	// No tenant context present — use the static connection.
	// This path is used in single-tenant mode and for bootstrap operations
	// (e.g., EnsureIndexes) that run before any tenant context is available.
	client, err := conn.Client(ctx)
	if err != nil {
		return nil, err
	}

	if client == nil {
		return nil, errors.New("mongo client is nil")
	}

	return client.Database(strings.ToLower(dbName)), nil
}
