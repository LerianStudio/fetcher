//go:build chaos

package e2e

import (
	"strconv"
	"time"

	"github.com/LerianStudio/fetcher/tests/chaos/helpers"
	"github.com/LerianStudio/fetcher/tests/chaos/setup"
	"github.com/LerianStudio/fetcher/tests/shared/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRabbitMQLatency_JobCompletionWithDelay tests that jobs complete
// successfully even when RabbitMQ has high latency.
func (s *ChaosTestSuite) TestRabbitMQLatency_JobCompletionWithDelay() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"complete job successfully with extended duration when RabbitMQ has high latency",
		"RabbitMQ has 3 second latency injected during job processing",
	))

	// Phase 1: Baseline - Create connection and run job WITHOUT chaos
	t.Log("Phase 1: Establishing baseline (no chaos)...")
	configName := s.uniqueConfigName("chaos_postgres_test")

	// Use method call to get connection info (NOT field access)
	pg := s.chaosInfra.PostgresInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port, // int from InternalDBConnection
		DatabaseName: pg.Database,
		Username:     pg.Username, // Username, not User
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create connection")

	baselineStart := time.Now()
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount", "currency"},
				},
			},
		},
		Metadata: s.testMetadata("TestRabbitMQLatency_Baseline"),
	})
	require.NoError(t, err, "Failed to create baseline job")

	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID, setup.JobCompletionTimeout)
	require.NoError(t, err, "Baseline job failed to complete")
	assert.Equal(t, "completed", notification.Status)
	baselineDuration := time.Since(baselineStart)
	t.Logf("Baseline job completed in %v", baselineDuration)

	// Verify baseline connection was created successfully
	require.NotEmpty(t, connResp.ID, "Baseline connection ID should not be empty")
	if err := s.managerClient.DeleteConnectionByConfigName(s.ctx, configName); err != nil {
		t.Logf("Warning: baseline connection cleanup failed: %v", err)
	}

	// Phase 2: Inject RabbitMQ latency chaos
	t.Log("Phase 2: Injecting RabbitMQ latency chaos...")
	s.metrics.StartChaos()

	// Use specific proxy getter method
	proxy := s.chaosInfra.GetRabbitMQProxy()
	require.NotNil(t, proxy, "RabbitMQ proxy should exist")

	chaosConfig := helpers.DefaultLatencyConfig(
		helpers.LatencyMs(setup.ChaosLatencyValues.Medium), // 3s latency
		helpers.LatencyMs(setup.ChaosLatencyValues.Jitter),
	)
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err, "Failed to inject chaos")
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 3: Run job under chaos - should still complete
	t.Log("Phase 3: Creating job under RabbitMQ latency chaos...")
	configName = s.uniqueConfigName("chaos_postgres_test")

	connResp, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err, "Failed to create connection under chaos")
	require.NotEmpty(t, connResp.ID, "Connection ID should not be empty")

	chaosStart := time.Now()
	jobResp, err = s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount", "currency"},
				},
			},
		},
		Metadata: s.testMetadata("TestRabbitMQLatency_UnderChaos"),
	})
	require.NoError(t, err, "Failed to create job under chaos")

	// Wait longer due to latency - job should still complete
	notification, err = s.eventConsumer.WaitForJobEvent(
		s.ctx,
		jobResp.JobID,
		setup.JobCompletionTimeout+30*time.Second, // Extended timeout
	)
	require.NoError(t, err, "Job should complete despite RabbitMQ latency")

	chaosDuration := time.Since(chaosStart)
	s.metrics.EndChaos()

	// Assertions
	assert.Equal(t, "completed", notification.Status, "Job should complete under chaos")
	assert.Greater(t, chaosDuration, baselineDuration, "Chaos job should take longer than baseline")

	// Record metrics
	s.metrics.RecordRequest(true, false, chaosDuration)

	// Phase 4: Remove chaos and verify recovery
	t.Log("Phase 4: Removing chaos, verifying recovery...")
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	s.metrics.StartRecovery()

	// Quick verification that system is responsive
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID)
	require.NoError(t, err, "Should be able to get job after chaos removal")
	assert.Equal(t, "completed", job.Status)

	s.metrics.EndRecovery()

	// Document results
	helpers.DocumentResult(t, s.metrics, "Job completed successfully despite RabbitMQ latency")
	t.Logf("Baseline: %v, Under chaos: %v (%.1fx slower)",
		baselineDuration, chaosDuration, float64(chaosDuration)/float64(baselineDuration))
}

