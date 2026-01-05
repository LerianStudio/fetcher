//go:build chaos

package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/tests/chaos/helpers"
	"github.com/LerianStudio/fetcher/tests/chaos/setup"
	"github.com/LerianStudio/fetcher/tests/shared/client"
	"github.com/LerianStudio/fetcher/tests/shared/fixtures"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ChaosTestSuite provides full E2E testing with chaos injection capabilities.
// All tests interact with the system through the Manager API, just like production.
type ChaosTestSuite struct {
	suite.Suite

	ctx           context.Context
	cancel        context.CancelFunc
	chaosInfra    *setup.ChaosInfrastructure
	managerClient *client.ManagerClient
	seaweedClient *client.SeaweedFSClient
	eventConsumer *client.RabbitMQEventConsumer
	metrics       *helpers.ChaosMetrics
}

// SetupSuite starts full infrastructure with Toxiproxy proxies.
func (s *ChaosTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), setup.ChaosInfraStartupTimeout)

	// Start chaos infrastructure (includes all containers + Toxiproxy)
	var err error
	s.chaosInfra, err = setup.StartChaosInfrastructure(s.ctx)
	require.NoError(s.T(), err, "Failed to start chaos infrastructure")

	// Initialize clients using PROXIED endpoints (traffic goes through Toxiproxy)
	// Note: For chaos tests, we use proxy URLs so we can inject faults
	s.managerClient = client.NewManagerClient(
		s.chaosInfra.ManagerProxyURL,
		fixtures.TestOrganizationID,
	)
	s.seaweedClient = client.NewSeaweedFSClient(s.chaosInfra.SeaweedFSProxyURL)

	// Event consumer connects through RabbitMQ proxy
	s.eventConsumer, err = client.NewRabbitMQEventConsumer(s.chaosInfra.RabbitMQProxyURI)
	require.NoError(s.T(), err, "Failed to create event consumer")

	// Wait for Manager to be ready
	err = s.waitForManagerReady()
	require.NoError(s.T(), err, "Manager not ready")

	s.T().Log("Chaos test suite ready")
}

// TearDownSuite cleans up all infrastructure.
func (s *ChaosTestSuite) TearDownSuite() {
	if s.eventConsumer != nil {
		s.eventConsumer.Close()
	}

	// Use fresh context for cleanup - the original ctx may be expired
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cleanupCancel()

	if s.chaosInfra != nil {
		if err := s.chaosInfra.Stop(cleanupCtx); err != nil {
			s.T().Logf("Warning: failed to stop chaos infrastructure: %v", err)
		}
	}

	if s.cancel != nil {
		s.cancel()
	}
}

// SetupTest runs before each test - cleans up chaos and connections.
func (s *ChaosTestSuite) SetupTest() {
	// Remove any lingering chaos from previous tests
	if s.chaosInfra != nil {
		if err := s.chaosInfra.ResetChaos(); err != nil {
			s.T().Logf("Warning: failed to reset chaos in SetupTest: %v", err)
		}
	}

	// Reset metrics for new test
	s.metrics = helpers.NewChaosMetrics()
	s.metrics.StartTest()

	// Clean up test connections - log warnings for failed deletions
	testConnections := []string{
		"chaos_postgres_test",
		"chaos_mysql_test",
		"chaos_sqlserver_test",
		"chaos_oracle_test",
		"chaos_mongodb_test",
	}
	for _, configName := range testConnections {
		if err := s.managerClient.DeleteConnectionByConfigName(s.ctx, configName); err != nil {
			// Only log if it's not a "not found" error (which is expected for clean state)
			s.T().Logf("Warning: failed to clean up connection %s: %v", configName, err)
		}
	}
}

// TearDownTest runs after each test.
func (s *ChaosTestSuite) TearDownTest() {
	if s.chaosInfra != nil {
		if err := s.chaosInfra.ResetChaos(); err != nil {
			s.T().Logf("Warning: failed to reset chaos in TearDownTest: %v", err)
		}
	}
	if s.metrics != nil {
		s.metrics.EndTest()
	}
}

// waitForManagerReady waits for the Manager API to be accessible.
func (s *ChaosTestSuite) waitForManagerReady() error {
	deadline := time.Now().Add(setup.ManagerReadyTimeout)
	for {
		if time.Now().After(deadline) {
			return context.DeadlineExceeded
		}

		err := s.managerClient.HealthCheck(s.ctx)
		if err == nil {
			return nil
		}

		time.Sleep(setup.ManagerReadyPollInterval)
	}
}

// uniqueConfigName generates unique config names to avoid idempotency conflicts.
func (s *ChaosTestSuite) uniqueConfigName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// testMetadata returns metadata to make each request unique.
func (s *ChaosTestSuite) testMetadata(testName string) map[string]any {
	return map[string]any{
		"source":    "chaos_test",
		"testName":  testName,
		"timestamp": time.Now().UnixNano(),
	}
}

// waitForJobProcessing polls until job reaches a processing state (not pending).
func (s *ChaosTestSuite) waitForJobProcessing(jobID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for job %s to start processing", jobID)
		}

		job, err := s.managerClient.GetJob(s.ctx, jobID)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Job is processing if status is not pending/queued
		if job.Status != "pending" && job.Status != "queued" && job.Status != "" {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// TestChaosE2E runs the chaos test suite.
func TestChaosE2E(t *testing.T) {
	if os.Getenv("SKIP_CHAOS_TESTS") == "true" {
		t.Skip("Skipping chaos tests (SKIP_CHAOS_TESTS=true)")
	}
	suite.Run(t, new(ChaosTestSuite))
}

// containsString checks if a string contains a substring (case-insensitive).
func containsString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
