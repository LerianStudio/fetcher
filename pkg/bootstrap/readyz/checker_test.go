package readyz

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPerDepTimeout_KnownDeps(t *testing.T) {
	tests := []struct {
		dep  string
		want time.Duration
	}{
		{dep: "postgres", want: 2 * time.Second},
		{dep: "mongodb", want: 2 * time.Second},
		{dep: "mysql", want: 2 * time.Second},
		{dep: "oracle", want: 2 * time.Second},
		{dep: "sqlserver", want: 2 * time.Second},
		{dep: "redis", want: 1 * time.Second},
		{dep: "valkey", want: 1 * time.Second},
		{dep: "rabbitmq", want: 2 * time.Second},
		{dep: "s3", want: 2 * time.Second},
		{dep: "seaweedfs", want: 2 * time.Second},
		{dep: "tenant_manager", want: 1 * time.Second},
		{dep: "upstream_fees", want: 1 * time.Second},
		{dep: "rabbitmq_jobs", want: 2 * time.Second},
		{dep: "unknown_xyz", want: 2 * time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.dep, func(t *testing.T) {
			assert.Equal(t, tc.want, PerDepTimeout(tc.dep))
		})
	}
}

func TestStubChecker_ReturnsSkippedWithReason(t *testing.T) {
	c := NewStubChecker("mongodb", "pending Gate 6 implementation")

	assert.Equal(t, "mongodb", c.Name())

	got := c.Check(context.Background())
	assert.Equal(t, StatusSkipped, got.Status)
	assert.Equal(t, "pending Gate 6 implementation", got.Reason)
	assert.Empty(t, got.Error)
	assert.Zero(t, got.LatencyMs)
	assert.Nil(t, got.TLS)
}

func TestStubChecker_IgnoresContextCancellation(t *testing.T) {
	// The stub performs no I/O so it must not care about ctx. The handler
	// still enforces deadlines externally — this is the contract.
	c := NewStubChecker("redis", "disabled")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := c.Check(ctx)
	assert.Equal(t, StatusSkipped, got.Status)
}
