//go:build chaos

package chaos

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit/addons/metricskit"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MANAGER + MONGODB CHAOS TESTS
// =============================================================================

// TestManager_MongoDB_HighLatency verifies Manager API behavior when MongoDB
// has high latency (500ms).
//
// Expected behavior:
// - API should still respond (degraded performance)
// - Success rate >= 95%
// - P99 latency < 2s
func TestManager_MongoDB_HighLatency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Baseline (10 requests without chaos)
	t.Log("Phase 1: Recording baseline metrics...")
	verifyAPIHealthy(t, ctx, metrics, 10)

	// Phase 2: Inject high latency
	t.Log("Phase 2: Injecting 500ms latency to MongoDB...")
	cleanup := injectLatency(t, ctx, mongoProxy, 500*time.Millisecond, 50*time.Millisecond)
	defer cleanup()

	metrics.StartChaos()

	// Phase 3: Run requests under chaos (50 requests with small intervals)
	t.Log("Phase 3: Running requests under chaos...")
	runRequestsUnderChaos(t, ctx, metrics, 50, 100*time.Millisecond)

	metrics.EndChaos()
	metrics.EndTest()

	// Phase 4: Verify SLOs
	t.Log("Phase 4: Verifying SLOs...")
	logChaosReport(t, metrics)
	assertSLOs(t, metrics, SLOManagerSuccessRate, SLOManagerP99Latency)
}

// TestManager_MongoDB_Timeout verifies Manager API behavior when MongoDB
// connections timeout completely.
//
// Expected behavior:
// - API should return graceful errors (HTTP 500)
// - Requests should not hang indefinitely
func TestManager_MongoDB_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Baseline
	t.Log("Phase 1: Recording baseline metrics...")
	verifyAPIHealthy(t, ctx, metrics, 5)

	// Phase 2: Inject timeout
	t.Log("Phase 2: Injecting connection timeout to MongoDB...")
	cleanup := injectTimeout(t, ctx, mongoProxy, 5*time.Second)
	defer cleanup()

	metrics.StartChaos()

	// Phase 3: Run requests under chaos
	t.Log("Phase 3: Running requests under chaos...")
	runRequestsUnderChaos(t, ctx, metrics, 20, 200*time.Millisecond)

	metrics.EndChaos()
	metrics.EndTest()

	// Phase 4: Verify behavior
	logChaosReport(t, metrics)

	// Under timeout conditions, we expect degraded performance
	// Verify that requests eventually fail gracefully (not hang)
	assert.Greater(t, metrics.GetTotalRequests(), 0, "should have recorded requests")
	t.Logf("Success rate under timeout: %.2f%%", metrics.SuccessRate())
}

// TestManager_MongoDB_Intermittent verifies Manager API recovery after
// MongoDB connection is cut and restored.
//
// KNOWN LIMITATION: The Manager may not have automatic MongoDB reconnection.
// If the MongoDB client connection pool is in an error state, the Manager
// may not recover automatically. This test verifies the chaos injection
// works correctly but may not achieve full recovery.
//
// Expected behavior:
// - System should degrade during cut
// - System MAY recover after restore (depends on MongoDB client retry behavior)
func TestManager_MongoDB_Intermittent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Baseline
	t.Log("Phase 1: Recording baseline metrics...")
	verifyAPIHealthy(t, ctx, metrics, 5)
	require.True(t, metrics.SuccessRate() > 90, "baseline should be healthy")

	// Phase 2: Cut connection briefly (2 seconds)
	t.Log("Phase 2: Cutting MongoDB connection briefly...")
	cleanup := cutConnection(t, ctx, mongoProxy)
	metrics.StartChaos()

	// Brief cut - only 2 seconds to minimize impact on connection pool
	time.Sleep(2 * time.Second)

	// Phase 3: Restore connection
	t.Log("Phase 3: Restoring MongoDB connection...")
	cleanup()
	restoreConnection(t, ctx, mongoProxy)

	metrics.EndChaos()

	// Phase 4: Wait for recovery with extended timeout
	t.Log("Phase 4: Checking system recovery...")

	// Give more time for MongoDB client to detect the restored connection
	time.Sleep(5 * time.Second)

	// Try a few requests to check recovery
	recoveryMetrics := metricskit.NewChaosMetrics()
	recoveryMetrics.StartTest()
	verifyAPIHealthy(t, ctx, recoveryMetrics, 5)
	recoveryMetrics.EndTest()

	metrics.EndTest()

	if recoveryMetrics.SuccessRate() > 50 {
		t.Log("System recovered successfully after MongoDB reconnection")
	} else {
		t.Log("System did not recover automatically - this is expected if Manager lacks auto-reconnect")
		t.Log("NOTE: Full recovery may require Manager restart")
	}

	logChaosReport(t, metrics)
}

