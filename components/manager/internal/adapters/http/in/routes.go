package in

import (
	"errors"
	"strings"

	"github.com/LerianStudio/fetcher/v2/pkg/bootstrap/readyz"
	"github.com/LerianStudio/fetcher/v2/pkg/net/http"
	middlewareAuth "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	commonsHttp "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	"github.com/LerianStudio/lib-observability/log"
	obsMiddleware "github.com/LerianStudio/lib-observability/middleware"
	opentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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
	swaggerEnabled bool,
) (*fiber.App, error) {
	authEnabled, err := validateRuntimeSecurity(auth, ttMiddleware)
	if err != nil {
		return nil, err
	}

	if err = validateRuntimeHandlerGraph(connectionHandler, migrationHandler, fetcherHandler); err != nil {
		return nil, err
	}

	handlers := NewOperationHandlers(connectionHandler, migrationHandler, fetcherHandler)

	var authorize operationAuthorizer
	if authEnabled {
		authorize = func(resource, action string) fiber.Handler {
			return auth.Authorize(applicationName, resource, action)
		}
	}

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

	_, err = mountClientAPI(
		f,
		authEnabled,
		handlers,
		operationMiddlewareFactory(authorize, ttMiddleware),
		lg,
		swaggerEnabled,
	)
	if err != nil {
		return nil, err
	}

	f.Use(tlMid.EndTracingSpans)

	return f, nil
}

type operationAuthorizer func(resource, action string) fiber.Handler

func validateRuntimeSecurity(auth *middlewareAuth.AuthClient, tenantMiddleware fiber.Handler) (bool, error) {
	authEnabled := auth != nil && auth.Enabled && strings.TrimSpace(auth.Address) != ""
	if auth != nil && auth.Enabled && !authEnabled {
		return false, errors.New("auth middleware is enabled but its address is empty")
	}

	if tenantMiddleware != nil && !authEnabled {
		return false, errors.New("tenant middleware requires effective authentication")
	}

	return authEnabled, nil
}

func validateRuntimeHandlerGraph(
	connections *ConnectionHandler,
	migration *MigrationHandler,
	fetcher *FetcherHandler,
) error {
	if connections == nil {
		return errors.New("connection handler is required")
	}

	connectionDependencies := []struct {
		missing bool
		name    string
	}{
		{connections.CreateCmd == nil, "create connection command"},
		{connections.UpdateCmd == nil, "update connection command"},
		{connections.DeleteCmd == nil, "delete connection command"},
		{connections.GetQuery == nil, "get connection query"},
		{connections.ListQuery == nil, "list connections query"},
		{connections.TestQuery == nil, "test connection query"},
		{connections.ValidateSchemaQuery == nil, "validate schema query"},
		{connections.GetSchemaQuery == nil, "get connection schema query"},
	}
	for _, dependency := range connectionDependencies {
		if dependency.missing {
			return errors.New(dependency.name + " is required")
		}
	}

	if migration == nil {
		return errors.New("migration handler is required")
	}

	if migration.AssignCmd == nil {
		return errors.New("assign connection command is required")
	}

	if migration.ListUnassignedQry == nil {
		return errors.New("list unassigned connections query is required")
	}

	if fetcher == nil {
		return errors.New("fetcher handler is required")
	}

	if fetcher.CreateJobCmd == nil {
		return errors.New("create fetcher job command is required")
	}

	if fetcher.GetJobQuery == nil {
		return errors.New("get fetcher job query is required")
	}

	return nil
}

func operationMiddlewareFactory(
	authorize operationAuthorizer,
	tenantMiddleware fiber.Handler,
) OperationMiddlewareFactory {
	return func(resource, action string) []fiber.Handler {
		middlewares := make([]fiber.Handler, 0, 4)
		if authorize != nil {
			middlewares = append(middlewares, problemMiddlewareChain(
				authorize(resource, action),
			)...)
		}

		if tenantMiddleware != nil {
			middlewares = append(middlewares, problemMiddlewareChain(tenantMiddleware)...)
		}

		return middlewares
	}
}

func mountClientAPI(
	app *fiber.App,
	authEnabled bool,
	handlers OperationHandlers,
	middlewareFactory OperationMiddlewareFactory,
	logger log.Logger,
	swaggerEnabled bool,
) (huma.API, error) {
	if err := validateOperationCallbacks(handlers); err != nil {
		return nil, err
	}

	api := AssembleHumaAPI(app, authEnabled, handlers, middlewareFactory)
	ServeHumaSpec(app, api, logger, swaggerEnabled)

	return api, nil
}

func validateOperationCallbacks(handlers OperationHandlers) error {
	callbacks := []struct {
		missing bool
		name    string
	}{
		{handlers.CreateConnection == nil, "create connection"},
		{handlers.ListConnections == nil, "list connections"},
		{handlers.ValidateSchema == nil, "validate schema"},
		{handlers.ListUnassignedConnections == nil, "list unassigned connections"},
		{handlers.AssignConnection == nil, "assign connection"},
		{handlers.GetConnection == nil, "get connection"},
		{handlers.TestConnection == nil, "test connection"},
		{handlers.GetConnectionSchema == nil, "get connection schema"},
		{handlers.UpdateConnection == nil, "update connection"},
		{handlers.DeleteConnection == nil, "delete connection"},
		{handlers.CreateJob == nil, "create fetcher job"},
		{handlers.GetJob == nil, "get fetcher job"},
	}

	for _, callback := range callbacks {
		if callback.missing {
			return errors.New(callback.name + " operation handler is required")
		}
	}

	return nil
}
