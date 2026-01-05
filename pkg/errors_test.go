package pkg

import (
	"errors"
	"net/http"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  ValidationError
		want string
	}{
		{
			name: "with code and message",
			err: ValidationError{
				Code:    "VALIDATION_001",
				Message: "Field is required",
			},
			want: "VALIDATION_001 - Field is required",
		},
		{
			name: "with message only",
			err: ValidationError{
				Message: "Field is required",
			},
			want: "Field is required",
		},
		{
			name: "with whitespace code",
			err: ValidationError{
				Code:    "   ",
				Message: "Field is required",
			},
			want: "Field is required",
		},
		{
			name: "empty message",
			err: ValidationError{
				Code: "ERR_001",
			},
			want: "ERR_001 - ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := ValidationError{
		Message: "outer error",
		Err:     innerErr,
	}

	if got := err.Unwrap(); got != innerErr {
		t.Errorf("ValidationError.Unwrap() = %v, want %v", got, innerErr)
	}
}

func TestUnauthorizedError_Error(t *testing.T) {
	err := UnauthorizedError{Message: "Access denied"}
	if got := err.Error(); got != "Access denied" {
		t.Errorf("UnauthorizedError.Error() = %v, want %v", got, "Access denied")
	}
}

func TestForbiddenError_Error(t *testing.T) {
	err := ForbiddenError{Message: "Not allowed"}
	if got := err.Error(); got != "Not allowed" {
		t.Errorf("ForbiddenError.Error() = %v, want %v", got, "Not allowed")
	}
}

func TestUnprocessableOperationError_Error(t *testing.T) {
	err := UnprocessableOperationError{Message: "Cannot process"}
	if got := err.Error(); got != "Cannot process" {
		t.Errorf("UnprocessableOperationError.Error() = %v, want %v", got, "Cannot process")
	}
}

func TestHTTPError_Error(t *testing.T) {
	err := HTTPError{Message: "HTTP failed"}
	if got := err.Error(); got != "HTTP failed" {
		t.Errorf("HTTPError.Error() = %v, want %v", got, "HTTP failed")
	}
}

func TestFailedPreconditionError_Error(t *testing.T) {
	err := FailedPreconditionError{Message: "Precondition failed"}
	if got := err.Error(); got != "Precondition failed" {
		t.Errorf("FailedPreconditionError.Error() = %v, want %v", got, "Precondition failed")
	}
}

func TestInternalServerError_Error(t *testing.T) {
	err := InternalServerError{Message: "Server error"}
	if got := err.Error(); got != "Server error" {
		t.Errorf("InternalServerError.Error() = %v, want %v", got, "Server error")
	}
}

func TestResponseError_Error(t *testing.T) {
	err := ResponseError{Message: "Response error"}
	if got := err.Error(); got != "Response error" {
		t.Errorf("ResponseError.Error() = %v, want %v", got, "Response error")
	}
}

func TestResponseErrorWithStatusCode_Error(t *testing.T) {
	err := ResponseErrorWithStatusCode{Message: "Status error"}
	if got := err.Error(); got != "Status error" {
		t.Errorf("ResponseErrorWithStatusCode.Error() = %v, want %v", got, "Status error")
	}
}

func TestValidationKnownFieldsError_Error(t *testing.T) {
	err := ValidationKnownFieldsError{Message: "Known fields error"}
	if got := err.Error(); got != "Known fields error" {
		t.Errorf("ValidationKnownFieldsError.Error() = %v, want %v", got, "Known fields error")
	}
}

func TestValidationUnknownFieldsError_Error(t *testing.T) {
	err := ValidationUnknownFieldsError{Message: "Unknown fields error"}
	if got := err.Error(); got != "Unknown fields error" {
		t.Errorf("ValidationUnknownFieldsError.Error() = %v, want %v", got, "Unknown fields error")
	}
}

