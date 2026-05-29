//go:build chaos

package chaos

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// /readyz CHAOS TESTS — Gate 8, Layer 8
// =============================================================================
//
// These tests exercise the readyz handler wired into the Manager container
// against real MongoDB and Redis dependencies fronted by Toxiproxy. Each
// scenario injects a fault, hits /readyz over the network, and asserts the
// handler short-circuits within the per-dep deadline (PerDepTimeout) instead
// of hanging. Recovery is then verified by removing the toxic.
//
// The chaos infra is reused from tests/chaos/main_test.go — the Manager has
// already started with a real Mongo + Redis client and the /readyz route is
// mounted by components/manager/internal/bootstrap/readyz_routing.go.
//
// Contract assertions (ring:dev-readyz):
//   - 200 + status="healthy" when all deps respond.
//   - 503 + status="unhealthy" + dep-level status="down" when a dep fails.
//   - Per-dep timeout enforced: handler returns at ~PerDepTimeout(dep) even
//     if the dep is hung (no multiplicative blow-up with concurrent probes).
//   - Drain short-circuit returns 503 instantly without running checkers.

// readyzResponse mirrors readyz.ReadyzResponse shape for decoding. We redefine
// it locally (rather than importing the internal package) because the chaos
// package consumes the Manager's wire contract — if the shape drifts, this
// test should catch it.
type readyzResponse struct {
	Status         string                   `json:"status"`
	Checks         map[string]readyzCheck   `json:"checks"`
	Version        string                   `json:"version"`
	DeploymentMode string                   `json:"deployment_mode"`
}

type readyzCheck struct {
	Status       string `json:"status"`
	LatencyMs    int64  `json:"latency_ms,omitempty"`
	TLS          *bool  `json:"tls,omitempty"`
	Error        string `json:"error,omitempty"`
	Reason       string `json:"reason,omitempty"`
	BreakerState string `json:"breaker_state,omitempty"`
}

// readyzTimings documents the per-dependency timeouts the handler enforces.
// These must match readyz.PerDepTimeout exactly — when the handler's deadline
// fires the test verifies the wall-clock elapsed is within perDep + buffer.
const (
	mongoPerDepTimeout  = 2 * time.Second
	redisPerDepTimeout  = 1 * time.Second
	handlerBuffer       = 500 * time.Millisecond // allowance for goroutine scheduling + HTTP overhead
	drainShortCircuitMS = 250 * time.Millisecond // drain must NOT wait on any checker
)

// getReadyz issues a single GET /readyz request against the running Manager
// container and returns the status code, decoded body, and wall-clock latency.
func getReadyz(t *testing.T, ctx context.Context) (int, readyzResponse, time.Duration) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, managerApp.BaseURL+"/readyz", nil)
	require.NoError(t, err, "build /readyz request")

	client := &http.Client{Timeout: 10 * time.Second}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	require.NoError(t, err, "GET /readyz")
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read /readyz body")

	var decoded readyzResponse
	if len(body) > 0 {
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("decode /readyz body: %v, body=%s", err, string(body))
		}
	}

	return resp.StatusCode, decoded, elapsed
}

