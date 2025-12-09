package in

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

// setupTestApp creates a Fiber app with test context middleware.
func setupTestApp() *fiber.App {
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

// TestFetcherHandler_CreateJob_InvalidJSON tests that invalid JSON returns 400.
func TestFetcherHandler_CreateJob_InvalidJSON(t *testing.T) {
	app := setupTestApp()

	// Create handler without actual service (we expect early failure)
	handler := &FetcherHandler{CreateJobCmd: nil}

	app.Post("/v1/fetcher", handler.CreateJob)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "invalid JSON - missing closing brace",
			body:     `{"dataRequest": {"mappedFields": {"ds1": {"t1": ["f1"]}}`,
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
			req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected status %d, got %d. Body: %s", tt.wantCode, resp.StatusCode, string(body))
			}
		})
	}
}

// TestFetcherHandler_CreateJob_MissingOrgHeader tests that missing or invalid X-Organization-Id returns 400.
func TestFetcherHandler_CreateJob_MissingOrgHeader(t *testing.T) {
	app := setupTestApp()

	handler := &FetcherHandler{CreateJobCmd: nil}
	app.Post("/v1/fetcher", handler.CreateJob)

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

	validPayload := `{"dataRequest":{"mappedFields":{"ds1":{"t1":["f1"]}}}}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(validPayload))
			req.Header.Set("Content-Type", "application/json")

			if tt.setHeader {
				req.Header.Set("X-Organization-Id", tt.orgHeader)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected status %d, got %d. Body: %s", tt.wantCode, resp.StatusCode, string(body))
			}
		})
	}
}

// TestFetcherHandler_CreateJob_ContentTypeValidation tests content type handling.
func TestFetcherHandler_CreateJob_ContentTypeValidation(t *testing.T) {
	app := setupTestApp()

	handler := &FetcherHandler{CreateJobCmd: nil}
	app.Post("/v1/fetcher", handler.CreateJob)

	tests := []struct {
		name        string
		contentType string
		body        string
		wantCode    int
	}{
		{
			name:        "valid JSON content type but invalid body",
			contentType: "application/json",
			body:        `{invalid`,
			wantCode:    fiber.StatusBadRequest,
		},
		{
			name:        "JSON with charset but invalid body",
			contentType: "application/json; charset=utf-8",
			body:        `{invalid`,
			wantCode:    fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/fetcher", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			req.Header.Set("X-Organization-Id", uuid.New().String())

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected status %d, got %d. Body: %s", tt.wantCode, resp.StatusCode, string(body))
			}
		})
	}
}

// createLargePayload generates a valid JSON payload of approximately the specified size.
func createLargePayload(targetSize int) []byte {
	// Build a JSON structure with enough data to exceed targetSize
	var sb strings.Builder
	sb.WriteString(`{"dataRequest":{"mappedFields":{`)

	// Add datasources until we reach target size
	datasourceNum := 0
	for sb.Len() < targetSize-100 { // Leave room for closing braces
		if datasourceNum > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"datasource`)
		sb.WriteString(strings.Repeat("x", 50)) // Pad datasource name
		sb.WriteString(`_`)
		sb.WriteString(string(rune('0' + (datasourceNum % 10))))
		sb.WriteString(`":{"table1":["field1","field2","field3","field4","field5","field6","field7","field8","field9","field10"]}`)
		datasourceNum++
	}

	sb.WriteString(`}},"metadata":{"padding":"`)
	// Add padding to reach exact size
	remaining := targetSize - sb.Len() - 5 // Account for closing characters
	if remaining > 0 {
		sb.WriteString(strings.Repeat("x", remaining))
	}
	sb.WriteString(`"}}`)

	return []byte(sb.String())
}
