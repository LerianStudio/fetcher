// Package readyz implements the canonical /readyz readiness endpoint for
// Lerian services. It exposes wire types, a DependencyChecker plug-in
// interface, atomic draining/self-probe state, an env-based Config loader,
// and a Fiber handler that runs every checker in parallel under
// per-dependency deadlines, aggregates the results, and emits the
// canonical JSON response with the appropriate HTTP status.
package readyz

// DependencyCheck is the per-dependency entry of the /readyz response.
//
//	Status       — one of {up, down, degraded, skipped, n/a}.
//	LatencyMs    — emitted only when a probe ran (up / degraded).
//	TLS          — three states via pointer: nil (omitted; no TLS concept),
//	               *false (explicitly off), *true (on). Reflects configured
//	               posture, not certificate validity.
//	Error        — present for down / degraded; must not leak credentials.
//	Reason       — present for skipped / n/a; explains why the probe did not run.
//	BreakerState — set only when the dep is wrapped by a circuit breaker:
//	               "closed" / "half-open" / "open".
type DependencyCheck struct {
	Status       string `json:"status"`
	LatencyMs    int64  `json:"latency_ms,omitempty"`
	TLS          *bool  `json:"tls,omitempty"`
	Error        string `json:"error,omitempty"`
	Reason       string `json:"reason,omitempty"`
	BreakerState string `json:"breaker_state,omitempty"`
}

// ReadyzResponse is the top-level /readyz response.
//
// Status is "healthy" iff every entry in Checks has Status in
// {"up", "skipped", "n/a"}; any "down" or "degraded" forces "unhealthy"
// and HTTP 503. TenantID is populated only on /readyz/tenant/:id and
// omitted from the global response to keep the shape stable for dashboards.
type ReadyzResponse struct {
	Status         string                     `json:"status"`
	Checks         map[string]DependencyCheck `json:"checks"`
	Version        string                     `json:"version"`
	DeploymentMode string                     `json:"deployment_mode"`
	TenantID       string                     `json:"tenant_id,omitempty"`
}

const (
	StatusUp       = "up"
	StatusDown     = "down"
	StatusDegraded = "degraded"
	StatusSkipped  = "skipped"
	StatusNA       = "n/a"

	TopStatusHealthy   = "healthy"
	TopStatusUnhealthy = "unhealthy"
)

// TLSPtr is sugar for taking the address of a bool literal at the call site.
func TLSPtr(v bool) *bool {
	return &v
}
