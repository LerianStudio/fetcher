//go:build chaos

package chaos

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/itestkit"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/addons/metricskit"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/addons/queuekit"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/mongodb"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/rabbitmq"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/redis"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/seaweedfs"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/v2/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SLO THRESHOLDS
// =============================================================================

// SLO thresholds define the service level objectives for chaos testing conditions.
// These thresholds represent the minimum acceptable performance under failure scenarios.
const (
	// SLOManagerSuccessRate is the minimum acceptable success rate for Manager API requests
	// under chaos conditions. 95% means we allow up to 5% of requests to fail.
	SLOManagerSuccessRate = 95.0
	// SLOManagerP99Latency is the maximum acceptable P99 latency for Manager API requests
	// under chaos conditions.
	SLOManagerP99Latency = 2 * time.Second

	// SLOWorkerJobSuccessRate is the minimum acceptable success rate for Worker job processing
	// under chaos conditions. 90% means we allow up to 10% of jobs to fail.
	SLOWorkerJobSuccessRate = 90.0
	// SLOWorkerP99Duration is the maximum acceptable P99 duration for job completion
	// under chaos conditions.
	SLOWorkerP99Duration = 60 * time.Second

	// SLORecoveryTime is the maximum acceptable time for the system to recover
	// after chaos is removed and normal operation should resume.
	SLORecoveryTime = 35 * time.Second

	// CircuitBreakerThreshold is the number of consecutive failures before the circuit breaker opens.
	// This matches the configuration in pkg/rabbitmq/rabbitmq.go.
	CircuitBreakerThreshold = 5
	// CircuitBreakerCooldown is the duration the circuit breaker remains open before attempting
	// to half-open and allow test requests through.
	CircuitBreakerCooldown = 30 * time.Second

	// chaosProductName is the product name used for chaos test connections and job requests.
	// It must match in both CreateConnection (X-Product-Name header) and FetcherRequest
	// (metadata.source) so that validateProductOwnership passes during job creation.
	chaosProductName = "chaos-test"
)

// Test timeout constants define the maximum duration for various test scenarios.
const (
	// DefaultChaosTestTimeout is the default timeout for most chaos tests.
	DefaultChaosTestTimeout = 5 * time.Minute
	// LongChaosTestTimeout is used for tests that require extended observation periods.
	LongChaosTestTimeout = 10 * time.Minute
	// DefaultJobTimeout is the maximum time to wait for a single job to complete.
	DefaultJobTimeout = 2 * time.Minute
)

// =============================================================================
// CHAOS INFRASTRUCTURE
// =============================================================================

// ChaosInfra holds chaos-enabled infrastructure components.
// Similar to CoreInfra from tests/shared but all components are configured with
// EnableProxy: true so traffic can be routed through Toxiproxy for fault injection.
type ChaosInfra struct {
	// MongoDB is the primary data store with chaos proxy enabled for fault injection.
	MongoDB *mongodb.MongoDBInfra
	// RabbitMQ is the message broker with chaos proxy enabled for fault injection.
	RabbitMQ *rabbitmq.RabbitInfra
	// Redis is the cache layer with chaos proxy enabled for fault injection.
	Redis *redis.RedisInfra
	// SeaweedFS is the distributed file system with chaos proxy enabled for fault injection.
	SeaweedFS *seaweedfs.SeaweedFSInfra
}

