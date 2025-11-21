package in

import (
	middlewareAuth "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	commonsHttp "github.com/LerianStudio/lib-commons/v2/commons/net/http"
	"github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// NewRoutes creates a new fiber router with the specified handlers and middleware.
func NewRoutes(lg log.Logger, tl *opentelemetry.Telemetry, auth *middlewareAuth.AuthClient, licenseClient *libLicense.LicenseClient) *fiber.App {
	f := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			return commonsHttp.HandleFiberError(ctx, err)
		},
	})
	tlMid := commonsHttp.NewTelemetryMiddleware(tl)

	f.Use(tlMid.WithTelemetry(tl))
	f.Use(cors.New())
	f.Use(commonsHttp.WithHTTPLogging(commonsHttp.WithCustomLogger(lg)))
	// f.Use(licenseClient.Middleware())

	// Doc Swagger
	f.Get("/swagger/*", WithSwaggerEnvConfig(), fiberSwagger.WrapHandler)

	// Health
	f.Get("/health", commonsHttp.Ping)

	// Version
	f.Get("/version", commonsHttp.Version)

	f.Use(tlMid.EndTracingSpans)

	return f
}
