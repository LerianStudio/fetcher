package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestWithRecover(t *testing.T) {
	tests := []struct {
		name           string
		handler        fiber.Handler
		wantStatusCode int
		wantPanic      bool
	}{
		{
			name: "no panic returns normally",
			handler: func(c *fiber.Ctx) error {
				return c.SendString("OK")
			},
			wantStatusCode: http.StatusOK,
			wantPanic:      false,
		},
		{
			name: "panic with string is recovered",
			handler: func(c *fiber.Ctx) error {
				panic("test panic")
			},
			wantStatusCode: http.StatusInternalServerError,
			wantPanic:      true,
		},
		{
			name: "panic with error is recovered",
			handler: func(c *fiber.Ctx) error {
				panic(fiber.NewError(fiber.StatusBadRequest, "bad request panic"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantPanic:      true,
		},
		{
			name: "panic with int value is recovered",
			handler: func(c *fiber.Ctx) error {
				panic(42)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantPanic:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(WithRecover())
			app.Get("/test", tt.handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test app: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %v, want %v", resp.StatusCode, tt.wantStatusCode)
			}

			if tt.wantPanic {
				body, _ := io.ReadAll(resp.Body)
				if len(body) == 0 {
					t.Error("Expected error response body for panic, got empty")
				}
			}
		})
	}
}

func TestWithRecoverLogger(t *testing.T) {
	opt := WithRecoverLogger(nil)
	mid := &recoverMiddleware{}
	opt(mid)

	if mid.Logger != nil {
		t.Error("Expected Logger to be nil after applying option with nil")
	}
}

func TestBuildRecoverOpts(t *testing.T) {
	t.Run("default logger is set", func(t *testing.T) {
		mid := buildRecoverOpts()
		if mid.Logger == nil {
			t.Error("Expected default Logger to be set")
		}
	})

	t.Run("custom logger is applied", func(t *testing.T) {
		mid := buildRecoverOpts(WithRecoverLogger(nil))
		if mid.Logger != nil {
			t.Error("Expected Logger to be nil when custom nil logger is provided")
		}
	})
}
