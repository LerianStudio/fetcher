package readyz

import (
	"context"
	"errors"
	"time"

	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
)

// TMClient is the narrow Tenant-Manager-client surface used both by the
// readiness checker and the per-tenant handler's tenant-existence
// validation. *tmclient.Client satisfies it; tests inject a fake.
//
// GetActiveTenantsByService is the same call used by the worker's
// initial-tenant-sync on startup; reusing it here exercises cache warming
// in both codepaths.
type TMClient interface {
	GetActiveTenantsByService(ctx context.Context, service string) ([]*tmclient.TenantSummary, error)
}

// TenantManagerClientChecker probes the Tenant Manager HTTP service. The
// underlying client's circuit breaker has no exported State(), so the
// checker uses a probe-and-catch model: it issues the call and, if the
// error chain contains tmcore.ErrCircuitBreakerOpen, reports breaker=open.
// When enabled=false the checker reports "skipped" rather than being
// omitted, so operators can distinguish "unused" from "not wired".
type TenantManagerClientChecker struct {
	name    string
	client  TMClient
	service string
	url     string
	enabled bool
}

// NewTenantManagerClientChecker reports under dep name "tenant_manager".
// client may be nil only when enabled=false; url is used only for TLS
// detection.
func NewTenantManagerClientChecker(client TMClient, service, url string, enabled bool) *TenantManagerClientChecker {
	return &TenantManagerClientChecker{
		name:    "tenant_manager",
		client:  client,
		service: service,
		url:     url,
		enabled: enabled,
	}
}

func (c *TenantManagerClientChecker) Name() string { return c.name }

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
