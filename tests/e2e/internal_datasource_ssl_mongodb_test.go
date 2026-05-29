//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/mongodb"
	"github.com/LerianStudio/fetcher/pkg/resolver"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/LerianStudio/fetcher/tests/shared/fixtures/ssl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are the MongoDB counterpart of internal_datasource_ssl_postgres_test.go.
// They validate the fetcher-012 SSL/TLS env-loader fix end-to-end for the
// plugin_crm internal datasource (MongoDB) by exercising the
// POST /v1/management/connections/{id}/test endpoint:
//
//   - Success path: TLS-required mongo + DATASOURCE_PLUGIN_CRM_SSLMODE=insecure.
//     The "insecure" mode is the only allowlisted value (see
//     pkg/datasource/sslmode/mongodb.go) that triggers tlsInsecure=true in
//     appendMongoDBSSLParams (pkg/datasource/datasource_factory.go), which is
//     what we need against a self-signed test cert.
//   - Bug reproduction: same TLS-required mongo, but SSLMODE omitted.
//     conn.SSL stays nil → URI has no tls=true → plaintext attempt against
//     a requireTLS server → HTTP 500 / "Database Connection Error".
//   - Backward-compat: plain (non-TLS) mongo + SSLMODE omitted → still succeeds.
//
// Each TLS test spins its own mongo container (different Name → different
// network alias) so it can run with t.Parallel(). The backward-compat scenario
// reuses the shared coreInfra.MongoDB to avoid a third mongo container per run.

const (
	// sslMongoConfigName is the registered MongoDB internal datasource configName
	// (see pkg/resolver/registry.go).
	sslMongoConfigName = "plugin_crm"

	// sslMongoEnvPrefix is the uppercased prefix the env loader scans
	// (pkg/resolver/env_loader.go). It must match the configName segment
	// of DATASOURCE_{NAME}_CONFIG_NAME exactly.
	sslMongoEnvPrefix = "DATASOURCE_PLUGIN_CRM"

	sslMongoDatabase = "crm"
	sslMongoUser     = "ssluser"
	sslMongoPassword = "sslpass"

	// sslMongoCombinedPEMPath is where the combined cert+key PEM is mounted
	// inside the mongo container — referenced by --tlsCertificateKeyFile.
	sslMongoCombinedPEMPath = "/etc/mongo/server.pem"

	// sslMongoCAPath is where the CA cert PEM is mounted inside the mongo
	// container — referenced by --tlsCAFile. mongo 7 refuses to start with
	// --tlsCertificateKeyFile alone ("InvalidOptions: The use of TLS without
	// specifying a chain of trust is no longer supported" — SERVER-72839), so
	// the self-signed CA must be supplied even though the fetcher connects with
	// tlsInsecure=true and won't actually verify the chain.
	sslMongoCAPath = "/etc/mongo/ca.crt"
)

// internalSSLMongoAppEnv rebuilds the AppEnv against the shared core infra so
// the per-test Manager joins the same Docker network as everything else.
// Mirrors internalSSLAppEnv from the postgres SSL e2e.
func internalSSLMongoAppEnv(t *testing.T) *e2eshared.AppEnv {
	t.Helper()

	appEnv, err := e2eshared.BuildAppEnv(
		suite.Network(),
		coreInfra.MongoDB,
		coreInfra.RabbitMQ,
		coreInfra.Redis,
		coreInfra.SeaweedFS,
		coreInfra.Minio,
	)
	require.NoError(t, err, "rebuild app env for internal-ssl mongo manager")

	return appEnv
}

// writeCombinedMongoPEM writes the server cert and key to a single PEM file
// (cert first, then key) at <dir>/server.pem with mode 0o600 and returns the
// absolute path. mongod's --tlsCertificateKeyFile flag expects a single PEM
// containing both blocks, unlike postgres which takes them separately.
func writeCombinedMongoPEM(t *testing.T, dir string, bundle *ssl.CertificateBundle) string {
	t.Helper()

	combined := bundle.ServerCertPEM
	if !strings.HasSuffix(combined, "\n") {
		combined += "\n"
	}

	combined += bundle.ServerKeyPEM

	pemPath := filepath.Join(dir, "server.pem")

	// 0o644 (not 0o600): testcontainers mounts the file as root:root inside the
	// container, but mongod runs as UID 999 (the "mongodb" user). With 0o600,
	// mongod aborts global init with "Cannot read certificate file ...
	// Permission denied" + "InvalidSSLConfiguration: Can not set up PEM key
	// file." The PEM still carries the unencrypted private key, but that's a
	// test-only artifact inside an ephemeral t.TempDir() — host exposure is nil.
	err := os.WriteFile(pemPath, []byte(combined), 0o644)
	require.NoError(t, err, "write combined mongo PEM")

	return pemPath
}

