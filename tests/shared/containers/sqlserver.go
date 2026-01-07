package containers

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mssql"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// SQLServerContainer wraps a SQL Server testcontainer with connection info.
type SQLServerContainer struct {
	Container    *mssql.MSSQLServerContainer
	URL          string
	Host         string
	Port         string
	InternalHost string
	Internal     config.InternalDBConnection
	SSL          *SSLConnectionInfo
}

// SQLServerOptions configures SQL Server container startup.
type SQLServerOptions struct {
	NetworkName   string
	NetworkAlias  string
	FixedHostPort string
	Password      string
	Database      string
	InitScript    string
	SSL           *SSLConfig
}

// DefaultSQLServerOptions returns default SQL Server options.
func DefaultSQLServerOptions(networkName string) SQLServerOptions {
	return SQLServerOptions{
		NetworkName:  networkName,
		NetworkAlias: "sqlserver-external",
		Password:     "TestPass123!",
		Database:     "testdb",
	}
}

// DefaultSQLServerSSLOptions returns SQL Server options with SSL enabled.
func DefaultSQLServerSSLOptions(networkName string) SQLServerOptions {
	opts := DefaultSQLServerOptions(networkName)
	opts.NetworkAlias = "sqlserver-external-ssl"
	opts.SSL = &SSLConfig{
		Enabled: true,
		Mode:    "true",
	}

	return opts
}

// StartSQLServer starts a SQL Server container with the given options.
func StartSQLServer(ctx context.Context, opts SQLServerOptions) (*SQLServerContainer, error) {
	containerOpts := []testcontainers.ContainerCustomizer{
		mssql.WithAcceptEULA(),
		mssql.WithPassword(opts.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("SQL Server is now ready for client connections").
				WithStartupTimeout(config.SQLServerStartupTimeout),
		),
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
		containerOpts = append(containerOpts, WithFixedPort("1433/tcp", opts.FixedHostPort))
	}

	// Add SSL configuration if enabled
	// SQL Server uses mssql.conf file for TLS configuration.
	// SQL Server 2022 runs as mssql user (uid 10001), so we use /var/opt/mssql/ssl
	// which is writable by the mssql user.
	// Reference: https://learn.microsoft.com/en-us/sql/linux/sql-server-linux-docker-container-security
	if opts.SSL != nil && opts.SSL.Enabled && opts.SSL.CertBundle != nil {
		sslWrapperScript := `#!/bin/bash
set -e

# Create SSL directory in mssql's data directory (writable by mssql user)
mkdir -p /var/opt/mssql/ssl

# Write certificates from environment variables
echo "$SSL_SERVER_CERT" > /var/opt/mssql/ssl/mssql.pem
echo "$SSL_SERVER_KEY" > /var/opt/mssql/ssl/mssql.key

# Set correct permissions for certificates
chmod 644 /var/opt/mssql/ssl/mssql.pem
chmod 600 /var/opt/mssql/ssl/mssql.key

# Create mssql.conf with TLS settings
cat > /var/opt/mssql/mssql.conf << 'CONF'
[network]
tlscert = /var/opt/mssql/ssl/mssql.pem
tlskey = /var/opt/mssql/ssl/mssql.key
tlsprotocols = 1.2
forceencryption = 1
CONF

# Call the original entrypoint with the original command
exec /opt/mssql/bin/launch_sqlservr.sh /opt/mssql/bin/sqlservr
`
		containerOpts = append(containerOpts,
			testcontainers.WithEnv(map[string]string{
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
					Cmd:        []string{},
				},
			}),
		)
	}

	container, err := mssql.Run(ctx, "mcr.microsoft.com/mssql/server:2022-latest", containerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start SQL Server: %w", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get SQL Server connection string: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get SQL Server host: %w", err)
	}

	port, err := container.MappedPort(ctx, "1433")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get SQL Server port: %w", err)
	}

	// Run init script if provided
	if opts.InitScript != "" {
		// Wait a bit for SQL Server to fully initialize
		time.Sleep(config.SQLServerInitDelay)

		if err := runSQLServerInit(ctx, connStr, opts.InitScript); err != nil {
			_ = container.Terminate(ctx)
			return nil, fmt.Errorf("failed to run init script: %w", err)
		}
	}

	// Build internal connection with SSL info
	internal := config.InternalDBConnection{
		Host:     opts.NetworkAlias,
		Port:     1433,
		Username: "sa",
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

	return &SQLServerContainer{
		Container:    container,
		URL:          connStr,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
		Internal:     internal,
		SSL:          sslConnInfo,
	}, nil
}

// runSQLServerInit executes the init script on SQL Server.
// It splits the script by GO statements and executes each batch separately.
// Uses a dedicated connection to ensure USE statements persist across batches.
func runSQLServerInit(ctx context.Context, connStr, script string) error {
	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	// Verify connection is working before proceeding
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Get a dedicated connection to ensure USE statements persist across batches
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Close()

	// Split script by GO statements (case-insensitive, on its own line)
	// The regex matches GO on its own line, possibly with whitespace
	goPattern := regexp.MustCompile(`(?mi)^\s*GO\s*$`)
	batches := goPattern.Split(script, -1)

	for i, batch := range batches {
		batch = strings.TrimSpace(batch)
		if batch == "" {
			continue
		}

		_, err = conn.ExecContext(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to run batch %d: %w", i+1, err)
		}
	}

	return nil
}

// Stop terminates the SQL Server container.
func (s *SQLServerContainer) Stop(ctx context.Context) error {
	if s.Container != nil {
		return s.Container.Terminate(ctx)
	}

	return nil
}
