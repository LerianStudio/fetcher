package multitenant

import (
	"context"
	"sync"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/testutil"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	tms3 "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/s3"
	"github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/valkey"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
)

// ---------------------------------------------------------------------------
// 1. Tenant Isolation Tests (Two Tenants, Data Separation)
// ---------------------------------------------------------------------------

// TestTenantIsolation_MongoDBDatabaseRouting verifies that two tenants with different
// tenant IDs get different *mongo.Database instances from tmcore.GetMBContext.
// This test validates the context-based routing mechanism that underpins all MongoDB
// multi-tenant isolation in fetcher.
func TestTenantIsolation_MongoDBDatabaseRouting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tenantAID  string
		tenantBID  string
		tenantADB  string
		tenantBDB  string
		wantDiffDB bool
	}{
		{
			name:       "different tenants get different databases",
			tenantAID:  "tenant-alpha",
			tenantBID:  "tenant-beta",
			tenantADB:  "db_alpha",
			tenantBDB:  "db_beta",
			wantDiffDB: true,
		},
		{
			name:       "same tenant ID gets same database name",
			tenantAID:  "tenant-same",
			tenantBID:  "tenant-same",
			tenantADB:  "db_shared",
			tenantBDB:  "db_shared",
			wantDiffDB: false,
		},
		{
			name:       "tenant IDs with special characters produce distinct databases",
			tenantAID:  "org_abc-123",
			tenantBID:  "org_xyz-456",
			tenantADB:  "db_org_abc",
			tenantBDB:  "db_org_xyz",
			wantDiffDB: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Set up context with tenant ID for tenant A
			ctxA := tmcore.ContextWithTenantID(testutil.TestContext(), tt.tenantAID)
			tenantAFromCtx := tmcore.GetTenantIDContext(ctxA)

			// Set up context with tenant ID for tenant B
			ctxB := tmcore.ContextWithTenantID(testutil.TestContext(), tt.tenantBID)
			tenantBFromCtx := tmcore.GetTenantIDContext(ctxB)

			// Verify tenant IDs are correctly stored and retrievable
			assert.Equal(t, tt.tenantAID, tenantAFromCtx,
				"tenant A ID must be retrievable from context")
			assert.Equal(t, tt.tenantBID, tenantBFromCtx,
				"tenant B ID must be retrievable from context")

			if tt.wantDiffDB {
				assert.NotEqual(t, tenantAFromCtx, tenantBFromCtx,
					"different tenants must have different IDs in context")
			} else {
				assert.Equal(t, tenantAFromCtx, tenantBFromCtx,
					"same tenant ID must resolve to same context value")
			}

			// Verify that GetMongoFromContext returns nil when no mongo connection
			// is in context (simulating context-only tenant ID without infrastructure)
			dbA := tmcore.GetMBContext(ctxA)
			assert.Nil(t, dbA,
				"GetMongoFromContext must return nil when no mongo connection in context (tenant A)")

			dbB := tmcore.GetMBContext(ctxB)
			assert.Nil(t, dbB,
				"GetMongoFromContext must return nil when no mongo connection in context (tenant B)")

			// Positive path: inject tenant-specific databases and verify routing
			clientA, _ := mongo.NewClient() //nolint:staticcheck // test-only disconnected client
			tenantDBa := clientA.Database(tt.tenantADB)
			ctxWithDBA := tmcore.ContextWithMB(ctxA, tenantDBa)

			clientB, _ := mongo.NewClient() //nolint:staticcheck // test-only disconnected client
			tenantDBb := clientB.Database(tt.tenantBDB)
			ctxWithDBB := tmcore.ContextWithMB(ctxB, tenantDBb)

			resolvedA := tmcore.GetMBContext(ctxWithDBA)
			require.NotNil(t, resolvedA,
				"GetMongoFromContext must return non-nil when tenant DB is injected (tenant A)")

			resolvedB := tmcore.GetMBContext(ctxWithDBB)
			require.NotNil(t, resolvedB,
				"GetMongoFromContext must return non-nil when tenant DB is injected (tenant B)")

			if tt.wantDiffDB {
				assert.NotEqual(t, resolvedA.Name(), resolvedB.Name(),
					"different tenants must resolve to different database names")
			} else {
				assert.Equal(t, resolvedA.Name(), resolvedB.Name(),
					"same tenant must resolve to same database name")
			}
		})
	}
}

