package readyz

import (
	"context"
	"errors"
	"sync"
	"time"

	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"github.com/gofiber/fiber/v2"
)

// tenantExistenceTimeout is the budget for the tmclient call that validates
// the URL-path tenant ID. Kept equal to PerDepTimeout("tenant_manager") so
// the handler's latency envelope stays consistent with the global /readyz.
const tenantExistenceTimeout = 1 * time.Second

// TenantFiberHandler is the /readyz/tenant/:id handler. It validates the
// URL-path tenant ID against the Tenant Manager active-tenants list, then
// runs every registered TenantChecker in parallel with the tenant ID carried
// on ctx via tmcore.ContextWithTenantID.
//
// The response shape is identical to the global /readyz response plus an
// informational tenant_id field (omitempty). Aggregation follows the same
// rule: healthy iff every check is in {up, skipped, n/a}.
//
// The draining short-circuit works identically to the global handler: when
// IsDraining() is true the handler emits the synthetic "draining" check and
// returns 503 without looking at checkers.
type TenantFiberHandler struct {
	cfg      *Config
	tmClient TMClient
	service  string
	checkers []TenantChecker
}

// NewTenantHandler constructs a /readyz/tenant/:id handler. Pass nil for
// tmClient to let the handler decline to serve (returns 503) — the
// bootstrap is expected to use NewDisabledTenantHandler in that case.
func NewTenantHandler(cfg *Config, tmClient TMClient, service string, checkers ...TenantChecker) *TenantFiberHandler {
	if cfg == nil {
		cfg = LoadConfig()
	}

	return &TenantFiberHandler{
		cfg:      cfg,
		tmClient: tmClient,
		service:  service,
		checkers: checkers,
	}
}

// Fiber returns the fiber.Handler. Extracted so it can be wired into an
// existing router or registered via readyz.Register.
func (h *TenantFiberHandler) Fiber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if IsDraining() {
			return c.Status(fiber.StatusServiceUnavailable).
				JSON(h.drainingResponse(""))
		}

		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).
				JSON(fiber.Map{"error": "tenant id required"})
		}

		if h.tmClient == nil {
			return c.Status(fiber.StatusServiceUnavailable).
				JSON(fiber.Map{"error": "tenant manager client not configured"})
		}

		// Phase 1: validate tenant exists. Short 1s budget — the probe
		// below still needs to fit under the Kubernetes readiness probe.
		vctx, cancel := context.WithTimeout(c.UserContext(), tenantExistenceTimeout)
		defer cancel()

		tenants, err := h.tmClient.GetActiveTenantsByService(vctx, h.service)
		if err != nil {
			if errors.Is(err, tmcore.ErrCircuitBreakerOpen) {
				return c.Status(fiber.StatusServiceUnavailable).
					JSON(fiber.Map{"error": "tenant manager circuit breaker open"})
			}

			return c.Status(fiber.StatusServiceUnavailable).
				JSON(fiber.Map{"error": "tenant manager unreachable: " + classifyErr(vctx, err)})
		}

		if !tenantExists(tenants, id) {
			return c.Status(fiber.StatusNotFound).
				JSON(fiber.Map{"error": "tenant not found", "tenant_id": id})
		}

		// Phase 2: run per-tenant checks under the ctx with tenant ID
		// installed. The per-dep deadline comes from PerDepTimeout by
		// checker name (same budgets as /readyz).
		runCtx := tmcore.ContextWithTenantID(c.UserContext(), id)
		resp := h.runTenantChecks(runCtx, id)

		status := fiber.StatusOK
		if resp.Status == TopStatusUnhealthy {
			status = fiber.StatusServiceUnavailable
		}

		return c.Status(status).JSON(resp)
	}
}

