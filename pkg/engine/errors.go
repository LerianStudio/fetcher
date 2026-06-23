// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import (
	"context"
	"errors"
	"fmt"
)

// ErrorCategory is a stable, safe classification for Engine failures. Hosts
// map categories to transport status codes (HTTP, gRPC) without depending on
// Engine internals. Values are stable string constants; changing them is a
// breaking change.
type ErrorCategory string

const (
	// CategoryValidation indicates malformed or invalid input.
	CategoryValidation ErrorCategory = "validation"
	// CategoryNotFound indicates a referenced resource does not exist.
	CategoryNotFound ErrorCategory = "not_found"
	// CategoryUnauthorized indicates missing or invalid authentication.
	CategoryUnauthorized ErrorCategory = "unauthorized"
	// CategoryForbidden indicates the caller lacks ownership or permission.
	CategoryForbidden ErrorCategory = "forbidden"
	// CategoryLimitExceeded indicates an Engine limit was breached.
	CategoryLimitExceeded ErrorCategory = "limit_exceeded"
	// CategoryConflict indicates the operation conflicts with the current state
	// of the resource — e.g. a mutation blocked because active work references it.
	// Hosts map it to HTTP 409 Conflict.
	CategoryConflict ErrorCategory = "conflict"
	// CategoryUnavailable indicates a dependency (datasource, store) is down.
	CategoryUnavailable ErrorCategory = "unavailable"
	// CategoryConnect indicates the connector failed to ESTABLISH connectivity to
	// the datasource (the connect stage), as distinct from a failure while READING
	// from an already-connected datasource (which stays CategoryUnavailable). It
	// lets a host distinguish "could not connect" (e.g. TLS required, auth refused,
	// host unreachable) from "connected but the read failed" without ever
	// surfacing the raw, secret-bearing connector error. Hosts map it to the same
	// transport family as CategoryUnavailable (a 5xx dependency failure); the
	// distinction exists for stage attribution, not a different status class.
	CategoryConnect ErrorCategory = "connect"
	// CategoryTimeout indicates an operation exceeded its deadline
	// (context.DeadlineExceeded). Hosts map it to HTTP 504 Gateway Timeout — the
	// execution outran its own bound, distinct from a host-initiated cancel.
	CategoryTimeout ErrorCategory = "timeout"
	// CategoryCanceled indicates the host canceled the operation before it
	// finished (context.Canceled). It is DISTINCT from CategoryTimeout: the host
	// withdrew the request rather than the execution overrunning a deadline. Hosts
	// map it to HTTP 499 Client Closed Request.
	CategoryCanceled ErrorCategory = "canceled"
	// CategoryInternal indicates an unexpected internal failure.
	CategoryInternal ErrorCategory = "internal"
)

var validErrorCategories = map[ErrorCategory]struct{}{
	CategoryValidation:    {},
	CategoryNotFound:      {},
	CategoryUnauthorized:  {},
	CategoryForbidden:     {},
	CategoryLimitExceeded: {},
	CategoryConflict:      {},
	CategoryUnavailable:   {},
	CategoryConnect:       {},
	CategoryTimeout:       {},
	CategoryCanceled:      {},
	CategoryInternal:      {},
}

// IsValid reports whether the category is a known, stable Engine category.
func (c ErrorCategory) IsValid() bool {
	_, ok := validErrorCategories[c]
	return ok
}

// EngineError is the safe, stable error contract returned across the Engine
// boundary. Callers MUST only place pre-redacted, credential-free text in
// Message. The Engine never embeds passwords, DSNs, or tokens in errors.
type EngineError struct {
	// Category classifies the failure for transport mapping.
	Category ErrorCategory
	// Message is a safe, human-readable description. It MUST NOT contain
	// credentials, connection strings, or any secret material.
	Message string
	// cause is an OPTIONAL underlying error preserved ONLY for errors.As /
	// errors.Is transparency (so a host can still recognize a typed error it
	// wrapped, e.g. a host-safety pkg.ValidationError). It is DELIBERATELY
	// unexported and NEVER rendered by Error(): the cause MAY embed a DSN,
	// credential, or driver internals, so the public string stays the safe
	// Message. Set it only via NewWrappedEngineError when the wrapped cause is
	// itself something the host must inspect.
	cause error
}

// NewEngineError constructs a safe Engine error. Callers are responsible for
// passing a credential-free message; this constructor performs no decryption
// and never inspects secret-bearing structures.
func NewEngineError(category ErrorCategory, message string) *EngineError {
	return &EngineError{
		Category: category,
		Message:  message,
	}
}

// NewWrappedEngineError constructs a safe Engine error that PRESERVES an
// underlying cause for errors.As / errors.Is transparency WITHOUT leaking it.
// The rendered Error() string is still only the safe, credential-free message;
// the cause is reachable solely through Unwrap, so a host can recognize a typed
// error it raised (e.g. a host-safety pkg.ValidationError) while the public
// boundary message stays redacted. Use it ONLY when the host genuinely needs to
// see through to the cause; plain failures stay on NewEngineError.
func NewWrappedEngineError(category ErrorCategory, message string, cause error) *EngineError {
	return &EngineError{
		Category: category,
		Message:  message,
		cause:    cause,
	}
}

// Unwrap returns the preserved cause (or nil), making errors.As / errors.Is
// transparent through the safe boundary error. Error() never renders the cause,
// so unwrapping exposes the typed error for inspection without leaking its text.
func (e *EngineError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

// contextError maps a context error to a safe, category-correct EngineError,
// keeping the timeout/canceled distinction the host relies on for transport
// mapping. context.DeadlineExceeded means the execution outran its own bound
// (timeout); context.Canceled means the host withdrew the request (canceled).
// The message is fixed and credential-free: a context error carries no payload,
// DSN, or driver internals to redact. A nil (or non-context) error returns nil
// so callers can use it as a pass-through guard.
func contextError(err error) *EngineError {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.DeadlineExceeded):
		return NewEngineError(CategoryTimeout, "execution exceeded its deadline")
	case errors.Is(err, context.Canceled):
		return NewEngineError(CategoryCanceled, "execution canceled")
	default:
		return nil
	}
}

// Error implements the error interface using only safe fields.
func (e *EngineError) Error() string {
	if e == nil {
		return ""
	}

	category := e.Category
	if category == "" {
		category = CategoryInternal
	}

	return fmt.Sprintf("engine: [%s] %s", category, e.Message)
}
