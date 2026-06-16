package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// extractInt64Sum finds an Int64 Sum (counter) metric by name in the collected
// resource metrics.
func extractInt64Sum(t *testing.T, rm metricdata.ResourceMetrics, name string) metricdata.Sum[int64] {
	t.Helper()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				sum, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Fatalf("metric %q data is not Sum[int64], got %T", name, m.Data)
				}

				return sum
			}
		}
	}

	t.Fatalf("metric %q not found in resource metrics", name)

	return metricdata.Sum[int64]{}
}

// extractInt64Gauge finds an Int64 Gauge metric by name in the collected
// resource metrics.
func extractInt64Gauge(t *testing.T, rm metricdata.ResourceMetrics, name string) metricdata.Gauge[int64] {
	t.Helper()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				gauge, ok := m.Data.(metricdata.Gauge[int64])
				if !ok {
					t.Fatalf("metric %q data is not Gauge[int64], got %T", name, m.Data)
				}

				return gauge
			}
		}
	}

	t.Fatalf("metric %q not found in resource metrics", name)

	return metricdata.Gauge[int64]{}
}

// extractFloat64Histogram finds a Float64 Histogram metric by name in the
// collected resource metrics.
func extractFloat64Histogram(t *testing.T, rm metricdata.ResourceMetrics, name string) metricdata.Histogram[float64] {
	t.Helper()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				h, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Fatalf("metric %q data is not Histogram[float64], got %T", name, m.Data)
				}

				return h
			}
		}
	}

	t.Fatalf("metric %q not found in resource metrics", name)

	return metricdata.Histogram[float64]{}
}

// reconcileOnce on the happy path emits worker_tenant_reconcile_total{result=ok},
// records the reconcile duration histogram, and sets
// worker_tenant_consumer_active to the KnownTenants count — all labelled with
// service only (NO tenant_id).
func TestTenantConsumerReconciler_Metrics_OkPath(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	mat := newFakeMaterializer()
	lister := &fakeTenantLister{responses: []listerResponse{{tenants: tenants("tenant-a", "tenant-b")}}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger(), WithMeterProvider(mp))

	require.NoError(t, r.reconcileOnce(testContext()))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	// Counter: exactly one ok data point, value 1, service-only attribute.
	total := extractInt64Sum(t, rm, "worker_tenant_reconcile_total")
	require.Len(t, total.DataPoints, 1, "exactly one reconcile_total data point on a single ok pass")
	dp := total.DataPoints[0]
	assert.Equal(t, int64(1), dp.Value)

	resultAttr, hasResult := dp.Attributes.Value("result")
	require.True(t, hasResult, "reconcile_total must carry a result attribute")
	assert.Equal(t, "ok", resultAttr.AsString())

	svcAttr, hasSvc := dp.Attributes.Value("service")
	require.True(t, hasSvc, "reconcile_total must carry a service attribute")
	assert.Equal(t, "fetcher", svcAttr.AsString())

	_, hasTenant := dp.Attributes.Value("tenant_id")
	assert.False(t, hasTenant, "reconcile_total must NOT carry tenant_id (low cardinality)")

	// Histogram: one data point recorded, service-only attribute, no tenant_id.
	dur := extractFloat64Histogram(t, rm, "worker_tenant_reconcile_duration_seconds")
	require.Len(t, dur.DataPoints, 1, "exactly one duration data point on a single pass")
	assert.Equal(t, uint64(1), dur.DataPoints[0].Count)
	_, durHasTenant := dur.DataPoints[0].Attributes.Value("tenant_id")
	assert.False(t, durHasTenant, "duration must NOT carry tenant_id")
	durSvc, durHasSvc := dur.DataPoints[0].Attributes.Value("service")
	require.True(t, durHasSvc)
	assert.Equal(t, "fetcher", durSvc.AsString())

	// Gauge: reflects KnownTenants count (2 after materializing a + b).
	gauge := extractInt64Gauge(t, rm, "worker_tenant_consumer_active")
	require.Len(t, gauge.DataPoints, 1)
	assert.Equal(t, int64(2), gauge.DataPoints[0].Value, "active gauge must reflect KnownTenants count")
	_, gaugeHasTenant := gauge.DataPoints[0].Attributes.Value("tenant_id")
	assert.False(t, gaugeHasTenant, "active gauge must NOT carry tenant_id")
}

// reconcileOnce on a Tenant Manager list error emits
// worker_tenant_reconcile_total{result=tm_unreachable} and still records the
// duration histogram, with service-only labels.
func TestTenantConsumerReconciler_Metrics_TMUnreachablePath(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	mat := newFakeMaterializer("tenant-a")
	listErr := errors.New("tm 503 unreachable")
	lister := &fakeTenantLister{responses: []listerResponse{{err: listErr}}}

	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger(), WithMeterProvider(mp))

	require.ErrorIs(t, r.reconcileOnce(testContext()), listErr)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	total := extractInt64Sum(t, rm, "worker_tenant_reconcile_total")
	require.Len(t, total.DataPoints, 1, "exactly one tm_unreachable data point")
	dp := total.DataPoints[0]
	assert.Equal(t, int64(1), dp.Value)

	resultAttr, hasResult := dp.Attributes.Value("result")
	require.True(t, hasResult)
	assert.Equal(t, "tm_unreachable", resultAttr.AsString())

	_, hasTenant := dp.Attributes.Value("tenant_id")
	assert.False(t, hasTenant, "tm_unreachable must NOT carry tenant_id")

	// Duration is recorded even on the error path.
	dur := extractFloat64Histogram(t, rm, "worker_tenant_reconcile_duration_seconds")
	require.Len(t, dur.DataPoints, 1)
	assert.Equal(t, uint64(1), dur.DataPoints[0].Count)
}

// A nil MeterProvider must be safe: the reconciler falls back to a no-op meter,
// reconcileOnce works, and no metric emission panics.
func TestTenantConsumerReconciler_Metrics_NilProviderIsNoOpSafe(t *testing.T) {
	t.Parallel()

	mat := newFakeMaterializer()
	lister := &fakeTenantLister{responses: []listerResponse{{tenants: tenants("tenant-a")}}}

	// No WithMeterProvider option at all -> provider stays nil.
	r := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger())

	assert.NotPanics(t, func() {
		require.NoError(t, r.reconcileOnce(testContext()))
	}, "nil MeterProvider must fall back to a no-op meter; reconcileOnce must not panic")

	// Explicit nil provider via the option is equally safe.
	r2 := NewTenantConsumerReconciler(mat, lister, "fetcher", testLogger(), WithMeterProvider(nil))
	assert.NotPanics(t, func() {
		require.NoError(t, r2.reconcileOnce(testContext()))
	}, "explicit nil provider via option must be no-op safe")
}
