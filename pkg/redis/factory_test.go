package redis

import (
	"testing"
	"time"

	"github.com/LerianStudio/lib-commons/v2/commons/zap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheWithFallback_RedisUnavailable_ReturnsMemoryOnlyCache(t *testing.T) {
	logger := zap.InitializeLogger()

	// Use invalid Redis config that will fail to connect
	cfg := RedisConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     "6379",
		Password: "",
		DB:       0,
	}

	ttl := 5 * time.Minute

	cache, err := NewCacheWithFallback[string](cfg, logger, ttl, "test:")

	// Should NOT return error - graceful degradation
	assert.NoError(t, err)
	require.NotNil(t, cache)

	// Should be healthy (in-memory is always healthy)
	assert.True(t, cache.IsHealthy(nil))

	// Cleanup
	if closeable, ok := cache.(Closeable); ok {
		closeable.Close()
	}
}

func TestNewCacheWithFallback_ZeroTTL_UsesDefault(t *testing.T) {
	logger := zap.InitializeLogger()

	// Use invalid Redis config that will fail to connect
	cfg := RedisConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     "6379",
		Password: "",
		DB:       0,
	}

	// Zero TTL should use default
	cache, err := NewCacheWithFallback[string](cfg, logger, 0, "test:")

	assert.NoError(t, err)
	require.NotNil(t, cache)

	// Cleanup
	if closeable, ok := cache.(Closeable); ok {
		closeable.Close()
	}
}

func TestNewCacheWithFallback_RedisAvailable_ReturnsFallbackCache(t *testing.T) {
	// Skip if Redis is not available in test environment
	t.Skip("Requires Redis to be running - run manually or in integration tests")

	logger := zap.InitializeLogger()

	cfg := RedisConfig{
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       0,
	}

	ttl := 5 * time.Minute

	cache, err := NewCacheWithFallback[string](cfg, logger, ttl, "test:")

	assert.NoError(t, err)
	require.NotNil(t, cache)

	// Cleanup
	if closeable, ok := cache.(Closeable); ok {
		closeable.Close()
	}
}
