package readyz

import (
	"context"
	"time"
)

// DependencyChecker is the extension point for per-dependency probers wired
// into the /readyz handler.
//
// Name returns the stable, lowercase identifier used as the key in the
// /readyz "checks" map. It must be safe to call concurrently and must not
// block.
//
// Check is invoked concurrently, one goroutine per checker, under a context
// whose deadline equals PerDepTimeout(Name). Implementations must honor
// ctx.Done() and report Status="down" with an error mentioning "timeout"
// or "context deadline" on budget exhaustion — the handler trusts the
// returned DependencyCheck rather than measuring elapsed time itself.
// Check must not panic. Disabled optional deps return Status="skipped";
// deps that do not apply in the current deployment mode return
// Status="n/a"; both with a Reason.
type DependencyChecker interface {
	Name() string
	Check(ctx context.Context) DependencyCheck
}

// PerDepTimeout returns the per-dependency deadline. Values are fixed (not
// configurable) so every service exhibits the same /readyz latency envelope
// and dashboard thresholds remain portable across the fleet.
//
//	database  (postgres, mongodb, mysql, oracle, sqlserver) → 2s
//	cache     (redis, valkey)                               → 1s
//	queue     (rabbitmq, rabbitmq_*)                        → 2s
//	storage   (s3, seaweedfs)                               → 2s
//	http      (upstream_*, tenant_manager)                  → 1s
//	default                                                 → 2s
func PerDepTimeout(depName string) time.Duration {
	switch depName {
	case "postgres", "mongodb", "mysql", "oracle", "sqlserver":
		return 2 * time.Second
	case "redis", "valkey":
		return 1 * time.Second
	case "rabbitmq":
		return 2 * time.Second
	case "s3", "seaweedfs":
		return 2 * time.Second
	case "tenant_manager":
		return 1 * time.Second
	}

	if len(depName) >= len("upstream_") && depName[:len("upstream_")] == "upstream_" {
		return 1 * time.Second
	}

	if len(depName) >= len("rabbitmq_") && depName[:len("rabbitmq_")] == "rabbitmq_" {
		return 2 * time.Second
	}

	return 2 * time.Second
}

// StubChecker is a DependencyChecker that always returns Status="skipped"
// with the configured reason. It lets integration tests exercise the handler
// end-to-end without live dependencies.
type StubChecker struct {
	name   string
	reason string
}

func NewStubChecker(name, reason string) *StubChecker {
	return &StubChecker{name: name, reason: reason}
}

func (c *StubChecker) Name() string {
	return c.name
}

func (c *StubChecker) Check(_ context.Context) DependencyCheck {
	return DependencyCheck{
		Status: StatusSkipped,
		Reason: c.reason,
	}
}
