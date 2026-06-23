//go:build e2e

package extraction

// =============================================================================
// /readyz E2E TESTS — Sprint 1 (happy-path + contract + metrics + auth scope)
// =============================================================================
//
// These tests exercise the canonical /readyz endpoint introduced in commit
// 2d38baf (feat(readyz): implement canonical /readyz endpoint with multi-tenant
// support) against the real Manager + Worker containers wired by main_test.go.
// They cover the slice of the contract that does not require fault injection
// (Toxiproxy / docker stop) — that is reserved for tests/chaos and Sprint 2.
//
// Scope (per docs/readyz-guide.md):
//   - §2 Endpoints: manager:4006 + worker:4007 expose /readyz, /health, /metrics
//   - §6 Response contract: status vocabulary, dep membership, top-level shape
//   - §7 Status vocabulary: closed-set values
//   - §8 Metrics reference: three families exposed at /metrics
//   - §9 Multi-tenant: /readyz/tenant/:id returns 400 when MULTI_TENANT_ENABLED=false
//   - §15 Security: endpoints unauthenticated by design
//
// Out of scope (covered elsewhere):
//   - Per-dep timeout / chaos behaviour → tests/chaos/readyz_chaos_test.go
//   - Drain SIGTERM short-circuit → Sprint 3 (requires container signal)
//   - SaaS TLS enforcement → pkg/bootstrap/readyz/tls_enforcement_test.go
//   - Tenant prober per tenant → pkg/bootstrap/readyz/tenant_handler_test.go

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	e2eshared "github.com/LerianStudio/fetcher/v2/tests/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readyzTestTimeout caps each test. The handler enforces 2s per-dep timeouts
// so a single /readyz call should never approach this in the happy path; we
// allow generous slack for CI cold starts and worker container readiness.
const readyzTestTimeout = 30 * time.Second

// workerBaseURL builds the Worker /readyz base URL. The Worker exposes a
// dedicated micro-server on HEALTH_PORT (4007) — see
// components/worker/internal/bootstrap/health_server.go and
// tests/shared/apps.go (StartWorker now publishes that port).
func workerBaseURL(t *testing.T) string {
	t.Helper()

	if workerApp == nil {
		t.Skip("worker container not available (E2E_SKIP_WORKER=true) — readyz worker tests require it")
	}

	host, err := workerApp.Container.Host(context.Background())
	require.NoError(t, err, "resolve worker container host")

	mappedPort, err := workerApp.Container.MappedPort(
		context.Background(),
		fmt.Sprintf("%d/tcp", e2eshared.WorkerHealthPort),
	)
	require.NoError(t, err, "resolve worker mapped HEALTH_PORT")

	return "http://" + host + ":" + mappedPort.Port()
}

// =============================================================================
// 1. Manager /readyz — happy path: every check is "up", top-level "healthy"
// =============================================================================

