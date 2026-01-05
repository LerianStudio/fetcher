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

// TestMongoDBExternalTimeout_ExtractionJobFails tests that the system properly rejects
// job creation when MongoDB external is unavailable due to timeout chaos.
func (s *ChaosTestSuite) TestMongoDBExternalTimeout_ExtractionJobFails() {
	t := s.T()

	helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
		"reject job creation with connection error when MongoDB external times out",
		"MongoDB external has timeout chaos injected before job creation",
	))

	// Phase 1: Create connection to external MongoDB (before chaos)
	t.Log("Phase 1: Creating MongoDB external connection...")
	configName := s.uniqueConfigName("chaos_mongodb_test")
	mongo := s.chaosInfra.MongoExternalProxyInternal()

	connResp, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   configName,
		Type:         "MONGODB",
		Host:         mongo.Host,
		Port:         mongo.Port,
		DatabaseName: mongo.Database,
		Username:     mongo.Username,
		Password:     mongo.Password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, connResp.ID, "Connection ID should not be empty")

	// Phase 2: Inject MongoDB external timeout BEFORE creating job
	t.Log("Phase 2: Injecting MongoDB external timeout chaos...")
	s.metrics.StartChaos()

	proxy := s.chaosInfra.GetMongoExternalProxy()
	require.NotNil(t, proxy, "MongoDB external proxy should exist")

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
				configName: {"transactions": {"account_id", "amount", "currency"}},
			},
		},
		Metadata: s.testMetadata("TestMongoDBExternalTimeout"),
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
	recoveryConfigName := s.uniqueConfigName("chaos_mongodb_recovery")
	_, err = s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{
		ConfigName:   recoveryConfigName,
		Type:         "MONGODB",
		Host:         mongo.Host,
		Port:         mongo.Port,
		DatabaseName: mongo.Database,
		Username:     mongo.Username,
		Password:     mongo.Password,
	})
	require.NoError(t, err, "Recovery connection should succeed")

	recoveryJob, err := s.managerClient.CreateFetcherJob(s.ctx, client.FetcherRequest{
		DataRequest: client.DataRequest{
			MappedFields: map[string]map[string][]string{
				recoveryConfigName: {"transactions": {"account_id", "amount"}},
			},
		},
		Metadata: s.testMetadata("TestMongoDBExternalTimeout_Recovery"),
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

	helpers.DocumentResult(t, s.metrics, "MongoDB external timeout correctly rejected job, recovery successful")
}
