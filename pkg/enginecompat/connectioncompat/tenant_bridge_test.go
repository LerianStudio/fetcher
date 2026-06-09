// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package connectioncompat_test

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"

	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantContextFromRequest_SingleTenantDefault(t *testing.T) {
	t.Parallel()

	tenant, err := connectioncompat.TenantContextFromRequest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, connectioncompat.SingleTenantID, tenant.TenantID,
		"absent tmcore tenant id must fall back to the single-tenant default")
}

func TestTenantContextFromRequest_DerivesTmcoreTenant(t *testing.T) {
	t.Parallel()

	ctx := tmcore.ContextWithTenantID(context.Background(), "tenant-xyz")

	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, "tenant-xyz", tenant.TenantID,
		"engine tenant must be derived from the request's tmcore tenant id")
}

func TestTenantContextFromRequest_PerRequestScope(t *testing.T) {
	t.Parallel()

	// Two distinct requests must yield two distinct tenant scopes — proving the
	// derivation is per-request, never ambient/global.
	ctxA := tmcore.ContextWithTenantID(context.Background(), "tenant-a")
	ctxB := tmcore.ContextWithTenantID(context.Background(), "tenant-b")

	tenantA, err := connectioncompat.TenantContextFromRequest(ctxA)
	require.NoError(t, err)
	tenantB, err := connectioncompat.TenantContextFromRequest(ctxB)
	require.NoError(t, err)

	assert.Equal(t, "tenant-a", tenantA.TenantID)
	assert.Equal(t, "tenant-b", tenantB.TenantID)
	assert.NotEqual(t, tenantA.TenantID, tenantB.TenantID)
}