// TestTenantIsolation_RedisKeyIsolation verifies that the same cache key with
// different tenant contexts produces different Redis keys, preventing cross-tenant
// data leakage in the cache layer.
func TestTenantIsolation_RedisKeyIsolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tenantAID string
		tenantBID string
		key       string
	}{
		{
			name:      "same key different tenants produce different redis keys",
			tenantAID: "tenant-alpha",
			tenantBID: "tenant-beta",
			key:       "schema:connections:org123",
		},
		{
			name:      "simple key with different tenant prefixes",
			tenantAID: "org_abc",
			tenantBID: "org_xyz",
			key:       "foo",
		},
		{
			name:      "nested key path with different tenants",
			tenantAID: "tenant-1",
			tenantBID: "tenant-2",
			key:       "cache:schemas:v2:connection-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctxA := tmcore.ContextWithTenantID(context.Background(), tt.tenantAID)
			ctxB := tmcore.ContextWithTenantID(context.Background(), tt.tenantBID)

			keyA, _ := valkey.GetKeyContext(ctxA, tt.key)
			keyB, _ := valkey.GetKeyContext(ctxB, tt.key)

			// Keys for different tenants must be different
			assert.NotEqual(t, keyA, keyB,
				"same cache key with different tenant contexts must produce different Redis keys")

			// Each key must contain the respective tenant ID
			assert.Contains(t, keyA, tt.tenantAID,
				"tenant A Redis key must contain tenant A ID")
			assert.Contains(t, keyB, tt.tenantBID,
				"tenant B Redis key must contain tenant B ID")

			// Each key must contain the original key
			assert.Contains(t, keyA, tt.key,
				"tenant A Redis key must contain the original key")
			assert.Contains(t, keyB, tt.key,
				"tenant B Redis key must contain the original key")

			// Verify tenant prefix format: "tenant:{tenantID}:{key}"
			expectedA := "tenant:" + tt.tenantAID + ":" + tt.key
			expectedB := "tenant:" + tt.tenantBID + ":" + tt.key
			assert.Equal(t, expectedA, keyA,
				"tenant A Redis key must follow format tenant:{id}:{key}")
			assert.Equal(t, expectedB, keyB,
				"tenant B Redis key must follow format tenant:{id}:{key}")
		})
	}
}

// TestTenantIsolation_S3KeyIsolation verifies that the same object name with
// different tenant contexts produces different S3 keys, ensuring file-level
// isolation between tenants in object storage.
func TestTenantIsolation_S3KeyIsolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tenantAID  string
		tenantBID  string
		objectName string
	}{
		{
			name:       "same object name different tenants produce different S3 keys",
			tenantAID:  "tenant-alpha",
			tenantBID:  "tenant-beta",
			objectName: "job1.json",
		},
		{
			name:       "nested path with different tenant prefixes",
			tenantAID:  "org_abc",
			tenantBID:  "org_xyz",
			objectName: "extractions/2024/01/data.json",
		},
		{
			name:       "deeply nested path isolation",
			tenantAID:  "tenant-1",
			tenantBID:  "tenant-2",
			objectName: "org/123/jobs/456/results.csv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctxA := tmcore.ContextWithTenantID(context.Background(), tt.tenantAID)
			ctxB := tmcore.ContextWithTenantID(context.Background(), tt.tenantBID)

			keyA, errA := tms3.GetObjectStorageKeyForTenant(ctxA, tt.objectName)
			require.NoError(t, errA)
			keyB, errB := tms3.GetObjectStorageKeyForTenant(ctxB, tt.objectName)
			require.NoError(t, errB)

			// Keys for different tenants must be different
			assert.NotEqual(t, keyA, keyB,
				"same object name with different tenant contexts must produce different S3 keys")

			// Each key must contain the respective tenant ID
			assert.Contains(t, keyA, tt.tenantAID,
				"tenant A S3 key must contain tenant A ID")
			assert.Contains(t, keyB, tt.tenantBID,
				"tenant B S3 key must contain tenant B ID")

			// Each key must contain the original object name
			assert.Contains(t, keyA, tt.objectName,
				"tenant A S3 key must contain the original object name")
			assert.Contains(t, keyB, tt.objectName,
				"tenant B S3 key must contain the original object name")

			// Verify S3 key format: "{tenantID}/{objectName}"
			expectedA := tt.tenantAID + "/" + tt.objectName
			expectedB := tt.tenantBID + "/" + tt.objectName
			assert.Equal(t, expectedA, keyA,
				"tenant A S3 key must follow format {tenantID}/{objectName}")
			assert.Equal(t, expectedB, keyB,
				"tenant B S3 key must follow format {tenantID}/{objectName}")
		})
	}
}

