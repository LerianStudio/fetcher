package readyz

import (
	"context"
	"time"
)

// BreakerState is the local, package-scoped enumeration of circuit-breaker
// states. It exists so pkg/bootstrap/readyz does not need to import the
// fetcher's own pkg/rabbitmq (no cycle today, but the low-level /readyz
// package MUST stay at the bottom of the dependency graph — a direct import
// of a higher-level package would close off that guarantee).
//
// Callers (the manager and worker bootstraps) adapt their concrete
// CircuitState to this enum with a trivial mapping helper. See
// (pkg/rabbitmq).CircuitState for the producer side.
type BreakerState int

// Breaker state constants. Ordering mirrors the fetcher RabbitMQAdapter so
// mapping helpers can be written as a simple one-to-one switch; however,
// callers MUST go through the mapping function rather than casting, because
// the value layout is not part of a stable contract with pkg/rabbitmq.
const (
	// BreakerClosed means the breaker is closed — requests flow normally and
	// the checker will run a probe.
	BreakerClosed BreakerState = iota
	// BreakerOpen means the breaker has tripped — the checker skips the
	// probe and reports Status=down, BreakerState="open".
	BreakerOpen
	// BreakerHalfOpen means the breaker is probing for recovery — the
	// checker skips the real probe and reports Status=degraded,
	// BreakerState="half-open". A single concurrent probe elsewhere owns the
	// recovery decision.
	BreakerHalfOpen
)

// String returns the wire-format label ("closed" / "open" / "half-open")
// used in the /readyz DependencyCheck.BreakerState field. The mapping is
// authoritative — dashboards and alerts depend on these exact strings.
func (s BreakerState) String() string {
	switch s {
	case BreakerClosed:
		return "closed"
	case BreakerOpen:
		return "open"
	case BreakerHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// RabbitMQAdapterProbe is the minimal surface the RabbitMQAdapterChecker
// needs from the fetcher's RabbitMQAdapter. Splitting state-inspection from
// the probe lets the checker short-circuit the probe when the breaker is
// open/half-open (per the Gate 6 contract — probing an open breaker is both
// pointless and actively harmful, because the breaker treats every probe as
// a potential recovery attempt).
type RabbitMQAdapterProbe interface {
	// State returns the current breaker state. MUST be non-blocking —
	// implementations typically read an atomic.
	State() BreakerState
	// Ping executes a lightweight liveness probe (e.g. open+close a
	// channel, or a cheap declaration). It is only invoked when State()
	// returns BreakerClosed. MUST respect ctx.
	Ping(ctx context.Context) error
}

// RabbitMQAdapterChecker is the DependencyChecker for the single-tenant
// RabbitMQ connection. In multi-tenant mode this checker is NOT registered
// at the global /readyz — an NAChecker carves the dep out and the real
// per-tenant check runs at /readyz/tenant/:id (via TenantRabbitMQChecker).
//
// State semantics (per ring:dev-readyz Gate 6):
//
//	breaker=closed   → probe, map probe outcome; BreakerState="closed"
//	breaker=half-open → skip probe, Status=degraded, BreakerState="half-open"
//	breaker=open     → skip probe, Status=down,     BreakerState="open"
//
// The TLS field is derived from the supplied AMQP URL via the Gate 3
// detector; a scheme of "amqps" reports TLS=true.
type RabbitMQAdapterChecker struct {
	name    string
	adapter RabbitMQAdapterProbe
	url     string
}

// NewRabbitMQAdapterChecker constructs a checker under the dep name
// "rabbitmq". The url is used only for TLS posture detection.
func NewRabbitMQAdapterChecker(adapter RabbitMQAdapterProbe, url string) *RabbitMQAdapterChecker {
	return &RabbitMQAdapterChecker{
		name:    "rabbitmq",
		adapter: adapter,
		url:     url,
	}
}

// Name returns the stable dep identifier ("rabbitmq").
func (c *RabbitMQAdapterChecker) Name() string { return c.name }

// Check reads the breaker state first, then decides whether to probe.
//
// This ordering is deliberate — the ring:dev-readyz Gate 6 contract forbids
// probing an open or half-open breaker. An open-breaker probe would be
// rejected by the breaker itself (returning ErrCircuitOpen), yielding a
// confusing "failed probe" response for what is actually a healthy
// fail-fast. A half-open probe would be treated by the breaker as a
// recovery attempt, stealing the single in-flight slot from the real
// workload.
func (c *RabbitMQAdapterChecker) Check(ctx context.Context) DependencyCheck {
	tlsOn := tlsOrFalse(detectAMQPTLS(c.url))

	if c.adapter == nil {
		return DependencyCheck{
			Status: StatusDown,
			TLS:    TLSPtr(tlsOn),
			Error:  "rabbitmq adapter not initialized",
		}
	}

	switch c.adapter.State() {
	case BreakerOpen:
		return DependencyCheck{
			Status:       StatusDown,
			TLS:          TLSPtr(tlsOn),
			Error:        "circuit breaker open",
			BreakerState: BreakerOpen.String(),
		}
	case BreakerHalfOpen:
		return DependencyCheck{
			Status:       StatusDegraded,
			TLS:          TLSPtr(tlsOn),
			BreakerState: BreakerHalfOpen.String(),
		}
	}

	// BreakerClosed or unknown → run the probe.
	start := time.Now()
	err := c.adapter.Ping(ctx)
	elapsed := time.Since(start)

	if err == nil {
		return DependencyCheck{
			Status:       StatusUp,
			LatencyMs:    elapsed.Milliseconds(),
			TLS:          TLSPtr(tlsOn),
			BreakerState: BreakerClosed.String(),
		}
	}

	return DependencyCheck{
		Status:       StatusDown,
		LatencyMs:    elapsed.Milliseconds(),
		TLS:          TLSPtr(tlsOn),
		BreakerState: BreakerClosed.String(),
		Error:        classifyErr(ctx, err),
	}
}
