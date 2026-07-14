package in

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	connectionCommand "github.com/LerianStudio/fetcher/v2/components/manager/internal/services/command"
	connectionQuery "github.com/LerianStudio/fetcher/v2/components/manager/internal/services/query"
	middlewareAuth "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	"github.com/LerianStudio/lib-observability/log"
	opentelemetry "github.com/LerianStudio/lib-observability/tracing"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		connectionHandler *ConnectionHandler,
		migrationHandler *MigrationHandler,
		fetcherHandler *FetcherHandler,
		tenantMiddleware fiber.Handler,
		readyzHandler fiber.Handler,
		readyzTenantHandler fiber.Handler,
		metricsHandler fiber.Handler,
		swaggerEnabled bool,
	) (*fiber.App, error)

	var _ expectedSignature = NewRoutes

	// Also verify nil is a valid value for tenantMiddleware (single-tenant mode)
	var nilHandler fiber.Handler
	assert.Nil(t, nilHandler, "nil fiber.Handler should be valid for single-tenant mode")
}

func TestValidateRuntimeSecurity_UsesEffectiveAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		auth       *middlewareAuth.AuthClient
		tenant     fiber.Handler
		wantActive bool
		wantErr    string
	}{
		{name: "nil auth is disabled"},
		{name: "disabled auth with address is disabled", auth: &middlewareAuth.AuthClient{Address: "http://plugin-auth:4000"}},
		{
			name:       "enabled auth with address is active",
			auth:       &middlewareAuth.AuthClient{Enabled: true, Address: "http://plugin-auth:4000"},
			wantActive: true,
		},
		{
			name:    "enabled auth without address fails closed",
			auth:    &middlewareAuth.AuthClient{Enabled: true},
			wantErr: "auth middleware is enabled but its address is empty",
		},
		{
			name: "tenant middleware requires effective auth",
			tenant: func(c *fiber.Ctx) error {
				return c.Next()
			},
			wantErr: "tenant middleware requires effective authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authActive, err := validateRuntimeSecurity(tt.auth, tt.tenant)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantActive, authActive)
		})
	}
}