// NewChaosInfra creates the chaos-enabled infrastructure configuration.
// All infrastructure components have EnableProxy: true so that traffic
// can be routed through Toxiproxy for chaos injection.
func NewChaosInfra() *ChaosInfra {
	return &ChaosInfra{
		MongoDB: mongodb.NewMongoDBInfra(mongodb.MongoDBConfig{
			Name:        "fetcher-chaos",
			Username:    e2eshared.CoreInfraUsername,
			Password:    e2eshared.CoreInfraPassword,
			EnableProxy: true,
		}),
		RabbitMQ: rabbitmq.NewRabbitInfra(rabbitmq.RabbitConfig{
			Name:        "fetcher-chaos",
			Username:    e2eshared.CoreInfraUsername,
			Password:    e2eshared.CoreInfraPassword,
			EnableProxy: true,
			Options: []rabbitmq.RabbitOption{
				rabbitmq.WithRabbitDefinitions(definitionsPath()),
			},
		}),
		Redis: redis.NewRedisInfra(redis.RedisConfig{
			Name:        "fetcher-chaos",
			Password:    e2eshared.CoreInfraPassword,
			EnableProxy: true,
		}),
		SeaweedFS: seaweedfs.NewSeaweedFSInfra(seaweedfs.SeaweedFSConfig{
			Name:        "fetcher-chaos",
			EnableProxy: true,
		}),
	}
}

// definitionsPath returns the absolute path to the RabbitMQ topology definitions file.
// The path is resolved relative to this source file to ensure correct resolution
// regardless of the working directory.
func definitionsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "shared", "testdata", "definitions.json")
}

// Infras returns all infrastructure components as a slice of itestkit.Infra interfaces.
// This is used by the itestkit.Suite builder to start all infrastructure in the correct order.
func (c *ChaosInfra) Infras() []itestkit.Infra {
	return []itestkit.Infra{c.MongoDB, c.RabbitMQ, c.Redis, c.SeaweedFS}
}

// =============================================================================
// PROXIED APP ENVIRONMENT
// =============================================================================

// ProxiedAppEnv holds environment configuration with proxied endpoints.
// This struct configures Manager and Worker containers to connect through
// Toxiproxy, enabling fault injection during chaos tests.
type ProxiedAppEnv struct {
	// Network is the Docker network name that enables container-to-container communication.
	// When set, containers can reach Toxiproxy using its network alias.
	Network string

	// MongoHost is the hostname for the MongoDB connection (proxied through Toxiproxy).
	MongoHost string
	// MongoPort is the port for the MongoDB connection.
	MongoPort string
	// MongoUser is the username for MongoDB authentication.
	MongoUser string
	// MongoPassword is the password for MongoDB authentication.
	MongoPassword string

	// RabbitHost is the hostname for the RabbitMQ connection (proxied through Toxiproxy).
	RabbitHost string
	// RabbitPort is the AMQP port for the RabbitMQ connection.
	RabbitPort string
	// RabbitUser is the username for RabbitMQ authentication.
	RabbitUser string
	// RabbitPassword is the password for RabbitMQ authentication.
	RabbitPassword string

	// RedisHost is the hostname for the Redis connection.
	RedisHost string
	// RedisPort is the port for the Redis connection.
	RedisPort string
	// RedisPassword is the password for Redis authentication.
	RedisPassword string

	// SeaweedFSHost is the hostname for the SeaweedFS filer (proxied through Toxiproxy).
	SeaweedFSHost string
	// SeaweedFSPort is the filer port for the SeaweedFS connection.
	SeaweedFSPort string
}

// buildProxiedAppEnv constructs environment configuration from infrastructure endpoints.
// ContainerHostPort preserves in-network proxy addresses for app containers while the
// public HostPort contract remains host-usable for test code.
func buildProxiedAppEnv() (*ProxiedAppEnv, error) {
	// The app containers must use the in-network endpoints so traffic flows
	// through Toxiproxy when chaos mode is enabled.
	mongoHost, mongoPort, err := chaosInfra.MongoDB.HostPort()
	if err != nil {
		return nil, fmt.Errorf("mongo host/port: %w", err)
	}

	rabbitHost, rabbitPort, err := chaosInfra.RabbitMQ.HostPort()
	if err != nil {
		return nil, fmt.Errorf("rabbit host/port: %w", err)
	}

	redisHost, redisPort, err := chaosInfra.Redis.HostPort()
	if err != nil {
		return nil, fmt.Errorf("redis host/port: %w", err)
	}

	seaweedHost, seaweedPort, err := chaosInfra.SeaweedFS.HostPort()
	if err != nil {
		return nil, fmt.Errorf("seaweedfs host/port: %w", err)
	}

	return &ProxiedAppEnv{
		Network:        suite.Network(),
		MongoHost:      mongoHost,
		MongoPort:      strconv.Itoa(mongoPort),
		MongoUser:      e2eshared.CoreInfraUsername,
		MongoPassword:  e2eshared.CoreInfraPassword,
		RabbitHost:     rabbitHost,
		RabbitPort:     strconv.Itoa(rabbitPort),
		RabbitUser:     e2eshared.CoreInfraUsername,
		RabbitPassword: e2eshared.CoreInfraPassword,
		RedisHost:      redisHost,
		RedisPort:      strconv.Itoa(redisPort),
		RedisPassword:  e2eshared.CoreInfraPassword,
		SeaweedFSHost:  seaweedHost,
		SeaweedFSPort:  strconv.Itoa(seaweedPort),
	}, nil
}

