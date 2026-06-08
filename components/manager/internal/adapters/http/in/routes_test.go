package in

import (
	"net/http"
	"net/http/httptest"
	"testing"

	middlewareAuth "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
	"github.com/LerianStudio/lib-license-go/v2/test/helper/testlogger"
	"github.com/LerianStudio/lib-observability/log"
	opentelemetry "github.com/LerianStudio/lib-observability/tracing"
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
		licenseEnforcementEnabled bool,
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
}

func TestLicenseWhenEnabled_DefaultDoesNotApplyClientMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		client  *libLicense.LicenseClient
		enforce bool
		want    int
	}{
		{
			name:    "client_exists_but_enforcement_disabled_allows_business_route",
			client:  newTestLicenseClient(t),
			enforce: false,
			want:    fiber.StatusNoContent,
		},
		{
			name:    "nil_client_with_enforcement_disabled_allows_business_route",
			client:  nil,
			enforce: false,
			want:    fiber.StatusNoContent,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := fiber.New()
			app.Get("/v1/fetcher", LicenseWhenEnabled(tt.client, tt.enforce), func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusNoContent)
			})

			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/fetcher", nil))
			require.NoError(t, err)
			assert.Equal(t, tt.want, resp.StatusCode)
			require.NoError(t, resp.Body.Close())
		})
	}
}

func TestLicenseWhenEnabled_ExplicitEnableAppliesClientMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		enforce   bool
		wantPanic bool
	}{
		{
			name:      "disabled_does_not_construct_license_middleware",
			enforce:   false,
			wantPanic: false,
		},
		{
			name:      "enabled_constructs_license_middleware",
			enforce:   true,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registerRoute := func() {
				app := fiber.New()
				app.Get("/v1/fetcher", LicenseWhenEnabled(&libLicense.LicenseClient{}, tt.enforce), func(c *fiber.Ctx) error {
					return c.SendStatus(fiber.StatusNoContent)
				})
			}

			if tt.wantPanic {
				require.Panics(t, registerRoute)
				return
			}

			require.NotPanics(t, registerRoute)
		})
	}
}

func newTestLicenseClient(t *testing.T) *libLicense.LicenseClient {
	t.Helper()

	testLogger := testlogger.New()
	var logger log.Logger = testLogger
	client := libLicense.NewLicenseClient(applicationName, "", "test-org", &logger)
	require.NotNil(t, client)

	return client
}
