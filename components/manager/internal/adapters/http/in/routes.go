package in

import (
	"github.com/LerianStudio/fetcher/v2/pkg/bootstrap/readyz"
	"github.com/LerianStudio/fetcher/v2/pkg/net/http"
	middlewareAuth "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	commonsHttp "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	"github.com/LerianStudio/lib-observability/log"
	obsMiddleware "github.com/LerianStudio/lib-observability/middleware"
	opentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

const (
	applicationName     = "fetcher"
	connectionsResource = "connections"
	fetcherResource     = "fetcher"
)

// NewRoutes wires the fiber router. ttMiddleware is the multi-tenant DB
// resolver; pass nil for single-tenant mode. readyzHandler /
// readyzTenantHandler / metricsHandler are mounted before auth so probes
// from Kubernetes and load-balancers stay unauthenticated.
func NewRoutes(
	lg log.Logger,
	tl *opentelemetry.Telemetry,
	auth *middlewareAuth.AuthClient,
	connectionHandler *ConnectionHandler,
	migrationHandler *MigrationHandler,
	fetcherHandler *FetcherHandler,
	ttMiddleware fiber.Handler,
	readyzHandler fiber.Handler,
	readyzTenantHandler fiber.Handler,
	metricsHandler fiber.Handler,
) *fiber.App {
	f := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			return commonsHttp.FiberErrorHandler(ctx, err)
		},
	})
	tlMid := obsMiddleware.NewTelemetryMiddleware(tl)

	f.Use(http.WithRecover(http.WithRecoverLogger(lg)))
	f.Use(tlMid.WithTelemetry(tl))
	f.Use(cors.New())
	f.Use(obsMiddleware.WithHTTPLogging(obsMiddleware.WithCustomLogger(lg)))

	// Doc Swagger
	f.Get("/swagger/*", WithSwaggerEnvConfig(), fiberSwagger.WrapHandler)

	// /health is gated on the startup self-probe — returns 503 until
	// RunSelfProbe flips the flag, so the kubelet restarts the pod when a
	// dep was unreachable at boot.
	f.Get("/health", readyz.HealthHandler())

	if readyzHandler != nil {
		f.Get("/readyz", readyzHandler)
	}

	if readyzTenantHandler != nil {
		f.Get("/readyz/tenant/:id", readyzTenantHandler)
	}

	if metricsHandler != nil {
		f.Get("/metrics", metricsHandler)
	}

	// Version
	f.Get("/version", commonsHttp.Version)

	// Connections
	f.Post("/v1/management/connections", auth.Authorize(applicationName, connectionsResource, "post"), WhenEnabled(ttMiddleware), connectionHandler.CreateConnection)
	f.Get("/v1/management/connections", auth.Authorize(applicationName, connectionsResource, "get"), WhenEnabled(ttMiddleware), connectionHandler.ListConnections)
	// Schema Validation - must be before :id routes to avoid conflict
	f.Post("/v1/management/connections/validate-schema", auth.Authorize(applicationName, connectionsResource, "post"), WhenEnabled(ttMiddleware), connectionHandler.ValidateSchema)
	// Migration - must be before :id routes to avoid conflict
	f.Get("/v1/management/connections/unassigned", auth.Authorize(applicationName, connectionsResource, "get"), WhenEnabled(ttMiddleware), migrationHandler.ListUnassignedConnections)
	f.Post("/v1/management/connections/:id/assign", auth.Authorize(applicationName, connectionsResource, "post"), WhenEnabled(ttMiddleware), migrationHandler.AssignConnectionToProduct)
	f.Get("/v1/management/connections/:id", auth.Authorize(applicationName, connectionsResource, "get"), WhenEnabled(ttMiddleware), connectionHandler.GetConnection)
	f.Post("/v1/management/connections/:id/test", auth.Authorize(applicationName, connectionsResource, "post"), WhenEnabled(ttMiddleware), connectionHandler.TestConnection)
	f.Get("/v1/management/connections/:id/schema", auth.Authorize(applicationName, connectionsResource, "get"), WhenEnabled(ttMiddleware), connectionHandler.GetConnectionSchema)
	f.Patch("/v1/management/connections/:id", auth.Authorize(applicationName, connectionsResource, "patch"), WhenEnabled(ttMiddleware), connectionHandler.UpdateConnection)
	f.Delete("/v1/management/connections/:id", auth.Authorize(applicationName, connectionsResource, "delete"), WhenEnabled(ttMiddleware), connectionHandler.DeleteConnection)

	// Fetcher
	f.Post("/v1/fetcher", auth.Authorize(applicationName, fetcherResource, "post"), WhenEnabled(ttMiddleware), fetcherHandler.CreateJob)
	f.Get("/v1/fetcher/:id", auth.Authorize(applicationName, fetcherResource, "get"), WhenEnabled(ttMiddleware), fetcherHandler.GetJob)

	f.Use(tlMid.EndTracingSpans)

	return f
}

// WhenEnabled is a helper that conditionally applies a middleware if it's not nil.
func WhenEnabled(middleware fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware == nil {
			return c.Next()
		}

		return middleware(c)
	}
}
