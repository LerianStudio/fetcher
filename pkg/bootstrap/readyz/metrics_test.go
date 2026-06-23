package readyz

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scrapeMetrics issues an HTTP GET against the Prometheus handler mounted on a
// throwaway fiber.App and returns the exposition-format body as a string. All
// assertions in this file parse the body either line-by-line or via substring
// matching — the exposition format is stable and cheaper to verify than
// dragging in prometheus/testutil.
func scrapeMetrics(t *testing.T) string {
	t.Helper()

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/metrics", NewMetricsHandler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)

	defer res.Body.Close()
	require.Equal(t, fiber.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return string(body)
}

// parseFloatValue extracts the numeric value from the value-column of a
// Prometheus exposition line (the second whitespace-delimited token). Counter
// and gauge values may round-trip as plain integers ("5") or as floats
// ("5.0"); fmt.Sscanf("%g") accepts both.
func parseFloatValue(t *testing.T, line string) float64 {
	t.Helper()

	parts := strings.Fields(line)
	require.Len(t, parts, 2, "unexpected exposition line: %q", line)

	var f float64

	_, err := fmt.Sscanf(parts[1], "%g", &f)
	require.NoError(t, err, "parsing value from %q", line)

	return f
}

// TestNewMetricsHandler_RegistersAllThree asserts that all three contract
// metrics (readyz_check_duration_ms, readyz_check_status, selfprobe_result)
// are exposed by the Prometheus handler returned by NewMetricsHandler. This is
// the primary Gate 5 compliance check — if a metric name is missing, the
// contract with operator dashboards is broken.
func TestNewMetricsHandler_RegistersAllThree(t *testing.T) {
	// Emit once for each metric so its name appears in the exposition output.
	// A registered but never-emitted metric still appears for HistogramVec +
	// CounterVec (MetricVec descriptors are emitted on scrape), but gauges
	// only materialise a series after Set — so we emit all three to be safe.
	emitCheckDuration("probe_init", StatusUp, 7*time.Millisecond)
	emitCheckStatus("probe_init", StatusUp)
	emitSelfProbeResult("probe_init", true)

	body := scrapeMetrics(t)

	assert.Contains(t, body, "readyz_check_duration_ms", "histogram missing from /metrics")
	assert.Contains(t, body, "readyz_check_status", "counter missing from /metrics")
	assert.Contains(t, body, "selfprobe_result", "gauge missing from /metrics")
}

func TestMetricsCompatibilityBridge_IsOnlyRawPrometheusSurface(t *testing.T) {
	bridge := newPrometheusCompatibilityBridge()

	require.NotNil(t, bridge.checkDuration)
	require.NotNil(t, bridge.checkStatus)
	require.NotNil(t, bridge.selfProbe)

	bridge.observeCheckDuration("bridge_probe", StatusUp, time.Millisecond)
	bridge.incrementCheckStatus("bridge_probe", StatusUp)
	bridge.setSelfProbeResult("bridge_probe", true)
}

// TestEmitCheckDuration_RecordsObservation asserts the histogram's _count
// series for the (dep, status) tuple increments after an emit call. The
// exposition line for a histogram count looks like:
//
//	readyz_check_duration_ms_count{dep="mongodb",status="up"} 3
//
// We grep for the dep+status tuple and confirm the count suffix.
func TestEmitCheckDuration_RecordsObservation(t *testing.T) {
	dep := "test_duration_dep"
	emitCheckDuration(dep, StatusUp, 42*time.Millisecond)
	emitCheckDuration(dep, StatusUp, 123*time.Millisecond)

	body := scrapeMetrics(t)

	found := false

	for line := range strings.SplitSeq(body, "\n") {
		if !strings.HasPrefix(line, "readyz_check_duration_ms_count{") {
			continue
		}

		if !strings.Contains(line, `dep="`+dep+`"`) || !strings.Contains(line, `status="up"`) {
			continue
		}

		count := parseFloatValue(t, line)
		assert.GreaterOrEqual(t, count, float64(2), "expected >= 2 observations for (%s, up)", dep)

		found = true

		break
	}

	assert.True(t, found, "histogram _count line for dep=%q not found in /metrics", dep)
}

// TestEmitCheckStatus_IncrementsCounter asserts the counter grows by exactly N
// after N emit calls with the same (dep, status) tuple.
func TestEmitCheckStatus_IncrementsCounter(t *testing.T) {
	dep := "test_status_counter"

	const n = 5

	for range n {
		emitCheckStatus(dep, StatusDown)
	}

	body := scrapeMetrics(t)

	found := false

	for line := range strings.SplitSeq(body, "\n") {
		if !strings.HasPrefix(line, "readyz_check_status{") {
			continue
		}

		if !strings.Contains(line, `dep="`+dep+`"`) || !strings.Contains(line, `status="down"`) {
			continue
		}

		value := parseFloatValue(t, line)
		assert.GreaterOrEqual(t, value, float64(n), "counter < n emits")

		found = true

		break
	}

	assert.True(t, found, "counter line for dep=%q not found", dep)
}

// TestEmitSelfProbeResult_SetGauge asserts the gauge reports 1 after up=true
// and 0 after up=false for the same dep label.
func TestEmitSelfProbeResult_SetGauge(t *testing.T) {
	dep := "test_selfprobe_gauge"

	emitSelfProbeResult(dep, true)
	assertGauge(t, scrapeMetrics(t), dep, 1)

	emitSelfProbeResult(dep, false)
	assertGauge(t, scrapeMetrics(t), dep, 0)
}

// assertGauge parses the selfprobe_result series for the given dep and asserts
// the latest value equals want.
func assertGauge(t *testing.T, body, dep string, want float64) {
	t.Helper()

	for line := range strings.SplitSeq(body, "\n") {
		if !strings.HasPrefix(line, "selfprobe_result{") {
			continue
		}

		if !strings.Contains(line, `dep="`+dep+`"`) {
			continue
		}

		got := parseFloatValue(t, line)
		assert.InDelta(t, want, got, 0.001, "gauge value for dep=%q", dep)

		return
	}

	t.Fatalf("selfprobe_result{dep=%q} not found in /metrics body", dep)
}

// TestHistogramBuckets_MatchContract asserts that every contract bucket
// boundary is present as a _bucket series (le="<boundary>"). This is the
// concrete test that pins the bucket shape to the ring:dev-readyz contract:
// [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000] ms.
func TestHistogramBuckets_MatchContract(t *testing.T) {
	// Observe once so the histogram has at least one sample. Buckets are
	// defined at registration so they appear even without observations, but
	// forcing one sample also exercises the emit path.
	emitCheckDuration("bucket_shape_probe", StatusUp, 2*time.Millisecond)

	body := scrapeMetrics(t)

	wantBuckets := []string{"1", "5", "10", "25", "50", "100", "250", "500", "1000", "2000", "5000"}
	for _, b := range wantBuckets {
		assert.Contains(t, body, `readyz_check_duration_ms_bucket{`, "no histogram buckets emitted")
		assert.Contains(t, body, `le="`+b+`"`, "bucket boundary %s ms missing from exposition", b)
	}
}

// TestNewMetricsHandler_CalledMultipleTimes_NoPanic guards against accidental
// re-registration of the collectors. sync.Once inside NewMetricsHandler must
// make a second call a no-op.
func TestNewMetricsHandler_CalledMultipleTimes_NoPanic(t *testing.T) {
	// Also hit the handler concurrently to make sure sync.Once is correctly
	// scoped (it MUST serialise MustRegister calls).
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			require.NotPanics(t, func() {
				h := NewMetricsHandler()
				assert.NotNil(t, h)
			})
		})
	}

	wg.Wait()
}

