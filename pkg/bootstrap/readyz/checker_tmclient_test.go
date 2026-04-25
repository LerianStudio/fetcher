package readyz

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	tmclient "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestTenantManagerClientChecker_Up(t *testing.T) {
	tm := &fakeTMClient{tenants: []*tmclient.TenantSummary{{ID: "t1", Name: "acme", Status: "active"}}}
	c := NewTenantManagerClientChecker(tm, "fetcher", "https://tenants.prod.example.com", true)

	assert.Equal(t, "tenant_manager", c.Name())

	res := c.Check(context.Background())
	assert.Equal(t, StatusUp, res.Status)
	assert.Equal(t, "closed", res.BreakerState)
	assert.Empty(t, res.Error)
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

func TestTenantManagerClientChecker_BreakerOpen(t *testing.T) {
	// Wrap the sentinel to mirror real propagation — the client wraps with fmt.Errorf.
	tm := &fakeTMClient{err: fmt.Errorf("get: %w", tmcore.ErrCircuitBreakerOpen)}
	c := NewTenantManagerClientChecker(tm, "fetcher", "https://tm", true)

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "open", res.BreakerState)
	assert.Equal(t, "circuit breaker open", res.Error)
}

func TestTenantManagerClientChecker_GenericError(t *testing.T) {
	tm := &fakeTMClient{err: errors.New("unexpected http 500 from tenant manager")}
	c := NewTenantManagerClientChecker(tm, "fetcher", "http://tm.local:8080", true)

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "closed", res.BreakerState, "non-breaker errors stay under closed")
	assert.Contains(t, res.Error, "http 500")
	if assert.NotNil(t, res.TLS) {
		assert.False(t, *res.TLS)
	}
}

func TestTenantManagerClientChecker_Timeout(t *testing.T) {
	tm := &fakeTMClient{delay: 50 * time.Millisecond, err: context.DeadlineExceeded}
	c := NewTenantManagerClientChecker(tm, "fetcher", "https://tm", true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	res := c.Check(ctx)
	assert.Equal(t, StatusDown, res.Status)
	assert.Equal(t, "timeout", res.Error)
	assert.Equal(t, "closed", res.BreakerState)
}

func TestTenantManagerClientChecker_NilClient_Down(t *testing.T) {
	c := NewTenantManagerClientChecker(nil, "fetcher", "https://tm", true)

	res := c.Check(context.Background())
	assert.Equal(t, StatusDown, res.Status)
	assert.Contains(t, res.Error, "not initialized")
}

func TestTenantManagerClientChecker_SanitizesError(t *testing.T) {
	tm := &fakeTMClient{err: errors.New("dial https://admin:hunter2@tm.prod failed")}
	c := NewTenantManagerClientChecker(tm, "fetcher", "https://tm", true)

	res := c.Check(context.Background())
	require.Equal(t, StatusDown, res.Status)
	assert.NotContains(t, res.Error, "hunter2")
}
