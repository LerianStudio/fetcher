//go:build chaos

package e2e

import (
	"time"

	"github.com/LerianStudio/fetcher/tests/chaos/helpers"
	"github.com/LerianStudio/fetcher/tests/chaos/setup"
	"github.com/LerianStudio/fetcher/tests/shared/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullFlow_ChaosMidExtraction tests complete E2E flow with chaos
// injected while extraction is in progress.
func (s *ChaosTestSuite) TestFullFlow_ChaosMidExtraction() {
	t := s.T()

	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"recover and complete job after chaos is removed mid-extraction",
		"PostgreSQL chaos is injected after job starts, then removed",
	))

	// Create connection
	configName := s.uniqueConfigName("chaos_postgres_test")
	pg := s.chaosInfra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Start job
	t.Log("Starting extraction job...")
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id", "amount", "currency", "type"}},
			},
		},
		Metadata: s.testMetadata("TestFullFlow_ChaosMidExtraction"),
	})
	require.NoError(t, err)

	// Wait for job to start processing before injecting chaos
	t.Log("Waiting for job to start processing...")
	err = s.waitForJobProcessing(jobResp.JobID, 10*time.Second)
	require.NoError(t, err, "Job should start processing")

	t.Log("Injecting chaos mid-extraction...")

	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetPostgresProxy()
	require.NotNil(t, proxy)

	chaosConfig := helpers.DefaultLatencyConfig(
		helpers.LatencyMs(setup.ChaosLatencyValues.High),
		helpers.LatencyMs(setup.ChaosLatencyValues.Jitter),
	)
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)

	// Let chaos affect for a bit
	time.Sleep(setup.ChaosInjectionDuration)

	// Remove chaos
	t.Log("Removing chaos...")
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	s.metrics.EndChaos()

	// Wait for job to complete
	t.Log("Waiting for job completion...")
	s.metrics.StartRecovery()

	notification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		jobResp.JobID,
		setup.JobCompletionTimeoutSlow,
	)

	s.metrics.EndRecovery()

	// Job should complete successfully after chaos is removed
	require.NoError(t, err, "Should receive job event")
	assert.Equal(t, "completed", notification.Status, "Job should complete successfully after chaos removal")
	t.Logf("Job completed with status: %s", notification.Status)

	// Verify job status via API
	job, err := s.managerClient.GetJob(s.ctx, jobResp.JobID)
	require.NoError(t, err)
	assert.Equal(t, "completed", job.Status)

	helpers.DocumentResult(t, s.metrics, "Full flow handled mid-extraction chaos")
}

// TestFullFlow_MultiPointChaos tests the system under simultaneous
// chaos on multiple components.
func (s *ChaosTestSuite) TestFullFlow_MultiPointChaos() {
	t := s.T()

	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"degrade gracefully when multiple components experience chaos simultaneously",
		"RabbitMQ and PostgreSQL both have latency injected",
	))

	// Create connection
	configName := s.uniqueConfigName("chaos_postgres_test")
	pg := s.chaosInfra.PostgresInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	// Inject chaos on multiple components
	s.metrics.StartChaos()

	rabbitmqProxy := s.chaosInfra.GetRabbitMQProxy()
	postgresProxy := s.chaosInfra.GetPostgresProxy()
	require.NotNil(t, rabbitmqProxy)
	require.NotNil(t, postgresProxy)

	// RabbitMQ latency
	rabbitmqChaos := helpers.DefaultLatencyConfig(
		helpers.LatencyMs(setup.ChaosLatencyValues.Medium),
		helpers.LatencyMs(setup.ChaosLatencyValues.Jitter),
	)
	rabbitmqChaos.Name = "rabbitmq_multipoint_latency"
	_, err = helpers.InjectChaos(rabbitmqProxy, rabbitmqChaos)
	require.NoError(t, err)
	defer func() { _ = helpers.RemoveChaos(rabbitmqProxy, rabbitmqChaos.Name) }()

	// PostgreSQL latency
	postgresChaos := helpers.DefaultLatencyConfig(
		helpers.LatencyMs(setup.ChaosLatencyValues.Medium),
		helpers.LatencyMs(setup.ChaosLatencyValues.Jitter),
	)
	postgresChaos.Name = "postgres_multipoint_latency"
	_, err = helpers.InjectChaos(postgresProxy, postgresChaos)
	require.NoError(t, err)
	defer func() { _ = helpers.RemoveChaos(postgresProxy, postgresChaos.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Create job under multi-point chaos
	t.Log("Creating job under multi-point chaos...")
	start := time.Now()
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestFullFlow_MultiPointChaos"),
	})
	require.NoError(t, err)

	// Wait for completion
	notification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		jobResp.JobID,
		setup.JobCompletionTimeoutSlow,
	)

	duration := time.Since(start)
	s.metrics.EndChaos()

	// System should complete despite degraded conditions
	require.NoError(t, err, "Job should complete under multi-point chaos")
	assert.Equal(t, "completed", notification.Status, "Job should succeed under multi-point chaos")

	s.metrics.RecordRequest(notification.Status == "completed", false, duration)
	t.Logf("Job %s in %v under multi-point chaos", notification.Status, duration)

	helpers.DocumentResult(t, s.metrics, "System operated under multi-point chaos")
}