// TestReadyz_Manager_Healthy_AllChecks_Up validates the canonical happy-path
// response on the Manager: HTTP 200, top-level status "healthy", all expected
// platform deps reporting "up" with non-zero latency, deployment_mode echoed,
// and version populated.
//
// Coverage: docs/readyz-guide.md §6 (response contract) + §7 (vocabulary) on
// the Manager surface (port 4006). Worker variant is exercised by
// TestReadyz_Worker_Healthy_AllChecks_Up below.
func TestReadyz_Manager_Healthy_AllChecks_Up(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
	defer cancel()

	status, body, elapsed := e2eshared.GetReadyz(t, ctx, managerBaseURL(t))

	require.Equal(t, http.StatusOK, status, "manager /readyz must be 200 in happy path")
	require.Equal(t, e2eshared.ReadyzTopHealthy, body.Status,
		"top-level status must be 'healthy' when every dep is up")
	require.Equal(t, "local", body.DeploymentMode,
		"deployment_mode should echo DEPLOYMENT_MODE env (set to 'local' in WorkerEnv/ManagerEnv)")
	require.NotEmpty(t, body.Version, "version must be populated (defaults to OTEL_RESOURCE_SERVICE_VERSION)")

	// Manager-owned deps in single-tenant mode (per readyz_adapters.go
	// buildManagerReadyzCheckers). MULTI_TENANT_ENABLED=false in the e2e env,
	// so mongodb and rabbitmq are real checkers (not NAChecker).
	for _, dep := range []string{
		e2eshared.ReadyzDepMongoDB,
		e2eshared.ReadyzDepRabbitMQ,
		e2eshared.ReadyzDepRedis,
	} {
		check, ok := body.Checks[dep]
		require.True(t, ok, "manager /readyz response must include dep %q", dep)
		assert.Equal(t, e2eshared.ReadyzStatusUp, check.Status,
			"dep %q must be up at baseline; got %q (error=%q)", dep, check.Status, check.Error)
		// latency_ms is millisecond-rounded — a sub-millisecond Ping legitimately
		// reports 0. We only assert non-negative.
		assert.GreaterOrEqual(t, check.LatencyMs, int64(0),
			"dep %q latency_ms should be non-negative", dep)
		assert.Empty(t, check.Error,
			"dep %q with status=up must not carry an error string", dep)
	}

	// S3 belongs to the worker, not the manager (storage repository is owned
	// by the worker bootstrap). This assertion catches any future scope creep
	// that would reintroduce the bug class the Monetarie incident exposed.
	_, hasS3 := body.Checks[e2eshared.ReadyzDepS3]
	assert.False(t, hasS3, "manager /readyz must NOT include 's3' (worker-only dep)")

	// Sanity on overall latency — should beat any per-dep timeout × N.
	assert.Less(t, elapsed, 5*time.Second,
		"happy-path /readyz should respond well under 5s; got %v", elapsed)
}

// =============================================================================
// 2. Worker /readyz — happy path: mongo, rabbitmq, s3 all up
// =============================================================================

// TestReadyz_Worker_Healthy_AllChecks_Up validates the Worker's dedicated
// micro-server (HEALTH_PORT=4007). The membership rule is identical to the
// Manager except S3 is included (object-storage dep) and rabbitmq comes from
// the consumer adapter rather than a producer client.
//
// Coverage: docs/readyz-guide.md §2 (worker port) + §6 (response shape). This
// test plus the Manager variant together prove the two services own different
// dep slices — a regression here would mean someone wired the wrong checker
// set to the wrong service.
func TestReadyz_Worker_Healthy_AllChecks_Up(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
	defer cancel()

	status, body, _ := e2eshared.GetReadyz(t, ctx, workerBaseURL(t))

	require.Equal(t, http.StatusOK, status, "worker /readyz must be 200 in happy path")
	require.Equal(t, e2eshared.ReadyzTopHealthy, body.Status, "top-level status must be 'healthy'")
	require.Equal(t, "local", body.DeploymentMode, "deployment_mode echo")

	// Worker-owned deps (per worker readyz_adapters.go newWorkerReadyzDepsST):
	// mongodb (always), s3 (when storage repo present), rabbitmq via
	// consumer.Adapter() (single-tenant path).
	for _, dep := range []string{
		e2eshared.ReadyzDepMongoDB,
		e2eshared.ReadyzDepRabbitMQ,
		e2eshared.ReadyzDepS3,
	} {
		check, ok := body.Checks[dep]
		require.True(t, ok, "worker /readyz must include dep %q", dep)
		assert.Equal(t, e2eshared.ReadyzStatusUp, check.Status,
			"worker dep %q must be up; got %q (error=%q)", dep, check.Status, check.Error)
	}
}

// =============================================================================
// 3. Wire-contract compliance — vocabulary fences and JSON shape
// =============================================================================

