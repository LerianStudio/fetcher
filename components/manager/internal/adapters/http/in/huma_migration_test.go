package in

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/v2/components/manager/internal/services/query"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/v2/pkg/net/http"
	connRepo "github.com/LerianStudio/fetcher/v2/pkg/ports/connection"
	"github.com/LerianStudio/lib-commons/v5/commons/net/http/problem"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHumaMigration_OpenAPIContract(t *testing.T) {
	app := fiber.New()
	api := BuildHumaAPI(app, false)

	var middlewareCalls []string
	registerMigrationOperations(app, api, OperationHandlers{
		ListUnassignedConnections: func(context.Context, *emptyInput) (*listConnectionsOutput, error) {
			return &listConnectionsOutput{}, nil
		},
		AssignConnection: func(context.Context, *idInput) (*connectionOutput, error) {
			return &connectionOutput{}, nil
		},
	}, func(resource, action string) []fiber.Handler {
		middlewareCalls = append(middlewareCalls, resource+"/"+action)
		return nil
	})

	document := openAPIDocumentMap(t, api)

	list := openAPIOperation(t, document, "/v1/management/connections/unassigned", "get")
	assert.Equal(t, []any{"Migration"}, list["tags"])
	assertResponseStatuses(t, list, "200", "400", "401", "403", "500")
	assertOpenAPIParameter(t, list, "query", "page", false)
	assertOpenAPIParameter(t, list, "query", "limit", false)
	assertOpenAPIParameter(t, list, "query", "sortOrder", false)
	assertOpenAPIParameter(t, list, "query", "startDate", false)
	assertOpenAPIParameter(t, list, "query", "endDate", false)

	assign := openAPIOperation(t, document, "/v1/management/connections/{id}/assign", "post")
	assert.Equal(t, []any{"Migration"}, assign["tags"])
	assertResponseStatuses(t, assign, "200", "400", "401", "403", "404", "409", "500")
	assertOpenAPIParameter(t, assign, "path", "id", true)
	assertOpenAPIParameter(t, assign, "header", "X-Product-Name", true)

	assert.ElementsMatch(t, []string{"connections/get", "connections/post"}, middlewareCalls)
}

func TestHumaMigration_RegisterHumaOperationsDoesNotShadowStaticRoute(t *testing.T) {
	app := fiber.New()
	api := BuildHumaAPI(app, false)

	migrationCalled := false
	getConnectionCalled := false
	RegisterHumaOperations(app, api, OperationHandlers{
		ListUnassignedConnections: func(context.Context, *emptyInput) (*listConnectionsOutput, error) {
			migrationCalled = true
			return &listConnectionsOutput{Body: connectionPage{Items: []*model.ConnectionResponse{}}}, nil
		},
		GetConnection: func(context.Context, *idInput) (*connectionOutput, error) {
			getConnectionCalled = true
			return &connectionOutput{}, nil
		},
	}, func(_, _ string) []fiber.Handler { return nil })

	request := httptest.NewRequest("GET", "/v1/management/connections/unassigned", nil)
	response, err := app.Test(request)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	require.Equal(t, fiber.StatusOK, response.StatusCode)
	assert.True(t, migrationCalled)
	assert.False(t, getConnectionCalled)
}

