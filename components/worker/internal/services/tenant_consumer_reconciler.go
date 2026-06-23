package services

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LerianStudio/lib-commons/v5/commons"
	libBackoff "github.com/LerianStudio/lib-commons/v5/commons/backoff"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	obsRuntime "github.com/LerianStudio/lib-observability/runtime"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

const (
	defaultReconcileInterval      = 60 * time.Second
	defaultReconcileGraceTicks    = 3
	reconcileListTimeout          = 5 * time.Second
	reconcileImmediateMaxAttempts = 4
	reconcileImmediateBackoffBase = time.Second
)

const (
	// reconcilerMeterName is the OpenTelemetry meter name for reconciler metrics,
	// mirroring the package-path convention used elsewhere in the worker.
	reconcilerMeterName = "github.com/LerianStudio/fetcher/v2/components/worker/internal/services"

	metricReconcileTotal           = "worker_tenant_reconcile_total"
	metricReconcileDurationSeconds = "worker_tenant_reconcile_duration_seconds"
	metricConsumerActive           = "worker_tenant_consumer_active"

	// Low-cardinality attribute keys. There is deliberately NO tenant_id here.
	attrService = "service"
	attrResult  = "result"

	reconcileResultOK            = "ok"
	reconcileResultTMUnreachable = "tm_unreachable"
)

// reconcilerMetrics holds the OpenTelemetry instruments for the reconciler.
// Instruments are always non-nil: when no MeterProvider is supplied the
// instruments come from a no-op meter, so every record site is panic-free and
// branch-free. Cardinality is kept low — the only attributes are a constant
// service label and a bounded result label; tenant_id is never recorded.
type reconcilerMetrics struct {
	serviceAttr    attribute.KeyValue
	reconcileTotal otelmetric.Int64Counter
	reconcileDur   otelmetric.Float64Histogram
	consumerActive otelmetric.Int64Gauge
}

