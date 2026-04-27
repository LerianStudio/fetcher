package readyz

import "context"

// NAChecker reports Status="n/a" for tenant-scoped deps on the global
// /readyz when the service runs in multi-tenant mode. Tenant-scoped deps
// must remain visible in the global response with a reason pointing to
// /readyz/tenant/:id, rather than being silently omitted. The optional
// TLS pointer surfaces the configured posture without probing.
type NAChecker struct {
	name   string
	reason string
	tls    *bool
}

func NewNAChecker(name, reason string, tls *bool) *NAChecker {
	return &NAChecker{name: name, reason: reason, tls: tls}
}

func (c *NAChecker) Name() string { return c.name }

func (c *NAChecker) Check(_ context.Context) DependencyCheck {
	return DependencyCheck{
		Status: StatusNA,
		Reason: c.reason,
		TLS:    c.tls,
	}
}