func TestHumaMigration_CallbacksExecuteAndSerialize(t *testing.T) {
	app := fiber.New()
	api := BuildHumaAPI(app, false)
	connectionID := uuid.New()

	listCalled := false
	assignCalled := false
	postMiddlewareCalled := false
	registerMigrationOperations(app, api, OperationHandlers{
		ListUnassignedConnections: func(context.Context, *emptyInput) (*listConnectionsOutput, error) {
			listCalled = true
			return &listConnectionsOutput{Body: connectionPage{
				Items: []*model.ConnectionResponse{{ID: connectionID}},
				Page:  1,
				Limit: 10,
				Total: 1,
			}}, nil
		},
		AssignConnection: func(_ context.Context, input *idInput) (*connectionOutput, error) {
			assignCalled = true
			assert.Equal(t, connectionID.String(), input.ID)
			return &connectionOutput{Body: model.ConnectionResponse{ID: connectionID}}, nil
		},
	}, func(resource, action string) []fiber.Handler {
		if resource != "connections" || action != "post" {
			return nil
		}

		return []fiber.Handler{func(ctx *fiber.Ctx) error {
			postMiddlewareCalled = true
			assert.Equal(t, "midaz", ctx.Get("X-Product-Name"))
			return ctx.Next()
		}}
	})

	listRequest := httptest.NewRequest(
		"GET",
		"/v1/management/connections/unassigned?page=1&limit=10&sortOrder=desc",
		nil,
	)
	listResponse, err := app.Test(listRequest)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, listResponse.Body.Close()) })
	require.Equal(t, fiber.StatusOK, listResponse.StatusCode)

	var page connectionPage
	require.NoError(t, json.NewDecoder(listResponse.Body).Decode(&page))
	assert.True(t, listCalled)
	require.Len(t, page.Items, 1)
	assert.Equal(t, connectionID, page.Items[0].ID)
	assert.Equal(t, 1, page.Total)

	assignRequest := httptest.NewRequest(
		"POST",
		"/v1/management/connections/"+connectionID.String()+"/assign",
		nil,
	)
	assignRequest.Header.Set("X-Product-Name", "midaz")
	assignResponse, err := app.Test(assignRequest)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, assignResponse.Body.Close()) })
	require.Equal(t, fiber.StatusOK, assignResponse.StatusCode)

	var assigned model.ConnectionResponse
	require.NoError(t, json.NewDecoder(assignResponse.Body).Decode(&assigned))
	assert.True(t, assignCalled)
	assert.True(t, postMiddlewareCalled)
	assert.Equal(t, connectionID, assigned.ID)
}

func setupHumaMigrationBinderTestApp(t *testing.T, handler *MigrationHandler) *fiber.App {
	t.Helper()

	app := setupMigrationTestApp()
	api := BuildHumaAPI(app, false)
	var handlers OperationHandlers
	bindMigrationHandlers(&handlers, handler)
	registerMigrationOperations(app, api, handlers, func(_, _ string) []fiber.Handler { return nil })

	return app
}

func TestHumaMigration_ListBinderForwardsQueriesAndSerializesPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()

	gomock.InOrder(
		mockConnRepo.EXPECT().ListUnassigned(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, filters httpUtils.QueryHeader) ([]*model.Connection, int64, error) {
				assert.Equal(t, 1, filters.Page)
				assert.Equal(t, 10, filters.Limit)
				assert.Equal(t, "desc", filters.SortOrder)
				assert.Empty(t, filters.Metadata)
				assert.False(t, filters.UseMetadata)
				assert.Empty(t, filters.Type)

				return nil, 0, nil
			},
		),
		mockConnRepo.EXPECT().ListUnassigned(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, filters httpUtils.QueryHeader) ([]*model.Connection, int64, error) {
				assert.Equal(t, 2, filters.Page)
				assert.Equal(t, 25, filters.Limit)
				assert.Equal(t, "asc", filters.SortOrder)
				assert.Equal(t, "POSTGRESQL", filters.Type)
				assert.Equal(t, map[string]string{"metadata.region": "br"}, filters.Metadata)
				assert.True(t, filters.UseMetadata)

				return []*model.Connection{createTestConnectionForMigration(connectionID)}, 1, nil
			},
		),
	)

	app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{
		ListUnassignedQry: query.NewListUnassignedConnections(mockConnRepo),
	})

	defaultRequest := httptest.NewRequest("GET", "/v1/management/connections/unassigned", nil)
	defaultResponse, err := app.Test(defaultRequest)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, defaultResponse.Body.Close()) })
	require.Equal(t, fiber.StatusOK, defaultResponse.StatusCode)

	var defaultPage connectionPage
	require.NoError(t, json.NewDecoder(defaultResponse.Body).Decode(&defaultPage))
	assert.Equal(t, 1, defaultPage.Page)
	assert.Equal(t, 10, defaultPage.Limit)
	assert.Zero(t, defaultPage.Total)
	assert.Empty(t, defaultPage.Items)

	explicitRequest := httptest.NewRequest(
		"GET",
		"/v1/management/connections/unassigned?page=2&limit=25&sortOrder=ASC&type=postgresql&metadata.region=br",
		nil,
	)
	explicitResponse, err := app.Test(explicitRequest)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, explicitResponse.Body.Close()) })
	require.Equal(t, fiber.StatusOK, explicitResponse.StatusCode)

	var explicitPage connectionPage
	require.NoError(t, json.NewDecoder(explicitResponse.Body).Decode(&explicitPage))
	assert.Equal(t, 2, explicitPage.Page)
	assert.Equal(t, 25, explicitPage.Limit)
	assert.Equal(t, 1, explicitPage.Total)
	require.Len(t, explicitPage.Items, 1)
	assert.Equal(t, connectionID, explicitPage.Items[0].ID)
}