// ToAppEnv converts ProxiedAppEnv to the standard AppEnv format from tests/shared.
// This allows reusing the StartManager and StartWorker functions from the shared package.
func (e *ProxiedAppEnv) ToAppEnv() *e2eshared.AppEnv {
	return &e2eshared.AppEnv{
		Network:        e.Network,
		MongoHost:      e.MongoHost,
		MongoPort:      e.MongoPort,
		MongoUser:      e.MongoUser,
		MongoPassword:  e.MongoPassword,
		RabbitHost:     e.RabbitHost,
		RabbitPort:     e.RabbitPort,
		RabbitUser:     e.RabbitUser,
		RabbitPassword: e.RabbitPassword,
		RedisHost:      e.RedisHost,
		RedisPort:      e.RedisPort,
		RedisPassword:  e.RedisPassword,
		SeaweedFSHost:  e.SeaweedFSHost,
		SeaweedFSPort:  e.SeaweedFSPort,
	}
}

// RabbitAMQPURL returns the AMQP connection URL for the proxied RabbitMQ.
// This URL is used by queuekit for consuming job completion notifications.
func (e *ProxiedAppEnv) RabbitAMQPURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/",
		e.RabbitUser, e.RabbitPassword, e.RabbitHost, e.RabbitPort)
}

// =============================================================================
// TEST HELPERS
// =============================================================================

// createTestConnection creates a PostgreSQL connection configuration for chaos testing.
// The connection uses the PostgreSQL container address that is accessible from
// the Manager container via Docker networking.
//
// If the initial connection attempt fails, it retries using the Docker gateway IP
// as a fallback, which may be needed depending on the network configuration.
//
// The connection is automatically cleaned up when the test completes.
func createTestConnection(t *testing.T, ctx context.Context, prefix string) *e2eshared.ConnectionResponse {
	t.Helper()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Log the address being used for debugging
	t.Logf("PostgreSQL address for connection: %s:%d", pgHost, pgPort)

	// Product name must match metadata.source in createBasicFetcherRequest
	// so that validateProductOwnership passes during job creation.
	uniqueName := fmt.Sprintf("chaos-%s-%s", prefix, uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, chaosProductName, e2eshared.ConnectionInput{
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     e2eshared.SourceDBUsername,
		Password:     e2eshared.SourceDBPassword,
	})
	if err != nil {
		t.Logf("Failed to create connection with host=%s port=%d: %v", pgHost, pgPort, err)
		// Try with explicit Docker gateway as fallback
		gatewayIP := itestkit.HostGatewayIP()
		t.Logf("Retrying with Docker gateway IP: %s:%d", gatewayIP, pgPort)
		conn, err = apiClient.CreateConnection(ctx, chaosProductName, e2eshared.ConnectionInput{
			ConfigName:   uniqueName,
			Type:         e2eshared.DBTypePostgreSQL,
			Host:         gatewayIP,
			Port:         pgPort,
			DatabaseName: "testdb",
			Username:     e2eshared.SourceDBUsername,
			Password:     e2eshared.SourceDBPassword,
		})
	}
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	return conn
}

