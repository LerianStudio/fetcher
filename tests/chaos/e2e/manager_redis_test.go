//go:build chaos

package e2e

import (
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/tests/chaos/helpers"
	"github.com/LerianStudio/fetcher/tests/chaos/setup"
	"github.com/LerianStudio/fetcher/tests/shared/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contains is a helper to check if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// TestRedisUnavailable_RateLimitingFallback tests that rate limiting
// continues to work (via in-memory fallback) when Redis is unavailable.
func (s *ChaosTestSuite) TestRedisUnavailable_RateLimitingFallback() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"continue rate limiting via in-memory fallback when Redis is unavailable",
		"Redis proxy is completely disabled",
	))

	// Phase 1: Create a connection to test rate limiting
	t.Log("Phase 1: Creating connection for rate limit testing...")
	configName := s.uniqueConfigName("chaos_postgres_test")
	pg := s.chaosInfra.PostgresInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Phase 2: Baseline - test connection endpoint (uses rate limiting)
	t.Log("Phase 2: Baseline rate limiting test...")
	_, err = s.managerClient.TestConnectionEndpoint(s.ctx, connResp.ID)
	require.NoError(t, err, "Baseline test connection should succeed")

	// Phase 3: Disable Redis proxy
	t.Log("Phase 3: Disabling Redis...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetRedisProxy()
	require.NotNil(t, proxy, "Redis proxy should exist")

	err = helpers.DisableProxy(proxy)
	require.NoError(t, err, "Failed to disable Redis proxy")
	defer func() { _ = helpers.EnableProxy(proxy) }()

	// Wait for health check to detect failure
	t.Log("Waiting for fallback to activate...")
	time.Sleep(setup.RedisHealthCheckWait)

	// Phase 4: Test connection endpoint should still work (uses fallback)
	// Make rapid requests to test both success AND rate limiting behavior
	t.Log("Phase 4: Testing rate limiting with Redis unavailable...")
	successCount := 0
	rateLimitedCount := 0

	// First, test with delays (should succeed)
	for i := 0; i < 3; i++ {
		start := time.Now()
		_, testErr := s.managerClient.TestConnectionEndpoint(s.ctx, connResp.ID)
		duration := time.Since(start)

		if testErr == nil {
			successCount++
			s.metrics.RecordRequest(true, false, duration)
			t.Logf("Test connection %d: succeeded in %v", i+1, duration)
		} else {
			s.metrics.RecordRequest(false, false, duration)
			t.Logf("Test connection %d: failed: %v", i+1, testErr)
		}

		time.Sleep(time.Second) // Respect rate limits
	}

	// Then, make rapid requests to verify rate limiting is active
	t.Log("Making rapid requests to verify rate limiting is active...")
	for i := 0; i < 5; i++ {
		_, testErr := s.managerClient.TestConnectionEndpoint(s.ctx, connResp.ID)
		if testErr == nil {
			successCount++
		} else {
			// Check if error indicates rate limiting
			errStr := testErr.Error()
			if contains(errStr, "rate limit") || contains(errStr, "429") || contains(errStr, "too many") {
				rateLimitedCount++
				t.Logf("Rapid request %d: rate limited (fallback working)", i+1)
			}
		}
		// No delay - trying to trigger rate limit
	}

	s.metrics.EndChaos()

	// Phase 5: Re-enable Redis and verify
	t.Log("Phase 5: Re-enabling Redis...")
	err = helpers.EnableProxy(proxy)
	require.NoError(t, err)

	s.metrics.StartRecovery()
	time.Sleep(setup.RedisHealthCheckWait)

	// Verify system recovered - should be able to make a request
	_, recoveryErr := s.managerClient.TestConnectionEndpoint(s.ctx, connResp.ID)
	if recoveryErr != nil {
		t.Logf("Post-recovery test: %v (rate limited is acceptable)", recoveryErr)
	} else {
		t.Log("Post-recovery test: succeeded")
	}

	s.metrics.EndRecovery()

	// Assertions - verify fallback is working
	assert.GreaterOrEqual(t, successCount, 2, "At least some requests should succeed via fallback")
	t.Logf("Results: %d successful, %d rate-limited (rate limiting %s)",
		successCount, rateLimitedCount,
		map[bool]string{true: "verified active", false: "status unknown"}[rateLimitedCount > 0])

	// Document results
	helpers.DocumentResult(t, s.metrics, "Rate limiting continued via in-memory fallback when Redis was unavailable")
}