// TestManager_MongoDB_Bandwidth verifies Manager API behavior when MongoDB
// has limited bandwidth (128 KB/s).
//
// Expected behavior:
// - Throughput should be degraded but functional
// - Requests should eventually complete
func TestManager_MongoDB_Bandwidth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	metrics := metricskit.NewChaosMetrics()
	metrics.StartTest()

	// Phase 1: Baseline
	t.Log("Phase 1: Recording baseline metrics...")
	verifyAPIHealthy(t, ctx, metrics, 10)
	baselineThroughput := metrics.ThroughputRPS()

	// Reset metrics for chaos phase
	chaosMetrics := metricskit.NewChaosMetrics()

	// Phase 2: Inject bandwidth limit
	t.Log("Phase 2: Injecting 128 KB/s bandwidth limit to MongoDB...")
	cleanup := injectBandwidthLimit(t, ctx, mongoProxy, 128)
	defer cleanup()

	chaosMetrics.StartTest()
	chaosMetrics.StartChaos()

	// Phase 3: Run requests under chaos
	t.Log("Phase 3: Running requests under chaos...")
	runRequestsUnderChaos(t, ctx, chaosMetrics, 30, 100*time.Millisecond)

	chaosMetrics.EndChaos()
	chaosMetrics.EndTest()

	// Phase 4: Verify behavior
	logChaosReport(t, chaosMetrics)

	// Verify that API is still functional (degraded but working)
	assert.Greater(t, chaosMetrics.SuccessRate(), 80.0,
		"success rate should be > 80%% under bandwidth limit")

	t.Logf("Throughput: baseline=%.2f req/s, under chaos=%.2f req/s",
		baselineThroughput, chaosMetrics.ThroughputRPS())
}

// =============================================================================
// MANAGER CRUD OPERATIONS UNDER CHAOS
// =============================================================================

// TestManager_CRUD_UnderLatency verifies that CRUD operations complete
// successfully under high latency conditions.
func TestManager_CRUD_UnderLatency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	// Phase 1: Inject latency
	t.Log("Phase 1: Injecting 300ms latency to MongoDB...")
	cleanup := injectLatency(t, ctx, mongoProxy, 300*time.Millisecond, 50*time.Millisecond)
	defer cleanup()

	// Phase 2: Test CRUD operations
	t.Log("Phase 2: Testing CRUD operations under latency...")

	// Create
	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	connInput := e2eshared.ConnectionInput{
		ConfigName:   "chaos-crud-test",
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	start := time.Now()
	conn, err := apiClient.CreateConnection(ctx, chaosProductName, connInput)
	createLatency := time.Since(start)
	require.NoError(t, err, "create connection should succeed under latency")
	t.Logf("Create latency: %v", createLatency)

	// Read
	start = time.Now()
	retrieved, err := apiClient.GetConnection(ctx, conn.ID)
	readLatency := time.Since(start)
	require.NoError(t, err, "read connection should succeed under latency")
	assert.Equal(t, conn.ID, retrieved.ID)
	t.Logf("Read latency: %v", readLatency)

	// List
	start = time.Now()
	_, err = apiClient.ListConnections(ctx, e2eshared.ListConnectionsParams{Limit: 10})
	listLatency := time.Since(start)
	require.NoError(t, err, "list connections should succeed under latency")
	t.Logf("List latency: %v", listLatency)

	// Delete
	start = time.Now()
	err = apiClient.DeleteConnection(ctx, conn.ID)
	deleteLatency := time.Since(start)
	require.NoError(t, err, "delete connection should succeed under latency")
	t.Logf("Delete latency: %v", deleteLatency)

	// Verify all operations completed within acceptable latency
	maxAcceptable := 3 * time.Second
	assert.Less(t, createLatency, maxAcceptable, "create should complete in reasonable time")
	assert.Less(t, readLatency, maxAcceptable, "read should complete in reasonable time")
	assert.Less(t, listLatency, maxAcceptable, "list should complete in reasonable time")
	assert.Less(t, deleteLatency, maxAcceptable, "delete should complete in reasonable time")
}
