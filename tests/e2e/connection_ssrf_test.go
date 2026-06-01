//go:build e2e

// Package extraction — SSRF host safety E2E tests (fetcher-013).
//
// These tests verify the full request path of the SSRF guard wired in
// Marcos 1-3:
//
//   - Marco 1: pkg/datasource/hostsafety — denylist core (CIDRs, suffixes,
//     DNS resolution, internal-datasource bypass).
//   - Marco 2: pkg/datasource/datasource_factory.go — factory pre-dial guard.
//   - Marco 3: pkg/net/http/with_body.go (validator) +
//     components/manager/internal/bootstrap/config.go (bootstrap activation).
//
// The four scenarios mirror the plan §5.4:
//
//   1. POST loopback IP literal → 400 + FET-0414 (DTO `safe_host` rejects).
//   2. POST cloud-metadata IP literal → 400 + FET-0414.
//   3. POST connection with a non-resolvable hostname → 201 (guard defers
//      DNS errors to the driver per Marco 1 design — the create persists,
//      only a later /test would fail at dial). This proves the guard does
//      NOT block hostnames that fail DNS.
//   4. POST connection with a real PostgreSQL fixture + /test → 200 with
//      status=success. Proves the guard does NOT break the happy path
//      when the host is legitimate.
//
// ## Activation
//
// All tests in this file are SKIPPED unless the Manager under test is
// running with MULTI_TENANT_ENABLED=true AND the operator opts in by
// setting `E2E_SSRF_MT_ENABLED=true`. The shared E2E Manager in
// tests/shared/apps.go currently hardcodes `MULTI_TENANT_ENABLED=false`
// (see apps.go:183), which is the correct default for the rest of the
// suite. Activating these tests requires a follow-up that either:
//
//   (a) Adds an EnvOverride field to e2eshared.AppStartConfig so the
//       Manager can be spawned per-test with MT=true plus the required
//       MULTI_TENANT_URL / MULTI_TENANT_SERVICE_API_KEY fixtures, or
//   (b) Toggles MT on suite-wide behind an env var like
//       `E2E_ENABLE_MULTI_TENANT=true` and wires a tenant-manager mock
//       fixture into CoreInfra.
//
// Either path is outside Marco 4's scope (Manager-only, no shared infra
// changes). The tests below are kept as a runnable spec so a future MT
// suite-level enablement activates them atomically without rewrites.
package extraction

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ssrfErrorCode is the canonical error code emitted by the host safety guard.
// Declared as a const so a future schema change (e.g., FET-0414 → FET-0414A)
// surfaces a single edit point.
const ssrfErrorCode = "FET-0414"

// skipIfSSRFNotEnabled gates every test in this file behind explicit operator
// opt-in. The default shared E2E Manager runs with MULTI_TENANT_ENABLED=false,
// which disables the guard — under that config, scenarios 1 and 2 would NOT
// receive 400s, producing false negatives. Forcing an explicit opt-in env
// var prevents that confusion and makes the activation surface auditable.
func skipIfSSRFNotEnabled(t *testing.T) {
	t.Helper()

	if os.Getenv("E2E_SSRF_MT_ENABLED") != "true" {
		t.Skip("SSRF host safety E2E tests skipped: set E2E_SSRF_MT_ENABLED=true and run Manager with MULTI_TENANT_ENABLED=true")
	}
}

// TestConnectionSSRF_LoopbackBlocked verifies that POSTing a connection
// whose Host is an IPv4 loopback literal is rejected by the `safe_host`
// DTO validator with HTTP 400 + FET-0414. The cryptor / persistence layer
// is never reached because the validator runs at request parse time.
func TestConnectionSSRF_LoopbackBlocked(t *testing.T) {
	t.Parallel()
	skipIfSSRFNotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	connInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-ssrf-loop-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         "127.0.0.1",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	// Generic-message contract: the body must surface the error CODE so
	// clients can branch, but MUST NOT echo the host or reveal which CIDR
	// matched. AssertAPIError checks the body for ssrfErrorCode as a
	// substring — sufficient because the renderer emits the code verbatim.
	e2eshared.AssertAPIError(t, resp, 400, ssrfErrorCode)
}