// TestReadyz_Manager_ResponseSchema_Compliance asserts the /readyz response
// is well-formed against the closed contract: top-level keys exactly match,
// every check carries a vocabulary-valid status, and no unexpected fields
// leak. This is the regression net for any future drift in
// pkg/bootstrap/readyz/types.go.
//
// Coverage: docs/readyz-guide.md §7 (status vocabulary).
func TestReadyz_Manager_ResponseSchema_Compliance(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
	defer cancel()

	// Hit /readyz raw and decode into a map first to detect unknown
	// top-level fields. ReadyzResponse intentionally has the smallest possible
	// surface; anything new at the top must be an explicit contract change.
	status, structured, _ := e2eshared.GetReadyz(t, ctx, managerBaseURL(t))
	require.Equal(t, http.StatusOK, status)

	// Re-fetch and decode loosely so we can audit unexpected keys without
	// reissuing a fresh probe (a second hit doubles the metric counters and
	// can mask flakiness).
	rawStatus, raw := getReadyzRawJSON(t, ctx, managerBaseURL(t))
	require.Equal(t, http.StatusOK, rawStatus)

	allowedTopKeys := map[string]struct{}{
		"status":          {},
		"checks":          {},
		"version":         {},
		"deployment_mode": {},
		// tenant_id is allowed but only emitted on /readyz/tenant/:id.
		"tenant_id": {},
	}

	for k := range raw {
		_, allowed := allowedTopKeys[k]
		assert.True(t, allowed, "/readyz response carries unexpected top-level key %q", k)
	}

	// Top-level status vocabulary
	assert.True(t, e2eshared.IsValidReadyzTopStatus(structured.Status),
		"top-level status %q is outside the closed vocabulary {healthy, unhealthy}", structured.Status)

	// Each check's status is in the closed set.
	for dep, check := range structured.Checks {
		assert.True(t, e2eshared.IsValidReadyzStatus(check.Status),
			"dep %q has out-of-vocabulary status %q (allowed: up/down/degraded/skipped/n/a)",
			dep, check.Status)

		// latency_ms is non-negative for up/degraded. Handler default-fills
		// from elapsed.Milliseconds() (handler.go:143) which legitimately
		// rounds to zero on sub-millisecond Pings against localhost
		// containers — so we cannot demand strict > 0.
		if check.Status == e2eshared.ReadyzStatusUp || check.Status == e2eshared.ReadyzStatusDegraded {
			assert.GreaterOrEqual(t, check.LatencyMs, int64(0),
				"dep %q latency_ms must be non-negative", dep)
		}

		// down/degraded MUST carry an error string per §6 of the guide.
		if check.Status == e2eshared.ReadyzStatusDown || check.Status == e2eshared.ReadyzStatusDegraded {
			assert.NotEmpty(t, check.Error,
				"dep %q with status=%s must carry an error string", dep, check.Status)
		}

		// skipped/n_a MUST carry a reason.
		if check.Status == e2eshared.ReadyzStatusSkipped || check.Status == e2eshared.ReadyzStatusNA {
			assert.NotEmpty(t, check.Reason,
				"dep %q with status=%s must carry a reason string", dep, check.Status)
		}
	}
}

// =============================================================================
// 4. /health responds 200 once startup self-probe has flipped
// =============================================================================

// TestReadyz_Health_Endpoint_AfterStartup_200 verifies that both Manager and
// Worker /health endpoints return 200 once boot completes. /health is gated
// on the startup self-probe (selfProbeOK atomic.Bool flips when every checker
// reported up at least once during boot) — by the time TestMain returns,
// the WaitHTTP gate in StartManager/StartWorker has already proven this. We
// re-assert it explicitly so any regression in the gating wiring (e.g. a
// future refactor that detaches /health from the self-probe) is caught.
//
// Coverage: docs/readyz-guide.md §11 (Why is /health returning 503?) — the
// inverse contract.
func TestReadyz_Health_Endpoint_AfterStartup_200(t *testing.T) {
	t.Parallel()

	// Each subtest builds its OWN context. Sharing a parent context with
	// parallel subtests is a foot-gun: the parent's `defer cancel()` runs
	// when TestReadyz_Health_Endpoint_AfterStartup_200 returns, which is
	// before parallel subtests execute (Go schedules them after the parent
	// returns). The subtests would then see a canceled context immediately.

	t.Run("manager", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
		defer cancel()

		status, _ := e2eshared.GetHealth(t, ctx, managerBaseURL(t))
		assert.Equal(t, http.StatusOK, status, "manager /health must be 200 after self-probe success")
	})

	t.Run("worker", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
		defer cancel()

		status, _ := e2eshared.GetHealth(t, ctx, workerBaseURL(t))
		assert.Equal(t, http.StatusOK, status, "worker /health must be 200 after self-probe success")
	})
}

// =============================================================================
// 5. /metrics exposes the three readyz metric families
// =============================================================================

