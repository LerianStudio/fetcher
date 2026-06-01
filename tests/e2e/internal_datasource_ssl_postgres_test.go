//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/postgres"
	"github.com/LerianStudio/fetcher/pkg/resolver"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/LerianStudio/fetcher/tests/shared/fixtures/ssl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// These tests reproduce the LaFinteca incident (AWS RDS PostgreSQL with
// hostssl-only pg_hba) and validate the fetcher-012 fix in pkg/resolver/env_loader.go.
// They cover:
//
//   - The bug reproduction: internal datasource configured WITHOUT _SSLMODE,
//     pointed at a TLS-required postgres → connection test returns HTTP 500
//     "Database Connection Error" (driver-level SQLSTATE 28000 in Manager logs).
//   - The fix: same setup with DATASOURCE_*_SSLMODE=require → connection test
//     succeeds (HTTP 200, status=success).
//   - Backward-compat: internal datasource WITHOUT _SSLMODE pointed at plain
//     postgres (no TLS) → connection test still succeeds, protecting existing
//     non-managed deployments.
//
// Each test brings up its own postgres container (different Name → different
// network alias) and spawns a dedicated Manager via AppStartConfig.ExtraEnv to
// inject the DATASOURCE_MIDAZ_ONBOARDING_* env vars per-scenario. This keeps
// the shared managerApp untouched and lets the tests run with t.Parallel().

const (
	// sslInternalConfigName is one of the registered internal datasource
	// configNames (see pkg/resolver/registry.go). We use it for the SSL
	// e2e because the registry already contains it and the deterministic
	// UUID lookup goes through TestConnection.Execute → registry.FindConfigByID.
	sslInternalConfigName = "midaz_onboarding"

	// sslInternalEnvPrefix is the uppercased prefix the env loader scans
	// (pkg/resolver/env_loader.go). It must match the configName segment
	// of DATASOURCE_{NAME}_CONFIG_NAME exactly.
	sslInternalEnvPrefix = "DATASOURCE_MIDAZ_ONBOARDING"

	sslTestDatabase = "onboarding"
	sslTestUser     = "ssluser"
	sslTestPassword = "sslpass"
)

// internalSSLAppEnv rebuilds the AppEnv against the shared core infra so the
// per-test Manager joins the same Docker network as everything else.
func internalSSLAppEnv(t *testing.T) *e2eshared.AppEnv {
	t.Helper()

	appEnv, err := e2eshared.BuildAppEnv(
		suite.Network(),
		coreInfra.MongoDB,
		coreInfra.RabbitMQ,
		coreInfra.Redis,
		coreInfra.SeaweedFS,
		coreInfra.Minio,
	)
	require.NoError(t, err, "rebuild app env for internal-ssl manager")

	return appEnv
}