// newReconcilerMetrics builds the reconciler instruments from provider. A nil
// provider falls back to a no-op MeterProvider so the reconciler still works
// with zero-cost metrics — mirroring pkg/metrics.TenantMetrics. Instrument
// construction errors degrade to no-op instruments rather than failing
// construction; metrics are side-effects and must never block the worker.
func newReconcilerMetrics(provider otelmetric.MeterProvider, service string) *reconcilerMetrics {
	if provider == nil {
		provider = noop.NewMeterProvider()
	}

	meter := provider.Meter(reconcilerMeterName)

	reconcileTotal, err := meter.Int64Counter(
		metricReconcileTotal,
		otelmetric.WithDescription("Total tenant consumer reconcile passes by result"),
		otelmetric.WithUnit("{pass}"),
	)
	if err != nil {
		reconcileTotal, _ = noop.NewMeterProvider().Meter(reconcilerMeterName).Int64Counter(metricReconcileTotal)
	}

	reconcileDur, err := meter.Float64Histogram(
		metricReconcileDurationSeconds,
		otelmetric.WithDescription("Duration of a tenant consumer reconcile pass"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		reconcileDur, _ = noop.NewMeterProvider().Meter(reconcilerMeterName).Float64Histogram(metricReconcileDurationSeconds)
	}

	consumerActive, err := meter.Int64Gauge(
		metricConsumerActive,
		otelmetric.WithDescription("Number of materialized per-tenant consumers"),
		otelmetric.WithUnit("{consumer}"),
	)
	if err != nil {
		consumerActive, _ = noop.NewMeterProvider().Meter(reconcilerMeterName).Int64Gauge(metricConsumerActive)
	}

	return &reconcilerMetrics{
		serviceAttr:    attribute.String(attrService, service),
		reconcileTotal: reconcileTotal,
		reconcileDur:   reconcileDur,
		consumerActive: consumerActive,
	}
}

// recordPass records the per-pass counter (with the result label) and the
// duration histogram. Both carry the constant service attribute only.
func (m *reconcilerMetrics) recordPass(ctx context.Context, result string, seconds float64) {
	m.reconcileTotal.Add(ctx, 1, otelmetric.WithAttributes(m.serviceAttr, attribute.String(attrResult, result)))
	m.reconcileDur.Record(ctx, seconds, otelmetric.WithAttributes(m.serviceAttr))
}

// recordActiveConsumers sets the active-consumer gauge to count, labelled with
// the constant service attribute only.
func (m *reconcilerMetrics) recordActiveConsumers(ctx context.Context, count int) {
	m.consumerActive.Record(ctx, int64(count), otelmetric.WithAttributes(m.serviceAttr))
}

// tenantConsumerMaterializer is the subset of the bootstrap multi-tenant
// consumer the reconciler drives. The bootstrap workerMultiTenantConsumer
// satisfies it: EnsureConsumerStarted is idempotent, StopConsumer tears a
// tenant down, and KnownTenants reports the tenants currently materialized.
// The methods return nothing, mirroring the concrete consumer exactly.
type tenantConsumerMaterializer interface {
	EnsureConsumerStarted(ctx context.Context, tenantID string)
	StopConsumer(tenantID string)
	KnownTenants() []string
}

// TenantConsumerReconcilerOption configures a TenantConsumerReconciler at
// construction time.
type TenantConsumerReconcilerOption func(*TenantConsumerReconciler)

// WithReconcileInterval overrides the ticker interval (default 60s). A
// non-positive value is ignored.
func WithReconcileInterval(interval time.Duration) TenantConsumerReconcilerOption {
	return func(r *TenantConsumerReconciler) {
		if interval > 0 {
			r.interval = interval
		}
	}
}

// WithMeterProvider injects the OpenTelemetry MeterProvider the reconciler uses
// to build its instruments. A nil provider (or omitting this option) is safe:
// the reconciler falls back to a no-op meter, so metric emission is zero-cost
// and never panics. Reuse the worker's existing libOtel.Telemetry.MeterProvider
// at the construction site.
func WithMeterProvider(provider otelmetric.MeterProvider) TenantConsumerReconcilerOption {
	return func(r *TenantConsumerReconciler) {
		r.meterProvider = provider
	}
}

// WithReconcileGraceTicks overrides the number of consecutive misses a known
// tenant must accumulate before its consumer is stopped (default 3). A
// non-positive value is ignored.
func WithReconcileGraceTicks(graceTicks int) TenantConsumerReconcilerOption {
	return func(r *TenantConsumerReconciler) {
		if graceTicks > 0 {
			r.graceTicks = graceTicks
		}
	}
}

// TenantConsumerReconciler is a long-running launcher app that periodically
// reconciles the set of materialized per-tenant consumers against the active
// tenants reported by the Tenant Manager. It materializes consumers for newly
// active tenants (idempotently) and stops consumers for tenants that have been
// absent for graceTicks consecutive reconcile passes.
//
// The absence map is touched only by the single loop goroutine, so it needs no
// locking.
type TenantConsumerReconciler struct {
	materializer tenantConsumerMaterializer
	lister       activeTenantLister
	service      string
	logger       libLog.Logger
	interval     time.Duration
	graceTicks   int

	meterProvider otelmetric.MeterProvider
	metrics       *reconcilerMetrics

	absences map[string]int
}

// NewTenantConsumerReconciler builds a reconciler. service is the Tenant
// Manager service identifier (constant.ApplicationName). Defaults: interval
// 60s, graceTicks 3. A nil logger falls back to a no-op logger.
func NewTenantConsumerReconciler(
	materializer tenantConsumerMaterializer,
	lister activeTenantLister,
	service string,
	logger libLog.Logger,
	opts ...TenantConsumerReconcilerOption,
) *TenantConsumerReconciler {
	if logger == nil {
		logger = libLog.NewNop()
	}

	r := &TenantConsumerReconciler{
		materializer: materializer,
		lister:       lister,
		service:      service,
		logger:       logger,
		interval:     defaultReconcileInterval,
		graceTicks:   defaultReconcileGraceTicks,
		absences:     make(map[string]int),
	}

	for _, opt := range opts {
		opt(r)
	}

	// Built after options so WithMeterProvider is honored; a nil provider
	// yields a no-op meter inside newReconcilerMetrics.
	r.metrics = newReconcilerMetrics(r.meterProvider, service)

	return r
}

// Run satisfies the commons.Launcher app contract (Run(*commons.Launcher)
// error), mirroring TerminalEventRepairer so Task 1.1.3 can register it
// directly. It wires a cancelable context, a SIGINT/SIGTERM handler, and
// delegates to start.
func (r *TenantConsumerReconciler) Run(launcher *commons.Launcher) error {
	if r == nil || r.materializer == nil || r.lister == nil {
		return nil
	}

	logger := r.logger
	if launcher != nil && launcher.Logger != nil {
		logger = launcher.Logger
	}

	ctx := observability.ContextWithLogger(context.Background(), logger)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)

	obsRuntime.SafeGoWithContext(ctx, logger, "tenant-consumer-reconciler-signal-handler", obsRuntime.KeepRunning, func(ctx context.Context) {
		select {
		case <-sigs:
			cancel()
		case <-ctx.Done():
		}
	})

	return r.start(ctx)
}