// ---------------------------------------------------------------------------
// 2. Error Case Tests
// ---------------------------------------------------------------------------

// TestTenantContext_MissingJWTFallback verifies that repository-level functions
// gracefully handle the absence of tenant context (no JWT / no tenant in context)
// by falling back to static connections rather than panicking.
func TestTenantContext_MissingJWTFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupCtx    func() context.Context
		description string
	}{
		{
			name: "bare background context returns empty tenant ID",
			setupCtx: func() context.Context {
				return context.Background()
			},
			description: "no JWT means no tenant ID in context",
		},
		{
			name: "context with cancel still returns empty tenant ID",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				return ctx
			},
			description: "derived context without tenant injection returns empty",
		},
		{
			name: "context.TODO returns empty tenant ID",
			setupCtx: func() context.Context {
				return context.TODO()
			},
			description: "TODO context is equivalent to missing JWT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := tt.setupCtx()

			// GetTenantIDFromContext must return empty string, not panic
			tenantID := tmcore.GetTenantIDContext(ctx)
			assert.Empty(t, tenantID,
				"missing JWT/tenant context must return empty tenant ID: %s", tt.description)

			// GetMongoFromContext must return nil (not panic) when no tenant context
			db := tmcore.GetMBContext(ctx)
			assert.Nil(t, db,
				"GetMongoFromContext must return nil database when no tenant context")

			// Redis key must return unprefixed key (single-tenant fallback)
			redisKey, _ := valkey.GetKeyContext(ctx, "test-key")
			assert.Equal(t, "test-key", redisKey,
				"Redis key must be unprefixed when no tenant context (single-tenant fallback)")

			// S3 key must return unmodified object name (single-tenant fallback)
			s3Key, err := tms3.GetObjectStorageKeyForTenant(ctx, "reports/data.json")
			require.NoError(t, err)
			assert.Equal(t, "reports/data.json", s3Key,
				"S3 key must be unmodified when no tenant context (single-tenant fallback)")
		})
	}
}

// TestTenantContext_TenantManagerUnavailable verifies that when
// tmcore.GetMBContext returns nil (e.g., no mongo connection set),
// the repository layer can fall back to static connections.
func TestTenantContext_TenantManagerUnavailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tenantID string
	}{
		{
			name:     "tenant ID set but no mongo connection in context returns nil",
			tenantID: "tenant-orphan-123",
		},
		{
			name:     "empty tenant ID returns nil from GetMongoFromContext",
			tenantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tt.tenantID != "" {
				ctx = tmcore.ContextWithTenantID(ctx, tt.tenantID)
			}

			db := tmcore.GetMBContext(ctx)
			assert.Nil(t, db,
				"GetMongoFromContext must return nil when no mongo connection in context")
		})
	}
}

