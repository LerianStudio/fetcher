package bootstrap

import (
	"context"
	"testing"
	"time"

	pkgRabbitmq "github.com/LerianStudio/fetcher/v2/pkg/rabbitmq"
	"github.com/LerianStudio/fetcher/v2/pkg/testutil"
	tmcore "github.com/LerianStudio/lib-commons/v6/commons/tenant-manager/core"
	"github.com/LerianStudio/lib-commons/v6/commons/tenant-manager/tenantcache"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The two spellings of ONE tenant UUID. Access Manager tokens carry the dashless
// form, which is why the asymmetry described below is latent rather than live.
const (
	canonicalTestTenantDashed   = "9b2f4c1e-3d5a-4f8b-9c7d-1e2f3a4b5c6d"
	canonicalTestTenantDashless = "9b2f4c1e3d5a4f8b9c7d1e2f3a4b5c6d"
)

// TestAuthoritativeTenantHeaderAcceptsBothUUIDSpellings covers the worst outcome
// of the spelling asymmetry: a legitimate message rejected as a cross-tenant
// replay because the publisher spelled the tenant one way and the authoritative
// context the other. lib-commons canonicalizes what its middleware puts in the
// context but not what its client returns, so the two sides agreeing is a
// coincidence rather than a guarantee.
func TestAuthoritativeTenantHeaderAcceptsBothUUIDSpellings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		contextTenant string
		headerTenant  string
	}{
		{name: "both dashless", contextTenant: canonicalTestTenantDashless, headerTenant: canonicalTestTenantDashless},
		{name: "both dashed", contextTenant: canonicalTestTenantDashed, headerTenant: canonicalTestTenantDashed},
		{name: "dashless context, dashed header", contextTenant: canonicalTestTenantDashless, headerTenant: canonicalTestTenantDashed},
		{name: "dashed context, dashless header", contextTenant: canonicalTestTenantDashed, headerTenant: canonicalTestTenantDashless},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := tmcore.ContextWithTenantID(testutil.TestContext(), tt.contextTenant)
			headers := map[string]any{pkgRabbitmq.HeaderTenantID: tt.headerTenant}

			assert.NoError(t, validateAuthoritativeTenantHeader(ctx, headers))
		})
	}
}

// A genuinely different tenant must still be refused — canonicalization must not
// widen the check into accepting anything.
func TestAuthoritativeTenantHeaderStillRejectsForeignTenant(t *testing.T) {
	t.Parallel()

	ctx := tmcore.ContextWithTenantID(testutil.TestContext(), canonicalTestTenantDashless)
	headers := map[string]any{pkgRabbitmq.HeaderTenantID: "11111111111111111111111111111111"}

	require.Error(t, validateAuthoritativeTenantHeader(ctx, headers))
}

// TestKnownTenantsKeyIsSpellingIndependent covers the other half: the consumer's
// tenant bookkeeping. Two spellings keying two entries would start two consumers
// on one tenant's queue, and a stop request in the other spelling would leave one
// of them running.
func TestKnownTenantsKeyIsSpellingIndependent(t *testing.T) {
	t.Parallel()

	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		TenantCache: tenantcache.NewTenantCache(),
		RabbitMQ:    &fakeWorkerRabbitMQManager{channel: newFakeWorkerRabbitMQChannel()},
		Logger:      testBootstrapLogger(),
	})

	consumer.markTenantKnown(canonicalTestTenantDashless)

	assert.True(t, consumer.OwnsTenant(canonicalTestTenantDashless))
	assert.True(t, consumer.OwnsTenant(canonicalTestTenantDashed),
		"a tenant known by its dashless id must be recognized by its dashed id")

	// Stopping by the other spelling must clear the same entry.
	consumer.StopConsumer(canonicalTestTenantDashed)

	assert.False(t, consumer.OwnsTenant(canonicalTestTenantDashless))
	assert.False(t, consumer.OwnsTenant(canonicalTestTenantDashed))
}

// TestRegisterCanonicalizesCacheTenantIDs covers the third entry point into the
// tenant bookkeeping: Register imports whatever spelling the Tenant Manager put
// in the cache. A dashed cache key must land in knownTenants canonically, or a
// later canonical start would key a second consumer for the same tenant and a
// stop by the other spelling would leave one running.
func TestRegisterCanonicalizesCacheTenantIDs(t *testing.T) {
	t.Parallel()

	cache := tenantcache.NewTenantCache()
	cache.Set(canonicalTestTenantDashed, &tmcore.TenantConfig{ID: canonicalTestTenantDashed}, time.Minute)

	manager := &fakeWorkerRabbitMQManager{channel: newFakeWorkerRabbitMQChannel()}
	consumer := newWorkerMultiTenantConsumer(workerMultiTenantConsumerConfig{
		TenantCache: cache,
		RabbitMQ:    manager,
		Logger:      testBootstrapLogger(),
	})

	require.NoError(t, consumer.Register("jobs", func(context.Context, amqp.Delivery) error { return nil }))

	assert.True(t, consumer.OwnsTenant(canonicalTestTenantDashed))
	assert.True(t, consumer.OwnsTenant(canonicalTestTenantDashless),
		"a tenant imported from a dashed cache key must be recognized by its canonical id")
	assert.Equal(t, []string{canonicalTestTenantDashless}, consumer.KnownTenants(),
		"the cache import must key knownTenants by the canonical spelling only")

	// A canonical-spelling start after the import must not open a second
	// consumer for the tenant Register already started.
	consumer.EnsureConsumerStarted(testutil.TestContext(), canonicalTestTenantDashless)

	assert.Eventually(t, func() bool { return manager.calls.Load() == 1 }, time.Second, 10*time.Millisecond)
	assert.Never(t, func() bool { return manager.calls.Load() > 1 }, 100*time.Millisecond, 10*time.Millisecond)
	require.NoError(t, consumer.Close())
}
