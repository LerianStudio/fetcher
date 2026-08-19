package in

import (
	"github.com/LerianStudio/lib-commons/v6/commons/net/http/openapi"
	"github.com/LerianStudio/lib-commons/v6/commons/net/http/problem"
	libLog "github.com/LerianStudio/lib-observability/v2/log"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

const (
	DocTitle       = "Fetcher Manager API"
	DocVersion     = "1.0.0"
	DocDescription = "API documentation for the Fetcher Manager component"

	bearerAuthSecurityScheme = "BearerAuth"
)

// BuildHumaAPI creates the single OpenAPI assembly shared by runtime and
// spec generation. problem.Install must stay before operation registration.
func BuildHumaAPI(app *fiber.App, authEnabled bool) huma.API {
	api := openapi.New(app, app.Group("/"), openapi.Config{
		Title:       DocTitle,
		Version:     DocVersion,
		Description: DocDescription,
	})

	problem.Install()
	openapi.DeclareBearerAuth(api)

	if authEnabled {
		api.OpenAPI().Security = []map[string][]string{{bearerAuthSecurityScheme: {}}}
	}

	doc := api.OpenAPI()
	doc.Info.Contact = &huma.Contact{
		Name:  "Lerian",
		URL:   "https://github.com/LerianStudio/fetcher/discussions",
		Email: "contact@lerian.studio",
	}
	doc.Info.License = &huma.License{
		Name:       "Elastic License 2.0 (Source Available)",
		Identifier: "Elastic-2.0",
	}
	doc.Tags = []*huma.Tag{
		{Name: "Connections", Description: "Database connection lifecycle and schema operations."},
		{Name: "Migration", Description: "Legacy connection assignment and migration operations."},
		{Name: "Fetcher", Description: "Data extraction job creation and status operations."},
	}

	return api
}

// AssembleHumaAPI is the single runtime/spec-gen assembly path. It guarantees
// problem.Install and document decoration happen before any operation is
// registered.
func AssembleHumaAPI(
	app *fiber.App,
	authEnabled bool,
	handlers OperationHandlers,
	middlewareFactory OperationMiddlewareFactory,
) huma.API {
	api := BuildHumaAPI(app, authEnabled)
	RegisterHumaOperations(app, api, handlers, middlewareFactory)

	return api
}

// GenerateCanonicalSpec renders the deterministic auth-enabled client
// contract committed to the repository.
func GenerateCanonicalSpec() ([]byte, error) {
	app := fiber.New()
	api := AssembleHumaAPI(app, true, OperationHandlers{}, nil)

	return api.OpenAPI().YAML()
}

// ServeHumaSpec mounts the immutable spec snapshot and Scalar UI only when the
// startup configuration explicitly enables the surface.
func ServeHumaSpec(app *fiber.App, api huma.API, logger libLog.Logger, enabled bool) {
	if !enabled {
		return
	}

	openapi.ServeSpec(app, api, logger, "/swagger", DocTitle)
}
