// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package connectioncompat

import (
	"context"

	"github.com/LerianStudio/fetcher/pkg/engine"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/LerianStudio/lib-observability"
)

// SingleTenantID is the tenant identity used when the Manager runs in
// single-tenant mode (no tmcore tenant id is injected into the request
// context). The Engine requires a non-empty, shape-valid tenantId on every
// owned-resource operation (B2: tenantId is the SOLE isolation boundary), so a
// stable single-tenant default keeps the Engine gate operative without
// reintroducing organization/product scoping into the Engine.
//
// It is a host-level constant: the Engine treats it as an opaque, valid tenant
// id like any other. In multi-tenant mode it is never used because a real
// tmcore tenant id is present per request.
const SingleTenantID = "default"

// TenantContextFromRequest derives the Engine's per-request TenantContext from
// the live request context. This is the host-bridge that reconciles the
// Manager's host-level tenant model with the Engine's owner-locked tenantId
// scope (B2):
//
//   - The Engine carries ONLY tenantId (NO organization, NO product). The
//     Manager keeps organization / product / X-Product-Name as HOST concepts in
//     its HTTP layer and response models; they never scope the Engine.
//   - The tenantId is taken from the tmcore tenant id that the multi-tenant
//     middleware injected into THIS request's context. It is therefore scoped to
//     the request's actual tenant — never a global or ambient tenant. Getting
//     this wrong would be cross-tenant leakage, so the value is read fresh from
//     the supplied ctx on every call and never cached.
//   - In single-tenant mode the tmcore tenant id is empty, so the Manager's
//     existing single-tenant resolution applies: SingleTenantID.
//
// The request id is propagated for traceability only; it never participates in
// authorization.
func TenantContextFromRequest(ctx context.Context) (engine.TenantContext, error) {
	tenantID := tmcore.GetTenantIDContext(ctx)
	if tenantID == "" {
		tenantID = SingleTenantID
	}

	tenant, err := engine.NewTenantContext(tenantID)
	if err != nil {
		return engine.TenantContext{}, err
	}

	if _, _, requestID, _ := observability.NewTrackingFromContext(ctx); requestID != "" {
		tenant = tenant.WithRequestID(requestID)
	}

	return tenant, nil
}
