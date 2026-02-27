package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// extractSum is a test helper that finds a Sum metric by name in the resource metrics.
func extractSum(t *testing.T, rm metricdata.ResourceMetrics, name string) *metricdata.Sum[int64] {
	t.Helper()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				sum, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Fatalf("metric %q data is not Sum[int64], got %T", name, m.Data)
				}

				return &sum
			}
		}
	}

	t.Fatalf("metric %q not found in resource metrics", name)

	return nil
}

// extractGauge is a test helper that finds a Gauge metric by name in the resource metrics.
func extractGauge(t *testing.T, rm metricdata.ResourceMetrics, name string) *metricdata.Gauge[int64] {
	t.Helper()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				gauge, ok := m.Data.(metricdata.Gauge[int64])
				if !ok {
					t.Fatalf("metric %q data is not Gauge[int64], got %T", name, m.Data)
				}

				return &gauge
			}
		}
	}

	t.Fatalf("metric %q not found in resource metrics", name)

	return nil
}

func TestTenantMetrics_NewTenantMetrics(t *testing.T) {
	tests := []struct {
		name             string
		multiTenantState bool
		description      string
	}{
		{
			name:             "creates metrics when multi-tenant enabled",
			multiTenantState: true,
			description:      "should return non-nil TenantMetrics with real OTel instruments",
		},
		{
			name:             "creates no-op metrics when multi-tenant disabled",
			multiTenantState: false,
			description:      "should return non-nil TenantMetrics with no-op instruments (zero overhead)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := metric.NewMeterProvider()
			tm, err := NewTenantMetrics(tt.multiTenantState, mp)
			require.NoError(t, err, tt.description)
			assert.NotNil(t, tm, tt.description)
		})
	}
}

func TestTenantMetrics_IncrementTenantConnectionsTotal(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	tm, err := NewTenantMetrics(true, mp)
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := "tenant-abc-123"

	tm.IncrementTenantConnectionsTotal(ctx, tenantID)
	tm.IncrementTenantConnectionsTotal(ctx, tenantID)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	sum := extractSum(t, rm, "tenant_connections_total")
	require.Len(t, sum.DataPoints, 1, "expected 1 data point for single tenant")
	assert.Equal(t, int64(2), sum.DataPoints[0].Value, "expected counter value of 2 after two increments")
}

func TestTenantMetrics_IncrementTenantConnectionErrorsTotal(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	tm, err := NewTenantMetrics(true, mp)
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := "tenant-abc-123"

	tm.IncrementTenantConnectionErrorsTotal(ctx, tenantID)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	sum := extractSum(t, rm, "tenant_connection_errors_total")
	require.Len(t, sum.DataPoints, 1)
	assert.Equal(t, int64(1), sum.DataPoints[0].Value, "expected counter value of 1 after one increment")
}

func TestTenantMetrics_SetTenantConsumersActive(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	tm, err := NewTenantMetrics(true, mp)
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := "tenant-abc-123"

	tm.SetTenantConsumersActive(ctx, tenantID, 5)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	gauge := extractGauge(t, rm, "tenant_consumers_active")
	require.Len(t, gauge.DataPoints, 1)
	assert.Equal(t, int64(5), gauge.DataPoints[0].Value, "expected gauge value of 5")
}

func TestTenantMetrics_IncrementTenantMessagesProcessedTotal(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	tm, err := NewTenantMetrics(true, mp)
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := "tenant-abc-123"

	tm.IncrementTenantMessagesProcessedTotal(ctx, tenantID)
	tm.IncrementTenantMessagesProcessedTotal(ctx, tenantID)
	tm.IncrementTenantMessagesProcessedTotal(ctx, tenantID)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	sum := extractSum(t, rm, "tenant_messages_processed_total")
	require.Len(t, sum.DataPoints, 1)
	assert.Equal(t, int64(3), sum.DataPoints[0].Value, "expected counter value of 3 after three increments")
}