func TestValidateInternalError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		entityType string
		wantType   string
	}{
		{
			name:       "internal server error",
			err:        constant.ErrInternalServer,
			entityType: "test",
			wantType:   "InternalServerError",
		},
		{
			name:       "bad request error",
			err:        constant.ErrBadRequest,
			entityType: "test",
			wantType:   "ValidationError",
		},
		{
			name:       "service unavailable error",
			err:        constant.ErrServiceUnavailable,
			entityType: "test",
			wantType:   "ResponseErrorWithStatusCode",
		},
		{
			name:       "conflict error",
			err:        constant.ErrConflict,
			entityType: "test",
			wantType:   "ResponseErrorWithStatusCode",
		},
		{
			name:       "not found error",
			err:        constant.ErrNotFound,
			entityType: "test",
			wantType:   "ResponseErrorWithStatusCode",
		},
		{
			name:       "unknown error defaults to internal server error",
			err:        errors.New("unknown error"),
			entityType: "test",
			wantType:   "InternalServerError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateInternalError(tt.err, tt.entityType)
			switch tt.wantType {
			case "InternalServerError":
				if _, ok := got.(InternalServerError); !ok {
					t.Errorf("ValidateInternalError() type = %T, want InternalServerError", got)
				}
			case "ValidationError":
				if _, ok := got.(ValidationError); !ok {
					t.Errorf("ValidateInternalError() type = %T, want ValidationError", got)
				}
			case "ResponseErrorWithStatusCode":
				if _, ok := got.(ResponseErrorWithStatusCode); !ok {
					t.Errorf("ValidateInternalError() type = %T, want ResponseErrorWithStatusCode", got)
				}
			}
		})
	}
}

func TestValidateBadRequestFieldsError(t *testing.T) {
	tests := []struct {
		name               string
		requiredFields     map[string]string
		knownInvalidFields map[string]string
		entityType         string
		unknownFields      map[string]any
		wantType           string
		wantCode           string
	}{
		{
			name:               "unknown fields present",
			requiredFields:     nil,
			knownInvalidFields: nil,
			entityType:         "test",
			unknownFields:      map[string]any{"extra": "value"},
			wantType:           "ValidationUnknownFieldsError",
			wantCode:           constant.ErrUnexpectedFieldsInTheRequest.Error(),
		},
		{
			name:               "required fields missing",
			requiredFields:     map[string]string{"name": "name is required"},
			knownInvalidFields: nil,
			entityType:         "test",
			unknownFields:      nil,
			wantType:           "ValidationKnownFieldsError",
			wantCode:           constant.ErrMissingFieldsInRequest.Error(),
		},
		{
			name:               "known invalid fields",
			requiredFields:     nil,
			knownInvalidFields: map[string]string{"email": "invalid email format"},
			entityType:         "test",
			unknownFields:      nil,
			wantType:           "ValidationKnownFieldsError",
			wantCode:           constant.ErrBadRequest.Error(),
		},
		{
			name:               "all empty returns error",
			requiredFields:     nil,
			knownInvalidFields: nil,
			entityType:         "test",
			unknownFields:      nil,
			wantType:           "error",
			wantCode:           "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateBadRequestFieldsError(tt.requiredFields, tt.knownInvalidFields, tt.entityType, tt.unknownFields)
			if got == nil {
				t.Error("ValidateBadRequestFieldsError() returned nil")
				return
			}

			switch tt.wantType {
			case "ValidationUnknownFieldsError":
				if e, ok := got.(ValidationUnknownFieldsError); ok {
					if e.Code != tt.wantCode {
						t.Errorf("Code = %v, want %v", e.Code, tt.wantCode)
					}
				} else {
					t.Errorf("type = %T, want ValidationUnknownFieldsError", got)
				}
			case "ValidationKnownFieldsError":
				if e, ok := got.(ValidationKnownFieldsError); ok {
					if e.Code != tt.wantCode {
						t.Errorf("Code = %v, want %v", e.Code, tt.wantCode)
					}
				} else {
					t.Errorf("type = %T, want ValidationKnownFieldsError", got)
				}
			case "error":
				// Just verify it's an error
				if got == nil {
					t.Error("expected error, got nil")
				}
			}
		})
	}
}

