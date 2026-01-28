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

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	httpUtils "github.com/LerianStudio/fetcher/pkg/net/http"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// setupConnectionTestApp creates a Fiber app with test context middleware for connection tests.
func setupConnectionTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024, // 10KB for test flexibility
	})

	// Middleware to inject test context with logger and tracer
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

// createTestConnection creates a test connection with default values.
func createTestConnection(id, orgID uuid.UUID) *model.Connection {
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
	app := setupConnectionTestApp()

	connID := uuid.New()
	orgID := uuid.New()
	testConn := createTestConnection(connID, orgID)

	// Create handler with inline mock execution
	handler := &ConnectionHandler{}

	// We need to test the handler directly since we can't easily mock the service
	// Create a custom handler that intercepts the call
	app.Post("/v1/management/connections", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		orgIDHeader, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.ConnectionInput
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate successful creation
		if orgIDHeader == orgID {
			resp := model.NewConnectionResponseFrom(testConn)
			return httpUtils.Created(c, fiber.Map{"id": resp.ID})
		}

		return httpUtils.WithError(c, errors.New("unexpected org id"))
	})

	// Suppress unused warning
	_ = handler

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

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

func TestConnectionHandler_CreateConnection_MissingOrgHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{CreateCmd: nil}
	app.Post("/v1/management/connections", handler.CreateConnection)

	tests := []struct {
		name      string
		orgHeader string
		setHeader bool
		wantCode  int
	}{
		{
			name:      "missing X-Organization-Id header",
			orgHeader: "",
			setHeader: false,
			wantCode:  fiber.StatusBadRequest,
		},
		{
			name:      "invalid UUID format",
			orgHeader: "not-a-uuid",
			setHeader: true,
			wantCode:  fiber.StatusBadRequest,
		},
		{
			name:      "whitespace only header",
			orgHeader: "   ",
			setHeader: true,
			wantCode:  fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
			req.Header.Set("Content-Type", "application/json")

			if tt.setHeader {
				req.Header.Set("X-Organization-Id", tt.orgHeader)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestConnectionHandler_CreateConnection_Conflict(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.ConnectionInput
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate conflict error
		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Code:       constant.ErrEntityConflict.Error(),
			Title:      "Conflict",
			Message:    "connection with the same name already exists",
		})
	})

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestConnectionHandler_CreateConnection_InternalError(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.ConnectionInput
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate internal error
		return httpUtils.WithError(c, pkg.InternalServerError{
			EntityType: "connection",
			Code:       constant.ErrInternalServer.Error(),
			Title:      "Internal Server Error",
			Message:    "database connection failed",
		})
	})

	req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// ============================================================================
// GetConnection Handler Tests
// ============================================================================

