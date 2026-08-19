package in

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cacheRepo "github.com/LerianStudio/fetcher/v2/components/manager/internal/adapters/cache"
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/query"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/model/datasource"
	jobRepo "github.com/LerianStudio/fetcher/v2/pkg/mongodb/job"
	httpUtils "github.com/LerianStudio/fetcher/v2/pkg/net/http"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/lib-commons/v6/commons/net/http/openapi"
	"github.com/LerianStudio/lib-commons/v6/commons/net/http/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupHumaConnectionTestApp(t *testing.T, handler *ConnectionHandler) (*fiber.App, huma.API) {
	t.Helper()

	app := setupConnectionTestApp()
	api := openapi.New(app, app.Group(""), openapi.Config{
		Title:       "Fetcher test API",
		Version:     "test",
		Description: "Typed connection operation tests.",
	})

	problem.Install()

	var handlers OperationHandlers
	bindConnectionHandlers(&handlers, handler)
	registerConnectionOperations(app, api, handlers, func(_, _ string) []fiber.Handler { return nil })

	return app, api
}

func TestHumaConnections_CreateSuccessReturnsTypedBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	var persistedID uuid.UUID

	mockCryptor.EXPECT().Encrypt(gomock.Any(), "secretpassword").Return("encrypted-password", "v1", nil)
	mockConnRepo.EXPECT().FindByName(gomock.Any(), "test-connection").Return(nil, nil)
	mockConnRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, connection *model.Connection) (*model.Connection, error) {
			assert.Equal(t, "test-product", connection.ProductName)
			assert.Equal(t, model.TypePostgreSQL, connection.Type)
			assert.Equal(t, "localhost", connection.Host)
			persistedID = connection.ID
			return connection, nil
		},
	)

	handler := &ConnectionHandler{
		CreateCmd: command.NewCreateConnection(mockCryptor, connectionEngineForConnRepo(t, mockConnRepo, nil)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Product-Name", "test-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var body model.ConnectionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotEqual(t, uuid.Nil, persistedID)
	assert.Equal(t, persistedID, body.ID)
	assert.Equal(t, "test-connection", body.ConfigName)
	assert.Equal(t, string(model.TypePostgreSQL), body.Type)
}

func TestHumaConnections_ListSuccessReturnsTypedPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	first := createTestConnection(uuid.New())
	first.ConfigName = "primary"
	second := createTestConnection(uuid.New())
	second.ConfigName = "secondary"

	mockConnRepo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, filters httpUtils.QueryHeader) ([]*model.Connection, int64, error) {
			assert.Equal(t, 1, filters.Page)
			assert.Equal(t, 10, filters.Limit)
			return []*model.Connection{first, second}, 2, nil
		},
	)

	handler := &ConnectionHandler{
		ListQuery: query.NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("GET", "/v1/management/connections?page=1&limit=10", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var page connectionPage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	require.Len(t, page.Items, 2)
	assert.Equal(t, 2, page.Total)
	assert.Equal(t, "primary", page.Items[0].ConfigName)
	assert.Equal(t, "secondary", page.Items[1].ConfigName)
}

func TestHumaConnections_GetSuccessReturnsTypedConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()
	connection := createTestConnection(connectionID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)

	handler := &ConnectionHandler{
		GetQuery: query.NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("GET", "/v1/management/connections/"+connectionID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var body model.ConnectionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, connectionID, body.ID)
	assert.Equal(t, "test-connection", body.ConfigName)
}

func TestHumaConnections_UpdateSuccessReturnsTypedConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()
	connection := createTestConnection(connectionID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), connection.ConfigName).Return(false, nil)
	mockConnRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, updated *model.Connection) (*model.Connection, error) {
			assert.Equal(t, "new-host", updated.Host)
			assert.Equal(t, 5433, updated.Port)
			return updated, nil
		},
	)

	handler := &ConnectionHandler{
		UpdateCmd: command.NewUpdateConnection(nil, connectionEngineForJobRepo(t, mockConnRepo, mockJobRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest(
		"PATCH",
		"/v1/management/connections/"+connectionID.String(),
		strings.NewReader(`{"host":"new-host","port":5433}`),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var body model.ConnectionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "new-host", body.Host)
	assert.Equal(t, 5433, body.Port)
}

func TestHumaConnections_TestSuccessReturnsLatency(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRateLimiter := query.NewMockRateLimiterStore(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)
	connectionID := uuid.New()
	connection := createTestConnection(connectionID)

	mockRateLimiter.EXPECT().Take(gomock.Any(), gomock.Any()).Return(
		uint64(1), uint64(9), uint64(time.Now().Add(time.Minute).UnixNano()), true, nil,
	)
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)

	handler := &ConnectionHandler{
		TestQuery: query.NewTestConnection(
			mockConnRepo,
			mockCryptor,
			mockRateLimiter,
			func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
				return mockDataSource, nil
			},
			nil,
			nil,
		),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connectionID.String()+"/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var body model.ConnectionTestResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "success", body.Status)
	assert.Equal(t, "Connection successful", body.Message)
	assert.GreaterOrEqual(t, body.LatencyMs, int64(0))
}

func TestHumaConnections_TestDatabaseConnectionErrorPreservesSafeTitleAndScrubsDetail(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockRateLimiter := query.NewMockRateLimiterStore(ctrl)
	connectionID := uuid.New()
	connection := createTestConnection(connectionID)
	secret := "postgres password=super-secret"

	mockRateLimiter.EXPECT().Take(gomock.Any(), gomock.Any()).Return(
		uint64(1), uint64(9), uint64(time.Now().Add(time.Minute).UnixNano()), true, nil,
	)
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)

	handler := &ConnectionHandler{
		TestQuery: query.NewTestConnection(
			mockConnRepo,
			nil,
			mockRateLimiter,
			func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
				return nil, errors.New(secret)
			},
			nil,
			nil,
		),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	resp, err := app.Test(httptest.NewRequest("POST", "/v1/management/connections/"+connectionID.String()+"/test", nil))
	require.NoError(t, err)
	detail := requireConnectionProblem(t, resp, fiber.StatusInternalServerError, "500", "Database Connection Error", "internal error")
	assert.NotContains(t, detail.Detail, secret)
}

