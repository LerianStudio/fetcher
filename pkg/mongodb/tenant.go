package mongodb

import (
	"context"
	"errors"
	"strings"

	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetDatabaseForContext resolves a tenant-specific database from context,
// falling back to the static connection when no tenant context is present.
//
// In multi-tenant mode, the tenant-specific *mongo.Database is injected into
// context by the TenantMiddleware (Manager) or RabbitMQ message handler (Worker).
// When the connection implements MultiTenantChecker and reports multi-tenant mode,
// but no tenant context is found, it returns ErrTenantContextRequired instead of
// silently falling back. In single-tenant mode (no tenant in context), it uses
// the static provider and database name.
func GetDatabaseForContext(ctx context.Context, conn MongoClientProvider, dbName string) (*mongo.Database, error) {
	if conn == nil {
		return nil, errors.New("mongo client provider is nil")
	}

	// Check if there is a tenant-specific database in context
	if db := tmcore.GetMongoFromContext(ctx); db != nil {
		return db, nil
	}

	// If the provider is multi-tenant aware, require tenant context
	if checker, ok := conn.(interface{ IsMultiTenant() bool }); ok && checker.IsMultiTenant() {
		return nil, tmcore.ErrTenantContextRequired
	}

	// Single-tenant fallback
	client, err := conn.Client(ctx)
	if err != nil {
		return nil, err
	}

	if client == nil {
		return nil, errors.New("mongo client is nil")
	}

	return client.Database(strings.ToLower(dbName)), nil
}