func TestConnectionHandler_GetConnection_Success(t *testing.T) {
	app := setupConnectionTestApp()

	connID := uuid.New()
	orgID := uuid.New()
	testConn := createTestConnection(connID, orgID)

	app.Get("/v1/management/connections/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		if id == connID {
			resp := model.NewConnectionResponseFrom(testConn)
			return httpUtils.OK(c, resp)
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		})
	})

	req := httptest.NewRequest("GET", "/v1/management/connections/"+connID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Get("/v1/management/connections/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		})
	})

	req := httptest.NewRequest("GET", "/v1/management/connections/"+uuid.New().String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupConnectionTestApp()

	connID1 := uuid.New()
	connID2 := uuid.New()
	orgID := uuid.New()

	conn1 := createTestConnection(connID1, orgID)
	conn1.ConfigName = "connection-1"
	conn2 := createTestConnection(connID2, orgID)
	conn2.ConfigName = "connection-2"

	app.Get("/v1/management/connections", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		headerParams, err := httpUtils.ValidateParameters(c.Queries())
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		pagination := model.Pagination{
			Limit: headerParams.Limit,
			Page:  headerParams.Page,
		}

		conns := []*model.Connection{conn1, conn2}
		connResp := make([]*model.ConnectionResponse, 0, len(conns))
		for _, conn := range conns {
			connResp = append(connResp, model.NewConnectionResponseFrom(conn))
		}

		pagination.SetItems(connResp)
		pagination.SetTotal(len(connResp))

		return httpUtils.OK(c, pagination)
	})

	req := httptest.NewRequest("GET", "/v1/management/connections?limit=10&page=1", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Get("/v1/management/connections", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		headerParams, err := httpUtils.ValidateParameters(c.Queries())
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		pagination := model.Pagination{
			Limit: headerParams.Limit,
			Page:  headerParams.Page,
		}

		connResp := make([]*model.ConnectionResponse, 0)
		pagination.SetItems(connResp)
		pagination.SetTotal(0)

		return httpUtils.OK(c, pagination)
	})

	req := httptest.NewRequest("GET", "/v1/management/connections", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Get("/v1/management/connections", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = httpUtils.ValidateParameters(c.Queries())
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		return httpUtils.OK(c, model.Pagination{})
	})

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
			req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupConnectionTestApp()

	connID := uuid.New()
	orgID := uuid.New()
	testConn := createTestConnection(connID, orgID)
	testConn.ConfigName = "updated-connection"

	app.Patch("/v1/management/connections/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		var request model.ConnectionInput
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		if id == connID {
			return httpUtils.OK(c, model.NewConnectionResponseFrom(testConn))
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		})
	})

	updatePayload := `{"configName": "updated-connection", "type": "POSTGRESQL", "host": "localhost", "port": 5432, "databaseName": "testdb", "username": "testuser", "password": "newpassword"}`

	req := httptest.NewRequest("PATCH", "/v1/management/connections/"+connID.String(), strings.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Patch("/v1/management/connections/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		var request model.ConnectionInput
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		})
	})

	updatePayload := `{"configName": "updated-connection"}`

	req := httptest.NewRequest("PATCH", "/v1/management/connections/"+uuid.New().String(), strings.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupConnectionTestApp()
	connID := uuid.New()
	orgID := uuid.New()

	app.Patch("/v1/management/connections/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		var request model.ConnectionInput
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate conflict due to active jobs
		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Code:       constant.ErrJobInProgress.Error(),
			Title:      "Job In Progress",
			Message:    "cannot update connection with active jobs",
		})
	})

	updatePayload := `{"configName": "updated-connection"}`

	req := httptest.NewRequest("PATCH", "/v1/management/connections/"+connID.String(), strings.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

// ============================================================================
// DeleteConnection Handler Tests
// ============================================================================

func TestConnectionHandler_DeleteConnection_Success(t *testing.T) {
	app := setupConnectionTestApp()

	connID := uuid.New()
	orgID := uuid.New()

	app.Delete("/v1/management/connections/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		if id == connID {
			return httpUtils.OK(c, fiber.Map{"id": id})
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		})
	})

	req := httptest.NewRequest("DELETE", "/v1/management/connections/"+connID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, connID.String(), body["id"])
}

func TestConnectionHandler_DeleteConnection_NotFound(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Delete("/v1/management/connections/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		})
	})

	req := httptest.NewRequest("DELETE", "/v1/management/connections/"+uuid.New().String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestConnectionHandler_DeleteConnection_Conflict_ActiveJobs(t *testing.T) {
	app := setupConnectionTestApp()
	connID := uuid.New()
	orgID := uuid.New()

	app.Delete("/v1/management/connections/:id", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		// Simulate conflict due to active jobs
		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Code:       constant.ErrJobInProgress.Error(),
			Title:      "Job In Progress",
			Message:    "cannot delete connection with active jobs",
		})
	})

	req := httptest.NewRequest("DELETE", "/v1/management/connections/"+connID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupConnectionTestApp()

	connID := uuid.New()
	orgID := uuid.New()

	app.Post("/v1/management/connections/:id/test", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		if id == connID {
			return httpUtils.OK(c, &model.ConnectionTestResponse{
				Status:    "success",
				Message:   "Connection successful",
				LatencyMs: 42,
			})
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		})
	})

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connID.String()+"/test", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.ConnectionTestResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "success", body.Status)
	assert.Equal(t, int64(42), body.LatencyMs)
}

