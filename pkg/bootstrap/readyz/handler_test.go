package readyz

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// fakeChecker is a test double that returns a canned DependencyCheck and
// optionally sleeps to simulate slow/hung probers.
type fakeChecker struct {
	name  string
	out   DependencyCheck
	sleep time.Duration
}

func (f *fakeChecker) Name() string { return f.name }
func (f *fakeChecker) Check(ctx context.Context) DependencyCheck {
	if f.sleep == 0 {
		return f.out
	}
	select {
	case <-time.After(f.sleep):
		return f.out
	case <-ctx.Done():
		return DependencyCheck{Status: StatusDown, Error: "ctx cancelled in fake"}
	}
}

func baseCfg() *Config {
	return &Config{
		DeploymentMode: DeploymentModeLocal,
		HealthPort:     4007,
		DrainDelay:     12 * time.Second,
		Version:        "test-1.0.0",
	}
}

// resetDrainingAndCleanup clears the package-level draining flag at test
// entry and registers a Cleanup to clear it again when the test exits.
// Called by every test in this file (and tenant_handler_test.go) that
// exercises the Run/Fiber paths so that any future addition of t.Parallel()
// will not race on the shared atomic.Bool.
func resetDrainingAndCleanup(t *testing.T) {
	t.Helper()
	SetDraining(false)
	t.Cleanup(func() { SetDraining(false) })
}

func TestHandler_Run_AllUp_IsHealthy(t *testing.T) {
	resetDrainingAndCleanup(t)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "mongodb", out: DependencyCheck{Status: StatusUp, LatencyMs: 3, TLS: TLSPtr(true)}},
		&fakeChecker{name: "redis", out: DependencyCheck{Status: StatusUp, LatencyMs: 1}},
	)

	resp := h.Run(context.Background())

	assert.Equal(t, TopStatusHealthy, resp.Status)
	assert.Equal(t, "test-1.0.0", resp.Version)
	assert.Equal(t, DeploymentModeLocal, resp.DeploymentMode)
	require.Contains(t, resp.Checks, "mongodb")
	require.Contains(t, resp.Checks, "redis")
	assert.Equal(t, StatusUp, resp.Checks["mongodb"].Status)
}

func TestHandler_Run_AnyDown_IsUnhealthy(t *testing.T) {
	resetDrainingAndCleanup(t)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "mongodb", out: DependencyCheck{Status: StatusUp}},
		&fakeChecker{name: "rabbitmq", out: DependencyCheck{Status: StatusDown, Error: "dial refused"}},
	)

	resp := h.Run(context.Background())

	assert.Equal(t, TopStatusUnhealthy, resp.Status)
	assert.Equal(t, "dial refused", resp.Checks["rabbitmq"].Error)
}

func TestHandler_Run_AnyDegraded_IsUnhealthy(t *testing.T) {
	resetDrainingAndCleanup(t)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "upstream_fees", out: DependencyCheck{
			Status:       StatusDegraded,
			BreakerState: "half-open",
			LatencyMs:    12,
		}},
	)

	resp := h.Run(context.Background())

	assert.Equal(t, TopStatusUnhealthy, resp.Status)
	assert.Equal(t, "half-open", resp.Checks["upstream_fees"].BreakerState)
}

func TestHandler_Run_AllSkipped_IsHealthy(t *testing.T) {
	resetDrainingAndCleanup(t)

	h := NewHandler(baseCfg(),
		NewStubChecker("mongodb", "pending"),
		NewStubChecker("redis", "pending"),
	)

	resp := h.Run(context.Background())

	assert.Equal(t, TopStatusHealthy, resp.Status)
}

func TestHandler_Run_NAStatusCountsAsHealthy(t *testing.T) {
	resetDrainingAndCleanup(t)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "rabbitmq", out: DependencyCheck{
			Status: StatusNA,
			Reason: "multi-tenant: see /readyz/tenant/:id",
			TLS:    TLSPtr(true),
		}},
	)

	resp := h.Run(context.Background())

	assert.Equal(t, TopStatusHealthy, resp.Status)
}

func TestHandler_Run_CheckerBlocksPastDeadline_ReportsDown(t *testing.T) {
	resetDrainingAndCleanup(t)
	// 'redis' has a 1s deadline. A fake that sleeps 3s forces the handler
	// to fall back to its synthetic timeout response.
	h := NewHandler(baseCfg(),
		&fakeChecker{
			name:  "redis",
			out:   DependencyCheck{Status: StatusUp},
			sleep: 3 * time.Second,
		},
	)

	start := time.Now()
	resp := h.Run(context.Background())
	elapsed := time.Since(start)

	// Must have bailed out at the per-dep deadline (≈1s) — not waited 3s.
	assert.Less(t, elapsed, 2*time.Second, "handler did not honour per-dep deadline")
	assert.Equal(t, TopStatusUnhealthy, resp.Status)
	assert.Equal(t, StatusDown, resp.Checks["redis"].Status)
	assert.Contains(t, strings.ToLower(resp.Checks["redis"].Error), "timeout")
}

