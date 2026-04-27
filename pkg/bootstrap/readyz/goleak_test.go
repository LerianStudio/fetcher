//go:build !chaos && !e2e

package readyz

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// =============================================================================
// GOROUTINE LEAK DETECTION — Gate 8, Layer 9
// =============================================================================
//
// The readyz handler spawns one goroutine per registered checker every request
// (see handler.go: Handler.Run, runWithDeadline, and selfprobe.go:
// RunSelfProbe). A misbehaving prober that ignores its context could leak a
// goroutine per request — in production that would be a SEV-1 memory leak on
// any service behind a Kubernetes readinessProbe (probes fire every few
// seconds per pod).
//
// These tests use go.uber.org/goleak to assert the hot paths do not leak
// goroutines even under pathological inputs (hung probers, exceeded per-dep
// deadlines, drain short-circuit, self-probe fanout, tenant handler fanout).
//
// Per-test goleak.VerifyNone is preferred over goleak.VerifyTestMain because:
//   1. It isolates each leak to a specific exercise, making root cause obvious.
//   2. It doesn't fail the whole package if an unrelated test leaves a known
//      benign goroutine parked (e.g. metrics emission with a lazy OTel exporter
//      that is process-wide).
//
// Covered paths (minimum 5 per spec):
//   1. Handler.Run happy path — all checkers report up instantly.
//   2. Handler.Run with a checker that EXCEEDS its per-dep deadline — the
//      most dangerous path because a leaked goroutine would spam production.
//   3. Handler.Fiber drain short-circuit — must not even spawn checker
//      goroutines.
//   4. RunSelfProbe — same fanout as Handler.Run but on the startup path.
//   5. TenantFiberHandler.runTenantChecks — per-tenant fanout.
//
// Known-safe background goroutines are declared in goleakIgnores and shared
// across every assertion. These are process-level singletons owned by imported
// libraries (fasthttp's date-updater loop) that we cannot and should not
// terminate from test code.

// goleakIgnores declares goroutines we intentionally exclude from leak
// detection. These are parked by third-party libraries (fasthttp) at process
// scope — once the first fiber.App.Test runs, a singleton goroutine loops
// forever updating Date headers. It is not owned by any /readyz code path.
func goleakIgnores() []goleak.Option {
	return []goleak.Option{
		// fasthttp parks a permanent "update server date" goroutine once its
		// HTTP server code is first exercised. It is a well-known background
		// task documented by goleak users across the ecosystem. Using
		// IgnoreAnyFunction because the goroutine's top-of-stack is the stdlib
		// time.Sleep — the signature frame is updateServerDate.func1 deeper in.
		goleak.IgnoreAnyFunction("github.com/valyala/fasthttp.updateServerDate.func1"),
	}
}

// leakySleepChecker sleeps for a configurable duration, respecting ctx. It is
// the worst case for leak detection: if the handler's deadline mechanism is
// broken, the checker's internal goroutine (inside runWithDeadline) will leak
// because the outer WaitGroup never observes Done().
//
// Note: this fake DOES respect ctx — which is exactly the scenario we want
// covered. The handler uses a buffered done channel (size 1) inside
// runWithDeadline precisely so the spawned goroutine can send its result and
// exit even after the caller moved on.
type leakySleepChecker struct {
	name  string
	sleep time.Duration
	out   DependencyCheck
}

func (l *leakySleepChecker) Name() string { return l.name }

func (l *leakySleepChecker) Check(ctx context.Context) DependencyCheck {
	if l.sleep == 0 {
		return l.out
	}
	select {
	case <-time.After(l.sleep):
		return l.out
	case <-ctx.Done():
		return DependencyCheck{Status: StatusDown, Error: "ctx cancelled"}
	}
}

// TestHandler_Run_NoGoroutineLeaks_HappyPath exercises the handler's parallel
// fanout with trivially-passing checkers and asserts the goroutine count
// returns to baseline after Run returns.
func TestHandler_Run_NoGoroutineLeaks_HappyPath(t *testing.T) {
	defer goleak.VerifyNone(t, goleakIgnores()...)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "mongodb", out: DependencyCheck{Status: StatusUp}},
		&fakeChecker{name: "redis", out: DependencyCheck{Status: StatusUp}},
		&fakeChecker{name: "rabbitmq", out: DependencyCheck{Status: StatusUp}},
	)

	resp := h.Run(context.Background())
	require.Equal(t, TopStatusHealthy, resp.Status)
}

// TestHandler_Run_NoGoroutineLeaks_DeadlineExceeded exercises the dangerous
// path: a checker that would take longer than its per-dep deadline. The
// handler's runWithDeadline must release the outer wg.Done() even though the
// inner probe goroutine is still sleeping. The sleep here is LONGER than
// PerDepTimeout("redis")=1s to force the deadline branch.
//
// If goleak reports a leak here, the handler's runWithDeadline is broken —
// specifically the buffered done-channel + select pattern.
func TestHandler_Run_NoGoroutineLeaks_DeadlineExceeded(t *testing.T) {
	defer goleak.VerifyNone(t, goleakIgnores()...)

	h := NewHandler(baseCfg(),
		&leakySleepChecker{
			name:  "redis", // PerDepTimeout == 1s
			sleep: 3 * time.Second,
			out:   DependencyCheck{Status: StatusUp}, // ignored — deadline fires first
		},
	)

	resp := h.Run(context.Background())
	require.Equal(t, TopStatusUnhealthy, resp.Status)
	require.Equal(t, StatusDown, resp.Checks["redis"].Status)
}

