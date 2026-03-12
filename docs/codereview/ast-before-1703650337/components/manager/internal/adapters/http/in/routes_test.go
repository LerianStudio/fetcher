package in

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: Tests that call NewRoutes with app.Test() were removed because lib-commons
// telemetry middleware has a data race issue when running with -race flag.
// The race occurs between ContextWithLogger() and NewLoggerFromContext() in
// background goroutines spawned by the telemetry metrics collection.
// See: lib-commons/v2/commons/net/http/withTelemetry.go:158
//
// Routes are tested indirectly through connection_test.go and fetcher_test.go
// which test the handlers directly without the telemetry middleware.

func TestNewRoutes_Constants(t *testing.T) {
	// Test that constants are defined correctly
	assert.Equal(t, "fetcher", applicationName)
	assert.Equal(t, "connections", connectionsResource)
	assert.Equal(t, "fetcher", fetcherResource)
}
