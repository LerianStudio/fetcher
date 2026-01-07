package containers

import (
	"os"
	"path/filepath"

	"github.com/LerianStudio/fetcher/tests/shared/fixtures/ssl"
)

// PostgresSSLConfigContent contains the PostgreSQL configuration for SSL.
// This content is written to a temporary file and mounted into the container.
const PostgresSSLConfigContent = `# PostgreSQL SSL Configuration for testcontainers
# This file configures PostgreSQL to accept SSL connections.

listen_addresses = '*'

# SSL Settings
ssl = on
ssl_ca_file = '/tmp/testcontainers-go/postgres/ca.crt'
ssl_cert_file = '/tmp/testcontainers-go/postgres/server.crt'
ssl_key_file = '/tmp/testcontainers-go/postgres/server.key'
`

// MySQLSSLConfigContent contains the MySQL configuration for SSL.
// This content is written to a temporary file and mounted into the container.
// Uses /etc/mysql/conf.d/ path which exists by default in the MySQL Docker image.
const MySQLSSLConfigContent = `[mysqld]
ssl_ca=/etc/mysql/conf.d/ca.pem
ssl_cert=/etc/mysql/conf.d/server-cert.pem
ssl_key=/etc/mysql/conf.d/server-key.pem
`

// WritePostgresSSLConfig writes the PostgreSQL SSL config file to the specified directory.
// Returns the path to the written config file.
func WritePostgresSSLConfig(dir string) (string, error) {
	configPath := filepath.Join(dir, "postgres-ssl.conf")
	if err := os.WriteFile(configPath, []byte(PostgresSSLConfigContent), 0600); err != nil {
		return "", err
	}

	return configPath, nil
}

// WriteMySQLSSLConfig writes the MySQL SSL config file to the specified directory.
// Returns the path to the written config file.
func WriteMySQLSSLConfig(dir string) (string, error) {
	configPath := filepath.Join(dir, "mysql-ssl.cnf")
	if err := os.WriteFile(configPath, []byte(MySQLSSLConfigContent), 0600); err != nil {
		return "", err
	}

	return configPath, nil
}

// SSLConfig contains SSL/TLS configuration for database containers.
type SSLConfig struct {
	// Enabled indicates whether SSL is enabled for this container
	Enabled bool

	// CertBundle contains the generated certificates
	CertBundle *ssl.CertificateBundle

	// Mode specifies the SSL mode (database-specific interpretation)
	// PostgreSQL: "disable", "require", "verify-ca", "verify-full"
	// MySQL: "DISABLED", "PREFERRED", "REQUIRED", "VERIFY_CA", "VERIFY_IDENTITY"
	// SQL Server: "disable", "true", "strict"
	// Oracle: "TCPS" or empty
	// MongoDB: "true", "false"
	Mode string

	// RequireClientCert indicates whether client certificate is required
	RequireClientCert bool

	// PostgresConfigPath is the path to the PostgreSQL SSL config file
	// (only needed for PostgreSQL SSL containers)
	PostgresConfigPath string

	// MySQLConfigPath is the path to the MySQL SSL config file
	// (only needed for MySQL SSL containers)
	MySQLConfigPath string
}

// SSLConnectionInfo contains SSL connection details for client connections.
type SSLConnectionInfo struct {
	// Enabled indicates SSL is available for this connection
	Enabled bool

	// Mode is the SSL mode to use in connection strings
	Mode string

	// CACert is the CA certificate content (PEM format)
	CACert string

	// ClientCert is the client certificate content (PEM format)
	ClientCert string

	// ClientKey is the client private key content (PEM format)
	ClientKey string

	// CACertPath is the path to the CA certificate file (for drivers that need file paths)
	CACertPath string

	// ClientCertPath is the path to the client certificate file
	ClientCertPath string

	// ClientKeyPath is the path to the client key file
	ClientKeyPath string
}

// DefaultSSLConfig returns a disabled SSL configuration.
func DefaultSSLConfig() *SSLConfig {
	return &SSLConfig{
		Enabled: false,
	}
}

// NewSSLConfig creates an enabled SSL configuration with the specified mode.
func NewSSLConfig(mode string, certBundle *ssl.CertificateBundle) *SSLConfig {
	return &SSLConfig{
		Enabled:    true,
		CertBundle: certBundle,
		Mode:       mode,
	}
}

// ToConnectionInfo converts SSLConfig to SSLConnectionInfo for client use.
func (c *SSLConfig) ToConnectionInfo() *SSLConnectionInfo {
	if c == nil || !c.Enabled || c.CertBundle == nil {
		return &SSLConnectionInfo{Enabled: false}
	}

	return &SSLConnectionInfo{
		Enabled:        true,
		Mode:           c.Mode,
		CACert:         c.CertBundle.CACertPEM,
		ClientCert:     c.CertBundle.ClientCertPEM,
		ClientKey:      c.CertBundle.ClientKeyPEM,
		CACertPath:     c.CertBundle.CACertPath,
		ClientCertPath: c.CertBundle.ClientCertPath,
		ClientKeyPath:  c.CertBundle.ClientKeyPath,
	}
}