// writeMongoCAPEM writes the CA cert PEM to <dir>/ca.crt with mode 0o600 and
// returns the absolute path. mongo 7 requires a chain of trust to be present
// alongside --tlsCertificateKeyFile (SERVER-72839); without --tlsCAFile mongod
// aborts global init with "InvalidOptions: The use of TLS without specifying a
// chain of trust is no longer supported". Even with tlsInsecure=true on the
// client, the server still needs the CA file to start.
func writeMongoCAPEM(t *testing.T, dir string, bundle *ssl.CertificateBundle) string {
	t.Helper()

	caPath := filepath.Join(dir, "ca.crt")

	err := os.WriteFile(caPath, []byte(bundle.CACertPEM), 0o600)
	require.NoError(t, err, "write mongo CA PEM")

	return caPath
}

// startTLSMongoDB provisions a fresh mongo container with --tlsMode requireTLS
// and a self-signed cert. It is attached to the shared suite network so the
// per-test Manager can reach it by network alias.
//
// Returns (alias, port). The alias is what the Manager (running in another
// container on the same network) uses as DATASOURCE_*_HOST; the port is the
// internal mongod port (27017) on the shared network.
//
// With tlsInsecure=true (the only mongo TLS mode that lets us trust a self
// signed cert), the driver does not validate the hostname, so we don't need
// to inject the alias into ServerDNSNames.
func startTLSMongoDB(t *testing.T, ctx context.Context, name string) (alias string, port int) {
	t.Helper()

	// 1. Generate a CA + server cert/key bundle. Hostname/CA verification is
	//    bypassed via tlsInsecure=true (insecure mode), so the default DNS
	//    names in the bundle are sufficient.
	bundle, err := ssl.GenerateCertificates(ssl.DefaultGenerateOptions())
	require.NoError(t, err, "generate SSL certificate bundle for mongo")

	certDir := t.TempDir()
	combinedPEM := writeCombinedMongoPEM(t, certDir, bundle)
	caPEM := writeMongoCAPEM(t, certDir, bundle)

	// 2. Bring up mongo with TLS enforced. The official mongo Docker image
	//    runs MONGO_INITDB_ROOT_USERNAME/PASSWORD bootstrap BEFORE the user
	//    CMD is applied, so the root user is created against the pre-TLS
	//    mongod and the final mongod is the one running with requireTLS.
	//
	//    --tlsCAFile is mandatory on mongo 7+: without it, mongod aborts global
	//    init with "InvalidOptions: The use of TLS without specifying a chain
	//    of trust is no longer supported" (SERVER-72839). We point it at our
	//    self-signed CA; the fetcher still uses tlsInsecure=true so client-side
	//    chain validation stays disabled.
	//
	//    --tlsAllowConnectionsWithoutCertificates is required because mongo
	//    defaults to mutual TLS when --tlsCertificateKeyFile is set. The
	//    fetcher uses server-side TLS only (no client cert), which mirrors
	//    typical managed-mongo deployments (Atlas, DocumentDB).
	infra := mongodb.NewMongoDBInfra(mongodb.MongoDBConfig{
		Name:     name,
		Username: sslMongoUser,
		Password: sslMongoPassword,
		Options: []mongodb.MongoDBOption{
			// 0o644 on the in-container modes: mongod runs as UID 999 and must
			// be able to read both PEMs (testcontainers mounts as root:root).
			// 0o600 here = "Permission denied" + InvalidSSLConfiguration.
			mongodb.WithMongoDBFile(combinedPEM, sslMongoCombinedPEMPath, 0o644),
			mongodb.WithMongoDBFile(caPEM, sslMongoCAPath, 0o644),
			mongodb.WithMongoDBCommand(
				"--tlsMode", "requireTLS",
				"--tlsCertificateKeyFile", sslMongoCombinedPEMPath,
				"--tlsCAFile", sslMongoCAPath,
				"--tlsAllowConnectionsWithoutCertificates",
			),
		},
	})

	require.NoError(t, infra.Start(ctx, suite.Env()), "start TLS-required mongodb")

	t.Cleanup(func() {
		_ = infra.Terminate(context.Background())
	})

	host, p, err := infra.HostPort()
	require.NoError(t, err, "TLS mongo host/port")

	return host, p
}

