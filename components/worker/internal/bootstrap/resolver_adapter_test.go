// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"errors"
	"testing"

	tmclient "github.com/LerianStudio/lib-commons/v6/commons/tenant-manager/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// TestBuildResolverTenantAdapter exercises the worker's resolver-adapter seam in
// isolation via an injected recording newClient.
//
// The worker variant DIVERGES from the manager: it receives the already-built,
// already-lifecycle-owned sharedTMClient as the default and builds ONLY the
// per-service clients. The distinguishing invariant asserted here is that the
// injected default client is NEVER passed through newClient (no second
// own-identity client is created), so `recorded` contains only per-service keys.
//
// It also asserts that the per-service clients map is surfaced to the caller so
// the Service lifecycle can register each client's Close (each per-service client
// owns an in-memory cache cleanup goroutine that would otherwise leak).
func TestBuildResolverTenantAdapter(t *testing.T) {
	// sharedTMClient stand-in: the default is injected already-built, so it must
	// never reach newClient.
	defaultClient := &tmclient.Client{}

	t.Run("reuses the injected default client when no per-service env keys", func(t *testing.T) {
		recorded := make([]string, 0)
		newClient := func(apiKey string) (*tmclient.Client, error) {
			recorded = append(recorded, apiKey)
			return &tmclient.Client{}, nil
		}

		adapter, serviceClients, err := buildResolverTenantAdapter(defaultClient, testBootstrapLogger(), newClient)

		require.NoError(t, err)
		require.NotNil(t, adapter)
		// The injected default is used as-is; newClient is never called.
		assert.Empty(t, recorded)
		// No per-service clients built, so nothing to register for Close.
		assert.Empty(t, serviceClients)
	})

	t.Run("routes a single per-service key to newClient without touching the default", func(t *testing.T) {
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "crm-key")

		recorded := make([]string, 0)
		newClient := func(apiKey string) (*tmclient.Client, error) {
			recorded = append(recorded, apiKey)
			return &tmclient.Client{}, nil
		}

		adapter, serviceClients, err := buildResolverTenantAdapter(defaultClient, testBootstrapLogger(), newClient)

		require.NoError(t, err)
		require.NotNil(t, adapter)
		// Only the per-service key reaches newClient; the default is the injected
		// sharedTMClient, never rebuilt.
		assert.ElementsMatch(t, []string{"crm-key"}, recorded)
		// The single per-service client is surfaced (keyed by normalized token) so
		// the caller can register its Close.
		require.Len(t, serviceClients, 1)
		assert.Contains(t, serviceClients, "PLUGIN_CRM")
	})

	t.Run("routes two distinct per-service keys without touching the default", func(t *testing.T) {
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "crm-key")
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_LEDGER", "ledger-key")

		recorded := make([]string, 0)
		newClient := func(apiKey string) (*tmclient.Client, error) {
			recorded = append(recorded, apiKey)
			return &tmclient.Client{}, nil
		}

		adapter, serviceClients, err := buildResolverTenantAdapter(defaultClient, testBootstrapLogger(), newClient)

		require.NoError(t, err)
		require.NotNil(t, adapter)
		assert.ElementsMatch(t, []string{"crm-key", "ledger-key"}, recorded)
		// Both per-service clients are surfaced for Close registration.
		require.Len(t, serviceClients, 2)
		assert.Contains(t, serviceClients, "PLUGIN_CRM")
		assert.Contains(t, serviceClients, "LEDGER")
	})

	t.Run("propagates loader collision error", func(t *testing.T) {
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN-CRM", "a")
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "b")

		newClient := func(apiKey string) (*tmclient.Client, error) {
			return &tmclient.Client{}, nil
		}

		adapter, serviceClients, err := buildResolverTenantAdapter(defaultClient, testBootstrapLogger(), newClient)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Nil(t, serviceClients)
		assert.Contains(t, err.Error(), "load per-service tenant manager API keys for resolver")
	})

	t.Run("propagates builder error from newClient", func(t *testing.T) {
		t.Setenv("MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM", "crm-key")

		sentinel := errors.New("boom")
		newClient := func(apiKey string) (*tmclient.Client, error) {
			return nil, sentinel
		}

		adapter, serviceClients, err := buildResolverTenantAdapter(defaultClient, testBootstrapLogger(), newClient)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Nil(t, serviceClients)
		assert.ErrorIs(t, err, sentinel)
		assert.Contains(t, err.Error(), "create per-service tenant manager clients for resolver")
	})
}

