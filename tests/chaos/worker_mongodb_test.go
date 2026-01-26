//go:build chaos

package e2e

import (
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/tests/shared/chaos"
	"github.com/LerianStudio/fetcher/tests/shared/client"
	"github.com/LerianStudio/fetcher/tests/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMongoDBExternalTimeout_ExtractionJobFails tests that the system properly rejects
// job creation when MongoDB external is unavailable due to timeout chaos.
func (s *ChaosTestSuite) TestMongoDBExternalTimeout_ExtractionJobFails() {
	t := s.T()

	chaos.DocumentHypothesis(t, chaos.FormatHypothesis(
		"reject job creation with connection error when MongoDB external times out",
		"MongoDB external has timeout chaos injected before job creation",
	))

	// Phase 1: Create connection to external MongoDB (before chaos)
	t.Log("Phase 1: Creating MongoDB external connection...")
	configName := s.uniqueConfigName("chaos_mongodb_test")
	mongo := s.chaosInfra.ProxyRouter.GetProxyConnection(chaos.ServiceMongoExternal, s.chaosInfra.MongoExternalInternal())

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

	chaosConfig := chaos.DefaultTimeoutConfig(chaos.TimeoutMs(chaos.ChaosTimeoutValues.Medium))
	toxic, err := s.chaosInfra.ChaosOps.AddChaos(chaos.ServiceMongoExternal, chaosConfig)
	require.NoError(t, err)
	require.NotNil(t, toxic)
	defer func() { _ = s.chaosInfra.ChaosOps.RemoveChaos(chaos.ServiceMongoExternal, chaosConfig.Name) }()

	time.Sleep(chaos.StabilizationDelay)

	// Phase 3: Create job - should be REJECTED because connection test fails
	t.Log("Phase 3: Creating extraction job under chaos (expecting rejection)...")
	_, err = s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
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
	err = s.chaosInfra.ChaosOps.RemoveChaos(chaos.ServiceMongoExternal, chaosConfig.Name)
	require.NoError(t, err)

	s.metrics.StartRecovery()
	time.Sleep(chaos.RecoveryObservationTime)

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

	recoveryJob, err := s.managerClient.CreateFetcherJob(s.ctx, model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				recoveryConfigName: {"transactions": {"account_id", "amount"}},
			},
		},
		Metadata: s.testMetadata("TestMongoDBExternalTimeout_Recovery"),
	})
	require.NoError(t, err, "Recovery job creation should succeed")

	recoveryNotification, err := s.eventConsumer.WaitForJobEvent(
		s.ctx,
		recoveryJob.JobID.String(),
		config.JobCompletionTimeout,
	)
	require.NoError(t, err, "Recovery job should complete")
	assert.Equal(t, "completed", recoveryNotification.Status, "Recovery job should complete successfully")

	s.metrics.EndRecovery()

	chaos.DocumentResult(t, s.metrics, "MongoDB external timeout correctly rejected job, recovery successful")
}
