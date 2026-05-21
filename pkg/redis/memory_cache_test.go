package redis

import (
	"context"
	"testing"
	"time"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryCache_GetSet(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

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

func TestInMemoryCache_Delete(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1"}

	// Set value
	err := cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Delete
	err = cache.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify deleted
	_, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestInMemoryCache_Clear(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 5; i++ {
		key := "item" + string(rune('0'+i))
		err := cache.Set(ctx, key, testStruct{ID: key}, 0)
		require.NoError(t, err)
	}

	// Clear
	err := cache.Clear(ctx)
	assert.NoError(t, err)

	// Verify all cleared
	for i := 0; i < 5; i++ {
		key := "item" + string(rune('0'+i))
		_, found, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.False(t, found)
	}
}

func TestInMemoryCache_TTLExpiration(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1"}

	// Set with very short TTL
	err := cache.Set(ctx, key, value, 50*time.Millisecond)
	require.NoError(t, err)

	// Verify exists immediately
	_, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.True(t, found)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Verify expired
	_, found, err = cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestInMemoryCache_IsHealthy(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	// Always healthy
	assert.True(t, cache.IsHealthy(context.Background()))
}

func TestInMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "item" + string(rune('0'+id%10))
			_ = cache.Set(ctx, key, testStruct{ID: key, Value: id}, 0)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "item" + string(rune('0'+id%10))
			_, _, _ = cache.Get(ctx, key)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestInMemoryCache_Close_Idempotent(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})

	// First close should succeed
	err := cache.Close()
	assert.NoError(t, err)

	// Second close should also succeed (idempotent)
	err = cache.Close()
	assert.NoError(t, err)

	// Third close should also succeed
	err = cache.Close()
	assert.NoError(t, err)
}

func TestInMemoryCache_DefaultTTL(t *testing.T) {
	// Test that zero TTL uses default
	cache := NewInMemoryCache[testStruct](0, &mockLogger{})
	defer cache.Close()

	assert.Equal(t, DefaultCacheTTL, cache.ttl)
}

func TestInMemoryCache_NegativeTTL(t *testing.T) {
	// Test that negative TTL uses default
	cache := NewInMemoryCache[testStruct](-5*time.Minute, &mockLogger{})
	defer cache.Close()

	assert.Equal(t, DefaultCacheTTL, cache.ttl)
}

func TestInMemoryCache_Set_UsesDefaultTTL(t *testing.T) {
	cache := NewInMemoryCache[testStruct](10*time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()

	// Set with 0 TTL should use cache's default
	err := cache.Set(ctx, "key1", testStruct{ID: "1"}, 0)
	require.NoError(t, err)

	// Get should succeed
	result, found, err := cache.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1", result.ID)
}

func TestInMemoryCache_Set_NegativeTTL_UsesDefault(t *testing.T) {
	cache := NewInMemoryCache[testStruct](10*time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()

	// Set with negative TTL should use cache's default
	err := cache.Set(ctx, "key1", testStruct{ID: "1"}, -5*time.Minute)
	require.NoError(t, err)

	// Get should succeed
	result, found, err := cache.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1", result.ID)
}

func TestInMemoryCache_Get_Expired(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()

	// Manually insert an expired entry
	cache.mu.Lock()
	cache.entries["expired_key"] = &cacheEntry[testStruct]{
		value:     testStruct{ID: "expired"},
		expiresAt: time.Now().UTC().Add(-1 * time.Hour), // Already expired
	}
	cache.mu.Unlock()

	// Get should report not found for expired entry
	result, found, err := cache.Get(ctx, "expired_key")
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, testStruct{}, result)
}

func TestInMemoryCache_MultipleTypes(t *testing.T) {
	t.Run("string type", func(t *testing.T) {
		cache := NewInMemoryCache[string](time.Minute, &mockLogger{})
		defer cache.Close()

		ctx := context.Background()

		err := cache.Set(ctx, "key1", "hello world", 0)
		require.NoError(t, err)

		result, found, err := cache.Get(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "hello world", result)
	})

	t.Run("int type", func(t *testing.T) {
		cache := NewInMemoryCache[int](time.Minute, &mockLogger{})
		defer cache.Close()

		ctx := context.Background()

		err := cache.Set(ctx, "key1", 42, 0)
		require.NoError(t, err)

		result, found, err := cache.Get(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 42, result)
	})

	t.Run("slice type", func(t *testing.T) {
		cache := NewInMemoryCache[[]string](time.Minute, &mockLogger{})
		defer cache.Close()

		ctx := context.Background()

		expected := []string{"a", "b", "c"}
		err := cache.Set(ctx, "key1", expected, 0)
		require.NoError(t, err)

		result, found, err := cache.Get(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, expected, result)
	})
}

func TestInMemoryCache_DeleteNonExistentKey(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()

	// Delete non-existent key should not error
	err := cache.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestInMemoryCache_GetSet_WithTenantContext(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	t.Run("tenant-scoped keys are isolated", func(t *testing.T) {
		tenant1Ctx := tmcore.ContextWithTenantID(context.Background(), "tenant1")
		tenant2Ctx := tmcore.ContextWithTenantID(context.Background(), "tenant2")
		key := "shared-schema-key"

		value1 := testStruct{ID: "1", Name: "Tenant1"}
		value2 := testStruct{ID: "2", Name: "Tenant2"}

		err := cache.Set(tenant1Ctx, key, value1, 0)
		require.NoError(t, err)

		err = cache.Set(tenant2Ctx, key, value2, 0)
		require.NoError(t, err)

		result1, found1, err := cache.Get(tenant1Ctx, key)
		assert.NoError(t, err)
		assert.True(t, found1)
		assert.Equal(t, value1, result1)

		result2, found2, err := cache.Get(tenant2Ctx, key)
		assert.NoError(t, err)
		assert.True(t, found2)
		assert.Equal(t, value2, result2)
	})

	t.Run("no tenant context returns original key behavior", func(t *testing.T) {
		ctx := context.Background()
		key := "no-tenant-key"
		value := testStruct{ID: "3", Name: "NoTenant"}

		err := cache.Set(ctx, key, value, 0)
		require.NoError(t, err)

		result, found, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, value, result)
	})

	t.Run("delete only affects the tenant that owns the key", func(t *testing.T) {
		tenant1Ctx := tmcore.ContextWithTenantID(context.Background(), "tenant_del_1")
		tenant2Ctx := tmcore.ContextWithTenantID(context.Background(), "tenant_del_2")
		key := "delete-test-key"

		value1 := testStruct{ID: "d1", Name: "DelTenant1"}
		value2 := testStruct{ID: "d2", Name: "DelTenant2"}

		err := cache.Set(tenant1Ctx, key, value1, 0)
		require.NoError(t, err)

		err = cache.Set(tenant2Ctx, key, value2, 0)
		require.NoError(t, err)

		// Delete tenant1's key
		err = cache.Delete(tenant1Ctx, key)
		require.NoError(t, err)

		// Tenant1's key should be gone
		_, found, err := cache.Get(tenant1Ctx, key)
		assert.NoError(t, err)
		assert.False(t, found)

		// Tenant2's key should still exist
		result, found, err := cache.Get(tenant2Ctx, key)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, value2, result)
	})

	t.Run("clear only affects the tenant in context", func(t *testing.T) {
		tenant1Ctx := tmcore.ContextWithTenantID(context.Background(), "tenant_clear_1")
		tenant2Ctx := tmcore.ContextWithTenantID(context.Background(), "tenant_clear_2")
		key := "clear-test-key"

		value1 := testStruct{ID: "c1", Name: "ClearTenant1"}
		value2 := testStruct{ID: "c2", Name: "ClearTenant2"}

		err := cache.Set(tenant1Ctx, key, value1, 0)
		require.NoError(t, err)

		err = cache.Set(tenant2Ctx, key, value2, 0)
		require.NoError(t, err)

		err = cache.Clear(tenant1Ctx)
		require.NoError(t, err)

		_, found, err := cache.Get(tenant1Ctx, key)
		assert.NoError(t, err)
		assert.False(t, found)

		result, found, err := cache.Get(tenant2Ctx, key)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, value2, result)
	})
}

func TestInMemoryCache_ClearEmpty(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()

	// Clear empty cache should not error
	err := cache.Clear(ctx)
	assert.NoError(t, err)
}

func TestInMemoryCache_UpdateExistingKey(t *testing.T) {
	cache := NewInMemoryCache[testStruct](time.Minute, &mockLogger{})
	defer cache.Close()

	ctx := context.Background()
	key := "item1"

	// Set initial value
	err := cache.Set(ctx, key, testStruct{ID: "1", Name: "Original"}, 0)
	require.NoError(t, err)

	// Update with new value
	err = cache.Set(ctx, key, testStruct{ID: "1", Name: "Updated"}, 0)
	require.NoError(t, err)

	// Get should return updated value
	result, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Updated", result.Name)
}
