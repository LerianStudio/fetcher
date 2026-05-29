package bootstrap

import (
	"context"
	"crypto/tls"
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

// TestNewReadyzRedisClient_TLS_EnablesTLSConfig is a regression for the
// CodeRabbit PR #223 finding: the dedicated /readyz Redis client built
// plaintext-only options even when REDIS_TLS=true, so the readyz Ping
// failed with a TLS-on-plaintext error and reported redis=down on
// healthy TLS-only deployments. The fix mirrors the multi-tenant Redis
// client: opts.TLSConfig is populated with TLS 1.2 as the floor.
func TestNewReadyzRedisClient_TLS_EnablesTLSConfig(t *testing.T) {
	cfg := &Config{
		RedisTLS:  true,
		RedisHost: "localhost",
	}

	client := newReadyzRedisClient(cfg)
	require.NotNil(t, client, "client must be returned for non-empty REDIS_HOST")

	t.Cleanup(func() { _ = client.Close() })

	opts := client.Options()
	require.NotNil(t, opts.TLSConfig, "TLSConfig must be non-nil when RedisTLS=true")
	assert.Equal(t, uint16(tls.VersionTLS12), opts.TLSConfig.MinVersion,
		"TLS minimum version must be 1.2 to match the multi-tenant Redis client")
}

// TestNewReadyzRedisClient_NoTLS_NilTLSConfig verifies the counterpart:
// when REDIS_TLS=false, opts.TLSConfig must remain nil so the client
// speaks plaintext. This is the legacy single-tenant local-dev path.
func TestNewReadyzRedisClient_NoTLS_NilTLSConfig(t *testing.T) {
	cfg := &Config{
		RedisTLS:  false,
		RedisHost: "localhost",
	}

	client := newReadyzRedisClient(cfg)
	require.NotNil(t, client)

	t.Cleanup(func() { _ = client.Close() })

	opts := client.Options()
	assert.Nil(t, opts.TLSConfig, "TLSConfig must be nil when RedisTLS=false")
}

// TestBuildManagerReadyzCheckers_MultiTenant_RegistersMTRedisChecker is a
// regression for the CodeRabbit PR #223 finding (#5): when MT is enabled
// and MULTI_TENANT_REDIS_HOST is set, the global /readyz must include a
// `multi_tenant_redis` dep so the event-discovery Redis is observable.
// Previously this function only emitted the schema-cache `redis` and
// `tenant_manager` deps, leaving MT-Redis silently absent.
func TestBuildManagerReadyzCheckers_MultiTenant_RegistersMTRedisChecker(t *testing.T) {
	cfg := &Config{
		MultiTenantEnabled:   true,
		MultiTenantURL:       "http://tenant-manager:8080",
		MultiTenantRedisHost: "mt-redis.local",
		MultiTenantRedisPort: "6379",
	}

	plat := &managerPlatformDependencies{
		multiTenantReadyzRedisClient: newReadyzMultiTenantRedisClient(cfg),
	}
	require.NotNil(t, plat.multiTenantReadyzRedisClient,
		"newReadyzMultiTenantRedisClient must return a non-nil client when MT enabled with a host")

	t.Cleanup(func() { _ = plat.multiTenantReadyzRedisClient.Close() })

	got := buildManagerReadyzCheckers(cfg, nil, plat)

	mt := findCheckerByName(t, got, "multi_tenant_redis")
	require.NotNil(t, mt,
		"multi_tenant_redis checker must be registered when MT enabled with a host")
}

// TestBuildManagerReadyzCheckers_MultiTenant_NoMTRedisHost_OmitsMTChecker
// verifies the inverse: MT enabled but MULTI_TENANT_REDIS_HOST empty must
// NOT emit a multi_tenant_redis checker (a dep that was never configured
// shouldn't show up as a /readyz line).
func TestBuildManagerReadyzCheckers_MultiTenant_NoMTRedisHost_OmitsMTChecker(t *testing.T) {
	cfg := &Config{
		MultiTenantEnabled:   true,
		MultiTenantURL:       "http://tenant-manager:8080",
		MultiTenantRedisHost: "",
	}

	plat := &managerPlatformDependencies{
		multiTenantReadyzRedisClient: newReadyzMultiTenantRedisClient(cfg),
	}
	assert.Nil(t, plat.multiTenantReadyzRedisClient,
		"newReadyzMultiTenantRedisClient must return nil when MULTI_TENANT_REDIS_HOST is empty")

	got := buildManagerReadyzCheckers(cfg, nil, plat)

	mt := findCheckerByName(t, got, "multi_tenant_redis")
	assert.Nil(t, mt,
		"multi_tenant_redis checker must NOT be registered when MULTI_TENANT_REDIS_HOST is empty")
}

// TestBuildManagerReadyzCheckers_SingleTenant_OmitsMTRedisChecker covers
// the third corner: MT disabled — even if a host is somehow configured on
// the cfg, the checker stays out of the registry to avoid misleading
// operators about a dep the manager won't actually probe.
func TestBuildManagerReadyzCheckers_SingleTenant_OmitsMTRedisChecker(t *testing.T) {
	cfg := &Config{
		MultiTenantEnabled:   false,
		MultiTenantRedisHost: "mt-redis.local", // intentionally set; should still be ignored
	}

	plat := &managerPlatformDependencies{
		multiTenantReadyzRedisClient: newReadyzMultiTenantRedisClient(cfg),
	}
	assert.Nil(t, plat.multiTenantReadyzRedisClient,
		"newReadyzMultiTenantRedisClient must return nil when MT is disabled, regardless of host")

	got := buildManagerReadyzCheckers(cfg, nil, plat)

	mt := findCheckerByName(t, got, "multi_tenant_redis")
	assert.Nil(t, mt,
		"multi_tenant_redis checker must NOT be registered when MultiTenantEnabled=false")
}

// TestNewReadyzMultiTenantRedisClient_TLS_EnablesTLSConfig verifies the
// TLS branch on the manager's MT-Redis readyz client mirrors the worker
// side: TLSConfig is populated with TLS 1.2 as the floor when
// MULTI_TENANT_REDIS_TLS=true.
func TestNewReadyzMultiTenantRedisClient_TLS_EnablesTLSConfig(t *testing.T) {
	cfg := &Config{
		MultiTenantEnabled:   true,
		MultiTenantRedisHost: "mt-redis.local",
		MultiTenantRedisTLS:  true,
	}

	client := newReadyzMultiTenantRedisClient(cfg)
	require.NotNil(t, client, "client must be returned for MT-enabled config with non-empty host")

	t.Cleanup(func() { _ = client.Close() })

	opts := client.Options()
	require.NotNil(t, opts.TLSConfig, "TLSConfig must be non-nil when MultiTenantRedisTLS=true")
	assert.Equal(t, uint16(tls.VersionTLS12), opts.TLSConfig.MinVersion,
		"TLS minimum version must be 1.2 to match the event-listener Redis client")
}

// TestNewReadyzMultiTenantRedisClient_NoTLS_NilTLSConfig is the inverse —
// MultiTenantRedisTLS=false leaves TLSConfig nil so the client speaks
// plaintext on the local-dev path.
func TestNewReadyzMultiTenantRedisClient_NoTLS_NilTLSConfig(t *testing.T) {
	cfg := &Config{
		MultiTenantEnabled:   true,
		MultiTenantRedisHost: "mt-redis.local",
		MultiTenantRedisTLS:  false,
	}

	client := newReadyzMultiTenantRedisClient(cfg)
	require.NotNil(t, client)

	t.Cleanup(func() { _ = client.Close() })

	opts := client.Options()
	assert.Nil(t, opts.TLSConfig, "TLSConfig must be nil when MultiTenantRedisTLS=false")
}
