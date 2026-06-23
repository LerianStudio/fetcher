package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestUnauthorized(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		title   string
		message string
	}{
		{
			name:    "standard unauthorized",
			code:    "AUTH_001",
			title:   "Unauthorized",
			message: "Authentication required",
		},
		{
			name:    "empty strings",
			code:    "",
			title:   "",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return Unauthorized(c, tt.code, tt.title, tt.message)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

			var result map[string]string
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)

			assert.Equal(t, tt.code, result["code"])
			assert.Equal(t, tt.title, result["title"])
			assert.Equal(t, tt.message, result["message"])
		})
	}
}

func TestForbidden(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		title   string
		message string
	}{
		{
			name:    "standard forbidden",
			code:    "FORB_001",
			title:   "Forbidden",
			message: "Access denied",
		},
		{
			name:    "empty strings",
			code:    "",
			title:   "",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return Forbidden(c, tt.code, tt.title, tt.message)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusForbidden, resp.StatusCode)

			var result map[string]string
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)

			assert.Equal(t, tt.code, result["code"])
			assert.Equal(t, tt.title, result["title"])
			assert.Equal(t, tt.message, result["message"])
		})
	}
}

func TestBadRequest(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{
			name:    "with map",
			payload: fiber.Map{"error": "invalid input"},
		},
		{
			name:    "with struct",
			payload: struct{ Error string }{"validation failed"},
		},
		{
			name:    "with string",
			payload: "bad request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return BadRequest(c, tt.payload)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestCreated(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{
			name:    "with map",
			payload: fiber.Map{"id": "123", "name": "test"},
		},
		{
			name:    "with struct",
			payload: struct{ ID string }{"456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/test", func(c *fiber.Ctx) error {
				return Created(c, tt.payload)
			})

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		})
	}
}

func TestOK(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{
			name:    "with map",
			payload: fiber.Map{"status": "success"},
		},
		{
			name:    "with array",
			payload: []string{"item1", "item2"},
		},
		{
			name:    "with nil",
			payload: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return OK(c, tt.payload)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestNoContent(t *testing.T) {
	app := fiber.New()
	app.Delete("/test", func(c *fiber.Ctx) error {
		return NoContent(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/test", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Empty(t, body)
}

func TestAccepted(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{
			name:    "with map",
			payload: fiber.Map{"message": "processing"},
		},
		{
			name:    "with empty map",
			payload: fiber.Map{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/test", func(c *fiber.Ctx) error {
				return Accepted(c, tt.payload)
			})

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		})
	}
}

func TestPartialContent(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{
			name:    "with map",
			payload: fiber.Map{"data": []int{1, 2, 3}},
		},
		{
			name:    "with struct",
			payload: struct{ Items []string }{[]string{"a", "b"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return PartialContent(c, tt.payload)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusPartialContent, resp.StatusCode)
		})
	}
}

func TestRangeNotSatisfiable(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return RangeNotSatisfiable(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusRequestedRangeNotSatisfiable, resp.StatusCode)
}

func TestNotFound(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		title   string
		message string
	}{
		{
			name:    "standard not found",
			code:    "NOT_FOUND",
			title:   "Not Found",
			message: "Resource not found",
		},
		{
			name:    "custom not found",
			code:    "USER_NOT_FOUND",
			title:   "User Not Found",
			message: "The requested user does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return NotFound(c, tt.code, tt.title, tt.message)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)

			var result map[string]string
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)

			assert.Equal(t, tt.code, result["code"])
			assert.Equal(t, tt.title, result["title"])
			assert.Equal(t, tt.message, result["message"])
		})
	}
}

func TestConflict(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		title   string
		message string
	}{
		{
			name:    "standard conflict",
			code:    "CONFLICT",
			title:   "Conflict",
			message: "Resource already exists",
		},
		{
			name:    "duplicate entry",
			code:    "DUPLICATE_ENTRY",
			title:   "Duplicate Entry",
			message: "Email already in use",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/test", func(c *fiber.Ctx) error {
				return Conflict(c, tt.code, tt.title, tt.message)
			})

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusConflict, resp.StatusCode)

			var result map[string]string
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)

			assert.Equal(t, tt.code, result["code"])
			assert.Equal(t, tt.title, result["title"])
			assert.Equal(t, tt.message, result["message"])
		})
	}
}

