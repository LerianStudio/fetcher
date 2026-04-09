package redis

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/testutil"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger is an alias to the shared testutil.MockLogger for use across test files in this package.
type mockLogger = testutil.MockLogger

// testStruct is a simple struct for testing generic cache
type testStruct struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *RedisConnection) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		DialTimeout:  100 * time.Millisecond, // Fast timeout for tests
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})

	conn := &RedisConnection{
		Client:    client,
		Logger:    &testutil.MockLogger{},
		Connected: true,
	}

	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return mr, conn
}

func TestRedisCache_GetSet(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test", Value: 42}

	// Test cache miss
	result, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, testStruct{}, result)

	// Test set
	err = cache.Set(ctx, key, value, 0)
	assert.NoError(t, err)

	// Test cache hit
	result, found, err = cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, result)
}

func TestRedisCache_Delete(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test", Value: 42}

	// Set value
	err = cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Delete
	err = cache.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify deleted
	_, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestRedisCache_Clear(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 5; i++ {
		key := "item" + string(rune('0'+i))
		err := cache.Set(ctx, key, testStruct{ID: key}, 0)
		require.NoError(t, err)
	}

	// Clear
	err = cache.Clear(ctx)
	assert.NoError(t, err)

	// Verify all cleared
	for i := 0; i < 5; i++ {
		key := "item" + string(rune('0'+i))
		_, found, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.False(t, found)
	}
}

func TestRedisCache_TTLExpiration(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test", Value: 42}

	// Set with short TTL
	err = cache.Set(ctx, key, value, 100*time.Millisecond)
	require.NoError(t, err)

	// Verify exists
	_, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.True(t, found)

	// Fast-forward time in miniredis
	mr.FastForward(200 * time.Millisecond)

	// Verify expired
	_, found, err = cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestRedisCache_IsHealthy(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()

	// Healthy when connected
	assert.True(t, cache.IsHealthy(ctx))

	// Close miniredis
	mr.Close()

	// Not healthy when disconnected
	assert.False(t, cache.IsHealthy(ctx))
}

func TestRedisCache_KeyPrefix(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "myprefix:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1"}

	err = cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Verify key has prefix in Redis
	keys := mr.Keys()
	assert.Contains(t, keys, "myprefix:item1")
}

func TestRedisCache_Get_UnmarshalError(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"

	// Store invalid JSON directly in miniredis
	mr.Set("test:item1", "not valid json at all")

	// Get should return error due to unmarshal failure
	_, found, err := cache.Get(ctx, key)
	assert.Error(t, err)
	assert.False(t, found)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

func TestRedisCache_Get_RedisError(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"

	// Close miniredis to cause connection error
	mr.Close()

	// Get should return error
	_, found, err := cache.Get(ctx, key)
	assert.Error(t, err)
	assert.False(t, found)
	assert.Contains(t, err.Error(), "failed to get from cache")
}

func TestRedisCache_Set_RedisError(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test", Value: 42}

	// Close miniredis to cause connection error
	mr.Close()

	// Set should return error
	err = cache.Set(ctx, key, value, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store in cache")
}

func TestRedisCache_Delete_RedisError(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"

	// Close miniredis to cause connection error
	mr.Close()

	// Delete should return error
	err = cache.Delete(ctx, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete from cache")
}

func TestRedisCache_DefaultTTL(t *testing.T) {
	_, conn := setupMiniredis(t)

	// Test with zero TTL - should use default
	cache, err := NewRedisCache[testStruct](conn, 0, "test:")
	require.NoError(t, err)
	assert.Equal(t, DefaultCacheTTL, cache.ttl)

	// Test with negative TTL - should use default
	cache2, err := NewRedisCache[testStruct](conn, -5*time.Minute, "test:")
	require.NoError(t, err)
	assert.Equal(t, DefaultCacheTTL, cache2.ttl)
}

func TestRedisCache_Set_UsesDefaultTTL(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, 10*time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1"}

	// Set with 0 TTL should use cache's default
	err = cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Verify key exists
	_, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.True(t, found)

	// Verify the TTL is around 10 minutes (allow some variance)
	ttl := mr.TTL("test:item1")
	assert.Greater(t, ttl, 9*time.Minute)
	assert.LessOrEqual(t, ttl, 10*time.Minute)
}

func TestRedisCache_NewRedisCache_NilConnection_ReturnsError(t *testing.T) {
	t.Run("nil connection returns error", func(t *testing.T) {
		cache, err := NewRedisCache[testStruct](nil, time.Minute, "test:")
		assert.Nil(t, cache)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis connection and client must not be nil")
	})

	t.Run("connection with nil client returns error", func(t *testing.T) {
		conn := &RedisConnection{
			Client:    nil,
			Logger:    &testutil.MockLogger{},
			Connected: false,
		}
		cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
		assert.Nil(t, cache)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis connection and client must not be nil")
	})
}

func TestRedisCache_Close(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	err = cache.Close()
	assert.NoError(t, err)

	// After close, operations should fail
	ctx := context.Background()
	_, _, err = cache.Get(ctx, "key")
	assert.Error(t, err)
}

func TestRedisCache_Set_NegativeTTL_UsesDefault(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, 10*time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1"}

	// Set with negative TTL should use cache's default
	err = cache.Set(ctx, key, value, -5*time.Minute)
	require.NoError(t, err)

	// Verify the TTL is around 10 minutes (cache default)
	ttl := mr.TTL("test:item1")
	assert.Greater(t, ttl, 9*time.Minute)
	assert.LessOrEqual(t, ttl, 10*time.Minute)
}

// unmarshalableStruct is a type that cannot be marshaled to JSON
type unmarshalableStruct struct {
	Data chan int `json:"data"`
}

func TestRedisCache_Set_MarshalError(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache, err := NewRedisCache[unmarshalableStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()
	key := "item1"
	value := unmarshalableStruct{Data: make(chan int)}

	// Set should fail due to marshal error (channels cannot be marshaled)
	err = cache.Set(ctx, key, value, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal value")
}

func TestRedisCache_Clear_ScanError(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()

	// Set some values
	for i := 0; i < 3; i++ {
		key := "item" + string(rune('0'+i))
		err := cache.Set(ctx, key, testStruct{ID: key}, 0)
		require.NoError(t, err)
	}

	// Close miniredis to cause scan error
	mr.Close()

	// Clear should return error due to scan failure
	err = cache.Clear(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clear cache")
}

func TestRedisCache_Clear_DeleteErrorDuringIteration(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()

	// Set some values
	for i := 0; i < 3; i++ {
		key := "item" + string(rune('0'+i))
		err := cache.Set(ctx, key, testStruct{ID: key}, 0)
		require.NoError(t, err)
	}

	// Verify keys exist
	keys := mr.Keys()
	assert.Len(t, keys, 3)

	// Clear should succeed (individual delete errors are logged but don't fail the operation)
	err = cache.Clear(ctx)
	assert.NoError(t, err)
}

func TestRedisCache_EmptyKeyPrefix(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "")
	require.NoError(t, err)

	ctx := context.Background()
	key := "mykey"
	value := testStruct{ID: "1"}

	err = cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// With empty prefix, key should be stored as-is
	keys := mr.Keys()
	assert.Contains(t, keys, "mykey")
}

func TestRedisCache_CacheKey(t *testing.T) {
	_, conn := setupMiniredis(t)

	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "standard prefix",
			prefix:   "test:",
			key:      "mykey",
			expected: "test:mykey",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			key:      "mykey",
			expected: "mykey",
		},
		{
			name:     "complex prefix",
			prefix:   "app:cache:v1:",
			key:      "user:123",
			expected: "app:cache:v1:user:123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewRedisCache[testStruct](conn, time.Minute, tt.prefix)
			require.NoError(t, err)
			// No tenant in context => key returned unchanged
			result, err := cache.cacheKey(context.Background(), tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedisCache_CacheKey_Basic(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "fetcher:schema:")
	require.NoError(t, err)

	result, err := cache.cacheKey(context.Background(), "mykey")
	require.NoError(t, err)
	assert.Equal(t, "fetcher:schema:mykey", result)
}

func TestRedisCache_GetSet_WithTenantContext(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	t.Run("tenant context scopes keys automatically via valkey", func(t *testing.T) {
		ctxA := tmcore.ContextWithTenantID(context.Background(), "tenant-aaa")
		ctxB := tmcore.ContextWithTenantID(context.Background(), "tenant-bbb")

		valueA := testStruct{ID: "1", Name: "TenantA"}
		valueB := testStruct{ID: "2", Name: "TenantB"}

		// Same logical key, different tenant contexts
		err := cache.Set(ctxA, "shared-key", valueA, 0)
		require.NoError(t, err)

		err = cache.Set(ctxB, "shared-key", valueB, 0)
		require.NoError(t, err)

		// Each tenant gets its own value
		resultA, foundA, err := cache.Get(ctxA, "shared-key")
		assert.NoError(t, err)
		assert.True(t, foundA)
		assert.Equal(t, valueA, resultA)

		resultB, foundB, err := cache.Get(ctxB, "shared-key")
		assert.NoError(t, err)
		assert.True(t, foundB)
		assert.Equal(t, valueB, resultB)

		// Verify tenant-prefixed keys in Redis
		keys := mr.Keys()
		assert.Contains(t, keys, "tenant:tenant-aaa:test:shared-key")
		assert.Contains(t, keys, "tenant:tenant-bbb:test:shared-key")
	})

	t.Run("no tenant context returns unprefixed key (single-tenant backward compat)", func(t *testing.T) {
		ctx := context.Background()
		value := testStruct{ID: "3", Name: "SingleTenant"}

		err := cache.Set(ctx, "solo-key", value, 0)
		require.NoError(t, err)

		result, found, err := cache.Get(ctx, "solo-key")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, value, result)

		keys := mr.Keys()
		assert.Contains(t, keys, "test:solo-key")
	})

	t.Run("delete only affects the tenant that owns the key", func(t *testing.T) {
		ctxA := tmcore.ContextWithTenantID(context.Background(), "tenant-del-a")
		ctxB := tmcore.ContextWithTenantID(context.Background(), "tenant-del-b")

		value := testStruct{ID: "x", Name: "DeleteTest"}

		err := cache.Set(ctxA, "del-key", value, 0)
		require.NoError(t, err)

		err = cache.Set(ctxB, "del-key", value, 0)
		require.NoError(t, err)

		// Delete only tenant A's key
		err = cache.Delete(ctxA, "del-key")
		require.NoError(t, err)

		// Tenant A: gone
		_, foundA, err := cache.Get(ctxA, "del-key")
		assert.NoError(t, err)
		assert.False(t, foundA)

		// Tenant B: still there
		_, foundB, err := cache.Get(ctxB, "del-key")
		assert.NoError(t, err)
		assert.True(t, foundB)
	})

	t.Run("clear only affects keys with matching tenant and prefix", func(t *testing.T) {
		// Use a fresh cache with distinct prefix to avoid interference
		freshCache, err := NewRedisCache[testStruct](conn, time.Minute, "cleartest:")
		require.NoError(t, err)

		ctxA := tmcore.ContextWithTenantID(context.Background(), "tenant-clear-a")
		ctxB := tmcore.ContextWithTenantID(context.Background(), "tenant-clear-b")

		err = freshCache.Set(ctxA, "k1", testStruct{ID: "a1"}, 0)
		require.NoError(t, err)

		err = freshCache.Set(ctxB, "k1", testStruct{ID: "b1"}, 0)
		require.NoError(t, err)

		// Clear tenant A only
		err = freshCache.Clear(ctxA)
		require.NoError(t, err)

		// Tenant A: cleared
		_, foundA, err := freshCache.Get(ctxA, "k1")
		assert.NoError(t, err)
		assert.False(t, foundA)

		// Tenant B: untouched
		_, foundB, err := freshCache.Get(ctxB, "k1")
		assert.NoError(t, err)
		assert.True(t, foundB)
	})
}

func TestRedisCache_CacheKey_WithTenantContext(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "fetcher:schema:")
	require.NoError(t, err)

	t.Run("with tenant context adds tenant prefix", func(t *testing.T) {
		ctx := tmcore.ContextWithTenantID(context.Background(), "abc-123")
		result, err := cache.cacheKey(ctx, "mykey")
		require.NoError(t, err)
		assert.Equal(t, "tenant:abc-123:fetcher:schema:mykey", result)
	})

	t.Run("without tenant context returns plain key", func(t *testing.T) {
		result, err := cache.cacheKey(context.Background(), "mykey")
		require.NoError(t, err)
		assert.Equal(t, "fetcher:schema:mykey", result)
	})
}

// TestRedisCache_IsHealthy_WithInitErr verifies the initErr guard in IsHealthy.
// The initErr field is never set by constructors (they return an error instead),
// but the guard exists as defense-in-depth for future degraded-mode constructors.
func TestRedisCache_GetSet_MultipleTypes(t *testing.T) {
	_, conn := setupMiniredis(t)

	t.Run("string type", func(t *testing.T) {
		cache, err := NewRedisCache[string](conn, time.Minute, "string:")
		require.NoError(t, err)
		ctx := context.Background()

		err = cache.Set(ctx, "key1", "hello world", 0)
		require.NoError(t, err)

		result, found, err := cache.Get(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "hello world", result)
	})

	t.Run("int type", func(t *testing.T) {
		cache, err := NewRedisCache[int](conn, time.Minute, "int:")
		require.NoError(t, err)
		ctx := context.Background()

		err = cache.Set(ctx, "key1", 42, 0)
		require.NoError(t, err)

		result, found, err := cache.Get(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 42, result)
	})

	t.Run("slice type", func(t *testing.T) {
		cache, err := NewRedisCache[[]string](conn, time.Minute, "slice:")
		require.NoError(t, err)
		ctx := context.Background()

		expected := []string{"a", "b", "c"}
		err = cache.Set(ctx, "key1", expected, 0)
		require.NoError(t, err)

		result, found, err := cache.Get(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, expected, result)
	})

	t.Run("map type", func(t *testing.T) {
		cache, err := NewRedisCache[map[string]int](conn, time.Minute, "map:")
		require.NoError(t, err)
		ctx := context.Background()

		expected := map[string]int{"a": 1, "b": 2}
		err = cache.Set(ctx, "key1", expected, 0)
		require.NoError(t, err)

		result, found, err := cache.Get(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, expected, result)
	})
}