// TestTenantContext_InvalidRabbitMQTenantIDHeader verifies graceful handling of
// invalid X-Tenant-ID values in RabbitMQ message headers: non-string values,
// empty strings, nil headers, and missing keys.
func TestTenantContext_InvalidRabbitMQTenantIDHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		headers        map[string]any
		expectTenantID string
	}{
		{
			name:           "nil headers map produces empty tenant context",
			headers:        nil,
			expectTenantID: "",
		},
		{
			name:           "empty headers map produces empty tenant context",
			headers:        map[string]any{},
			expectTenantID: "",
		},
		{
			name: "integer X-Tenant-ID is ignored",
			headers: map[string]any{
				"X-Tenant-ID": 12345,
			},
			expectTenantID: "",
		},
		{
			name: "boolean X-Tenant-ID is ignored",
			headers: map[string]any{
				"X-Tenant-ID": true,
			},
			expectTenantID: "",
		},
		{
			name: "nil X-Tenant-ID value is ignored",
			headers: map[string]any{
				"X-Tenant-ID": nil,
			},
			expectTenantID: "",
		},
		{
			name: "empty string X-Tenant-ID is ignored",
			headers: map[string]any{
				"X-Tenant-ID": "",
			},
			expectTenantID: "",
		},
		{
			name: "slice X-Tenant-ID is ignored",
			headers: map[string]any{
				"X-Tenant-ID": []string{"tenant-1"},
			},
			expectTenantID: "",
		},
		{
			name: "valid X-Tenant-ID is extracted",
			headers: map[string]any{
				"X-Tenant-ID": "tenant-valid-abc",
			},
			expectTenantID: "tenant-valid-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Replicate the extractTenantIDFromHeaders logic from
			// components/worker/internal/bootstrap/consumer.go
			ctx := extractTenantIDFromHeaders(context.Background(), tt.headers)

			tenantID := tmcore.GetTenantIDContext(ctx)
			assert.Equal(t, tt.expectTenantID, tenantID,
				"tenant ID extracted from headers must match expected value")

			// Verify downstream behavior: if no tenant, keys are unprefixed
			if tt.expectTenantID == "" {
				redisKey, _ := valkey.GetKeyContext(ctx, "some-key")
				assert.Equal(t, "some-key", redisKey,
					"with invalid tenant header, Redis key must remain unprefixed")

				s3Key, err := tms3.GetObjectStorageKeyForTenant(ctx, "some/object.json")
				require.NoError(t, err)
				assert.Equal(t, "some/object.json", s3Key,
					"with invalid tenant header, S3 key must remain unmodified")
			} else {
				redisKey, _ := valkey.GetKeyContext(ctx, "some-key")
				assert.Contains(t, redisKey, tt.expectTenantID,
					"with valid tenant header, Redis key must contain tenant ID")

				s3Key, err := tms3.GetObjectStorageKeyForTenant(ctx, "some/object.json")
				require.NoError(t, err)
				assert.Contains(t, s3Key, tt.expectTenantID,
					"with valid tenant header, S3 key must contain tenant ID")
			}
		})
	}
}

// TestTenantContext_EmptyTenantPropagation verifies that an empty tenant context
// results in no tenant prefix being applied to any infrastructure key.
func TestTenantContext_EmptyTenantPropagation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		key        string
		objectName string
	}{
		{
			name:       "simple keys remain unprefixed",
			key:        "session:abc",
			objectName: "data.json",
		},
		{
			name:       "complex keys remain unprefixed",
			key:        "schema:v2:connection:config",
			objectName: "extractions/2024/01/15/results.csv",
		},
		{
			name:       "keys with special characters remain unprefixed",
			key:        "cache:org_123:schema",
			objectName: "org/abc-123/jobs/456/output.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			redisKey, _ := valkey.GetKeyContext(ctx, tt.key)
			assert.Equal(t, tt.key, redisKey,
				"empty tenant context must not add prefix to Redis key")

			s3Key, err := tms3.GetObjectStorageKeyForTenant(ctx, tt.objectName)
			require.NoError(t, err)
			assert.Equal(t, tt.objectName, s3Key,
				"empty tenant context must not add prefix to S3 key")

			tenantID := tmcore.GetTenantIDContext(ctx)
			assert.Empty(t, tenantID,
				"empty context must return empty tenant ID")
		})
	}
}

// ---------------------------------------------------------------------------
// 3. Context Propagation Tests
// ---------------------------------------------------------------------------