func TestHumaMigration_ListBinderMapsValidationAndServiceErrors(t *testing.T) {
	t.Run("invalid query is rejected before the repository", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockConnRepo := connRepo.NewMockRepository(ctrl)
		app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{
			ListUnassignedQry: query.NewListUnassignedConnections(mockConnRepo),
		})

		request := httptest.NewRequest("GET", "/v1/management/connections/unassigned?limit=not-a-number", nil)
		response, err := app.Test(request)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
		require.Equal(t, fiber.StatusBadRequest, response.StatusCode)
		assert.Contains(t, response.Header.Get("Content-Type"), "application/problem+json")

		var detail problem.Detail
		require.NoError(t, json.NewDecoder(response.Body).Decode(&detail))
		assert.Equal(t, "Bad Request", detail.Title)
		assert.Equal(t, fiber.StatusBadRequest, detail.Status)
		assert.Contains(t, detail.Detail, "query parameters")
		assert.Equal(t, constant.ErrInvalidQueryParameter.Error(), detail.Code)
		assert.Equal(t, problem.BaseURI+"/"+constant.ErrInvalidQueryParameter.Error(), detail.Type)
		assert.Empty(t, detail.Errors)
	})

	t.Run("repository failure is mapped and scrubbed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockConnRepo := connRepo.NewMockRepository(ctrl)
		secret := errors.New("sentinel-internal-secret")
		mockConnRepo.EXPECT().ListUnassigned(gomock.Any(), gomock.Any()).Return(nil, int64(0), secret)
		app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{
			ListUnassignedQry: query.NewListUnassignedConnections(mockConnRepo),
		})

		request := httptest.NewRequest("GET", "/v1/management/connections/unassigned", nil)
		response, err := app.Test(request)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
		require.Equal(t, fiber.StatusInternalServerError, response.StatusCode)
		assert.Contains(t, response.Header.Get("Content-Type"), "application/problem+json")

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		assert.NotContains(t, string(body), secret.Error())
		var detail problem.Detail
		require.NoError(t, json.Unmarshal(body, &detail))
		assert.Equal(t, "Internal Server Error", detail.Title)
		assert.Equal(t, fiber.StatusInternalServerError, detail.Status)
		assert.Equal(t, "internal error", detail.Detail)
		assert.Equal(t, constant.ErrInternalServer.Error(), detail.Code)
		assert.Equal(t, problem.BaseURI+"/"+constant.ErrInternalServer.Error(), detail.Type)
		assert.Empty(t, detail.Errors)
	})
}

func TestHumaMigration_AssignBinderNormalizesHeaderAndSerializesResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	connectionID := uuid.New()
	connection := createTestConnectionForMigration(connectionID)
	connection.ProductName = "reporter"

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)
	mockConnRepo.EXPECT().AssignProductName(gomock.Any(), connectionID, "reporter").Return(connection, nil)

	app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{
		AssignCmd: command.NewAssignConnection(mockConnRepo),
	})
	request := httptest.NewRequest(
		"POST",
		"/v1/management/connections/"+connectionID.String()+"/assign",
		nil,
	)
	request.Header.Set("X-Product-Name", "  RePorTer  ")

	response, err := app.Test(request)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	require.Equal(t, fiber.StatusOK, response.StatusCode)

	var assigned model.ConnectionResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&assigned))
	assert.Equal(t, connectionID, assigned.ID)
	assert.Equal(t, "reporter", assigned.ProductName)
}