func TestValidateBusinessError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		entityType string
		args       []any
		wantType   string
	}{
		{
			name:       "invalid query parameter",
			err:        constant.ErrInvalidQueryParameter,
			entityType: "test",
			args:       []any{"limit"},
			wantType:   "ValidationError",
		},
		{
			name:       "entity not found",
			err:        constant.ErrEntityNotFound,
			entityType: "connection",
			args:       nil,
			wantType:   "ResponseErrorWithStatusCode",
		},
		{
			name:       "entity conflict",
			err:        constant.ErrEntityConflict,
			entityType: "connection",
			args:       nil,
			wantType:   "ResponseErrorWithStatusCode",
		},
		{
			name:       "unknown error returns original",
			err:        errors.New("custom error"),
			entityType: "test",
			args:       nil,
			wantType:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateBusinessError(tt.err, tt.entityType, tt.args...)
			if got == nil {
				t.Error("ValidateBusinessError() returned nil")
				return
			}

			switch tt.wantType {
			case "ValidationError":
				if _, ok := got.(ValidationError); !ok {
					t.Errorf("type = %T, want ValidationError", got)
				}
			case "ResponseErrorWithStatusCode":
				if e, ok := got.(ResponseErrorWithStatusCode); ok {
					if e.StatusCode != http.StatusNotFound && e.StatusCode != http.StatusConflict {
						t.Errorf("StatusCode = %v, want 404 or 409", e.StatusCode)
					}
				} else {
					t.Errorf("type = %T, want ResponseErrorWithStatusCode", got)
				}
			}
		})
	}
}

