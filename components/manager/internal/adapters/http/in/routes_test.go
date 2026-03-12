package in

import (
	"testing"

	middlewareAuth "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNewRoutes_RegistersCriticalStaticRoutesBeforeIDRoutes(t *testing.T) {
	logger := libLog.NewNop()
	telemetry, err := libOtel.NewTelemetry(libOtel.TelemetryConfig{
		LibraryName:     "test",
		ServiceName:     "test",
		ServiceVersion:  "1.0.0",
		DeploymentEnv:   "test",
		EnableTelemetry: false,
		Logger:          logger,
	})
	require.NoError(t, err)

	auth := middlewareAuth.NewAuthClient("", false, nil)
	app := NewRoutes(
		logger,
		telemetry,
		auth,
		nil,
		&ConnectionHandler{},
		&ProductHandler{},
		&MigrationHandler{},
		&FetcherHandler{},
	)

	var getRoutes []string
	var postRoutes []string
	for _, route := range app.GetRoutes(true) {
		if route.Method == "GET" {
			getRoutes = append(getRoutes, route.Path)
		}
		if route.Method == "POST" {
			postRoutes = append(postRoutes, route.Path)
		}
	}

	unassignedIdx := indexOfRoute(getRoutes, "/v1/management/connections/unassigned")
	connectionIDIdx := indexOfRoute(getRoutes, "/v1/management/connections/:id")
	validateIdx := indexOfRoute(postRoutes, "/v1/management/connections/validate-schema")

	require.NotEqual(t, -1, validateIdx)
	require.NotEqual(t, -1, unassignedIdx)
	require.NotEqual(t, -1, connectionIDIdx)
	assert.Less(t, unassignedIdx, connectionIDIdx)

	assert.NotEqual(t, -1, indexOfRoute(getRoutes, "/health"))
	assert.NotEqual(t, -1, indexOfRoute(getRoutes, "/version"))
	assert.NotEqual(t, -1, indexOfRoute(getRoutes, "/v1/management/products"))
	assert.NotEqual(t, -1, indexOfRoute(getRoutes, "/v1/fetcher/:id"))
}

func indexOfRoute(routes []string, target string) int {
	for idx, route := range routes {
		if route == target {
			return idx
		}
	}

	return -1
}