func TestHumaMigration_AssignBinderRejectsInvalidIDBeforeHeaderAndService(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{
		AssignCmd: command.NewAssignConnection(mockConnRepo),
	})

	request := httptest.NewRequest("POST", "/v1/management/connections/not-a-uuid/assign", nil)
	response, err := app.Test(request)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	require.Equal(t, fiber.StatusBadRequest, response.StatusCode)

	var detail problem.Detail
	require.NoError(t, json.NewDecoder(response.Body).Decode(&detail))
	assert.Equal(t, "Bad Request", detail.Title)
	assert.Equal(t, fiber.StatusBadRequest, detail.Status)
	assert.Equal(t, constant.ErrInvalidPathParameter.Error(), detail.Code)
	assert.Equal(t, "invalid connection id", detail.Detail)
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrInvalidPathParameter.Error(), detail.Type)
	assert.Empty(t, detail.Errors)
}

func TestHumaMigration_AssignBinderMapsDomainProblems(t *testing.T) {
	t.Run("missing product header", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockConnRepo := connRepo.NewMockRepository(ctrl)
		app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{
			AssignCmd: command.NewAssignConnection(mockConnRepo),
		})

		request := httptest.NewRequest(
			"POST",
			"/v1/management/connections/"+uuid.NewString()+"/assign",
			nil,
		)
		response, err := app.Test(request)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, response.Body.Close()) })

		assertHumaMigrationProblem(
			t,
			response,
			fiber.StatusBadRequest,
			"Bad Request",
			"X-Product-Name header is required and must not be empty",
			constant.ErrInvalidHeaderParameter.Error(),
		)
	})

	t.Run("connection not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockConnRepo := connRepo.NewMockRepository(ctrl)
		connectionID := uuid.New()
		mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(nil, nil)
		app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{
			AssignCmd: command.NewAssignConnection(mockConnRepo),
		})

		request := httptest.NewRequest(
			"POST",
			"/v1/management/connections/"+connectionID.String()+"/assign",
			nil,
		)
		request.Header.Set("X-Product-Name", "reporter")
		response, err := app.Test(request)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, response.Body.Close()) })

		assertHumaMigrationProblem(
			t,
			response,
			fiber.StatusNotFound,
			"Not Found",
			"It was not possible to find the connection entity during the requested flow. Please review the data provided in the request.",
			constant.ErrEntityNotFound.Error(),
		)
	})

	t.Run("connection already assigned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockConnRepo := connRepo.NewMockRepository(ctrl)
		connectionID := uuid.New()
		connection := createTestConnectionForMigration(connectionID)
		mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)
		mockConnRepo.EXPECT().AssignProductName(gomock.Any(), connectionID, "reporter").Return(nil, nil)
		app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{
			AssignCmd: command.NewAssignConnection(mockConnRepo),
		})

		request := httptest.NewRequest(
			"POST",
			"/v1/management/connections/"+connectionID.String()+"/assign",
			nil,
		)
		request.Header.Set("X-Product-Name", "reporter")
		response, err := app.Test(request)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, response.Body.Close()) })

		assertHumaMigrationProblem(
			t,
			response,
			fiber.StatusConflict,
			"Conflict",
			"The connection is already assigned to a product and cannot be reassigned.",
			constant.ErrConnectionAlreadyAssigned.Error(),
		)
	})

	t.Run("repository failure is scrubbed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockConnRepo := connRepo.NewMockRepository(ctrl)
		connectionID := uuid.New()
		connection := createTestConnectionForMigration(connectionID)
		secret := errors.New("sentinel-assign-secret")
		mockConnRepo.EXPECT().FindByID(gomock.Any(), connectionID).Return(connection, nil)
		mockConnRepo.EXPECT().AssignProductName(gomock.Any(), connectionID, "reporter").Return(nil, secret)
		app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{
			AssignCmd: command.NewAssignConnection(mockConnRepo),
		})

		request := httptest.NewRequest(
			"POST",
			"/v1/management/connections/"+connectionID.String()+"/assign",
			nil,
		)
		request.Header.Set("X-Product-Name", "reporter")
		response, err := app.Test(request)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, response.Body.Close()) })

		assertHumaMigrationInternalProblem(t, response, secret.Error())
	})
}