// createBasicFetcherRequest creates a basic fetcher request for testing data extraction.
// The request extracts id, account_id, amount, and status columns from the transactions table.
func createBasicFetcherRequest(configName string) model.FetcherRequest {
	return model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				configName: {
					"transactions": {"id", "account_id", "amount", "status"},
				},
			},
		},
		Metadata: map[string]any{
			"source": chaosProductName,
			"test":   "chaos-validation",
		},
	}
}

// waitForJobCompletion waits for a job to complete by consuming RabbitMQ notifications.
// It uses queuekit to subscribe to the notifications queue and wait for a message
// matching the given job ID. This is more efficient than polling the API.
//
// On timeout, it fetches and logs the current job status for debugging.
// Returns the final job response after completion.
func waitForJobCompletion(t *testing.T, ctx context.Context, jobID string, timeout time.Duration) *model.JobResponse {
	t.Helper()

	backend, err := queuekit.NewAMQPConsumerBuilder(amqpURL).
		FromQueue(e2eshared.NotificationsQueue).
		WithAutoAck(true).
		Build()
	require.NoError(t, err, "create AMQP consumer")
	defer backend.Close()

	consumer := queuekit.NewConsumer[e2eshared.JobNotification](t, backend).
		WithMatcher(queuekit.MatchJSONField("jobId", jobID)).
		WithTimeout(timeout).
		Build()
	defer consumer.Close()

	msg, err := consumer.WaitForMessage(ctx)
	if err != nil {
		// On timeout, fetch job status for debugging
		jobResult, jobErr := apiClient.GetJob(ctx, jobID)
		if jobErr == nil {
			t.Logf("Job status after timeout: status=%s, resultPath=%s", jobResult.Status, jobResult.ResultPath)
		}
		require.NoError(t, err, "wait for job completion")
	}

	queuekit.AssertMessage(t, msg).
		PayloadSatisfies("job ID matches", func(n e2eshared.JobNotification) bool {
			return n.JobID == jobID
		})

	// Fetch final job status
	jobResult, err := apiClient.GetJob(ctx, jobID)
	require.NoError(t, err, "get job status")

	return jobResult
}

// verifyAPIHealthy makes health check requests to the Manager API and records metrics.
// It performs numRequests sequential requests to ListConnections and records success rate,
// latency, and any errors in the provided ChaosMetrics collector.
func verifyAPIHealthy(t *testing.T, ctx context.Context, metrics *metricskit.ChaosMetrics, numRequests int) {
	t.Helper()

	for i := 0; i < numRequests; i++ {
		start := time.Now()
		_, err := apiClient.ListConnections(ctx, e2eshared.ListConnectionsParams{Limit: 10})
		latency := time.Since(start)

		success := err == nil
		timeout := err != nil && strings.Contains(err.Error(), "timeout")
		metrics.RecordRequest(success, timeout, latency)

		if err != nil {
			metrics.RecordError(err.Error())
		}
	}
}

