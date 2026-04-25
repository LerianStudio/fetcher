package readyz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticTenantChecker is a helper producing canned DependencyCheck responses
// for tenant-handler tests. It captures the tenant ID it was called with so
// tests can assert context propagation.
type staticTenantChecker struct {
	name     string
	check    DependencyCheck
	delay    time.Duration
	seenID   string
	seenCtx  context.Context //nolint:containedctx // test-only capture
	seenCtxM bool
}

func (s *staticTenantChecker) Name() string { return s.name }
func (s *staticTenantChecker) CheckForTenant(ctx context.Context, id string) DependencyCheck {
	s.seenID = id
	if !s.seenCtxM {
		s.seenCtx = ctx
		s.seenCtxM = true
	}

	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return DependencyCheck{Status: StatusDown, Error: "timeout"}
		}
	}

	return s.check
}

func newApp(h *TenantFiberHandler) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/readyz/tenant/:id", h.Fiber())

	return app
}

func doReq(t *testing.T, app *fiber.App, path string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)

	resp, err := app.Test(req, 3_000)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()

	return resp.StatusCode, body
}

func TestTenantHandler_HappyPath(t *testing.T) {
	resetDrainingAndCleanup(t)

	tm := &fakeTMClient{tenants: []*tmclient.TenantSummary{{ID: "t1"}}}
	mongoCk := &staticTenantChecker{name: "mongodb", check: DependencyCheck{Status: StatusUp}}
	rabbitCk := &staticTenantChecker{name: "rabbitmq", check: DependencyCheck{Status: StatusUp}}

	cfg := &Config{DeploymentMode: DeploymentModeLocal, Version: "v1.0.0"}
	h := NewTenantHandler(cfg, tm, "fetcher", mongoCk, rabbitCk)
	app := newApp(h)

	status, body := doReq(t, app, "/readyz/tenant/t1")
	require.Equal(t, fiber.StatusOK, status)

	var resp ReadyzResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, TopStatusHealthy, resp.Status)
	assert.Equal(t, "t1", resp.TenantID)
	assert.Len(t, resp.Checks, 2)
	assert.Equal(t, StatusUp, resp.Checks["mongodb"].Status)
	assert.Equal(t, StatusUp, resp.Checks["rabbitmq"].Status)
	assert.Equal(t, "t1", mongoCk.seenID)

	// Verify tenant ID made it onto the ctx as well.
	require.True(t, mongoCk.seenCtxM)
	assert.Equal(t, "t1", tmcore.GetTenantIDContext(mongoCk.seenCtx))
}

func TestTenantHandler_TenantNotFound(t *testing.T) {
	resetDrainingAndCleanup(t)

	tm := &fakeTMClient{tenants: []*tmclient.TenantSummary{{ID: "other"}}}
	h := NewTenantHandler(nil, tm, "fetcher")
	app := newApp(h)

	status, body := doReq(t, app, "/readyz/tenant/ghost")
	assert.Equal(t, fiber.StatusNotFound, status)
	assert.Contains(t, string(body), "tenant not found")
}

func TestTenantHandler_TMClientBreakerOpenDuringValidation(t *testing.T) {
	resetDrainingAndCleanup(t)

	tm := &fakeTMClient{err: fmt.Errorf("lookup: %w", tmcore.ErrCircuitBreakerOpen)}
	h := NewTenantHandler(nil, tm, "fetcher")
	app := newApp(h)

	status, body := doReq(t, app, "/readyz/tenant/t1")
	assert.Equal(t, fiber.StatusServiceUnavailable, status)
	assert.Contains(t, string(body), "circuit breaker open")
}

func TestTenantHandler_TMClientGenericError(t *testing.T) {
	resetDrainingAndCleanup(t)

	tm := &fakeTMClient{err: errors.New("http 500")}
	h := NewTenantHandler(nil, tm, "fetcher")
	app := newApp(h)

	status, body := doReq(t, app, "/readyz/tenant/t1")
	assert.Equal(t, fiber.StatusServiceUnavailable, status)
	assert.Contains(t, string(body), "unreachable")
}

func TestTenantHandler_NilTMClient(t *testing.T) {
	resetDrainingAndCleanup(t)

	h := NewTenantHandler(nil, nil, "fetcher")
	app := newApp(h)

	status, _ := doReq(t, app, "/readyz/tenant/t1")
	assert.Equal(t, fiber.StatusServiceUnavailable, status)
}

func TestTenantHandler_PerTenantDepDown_Returns503(t *testing.T) {
	resetDrainingAndCleanup(t)

	tm := &fakeTMClient{tenants: []*tmclient.TenantSummary{{ID: "t1"}}}
	down := &staticTenantChecker{name: "mongodb", check: DependencyCheck{
		Status: StatusDown, Error: "dial tcp: refused",
	}}
	up := &staticTenantChecker{name: "rabbitmq", check: DependencyCheck{Status: StatusUp}}

	h := NewTenantHandler(nil, tm, "fetcher", down, up)
	app := newApp(h)

	status, body := doReq(t, app, "/readyz/tenant/t1")
	assert.Equal(t, fiber.StatusServiceUnavailable, status)

	var resp ReadyzResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, TopStatusUnhealthy, resp.Status)
	assert.Equal(t, StatusDown, resp.Checks["mongodb"].Status)
	assert.Equal(t, StatusUp, resp.Checks["rabbitmq"].Status)
	assert.Equal(t, "t1", resp.TenantID)
}

func TestTenantHandler_DrainingShortCircuits(t *testing.T) {
	SetDraining(true)
	t.Cleanup(func() { SetDraining(false) })

	tm := &fakeTMClient{tenants: []*tmclient.TenantSummary{{ID: "t1"}}}
	ck := &staticTenantChecker{name: "mongodb", check: DependencyCheck{Status: StatusUp}}

	h := NewTenantHandler(nil, tm, "fetcher", ck)
	app := newApp(h)

	status, body := doReq(t, app, "/readyz/tenant/t1")
	assert.Equal(t, fiber.StatusServiceUnavailable, status)

	var resp ReadyzResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, TopStatusUnhealthy, resp.Status)
	assert.Contains(t, resp.Checks, drainingDepName)

	// Draining short-circuits BEFORE validation; the checker must not be
	// called.
	assert.Empty(t, ck.seenID, "draining MUST short-circuit before checkers run")
}

func TestDisabledTenantHandler_Returns400(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/readyz/tenant/:id", NewDisabledTenantHandler())

	req := httptest.NewRequest("GET", "/readyz/tenant/t1", nil)
	resp, err := app.Test(req, 3_000)
	require.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), "multi-tenant mode is disabled")
}

func TestTenantExists(t *testing.T) {
	tenants := []*tmclient.TenantSummary{{ID: "a"}, nil, {ID: "b"}}
	assert.True(t, tenantExists(tenants, "a"))
	assert.True(t, tenantExists(tenants, "b"))
	assert.False(t, tenantExists(tenants, "c"))
	assert.False(t, tenantExists(nil, "a"))
}
