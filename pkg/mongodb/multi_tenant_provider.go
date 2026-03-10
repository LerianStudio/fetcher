package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// MultiTenantMongoProvider wraps a MongoClientProvider and implements the
// tmcore.MultiTenantChecker interface (IsMultiTenant() bool). This enables
// tmcore.ResolveMongo to return ErrTenantContextRequired instead of silently
// falling back to the default database when multi-tenant mode is active but
// no tenant context is present in the request.
//
// Without this wrapper, the safety net in ResolveMongo is disabled because
// *libMongo.MongoConnection does not implement MultiTenantChecker.
type MultiTenantMongoProvider struct {
	inner       MongoClientProvider
	multiTenant bool
}

// NewMultiTenantMongoProvider creates a MongoClientProvider that also satisfies
// tmcore.MultiTenantChecker. When multiTenant is true, ResolveMongo will reject
// requests that lack tenant context instead of falling back to the default DB.
func NewMultiTenantMongoProvider(inner MongoClientProvider, multiTenant bool) *MultiTenantMongoProvider {
	return &MultiTenantMongoProvider{
		inner:       inner,
		multiTenant: multiTenant,
	}
}

// GetDB delegates to the underlying MongoClientProvider.
func (p *MultiTenantMongoProvider) GetDB(ctx context.Context) (*mongo.Client, error) {
	return p.inner.GetDB(ctx)
}

// IsMultiTenant returns true when the application is running in multi-tenant mode.
// This satisfies the tmcore.MultiTenantChecker interface checked by tmcore.ResolveMongo.
func (p *MultiTenantMongoProvider) IsMultiTenant() bool {
	return p.multiTenant
}