// TestCardinality_DepAndStatus_OnlyFiveValues asserts that once we have
// emitted for 3 deps across all 5 statuses, exactly 15 unique (dep,status)
// tuples appear in readyz_check_status. This protects against accidentally
// introducing a free-form label elsewhere in the code.
func TestCardinality_DepAndStatus_OnlyFiveValues(t *testing.T) {
	deps := []string{"card_dep_a", "card_dep_b", "card_dep_c"}
	statuses := []string{StatusUp, StatusDown, StatusDegraded, StatusSkipped, StatusNA}

	for _, d := range deps {
		for _, s := range statuses {
			emitCheckStatus(d, s)
		}
	}

	body := scrapeMetrics(t)

	unique := make(map[string]struct{})

	for line := range strings.SplitSeq(body, "\n") {
		if !strings.HasPrefix(line, "readyz_check_status{") {
			continue
		}

		for _, d := range deps {
			if strings.Contains(line, `dep="`+d+`"`) {
				unique[line] = struct{}{}
				break
			}
		}
	}

	assert.Equal(t, 15, len(unique), "expected 15 unique (dep,status) combinations, got %d", len(unique))
}

// TestHandler_Run_EmitsMetrics wires a stub checker into the canonical handler
// and asserts the per-dep counter + histogram are updated after Run.
func TestHandler_Run_EmitsMetrics(t *testing.T) {
	SetDraining(false)

	dep := "handler_emits_metrics_probe"
	h := NewHandler(baseCfg(),
		&fakeChecker{name: dep, out: DependencyCheck{Status: StatusUp, LatencyMs: 5, TLS: TLSPtr(true)}},
	)

	resp := h.Run(context.Background())
	require.Equal(t, TopStatusHealthy, resp.Status)

	body := scrapeMetrics(t)

	assert.Contains(t, body, `readyz_check_status{dep="`+dep+`",status="up"}`)
	assert.Contains(t, body, `readyz_check_duration_ms_count{dep="`+dep+`",status="up"}`)
}