func TestHumaConnections_ValidateSchemaSuccessReturnsTypedBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)
	connection := createTestConnection(uuid.New())
	connection.ConfigName = "ds1"

	schema := model.NewDataSourceSchema("ds1")
	schema.AddTable("table1", []string{"field1", "field2"})
	schema.AddTable("table2", []string{"field3"})
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), []string{"ds1"}).Return([]*model.Connection{connection}, nil)
	mockSchemaCache.EXPECT().Get(gomock.Any(), "ds1").Return(schema, nil)

	handler := &ConnectionHandler{
		ValidateSchemaQuery: query.NewValidateSchema(
			mockConnRepo,
			newSchemaEngineForTest(t, nil, mockSchemaCache),
			nil,
		),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest(
		"POST",
		"/v1/management/connections/validate-schema",
		strings.NewReader(validSchemaValidationRequest()),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var body model.SchemaValidationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, model.StatusSuccess, body.Status)
	assert.Empty(t, body.Errors)
}

func TestHumaConnections_GetSchemaSuccessReturnsTypedBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockDataSource := datasource.NewMockDataSource(ctrl)
	connectionID := uuid.New()
	connection := createTestConnection(connectionID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)
	schema := model.NewDataSourceSchema(connection.ConfigName)
	schema.AddTable("users", []string{"id", "name"})
	mockDataSource.EXPECT().GetSchemaInfo(gomock.Any(), gomock.Any()).Return(schema, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)

	handler := &ConnectionHandler{
		GetSchemaQuery: query.NewGetConnectionSchema(
			nil,
			nil,
			connectionEngineForConnRepo(t, mockConnRepo, nil),
			newSchemaEngineForTest(t, func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
				return mockDataSource, nil
			}, nil),
			false,
		),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("GET", "/v1/management/connections/"+connectionID.String()+"/schema", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var body model.ConnectionSchemaResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Len(t, body.Tables, 1)
	assert.Equal(t, "users", body.Tables[0].Name)
}

