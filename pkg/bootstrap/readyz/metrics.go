package readyz

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric names, labels, and histogram buckets are part of the platform
// contract — dashboards and alerts across the fleet expect them verbatim.
// Registration goes against prometheus.DefaultRegisterer because operator
// dashboards rely on the Go runtime collectors that promhttp.Handler()
// exposes from the default registry; a private registry would hide them.
var (
	// readyzCheckDurationMs records per-dep probe duration. Unit is
	// milliseconds, encoded in the name suffix; buckets span the
	// realistic range from a 1ms cache hit to a 5s timeout.
	readyzCheckDurationMs = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "readyz_check_duration_ms",
			Help:    "Duration of /readyz dependency checks in milliseconds.",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000},
		},
		[]string{"dep", "status"},
	)

	readyzCheckStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "readyz_check_status",
			Help: "Count of /readyz dependency check outcomes per dep and status.",
		},
		[]string{"dep", "status"},
	)

	// selfProbeResult: 1.0 = up, 0.0 = down.
	selfProbeResult = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "selfprobe_result",
			Help: "Last startup self-probe result per dependency (1=up, 0=down).",
		},
		[]string{"dep"},
	)

	// registerOnce makes MustRegister idempotent so repeated
	// NewMetricsHandler invocations (tests, multi-server bootstraps) do not
	// panic on duplicate collector registration.
	registerOnce sync.Once
)

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

func emitCheckDuration(dep, status string, d time.Duration) {
	readyzCheckDurationMs.
		WithLabelValues(dep, status).
		Observe(float64(d.Milliseconds()))
}

// emitCheckStatus must be called unconditionally per probe — conditional
// emission breaks rate() queries because absent series default to zero.
func emitCheckStatus(dep, status string) {
	readyzCheckStatus.
		WithLabelValues(dep, status).
		Inc()
}

func emitSelfProbeResult(dep string, up bool) {
	v := 0.0
	if up {
		v = 1.0
	}

	selfProbeResult.WithLabelValues(dep).Set(v)
}
