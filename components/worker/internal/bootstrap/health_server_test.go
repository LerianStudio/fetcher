package bootstrap

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthServer_DefaultsToPort4007(t *testing.T) {
	cfg := &Config{HealthPort: 0}
	srv := NewHealthServer(cfg, nil, nil, nil)

	assert.Equal(t, ":4007", srv.Address())
	assert.NotNil(t, srv.App())
}

func TestNewHealthServer_HonoursConfiguredPort(t *testing.T) {
	cfg := &Config{HealthPort: 9999}
	srv := NewHealthServer(cfg, nil, nil, nil)

	assert.Equal(t, ":9999", srv.Address())
}

func TestHealthServer_ServesReadyzWithNilDeps(t *testing.T) {
	// Gate 6 of ring:dev-readyz: the worker now registers REAL checkers.
	// With deps=nil, the checker list is empty — the response is healthy
	// with an empty Checks map. The dashboards notice "no deps" rather than
	// stale "skipped" entries, which is the desired behaviour for
	// misconfigured bootstraps.
	readyz.SetDraining(false)

	cfg := &Config{
		HealthPort:          4007,
		DeploymentMode:      "local",
		ReadyzDrainDelaySec: 12,
		OtelServiceVersion:  "test-v1",
	}
	srv := NewHealthServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/readyz", nil)
	res, err := srv.App().Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var parsed readyz.ReadyzResponse
	require.NoError(t, json.Unmarshal(body, &parsed))

	assert.Equal(t, readyz.TopStatusHealthy, parsed.Status)
	assert.Equal(t, "test-v1", parsed.Version)
	assert.Equal(t, "local", parsed.DeploymentMode)
	assert.Empty(t, parsed.Checks)
}

func TestHealthServer_ServesHealthAndMetrics(t *testing.T) {
	// Gate 7 of ring:dev-readyz: /health is gated on the startup self-probe.
	// The test is checking route wiring, not self-probe behaviour — pre-flip
	// the flag to isolate from the default ("unhealthy until proven
	// otherwise"). The dedicated gating test below exercises the 503 path.
	readyz.SetSelfProbe(true)
	t.Cleanup(func() { readyz.SetSelfProbe(false) })

	cfg := &Config{HealthPort: 4007}
	srv := NewHealthServer(cfg, nil, nil, nil)

	// /health (gated, now pre-flipped) and /metrics are 200. /readyz/tenant/:id
	// with no MT deps falls back to the disabled handler, which returns 400.
	tests := []struct {
		path string
		want int
	}{
		{"/health", fiber.StatusOK},
		{"/metrics", fiber.StatusOK},
		{"/readyz/tenant/foo", fiber.StatusBadRequest},
	}

	for _, tc := range tests {
		req := httptest.NewRequest("GET", tc.path, nil)
		res, err := srv.App().Test(req, 2000)
		require.NoError(t, err, "path %s", tc.path)
		_ = res.Body.Close()
		assert.Equal(t, tc.want, res.StatusCode, "path %s", tc.path)
	}
}

// TestHealthServer_HealthReturns503BeforeSelfProbe asserts the Gate 7
// contract: before RunSelfProbe has succeeded, GET /health returns 503 with
// the canonical unhealthy body so K8s' livenessProbe restarts the pod.
func TestHealthServer_HealthReturns503BeforeSelfProbe(t *testing.T) {
	readyz.SetSelfProbe(false)
	t.Cleanup(func() { readyz.SetSelfProbe(false) })

	cfg := &Config{HealthPort: 4007}
	srv := NewHealthServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	res, err := srv.App().Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusServiceUnavailable, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var parsed map[string]string
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Equal(t, "unhealthy", parsed["status"])
	assert.Equal(t, "self-probe failed", parsed["reason"])
}

// TestHealthServer_HealthReturns200AfterSelfProbe asserts the symmetric
// success case: once the self-probe has flipped the flag, GET /health
// returns 200 with the canonical healthy body.
func TestHealthServer_HealthReturns200AfterSelfProbe(t *testing.T) {
	readyz.SetSelfProbe(true)
	t.Cleanup(func() { readyz.SetSelfProbe(false) })

	cfg := &Config{HealthPort: 4007}
	srv := NewHealthServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	res, err := srv.App().Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var parsed map[string]string
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Equal(t, "healthy", parsed["status"])
}

