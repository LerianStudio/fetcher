package services

import (
	"context"
	"errors"
	"sync"
	"syscall"
	"testing"
	"time"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// fakeMaterializer is a test double for tenantConsumerMaterializer. It records
// EnsureConsumerStarted / StopConsumer calls and exposes a configurable known
// set. All access is guarded by a mutex because the reconciler invokes it from
// its own goroutine while the test reads from the test goroutine.
type fakeMaterializer struct {
	mu      sync.Mutex
	started []string
	stopped []string
	known   map[string]bool
}

func newFakeMaterializer(known ...string) *fakeMaterializer {
	m := &fakeMaterializer{known: make(map[string]bool)}
	for _, id := range known {
		m.known[id] = true
	}

	return m
}

func (m *fakeMaterializer) EnsureConsumerStarted(_ context.Context, tenantID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.started = append(m.started, tenantID)
	m.known[tenantID] = true
}

func (m *fakeMaterializer) StopConsumer(tenantID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopped = append(m.stopped, tenantID)
	delete(m.known, tenantID)
}

func (m *fakeMaterializer) KnownTenants() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, 0, len(m.known))
	for id := range m.known {
		ids = append(ids, id)
	}

	return ids
}

func (m *fakeMaterializer) startedSnapshot() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]string(nil), m.started...)
}

func (m *fakeMaterializer) stoppedSnapshot() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]string(nil), m.stopped...)
}

// fakeTenantLister is a test double for activeTenantLister. Each call to
// GetActiveTenantsByService pops the next scripted response (tenants + error).
// When the script is exhausted it repeats the last entry, so a steady-state
// tenant set can be expressed with a single trailing entry.
type fakeTenantLister struct {
	mu        sync.Mutex
	responses []listerResponse
	calls     int
}

type listerResponse struct {
	tenants []*tmclient.TenantSummary
	err     error
}

func (l *fakeTenantLister) GetActiveTenantsByService(_ context.Context, _ string) ([]*tmclient.TenantSummary, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.calls++

	idx := len(l.responses) - 1
	if l.calls-1 < len(l.responses) {
		idx = l.calls - 1
	}

	if idx < 0 {
		return nil, nil
	}

	resp := l.responses[idx]

	return resp.tenants, resp.err
}

func (l *fakeTenantLister) callCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.calls
}

func tenants(ids ...string) []*tmclient.TenantSummary {
	out := make([]*tmclient.TenantSummary, 0, len(ids))
	for _, id := range ids {
		out = append(out, &tmclient.TenantSummary{ID: id, Status: "ACTIVE"})
	}

	return out
}

// WithReconcileInterval: a non-positive interval is ignored and the package
// default (60s) is kept. This guards the time.NewTicker(0) panic path in start.
func TestWithReconcileInterval_NonPositiveKeepsDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		interval time.Duration
	}{
		{name: "zero", interval: 0},
		{name: "negative", interval: -1 * time.Second},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewTenantConsumerReconciler(
				newFakeMaterializer(),
				&fakeTenantLister{},
				"fetcher",
				testLogger(),
				WithReconcileInterval(tt.interval),
			)

			require.Equal(t, defaultReconcileInterval, r.interval,
				"non-positive interval must keep the package default")
		})
	}
}

// WithReconcileGraceTicks: a non-positive grace value is ignored and the package
// default (3) is kept.
func TestWithReconcileGraceTicks_NonPositiveKeepsDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		graceTicks int
	}{
		{name: "zero", graceTicks: 0},
		{name: "negative", graceTicks: -1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewTenantConsumerReconciler(
				newFakeMaterializer(),
				&fakeTenantLister{},
				"fetcher",
				testLogger(),
				WithReconcileGraceTicks(tt.graceTicks),
			)

			require.Equal(t, defaultReconcileGraceTicks, r.graceTicks,
				"non-positive grace ticks must keep the package default")
		})
	}
}

// reconcileOnce: nil and empty-ID summaries in a single pass are skipped (no
// panic, not materialized); only the well-formed tenant is materialized.
func TestTenantConsumerReconciler_ReconcileOnce_SkipsNilAndEmptyIDSummaries(t *testing.T) {
	t.Parallel()

	mat := newFakeMaterializer()
	lister := &fakeTenantLister{responses: []listerResponse{{
		tenants: []*tmclient.TenantSummary{nil, {ID: ""}, {ID: "t-a"}},
	}}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger())

	require.NoError(t, r.reconcileOnce(testContext()))

	require.Equal(t, []string{"t-a"}, mat.startedSnapshot(),
		"nil and empty-ID summaries must be skipped; only t-a materialized")
	require.Empty(t, mat.stoppedSnapshot())
}

// reconcileOnce: a fresh active tenant is materialized via EnsureConsumerStarted.
func TestTenantConsumerReconciler_ReconcileOnce_MaterializesNewTenants(t *testing.T) {
	t.Parallel()

	mat := newFakeMaterializer()
	lister := &fakeTenantLister{responses: []listerResponse{{tenants: tenants("tenant-a", "tenant-b")}}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger())

	require.NoError(t, r.reconcileOnce(testContext()))

	require.ElementsMatch(t, []string{"tenant-a", "tenant-b"}, mat.startedSnapshot())
	require.Empty(t, mat.stoppedSnapshot())
}

