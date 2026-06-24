// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package resolver

import (
	"testing"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantManagerAdapter_pickClient(t *testing.T) {
	// Distinct sentinel pointers so the test asserts WHICH client was selected,
	// without needing a real HTTP round-trip.
	ledgerClient := &tmclient.Client{}
	crmClient := &tmclient.Client{}
	defaultClient := &tmclient.Client{}

	t.Run("per-service hit routes to the matching client", func(t *testing.T) {
		a := NewTenantManagerAdapterWithClients(map[string]*tmclient.Client{
			"LEDGER":     ledgerClient,
			"PLUGIN_CRM": crmClient,
		}, defaultClient)

		assert.Same(t, ledgerClient, a.pickClient("ledger"))
		assert.Same(t, crmClient, a.pickClient("plugin-crm"))
	})

	t.Run("missing per-service entry falls back to default", func(t *testing.T) {
		a := NewTenantManagerAdapterWithClients(map[string]*tmclient.Client{
			"LEDGER": ledgerClient,
		}, defaultClient)

		assert.Same(t, defaultClient, a.pickClient("plugin-crm"))
	})

	t.Run("no per-service map and only default still resolves via default", func(t *testing.T) {
		a := NewTenantManagerAdapterWithClients(nil, defaultClient)

		assert.Same(t, defaultClient, a.pickClient("anything"))
	})

	t.Run("both nil yields nil (caller turns this into an explicit error)", func(t *testing.T) {
		a := NewTenantManagerAdapterWithClients(nil, nil)

		assert.Nil(t, a.pickClient("ledger"))
	})

	t.Run("service name is normalized before lookup", func(t *testing.T) {
		a := NewTenantManagerAdapterWithClients(map[string]*tmclient.Client{
			"MIDAZ_ONBOARDING": ledgerClient,
		}, defaultClient)

		// Hyphenated/lower-case service name must map onto the normalized key.
		assert.Same(t, ledgerClient, a.pickClient("midaz-onboarding"))
	})
}

func TestNewTenantManagerAdapterWithClients_NilSafe(t *testing.T) {
	// Constructing with a nil map must not panic and pickClient must fall back
	// to the default cleanly.
	defaultClient := &tmclient.Client{}

	a := NewTenantManagerAdapterWithClients(nil, defaultClient)
	require.NotNil(t, a)
	assert.Same(t, defaultClient, a.pickClient("whatever"))
}