func TestHandler_Run_NoCheckersRegistered_IsHealthyWithEmptyChecks(t *testing.T) {
	resetDrainingAndCleanup(t)

	h := NewHandler(baseCfg())

	resp := h.Run(context.Background())

	assert.Equal(t, TopStatusHealthy, resp.Status)
	assert.Empty(t, resp.Checks)
}

func TestHandler_Fiber_DrainingReturns503AndSkipsCheckers(t *testing.T) {
	SetDraining(false)
	t.Cleanup(func() { SetDraining(false) })

	var calls atomic.Int32

	checker := &fakeCheckerCounted{
		fake:  &fakeChecker{name: "mongodb", out: DependencyCheck{Status: StatusUp}},
		calls: &calls,
	}

	h := NewHandler(baseCfg(), checker)
	app := fiber.New()
	app.Get("/readyz", h.Fiber())

	SetDraining(true)

	req := httptest.NewRequest("GET", "/readyz", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusServiceUnavailable, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var parsed ReadyzResponse
	require.NoError(t, json.Unmarshal(body, &parsed))

	assert.Equal(t, TopStatusUnhealthy, parsed.Status)
	require.Contains(t, parsed.Checks, "draining")
	assert.Equal(t, StatusDown, parsed.Checks["draining"].Status)
	assert.Contains(t, parsed.Checks["draining"].Reason, "graceful drain")

	// The real checker MUST NOT have been called during draining.
	assert.Equal(t, int32(0), calls.Load(), "checkers executed while draining")
}

type fakeCheckerCounted struct {
	fake  *fakeChecker
	calls *atomic.Int32
}

func (f *fakeCheckerCounted) Name() string { return f.fake.Name() }
func (f *fakeCheckerCounted) Check(ctx context.Context) DependencyCheck {
	f.calls.Add(1)
	return f.fake.Check(ctx)
}

func TestHandler_Fiber_HealthyPath_Returns200(t *testing.T) {
	SetDraining(false)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "mongodb", out: DependencyCheck{Status: StatusUp, LatencyMs: 3, TLS: TLSPtr(true)}},
	)

	app := fiber.New()
	app.Get("/readyz", h.Fiber())

	req := httptest.NewRequest("GET", "/readyz", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var parsed ReadyzResponse
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Equal(t, TopStatusHealthy, parsed.Status)
	assert.Equal(t, "test-1.0.0", parsed.Version)
	assert.Equal(t, DeploymentModeLocal, parsed.DeploymentMode)
}

func TestHandler_Fiber_UnhealthyPath_Returns503(t *testing.T) {
	SetDraining(false)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "mongodb", out: DependencyCheck{Status: StatusDown, Error: "boom"}},
	)

	app := fiber.New()
	app.Get("/readyz", h.Fiber())

	req := httptest.NewRequest("GET", "/readyz", nil)
	res, err := app.Test(req, 2000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusServiceUnavailable, res.StatusCode)
}

func TestHandler_Run_ConcurrentRequests_NoRace(t *testing.T) {
	SetDraining(false)

	h := NewHandler(baseCfg(),
		&fakeChecker{name: "mongodb", out: DependencyCheck{Status: StatusUp}},
		&fakeChecker{name: "redis", out: DependencyCheck{Status: StatusUp}},
		&fakeChecker{name: "rabbitmq", out: DependencyCheck{Status: StatusUp}},
	)

	var wg sync.WaitGroup

	for range 32 {
		wg.Go(func() {
			for range 10 {
				resp := h.Run(context.Background())
				assert.Equal(t, TopStatusHealthy, resp.Status)
			}
		})
	}

	wg.Wait()
}

