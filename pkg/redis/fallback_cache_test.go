package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFallbackCache(t *testing.T) (*miniredis.Miniredis, *FallbackCache[testStruct]) {
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
		Logger:    &mockLogger{},
		Connected: true,
	}

	redisCache := NewRedisCache[testStruct](conn, time.Minute, "test:")
	fallback := NewFallbackCache[testStruct](redisCache, &mockLogger{}, time.Minute)

	t.Cleanup(func() {
		fallback.Close()
		client.Close()
		mr.Close()
	})

	return mr, fallback
}

func TestFallbackCache_GetSet(t *testing.T) {
	_, cache := setupFallbackCache(t)

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

func TestFallbackCache_FallsBackToInMemory(t *testing.T) {
	mr, cache := setupFallbackCache(t)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test", Value: 42}

	// Set value while Redis is available
	err := cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Close Redis
	mr.Close()

	// Should still get value from in-memory fallback
	result, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, result)
}

func TestFallbackCache_Delete(t *testing.T) {
	_, cache := setupFallbackCache(t)

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

func TestFallbackCache_Clear(t *testing.T) {
	_, cache := setupFallbackCache(t)

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

func TestFallbackCache_Close_Idempotent(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	conn := &RedisConnection{
		Client:    client,
		Logger:    &mockLogger{},
		Connected: true,
	}

	redisCache := NewRedisCache[testStruct](conn, time.Minute, "test:")
	fallback := NewFallbackCache[testStruct](redisCache, &mockLogger{}, time.Minute)

	// First close should succeed
	err = fallback.Close()
	assert.NoError(t, err)

	// Second close should also succeed (idempotent)
	err = fallback.Close()
	assert.NoError(t, err)

	// Third close should also succeed
	err = fallback.Close()
	assert.NoError(t, err)
}

func TestFallbackCache_IsHealthy(t *testing.T) {
	_, cache := setupFallbackCache(t)

	// IsHealthy should return true (at least in-memory is always available)
	assert.True(t, cache.IsHealthy(context.Background()))
}

func TestFallbackCache_IsHealthy_RedisDown(t *testing.T) {
	mr, cache := setupFallbackCache(t)

	// IsHealthy should return true when Redis is up
	assert.True(t, cache.IsHealthy(context.Background()))

	// Close Redis
	mr.Close()

	// IsHealthy should still return true (in-memory fallback is available)
	assert.True(t, cache.IsHealthy(context.Background()))
}

func TestFallbackCache_DefaultTTL(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	conn := &RedisConnection{
		Client:    client,
		Logger:    &mockLogger{},
		Connected: true,
	}

	redisCache := NewRedisCache[testStruct](conn, time.Minute, "test:")

	// Test that zero TTL uses default
	fallback := NewFallbackCache[testStruct](redisCache, &mockLogger{}, 0)
	defer fallback.Close()

	// Verify cache works (default TTL is applied internally)
	ctx := context.Background()
	err = fallback.Set(ctx, "key1", testStruct{ID: "1"}, 0)
	assert.NoError(t, err)

	result, found, err := fallback.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1", result.ID)
}

func TestFallbackCache_NegativeTTL(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	conn := &RedisConnection{
		Client:    client,
		Logger:    &mockLogger{},
		Connected: true,
	}

	redisCache := NewRedisCache[testStruct](conn, time.Minute, "test:")

	// Test that negative TTL uses default
	fallback := NewFallbackCache[testStruct](redisCache, &mockLogger{}, -5*time.Minute)
	defer fallback.Close()

	// Should not panic and cache should work
	ctx := context.Background()
	err = fallback.Set(ctx, "key1", testStruct{ID: "1"}, 0)
	assert.NoError(t, err)
}

func TestFallbackCache_Get_RedisError_FallsBackToMemory(t *testing.T) {
	mr, cache := setupFallbackCache(t)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test"}

	// Set value (goes to both Redis and memory)
	err := cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Close Redis to simulate connection failure
	mr.Close()

	// Get should still succeed by falling back to in-memory cache
	result, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, result)
}

func TestFallbackCache_Set_RedisError_StillSavesToMemory(t *testing.T) {
	mr, cache := setupFallbackCache(t)

	// Close Redis first
	mr.Close()

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test"}

	// Set should still succeed (stores in memory)
	err := cache.Set(ctx, key, value, 0)
	assert.NoError(t, err)

	// Get should succeed from memory
	result, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, result)
}

func TestFallbackCache_Delete_RedisError(t *testing.T) {
	mr, cache := setupFallbackCache(t)

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1"}

	// Set value
	err := cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Close Redis
	mr.Close()

	// Delete should still succeed (deletes from memory, logs Redis error)
	err = cache.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify deleted from memory
	_, found, err := cache.Get(ctx, key)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestFallbackCache_Clear_RedisError(t *testing.T) {
	mr, cache := setupFallbackCache(t)

	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 3; i++ {
		key := "item" + string(rune('0'+i))
		err := cache.Set(ctx, key, testStruct{ID: key}, 0)
		require.NoError(t, err)
	}

	// Close Redis
	mr.Close()

	// Clear should still succeed (clears memory, logs Redis error)
	err := cache.Clear(ctx)
	assert.NoError(t, err)

	// Verify cleared from memory
	for i := 0; i < 3; i++ {
		key := "item" + string(rune('0'+i))
		_, found, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.False(t, found)
	}
}

func TestFallbackCache_UseRedisFlag_Switch(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	conn := &RedisConnection{
		Client:    client,
		Logger:    &mockLogger{},
		Connected: true,
	}

	redisCache := NewRedisCache[testStruct](conn, time.Minute, "test:")
	cache := NewFallbackCache[testStruct](redisCache, &mockLogger{}, time.Minute)
	defer cache.Close()

	ctx := context.Background()

	// Initially useRedis is true
	cache.mu.RLock()
	assert.True(t, cache.useRedis)
	cache.mu.RUnlock()

	// Set value with Redis available
	err = cache.Set(ctx, "key1", testStruct{ID: "1"}, 0)
	require.NoError(t, err)

	// Manually set useRedis to false to simulate Redis being unavailable
	cache.mu.Lock()
	cache.useRedis = false
	cache.mu.Unlock()

	// Set should still work (goes to memory only)
	err = cache.Set(ctx, "key2", testStruct{ID: "2"}, 0)
	assert.NoError(t, err)

	// Get should work from memory
	result, found, err := cache.Get(ctx, "key2")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "2", result.ID)

	// Delete should work
	err = cache.Delete(ctx, "key2")
	assert.NoError(t, err)

	// Clear should work
	err = cache.Clear(ctx)
	assert.NoError(t, err)
}

func TestFallbackCache_Get_MissFromBothCaches(t *testing.T) {
	_, cache := setupFallbackCache(t)

	ctx := context.Background()

	// Get non-existent key should return not found
	result, found, err := cache.Get(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, testStruct{}, result)
}

func TestFallbackCache_ConcurrentAccess(t *testing.T) {
	_, cache := setupFallbackCache(t)

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

func TestFallbackCache_Close_StopsHealthMonitor(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})

	conn := &RedisConnection{
		Client:    client,
		Logger:    &mockLogger{},
		Connected: true,
	}

	redisCache := NewRedisCache[testStruct](conn, time.Minute, "test:")
	cache := NewFallbackCache[testStruct](redisCache, &mockLogger{}, time.Minute)

	// Close should stop the health monitor goroutine
	err = cache.Close()
	assert.NoError(t, err)

	// Verify closed flag is set
	cache.mu.RLock()
	assert.True(t, cache.closed)
	cache.mu.RUnlock()
}