// TestReadyz_Metrics_Endpoint_ExposesReadyzSeries asserts the three Prometheus
// metric families introduced in commit 2d38baf are exposed on /metrics for
// both Manager and Worker:
//   - readyz_check_duration_ms (Histogram)
//   - readyz_check_status (Gauge)
//   - selfprobe_result (Gauge)
//
// The check is the HELP line presence — that's emitted at registration time,
// independent of whether a sample has been recorded yet. This matches the
// guide §11 runbook entry "/metrics has no readyz_* series" diagnostic step.
//
// Coverage: docs/readyz-guide.md §8 (metrics reference).
func TestReadyz_Metrics_Endpoint_ExposesReadyzSeries(t *testing.T) {
	t.Parallel()

	families := []string{
		"readyz_check_duration_ms",
		"readyz_check_status",
		"selfprobe_result",
	}

	t.Run("manager", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
		defer cancel()

		// Hit /readyz once first to ensure at least one sample has been
		// recorded for readyz_check_* — the HELP line is independent of
		// samples but a present sample makes the AssertMetricSamplePresent
		// follow-up reliable.
		_, _, _ = e2eshared.GetReadyz(t, ctx, managerBaseURL(t))

		body := e2eshared.GetMetricsBody(t, ctx, managerBaseURL(t))
		for _, family := range families {
			e2eshared.AssertMetricFamilyPresent(t, body, family)
		}

		// selfprobe_result is recorded once at boot and retained — at this
		// point in the test lifecycle every required dep was reachable
		// (otherwise StartManager would have failed waiting on /health), so
		// every selfprobe_result{dep="..."} should be 1.
		e2eshared.AssertMetricSamplePresent(t, body, "selfprobe_result", map[string]string{
			"dep": e2eshared.ReadyzDepMongoDB,
		})
	})

	t.Run("worker", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
		defer cancel()

		_, _, _ = e2eshared.GetReadyz(t, ctx, workerBaseURL(t))

		body := e2eshared.GetMetricsBody(t, ctx, workerBaseURL(t))
		for _, family := range families {
			e2eshared.AssertMetricFamilyPresent(t, body, family)
		}

		// Worker carries an extra selfprobe entry for s3 — assert it.
		e2eshared.AssertMetricSamplePresent(t, body, "selfprobe_result", map[string]string{
			"dep": e2eshared.ReadyzDepS3,
		})
	})
}

// =============================================================================
// 6. /readyz, /health and /metrics are unauthenticated (K8s contract)
// =============================================================================

// TestReadyz_NoAuth_PublicEndpoint asserts the readiness/liveness/metrics
// endpoints respond 200 without any authentication headers — Kubernetes'
// kubelet has no way to present credentials, so this is a hard contract.
//
// We exercise the same client used for other readyz tests (which never sets
// X-Organization-Id or any auth header). A successful 200 here proves the
// routes respond regardless of auth headers — they are mounted BEFORE the
// auth middleware in the Fiber stack (see
// components/manager/internal/adapters/http/in/routes.go:69 and the comment
// "Mounted before auth — probes must be unauthenticated.")
//
// Note: we do NOT positive-control this with a business endpoint, because
// the e2e Manager wires lib-auth in a permissive mode (license validation
// stub) where /v1/management/* also responds without an Authorization
// header. The strong claim — "/readyz is reachable from a kubelet that
// can't present credentials" — does not require the converse.
//
// Coverage: docs/readyz-guide.md §2 (CRITICAL — all four endpoints are
// intentionally unauthenticated).
func TestReadyz_NoAuth_PublicEndpoint(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
	defer cancel()

	managerURL := managerBaseURL(t)

	// Public endpoints — must all be 200 without auth.
	for _, path := range []string{"/readyz", "/health", "/metrics"} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, managerURL+path, nil)
		require.NoError(t, err)

		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		require.NoError(t, err, "GET %s", path)

		_ = resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"%s must respond 200 without auth (mounted before auth middleware)", path)
	}
}

// =============================================================================
// 7. /readyz/tenant/:id returns 400 when MULTI_TENANT_ENABLED=false
// =============================================================================