// mongoInternalDatasourceEnv builds the DATASOURCE_PLUGIN_CRM_* env-var set
// the resolver scans at Manager boot. sslMode == "" omits SSLMODE entirely
// (reproducing the LaFinteca misconfiguration applied to mongo); a non-empty
// value gates the SSL parser in pkg/resolver/env_loader.go.
//
// The internal mongo datasource defaults authSource to the database name (see
// buildMongoDBOptions in pkg/datasource/datasource_factory.go), so we create
// the root user against the default admin db AND tell the factory to use
// authSource=admin via OPTIONS — that's what matches the mongo bootstrap user.
func mongoInternalDatasourceEnv(host string, port int, sslMode string) map[string]string {
	env := map[string]string{
		sslMongoEnvPrefix + "_CONFIG_NAME": sslMongoConfigName,
		sslMongoEnvPrefix + "_TYPE":        "mongodb",
		sslMongoEnvPrefix + "_HOST":        host,
		sslMongoEnvPrefix + "_PORT":        fmt.Sprintf("%d", port),
		sslMongoEnvPrefix + "_DATABASE":    sslMongoDatabase,
		sslMongoEnvPrefix + "_USER":        sslMongoUser,
		sslMongoEnvPrefix + "_PASSWORD":    sslMongoPassword,
		sslMongoEnvPrefix + "_OPTIONS":     "authSource=admin",
	}

	if sslMode != "" {
		env[sslMongoEnvPrefix+"_SSLMODE"] = sslMode
	}

	return env
}

// deterministicPluginCRMID returns the UUID the Manager assigns to the
// plugin_crm internal datasource in single-tenant mode. tenantID is empty
// because MULTI_TENANT_ENABLED=false in the e2e Manager env (apps.go
// ManagerEnv()), so FindConfigByID receives the empty tenant context.
func deterministicPluginCRMID() string {
	registry := resolver.NewInternalDatasourceRegistry()

	return registry.GetDeterministicID("", sslMongoConfigName).String()
}

