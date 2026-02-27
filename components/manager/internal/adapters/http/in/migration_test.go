package in

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/query"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"

	libCommons "github.com/LerianStudio/lib-commons/v3/commons"
	libLog "github.com/LerianStudio/lib-commons/v3/commons/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.uber.org/mock/gomock"
)

func setupMigrationTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024,
	})

	app.Use(func(c *fiber.Ctx) error {
		logger := &libLog.GoLogger{Level: libLog.DebugLevel}
		values := &libCommons.CustomContextKeyValue{
			HeaderID: "test-request-id",
			Logger:   logger,
			Tracer:   otel.Tracer("test"),
		}

		ctx := c.UserContext()
		ctx = context.WithValue(ctx, libCommons.CustomContextKey, values)
		c.SetUserContext(ctx)

		return c.Next()
	})

	return app
}

func createTestConnectionForMigration(id, orgID uuid.UUID) *model.Connection {
	now := time.Now().UTC()
	return &model.Connection{
		ID:                   id,
		OrganizationID:       orgID,
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

// ============================================================================
// ListUnassignedConnections Handler Tests
// ============================================================================

func TestMigrationHandler_ListUnassignedConnections_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	connections := []*model.Connection{
		createTestConnectionForMigration(uuid.New(), orgID),
		createTestConnectionForMigration(uuid.New(), orgID),
	}

	mockConnRepo.EXPECT().ListUnassigned(gomock.Any(), orgID, gomock.Any()).Return(connections, int64(2), nil)

	listQry := query.NewListUnassignedConnections(mockConnRepo)
	handler := &MigrationHandler{ListUnassignedQry: listQry}

	app := setupMigrationTestApp()
	app.Get("/v1/management/connections/unassigned", handler.ListUnassignedConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections/unassigned", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var body map[string]any
	err = json.Unmarshal(bodyBytes, &body)
	require.NoError(t, err)

	items, ok := body["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestMigrationHandler_ListUnassignedConnections_MissingOrgHeader(t *testing.T) {
	handler := &MigrationHandler{ListUnassignedQry: nil}

	app := setupMigrationTestApp()
	app.Get("/v1/management/connections/unassigned", handler.ListUnassignedConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections/unassigned", nil)
	// No org header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestMigrationHandler_ListUnassignedConnections_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	orgID := uuid.New()

	mockConnRepo.EXPECT().ListUnassigned(gomock.Any(), orgID, gomock.Any()).Return(nil, int64(0), nil)

	listQry := query.NewListUnassignedConnections(mockConnRepo)
	handler := &MigrationHandler{ListUnassignedQry: listQry}

	app := setupMigrationTestApp()
	app.Get("/v1/management/connections/unassigned", handler.ListUnassignedConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections/unassigned", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ============================================================================
// AssignConnectionToProduct Handler Tests
// ============================================================================

func TestMigrationHandler_AssignConnectionToProduct_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()
	testConn := createTestConnectionForMigration(connID, orgID)
	testConn.ProductName = "reporter"

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID, orgID).Return(testConn, nil)
	mockConnRepo.EXPECT().AssignProductName(gomock.Any(), connID, orgID, "reporter").Return(testConn, nil)

	assignCmd := command.NewAssignConnection(mockConnRepo)
	handler := &MigrationHandler{AssignCmd: assignCmd}

	app := setupMigrationTestApp()
	app.Post("/v1/management/connections/:id/assign", handler.AssignConnectionToProduct)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connID.String()+"/assign", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Product-Name", "reporter")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestMigrationHandler_AssignConnectionToProduct_MissingOrgHeader(t *testing.T) {
	handler := &MigrationHandler{AssignCmd: nil}

	app := setupMigrationTestApp()
	app.Post("/v1/management/connections/:id/assign", handler.AssignConnectionToProduct)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+uuid.New().String()+"/assign", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Product-Name", "reporter")
	// No org header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestMigrationHandler_AssignConnectionToProduct_InvalidConnectionID(t *testing.T) {
	handler := &MigrationHandler{AssignCmd: nil}

	app := setupMigrationTestApp()
	app.Post("/v1/management/connections/:id/assign", handler.AssignConnectionToProduct)

	req := httptest.NewRequest("POST", "/v1/management/connections/not-a-uuid/assign", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())
	req.Header.Set("X-Product-Name", "reporter")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestMigrationHandler_AssignConnectionToProduct_MissingProductName(t *testing.T) {
	handler := &MigrationHandler{AssignCmd: nil}

	app := setupMigrationTestApp()
	app.Post("/v1/management/connections/:id/assign", handler.AssignConnectionToProduct)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+uuid.New().String()+"/assign", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())
	// No X-Product-Name header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestMigrationHandler_AssignConnectionToProduct_ConnectionNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	connID := uuid.New()

	mockConnRepo.EXPECT().FindByID(gomock.Any(), connID, orgID).Return(nil, nil)

	assignCmd := command.NewAssignConnection(mockConnRepo)
	handler := &MigrationHandler{AssignCmd: assignCmd}

	app := setupMigrationTestApp()
	app.Post("/v1/management/connections/:id/assign", handler.AssignConnectionToProduct)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connID.String()+"/assign", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())
	req.Header.Set("X-Product-Name", "reporter")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
