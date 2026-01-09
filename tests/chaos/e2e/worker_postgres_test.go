//go:build chaos

package e2e

import (
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/tests/chaos/helpers"
	"github.com/LerianStudio/fetcher/tests/chaos/setup"
	"github.com/LerianStudio/fetcher/tests/shared/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgresTimeout_ExtractionJobFails tests that the system properly rejects
// job creation when PostgreSQL is unavailable due to timeout chaos.
func (s *ChaosTestSuite) TestPostgresTimeout_ExtractionJobFails() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"reject job creation with connection error when PostgreSQL times out",
		"PostgreSQL has timeout chaos injected before job creation",
	))

	// Phase 1: Create connection (before chaos)
	t.Log("Phase 1: Creating PostgreSQL connection...")
	configName := s.uniqueConfigName("chaos_postgres_test")
	pg := s.chaosInfra.PostgresProxyInternal()

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
	assert.NotEmpty(t, connResp.ID)

	// Phase 2: Inject PostgreSQL timeout BEFORE creating job
	t.Log("Phase 2: Injecting PostgreSQL timeout chaos...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetPostgresProxy()
	require.NotNil(t, proxy, "PostgreSQL proxy should exist")

	chaosConfig := helpers.DefaultTimeoutConfig(helpers.TimeoutMs(setup.ChaosTimeoutValues.Medium))
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 3: Create job - should be REJECTED because connection test fails
	t.Log("Phase 3: Creating extraction job under chaos (expecting rejection)...")
	_, err = s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount"},
				},
			},
		},
		Metadata: s.testMetadata("TestPostgresTimeout"),
	})

	s.metrics.EndChaos()

	// Job creation should fail with connection error
	require.Error(t, err, "Job creation should be rejected when database is unavailable")
	assert.Contains(t, err.Error(), "400", "Should receive HTTP 400 for connection failure")
	t.Logf("Job creation correctly rejected: %v", err)

	// Phase 4: Remove chaos and verify new jobs work
	t.Log("Phase 4: Removing chaos, verifying recovery...")
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	s.metrics.StartRecovery()
	time.Sleep(setup.RecoveryObservationTime)

	// Create new connection and job - should succeed
	configName = s.uniqueConfigName("chaos_postgres_recovery")
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

	recoveryJob, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestPostgresTimeout_Recovery"),
	})
	require.NoError(t, err, "Recovery job creation should succeed")

	recoveryNotification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		recoveryJob.JobID.String(),
		setup.JobCompletionTimeout,
	)
	require.NoError(t, err, "Recovery job should complete")
	assert.Equal(t, "completed", recoveryNotification.Status)

	s.metrics.EndRecovery()

	// Document results
	helpers.DocumentResult(t, s.metrics, "PostgreSQL timeout correctly rejected job, recovery successful")
}

// TestPostgresLatency_SlowExtractionCompletes tests that extraction
// completes despite PostgreSQL latency.
func (s *ChaosTestSuite) TestPostgresLatency_SlowExtractionCompletes() {
	t := s.T()

	// Document hypothesis
	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"complete extraction with extended duration when PostgreSQL has network latency",
		"PostgreSQL has 500ms latency injected (Low - to avoid schema discovery timeout)",
	))

	// Create connection
	configName := s.uniqueConfigName("chaos_postgres_test")
	pg := s.chaosInfra.PostgresProxyInternal()

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

	// Inject latency
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetPostgresProxy()
	require.NotNil(t, proxy)

	// Use Low latency (500ms) to avoid schema discovery timeout
	// Schema discovery makes multiple queries, so 3s per query exceeds the 30s timeout
	chaosConfig := helpers.DefaultLatencyConfig(
		helpers.LatencyMs(setup.ChaosLatencyValues.Low),
		helpers.LatencyMs(setup.ChaosLatencyValues.Jitter),
	)
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Create job under latency
	start := time.Now()
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id", "amount"}},
			},
		},
		Metadata: s.testMetadata("TestPostgresLatency"),
	})
	require.NoError(t, err)

	// Wait for completion - should succeed despite latency
	notification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		jobResp.JobID.String(),
		setup.JobCompletionTimeoutSlow, // Extended timeout
	)

	duration := time.Since(start)
	s.metrics.EndChaos()

	require.NoError(t, err, "Job should complete despite latency")
	assert.Equal(t, "completed", notification.Status)
	s.metrics.RecordRequest(true, false, duration)

	// Cleanup
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	helpers.DocumentResult(t, s.metrics, "PostgreSQL extraction completed despite latency")
	t.Logf("Job completed in %v (expected slower due to 500ms latency per query)", duration)
}
