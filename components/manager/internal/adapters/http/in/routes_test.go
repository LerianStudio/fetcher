package in

import (
	"net/http/httptest"
	"testing"

	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRoutes_CreatesApp(t *testing.T) {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	tl := &opentelemetry.Telemetry{}

	// Create handlers with nil dependencies (they won't be called in route registration)
	connectionHandler := &ConnectionHandler{}
	fetcherHandler := &FetcherHandler{}

	// Create routes without auth (pass nil for auth and license)
	app := NewRoutes(logger, tl, nil, nil, connectionHandler, fetcherHandler)

	assert.NotNil(t, app, "NewRoutes should return a non-nil Fiber app")
}

func TestNewRoutes_HealthEndpoint(t *testing.T) {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	tl := &opentelemetry.Telemetry{}

	connectionHandler := &ConnectionHandler{}
	fetcherHandler := &FetcherHandler{}

	app := NewRoutes(logger, tl, nil, nil, connectionHandler, fetcherHandler)

	req := httptest.NewRequest("GET", "/health", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Health endpoint should return 200 OK
	assert.Equal(t, 200, resp.StatusCode)
}

func TestNewRoutes_VersionEndpoint(t *testing.T) {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	tl := &opentelemetry.Telemetry{}

	connectionHandler := &ConnectionHandler{}
	fetcherHandler := &FetcherHandler{}

	app := NewRoutes(logger, tl, nil, nil, connectionHandler, fetcherHandler)

	req := httptest.NewRequest("GET", "/version", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Version endpoint should return 200 OK
	assert.Equal(t, 200, resp.StatusCode)
}

func TestNewRoutes_SwaggerEndpoint(t *testing.T) {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	tl := &opentelemetry.Telemetry{}

	connectionHandler := &ConnectionHandler{}
	fetcherHandler := &FetcherHandler{}

	app := NewRoutes(logger, tl, nil, nil, connectionHandler, fetcherHandler)

	req := httptest.NewRequest("GET", "/swagger/index.html", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Swagger endpoint should be accessible (may redirect or return content)
	assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 400,
		"Swagger endpoint should be accessible, got status %d", resp.StatusCode)
}

func TestNewRoutes_ConnectionsRoutesRegistered(t *testing.T) {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	tl := &opentelemetry.Telemetry{}

	connectionHandler := &ConnectionHandler{}
	fetcherHandler := &FetcherHandler{}

	app := NewRoutes(logger, tl, nil, nil, connectionHandler, fetcherHandler)

	// Test that routes exist (they will fail auth but route should be registered)
	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/v1/management/connections"},
		{"GET", "/v1/management/connections"},
		{"POST", "/v1/management/connections/validate-schema"},
		{"GET", "/v1/management/connections/123e4567-e89b-12d3-a456-426614174000"},
		{"POST", "/v1/management/connections/123e4567-e89b-12d3-a456-426614174000/test"},
		{"PATCH", "/v1/management/connections/123e4567-e89b-12d3-a456-426614174000"},
		{"DELETE", "/v1/management/connections/123e4567-e89b-12d3-a456-426614174000"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Route should exist (not 404), may fail auth (401/403) or bad request (400)
			assert.NotEqual(t, 404, resp.StatusCode,
				"Route %s %s should be registered", route.method, route.path)
		})
	}
}

func TestNewRoutes_FetcherRoutesRegistered(t *testing.T) {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	tl := &opentelemetry.Telemetry{}

	connectionHandler := &ConnectionHandler{}
	fetcherHandler := &FetcherHandler{}

	app := NewRoutes(logger, tl, nil, nil, connectionHandler, fetcherHandler)

	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/v1/fetcher"},
		{"GET", "/v1/fetcher/123e4567-e89b-12d3-a456-426614174000"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Route should exist (not 404)
			assert.NotEqual(t, 404, resp.StatusCode,
				"Route %s %s should be registered", route.method, route.path)
		})
	}
}

func TestNewRoutes_NonExistentRoute(t *testing.T) {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	tl := &opentelemetry.Telemetry{}

	connectionHandler := &ConnectionHandler{}
	fetcherHandler := &FetcherHandler{}

	app := NewRoutes(logger, tl, nil, nil, connectionHandler, fetcherHandler)

	req := httptest.NewRequest("GET", "/non-existent-route", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Non-existent route should return 404
	assert.Equal(t, 404, resp.StatusCode)
}

func TestNewRoutes_CORSEnabled(t *testing.T) {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	tl := &opentelemetry.Telemetry{}

	connectionHandler := &ConnectionHandler{}
	fetcherHandler := &FetcherHandler{}

	app := NewRoutes(logger, tl, nil, nil, connectionHandler, fetcherHandler)

	// Test CORS preflight request
	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// CORS should be enabled - check for CORS headers
	corsHeader := resp.Header.Get("Access-Control-Allow-Origin")
	assert.NotEmpty(t, corsHeader, "CORS should be enabled")
}

func TestNewRoutes_Constants(t *testing.T) {
	// Test that constants are defined correctly
	assert.Equal(t, "fetcher", applicationName)
	assert.Equal(t, "connections", connectionsResource)
	assert.Equal(t, "fetcher", fetcherResource)
}
