package command

import (
	"context"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"go.opentelemetry.io/otel"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: "test-request-id",
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}

	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}
