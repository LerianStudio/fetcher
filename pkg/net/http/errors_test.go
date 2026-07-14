package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/lib-commons/v5/commons/net/http/problem"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithError_RendersCodedRFC9457Problem(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return WithError(c, pkg.ValidationError{
			Code:    constant.ErrInvalidPathParameter.Error(),
			Title:   "Invalid Path Parameter",
			Message: "invalid connection id",
		})
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get(fiber.HeaderContentType))

	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, http.StatusBadRequest, detail.Status)
	assert.Equal(t, http.StatusText(http.StatusBadRequest), detail.Title)
	assert.Equal(t, "invalid connection id", detail.Detail)
	assert.Equal(t, constant.ErrInvalidPathParameter.Error(), detail.Code)
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrInvalidPathParameter.Error(), detail.Type)
}

func TestMapError_ReturnsCodedProblemForHumaHandlers(t *testing.T) {
	t.Parallel()

	mapped := MapError(pkg.ResponseErrorWithStatusCode{
		StatusCode: http.StatusConflict,
		Code:       constant.ErrEntityConflict.Error(),
		Message:    "connection already exists",
	})

	detail, ok := mapped.(*problem.Detail)
	require.True(t, ok)
	assert.Equal(t, http.StatusConflict, detail.Status)
	assert.Equal(t, http.StatusText(http.StatusConflict), detail.Title)
	assert.Equal(t, constant.ErrEntityConflict.Error(), detail.Code)
	assert.Equal(t, "connection already exists", detail.Detail)
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrEntityConflict.Error(), detail.Type)
	assert.Empty(t, detail.Errors)
}

func TestMapError_PreservesAllowlistedResponseError5xxTitles(t *testing.T) {
	t.Parallel()

	for _, title := range []string{"Database Connection Error", "Schema Retrieval Error"} {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			mapped := MapError(pkg.ResponseError{
				Code:    http.StatusInternalServerError,
				Title:   title,
				Message: "safe public message",
			})

			detail, ok := mapped.(*problem.Detail)
			require.True(t, ok)
			assert.Equal(t, http.StatusInternalServerError, detail.Status)
			assert.Equal(t, title, detail.Title)
			assert.Equal(t, "internal error", detail.Detail)
			assert.Equal(t, "500", detail.Code)
			assert.Equal(t, problem.BaseURI+"/500", detail.Type)
			assert.Empty(t, detail.Errors)
		})
	}
}

func TestMapError_ScrubsUntrustedResponseError5xxTitle(t *testing.T) {
	t.Parallel()

	mapped := MapError(pkg.ResponseError{
		Code:    http.StatusInternalServerError,
		Title:   "password=super-secret",
		Message: "host=db.internal password=super-secret",
	})

	detail, ok := mapped.(*problem.Detail)
	require.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, detail.Status)
	assert.Equal(t, http.StatusText(http.StatusInternalServerError), detail.Title)
	assert.Equal(t, "internal error", detail.Detail)
	assert.Equal(t, "500", detail.Code)
	assert.Equal(t, problem.BaseURI+"/500", detail.Type)
	assert.NotContains(t, detail.Title, "super-secret")
	assert.NotContains(t, detail.Detail, "super-secret")
	assert.Empty(t, detail.Errors)
}

func TestMapError_PreservesValidationFieldDetails(t *testing.T) {
	t.Parallel()

	t.Run("known fields", func(t *testing.T) {
		mapped := MapError(pkg.ValidationKnownFieldsError{
			Code:    constant.ErrBadRequest.Error(),
			Message: "request validation failed",
			Fields: pkg.FieldValidations{
				"port": "must be a number",
				"host": "is required",
			},
		})

		detail, ok := mapped.(*problem.Detail)
		require.True(t, ok)
		require.Len(t, detail.Errors, 2)
		assert.Equal(t, "body.host", detail.Errors[0].Location)
		assert.Equal(t, "is required", detail.Errors[0].Message)
		assert.Equal(t, "body.port", detail.Errors[1].Location)
		assert.Equal(t, "must be a number", detail.Errors[1].Message)
	})

	t.Run("unknown fields", func(t *testing.T) {
		mapped := MapError(pkg.ValidationUnknownFieldsError{
			Code:    constant.ErrUnexpectedFieldsInTheRequest.Error(),
			Message: "unexpected fields",
			Fields: pkg.UnknownFields{
				"surprise": "value",
			},
		})

		detail, ok := mapped.(*problem.Detail)
		require.True(t, ok)
		require.Len(t, detail.Errors, 1)
		assert.Equal(t, "body.surprise", detail.Errors[0].Location)
		assert.Equal(t, "unexpected field", detail.Errors[0].Message)
		assert.Equal(t, "value", detail.Errors[0].Value)
	})
}

func TestWithError_ScrubsUnrecognizedInternalError(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return WithError(c, errors.New("secret database address"))
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

	var detail problem.Detail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, http.StatusInternalServerError, detail.Status)
	assert.Equal(t, http.StatusText(http.StatusInternalServerError), detail.Title)
	assert.Equal(t, "internal error", detail.Detail)
	assert.Equal(t, constant.ErrInternalServer.Error(), detail.Code)
	assert.Equal(t, problem.BaseURI+"/"+constant.ErrInternalServer.Error(), detail.Type)
	assert.Nil(t, detail.Errors)
}

func TestWithError_PreservesHistoricallyUnmappedErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]error{
		"HTTPError": pkg.HTTPError{
			Code:    "UPSTREAM",
			Message: "upstream leaked detail",
		},
		"FailedPreconditionError": pkg.FailedPreconditionError{
			Code:    "PRECONDITION",
			Message: "precondition leaked detail",
		},
		"FET-1062": constant.ErrSchemaValidationNotFound,
	}

	for name, mappedErr := range tests {
		mappedErr := mappedErr
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error { return WithError(c, mappedErr) })
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

			var detail problem.Detail
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			assert.Equal(t, "application/problem+json", resp.Header.Get(fiber.HeaderContentType))
			assert.Equal(t, http.StatusInternalServerError, detail.Status)
			assert.Equal(t, http.StatusText(http.StatusInternalServerError), detail.Title)
			assert.Equal(t, "internal error", detail.Detail)
			assert.Equal(t, constant.ErrInternalServer.Error(), detail.Code)
			assert.Equal(t, problem.BaseURI+"/"+constant.ErrInternalServer.Error(), detail.Type)
			assert.Empty(t, detail.Errors)
		})
	}
}

func TestWithError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantTitle  string
		wantDetail string
		wantCode   string
		wantErrors []*huma.ErrorDetail
	}{
		{
			name: "ValidationError",
			err: pkg.ValidationError{
				Code:    "TEST_001",
				Title:   "Validation Error",
				Message: "Test validation error",
			},
			wantStatus: http.StatusBadRequest,
			wantTitle:  http.StatusText(http.StatusBadRequest),
			wantDetail: "Test validation error",
			wantCode:   "TEST_001",
		},
		{
			name: "UnprocessableOperationError",
			err: pkg.UnprocessableOperationError{
				Code:    "TEST_002",
				Title:   "Unprocessable",
				Message: "Cannot process this operation",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantTitle:  http.StatusText(http.StatusUnprocessableEntity),
			wantDetail: "Cannot process this operation",
			wantCode:   "TEST_002",
		},
		{
			name: "UnauthorizedError",
			err: pkg.UnauthorizedError{
				Code:    "TEST_003",
				Title:   "Unauthorized",
				Message: "Authentication required",
			},
			wantStatus: http.StatusUnauthorized,
			wantTitle:  http.StatusText(http.StatusUnauthorized),
			wantDetail: "Authentication required",
			wantCode:   "TEST_003",
		},
		{
			name: "ForbiddenError",
			err: pkg.ForbiddenError{
				Code:    "TEST_004",
				Title:   "Forbidden",
				Message: "Access denied",
			},
			wantStatus: http.StatusForbidden,
			wantTitle:  http.StatusText(http.StatusForbidden),
			wantDetail: "Access denied",
			wantCode:   "TEST_004",
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
			wantStatus: http.StatusBadRequest,
			wantTitle:  http.StatusText(http.StatusBadRequest),
			wantDetail: "Invalid fields",
			wantCode:   "TEST_005",
			wantErrors: []*huma.ErrorDetail{{
				Location: "body.email",
				Message:  "invalid email format",
			}},
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
			wantStatus: http.StatusBadRequest,
			wantTitle:  http.StatusText(http.StatusBadRequest),
			wantDetail: "Unknown fields present",
			wantCode:   "TEST_006",
			wantErrors: []*huma.ErrorDetail{{
				Location: "body.unexpected",
				Message:  "unexpected field",
				Value:    "value",
			}},
		},
		{
			name: "ResponseError",
			err: pkg.ResponseError{
				Code:    http.StatusBadRequest,
				Title:   "Bad Request",
				Message: "Invalid request",
			},
			wantStatus: http.StatusBadRequest,
			wantTitle:  http.StatusText(http.StatusBadRequest),
			wantDetail: "Invalid request",
			wantCode:   "400",
		},
		{
			name: "ResponseErrorWithStatusCode",
			err: pkg.ResponseErrorWithStatusCode{
				StatusCode: http.StatusNotFound,
				Code:       "NOT_FOUND",
				Title:      "Not Found",
				Message:    "Resource not found",
			},
			wantStatus: http.StatusNotFound,
			wantTitle:  http.StatusText(http.StatusNotFound),
			wantDetail: "Resource not found",
			wantCode:   "NOT_FOUND",
		},
		{
			name:       "Default InternalServerError",
			err:        constant.ErrInternalServer,
			wantStatus: http.StatusInternalServerError,
			wantTitle:  http.StatusText(http.StatusInternalServerError),
			wantDetail: "internal error",
			wantCode:   constant.ErrInternalServer.Error(),
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
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			assert.Equal(t, "application/problem+json", resp.Header.Get(fiber.HeaderContentType))

			var detail problem.Detail
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&detail))
			assert.Equal(t, tt.wantStatus, detail.Status)
			assert.Equal(t, tt.wantTitle, detail.Title)
			assert.Equal(t, tt.wantDetail, detail.Detail)
			assert.Equal(t, tt.wantCode, detail.Code)
			assert.Equal(t, problem.BaseURI+"/"+tt.wantCode, detail.Type)
			assert.Equal(t, tt.wantErrors, detail.Errors)
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

			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

			body, errBody := io.ReadAll(resp.Body)
			require.NoError(t, errBody)

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
				"ValidateStruct result must render as 400, not 500 (body: %s)", string(body))

			assert.True(t, strings.Contains(string(body), tc.wantCode),
				"body must surface validation error code %q, got: %s", tc.wantCode, string(body))
		})
	}
}
