package bootstrap

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findCheckerByName walks the slice and returns the first checker whose
// Name() matches. Order-agnostic so tests survive any future ordering
// changes in buildWorkerReadyzCheckers.
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

// TestBuildWorkerReadyzCheckers_NilDeps_ReturnsEmpty verifies the existing
// nil-deps guard remains intact after M-B's extension to also cover nil cfg.
func TestBuildWorkerReadyzCheckers_NilDeps_ReturnsEmpty(t *testing.T) {
	require.NotPanics(t, func() {
		got := buildWorkerReadyzCheckers(nil)
		assert.Empty(t, got)
	})
}

// TestBuildWorkerReadyzCheckers_NilDepsCfg_ReturnsEmpty verifies the
// extension added in M-B: a deps with nil cfg must not panic on cfg.*
// dereferences.
func TestBuildWorkerReadyzCheckers_NilDepsCfg_ReturnsEmpty(t *testing.T) {
	require.NotPanics(t, func() {
		got := buildWorkerReadyzCheckers(&workerReadyzDeps{cfg: nil})
		assert.Empty(t, got)
	})
}

// TestBuildWorkerReadyzCheckers_MTMongo_TLSFromAtlasURI_TrueWithoutCACert
// mirrors the manager test: an Atlas-style mongodb+srv URI must report
// tls=true on the worker's global /readyz NAChecker even without a CA cert.
// This is the operational outcome of M-A on the worker side.
func TestBuildWorkerReadyzCheckers_MTMongo_TLSFromAtlasURI_TrueWithoutCACert(t *testing.T) {
	deps := &workerReadyzDeps{
		cfg: &Config{
			MultiTenantEnabled: true,
			MongoURI:           "mongodb+srv",
			MongoDBHost:        "atlas.example.com",
			MongoDBUser:        "u",
			MongoDBPassword:    "p",
			MongoDBPort:        "27017",
			MongoTLSCACert:     "",
		},
	}

	got := buildWorkerReadyzCheckers(deps)

	mongo := findCheckerByName(t, got, "mongodb")
	require.NotNil(t, mongo, "mongodb NAChecker missing from MT global /readyz")

	check := mongo.Check(context.Background())
	require.NotNil(t, check.TLS)
	assert.True(t, *check.TLS, "Atlas-style mongodb+srv URI must report tls=true even when MongoTLSCACert is empty")
}

// TestBuildWorkerReadyzCheckers_MTMongo_TLSFromCACertFallback_True verifies
// the CA-cert fallback still kicks in for legacy operator-supplied configs
// where the URI scheme alone does not carry TLS information.
func TestBuildWorkerReadyzCheckers_MTMongo_TLSFromCACertFallback_True(t *testing.T) {
	deps := &workerReadyzDeps{
		cfg: &Config{
			MultiTenantEnabled: true,
			MongoURI:           "mongodb",
			MongoDBHost:        "mongo.internal",
			MongoDBUser:        "u",
			MongoDBPassword:    "p",
			MongoDBPort:        "27017",
			MongoTLSCACert:     "BASE64_CERT",
		},
	}

	got := buildWorkerReadyzCheckers(deps)

	mongo := findCheckerByName(t, got, "mongodb")
	require.NotNil(t, mongo)

	check := mongo.Check(context.Background())
	require.NotNil(t, check.TLS)
	assert.True(t, *check.TLS, "CA-cert presence must be honoured as TLS-on fallback")
}

// TestBuildWorkerReadyzCheckers_MTMongo_NoTLSAtAll_False verifies the
// genuine non-TLS case: plain mongodb scheme + no CA cert -> tls=false.
func TestBuildWorkerReadyzCheckers_MTMongo_NoTLSAtAll_False(t *testing.T) {
	deps := &workerReadyzDeps{
		cfg: &Config{
			MultiTenantEnabled: true,
			MongoURI:           "mongodb",
			MongoDBHost:        "mongo.local",
			MongoDBUser:        "u",
			MongoDBPassword:    "p",
			MongoDBPort:        "27017",
			MongoTLSCACert:     "",
		},
	}

	got := buildWorkerReadyzCheckers(deps)

	mongo := findCheckerByName(t, got, "mongodb")
	require.NotNil(t, mongo)

	check := mongo.Check(context.Background())
	require.NotNil(t, check.TLS)
	assert.False(t, *check.TLS, "plain mongodb + no CA cert must report tls=false")
}

