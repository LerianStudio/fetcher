package hostsafety

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withGuardState toggles the global gate for one subtest and restores both the
// flag and the resolver afterward. Package-level state is global by design
// (matches production reality where SetHostSafetyEnabled is called once at
// bootstrap) — these tests must run serially, not via t.Parallel().
func withGuardState(t *testing.T, enabled bool, resolver lookupFunc) {
	t.Helper()

	prevFlag := hostSafetyEnabled.Load()
	prevResolver := hostResolver

	hostSafetyEnabled.Store(enabled)
	if resolver != nil {
		hostResolver = resolver
	}

	t.Cleanup(func() {
		hostSafetyEnabled.Store(prevFlag)
		hostResolver = prevResolver
	})
}

// fakeLookup returns a lookupFunc that yields the given IPs deterministically.
// Mirrors the helper used in lib-commons/commons/security/ssrf/ssrf_test.go.
func fakeLookup(ips ...net.IPAddr) lookupFunc {
	return func(_ context.Context, _ string) ([]net.IPAddr, error) {
		return ips, nil
	}
}

func failLookup(err error) lookupFunc {
	return func(_ context.Context, _ string) ([]net.IPAddr, error) {
		return nil, err
	}
}

func TestSetHostSafetyEnabledTogglesFlag(t *testing.T) {
	prev := hostSafetyEnabled.Load()
	t.Cleanup(func() { hostSafetyEnabled.Store(prev) })

	SetHostSafetyEnabled(true)
	assert.True(t, IsEnabled(), "IsEnabled must reflect Set true")

	SetHostSafetyEnabled(false)
	assert.False(t, IsEnabled(), "IsEnabled must reflect Set false")
}

// TestValidateHostForConnection_GuardOff covers the master kill-switch: when
// the flag is off NO check runs, even for the most dangerous hosts. Production
// single-tenant deployments depend on this branch.
func TestValidateHostForConnection_GuardOff(t *testing.T) {
	// Wire a stub that would explode if the code reached it.
	withGuardState(t, false, failLookup(errors.New("resolver must not be called when guard is off")))

	dangerous := []string{
		"127.0.0.1",
		"169.254.169.254",
		"10.0.0.1",
		"metadata.google.internal",
		"foo.svc.cluster.local",
	}

	for _, host := range dangerous {
		t.Run(host, func(t *testing.T) {
			conn := &model.Connection{Host: host, EncryptionKeyVersion: "v1"}
			assert.NoError(t, ValidateHostForConnection(context.Background(), conn))
		})
	}
}

// TestValidateHostForConnection_NilConnection ensures the nil-conn branch
// returns nil silently (callers can pass *Connection without guarding).
func TestValidateHostForConnection_NilConnection(t *testing.T) {
	withGuardState(t, true, failLookup(errors.New("resolver must not be called for nil conn")))

	assert.NoError(t, ValidateHostForConnection(context.Background(), nil))
}

// TestValidateHostForConnection_InternalDatasourceBypass covers the
// EncryptionKeyVersion == "" exemption. Internal datasources loaded by the
// platform operator via env_loader.go and the multi-tenant resolver are
// trusted by construction and intentionally exempt from the guard.
func TestValidateHostForConnection_InternalDatasourceBypass(t *testing.T) {
	withGuardState(t, true, failLookup(errors.New("resolver must not be called for internal datasource")))

	conn := &model.Connection{
		Host:                 "127.0.0.1", // would be blocked if guard ran
		EncryptionKeyVersion: "",          // marker for internal datasource
	}
	assert.NoError(t, ValidateHostForConnection(context.Background(), conn))
}

// TestValidateHostForConnection_BlockedHostnameLiteral verifies the
// hostname-literal short-circuit. libSSRF.IsBlockedHostname catches names
// like "localhost", "metadata.google.internal", suffixes ".local", ".internal",
// ".cluster.local" — we delegate to it and never need to consult DNS.
func TestValidateHostForConnection_BlockedHostnameLiteral(t *testing.T) {
	cases := []string{
		"metadata.google.internal",
		"foo.svc.cluster.local",
		"some-node.cluster.local",
		"localhost",
	}

	for _, host := range cases {
		t.Run(host, func(t *testing.T) {
			// Resolver must NOT be consulted — hostname-literal check short-circuits first.
			withGuardState(t, true, failLookup(errors.New("resolver must not be called for blocked hostname literal")))

			conn := &model.Connection{Host: host, EncryptionKeyVersion: "v1"}
			err := ValidateHostForConnection(context.Background(), conn)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "FET-0414")
		})
	}
}

