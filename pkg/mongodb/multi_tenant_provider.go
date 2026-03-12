package mongodb

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

// MultiTenantMongoProvider wraps a MongoClientProvider and implements the
// tmcore.MultiTenantChecker interface (IsMultiTenant() bool). This enables
// GetDatabaseForContext to return ErrTenantContextRequired instead of silently
// falling back to the default database when multi-tenant mode is active but
// no tenant context is present in the request.
//
// Without this wrapper, the safety net in GetDatabaseForContext is disabled because
// *libMongo.Client does not implement MultiTenantChecker.
type MultiTenantMongoProvider struct {
	inner       MongoClientProvider
	multiTenant bool
}

// NewMultiTenantMongoProvider creates a MongoClientProvider that also satisfies
// the MultiTenantChecker interface. When multiTenant is true, GetDatabaseForContext
// will reject requests that lack tenant context instead of falling back to the default DB.
func NewMultiTenantMongoProvider(inner MongoClientProvider, multiTenant bool) *MultiTenantMongoProvider {
	return &MultiTenantMongoProvider{
		inner:       inner,
		multiTenant: multiTenant,
	}
}

// Client delegates to the underlying MongoClientProvider.
func (p *MultiTenantMongoProvider) Client(ctx context.Context) (*mongo.Client, error) {
	if p == nil || p.inner == nil {
		return nil, errors.New("mongo client provider is nil")
	}

	return p.inner.Client(ctx)
}

// IsMultiTenant returns true when the application is running in multi-tenant mode.
// This satisfies the MultiTenantChecker interface checked by GetDatabaseForContext.
func (p *MultiTenantMongoProvider) IsMultiTenant() bool {
	if p == nil {
		return false
	}

	return p.multiTenant
}
