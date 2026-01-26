package chaos

import (
	"testing"

	"github.com/LerianStudio/fetcher/tests/shared/config"
	"github.com/stretchr/testify/assert"
)

func TestProxyRouter_GetProxyConnection_Postgres(t *testing.T) {
	router := NewProxyRouter("toxiproxy")

	direct := config.InternalDBConnection{
		Host:     "postgres-external",
		Port:     5432,
		Username: "test",
		Password: "pass",
		Database: "testdb",
	}

	proxied := router.GetProxyConnection(ServicePostgres, direct)

	assert.Equal(t, "toxiproxy", proxied.Host, "Host should be Toxiproxy")
	assert.Equal(t, 5433, proxied.Port, "Port should be Toxiproxy proxy port")
	assert.Equal(t, "test", proxied.Username, "Username should be preserved")
	assert.Equal(t, "pass", proxied.Password, "Password should be preserved")
	assert.Equal(t, "testdb", proxied.Database, "Database should be preserved")
}

func TestProxyRouter_GetProxyConnection_MySQL(t *testing.T) {
	router := NewProxyRouter("toxiproxy")

	direct := config.InternalDBConnection{
		Host:     "mysql-external",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "mydb",
	}

	proxied := router.GetProxyConnection(ServiceMySQL, direct)

	assert.Equal(t, "toxiproxy", proxied.Host)
	assert.Equal(t, 3307, proxied.Port)
}

func TestProxyRouter_GetProxyConnection_UnknownService(t *testing.T) {
	router := NewProxyRouter("toxiproxy")

	direct := config.InternalDBConnection{
		Host: "unknown",
		Port: 1234,
	}

	// Unknown service should return the direct connection unchanged
	proxied := router.GetProxyConnection(ServiceName("unknown"), direct)

	assert.Equal(t, direct.Host, proxied.Host, "Unknown service should return direct connection")
	assert.Equal(t, direct.Port, proxied.Port)
}

func TestProxyRouter_GetProxyConnection_PreservesSSLFields(t *testing.T) {
	router := NewProxyRouter("toxiproxy")

	direct := config.InternalDBConnection{
		Host:          "postgres-external",
		Port:          5432,
		Username:      "test",
		Password:      "pass",
		Database:      "testdb",
		SSLEnabled:    true,
		SSLMode:       "verify-full",
		SSLCACert:     "/path/to/ca.crt",
		SSLClientCert: "/path/to/client.crt",
		SSLClientKey:  "/path/to/client.key",
	}

	proxied := router.GetProxyConnection(ServicePostgres, direct)

	// Verify SSL fields are preserved
	assert.Equal(t, true, proxied.SSLEnabled, "SSLEnabled should be preserved")
	assert.Equal(t, "verify-full", proxied.SSLMode, "SSLMode should be preserved")
	assert.Equal(t, "/path/to/ca.crt", proxied.SSLCACert, "SSLCACert should be preserved")
	assert.Equal(t, "/path/to/client.crt", proxied.SSLClientCert, "SSLClientCert should be preserved")
	assert.Equal(t, "/path/to/client.key", proxied.SSLClientKey, "SSLClientKey should be preserved")

	// Verify host/port are changed
	assert.Equal(t, "toxiproxy", proxied.Host)
	assert.Equal(t, 5433, proxied.Port)
}
