package http

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/lib-commons/v4/commons"
	"github.com/gofiber/fiber/v2"
)

// setRequestIDInContext sets the HeaderID (request ID) in the context using the commons library pattern
func setRequestIDInContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, commons.CustomContextKey, &commons.CustomContextKeyValue{
		HeaderID: requestID,
	})
}

func TestRequestIDHeader(t *testing.T) {
	t.Run("adds request ID to response header with default name", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequestIDHeader(""))
		app.Get("/test", func(c *fiber.Ctx) error {
			// Set a request ID in context
			ctx := setRequestIDInContext(c.UserContext(), "req-789")
			c.SetUserContext(ctx)
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		requestID := resp.Header.Get("X-Request-Id")
		if requestID != "req-789" {
			t.Fatalf("expected X-Request-Id 'req-789', got '%s'", requestID)
		}
	})

	t.Run("adds request ID to response header with custom name", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequestIDHeader("X-Custom-Request-Id"))
		app.Get("/test", func(c *fiber.Ctx) error {
			ctx := setRequestIDInContext(c.UserContext(), "custom-req-id")
			c.SetUserContext(ctx)
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		requestID := resp.Header.Get("X-Custom-Request-Id")
		if requestID != "custom-req-id" {
			t.Fatalf("expected X-Custom-Request-Id 'custom-req-id', got '%s'", requestID)
		}
	})

	t.Run("generates UUID when no request ID in context", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequestIDHeader(""))
		app.Get("/test", func(c *fiber.Ctx) error {
			// Don't set any tracking context - lib will generate UUID
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		requestID := resp.Header.Get("X-Request-Id")
		// When no request ID is set, the library generates a new UUID
		if requestID == "" {
			t.Fatal("expected X-Request-Id to be generated, got empty")
		}
		// Verify it looks like a UUID (36 chars with dashes)
		if len(requestID) != 36 {
			t.Fatalf("expected UUID format (36 chars), got '%s' with len %d", requestID, len(requestID))
		}
	})

	t.Run("propagates handler error", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequestIDHeader(""))
		app.Get("/test", func(c *fiber.Ctx) error {
			return fiber.ErrBadRequest
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("works with context that has no tracking", func(t *testing.T) {
		app := fiber.New()

		app.Use(RequestIDHeader(""))
		app.Get("/test", func(c *fiber.Ctx) error {
			// Set a plain context without tracking
			c.SetUserContext(context.Background())
			return c.SendString("ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "ok" {
			t.Fatalf("expected body 'ok', got '%s'", string(body))
		}
	})
}