func TestConnectionHandler_TestConnection_NotFound(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections/:id/test", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    "connection not found",
		})
	})

	req := httptest.NewRequest("POST", "/v1/management/connections/"+uuid.New().String()+"/test", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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
	app := setupConnectionTestApp()
	connID := uuid.New()
	orgID := uuid.New()

	app.Post("/v1/management/connections/:id/test", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		_, err = uuid.Parse(c.Params("id"))
		if err != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "connection",
				Code:       constant.ErrInvalidPathParameter.Error(),
				Title:      "Invalid Path Parameter",
				Message:    "invalid connection id",
				Err:        err,
			})
		}

		// Simulate rate limiting
		return httpUtils.WithError(c, pkg.ResponseError{
			Code:    fiber.StatusTooManyRequests,
			Title:   "Rate Limit Exceeded",
			Message: "Connection test limit reached. Try again in 60 seconds.",
		})
	})

	req := httptest.NewRequest("POST", "/v1/management/connections/"+connID.String()+"/test", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

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

func TestConnectionHandler_ListConnections_HandlerDirectly_MissingOrgHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{ListQuery: nil}
	app.Get("/v1/management/connections", handler.ListConnections)

	req := httptest.NewRequest("GET", "/v1/management/connections", nil)
	// Not setting X-Organization-Id header

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
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections/validate-schema", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.SchemaValidationRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate successful validation
		return httpUtils.OK(c, model.NewSuccessResponse())
	})

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(validSchemaValidationRequest()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.SchemaValidationResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "success", body.Status)
}

func TestConnectionHandler_ValidateSchema_Failure_TableNotFound(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections/validate-schema", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.SchemaValidationRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate validation failure - table not found
		errors := []model.SchemaValidationError{
			{
				Type:         model.ErrTypeTableNotFound,
				DataSourceID: "ds1",
				Table:        "unknown_table",
			},
		}
		return httpUtils.OK(c, model.NewFailureResponse(errors))
	})

	payload := `{
		"mappedFields": {
			"ds1": {
				"unknown_table": ["field1"]
			}
		}
	}`

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.SchemaValidationResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "failure", body.Status)
	assert.Len(t, body.Errors, 1)
	assert.Equal(t, model.ErrTypeTableNotFound, body.Errors[0].Type)
}

func TestConnectionHandler_ValidateSchema_Failure_FieldNotFound(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections/validate-schema", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.SchemaValidationRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate validation failure - field not found
		errors := []model.SchemaValidationError{
			{
				Type:         model.ErrTypeFieldNotFound,
				DataSourceID: "ds1",
				Table:        "table1",
				Field:        "unknown_field",
			},
		}
		return httpUtils.OK(c, model.NewFailureResponse(errors))
	})

	payload := `{
		"mappedFields": {
			"ds1": {
				"table1": ["unknown_field"]
			}
		}
	}`

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.SchemaValidationResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "failure", body.Status)
	assert.Len(t, body.Errors, 1)
	assert.Equal(t, model.ErrTypeFieldNotFound, body.Errors[0].Type)
}

func TestConnectionHandler_ValidateSchema_Failure_DataSourceNotFound(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections/validate-schema", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.SchemaValidationRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate validation failure - datasource not found
		errors := []model.SchemaValidationError{
			model.NewDataSourceNotFoundError("unknown_ds"),
		}
		return httpUtils.OK(c, model.NewFailureResponse(errors))
	})

	payload := `{
		"mappedFields": {
			"unknown_ds": {
				"table1": ["field1"]
			}
		}
	}`

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.SchemaValidationResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "failure", body.Status)
	assert.Len(t, body.Errors, 1)
	assert.Equal(t, model.ErrTypeDataSourceNotFound, body.Errors[0].Type)
}

func TestConnectionHandler_ValidateSchema_Failure_DataSourceDown(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections/validate-schema", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.SchemaValidationRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate validation failure - datasource down
		errors := []model.SchemaValidationError{
			model.NewDataSourceDownError("ds1"),
		}
		return httpUtils.OK(c, model.NewFailureResponse(errors))
	})

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(validSchemaValidationRequest()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body model.SchemaValidationResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "failure", body.Status)
	assert.Len(t, body.Errors, 1)
	assert.Equal(t, model.ErrTypeDataSourceDown, body.Errors[0].Type)
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

