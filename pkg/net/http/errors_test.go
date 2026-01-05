package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestWithError(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		wantStatusCode     int
		wantResponseFields []string
	}{
		{
			name: "ValidationError",
			err: pkg.ValidationError{
				Code:    "TEST_001",
				Title:   "Validation Error",
				Message: "Test validation error",
			},
			wantStatusCode:     http.StatusBadRequest,
			wantResponseFields: []string{"code", "title", "message"},
		},
		{
			name: "UnprocessableOperationError",
			err: pkg.UnprocessableOperationError{
				Code:    "TEST_002",
				Title:   "Unprocessable",
				Message: "Cannot process this operation",
			},
			wantStatusCode:     http.StatusUnprocessableEntity,
			wantResponseFields: []string{"code", "title", "message"},
		},
		{
			name: "UnauthorizedError",
			err: pkg.UnauthorizedError{
				Code:    "TEST_003",
				Title:   "Unauthorized",
				Message: "Authentication required",
			},
			wantStatusCode:     http.StatusUnauthorized,
			wantResponseFields: []string{"code", "title", "message"},
		},
		{
			name: "ForbiddenError",
			err: pkg.ForbiddenError{
				Code:    "TEST_004",
				Title:   "Forbidden",
				Message: "Access denied",
			},
			wantStatusCode:     http.StatusForbidden,
			wantResponseFields: []string{"code", "title", "message"},
		},
		{
			name: "ValidationKnownFieldsError",
			err: pkg.ValidationKnownFieldsError{
				Code:    "TEST_005",
				Title:   "Field Validation Error",
				Message: "Invalid fields",
				Fields: map[string]string{
					"email": "invalid email format",
				},
			},
			wantStatusCode:     http.StatusBadRequest,
			wantResponseFields: []string{"code", "title", "message", "fields"},
		},
		{
			name: "ValidationUnknownFieldsError",
			err: pkg.ValidationUnknownFieldsError{
				Code:    "TEST_006",
				Title:   "Unknown Fields Error",
				Message: "Unknown fields present",
				Fields: map[string]any{
					"unexpected": "value",
				},
			},
			wantStatusCode:     http.StatusBadRequest,
			wantResponseFields: []string{"code", "title", "message", "fields"},
		},
		{
			name: "ResponseError",
			err: pkg.ResponseError{
				Code:    http.StatusBadRequest,
				Title:   "Bad Request",
				Message: "Invalid request",
			},
			wantStatusCode:     http.StatusBadRequest,
			wantResponseFields: []string{"code", "title", "message"},
		},
		{
			name: "ResponseErrorWithStatusCode",
			err: pkg.ResponseErrorWithStatusCode{
				StatusCode: http.StatusNotFound,
				Code:       "NOT_FOUND",
				Title:      "Not Found",
				Message:    "Resource not found",
			},
			wantStatusCode:     http.StatusNotFound,
			wantResponseFields: []string{"code", "title", "message"},
		},
		{
			name:               "Default InternalServerError",
			err:                constant.ErrInternalServer,
			wantStatusCode:     http.StatusInternalServerError,
			wantResponseFields: []string{"code", "title", "message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return WithError(c, tt.err)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatusCode, resp.StatusCode)
		})
	}
}