// TestBuildWorkerTenantHandler_NilDepsCfg_ReturnsDisabled verifies the
// extension added in M-B: a deps with nil cfg must short-circuit to the
// disabled handler rather than panic on deps.cfg.MultiTenantEnabled.
func TestBuildWorkerTenantHandler_NilDepsCfg_ReturnsDisabled(t *testing.T) {
	require.NotPanics(t, func() {
		h := buildWorkerTenantHandler(nil, &workerReadyzDeps{cfg: nil})
		assert.NotNil(t, h, "tenant handler must be returned even when cfg is nil")
	})
}

// TestNewWorkerReadyzConfig_NilCfg_FallsBackToLoadConfig verifies the
// nil-cfg guard added in M-B for the worker. A nil Config must not panic
// and must return a usable readyz.Config (env-derived).
func TestNewWorkerReadyzConfig_NilCfg_FallsBackToLoadConfig(t *testing.T) {
	require.NotPanics(t, func() {
		got := newWorkerReadyzConfig(nil)
		require.NotNil(t, got)
		assert.NotEmpty(t, got.Version)
	})
}

func TestBuildWorkerReadyzCheckers_MultiTenantRedisRegistration(t *testing.T) {
	client := newWorkerReadyzMultiTenantRedisClient(&Config{
		MultiTenantEnabled:   true,
		MultiTenantRedisHost: "localhost",
		MultiTenantRedisPort: "6380",
		MultiTenantRedisTLS:  true,
	})
	require.NotNil(t, client)
	t.Cleanup(func() { _ = client.Close() })

	checkers := buildWorkerReadyzCheckers(&workerReadyzDeps{
		cfg: &Config{
			MultiTenantEnabled:   true,
			MultiTenantRedisHost: "localhost",
			MultiTenantRedisPort: "6380",
			MultiTenantRedisTLS:  true,
		},
		mtRedisClient: client,
	})

	checker := findCheckerByName(t, checkers, "multi_tenant_redis")
	require.NotNil(t, checker, "multi-tenant Redis checker must be registered when MT is enabled and host is configured")
	result := checker.Check(context.Background())
	require.NotNil(t, result.TLS)
	assert.True(t, *result.TLS)
}

func TestBuildWorkerReadyzCheckers_MultiTenantRedisOmittedWhenNotConfigured(t *testing.T) {
	checkers := buildWorkerReadyzCheckers(&workerReadyzDeps{cfg: &Config{MultiTenantEnabled: true}})

	assert.Nil(t, findCheckerByName(t, checkers, "multi_tenant_redis"))
}

func TestNewWorkerReadyzMultiTenantRedisClient_TLSPosture(t *testing.T) {
	client := newWorkerReadyzMultiTenantRedisClient(&Config{
		MultiTenantEnabled:   true,
		MultiTenantRedisHost: "localhost",
		MultiTenantRedisTLS:  true,
	})
	require.NotNil(t, client)
	t.Cleanup(func() { _ = client.Close() })

	config := client.Options().TLSConfig
	require.NotNil(t, config)
	assert.Equal(t, uint16(tls.VersionTLS12), config.MinVersion)

	plainClient := newWorkerReadyzMultiTenantRedisClient(&Config{
		MultiTenantEnabled:   true,
		MultiTenantRedisHost: "localhost",
		MultiTenantRedisTLS:  false,
	})
	require.NotNil(t, plainClient)
	t.Cleanup(func() { _ = plainClient.Close() })
	assert.Nil(t, plainClient.Options().TLSConfig)
}
