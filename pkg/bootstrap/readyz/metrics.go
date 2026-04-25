package readyz

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Gate 5 of ring:dev-readyz: Prometheus metric registration + emission.
//
// This file defines the three metrics mandated by the ring:dev-readyz
// contract and the helpers that emit them. The metric names, labels, and
// histogram buckets are part of the contract — operator dashboards and
// alerting rules across the Lerian fleet expect them verbatim:
//
//   - readyz_check_duration_ms (HistogramVec, labels dep+status)
//   - readyz_check_status      (CounterVec, labels dep+status)
//   - selfprobe_result         (GaugeVec, label dep)
//
// The handler (handler.go) emits the first two for every dep check on every
// /readyz request. The draining short-circuit also emits them under the
// synthetic dep="draining" so rolling deploys are visible in the same
// dashboards. Gate 7 will emit selfprobe_result from the startup self-probe.
//
// We register against prometheus.DefaultRegisterer on purpose: operator
// dashboards at Lerian routinely alert on Go runtime metrics (GC pause,
// goroutine count, memory) which promhttp.Handler() surfaces from the default
// registry. A package-private registry would have hidden those metrics from
// the operator tooling without any benefit.
var (
	// readyzCheckDurationMs measures the wall-clock duration of every /readyz
	// per-dep probe. The unit is MILLISECONDS — it is encoded in the metric
	// name suffix, and the buckets are sized for millisecond-scale latencies
	// (1 ms cache hit → 5 s timeout).
	readyzCheckDurationMs = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "readyz_check_duration_ms",
			Help:    "Duration of /readyz dependency checks in milliseconds.",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000},
		},
		[]string{"dep", "status"},
	)

	// readyzCheckStatus counts probe outcomes per dep × status. Dashboards use
	// rate() over this counter to chart failure rates over time and to drive
	// alerts on drift (e.g. a rising "degraded" rate for a given dep).
	readyzCheckStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "readyz_check_status",
			Help: "Count of /readyz dependency check outcomes per dep and status.",
		},
		[]string{"dep", "status"},
	)

	// selfProbeResult reports the last startup self-probe outcome per
	// dependency. Gate 7 will set this from the self-probe goroutine; Gate 5
	// registers the collector so the series is visible from first scrape.
	// Convention: 1.0 = up, 0.0 = down (matches the ring:dev-readyz contract).
	selfProbeResult = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "selfprobe_result",
			Help: "Last startup self-probe result per dependency (1=up, 0=down).",
		},
		[]string{"dep"},
	)

	// registerOnce serialises the MustRegister calls below so repeated
	// NewMetricsHandler invocations are safe. This matters in tests where
	// multiple HealthServer / Routes instances construct metric handlers in
	// the same process, and in production where the manager may wire
	// /metrics on both the main HTTP server and the worker micro-server
	// (they live in separate processes in practice, but the package-level
	// registration must still be idempotent).
	registerOnce sync.Once
)

// NewMetricsHandler returns the Fiber handler that serves the Prometheus
// exposition-format response. It registers the three readyz collectors into
// the default registry exactly once (sync.Once), then adapts the standard
// promhttp.Handler to Fiber via the official middleware/adaptor.
//
// Gate 5 of ring:dev-readyz: this replaces the Gate 2 MetricsPlaceholder. The
// function signature matches MetricsPlaceholder so the swap in Register and
// in the component-level bootstraps is a single-line change.
func NewMetricsHandler() fiber.Handler {
	registerOnce.Do(func() {
		prometheus.DefaultRegisterer.MustRegister(
			readyzCheckDurationMs,
			readyzCheckStatus,
			selfProbeResult,
		)
	})

	return adaptor.HTTPHandler(promhttp.Handler())
}

// emitCheckDuration records a per-dep probe duration in milliseconds. The
// handler calls this once per checker per /readyz request, including the
// synthetic "draining" entry on the drain short-circuit path. The value is
// derived from time.Duration so conversion errors are impossible.
func emitCheckDuration(dep, status string, d time.Duration) {
	readyzCheckDurationMs.
		WithLabelValues(dep, status).
		Observe(float64(d.Milliseconds()))
}

// emitCheckStatus increments the per-dep-per-status outcome counter. The
// handler calls this exactly once per checker per /readyz request, labelled
// with the final DependencyCheck.Status. Dashboards rely on this call being
// unconditional — conditional emission breaks rate() queries because absent
// series default to zero.
func emitCheckStatus(dep, status string) {
	readyzCheckStatus.
		WithLabelValues(dep, status).
		Inc()
}

// emitSelfProbeResult sets the self-probe gauge for the given dep. Gate 7 of
// ring:dev-readyz calls this from the RunSelfProbe loop after each dep is
// probed.
//
// Contract: up=true → value 1.0, up=false → value 0.0.
func emitSelfProbeResult(dep string, up bool) {
	v := 0.0
	if up {
		v = 1.0
	}

	selfProbeResult.WithLabelValues(dep).Set(v)
}