// runTenantChecks executes every TenantChecker in parallel with a per-check
// deadline derived from PerDepTimeout(checker.Name()). It is exported on
// purpose: tests can call it without going through Fiber.
func (h *TenantFiberHandler) runTenantChecks(ctx context.Context, tenantID string) ReadyzResponse {
	checks := make(map[string]DependencyCheck, len(h.checkers))

	if len(h.checkers) == 0 {
		return ReadyzResponse{
			Status:         TopStatusHealthy,
			Checks:         checks,
			Version:        h.cfg.Version,
			DeploymentMode: h.cfg.DeploymentMode,
			TenantID:       tenantID,
		}
	}

	type result struct {
		name  string
		check DependencyCheck
	}

	results := make(chan result, len(h.checkers))

	var wg sync.WaitGroup

	for _, ck := range h.checkers {
		wg.Add(1)

		go func(c TenantChecker) {
			defer wg.Done()

			name := c.Name()

			depCtx, cancel := context.WithTimeout(ctx, PerDepTimeout(name))
			defer cancel()

			start := time.Now()
			check := runTenantWithDeadline(depCtx, c, tenantID)
			elapsed := time.Since(start)

			// Re-use the same handler metrics as the global /readyz so
			// cardinality stays bounded (dep + status, no tenant_id).
			emitCheckDuration(name, check.Status, elapsed)
			emitCheckStatus(name, check.Status)

			if check.LatencyMs == 0 && (check.Status == StatusUp || check.Status == StatusDegraded) {
				check.LatencyMs = elapsed.Milliseconds()
			}

			results <- result{name: name, check: check}
		}(ck)
	}

	wg.Wait()
	close(results)

	for r := range results {
		checks[r.name] = r.check
	}

	return ReadyzResponse{
		Status:         aggregateStatus(checks),
		Checks:         checks,
		Version:        h.cfg.Version,
		DeploymentMode: h.cfg.DeploymentMode,
		TenantID:       tenantID,
	}
}

// runTenantWithDeadline mirrors runWithDeadline for TenantChecker. A
// misbehaving checker cannot stall aggregation — ctx.Done() short-circuits
// to a synthetic down.
func runTenantWithDeadline(ctx context.Context, c TenantChecker, tenantID string) DependencyCheck {
	done := make(chan DependencyCheck, 1)

	go func() {
		done <- c.CheckForTenant(ctx, tenantID)
	}()

	select {
	case res := <-done:
		return res
	case <-ctx.Done():
		return DependencyCheck{
			Status: StatusDown,
			Error:  "check timeout: " + ctx.Err().Error(),
		}
	}
}

// drainingResponse emits the synthetic draining check. Mirrors the global
// handler's buildDrainingResponse but carries the tenant ID.
func (h *TenantFiberHandler) drainingResponse(tenantID string) ReadyzResponse {
	emitCheckDuration(drainingDepName, StatusDown, 0)
	emitCheckStatus(drainingDepName, StatusDown)

	return ReadyzResponse{
		Status: TopStatusUnhealthy,
		Checks: map[string]DependencyCheck{
			drainingDepName: {
				Status: StatusDown,
				Reason: "graceful drain in progress",
			},
		},
		Version:        h.cfg.Version,
		DeploymentMode: h.cfg.DeploymentMode,
		TenantID:       tenantID,
	}
}

// tenantExists reports whether id appears in the tenants slice. Kept as a
// standalone helper so unit tests don't need a fake tmclient to exercise it.
func tenantExists(tenants []*tmclient.TenantSummary, id string) bool {
	for _, t := range tenants {
		if t != nil && t.ID == id {
			return true
		}
	}

	return false
}

// NewDisabledTenantHandler returns a Fiber handler that responds 400 with a
// stable JSON body when multi-tenant mode is disabled. The bootstrap wires
// this when MULTI_TENANT_ENABLED=false so the route stays mounted (operators
// can tell the difference between "route missing" and "MT disabled" without
// reading code).
func NewDisabledTenantHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "multi-tenant mode is disabled"})
	}
}