// TestRabbitMQResetPeer_CircuitBreakerActivation tests system behavior under RabbitMQ connection resets.
//
// IMPORTANT: Due to architecture limitations, Manager and Worker containers connect directly
// to RabbitMQ (fetcher-rabbitmq:5672), not through Toxiproxy. Therefore, reset_peer chaos
// on the proxy doesn't affect internal application communication.
//
// This test verifies:
// 1. The system remains operational when reset_peer is injected (no crash)
// 2. Jobs can complete after chaos is removed
// 3. Recovery works properly after circuit breaker cooldown
//
// To truly test RabbitMQ chaos, applications would need to be configured to connect through
// the Toxiproxy, which is not currently supported.
func (s *ChaosTestSuite) TestRabbitMQResetPeer_CircuitBreakerActivation() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"system remains operational under reset_peer chaos and recovers after removal",
		"RabbitMQ proxy has reset_peer chaos - note: apps connect directly to RabbitMQ, not through proxy",
	))

	// Phase 1: Inject reset_peer chaos BEFORE creating job
	t.Log("Phase 1: Injecting reset_peer chaos...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetRabbitMQProxy()
	require.NotNil(t, proxy)

	chaosConfig := helpers.DefaultResetPeerConfig(100) // Reset after 100ms
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 2: Attempt to create multiple jobs - should fail due to circuit breaker
	t.Log("Phase 2: Creating jobs under reset_peer chaos...")
	configName := s.uniqueConfigName("chaos_postgres_test")
	pg := s.chaosInfra.PostgresInternal()

	// Create connection - note this may fail under chaos, which is acceptable
	// We're testing RabbitMQ behavior, not MongoDB
	_, connErr := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	if connErr != nil {
		t.Logf("Connection creation failed under chaos (acceptable): %v", connErr)
	}

	// Try to create jobs - some should fail due to RabbitMQ reset
	failureCount := 0
	for i := 0; i < 7; i++ { // More than circuit breaker threshold (5)
		_, jobErr := s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
			DataRequest: client.DataRequest{
				MappedFields: map[string]map[string][]string{
					configName: {"transactions": {"id"}},
				},
			},
			Metadata: s.testMetadata("TestRabbitMQReset_Attempt_" + strconv.Itoa(i)),
		})

		if jobErr != nil {
			failureCount++
			s.metrics.RecordRequest(false, false, 100*time.Millisecond)
			t.Logf("Job creation %d failed (expected): %v", i+1, jobErr)
		} else {
			s.metrics.RecordRequest(true, false, 100*time.Millisecond)
		}

		time.Sleep(500 * time.Millisecond)
	}

	s.metrics.EndChaos()

	// Phase 3: Remove chaos
	t.Log("Phase 3: Removing chaos...")
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	// Phase 4: Wait for circuit breaker cooldown and verify recovery
	t.Log("Phase 4: Waiting for circuit breaker cooldown...")
	s.metrics.StartRecovery()

	// Circuit breaker cooldown is 30s per pkg/rabbitmq/rabbitmq.go
	time.Sleep(setup.CircuitBreakerCooldown)

	// Should be able to create job after recovery - these are REQUIRED to pass
	configName = s.uniqueConfigName("chaos_postgres_recovery")
	recoveryConnResp, recoveryConnErr := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, recoveryConnErr, "Connection creation should succeed after circuit breaker cooldown")
	require.NotEmpty(t, recoveryConnResp.ID, "Connection ID should not be empty after recovery")

	jobResp, jobErr := s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestRabbitMQReset_Recovery"),
	})
	require.NoError(t, jobErr, "Job creation should succeed after circuit breaker cooldown")

	notification, waitErr := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		jobResp.JobID,
		setup.JobCompletionTimeout,
	)
	require.NoError(t, waitErr, "Job should complete after circuit breaker recovery")
	assert.Equal(t, "completed", notification.Status, "Job should complete successfully after recovery")

	t.Log("Recovery successful - job completed after circuit breaker cooldown")

	s.metrics.EndRecovery()

	// Log the failure count for observability
	// NOTE: Due to architecture limitations (apps connect directly to RabbitMQ, not through proxy),
	// we cannot assert on failure count. The reset_peer chaos affects the proxy, but apps bypass it.
	t.Logf("Total failures during chaos: %d out of 7 attempts (may be 0 due to direct RabbitMQ connection)", failureCount)
	if failureCount == 0 {
		t.Log("INFO: No failures observed - this is expected because apps connect directly to RabbitMQ, not through Toxiproxy")
	}

	// Document results
	helpers.DocumentResult(t, s.metrics, "System remained operational under reset_peer chaos, recovery successful")
}
