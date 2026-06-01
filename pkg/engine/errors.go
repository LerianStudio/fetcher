// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "fmt"

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
	// CategoryTimeout indicates an operation exceeded its deadline.
	CategoryTimeout ErrorCategory = "timeout"
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
	CategoryTimeout:       {},
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
