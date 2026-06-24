// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package resolver

import (
	"errors"
	"testing"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeServiceToken(t *testing.T) {
	tests := []struct {
		name    string
		service string
		want    string
	}{
		{name: "hyphenated lower-case", service: "plugin-crm", want: "PLUGIN_CRM"},
		{name: "single segment", service: "ledger", want: "LEDGER"},
		{name: "already upper", service: "LEDGER", want: "LEDGER"},
		{name: "mixed case multi-hyphen", service: "midaz-onboarding-svc", want: "MIDAZ_ONBOARDING_SVC"},
		{name: "empty", service: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeServiceToken(tt.service))
		})
	}
}

func TestLoadServiceAPIKeysFromEnv(t *testing.T) {
	t.Run("picks up per-service keys, ignores bare prefix, skips empty", func(t *testing.T) {
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_LEDGER", "ledger-key")
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "crm-key")
		// Bare prefix is the default key, handled elsewhere — must NOT appear.
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY", "default-key")
		// Empty-valued per-service var must be skipped.
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_EMPTY", "")

		got, err := LoadServiceAPIKeysFromEnv()

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "ledger-key", got["LEDGER"])
		assert.Equal(t, "crm-key", got["PLUGIN_CRM"])

		_, hasBare := got[""]
		assert.False(t, hasBare, "bare MULTI_TENANT_SERVICE_API_KEY must not be loaded as a per-service key")

		_, hasEmpty := got["EMPTY"]
		assert.False(t, hasEmpty, "empty-valued per-service var must be skipped")
	})

	t.Run("returns non-nil empty map when no service keys set", func(t *testing.T) {
		got, err := LoadServiceAPIKeysFromEnv()
		require.NoError(t, err)
		require.NotNil(t, got)
		// No MULTI_TENANT_SERVICE_API_KEY_* set in this subtest's process env beyond
		// inherited ones; assert the contract is a usable map, not nil.
		_, hasBare := got[""]
		assert.False(t, hasBare)
	})

	t.Run("normalizes lowercase/hyphen suffix to the canonical token", func(t *testing.T) {
		// FIX 1: the loader must store under normalizeServiceToken(...) so the
		// adapter's normalized lookup hits, instead of silently falling back to
		// the default (wrong credential, CWE-178).
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN-CRM", "crm-key")

		got, err := LoadServiceAPIKeysFromEnv()

		require.NoError(t, err)
		assert.Equal(t, "crm-key", got["PLUGIN_CRM"], "hyphen/lower suffix must be stored under the normalized token")
		_, raw := got["PLUGIN-CRM"]
		assert.False(t, raw, "raw un-normalized suffix must not be a map key")
	})

	t.Run("fails fast on token collision", func(t *testing.T) {
		// FIX 2: two distinct env vars that normalize to the same token must be
		// an error, not silent last-writer-wins (wrong-credential, CWE-863).
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN-CRM", "crm-key-a")
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "crm-key-b")

		got, err := LoadServiceAPIKeysFromEnv()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "PLUGIN_CRM", "error must name the colliding token")
		assert.Nil(t, got, "no map should be returned on collision")
	})
}

func TestBuildServiceClients(t *testing.T) {
	t.Run("builds one client per key plus default, recording api keys", func(t *testing.T) {
		recorded := make([]string, 0)
		newClient := func(apiKey string) (*tmclient.Client, error) {
			recorded = append(recorded, apiKey)
			return &tmclient.Client{}, nil
		}

		keys := map[string]string{
			"LEDGER":     "ledger-key",
			"PLUGIN_CRM": "crm-key",
		}

		clients, defaultClient, err := BuildServiceClients(keys, "default-key", newClient)

		require.NoError(t, err)
		require.NotNil(t, clients)
		assert.Len(t, clients, 2)
		assert.NotNil(t, clients["LEDGER"])
		assert.NotNil(t, clients["PLUGIN_CRM"])
		require.NotNil(t, defaultClient)

		assert.ElementsMatch(t, []string{"ledger-key", "crm-key", "default-key"}, recorded)
	})

	t.Run("empty default key yields nil default client and no error", func(t *testing.T) {
		newClient := func(apiKey string) (*tmclient.Client, error) {
			return &tmclient.Client{}, nil
		}

		clients, defaultClient, err := BuildServiceClients(map[string]string{"LEDGER": "k"}, "", newClient)

		require.NoError(t, err)
		assert.Len(t, clients, 1)
		assert.Nil(t, defaultClient)
	})

	t.Run("surfaces newClient error wrapped with the failing token", func(t *testing.T) {
		sentinel := errors.New("boom")
		newClient := func(apiKey string) (*tmclient.Client, error) {
			if apiKey == "bad-key" {
				return nil, sentinel
			}
			return &tmclient.Client{}, nil
		}

		keys := map[string]string{"PLUGIN_CRM": "bad-key"}

		clients, defaultClient, err := BuildServiceClients(keys, "default-key", newClient)

		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Contains(t, err.Error(), "PLUGIN_CRM")
		assert.Nil(t, clients)
		assert.Nil(t, defaultClient)
	})

	t.Run("surfaces newClient error for default key", func(t *testing.T) {
		sentinel := errors.New("default-boom")
		newClient := func(apiKey string) (*tmclient.Client, error) {
			if apiKey == "default-key" {
				return nil, sentinel
			}
			return &tmclient.Client{}, nil
		}

		clients, defaultClient, err := BuildServiceClients(map[string]string{}, "default-key", newClient)

		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, clients)
		assert.Nil(t, defaultClient)
	})

	t.Run("empty keys and empty default yields empty map and nil default", func(t *testing.T) {
		called := false
		newClient := func(apiKey string) (*tmclient.Client, error) {
			called = true
			return &tmclient.Client{}, nil
		}

		clients, defaultClient, err := BuildServiceClients(map[string]string{}, "", newClient)

		require.NoError(t, err)
		require.NotNil(t, clients)
		assert.Empty(t, clients)
		assert.Nil(t, defaultClient)
		assert.False(t, called, "newClient must not be called when there is nothing to build")
	})

	t.Run("errors when newClient returns a nil client for a service", func(t *testing.T) {
		// FIX 3: (nil, nil) from newClient must not be stored — it would be a
		// latent nil-deref at request time. Fail fast naming the service.
		newClient := func(apiKey string) (*tmclient.Client, error) {
			return nil, nil
		}

		clients, defaultClient, err := BuildServiceClients(map[string]string{"PLUGIN_CRM": "crm-key"}, "default-key", newClient)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "PLUGIN_CRM")
		assert.Nil(t, clients)
		assert.Nil(t, defaultClient)
	})

	t.Run("errors when newClient returns a nil default client", func(t *testing.T) {
		// FIX 3: same guard for the default client.
		newClient := func(apiKey string) (*tmclient.Client, error) {
			if apiKey == "default-key" {
				return nil, nil
			}
			return &tmclient.Client{}, nil
		}

		clients, defaultClient, err := BuildServiceClients(map[string]string{}, "default-key", newClient)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "default")
		assert.Nil(t, clients)
		assert.Nil(t, defaultClient)
	})
}
