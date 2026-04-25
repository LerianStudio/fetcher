package readyz

import (
	"context"
	"time"
)

// DependencyChecker is the extension point for plugging per-dependency probers
// (Mongo, Redis, RabbitMQ, S3, tenant-manager, upstream HTTP, …) into the
// /readyz handler.
//
// Gate 2 ships the interface plus a StubChecker so the handler and the
// routing can be wired end-to-end. Gate 6 replaces the stubs with real
// implementations backed by circuit breakers and per-tenant carve-outs.
//
// Contract notes:
//
//   - Name() MUST return the stable, lowercase identifier that appears as the
//     key in the /readyz "checks" map (e.g. "mongodb", "redis",
//     "rabbitmq", "s3", "tenant_manager", "upstream_fees"). It is safe to
//     call from any goroutine and must not block.
//
//   - Check(ctx) is called concurrently, one goroutine per checker. Each call
//     is scoped to a context whose deadline is bounded by PerDepTimeout(Name).
//     Implementations MUST respect ctx.Done() and return a DependencyCheck
//     with Status="down" and an error message mentioning "timeout" or
//     "context deadline" when the budget is exceeded — the handler trusts the
//     returned DependencyCheck rather than measuring elapsed time itself.
//
//   - Check(ctx) MUST NOT panic. Internal errors should be reported as
//     Status="down" with a non-credential Error string. Optional deps that
//     were intentionally disabled should return Status="skipped" with a
//     Reason. Deps that do not apply in the current deployment mode (e.g.
//     a multi-tenant broker on a single-tenant instance) should return
//     Status="n/a" with a Reason.
type DependencyChecker interface {
	Name() string
	Check(ctx context.Context) DependencyCheck
}

// PerDepTimeout returns the per-dependency deadline mandated by the
// ring:dev-readyz contract. Values are fixed (not configurable) so every
// service in the fleet exhibits the same /readyz latency envelope, making
// dashboard thresholds portable.
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

	// Prefix-based fallback for conventional upstream names.
	if len(depName) >= len("upstream_") && depName[:len("upstream_")] == "upstream_" {
		return 1 * time.Second
	}

	if len(depName) >= len("rabbitmq_") && depName[:len("rabbitmq_")] == "rabbitmq_" {
		return 2 * time.Second
	}

	return 2 * time.Second
}

// StubChecker is a placeholder DependencyChecker used during Gate 2 while the
// real probers (Gate 6) are not yet implemented. It always returns
// Status="skipped" with the configured reason — this keeps the response
// well-formed and lets integration tests exercise the handler end-to-end
// without needing live dependencies.
type StubChecker struct {
	name   string
	reason string
}

// NewStubChecker constructs a StubChecker that always reports "skipped" with
// the given reason. Typical use during Gate 2:
//
//	readyz.NewStubChecker("mongodb", "pending Gate 6 implementation")
func NewStubChecker(name, reason string) *StubChecker {
	return &StubChecker{name: name, reason: reason}
}

// Name returns the dependency name used as the key in the /readyz response.
func (c *StubChecker) Name() string {
	return c.name
}

// Check returns the canned "skipped" response. It ignores ctx because the
// stub performs no I/O — this is safe: the handler still enforces the
// per-dep deadline upstream.
func (c *StubChecker) Check(_ context.Context) DependencyCheck {
	return DependencyCheck{
		Status: StatusSkipped,
		Reason: c.reason,
	}
}