func TestValidateBusinessError_AllErrorTypes(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		entityType     string
		args           []any
		wantType       string
		wantCode       string
		wantStatusCode int
	}{
		// Date validation errors
		{
			name:       "invalid date format",
			err:        constant.ErrInvalidDateFormat,
			entityType: "job",
			args:       nil,
			wantType:   "ValidationError",
			wantCode:   constant.ErrInvalidDateFormat.Error(),
		},
		{
			name:       "invalid final date",
			err:        constant.ErrInvalidFinalDate,
			entityType: "job",
			args:       nil,
			wantType:   "ValidationError",
			wantCode:   constant.ErrInvalidFinalDate.Error(),
		},
		// Pagination errors
		{
			name:       "pagination limit exceeded",
			err:        constant.ErrPaginationLimitExceeded,
			entityType: "list",
			args:       []any{100},
			wantType:   "ValidationError",
			wantCode:   constant.ErrPaginationLimitExceeded.Error(),
		},
		{
			name:       "invalid sort order",
			err:        constant.ErrInvalidSortOrder,
			entityType: "list",
			args:       nil,
			wantType:   "ValidationError",
			wantCode:   constant.ErrInvalidSortOrder.Error(),
		},
		// Metadata errors
		{
			name:       "metadata key length exceeded",
			err:        constant.ErrMetadataKeyLengthExceeded,
			entityType: "connection",
			args:       []any{"very_long_key", 50},
			wantType:   "ValidationError",
			wantCode:   constant.ErrMetadataKeyLengthExceeded.Error(),
		},
		{
			name:       "metadata value length exceeded",
			err:        constant.ErrMetadataValueLengthExceeded,
			entityType: "connection",
			args:       []any{"value", 100},
			wantType:   "ValidationError",
			wantCode:   constant.ErrMetadataValueLengthExceeded.Error(),
		},
		{
			name:       "invalid metadata nesting",
			err:        constant.ErrInvalidMetadataNesting,
			entityType: "connection",
			args:       []any{"nested.key"},
			wantType:   "ValidationError",
			wantCode:   constant.ErrInvalidMetadataNesting.Error(),
		},
		// Header and path errors
		{
			name:       "invalid header parameter",
			err:        constant.ErrInvalidHeaderParameter,
			entityType: "request",
			args:       []any{"Authorization"},
			wantType:   "ValidationError",
			wantCode:   constant.ErrInvalidHeaderParameter.Error(),
		},
		{
			name:       "invalid path parameter",
			err:        constant.ErrInvalidPathParameter,
			entityType: "request",
			args:       []any{"id"},
			wantType:   "ValidationError",
			wantCode:   constant.ErrInvalidPathParameter.Error(),
		},
		// Data request errors
		{
			name:       "invalid data request with custom message",
			err:        constant.ErrInvalidDataRequest,
			entityType: "job",
			args:       []any{"Custom error message"},
			wantType:   "ValidationError",
			wantCode:   constant.ErrInvalidDataRequest.Error(),
		},
		{
			name:       "invalid data request without args",
			err:        constant.ErrInvalidDataRequest,
			entityType: "job",
			args:       nil,
			wantType:   "ValidationError",
			wantCode:   constant.ErrInvalidDataRequest.Error(),
		},
		// Job related errors
		{
			name:       "missing data source",
			err:        constant.ErrMissingDataSource,
			entityType: "job",
			args:       []any{"transactions"},
			wantType:   "ValidationError",
			wantCode:   constant.ErrMissingDataSource.Error(),
		},
		{
			name:           "job in progress with message",
			err:            constant.ErrJobInProgress,
			entityType:     "job",
			args:           []any{"Job is already running for connection abc-123"},
			wantType:       "ResponseErrorWithStatusCode",
			wantCode:       constant.ErrJobInProgress.Error(),
			wantStatusCode: http.StatusConflict,
		},
		{
			name:           "job in progress without args",
			err:            constant.ErrJobInProgress,
			entityType:     "job",
			args:           nil,
			wantType:       "ResponseErrorWithStatusCode",
			wantCode:       constant.ErrJobInProgress.Error(),
			wantStatusCode: http.StatusConflict,
		},
		// Connection errors
		{
			name:       "connection down with message",
			err:        constant.ErrConnectionDown,
			entityType: "connection",
			args:       []any{"Database connection refused"},
			wantType:   "ValidationError",
			wantCode:   constant.ErrConnectionDown.Error(),
		},
		{
			name:       "connection down without args",
			err:        constant.ErrConnectionDown,
			entityType: "connection",
			args:       nil,
			wantType:   "ValidationError",
			wantCode:   constant.ErrConnectionDown.Error(),
		},
		// Schema validation errors
		{
			name:       "schema validation failed with message",
			err:        constant.ErrSchemaValidationFailed,
			entityType: "schema",
			args:       []any{"Missing required column: id"},
			wantType:   "ValidationError",
			wantCode:   constant.ErrSchemaValidationFailed.Error(),
		},
		{
			name:       "schema validation failed without args",
			err:        constant.ErrSchemaValidationFailed,
			entityType: "schema",
			args:       nil,
			wantType:   "ValidationError",
			wantCode:   constant.ErrSchemaValidationFailed.Error(),
		},
		{
			name:       "schema validation limit with message",
			err:        constant.ErrSchemaValidationLimit,
			entityType: "schema",
			args:       []any{"Too many tables to validate"},
			wantType:   "ValidationError",
			wantCode:   constant.ErrSchemaValidationLimit.Error(),
		},
		{
			name:       "schema validation limit without args",
			err:        constant.ErrSchemaValidationLimit,
			entityType: "schema",
			args:       nil,
			wantType:   "ValidationError",
			wantCode:   constant.ErrSchemaValidationLimit.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateBusinessError(tt.err, tt.entityType, tt.args...)
			require.NotNil(t, got, "ValidateBusinessError() should not return nil")

			switch tt.wantType {
			case "ValidationError":
				e, ok := got.(ValidationError)
				require.True(t, ok, "expected ValidationError, got %T", got)
				assert.Equal(t, tt.wantCode, e.Code)
				assert.Equal(t, tt.entityType, e.EntityType)
				assert.NotEmpty(t, e.Title)
				assert.NotEmpty(t, e.Message)

			case "ResponseErrorWithStatusCode":
				e, ok := got.(ResponseErrorWithStatusCode)
				require.True(t, ok, "expected ResponseErrorWithStatusCode, got %T", got)
				assert.Equal(t, tt.wantCode, e.Code)
				assert.Equal(t, tt.wantStatusCode, e.StatusCode)
				assert.NotEmpty(t, e.Title)
				assert.NotEmpty(t, e.Message)
			}
		})
	}
}

