package readyz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNAChecker_ReturnsConfiguredReason(t *testing.T) {
	c := NewNAChecker("mongodb", "multi-tenant carve-out", nil)

	assert.Equal(t, "mongodb", c.Name())

	res := c.Check(context.Background())
	assert.Equal(t, StatusNA, res.Status)
	assert.Equal(t, "multi-tenant carve-out", res.Reason)
	assert.Nil(t, res.TLS)
	assert.Empty(t, res.Error)
	assert.Empty(t, res.BreakerState)
	assert.Zero(t, res.LatencyMs)
}

func TestNAChecker_PreservesTLSPointer(t *testing.T) {
	c := NewNAChecker("rabbitmq", "carved out", TLSPtr(true))

	res := c.Check(context.Background())
	if assert.NotNil(t, res.TLS) {
		assert.True(t, *res.TLS)
	}

	c2 := NewNAChecker("redis", "carved out", TLSPtr(false))

	res2 := c2.Check(context.Background())
	if assert.NotNil(t, res2.TLS) {
		assert.False(t, *res2.TLS)
	}
}

func TestNAChecker_IgnoresCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewNAChecker("dep", "reason", nil)

	res := c.Check(ctx)
	assert.Equal(t, StatusNA, res.Status)
	assert.Equal(t, "reason", res.Reason)
}
