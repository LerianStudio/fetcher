// Package readyz implements the canonical /readyz readiness endpoint for Lerian
// services, following the contract defined in the ring:dev-readyz skill.
//
// The package provides:
//   - Canonical response types with the five-value status vocabulary
//     (up / down / degraded / skipped / n/a).
//   - An atomic drain-state flag that causes the handler to short-circuit to 503
//     once the service begins graceful shutdown.
//   - A configuration loader that reads readyz-specific environment variables
//     without taking a hard dependency on the enclosing service's config struct.
//   - A DependencyChecker interface so per-dependency probers (Mongo, Redis,
//     RabbitMQ, S3, tenant-manager, upstream HTTP) can be plugged in without
//     changing the handler.
//   - A Fiber-compatible HTTP handler that runs all registered checkers in
//     parallel under per-dependency deadlines, aggregates their results, and
//     emits the canonical JSON response with the correct HTTP status code.
//
// This file defines the wire types. The handler in handler.go is responsible
// for producing instances of these types; callers consume them as read-only
// JSON payloads.
package readyz

// DependencyCheck is the per-dependency entry inside the /readyz response.
//
// The field rules mirror the ring:dev-readyz contract:
//   - Status is always populated with one of: "up", "down", "degraded",
//     "skipped", or "n/a". Any other value is a contract violation.
//   - LatencyMs is emitted only when a probe actually ran (Status "up" or
//     "degraded"). "omitempty" causes it to drop out of the JSON payload when
//     the probe was skipped or not applicable.
//   - TLS uses a pointer to differentiate three states: pointer-nil means the
//     dependency has no TLS concept (field is omitted from JSON); *false means
//     the dep is explicitly not using TLS (emitted as "tls": false); *true
//     means TLS is enabled (emitted as "tls": true). The field reflects the
//     configured TLS posture, NOT certificate validity.
//   - Error is present only for "down" or "degraded" and MUST NOT leak
//     credentials — it is surfaced to operators via Grafana/Loki.
//   - Reason is present only for "skipped" or "n/a" and explains why the check
//     did not run (feature flag off, multi-tenant carve-out, etc.).
//   - BreakerState is optional and is set only when the dependency is wrapped
//     by a circuit breaker. Values: "closed" / "half-open" / "open".
type DependencyCheck struct {
	Status       string `json:"status"`
	LatencyMs    int64  `json:"latency_ms,omitempty"`
	TLS          *bool  `json:"tls,omitempty"`
	Error        string `json:"error,omitempty"`
	Reason       string `json:"reason,omitempty"`
	BreakerState string `json:"breaker_state,omitempty"`
}

// ReadyzResponse is the top-level /readyz response payload.
//
// Aggregation rule (ring:dev-readyz contract):
//
//	Status == "healthy"  iff  every entry in Checks has Status in
//	                          {"up", "skipped", "n/a"}.
//	Any "down" or "degraded" forces Status == "unhealthy" and HTTP 503.
//
// Version is resolved at startup from OTEL_RESOURCE_SERVICE_VERSION, then
// VERSION, then "unknown". DeploymentMode reflects the DEPLOYMENT_MODE env var
// ("saas" / "byoc" / "local") and drives the SaaS-specific TLS enforcement in
// later gates.
type ReadyzResponse struct {
	Status         string                     `json:"status"`
	Checks         map[string]DependencyCheck `json:"checks"`
	Version        string                     `json:"version"`
	DeploymentMode string                     `json:"deployment_mode"`
	// TenantID is set only on /readyz/tenant/:id responses so a caller can
	// round-trip the probed tenant without parsing the URL. Omitted from
	// global /readyz to keep the shape stable for existing dashboards.
	TenantID string `json:"tenant_id,omitempty"`
}

// Status-vocabulary constants. Exported so tests and checker implementations
// refer to the same string literals the JSON contract mandates.
const (
	StatusUp       = "up"
	StatusDown     = "down"
	StatusDegraded = "degraded"
	StatusSkipped  = "skipped"
	StatusNA       = "n/a"

	// TopStatusHealthy / TopStatusUnhealthy are the only valid values for
	// ReadyzResponse.Status.
	TopStatusHealthy   = "healthy"
	TopStatusUnhealthy = "unhealthy"
)

// TLSPtr is a tiny helper to produce a *bool for the TLS field. Call sites
// read far better as `TLS: readyz.TLSPtr(true)` than as
// `TLS: func() *bool { b := true; return &b }()`.
func TLSPtr(v bool) *bool {
	return &v
}
