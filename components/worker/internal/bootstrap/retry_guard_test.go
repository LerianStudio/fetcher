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
	t.Parallel()

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
			name:     "json.SyntaxError is non-retryable",
			err:      &json.SyntaxError{Offset: 1},
			expected: true,
		},
		{
			name:     "json.UnmarshalTypeError is non-retryable",
			err:      &json.UnmarshalTypeError{Value: "string", Type: nil},
			expected: true,
		},
		{
			name:     "wrapped json.SyntaxError is non-retryable",
			err:      fmt.Errorf("parse message: %w", &json.SyntaxError{Offset: 5}),
			expected: true,
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
		// Wrapped source errors (FET-005x codes from error wrapping at source)
		{
			name:     "ValidationError FET-0050 (null payload) is non-retryable",
			err:      pkg.ValidationError{Code: "FET-0050", Title: "Invalid Message", Message: "message payload is null"},
			expected: true,
		},
		{
			name:     "ValidationError FET-0054 (connection not found) is non-retryable",
			err:      pkg.ValidationError{Code: "FET-0054", Title: "Connection Not Found", Message: "connection not found for database: mydb"},
			expected: true,
		},
		{
			name:     "FailedPreconditionError FET-0056 (encryption not configured) is non-retryable",
			err:      pkg.FailedPreconditionError{Code: "FET-0056", Title: "Encryption Not Configured", Message: "storage encrypt secret key not configured"},
			expected: true,
		},
		{
			name:     "FailedPreconditionError FET-0057 (CRM crypto) is non-retryable",
			err:      pkg.FailedPreconditionError{Code: "FET-0057", Title: "CRM Crypto Not Configured", Message: "CRM hash secret key not configured"},
			expected: true,
		},
		{
			name:     "wrapped ValidationError FET-0050 is non-retryable",
			err:      fmt.Errorf("parse: %w", pkg.ValidationError{Code: "FET-0050", Title: "Invalid Message", Message: "message payload is null"}),
			expected: true,
		},
		{
			name:     "wrapped FailedPreconditionError is non-retryable",
			err:      fmt.Errorf("query: %w", pkg.FailedPreconditionError{Code: "FET-0058", Title: "CRM Crypto Not Configured", Message: "CRM encrypt secret key not configured"}),
			expected: true,
		},
		// Heuristic pattern fallback (safety net for untyped errors)
		{
			name:     "pattern: 'key not configured' is non-retryable",
			err:      fmt.Errorf("CRM hash secret key not configured"),
			expected: true,
		},
		{
			name:     "pattern: 'failed to initialize cipher' is non-retryable",
			err:      fmt.Errorf("failed to initialize cipher: invalid key size"),
			expected: true,
		},
		{
			name:     "pattern: 'connection not found for database' is non-retryable",
			err:      fmt.Errorf("connection not found for database: prod_db"),
			expected: true,
		},
		{
			name:     "pattern: 'no collections found matching prefix' is non-retryable",
			err:      fmt.Errorf("no collections found matching prefix holders_"),
			expected: true,
		},
		{
			name:     "pattern: 'is unavailable (initialization failed)' is non-retryable",
			err:      fmt.Errorf("datasource mydb is unavailable (initialization failed)"),
			expected: true,
		},
		{
			name:     "pattern: 'unsupported database type' is non-retryable",
			err:      fmt.Errorf("unsupported database type: oracle"),
			expected: true,
		},
		// Negative heuristic cases — transient errors must NOT match
		{
			name:     "pattern negative: 'service unavailable' without init context is retryable",
			err:      fmt.Errorf("service temporarily unavailable"),
			expected: false,
		},
		{
			name:     "pattern negative: generic 'not found' is retryable",
			err:      fmt.Errorf("document not found in collection"),
			expected: false,
		},
		{
			name:     "pattern negative: 'required' without 'is required' is retryable",
			err:      fmt.Errorf("additional authentication required"),
			expected: false,
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
			t.Parallel()

			result := isNonRetryableHandlerError(tt.err)
			assert.Equal(t, tt.expected, result,
				"isNonRetryableHandlerError(%v) = %v, want %v", tt.err, result, tt.expected)
		})
	}
}

func TestHandlerGenerateReport_PermanentErrorClassification(t *testing.T) {
	t.Parallel()

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