// TestConnectionSSRF_CloudMetadataBlocked verifies that the AWS/GCP/Azure
// IMDS endpoint (169.254.169.254) is rejected. This is the canonical
// "credentials exfiltration" target that motivates the whole guard.
func TestConnectionSSRF_CloudMetadataBlocked(t *testing.T) {
	t.Parallel()
	skipIfSSRFNotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	connInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-ssrf-imds-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         "169.254.169.254",
		Port:         80,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 400, ssrfErrorCode)
}

// TestConnectionSSRF_RFC1918Blocked exercises the third major denylist
// family (private RFC1918 addresses). Picks 10.0.0.1 as a representative
// of the 10/8 range — the largest tenant-side privately-routable network.
func TestConnectionSSRF_RFC1918Blocked(t *testing.T) {
	t.Parallel()
	skipIfSSRFNotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	connInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-ssrf-rfc1918-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         "10.0.0.1",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	resp, err := apiClient.CreateConnectionRaw(ctx, productName, connInput)
	require.NoError(t, err, "request should succeed")

	e2eshared.AssertAPIError(t, resp, 400, ssrfErrorCode)
}

// TestConnectionSSRF_PublicHostnameAccepted verifies that a non-resolvable
// public hostname passes BOTH the DTO `safe_host` validator (which only
// screens IP literals) AND the factory's DNS-aware guard (which lets DNS
// failures through per Marco 1 design — the "no oracle for non-existent
// hosts" decision).
//
// Why a non-resolvable host instead of a real public IP:
//   - Using 8.8.8.8 + port 5432 would pass the guard, then the factory
//     opens a real PostgreSQL dial that depending on outbound network
//     policies might (a) succeed against a misconfigured Google server
//     (nonsense), (b) hang on RTT, or (c) be NAT-rewritten by the test
//     container's egress. None of these are deterministic.
//   - A bogus TLD like `.test` is reserved by RFC 2606 specifically for
//     tests, NXDOMAIN-stable, and proves the DNS-error bypass branch.
//
// The factory will queue the connection as persisted (the guard returned
// nil on resolver failure). A subsequent /test would surface the driver
// error — but that's Scenario 4's job, not this one's.
func TestConnectionSSRF_PublicHostnameAccepted(t *testing.T) {
	t.Parallel()
	skipIfSSRFNotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	productName := e2eshared.GenerateProductName()

	connInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-ssrf-pub-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         fmt.Sprintf("nx-%s.example.test", uuid.New().String()[:8]),
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "public hostname must pass the guard")
	require.NotNil(t, conn)

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	assert.Equal(t, connInput.Host, conn.Host, "host should be persisted verbatim")
	assert.NotEmpty(t, conn.ID, "connection should receive a server-generated ID")
}

// TestConnectionSSRF_HappyPathTestConnection is the regression guard: a
// connection pointing at the real PostgreSQL fixture (legitimate host,
// reachable via the E2E network) must continue to work end-to-end with
// the SSRF guard active. This proves Marcos 1-3 did not break the happy
// path when MT is enabled.
//
// Note: pgHost from postgresInfra resolves to an in-network address that
// MAY fall in a private CIDR (Docker bridge typically uses 172.16/12 or
// 10/8). If the guard rejects it, that's a real bug worth investigating —
// the test surfaces it via require.NoError.
func TestConnectionSSRF_HappyPathTestConnection(t *testing.T) {
	t.Parallel()
	skipIfSSRFNotEnabled(t)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	productName := e2eshared.GenerateProductName()

	connInput := e2eshared.ConnectionInput{
		ConfigName:   fmt.Sprintf("e2e-ssrf-happy-%s", uuid.New().String()[:8]),
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, connInput)

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "connection must become available — guard did not block fixture host")

	result, err := apiClient.TestConnection(ctx, conn.ID)
	require.NoError(t, err, "POST /test on legitimate connection must succeed")
	assert.Equal(t, "success", result.Status, "test connection status should be success")
	assert.Greater(t, result.LatencyMs, int64(0), "latency should be positive")
}
