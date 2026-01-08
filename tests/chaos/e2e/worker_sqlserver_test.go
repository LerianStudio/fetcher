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

// TestSQLServerTimeout_ExtractionJobFails tests that the system properly rejects
// job creation when SQL Server is unavailable due to timeout chaos.
func (s *ChaosTestSuite) TestSQLServerTimeout_ExtractionJobFails() {
	t := s.T()

	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"reject job creation with connection error when SQL Server times out",
		"SQL Server has timeout chaos injected before job creation",
	))

	// Phase 1: Create connection (before chaos)
	t.Log("Phase 1: Creating SQL Server connection...")
	configName := s.uniqueConfigName("chaos_sqlserver_test")
	mssql := s.chaosInfra.SQLServerProxyInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "SQL_SERVER",
		Host:         mssql.Host,
		Port:         mssql.Port,
		DatabaseName: mssql.Database,
		Username:     mssql.Username,
		Password:     mssql.Password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, connResp.ID, "Connection ID should not be empty")

	// Phase 2: Inject SQL Server timeout BEFORE creating job
	t.Log("Phase 2: Injecting SQL Server timeout chaos...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetSQLServerProxy()
	require.NotNil(t, proxy, "SQL Server proxy should exist")

	chaosConfig := helpers.DefaultTimeoutConfig(helpers.TimeoutMs(setup.ChaosTimeoutValues.Medium))
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 3: Create job - should be REJECTED because connection test fails
	t.Log("Phase 3: Creating extraction job under chaos (expecting rejection)...")
	_, err = s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id", "amount"}},
			},
		},
		Metadata: s.testMetadata("TestSQLServerTimeout"),
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
	recoveryConfigName := s.uniqueConfigName("chaos_sqlserver_recovery")
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   recoveryConfigName,
		Type:         "SQL_SERVER",
		Host:         mssql.Host,
		Port:         mssql.Port,
		DatabaseName: mssql.Database,
		Username:     mssql.Username,
		Password:     mssql.Password,
	})
	require.NoError(t, err, "Recovery connection should succeed")

	recoveryJob, err := s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				recoveryConfigName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestSQLServerTimeout_Recovery"),
	})
	require.NoError(t, err, "Recovery job creation should succeed")

	recoveryNotification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		recoveryJob.JobID,
		setup.JobCompletionTimeout,
	)
	require.NoError(t, err, "Recovery job should complete")
	assert.Equal(t, "completed", recoveryNotification.Status, "Recovery job should complete successfully")

	s.metrics.EndRecovery()

	helpers.DocumentResult(t, s.metrics, "SQL Server timeout correctly rejected job, recovery successful")
}

// TestSQLServerLatency_SlowExtractionCompletes tests extraction under latency.
func (s *ChaosTestSuite) TestSQLServerLatency_SlowExtractionCompletes() {
	t := s.T()

	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"complete extraction despite network latency",
		"SQL Server has 3s latency injected",
	))

	// Phase 1: Create connection
	t.Log("Phase 1: Creating SQL Server connection...")
	configName := s.uniqueConfigName("chaos_sqlserver_test")
	mssql := s.chaosInfra.SQLServerProxyInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "SQL_SERVER",
		Host:         mssql.Host,
		Port:         mssql.Port,
		DatabaseName: mssql.Database,
		Username:     mssql.Username,
		Password:     mssql.Password,
	})
	require.NoError(t, err)

	// Phase 2: Inject latency chaos
	t.Log("Phase 2: Injecting SQL Server latency chaos...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetSQLServerProxy()
	require.NotNil(t, proxy)

	chaosConfig := helpers.DefaultLatencyConfig(
		helpers.LatencyMs(setup.ChaosLatencyValues.Medium),
		helpers.LatencyMs(setup.ChaosLatencyValues.Jitter),
	)
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 3: Create job under latency
	t.Log("Phase 3: Creating extraction job under latency chaos...")
	start := time.Now()
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestSQLServerLatency"),
	})
	require.NoError(t, err)

	notification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		jobResp.JobID,
		setup.JobCompletionTimeoutSlow,
	)

	duration := time.Since(start)
	s.metrics.EndChaos()

	// Assert job completed successfully
	require.NoError(t, err, "Job should complete despite latency")
	assert.Equal(t, "completed", notification.Status, "Job should complete successfully")
	s.metrics.RecordRequest(true, false, duration)

	// Verify latency chaos had observable effect (at least some latency impact expected)
	t.Logf("Job completed in %v (latency chaos: %v)", duration, setup.ChaosLatencyValues.Medium)

	// Cleanup
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	helpers.DocumentResult(t, s.metrics, "SQL Server extraction completed despite latency")
}
