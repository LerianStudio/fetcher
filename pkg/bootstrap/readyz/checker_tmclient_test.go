package readyz

import (
	"context"
	"testing"
	"time"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	"github.com/stretchr/testify/assert"
)

// fakeTMClient is the shared Tenant-Manager-client double for the readyz
// package. It is also used by tenant_handler_test.go and goleak_test.go,
// which still exercise the live tenant-existence lookup path — so it keeps
// the full tenants/err/delay shape even though the readiness checker no
// longer issues a live call. The calls counter lets the readiness tests
// assert that the nil-check probe never touches the Tenant Manager.
type fakeTMClient struct {
	tenants []*tmclient.TenantSummary
	err     error
	delay   time.Duration
	calls   int
}

func (f *fakeTMClient) GetActiveTenantsByService(ctx context.Context, _ string) ([]*tmclient.TenantSummary, error) {
	f.calls++

	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return f.tenants, f.err
}

func TestTenantManagerClientChecker_Up_DoesNotProbe(t *testing.T) {
	// A configured client reports Up purely on the nil-check — the checker
	// must NOT call the Tenant Manager (its breaker + request-path cache own
	// liveness; /readyz only reports "client configured").
	tm := &fakeTMClient{tenants: []*tmclient.TenantSummary{{ID: "t1", Name: "acme", Status: "active"}}}
	c := NewTenantManagerClientChecker(tm, "fetcher", "https://tenants.prod.example.com", true)

	assert.Equal(t, "tenant_manager", c.Name())

	res := c.Check(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	assert.Empty(t, res.Error)
	assert.Empty(t, res.BreakerState, "nil-check probe never reports breaker state")
	assert.Zero(t, res.LatencyMs, "nil-check probe issues no live call, so no latency")
	assert.Zero(t, tm.calls, "readiness probe must not call the Tenant Manager")
	if assert.NotNil(t, res.TLS) {
		assert.True(t, *res.TLS)
	}
}

func TestTenantManagerClientChecker_Disabled_Skipped(t *testing.T) {
	c := NewTenantManagerClientChecker(nil, "fetcher", "", false)

	res := c.Check(context.Background())
	assert.Equal(t, StatusSkipped, res.Status)
	assert.Equal(t, "MULTI_TENANT_ENABLED=false", res.Reason)
	assert.Empty(t, res.BreakerState, "skipped deps never carry breaker state")
}

func TestTenantManagerClientChecker_NilClient_Down(t *testing.T) {
	c := NewTenantManagerClientChecker(nil, "fetcher", "https://tm", true)

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "not initialized")
	assert.Empty(t, res.BreakerState, "nil-check probe never reports breaker state")
}

func TestTenantManagerClientChecker_NoTLS_HTTPUpstream(t *testing.T) {
	// Even when the client never errors, an http:// upstream must surface
	// TLS=false. The probe still issues no live call.
	tm := &fakeTMClient{tenants: []*tmclient.TenantSummary{{ID: "t1"}}}
	c := NewTenantManagerClientChecker(tm, "fetcher", "http://tm.local:8080", true)

	res := c.Check(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	assert.Zero(t, tm.calls, "readiness probe must not call the Tenant Manager")
	if assert.NotNil(t, res.TLS) {
		assert.False(t, *res.TLS)
	}
}
