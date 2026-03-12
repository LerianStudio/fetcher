package in

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cacheRepo "github.com/LerianStudio/fetcher/components/manager/internal/adapters/cache"
	commandSvc "github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	querySvc "github.com/LerianStudio/fetcher/components/manager/internal/services/query"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	datasourceModel "github.com/LerianStudio/fetcher/pkg/model/datasource"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	productRepo "github.com/LerianStudio/fetcher/pkg/mongodb/product"
	nethttp "github.com/LerianStudio/fetcher/pkg/net/http"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createTestProduct(id, orgID uuid.UUID) *model.Product {
	now := time.Now().UTC()
	return &model.Product{
		ID:             id,
		OrganizationID: orgID,
		Code:           "reporter",
		Name:           "Reporter",
		Description:    "Reporting product",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestConnectionHandler_CreateConnection_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	orgID := uuid.New()
	productID := uuid.New()
	product := createTestProduct(productID, orgID)

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	mockProductRepo.EXPECT().FindByID(gomock.Any(), productID, orgID).Return(product, nil)
	mockCryptor.EXPECT().Encrypt(gomock.Any(), "secretpassword").Return("encrypted-password", "v1", nil)
	mockConnRepo.EXPECT().FindByOrganizationAndName(gomock.Any(), orgID, "test-connection").Return(nil, nil)
	mockConnRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, conn *model.Connection) (*model.Connection, error) {
		assert.Equal(t, orgID, conn.OrganizationID)
		require.NotNil(t, conn.ProductID)
		assert.Equal(t, productID, *conn.ProductID)
		assert.Equal(t, "test-connection", conn.ConfigName)
		assert.Equal(t, model.TypePostgreSQL, conn.Type)
		return conn, nil
	})

	handler := &ConnectionHandler{
		CreateCmd: commandSvc.NewCreateConnection(mockConnRepo, mockProductRepo, mockCryptor),
	}
	app.Post("/v1/management/connections", handler.CreateConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(`{
		"productId":"`+productID.String()+`",
		"configName":"test-connection",
		"type":"POSTGRESQL",
		"host":"localhost",
		"port":5432,
		"databaseName":"testdb",
		"userName":"testuser",
		"password":"secretpassword"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode)

	var body model.ConnectionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "test-connection", body.ConfigName)
	require.NotNil(t, body.ProductID)
	assert.Equal(t, productID, *body.ProductID)
}

func TestConnectionHandler_ListConnections_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	orgID := uuid.New()
	conn1 := createTestConnection(uuid.New(), orgID)
	conn2 := createTestConnection(uuid.New(), orgID)
	conn2.ConfigName = "secondary"

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockConnRepo.EXPECT().List(gomock.Any(), orgID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, filters nethttp.QueryHeader) ([]*model.Connection, int64, error) {
		assert.Equal(t, 10, filters.Limit)
		assert.Equal(t, 1, filters.Page)
		return []*model.Connection{conn1, conn2}, 2, nil
	})

	handler := &ConnectionHandler{
		ListQuery: querySvc.NewListConnections(mockConnRepo, nil),
	}
	app.Get("/v1/management/connections", handler.ListConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections?limit=10&page=1", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	items, ok := body["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 2)
	assert.EqualValues(t, 2, body["total"])
}

func TestConnectionHandler_UpdateConnection_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	orgID := uuid.New()
	connID := uuid.New()
	current := createTestConnection(connID, orgID)

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID, orgID).Return(current, nil)
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), orgID, current.ConfigName).Return(false, nil)
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
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	var body model.ConnectionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "new-host", body.Host)
	assert.Equal(t, 5433, body.Port)
}

func TestConnectionHandler_ValidateSchema_RealHandlerFailureResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	orgID := uuid.New()
	connID := uuid.New()
	conn := createTestConnection(connID, orgID)
	conn.ConfigName = "db1"

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockSchemaCache := cacheRepo.NewMockSchemaCacheRepository(ctrl)

	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), orgID, []string{"db1"}).Return([]*model.Connection{conn}, nil)
	mockSchemaCache.EXPECT().Get(gomock.Any(), "db1").Return(model.NewDataSourceSchema("db1"), nil)

	handler := &ConnectionHandler{
		ValidateSchemaQuery: querySvc.NewValidateSchema(mockConnRepo, nil, mockSchemaCache),
	}
	app.Post("/v1/management/connections/validate-schema", handler.ValidateSchema)

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(`{
		"mappedFields": {"db1": {"users": ["id"]}}
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 422, resp.StatusCode)
	var body model.SchemaValidationErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, constant.ErrSchemaValidationFailed.Error(), body.Code)
	require.Len(t, body.Errors, 1)
	assert.Equal(t, model.ErrTypeTableNotFound, body.Errors[0].Type)
}

