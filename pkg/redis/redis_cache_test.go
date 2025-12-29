package redis

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStruct is a simple struct for testing generic cache
type testStruct struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// mockLogger implements log.Logger for testing.
//
// NOTE: This manual mock is intentionally retained because log.Logger is an external
// interface from github.com/LerianStudio/lib-commons/v2/commons/log. Generating mockgen
// mocks for external interfaces requires either:
// 1. A local wrapper interface (adds unnecessary indirection)
// 2. Reflect mode with full package path (fragile to library changes)
// For simple logging interfaces used only in tests, a manual mock is more maintainable.
type mockLogger struct{}

func (m *mockLogger) Info(args ...any)                                     {}
func (m *mockLogger) Infof(format string, args ...any)                     {}
func (m *mockLogger) Infoln(args ...any)                                   {}
func (m *mockLogger) Error(args ...any)                                    {}
func (m *mockLogger) Errorf(format string, args ...any)                    {}
func (m *mockLogger) Errorln(args ...any)                                  {}
func (m *mockLogger) Warn(args ...any)                                     {}
func (m *mockLogger) Warnf(format string, args ...any)                     {}
func (m *mockLogger) Warnln(args ...any)                                   {}
func (m *mockLogger) Debug(args ...any)                                    {}
func (m *mockLogger) Debugf(format string, args ...any)                    {}
func (m *mockLogger) Debugln(args ...any)                                  {}
func (m *mockLogger) Fatal(args ...any)                                    {}
func (m *mockLogger) Fatalf(format string, args ...any)                    {}
func (m *mockLogger) Fatalln(args ...any)                                  {}
func (m *mockLogger) WithFields(fields ...any) log.Logger                  { return m }
func (m *mockLogger) WithDefaultMessageTemplate(message string) log.Logger { return m }
func (m *mockLogger) Sync() error                                          { return nil }

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *RedisConnection) {
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

	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return mr, conn
}

func TestRedisCache_GetSet(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache := NewRedisCache[testStruct](conn, time.Minute, "test:")

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
	cache := NewRedisCache[testStruct](conn, time.Minute, "test:")

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test", Value: 42}

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

func TestRedisCache_Clear(t *testing.T) {
	_, conn := setupMiniredis(t)
	cache := NewRedisCache[testStruct](conn, time.Minute, "test:")

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

func TestRedisCache_TTLExpiration(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache := NewRedisCache[testStruct](conn, time.Minute, "test:")

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test", Value: 42}

	// Set with short TTL
	err := cache.Set(ctx, key, value, 100*time.Millisecond)
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
	cache := NewRedisCache[testStruct](conn, time.Minute, "test:")

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
	cache := NewRedisCache[testStruct](conn, time.Minute, "myprefix:")

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1"}

	err := cache.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Verify key has prefix in Redis
	keys := mr.Keys()
	assert.Contains(t, keys, "myprefix:item1")
}

func TestRedisCache_Get_UnmarshalError(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache := NewRedisCache[testStruct](conn, time.Minute, "test:")

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
	cache := NewRedisCache[testStruct](conn, time.Minute, "test:")

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
	cache := NewRedisCache[testStruct](conn, time.Minute, "test:")

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1", Name: "Test", Value: 42}

	// Close miniredis to cause connection error
	mr.Close()

	// Set should return error
	err := cache.Set(ctx, key, value, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store in cache")
}

func TestRedisCache_Delete_RedisError(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache := NewRedisCache[testStruct](conn, time.Minute, "test:")

	ctx := context.Background()
	key := "item1"

	// Close miniredis to cause connection error
	mr.Close()

	// Delete should return error
	err := cache.Delete(ctx, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete from cache")
}

func TestRedisCache_DefaultTTL(t *testing.T) {
	_, conn := setupMiniredis(t)

	// Test with zero TTL - should use default
	cache := NewRedisCache[testStruct](conn, 0, "test:")
	assert.Equal(t, DefaultCacheTTL, cache.ttl)

	// Test with negative TTL - should use default
	cache2 := NewRedisCache[testStruct](conn, -5*time.Minute, "test:")
	assert.Equal(t, DefaultCacheTTL, cache2.ttl)
}

func TestRedisCache_Set_UsesDefaultTTL(t *testing.T) {
	mr, conn := setupMiniredis(t)
	cache := NewRedisCache[testStruct](conn, 10*time.Minute, "test:")

	ctx := context.Background()
	key := "item1"
	value := testStruct{ID: "1"}

	// Set with 0 TTL should use cache's default
	err := cache.Set(ctx, key, value, 0)
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