// startTLSPostgres provisions a fresh postgres container that REQUIRES TLS via
// the custom postgres.conf + pg_hba_ssl_only.conf fixture pair. The container
// is attached to the shared suite network under the alias "postgres-<name>".
//
// Returns the (host, port, alias) so callers can build DATASOURCE_*_HOST/PORT
// env vars; certs are written to t.TempDir() (cleaned by the test runtime).
func startTLSPostgres(t *testing.T, ctx context.Context, name string) (alias string, port int) {
	t.Helper()

	// 1. Generate a CA + server cert/key bundle. Hostname verification is OFF
	//    with sslmode=require, so we do not need the alias in ServerDNSNames.
	bundle, err := ssl.GenerateCertificates(ssl.DefaultGenerateOptions())
	require.NoError(t, err, "generate SSL certificate bundle")

	certDir := t.TempDir()
	require.NoError(t, bundle.WriteToDir(certDir), "write SSL certs to temp dir")

	// 2. Stand up postgres with:
	//      - WithSSLCert: mounts CA/server cert/key at the testcontainers paths
	//        referenced by tests/shared/fixtures/ssl/postgres.conf.
	//      - WithConfigFile: mounts our postgres.conf at /etc/postgresql.conf
	//        and passes -c config_file=/etc/postgresql.conf to the postmaster.
	//      - CCopyFile: mounts pg_hba_ssl_only.conf at /etc/postgresql/pg_hba.conf
	//        (the path referenced by hba_file in postgres.conf). The combination
	//        rejects plaintext network connections, mimicking AWS RDS hostssl.
	infra := postgres.NewPostgresInfra(postgres.PostgresConfig{
		Name:     name,
		Database: sslTestDatabase,
		Username: sslTestUser,
		Password: sslTestPassword,
		// DSN field is unused here (we connect via Manager) — keep SSLMode
		// empty so the helper retains backward-compat defaults.
		Options: []postgres.PostgresOption{
			postgres.WithPGCustomizers(
				pg.WithSSLCert(bundle.CACertPath, bundle.ServerCertPath, bundle.ServerKeyPath),
				pg.WithConfigFile(fixturesPath("ssl/postgres.conf")),
				// 0o644 (not 0o600): testcontainers mounts the file as root:root inside
				// the container, but postgres runs as UID 70 (the "postgres" user) and
				// must be able to read pg_hba.conf at startup. 0o600 = "only root reads"
				// = postmaster aborts with `could not open file ... Permission denied`
				// + `FATAL: could not load /etc/postgresql/pg_hba.conf`. 0o644 keeps the
				// file world-readable, matching what the postgres base image ships for
				// its own pg_hba.conf inside PGDATA.
				itestkit.CCopyFile(fixturesPath("ssl/pg_hba_ssl_only.conf"), "/etc/postgresql/pg_hba.conf", 0o644),
			),
		},
	})

	require.NoError(t, infra.Start(ctx, suite.Env()), "start TLS-required postgres")

	t.Cleanup(func() {
		_ = infra.Terminate(context.Background())
	})

	host, p, err := infra.HostPort()
	require.NoError(t, err, "TLS postgres host/port")

	return host, p
}

// internalDatasourceEnv builds the DATASOURCE_MIDAZ_ONBOARDING_* env-var set
// the resolver scans at Manager boot. sslMode == "" omits SSLMODE entirely
// (reproducing the LaFinteca misconfiguration); a non-empty value gates the
// SSL parser in pkg/resolver/env_loader.go.
func internalDatasourceEnv(host string, port int, sslMode string) map[string]string {
	env := map[string]string{
		sslInternalEnvPrefix + "_CONFIG_NAME": sslInternalConfigName,
		sslInternalEnvPrefix + "_TYPE":        "postgresql",
		sslInternalEnvPrefix + "_HOST":        host,
		sslInternalEnvPrefix + "_PORT":        fmt.Sprintf("%d", port),
		sslInternalEnvPrefix + "_DATABASE":    sslTestDatabase,
		sslInternalEnvPrefix + "_USER":        sslTestUser,
		sslInternalEnvPrefix + "_PASSWORD":    sslTestPassword,
	}

	if sslMode != "" {
		env[sslInternalEnvPrefix+"_SSLMODE"] = sslMode
	}

	return env
}

// deterministicInternalID returns the UUID the Manager assigns to the internal
// datasource in single-tenant mode. tenantID is empty because MULTI_TENANT_ENABLED
// is "false" in the e2e Manager env (tests/shared/apps.go ManagerEnv()), and
// pkg/services/query/test_connection.go calls FindConfigByID with the empty
// tenant context that yields.
func deterministicInternalID() string {
	registry := resolver.NewInternalDatasourceRegistry()

	return registry.GetDeterministicID("", sslInternalConfigName).String()
}