// runRequestsUnderChaos executes API requests while chaos conditions are active and records metrics.
// It performs numRequests sequential requests with the given interval between them.
// Each request's success, timeout status, and latency are recorded in the ChaosMetrics collector.
func runRequestsUnderChaos(t *testing.T, ctx context.Context, metrics *metricskit.ChaosMetrics, numRequests int, interval time.Duration) {
	t.Helper()

	for i := 0; i < numRequests; i++ {
		start := time.Now()
		_, err := apiClient.ListConnections(ctx, e2eshared.ListConnectionsParams{Limit: 10})
		latency := time.Since(start)

		success := err == nil
		timeout := err != nil && (strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded"))
		metrics.RecordRequest(success, timeout, latency)

		if err != nil {
			metrics.RecordError(err.Error())
		}

		if interval > 0 && i < numRequests-1 {
			time.Sleep(interval)
		}
	}
}

// assertSLOs verifies that the recorded metrics meet the specified SLO thresholds.
// If any SLO is violated, it logs the full metrics report and assertion summary,
// then fails the test. This is used to validate system behavior under chaos conditions.
func assertSLOs(t *testing.T, metrics *metricskit.ChaosMetrics, successRate float64, p99Latency time.Duration) {
	t.Helper()

	assertions := metricskit.Assert(metrics).
		SuccessRateAbove(successRate).
		P99Below(p99Latency)

	if assertions.Failed() {
		t.Log(metricskit.Report(metrics).String())
		t.Log(assertions.Summary())
		t.Fail()
	}
}

// logChaosReport logs the complete chaos test metrics report to the test output.
// The report includes success rate, latency percentiles, error counts, and other statistics.
func logChaosReport(t *testing.T, metrics *metricskit.ChaosMetrics) {
	t.Helper()
	t.Log(metricskit.Report(metrics).String())
}

// =============================================================================
// CHAOS INJECTION HELPERS
// =============================================================================

// injectLatency adds network latency to the specified Toxiproxy proxy.
// The latency parameter specifies the base delay, and jitter adds randomness.
// It removes any existing toxics first to avoid conflicts when running tests in parallel.
//
// Returns a cleanup function that removes the latency toxic when called.
// The cleanup function should be deferred immediately after calling this function.
func injectLatency(t *testing.T, ctx context.Context, proxyName string, latency, jitter time.Duration) func() {
	t.Helper()

	chaos := suite.Chaos()
	require.NotNil(t, chaos, "chaos interface should be available")

	// Remove any existing toxics first to avoid conflicts
	_ = chaos.RemoveAllToxics(ctx, proxyName)

	err := chaos.AddLatency(ctx, proxyName, latency, jitter)
	require.NoError(t, err, "add latency to %s", proxyName)

	t.Logf("Injected latency %v (jitter %v) to %s", latency, jitter, proxyName)

	return func() {
		if err := chaos.RemoveAllToxics(ctx, proxyName); err != nil {
			t.Logf("Warning: failed to remove toxics from %s: %v", proxyName, err)
		}
	}
}

// injectTimeout adds a timeout toxic to the specified Toxiproxy proxy.
// The timeout causes connections to hang without receiving data, simulating
// network timeouts and unresponsive services.
// It removes any existing toxics first to avoid conflicts when running tests in parallel.
//
// Returns a cleanup function that removes the timeout toxic when called.
func injectTimeout(t *testing.T, ctx context.Context, proxyName string, timeout time.Duration) func() {
	t.Helper()

	chaos := suite.Chaos()
	require.NotNil(t, chaos, "chaos interface should be available")

	// Remove any existing toxics first to avoid conflicts
	_ = chaos.RemoveAllToxics(ctx, proxyName)

	err := chaos.AddTimeout(ctx, proxyName, timeout)
	require.NoError(t, err, "add timeout to %s", proxyName)

	t.Logf("Injected timeout %v to %s", timeout, proxyName)

	return func() {
		if err := chaos.RemoveAllToxics(ctx, proxyName); err != nil {
			t.Logf("Warning: failed to remove toxics from %s: %v", proxyName, err)
		}
	}
}

// injectBandwidthLimit adds bandwidth throttling to the specified Toxiproxy proxy.
// The rateKBps parameter specifies the maximum throughput in kilobytes per second.
// This simulates degraded network conditions or congested links.
// It removes any existing toxics first to avoid conflicts when running tests in parallel.
//
// Returns a cleanup function that removes the bandwidth toxic when called.
func injectBandwidthLimit(t *testing.T, ctx context.Context, proxyName string, rateKBps int64) func() {
	t.Helper()

	chaos := suite.Chaos()
	require.NotNil(t, chaos, "chaos interface should be available")

	// Remove any existing toxics first to avoid conflicts
	_ = chaos.RemoveAllToxics(ctx, proxyName)

	err := chaos.AddBandwidth(ctx, proxyName, rateKBps)
	require.NoError(t, err, "add bandwidth limit to %s", proxyName)

	t.Logf("Injected bandwidth limit %d KB/s to %s", rateKBps, proxyName)

	return func() {
		if err := chaos.RemoveAllToxics(ctx, proxyName); err != nil {
			t.Logf("Warning: failed to remove toxics from %s: %v", proxyName, err)
		}
	}
}

// cutConnection completely cuts the network connection through the specified proxy.
// This simulates a total network partition or service outage.
// It removes any existing toxics first to avoid conflicts when running tests in parallel.
//
// Returns a cleanup function that restores the connection when called.
func cutConnection(t *testing.T, ctx context.Context, proxyName string) func() {
	t.Helper()

	chaos := suite.Chaos()
	require.NotNil(t, chaos, "chaos interface should be available")

	// Remove any existing toxics first to avoid conflicts
	_ = chaos.RemoveAllToxics(ctx, proxyName)

	err := chaos.CutConnection(ctx, proxyName)
	require.NoError(t, err, "cut connection to %s", proxyName)

	t.Logf("Cut connection to %s", proxyName)

	return func() {
		if err := chaos.RemoveAllToxics(ctx, proxyName); err != nil {
			t.Logf("Warning: failed to remove toxics from %s: %v", proxyName, err)
		}
	}
}

// restoreConnection removes all toxics from a proxy, restoring normal network connectivity.
// This is used after chaos injection to verify system recovery behavior.
func restoreConnection(t *testing.T, ctx context.Context, proxyName string) {
	t.Helper()

	chaos := suite.Chaos()
	require.NotNil(t, chaos, "chaos interface should be available")

	err := chaos.RemoveAllToxics(ctx, proxyName)
	require.NoError(t, err, "restore connection to %s", proxyName)

	t.Logf("Restored connection to %s", proxyName)
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// waitForRecovery waits for the system to recover after chaos conditions are removed.
// It repeatedly calls the checkFn function (every 500ms) until it returns true,
// indicating successful recovery.
//
// Returns the actual recovery time duration. Fails the test if:
//   - The system doesn't recover within the specified timeout
//   - The context is cancelled
func waitForRecovery(t *testing.T, ctx context.Context, checkFn func() bool, timeout time.Duration) time.Duration {
	t.Helper()

	start := time.Now()
	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("System did not recover within %v", timeout)
			return timeout
		case <-ticker.C:
			if checkFn() {
				recoveryTime := time.Since(start)
				t.Logf("System recovered in %v", recoveryTime)
				return recoveryTime
			}
		case <-ctx.Done():
			t.Fatalf("Context cancelled while waiting for recovery")
			return time.Since(start)
		}
	}
}