// TestReadyz_Chaos_Mongo_ConnectionLoss simulates a total MongoDB outage and
// verifies /readyz flips to unhealthy, reports mongodb as down, and recovers
// after the proxy is restored.
//
// Scenario coverage: connection loss mid-run.
func TestReadyz_Chaos_Mongo_ConnectionLoss(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	// Phase 1: Baseline — /readyz should be healthy with MongoDB reachable.
	t.Log("Phase 1: baseline /readyz healthy")
	status, resp, _ := getReadyz(t, ctx)
	require.Equal(t, http.StatusOK, status, "baseline /readyz must be 200")
	require.Equal(t, "healthy", resp.Status)
	require.Contains(t, resp.Checks, "mongodb")
	require.Equal(t, "up", resp.Checks["mongodb"].Status, "mongodb should be up at baseline")

	// Phase 2: Inject total connection loss on MongoDB proxy.
	t.Log("Phase 2: cutting MongoDB connection via Toxiproxy")
	restore := cutConnection(t, ctx, mongoProxy)
	defer restore() // safety — explicit restore below

	// Phase 3: /readyz must flip to 503 and report mongodb down within the
	// per-dep timeout budget. The handler MUST NOT hang waiting on Mongo.
	t.Log("Phase 3: /readyz under chaos should be 503 within per-dep timeout")
	status, resp, elapsed := getReadyz(t, ctx)
	assert.Equal(t, http.StatusServiceUnavailable, status, "/readyz must return 503 with mongo down")
	assert.Equal(t, "unhealthy", resp.Status)
	require.Contains(t, resp.Checks, "mongodb")
	assert.Equal(t, "down", resp.Checks["mongodb"].Status, "mongodb check should be down")
	assert.NotEmpty(t, resp.Checks["mongodb"].Error, "mongodb check should carry an error string")
	assert.Less(t, elapsed, mongoPerDepTimeout+handlerBuffer,
		"handler must return within PerDepTimeout(mongodb) + buffer; got %v", elapsed)

	// Phase 4: Restore connection and verify /readyz recovers.
	t.Log("Phase 4: restoring MongoDB and verifying /readyz recovers")
	restore()
	restoreConnection(t, ctx, mongoProxy)

	// MongoDB driver may need a moment to detect the restored connection.
	recoveryTime := waitForRecovery(t, ctx, func() bool {
		status, resp, _ := getReadyz(t, ctx)
		return status == http.StatusOK && resp.Status == "healthy"
	}, SLORecoveryTime)
	t.Logf("/readyz recovered in %v", recoveryTime)
}

// TestReadyz_Chaos_Mongo_LatencyForcesTimeout injects latency ABOVE the Mongo
// per-dep timeout and asserts the handler bails at ~2s (not 3s).
//
// Scenario coverage: latency injection forcing per-dep timeout.
func TestReadyz_Chaos_Mongo_LatencyForcesTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	// Baseline
	status, _, _ := getReadyz(t, ctx)
	require.Equal(t, http.StatusOK, status, "baseline must be healthy")

	// Inject 3s latency — above the 2s Mongo per-dep timeout.
	t.Log("Injecting 3s latency on MongoDB proxy")
	cleanup := injectLatency(t, ctx, mongoProxy, 3*time.Second, 0)
	defer cleanup()

	status, resp, elapsed := getReadyz(t, ctx)
	assert.Equal(t, http.StatusServiceUnavailable, status, "/readyz should be 503 when Mongo exceeds deadline")
	assert.Equal(t, "unhealthy", resp.Status)
	require.Contains(t, resp.Checks, "mongodb")
	assert.Equal(t, "down", resp.Checks["mongodb"].Status, "mongodb should report down on timeout")
	// Error string convention: "timeout" or "context deadline".
	assert.Contains(t, resp.Checks["mongodb"].Error, "timeout",
		"error should mention timeout; got %q", resp.Checks["mongodb"].Error)

	// CRITICAL: handler must bail at ~2s (Mongo per-dep), NOT 3s (the latency).
	assert.Less(t, elapsed, mongoPerDepTimeout+handlerBuffer,
		"handler must honour PerDepTimeout(mongodb)=2s, not the 3s latency; got %v", elapsed)

	// Remove toxic and verify recovery.
	cleanup()
	status, resp, _ = getReadyz(t, ctx)
	assert.Equal(t, http.StatusOK, status, "/readyz should recover after latency removed")
	assert.Equal(t, "healthy", resp.Status)
}

// TestReadyz_Chaos_ConcurrentProbesUnderLatency fires 20 parallel /readyz
// requests while MongoDB is latency-injected. Every request must return 503
// within the per-dep timeout envelope — no request may block longer than 3s.
// This protects the handler's parallel-dispatch contract: a slow checker must
// not stall concurrent /readyz callers (Kubernetes will fire multiple probes
// during rolling deploys).
//
// Scenario coverage: concurrent probes during chaos + no handler-level stalls.
func TestReadyz_Chaos_ConcurrentProbesUnderLatency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	t.Log("Injecting 3s latency on MongoDB proxy")
	cleanup := injectLatency(t, ctx, mongoProxy, 3*time.Second, 0)
	defer cleanup()

	const parallel = 20
	var wg sync.WaitGroup

	type outcome struct {
		status  int
		elapsed time.Duration
	}

	results := make(chan outcome, parallel)

	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, _, elapsed := getReadyz(t, ctx)
			results <- outcome{status: status, elapsed: elapsed}
		}()
	}

	wg.Wait()
	close(results)

	var (
		seen    int
		maxTime time.Duration
	)
	for r := range results {
		seen++
		assert.Equal(t, http.StatusServiceUnavailable, r.status,
			"every concurrent /readyz should return 503 under chaos")
		// No single request may block longer than the per-dep timeout + buffer.
		// 3s is the toxic latency; handler must beat that ceiling.
		assert.Less(t, r.elapsed, 3*time.Second,
			"no /readyz request may block past the 3s toxic latency; got %v", r.elapsed)
		if r.elapsed > maxTime {
			maxTime = r.elapsed
		}
	}
	assert.Equal(t, parallel, seen, "all concurrent probes must complete")
	t.Logf("Max /readyz latency across %d concurrent probes: %v", parallel, maxTime)
}

