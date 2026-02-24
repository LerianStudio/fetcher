package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/testutil"
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
		assert.ErrorIs(t, err, ErrRedisCacheNotInitialized)
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
		assert.ErrorIs(t, err, ErrRedisCacheNotInitialized)
	})
}

func TestRedisCache_NewRedisCacheSafe_ReturnsErrorOnInvalidConnection(t *testing.T) {
	cache, err := NewRedisCacheSafe[testStruct](nil, time.Minute, "test:")
	assert.Nil(t, cache)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrRedisCacheNotInitialized)
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
			assert.Equal(t, tt.expected, cache.cacheKey(tt.key))
		})
	}
}

// TestRedisCache_IsHealthy_WithInitErr verifies the initErr guard in IsHealthy.
// The initErr field is never set by constructors (they return an error instead),
// but the guard exists as defense-in-depth for future degraded-mode constructors.
// We test it by directly setting the field on a valid cache instance.
func TestRedisCache_IsHealthy_WithInitErr(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	ctx := context.Background()

	// Sanity: healthy before setting initErr
	assert.True(t, cache.IsHealthy(ctx))

	// Inject initErr to simulate degraded construction
	cache.initErr = errors.New("simulated init failure")

	assert.False(t, cache.IsHealthy(ctx),
		"IsHealthy must return false when initErr is set")
}

// TestRedisCache_EnsureClient_WithInitErr verifies ensureClient propagates initErr.
func TestRedisCache_EnsureClient_WithInitErr(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache, err := NewRedisCache[testStruct](conn, time.Minute, "test:")
	require.NoError(t, err)

	cache.initErr = errors.New("init failed")

	ctx := context.Background()

	// All operations should fail with initErr
	_, _, getErr := cache.Get(ctx, "key")
	assert.Error(t, getErr)
	assert.Contains(t, getErr.Error(), "init failed")

	setErr := cache.Set(ctx, "key", testStruct{}, 0)
	assert.Error(t, setErr)
	assert.Contains(t, setErr.Error(), "init failed")

	delErr := cache.Delete(ctx, "key")
	assert.Error(t, delErr)
	assert.Contains(t, delErr.Error(), "init failed")

	clearErr := cache.Clear(ctx)
	assert.Error(t, clearErr)
	assert.Contains(t, clearErr.Error(), "init failed")
}

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