func TestHumaConnections_GetSchemaDatabaseConnectionErrorPreservesSafeTitleAndScrubsDetail(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()
	connection := createTestConnection(connectionID)
	secret := "mongodb tls failure password=super-secret"

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)

	handler := &ConnectionHandler{
		GetSchemaQuery: query.NewGetConnectionSchema(
			nil,
			nil,
			connectionEngineForConnRepo(t, mockConnRepo, nil),
			newSchemaEngineForTest(t, func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
				return nil, errors.New(secret)
			}, nil),
			false,
		),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	resp, err := app.Test(httptest.NewRequest("GET", "/v1/management/connections/"+connectionID.String()+"/schema", nil))
	require.NoError(t, err)
	detail := requireConnectionProblem(t, resp, fiber.StatusInternalServerError, "500", "Database Connection Error", "internal error")
	assert.NotContains(t, detail.Detail, secret)
}

func TestHumaConnections_CreateValidatesBeforeCallingService(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})

	req := httptest.NewRequest(
		"POST",
		"/v1/management/connections",
		strings.NewReader(`{"configName":"test-connection"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Product-Name", "test-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var detail problem.Detail
	require.NoError(t, json.Unmarshal(body, &detail))
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrMissingFieldsInRequest.Error(), detail.Type)
	assert.Equal(t, "Bad Request", detail.Title)
	assert.Equal(t, fiber.StatusBadRequest, detail.Status)
	assert.Equal(t, "Your request is missing one or more required fields. Please refer to the documentation to ensure all necessary fields are included in your request.", detail.Detail)
	assert.Equal(t, constant.ErrMissingFieldsInRequest.Error(), detail.Code)
	assert.Equal(t, []*huma.ErrorDetail{
		{Message: "databaseName is a required field", Location: "body.databaseName"},
		{Message: "host is a required field", Location: "body.host"},
		{Message: "password is a required field", Location: "body.password"},
		{Message: "port is a required field", Location: "body.port"},
		{Message: "type is a required field", Location: "body.type"},
		{Message: "userName is a required field", Location: "body.userName"},
	}, detail.Errors)
}

func TestHumaConnections_CreateRejectsInvalidJSON(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(`{"configName":`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Product-Name", "test-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusBadRequest, constant.ErrBadRequest.Error(), "Bad Request", "unable to parse request body")
}

func TestHumaConnections_CreateRejectsMissingProductHeader(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusBadRequest, constant.ErrInvalidHeaderParameter.Error(), "Bad Request", "X-Product-Name header is required and must not be empty")
}

func TestHumaConnections_CreateConflictPreservesFETCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	mockCryptor.EXPECT().Encrypt(gomock.Any(), "secretpassword").Return("encrypted-password", "v1", nil)
	mockConnRepo.EXPECT().FindByName(gomock.Any(), "test-connection").Return(createTestConnection(uuid.New()), nil)

	handler := &ConnectionHandler{
		CreateCmd: command.NewCreateConnection(mockCryptor, connectionEngineForConnRepo(t, mockConnRepo, nil)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Product-Name", "test-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusConflict, constant.ErrEntityConflict.Error(), "Conflict", "An entity of type connection with the same unique attributes already exists. Please use different values to avoid conflicts and review the data provided in the request.")
}

func TestHumaConnections_ListEmptyReturnsEmptyItems(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockConnRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(0), nil)

	handler := &ConnectionHandler{
		ListQuery: query.NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	resp, err := app.Test(httptest.NewRequest("GET", "/v1/management/connections", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var page connectionPage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	assert.Empty(t, page.Items)
	assert.Equal(t, 0, page.Total)
}

func TestHumaConnections_ListRejectsInvalidPagination(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})

	resp, err := app.Test(httptest.NewRequest("GET", "/v1/management/connections?sortOrder=invalid", nil))
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusBadRequest, constant.ErrInvalidSortOrder.Error(), "Bad Request", "The 'sort_order' field must be 'asc' or 'desc'. Please provide a valid sort order and try again.")
}

func TestHumaConnections_ListPassesProductFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	connection := createTestConnection(uuid.New())
	connection.ProductName = "my-product"

	mockConnRepo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, filters httpUtils.QueryHeader) ([]*model.Connection, int64, error) {
			assert.Equal(t, "my-product", filters.ProductName)
			return []*model.Connection{connection}, 1, nil
		},
	)

	handler := &ConnectionHandler{
		ListQuery: query.NewListConnections(nil, scopeAuthorityEngine(t, mockConnRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("GET", "/v1/management/connections", nil)
	req.Header.Set("X-Product-Name", "my-product")
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var page connectionPage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	require.Len(t, page.Items, 1)
	assert.Equal(t, "my-product", page.Items[0].ProductName)
}

func TestHumaConnections_ListRejectsInvalidProductHeader(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})

	req := httptest.NewRequest("GET", "/v1/management/connections", nil)
	req.Header.Set("X-Product-Name", "invalid name!@#")
	resp, err := app.Test(req)
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusBadRequest, constant.ErrInvalidHeaderParameter.Error(), "Bad Request", "X-Product-Name can only contain alphanumeric characters, underscores, and hyphens")
}

func TestHumaConnections_GetNotFoundPreservesFETCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(nil, nil)

	handler := &ConnectionHandler{
		GetQuery: query.NewGetConnection(nil, nil, scopeAuthorityEngine(t, mockConnRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	resp, err := app.Test(httptest.NewRequest("GET", "/v1/management/connections/"+connectionID.String(), nil))
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusNotFound, constant.ErrEntityNotFound.Error(), "Not Found", "It was not possible to find the connection entity during the requested flow. Please review the data provided in the request.")
}

func TestHumaConnections_UpdateNotFoundPreservesFETCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(nil, nil)

	handler := &ConnectionHandler{
		UpdateCmd: command.NewUpdateConnection(nil, connectionEngineForJobRepo(t, mockConnRepo, mockJobRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest(
		"PATCH",
		"/v1/management/connections/"+connectionID.String(),
		strings.NewReader(`{"configName":"updated"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusNotFound, constant.ErrEntityNotFound.Error(), "Not Found", "It was not possible to find the connection entity during the requested flow. Please review the data provided in the request.")
}

func TestHumaConnections_UpdateRejectsInvalidJSON(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})
	req := httptest.NewRequest(
		"PATCH",
		"/v1/management/connections/"+uuid.NewString(),
		strings.NewReader(`{"host":`),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusBadRequest, constant.ErrBadRequest.Error(), "Bad Request", "unable to parse request body")
}

