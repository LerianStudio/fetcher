package readyz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// drainingDepName is the synthetic dependency emitted while the process is
// draining; defined as a constant so dashboard queries stay stable.
const drainingDepName = "draining"

// Handler runs every registered checker in parallel on every request — it
// has no caching, no TTL, no background state. Caching produces blind
// windows in which Kubernetes keeps routing traffic to a degraded pod.
// Clients must not rely on iteration order in the JSON response;
// dashboards key off the dependency name.
type Handler struct {
	cfg      *Config
	checkers []DependencyChecker
	now      func() time.Time
}

func NewHandler(cfg *Config, checkers ...DependencyChecker) *Handler {
	if cfg == nil {
		cfg = LoadConfig()
	}

	return &Handler{
		cfg:      cfg,
		checkers: checkers,
		now:      time.Now,
	}
}

// Fiber returns the handler suitable for router.Get("/readyz", h.Fiber()).
// While draining, real probers are skipped and a synthetic "draining" check
// is emitted (with metrics) so alerts still rate() during rolling deploys.
// Otherwise probes run in parallel; "healthy" → 200, "unhealthy" → 503.
func (h *Handler) Fiber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if IsDraining() {
			emitCheckDuration(drainingDepName, StatusDown, 0)
			emitCheckStatus(drainingDepName, StatusDown)

			return c.Status(fiber.StatusServiceUnavailable).JSON(h.buildDrainingResponse())
		}

		resp := h.Run(c.UserContext())
		if resp.Status == TopStatusHealthy {
			return c.Status(fiber.StatusOK).JSON(resp)
		}

		return c.Status(fiber.StatusServiceUnavailable).JSON(resp)
	}
}

// Run dispatches all checkers in parallel — one goroutine per checker, each
// under context.WithTimeout(PerDepTimeout(name)) — and aggregates results.
// The buffered channel plus WaitGroup.Wait avoids leaking slow goroutines.
// Exported so unit tests can invoke the handler without a Fiber app.
func (h *Handler) Run(ctx context.Context) ReadyzResponse {
	checks := make(map[string]DependencyCheck, len(h.checkers))

	if len(h.checkers) == 0 {
		return ReadyzResponse{
			Status:         TopStatusHealthy,
			Checks:         checks,
			Version:        h.cfg.Version,
			DeploymentMode: h.cfg.DeploymentMode,
		}
	}

	type result struct {
		name  string
		check DependencyCheck
	}

	results := make(chan result, len(h.checkers))

	var wg sync.WaitGroup

	for _, checker := range h.checkers {
		wg.Add(1)

		go func(c DependencyChecker) {
			defer wg.Done()

			name := c.Name()

			depCtx, cancel := context.WithTimeout(ctx, PerDepTimeout(name))
			defer cancel()

			// runWithDeadline guards aggregation from a checker that ignores
			// ctx — the handler substitutes a synthetic "down" rather than
			// blocking forever. Histogram records the wall-clock elapsed
			// (not the checker-reported LatencyMs) because operators care
			// about the end-to-end handler contribution.
			start := time.Now()
			check := runWithDeadline(depCtx, c)
			elapsed := time.Since(start)

			emitCheckDuration(name, check.Status, elapsed)
			emitCheckStatus(name, check.Status)

			if check.LatencyMs == 0 && (check.Status == StatusUp || check.Status == StatusDegraded) {
				check.LatencyMs = elapsed.Milliseconds()
			}

			results <- result{name: name, check: check}
		}(checker)
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
	}
}

// runWithDeadline enforces ctx at the boundary so a checker that ignores
// its deadline cannot block aggregation indefinitely. On deadline fire the
// handler substitutes a synthetic "down" with a timeout-flagged error so
// operators see a consistent failure shape.
//
// A child context is created and cancelled before returning. This is the
// only signal we have with the inner Check — well-behaved checkers will
// observe ctx.Done() and exit. The buffered done channel ensures even
// checkers that ignore ctx can still send their eventual result without
// leaking the goroutine on a channel with no receiver.
func runWithDeadline(ctx context.Context, c DependencyChecker) DependencyCheck {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan DependencyCheck, 1)

	go func() {
		done <- c.Check(childCtx)
	}()

	select {
	case res := <-done:
		return res
	case <-ctx.Done():
		return DependencyCheck{
			Status: StatusDown,
			Error:  fmt.Sprintf("check timeout: %s", ctx.Err()),
		}
	}
}

// aggregateStatus is "healthy" iff every check is in {up, skipped, n/a}.
// Any "down" or "degraded" flips the response to "unhealthy" (HTTP 503).
func aggregateStatus(checks map[string]DependencyCheck) string {
	for _, c := range checks {
		switch c.Status {
		case StatusDown, StatusDegraded:
			return TopStatusUnhealthy
		}
	}

	return TopStatusHealthy
}

// buildDrainingResponse emits a synthetic "draining" check with Status=down
// so the aggregation rule naturally yields "unhealthy" without special-case
// logic. HTTP 503 is set by the caller.
func (h *Handler) buildDrainingResponse() ReadyzResponse {
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
	}
}