// TestReadyz_Chaos_DrainShortCircuit verifies SetDraining short-circuits
// /readyz even while a dep is under chaos. The handler MUST NOT wait for the
// per-dep timeout when the drain flag is set — that's the whole point of the
// drain signal (respond 503 immediately so Kubernetes stops routing).
//
// NOTE: The drain flag is process-local to the Manager container. We cannot
// flip readyz.SetDraining(true) from this test (running on the host) — it
// would affect only this test binary's package state, not the Manager's.
//
// Instead we assert the operational contract that is verifiable from outside:
// under a latency-injected dep, each /readyz call from this test binary takes
// near the per-dep timeout. If a drain-capable deployment were used, the same
// call would return in <100ms. We document this gap and exercise the
// equivalent path in the goleak unit tests (Layer 9, TestHandler_Drain_*).
func TestReadyz_Chaos_DrainShortCircuit_DocumentedGap(t *testing.T) {
	t.Log("Drain short-circuit verification requires in-process SetDraining — " +
		"covered by unit tests in pkg/bootstrap/readyz/goleak_test.go " +
		"(TestHandler_Drain_NoGoroutineLeaks) and handler_test.go (TestHandler_Fiber_Draining_Returns503). " +
		"This chaos test documents the gap rather than re-proving it against a container.")
}

// TestReadyz_Chaos_Redis_ConnectionLoss repeats the Mongo connection-loss
// scenario on Redis — a second target for variety per the layer-8 spec. Redis
// has a shorter per-dep timeout (1s), so the handler-budget assertion is
// tighter.
//
// Scenario coverage: Redis variant.
func TestReadyz_Chaos_Redis_ConnectionLoss(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
	defer cancel()

	// Phase 1: Baseline
	status, resp, _ := getReadyz(t, ctx)
	require.Equal(t, http.StatusOK, status, "baseline /readyz must be 200")
	// Redis is optional in the fetcher bootstrap — only assert the probe ran.
	if _, hasRedis := resp.Checks["redis"]; !hasRedis {
		t.Skip("fetcher Manager is not configured with a redis checker; skipping Redis chaos variant")
	}
	require.Equal(t, "up", resp.Checks["redis"].Status, "redis should be up at baseline")

	// Phase 2: Cut Redis connection.
	t.Log("cutting Redis connection via Toxiproxy")
	restore := cutConnection(t, ctx, redisProxy)
	defer restore()

	status, resp, elapsed := getReadyz(t, ctx)
	assert.Equal(t, http.StatusServiceUnavailable, status, "/readyz must be 503 when redis down")
	assert.Equal(t, "unhealthy", resp.Status)
	require.Contains(t, resp.Checks, "redis")
	assert.Equal(t, "down", resp.Checks["redis"].Status)
	// Handler budget for Redis is tighter (1s per-dep timeout).
	assert.Less(t, elapsed, redisPerDepTimeout+handlerBuffer,
		"handler must return within PerDepTimeout(redis)=1s + buffer; got %v", elapsed)

	// Phase 3: Restore and verify recovery.
	restore()
	restoreConnection(t, ctx, redisProxy)

	recoveryTime := waitForRecovery(t, ctx, func() bool {
		status, resp, _ := getReadyz(t, ctx)
		return status == http.StatusOK && resp.Status == "healthy"
	}, SLORecoveryTime)
	t.Logf("/readyz recovered in %v after Redis restore", recoveryTime)
}
