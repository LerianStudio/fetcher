package pkg

import (
	"errors"
	"net/http"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/constant"
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
