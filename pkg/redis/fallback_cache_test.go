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
		Addr: mr.Addr(),
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
		Addr: mr.Addr(),
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
