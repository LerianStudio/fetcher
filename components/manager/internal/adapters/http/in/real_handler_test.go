package in

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
	commandSvc "github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	querySvc "github.com/LerianStudio/fetcher/components/manager/internal/services/query"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	datasourceModel "github.com/LerianStudio/fetcher/pkg/model/datasource"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	nethttp "github.com/LerianStudio/fetcher/pkg/net/http"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestConnectionHandler_CreateConnection_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	mockCryptor.EXPECT().Encrypt(gomock.Any(), "secretpassword").Return("encrypted-password", "v1", nil)
	mockConnRepo.EXPECT().FindByName(gomock.Any(), "test-connection").Return(nil, nil)
	mockConnRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, conn *model.Connection) (*model.Connection, error) {
		assert.Equal(t, "test-connection", conn.ConfigName)
		assert.Equal(t, model.TypePostgreSQL, conn.Type)
		return conn, nil
	})

	handler := &ConnectionHandler{
		CreateCmd: commandSvc.NewCreateConnection(mockConnRepo, mockCryptor),
	}
	app.Post("/v1/management/connections", handler.CreateConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(`{
		"configName":"test-connection",
		"type":"POSTGRESQL",
		"host":"localhost",
		"port":5432,
		"databaseName":"testdb",
		"userName":"testuser",
		"password":"secretpassword"
	}`))
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("X-Product-Name", "test-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode)

	var body model.ConnectionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "test-connection", body.ConfigName)
}

func TestConnectionHandler_ListConnections_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	conn1 := createTestConnection(uuid.New())
	conn2 := createTestConnection(uuid.New())
	conn2.ConfigName = "secondary"

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockConnRepo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, filters nethttp.QueryHeader) ([]*model.Connection, int64, error) {
		assert.Equal(t, 10, filters.Limit)
		assert.Equal(t, 1, filters.Page)
		return []*model.Connection{conn1, conn2}, 2, nil
	})

	handler := &ConnectionHandler{
		ListQuery: querySvc.NewListConnections(mockConnRepo, nil),
	}
	app.Get("/v1/management/connections", handler.ListConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections?limit=10&page=1", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var respBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	items, ok := respBody["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 2)
	assert.EqualValues(t, 2, respBody["total"])
}

func TestConnectionHandler_UpdateConnection_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	connID := uuid.New()
	current := createTestConnection(connID)

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(current, nil)
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), current.ConfigName).Return(false, nil)
	mockConnRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, conn *model.Connection) (*model.Connection, error) {
		assert.Equal(t, "new-host", conn.Host)
		assert.Equal(t, 5433, conn.Port)
		return conn, nil
	})

	handler := &ConnectionHandler{
		UpdateCmd: commandSvc.NewUpdateConnection(mockConnRepo, mockJobRepo, nil),
	}
	app.Patch("/v1/management/connections/:id", handler.UpdateConnection)

	req := httptest.NewRequest("PATCH", "/v1/management/connections/"+connID.String(), strings.NewReader(`{
		"host":"new-host",
		"port":5433
	}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	var respBody model.ConnectionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	assert.Equal(t, "new-host", respBody.Host)
	assert.Equal(t, 5433, respBody.Port)
}

func TestConnectionHandler_ValidateSchema_RealHandlerFailureResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	connID := uuid.New()
	conn := createTestConnection(connID)
	conn.ConfigName = "db1"

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), []string{"db1"}).Return([]*model.Connection{conn}, nil)
	mockSchemaCache.EXPECT().Get(gomock.Any(), "db1").Return(model.NewDataSourceSchema("db1"), nil)

	handler := &ConnectionHandler{
		ValidateSchemaQuery: querySvc.NewValidateSchema(mockConnRepo, nil, mockSchemaCache, nil, nil),
	}
	app.Post("/v1/management/connections/validate-schema", handler.ValidateSchema)

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(`{
		"mappedFields": {"db1": {"users": ["id"]}}
	}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 422, resp.StatusCode)
	var respBody model.SchemaValidationErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	assert.Equal(t, constant.ErrSchemaValidationFailed.Error(), respBody.Code)
	require.Len(t, respBody.Errors, 1)
	assert.Equal(t, model.ErrTypeTableNotFound, respBody.Errors[0].Type)
}

func TestConnectionHandler_GetConnectionSchema_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	connID := uuid.New()
	conn := createTestConnection(connID)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockDataSource := datasourceModel.NewMockDataSource(ctrl)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(conn, nil)
	schema := model.NewDataSourceSchema(conn.ConfigName)
	schema.AddTable("users", []string{"id", "name"})
	mockDataSource.EXPECT().GetSchemaInfo(gomock.Any(), gomock.Any()).Return(schema, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)

	handler := &ConnectionHandler{
		GetSchemaQuery: querySvc.NewGetConnectionSchema(mockConnRepo, nil, func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasourceModel.DataSource, error) {
			return mockDataSource, nil
		}, nil, nil),
	}
	app.Get("/v1/management/connections/:id/schema", handler.GetConnectionSchema)

	req := httptest.NewRequest("GET", "/v1/management/connections/"+connID.String()+"/schema", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	var respBody model.ConnectionSchemaResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	assert.Len(t, respBody.Tables, 1)
	assert.Equal(t, "users", respBody.Tables[0].Name)
}

func TestMigrationHandler_ListUnassignedConnections_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	conn := createTestConnection(uuid.New())
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockConnRepo.EXPECT().ListUnassigned(gomock.Any(), gomock.Any()).Return([]*model.Connection{conn}, int64(1), nil)

	handler := &MigrationHandler{ListUnassignedQry: querySvc.NewListUnassignedConnections(mockConnRepo)}
	app.Get("/v1/management/connections/unassigned", handler.ListUnassignedConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections/unassigned?limit=10&page=1", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	var respBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	items, ok := respBody["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
}

func TestMigrationHandler_AssignConnectionToProduct_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	connID := uuid.New()
	productName := "my-product"
	conn := createTestConnection(connID)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(conn, nil)
	mockConnRepo.EXPECT().AssignProductName(gomock.Any(), connID, productName).DoAndReturn(func(_ context.Context, _ uuid.UUID, name string) (*model.Connection, error) {
		conn.ProductName = name
		return conn, nil
	})

	handler := &MigrationHandler{AssignCmd: commandSvc.NewAssignConnection(mockConnRepo)}
	app.Post("/v1/management/connections/:id/assign", handler.AssignConnectionToProduct)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connID.String()+"/assign", nil)
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("X-Product-Name", productName)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	var respBody model.ConnectionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&respBody))
	assert.Equal(t, productName, respBody.ProductName)
}
