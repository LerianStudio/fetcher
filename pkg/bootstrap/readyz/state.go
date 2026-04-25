package readyz

import "sync/atomic"

// drainingState holds the global draining flag for the process. It is flipped
// to true once SIGTERM arrives and the graceful-drain window begins. While
// true, the /readyz handler MUST short-circuit to 503 so Kubernetes removes
// the pod from the Service endpoints before connections are closed.
//
// The variable is package-level (not per-Handler) on purpose: draining is a
// process-wide condition, and both the manager HTTP server and the worker
// health-port micro-server share the same lifecycle. The actual SIGTERM
// wiring happens in Gate 7 — Gate 2 provides the storage and the helpers.
var drainingState atomic.Bool

// SetDraining flips the process-wide drain flag. When called with true, the
// /readyz handler immediately begins returning 503 with a synthetic
// "draining" dependency entry.
func SetDraining(v bool) {
	drainingState.Store(v)
}

// IsDraining returns the current value of the drain flag. Safe for concurrent
// use from any goroutine.
func IsDraining() bool {
	return drainingState.Load()
}

// selfProbeOK tracks whether the service has passed its startup self-probe.
// /health consults this flag via HealthHandler (Gate 7 of ring:dev-readyz) —
// when false, /health returns 503 so Kubernetes' livenessProbe restarts the
// pod. Default is FALSE ("unhealthy until proven otherwise"): a service that
// never ran the Gate 7 startup self-probe must not serve traffic. Tests that
// exercise /health in isolation therefore MUST explicitly SetSelfProbe(true)
// in their setup — the previous Gate 2 optimistic default was a temporary
// placeholder while the self-probe pipeline was being wired.
var selfProbeOK atomic.Bool

// SetSelfProbe updates the self-probe flag. Gate 7 of ring:dev-readyz flips
// this to true from RunSelfProbe after every registered dependency checker
// reports up / skipped / n/a. Flipping to false causes HealthHandler to
// return 503 immediately.
func SetSelfProbe(v bool) {
	selfProbeOK.Store(v)
}

// IsSelfProbeOK returns the current value of the self-probe flag.
func IsSelfProbeOK() bool {
	return selfProbeOK.Load()
}