func TestHumaConnections_UpdateConflictPreservesFETCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()
	connection := createTestConnection(connectionID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), connection.ConfigName).Return(true, nil)

	handler := &ConnectionHandler{
		UpdateCmd: command.NewUpdateConnection(nil, connectionEngineForJobRepo(t, mockConnRepo, mockJobRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)
	req := httptest.NewRequest(
		"PATCH",
		"/v1/management/connections/"+connectionID.String(),
		strings.NewReader(`{"configName":"updated"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusConflict, constant.ErrJobInProgress.Error(), "Conflict", "cannot update connection with active jobs")
}

func TestHumaConnections_UpdateRejectsInvalidID(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})
	req := httptest.NewRequest(
		"PATCH",
		"/v1/management/connections/not-a-uuid",
		strings.NewReader(`{"configName":"updated"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusBadRequest, constant.ErrInvalidPathParameter.Error(), "Bad Request", "invalid connection id")
}

func TestHumaConnections_DeleteNotFoundPreservesFETCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(nil, nil)

	handler := &ConnectionHandler{
		DeleteCmd: command.NewDeleteConnection(connectionEngineForJobRepo(t, mockConnRepo, mockJobRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	resp, err := app.Test(httptest.NewRequest("DELETE", "/v1/management/connections/"+connectionID.String(), nil))
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusNotFound, constant.ErrEntityNotFound.Error(), "Not Found", "It was not possible to find the connection entity during the requested flow. Please review the data provided in the request.")
}

func TestHumaConnections_DeleteConflictPreservesFETCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()
	connection := createTestConnection(connectionID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), connection.ConfigName).Return(true, nil)

	handler := &ConnectionHandler{
		DeleteCmd: command.NewDeleteConnection(connectionEngineForJobRepo(t, mockConnRepo, mockJobRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	resp, err := app.Test(httptest.NewRequest("DELETE", "/v1/management/connections/"+connectionID.String(), nil))
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusConflict, constant.ErrJobInProgress.Error(), "Conflict", "cannot delete connection with active jobs")
}

func TestHumaConnections_DeleteRejectsInvalidID(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})
	resp, err := app.Test(httptest.NewRequest("DELETE", "/v1/management/connections/not-a-uuid", nil))
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusBadRequest, constant.ErrInvalidPathParameter.Error(), "Bad Request", "invalid connection id")
}

func TestHumaConnections_TestNotFoundPreservesFETCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockRateLimiter := query.NewMockRateLimiterStore(ctrl)
	connectionID := uuid.New()

	mockRateLimiter.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(1), uint64(9), uint64(0), true, nil)
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(nil, nil)

	handler := &ConnectionHandler{
		TestQuery: query.NewTestConnection(mockConnRepo, nil, mockRateLimiter, nil, nil, nil),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	resp, err := app.Test(httptest.NewRequest("POST", "/v1/management/connections/"+connectionID.String()+"/test", nil))
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusNotFound, constant.ErrEntityNotFound.Error(), "Not Found", "It was not possible to find the connection entity during the requested flow. Please review the data provided in the request.")
}

