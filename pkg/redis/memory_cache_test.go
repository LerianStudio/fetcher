package redis

import (
	"context"
	"testing"
	"time"

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
