package readyz

import (
	"context"
	"time"
)

// BreakerState is a package-local enum of circuit-breaker states. Defining
// it here keeps pkg/bootstrap/readyz from importing higher-level packages
// like pkg/rabbitmq, preserving its position at the bottom of the dependency
// graph. Callers convert their concrete state through a mapping helper —
// the value layout is intentionally not a stable contract.
type BreakerState int

const (
	BreakerClosed BreakerState = iota
	BreakerOpen
	BreakerHalfOpen
)

// String returns the wire-format label used in DependencyCheck.BreakerState.
// Dashboards and alerts depend on these exact strings.
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

// RabbitMQAdapterProbe is the minimal surface the checker requires.
// Separating State from Ping lets the checker short-circuit when the breaker
// is open or half-open: probing an open breaker yields a misleading "failed
// probe" for healthy fail-fast, and probing a half-open breaker steals the
// single recovery slot from the real workload.
type RabbitMQAdapterProbe interface {
	State() BreakerState
	Ping(ctx context.Context) error
}

// RabbitMQAdapterChecker probes the single-tenant RabbitMQ connection. In
// multi-tenant mode it is replaced at the global /readyz by an NAChecker;
// the real per-tenant check runs via TenantRabbitMQChecker.
//
//	breaker=closed    → probe, map probe outcome
//	breaker=half-open → skip probe, Status=degraded
//	breaker=open      → skip probe, Status=down
type RabbitMQAdapterChecker struct {
	name    string
	adapter RabbitMQAdapterProbe
	url     string
}

// NewRabbitMQAdapterChecker reports under the dep name "rabbitmq". The url is
// used only for TLS posture detection.
func NewRabbitMQAdapterChecker(adapter RabbitMQAdapterProbe, url string) *RabbitMQAdapterChecker {
	return &RabbitMQAdapterChecker{
		name:    "rabbitmq",
		adapter: adapter,
		url:     url,
	}
}

func (c *RabbitMQAdapterChecker) Name() string { return c.name }

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
