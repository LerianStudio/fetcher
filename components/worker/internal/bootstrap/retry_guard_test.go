package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/LerianStudio/fetcher/pkg"
	tmcore "github.com/LerianStudio/lib-commons/v4/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
)

func TestIsNonRetryableHandlerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error is retryable",
			err:      nil,
			expected: false,
		},
		{
			name:     "context.Canceled is non-retryable",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "context.DeadlineExceeded is retryable (intentional for fetcher)",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "wrapped context.Canceled is non-retryable",
			err:      fmt.Errorf("handler failed: %w", context.Canceled),
			expected: true,
		},
		{
			name:     "wrapped context.DeadlineExceeded is retryable",
			err:      fmt.Errorf("query timed out: %w", context.DeadlineExceeded),
			expected: false,
		},
		// Permanent tenant errors
		{
			name:     "ErrTenantNotFound is non-retryable",
			err:      tmcore.ErrTenantNotFound,
			expected: true,
		},
		{
			name:     "ErrServiceNotConfigured is non-retryable",
			err:      tmcore.ErrServiceNotConfigured,
			expected: true,
		},
		{
			name:     "ErrManagerClosed is non-retryable",
			err:      tmcore.ErrManagerClosed,
			expected: true,
		},
		{
			name: "TenantSuspendedError is non-retryable",
			err: &tmcore.TenantSuspendedError{
				TenantID: "t-1",
				Status:   "suspended",
			},
			expected: true,
		},
		// JSON parse errors
		{
			name:         "json.SyntaxError is non-retryable",
			err:          &json.SyntaxError{Offset: 1},
			expected:     true,
		},
		{
			name:         "json.UnmarshalTypeError is non-retryable",
			err:          &json.UnmarshalTypeError{Value: "string", Type: nil},
			expected:     true,
		},
		{
			name:         "wrapped json.SyntaxError is non-retryable",
			err:          fmt.Errorf("parse message: %w", &json.SyntaxError{Offset: 5}),
			expected:     true,
		},
		// Domain error types
		{
			name:     "ValidationError is non-retryable",
			err:      pkg.ValidationError{Code: "FET-1000", Message: "invalid input"},
			expected: true,
		},
		{
			name:     "ForbiddenError is non-retryable",
			err:      pkg.ForbiddenError{Code: "FET-1002", Message: "forbidden"},
			expected: true,
		},
		{
			name:     "UnauthorizedError is non-retryable",
			err:      pkg.UnauthorizedError{Code: "FET-1003", Message: "unauthorized"},
			expected: true,
		},
		{
			name:     "UnprocessableOperationError is non-retryable",
			err:      pkg.UnprocessableOperationError{Code: "FET-1004", Message: "unprocessable"},
			expected: true,
		},
		{
			name:     "FailedPreconditionError is non-retryable",
			err:      pkg.FailedPreconditionError{Code: "FET-1005", Message: "precondition failed"},
			expected: true,
		},
		{
			name:     "ValidationKnownFieldsError is non-retryable",
			err:      pkg.ValidationKnownFieldsError{Code: "FET-1006", Message: "known fields invalid"},
			expected: true,
		},
		{
			name:     "ValidationUnknownFieldsError is non-retryable",
			err:      pkg.ValidationUnknownFieldsError{Code: "FET-1007", Message: "unknown fields"},
			expected: true,
		},
		// FET-* string match
		{
			name:     "FET-coded string error is non-retryable",
			err:      errors.New("FET-1001 - entity not found"),
			expected: true,
		},
		{
			name:     "wrapped FET-coded error is non-retryable",
			err:      fmt.Errorf("handler: %w", errors.New("FET-1001 - entity not found")),
			expected: true,
		},
		{
			name:     "wrapped ValidationError is non-retryable",
			err:      fmt.Errorf("extract: %w", pkg.ValidationError{Code: "FET-1000", Message: "bad data"}),
			expected: true,
		},
		// Transient errors
		{
			name:     "connection refused is retryable (transient)",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "ErrCircuitBreakerOpen is retryable (transient)",
			err:      tmcore.ErrCircuitBreakerOpen,
			expected: false,
		},
		{
			name:     "wrapped network error is retryable (transient)",
			err:      fmt.Errorf("mongo: %w", errors.New("i/o timeout")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNonRetryableHandlerError(tt.err)
			assert.Equal(t, tt.expected, result,
				"isNonRetryableHandlerError(%v) = %v, want %v", tt.err, result, tt.expected)
		})
	}
}

func TestHandlerGenerateReport_PermanentErrorClassification(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "wrapped validation error from extractExternalData is non-retryable",
			err:      fmt.Errorf("failed to generate report: %w", pkg.ValidationError{Code: "FET-1000", Message: "invalid schema"}),
			expected: true,
		},
		{
			name:     "FET-coded string error is non-retryable",
			err:      errors.New("FET-1081 - tenant not found in fetcher context"),
			expected: true,
		},
		{
			name:     "generic operation failed error is retryable (transient)",
			err:      errors.New("operation failed"),
			expected: false,
		},
		{
			name:     "transient DB error is retryable",
			err:      fmt.Errorf("mongo query: %w", errors.New("connection pool exhausted")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNonRetryableHandlerError(tt.err)
			assert.Equal(t, tt.expected, result,
				"isNonRetryableHandlerError(%v) = %v, want %v", tt.err, result, tt.expected)
		})
	}
}
