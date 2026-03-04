package in

import (
	"github.com/LerianStudio/fetcher/pkg/net/http"
	middlewareAuth "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
	commonsHttp "github.com/LerianStudio/lib-commons/v3/commons/net/http"
	"github.com/LerianStudio/lib-commons/v3/commons/opentelemetry"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

const (
	applicationName     = "fetcher"
	connectionsResource = "connections"
	fetcherResource     = "fetcher"
)

// NewRoutes creates a new fiber router with the specified handlers and middleware.
// The tenantMiddleware parameter accepts a fiber.Handler for multi-tenant DB resolution.
// Pass nil to disable tenant middleware (single-tenant mode).
func NewRoutes(
	lg log.Logger,
	tl *opentelemetry.Telemetry,
	auth *middlewareAuth.AuthClient,
	licenseClient *libLicense.LicenseClient,
	connectionHandler *ConnectionHandler,
	migrationHandler *MigrationHandler,
	fetcherHandler *FetcherHandler,
	tenantMiddleware fiber.Handler,
) *fiber.App {
	f := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			return commonsHttp.HandleFiberError(ctx, err)
		},
	})
	tlMid := commonsHttp.NewTelemetryMiddleware(tl)

	f.Use(http.WithRecover(http.WithRecoverLogger(lg)))
	f.Use(tlMid.WithTelemetry(tl))
	f.Use(cors.New())
	f.Use(commonsHttp.WithHTTPLogging(commonsHttp.WithCustomLogger(lg)))
	// TODO: Enable license middleware when ready
	// f.Use(licenseClient.Middleware())

	// Public endpoints (no tenant context needed)
	// Doc Swagger
	f.Get("/swagger/*", WithSwaggerEnvConfig(), fiberSwagger.WrapHandler)

	// Health
	f.Get("/health", commonsHttp.Ping)

	// Version
	f.Get("/version", commonsHttp.Version)

	// Protected routes: auth.Authorize runs FIRST (validates JWT signature), THEN tenant
	// middleware resolves the tenant DB connection. This ordering prevents forged JWTs from
	// triggering Tenant Manager API calls before authentication is verified.
	//
	// withTenant is a helper that appends tenantMiddleware after the auth handler.
	// When tenantMiddleware is nil (single-tenant mode), only auth runs.
	withTenant := func(authHandler fiber.Handler, routeHandler fiber.Handler) []fiber.Handler {
		if tenantMiddleware != nil {
			return []fiber.Handler{authHandler, tenantMiddleware, routeHandler}
		}

		return []fiber.Handler{authHandler, routeHandler}
	}

	// Connections
	f.Post("/v1/management/connections", withTenant(auth.Authorize(applicationName, connectionsResource, "post"), connectionHandler.CreateConnection)...)
	f.Get("/v1/management/connections", withTenant(auth.Authorize(applicationName, connectionsResource, "get"), connectionHandler.ListConnections)...)
	// Schema Validation - must be before :id routes to avoid conflict
	f.Post("/v1/management/connections/validate-schema", withTenant(auth.Authorize(applicationName, connectionsResource, "post"), connectionHandler.ValidateSchema)...)
	// Migration - must be before :id routes to avoid conflict
	f.Get("/v1/management/connections/unassigned", withTenant(auth.Authorize(applicationName, connectionsResource, "get"), migrationHandler.ListUnassignedConnections)...)
	f.Post("/v1/management/connections/:id/assign", withTenant(auth.Authorize(applicationName, connectionsResource, "post"), migrationHandler.AssignConnectionToProduct)...)
	f.Get("/v1/management/connections/:id", withTenant(auth.Authorize(applicationName, connectionsResource, "get"), connectionHandler.GetConnection)...)
	f.Post("/v1/management/connections/:id/test", withTenant(auth.Authorize(applicationName, connectionsResource, "post"), connectionHandler.TestConnection)...)
	f.Get("/v1/management/connections/:id/schema", withTenant(auth.Authorize(applicationName, connectionsResource, "get"), connectionHandler.GetConnectionSchema)...)
	f.Patch("/v1/management/connections/:id", withTenant(auth.Authorize(applicationName, connectionsResource, "patch"), connectionHandler.UpdateConnection)...)
	f.Delete("/v1/management/connections/:id", withTenant(auth.Authorize(applicationName, connectionsResource, "delete"), connectionHandler.DeleteConnection)...)

	// Fetcher
	f.Post("/v1/fetcher", withTenant(auth.Authorize(applicationName, fetcherResource, "post"), fetcherHandler.CreateJob)...)
	f.Get("/v1/fetcher/:id", withTenant(auth.Authorize(applicationName, fetcherResource, "get"), fetcherHandler.GetJob)...)

	f.Use(tlMid.EndTracingSpans)

	return f
}
