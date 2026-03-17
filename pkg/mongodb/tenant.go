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
// If a tenant ID exists in context but no tenant database was injected, it fails
// closed with ErrTenantContextRequired instead of silently falling back.
// When no tenant ID is present (e.g., bootstrap operations), it falls back to the
// static provider and database name regardless of multi-tenant mode.
func GetDatabaseForContext(ctx context.Context, conn MongoClientProvider, dbName string) (*mongo.Database, error) {
	if conn == nil {
		return nil, errors.New("mongo client provider is nil")
	}

	// Check if there is a tenant-specific database in context
	if db := tmcore.GetMongoFromContext(ctx); db != nil {
		return db, nil
	}

	// If a tenant ID exists in context but no tenant DB was injected,
	// fail closed rather than silently falling back to the shared database.
	if tmcore.GetTenantIDFromContext(ctx) != "" {
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