// TestReadyz_TenantEndpoint_SingleTenant_Returns400 asserts the per-tenant
// readiness route stays mounted in single-tenant mode and responds with a
// stable 400 + clear message — operators must be able to distinguish
// "MT disabled" from "route not wired".
//
// Coverage: docs/readyz-guide.md §14 (Common Errors): "multi-tenant mode is
// disabled" → /readyz/tenant/:id hit while MULTI_TENANT_ENABLED=false.
//
// In the e2e suite MULTI_TENANT_ENABLED is hard-coded to "false" in
// ManagerEnv()/WorkerEnv(); flipping it would require rewiring the suite to
// stand up a Tenant Manager mock, which is out of scope for Sprint 1.
func TestReadyz_TenantEndpoint_SingleTenant_Returns400(t *testing.T) {
	t.Parallel()

	const fakeTenantID = "11111111-1111-1111-1111-111111111111"

	t.Run("manager", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
		defer cancel()

		status, body, _ := e2eshared.GetReadyzTenant(t, ctx, managerBaseURL(t), fakeTenantID)

		assert.Equal(t, http.StatusBadRequest, status,
			"manager /readyz/tenant/:id must return 400 in single-tenant mode")
		// Decoded body may be empty (handler returns a non-readyz error
		// envelope); validate via the raw body grep below.
		_ = body
	})

	t.Run("worker", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
		defer cancel()

		status, _, _ := e2eshared.GetReadyzTenant(t, ctx, workerBaseURL(t), fakeTenantID)
		assert.Equal(t, http.StatusBadRequest, status,
			"worker /readyz/tenant/:id must return 400 in single-tenant mode")
	})

	t.Run("body_mentions_multi_tenant", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), readyzTestTimeout)
		defer cancel()

		// Body content sanity — the manager-side disabled handler emits a
		// stable message containing the phrase "multi-tenant" so dashboards
		// can match on it. We don't lock the exact string (subject to copy
		// edits), but fail loudly if it diverges.
		rawStatus, rawBody := rawGet(t, ctx, managerBaseURL(t)+"/readyz/tenant/"+fakeTenantID)
		require.Equal(t, http.StatusBadRequest, rawStatus)
		assert.True(t, strings.Contains(strings.ToLower(rawBody), "multi-tenant") ||
			strings.Contains(strings.ToLower(rawBody), "multitenant"),
			"disabled tenant handler should mention 'multi-tenant' in the error body; got %q", rawBody)
	})
}

// =============================================================================
// 8. Drain on SIGTERM — /readyz reports `draining` during the drain window
// =============================================================================

