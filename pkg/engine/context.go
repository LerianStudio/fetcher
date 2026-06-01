// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine

import "strings"

// TenantContext carries the multi-tenant identity that the embedded Engine
// must honor before resolving connections, reading schema, or extracting data.
//
// It is a pure contract value: it holds identity only and performs no I/O.
// Both DUAL-MODE deployments are representable — SaaS callers populate
// OrganizationID while single-tenant hosts may leave it empty and rely on
// ProductName scoping. Engine core never invents tenant identity; the host
// supplies it.
type TenantContext struct {
	// OrganizationID is the tenant boundary in SaaS/multi-tenant mode.
	// It may be empty in single-tenant deployments.
	OrganizationID string

	// ProductName scopes ownership of connections and jobs within a tenant.
	ProductName string

	// RequestID correlates a single host request across Engine operations.
	// It is optional and used only for traceability, never for authorization.
	RequestID string
}

// NewTenantContext builds a TenantContext from the host-provided identity.
// Inputs are trimmed; no validation error is returned because callers in
// single-tenant mode legitimately pass an empty OrganizationID.
func NewTenantContext(organizationID, productName string) TenantContext {
	return TenantContext{
		OrganizationID: strings.TrimSpace(organizationID),
		ProductName:    strings.TrimSpace(productName),
	}
}

// WithRequestID returns a copy of the context carrying the given request ID.
func (tc TenantContext) WithRequestID(requestID string) TenantContext {
	tc.RequestID = strings.TrimSpace(requestID)
	return tc
}

// IsZero reports whether the context carries no identity at all.
func (tc TenantContext) IsZero() bool {
	return tc.OrganizationID == "" && tc.ProductName == "" && tc.RequestID == ""
}

// IsMultiTenant reports whether an organization boundary is present.
func (tc TenantContext) IsMultiTenant() bool {
	return tc.OrganizationID != ""
}
