package readyz

import "context"

// NAChecker reports status "n/a" — used for tenant-scoped deps on the GLOBAL
// /readyz endpoint when the service runs in multi-tenant mode.
//
// The ring:dev-readyz contract forbids silently omitting a tenant-scoped dep
// from the global response: operators must see it, labelled n/a with a reason
// pointing them to /readyz/tenant/:id. A separate checker type (rather than
// a parameter on the real prober) keeps the wiring explicit — the bootstrap
// code picks between NAChecker and the real checker at construction time.
//
// NAChecker performs no I/O and is safe to register under any deadline. The
// optional TLS pointer lets callers surface the configured TLS posture even
// when the probe does not run — useful for Mongo/Rabbit in multi-tenant mode
// where all tenants share the same TLS configuration.
type NAChecker struct {
	name   string
	reason string
	tls    *bool
}

// NewNAChecker constructs a carve-out checker that always reports "n/a" with
// the supplied reason. Typical use at bootstrap:
//
//	readyz.NewNAChecker("mongodb", "multi-tenant: see /readyz/tenant/:id", nil)
func NewNAChecker(name, reason string, tls *bool) *NAChecker {
	return &NAChecker{name: name, reason: reason, tls: tls}
}

// Name returns the dep-name key used in the /readyz "checks" map.
func (c *NAChecker) Name() string { return c.name }

// Check ignores ctx and returns the canned n/a entry. The handler's per-dep
// deadline is harmless here — no I/O is attempted.
func (c *NAChecker) Check(_ context.Context) DependencyCheck {
	return DependencyCheck{
		Status: StatusNA,
		Reason: c.reason,
		TLS:    c.tls,
	}
}
