package containers

import (
	"context"
	"fmt"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// MySQLContainer wraps a MySQL testcontainer with connection info.
type MySQLContainer struct {
	Container    *mysql.MySQLContainer
	URL          string
	Host         string
	Port         string
	InternalHost string
	Internal     config.InternalDBConnection
	SSL          *SSLConnectionInfo
}

// MySQLOptions configures MySQL container startup.
type MySQLOptions struct {
	NetworkName   string
	NetworkAlias  string
	FixedHostPort string
	Username      string
	Password      string
	RootPassword  string
	Database      string
	InitScript    string
	SSL           *SSLConfig
}

// DefaultMySQLOptions returns default MySQL options.
func DefaultMySQLOptions(networkName string) MySQLOptions {
	return MySQLOptions{
		NetworkName:  networkName,
		NetworkAlias: "mysql-external",
		Username:     "testuser",
		Password:     "testpass",
		RootPassword: "rootpass",
		Database:     "testdb",
	}
}

// DefaultMySQLSSLOptions returns MySQL options with SSL enabled.
func DefaultMySQLSSLOptions(networkName string) MySQLOptions {
	opts := DefaultMySQLOptions(networkName)
	opts.NetworkAlias = "mysql-external-ssl"
	opts.SSL = &SSLConfig{
		Enabled: true,
		Mode:    "REQUIRED",
	}

	return opts
}

// StartMySQL starts a MySQL container with the given options.
func StartMySQL(ctx context.Context, opts MySQLOptions) (*MySQLContainer, error) {
	containerOpts := []testcontainers.ContainerCustomizer{
		mysql.WithUsername(opts.Username),
		mysql.WithPassword(opts.Password),
		mysql.WithDatabase(opts.Database),
		testcontainers.WithEnv(map[string]string{
			"MYSQL_ROOT_PASSWORD": opts.RootPassword,
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(config.MySQLStartupTimeout),
		),
	}

	if opts.InitScript != "" {
		containerOpts = append(containerOpts,
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Files: []testcontainers.ContainerFile{
						{
							ContainerFilePath: "/docker-entrypoint-initdb.d/init.sql",
							Reader:            strings.NewReader(opts.InitScript),
							FileMode:          0644,
						},
					},
				},
			}),
		)
	}

	// Add SSL configuration if enabled
	// MySQL SSL setup: MySQL 8 auto-generates SSL certs by default.
	// For custom certs, we need to:
	// 1. Pass certificates via environment variables (avoids file permission issues)
	// 2. Use a wrapper script that creates files with correct ownership before MySQL starts
	// The wrapper script runs as root, creates certs, fixes permissions, then execs mysqld.
	if opts.SSL != nil && opts.SSL.Enabled && opts.SSL.CertBundle != nil {
		// MySQL SSL wrapper script that sets up certificates with correct permissions
		// before calling the original entrypoint. MySQL requires the key file to be
		// owned by mysql user (uid 999) with mode 0600.
		sslWrapperScript := `#!/bin/bash
set -e
# Create SSL directory
mkdir -p /var/lib/mysql-ssl
# Write certificates from environment variables
echo "$SSL_CA_CERT" > /var/lib/mysql-ssl/ca.crt
echo "$SSL_SERVER_CERT" > /var/lib/mysql-ssl/server.crt
echo "$SSL_SERVER_KEY" > /var/lib/mysql-ssl/server.key
# Set correct permissions and ownership (mysql user is uid 999)
chmod 644 /var/lib/mysql-ssl/ca.crt /var/lib/mysql-ssl/server.crt
chmod 600 /var/lib/mysql-ssl/server.key
chown -R mysql:mysql /var/lib/mysql-ssl
# Call the original entrypoint
exec /usr/local/bin/docker-entrypoint.sh "$@"
`
		containerOpts = append(containerOpts,
			testcontainers.WithEnv(map[string]string{
				"SSL_CA_CERT":     opts.SSL.CertBundle.CACertPEM,
				"SSL_SERVER_CERT": opts.SSL.CertBundle.ServerCertPEM,
				"SSL_SERVER_KEY":  opts.SSL.CertBundle.ServerKeyPEM,
			}),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Files: []testcontainers.ContainerFile{
						{
							ContainerFilePath: "/ssl-wrapper.sh",
							Reader:            strings.NewReader(sslWrapperScript),
							FileMode:          0755,
						},
					},
					Entrypoint: []string{"/ssl-wrapper.sh"},
					// Pass SSL arguments to mysqld
					Cmd: []string{
						"mysqld",
						"--ssl-ca=/var/lib/mysql-ssl/ca.crt",
						"--ssl-cert=/var/lib/mysql-ssl/server.crt",
						"--ssl-key=/var/lib/mysql-ssl/server.key",
					},
				},
			}),
		)
	}

	if opts.NetworkName != "" {
		containerOpts = append(containerOpts,
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{opts.NetworkName},
					NetworkAliases: map[string][]string{opts.NetworkName: {opts.NetworkAlias}},
				},
			}),
		)
	}

	if opts.FixedHostPort != "" {
		containerOpts = append(containerOpts, WithFixedPort("3306/tcp", opts.FixedHostPort))
	}

	container, err := mysql.Run(ctx, "mysql:8", containerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start MySQL: %w", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MySQL connection string: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MySQL host: %w", err)
	}

	port, err := container.MappedPort(ctx, "3306")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MySQL port: %w", err)
	}

	// Build internal connection with SSL info
	internal := config.InternalDBConnection{
		Host:     opts.NetworkAlias,
		Port:     3306,
		Username: opts.Username,
		Password: opts.Password,
		Database: opts.Database,
	}

	// Populate SSL connection info
	var sslConnInfo *SSLConnectionInfo

	if opts.SSL != nil && opts.SSL.Enabled {
		internal.SSLEnabled = true

		internal.SSLMode = opts.SSL.Mode
		if opts.SSL.CertBundle != nil {
			internal.SSLCACert = opts.SSL.CertBundle.CACertPEM
			internal.SSLClientCert = opts.SSL.CertBundle.ClientCertPEM
			internal.SSLClientKey = opts.SSL.CertBundle.ClientKeyPEM
		}

		sslConnInfo = opts.SSL.ToConnectionInfo()
	}

	return &MySQLContainer{
		Container:    container,
		URL:          connStr,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
		Internal:     internal,
		SSL:          sslConnInfo,
	}, nil
}

// Stop terminates the MySQL container.
func (m *MySQLContainer) Stop(ctx context.Context) error {
	if m.Container != nil {
		return m.Container.Terminate(ctx)
	}

	return nil
}