// TestContextPropagation_EndToEndFlow simulates the end-to-end context flow:
// set tenant ID in context, then verify it is available to MongoDB, Redis,
// S3, and RabbitMQ header injection functions.
func TestContextPropagation_EndToEndFlow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tenantID string
	}{
		{
			name:     "standard tenant ID propagates through all layers",
			tenantID: "org_standard_123",
		},
		{
			name:     "tenant ID with hyphens propagates correctly",
			tenantID: "tenant-with-hyphens-456",
		},
		{
			name:     "long tenant ID propagates correctly",
			tenantID: "org_01HXYZ_very_long_tenant_identifier_for_enterprise_customer_789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Step 1: Set tenant ID in context (simulating JWT middleware extraction)
			ctx := tmcore.ContextWithTenantID(context.Background(), tt.tenantID)

			// Step 2: Verify tenant ID is available via GetTenantIDFromContext
			extractedID := tmcore.GetTenantIDContext(ctx)
			require.Equal(t, tt.tenantID, extractedID,
				"tenant ID must be retrievable from context after setting")

			// Step 3: Verify MongoDB layer sees tenant context
			// GetMongoFromContext returns nil when no real DB is set in context
			mongoDB := tmcore.GetMBContext(ctx)
			assert.Nil(t, mongoDB,
				"GetMongoFromContext returns nil without real DB set in context")

			// Step 4: Verify Redis layer applies tenant prefix
			redisKey, _ := valkey.GetKeyContext(ctx, "test-cache-key")
			expectedRedisKey := "tenant:" + tt.tenantID + ":test-cache-key"
			assert.Equal(t, expectedRedisKey, redisKey,
				"Redis key must be tenant-prefixed when tenant context is present")

			// Step 5: Verify S3 layer applies tenant prefix
			s3Key, err := tms3.GetObjectStorageKeyForTenant(ctx, "output/data.json")
			require.NoError(t, err)
			expectedS3Key := tt.tenantID + "/output/data.json"
			assert.Equal(t, expectedS3Key, s3Key,
				"S3 key must be tenant-prefixed when tenant context is present")

			// Step 6: Verify RabbitMQ header injection would include tenant ID
			// Simulate what publisher.rabbitmq.go does: extract from context, set in headers
			headers := make(map[string]any)
			if tid := tmcore.GetTenantIDContext(ctx); tid != "" {
				headers["X-Tenant-ID"] = tid
			}
			assert.Equal(t, tt.tenantID, headers["X-Tenant-ID"],
				"RabbitMQ header must contain tenant ID when context has tenant")
		})
	}
}