// start runs the immediate bounded-retry pass, then the steady-state ticker
// loop, until ctx is cancelled. It returns nil on graceful shutdown.
func (r *TenantConsumerReconciler) start(ctx context.Context) error {
	r.immediatePass(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.reconcileOnce(ctx); err != nil {
				r.logger.Log(ctx, libLog.LevelWarn, "tenant consumer reconcile tick failed", libLog.Err(err))
			}
		}
	}
}

// immediatePass performs the boot-time reconcile with a bounded short retry. It
// retries ONLY on a Tenant Manager list error, up to reconcileImmediateMaxAttempts
// attempts, backing off with ExponentialWithJitter + WaitContext. If the budget
// is exhausted (TM still down), it falls through so the ticker loop can take
// over — boot is never blocked.
func (r *TenantConsumerReconciler) immediatePass(ctx context.Context) {
	for attempt := 0; attempt < reconcileImmediateMaxAttempts; attempt++ {
		if listErr := r.reconcileOnce(ctx); listErr == nil {
			return
		}

		if ctx.Err() != nil {
			return
		}

		delay := libBackoff.ExponentialWithJitter(reconcileImmediateBackoffBase, attempt)
		if waitErr := libBackoff.WaitContext(ctx, delay); waitErr != nil {
			return
		}
	}

	r.logger.Log(ctx, libLog.LevelWarn, "tenant consumer reconciler immediate pass exhausted retry budget; falling through to ticker loop",
		libLog.String("service", r.service),
		libLog.Int("attempts", reconcileImmediateMaxAttempts),
	)
}

// reconcileOnce lists the active tenants (under a short timeout) and reconciles
// the materialized consumer set against them. It returns the Tenant Manager
// list error (so the immediate pass can decide whether to retry and the ticker
// loop can log it); on success it returns nil. The loop always survives a list
// error — it is logged at WARN here and the next tick is the retry — and a list
// error never stops a consumer.
func (r *TenantConsumerReconciler) reconcileOnce(ctx context.Context) error {
	logger, tracer, _, _ := observability.NewTrackingFromContext(ctx)
	if logger == nil {
		logger = r.logger
	}

	ctx, span := tracer.Start(ctx, "service.tenant_consumer_reconciler.reconcile_once")
	defer span.End()

	start := time.Now()

	listCtx, cancel := context.WithTimeout(ctx, reconcileListTimeout)
	defer cancel()

	summaries, err := r.lister.GetActiveTenantsByService(listCtx, r.service)
	if err != nil {
		logger.Log(ctx, libLog.LevelWarn, "tenant manager unreachable during consumer reconcile; tm_unreachable",
			libLog.String("service", r.service),
			libLog.Err(err),
		)

		libOtel.HandleSpanError(span, "tenant manager unreachable during consumer reconcile", err)

		r.metrics.recordPass(ctx, reconcileResultTMUnreachable, time.Since(start).Seconds())

		return err
	}

	active := make(map[string]bool, len(summaries))

	for _, summary := range summaries {
		if summary == nil || summary.ID == "" {
			continue
		}

		active[summary.ID] = true
		delete(r.absences, summary.ID)
		r.materializer.EnsureConsumerStarted(ctx, summary.ID)
	}

	r.reapAbsentTenants(ctx, active, logger)

	r.metrics.recordPass(ctx, reconcileResultOK, time.Since(start).Seconds())
	r.metrics.recordActiveConsumers(ctx, len(r.materializer.KnownTenants()))

	return nil
}

// reapAbsentTenants increments the absence counter for every known tenant that
// is not in the active set, and stops the consumer once a tenant reaches
// graceTicks consecutive misses (resetting its counter).
func (r *TenantConsumerReconciler) reapAbsentTenants(ctx context.Context, active map[string]bool, logger libLog.Logger) {
	for _, tenantID := range r.materializer.KnownTenants() {
		if tenantID == "" || active[tenantID] {
			continue
		}

		r.absences[tenantID]++
		if r.absences[tenantID] < r.graceTicks {
			continue
		}

		logger.Log(ctx, libLog.LevelInfo, "stopping consumer for tenant absent beyond grace window",
			libLog.String("tenant_id", tenantID),
			libLog.Int("grace_ticks", r.graceTicks),
		)

		r.materializer.StopConsumer(tenantID)
		delete(r.absences, tenantID)
	}
}
