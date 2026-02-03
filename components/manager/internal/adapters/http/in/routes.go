package in

import (
	"github.com/LerianStudio/fetcher/pkg/net/http"
	middlewareAuth "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	commonsHttp "github.com/LerianStudio/lib-commons/v2/commons/net/http"
	"github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
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
func NewRoutes(
	lg log.Logger,
	tl *opentelemetry.Telemetry,
	auth *middlewareAuth.AuthClient,
	licenseClient *libLicense.LicenseClient,
	connectionHandler *ConnectionHandler,
	fetcherHandler *FetcherHandler,
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

	// Doc Swagger
	f.Get("/swagger/*", WithSwaggerEnvConfig(), fiberSwagger.WrapHandler)

	// Health
	f.Get("/health", commonsHttp.Ping)

	// Version
	f.Get("/version", commonsHttp.Version)

	// Connections
	f.Post("/v1/management/connections", auth.Authorize(applicationName, connectionsResource, "post"), connectionHandler.CreateConnection)
	f.Get("/v1/management/connections", auth.Authorize(applicationName, connectionsResource, "get"), connectionHandler.ListConnections)
	// Schema Validation - must be before :id routes to avoid conflict
	f.Post("/v1/management/connections/validate-schema", auth.Authorize(applicationName, connectionsResource, "post"), connectionHandler.ValidateSchema)
	f.Get("/v1/management/connections/:id", auth.Authorize(applicationName, connectionsResource, "get"), connectionHandler.GetConnection)
	f.Post("/v1/management/connections/:id/test", auth.Authorize(applicationName, connectionsResource, "post"), connectionHandler.TestConnection)
	f.Get("/v1/management/connections/:id/schema", auth.Authorize(applicationName, connectionsResource, "get"), connectionHandler.GetConnectionSchema)
	f.Patch("/v1/management/connections/:id", auth.Authorize(applicationName, connectionsResource, "patch"), connectionHandler.UpdateConnection)
	f.Delete("/v1/management/connections/:id", auth.Authorize(applicationName, connectionsResource, "delete"), connectionHandler.DeleteConnection)

	// Fetcher
	f.Post("/v1/fetcher", auth.Authorize(applicationName, fetcherResource, "post"), fetcherHandler.CreateJob)
	f.Get("/v1/fetcher/:id", auth.Authorize(applicationName, fetcherResource, "get"), fetcherHandler.GetJob)

	f.Use(tlMid.EndTracingSpans)

	return f
}
