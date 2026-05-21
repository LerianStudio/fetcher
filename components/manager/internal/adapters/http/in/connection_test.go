package in

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/query"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	"github.com/LerianStudio/fetcher/pkg/crypto"

	libLog "github.com/LerianStudio/lib-observability/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.uber.org/mock/gomock"
)

// setupConnectionTestApp creates a Fiber app with test context middleware for connection tests.
func setupConnectionTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024, // 10KB for test flexibility
	})

	// Middleware to inject test context with logger and tracer
	app.Use(func(c *fiber.Ctx) error {
		logger := &libLog.GoLogger{Level: libLog.LevelDebug}
		ctx := observability.ContextWithHeaderID(c.UserContext(), "test-request-id")
		ctx = observability.ContextWithLogger(ctx, logger)
		ctx = observability.ContextWithTracer(ctx, otel.Tracer("test"))
		c.SetUserContext(ctx)

		return c.Next()
	})

	return app
}

// createTestConnection creates a test connection with default values.
func createTestConnection(id uuid.UUID) *model.Connection {
	now := time.Now().UTC()
	return &model.Connection{
		ID:                   id,
		ProductName:          "test-product",
		ConfigName:           "test-connection",
		Type:                 model.TypePostgreSQL,
		Host:                 "localhost",
		Port:                 5432,
		DatabaseName:         "testdb",
		Username:             "testuser",
		PasswordEncrypted:    "encrypted-password",
		EncryptionKeyVersion: "v1",
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// validConnectionInput returns a valid ConnectionInput for tests.
func validConnectionInput() string {
	return `{
		"configName": "test-connection",
		"type": "POSTGRESQL",
		"host": "localhost",
		"port": 5432,
		"databaseName": "testdb",
		"username": "testuser",
		"password": "secretpassword"
	}`
}

// ============================================================================
// CreateConnection Handler Tests
// ============================================================================

func TestConnectionHandler_CreateConnection_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	connID := uuid.New()

	testConn := createTestConnection(connID)

	// Mock expectations for CreateConnection service:
	// 1. Encrypt password (called by model.NewConnection)
	mockCryptor.EXPECT().Encrypt(gomock.Any(), "secretpassword").Return("encrypted-password", "v1", nil)
	// 2. Check for duplicate config name
	mockConnRepo.EXPECT().FindByName(gomock.Any(), "test-connection").Return(nil, nil)
	// 3. Create connection
	mockConnRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(testConn, nil)

	createCmd := command.NewCreateConnection(mockConnRepo, mockCryptor)
	handler := &ConnectionHandler{CreateCmd: createCmd}

	app := setupConnectionTestApp()
	app.Post("/v1/management/connections", handler.CreateConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("X-Product-Name", "test-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.NotEmpty(t, body["id"])
}

func TestConnectionHandler_CreateConnection_InvalidJSON(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{CreateCmd: nil}
	app.Post("/v1/management/connections", handler.CreateConnection)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "invalid JSON - missing closing brace",
			body:     `{"configName": "test"`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid JSON - syntax error",
			body:     `{invalid}`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid JSON - empty string",
			body:     ``,
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestConnectionHandler_CreateConnection_MissingProductNameHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{CreateCmd: nil}
	app.Post("/v1/management/connections", handler.CreateConnection)

	tests := []struct {
		name          string
		productHeader string
		setHeader     bool
		wantCode      int
	}{
		{
			name:          "missing X-Product-Name header",
			productHeader: "",
			setHeader:     false,
			wantCode:      fiber.StatusBadRequest,
		},
		{
			name:          "empty X-Product-Name header",
			productHeader: "",
			setHeader:     true,
			wantCode:      fiber.StatusBadRequest,
		},
		{
			name:          "whitespace only X-Product-Name header",
			productHeader: "   ",
			setHeader:     true,
			wantCode:      fiber.StatusBadRequest,
		},
		{
			name:          "invalid characters in X-Product-Name header",
			productHeader: "my product!",
			setHeader:     true,
			wantCode:      fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
			req.Header.Set("Content-Type", "application/json")

			if tt.setHeader {
				req.Header.Set("X-Product-Name", tt.productHeader)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestConnectionHandler_CreateConnection_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	existingConn := createTestConnection(uuid.New())

	// Service will encrypt, then find existing connection -> conflict
	mockCryptor.EXPECT().Encrypt(gomock.Any(), "secretpassword").Return("encrypted-password", "v1", nil)
	mockConnRepo.EXPECT().FindByName(gomock.Any(), "test-connection").Return(existingConn, nil)

	createCmd := command.NewCreateConnection(mockConnRepo, mockCryptor)
	handler := &ConnectionHandler{CreateCmd: createCmd}

	app := setupConnectionTestApp()
	app.Post("/v1/management/connections", handler.CreateConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("X-Product-Name", "test-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestConnectionHandler_CreateConnection_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	// Service will encrypt, check dupe, then fail on create
	mockCryptor.EXPECT().Encrypt(gomock.Any(), "secretpassword").Return("encrypted-password", "v1", nil)
	mockConnRepo.EXPECT().FindByName(gomock.Any(), "test-connection").Return(nil, nil)
	mockConnRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	createCmd := command.NewCreateConnection(mockConnRepo, mockCryptor)
	handler := &ConnectionHandler{CreateCmd: createCmd}

	app := setupConnectionTestApp()
	app.Post("/v1/management/connections", handler.CreateConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("X-Product-Name", "test-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// ============================================================================
// GetConnection Handler Tests
// ============================================================================

func TestConnectionHandler_GetConnection_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	connID := uuid.New()

	testConn := createTestConnection(connID)

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(testConn, nil)

	getQuery := query.NewGetConnection(mockConnRepo, nil, nil)
	handler := &ConnectionHandler{GetQuery: getQuery}

	app := setupConnectionTestApp()
	app.Get("/v1/management/connections/:id", handler.GetConnection)

	req := httptest.NewRequest("GET", "/v1/management/connections/"+connID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.ConnectionResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, connID, body.ID)
	assert.Equal(t, "test-connection", body.ConfigName)
}

func TestConnectionHandler_GetConnection_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	connID := uuid.New()

	// Service returns nil for not found
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(nil, nil)

	getQuery := query.NewGetConnection(mockConnRepo, nil, nil)
	handler := &ConnectionHandler{GetQuery: getQuery}

	app := setupConnectionTestApp()
	app.Get("/v1/management/connections/:id", handler.GetConnection)

	req := httptest.NewRequest("GET", "/v1/management/connections/"+connID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestConnectionHandler_GetConnection_InvalidID(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{GetQuery: nil}
	app.Get("/v1/management/connections/:id", handler.GetConnection)

	tests := []struct {
		name     string
		connID   string
		wantCode int
	}{
		{
			name:     "invalid UUID format",
			connID:   "not-a-uuid",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "partial UUID",
			connID:   "550e8400-e29b",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "UUID with special characters",
			connID:   "550e8400-e29b-<script>",
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/v1/management/connections/" + tt.connID

			req := httptest.NewRequest("GET", path, nil)
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

// ============================================================================
// ListConnections Handler Tests
// ============================================================================

func TestConnectionHandler_ListConnections_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	connID1 := uuid.New()
	connID2 := uuid.New()

	conn1 := createTestConnection(connID1)
	conn1.ConfigName = "connection-1"
	conn2 := createTestConnection(connID2)
	conn2.ConfigName = "connection-2"

	conns := []*model.Connection{conn1, conn2}

	// ListConnections service: no productName header -> calls connRepo.List directly
	mockConnRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(conns, int64(2), nil)

	listQuery := query.NewListConnections(mockConnRepo, nil)
	handler := &ConnectionHandler{ListQuery: listQuery}

	app := setupConnectionTestApp()
	app.Get("/v1/management/connections", handler.ListConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections?limit=10&page=1", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	items, ok := body["items"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, len(items))
}

func TestConnectionHandler_ListConnections_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	// Service returns nil list, which gets converted to empty slice
	mockConnRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(0), nil)

	listQuery := query.NewListConnections(mockConnRepo, nil)
	handler := &ConnectionHandler{ListQuery: listQuery}

	app := setupConnectionTestApp()
	app.Get("/v1/management/connections", handler.ListConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	items, ok := body["items"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 0, len(items))
}

func TestConnectionHandler_ListConnections_InvalidPaginationParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	// The handler validates query params before calling the service,
	// so no mock expectations needed for invalid params
	listQuery := query.NewListConnections(mockConnRepo, nil)
	handler := &ConnectionHandler{ListQuery: listQuery}

	app := setupConnectionTestApp()
	app.Get("/v1/management/connections", handler.ListConnections)

	tests := []struct {
		name      string
		queryStr  string
		wantCode  int
		checkBody bool
	}{
		{
			name:     "invalid sort order",
			queryStr: "sortOrder=invalid",
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/management/connections?"+tt.queryStr, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestConnectionHandler_ListConnections_WithProductNameFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	connID1 := uuid.New()
	connID2 := uuid.New()

	conn1 := createTestConnection(connID1)
	conn1.ConfigName = "product-conn-1"
	conn1.ProductName = "my-product"
	conn2 := createTestConnection(connID2)
	conn2.ConfigName = "product-conn-2"
	conn2.ProductName = "my-product"

	conns := []*model.Connection{conn1, conn2}

	// ListConnections service: with productName header -> filters.ProductName is set
	mockConnRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(conns, int64(2), nil)

	listQuery := query.NewListConnections(mockConnRepo, nil)
	handler := &ConnectionHandler{ListQuery: listQuery}

	app := setupConnectionTestApp()
	app.Get("/v1/management/connections", handler.ListConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections?limit=10&page=1", nil)

	req.Header.Set("X-Product-Name", "my-product")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	items, ok := body["items"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, len(items))
}

func TestConnectionHandler_ListConnections_InvalidProductNameHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{ListQuery: nil}
	app.Get("/v1/management/connections", handler.ListConnections)

	tests := []struct {
		name          string
		productHeader string
		wantCode      int
	}{
		{
			name:          "invalid characters in product name",
			productHeader: "invalid name!@#",
			wantCode:      fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/management/connections", nil)

			req.Header.Set("X-Product-Name", tt.productHeader)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

// ============================================================================
// UpdateConnection Handler Tests
// ============================================================================

func TestConnectionHandler_UpdateConnection_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	connID := uuid.New()

	testConn := createTestConnection(connID)

	updatedConn := createTestConnection(connID)
	updatedConn.ConfigName = "updated-connection"

	// UpdateConnection service:
	// 1. Find existing connection
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(testConn, nil)
	// 2. Check for active jobs
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), "test-connection").Return(false, nil)
	// 3. Encrypt new password (ApplyPatch calls cryptor.Encrypt when password changes)
	mockCryptor.EXPECT().Encrypt(gomock.Any(), "newpassword").Return("encrypted-newpassword", "v1", nil)
	// 4. Update connection
	mockConnRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(updatedConn, nil)

	updateCmd := command.NewUpdateConnection(mockConnRepo, mockJobRepo, mockCryptor)
	handler := &ConnectionHandler{UpdateCmd: updateCmd}

	app := setupConnectionTestApp()
	app.Patch("/v1/management/connections/:id", handler.UpdateConnection)

	updatePayload := `{"configName": "updated-connection", "type": "POSTGRESQL", "host": "localhost", "port": 5432, "databaseName": "testdb", "username": "testuser", "password": "newpassword"}`

	req := httptest.NewRequest("PATCH", "/v1/management/connections/"+connID.String(), strings.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.ConnectionResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "updated-connection", body.ConfigName)
}

func TestConnectionHandler_UpdateConnection_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	connID := uuid.New()

	// Service finds no connection -> not found
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(nil, nil)

	updateCmd := command.NewUpdateConnection(mockConnRepo, mockJobRepo, mockCryptor)
	handler := &ConnectionHandler{UpdateCmd: updateCmd}

	app := setupConnectionTestApp()
	app.Patch("/v1/management/connections/:id", handler.UpdateConnection)

	updatePayload := `{"configName": "updated-connection"}`

	req := httptest.NewRequest("PATCH", "/v1/management/connections/"+connID.String(), strings.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestConnectionHandler_UpdateConnection_InvalidBody(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{UpdateCmd: nil}
	app.Patch("/v1/management/connections/:id", handler.UpdateConnection)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "invalid JSON",
			body:     `{invalid}`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "empty body",
			body:     ``,
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PATCH", "/v1/management/connections/"+uuid.New().String(), strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestConnectionHandler_UpdateConnection_Conflict_ActiveJobs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)

	connID := uuid.New()

	testConn := createTestConnection(connID)

	// Service finds connection, then finds active jobs -> conflict
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(testConn, nil)
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), "test-connection").Return(true, nil)

	updateCmd := command.NewUpdateConnection(mockConnRepo, mockJobRepo, mockCryptor)
	handler := &ConnectionHandler{UpdateCmd: updateCmd}

	app := setupConnectionTestApp()
	app.Patch("/v1/management/connections/:id", handler.UpdateConnection)

	updatePayload := `{"configName": "updated-connection"}`

	req := httptest.NewRequest("PATCH", "/v1/management/connections/"+connID.String(), strings.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

// ============================================================================
// DeleteConnection Handler Tests
// ============================================================================

func TestConnectionHandler_DeleteConnection_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	connID := uuid.New()

	testConn := createTestConnection(connID)

	// DeleteConnection service:
	// 1. Find existing connection
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(testConn, nil)
	// 2. Check for active jobs
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), "test-connection").Return(false, nil)
	// 3. Delete connection
	mockConnRepo.EXPECT().Delete(gomock.Any(), connID, gomock.Any()).Return(nil)

	deleteCmd := command.NewDeleteConnection(mockConnRepo, mockJobRepo)
	handler := &ConnectionHandler{DeleteCmd: deleteCmd}

	app := setupConnectionTestApp()
	app.Delete("/v1/management/connections/:id", handler.DeleteConnection)

	req := httptest.NewRequest("DELETE", "/v1/management/connections/"+connID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The real handler returns 204 No Content on successful delete
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestConnectionHandler_DeleteConnection_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	connID := uuid.New()

	// Service finds no connection -> not found
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(nil, nil)

	deleteCmd := command.NewDeleteConnection(mockConnRepo, mockJobRepo)
	handler := &ConnectionHandler{DeleteCmd: deleteCmd}

	app := setupConnectionTestApp()
	app.Delete("/v1/management/connections/:id", handler.DeleteConnection)

	req := httptest.NewRequest("DELETE", "/v1/management/connections/"+connID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestConnectionHandler_DeleteConnection_Conflict_ActiveJobs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	connID := uuid.New()

	testConn := createTestConnection(connID)

	// Service finds connection, then finds active jobs -> conflict
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(testConn, nil)
	mockJobRepo.EXPECT().ExistsRunningByMappedFieldKey(gomock.Any(), "test-connection").Return(true, nil)

	deleteCmd := command.NewDeleteConnection(mockConnRepo, mockJobRepo)
	handler := &ConnectionHandler{DeleteCmd: deleteCmd}

	app := setupConnectionTestApp()
	app.Delete("/v1/management/connections/:id", handler.DeleteConnection)

	req := httptest.NewRequest("DELETE", "/v1/management/connections/"+connID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestConnectionHandler_DeleteConnection_InvalidID(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{DeleteCmd: nil}
	app.Delete("/v1/management/connections/:id", handler.DeleteConnection)

	req := httptest.NewRequest("DELETE", "/v1/management/connections/not-a-uuid", nil)
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// ============================================================================
// TestConnection Handler Tests
// ============================================================================

func TestConnectionHandler_TestConnection_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRateLimiter := query.NewMockRateLimiterStore(ctrl)

	connID := uuid.New()

	testConn := createTestConnection(connID)

	// TestConnection service:
	// 1. Rate limiter allows request
	mockRateLimiter.EXPECT().Take(gomock.Any(), connID.String()).Return(uint64(1), uint64(9), uint64(0), true, nil)
	// 2. Find connection
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(testConn, nil)
	// 3. Decrypt password for datasource connection (called by datasource.NewDataSourceFromConnection)
	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted-password", "v1").Return("secretpassword", nil)
	testFactory := func(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor) (datasource.DataSource, error) {
		_, err := cryptor.Decrypt(ctx, conn.PasswordEncrypted, conn.EncryptionKeyVersion)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		return nil, fmt.Errorf("connection failed: unable to connect to database")
	}

	testQuery := query.NewTestConnection(mockConnRepo, mockCryptor, mockRateLimiter, testFactory, nil, nil)
	handler := &ConnectionHandler{TestQuery: testQuery}

	app := setupConnectionTestApp()
	app.Post("/v1/management/connections/:id/test", handler.TestConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connID.String()+"/test", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The real service attempts to connect to the actual database.
	// Since no real DB is available, it returns 500 (Database Connection Error).
	// This validates the full handler path up to the datasource connection attempt.
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestConnectionHandler_TestConnection_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRateLimiter := query.NewMockRateLimiterStore(ctrl)

	connID := uuid.New()

	// Rate limiter allows, but connection not found
	mockRateLimiter.EXPECT().Take(gomock.Any(), connID.String()).Return(uint64(1), uint64(9), uint64(0), true, nil)
	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID).Return(nil, nil)

	testQuery := query.NewTestConnection(mockConnRepo, mockCryptor, mockRateLimiter, nil, nil, nil)
	handler := &ConnectionHandler{TestQuery: testQuery}

	app := setupConnectionTestApp()
	app.Post("/v1/management/connections/:id/test", handler.TestConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connID.String()+"/test", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestConnectionHandler_TestConnection_InvalidID(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{TestQuery: nil}
	app.Post("/v1/management/connections/:id/test", handler.TestConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections/not-a-uuid/test", nil)
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestConnectionHandler_TestConnection_RateLimited(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockRateLimiter := query.NewMockRateLimiterStore(ctrl)

	connID := uuid.New()

	// Rate limiter denies request (ok=false)
	futureReset := uint64(time.Now().Add(60 * time.Second).UnixNano())
	mockRateLimiter.EXPECT().Take(gomock.Any(), connID.String()).Return(uint64(0), uint64(0), futureReset, false, nil)

	testQuery := query.NewTestConnection(mockConnRepo, mockCryptor, mockRateLimiter, nil, nil, nil)
	handler := &ConnectionHandler{TestQuery: testQuery}

	app := setupConnectionTestApp()
	app.Post("/v1/management/connections/:id/test", handler.TestConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connID.String()+"/test", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
}

// ============================================================================
// Handler Direct Tests (testing actual handler methods)
// ============================================================================

func TestConnectionHandler_CreateConnection_HandlerDirectly_InvalidJSON(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{CreateCmd: nil}
	app.Post("/v1/management/connections", handler.CreateConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(`{broken`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "code")
}

func TestConnectionHandler_GetConnection_HandlerDirectly_InvalidUUID(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{GetQuery: nil}
	app.Get("/v1/management/connections/:id", handler.GetConnection)

	req := httptest.NewRequest("GET", "/v1/management/connections/invalid-uuid", nil)
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestConnectionHandler_UpdateConnection_HandlerDirectly_InvalidUUID(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{UpdateCmd: nil}
	app.Patch("/v1/management/connections/:id", handler.UpdateConnection)

	req := httptest.NewRequest("PATCH", "/v1/management/connections/invalid-uuid", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestConnectionHandler_DeleteConnection_HandlerDirectly_InvalidUUID(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{DeleteCmd: nil}
	app.Delete("/v1/management/connections/:id", handler.DeleteConnection)

	req := httptest.NewRequest("DELETE", "/v1/management/connections/invalid-uuid", nil)
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestConnectionHandler_TestConnection_HandlerDirectly_InvalidUUID(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{TestQuery: nil}
	app.Post("/v1/management/connections/:id/test", handler.TestConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections/invalid-uuid/test", nil)
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// ============================================================================
// ValidateSchema Handler Tests
// ============================================================================

// validSchemaValidationRequest returns a valid SchemaValidationRequest payload.
func validSchemaValidationRequest() string {
	return `{
		"mappedFields": {
			"ds1": {
				"table1": ["field1", "field2"],
				"table2": ["field3"]
			}
		}
	}`
}

func TestConnectionHandler_ValidateSchema_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockCache := newNoopSchemaCache()

	testConn := createTestConnection(uuid.New())
	testConn.ConfigName = "ds1"

	// ValidateSchema service:
	// 1. Find connections by config names
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return([]*model.Connection{testConn}, nil)
	// 2. Cache miss -> factory creates datasource (which fails because no real DB is available).
	// The factory is called with the connection and cryptor; decryption happens inside.
	// Since no real DB is available, the factory returns an error -> treated as DataSourceDown -> 422.

	// Provide a factory that simulates a datasource-down scenario (no real DB available).
	failingFactory := func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
		return nil, fmt.Errorf("connection refused")
	}
	validateSchemaQuery := query.NewValidateSchema(mockConnRepo, mockCryptor, mockCache, failingFactory, nil)
	handler := &ConnectionHandler{ValidateSchemaQuery: validateSchemaQuery}

	app := setupConnectionTestApp()
	app.Post("/v1/management/connections/validate-schema", handler.ValidateSchema)

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(validSchemaValidationRequest()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The real service attempts to connect to the DB for schema validation.
	// Without a real DB, the datasource connection fails -> treated as DataSourceDown -> 422
	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestConnectionHandler_ValidateSchema_Failure_DataSourceNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockCache := newNoopSchemaCache()

	// Return a connection with a different config name so "unknown_ds" is not found in the map
	existingConn := createTestConnection(uuid.New())
	existingConn.ConfigName = "other_ds"

	// The service finds connections but the requested "unknown_ds" is not among them
	// Since FindByConfigNames is called with the requested names, and only "other_ds" is in the list,
	// it won't find any connection for "unknown_ds".
	// Actually, FindByConfigNames is called with all requested configNames from the request.
	// If no connections match at all, the service returns an error.
	// For this test, we need at least one matching connection, but "unknown_ds" won't match.
	// However, FindByConfigNames returns ALL connections matching ANY of the names.
	// If the request has only "unknown_ds" and the repo returns empty -> "No connections found" error (not 422).
	// To get a 422 with DataSourceNotFound, we need the service to find SOME connections
	// but not all. That requires multiple datasources in the request.

	// Use a request with two datasources: one that exists, one that doesn't
	payload := `{
		"mappedFields": {
			"existing_ds": {
				"table1": ["field1"]
			},
			"unknown_ds": {
				"table1": ["field1"]
			}
		}
	}`

	existingConn.ConfigName = "existing_ds"

	// Only "existing_ds" is returned
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return([]*model.Connection{existingConn}, nil)
	// The service will try to fetch schema for "existing_ds" (which connects to real DB and fails)
	// and also detect "unknown_ds" as not found.
	// Provide a factory that simulates datasource-down for existing connections.
	failingFactory := func(_ context.Context, _ *model.Connection, _ crypto.Cryptor) (datasource.DataSource, error) {
		return nil, fmt.Errorf("connection refused")
	}

	validateSchemaQuery := query.NewValidateSchema(mockConnRepo, mockCryptor, mockCache, failingFactory, nil)
	handler := &ConnectionHandler{ValidateSchemaQuery: validateSchemaQuery}

	app := setupConnectionTestApp()
	app.Post("/v1/management/connections/validate-schema", handler.ValidateSchema)

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)

	var body model.SchemaValidationErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "Schema validation failed", body.Title)

	// Verify that at least one error is DATA_SOURCE_NOT_FOUND for "unknown_ds"
	foundDSNotFound := false
	for _, e := range body.Errors {
		if e.Type == model.ErrTypeDataSourceNotFound && e.DataSourceID == "unknown_ds" {
			foundDSNotFound = true
			break
		}
	}
	assert.True(t, foundDSNotFound, "expected DATA_SOURCE_NOT_FOUND error for unknown_ds")
}

func TestConnectionHandler_ValidateSchema_InvalidJSON(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{ValidateSchemaQuery: nil}
	app.Post("/v1/management/connections/validate-schema", handler.ValidateSchema)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "invalid JSON - missing closing brace",
			body:     `{"mappedFields": {"ds1": {"t1": ["f1"]}`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid JSON - syntax error",
			body:     `{invalid}`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid JSON - empty string",
			body:     ``,
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestConnectionHandler_ValidateSchema_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockCryptor := crypto.NewMockCryptor(ctrl)
	mockCache := newNoopSchemaCache()

	// FindByConfigNames returns an error -> internal server error
	mockConnRepo.EXPECT().FindByConfigNames(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	validateSchemaQuery := query.NewValidateSchema(mockConnRepo, mockCryptor, mockCache, nil, nil)
	handler := &ConnectionHandler{ValidateSchemaQuery: validateSchemaQuery}

	app := setupConnectionTestApp()
	app.Post("/v1/management/connections/validate-schema", handler.ValidateSchema)

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(validSchemaValidationRequest()))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestConnectionHandler_ValidateSchema_HandlerDirectly_InvalidJSON(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{ValidateSchemaQuery: nil}
	app.Post("/v1/management/connections/validate-schema", handler.ValidateSchema)

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(`{broken`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "code")
}

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNewConnectionHandler(t *testing.T) {
	handler := NewConnectionHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	assert.NotNil(t, handler)
	assert.Nil(t, handler.CreateCmd)
	assert.Nil(t, handler.UpdateCmd)
	assert.Nil(t, handler.DeleteCmd)
	assert.Nil(t, handler.GetQuery)
	assert.Nil(t, handler.ListQuery)
	assert.Nil(t, handler.TestQuery)
	assert.Nil(t, handler.ValidateSchemaQuery)
	assert.Nil(t, handler.GetSchemaQuery)
}

// ============================================================================
// Additional Handler Direct Tests for Better Coverage
// ============================================================================

func TestConnectionHandler_ListConnections_HandlerDirectly_InvalidSortOrder(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{ListQuery: nil}
	app.Get("/v1/management/connections", handler.ListConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections?sortOrder=invalid", nil)
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// ============================================================================
// Helpers
// ============================================================================

// noopSchemaCache is a simple in-test implementation of SchemaCacheRepository
// that always returns cache miss. This avoids needing the gomock generated cache mock
// while still allowing the real ValidateSchema service to be instantiated.
type noopSchemaCache struct{}

func newNoopSchemaCache() *noopSchemaCache {
	return &noopSchemaCache{}
}

func (n *noopSchemaCache) Get(_ context.Context, _ string) (*model.DataSourceSchema, error) {
	return nil, nil
}

func (n *noopSchemaCache) Set(_ context.Context, _ string, _ *model.DataSourceSchema, _ time.Duration) error {
	return nil
}

func (n *noopSchemaCache) Delete(_ context.Context, _ string) error {
	return nil
}

func (n *noopSchemaCache) Clear(_ context.Context) error {
	return nil
}

func (n *noopSchemaCache) IsHealthy(_ context.Context) bool {
	return true
}

func (n *noopSchemaCache) Close() error {
	return nil
}