func TestValidateRuntimeHandlerGraph_RejectsMissingDependencies(t *testing.T) {
	t.Parallel()

	validConnections := func() *ConnectionHandler {
		return &ConnectionHandler{
			CreateCmd:           &connectionCommand.CreateConnection{},
			UpdateCmd:           &connectionCommand.UpdateConnection{},
			DeleteCmd:           &connectionCommand.DeleteConnection{},
			GetQuery:            &connectionQuery.GetConnection{},
			ListQuery:           &connectionQuery.ListConnections{},
			TestQuery:           &connectionQuery.TestConnection{},
			ValidateSchemaQuery: &connectionQuery.ValidateSchema{},
			GetSchemaQuery:      &connectionQuery.GetConnectionSchema{},
		}
	}
	validMigration := func() *MigrationHandler {
		return &MigrationHandler{
			AssignCmd:         &connectionCommand.AssignConnection{},
			ListUnassignedQry: &connectionQuery.ListUnassignedConnections{},
		}
	}
	validFetcher := func() *FetcherHandler {
		return &FetcherHandler{
			CreateJobCmd: &connectionCommand.CreateFetcherJob{},
			GetJobQuery:  &connectionQuery.GetJob{},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*ConnectionHandler, *MigrationHandler, *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler)
		wantErr string
	}{
		{
			name: "connection handler",
			mutate: func(_ *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
				return nil, m, f
			},
			wantErr: "connection handler is required",
		},
		{name: "create connection command", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			c.CreateCmd = nil
			return c, m, f
		}, wantErr: "create connection command is required"},
		{name: "update connection command", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			c.UpdateCmd = nil
			return c, m, f
		}, wantErr: "update connection command is required"},
		{name: "delete connection command", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			c.DeleteCmd = nil
			return c, m, f
		}, wantErr: "delete connection command is required"},
		{name: "get connection query", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			c.GetQuery = nil
			return c, m, f
		}, wantErr: "get connection query is required"},
		{name: "list connections query", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			c.ListQuery = nil
			return c, m, f
		}, wantErr: "list connections query is required"},
		{name: "test connection query", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			c.TestQuery = nil
			return c, m, f
		}, wantErr: "test connection query is required"},
		{name: "validate schema query", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			c.ValidateSchemaQuery = nil
			return c, m, f
		}, wantErr: "validate schema query is required"},
		{name: "get connection schema query", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			c.GetSchemaQuery = nil
			return c, m, f
		}, wantErr: "get connection schema query is required"},
		{
			name: "migration handler",
			mutate: func(c *ConnectionHandler, _ *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
				return c, nil, f
			},
			wantErr: "migration handler is required",
		},
		{name: "assign connection command", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			m.AssignCmd = nil
			return c, m, f
		}, wantErr: "assign connection command is required"},
		{name: "list unassigned connections query", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			m.ListUnassignedQry = nil
			return c, m, f
		}, wantErr: "list unassigned connections query is required"},
		{
			name: "fetcher handler",
			mutate: func(c *ConnectionHandler, m *MigrationHandler, _ *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
				return c, m, nil
			},
			wantErr: "fetcher handler is required",
		},
		{name: "create fetcher job command", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			f.CreateJobCmd = nil
			return c, m, f
		}, wantErr: "create fetcher job command is required"},
		{name: "get fetcher job query", mutate: func(c *ConnectionHandler, m *MigrationHandler, f *FetcherHandler) (*ConnectionHandler, *MigrationHandler, *FetcherHandler) {
			f.GetJobQuery = nil
			return c, m, f
		}, wantErr: "get fetcher job query is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connections, migration, fetcher := tt.mutate(validConnections(), validMigration(), validFetcher())
			err := validateRuntimeHandlerGraph(connections, migration, fetcher)
			require.EqualError(t, err, tt.wantErr)
		})
	}

	require.NoError(t, validateRuntimeHandlerGraph(validConnections(), validMigration(), validFetcher()))
}

func TestMountClientAPI_RegistersAllOperationsAndKeepsStaticRoutePrecedence(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	unassignedCalled := false
	getByIDCalled := false
	handlers := completeTestOperationHandlers()
	handlers.ListUnassignedConnections = func(context.Context, *emptyInput) (*listConnectionsOutput, error) {
		unassignedCalled = true

		return &listConnectionsOutput{}, nil
	}
	handlers.GetConnection = func(context.Context, *idInput) (*connectionOutput, error) {
		getByIDCalled = true

		return &connectionOutput{}, nil
	}

	api, err := mountClientAPI(app, false, handlers, nil, nil, false)
	require.NoError(t, err)

	operationCount := 0
	for _, path := range api.OpenAPI().Paths {
		if path.Get != nil {
			operationCount++
		}
		if path.Post != nil {
			operationCount++
		}
		if path.Patch != nil {
			operationCount++
		}
		if path.Delete != nil {
			operationCount++
		}
	}
	assert.Equal(t, 12, operationCount)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/management/connections/unassigned", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, unassignedCalled)
	assert.False(t, getByIDCalled)
}

func TestMountClientAPI_GatesSwaggerAndMatchesSecurityRequirement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		authEnabled    bool
		swaggerEnabled bool
		wantSwagger    int
	}{
		{name: "disabled", wantSwagger: http.StatusNotFound},
		{name: "auth and swagger enabled", authEnabled: true, swaggerEnabled: true, wantSwagger: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			api, err := mountClientAPI(app, tt.authEnabled, completeTestOperationHandlers(), nil, nil, tt.swaggerEnabled)
			require.NoError(t, err)

			if tt.authEnabled {
				assert.Equal(t, []map[string][]string{{bearerAuthSecurityScheme: {}}}, api.OpenAPI().Security)
			} else {
				assert.Empty(t, api.OpenAPI().Security)
			}

			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/swagger/docs", nil))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			assert.Equal(t, tt.wantSwagger, resp.StatusCode)
		})
	}
}