// reconcileOnce: a TM list error surfaces to the caller (so the immediate pass
// can retry and the ticker loop can log it) but must NOT stop any consumer and
// must NOT materialize anything — the loop survives the next tick.
func TestTenantConsumerReconciler_ReconcileOnce_TenantManagerError_LoopSurvives(t *testing.T) {
	t.Parallel()

	mat := newFakeMaterializer("tenant-a")
	listErr := errors.New("tm 503 unreachable")
	lister := &fakeTenantLister{responses: []listerResponse{{err: listErr}}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger())

	require.ErrorIs(t, r.reconcileOnce(testContext()), listErr)

	require.Empty(t, mat.startedSnapshot(), "a list error must not materialize")
	require.Empty(t, mat.stoppedSnapshot(), "a list error must never stop a consumer")
}

// reconcileOnce: a known tenant that disappears from the active set is only
// stopped after graceTicks consecutive misses.
func TestTenantConsumerReconciler_ReconcileOnce_StopsAfterGraceTicks(t *testing.T) {
	t.Parallel()

	mat := newFakeMaterializer("tenant-gone")
	// Always returns an empty active set -> tenant-gone is missing every tick.
	lister := &fakeTenantLister{responses: []listerResponse{{tenants: tenants()}}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger(), WithReconcileGraceTicks(3))
	ctx := testContext()

	require.NoError(t, r.reconcileOnce(ctx))
	require.Empty(t, mat.stoppedSnapshot(), "miss 1 must not stop")

	require.NoError(t, r.reconcileOnce(ctx))
	require.Empty(t, mat.stoppedSnapshot(), "miss 2 must not stop")

	require.NoError(t, r.reconcileOnce(ctx))
	require.Equal(t, []string{"tenant-gone"}, mat.stoppedSnapshot(), "miss 3 must stop")
}

// reconcileOnce: a tenant that reappears before grace resets its counter and is
// never stopped.
func TestTenantConsumerReconciler_ReconcileOnce_ReappearsBeforeGrace_NotStopped(t *testing.T) {
	t.Parallel()

	mat := newFakeMaterializer("tenant-flap")
	lister := &fakeTenantLister{responses: []listerResponse{
		{tenants: tenants()},              // miss 1
		{tenants: tenants()},              // miss 2
		{tenants: tenants("tenant-flap")}, // reappears -> reset
		{tenants: tenants()},              // miss 1 again
		{tenants: tenants()},              // miss 2 again
	}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger(), WithReconcileGraceTicks(3))
	ctx := testContext()

	for i := 0; i < 5; i++ {
		require.NoError(t, r.reconcileOnce(ctx))
	}

	require.Empty(t, mat.stoppedSnapshot(), "counter reset on reappearance must prevent stop")
}

// Immediate pass: transient 5xx errors within the retry budget recover and the
// tenant is eventually materialized without blocking boot.
func TestTenantConsumerReconciler_Start_ImmediatePassRecoversFromTransient5xx(t *testing.T) {
	defer goleak.VerifyNone(t)

	mat := newFakeMaterializer()
	lister := &fakeTenantLister{responses: []listerResponse{
		{err: errors.New("tm 500")},
		{err: errors.New("tm 503")},
		{tenants: tenants("tenant-late")},
	}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger(),
		WithReconcileInterval(time.Hour), // keep the ticker out of the way
	)

	ctx, cancel := context.WithCancel(testContext())

	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = r.start(ctx)
	}()

	require.Eventually(t, func() bool {
		return len(mat.startedSnapshot()) == 1
	}, 15*time.Second, 10*time.Millisecond, "immediate pass should retry past transient 5xx and materialize")

	require.GreaterOrEqual(t, lister.callCount(), 3)

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("start did not return after context cancel")
	}

	require.Equal(t, []string{"tenant-late"}, mat.startedSnapshot())
}

// Cancellation: the loop goroutine exits promptly when the context is cancelled
// (goleak.VerifyNone asserts no goroutine survives this test).
func TestTenantConsumerReconciler_Start_ExitsOnContextCancel(t *testing.T) {
	defer goleak.VerifyNone(t)

	mat := newFakeMaterializer()
	lister := &fakeTenantLister{responses: []listerResponse{{tenants: tenants("tenant-a")}}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger(),
		WithReconcileInterval(20*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(testContext())

	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = r.start(ctx)
	}()

	// Let a few ticks fire.
	require.Eventually(t, func() bool {
		return len(mat.startedSnapshot()) >= 1
	}, 2*time.Second, 5*time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("start did not return after context cancel")
	}
}

// Run: a reconciler missing its required collaborators returns nil immediately
// without starting any goroutine (the launcher app contract tolerates a no-op).
func TestTenantConsumerReconciler_Run_NilDependencies_NoOp(t *testing.T) {
	t.Parallel()

	r := NewTenantConsumerReconciler(nil, nil, "fetcher", testLogger())

	require.NoError(t, r.Run(nil))
}

// Run: the launcher app contract — Run materializes the active tenants on the
// immediate pass and returns nil when the process receives SIGTERM, mirroring
// TerminalEventRepairer so Task 1.1.3 can register it directly. This test is
// intentionally NOT parallel: it drives a process-wide SIGTERM, so it must run
// in the serial phase where no paused parallel sibling can observe the signal.
func TestTenantConsumerReconciler_Run_HonorsSignalShutdown(t *testing.T) {
	defer goleak.VerifyNone(t)

	mat := newFakeMaterializer()
	lister := &fakeTenantLister{responses: []listerResponse{{tenants: tenants("tenant-a")}}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger(),
		WithReconcileInterval(time.Hour),
	)

	done := make(chan error, 1)

	go func() {
		done <- r.Run(nil)
	}()

	require.Eventually(t, func() bool {
		return len(mat.startedSnapshot()) == 1
	}, 5*time.Second, 10*time.Millisecond, "Run should materialize active tenants on the immediate pass")

	require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after SIGTERM")
	}
}
