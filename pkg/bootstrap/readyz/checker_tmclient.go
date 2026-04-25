package readyz

import (
	"context"
	"errors"
	"time"

	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
)

// TMClient is the narrow Tenant-Manager-client surface needed by the
// readiness checker AND the per-tenant handler's tenant-existence validation.
// The real *tmclient.Client satisfies this — tests inject a fake.
//
// GetActiveTenantsByService is chosen because it is the same call the
// worker's initial-tenant-sync uses on startup; reusing it in /readyz
// exercises the cache warming behaviour in both codepaths.
type TMClient interface {
	GetActiveTenantsByService(ctx context.Context, service string) ([]*tmclient.TenantSummary, error)
}

// TenantManagerClientChecker probes the Tenant Manager HTTP service. The
// client is wrapped by a circuit breaker whose state is OPAQUE (the tmclient
// package does not expose a State() method). The contract is therefore
// "probe-and-catch": issue the call, and if the error chain contains
// tmcore.ErrCircuitBreakerOpen, classify the response as breaker=open.
//
// The checker is wired ONLY when MULTI_TENANT_ENABLED=true. The bootstrap
// constructs an instance with enabled=true on the multi-tenant path and
// omits it entirely on the single-tenant path (so /readyz does not emit a
// "tenant_manager" dep at all).
type TenantManagerClientChecker struct {
	name    string
	client  TMClient
	service string
	url     string
	enabled bool
}

// NewTenantManagerClientChecker constructs a probe-and-catch checker.
//
// Parameters:
//   - client:   the tmclient. May be nil only when enabled=false.
//   - service:  the service name to probe (fetcher uses constant.ApplicationName
//     = "fetcher"). Used as the GetActiveTenantsByService argument.
//   - url:      the Tenant Manager base URL; used only for TLS detection.
//   - enabled:  the MULTI_TENANT_ENABLED flag. When false the checker returns
//     Status="skipped" with a reason — we don't omit the dep so operators
//     can distinguish "tenant manager is unused" from "dep forgot to wire".
func NewTenantManagerClientChecker(client TMClient, service, url string, enabled bool) *TenantManagerClientChecker {
	return &TenantManagerClientChecker{
		name:    "tenant_manager",
		client:  client,
		service: service,
		url:     url,
		enabled: enabled,
	}
}

// Name returns the stable dep identifier ("tenant_manager").
func (c *TenantManagerClientChecker) Name() string { return c.name }

// Check is the probe-and-catch implementation. Outcomes:
//
//	enabled=false                              → skipped, reason="MULTI_TENANT_ENABLED=false"
//	err == nil                                 → up,      breaker_state="closed"
//	errors.Is(err, tmcore.ErrCircuitBreakerOpen) → down,   breaker_state="open"
//	ctx deadline / canceled                     → down,   breaker_state="closed", error="timeout"/"canceled"
//	other                                       → down,   breaker_state="closed", sanitized error
func (c *TenantManagerClientChecker) Check(ctx context.Context) DependencyCheck {
	tlsOn := tlsOrFalse(detectHTTPUpstreamTLS(c.url))

	if !c.enabled {
		return DependencyCheck{
			Status: StatusSkipped,
			Reason: "MULTI_TENANT_ENABLED=false",
			TLS:    TLSPtr(tlsOn),
		}
	}

	if c.client == nil {
		return DependencyCheck{
			Status: StatusDown,
			TLS:    TLSPtr(tlsOn),
			Error:  "tenant manager client not initialized",
		}
	}

	start := time.Now()
	_, err := c.client.GetActiveTenantsByService(ctx, c.service)
	elapsed := time.Since(start)

	if err == nil {
		return DependencyCheck{
			Status:       StatusUp,
			LatencyMs:    elapsed.Milliseconds(),
			TLS:          TLSPtr(tlsOn),
			BreakerState: BreakerClosed.String(),
		}
	}

	if errors.Is(err, tmcore.ErrCircuitBreakerOpen) {
		return DependencyCheck{
			Status:       StatusDown,
			LatencyMs:    elapsed.Milliseconds(),
			TLS:          TLSPtr(tlsOn),
			Error:        "circuit breaker open",
			BreakerState: BreakerOpen.String(),
		}
	}

	return DependencyCheck{
		Status:       StatusDown,
		LatencyMs:    elapsed.Milliseconds(),
		TLS:          TLSPtr(tlsOn),
		Error:        classifyErr(ctx, err),
		BreakerState: BreakerClosed.String(),
	}
}
