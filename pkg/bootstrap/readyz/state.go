package readyz

import "sync/atomic"

// drainingState is process-wide because draining is a process-wide
// condition shared by the manager HTTP server and the worker health-port
// micro-server. While true, /readyz short-circuits to 503 so Kubernetes
// removes the pod from Service endpoints before connections close.
var drainingState atomic.Bool

func SetDraining(v bool) {
	drainingState.Store(v)
}

func IsDraining() bool {
	return drainingState.Load()
}

// selfProbeOK defaults to false — a service that never ran the startup
// self-probe must not serve traffic. Tests that exercise /health in
// isolation must call SetSelfProbe(true) in setup.
var selfProbeOK atomic.Bool

func SetSelfProbe(v bool) {
	selfProbeOK.Store(v)
}

func IsSelfProbeOK() bool {
	return selfProbeOK.Load()
}
