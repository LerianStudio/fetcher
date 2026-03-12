// Package metrics provides multi-tenant metric instrumentation using OpenTelemetry.
//
// When multi-tenant mode is disabled (MULTI_TENANT_ENABLED=false), all metric
// operations use no-op implementations for zero overhead in single-tenant deployments.
package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

const (
	// meterName is the OpenTelemetry meter name for tenant metrics.
	meterName = "fetcher.tenant"

	// Metric names as specified by multi-tenant.md standards.
	metricTenantConnectionsTotal       = "tenant_connections_total"
	metricTenantConnectionErrorsTotal  = "tenant_connection_errors_total"
	metricTenantConsumersActive        = "tenant_consumers_active"
	metricTenantMessagesProcessedTotal = "tenant_messages_processed_total"

	// tenantIDAttrKey is the attribute key for the tenant identifier.
	tenantIDAttrKey = "tenant_id"
)

// TenantMetrics provides instrumentation for multi-tenant operations.
// When multi-tenant is disabled, all operations are no-ops with zero overhead.
type TenantMetrics struct {
	connectionsTotal       otelmetric.Int64Counter
	connectionErrorsTotal  otelmetric.Int64Counter
	consumersActive        otelmetric.Int64Gauge
	messagesProcessedTotal otelmetric.Int64Counter
}

// NewTenantMetrics creates a TenantMetrics instance using the provided MeterProvider.
// When multiTenantEnabled is false, the provider is ignored and a no-op implementation
// is returned with zero overhead in single-tenant mode.
func NewTenantMetrics(multiTenantEnabled bool, provider otelmetric.MeterProvider) (*TenantMetrics, error) {
	if !multiTenantEnabled {
		return newNoOpTenantMetrics()
	}

	return newOTelTenantMetrics(provider)
}

// newNoOpTenantMetrics creates a TenantMetrics with no-op instruments.
// All operations are zero-cost when multi-tenant is disabled.
func newNoOpTenantMetrics() (*TenantMetrics, error) {
	meter := noop.Meter{}

	connectionsTotal, err := meter.Int64Counter(metricTenantConnectionsTotal)
	if err != nil {
		return nil, err
	}

	connectionErrorsTotal, err := meter.Int64Counter(metricTenantConnectionErrorsTotal)
	if err != nil {
		return nil, err
	}

	consumersActive, err := meter.Int64Gauge(metricTenantConsumersActive)
	if err != nil {
		return nil, err
	}

	messagesProcessedTotal, err := meter.Int64Counter(metricTenantMessagesProcessedTotal)
	if err != nil {
		return nil, err
	}

	return &TenantMetrics{
		connectionsTotal:       connectionsTotal,
		connectionErrorsTotal:  connectionErrorsTotal,
		consumersActive:        consumersActive,
		messagesProcessedTotal: messagesProcessedTotal,
	}, nil
}

// newOTelTenantMetrics creates a TenantMetrics with real OpenTelemetry instruments.
func newOTelTenantMetrics(provider otelmetric.MeterProvider) (*TenantMetrics, error) {
	meter := provider.Meter(meterName)

	connectionsTotal, err := meter.Int64Counter(
		metricTenantConnectionsTotal,
		otelmetric.WithDescription("Total tenant connections created"),
	)
	if err != nil {
		return nil, err
	}

	connectionErrorsTotal, err := meter.Int64Counter(
		metricTenantConnectionErrorsTotal,
		otelmetric.WithDescription("Connection failures per tenant"),
	)
	if err != nil {
		return nil, err
	}

	consumersActive, err := meter.Int64Gauge(
		metricTenantConsumersActive,
		otelmetric.WithDescription("Active message consumers"),
	)
	if err != nil {
		return nil, err
	}

	messagesProcessedTotal, err := meter.Int64Counter(
		metricTenantMessagesProcessedTotal,
		otelmetric.WithDescription("Messages processed per tenant"),
	)
	if err != nil {
		return nil, err
	}

	return &TenantMetrics{
		connectionsTotal:       connectionsTotal,
		connectionErrorsTotal:  connectionErrorsTotal,
		consumersActive:        consumersActive,
		messagesProcessedTotal: messagesProcessedTotal,
	}, nil
}

// IncrementTenantConnectionsTotal increments the total tenant connections counter.
func (tm *TenantMetrics) IncrementTenantConnectionsTotal(ctx context.Context, tenantID string) {
	tm.connectionsTotal.Add(ctx, 1,
		otelmetric.WithAttributes(attribute.String(tenantIDAttrKey, tenantID)),
	)
}

// IncrementTenantConnectionErrorsTotal increments the tenant connection errors counter.
func (tm *TenantMetrics) IncrementTenantConnectionErrorsTotal(ctx context.Context, tenantID string) {
	tm.connectionErrorsTotal.Add(ctx, 1,
		otelmetric.WithAttributes(attribute.String(tenantIDAttrKey, tenantID)),
	)
}

// SetTenantConsumersActive sets the number of active consumers for a tenant.
func (tm *TenantMetrics) SetTenantConsumersActive(ctx context.Context, tenantID string, count int64) {
	tm.consumersActive.Record(ctx, count,
		otelmetric.WithAttributes(attribute.String(tenantIDAttrKey, tenantID)),
	)
}

// IncrementTenantMessagesProcessedTotal increments the total messages processed counter.
func (tm *TenantMetrics) IncrementTenantMessagesProcessedTotal(ctx context.Context, tenantID string) {
	tm.messagesProcessedTotal.Add(ctx, 1,
		otelmetric.WithAttributes(attribute.String(tenantIDAttrKey, tenantID)),
	)
}