// TestValidateHostForConnection_BlockedIPLiteral verifies that IP literals
// in Host are checked directly (no DNS round-trip) and rejected. Covers the
// common attacker pattern of just typing "127.0.0.1" or "169.254.169.254".
func TestValidateHostForConnection_BlockedIPLiteral(t *testing.T) {
	cases := []string{
		"127.0.0.1",
		"10.0.0.1",
		"169.254.169.254",
		"::1",
	}

	for _, host := range cases {
		t.Run(host, func(t *testing.T) {
			// Resolver must NOT be consulted for IP literals.
			withGuardState(t, true, failLookup(errors.New("resolver must not be called for IP literal")))

			conn := &model.Connection{Host: host, EncryptionKeyVersion: "v1"}
			err := ValidateHostForConnection(context.Background(), conn)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "FET-0414")
		})
	}
}

// TestValidateHostForConnection_BlockedAfterResolution verifies the
// hostname-resolves-to-internal-IP path. A hostname that passes the literal
// check but resolves to a denylisted IP is still blocked.
func TestValidateHostForConnection_BlockedAfterResolution(t *testing.T) {
	withGuardState(t, true, fakeLookup(net.IPAddr{IP: net.ParseIP("127.0.0.1")}))

	conn := &model.Connection{Host: "db.example.com", EncryptionKeyVersion: "v1"}
	err := ValidateHostForConnection(context.Background(), conn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FET-0414")
}

// TestValidateHostForConnection_PublicHostnameAllowed is the happy path:
// hostname literal not in libSSRF's blocklist + DNS resolves to a public IP
// → no error, factory proceeds to dial.
func TestValidateHostForConnection_PublicHostnameAllowed(t *testing.T) {
	withGuardState(t, true, fakeLookup(net.IPAddr{IP: net.ParseIP("93.184.216.34")}))

	conn := &model.Connection{Host: "db.example.com", EncryptionKeyVersion: "v1"}
	assert.NoError(t, ValidateHostForConnection(context.Background(), conn))
}

// TestValidateHostForConnection_DNSErrorPassesThrough documents the
// intentional pass-through on DNS errors. Forcing a 400 here would leak a
// different error than for "doesn't exist", recreating the same reconnaissance
// oracle the guard is built to close. The driver dial that follows surfaces
// its own connect error.
func TestValidateHostForConnection_DNSErrorPassesThrough(t *testing.T) {
	withGuardState(t, true, failLookup(errors.New("NXDOMAIN")))

	conn := &model.Connection{Host: "nx.example.test", EncryptionKeyVersion: "v1"}
	assert.NoError(t, ValidateHostForConnection(context.Background(), conn))
}

// TestValidateSafeHostString covers the DTO-layer screening of IP literals.
// Hostnames always pass at this layer (the factory's DNS-aware guard catches
// them downstream).
func TestValidateSafeHostString(t *testing.T) {
	t.Run("guard off accepts any literal", func(t *testing.T) {
		withGuardState(t, false, nil)
		assert.True(t, ValidateSafeHostString("127.0.0.1"))
		assert.True(t, ValidateSafeHostString("10.0.0.1"))
		assert.True(t, ValidateSafeHostString("metadata.google.internal"))
	})

	t.Run("guard on rejects forbidden IP literals", func(t *testing.T) {
		withGuardState(t, true, nil)
		assert.False(t, ValidateSafeHostString("127.0.0.1"))
		assert.False(t, ValidateSafeHostString("169.254.169.254"))
		assert.False(t, ValidateSafeHostString("10.0.0.1"))
		assert.False(t, ValidateSafeHostString("::1"))
		assert.False(t, ValidateSafeHostString("::ffff:127.0.0.1"))
	})

	t.Run("guard on accepts public IP literal", func(t *testing.T) {
		withGuardState(t, true, nil)
		assert.True(t, ValidateSafeHostString("1.1.1.1"))
		assert.True(t, ValidateSafeHostString("8.8.8.8"))
	})

	t.Run("guard on passes hostnames through to factory layer", func(t *testing.T) {
		withGuardState(t, true, nil)
		assert.True(t, ValidateSafeHostString("db.example.com"))
		// Even hostnames in libSSRF's blocklist pass the literal-only screen;
		// the factory's hostname-literal check catches them at the next layer.
		assert.True(t, ValidateSafeHostString("metadata.google.internal"))
	})

	t.Run("trims whitespace before parsing", func(t *testing.T) {
		withGuardState(t, true, nil)
		assert.False(t, ValidateSafeHostString("  127.0.0.1  "))
	})
}