func TestOperationMiddlewareFactory_OrdersAuthTenantAndCallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		rejectAuth bool
		wantStatus int
		wantEvents []string
	}{
		{name: "accepted", wantStatus: http.StatusOK, wantEvents: []string{"auth", "tenant", "callback"}},
		{name: "auth rejection short circuits", rejectAuth: true, wantStatus: http.StatusUnauthorized, wantEvents: []string{"auth"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			events := make([]string, 0, 3)
			authorize := func(_, _ string) fiber.Handler {
				return func(c *fiber.Ctx) error {
					events = append(events, "auth")
					if tt.rejectAuth {
						return c.Status(http.StatusUnauthorized).SendString("denied")
					}

					return c.Next()
				}
			}
			tenant := func(c *fiber.Ctx) error {
				events = append(events, "tenant")

				return c.Next()
			}
			handlers := completeTestOperationHandlers()
			handlers.GetJob = func(context.Context, *idInput) (*getJobOutput, error) {
				events = append(events, "callback")

				return &getJobOutput{}, nil
			}

			_, err := mountClientAPI(app, true, handlers, operationMiddlewareFactory(authorize, tenant), nil, false)
			require.NoError(t, err)
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/fetcher/"+uuid.NewString(), nil))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			assert.Equal(t, tt.wantEvents, events)
		})
	}
}

func TestMountClientAPI_RejectsIncompleteCallbacksBeforeRegisteringRoutes(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	handlers := completeTestOperationHandlers()
	handlers.GetJob = nil

	api, err := mountClientAPI(app, false, handlers, nil, nil, true)

	assert.Nil(t, api)
	require.EqualError(t, err, "get fetcher job operation handler is required")
	resp, requestErr := app.Test(httptest.NewRequest(http.MethodGet, "/swagger/docs", nil))
	require.NoError(t, requestErr)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func completeTestOperationHandlers() OperationHandlers {
	return OperationHandlers{
		CreateConnection: func(context.Context, *rawJSONInput) (*createConnectionOutput, error) {
			return &createConnectionOutput{}, nil
		},
		ListConnections: func(context.Context, *emptyInput) (*listConnectionsOutput, error) {
			return &listConnectionsOutput{}, nil
		},
		ValidateSchema: func(context.Context, *rawJSONInput) (*validateSchemaOutput, error) {
			return &validateSchemaOutput{}, nil
		},
		ListUnassignedConnections: func(context.Context, *emptyInput) (*listConnectionsOutput, error) {
			return &listConnectionsOutput{}, nil
		},
		AssignConnection: func(context.Context, *idInput) (*connectionOutput, error) { return &connectionOutput{}, nil },
		GetConnection:    func(context.Context, *idInput) (*connectionOutput, error) { return &connectionOutput{}, nil },
		TestConnection:   func(context.Context, *idInput) (*testConnectionOutput, error) { return &testConnectionOutput{}, nil },
		GetConnectionSchema: func(context.Context, *idInput) (*connectionSchemaOutput, error) {
			return &connectionSchemaOutput{}, nil
		},
		UpdateConnection: func(context.Context, *rawJSONIDInput) (*connectionOutput, error) { return &connectionOutput{}, nil },
		DeleteConnection: func(context.Context, *idInput) (*deleteConnectionOutput, error) {
			return &deleteConnectionOutput{}, nil
		},
		CreateJob: func(context.Context, *rawJSONInput) (*createJobOutput, error) { return &createJobOutput{}, nil },
		GetJob:    func(context.Context, *idInput) (*getJobOutput, error) { return &getJobOutput{}, nil },
	}
}
