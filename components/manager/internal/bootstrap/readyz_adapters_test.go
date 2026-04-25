package bootstrap

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findCheckerByName walks a slice of DependencyCheckers and returns the
// first one whose Name() matches. Test helper to avoid order-coupling.
func findCheckerByName(t *testing.T, checkers []readyz.DependencyChecker, name string) readyz.DependencyChecker {
	t.Helper()

	for _, c := range checkers {
		if c == nil {
			continue
		}

		if c.Name() == name {
			return c
		}
	}

	return nil
}

// TestBuildManagerReadyzCheckers_NilCfg_ReturnsEmpty verifies the nil-cfg
// guard added in M-B does NOT panic and returns an empty slice. Mirrors the
// nil-tolerance pattern used by newReadyzRedisClient.
func TestBuildManagerReadyzCheckers_NilCfg_ReturnsEmpty(t *testing.T) {
	require.NotPanics(t, func() {
		got := buildManagerReadyzCheckers(nil, nil, nil)
		assert.Empty(t, got)
	})
}

// TestBuildManagerReadyzCheckers_MTMongo_TLSFromAtlasURI_TrueWithoutCACert
// verifies M-A: a multi-tenant deployment using an Atlas-style mongodb+srv
// URI must surface tls=true on the global /readyz NAChecker even when no
// MongoTLSCACert is configured. Previously the NAChecker reported tls=false
// for Atlas because it computed posture from CA-cert presence alone.
func TestBuildManagerReadyzCheckers_MTMongo_TLSFromAtlasURI_TrueWithoutCACert(t *testing.T) {
	cfg := &Config{
		MultiTenantEnabled: true,
		MongoURI:           "mongodb+srv",
		MongoDBHost:        "atlas.example.com",
		MongoTLSCACert:     "", // intentionally empty — Atlas uses system trust store
	}

	got := buildManagerReadyzCheckers(cfg, nil, nil)

	mongo := findCheckerByName(t, got, "mongodb")
	require.NotNil(t, mongo, "mongodb NAChecker missing from MT global /readyz")

	check := mongo.Check(context.Background())
	require.NotNil(t, check.TLS, "TLS pointer must be set on the NAChecker output")
	assert.True(t, *check.TLS, "Atlas-style mongodb+srv URI must report tls=true even when MongoTLSCACert is empty")
}

// TestBuildManagerReadyzCheckers_MTMongo_TLSFromCACertFallback_TrueWithoutSchemeTLS
// verifies M-A: when the URI alone does not signal TLS (plain "mongodb"
// scheme without ?tls=true), the CA-cert presence still serves as a fallback
// signal so legacy deployments do not regress.
func TestBuildManagerReadyzCheckers_MTMongo_TLSFromCACertFallback_TrueWithoutSchemeTLS(t *testing.T) {
	cfg := &Config{
		MultiTenantEnabled: true,
		MongoURI:           "mongodb",
		MongoDBHost:        "mongo.internal",
		MongoTLSCACert:     "BASE64_CERT_DATA", // non-empty: legacy operator-supplied CA
	}

	got := buildManagerReadyzCheckers(cfg, nil, nil)

	mongo := findCheckerByName(t, got, "mongodb")
	require.NotNil(t, mongo, "mongodb NAChecker missing")

	check := mongo.Check(context.Background())
	require.NotNil(t, check.TLS)
	assert.True(t, *check.TLS, "operator-supplied CA cert must be honoured as TLS-on fallback")
}

// TestBuildManagerReadyzCheckers_MTMongo_NoTLSAtAll_False verifies the
// genuine non-TLS case: plain mongodb scheme + no CA cert -> tls=false.
// This is the only path where reporting tls=false is correct.
func TestBuildManagerReadyzCheckers_MTMongo_NoTLSAtAll_False(t *testing.T) {
	cfg := &Config{
		MultiTenantEnabled: true,
		MongoURI:           "mongodb",
		MongoDBHost:        "mongo.local",
		MongoTLSCACert:     "",
	}

	got := buildManagerReadyzCheckers(cfg, nil, nil)

	mongo := findCheckerByName(t, got, "mongodb")
	require.NotNil(t, mongo)

	check := mongo.Check(context.Background())
	require.NotNil(t, check.TLS)
	assert.False(t, *check.TLS, "plain mongodb scheme + no CA cert must report tls=false")
}

// TestNewReadyzConfig_NilCfg_FallsBackToLoadConfig verifies the nil-cfg
// guard added in M-B for the manager. A nil Config must not panic and must
// return a usable readyz.Config (env-derived).
func TestNewReadyzConfig_NilCfg_FallsBackToLoadConfig(t *testing.T) {
	require.NotPanics(t, func() {
		got := newReadyzConfig(nil)
		require.NotNil(t, got)
		// Sanity: LoadConfig sets Version (defaulting to "unknown" in absence
		// of OTEL_RESOURCE_SERVICE_VERSION / VERSION).
		assert.NotEmpty(t, got.Version)
	})
}