func TestTenantMetrics_NoOpWhenDisabled(t *testing.T) {
	// When multi-tenant is disabled, all metric operations should be no-ops
	// with zero overhead. They must not panic or allocate.
	// Provider is ignored when disabled, so nil is acceptable.
	tm, err := NewTenantMetrics(false, nil)
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := "tenant-abc-123"

	// All of these must not panic
	assert.NotPanics(t, func() {
		tm.IncrementTenantConnectionsTotal(ctx, tenantID)
	}, "IncrementTenantConnectionsTotal should be no-op when disabled")

	assert.NotPanics(t, func() {
		tm.IncrementTenantConnectionErrorsTotal(ctx, tenantID)
	}, "IncrementTenantConnectionErrorsTotal should be no-op when disabled")

	assert.NotPanics(t, func() {
		tm.SetTenantConsumersActive(ctx, tenantID, 10)
	}, "SetTenantConsumersActive should be no-op when disabled")

	assert.NotPanics(t, func() {
		tm.IncrementTenantMessagesProcessedTotal(ctx, tenantID)
	}, "IncrementTenantMessagesProcessedTotal should be no-op when disabled")
}

func TestTenantMetrics_MultipleTenants(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	tm, err := NewTenantMetrics(true, mp)
	require.NoError(t, err)

	ctx := context.Background()

	// Increment connections for two different tenants
	tm.IncrementTenantConnectionsTotal(ctx, "tenant-1")
	tm.IncrementTenantConnectionsTotal(ctx, "tenant-1")
	tm.IncrementTenantConnectionsTotal(ctx, "tenant-2")

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	sum := extractSum(t, rm, "tenant_connections_total")
	require.Len(t, sum.DataPoints, 2, "expected 2 data points for two tenants")

	// Verify that both tenants have separate counts
	totalValue := int64(0)
	for _, dp := range sum.DataPoints {
		totalValue += dp.Value
	}

	assert.Equal(t, int64(3), totalValue, "expected total counter value of 3 across both tenants")
}

func TestTenantMetrics_TenantIDAttribute(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	tm, err := NewTenantMetrics(true, mp)
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := "org-xyz-456"

	// Emit all four metrics for the same tenant
	tm.IncrementTenantConnectionsTotal(ctx, tenantID)
	tm.IncrementTenantConnectionErrorsTotal(ctx, tenantID)
	tm.SetTenantConsumersActive(ctx, tenantID, 3)
	tm.IncrementTenantMessagesProcessedTotal(ctx, tenantID)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	// Verify tenant_id attribute on tenant_connections_total
	connectionsSum := extractSum(t, rm, "tenant_connections_total")
	require.Len(t, connectionsSum.DataPoints, 1)

	connAttrs := connectionsSum.DataPoints[0].Attributes
	connTenantAttr, connFound := connAttrs.Value("tenant_id")
	assert.True(t, connFound, "expected tenant_id attribute on tenant_connections_total")
	assert.Equal(t, tenantID, connTenantAttr.AsString(), "tenant_id must match on tenant_connections_total")

	// Verify tenant_id attribute on tenant_connection_errors_total
	errorsSum := extractSum(t, rm, "tenant_connection_errors_total")
	require.Len(t, errorsSum.DataPoints, 1)

	errAttrs := errorsSum.DataPoints[0].Attributes
	errTenantAttr, errFound := errAttrs.Value("tenant_id")
	assert.True(t, errFound, "expected tenant_id attribute on tenant_connection_errors_total")
	assert.Equal(t, tenantID, errTenantAttr.AsString(), "tenant_id must match on tenant_connection_errors_total")

	// Verify tenant_id attribute on tenant_consumers_active
	consumersGauge := extractGauge(t, rm, "tenant_consumers_active")
	require.Len(t, consumersGauge.DataPoints, 1)

	consAttrs := consumersGauge.DataPoints[0].Attributes
	consTenantAttr, consFound := consAttrs.Value("tenant_id")
	assert.True(t, consFound, "expected tenant_id attribute on tenant_consumers_active")
	assert.Equal(t, tenantID, consTenantAttr.AsString(), "tenant_id must match on tenant_consumers_active")

	// Verify tenant_id attribute on tenant_messages_processed_total
	messagesSum := extractSum(t, rm, "tenant_messages_processed_total")
	require.Len(t, messagesSum.DataPoints, 1)

	msgAttrs := messagesSum.DataPoints[0].Attributes
	msgTenantAttr, msgFound := msgAttrs.Value("tenant_id")
	assert.True(t, msgFound, "expected tenant_id attribute on tenant_messages_processed_total")
	assert.Equal(t, tenantID, msgTenantAttr.AsString(), "tenant_id must match on tenant_messages_processed_total")
}
