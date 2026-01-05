package in

import (
	"context"
	"net/http/httptest"
	"testing"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
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
// ParseHeaderParameters Middleware Tests
// ============================================================================

func TestParseHeaderParameters_ValidOrgID(t *testing.T) {
	app := setupMiddlewareTestApp()
	validOrgID := uuid.New()

	app.Get("/test", ParseHeaderParameters, func(c *fiber.Ctx) error {
		storedOrgID := c.Locals(OrgIDHeaderParameter)
		assert.NotNil(t, storedOrgID)
		assert.Equal(t, validOrgID, storedOrgID)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Organization-Id", validOrgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestParseHeaderParameters_MissingHeader(t *testing.T) {
	app := setupMiddlewareTestApp()

	app.Get("/test", ParseHeaderParameters, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// Not setting X-Organization-Id header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestParseHeaderParameters_InvalidOrgID(t *testing.T) {
	tests := []struct {
		name     string
		orgID    string
		wantCode int
	}{
		{
			name:     "invalid UUID format - random string",
			orgID:    "not-a-uuid",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - partial UUID",
			orgID:    "550e8400-e29b",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - empty string",
			orgID:    "",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - whitespace only",
			orgID:    "   ",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - special characters",
			orgID:    "550e8400-e29b-<script>",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - alphanumeric mix",
			orgID:    "abc123def456",
			wantCode: fiber.StatusBadRequest,
		},
		{
			name:     "invalid UUID format - with extra hyphen",
			orgID:    "550e8400-e29b-41d4-a716-446655440000-extra",
			wantCode: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupMiddlewareTestApp()

			app.Get("/test", ParseHeaderParameters, func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Organization-Id", tt.orgID)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

func TestParseHeaderParameters_CaseInsensitiveHeaderName(t *testing.T) {
	app := setupMiddlewareTestApp()
	validOrgID := uuid.New()

	app.Get("/test", ParseHeaderParameters, func(c *fiber.Ctx) error {
		storedOrgID := c.Locals(OrgIDHeaderParameter)
		assert.NotNil(t, storedOrgID)
		return c.SendStatus(fiber.StatusOK)
	})

	// HTTP headers are case-insensitive
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("x-organization-id", validOrgID.String()) // lowercase

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestParseHeaderParameters_MultipleCalls(t *testing.T) {
	app := setupMiddlewareTestApp()

	callCount := 0

	app.Get("/test", ParseHeaderParameters, func(c *fiber.Ctx) error {
		callCount++
		storedOrgID := c.Locals(OrgIDHeaderParameter)
		assert.NotNil(t, storedOrgID)
		return c.SendStatus(fiber.StatusOK)
	})

	// First request
	orgID1 := uuid.New()
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-Organization-Id", orgID1.String())
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	resp1.Body.Close()

	// Second request with different org ID
	orgID2 := uuid.New()
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Organization-Id", orgID2.String())
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	resp2.Body.Close()

	assert.Equal(t, 2, callCount)
}

// ============================================================================
// Middleware Chain Tests
// ============================================================================

func TestMiddlewareChain_BothMiddlewares(t *testing.T) {
	app := setupMiddlewareTestApp()
	validOrgID := uuid.New()
	validResourceID := uuid.New()

	app.Get("/test/:id", ParseHeaderParameters, ParsePathParametersUUID, func(c *fiber.Ctx) error {
		// Verify both values are stored
		storedOrgID := c.Locals(OrgIDHeaderParameter)
		storedResourceID := c.Locals(UUIDPathParameter)

		assert.Equal(t, validOrgID, storedOrgID)
		assert.Equal(t, validResourceID, storedResourceID)

		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test/"+validResourceID.String(), nil)
	req.Header.Set("X-Organization-Id", validOrgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestMiddlewareChain_HeaderMiddlewareFails(t *testing.T) {
	app := setupMiddlewareTestApp()
	validResourceID := uuid.New()

	app.Get("/test/:id", ParseHeaderParameters, ParsePathParametersUUID, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test/"+validResourceID.String(), nil)
	// Missing X-Organization-Id header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should fail at header middleware
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestMiddlewareChain_PathMiddlewareFails(t *testing.T) {
	app := setupMiddlewareTestApp()
	validOrgID := uuid.New()

	app.Get("/test/:id", ParseHeaderParameters, ParsePathParametersUUID, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test/invalid-uuid", nil)
	req.Header.Set("X-Organization-Id", validOrgID.String())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should fail at path parameter middleware
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
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

func TestParseHeaderParameters_UUIDWithWhitespace(t *testing.T) {
	app := setupMiddlewareTestApp()
	validOrgID := uuid.New()

	app.Get("/test", ParseHeaderParameters, func(c *fiber.Ctx) error {
		storedOrgID := c.Locals(OrgIDHeaderParameter)
		assert.NotNil(t, storedOrgID)
		return c.SendStatus(fiber.StatusOK)
	})

	// UUID with leading/trailing whitespace
	// Note: google/uuid.Parse may trim whitespace internally
	// so this test verifies the actual behavior
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Organization-Id", "  "+validOrgID.String()+"  ")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The google/uuid.Parse function trims whitespace, so this will succeed
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ============================================================================
// Variables Tests
// ============================================================================

func TestMiddlewareVariables(t *testing.T) {
	// Test that the exported variables have expected values
	assert.Equal(t, "id", UUIDPathParameter)
	assert.Equal(t, "X-Organization-Id", OrgIDHeaderParameter)
}
