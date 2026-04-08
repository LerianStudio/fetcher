package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/LerianStudio/fetcher/pkg"
)

// isNonRetryableHandlerError classifies whether a handler error is permanent
// (non-retryable) or transient (retryable). Permanent errors are those that
// will never succeed on retry — validation failures, authorization errors,
// business rule violations (FET-* codes), canceled contexts, and permanent
// tenant errors.
//
// When this returns true, the caller should return nil to lib-commons so the
// message is Acked (dropped) instead of Nacked+requeued, preventing infinite
// redelivery loops.
//
// NOTE: context.DeadlineExceeded is intentionally RETRYABLE for fetcher.
// Unlike other services, fetcher query timeouts are transient because database
// load spikes may resolve on retry.
func isNonRetryableHandlerError(err error) bool {
	if err == nil {
		return false
	}

	// Canceled context — the consumer is shutting down; requeuing is pointless.
	if errors.Is(err, context.Canceled) {
		return true
	}

	// JSON parse errors — malformed messages will never succeed on retry.
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return true
	}

	// Permanent tenant errors (not found, suspended, service not configured, manager closed).
	if isPermanentTenantError(err) {
		return true
	}

	// Domain-level permanent errors (validation, auth, business rules).
	if isNonRetryableDomainError(err) {
		return true
	}

	// FET-* coded business errors embedded as string messages.
	if strings.Contains(err.Error(), "FET-") {
		return true
	}

	// Last-resort heuristic: catch permanent errors not yet wrapped in typed errors.
	if isPermanentErrorByPattern(err.Error()) {
		return true
	}

	// Everything else is considered transient — allow retry.
	return false
}

// isNonRetryableDomainError checks whether the error matches any of the fetcher
// domain error types that represent permanent failures.
func isNonRetryableDomainError(err error) bool {
	var validationErr pkg.ValidationError
	if errors.As(err, &validationErr) {
		return true
	}

	var forbiddenErr pkg.ForbiddenError
	if errors.As(err, &forbiddenErr) {
		return true
	}

	var unauthorizedErr pkg.UnauthorizedError
	if errors.As(err, &unauthorizedErr) {
		return true
	}

	var unprocessableErr pkg.UnprocessableOperationError
	if errors.As(err, &unprocessableErr) {
		return true
	}

	var failedPreconditionErr pkg.FailedPreconditionError
	if errors.As(err, &failedPreconditionErr) {
		return true
	}

	var knownFieldsErr pkg.ValidationKnownFieldsError
	if errors.As(err, &knownFieldsErr) {
		return true
	}

	var unknownFieldsErr pkg.ValidationUnknownFieldsError

	return errors.As(err, &unknownFieldsErr)
}

// isPermanentErrorByPattern is a last-resort safety net that catches permanent
// errors not yet wrapped in typed domain errors. It uses string matching on the
// error message — prefer wrapping errors at the source with typed errors instead.
//
// Patterns are intentionally specific to avoid false positives from transient
// errors that happen to contain common substrings. Each pattern targets a known
// permanent error message from the extraction pipeline.
func isPermanentErrorByPattern(errMsg string) bool {
	permanentPatterns := []string{
		"key not configured",              // crypto/storage config missing
		"client is not configured",        // storage client not injected
		"payload is null",                 // nil message body
		"has no tables",                   // empty mappedFields entry
		"does not support crm queries",    // type assertion failure for CRM datasource
		"connection not found for database", // datasource not in connections list
		"no collections found matching prefix", // CRM collection prefix has no matches
		"unsupported database type",       // unknown datasource type in config
		"unexpected schema result type",   // circuit breaker returned wrong type
		"unexpected query result type",    // circuit breaker returned wrong type
		"is unavailable (initialization failed)", // datasource permanently failed init
		"failed to initialize cipher",     // crypto key invalid
		"no job data found",               // job not in database
	}

	lower := strings.ToLower(errMsg)

	for _, pattern := range permanentPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}