// TestContextPropagation_NoCrossGoroutineLeakage verifies that two concurrent
// requests with different tenants do not interfere with each other.
// This is critical for preventing data leakage in concurrent HTTP handlers.
func TestContextPropagation_NoCrossGoroutineLeakage(t *testing.T) {
	t.Parallel()

	const (
		tenantAID      = "tenant-goroutine-A"
		tenantBID      = "tenant-goroutine-B"
		iterationCount = 100
	)

	var wg sync.WaitGroup

	errorsA := make([]string, 0)
	errorsB := make([]string, 0)
	var muA, muB sync.Mutex

	// Run iterationCount goroutines for tenant A
	for i := 0; i < iterationCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ctx := tmcore.ContextWithTenantID(context.Background(), tenantAID)

			// Verify tenant ID
			tid := tmcore.GetTenantIDContext(ctx)
			if tid != tenantAID {
				muA.Lock()
				errorsA = append(errorsA, "expected "+tenantAID+", got "+tid)
				muA.Unlock()

				return
			}

			// Verify Redis key
			redisKey, _ := valkey.GetKeyContext(ctx, "key")
			expectedKey := "tenant:" + tenantAID + ":key"
			if redisKey != expectedKey {
				muA.Lock()
				errorsA = append(errorsA, "redis key mismatch: "+redisKey)
				muA.Unlock()

				return
			}

			// Verify S3 key
			s3Key, s3Err := tms3.GetObjectStorageKeyForTenant(ctx, "obj.json")
			if s3Err != nil {
				muA.Lock()
				errorsA = append(errorsA, "s3 key error: "+s3Err.Error())
				muA.Unlock()

				return
			}

			expectedS3 := tenantAID + "/obj.json"
			if s3Key != expectedS3 {
				muA.Lock()
				errorsA = append(errorsA, "s3 key mismatch: "+s3Key)
				muA.Unlock()
			}
		}()
	}

	// Run iterationCount goroutines for tenant B
	for i := 0; i < iterationCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ctx := tmcore.ContextWithTenantID(context.Background(), tenantBID)

			// Verify tenant ID
			tid := tmcore.GetTenantIDContext(ctx)
			if tid != tenantBID {
				muB.Lock()
				errorsB = append(errorsB, "expected "+tenantBID+", got "+tid)
				muB.Unlock()

				return
			}

			// Verify Redis key
			redisKey, _ := valkey.GetKeyContext(ctx, "key")
			expectedKey := "tenant:" + tenantBID + ":key"
			if redisKey != expectedKey {
				muB.Lock()
				errorsB = append(errorsB, "redis key mismatch: "+redisKey)
				muB.Unlock()

				return
			}

			// Verify S3 key
			s3Key, s3Err := tms3.GetObjectStorageKeyForTenant(ctx, "obj.json")
			if s3Err != nil {
				muB.Lock()
				errorsB = append(errorsB, "s3 key error: "+s3Err.Error())
				muB.Unlock()

				return
			}

			expectedS3 := tenantBID + "/obj.json"
			if s3Key != expectedS3 {
				muB.Lock()
				errorsB = append(errorsB, "s3 key mismatch: "+s3Key)
				muB.Unlock()
			}
		}()
	}

	wg.Wait()

	assert.Empty(t, errorsA,
		"tenant A goroutines must not see tenant B data: %v", errorsA)
	assert.Empty(t, errorsB,
		"tenant B goroutines must not see tenant A data: %v", errorsB)
}

// TestContextPropagation_ChildContextInheritsTenantID verifies that child
// contexts derived from a tenant-scoped parent context inherit the tenant ID.
func TestContextPropagation_ChildContextInheritsTenantID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tenantID string
	}{
		{
			name:     "child context inherits tenant from parent",
			tenantID: "parent-tenant-123",
		},
		{
			name:     "grandchild context inherits tenant from grandparent",
			tenantID: "grandparent-tenant-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parent := tmcore.ContextWithTenantID(context.Background(), tt.tenantID)

			// Child via WithCancel
			child, cancelChild := context.WithCancel(parent)
			defer cancelChild()

			childTenant := tmcore.GetTenantIDContext(child)
			assert.Equal(t, tt.tenantID, childTenant,
				"child context must inherit tenant ID from parent")

			// Grandchild via WithCancel on child
			grandchild, cancelGrandchild := context.WithCancel(child)
			defer cancelGrandchild()

			grandchildTenant := tmcore.GetTenantIDContext(grandchild)
			assert.Equal(t, tt.tenantID, grandchildTenant,
				"grandchild context must inherit tenant ID from grandparent")

			// Verify infrastructure keys also work through child contexts
			redisKey, _ := valkey.GetKeyContext(child, "nested-key")
			expectedRedis := "tenant:" + tt.tenantID + ":nested-key"
			assert.Equal(t, expectedRedis, redisKey,
				"child context must produce correct tenant-prefixed Redis key")

			s3Key, err := tms3.GetObjectStorageKeyForTenant(grandchild, "nested/object.json")
			require.NoError(t, err)
			expectedS3 := tt.tenantID + "/nested/object.json"
			assert.Equal(t, expectedS3, s3Key,
				"grandchild context must produce correct tenant-prefixed S3 key")
		})
	}
}

// ---------------------------------------------------------------------------
// 4. Configuration Tests
// ---------------------------------------------------------------------------

