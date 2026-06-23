package testutil

import (
	"context"

	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	"go.opentelemetry.io/otel"
)

// TestContext creates a context with logger and tracer for testing.
// It includes a request ID, a debug-level logger, and a no-op tracer,
// which is sufficient for unit tests that call functions requiring
// tracking information from context.
func TestContext() context.Context {
	logger := &libLog.GoLogger{Level: libLog.LevelDebug}
	ctx := observability.ContextWithHeaderID(context.Background(), "test-request-id")
	ctx = observability.ContextWithLogger(ctx, logger)

	return observability.ContextWithTracer(ctx, otel.Tracer("test"))
}
