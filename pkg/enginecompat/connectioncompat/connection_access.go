// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package connectioncompat

import (
	"context"
	"errors"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
)

// AuthorizeAccess routes the per-request tenant-scope authority decision through
// the Engine for connection operations that keep their own (UUID-keyed)
// persistence and identity model (Manager command mutations; query get/list).
// The tenant is derived FRESH from this request's context via
// TenantContextFromRequest (single-tenant falls back to SingleTenantID), never
// ambient or cached. A nil engine (test-only construction) is a no-op so the
// operation proceeds, mirroring the conflict gate's defensive guard.
//
// This is the SINGLE home for the authorize-via-Engine contract shared by the
// command and query packages; both delegate here so the scope rule and its nil
// guard live in exactly one place.
func AuthorizeAccess(ctx context.Context, eng *engine.Engine) error {
	if eng == nil {
		return nil
	}

	tenant, err := TenantContextFromRequest(ctx)
	if err != nil {
		return err
	}

	return eng.AuthorizeConnectionAccess(ctx, tenant)
}

// FindByID routes a connection read through the Engine's ID-addressed op (Engine
// tenant-scope + ConnectionStore.FindByID -> repo.FindByID), returning the rich
// record unpacked from the opaque host payload.
//
// It returns (nil, nil) when the connection is not found so the caller maps it to
// the Manager's existing not-found business error — the not-found-mapping
// contract (Engine CategoryNotFound -> (nil, nil)) lives HERE, in one place,
// shared by every caller (command get-for-mutation, query GetConnection,
// GetConnectionSchema). A nil engine is unreachable in the assembled Manager;
// callers always pass a real engine.
func FindByID(ctx context.Context, eng *engine.Engine, connectionID string) (*model.Connection, error) {
	tenant, err := TenantContextFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	descriptor, err := eng.GetConnectionByID(ctx, tenant, connectionID)
	if err != nil {
		var engErr *engine.EngineError
		if errors.As(err, &engErr) && engErr.Category == engine.CategoryNotFound {
			return nil, nil
		}

		return nil, err
	}

	return ConnectionFromDescriptor(descriptor), nil
}