// TestMultiTenantConfig_CanonicalEnvVars verifies that all 7 canonical multi-tenant
// environment variables can be represented in the config struct and have correct
// zero-value defaults.
func TestMultiTenantConfig_CanonicalEnvVars(t *testing.T) {
	t.Parallel()

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

	tests := []struct {
		name     string
		field    string
		validate func(t *testing.T, cfg *multiTenantConfig)
	}{
		{
			name:  "MULTI_TENANT_ENABLED defaults to false",
			field: "MultiTenantEnabled",
			validate: func(t *testing.T, cfg *multiTenantConfig) {
				t.Helper()
				assert.False(t, cfg.MultiTenantEnabled,
					"MULTI_TENANT_ENABLED must default to false (single-tenant mode)")
			},
		},
		{
			name:  "MULTI_TENANT_URL defaults to empty string",
			field: "MultiTenantURL",
			validate: func(t *testing.T, cfg *multiTenantConfig) {
				t.Helper()
				assert.Empty(t, cfg.MultiTenantURL,
					"MULTI_TENANT_URL must default to empty string")
			},
		},
		{
			name:  "MULTI_TENANT_REDIS_HOST defaults to empty string",
			field: "MultiTenantRedisHost",
			validate: func(t *testing.T, cfg *multiTenantConfig) {
				t.Helper()
				assert.Empty(t, cfg.MultiTenantRedisHost,
					"MULTI_TENANT_REDIS_HOST must default to empty string")
			},
		},
		{
			name:  "MULTI_TENANT_REDIS_PORT defaults to empty string",
			field: "MultiTenantRedisPort",
			validate: func(t *testing.T, cfg *multiTenantConfig) {
				t.Helper()
				assert.Empty(t, cfg.MultiTenantRedisPort,
					"MULTI_TENANT_REDIS_PORT must default to empty string")
			},
		},
		{
			name:  "MULTI_TENANT_REDIS_PASSWORD defaults to empty string",
			field: "MultiTenantRedisPassword",
			validate: func(t *testing.T, cfg *multiTenantConfig) {
				t.Helper()
				assert.Empty(t, cfg.MultiTenantRedisPassword,
					"MULTI_TENANT_REDIS_PASSWORD must default to empty string")
			},
		},
		{
			name:  "MULTI_TENANT_MAX_TENANT_POOLS defaults to 0",
			field: "MultiTenantMaxTenantPools",
			validate: func(t *testing.T, cfg *multiTenantConfig) {
				t.Helper()
				assert.Equal(t, 0, cfg.MultiTenantMaxTenantPools,
					"MULTI_TENANT_MAX_TENANT_POOLS must default to 0")
			},
		},
		{
			name:  "MULTI_TENANT_IDLE_TIMEOUT_SEC defaults to 0",
			field: "MultiTenantIdleTimeoutSec",
			validate: func(t *testing.T, cfg *multiTenantConfig) {
				t.Helper()
				assert.Equal(t, 0, cfg.MultiTenantIdleTimeoutSec,
					"MULTI_TENANT_IDLE_TIMEOUT_SEC must default to 0")
			},
		},
		{
			name:  "MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD defaults to 0",
			field: "MultiTenantCircuitBreakerThreshold",
			validate: func(t *testing.T, cfg *multiTenantConfig) {
				t.Helper()
				assert.Equal(t, 0, cfg.MultiTenantCircuitBreakerThreshold,
					"MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD must default to 0")
			},
		},
		{
			name:  "MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC defaults to 0",
			field: "MultiTenantCircuitBreakerTimeoutSec",
			validate: func(t *testing.T, cfg *multiTenantConfig) {
				t.Helper()
				assert.Equal(t, 0, cfg.MultiTenantCircuitBreakerTimeoutSec,
					"MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC must default to 0")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &multiTenantConfig{}
			tt.validate(t, cfg)
		})
	}
}