func TestValidateInternalError_StatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		entityType     string
		wantStatusCode int
	}{
		{
			name:           "service unavailable returns 503",
			err:            constant.ErrServiceUnavailable,
			entityType:     "service",
			wantStatusCode: http.StatusServiceUnavailable,
		},
		{
			name:           "conflict returns 409",
			err:            constant.ErrConflict,
			entityType:     "resource",
			wantStatusCode: http.StatusConflict,
		},
		{
			name:           "not found returns 404",
			err:            constant.ErrNotFound,
			entityType:     "resource",
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateInternalError(tt.err, tt.entityType)
			require.NotNil(t, got)

			e, ok := got.(ResponseErrorWithStatusCode)
			require.True(t, ok, "expected ResponseErrorWithStatusCode, got %T", got)
			assert.Equal(t, tt.wantStatusCode, e.StatusCode)
		})
	}
}

func TestValidateBadRequestFieldsError_Priority(t *testing.T) {
	t.Run("unknown fields take priority over required fields", func(t *testing.T) {
		requiredFields := map[string]string{"name": "name is required"}
		unknownFields := map[string]any{"extra": "value"}

		got := ValidateBadRequestFieldsError(requiredFields, nil, "test", unknownFields)
		require.NotNil(t, got)

		e, ok := got.(ValidationUnknownFieldsError)
		require.True(t, ok, "expected ValidationUnknownFieldsError when unknown fields present")
		assert.Equal(t, constant.ErrUnexpectedFieldsInTheRequest.Error(), e.Code)
	})

	t.Run("required fields take priority over invalid fields", func(t *testing.T) {
		requiredFields := map[string]string{"name": "name is required"}
		knownInvalidFields := map[string]string{"email": "invalid format"}

		got := ValidateBadRequestFieldsError(requiredFields, knownInvalidFields, "test", nil)
		require.NotNil(t, got)

		e, ok := got.(ValidationKnownFieldsError)
		require.True(t, ok, "expected ValidationKnownFieldsError")
		assert.Equal(t, constant.ErrMissingFieldsInRequest.Error(), e.Code)
	})

	t.Run("empty maps return generic error", func(t *testing.T) {
		got := ValidateBadRequestFieldsError(map[string]string{}, map[string]string{}, "test", map[string]any{})
		require.NotNil(t, got)
		assert.Contains(t, got.Error(), "expected")
	})
}

func TestValidationError_WithAllFields(t *testing.T) {
	innerErr := errors.New("inner error")
	err := ValidationError{
		EntityType: "connection",
		Title:      "Validation Failed",
		Message:    "Field validation error",
		Code:       "VAL-001",
		Err:        innerErr,
	}

	assert.Equal(t, "VAL-001 - Field validation error", err.Error())
	assert.Equal(t, innerErr, err.Unwrap())
	assert.Equal(t, "connection", err.EntityType)
	assert.Equal(t, "Validation Failed", err.Title)
}

func TestValidationError_Unwrap_NilError(t *testing.T) {
	err := ValidationError{
		Message: "validation error",
		Err:     nil,
	}

	assert.Nil(t, err.Unwrap())
}

