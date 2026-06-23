//go:build chaos

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"testing"
	"time"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	libLog "github.com/LerianStudio/lib-observability/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/LerianStudio/fetcher/v2/pkg/itestkit"
)

// =============================================================================
// TENANT MANAGER FLAPPING -> CONSUMER CONVERGENCE CHAOS TEST
// =============================================================================
//
// Validates that the worker's TenantConsumerReconciler converges the
// materialized per-tenant consumer set onto the Tenant Manager's active-tenant
// set even when the Tenant Manager is intermittently unreachable.
//
// Harness shape (mirrors tests/chaos: a backing service behind a Toxiproxy
// proxy, with chaos toxics applied through the itestkit Toxiproxy primitive):
//
//	reconciler --tmclient--> [Toxiproxy proxy "tm-source"] --> fake Tenant Manager
//	                                (cut / restore)              (host-side HTTP)
//
//   - The reconciler lives behind Go's internal/ boundary
//     (components/worker/internal/services), so a black-box tests/chaos package
//     CANNOT import NewTenantConsumerReconciler. This chaos test is therefore
//     co-located with the package under test (build tag `chaos`) and drives the
//     REAL reconciler directly — no production seam is added.
//   - The fake Tenant Manager is a real host-side net/http server bound to
//     0.0.0.0 so the Toxiproxy *container* reaches it via the Docker host
//     gateway (host.docker.internal:host-gateway, wired by NewToxiproxyChaos).
//   - itestkit.NewToxiproxyChaos stands up a standalone Toxiproxy container
//     (no infra suite needed); CreateProxy/CutConnection/RemoveAllToxics flap
//     availability — the same primitives tests/chaos is built on.
//   - The reconciler runs against a REAL tmclient.Client pointed at the proxy,
//     so a cut proxy produces genuine network failures on
//     GetActiveTenantsByService — the surface production hits.
//
// Determinism / speed: a SHORT reconcile interval + short accelerated flap
// cycles emulate the "TM 5s up / 5s down" intent in seconds, not minutes.
// Convergence is observed via require.Eventually on KnownTenants() with a
// comfortable budget after the TM is restored.
//
// This test changes NO production code.

const (
	// tmFlapProxyName is the Toxiproxy proxy that fronts the fake Tenant Manager.
	tmFlapProxyName = "tm-source"
	// tmFlapService is the service identifier passed to GetActiveTenantsByService.
	tmFlapService = "fetcher"
	// tmFlapAPIKey is a non-empty service API key (tmclient.NewClient requires one).
	tmFlapAPIKey = "chaos-tm-key"

	// tmFlapReconcileInterval is the accelerated steady-state reconcile cadence.
	tmFlapReconcileInterval = 200 * time.Millisecond
	// tmFlapGraceTicks keeps a known tenant alive across short outages: a tenant
	// must miss this many CONSECUTIVE passes before its consumer is reaped. The
	// property under test is "active tenants keep / regain a consumer across
	// flapping", not reaping, so a generous grace ensures a transient cut never
	// reaps a still-active tenant.
	tmFlapGraceTicks = 1000

	// tmFlapUp / tmFlapDown are the accelerated up/down phase durations emulating
	// the "5s up / 5s down" intent without spending real minutes.
	tmFlapUp   = 1500 * time.Millisecond
	tmFlapDown = 1500 * time.Millisecond
	// tmFlapCycles is the number of up/down cycles before the final
	// restore-and-converge assertion.
	tmFlapCycles = 3

	// tmFlapConvergeBudget is the comfortable budget for convergence assertions.
	tmFlapConvergeBudget = 20 * time.Second
	// tmFlapSetupTimeout bounds the Toxiproxy container startup.
	tmFlapSetupTimeout = 2 * time.Minute
)

// fakeTenantManager is a real host-side HTTP server answering the Tenant
// Manager active-tenants endpoint the tmclient calls
// (GET /v1/tenants/active?service=...). It is bound to 0.0.0.0 so the Toxiproxy
// container reaches it through the Docker host gateway. The active set is
// swapped under a RWMutex so the test goroutine can change what the reconciler
// sees without racing the server's handler goroutines.
type fakeTenantManager struct {
	srv      *http.Server
	listener net.Listener

	mu     sync.RWMutex
	active []string
}

// newFakeTenantManager starts the fake Tenant Manager on an ephemeral 0.0.0.0
// port and returns it with the port the Toxiproxy container targets via the
// host gateway. The caller must Close it (registered via t.Cleanup).
func newFakeTenantManager(t *testing.T, initial ...string) (*fakeTenantManager, int) {
	t.Helper()

	ln, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err, "listen for fake tenant manager")

	f := &fakeTenantManager{
		listener: ln,
		active:   append([]string(nil), initial...),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/tenants/active", f.handleActive)

	f.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		// http.ErrServerClosed is the expected return on graceful shutdown.
		_ = f.srv.Serve(ln)
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	t.Logf("Fake Tenant Manager listening on 0.0.0.0:%d", port)

	return f, port
}

