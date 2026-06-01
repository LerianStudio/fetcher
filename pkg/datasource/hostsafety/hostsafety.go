// Package hostsafety is a thin adapter over
// github.com/LerianStudio/lib-commons/v5/commons/security/ssrf that adds two
// Fetcher-specific concerns the canonical library does NOT have:
//
//  1. A bootstrap-time gate (SetHostSafetyEnabled / IsEnabled) so the guard is
//     active only when MULTI_TENANT_ENABLED=true. Single-tenant operators
//     legitimately point at internal hosts and must see zero behavior change.
//  2. The "internal datasource" carve-out — connections loaded by the platform
//     operator via env_loader.go or the multi-tenant resolver carry
//     EncryptionKeyVersion == "" and are trusted by construction.
//
// All blocklist semantics (CIDRs, hostname literals, IPv4-mapped IPv6, etc.)
// live in libSSRF. When that package's denylist is updated, Fetcher inherits
// the update on the next `go mod tidy` — no fork to maintain here.
//
// This package is intentionally a leaf — it does not import pkg/mongodb,
// pkg/net/http, or pkg/datasource itself, so both the datasource factory and
// the HTTP DTO validator can depend on it without creating an import cycle.
package hostsafety

import (
	"context"
	"net"
	"strings"
	"sync/atomic"

	libSSRF "github.com/LerianStudio/lib-commons/v5/commons/security/ssrf"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
)

// lookupFunc is the resolver indirection point. Tests inject a deterministic
// stub via package-private assignment to `hostResolver`. We use this rather
// than libSSRF.WithLookupFunc because the latter is tied to the higher-level
// ValidateURL/ResolveAndValidate API that takes a full URL — our flow takes a
// raw hostname.
type lookupFunc func(ctx context.Context, host string) ([]net.IPAddr, error)

var (
	// hostSafetyEnabled gates every entry point. atomic.Bool because the flag
	// is read on every connection-creation request and may in principle be
	// flipped by a future runtime-config feature; sync-safe by default avoids
	// a data-race subclass of bug that `go test -race` would flag intermittently.
	hostSafetyEnabled atomic.Bool

	// hostResolver is the default DNS resolver. Tests reassign it to a stub.
	// Production never touches this — it's package-private.
	hostResolver lookupFunc = net.DefaultResolver.LookupIPAddr
)

// SetHostSafetyEnabled toggles the SSRF guard. Manager bootstrap calls this
// once at startup from cfg.MultiTenantEnabled. The Worker does NOT call it
// because all Connection writes flow through the Manager (see PROJECT_RULES.md
// § "Security: Host Safety / SSRF Guard").
func SetHostSafetyEnabled(enabled bool) {
	hostSafetyEnabled.Store(enabled)
}

// IsEnabled reports the current state of the guard. Useful for bootstrap tests
// and observability hooks that want to confirm the flag was wired correctly.
func IsEnabled() bool {
	return hostSafetyEnabled.Load()
}

// ValidateHostForConnection is the entry point used by the datasource factory.
// It rejects connections whose Host is denylisted by libSSRF (literal IP,
// literal hostname, or any resolved IP), but only when:
//
//   - the guard is enabled (SetHostSafetyEnabled(true) was called), AND
//   - the connection was supplied by a tenant (EncryptionKeyVersion != "").
//
// Internal datasources loaded from environment variables by platform operators
// have EncryptionKeyVersion == "" by construction (see env_loader.go and
// multi_tenant.go resolvers) and are exempt by design.
func ValidateHostForConnection(ctx context.Context, conn *model.Connection) error {
	if !hostSafetyEnabled.Load() || conn == nil || conn.EncryptionKeyVersion == "" {
		return nil
	}

	host := strings.TrimSpace(conn.Host)

	// Hostname-literal check first: catches "localhost", "metadata.google.internal",
	// suffixes ".local" / ".internal" / ".cluster.local" without consulting DNS.
	if libSSRF.IsBlockedHostname(host) {
		return pkg.ValidateBusinessError(constant.ErrForbiddenHost, "connection")
	}

	// IP literal? Check directly — no DNS round-trip needed.
	if ip := net.ParseIP(host); ip != nil {
		if libSSRF.IsBlockedIP(ip) {
			return pkg.ValidateBusinessError(constant.ErrForbiddenHost, "connection")
		}

		return nil
	}

	// Hostname (non-literal): resolve and check every returned IP.
	addrs, err := hostResolver(ctx, host)
	if err != nil {
		// Resolver failure (NXDOMAIN, timeout, etc.) is intentionally NOT
		// treated as a block. Forcing a 400 here would (1) leak a different
		// error than for "doesn't exist", recreating the same reconnaissance
		// oracle this guard exists to close, and (2) produce false negatives
		// during transient DNS hiccups. The driver dial that follows surfaces
		// its own connect error.
		return nil //nolint:nilerr // intentional: see comment above
	}

	for _, addr := range addrs {
		if libSSRF.IsBlockedIP(addr.IP) {
			return pkg.ValidateBusinessError(constant.ErrForbiddenHost, "connection")
		}
	}

	return nil
}

// ValidateSafeHostString is the DTO-layer validator (registered as the
// `safe_host` validator tag). It rejects IP literals that fall in libSSRF's
// denylist without consulting DNS — that catches the obvious "Host: 127.0.0.1"
// case at request parse time and gives the client a clean 400 before any
// service logic runs. Hostnames always return true at this layer; the
// factory's DNS-aware ValidateHostForConnection handles them downstream.
func ValidateSafeHostString(value string) bool {
	if !hostSafetyEnabled.Load() {
		return true
	}

	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return true
	}

	return !libSSRF.IsBlockedIP(ip)
}
