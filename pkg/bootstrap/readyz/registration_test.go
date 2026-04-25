package readyz

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthHandler_ReturnsHealthyWhenSelfProbeOK verifies the 200-path of
// Gate 7's /health gate: when IsSelfProbeOK() is true, the handler emits
// 200 with {"status":"healthy"}.
func TestHealthHandler_ReturnsHealthyWhenSelfProbeOK(t *testing.T) {
	SetSelfProbe(true)
	t.Cleanup(func() { SetSelfProbe(false) })

	app := fiber.New()
	app.Get("/health", HealthHandler())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)

	defer res.Body.Close()

	assert.Equal(t, fiber.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var parsed map[string]string
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Equal(t, "healthy", parsed["status"])
}

// TestHealthHandler_Returns503WhenSelfProbeFailed verifies the 503-path of
// Gate 7's /health gate: when IsSelfProbeOK() is false, the handler emits
// 503 with {"status":"unhealthy","reason":"self-probe failed"}.
func TestHealthHandler_Returns503WhenSelfProbeFailed(t *testing.T) {
	SetSelfProbe(false)
	t.Cleanup(func() { SetSelfProbe(false) })

	app := fiber.New()
	app.Get("/health", HealthHandler())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res, err := app.Test(req, 2000)
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

// TestHealthHandler_StateFlipsMidLifecycle asserts that a single handler
// instance reflects live state changes — /health is NOT cached. This matches
// the atomic-bool design in state.go and mirrors the contract of /readyz
// (no response caching).
func TestHealthHandler_StateFlipsMidLifecycle(t *testing.T) {
	SetSelfProbe(false)
	t.Cleanup(func() { SetSelfProbe(false) })

	app := fiber.New()
	app.Get("/health", HealthHandler())

	// 503 before flip.
	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil), 2000)
	require.NoError(t, err)
	_ = res.Body.Close()
	assert.Equal(t, fiber.StatusServiceUnavailable, res.StatusCode)

	// Flip → 200.
	SetSelfProbe(true)

	res, err = app.Test(httptest.NewRequest(http.MethodGet, "/health", nil), 2000)
	require.NoError(t, err)
	_ = res.Body.Close()
	assert.Equal(t, fiber.StatusOK, res.StatusCode)

	// Flip back → 503.
	SetSelfProbe(false)

	res, err = app.Test(httptest.NewRequest(http.MethodGet, "/health", nil), 2000)
	require.NoError(t, err)
	_ = res.Body.Close()
	assert.Equal(t, fiber.StatusServiceUnavailable, res.StatusCode)
}

// TestHealthPassthrough_DelegatesToHealthHandler asserts the backwards-compat
// shim still honours the self-probe gate. Gate 2 code paths that still use
// HealthPassthrough continue to work.
func TestHealthPassthrough_DelegatesToHealthHandler(t *testing.T) {
	SetSelfProbe(false)
	t.Cleanup(func() { SetSelfProbe(false) })

	app := fiber.New()
	app.Get("/health", HealthPassthrough())

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil), 2000)
	require.NoError(t, err)
	_ = res.Body.Close()
	assert.Equal(t, fiber.StatusServiceUnavailable, res.StatusCode)
}