// TestInternalDatasourceSSL_Postgres_NoSSLMode_AgainstHostssl_Reproduces28000
// reproduces the LaFinteca incident: an internal datasource configured WITHOUT
// _SSLMODE is pointed at a TLS-required postgres. The connection test must
// fail with HTTP 500 / "Database Connection Error" — equivalent to the
// observed pg_hba SQLSTATE 28000 ("no pg_hba.conf entry … no encryption").
func TestInternalDatasourceSSL_Postgres_NoSSLMode_AgainstHostssl_Reproduces28000(t *testing.T) {
	t.Parallel()

	_ = e2eshared.GenerateProductName() // isolation sanity (unused in negative path)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	host, port := startTLSPostgres(t, ctx, "internal-ssl-reproduce")

	appEnv := internalSSLAppEnv(t)

	t.Log("Spawning dedicated Manager — DATASOURCE_*_SSLMODE intentionally omitted to reproduce the LaFinteca bug")

	mgr, err := e2eshared.StartManager(t, ctx, appEnv, e2eshared.AppStartConfig{
		Image:     managerImage(),
		SkipBuild: skipBuild(),
		ExtraEnv:  internalDatasourceEnv(host, port, ""),
	})
	require.NoError(t, err, "start manager for sslmode-missing scenario")

	t.Cleanup(func() {
		_ = mgr.Container.Terminate(context.Background())
	})

	client := e2eshared.NewClientFromApp(mgr)

	resp, err := client.TestConnectionRaw(ctx, deterministicInternalID())
	require.NoError(t, err, "POST /test request must reach Manager")

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode(),
		"missing SSLMODE against hostssl postgres must surface as HTTP 500 (Database Connection Error)")

	body := string(resp.Body())
	assert.True(t, strings.Contains(body, "Database Connection Error"),
		"response body must carry the Database Connection Error title, got: %s", body)

	t.Logf("Reproduced LaFinteca bug: status=%d body=%s", resp.StatusCode(), body)

	// /schema is the endpoint that actually triggered the LaFinteca incident
	// (SQLSTATE 28000 on the schema listing query). Both /test and /schema
	// share the same factory error mapping in get_connection_schema.go /
	// test_connection.go (HTTP 500 + "Database Connection Error"), so the
	// schema path must reproduce the exact same failure surface.
	schemaResp, err := client.GetConnectionSchemaRaw(ctx, deterministicInternalID())
	require.NoError(t, err, "GET /schema request must reach Manager")

	require.Equal(t, http.StatusInternalServerError, schemaResp.StatusCode(),
		"missing SSLMODE against hostssl postgres must surface as HTTP 500 on /schema as well")

	schemaBody := string(schemaResp.Body())
	assert.True(t, strings.Contains(schemaBody, "Database Connection Error"),
		"schema response body must carry the Database Connection Error title, got: %s", schemaBody)

	t.Logf("Reproduced LaFinteca bug on /schema: status=%d body=%s", schemaResp.StatusCode(), schemaBody)
}

