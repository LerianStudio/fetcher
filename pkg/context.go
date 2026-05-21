package pkg

import (
	"context"

	observability "github.com/LerianStudio/lib-observability"
	"github.com/LerianStudio/lib-observability/log"
	"go.opentelemetry.io/otel/trace"
)

type customContextKey string

var CustomContextKey = customContextKey("custom_context")

type CustomContextKeyValue struct {
	Tracer trace.Tracer
	Logger log.Logger
}

// NewLoggerFromContext extracts the logger from the legacy fetcher context key.
// It is kept as a compatibility adapter while callers migrate to
// lib-observability.NewTrackingFromContext/NewLoggerFromContext directly.
func NewLoggerFromContext(ctx context.Context) log.Logger {
	if customContext, ok := ctx.Value(CustomContextKey).(*CustomContextKeyValue); ok &&
		customContext.Logger != nil {
		return customContext.Logger
	}

	if logger := observability.NewLoggerFromContext(ctx); logger != nil {
		return logger
	}

	return log.NewNop()
}

// NewTracerFromContext returns a tracer from the legacy fetcher context key.
// The trace.Tracer type remains in this compatibility API because existing
// tests and adapters accept that interface; fallback resolution delegates to
// lib-observability instead of constructing a raw OTel tracer here.
func NewTracerFromContext(ctx context.Context) trace.Tracer {
	if customContext, ok := ctx.Value(CustomContextKey).(*CustomContextKeyValue); ok &&
		customContext.Tracer != nil {
		return customContext.Tracer
	}

	logger, tracer, requestID, metricsFactory := observability.NewTrackingFromContext(ctx)
	_ = logger
	_ = requestID
	_ = metricsFactory

	return tracer
}

// ContextWithLogger returns a context within a Logger in "logger" value.
func ContextWithLogger(ctx context.Context, logger log.Logger) context.Context {
	values, _ := ctx.Value(CustomContextKey).(*CustomContextKeyValue)
	if values == nil {
		values = &CustomContextKeyValue{}
	}

	values.Logger = logger

	return observability.ContextWithLogger(context.WithValue(ctx, CustomContextKey, values), logger)
}

// ContextWithTracer returns a context within a trace.Tracer in "tracer" value.
func ContextWithTracer(ctx context.Context, tracer trace.Tracer) context.Context {
	values, _ := ctx.Value(CustomContextKey).(*CustomContextKeyValue)
	if values == nil {
		values = &CustomContextKeyValue{}
	}

	values.Tracer = tracer

	return observability.ContextWithTracer(context.WithValue(ctx, CustomContextKey, values), tracer)
}
