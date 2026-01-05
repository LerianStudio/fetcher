package in

import (
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/fetcher/components/manager/api"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSwaggerEnvConfig_DefaultValues(t *testing.T) {
	// Save original values
	originalTitle := api.SwaggerInfo.Title
	originalDesc := api.SwaggerInfo.Description
	originalVersion := api.SwaggerInfo.Version
	originalHost := api.SwaggerInfo.Host
	originalBasePath := api.SwaggerInfo.BasePath
	originalSchemes := api.SwaggerInfo.Schemes

	// Restore after test
	defer func() {
		api.SwaggerInfo.Title = originalTitle
		api.SwaggerInfo.Description = originalDesc
		api.SwaggerInfo.Version = originalVersion
		api.SwaggerInfo.Host = originalHost
		api.SwaggerInfo.BasePath = originalBasePath
		api.SwaggerInfo.Schemes = originalSchemes
	}()

	app := fiber.New()

	handlerCalled := false
	app.Get("/swagger/*", WithSwaggerEnvConfig(), func(c *fiber.Ctx) error {
		handlerCalled = true
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/swagger/index.html", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.True(t, handlerCalled, "next handler should be called")
}

func TestWithSwaggerEnvConfig_WithEnvVars(t *testing.T) {
	// Save original values
	originalTitle := api.SwaggerInfo.Title
	originalDesc := api.SwaggerInfo.Description
	originalVersion := api.SwaggerInfo.Version
	originalHost := api.SwaggerInfo.Host
	originalBasePath := api.SwaggerInfo.BasePath
	originalSchemes := api.SwaggerInfo.Schemes
	originalLeftDelim := api.SwaggerInfo.LeftDelim
	originalRightDelim := api.SwaggerInfo.RightDelim

	// Restore after test
	defer func() {
		api.SwaggerInfo.Title = originalTitle
		api.SwaggerInfo.Description = originalDesc
		api.SwaggerInfo.Version = originalVersion
		api.SwaggerInfo.Host = originalHost
		api.SwaggerInfo.BasePath = originalBasePath
		api.SwaggerInfo.Schemes = originalSchemes
		api.SwaggerInfo.LeftDelim = originalLeftDelim
		api.SwaggerInfo.RightDelim = originalRightDelim
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T)
	}{
		{
			name: "set title",
			envVars: map[string]string{
				"SWAGGER_TITLE": "Custom API Title",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "Custom API Title", api.SwaggerInfo.Title)
			},
		},
		{
			name: "set description",
			envVars: map[string]string{
				"SWAGGER_DESCRIPTION": "Custom API Description",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "Custom API Description", api.SwaggerInfo.Description)
			},
		},
		{
			name: "set version",
			envVars: map[string]string{
				"SWAGGER_VERSION": "2.0.0",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "2.0.0", api.SwaggerInfo.Version)
			},
		},
		{
			name: "set valid host",
			envVars: map[string]string{
				"SWAGGER_HOST": "localhost:8080",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "localhost:8080", api.SwaggerInfo.Host)
			},
		},
		{
			name: "set base path",
			envVars: map[string]string{
				"SWAGGER_BASE_PATH": "/api/v2",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "/api/v2", api.SwaggerInfo.BasePath)
			},
		},
		{
			name: "set schemes",
			envVars: map[string]string{
				"SWAGGER_SCHEMES": "https",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"https"}, api.SwaggerInfo.Schemes)
			},
		},
		{
			name: "set delimiters",
			envVars: map[string]string{
				"SWAGGER_LEFT_DELIM":  "[[",
				"SWAGGER_RIGHT_DELIM": "]]",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "[[", api.SwaggerInfo.LeftDelim)
				assert.Equal(t, "]]", api.SwaggerInfo.RightDelim)
			},
		},
		{
			name: "set multiple values",
			envVars: map[string]string{
				"SWAGGER_TITLE":     "Multi Config API",
				"SWAGGER_VERSION":   "3.0.0",
				"SWAGGER_BASE_PATH": "/v3",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "Multi Config API", api.SwaggerInfo.Title)
				assert.Equal(t, "3.0.0", api.SwaggerInfo.Version)
				assert.Equal(t, "/v3", api.SwaggerInfo.BasePath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to original values before each test
			api.SwaggerInfo.Title = originalTitle
			api.SwaggerInfo.Description = originalDesc
			api.SwaggerInfo.Version = originalVersion
			api.SwaggerInfo.Host = originalHost
			api.SwaggerInfo.BasePath = originalBasePath
			api.SwaggerInfo.Schemes = originalSchemes
			api.SwaggerInfo.LeftDelim = originalLeftDelim
			api.SwaggerInfo.RightDelim = originalRightDelim

			// Set env vars for this test
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			app := fiber.New()
			app.Get("/swagger/*", WithSwaggerEnvConfig(), func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest("GET", "/swagger/index.html", nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, fiber.StatusOK, resp.StatusCode)
			tt.validate(t)
		})
	}
}

func TestWithSwaggerEnvConfig_InvalidHost(t *testing.T) {
	// Save original host
	originalHost := api.SwaggerInfo.Host
	defer func() {
		api.SwaggerInfo.Host = originalHost
	}()

	// Set a known value first
	api.SwaggerInfo.Host = "original-host"

	// Set invalid host (empty after validation)
	t.Setenv("SWAGGER_HOST", "invalid host with spaces")

	app := fiber.New()
	app.Get("/swagger/*", WithSwaggerEnvConfig(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/swagger/index.html", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	// Host should remain unchanged due to invalid validation
	assert.Equal(t, "original-host", api.SwaggerInfo.Host)
}

func TestWithSwaggerEnvConfig_EmptyEnvVars(t *testing.T) {
	// Save original values
	originalTitle := api.SwaggerInfo.Title
	defer func() {
		api.SwaggerInfo.Title = originalTitle
	}()

	// Set a known value
	api.SwaggerInfo.Title = "Original Title"

	// Set empty env var (should not change the value)
	t.Setenv("SWAGGER_TITLE", "")

	app := fiber.New()
	app.Get("/swagger/*", WithSwaggerEnvConfig(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/swagger/index.html", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	// Title should remain unchanged since empty env var is ignored
	assert.Equal(t, "Original Title", api.SwaggerInfo.Title)
}

func TestWithSwaggerEnvConfig_CallsNext(t *testing.T) {
	app := fiber.New()

	nextCalled := false
	app.Get("/test", WithSwaggerEnvConfig(), func(c *fiber.Ctx) error {
		nextCalled = true
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.True(t, nextCalled, "middleware should call c.Next()")
}
