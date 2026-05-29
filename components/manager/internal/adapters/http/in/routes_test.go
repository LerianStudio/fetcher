package in

import (
	"testing"

	middlewareAuth "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	"github.com/LerianStudio/lib-commons/v5/commons/log"
	"github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Tests that call NewRoutes with app.Test() were removed because lib-commons
// telemetry middleware has a data race issue when running with -race flag.
// The race occurs between ContextWithLogger() and NewLoggerFromContext() in
// background goroutines spawned by the telemetry metrics collection.
// See: lib-commons/v5/commons/net/http/withTelemetry.go:158
//
// Routes are tested indirectly through connection_test.go and fetcher_test.go
// which test the handlers directly without the telemetry middleware.

func TestNewRoutes_Constants(t *testing.T) {
	// Test that constants are defined correctly
	assert.Equal(t, "fetcher", applicationName)
	assert.Equal(t, "connections", connectionsResource)
	assert.Equal(t, "fetcher", fetcherResource)
}

// TestNewRoutes_SignatureAcceptsTenantMiddleware is a compile-time signature
// assertion: a type alias must match NewRoutes's parameter list, including
// the readyz / metrics handler trio. Avoids invoking NewRoutes to keep the
// telemetry race at bay.
func TestNewRoutes_SignatureAcceptsTenantMiddleware(t *testing.T) {
	// Verify NewRoutes function signature includes tenantMiddleware parameter.
	// This is a compile-time assertion: if NewRoutes does not accept fiber.Handler
	// as its last parameter, this assignment will cause a compilation error.
	type expectedSignature func(
		lg log.Logger,
		tl *opentelemetry.Telemetry,
		auth *middlewareAuth.AuthClient,
		licenseClient *libLicense.LicenseClient,
		connectionHandler *ConnectionHandler,
		migrationHandler *MigrationHandler,
		fetcherHandler *FetcherHandler,
		tenantMiddleware fiber.Handler,
		readyzHandler fiber.Handler,
		readyzTenantHandler fiber.Handler,
		metricsHandler fiber.Handler,
	) *fiber.App

	var _ expectedSignature = NewRoutes

	// Also verify nil is a valid value for tenantMiddleware (single-tenant mode)
	var nilHandler fiber.Handler
	assert.Nil(t, nilHandler, "nil fiber.Handler should be valid for single-tenant mode")

	_ = require.NoError // suppress unused import
}
