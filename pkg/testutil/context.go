package testutil

import (
	"context"

	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	"go.opentelemetry.io/otel"
)

// TestContext creates a context with logger and tracer for testing.
// It includes a request ID, a debug-level logger, and a no-op tracer,
// which is sufficient for unit tests that call functions requiring
// tracking information from context.
func TestContext() context.Context {
	logger := &libLog.GoLogger{Level: libLog.LevelDebug}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: "test-request-id",
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}

	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}