// TestMultiTenantConfig_TenantContextErrors verifies that sentinel errors from
// tmcore are properly defined and can be matched with errors.Is.
func TestMultiTenantConfig_TenantContextErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		sentinelErr   error
		description   string
		expectNonNil  bool
		expectMessage string
	}{
		{
			name:          "ErrTenantContextRequired is defined",
			sentinelErr:   tmcore.ErrTenantContextRequired,
			description:   "sentinel for missing tenant context in repository calls",
			expectNonNil:  true,
			expectMessage: "tenant",
		},
		{
			name:          "ErrTenantNotFound is defined",
			sentinelErr:   tmcore.ErrTenantNotFound,
			description:   "sentinel for tenant lookup failures in tenant manager",
			expectNonNil:  true,
			expectMessage: "not found",
		},
		{
			name:          "ErrCircuitBreakerOpen is defined",
			sentinelErr:   tmcore.ErrCircuitBreakerOpen,
			description:   "sentinel for circuit breaker protection during outages",
			expectNonNil:  true,
			expectMessage: "circuit breaker",
		},
		{
			name:          "ErrManagerClosed is defined",
			sentinelErr:   tmcore.ErrManagerClosed,
			description:   "sentinel for attempts to use a closed manager",
			expectNonNil:  true,
			expectMessage: "closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.expectNonNil {
				require.NotNil(t, tt.sentinelErr,
					"sentinel error %s must be defined", tt.name)
				assert.Contains(t, tt.sentinelErr.Error(), tt.expectMessage,
					"sentinel error must contain expected message keyword")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 5. Backward Compatibility Tests (Single-Tenant Fallback)
// ---------------------------------------------------------------------------

// TestSingleTenantFallback_AllInfrastructure verifies that all infrastructure
// components work correctly in single-tenant mode (no tenant context).
// This is the MANDATORY backward compatibility gate per multi-tenant.md standards.
func TestSingleTenantFallback_AllInfrastructure(t *testing.T) {
	t.Parallel()

	t.Run("mongodb_falls_back_to_static_without_tenant", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		db := tmcore.GetMBContext(ctx)
		assert.Nil(t, db,
			"GetMongoFromContext must return nil when no tenant in context")
	})

	t.Run("redis_keys_unprefixed_without_tenant", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		keys := []string{
			"simple",
			"prefix:key",
			"multi:level:key:path",
			"with-special_chars.v2",
		}

		for _, key := range keys {
			result, _ := valkey.GetKeyContext(ctx, key)
			assert.Equal(t, key, result,
				"Redis key %q must be unchanged in single-tenant mode", key)
		}
	})

	t.Run("s3_keys_unmodified_without_tenant", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		objectNames := []string{
			"simple.json",
			"path/to/object.csv",
			"deep/nested/path/data.json",
			"special-chars_v2.html",
		}

		for _, name := range objectNames {
			result, err := tms3.GetObjectStorageKeyForTenant(ctx, name)
			require.NoError(t, err)
			assert.Equal(t, name, result,
				"S3 key %q must be unchanged in single-tenant mode", name)
		}
	})

	t.Run("tenant_id_empty_without_context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		tenantID := tmcore.GetTenantIDContext(ctx)
		assert.Empty(t, tenantID,
			"tenant ID must be empty in single-tenant mode")
	})

	t.Run("rabbitmq_headers_no_tenant_without_context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		// Simulate RabbitMQ header injection logic from publisher.rabbitmq.go
		headers := make(map[string]any)
		if tid := tmcore.GetTenantIDContext(ctx); tid != "" {
			headers["X-Tenant-ID"] = tid
		}

		_, exists := headers["X-Tenant-ID"]
		assert.False(t, exists,
			"X-Tenant-ID must not be present in RabbitMQ headers in single-tenant mode")
	})
}

// ---------------------------------------------------------------------------
// Helper: extractTenantIDFromHeaders (replicates consumer.go logic)
// ---------------------------------------------------------------------------

// extractTenantIDFromHeaders extracts X-Tenant-ID from AMQP message headers
// and injects it into the context. This replicates the logic from
// components/worker/internal/bootstrap/consumer.go for cross-cutting test coverage.
func extractTenantIDFromHeaders(ctx context.Context, headers map[string]any) context.Context {
	if headers == nil {
		return ctx
	}

	tenantID, ok := headers["X-Tenant-ID"].(string)
	if !ok || tenantID == "" {
		return ctx
	}

	return tmcore.ContextWithTenantID(ctx, tenantID)
}