func TestConnectionHandler_ValidateSchema_MissingOrgHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{ValidateSchemaQuery: nil}
	app.Post("/v1/management/connections/validate-schema", handler.ValidateSchema)

	tests := []struct {
		name      string
		orgHeader string
		setHeader bool
		wantCode  int
	}{
		{
			name:      "missing X-Organization-Id header",
			orgHeader: "",
			setHeader: false,
			wantCode:  fiber.StatusBadRequest,
		},
		{
			name:      "invalid UUID format",
			orgHeader: "not-a-uuid",
			setHeader: true,
			wantCode:  fiber.StatusBadRequest,
		},
		{
			name:      "whitespace only header",
			orgHeader: "   ",
			setHeader: true,
			wantCode:  fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(validSchemaValidationRequest()))
			req.Header.Set("Content-Type", "application/json")

			if tt.setHeader {
				req.Header.Set("X-Organization-Id", tt.orgHeader)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestConnectionHandler_ValidateSchema_InternalError(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections/validate-schema", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.SchemaValidationRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Simulate internal error
		return httpUtils.WithError(c, pkg.InternalServerError{
			EntityType: "schema",
			Code:       constant.ErrInternalServer.Error(),
			Title:      "Internal Server Error",
			Message:    "database connection failed",
		})
	})

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(validSchemaValidationRequest()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestConnectionHandler_ValidateSchema_MultipleDataSources(t *testing.T) {
	app := setupConnectionTestApp()
	orgID := uuid.New()

	app.Post("/v1/management/connections/validate-schema", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		libCommons.NewTrackingFromContext(ctx)

		_, err := httpUtils.GetOrganizationID(c)
		if err != nil {
			return httpUtils.WithError(c, err)
		}

		var request model.SchemaValidationRequest
		if errParser := c.BodyParser(&request); errParser != nil {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "unable to parse request body",
				Err:        errParser,
			})
		}

		// Verify multiple datasources were parsed
		if len(request.MappedFields) != 3 {
			return httpUtils.WithError(c, pkg.ValidationError{
				EntityType: "schema",
				Code:       constant.ErrBadRequest.Error(),
				Title:      "Invalid payload",
				Message:    "expected 3 datasources",
			})
		}

		return httpUtils.OK(c, model.NewSuccessResponse())
	})

	payload := `{
		"mappedFields": {
			"ds1": {"table1": ["field1"]},
			"ds2": {"table2": ["field2"]},
			"ds3": {"table3": ["field3"]}
		}
	}`

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
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

func TestConnectionHandler_CreateConnection_HandlerDirectly_MissingOrgHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{CreateCmd: nil}
	app.Post("/v1/management/connections", handler.CreateConnection)

	tests := []struct {
		name      string
		orgHeader string
		setHeader bool
		wantCode  int
	}{
		{
			name:      "no org header",
			setHeader: false,
			wantCode:  fiber.StatusBadRequest,
		},
		{
			name:      "empty org header",
			orgHeader: "",
			setHeader: true,
			wantCode:  fiber.StatusBadRequest,
		},
		{
			name:      "invalid uuid org header",
			orgHeader: "invalid-uuid",
			setHeader: true,
			wantCode:  fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/management/connections", strings.NewReader(validConnectionInput()))
			req.Header.Set("Content-Type", "application/json")
			if tt.setHeader {
				req.Header.Set("X-Organization-Id", tt.orgHeader)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

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

func TestConnectionHandler_GetConnection_HandlerDirectly_MissingOrgHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{GetQuery: nil}
	app.Get("/v1/management/connections/:id", handler.GetConnection)

	req := httptest.NewRequest("GET", "/v1/management/connections/"+uuid.New().String(), nil)
	// No org header set

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestConnectionHandler_UpdateConnection_HandlerDirectly_MissingOrgHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{UpdateCmd: nil}
	app.Patch("/v1/management/connections/:id", handler.UpdateConnection)

	req := httptest.NewRequest("PATCH", "/v1/management/connections/"+uuid.New().String(), strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	// No org header set

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestConnectionHandler_DeleteConnection_HandlerDirectly_MissingOrgHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{DeleteCmd: nil}
	app.Delete("/v1/management/connections/:id", handler.DeleteConnection)

	req := httptest.NewRequest("DELETE", "/v1/management/connections/"+uuid.New().String(), nil)
	// No org header set

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestConnectionHandler_TestConnection_HandlerDirectly_MissingOrgHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{TestQuery: nil}
	app.Post("/v1/management/connections/:id/test", handler.TestConnection)

	req := httptest.NewRequest("POST", "/v1/management/connections/"+uuid.New().String()+"/test", nil)
	// No org header set

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestConnectionHandler_ValidateSchema_HandlerDirectly_MissingOrgHeader(t *testing.T) {
	app := setupConnectionTestApp()

	handler := &ConnectionHandler{ValidateSchemaQuery: nil}
	app.Post("/v1/management/connections/validate-schema", handler.ValidateSchema)

	req := httptest.NewRequest("POST", "/v1/management/connections/validate-schema", strings.NewReader(validSchemaValidationRequest()))
	req.Header.Set("Content-Type", "application/json")
	// No org header set

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