func TestHumaConnections_TestRejectsInvalidID(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})
	resp, err := app.Test(httptest.NewRequest("POST", "/v1/management/connections/not-a-uuid/test", nil))
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusBadRequest, constant.ErrInvalidPathParameter.Error(), "Bad Request", "invalid connection id")
}

func TestHumaConnections_TestRateLimitPreservesResponseContract(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRateLimiter := query.NewMockRateLimiterStore(ctrl)
	connectionID := uuid.New()
	resetAt := uint64(time.Now().Add(time.Minute).UnixNano())
	mockRateLimiter.EXPECT().Take(gomock.Any(), gomock.Any()).Return(uint64(0), uint64(0), resetAt, false, nil)

	handler := &ConnectionHandler{
		TestQuery: query.NewTestConnection(nil, nil, mockRateLimiter, nil, nil, nil),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	resp, err := app.Test(httptest.NewRequest("POST", "/v1/management/connections/"+connectionID.String()+"/test", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, problem.BaseURI+"/429", detail.Type)
	assert.Equal(t, "Too Many Requests", detail.Title)
	assert.Equal(t, fiber.StatusTooManyRequests, detail.Status)
	assert.Regexp(t, `^Connection test limit reached\. Try again in [1-9][0-9]* seconds\.$`, detail.Detail)
	assert.Equal(t, "429", detail.Code)
	assert.Empty(t, detail.Errors)
}

func TestHumaConnections_InvalidIDReturnsManualProblem(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})

	req := httptest.NewRequest("GET", "/v1/management/connections/not-a-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrInvalidPathParameter.Error(), detail.Type)
	assert.Equal(t, "Bad Request", detail.Title)
	assert.Equal(t, fiber.StatusBadRequest, detail.Status)
	assert.Equal(t, constant.ErrInvalidPathParameter.Error(), detail.Code)
	assert.Equal(t, "invalid connection id", detail.Detail)
	assert.Empty(t, detail.Errors)
}

func TestHumaConnections_ServiceErrorIsMappedAndScrubbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	mockCryptor.EXPECT().Encrypt(gomock.Any(), "secretpassword").Return("encrypted-password", "v1", nil)
	mockConnRepo.EXPECT().FindByName(gomock.Any(), "test-connection").Return(nil, nil)
	mockConnRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	handler := &ConnectionHandler{
		CreateCmd: command.NewCreateConnection(mockCryptor, connectionEngineForConnRepo(t, mockConnRepo, nil)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Product-Name", "test-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(body), assert.AnError.Error())

	var detail problem.Detail
	require.NoError(t, json.Unmarshal(body, &detail))
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrInternalServer.Error(), detail.Type)
	assert.Equal(t, "Internal Server Error", detail.Title)
	assert.Equal(t, fiber.StatusInternalServerError, detail.Status)
	assert.Equal(t, "internal error", detail.Detail)
	assert.Equal(t, constant.ErrInternalServer.Error(), detail.Code)
	assert.Empty(t, detail.Errors)
}