// newRealTMClient builds a real tenant-manager client (no HTTP request is made).
// A real client is required for the goroutine-leak assertion below because its
// default in-memory cache starts a cleanupLoop goroutine that only Close stops;
// a zero-value client would not start one and could not prove the fix.
func newRealTMClient(t *testing.T, apiKey string) *tmclient.Client {
	t.Helper()

	client, err := tmclient.NewClient(
		"https://tenant-manager.example.com",
		testBootstrapLogger(),
		tmclient.WithServiceAPIKey(apiKey),
	)
	require.NoError(t, err)
	require.NotNil(t, client)

	return client
}

// TestNewTMClientChainCloser_ClosesSharedAndPerServiceClients proves the leak fix:
// the chain closer stops the in-memory cache cleanup goroutine of the shared
// client AND every distinct per-service client. Under the previous wiring (only
// sharedTMClient registered for Close), the per-service clients' goroutines
// survived the closer and goleak.VerifyNone would fail.
func TestNewTMClientChainCloser_ClosesSharedAndPerServiceClients(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	shared := newRealTMClient(t, "shared-key")
	perService := map[string]*tmclient.Client{
		"PLUGIN_CRM": newRealTMClient(t, "crm-key"),
		"LEDGER":     newRealTMClient(t, "ledger-key"),
	}

	closer := newTMClientChainCloser(shared, perService)
	require.NotNil(t, closer)

	// Closing must stop every cleanup goroutine (shared + both per-service).
	require.NoError(t, closer())
}

// TestNewTMClientChainCloser_NilInputs asserts the closer is nil-safe: it returns
// a non-nil closer when only the shared client is present (parity with the prior
// single-client closer) and nil when there is nothing to close.
func TestNewTMClientChainCloser_NilInputs(t *testing.T) {
	t.Run("nil shared and empty per-service returns nil closer", func(t *testing.T) {
		assert.Nil(t, newTMClientChainCloser(nil, nil))
		assert.Nil(t, newTMClientChainCloser(nil, map[string]*tmclient.Client{}))
	})

	t.Run("shared only returns a non-nil closer", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

		closer := newTMClientChainCloser(newRealTMClient(t, "shared-key"), nil)
		require.NotNil(t, closer)
		require.NoError(t, closer())
	})
}

// TestClosePerServiceTMClientsIfUnowned covers the bootstrap-abort guard: it
// closes the per-service clients when ownership was never transferred, and is a
// no-op once the Service owns them (so the Service closer does not double-close).
func TestClosePerServiceTMClientsIfUnowned(t *testing.T) {
	t.Run("closes per-service clients when unowned (aborted bootstrap)", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

		clients := map[string]*tmclient.Client{
			"PLUGIN_CRM": newRealTMClient(t, "crm-key"),
			"LEDGER":     newRealTMClient(t, "ledger-key"),
		}

		owned := false
		closePerServiceTMClientsIfUnowned(context.Background(), testBootstrapLogger(), clients, &owned)
	})

	t.Run("is a no-op when already owned by the Service", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

		// These clients are treated as owned by the Service; the guard must NOT
		// close them here. We close them ourselves at the end so the goroutines
		// do not leak past the test.
		clients := map[string]*tmclient.Client{
			"PLUGIN_CRM": newRealTMClient(t, "crm-key"),
		}

		owned := true
		closePerServiceTMClientsIfUnowned(context.Background(), testBootstrapLogger(), clients, &owned)

		// The guard left them open (proving no double-close would occur); clean up.
		for _, c := range clients {
			require.NoError(t, c.Close())
		}
	})
}
