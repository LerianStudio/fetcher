// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package resolver

import (
	"testing"

	tmclient "github.com/LerianStudio/lib-commons/v6/commons/tenant-manager/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantManagerAdapter_pickClient(t *testing.T) {
	t.Parallel()

	// Distinct sentinel pointers so the test asserts WHICH client was selected,
	// without needing a real HTTP round-trip.
	ledgerClient := &tmclient.Client{}
	crmClient := &tmclient.Client{}
	defaultClient := &tmclient.Client{}

	t.Run("per-service hit routes to the matching client", func(t *testing.T) {
		t.Parallel()
		a := NewTenantManagerAdapterWithClients(map[string]*tmclient.Client{
			"LEDGER":     ledgerClient,
			"PLUGIN_CRM": crmClient,
		}, defaultClient)

		got, err := a.pickClient("ledger")
		require.NoError(t, err)
		assert.Same(t, ledgerClient, got)

		got, err = a.pickClient("plugin-crm")
		require.NoError(t, err)
		assert.Same(t, crmClient, got)
	})

	t.Run("missing per-service entry falls back to default", func(t *testing.T) {
		t.Parallel()
		a := NewTenantManagerAdapterWithClients(map[string]*tmclient.Client{
			"LEDGER": ledgerClient,
		}, defaultClient)

		got, err := a.pickClient("plugin-crm")
		require.NoError(t, err)
		assert.Same(t, defaultClient, got)
	})

	t.Run("no per-service map and only default still resolves via default", func(t *testing.T) {
		t.Parallel()
		a := NewTenantManagerAdapterWithClients(nil, defaultClient)

		got, err := a.pickClient("anything")
		require.NoError(t, err)
		assert.Same(t, defaultClient, got)
	})

	t.Run("both nil yields an explicit error", func(t *testing.T) {
		t.Parallel()
		a := NewTenantManagerAdapterWithClients(nil, nil)

		got, err := a.pickClient("ledger")
		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("service name is normalized before lookup", func(t *testing.T) {
		t.Parallel()
		a := NewTenantManagerAdapterWithClients(map[string]*tmclient.Client{
			"MIDAZ_ONBOARDING": ledgerClient,
		}, defaultClient)

		// Hyphenated/lower-case service name must map onto the normalized key.
		got, err := a.pickClient("midaz-onboarding")
		require.NoError(t, err)
		assert.Same(t, ledgerClient, got)
	})
}

func TestNewTenantManagerAdapterWithClients_NilSafe(t *testing.T) {
	t.Parallel()

	// Constructing with a nil map must not panic and pickClient must fall back
	// to the default cleanly.
	defaultClient := &tmclient.Client{}

	a := NewTenantManagerAdapterWithClients(nil, defaultClient)
	require.NotNil(t, a)

	got, err := a.pickClient("whatever")
	require.NoError(t, err)
	assert.Same(t, defaultClient, got)
}