func TestHumaConnections_DeleteReturnsNoContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	connectionID := uuid.New()
	existing := createTestConnection(connectionID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(existing, nil)
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), "test-connection").Return(false, nil)
	mockConnRepo.EXPECT().Delete(gomock.Any(), connectionID, gomock.Any()).Return(nil)

	handler := &ConnectionHandler{
		DeleteCmd: command.NewDeleteConnection(connectionEngineForJobRepo(t, mockConnRepo, mockJobRepo)),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("DELETE", "/v1/management/connections/"+connectionID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Empty(t, body)
}

func TestHumaConnections_ValidateSchemaFailureReturnsStructuredProblemDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCache := newNoopSchemaCache()

	connection := createTestConnection(uuid.New())
	connection.ConfigName = "ds1"
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return([]*model.Connection{connection}, nil)

	failingFactory := func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
		return nil, assert.AnError
	}

	handler := &ConnectionHandler{
		ValidateSchemaQuery: query.NewValidateSchema(
			mockConnRepo,
			newSchemaEngineForTest(t, failingFactory, mockCache),
			nil,
		),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(validSchemaValidationRequest()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var topLevel map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &topLevel))
	assert.NotContains(t, topLevel, "message", "RFC 9457 uses detail plus structured errors, not a legacy top-level message")

	var detail problem.Detail
	require.NoError(t, json.Unmarshal(body, &detail))
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrSchemaValidationFailed.Error(), detail.Type)
	assert.Equal(t, "Unprocessable Entity", detail.Title)
	assert.Equal(t, fiber.StatusUnprocessableEntity, detail.Status)
	assert.Equal(t, "Schema validation found inconsistencies.", detail.Detail)
	assert.Equal(t, constant.ErrSchemaValidationFailed.Error(), detail.Code)
	assert.Equal(t, []*huma.ErrorDetail{{
		Message:  "DATA_SOURCE_DOWN",
		Location: "body.mappedFields.ds1",
		Value: map[string]any{
			"dataSourceId": "ds1",
			"field":        "",
			"table":        "",
		},
	}}, detail.Errors)
}