func TestAggregateStatus_MatchesContract(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]DependencyCheck
		want string
	}{
		{
			name: "all up",
			in: map[string]DependencyCheck{
				"a": {Status: StatusUp}, "b": {Status: StatusUp},
			},
			want: TopStatusHealthy,
		},
		{
			name: "up+skipped",
			in: map[string]DependencyCheck{
				"a": {Status: StatusUp}, "b": {Status: StatusSkipped},
			},
			want: TopStatusHealthy,
		},
		{
			name: "up+n/a",
			in: map[string]DependencyCheck{
				"a": {Status: StatusUp}, "b": {Status: StatusNA},
			},
			want: TopStatusHealthy,
		},
		{
			name: "one down",
			in: map[string]DependencyCheck{
				"a": {Status: StatusUp}, "b": {Status: StatusDown},
			},
			want: TopStatusUnhealthy,
		},
		{
			name: "one degraded",
			in: map[string]DependencyCheck{
				"a": {Status: StatusUp}, "b": {Status: StatusDegraded},
			},
			want: TopStatusUnhealthy,
		},
		{
			name: "empty is healthy",
			in:   map[string]DependencyCheck{},
			want: TopStatusHealthy,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, aggregateStatus(tc.in))
		})
	}
}

// blockingChecker blocks until release is closed OR ctx is cancelled.
// It records whether ctx cancellation was observed so tests can assert that
// the deadline-driven cancel actually propagates into the checker — which is
// the contract the runWithDeadline fix enforces.
type blockingChecker struct {
	name        string
	release     chan struct{}
	ctxObserved chan struct{}
	once        bool
}

func (b *blockingChecker) Name() string { return b.name }

func (b *blockingChecker) Check(ctx context.Context) DependencyCheck {
	select {
	case <-b.release:
		return DependencyCheck{Status: StatusUp}
	case <-ctx.Done():
		if !b.once {
			b.once = true
			close(b.ctxObserved)
		}

		return DependencyCheck{Status: StatusDown, Error: "ctx cancelled"}
	}
}

// TestRunWithDeadline_LeakBoundedByCtxCancel is the global-handler counterpart
// of TestRunTenantWithDeadline_LeakBoundedByCtxCancel. The same goroutine
// leak shape exists in handler.go's runWithDeadline; the fix introduces a
// child cancellable context that is cancelled BEFORE returning from the
// deadline branch so the inner goroutine receives a cancel signal even when
// the outer caller has moved on.
func TestRunWithDeadline_LeakBoundedByCtxCancel(t *testing.T) {
	ck := &blockingChecker{
		name:        "redis",
		release:     make(chan struct{}),
		ctxObserved: make(chan struct{}),
	}

	t.Cleanup(func() {
		select {
		case <-ck.release:
		default:
			close(ck.release)
		}
	})

	defer goleak.VerifyNone(t, goleakIgnores()...)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	res := runWithDeadline(ctx, ck)

	require.Equal(t, StatusDown, res.Status, "deadline branch must report down")
	require.Contains(t, res.Error, "check timeout")

	select {
	case <-ck.ctxObserved:
		// Good: ctx cancel propagated into the checker, bounding the leak.
	case <-time.After(time.Second):
		t.Fatal("checker did not observe ctx cancel within 1s — leak is unbounded")
	}
}

func TestRegistration_StubTenantHandler_ReturnsSkipped(t *testing.T) {
	app := fiber.New()
	app.Get("/readyz/tenant/:id", StubTenantHandler())

	req := httptest.NewRequest("GET", "/readyz/tenant/abc", nil)
	res, err := app.Test(req, 1000)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, fiber.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var parsed DependencyCheck
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Equal(t, StatusSkipped, parsed.Status)
	assert.Equal(t, "pending Gate 6 implementation", parsed.Reason)
}

// Gate 5 of ring:dev-readyz: MetricsPlaceholder was removed; /metrics is now
// served by NewMetricsHandler (see metrics_test.go for the behavioural
// coverage of the Prometheus exposition output).

func TestRegistration_Register_WiresAllRoutes(t *testing.T) {
	SetDraining(false)

	// Gate 7 of ring:dev-readyz: /health is gated on the startup self-probe.
	// Without this, /health would return 503 because selfProbeOK defaults to
	// false. Tests that exercise /health end-to-end must pre-flip the flag,
	// the same way RunSelfProbe does after a successful boot.
	SetSelfProbe(true)
	t.Cleanup(func() { SetSelfProbe(false) })

	app := fiber.New()
	h := NewHandler(baseCfg(),
		&fakeChecker{name: "mongodb", out: DependencyCheck{Status: StatusUp}},
	)
	Register(app, h)

	for _, path := range []string{"/health", "/readyz", "/metrics", "/readyz/tenant/foo"} {
		req := httptest.NewRequest("GET", path, nil)
		res, err := app.Test(req, 2000)
		require.NoError(t, err, "path %s", path)
		_ = res.Body.Close()
		assert.Equal(t, fiber.StatusOK, res.StatusCode, "path %s", path)
	}
}
