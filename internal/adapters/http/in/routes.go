package in

import (
	"golang-plugin-boilerplate/pkg/model"
	"golang-plugin-boilerplate/pkg/net/http"

	libLog "github.com/LerianStudio/lib-commons/commons/log"
	libHTTP "github.com/LerianStudio/lib-commons/commons/net/http"
	libOtel "github.com/LerianStudio/lib-commons/commons/opentelemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

func NewRoutes(lg libLog.Logger, tl *libOtel.Telemetry, exampleHandler *ExampleHandler) *fiber.App {
	f := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	tlMid := libHTTP.NewTelemetryMiddleware(tl)

	f.Use(tlMid.WithTelemetry(tl))
	f.Use(cors.New())
	f.Use(libHTTP.WithHTTPLogging(libHTTP.WithCustomLogger(lg)))

	// Example routes
	f.Post("/v1/example", http.WithBody(new(model.CreateExampleInput), exampleHandler.CreateExample))
	f.Get("/v1/example/:id", http.ParseUUIDPathParameters, exampleHandler.GetExampleByID)
	f.Get("/v1/example", exampleHandler.GetAllExample)
	f.Patch("/v1/example/:id", http.ParseUUIDPathParameters, http.WithBody(new(model.UpdateExampleInput), exampleHandler.UpdateExample))
	f.Delete("/v1/example/:id", http.ParseUUIDPathParameters, exampleHandler.DeleteExampleByID)

	// Health
	f.Get("/health", libHTTP.Ping)

	// Version
	f.Get("/version", libHTTP.Version)

	// Doc Swagger
	f.Get("/swagger/*", WithSwaggerEnvConfig(), fiberSwagger.WrapHandler)

	f.Use(tlMid.EndTracingSpans)

	return f
}
