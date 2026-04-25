package readyz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// drainingDepName is the synthetic dependency name inserted into the response
// when the process is draining. Making it a constant keeps dashboard queries
// stable.
const drainingDepName = "draining"

// Handler is the /readyz implementation. It owns an immutable slice of
// DependencyChecker instances and the Config that was loaded at bootstrap.
//
// The Handler has NO internal caching, NO response TTL, and NO background
// state. Every incoming request runs every registered checker in parallel
// under its per-dependency deadline. This is mandatory per the
// ring:dev-readyz contract — response caching produces "blind windows"
// during which Kubernetes would keep routing traffic to a degraded pod.
type Handler struct {
	cfg      *Config
	checkers []DependencyChecker
	// now is injected for tests. Production code passes time.Now.
	now func() time.Time
}

// NewHandler constructs a readyz handler. Callers pass the bootstrap Config
// and an arbitrary number of DependencyChecker instances.
//
// The order of checkers is preserved in the JSON response only insofar as
// Go's map-iteration order allows (i.e. not at all); clients MUST NOT rely
// on ordering. Dashboards key off the dependency name.
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

// Fiber returns a fiber.Handler suitable for router.Get("/readyz", h.Fiber()).
//
// Ordering rules enforced here:
//  1. If IsDraining() == true, return 503 with a synthetic "draining" check
//     and skip all real probers. This gives Kubernetes the graceful-drain
//     window it needs. Gate 5 of ring:dev-readyz: the drain short-circuit
//     still emits the readyz_check_* metrics under dep="draining" so
//     operators can alert on rate() during rolling deploys.
//  2. Otherwise, run every checker in parallel and aggregate.
//  3. Emit HTTP 200 for status "healthy", HTTP 503 for status "unhealthy".
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

// Run executes all registered DependencyCheckers in parallel under their
// per-dep deadlines, aggregates the outcomes, and returns the canonical
// response. Exported so unit tests can invoke the handler without spinning
// up a Fiber app.
//
// Parallel dispatch strategy: one goroutine per checker, each under
// context.WithTimeout(PerDepTimeout(name)). Results are collected on a
// fixed-size channel to avoid the classic "slow goroutine leaks" pattern
// — WaitGroup.Wait() ensures every goroutine has returned before we close
// the channel.
func (h *Handler) Run(ctx context.Context) ReadyzResponse {
	checks := make(map[string]DependencyCheck, len(h.checkers))

	if len(h.checkers) == 0 {
		// No checkers registered yet. The response is "healthy" with an
		// empty checks map — this keeps the endpoint bootable during very
		// early startup (Gate 2) before Gate 6 wires the real probers.
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

			// Wrap the checker call so a misbehaving prober that ignores
			// its deadline never blocks aggregation indefinitely. We trust
			// the returned DependencyCheck but fall back to a synthetic
			// "down" if the checker blew past its deadline.
			start := time.Now()
			check := runWithDeadline(depCtx, c)
			elapsed := time.Since(start)

			// Gate 5 of ring:dev-readyz: emit the two handler-side metrics
			// for every check — including timeouts. The measured elapsed
			// time is what the histogram records (not the checker-reported
			// LatencyMs) because the operator is interested in the
			// end-to-end /readyz handler contribution.
			emitCheckDuration(name, check.Status, elapsed)
			emitCheckStatus(name, check.Status)

			// If the checker left LatencyMs at zero and the status is a
			// successful probe, default-fill from the wall-clock elapsed
			// so the JSON response stays informative. A non-zero value
			// from the checker is always preserved.
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

// runWithDeadline invokes the checker and enforces the context deadline at
// the boundary. If the checker returns before the deadline, its result is
// passed through as-is. If the deadline fires first, the handler substitutes
// a synthetic "down" entry with a timeout-flagged error message so the
// operator sees a consistent failure shape in Grafana regardless of whether
// the individual checker honoured its context.
func runWithDeadline(ctx context.Context, c DependencyChecker) DependencyCheck {
	done := make(chan DependencyCheck, 1)

	go func() {
		done <- c.Check(ctx)
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

// aggregateStatus implements the single aggregation rule mandated by the
// ring:dev-readyz contract: top-level status is "healthy" iff every check is
// in {up, skipped, n/a}. Any "down" or "degraded" flips the response to
// "unhealthy" (HTTP 503).
func aggregateStatus(checks map[string]DependencyCheck) string {
	for _, c := range checks {
		switch c.Status {
		case StatusDown, StatusDegraded:
			return TopStatusUnhealthy
		}
	}

	return TopStatusHealthy
}

// buildDrainingResponse returns the canned draining response. The response
// has a single synthetic "draining" check with Status="down" so the
// aggregation rule naturally produces "unhealthy" without any special-case
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