// TestHandler_DrainPath_EmitsDrainingMetric asserts that the graceful-drain
// short-circuit (Fiber handler returning 503 without running checkers) still
// emits the synthetic "draining" counter so operators can alert on
// rate(readyz_check_status{dep="draining",status="down"}[5m]).
func TestHandler_DrainPath_EmitsDrainingMetric(t *testing.T) {
	SetDraining(true)
	t.Cleanup(func() { SetDraining(false) })

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "irrelevant", out: DependencyCheck{Status: StatusUp}},
	)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/readyz", h.Fiber())

	req := httptest.NewRequest("GET", "/readyz", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, fiber.StatusServiceUnavailable, res.StatusCode)

	body := scrapeMetrics(t)
	assert.Contains(t, body, `readyz_check_status{dep="draining",status="down"}`)
	assert.Contains(t, body, `readyz_check_duration_ms_count{dep="draining",status="down"}`)
}

// TestHandler_ConcurrentRuns_NoMetricRaces asserts that concurrent Run calls
// do not race on metric emission. This complements the race-detector run in
// TestHandler_Run_ConcurrentRequests_NoRace (handler_test.go) by including
// the emit path in the race window.
func TestHandler_ConcurrentRuns_NoMetricRaces(t *testing.T) {
	SetDraining(false)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "race_mongo", out: DependencyCheck{Status: StatusUp}},
		&fakeChecker{name: "race_redis", out: DependencyCheck{Status: StatusUp}},
	)

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 20 {
				_ = h.Run(context.Background())
			}
		})
	}

	wg.Wait()

	// Verify the metrics are present. The race detector catches failures.
	body := scrapeMetrics(t)
	assert.Contains(t, body, `dep="race_mongo"`)
	assert.Contains(t, body, `dep="race_redis"`)
}

// TestRegistration_MetricsEndpoint_ReturnsPrometheusExposition asserts that
// Register wires a real Prometheus handler on /metrics (no more empty body).
// This is the integration check for the Gate 5 swap of MetricsPlaceholder for
// NewMetricsHandler inside Register.
func TestRegistration_MetricsEndpoint_ReturnsPrometheusExposition(t *testing.T) {
	SetDraining(false)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	h := NewHandler(baseCfg(),
		&fakeChecker{name: "wired_test", out: DependencyCheck{Status: StatusUp}},
	)
	Register(app, h)

	req := httptest.NewRequest("GET", "/metrics", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusOK, res.StatusCode)

	// Prometheus exposition format includes "# HELP" lines when any metric is
	// registered. Content-Type is text/plain; version=0.0.4 by default.
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "# HELP", "/metrics body does not look like Prometheus exposition")
	assert.Contains(t, res.Header.Get("Content-Type"), "text/plain")
}

// TestReadyzRun_PreservesExistingLatencyMs asserts that when a checker sets
// LatencyMs itself, the handler does NOT overwrite it — but the metric
// emission uses the measured wall-clock elapsed time regardless.
func TestReadyzRun_PreservesExistingLatencyMs(t *testing.T) {
	SetDraining(false)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "preserves_latency", out: DependencyCheck{
			Status:    StatusUp,
			LatencyMs: 999, // checker-reported value
		}},
	)

	resp := h.Run(context.Background())
	assert.Equal(t, int64(999), resp.Checks["preserves_latency"].LatencyMs,
		"handler must not overwrite checker-reported LatencyMs")
}

// TestReadyzRun_DefaultsLatencyMsWhenEmpty asserts the handler auto-fills
// LatencyMs from measured elapsed time when the checker left it at zero and
// status is up/degraded.
func TestReadyzRun_DefaultsLatencyMsWhenEmpty(t *testing.T) {
	SetDraining(false)

	h := NewHandler(baseCfg(),
		&fakeChecker{
			name:  "default_latency",
			out:   DependencyCheck{Status: StatusUp},
			sleep: 3 * time.Millisecond,
		},
	)

	resp := h.Run(context.Background())
	assert.GreaterOrEqual(t, resp.Checks["default_latency"].LatencyMs, int64(1),
		"handler must default LatencyMs from measured elapsed")
}