// handleActive returns the current active tenant set as the tmclient expects:
// a JSON array of {id,name,status} objects.
func (f *fakeTenantManager) handleActive(w http.ResponseWriter, _ *http.Request) {
	f.mu.RLock()
	ids := append([]string(nil), f.active...)
	f.mu.RUnlock()

	summaries := make([]tmclient.TenantSummary, 0, len(ids))
	for _, id := range ids {
		summaries = append(summaries, tmclient.TenantSummary{ID: id, Name: id, Status: "ACTIVE"})
	}

	payload, err := json.Marshal(summaries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

// setActive swaps the active tenant set the server reports.
func (f *fakeTenantManager) setActive(ids ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.active = append([]string(nil), ids...)
}

// Close shuts the server down.
func (f *fakeTenantManager) Close(ctx context.Context) {
	_ = f.srv.Shutdown(ctx)
}

// chaosMaterializer is a thread-safe stand-in for the bootstrap
// workerMultiTenantConsumer the reconciler drives. It records the materialized
// consumer set without standing up real RabbitMQ consumers, isolating the
// reconcile-convergence property from broker wiring. It satisfies the method
// set the reconciler requires (EnsureConsumerStarted / StopConsumer /
// KnownTenants).
type chaosMaterializer struct {
	mu    sync.Mutex
	known map[string]bool
}

func newChaosMaterializer() *chaosMaterializer {
	return &chaosMaterializer{known: make(map[string]bool)}
}

func (m *chaosMaterializer) EnsureConsumerStarted(_ context.Context, tenantID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.known[tenantID] = true
}

func (m *chaosMaterializer) StopConsumer(tenantID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.known, tenantID)
}

func (m *chaosMaterializer) KnownTenants() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, 0, len(m.known))
	for id := range m.known {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	return ids
}