// TestInternalDatasourceSSL_Mongo_SSLModeInsecure_AgainstRequireTLS_Success
// validates the fetcher-012 fix for the MongoDB path: with
// DATASOURCE_PLUGIN_CRM_SSLMODE=insecure the resolver populates conn.SSL,
// the datasource factory appends tls=true&tlsInsecure=true to the URI
// (appendMongoDBSSLParams in pkg/datasource/datasource_factory.go), and a
// TLS handshake against a requireTLS mongo with a self-signed cert succeeds.
func TestInternalDatasourceSSL_Mongo_SSLModeInsecure_AgainstRequireTLS_Success(t *testing.T) {
	t.Parallel()

	_ = e2eshared.GenerateProductName() // isolation sanity

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	host, port := startTLSMongoDB(t, ctx, "internal-ssl-mongo-success")

	appEnv := internalSSLMongoAppEnv(t)

	t.Log("Spawning dedicated Manager with DATASOURCE_PLUGIN_CRM_SSLMODE=insecure (mongo TLS branch of the fetcher-012 fix)")

	mgr, err := e2eshared.StartManager(t, ctx, appEnv, e2eshared.AppStartConfig{
		Image:     managerImage(),
		SkipBuild: skipBuild(),
		ExtraEnv:  mongoInternalDatasourceEnv(host, port, "insecure"),
	})
	require.NoError(t, err, "start manager for mongo sslmode=insecure scenario")

	t.Cleanup(func() {
		_ = mgr.Container.Terminate(context.Background())
	})

	client := e2eshared.NewClientFromApp(mgr)

	// mongo TLS handshake on a freshly-booted container can be sluggish on
	// the first connection; mirror the postgres retry window (15s) so the
	// first-handshake jitter doesn't flake the test.
	deadline := time.Now().Add(15 * time.Second)

	var (
		result    *e2eshared.TestConnectionResult
		lastError error
	)

	for time.Now().Before(deadline) {
		result, lastError = client.TestConnection(ctx, deterministicPluginCRMID())
		if lastError == nil && result != nil && result.Status == "success" {
			break
		}

		select {
		case <-ctx.Done():
			t.Fatalf("ctx cancelled while waiting for mongo SSL connection test: %v", ctx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}

	require.NoError(t, lastError, "test connection with sslmode=insecure must eventually succeed")
	require.NotNil(t, result, "test connection result must not be nil")
	assert.Equal(t, "success", result.Status, "status must be success once mongo TLS handshake completes")
	assert.Greater(t, result.LatencyMs, int64(0), "latency must be positive on a real round-trip")

	t.Logf("fetcher-012 mongo fix validated: sslmode=insecure negotiated TLS with requireTLS mongo (latency=%dms)", result.LatencyMs)

	// /schema is the endpoint that actually failed in the LaFinteca incident.
	// With the connection now healthy over TLS, the schema call must succeed.
	// The target mongo database (sslMongoDatabase="crm") is empty — there are
	// no collections beyond mongo's system databases (admin/local/config), all
	// of which get_connection_schema.go filters out. We assert structurally.
	schema, err := client.GetConnectionSchema(ctx, deterministicPluginCRMID())
	require.NoError(t, err, "GET /schema over TLS with sslmode=insecure must succeed")
	require.NotNil(t, schema, "schema response must not be nil")
	assert.NotNil(t, schema.Tables, "schema.Tables slice must be present (may be empty for an unseeded mongo database)")

	t.Logf("fetcher-012 mongo fix validated on /schema: returned %d collections over TLS", len(schema.Tables))
}

// TestInternalDatasourceSSL_Mongo_NoSSLMode_AgainstRequireTLS_Failure
// reproduces the LaFinteca-style misconfiguration on the MongoDB path: an
// internal datasource without SSLMODE pointed at a TLS-required mongo. The
// resolver leaves conn.SSL == nil, so appendMongoDBSSLParams emits a URI
// without tls=true, the driver attempts a plaintext handshake, mongod rejects
// it, and POST /test surfaces HTTP 500 with "Database Connection Error".
func TestInternalDatasourceSSL_Mongo_NoSSLMode_AgainstRequireTLS_Failure(t *testing.T) {
	t.Parallel()

	_ = e2eshared.GenerateProductName() // isolation sanity (unused on negative path)

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	host, port := startTLSMongoDB(t, ctx, "internal-ssl-mongo-failure")

	appEnv := internalSSLMongoAppEnv(t)

	t.Log("Spawning dedicated Manager — DATASOURCE_PLUGIN_CRM_SSLMODE intentionally omitted to reproduce the bug on mongo")

	mgr, err := e2eshared.StartManager(t, ctx, appEnv, e2eshared.AppStartConfig{
		Image:     managerImage(),
		SkipBuild: skipBuild(),
		ExtraEnv:  mongoInternalDatasourceEnv(host, port, ""),
	})
	require.NoError(t, err, "start manager for mongo sslmode-missing scenario")

	t.Cleanup(func() {
		_ = mgr.Container.Terminate(context.Background())
	})

	client := e2eshared.NewClientFromApp(mgr)

	resp, err := client.TestConnectionRaw(ctx, deterministicPluginCRMID())
	require.NoError(t, err, "POST /test request must reach Manager")

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode(),
		"missing SSLMODE against requireTLS mongo must surface as HTTP 500 (Database Connection Error)")

	body := string(resp.Body())
	assert.True(t, strings.Contains(body, "Database Connection Error"),
		"response body must carry the Database Connection Error title, got: %s", body)

	t.Logf("Reproduced LaFinteca-style mongo bug: status=%d body=%s", resp.StatusCode(), body)

	// /schema is the endpoint that actually triggered the LaFinteca incident.
	// Both /test and /schema share the same factory error mapping in
	// get_connection_schema.go / test_connection.go (HTTP 500 +
	// "Database Connection Error"), so the schema path must reproduce the
	// exact same failure surface when SSLMODE is missing against requireTLS mongo.
	schemaResp, err := client.GetConnectionSchemaRaw(ctx, deterministicPluginCRMID())
	require.NoError(t, err, "GET /schema request must reach Manager")

	require.Equal(t, http.StatusInternalServerError, schemaResp.StatusCode(),
		"missing SSLMODE against requireTLS mongo must surface as HTTP 500 on /schema as well")

	schemaBody := string(schemaResp.Body())
	assert.True(t, strings.Contains(schemaBody, "Database Connection Error"),
		"schema response body must carry the Database Connection Error title, got: %s", schemaBody)

	t.Logf("Reproduced LaFinteca-style mongo bug on /schema: status=%d body=%s", schemaResp.StatusCode(), schemaBody)
}

// TestInternalDatasourceSSL_Mongo_NoSSL_AgainstPlainMongo_BackwardCompat
// protects existing deployments: when an internal mongo datasource is
// configured WITHOUT _SSLMODE and points at a plain (non-TLS-required)
// mongo, the connection test must still succeed.
//
// To avoid spinning yet another mongo container, this test reuses the
// shared coreInfra.MongoDB (which already runs in every e2e suite as the
// fetcher's internal store) — its credentials match
// e2eshared.CoreInfraUsername / CoreInfraPassword and authSource=admin is
// the mongo image default for the root user.
func TestInternalDatasourceSSL_Mongo_NoSSL_AgainstPlainMongo_BackwardCompat(t *testing.T) {
	t.Parallel()

	_ = e2eshared.GenerateProductName()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Reuse the shared plain mongo (coreInfra.MongoDB) — it accepts
	// plaintext and is the equivalent of a self-hosted mongo on a trusted
	// VPC. HostPort() returns the network alias when the suite runs on a
	// shared network, which is exactly what the per-test Manager needs.
	host, port, err := coreInfra.MongoDB.HostPort()
	require.NoError(t, err, "shared plain mongo host/port")

	appEnv := internalSSLMongoAppEnv(t)

	// The shared coreInfra.MongoDB uses the CoreInfra credentials, not the
	// per-test SSL credentials. Override DATABASE/USER/PASSWORD so the
	// internal datasource targets a DB the bootstrap user can read. The
	// fetcher-db database is created by the application on first boot;
	// authSource=admin lets the root user authenticate against it.
	envOverrides := mongoInternalDatasourceEnv(host, port, "")
	envOverrides[sslMongoEnvPrefix+"_DATABASE"] = "fetcher-db"
	envOverrides[sslMongoEnvPrefix+"_USER"] = e2eshared.CoreInfraUsername
	envOverrides[sslMongoEnvPrefix+"_PASSWORD"] = e2eshared.CoreInfraPassword

	t.Log("Spawning dedicated Manager without SSLMODE against plain mongo (mongo backward-compat path)")

	mgr, err := e2eshared.StartManager(t, ctx, appEnv, e2eshared.AppStartConfig{
		Image:     managerImage(),
		SkipBuild: skipBuild(),
		ExtraEnv:  envOverrides,
	})
	require.NoError(t, err, "start manager for mongo backward-compat scenario")

	t.Cleanup(func() {
		_ = mgr.Container.Terminate(context.Background())
	})

	client := e2eshared.NewClientFromApp(mgr)

	result, err := client.TestConnection(ctx, deterministicPluginCRMID())
	require.NoError(t, err, "plaintext mongo path must still succeed")
	require.NotNil(t, result, "test connection result must not be nil")
	assert.Equal(t, "success", result.Status, "no-SSL + plain mongo must remain a valid configuration")
	assert.Greater(t, result.LatencyMs, int64(0), "latency must be positive")

	t.Logf("backward-compat protected: no-SSL path against plain mongo still works (latency=%dms)", result.LatencyMs)

	// /schema must also stay healthy on the mongo backward-compat path —
	// this is the endpoint LaFinteca actually hit in production. The shared
	// coreInfra mongo ("fetcher-db") contains fetcher's own internal
	// collections (connections, jobs). After get_connection_schema.go filters
	// admin/local/config and system.* collections, schema.Tables MAY contain
	// fetcher's internal collections — but we only assert structurally so
	// this remains robust regardless of seeded state.
	schema, err := client.GetConnectionSchema(ctx, deterministicPluginCRMID())
	require.NoError(t, err, "GET /schema on the no-SSL backward-compat path must still succeed")
	require.NotNil(t, schema, "schema response must not be nil")
	assert.NotNil(t, schema.Tables, "schema.Tables slice must be present (may include fetcher's own collections)")

	t.Logf("backward-compat protected on /schema: returned %d collections over plaintext", len(schema.Tables))
}
