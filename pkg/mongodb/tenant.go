package mongodb

import (
	"context"
	"strings"

	tmcore "github.com/LerianStudio/lib-commons/v3/commons/tenant-manager/core"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetDatabaseForContext resolves a tenant-specific database from context,
// falling back to the static connection when no tenant context is present.
//
// In multi-tenant mode, the tenant-specific *mongo.Database is injected into
// context by the TenantMiddleware (Manager) or RabbitMQ message handler (Worker).
// In single-tenant mode (no tenant in context), it uses the static provider
// and database name.
func GetDatabaseForContext(ctx context.Context, conn MongoClientProvider, dbName string) (*mongo.Database, error) {
	tenantDB, err := tmcore.GetMongoForTenant(ctx)
	if err == nil && tenantDB != nil {
		return tenantDB, nil
	}

	client, err := conn.GetDB(ctx)
	if err != nil {
		return nil, err
	}

	return client.Database(strings.ToLower(dbName)), nil
}