// TestTenantConsumerReconciler_TMFlapping_Converges verifies that, while the
// Tenant Manager flaps (repeated cut/restore cycles via Toxiproxy), the
// reconciler keeps converging the materialized consumer set onto the active
// tenant set, and that AFTER the Tenant Manager is restored every active tenant
// has a materialized consumer. It also asserts the reconciler leaks no
// goroutine after shutdown (goleak).
func TestTenantConsumerReconciler_TMFlapping_Converges(t *testing.T) {
	// Per-test leak verification. The standalone Toxiproxy chaos container keeps
	// long-lived background goroutines alive (testcontainers reaper/log pumps,
	// the toxiproxy HTTP client, OTel exporters). Those are NOT what this test
	// asserts on; ignore them by top function so the check targets the
	// reconciler's own loop + signal-handler goroutines.
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
		goleak.IgnoreTopFunction("net/http.(*persistConn).readLoop"),
		goleak.IgnoreTopFunction("net/http.(*persistConn).writeLoop"),
		goleak.IgnoreTopFunction("net/http.(*Server).Serve"),
		goleak.IgnoreTopFunction("net/http.(*connReader).backgroundRead"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
		// testcontainers keeps a Ryuk reaper connection goroutine alive for the
		// lifetime of the process; not owned by the reconciler.
		goleak.IgnoreTopFunction("github.com/testcontainers/testcontainers-go.(*Reaper).connect.func1"),
		// tmclient's in-memory config cache runs a background TTL cleanup loop
		// with no Close hook; not owned by the reconciler.
		goleak.IgnoreTopFunction("github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/cache.(*InMemoryCache).cleanupLoop"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// The TM starts with a SUBSET active, and a late tenant is introduced DURING
	// the flapping. Convergence-after-recovery is therefore a genuine assertion:
	// the reconciler must materialize a tenant it could only have learned about
	// after the TM recovered — it cannot pass trivially on the initial set.
	initialTenants := []string{"tenant-alpha", "tenant-beta"}
	lateTenant := "tenant-gamma"
	finalTenants := append(append([]string(nil), initialTenants...), lateTenant)

	// Phase 1: stand up the standalone Toxiproxy chaos container.
	t.Log("Phase 1: starting standalone Toxiproxy chaos container...")

	setupCtx, setupCancel := context.WithTimeout(ctx, tmFlapSetupTimeout)
	defer setupCancel()

	chaos, err := itestkit.NewToxiproxyChaos(setupCtx, itestkit.ChaosConfig{Enabled: true}, "")
	require.NoError(t, err, "start toxiproxy chaos")
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer closeCancel()

		_ = chaos.Close(closeCtx)
	})

	// Phase 2: stand up the fake Tenant Manager (host-side) and front it with a
	// Toxiproxy proxy so the reconciler's HTTP traffic is fault-injectable.
	t.Log("Phase 2: standing up fake Tenant Manager + Toxiproxy proxy...")

	fakeTM, tmPort := newFakeTenantManager(t, initialTenants...)
	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		fakeTM.Close(shutdownCtx)
	})

	// The upstream the Toxiproxy *container* dials must be a host address the
	// container can actually reach. Across environments this varies: Docker
	// Desktop on Linux uses host.docker.internal; native Linux uses the bridge
	// gateway 172.17.0.1; WSL2 + Docker Desktop resolves host.docker.internal to
	// an IPv6 address that an IPv4 listener can't answer, so the reachable
	// address there is the host's outbound IPv4. We therefore probe a candidate
	// list (preferring HostGatewayIP) and pick the first that is reachable
	// THROUGH the clean proxy, skipping the test if none is (the documented
	// host-networking limitation, same class as worker_circuitbreaker_test.go).
	upstreamHost, ref := resolveReachableUpstream(t, ctx, chaos, tmPort)
	t.Logf("Tenant Manager proxy %q listening at %s -> %s:%d", tmFlapProxyName, ref.ListenAddr, upstreamHost, tmPort)

	// Phase 3: a REAL tmclient pointed at the proxy. http:// is opt-in; a
	// service API key is required; no circuit breaker so every flap surfaces as
	// a raw network error to the reconciler (the property under test is the
	// reconciler's convergence, not the client's fast-fail).
	tmBaseURL := fmt.Sprintf("http://%s", ref.ListenAddr)
	lister, err := tmclient.NewClient(
		tmBaseURL,
		libLog.NewNop(),
		tmclient.WithAllowInsecureHTTP(),
		tmclient.WithServiceAPIKey(tmFlapAPIKey),
		tmclient.WithTimeout(2*time.Second),
	)
	require.NoError(t, err, "build tenant-manager client against proxy")

	// Sanity: with the proxy clean (no toxics), the client reaches the fake TM.
	summaries, err := lister.GetActiveTenantsByService(ctx, tmFlapService)
	require.NoError(t, err, "fake tenant manager must be reachable through a clean proxy")
	require.Len(t, summaries, len(initialTenants), "fake TM should report the seeded active tenants")

	// Phase 4: run the real reconciler against the proxied client with an
	// accelerated interval and a generous grace so a transient cut never reaps a
	// still-active tenant.
	mat := newChaosMaterializer()
	reconciler := NewTenantConsumerReconciler(
		mat,
		lister,
		tmFlapService,
		libLog.NewNop(),
		WithReconcileInterval(tmFlapReconcileInterval),
		WithReconcileGraceTicks(tmFlapGraceTicks),
	)

	runCtx, runCancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)
		// start drives the immediate bounded-retry pass + steady-state ticker
		// loop until runCtx is cancelled; it returns nil on graceful shutdown.
		// (Same-package access to the unexported loop entry — no production seam,
		// exactly as the package's unit tests drive it.)
		_ = reconciler.start(runCtx)
	}()

	// The initial active tenants should materialize before any chaos is applied.
	require.Eventually(t, func() bool {
		return equalStringSets(mat.KnownTenants(), initialTenants)
	}, tmFlapConvergeBudget, 50*time.Millisecond, "initial active tenants must materialize before flapping")

	// Phase 5: flap the Tenant Manager — accelerated cut/restore cycles emulate
	// the "5s up / 5s down" intent in seconds. During each DOWN window the
	// reconciler's list calls fail; the generous grace means no still-active
	// tenant is reaped, and each UP window lets the reconciler re-list cleanly.
	t.Logf("Phase 5: flapping Tenant Manager for %d cycles (%v down / %v up)...", tmFlapCycles, tmFlapDown, tmFlapUp)

	for cycle := 0; cycle < tmFlapCycles; cycle++ {
		require.NoError(t, chaos.CutConnection(ctx, tmFlapProxyName), "cut TM proxy")
		t.Logf("  cycle %d: TM DOWN", cycle+1)

		// While the TM is unreachable on the first cycle, expand its active set
		// with a late-arriving tenant. The reconciler cannot see it until the TM
		// recovers, so its eventual materialization proves real convergence
		// rather than mere retention of the initial set.
		if cycle == 0 {
			fakeTM.setActive(finalTenants...)
			t.Logf("  introduced late tenant %q while TM is DOWN", lateTenant)
		}

		sleepCtx(ctx, tmFlapDown)

		require.NoError(t, chaos.RemoveAllToxics(ctx, tmFlapProxyName), "restore TM proxy")
		t.Logf("  cycle %d: TM UP", cycle+1)
		sleepCtx(ctx, tmFlapUp)
	}

	// Phase 6: with the Tenant Manager finally restored, every active tenant
	// must have a materialized consumer — the convergence-after-recovery
	// property.
	t.Log("Phase 6: asserting convergence after recovery...")
	require.NoError(t, chaos.RemoveAllToxics(ctx, tmFlapProxyName), "final restore TM proxy")

	require.Eventually(t, func() bool {
		return equalStringSets(mat.KnownTenants(), finalTenants)
	}, tmFlapConvergeBudget, 50*time.Millisecond,
		"after TM recovery every active tenant (incl. the late one) must have a materialized consumer")

	assert.ElementsMatch(t, finalTenants, mat.KnownTenants(),
		"materialized consumer set must equal the full active tenant set after convergence")

	// Phase 7: graceful shutdown — cancel the reconciler and confirm its loop
	// goroutine exits promptly (goleak at the top then asserts no reconciler
	// goroutine survives).
	t.Log("Phase 7: shutting reconciler down...")
	runCancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reconciler did not exit after context cancel")
	}
}