// TestHealthServer_MetricsEndpoint_ServesPrometheus asserts that the worker's
// /metrics endpoint (mounted on the health micro-server, HEALTH_PORT=4007 in
// production) serves real Prometheus exposition-format output after Gate 5.
// Specifically, Content-Type MUST identify the exposition format, the body
// MUST contain a "# HELP" header (baseline exposition marker), and once a
// /readyz request has fired, the readyz_check_* series MUST be present.
func TestHealthServer_MetricsEndpoint_ServesPrometheus(t *testing.T) {
	readyz.SetDraining(false)

	// Use a MT-mode deps struct so the checker set has at least one entry
	// (NAChecker for mongodb + rabbitmq) and the per-dep counters surface in
	// the /metrics output after one /readyz request.
	cfg := &Config{HealthPort: 4007, MultiTenantEnabled: true}
	srv := NewHealthServer(cfg, nil, nil, &workerReadyzDeps{cfg: cfg})

	// Fire one /readyz request so the per-dep histogram and counter have at
	// least one observation recorded — Prometheus only surfaces a series
	// after the first emit for that label tuple.
	readyzReq := httptest.NewRequest("GET", "/readyz", nil)
	readyzRes, err := srv.App().Test(readyzReq, 2000)
	require.NoError(t, err)
	_ = readyzRes.Body.Close()

	// Now scrape /metrics and verify the exposition output.
	req := httptest.NewRequest("GET", "/metrics", nil)
	res, err := srv.App().Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusOK, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "text/plain",
		"/metrics Content-Type must identify Prometheus exposition format")

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "# HELP",
		"body does not look like Prometheus exposition (no # HELP line)")
	// After one /readyz call, the counter and histogram descriptors MUST be
	// present — labels populated with dep values from the stub checkers.
	assert.Contains(t, bodyStr, "readyz_check_duration_ms",
		"readyz_check_duration_ms not exposed on /metrics after /readyz call")
	assert.Contains(t, bodyStr, "readyz_check_status",
		"readyz_check_status not exposed on /metrics after /readyz call")
}

func TestHealthServer_DrainingReturns503(t *testing.T) {
	readyz.SetDraining(true)
	t.Cleanup(func() { readyz.SetDraining(false) })

	cfg := &Config{HealthPort: 4007, DeploymentMode: "local"}
	srv := NewHealthServer(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/readyz", nil)
	res, err := srv.App().Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusServiceUnavailable, res.StatusCode)
}

func TestDefaultDrain(t *testing.T) {
	assert.Equal(t, defaultReadyzDrainDelay, defaultDrain(0), "zero → default")
	assert.Equal(t, time.Second, defaultDrain(-1), "negative → 1s")
	assert.Equal(t, 5*time.Second, defaultDrain(5), "positive passes through")
}

func TestNewWorkerReadyzConfig_AppliesDefaults(t *testing.T) {
	cfg := &Config{}
	got := newWorkerReadyzConfig(cfg)

	assert.Equal(t, readyz.DeploymentModeLocal, got.DeploymentMode)
	assert.Equal(t, defaultHealthPort, got.HealthPort)
	assert.Equal(t, defaultReadyzDrainDelay, got.DrainDelay)
	assert.Equal(t, "unknown", got.Version)
}

func TestNewWorkerReadyzConfig_HonoursCustomValues(t *testing.T) {
	cfg := &Config{
		DeploymentMode:      "saas",
		HealthPort:          8080,
		ReadyzDrainDelaySec: 20,
		OtelServiceVersion:  "9.9.9",
	}
	got := newWorkerReadyzConfig(cfg)

	assert.Equal(t, readyz.DeploymentModeSaaS, got.DeploymentMode)
	assert.Equal(t, 8080, got.HealthPort)
	assert.Equal(t, 20*time.Second, got.DrainDelay)
	assert.Equal(t, "9.9.9", got.Version)
}
