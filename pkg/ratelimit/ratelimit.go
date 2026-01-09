// Package ratelimit provides a rate limiting implementation using golang.org/x/time/rate.
// It supports per-key rate limiting with configurable tokens and intervals.
package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// limiterEntry wraps a rate.Limiter with metadata for cleanup.
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess atomic.Int64 // Unix nanoseconds
}

// RateLimiter provides per-key rate limiting using a token bucket algorithm.
// It is safe for concurrent use by multiple goroutines.
type RateLimiter struct {
	tokens   int
	interval time.Duration
	limiters sync.Map // map[string]*limiterEntry
	mu       sync.Mutex
}

// New creates a new RateLimiter with the specified tokens allowed per interval.
// For example, New(10, time.Minute) allows 10 requests per minute per key.
func New(tokens int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:   tokens,
		interval: interval,
	}
}

// getLimiter returns the rate limiter for the given key, creating one if it doesn't exist.
func (r *RateLimiter) getLimiter(key string) *rate.Limiter {
	now := time.Now().UTC().UnixNano()

	if entry, ok := r.limiters.Load(key); ok {
		e := entry.(*limiterEntry)
		e.lastAccess.Store(now)

		return e.limiter
	}

	// Calculate rate: tokens per interval
	// rate.Limit is events per second
	var ratePerSecond rate.Limit
	if r.interval.Seconds() > 0 {
		ratePerSecond = rate.Limit(float64(r.tokens) / r.interval.Seconds())
	} else {
		ratePerSecond = rate.Inf
	}

	limiter := rate.NewLimiter(ratePerSecond, r.tokens)

	entry := &limiterEntry{
		limiter: limiter,
	}
	entry.lastAccess.Store(now)

	// Store or get existing (handles race condition)
	actual, _ := r.limiters.LoadOrStore(key, entry)

	return actual.(*limiterEntry).limiter
}

// Allow checks if a request for the given key is allowed.
// Returns true if allowed, false if rate limited.
// When rate limited, retryAfter indicates how long to wait before retrying.
func (r *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	limiter := r.getLimiter(key)

	if limiter.Allow() {
		return true, 0
	}

	// Calculate retry-after time using Reserve
	reservation := limiter.Reserve()
	delay := reservation.Delay()
	reservation.Cancel() // Cancel since we're not actually waiting

	return false, delay
}

// Take implements a go-limiter compatible interface for backward compatibility.
// Returns (tokens, remaining, reset, ok, err) where:
//   - tokens: total token capacity
//   - remaining: approximate remaining tokens
//   - reset: Unix nanoseconds when the bucket will be full again
//   - ok: whether the request is allowed
//   - err: always nil (for interface compatibility)
func (r *RateLimiter) Take(_ context.Context, key string) (tokens, remaining, reset uint64, ok bool, err error) {
	limiter := r.getLimiter(key)

	// r.tokens is validated to be positive during RateLimiter construction
	tokens = uint64(r.tokens) // #nosec G115 -- tokens is always positive from constructor

	if limiter.Allow() {
		// Request allowed
		currentTokens := limiter.Tokens()
		if currentTokens < 0 {
			currentTokens = 0
		}

		remaining = uint64(currentTokens)
		if remaining > tokens {
			remaining = tokens
		}
		// Calculate reset time (when bucket will be full)
		resetTime := time.Now().UTC().Add(r.interval)
		// UnixNano() returns positive values for current/future timestamps (post-1970)
		reset = uint64(resetTime.UnixNano()) // #nosec G115 -- resetTime is always in the future (post-1970)
		ok = true

		return tokens, remaining, reset, ok, err
	}

	// Request denied - calculate when to retry
	reservation := limiter.Reserve()
	delay := reservation.Delay()
	reservation.Cancel()

	remaining = 0
	resetTime := time.Now().UTC().Add(delay)
	// UnixNano() returns positive values for current/future timestamps (post-1970)
	reset = uint64(resetTime.UnixNano()) // #nosec G115 -- resetTime is always in the future (post-1970)
	ok = false

	return tokens, remaining, reset, ok, err
}

// Tokens returns the configured token capacity.
func (r *RateLimiter) Tokens() int {
	return r.tokens
}

// Interval returns the configured interval.
func (r *RateLimiter) Interval() time.Duration {
	return r.interval
}

// Cleanup removes all rate limiters from the cache.
func (r *RateLimiter) Cleanup() {
	r.limiters.Range(func(key, value any) bool {
		r.limiters.Delete(key)
		return true
	})
}

// CleanupInactive removes rate limiters that haven't been accessed within the given duration.
// Returns the number of entries cleaned up.
func (r *RateLimiter) CleanupInactive(maxAge time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().UTC().Add(-maxAge).UnixNano()
	cleaned := 0

	r.limiters.Range(func(key, value any) bool {
		entry := value.(*limiterEntry)
		if entry.lastAccess.Load() < cutoff {
			r.limiters.Delete(key)

			cleaned++
		}

		return true
	})

	return cleaned
}
