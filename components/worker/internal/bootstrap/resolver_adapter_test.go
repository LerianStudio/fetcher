// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"errors"
	"testing"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildResolverTenantAdapter exercises the worker's resolver-adapter seam in
// isolation: the injected newClient closure stands in for
// initTenantManagerClientWithKey, so the test asserts the exact keys that reach
// client construction + error propagation, without real config or HTTP.
func TestBuildResolverTenantAdapter(t *testing.T) {
	const defaultKey = "default-key"
	logger := testBootstrapLogger()

	t.Run("builds adapter from default key when no per-service env keys", func(t *testing.T) {
		recorded := make([]string, 0)
		newClient := func(apiKey string) (*tmclient.Client, error) {
			recorded = append(recorded, apiKey)
			return &tmclient.Client{}, nil
		}

		adapter, err := buildResolverTenantAdapter(defaultKey, logger, newClient)

		require.NoError(t, err)
		require.NotNil(t, adapter)
		// Only the default client is built when no MULTI_TENANT_SERVICE_API_KEY_* set.
		assert.ElementsMatch(t, []string{defaultKey}, recorded)
	})

	t.Run("picks up a per-service env key and builds a client for it", func(t *testing.T) {
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "crm-key")

		recorded := make([]string, 0)
		newClient := func(apiKey string) (*tmclient.Client, error) {
			recorded = append(recorded, apiKey)
			return &tmclient.Client{}, nil
		}

		adapter, err := buildResolverTenantAdapter(defaultKey, logger, newClient)

		require.NoError(t, err)
		require.NotNil(t, adapter)
		assert.ElementsMatch(t, []string{"crm-key", defaultKey}, recorded)
	})

	t.Run("routes two distinct per-service keys plus the default", func(t *testing.T) {
		// Proves the map iteration builds N per-service clients, not just one.
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "crm-key")
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_LEDGER", "ledger-key")

		recorded := make([]string, 0)
		newClient := func(apiKey string) (*tmclient.Client, error) {
			recorded = append(recorded, apiKey)
			return &tmclient.Client{}, nil
		}

		adapter, err := buildResolverTenantAdapter(defaultKey, logger, newClient)

		require.NoError(t, err)
		require.NotNil(t, adapter)
		assert.ElementsMatch(t, []string{"crm-key", "ledger-key", defaultKey}, recorded)
	})

	t.Run("propagates loader collision error", func(t *testing.T) {
		// Two env vars normalizing to the same token must fail bootstrap.
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN-CRM", "a")
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "b")

		newClient := func(apiKey string) (*tmclient.Client, error) {
			return &tmclient.Client{}, nil
		}

		adapter, err := buildResolverTenantAdapter(defaultKey, logger, newClient)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "load per-service tenant manager API keys for resolver")
	})

	t.Run("propagates builder error from newClient", func(t *testing.T) {
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "crm-key")

		sentinel := errors.New("boom")
		newClient := func(apiKey string) (*tmclient.Client, error) {
			return nil, sentinel
		}

		adapter, err := buildResolverTenantAdapter(defaultKey, logger, newClient)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.ErrorIs(t, err, sentinel)
		assert.Contains(t, err.Error(), "create per-service tenant manager clients for resolver")
	})
}