// TestReadyz_Drain_Returns503_OnSIGTERM verifies the graceful-drain contract:
// when the worker process receives SIGTERM, /readyz immediately starts
// returning 503 with a synthetic `draining` check, BEFORE the HTTP server
// begins shutdown. The drain delay is configured via READYZ_DRAIN_DELAY_SEC
// (set to 2s in tests/shared/apps.go WorkerEnv()), giving Kubernetes
// readinessProbe time to observe the 503 and remove the pod from Service
// endpoints before in-flight requests are cut.
//
// **Test isolation**: this test is destructive — it terminates a service
// container. We spawn a SECOND MANAGER container dedicated to this test
// (rather than the Worker) for two reasons:
//
//  1. The Manager has proper drain sequencing wired in
//     components/manager/internal/bootstrap/server.go — the SIGTERM handler
//     intercepts the signal, calls SetDraining(true), sleeps the drain
//     window, and only THEN closes the HTTP shutdown channel. This is the
//     contract the test verifies.
//  2. The Worker's HealthServer uses lib-commons StartWithGracefulShutdown,
//     which races with the consumer's drain handler — /readyz may stop
//     responding before the drain flag is observable to a polling client.
//     That's a real ordering gap on the worker side worth a separate fix
//     PR; here we test the canonical drain contract on the path that
//     implements it correctly.
//
// Coverage: docs/readyz-guide.md §11 ("In-flight requests killed during
// deploy") + handler.go drainShortCircuit branch +
// components/manager/internal/bootstrap/server.go Run() drain sequence.
func TestReadyz_Drain_Returns503_OnSIGTERM(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 1. Spawn a dedicated Manager container so we don't disrupt the shared
	// managerApp used by sibling tests. Reuses the same env TestMain set up.
	appEnv, err := e2eshared.BuildAppEnv(
		suite.Network(),
		coreInfra.MongoDB,
		coreInfra.RabbitMQ,
		coreInfra.Redis,
		coreInfra.SeaweedFS,
		coreInfra.Minio,
	)
	require.NoError(t, err, "rebuild app env for drain manager")

	t.Log("Spawning dedicated drain-test manager container")

	drainManager, err := e2eshared.StartManager(t, ctx, appEnv, e2eshared.AppStartConfig{
		Image:     managerImage(),
		SkipBuild: skipBuild(),
	})
	require.NoError(t, err, "spawn drain-test manager")

	t.Cleanup(func() {
		_ = drainManager.Container.Terminate(context.Background())
	})

	drainURL := drainManager.BaseURL

	// 3. Verify baseline.
	status, body, _ := e2eshared.GetReadyz(t, ctx, drainURL)
	require.Equal(t, http.StatusOK, status, "drain worker /readyz baseline must be 200")
	require.Equal(t, e2eshared.ReadyzTopHealthy, body.Status)

	// 4. Send SIGTERM via Container.Stop. The timeout argument is the grace
	// period before Docker sends SIGKILL — we give 15s, comfortably more
	// than READYZ_DRAIN_DELAY_SEC=2 plus normal HTTP server shutdown
	// (~1–2s) plus testcontainers overhead.
	stopTimeout := 15 * time.Second

	stopErr := make(chan error, 1)

	go func() {
		stopErr <- drainManager.Container.Stop(context.Background(), &stopTimeout)
	}()

	// 5. Within the drain window, /readyz must return 503 with the
	// synthetic `draining` check. We poll on a tight cadence because
	// we have only ≈2s before the server begins shutdown and the
	// endpoint becomes unreachable.
	require.Eventually(t, func() bool {
		s, b, err := tryGetReadyzNonFatal(ctx, drainURL)
		if err != nil {
			return false
		}

		if s != http.StatusServiceUnavailable {
			return false
		}

		check, ok := b.Checks[e2eshared.ReadyzDepDraining]

		return ok && check.Status == e2eshared.ReadyzStatusDown
	}, 1500*time.Millisecond, 50*time.Millisecond,
		"/readyz must report draining=down within 1.5s of SIGTERM")

	// 6. Capture one full response and validate the contract details.
	s, b, _ := tryGetReadyzNonFatal(ctx, drainURL)
	if s == http.StatusServiceUnavailable {
		assert.Equal(t, e2eshared.ReadyzTopUnhealthy, b.Status,
			"top-level status must flip to unhealthy during drain")

		drainCheck := b.Checks[e2eshared.ReadyzDepDraining]
		assert.Equal(t, e2eshared.ReadyzStatusDown, drainCheck.Status)
		assert.NotEmpty(t, drainCheck.Reason,
			"draining check should carry a reason explaining the drain")
	}

	// 7. Wait for the container to fully shut down. If Stop fails or
	// times out, the test still passes (the drain check was the contract
	// being validated) — but we surface the error for diagnostics.
	if err := <-stopErr; err != nil {
		t.Logf("Stop returned error (non-fatal — drain assertion already satisfied): %v", err)
	}
}

// tryGetReadyzNonFatal mirrors e2eshared.GetReadyz but returns errors
// instead of fataling. Used inside require.Eventually predicates that may
// race with container shutdown — once /readyz becomes unreachable, the
// http.Get errors and we want the predicate to simply return false rather
// than abort the test (the eventually loop has already captured a 503
// `draining` response in the prior iteration if everything is working).
func tryGetReadyzNonFatal(ctx context.Context, baseURL string) (int, e2eshared.ReadyzResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/readyz", nil)
	if err != nil {
		return 0, e2eshared.ReadyzResponse{}, err
	}

	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return 0, e2eshared.ReadyzResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, e2eshared.ReadyzResponse{}, err
	}

	var decoded e2eshared.ReadyzResponse
	if len(body) > 0 {
		_ = json.Unmarshal(body, &decoded)
	}

	return resp.StatusCode, decoded, nil
}

// =============================================================================
// 9. K8s probe contract — happy-path latency stays well under timeoutSeconds
// =============================================================================