func TestNotImplemented(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "standard not implemented",
			message: "This feature is not yet implemented",
		},
		{
			name:    "empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return NotImplemented(c, tt.message)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)

			var result map[string]any
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)

			assert.Equal(t, float64(http.StatusNotImplemented), result["code"])
			assert.Equal(t, "Not Implemented", result["title"])
			assert.Equal(t, tt.message, result["message"])
		})
	}
}

func TestUnprocessableEntity(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		title   string
		message string
	}{
		{
			name:    "standard unprocessable",
			code:    "UNPROCESSABLE",
			title:   "Unprocessable Entity",
			message: "Cannot process the request",
		},
		{
			name:    "validation error",
			code:    "VALIDATION_ERROR",
			title:   "Validation Failed",
			message: "The provided data is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/test", func(c *fiber.Ctx) error {
				return UnprocessableEntity(c, tt.code, tt.title, tt.message)
			})

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

			var result map[string]string
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)

			assert.Equal(t, tt.code, result["code"])
			assert.Equal(t, tt.title, result["title"])
			assert.Equal(t, tt.message, result["message"])
		})
	}
}

func TestInternalServerError(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		title   string
		message string
	}{
		{
			name:    "standard internal error",
			code:    "INTERNAL_ERROR",
			title:   "Internal Server Error",
			message: "An unexpected error occurred",
		},
		{
			name:    "database error",
			code:    "DB_ERROR",
			title:   "Database Error",
			message: "Failed to connect to database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return InternalServerError(c, tt.code, tt.title, tt.message)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

			var result map[string]string
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)

			assert.Equal(t, tt.code, result["code"])
			assert.Equal(t, tt.title, result["title"])
			assert.Equal(t, tt.message, result["message"])
		})
	}
}

func TestJSONResponseError(t *testing.T) {
	tests := []struct {
		name string
		err  pkg.ResponseError
	}{
		{
			name: "standard response error",
			err: pkg.ResponseError{
				Code:    http.StatusBadRequest,
				Title:   "Bad Request",
				Message: "Invalid input",
			},
		},
		{
			name: "not found error",
			err: pkg.ResponseError{
				Code:    http.StatusNotFound,
				Title:   "Not Found",
				Message: "Resource not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return JSONResponseError(c, tt.err)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.err.Code, resp.StatusCode)

			var result pkg.ResponseError
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)

			assert.Equal(t, tt.err.Code, result.Code)
			assert.Equal(t, tt.err.Title, result.Title)
			assert.Equal(t, tt.err.Message, result.Message)
		})
	}
}

func TestJSONResponseErrorWithStatusCode(t *testing.T) {
	tests := []struct {
		name string
		err  pkg.ResponseErrorWithStatusCode
	}{
		{
			name: "not found with status code",
			err: pkg.ResponseErrorWithStatusCode{
				StatusCode: http.StatusNotFound,
				Code:       "NOT_FOUND",
				Title:      "Not Found",
				Message:    "Resource not found",
			},
		},
		{
			name: "conflict with status code",
			err: pkg.ResponseErrorWithStatusCode{
				StatusCode: http.StatusConflict,
				Code:       "CONFLICT",
				Title:      "Conflict",
				Message:    "Resource already exists",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return JSONResponseErrorWithStatusCode(c, tt.err)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.err.StatusCode, resp.StatusCode)

			var result map[string]string
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &result)

			assert.Equal(t, tt.err.Code, result["code"])
			assert.Equal(t, tt.err.Title, result["title"])
			assert.Equal(t, tt.err.Message, result["message"])
		})
	}
}

func TestJSONResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		payload    any
	}{
		{
			name:       "custom 200 with map",
			statusCode: http.StatusOK,
			payload:    fiber.Map{"status": "success"},
		},
		{
			name:       "custom 201 with struct",
			statusCode: http.StatusCreated,
			payload:    struct{ ID int }{123},
		},
		{
			name:       "custom 204 with nil",
			statusCode: http.StatusNoContent,
			payload:    nil,
		},
		{
			name:       "custom 404 with error",
			statusCode: http.StatusNotFound,
			payload:    fiber.Map{"error": "not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return JSONResponse(c, tt.statusCode, tt.payload)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}