func TestConnectionHandler_GetConnectionSchema_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	orgID := uuid.New()
	connID := uuid.New()
	conn := createTestConnection(connID, orgID)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockDataSource := datasourceModel.NewMockDataSource(ctrl)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID, orgID).Return(conn, nil)
	schema := model.NewDataSourceSchema(conn.ConfigName)
	schema.AddTable("users", []string{"id", "name"})
	mockDataSource.EXPECT().GetSchemaInfo(gomock.Any(), gomock.Any()).Return(schema, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)

	handler := &ConnectionHandler{
		GetSchemaQuery: querySvc.NewGetConnectionSchema(mockConnRepo, nil, func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasourceModel.DataSource, error) {
			return mockDataSource, nil
		}),
	}
	app.Get("/v1/management/connections/:id/schema", handler.GetConnectionSchema)

	req := httptest.NewRequest("GET", "/v1/management/connections/"+connID.String()+"/schema", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	var body model.ConnectionSchemaResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body.Tables, 1)
	assert.Equal(t, "users", body.Tables[0].Name)
}

func TestProductHandler_CreateProduct_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	orgID := uuid.New()
	mockProductRepo := productRepo.NewMockRepository(ctrl)

	mockProductRepo.EXPECT().FindByCode(gomock.Any(), "reporter", orgID).Return(nil, nil)
	mockProductRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, product *model.Product) (*model.Product, error) {
		assert.Equal(t, orgID, product.OrganizationID)
		assert.Equal(t, "reporter", product.Code)
		return product, nil
	})

	handler := &ProductHandler{CreateCmd: commandSvc.NewCreateProduct(mockProductRepo)}
	app.Post("/v1/management/products", handler.CreateProduct)

	req := httptest.NewRequest("POST", "/v1/management/products", strings.NewReader(`{"code":"reporter","name":"Reporter"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode)
	var body model.ProductResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "reporter", body.Code)
	assert.Equal(t, "Reporter", body.Name)
}

func TestProductHandler_ListProducts_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	orgID := uuid.New()
	product := createTestProduct(uuid.New(), orgID)
	mockProductRepo := productRepo.NewMockRepository(ctrl)

	mockProductRepo.EXPECT().List(gomock.Any(), orgID, gomock.Any()).Return([]*model.Product{product}, int64(1), nil)

	handler := &ProductHandler{ListQuery: querySvc.NewListProducts(mockProductRepo)}
	app.Get("/v1/management/products", handler.ListProducts)

	req := httptest.NewRequest("GET", "/v1/management/products?limit=10&page=1", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	items, ok := body["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
}

func TestMigrationHandler_ListUnassignedConnections_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	orgID := uuid.New()
	conn := createTestConnection(uuid.New(), orgID)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	mockConnRepo.EXPECT().ListUnassigned(gomock.Any(), orgID, gomock.Any()).Return([]*model.Connection{conn}, int64(1), nil)

	handler := &MigrationHandler{ListUnassignedQry: querySvc.NewListUnassignedConnections(mockConnRepo)}
	app.Get("/v1/management/connections/unassigned", handler.ListUnassignedConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections/unassigned?limit=10&page=1", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	items, ok := body["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
}

func TestMigrationHandler_AssignConnectionToProduct_RealHandlerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := setupConnectionTestApp()
	orgID := uuid.New()
	connID := uuid.New()
	productID := uuid.New()
	product := createTestProduct(productID, orgID)
	conn := createTestConnection(connID, orgID)
	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockProductRepo := productRepo.NewMockRepository(ctrl)

	mockProductRepo.EXPECT().FindByID(gomock.Any(), productID, orgID).Return(product, nil)
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID, orgID).Return(conn, nil)
	mockConnRepo.EXPECT().AssignProduct(gomock.Any(), connID, orgID, productID).DoAndReturn(func(_ context.Context, _ uuid.UUID, _ uuid.UUID, pid uuid.UUID) (*model.Connection, error) {
		conn.ProductID = &pid
		return conn, nil
	})

	handler := &MigrationHandler{AssignCmd: commandSvc.NewAssignConnection(mockConnRepo, mockProductRepo)}
	app.Post("/v1/management/connections/:id/assign", handler.AssignConnectionToProduct)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connID.String()+"/assign", strings.NewReader(`{"productId":"`+productID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	var body model.ConnectionResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.NotNil(t, body.ProductID)
	assert.Equal(t, productID, *body.ProductID)
}
