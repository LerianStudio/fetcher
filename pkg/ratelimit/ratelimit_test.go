package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	rl := ratelimit.New(10, time.Minute)
	require.NotNil(t, rl)
}

func TestRateLimiter_Allow_UnderLimit(t *testing.T) {
	rl := ratelimit.New(10, time.Minute)

	key := "test-key"
	allowed, retryAfter := rl.Allow(key)

	assert.True(t, allowed)
	assert.Equal(t, time.Duration(0), retryAfter)
}

func TestRateLimiter_Allow_ExceedsLimit(t *testing.T) {
	// 2 tokens per minute = 1 token every 30 seconds
	rl := ratelimit.New(2, time.Minute)

	key := "test-key"

	// First two requests should succeed (burst)
	allowed1, _ := rl.Allow(key)
	allowed2, _ := rl.Allow(key)

	// Third request should be rate limited
	allowed3, retryAfter := rl.Allow(key)

	assert.True(t, allowed1)
	assert.True(t, allowed2)
	assert.False(t, allowed3)
	assert.Greater(t, retryAfter, time.Duration(0))
}

func TestRateLimiter_Allow_DifferentKeys(t *testing.T) {
	rl := ratelimit.New(1, time.Minute)

	// First key
	allowed1, _ := rl.Allow("key1")
	assert.True(t, allowed1)

	// Second key should have its own limit
	allowed2, _ := rl.Allow("key2")
	assert.True(t, allowed2)

	// First key again should be rate limited
	allowed3, retryAfter := rl.Allow("key1")
	assert.False(t, allowed3)
	assert.Greater(t, retryAfter, time.Duration(0))
}

func TestRateLimiter_Take_UnderLimit(t *testing.T) {
	rl := ratelimit.New(10, time.Minute)

	key := "test-key"
	tokens, remaining, reset, ok, err := rl.Take(context.Background(), key)

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(10), tokens)
	assert.Equal(t, uint64(9), remaining)
	assert.Greater(t, reset, uint64(0))
}

func TestRateLimiter_Take_ExceedsLimit(t *testing.T) {
	rl := ratelimit.New(2, time.Minute)

	key := "test-key"

	// Exhaust tokens
	_, _, _, _, _ = rl.Take(context.Background(), key)
	_, _, _, _, _ = rl.Take(context.Background(), key)

	// Third request should be rate limited
	_, _, reset, ok, err := rl.Take(context.Background(), key)

	require.NoError(t, err)
	assert.False(t, ok)
	assert.Greater(t, reset, uint64(0))
}

func TestRateLimiter_Take_DifferentKeys(t *testing.T) {
	rl := ratelimit.New(1, time.Minute)

	// First key - should succeed
	_, _, _, ok1, err1 := rl.Take(context.Background(), "key1")
	require.NoError(t, err1)
	assert.True(t, ok1)

	// Second key - should succeed (independent limit)
	_, _, _, ok2, err2 := rl.Take(context.Background(), "key2")
	require.NoError(t, err2)
	assert.True(t, ok2)

	// First key again - should be rate limited
	_, _, _, ok3, err3 := rl.Take(context.Background(), "key1")
	require.NoError(t, err3)
	assert.False(t, ok3)
}

func TestRateLimiter_Take_ReturnsCorrectTokens(t *testing.T) {
	rl := ratelimit.New(5, time.Minute)

	key := "test-key"
	tokens, _, _, ok, err := rl.Take(context.Background(), key)

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(5), tokens)
}

func TestRateLimiter_Tokens(t *testing.T) {
	rl := ratelimit.New(15, time.Minute)
	assert.Equal(t, 15, rl.Tokens())
}

func TestRateLimiter_Interval(t *testing.T) {
	rl := ratelimit.New(10, 2*time.Minute)
	assert.Equal(t, 2*time.Minute, rl.Interval())
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := ratelimit.New(10, time.Minute)

	// Add some limiters
	rl.Allow("key1")
	rl.Allow("key2")
	rl.Allow("key3")

	// Cleanup should not panic
	rl.Cleanup()

	// After cleanup, new requests should work
	allowed, _ := rl.Allow("key1")
	assert.True(t, allowed)
}

func TestRateLimiter_CleanupInactive(t *testing.T) {
	rl := ratelimit.New(10, 100*time.Millisecond)

	// Add a limiter
	rl.Allow("key1")

	// Wait for it to become inactive
	time.Sleep(200 * time.Millisecond)

	// Cleanup inactive entries
	cleaned := rl.CleanupInactive(100 * time.Millisecond)

	assert.Equal(t, 1, cleaned)
}

func TestRateLimiter_CleanupInactive_ActiveEntryNotCleaned(t *testing.T) {
	rl := ratelimit.New(10, time.Minute)

	// Add a limiter
	rl.Allow("key1")

	// Immediately cleanup with 1 minute threshold
	cleaned := rl.CleanupInactive(time.Minute)

	// Should not clean recently accessed entry
	assert.Equal(t, 0, cleaned)
}

func TestRateLimiter_CleanupInactive_MultipleEntries(t *testing.T) {
	rl := ratelimit.New(10, 100*time.Millisecond)

	// Add limiters
	rl.Allow("key1")
	rl.Allow("key2")

	// Wait for them to become inactive
	time.Sleep(200 * time.Millisecond)

	// Add a new limiter (active)
	rl.Allow("key3")

	// Cleanup inactive entries
	cleaned := rl.CleanupInactive(100 * time.Millisecond)

	// key1 and key2 should be cleaned, key3 should remain
	assert.Equal(t, 2, cleaned)

	// key3 should still work
	allowed, _ := rl.Allow("key3")
	assert.True(t, allowed)
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := ratelimit.New(100, time.Minute)
	key := "concurrent-key"

	done := make(chan bool)

	// Run multiple goroutines accessing the same key
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				rl.Allow(key)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should work correctly after concurrent access
	allowed, _ := rl.Allow(key)
	// May or may not be allowed depending on timing, but should not panic
	_ = allowed
}

func TestRateLimiter_ZeroTokens(t *testing.T) {
	// Edge case: zero tokens should deny all requests
	rl := ratelimit.New(0, time.Minute)

	allowed, _ := rl.Allow("key")
	assert.False(t, allowed)
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	rl := ratelimit.New(10, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Take should still work even with cancelled context (no blocking operations)
	_, _, _, _, err := rl.Take(ctx, "key")
	require.NoError(t, err)
}
