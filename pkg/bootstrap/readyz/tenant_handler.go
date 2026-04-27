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

// tenantExistenceTimeout matches PerDepTimeout("tenant_manager") so the
// handler's latency envelope stays consistent with the global /readyz.
const tenantExistenceTimeout = 1 * time.Second

// TenantFiberHandler serves /readyz/tenant/:id. It validates the path tenant
// ID against the active-tenants list, then runs every TenantChecker in
// parallel with the tenant ID installed on ctx via
// tmcore.ContextWithTenantID. Aggregation and draining behavior mirror the
// global handler; the response carries an additional tenant_id field.
type TenantFiberHandler struct {
	cfg      *Config
	tmClient TMClient
	service  string
	checkers []TenantChecker
}

// NewTenantHandler accepts a nil tmClient, in which case the handler returns
// 503; the bootstrap should pair that with NewDisabledTenantHandler.
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

		runCtx := tmcore.ContextWithTenantID(c.UserContext(), id)
		resp := h.runTenantChecks(runCtx, id)

		status := fiber.StatusOK
		if resp.Status == TopStatusUnhealthy {
			status = fiber.StatusServiceUnavailable
		}

		return c.Status(status).JSON(resp)
	}
}

// runTenantChecks runs every TenantChecker in parallel under
// PerDepTimeout(checker.Name()). Exported so tests can drive it without
// Fiber.
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

			// Reuse the global metrics (dep + status only) to keep
			// cardinality bounded — tenant_id is intentionally omitted.
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

// runTenantWithDeadline mirrors runWithDeadline for TenantChecker so a
// misbehaving checker cannot stall aggregation.
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

// drainingResponse mirrors buildDrainingResponse but carries the tenant ID.
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

func tenantExists(tenants []*tmclient.TenantSummary, id string) bool {
	for _, t := range tenants {
		if t != nil && t.ID == id {
			return true
		}
	}

	return false
}

// NewDisabledTenantHandler keeps the route mounted (returning 400) when
// multi-tenant mode is disabled, so operators can distinguish "MT disabled"
// from "route missing" without reading code.
func NewDisabledTenantHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "multi-tenant mode is disabled"})
	}
}
