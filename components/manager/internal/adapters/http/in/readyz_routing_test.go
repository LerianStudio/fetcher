package in

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/bootstrap/readyz"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadyzRoutes_NoAuthRequired verifies that /readyz, /readyz/tenant/:id and
// /metrics are mounted without any auth middleware. We do not exercise the
// full NewRoutes wiring (which drags telemetry and license clients) — instead
// we register the same handlers NewRoutes uses onto a fresh fiber.App and
// assert that an unauthenticated request is NOT rejected with 401.
//
// This is the Gate 2 contract check: probers (Kubernetes, external LBs) must
// be able to reach /readyz without a token.
func TestReadyzRoutes_NoAuthRequired(t *testing.T) {
	readyz.SetDraining(false)

	app := fiber.New()

	cfg := &readyz.Config{
		DeploymentMode: readyz.DeploymentModeLocal,
		Version:        "test",
	}
	h := readyz.NewHandler(cfg,
		readyz.NewNAChecker("mongodb", "routing-test: not probed", nil),
	)

	app.Get("/readyz", h.Fiber())
	app.Get("/readyz/tenant/:id", readyz.NewDisabledTenantHandler())
	app.Get("/metrics", readyz.NewMetricsHandler())

	tests := []struct {
		name string
		path string
	}{
		{name: "readyz", path: "/readyz"},
		{name: "readyz tenant", path: "/readyz/tenant/foo"},
		{name: "metrics", path: "/metrics"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			// NO Authorization header — this is the whole point.
			res, err := app.Test(req, 2000)
			require.NoError(t, err)

			defer res.Body.Close()

			assert.NotEqual(t, fiber.StatusUnauthorized, res.StatusCode,
				"path %s returned 401 — must be unauthenticated", tc.path)
			assert.NotEqual(t, fiber.StatusForbidden, res.StatusCode,
				"path %s returned 403 — must be unauthenticated", tc.path)
			// Either 200 (healthy / skipped) or 503 (unhealthy) is acceptable.
			// 401/403 would mean the route was accidentally authed.
		})
	}
}

// TestRoutes_MetricsEndpoint_Unauthenticated asserts that the manager's
// /metrics endpoint — mounted BEFORE the auth middleware on SERVER_PORT=4006
// — returns 200 without an Authorization header. This matches the K8s and
// Prometheus scrape pattern: scrapers do not carry user tokens, so /metrics
// MUST remain unauthenticated alongside /readyz and /health.
func TestRoutes_MetricsEndpoint_Unauthenticated(t *testing.T) {
	readyz.SetDraining(false)

	app := fiber.New()
	app.Get("/metrics", readyz.NewMetricsHandler())

	req := httptest.NewRequest("GET", "/metrics", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)

	defer res.Body.Close()

	assert.Equal(t, fiber.StatusOK, res.StatusCode,
		"/metrics must be reachable without auth (scrape pattern)")
	assert.NotEqual(t, fiber.StatusUnauthorized, res.StatusCode)
	assert.NotEqual(t, fiber.StatusForbidden, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "text/plain",
		"/metrics must serve Prometheus exposition format")
}

// TestReadyzTenantDisabled_ReturnsBadRequest asserts that the disabled
// variant of the /readyz/tenant/:id handler (used when MULTI_TENANT_ENABLED
// is false) returns 400 with a stable error payload — Gate 6 of
// ring:dev-readyz. The route stays mounted so operators can tell the
// difference between "MT disabled" and "route not wired".
func TestReadyzTenantDisabled_ReturnsBadRequest(t *testing.T) {
	app := fiber.New()
	app.Get("/readyz/tenant/:id", readyz.NewDisabledTenantHandler())

	req := httptest.NewRequest("GET", "/readyz/tenant/tenant-xyz", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)

	defer res.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var parsed map[string]string
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Contains(t, parsed["error"], "multi-tenant mode is disabled")
}