func TestHumaConnections_ValidateSchemaReportsMissingDatasource(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)
	existing := createTestConnection(uuid.New())
	existing.ConfigName = "existing_ds"

	schema := model.NewDataSourceSchema("existing_ds")
	schema.AddTable("table1", []string{"field1"})
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return([]*model.Connection{existing}, nil)
	mockSchemaCache.EXPECT().Get(gomock.Any(), "existing_ds").Return(schema, nil)

	handler := &ConnectionHandler{
		ValidateSchemaQuery: query.NewValidateSchema(
			mockConnRepo,
			newSchemaEngineForTest(t, nil, mockSchemaCache),
			nil,
		),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)
	payload := `{
		"mappedFields": {
			"existing_ds": {"table1": ["field1"]},
			"unknown_ds": {"table1": ["field1"]}
		}
	}`
	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrSchemaValidationFailed.Error(), detail.Type)
	assert.Equal(t, "Unprocessable Entity", detail.Title)
	assert.Equal(t, fiber.StatusUnprocessableEntity, detail.Status)
	assert.Equal(t, "Schema validation found inconsistencies.", detail.Detail)
	assert.Equal(t, constant.ErrSchemaValidationFailed.Error(), detail.Code)
	assert.Equal(t, []*huma.ErrorDetail{{
		Message:  model.ErrTypeDataSourceNotFound,
		Location: "body.mappedFields.unknown_ds",
		Value: map[string]any{
			"dataSourceId": "unknown_ds",
			"field":        "",
			"table":        "",
		},
	}}, detail.Errors)
}

