package readyz

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// prometheusCompatibilityBridge is the narrow compatibility seam for /metrics.
//
// lib-observability is the required instrumentation surface for service code,
// but it intentionally does not expose a Prometheus HTTP exposition handler or
// the Go runtime/process collectors that existing Fetcher Kubernetes dashboards
// scrape from /metrics. Replacing this endpoint with pure OTel export would be
// a deployment contract change, not a code cleanup. Keeping Prometheus here is
// therefore explicit adapter code: all readiness business logic calls the emit*
// helpers below, and only this bridge owns raw prometheus collectors.
//
// Metric names, labels, and histogram buckets are part of that platform
// contract — dashboards and alerts across the fleet expect them verbatim.
type prometheusCompatibilityBridge struct {
	checkDuration *prometheus.HistogramVec
	checkStatus   *prometheus.CounterVec
	selfProbe     *prometheus.GaugeVec
	registerOnce  sync.Once
}

func newPrometheusCompatibilityBridge() *prometheusCompatibilityBridge {
	return &prometheusCompatibilityBridge{
		checkDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "readyz_check_duration_ms",
				Help:    "Duration of /readyz dependency checks in milliseconds.",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000},
			},
			[]string{"dep", "status"},
		),
		checkStatus: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "readyz_check_status",
				Help: "Count of /readyz dependency check outcomes per dep and status.",
			},
			[]string{"dep", "status"},
		),
		selfProbe: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "selfprobe_result",
				Help: "Last startup self-probe result per dependency (1=up, 0=down).",
			},
			[]string{"dep"},
		),
	}
}

// register goes against prometheus.DefaultRegisterer because operator
// dashboards rely on the Go runtime collectors that promhttp.Handler() exposes
// from the default registry; a private registry would hide them.
func (b *prometheusCompatibilityBridge) register() {
	b.registerOnce.Do(func() {
		prometheus.DefaultRegisterer.MustRegister(
			b.checkDuration,
			b.checkStatus,
			b.selfProbe,
		)
	})
}

func (b *prometheusCompatibilityBridge) observeCheckDuration(dep, status string, d time.Duration) {
	b.checkDuration.
		WithLabelValues(dep, status).
		Observe(float64(d.Milliseconds()))
}

func (b *prometheusCompatibilityBridge) incrementCheckStatus(dep, status string) {
	b.checkStatus.
		WithLabelValues(dep, status).
		Inc()
}

func (b *prometheusCompatibilityBridge) setSelfProbeResult(dep string, up bool) {
	v := 0.0
	if up {
		v = 1.0
	}

	b.selfProbe.WithLabelValues(dep).Set(v)
}

// readyzMetrics owns the only allowed Prometheus compatibility surface in
// this package. Do not add raw collectors outside prometheusCompatibilityBridge.
var readyzMetrics = newPrometheusCompatibilityBridge()

func NewMetricsHandler() fiber.Handler {
	readyzMetrics.register()

	return adaptor.HTTPHandler(promhttp.Handler())
}

func emitCheckDuration(dep, status string, d time.Duration) {
	readyzMetrics.observeCheckDuration(dep, status, d)
}

// emitCheckStatus must be called unconditionally per probe — conditional
// emission breaks rate() queries because absent series default to zero.
func emitCheckStatus(dep, status string) {
	readyzMetrics.incrementCheckStatus(dep, status)
}

func emitSelfProbeResult(dep string, up bool) {
	readyzMetrics.setSelfProbeResult(dep, up)
}