func TestErrorTypes_AllFields(t *testing.T) {
	t.Run("UnauthorizedError with all fields", func(t *testing.T) {
		err := UnauthorizedError{
			EntityType: "user",
			Title:      "Unauthorized",
			Message:    "Access denied",
			Code:       "AUTH-001",
			Err:        errors.New("token expired"),
		}
		assert.Equal(t, "Access denied", err.Error())
		assert.Equal(t, "user", err.EntityType)
	})

	t.Run("ForbiddenError with all fields", func(t *testing.T) {
		err := ForbiddenError{
			EntityType: "resource",
			Title:      "Forbidden",
			Message:    "Access forbidden",
			Code:       "FORBID-001",
			Err:        errors.New("insufficient permissions"),
		}
		assert.Equal(t, "Access forbidden", err.Error())
		assert.Equal(t, "resource", err.EntityType)
	})

	t.Run("UnprocessableOperationError with all fields", func(t *testing.T) {
		err := UnprocessableOperationError{
			EntityType: "job",
			Title:      "Unprocessable",
			Message:    "Cannot process job",
			Code:       "PROC-001",
			Err:        errors.New("invalid state"),
		}
		assert.Equal(t, "Cannot process job", err.Error())
		assert.Equal(t, "job", err.EntityType)
	})

	t.Run("HTTPError with all fields", func(t *testing.T) {
		err := HTTPError{
			EntityType: "api",
			Title:      "HTTP Error",
			Message:    "Request failed",
			Code:       "HTTP-001",
			Err:        errors.New("connection timeout"),
		}
		assert.Equal(t, "Request failed", err.Error())
		assert.Equal(t, "api", err.EntityType)
	})

	t.Run("FailedPreconditionError with all fields", func(t *testing.T) {
		err := FailedPreconditionError{
			EntityType: "operation",
			Title:      "Failed Precondition",
			Message:    "Precondition not met",
			Code:       "PREC-001",
			Err:        errors.New("resource not ready"),
		}
		assert.Equal(t, "Precondition not met", err.Error())
		assert.Equal(t, "operation", err.EntityType)
	})

	t.Run("InternalServerError with all fields", func(t *testing.T) {
		err := InternalServerError{
			EntityType: "server",
			Title:      "Internal Error",
			Message:    "Server error occurred",
			Code:       "INT-001",
			Err:        errors.New("database connection failed"),
		}
		assert.Equal(t, "Server error occurred", err.Error())
		assert.Equal(t, "server", err.EntityType)
	})
}

func TestResponseError_AllFields(t *testing.T) {
	err := ResponseError{
		Code:    500,
		Title:   "Internal Server Error",
		Message: "An unexpected error occurred",
	}

	assert.Equal(t, "An unexpected error occurred", err.Error())
	assert.Equal(t, 500, err.Code)
	assert.Equal(t, "Internal Server Error", err.Title)
}

func TestResponseErrorWithStatusCode_AllFields(t *testing.T) {
	err := ResponseErrorWithStatusCode{
		StatusCode: http.StatusNotFound,
		Code:       "NOT-FOUND",
		Title:      "Not Found",
		Message:    "Resource not found",
	}

	assert.Equal(t, "Resource not found", err.Error())
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
	assert.Equal(t, "NOT-FOUND", err.Code)
	assert.Equal(t, "Not Found", err.Title)
}

func TestValidationKnownFieldsError_AllFields(t *testing.T) {
	err := ValidationKnownFieldsError{
		EntityType: "user",
		Title:      "Validation Error",
		Code:       "VAL-001",
		Message:    "Field validation failed",
		Fields: FieldValidations{
			"email": "invalid email format",
			"age":   "must be positive",
		},
	}

	assert.Equal(t, "Field validation failed", err.Error())
	assert.Equal(t, "user", err.EntityType)
	assert.Equal(t, 2, len(err.Fields))
	assert.Equal(t, "invalid email format", err.Fields["email"])
}

func TestValidationUnknownFieldsError_AllFields(t *testing.T) {
	err := ValidationUnknownFieldsError{
		EntityType: "request",
		Title:      "Unknown Fields",
		Code:       "UNK-001",
		Message:    "Unexpected fields in request",
		Fields: UnknownFields{
			"extra_field":   "value1",
			"another_field": 123,
		},
	}

	assert.Equal(t, "Unexpected fields in request", err.Error())
	assert.Equal(t, "request", err.EntityType)
	assert.Equal(t, 2, len(err.Fields))
	assert.Equal(t, "value1", err.Fields["extra_field"])
}

func TestFieldValidations_Type(t *testing.T) {
	fields := FieldValidations{
		"field1": "error1",
		"field2": "error2",
	}

	assert.Equal(t, "error1", fields["field1"])
	assert.Equal(t, "error2", fields["field2"])
	assert.Equal(t, "", fields["nonexistent"])
}

func TestUnknownFields_Type(t *testing.T) {
	fields := UnknownFields{
		"string_field": "value",
		"int_field":    42,
		"bool_field":   true,
		"nested":       map[string]any{"key": "value"},
	}

	assert.Equal(t, "value", fields["string_field"])
	assert.Equal(t, 42, fields["int_field"])
	assert.Equal(t, true, fields["bool_field"])
	assert.Nil(t, fields["nonexistent"])
}
