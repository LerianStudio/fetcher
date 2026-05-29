package resolver

import (
	"strconv"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setEnvDatasource is a tiny helper that registers the standard env-var set
// for one internal datasource via t.Setenv (auto-cleanup at end of test).
// extra is a map of <suffix>=<value> applied on top of the base set.
func setEnvDatasource(t *testing.T, name, dbType, host, database string, port int, extra map[string]string) {
	t.Helper()
	t.Setenv("DATASOURCE_"+name+"_CONFIG_NAME", name)
	t.Setenv("DATASOURCE_"+name+"_TYPE", dbType)
	t.Setenv("DATASOURCE_"+name+"_HOST", host)
	t.Setenv("DATASOURCE_"+name+"_DATABASE", database)
	t.Setenv("DATASOURCE_"+name+"_PORT", strconv.Itoa(port))
	for k, v := range extra {
		t.Setenv("DATASOURCE_"+name+"_"+k, v)
	}
}

// registryWith adds an extra internal datasource entry for testing types that
// are not in the production MVP registry (mysql/oracle/sqlserver). It mutates
// a fresh registry — never used in production code.
func registryWith(name string, cfg InternalDSConfig) *InternalDatasourceRegistry {
	r := NewInternalDatasourceRegistry()
	r.datasources[name] = cfg
	return r
}

func TestLoadInternalConnectionsFromEnv_PostgreSQL_SSLModeRequire(t *testing.T) {
	setEnvDatasource(t, "midaz_onboarding", "postgresql", "rds.example.com", "onboarding", 5432, map[string]string{
		"SSLMODE": "require",
	})

	registry := NewInternalDatasourceRegistry()
	conns := LoadInternalConnectionsFromEnv(registry, testutil.TestLogger())

	require.Contains(t, conns, "midaz_onboarding")
	conn := conns["midaz_onboarding"]
	require.NotNil(t, conn.SSL, "SSL config must be set when SSLMODE is provided")
	assert.Equal(t, "require", conn.SSL.Mode)
	assert.Empty(t, conn.SSL.CA)
	assert.Empty(t, conn.SSL.Cert)
	assert.Empty(t, conn.SSL.Key)
	assert.Equal(t, model.TypePostgreSQL, conn.Type)
}

func TestLoadInternalConnectionsFromEnv_PostgreSQL_SSLModeInvalid_Skips(t *testing.T) {
	setEnvDatasource(t, "midaz_onboarding", "postgresql", "rds.example.com", "onboarding", 5432, map[string]string{
		"SSLMODE": "garbage",
	})

	registry := NewInternalDatasourceRegistry()
	conns := LoadInternalConnectionsFromEnv(registry, testutil.TestLogger())

	assert.NotContains(t, conns, "midaz_onboarding",
		"connection with invalid SSLMODE must be skipped, not silently downgraded")
}

func TestLoadInternalConnectionsFromEnv_PostgreSQL_SSLModeVerifyCA(t *testing.T) {
	setEnvDatasource(t, "midaz_transaction", "postgresql", "rds.example.com", "transaction", 5432, map[string]string{
		"SSLMODE": "verify-ca",
	})

	conns := LoadInternalConnectionsFromEnv(NewInternalDatasourceRegistry(), testutil.TestLogger())

	require.Contains(t, conns, "midaz_transaction")
	require.NotNil(t, conns["midaz_transaction"].SSL)
	assert.Equal(t, "verify-ca", conns["midaz_transaction"].SSL.Mode)
}

func TestLoadInternalConnectionsFromEnv_PostgreSQL_SSLModeVerifyFull(t *testing.T) {
	setEnvDatasource(t, "midaz_transaction", "postgresql", "rds.example.com", "transaction", 5432, map[string]string{
		"SSLMODE": "verify-full",
	})

	conns := LoadInternalConnectionsFromEnv(NewInternalDatasourceRegistry(), testutil.TestLogger())

	require.Contains(t, conns, "midaz_transaction")
	require.NotNil(t, conns["midaz_transaction"].SSL)
	assert.Equal(t, "verify-full", conns["midaz_transaction"].SSL.Mode)
}

func TestLoadInternalConnectionsFromEnv_PostgreSQL_FullSSLBundle(t *testing.T) {
	const fakePEM = "-----BEGIN CERTIFICATE-----\nMIIDfake\n-----END CERTIFICATE-----"
	setEnvDatasource(t, "midaz_onboarding", "postgresql", "rds.example.com", "onboarding", 5432, map[string]string{
		"SSLMODE":  "verify-full",
		"SSL_CA":   fakePEM,
		"SSL_CERT": "client-cert",
		"SSL_KEY":  "client-key",
	})

	conns := LoadInternalConnectionsFromEnv(NewInternalDatasourceRegistry(), testutil.TestLogger())

	require.Contains(t, conns, "midaz_onboarding")
	ssl := conns["midaz_onboarding"].SSL
	require.NotNil(t, ssl)
	assert.Equal(t, "verify-full", ssl.Mode)
	assert.Equal(t, fakePEM, ssl.CA)
	assert.Equal(t, "client-cert", ssl.Cert)
	assert.Equal(t, "client-key", ssl.Key)
}

func TestLoadInternalConnectionsFromEnv_SSLCAWithoutSSLMode_NoSSL(t *testing.T) {
	setEnvDatasource(t, "midaz_onboarding", "postgresql", "rds.example.com", "onboarding", 5432, map[string]string{
		// SSLMODE intentionally NOT set.
		"SSL_CA": "-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----",
	})

	conns := LoadInternalConnectionsFromEnv(NewInternalDatasourceRegistry(), testutil.TestLogger())

	require.Contains(t, conns, "midaz_onboarding")
	assert.Nil(t, conns["midaz_onboarding"].SSL,
		"SSLMODE is the gate; CA alone must not enable SSL because the driver default is disable")
}

// -----------------------------------------------------------------------------
// Marco 2 — Type coverage (mysql, oracle, sqlserver, mongodb) + edge cases.
// All allowlist values below were chosen from pkg/datasource/sslmode/{driver}.go:
//   - MySQL    "true"   (validMySQLModes:    "", false, true, skip-verify, preferred)
//   - Oracle   "verify" (validOracleModes:   "", disable, false, true, enable, verify, skip-verify)
//   - SQLServer "strict"(validSQLServerModes:"", disable, false, true, strict)
//   - MongoDB  "true"   (validMongoDBModes:  "", disable, false, true, enable, insecure)
// -----------------------------------------------------------------------------

func TestLoadInternalConnectionsFromEnv_MySQL_SSLModeValid(t *testing.T) {
	setEnvDatasource(t, "test_mysql", "mysql", "mysql.example.com", "appdb", 3306, map[string]string{
		"SSLMODE": "true",
	})

	r := registryWith("test_mysql", InternalDSConfig{Service: "test", Module: "test", DBType: model.TypeMySQL})
	conns := LoadInternalConnectionsFromEnv(r, testutil.TestLogger())

	require.Contains(t, conns, "test_mysql")
	require.NotNil(t, conns["test_mysql"].SSL)
	assert.Equal(t, "true", conns["test_mysql"].SSL.Mode)
}

func TestLoadInternalConnectionsFromEnv_MySQL_SSLModeInvalid_Skips(t *testing.T) {
	setEnvDatasource(t, "test_mysql", "mysql", "mysql.example.com", "appdb", 3306, map[string]string{
		"SSLMODE": "totally-bogus",
	})

	r := registryWith("test_mysql", InternalDSConfig{Service: "test", Module: "test", DBType: model.TypeMySQL})
	conns := LoadInternalConnectionsFromEnv(r, testutil.TestLogger())

	assert.NotContains(t, conns, "test_mysql")
}

func TestLoadInternalConnectionsFromEnv_Oracle_SSLModeValid(t *testing.T) {
	setEnvDatasource(t, "test_oracle", "oracle", "oracle.example.com", "ORCL", 1521, map[string]string{
		"SSLMODE": "verify",
	})

	r := registryWith("test_oracle", InternalDSConfig{Service: "test", Module: "test", DBType: model.TypeOracle})
	conns := LoadInternalConnectionsFromEnv(r, testutil.TestLogger())

	require.Contains(t, conns, "test_oracle")
	require.NotNil(t, conns["test_oracle"].SSL)
	assert.Equal(t, "verify", conns["test_oracle"].SSL.Mode)
}

func TestLoadInternalConnectionsFromEnv_Oracle_SSLModeInvalid_Skips(t *testing.T) {
	setEnvDatasource(t, "test_oracle", "oracle", "oracle.example.com", "ORCL", 1521, map[string]string{
		"SSLMODE": "garbage",
	})

	r := registryWith("test_oracle", InternalDSConfig{Service: "test", Module: "test", DBType: model.TypeOracle})
	conns := LoadInternalConnectionsFromEnv(r, testutil.TestLogger())

	assert.NotContains(t, conns, "test_oracle")
}

func TestLoadInternalConnectionsFromEnv_SQLServer_SSLModeValid(t *testing.T) {
	setEnvDatasource(t, "test_mssql", "sql_server", "mssql.example.com", "appdb", 1433, map[string]string{
		"SSLMODE": "strict",
	})

	r := registryWith("test_mssql", InternalDSConfig{Service: "test", Module: "test", DBType: model.TypeSQLServer})
	conns := LoadInternalConnectionsFromEnv(r, testutil.TestLogger())

	require.Contains(t, conns, "test_mssql")
	require.NotNil(t, conns["test_mssql"].SSL)
	assert.Equal(t, "strict", conns["test_mssql"].SSL.Mode)
}

func TestLoadInternalConnectionsFromEnv_SQLServer_SSLModeInvalid_Skips(t *testing.T) {
	setEnvDatasource(t, "test_mssql", "sql_server", "mssql.example.com", "appdb", 1433, map[string]string{
		"SSLMODE": "definitely-not-valid",
	})

	r := registryWith("test_mssql", InternalDSConfig{Service: "test", Module: "test", DBType: model.TypeSQLServer})
	conns := LoadInternalConnectionsFromEnv(r, testutil.TestLogger())

	assert.NotContains(t, conns, "test_mssql")
}

func TestLoadInternalConnectionsFromEnv_MongoDB_SSLModeTrue(t *testing.T) {
	setEnvDatasource(t, "plugin_crm", "mongodb", "docdb.example.com", "crm", 27017, map[string]string{
		"SSLMODE": "true",
	})

	conns := LoadInternalConnectionsFromEnv(NewInternalDatasourceRegistry(), testutil.TestLogger())

	require.Contains(t, conns, "plugin_crm")
	require.NotNil(t, conns["plugin_crm"].SSL)
	assert.Equal(t, "true", conns["plugin_crm"].SSL.Mode)
}

func TestLoadInternalConnectionsFromEnv_MongoDB_SSLModeInvalid_Skips(t *testing.T) {
	setEnvDatasource(t, "plugin_crm", "mongodb", "docdb.example.com", "crm", 27017, map[string]string{
		"SSLMODE": "not-a-valid-mongo-mode",
	})

	conns := LoadInternalConnectionsFromEnv(NewInternalDatasourceRegistry(), testutil.TestLogger())

	assert.NotContains(t, conns, "plugin_crm")
}

func TestLoadInternalConnectionsFromEnv_MixedSSLAndPlain(t *testing.T) {
	// midaz_onboarding gets SSL; midaz_transaction does not.
	setEnvDatasource(t, "midaz_onboarding", "postgresql", "rds-a.example.com", "onboarding", 5432, map[string]string{
		"SSLMODE": "require",
	})
	setEnvDatasource(t, "midaz_transaction", "postgresql", "rds-b.example.com", "transaction", 5432, nil)

	conns := LoadInternalConnectionsFromEnv(NewInternalDatasourceRegistry(), testutil.TestLogger())

	require.Contains(t, conns, "midaz_onboarding")
	require.Contains(t, conns, "midaz_transaction")

	require.NotNil(t, conns["midaz_onboarding"].SSL)
	assert.Equal(t, "require", conns["midaz_onboarding"].SSL.Mode)

	assert.Nil(t, conns["midaz_transaction"].SSL, "datasource without SSLMODE must keep SSL nil for backward compat")
}

func TestLoadInternalConnectionsFromEnv_NoSSLEnvVars_BackwardCompat(t *testing.T) {
	// No SSLMODE, no SSL_CA — pre-fetcher-012 behavior must be preserved.
	setEnvDatasource(t, "midaz_onboarding", "postgresql", "rds.example.com", "onboarding", 5432, nil)

	conns := LoadInternalConnectionsFromEnv(NewInternalDatasourceRegistry(), testutil.TestLogger())

	require.Contains(t, conns, "midaz_onboarding")
	assert.Nil(t, conns["midaz_onboarding"].SSL, "no SSLMODE -> SSL must remain nil (backward compat)")
}

func TestLoadInternalConnectionsFromEnv_InvalidTypeSkipsBeforeSSLParse(t *testing.T) {
	// TYPE is invalid; SSLMODE is set — the function must skip on TYPE first
	// and never read/validate SSLMODE. We assert by simply confirming the
	// connection is not loaded.
	t.Setenv("DATASOURCE_midaz_onboarding_CONFIG_NAME", "midaz_onboarding")
	t.Setenv("DATASOURCE_midaz_onboarding_TYPE", "definitely_not_a_db")
	t.Setenv("DATASOURCE_midaz_onboarding_HOST", "rds.example.com")
	t.Setenv("DATASOURCE_midaz_onboarding_DATABASE", "onboarding")
	t.Setenv("DATASOURCE_midaz_onboarding_PORT", "5432")
	t.Setenv("DATASOURCE_midaz_onboarding_SSLMODE", "require")

	conns := LoadInternalConnectionsFromEnv(NewInternalDatasourceRegistry(), testutil.TestLogger())

	assert.NotContains(t, conns, "midaz_onboarding")
}