// resolveReachableUpstream picks a host address the Toxiproxy container can
// actually reach the host-side fake TM on, and returns it together with the
// created proxy. It tries each candidate by creating a proxy and probing it
// with a plain HTTP GET through the clean proxy; the first reachable candidate
// wins. If none is reachable it SKIPS the test — this is the documented
// host-networking limitation (the toxiproxy container cannot reach a
// host-bound listener in some Docker/WSL setups), not a reconciler failure.
//
// Each candidate uses a distinct proxy name because the chaos interface has no
// delete; the chosen proxy keeps the canonical tmFlapProxyName so the flap
// phases drive it by that name. The chosen candidate's proxy is created last
// under tmFlapProxyName so cut/restore target it.
func resolveReachableUpstream(t *testing.T, ctx context.Context, chaos itestkit.ChaosInterface, port int) (string, itestkit.ProxyRef) {
	t.Helper()

	// Candidate host addresses, most-portable first. HostGatewayIP covers
	// Docker Desktop / native Linux; the host's outbound IPv4 covers WSL2 where
	// host.docker.internal resolves to an IPv6 an IPv4 listener can't answer.
	candidates := []string{itestkit.HostGatewayIP()}
	if outbound := outboundHostIPv4(); outbound != "" {
		candidates = append(candidates, outbound)
	}

	candidates = append(candidates, "172.17.0.1")

	for i, host := range candidates {
		name := fmt.Sprintf("%s-probe-%d", tmFlapProxyName, i)
		upstream := fmt.Sprintf("%s:%d", host, port)

		ref, err := chaos.CreateProxy(ctx, name, upstream)
		if err != nil {
			t.Logf("candidate %s: create proxy failed: %v", host, err)

			continue
		}

		if probeProxyReachable(ctx, ref.ListenAddr) {
			// Recreate under the canonical name so the flap phases drive it.
			canonical, err := chaos.CreateProxy(ctx, tmFlapProxyName, upstream)
			require.NoError(t, err, "create canonical TM proxy for %s", host)
			t.Logf("upstream %s is reachable through the proxy", host)

			return host, canonical
		}

		t.Logf("candidate %s: not reachable through the proxy", host)
	}

	t.Skipf("no host address reachable by the toxiproxy container (tried %v); "+
		"host-networking limitation, not a reconciler failure — build/vet verified instead", candidates)

	return "", itestkit.ProxyRef{}
}

// probeProxyReachable returns true if a plain HTTP GET to the active-tenants
// endpoint through listenAddr succeeds within a short budget.
func probeProxyReachable(ctx context.Context, listenAddr string) bool {
	url := fmt.Sprintf("http://%s/v1/tenants/active?service=%s", listenAddr, tmFlapService)
	client := &http.Client{Timeout: 2 * time.Second}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()

			return true
		}

		select {
		case <-ctx.Done():
			return false
		case <-time.After(750 * time.Millisecond):
		}
	}

	return false
}

// outboundHostIPv4 discovers the host's primary outbound IPv4 by opening a UDP
// socket toward a public address (no packets are sent). Containers on the same
// host can typically reach this address. Returns "" on failure.
func outboundHostIPv4() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}

	defer func() { _ = conn.Close() }()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || addr.IP.To4() == nil {
		return ""
	}

	return addr.IP.String()
}

// equalStringSets reports whether a and b contain the same elements (order
// independent). Both are expected to be de-duplicated already.
func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	set := make(map[string]bool, len(a))
	for _, v := range a {
		set[v] = true
	}

	for _, v := range b {
		if !set[v] {
			return false
		}
	}

	return true
}

// sleepCtx sleeps for d or until ctx is cancelled, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}
