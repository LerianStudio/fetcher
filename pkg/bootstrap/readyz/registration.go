package readyz

import (
	"github.com/gofiber/fiber/v2"
)

// TenantHandler is a fiber.Handler dedicated to /readyz/tenant/:id. Gate 2
// ships a stub; Gate 6 replaces it with the multi-tenant prober that runs
// per-tenant checks behind authentication. The stub is deliberately
// unauthenticated so probe callers (Kubernetes, load-balancer) can reach it
// without a token — Gate 6 will layer auth on top.
type TenantHandler fiber.Handler

// StubTenantHandler returns a fiber.Handler that responds 200 with a
// skipped-status JSON body. Used in Gate 2 while Gate 6 is pending.
func StubTenantHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(DependencyCheck{
			Status: StatusSkipped,
			Reason: "pending Gate 6 implementation",
		})
	}
}

// HealthHandler returns a Fiber handler that answers Kubernetes'
// livenessProbe. It is the Gate 7 replacement for lib-commons' Ping.
//
// Behaviour (ring:dev-readyz contract):
//
//   - IsSelfProbeOK() == true  → 200 {"status":"healthy"}
//   - IsSelfProbeOK() == false → 503 {"status":"unhealthy","reason":"self-probe failed"}
//
// /health is NOT split into /health/live + /health/ready — ring:dev-readyz
// requires a single /health gated by the startup self-probe and a separate
// /readyz for the periodic per-dep probe. Kubernetes wires livenessProbe to
// /health and readinessProbe to /readyz.
//
// The handler is free of I/O: it only reads an atomic bool. The cost of
// answering /health is therefore bounded and safe for high-frequency probes.
func HealthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !IsSelfProbeOK() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unhealthy",
				"reason": "self-probe failed",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "healthy",
		})
	}
}

// HealthPassthrough is kept as a thin alias of HealthHandler for backwards
// compatibility with call sites created in Gate 2. New code SHOULD prefer
// HealthHandler directly — the name is clearer about what the handler does.
//
// Deprecated: use HealthHandler. Kept to avoid touching the entire call graph
// in one commit; scheduled for removal after the Gate 11 activation guide
// lands.
func HealthPassthrough() fiber.Handler {
	return HealthHandler()
}

// Register wires /readyz, /readyz/tenant/:id, /metrics and /health onto the
// given fiber app using the supplied handler + checkers. The routes are
// registered BEFORE any auth middleware so they remain unauthenticated,
// per the ring:dev-readyz contract (Kubernetes and external load-balancers
// must reach them without a token).
//
// Gate 5 of ring:dev-readyz: /metrics is served by NewMetricsHandler, which
// is the Prometheus exposition endpoint backed by the readyz metric registry
// plus the Go runtime metrics from the default registerer. This replaces the
// empty Gate 2 placeholder.
//
// Gate 7 of ring:dev-readyz: /health is gated on the startup self-probe.
// Until RunSelfProbe has flipped selfProbeOK to true, /health returns 503
// so K8s' livenessProbe restarts the pod if a dep was unreachable at boot.
//
// The manager router composes its own route table (see
// components/manager/internal/adapters/http/in/routes.go) and uses this
// function's building blocks directly. Register is used by the worker
// micro-server where there is no richer routing to coordinate.
func Register(app *fiber.App, h *Handler) {
	app.Get("/health", HealthHandler())
	app.Get("/readyz", h.Fiber())
	app.Get("/readyz/tenant/:id", StubTenantHandler())
	app.Get("/metrics", NewMetricsHandler())
}
