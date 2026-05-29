package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// validatedInput is a fixture with required+enum tags. Used to drive
// ValidateStruct → WithError end-to-end so we exercise the same code path
// the Manager handlers (CreateConnection, UpdateConnection, ValidateSchema,
// CreateJob) use after M6 wired ValidateStruct in.
type validatedInput struct {
	Name string `json:"name" validate:"required"`
	Type string `json:"type" validate:"required,oneof=ORACLE POSTGRESQL"`
}

// TestWithError_ValidateStructResultRendersAs400 is the regression guard for
// the M6 E2E fallout (15 E2E tests returning 500 when they expected 400).
//
// Bug shape: ValidateStruct returned `*pkg.ValidationKnownFieldsError`
// (pointer-to-struct via `return &errPtr`), but WithError's errors.As target
// was declared as the value type `pkg.ValidationKnownFieldsError`. errors.As
// requires type identity between the wrapped err and the target type — pointer
// vs value didn't match, so control fell through to the InternalServerError
// default branch (HTTP 500) instead of mapping to BadRequest (HTTP 400).
//
// Before M6 this was dead code (handlers never invoked ValidateStruct). M6
// activated the path by wiring ValidateStruct(&request) into 4 handlers, and
// the pre-existing pointer-return bug surfaced in E2E as 500s on every
// invalid-payload test.
//
// Test contract: ValidateStruct + WithError must yield HTTP 400 (not 500)
// for any struct-tag violation. Body must surface FET-0402 (the canonical
// missing-fields error code emitted by malformedRequestErr → ValidateBadRequestFieldsError).
func TestWithError_ValidateStructResultRendersAs400(t *testing.T) {
	cases := []struct {
		name     string
		input    validatedInput
		wantCode string // expected error code in JSON body
	}{
		{
			name:     "missing required field",
			input:    validatedInput{Type: "POSTGRESQL"}, // Name missing
			wantCode: constant.ErrMissingFieldsInRequest.Error(),
		},
		{
			name:     "invalid enum value",
			input:    validatedInput{Name: "x", Type: "CASSANDRA"}, // Type not in oneof
			wantCode: constant.ErrBadRequest.Error(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStruct(&tc.input)
			require.Error(t, err, "ValidateStruct must reject invalid input")

			// Drive the same handler shape: ValidateStruct error → WithError.
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return WithError(c, err)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, errResp := app.Test(req)
			require.NoError(t, errResp)

			defer resp.Body.Close()

			body, errBody := io.ReadAll(resp.Body)
			require.NoError(t, errBody)

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
				"ValidateStruct result must render as 400, not 500 (body: %s)", string(body))

			assert.True(t, strings.Contains(string(body), tc.wantCode),
				"body must surface validation error code %q, got: %s", tc.wantCode, string(body))
		})
	}
}