// TestHandler_Fiber_Drain_NoGoroutineLeaks asserts the drain short-circuit
// does not spawn checker goroutines. Under drain, Fiber() must return 503
// from buildDrainingResponse WITHOUT invoking Run — verified by the absence
// of leaked goroutines from a checker that would sleep forever if it ran.
func TestHandler_Fiber_Drain_NoGoroutineLeaks(t *testing.T) {
	defer goleak.VerifyNone(t, goleakIgnores()...)

	SetDraining(true)
	t.Cleanup(func() { SetDraining(false) })

	// Register a checker that would hang if ever called. If drain short-circuit
	// is broken, the goroutine this spawns would leak.
	hang := &leakySleepChecker{
		name:  "mongodb",
		sleep: 5 * time.Minute,
	}
	h := NewHandler(baseCfg(), hang)

	app := fiber.New()
	app.Get("/readyz", h.Fiber())

	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, 2000)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode,
		"draining handler should return 503")
}

// TestRunSelfProbe_NoGoroutineLeaks exercises the startup fanout path. Same
// parallel goroutine-per-checker pattern as Handler.Run — same leak risk.
func TestRunSelfProbe_NoGoroutineLeaks(t *testing.T) {
	defer goleak.VerifyNone(t, goleakIgnores()...)

	// Mix a hung checker (forced to timeout) with a passing one so both
	// branches of the success-aggregation and failure-aggregation paths run.
	err := RunSelfProbe(context.Background(),
		[]DependencyChecker{
			&fakeChecker{name: "mongodb", out: DependencyCheck{Status: StatusUp}},
			&leakySleepChecker{
				name:  "redis",
				sleep: 3 * time.Second, // > PerDepTimeout("redis")=1s
			},
		},
		libLog.NewNop(),
	)
	// Probe is expected to fail because the redis checker blew past its budget,
	// but that is orthogonal to the goleak assertion.
	require.Error(t, err, "self-probe should fail when a checker exceeds its deadline")
	// Reset state so subsequent tests see a clean slate.
	SetSelfProbe(false)
}

// TestRunSelfProbe_EmptyCheckers_NoGoroutineLeaks verifies the fast path
// (zero checkers) also returns cleanly. This is the path bootstrap hits when
// a service has no external deps at all.
func TestRunSelfProbe_EmptyCheckers_NoGoroutineLeaks(t *testing.T) {
	defer goleak.VerifyNone(t, goleakIgnores()...)

	err := RunSelfProbe(context.Background(), nil, libLog.NewNop())
	require.NoError(t, err)
	require.True(t, IsSelfProbeOK())
	SetSelfProbe(false)
}

// TestTenantHandler_Run_NoGoroutineLeaks exercises the per-tenant fanout path
// (tenant_handler.go). Same goroutine-per-checker pattern as Handler.Run, but
// with the tenant-scoped context. A leak here would affect every tenant in
// every /readyz/tenant/:id call.
func TestTenantHandler_Run_NoGoroutineLeaks(t *testing.T) {
	defer goleak.VerifyNone(t, goleakIgnores()...)

	tm := &fakeTMClient{tenants: []*tmclient.TenantSummary{{ID: "t1"}}}

	h := NewTenantHandler(baseCfg(), tm, "fetcher",
		&staticTenantChecker{name: "mongodb", check: DependencyCheck{Status: StatusUp}},
		&staticTenantChecker{name: "rabbitmq", check: DependencyCheck{Status: StatusUp}},
	)

	resp := h.runTenantChecks(context.Background(), "t1")
	require.Equal(t, TopStatusHealthy, resp.Status)
	require.Equal(t, "t1", resp.TenantID)
}

// TestTenantHandler_Fiber_NoGoroutineLeaks runs the full Fiber path — tenant
// existence lookup + checker fanout — and asserts no goroutine leaks. This is
// the path Kubernetes hits on /readyz/tenant/:id probes in multi-tenant mode.
func TestTenantHandler_Fiber_NoGoroutineLeaks(t *testing.T) {
	defer goleak.VerifyNone(t, goleakIgnores()...)

	tm := &fakeTMClient{tenants: []*tmclient.TenantSummary{{ID: "t1"}}}

	h := NewTenantHandler(baseCfg(), tm, "fetcher",
		&staticTenantChecker{name: "mongodb", check: DependencyCheck{Status: StatusUp}},
	)

	app := fiber.New()
	app.Get("/readyz/tenant/:id", h.Fiber())

	req := httptest.NewRequest("GET", "/readyz/tenant/t1", nil)
	resp, err := app.Test(req, 3_000)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}
