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

// TestMySQLTimeout_ExtractionJobFails tests that the system properly rejects
// job creation when MySQL is unavailable due to timeout chaos.
func (s *ChaosTestSuite) TestMySQLTimeout_ExtractionJobFails() {
	t := s.T()

	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"reject job creation with connection error when MySQL times out",
		"MySQL has timeout chaos injected before job creation",
	))

	// Phase 1: Create connection (before chaos)
	t.Log("Phase 1: Creating MySQL connection...")
	configName := s.uniqueConfigName("chaos_mysql_test")
	mysql := s.chaosInfra.MySQLProxyInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "MYSQL",
		Host:         mysql.Host,
		Port:         mysql.Port,
		DatabaseName: mysql.Database,
		Username:     mysql.Username,
		Password:     mysql.Password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, connResp.ID, "Connection ID should not be empty")

	// Phase 2: Inject MySQL timeout BEFORE creating job
	t.Log("Phase 2: Injecting MySQL timeout chaos...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetMySQLProxy()
	require.NotNil(t, proxy, "MySQL proxy should exist")

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
				configName: {"transactions": {"id", "account_id", "amount"}},
			},
		},
		Metadata: s.testMetadata("TestMySQLTimeout"),
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
	recoveryConfigName := s.uniqueConfigName("chaos_mysql_recovery")
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   recoveryConfigName,
		Type:         "MYSQL",
		Host:         mysql.Host,
		Port:         mysql.Port,
		DatabaseName: mysql.Database,
		Username:     mysql.Username,
		Password:     mysql.Password,
	})
	require.NoError(t, err, "Recovery connection should succeed")

	recoveryJob, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				recoveryConfigName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestMySQLTimeout_Recovery"),
	})
	require.NoError(t, err, "Recovery job creation should succeed")

	recoveryNotification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		recoveryJob.JobID.String(),
		setup.JobCompletionTimeout,
	)
	require.NoError(t, err, "Recovery job should complete")
	assert.Equal(t, "completed", recoveryNotification.Status, "Recovery job should complete successfully")

	s.metrics.EndRecovery()

	helpers.DocumentResult(t, s.metrics, "MySQL timeout correctly rejected job, recovery successful")
}

// TestMySQLBandwidth_ThrottledExtractionCompletes tests extraction under bandwidth limits.
func (s *ChaosTestSuite) TestMySQLBandwidth_ThrottledExtractionCompletes() {
	t := s.T()

	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"complete extraction with reduced throughput when bandwidth is limited",
		"MySQL has bandwidth limited to 10KB/s",
	))

	// Phase 1: Create connection
	t.Log("Phase 1: Creating MySQL connection...")
	configName := s.uniqueConfigName("chaos_mysql_test")
	mysql := s.chaosInfra.MySQLProxyInternal()

	_, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "MYSQL",
		Host:         mysql.Host,
		Port:         mysql.Port,
		DatabaseName: mysql.Database,
		Username:     mysql.Username,
		Password:     mysql.Password,
	})
	require.NoError(t, err)

	// Phase 2: Inject bandwidth limit
	t.Log("Phase 2: Injecting bandwidth limit chaos...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetMySQLProxy()
	require.NotNil(t, proxy)

	chaosConfig := helpers.DefaultBandwidthConfig(setup.ChaosBandwidthValues.Medium)
	toxic, err := helpers.InjectChaos(proxy, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = helpers.RemoveChaos(proxy, chaosConfig.Name) }()

	time.Sleep(setup.StabilizationDelay)

	// Phase 3: Create job - should complete slowly
	t.Log("Phase 3: Creating extraction job under bandwidth chaos...")
	start := time.Now()
	jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {"transactions": {"id", "account_id"}},
			},
		},
		Metadata: s.testMetadata("TestMySQLBandwidth"),
	})
	require.NoError(t, err)

	notification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		jobResp.JobID.String(),
		setup.JobCompletionTimeoutSlow, // Extended timeout for bandwidth throttling
	)

	duration := time.Since(start)
	s.metrics.EndChaos()

	// Assert job completed successfully despite bandwidth throttling
	require.NoError(t, err, "Job should complete under bandwidth throttling")
	require.Equal(t, "completed", notification.Status,
		"Job status should be 'completed' not '%s'", notification.Status)
	s.metrics.RecordRequest(true, false, duration)
	t.Logf("Job completed in %v under bandwidth throttling", duration)

	// Cleanup
	err = helpers.RemoveChaos(proxy, chaosConfig.Name)
	require.NoError(t, err)

	helpers.DocumentResult(t, s.metrics, "MySQL extraction completed under bandwidth throttling")
}