func TestHumaConnections_ValidateSchemaRejectsInvalidJSON(t *testing.T) {
	app, _ := setupHumaConnectionTestApp(t, &ConnectionHandler{})
	req := httptest.NewRequest(
		"POST",
		"/v1/management/connections/validate-schema",
		strings.NewReader(`{"mappedFields":`),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	requireConnectionProblem(t, resp, fiber.StatusBadRequest, constant.ErrBadRequest.Error(), "Bad Request", "unable to parse request body")
}

func TestHumaConnections_ValidateSchemaInternalErrorIsScrubbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	handler := &ConnectionHandler{
		ValidateSchemaQuery: query.NewValidateSchema(
			mockConnRepo,
			newSchemaEngineForTest(t, nil, newNoopSchemaCache()),
			nil,
		),
	}
	app, _ := setupHumaConnectionTestApp(t, handler)
	req := httptest.NewRequest(
		"POST",
		"/v1/management/connections/validate-schema",
		strings.NewReader(validSchemaValidationRequest()),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	detail := requireConnectionProblem(t, resp, fiber.StatusInternalServerError, constant.ErrInternalServer.Error(), "Internal Server Error", "internal error")
	assert.NotContains(t, detail.Detail, assert.AnError.Error())
}

func TestRegisterConnectionOperations_OpenAPIContract(t *testing.T) {
	_, api := setupHumaConnectionTestApp(t, &ConnectionHandler{})
	document := openAPIDocumentMap(t, api)

	create := openAPIOperation(t, document, "/v1/management/connections", "post")
	assert.Equal(t, "create-connection", create["operationId"])
	assert.Equal(t, "Create connection", create["summary"])
	assert.Equal(t, []any{"Connections"}, create["tags"])
	assertResponseStatuses(t, create, "201", "400", "409", "500")
	assertOpenAPIParameter(t, create, "header", "X-Product-Name", true)
	assertRequestSchemaRef(t, create, "ConnectionInput")

	list := openAPIOperation(t, document, "/v1/management/connections", "get")
	assertResponseStatuses(t, list, "200", "400", "500")
	assertOpenAPIParameter(t, list, "header", "X-Product-Name", false)
	assertOpenAPIParameter(t, list, "query", "page", false)
	assertOpenAPIParameter(t, list, "query", "limit", false)
	assertOpenAPIParameter(t, list, "query", "sortOrder", false)

	validateSchema := openAPIOperation(t, document, "/v1/management/connections/validate-schema", "post")
	assert.Equal(t, "validate-schema", validateSchema["operationId"])
	assertResponseStatuses(t, validateSchema, "200", "400", "422", "500")
	assertRequestSchemaRef(t, validateSchema, "SchemaValidationRequest")

	item := openAPIOperation(t, document, "/v1/management/connections/{id}", "get")
	assertOpenAPIParameter(t, item, "path", "id", true)
	assertResponseStatuses(t, item, "200", "400", "404", "500")

	update := openAPIOperation(t, document, "/v1/management/connections/{id}", "patch")
	assertRequestSchemaRef(t, update, "ConnectionUpdateInput")
	assertResponseStatuses(t, update, "200", "400", "404", "409", "500")

	deleteOperation := openAPIOperation(t, document, "/v1/management/connections/{id}", "delete")
	assertResponseStatuses(t, deleteOperation, "204", "400", "404", "409", "500")

	testOperation := openAPIOperation(t, document, "/v1/management/connections/{id}/test", "post")
	assert.Equal(t, "test-connection", testOperation["operationId"])
	assertResponseStatuses(t, testOperation, "200", "400", "404", "429", "500")

	getSchema := openAPIOperation(t, document, "/v1/management/connections/{id}/schema", "get")
	assert.Equal(t, "get-connection-schema", getSchema["operationId"])
	assertResponseStatuses(t, getSchema, "200", "400", "404", "500")
}

func openAPIDocumentMap(t *testing.T, api huma.API) map[string]any {
	t.Helper()

	encoded, err := json.Marshal(api.OpenAPI())
	require.NoError(t, err)

	var document map[string]any
	require.NoError(t, json.Unmarshal(encoded, &document))

	return document
}

func openAPIOperation(t *testing.T, document map[string]any, path, method string) map[string]any {
	t.Helper()

	paths, ok := document["paths"].(map[string]any)
	require.True(t, ok)
	pathItem, ok := paths[path].(map[string]any)
	require.Truef(t, ok, "OpenAPI path %s is missing", path)
	operation, ok := pathItem[method].(map[string]any)
	require.Truef(t, ok, "OpenAPI operation %s %s is missing", method, path)

	return operation
}

func assertResponseStatuses(t *testing.T, operation map[string]any, statuses ...string) {
	t.Helper()

	responses, ok := operation["responses"].(map[string]any)
	require.True(t, ok)
	for _, status := range statuses {
		assert.Contains(t, responses, status)
	}
}

func assertOpenAPIParameter(t *testing.T, operation map[string]any, in, name string, required bool) {
	t.Helper()

	parameters, ok := operation["parameters"].([]any)
	require.True(t, ok)
	for _, candidate := range parameters {
		parameter, ok := candidate.(map[string]any)
		if ok && parameter["in"] == in && parameter["name"] == name {
			if required {
				assert.Equal(t, true, parameter["required"])
			} else {
				assert.NotEqual(t, true, parameter["required"])
			}
			return
		}
	}

	assert.Failf(t, "parameter is missing", "%s parameter %q is not declared", in, name)
}

func assertRequestSchemaRef(t *testing.T, operation map[string]any, component string) {
	t.Helper()

	requestBody, ok := operation["requestBody"].(map[string]any)
	require.True(t, ok)
	content, ok := requestBody["content"].(map[string]any)
	require.True(t, ok)
	mediaType, ok := content["application/json"].(map[string]any)
	require.True(t, ok)
	schema, ok := mediaType["schema"].(map[string]any)
	require.True(t, ok)
	ref, ok := schema["$ref"].(string)
	require.True(t, ok)
	assert.Equal(t, "#/components/schemas/"+component, ref)
	assert.NotContains(t, ref, "rawJSONInput")
}

func requireConnectionProblem(t *testing.T, response *http.Response, status int, code, title, expectedDetail string) problem.Detail {
	t.Helper()
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	require.Equal(t, status, response.StatusCode)
	assert.Equal(t, "application/problem+json", response.Header.Get("Content-Type"))

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	var detail problem.Detail
	require.NoError(t, json.Unmarshal(body, &detail), "problem body: %s", body)
	expectedType := ""
	if code != "" {
		expectedType = problem.BaseURI + "/" + code
	}
	assert.Equal(t, expectedType, detail.Type)
	assert.Equal(t, title, detail.Title)
	assert.Equal(t, status, detail.Status)
	assert.Equal(t, expectedDetail, detail.Detail)
	assert.Equal(t, code, detail.Code)
	assert.Empty(t, detail.Errors)

	return detail
}