// isAPIHealthy checks if the Manager API is responding correctly.
// It makes a simple ListConnections request and returns true if successful.
// This is used as the checkFn for waitForRecovery.
func isAPIHealthy(ctx context.Context) bool {
	_, err := apiClient.ListConnections(ctx, e2eshared.ListConnectionsParams{Limit: 1})
	return err == nil
}

// waitForJobCompletionPolling waits for a job to complete using API polling instead of queuekit.
// This is useful when RabbitMQ is under chaos or recovering from chaos, since the queue-based
// notification mechanism may not be reliable in those conditions.
//
// Polls the job status every 2 seconds until the job reaches a terminal state
// (not pending or processing). Returns the final job response.
// Fails the test if the job doesn't complete within the timeout.
func waitForJobCompletionPolling(t *testing.T, ctx context.Context, jobID string, timeout time.Duration) *model.JobResponse {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			t.Fatalf("Context cancelled while waiting for job %s", jobID)
			return nil
		case <-ticker.C:
			job, err := apiClient.GetJob(ctx, jobID)
			if err != nil {
				t.Logf("Error getting job status: %v", err)
				continue
			}
			t.Logf("Job %s status: %s", jobID, job.Status)
			if job.Status != e2eshared.JobStatusPending && job.Status != e2eshared.JobStatusProcessing {
				return job
			}
		}
	}

	// Timeout - get final status for debugging
	job, err := apiClient.GetJob(ctx, jobID)
	if err != nil {
		t.Fatalf("Job %s did not complete within timeout and failed to get status: %v", jobID, err)
		return nil
	}
	t.Fatalf("Job %s did not complete within timeout. Final status: %s", jobID, job.Status)
	return job
}
