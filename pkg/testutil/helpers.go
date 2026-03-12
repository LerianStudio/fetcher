package testutil

import (
	"github.com/google/uuid"

	libLog "github.com/LerianStudio/lib-commons/v3/commons/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// testOrgID is a fixed UUID used for test organization IDs across the project.
var testOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// testJobID is a fixed UUID used for test job IDs across the project.
var testJobID = uuid.MustParse("00000000-0000-0000-0000-000000000002")

// TestOrgID returns a fixed test organization UUID.
// Using a fixed UUID ensures deterministic tests and makes it easy
// to identify test data in logs and database queries.
func TestOrgID() uuid.UUID {
	return testOrgID
}

// TestJobID returns a fixed test job UUID.
// Using a fixed UUID ensures deterministic tests and makes it easy
// to identify test data in logs and database queries.
func TestJobID() uuid.UUID {
	return testJobID
}

// TestLogger returns a GoLogger at DebugLevel for use in unit tests.
func TestLogger() *libLog.GoLogger {
	return &libLog.GoLogger{Level: libLog.DebugLevel}
}

// TestTracer returns a no-op tracer for use in unit tests.
// The name parameter identifies the tracer in spans (e.g., "test").
func TestTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
