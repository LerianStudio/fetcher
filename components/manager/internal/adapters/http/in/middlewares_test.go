package in

import (
	"context"
	"net/http/httptest"
	"testing"

	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// setupMiddlewareTestApp creates a Fiber app for middleware tests.
func setupMiddlewareTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024, // 10KB for test flexibility
	})

	// Middleware to inject test context with logger and tracer
	app.Use(func(c *fiber.Ctx) error {
		logger := &libLog.GoLogger{Level: libLog.LevelDebug}
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

// ============================================================================
// ParsePathParametersUUID Middleware Tests
// ============================================================================

func TestParsePathParametersUUID_ValidUUID(t *testing.T) {
	app := setupMiddlewareTestApp()
	validUUID := uuid.New()

	// Setup route with middleware
	app.Get("/test/:id", ParsePathParametersUUID, func(c *fiber.Ctx) error {
		// Verify UUID was stored in locals
		storedID := c.Locals(UUIDPathParameter)
		assert.NotNil(t, storedID)
		assert.Equal(t, validUUID, storedID)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test/"+validUUID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestParsePathParametersUUID_InvalidUUID(t *testing.T) {
	tests := []struct {
		name     string
		pathID   string
		wantCode int
	}{
		{
			name:     "invalid UUID format - random string",
			pathID:   "not-a-uuid",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - partial UUID",
			pathID:   "550e8400-e29b",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - too short",
			pathID:   "123",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - special characters",
			pathID:   "550e8400-e29b-<script>",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - alphanumeric mix",
			pathID:   "abc123def456",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - with extra hyphen",
			pathID:   "550e8400-e29b-41d4-a716-446655440000-extra",
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupMiddlewareTestApp()

			app.Get("/test/:id", ParsePathParametersUUID, func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test/"+tt.pathID, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestParsePathParametersUUID_EmptyPathParameter(t *testing.T) {
	app := setupMiddlewareTestApp()

	// When path parameter is empty (route matches but param is empty)
	// This simulates a route like /test/:id where :id matches empty string
	app.Get("/test/:id", ParsePathParametersUUID, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Use a route that will match but with an effectively empty segment
	// Note: Fiber doesn't allow truly empty path params, so we test with whitespace
	req := httptest.NewRequest("GET", "/test/%20", nil) // URL-encoded space

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestParsePathParametersUUID_MultipleCalls(t *testing.T) {
	app := setupMiddlewareTestApp()

	callCount := 0
	var lastStoredID uuid.UUID

	app.Get("/test/:id", ParsePathParametersUUID, func(c *fiber.Ctx) error {
		callCount++
		storedID := c.Locals(UUIDPathParameter)
		lastStoredID = storedID.(uuid.UUID)
		return c.SendStatus(fiber.StatusOK)
	})

	// First request
	validUUID1 := uuid.New()
	req1 := httptest.NewRequest("GET", "/test/"+validUUID1.String(), nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	resp1.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp1.StatusCode)
	assert.Equal(t, validUUID1, lastStoredID)

	// Second request with different UUID
	validUUID2 := uuid.New()
	req2 := httptest.NewRequest("GET", "/test/"+validUUID2.String(), nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	resp2.Body.Close()
	assert.Equal(t, fiber.StatusOK, resp2.StatusCode)
	assert.Equal(t, validUUID2, lastStoredID)

	assert.Equal(t, 2, callCount)
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func TestParsePathParametersUUID_UUIDVersions(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		wantCode int
	}{
		{
			name:     "UUID v4",
			uuid:     "550e8400-e29b-41d4-a716-446655440000",
			wantCode: fiber.StatusOK,
		},
		{
			name:     "UUID v7 (time-based)",
			uuid:     "01918e3c-2f3b-7a5d-9c8e-1f2a3b4c5d6e",
			wantCode: fiber.StatusOK,
		},
		{
			name:     "Nil UUID",
			uuid:     "00000000-0000-0000-0000-000000000000",
			wantCode: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupMiddlewareTestApp()

			app.Get("/test/:id", ParsePathParametersUUID, func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test/"+tt.uuid, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestMiddlewareVariables(t *testing.T) {
	// Test that the exported variables have expected values
	assert.Equal(t, "id", UUIDPathParameter)
}
