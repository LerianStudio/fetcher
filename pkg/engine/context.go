// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "strings"

// maxTenantIDLength bounds a tenant ID. It MIRRORS lib-commons
// tenant-manager/core.MaxTenantIDLength (the source of truth the host bridges
// from). The value is re-declared locally rather than imported because
// tenant-manager/core transitively pulls mongo-driver and dbresolver, which
// would violate the embedded Engine's no-infrastructure dependency boundary
// (see dependency_test.go). The host maps its tenant identity onto this opaque
// value contract at the seam.
const maxTenantIDLength = 256

// TenantContext carries the tenant identity that the embedded Engine must honor
// before resolving connections, reading schema, or extracting data.
//
// The Engine is embedded INTO each product (matcher, reporter, ...). The product
// IS the host process, so it does not identify itself as a "product" inside its
// own embedded Engine, and organization grouping is the embedder's concern, not
// an Engine isolation concern. One embedded product may serve N tenants, so the
// tenantId is the SOLE isolation boundary and is non-negotiable: it must be
// present in the context for any owned resource.
//
// It is a pure contract value: it holds identity only and performs no I/O. The
// Engine core never invents tenant identity; the host supplies it.
type TenantContext struct {
	// TenantID is the isolation boundary. Every owned resource (connections,
	// schema, executions, results) is scoped by it. It is an opaque string value
	// to the Engine, validated for shape but never interpreted.
	TenantID string

	// RequestID correlates a single host request across Engine operations.
	// It is optional and used only for traceability, never for authorization.
	RequestID string
}

// NewTenantContext builds a TenantContext from the host-provided tenant ID. The
// ID is trimmed and validated as an opaque value (see isValidTenantID). An
// invalid tenant ID yields a CategoryValidation *EngineError; tenantId is the
// isolation boundary, so an empty or malformed one is rejected rather than
// silently accepted.
func NewTenantContext(tenantID string) (TenantContext, error) {
	trimmed := strings.TrimSpace(tenantID)
	if !isValidTenantID(trimmed) {
		return TenantContext{}, NewEngineError(CategoryValidation, "tenant id is invalid")
	}

	return TenantContext{TenantID: trimmed}, nil
}

// WithRequestID returns a copy of the context carrying the given request ID.
func (tc TenantContext) WithRequestID(requestID string) TenantContext {
	tc.RequestID = strings.TrimSpace(requestID)
	return tc
}

// IsZero reports whether the context carries no identity at all.
func (tc TenantContext) IsZero() bool {
	return tc.TenantID == "" && tc.RequestID == ""
}

// IsMultiTenant reports whether a tenant boundary is present.
func (tc TenantContext) IsMultiTenant() bool {
	return tc.TenantID != ""
}

// isValidTenantID validates a tenant ID as an opaque value. It MIRRORS
// lib-commons tenant-manager/core.IsValidTenantID (the hand-rolled equivalent of
// `^[a-zA-Z0-9][a-zA-Z0-9_-]*$`): non-empty, at most maxTenantIDLength bytes,
// first byte ASCII alphanumeric, remaining bytes ASCII alphanumeric, hyphen, or
// underscore. This rejects empty, whitespace-only, control, and otherwise
// malformed identifiers. The logic is re-implemented locally — not imported —
// to keep the Engine free of tenant-manager's infrastructure dependencies.
func isValidTenantID(id string) bool {
	n := len(id)
	if n == 0 || n > maxTenantIDLength {
		return false
	}

	if !isTenantIDAlnum(id[0]) {
		return false
	}

	for i := 1; i < n; i++ {
		c := id[i]
		if !isTenantIDAlnum(c) && c != '-' && c != '_' {
			return false
		}
	}

	return true
}

// isTenantIDAlnum reports whether c is an ASCII alphanumeric byte.
func isTenantIDAlnum(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
