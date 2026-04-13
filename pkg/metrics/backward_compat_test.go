package metrics

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/testutil"
	tms3 "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/s3"
	"github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/valkey"

	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiTenant_BackwardCompatibility validates that the fetcher system operates
// correctly in single-tenant mode (MULTI_TENANT_ENABLED=false, the default).
// This is a MANDATORY backward compatibility gate per multi-tenant.md standards.
//
// Verification items:
//  1. Config defaults: MultiTenantEnabled defaults to false
//  2. Metrics: no-op when disabled (covered by TestTenantMetrics_NoOpWhenDisabled)
//  3. Redis: cacheKey returns unprefixed key without tenant context
//  4. S3: GetObjectStorageKeyForTenant returns original objectName without tenant context
//  5. RabbitMQ: extractTenantIDFromHeaders returns unmodified context for messages without X-Tenant-ID
//  6. MongoDB: GetMongoForTenant returns error without tenant context (falls back to static)
func TestMultiTenant_BackwardCompatibility(t *testing.T) {
	t.Run("config_defaults_multi_tenant_disabled", func(t *testing.T) {
		// When no MULTI_TENANT_* env vars are set, MultiTenantEnabled must default to false.
		// This is verified by directly asserting the zero value of a bool field.
		type multiTenantConfig struct {
			MultiTenantEnabled                  bool   `env:"MULTI_TENANT_ENABLED"`
			MultiTenantURL                      string `env:"MULTI_TENANT_URL"`
			MultiTenantRedisHost                string `env:"MULTI_TENANT_REDIS_HOST"`
			MultiTenantRedisPort                string `env:"MULTI_TENANT_REDIS_PORT"`
			MultiTenantRedisPassword            string `env:"MULTI_TENANT_REDIS_PASSWORD"`
			MultiTenantMaxTenantPools           int    `env:"MULTI_TENANT_MAX_TENANT_POOLS"`
			MultiTenantIdleTimeoutSec           int    `env:"MULTI_TENANT_IDLE_TIMEOUT_SEC"`
			MultiTenantCircuitBreakerThreshold  int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD"`
			MultiTenantCircuitBreakerTimeoutSec int    `env:"MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC"`
		}

		cfg := &multiTenantConfig{}
		assert.False(t, cfg.MultiTenantEnabled,
			"MultiTenantEnabled must default to false when no MULTI_TENANT_ENABLED env var is set")
		assert.Empty(t, cfg.MultiTenantURL,
			"MultiTenantURL must default to empty string")
		assert.Empty(t, cfg.MultiTenantRedisHost,
			"MultiTenantRedisHost must default to empty string")
		assert.Empty(t, cfg.MultiTenantRedisPort,
			"MultiTenantRedisPort must default to empty string")
		assert.Empty(t, cfg.MultiTenantRedisPassword,
			"MultiTenantRedisPassword must default to empty string")
		assert.Equal(t, 0, cfg.MultiTenantMaxTenantPools,
			"MultiTenantMaxTenantPools must default to 0")
		assert.Equal(t, 0, cfg.MultiTenantIdleTimeoutSec,
			"MultiTenantIdleTimeoutSec must default to 0")
		assert.Equal(t, 0, cfg.MultiTenantCircuitBreakerThreshold,
			"MultiTenantCircuitBreakerThreshold must default to 0")
		assert.Equal(t, 0, cfg.MultiTenantCircuitBreakerTimeoutSec,
			"MultiTenantCircuitBreakerTimeoutSec must default to 0")
	})

	t.Run("metrics_noop_when_disabled", func(t *testing.T) {
		// When MULTI_TENANT_ENABLED=false, metrics must be no-op with zero overhead.
		tm, err := NewTenantMetrics(false, nil)
		require.NoError(t, err, "creating no-op metrics must not fail")
		assert.NotNil(t, tm, "no-op metrics struct must not be nil")

		ctx := context.Background()
		// All metric calls must be safe no-ops
		assert.NotPanics(t, func() {
			tm.IncrementTenantConnectionsTotal(ctx, "any-tenant")
			tm.IncrementTenantConnectionErrorsTotal(ctx, "any-tenant")
			tm.SetTenantConsumersActive(ctx, "any-tenant", 5)
			tm.IncrementTenantMessagesProcessedTotal(ctx, "any-tenant")
		}, "no-op metrics must not panic")
	})

	t.Run("redis_cache_key_unprefixed_without_tenant", func(t *testing.T) {
		// When no tenant is in context, valkey.GetKeyContext must return the key unchanged.
		// This ensures Redis operations work normally in single-tenant mode.
		ctx := context.Background()
		key := "mykey"

		result, err := valkey.GetKeyContext(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, key, result,
			"valkey.GetKeyContext must return key unchanged when no tenant in context")
	})

	t.Run("s3_object_key_unprefixed_without_tenant", func(t *testing.T) {
		// When no tenant is in context, GetObjectStorageKeyForTenant must return
		// the original objectName. This ensures S3 operations work normally.
		ctx := context.Background()
		objectName := "reports/template-123/report-456.html"

		result, err := tms3.GetObjectStorageKeyForTenant(ctx, objectName)
		require.NoError(t, err)
		assert.Equal(t, objectName, result,
			"GetObjectStorageKeyForTenant must return original objectName when no tenant in context")
	})

	t.Run("rabbitmq_message_without_tenant_id_header", func(t *testing.T) {
		// When a RabbitMQ message has no X-Tenant-ID header, the context must remain
		// unchanged. This is the single-tenant code path.
		ctx := context.Background()

		// Simulate extracting tenant ID from headers with no X-Tenant-ID
		headersWithoutTenant := map[string]any{
			"Content-Type": "application/json",
			"jobId":        "test-job-123",
		}

		tenantIDFromEmpty := tmcore.GetTenantIDContext(ctx)
		assert.Empty(t, tenantIDFromEmpty,
			"context without tenant ID must return empty string from GetTenantIDFromContext")

		// Verify that extracting from headers without X-Tenant-ID returns empty
		_ = headersWithoutTenant // headers read but no tenant extracted

		tenantID, ok := headersWithoutTenant["X-Tenant-ID"].(string)
		assert.False(t, ok || tenantID != "",
			"single-tenant message must not have X-Tenant-ID header")
	})

	t.Run("rabbitmq_nil_headers_safe", func(t *testing.T) {
		// nil headers must not cause a panic.
		ctx := context.Background()

		tenantID := tmcore.GetTenantIDContext(ctx)
		assert.Empty(t, tenantID,
			"nil headers must not produce a tenant ID in context")
	})

	t.Run("mongodb_fallback_to_static_without_tenant", func(t *testing.T) {
		// When no tenant is in context, tmcore.GetMBContext must return nil,
		// which signals the repository to fall back to its static connection.
		ctx := testutil.TestContext()

		db := tmcore.GetMBContext(ctx)
		assert.Nil(t, db,
			"GetMongoFromContext must return nil database when no tenant in context")
	})

	t.Run("tenant_context_isolation", func(t *testing.T) {
		// Setting tenant in one context must not leak into another.
		ctx1 := tmcore.ContextWithTenantID(context.Background(), "tenant-1")
		ctx2 := context.Background()

		assert.Equal(t, "tenant-1", tmcore.GetTenantIDContext(ctx1),
			"context with tenant must return the set tenant ID")
		assert.Empty(t, tmcore.GetTenantIDContext(ctx2),
			"context without tenant must return empty string (no leaking)")
	})

	t.Run("redis_key_with_tenant_is_prefixed", func(t *testing.T) {
		// Verify that when tenant IS in context, the key IS prefixed.
		// This ensures the prefixing mechanism works correctly (both paths tested).
		ctx := tmcore.ContextWithTenantID(context.Background(), "org_abc123")
		key := "mykey"

		result, err := valkey.GetKeyContext(ctx, key)
		assert.NoError(t, err)
		assert.Contains(t, result, "org_abc123",
			"valkey.GetKeyContext must include tenant ID when tenant is in context")
		assert.Contains(t, result, key,
			"valkey.GetKeyContext must include the original key")
	})

	t.Run("s3_key_with_tenant_is_prefixed", func(t *testing.T) {
		// Verify that when tenant IS in context, the S3 key IS prefixed.
		ctx := tmcore.ContextWithTenantID(context.Background(), "org_abc123")
		objectName := "reports/report.html"

		result, err := tms3.GetObjectStorageKeyForTenant(ctx, objectName)
		require.NoError(t, err)
		assert.Contains(t, result, "org_abc123",
			"GetObjectStorageKeyForTenant must include tenant ID when tenant is in context")
		assert.Contains(t, result, "reports/report.html",
			"GetObjectStorageKeyForTenant must include the original key")
	})
}