// TestInternalDatasourceSSL_Postgres_SSLModeRequire_AgainstHostssl_Success
// validates the fix: with DATASOURCE_*_SSLMODE=require, the same TLS-required
// postgres accepts the connection and the Manager reports success.
func TestInternalDatasourceSSL_Postgres_SSLModeRequire_AgainstHostssl_Success(t *testing.T) {
	t.Parallel()

	_ = e2eshared.GenerateProductName()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	host, port := startTLSPostgres(t, ctx, "internal-ssl-success")

	appEnv := internalSSLAppEnv(t)

	t.Log("Spawning dedicated Manager with DATASOURCE_*_SSLMODE=require — the fetcher-012 fix")

	mgr, err := e2eshared.StartManager(t, ctx, appEnv, e2eshared.AppStartConfig{
		Image:     managerImage(),
		SkipBuild: skipBuild(),
		ExtraEnv:  internalDatasourceEnv(host, port, "require"),
	})
	require.NoError(t, err, "start manager for sslmode=require scenario")

	t.Cleanup(func() {
		_ = mgr.Container.Terminate(context.Background())
	})

	client := e2eshared.NewClientFromApp(mgr)

	// Give the TLS handshake a beat — testcontainers WaitHTTP guards Manager
	// /health but the internal datasource is only exercised on /test.
	deadline := time.Now().Add(15 * time.Second)

	var (
		result    *e2eshared.TestConnectionResult
		lastError error
	)

	for time.Now().Before(deadline) {
		result, lastError = client.TestConnection(ctx, deterministicInternalID())
		if lastError == nil && result != nil && result.Status == "success" {
			break
		}

		select {
		case <-ctx.Done():
			t.Fatalf("ctx cancelled while waiting for SSL connection test: %v", ctx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}

	require.NoError(t, lastError, "test connection with sslmode=require must eventually succeed")
	require.NotNil(t, result, "test connection result must not be nil")
	assert.Equal(t, "success", result.Status, "status must be success once TLS handshake completes")
	assert.Greater(t, result.LatencyMs, int64(0), "latency must be positive on a real round-trip")

	t.Logf("fetcher-012 fix validated: sslmode=require negotiated TLS with hostssl postgres (latency=%dms)", result.LatencyMs)

	// /schema reproduces the exact production endpoint that failed at LaFinteca.
	// With the connection now healthy over TLS, the schema call must succeed.
	// The target database (sslTestDatabase) is empty — testcontainers postgres
	// only auto-creates the database, no user tables. After get_connection_schema.go
	// filters pg_* / information_schema, the resulting Tables slice may be empty.
	// We assert structurally (non-nil response) rather than NotEmpty(Tables).
	schema, err := client.GetConnectionSchema(ctx, deterministicInternalID())
	require.NoError(t, err, "GET /schema over TLS with sslmode=require must succeed")
	require.NotNil(t, schema, "schema response must not be nil")
	assert.NotNil(t, schema.Tables, "schema.Tables slice must be present (may be empty for an unseeded database)")

	t.Logf("fetcher-012 fix validated on /schema: returned %d tables over TLS", len(schema.Tables))
}

// TestInternalDatasourceSSL_Postgres_NoSSL_AgainstPlainPostgres_BackwardCompat
// protects existing deployments: when an internal datasource is configured
// WITHOUT _SSLMODE and points at a plain (non-TLS-required) postgres, the
// connection test must still succeed. This is the path used by every
// pre-fetcher-012 deployment running self-hosted postgres on a trusted network.
func TestInternalDatasourceSSL_Postgres_NoSSL_AgainstPlainPostgres_BackwardCompat(t *testing.T) {
	t.Parallel()

	_ = e2eshared.GenerateProductName()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Reuse the shared plain postgres (postgresInfra) — it accepts plaintext
	// and is the equivalent of a self-hosted postgres on a trusted VPC.
	host, port, err := postgresInfra.HostPort()
	require.NoError(t, err, "shared plain postgres host/port")

	appEnv := internalSSLAppEnv(t)

	// The shared postgresInfra was created with Database="testdb"/Username="testuser"/
	// Password="testpass" (see main_test.go). Override the per-test creds so the
	// internal datasource matches what the shared container expects.
	envOverrides := internalDatasourceEnv(host, port, "")
	envOverrides[sslInternalEnvPrefix+"_DATABASE"] = "testdb"
	envOverrides[sslInternalEnvPrefix+"_USER"] = "testuser"
	envOverrides[sslInternalEnvPrefix+"_PASSWORD"] = "testpass"

	t.Log("Spawning dedicated Manager without SSLMODE against plain postgres (backward-compat path)")

	mgr, err := e2eshared.StartManager(t, ctx, appEnv, e2eshared.AppStartConfig{
		Image:     managerImage(),
		SkipBuild: skipBuild(),
		ExtraEnv:  envOverrides,
	})
	require.NoError(t, err, "start manager for backward-compat scenario")

	t.Cleanup(func() {
		_ = mgr.Container.Terminate(context.Background())
	})

	client := e2eshared.NewClientFromApp(mgr)

	result, err := client.TestConnection(ctx, deterministicInternalID())
	require.NoError(t, err, "plaintext path must still succeed")
	require.NotNil(t, result, "test connection result must not be nil")
	assert.Equal(t, "success", result.Status, "no-SSL + plain postgres must remain a valid configuration")
	assert.Greater(t, result.LatencyMs, int64(0), "latency must be positive")

	t.Logf("backward-compat protected: no-SSL path against plain postgres still works (latency=%dms)", result.LatencyMs)

	// /schema must also stay healthy on the backward-compat path — this is the
	// endpoint LaFinteca actually hit in production. The shared postgresInfra
	// ("testdb") is not seeded by these tests with rich schemas, so we assert
	// structurally (non-nil response) rather than NotEmpty(Tables).
	schema, err := client.GetConnectionSchema(ctx, deterministicInternalID())
	require.NoError(t, err, "GET /schema on the no-SSL backward-compat path must still succeed")
	require.NotNil(t, schema, "schema response must not be nil")
	assert.NotNil(t, schema.Tables, "schema.Tables slice must be present (may be empty depending on testdb contents)")

	t.Logf("backward-compat protected on /schema: returned %d tables over plaintext", len(schema.Tables))
}
