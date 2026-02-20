package in

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/components/manager/internal/services/command"
	"github.com/LerianStudio/fetcher/components/manager/internal/services/query"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	productRepo "github.com/LerianStudio/fetcher/pkg/mongodb/product"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.uber.org/mock/gomock"
)

// setupProductTestApp creates a Fiber app with test context middleware for product tests.
func setupProductTestApp() *fiber.App {
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

func createTestProduct(id, orgID uuid.UUID) *model.Product {
	now := time.Now().UTC()
	return &model.Product{
		ID:             id,
		OrganizationID: orgID,
		Code:           "test-product",
		Name:           "Test Product",
		Description:    "A test product",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func validProductInput() string {
	return `{
		"code": "test-product",
		"name": "Test Product",
		"description": "A test product"
	}`
}

// ============================================================================
// CreateProduct Handler Tests
// ============================================================================

func TestProductHandler_CreateProduct_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	productID := uuid.New()
	testProduct := createTestProduct(productID, orgID)

	mockRepo.EXPECT().FindByCode(gomock.Any(), "test-product", orgID).Return(nil, nil)
	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(testProduct, nil)

	createCmd := command.NewCreateProduct(mockRepo)
	handler := &ProductHandler{CreateCmd: createCmd}

	app := setupProductTestApp()
	app.Post("/v1/management/products", handler.CreateProduct)

	req := httptest.NewRequest("POST", "/v1/management/products", strings.NewReader(validProductInput()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var body map[string]any
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, productID.String(), body["id"])
	assert.Equal(t, "test-product", body["code"])
}

func TestProductHandler_CreateProduct_InvalidJSON(t *testing.T) {
	handler := &ProductHandler{CreateCmd: nil}

	app := setupProductTestApp()
	app.Post("/v1/management/products", handler.CreateProduct)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "malformed JSON",
			body:     `{"code": "test"`,
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "syntax error",
			body:     `{invalid}`,
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/management/products", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestProductHandler_CreateProduct_EmptyBody(t *testing.T) {
	handler := &ProductHandler{CreateCmd: nil}

	app := setupProductTestApp()
	app.Post("/v1/management/products", handler.CreateProduct)

	req := httptest.NewRequest("POST", "/v1/management/products", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestProductHandler_CreateProduct_MissingOrgHeader(t *testing.T) {
	handler := &ProductHandler{CreateCmd: nil}

	app := setupProductTestApp()
	app.Post("/v1/management/products", handler.CreateProduct)

	tests := []struct {
		name      string
		orgHeader string
		wantCode  int
	}{
		{
			name:      "missing org header",
			orgHeader: "",
			wantCode:  fiber.StatusBadRequest,
		},
		{
			name:      "invalid org header",
			orgHeader: "not-a-uuid",
			wantCode:  fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/management/products", strings.NewReader(validProductInput()))
			req.Header.Set("Content-Type", "application/json")
			if tt.orgHeader != "" {
				req.Header.Set("X-Organization-Id", tt.orgHeader)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestProductHandler_CreateProduct_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productRepo.NewMockRepository(ctrl)

	orgID := uuid.New()

	mockRepo.EXPECT().FindByCode(gomock.Any(), gomock.Any(), orgID).Return(nil, errors.New("db error"))

	createCmd := command.NewCreateProduct(mockRepo)
	handler := &ProductHandler{CreateCmd: createCmd}

	app := setupProductTestApp()
	app.Post("/v1/management/products", handler.CreateProduct)

	req := httptest.NewRequest("POST", "/v1/management/products", strings.NewReader(validProductInput()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// ============================================================================
// GetProduct Handler Tests
// ============================================================================

func TestProductHandler_GetProduct_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	productID := uuid.New()
	testProduct := createTestProduct(productID, orgID)

	mockRepo.EXPECT().FindByID(gomock.Any(), productID, orgID).Return(testProduct, nil)

	getQuery := query.NewGetProduct(mockRepo)
	handler := &ProductHandler{GetQuery: getQuery}

	app := setupProductTestApp()
	app.Get("/v1/management/products/:id", handler.GetProduct)

	req := httptest.NewRequest("GET", "/v1/management/products/"+productID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]any
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, productID.String(), body["id"])
}

func TestProductHandler_GetProduct_InvalidUUID(t *testing.T) {
	handler := &ProductHandler{GetQuery: nil}

	app := setupProductTestApp()
	app.Get("/v1/management/products/:id", handler.GetProduct)

	req := httptest.NewRequest("GET", "/v1/management/products/not-a-uuid", nil)
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestProductHandler_GetProduct_MissingOrgHeader(t *testing.T) {
	handler := &ProductHandler{GetQuery: nil}

	app := setupProductTestApp()
	app.Get("/v1/management/products/:id", handler.GetProduct)

	req := httptest.NewRequest("GET", "/v1/management/products/"+uuid.New().String(), nil)
	// No org header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestProductHandler_GetProduct_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	productID := uuid.New()

	mockRepo.EXPECT().FindByID(gomock.Any(), productID, orgID).Return(nil, nil)

	getQuery := query.NewGetProduct(mockRepo)
	handler := &ProductHandler{GetQuery: getQuery}

	app := setupProductTestApp()
	app.Get("/v1/management/products/:id", handler.GetProduct)

	req := httptest.NewRequest("GET", "/v1/management/products/"+productID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// ============================================================================
// ListProducts Handler Tests
// ============================================================================

func TestProductHandler_ListProducts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	products := []*model.Product{
		createTestProduct(uuid.New(), orgID),
		createTestProduct(uuid.New(), orgID),
	}

	mockRepo.EXPECT().List(gomock.Any(), orgID, gomock.Any()).Return(products, int64(2), nil)

	listQuery := query.NewListProducts(mockRepo)
	handler := &ProductHandler{ListQuery: listQuery}

	app := setupProductTestApp()
	app.Get("/v1/management/products", handler.ListProducts)

	req := httptest.NewRequest("GET", "/v1/management/products", nil)
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

func TestProductHandler_ListProducts_MissingOrgHeader(t *testing.T) {
	handler := &ProductHandler{ListQuery: nil}

	app := setupProductTestApp()
	app.Get("/v1/management/products", handler.ListProducts)

	req := httptest.NewRequest("GET", "/v1/management/products", nil)
	// No org header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestProductHandler_ListProducts_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productRepo.NewMockRepository(ctrl)

	orgID := uuid.New()

	mockRepo.EXPECT().List(gomock.Any(), orgID, gomock.Any()).Return(nil, int64(0), nil)

	listQuery := query.NewListProducts(mockRepo)
	handler := &ProductHandler{ListQuery: listQuery}

	app := setupProductTestApp()
	app.Get("/v1/management/products", handler.ListProducts)

	req := httptest.NewRequest("GET", "/v1/management/products", nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ============================================================================
// UpdateProduct Handler Tests
// ============================================================================

func TestProductHandler_UpdateProduct_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := productRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	productID := uuid.New()
	testProduct := createTestProduct(productID, orgID)

	mockRepo.EXPECT().FindByID(gomock.Any(), productID, orgID).Return(testProduct, nil)
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(testProduct, nil)

	updateCmd := command.NewUpdateProduct(mockRepo)
	handler := &ProductHandler{UpdateCmd: updateCmd}

	app := setupProductTestApp()
	app.Patch("/v1/management/products/:id", handler.UpdateProduct)

	body := `{"name": "Updated Name"}`
	req := httptest.NewRequest("PATCH", "/v1/management/products/"+productID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestProductHandler_UpdateProduct_InvalidUUID(t *testing.T) {
	handler := &ProductHandler{UpdateCmd: nil}

	app := setupProductTestApp()
	app.Patch("/v1/management/products/:id", handler.UpdateProduct)

	body := `{"name": "Updated"}`
	req := httptest.NewRequest("PATCH", "/v1/management/products/not-a-uuid", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestProductHandler_UpdateProduct_EmptyBody(t *testing.T) {
	handler := &ProductHandler{UpdateCmd: nil}

	app := setupProductTestApp()
	app.Patch("/v1/management/products/:id", handler.UpdateProduct)

	req := httptest.NewRequest("PATCH", "/v1/management/products/"+uuid.New().String(), strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestProductHandler_UpdateProduct_MissingOrgHeader(t *testing.T) {
	handler := &ProductHandler{UpdateCmd: nil}

	app := setupProductTestApp()
	app.Patch("/v1/management/products/:id", handler.UpdateProduct)

	body := `{"name": "Updated"}`
	req := httptest.NewRequest("PATCH", "/v1/management/products/"+uuid.New().String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No org header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestProductHandler_UpdateProduct_InvalidJSON(t *testing.T) {
	handler := &ProductHandler{UpdateCmd: nil}

	app := setupProductTestApp()
	app.Patch("/v1/management/products/:id", handler.UpdateProduct)

	req := httptest.NewRequest("PATCH", "/v1/management/products/"+uuid.New().String(), strings.NewReader(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// ============================================================================
// DeleteProduct Handler Tests
// ============================================================================

func TestProductHandler_DeleteProduct_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	productID := uuid.New()
	testProduct := createTestProduct(productID, orgID)

	mockProductRepo.EXPECT().FindByID(gomock.Any(), productID, orgID).Return(testProduct, nil)
	mockConnRepo.EXPECT().CountByProduct(gomock.Any(), orgID, productID).Return(int64(0), nil)
	mockProductRepo.EXPECT().Delete(gomock.Any(), productID, orgID, gomock.Any()).Return(nil)

	deleteCmd := command.NewDeleteProduct(mockProductRepo, mockConnRepo)
	handler := &ProductHandler{DeleteCmd: deleteCmd}

	app := setupProductTestApp()
	app.Delete("/v1/management/products/:id", handler.DeleteProduct)

	req := httptest.NewRequest("DELETE", "/v1/management/products/"+productID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestProductHandler_DeleteProduct_InvalidUUID(t *testing.T) {
	handler := &ProductHandler{DeleteCmd: nil}

	app := setupProductTestApp()
	app.Delete("/v1/management/products/:id", handler.DeleteProduct)

	req := httptest.NewRequest("DELETE", "/v1/management/products/not-a-uuid", nil)
	req.Header.Set("X-Organization-Id", uuid.New().String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestProductHandler_DeleteProduct_MissingOrgHeader(t *testing.T) {
	handler := &ProductHandler{DeleteCmd: nil}

	app := setupProductTestApp()
	app.Delete("/v1/management/products/:id", handler.DeleteProduct)

	req := httptest.NewRequest("DELETE", "/v1/management/products/"+uuid.New().String(), nil)
	// No org header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestProductHandler_DeleteProduct_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	productID := uuid.New()

	mockProductRepo.EXPECT().FindByID(gomock.Any(), productID, orgID).Return(nil, nil)

	deleteCmd := command.NewDeleteProduct(mockProductRepo, mockConnRepo)
	handler := &ProductHandler{DeleteCmd: deleteCmd}

	app := setupProductTestApp()
	app.Delete("/v1/management/products/:id", handler.DeleteProduct)

	req := httptest.NewRequest("DELETE", "/v1/management/products/"+productID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestProductHandler_DeleteProduct_HasConnections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductRepo := productRepo.NewMockRepository(ctrl)
	mockConnRepo := connRepo.NewMockRepository(ctrl)

	orgID := uuid.New()
	productID := uuid.New()
	testProduct := createTestProduct(productID, orgID)

	mockProductRepo.EXPECT().FindByID(gomock.Any(), productID, orgID).Return(testProduct, nil)
	mockConnRepo.EXPECT().CountByProduct(gomock.Any(), orgID, productID).Return(int64(3), nil)

	deleteCmd := command.NewDeleteProduct(mockProductRepo, mockConnRepo)
	handler := &ProductHandler{DeleteCmd: deleteCmd}

	app := setupProductTestApp()
	app.Delete("/v1/management/products/:id", handler.DeleteProduct)

	req := httptest.NewRequest("DELETE", "/v1/management/products/"+productID.String(), nil)
	req.Header.Set("X-Organization-Id", orgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}