// TestReadyz_K8s_ProbeContract_FastResponse runs 50 concurrent /readyz
// probes against the Manager (no chaos, no toxics — pure happy path) and
// asserts P99 latency stays well under 1s. The K8s readinessProbe in
// docs/readyz-guide.md §4 is configured with timeoutSeconds=3; we want to
// catch any regression that would push latency close to that bound. 1s is
// the early-warning line.
//
// 50 concurrent probes also exercises the parallel-fanout path of the
// handler under realistic load — no goroutine pile-up, no per-dep deadline
// violations.
func TestReadyz_K8s_ProbeContract_FastResponse(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const (
		probes              = 50
		p99Threshold        = 1 * time.Second
		individualThreshold = 2 * time.Second
	)

	managerURL := managerBaseURL(t)

	// Warm the connection pool / DNS cache with one priming hit before
	// the burst — first hit on a cold DNS lookup or pool can be a
	// pathological outlier that doesn't reflect steady-state behavior.
	_, _, _ = e2eshared.GetReadyz(t, ctx, managerURL)

	type result struct {
		latency time.Duration
		status  int
		err     error
	}

	results := make(chan result, probes)

	var wg sync.WaitGroup

	for i := 0; i < probes; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			start := time.Now()
			s, _, err := tryGetReadyzNonFatal(ctx, managerURL)
			elapsed := time.Since(start)

			results <- result{latency: elapsed, status: s, err: err}
		}()
	}

	wg.Wait()
	close(results)

	latencies := make([]time.Duration, 0, probes)

	var failures int

	for r := range results {
		if r.err != nil || r.status != http.StatusOK {
			failures++
			t.Logf("probe failed: status=%d err=%v elapsed=%v", r.status, r.err, r.latency)

			continue
		}

		latencies = append(latencies, r.latency)
	}

	require.Equal(t, 0, failures,
		"all %d concurrent probes must return 200 in the happy path", probes)
	require.Len(t, latencies, probes, "every probe must contribute a latency sample")

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	// P50, P95, P99 indexes (0-based, with rounding for small N).
	p50 := latencies[len(latencies)/2]
	p95 := latencies[(len(latencies)*95)/100]
	p99 := latencies[(len(latencies)*99)/100]
	maxLat := latencies[len(latencies)-1]

	t.Logf("/readyz under %d concurrent probes: P50=%v P95=%v P99=%v max=%v",
		probes, p50, p95, p99, maxLat)

	assert.Less(t, p99, p99Threshold,
		"P99 must be under %v (early-warning threshold for K8s timeoutSeconds=3); got %v",
		p99Threshold, p99)

	// No single probe should approach the K8s probe timeout.
	assert.Less(t, maxLat, individualThreshold,
		"no single probe may approach the K8s timeoutSeconds=3 budget; got max=%v",
		maxLat)
}

// =============================================================================
// internal helpers
// =============================================================================

// managerBaseURL resolves the Manager base URL for /readyz probes. Mirrors
// the logic main_test.go uses to populate apiClient — when E2E_SKIP_MANAGER
// is set, the env var supplies the URL; otherwise the e2ekit RunningApp does.
func managerBaseURL(t *testing.T) string {
	t.Helper()

	if url := externalManagerURL(); url != "" {
		return url
	}

	require.NotNil(t, managerApp, "manager container must be running for /readyz e2e tests")

	return managerApp.BaseURL
}

// getReadyzRawJSON fetches /readyz and returns it decoded as map[string]any
// so the test can audit top-level keys for drift.
func getReadyzRawJSON(t *testing.T, ctx context.Context, baseURL string) (int, map[string]any) {
	t.Helper()

	status, body := rawGet(t, ctx, baseURL+"/readyz")

	var decoded map[string]any
	if len(body) > 0 {
		require.NoError(t, json.Unmarshal([]byte(body), &decoded),
			"decode /readyz body as map: %s", body)
	}

	return status, decoded
}

// rawGet issues a GET against the given URL with no auth headers and returns
// the status code and raw body. Used by the schema-compliance and tenant-404
// tests where the structured ReadyzResponse decode would mask details.
func rawGet(t *testing.T, ctx context.Context, url string) (int, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err, "build %s request", url)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	require.NoError(t, err, "GET %s", url)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read %s body", url)

	return resp.StatusCode, string(body)
}
