package readyz

import (
	"github.com/gofiber/fiber/v2"
)

// TenantHandler is a fiber.Handler dedicated to /readyz/tenant/:id. The
// route is intentionally unauthenticated so probe callers (Kubernetes,
// load-balancer) can reach it without a token.
type TenantHandler fiber.Handler

// StubTenantHandler responds 200 with a skipped DependencyCheck. Used as a
// placeholder until the per-tenant prober is wired.
func StubTenantHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(DependencyCheck{
			Status: StatusSkipped,
			Reason: "pending Gate 6 implementation",
		})
	}
}

// HealthHandler answers Kubernetes' livenessProbe. The endpoint is
// deliberately I/O free — it reads an atomic bool — so it is safe to probe
// at high frequency. /health is gated on the startup self-probe and is
// distinct from /readyz, which runs the periodic per-dep probe; we do not
// split /health into /live + /ready.
//
//	IsSelfProbeOK() == true  → 200 {"status":"healthy"}
//	IsSelfProbeOK() == false → 503 {"status":"unhealthy","reason":"self-probe failed"}
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

// Deprecated: use HealthHandler.
func HealthPassthrough() fiber.Handler {
	return HealthHandler()
}

// Register wires /readyz, /readyz/tenant/:id, /metrics and /health onto the
// given fiber app. The routes are registered before any auth middleware so
// Kubernetes and load-balancer probes can reach them without a token. The
// manager composes its own route table directly from these building blocks;
// Register is used by the worker micro-server.
func Register(app *fiber.App, h *Handler) {
	app.Get("/health", HealthHandler())
	app.Get("/readyz", h.Fiber())
	app.Get("/readyz/tenant/:id", StubTenantHandler())
	app.Get("/metrics", NewMetricsHandler())
}
