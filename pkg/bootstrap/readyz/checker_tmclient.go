package readyz

import (
	"context"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
)

// TMClient is the narrow Tenant-Manager-client surface used by the
// per-tenant handler's tenant-existence validation. *tmclient.Client
// satisfies it; tests inject a fake. The readiness checker holds a value of
// this type only to nil-check that the client is configured — it no longer
// calls GetActiveTenantsByService on the probe path.
type TMClient interface {
	GetActiveTenantsByService(ctx context.Context, service string) ([]*tmclient.TenantSummary, error)
}

// TenantManagerClientChecker reports whether the Tenant Manager client is
// configured. It deliberately does NOT issue a live call to the Tenant
// Manager on every probe: the underlying client already has its own circuit
// breaker and caches active tenants on the request path, so gating pod
// readiness on a live TM call would let a transient TM blip flap /readyz and
// evict pods that are otherwise serving fine. /readyz therefore only answers
// "is the client wired?" — enabled=false → skipped, nil client → down,
// configured client → up. Liveness of TM itself is the breaker's concern.
type TenantManagerClientChecker struct {
	name    string
	client  TMClient
	service string // retained for the stable constructor signature; not read by the nil-check probe
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

func (c *TenantManagerClientChecker) Check(_ context.Context) DependencyCheck {
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

	return DependencyCheck{
		Status: StatusUp,
		TLS:    TLSPtr(tlsOn),
	}
}