// TestFullFlow_RecoveryValidation validates that the system returns
// to steady state after chaos removal.
func (s *ChaosTestSuite) TestFullFlow_RecoveryValidation() {
	t := s.T()

	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"return to normal performance within 60 seconds after chaos removal",
		"severe chaos is applied then removed",
	))

	pg := s.chaosInfra.PostgresInternal()

	// Baseline measurement
	t.Log("Phase 1: Baseline measurement...")
	configName := s.uniqueConfigName("chaos_baseline")
	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	baselineStart := time.Now()
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestRecovery_Baseline"),
	})
	require.NoError(t, err)

	_, err = s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID, setup.JobCompletionTimeout)
	require.NoError(t, err)
	baselineDuration := time.Since(baselineStart)
	t.Logf("Baseline: %v", baselineDuration)

	// Apply severe chaos
	t.Log("Phase 2: Applying severe chaos...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetPostgresProxy()
	require.NotNil(t, proxy)

	chaosConfig := helpers.DefaultLatencyConfig(
		helpers.LatencyMs(setup.ChaosLatencyValues.High),
		helpers.LatencyMs(setup.ChaosLatencyValues.Jitter),
	)
	_, err = helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)

	// Run operations during chaos to stress the system
	t.Log("Stressing system during chaos...")
	chaosDeadline := time.Now().Add(setup.ChaosInjectionDuration)
	chaosOpsSuccess := 0
	chaosOpsFailed := 0
	for time.Now().Before(chaosDeadline) {
		chaosConfigName := s.uniqueConfigName("chaos_stress")
		_, createErr := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
			ConfigName:   chaosConfigName,
			Type:         "POSTGRESQL",
			Host:         pg.Host,
			Port:         pg.Port,
			DatabaseName: pg.Database,
			Username:     pg.Username,
			Password:     pg.Password,
		})
		if createErr == nil {
			chaosOpsSuccess++
			// Cleanup immediately
			_ = s.managerClient.DeleteConnectionByConfigName(s.ctx, chaosConfigName)
		} else {
			chaosOpsFailed++
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Logf("Operations during chaos: %d succeeded, %d failed", chaosOpsSuccess, chaosOpsFailed)

	// Remove chaos
	t.Log("Phase 3: Removing chaos...")
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	s.metrics.EndChaos()
	s.metrics.StartRecovery()

	// Wait briefly for system to stabilize
	time.Sleep(5 * time.Second)

	// Recovery measurement
	t.Log("Phase 4: Recovery measurement...")
	configName = s.uniqueConfigName("chaos_recovery")
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "POSTGRESQL",
		Host:         pg.Host,
		Port:         pg.Port,
		DatabaseName: pg.Database,
		Username:     pg.Username,
		Password:     pg.Password,
	})
	require.NoError(t, err)

	recoveryStart := time.Now()
	jobResp, err = s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestRecovery_PostChaos"),
	})
	require.NoError(t, err)

	notification, err := s.eventConsumer.WaitForJobEvent(s.ctx, jobResp.JobID, setup.JobCompletionTimeout)
	require.NoError(t, err)
	assert.Equal(t, "completed", notification.Status)

	recoveryDuration := time.Since(recoveryStart)
	s.metrics.EndRecovery()

	// Recovery should be comparable to baseline (within 3x)
	maxAcceptable := baselineDuration * 3
	t.Logf("Baseline: %v, Recovery: %v (max acceptable: %v)", baselineDuration, recoveryDuration, maxAcceptable)

	assert.Less(t, recoveryDuration, maxAcceptable, "Recovery should be within 3x of baseline")

	helpers.DocumentResult(t, s.metrics, "System returned to steady state after chaos removal")
}