func TestHumaMigration_NilListDependencyReturnsScrubbedProblem(t *testing.T) {
	app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{ListUnassignedQry: nil})

	request := httptest.NewRequest("GET", "/v1/management/connections/unassigned", nil)
	response, err := app.Test(request)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	assertHumaMigrationInternalProblem(t, response)
}

func TestHumaMigration_NilAssignDependencyReturnsScrubbedProblem(t *testing.T) {
	app := setupHumaMigrationBinderTestApp(t, &MigrationHandler{AssignCmd: nil})
	request := httptest.NewRequest("POST", "/v1/management/connections/"+uuid.NewString()+"/assign", nil)
	request.Header.Set("X-Product-Name", "reporter")

	response, err := app.Test(request)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	assertHumaMigrationInternalProblem(t, response)
}

func TestHumaMigration_NilListExecutorResultReturnsScrubbedProblem(t *testing.T) {
	app := setupMigrationTestApp()
	api := BuildHumaAPI(app, false)
	var handlers OperationHandlers
	bindMigrationExecutors(
		&handlers,
		func(context.Context, httpUtils.QueryHeader) (*model.Pagination, error) {
			return nil, nil
		},
		nil,
	)
	registerMigrationOperations(app, api, handlers, func(_, _ string) []fiber.Handler { return nil })

	request := httptest.NewRequest("GET", "/v1/management/connections/unassigned", nil)
	response, err := app.Test(request)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	assertHumaMigrationInternalProblem(t, response)
}

func TestHumaMigration_NilAssignExecutorResultReturnsScrubbedProblem(t *testing.T) {
	app := setupMigrationTestApp()
	api := BuildHumaAPI(app, false)
	var handlers OperationHandlers
	bindMigrationExecutors(
		&handlers,
		nil,
		func(context.Context, uuid.UUID, string) (*model.Connection, error) {
			return nil, nil
		},
	)
	registerMigrationOperations(app, api, handlers, func(_, _ string) []fiber.Handler { return nil })

	request := httptest.NewRequest("POST", "/v1/management/connections/"+uuid.NewString()+"/assign", nil)
	request.Header.Set("X-Product-Name", "reporter")
	response, err := app.Test(request)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	assertHumaMigrationInternalProblem(t, response)
}

func assertHumaMigrationInternalProblem(t *testing.T, response *stdhttp.Response, secrets ...string) {
	t.Helper()

	require.Equal(t, fiber.StatusInternalServerError, response.StatusCode)
	assert.Contains(t, response.Header.Get("Content-Type"), "application/problem+json")
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	for _, secret := range secrets {
		assert.NotContains(t, string(body), secret)
	}

	var detail problem.Detail
	require.NoError(t, json.Unmarshal(body, &detail))
	assert.Equal(t, "Internal Server Error", detail.Title)
	assert.Equal(t, fiber.StatusInternalServerError, detail.Status)
	assert.Equal(t, "internal error", detail.Detail)
	assert.Equal(t, constant.ErrInternalServer.Error(), detail.Code)
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrInternalServer.Error(), detail.Type)
	assert.Empty(t, detail.Errors)
}

func assertHumaMigrationProblem(
	t *testing.T,
	response *stdhttp.Response,
	status int,
	title string,
	detailMessage string,
	code string,
) {
	t.Helper()

	require.Equal(t, status, response.StatusCode)
	assert.Contains(t, response.Header.Get("Content-Type"), "application/problem+json")

	var detail problem.Detail
	require.NoError(t, json.NewDecoder(response.Body).Decode(&detail))
	assert.Equal(t, title, detail.Title)
	assert.Equal(t, status, detail.Status)
	assert.Equal(t, detailMessage, detail.Detail)
	assert.Equal(t, code, detail.Code)
	assert.Equal(t, problem.BaseURI+"/"+code, detail.Type)
	assert.Empty(t, detail.Errors)
}
