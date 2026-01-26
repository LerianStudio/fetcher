package containers

import (
	"context"
	"fmt"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// PostgresContainer wraps a PostgreSQL testcontainer with connection info.
type PostgresContainer struct {
	Container    *postgres.PostgresContainer
	URL          string
	Host         string
	Port         string
	InternalHost string
	Internal     config.InternalDBConnection
	SSL          *SSLConnectionInfo
}

// PostgresOptions configures PostgreSQL container startup.
type PostgresOptions struct {
	NetworkName   string
	NetworkAlias  string
	FixedHostPort string
	Username      string
	Password      string
	Database      string
	InitScript    string // SQL init script content
	SSL           *SSLConfig
}

// DefaultPostgresOptions returns default PostgreSQL options.
func DefaultPostgresOptions(networkName string) PostgresOptions {
	return PostgresOptions{
		NetworkName:  networkName,
		NetworkAlias: "postgres-external",
		Username:     "testuser",
		Password:     "testpass",
		Database:     "testdb",
	}
}

// DefaultPostgresSSLOptions returns PostgreSQL options with SSL enabled.
func DefaultPostgresSSLOptions(networkName string) PostgresOptions {
	opts := DefaultPostgresOptions(networkName)
	opts.NetworkAlias = "postgres-external-ssl"
	opts.SSL = &SSLConfig{
		Enabled: true,
		Mode:    "require", // Use 'require' for self-signed certs; 'verify-full' requires hostname match
	}

	return opts
}

// StartPostgres starts a PostgreSQL container with the given options.
func StartPostgres(ctx context.Context, opts PostgresOptions) (*PostgresContainer, error) {
	containerOpts := []testcontainers.ContainerCustomizer{
		postgres.WithUsername(opts.Username),
		postgres.WithPassword(opts.Password),
		postgres.WithDatabase(opts.Database),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(config.PostgresStartupTimeout),
		),
	}

	containerOpts = addInitScriptConfig(containerOpts, opts)

	var err error

	containerOpts, err = addSSLConfig(containerOpts, opts)
	if err != nil {
		return nil, err
	}

	containerOpts = addNetworkConfig(containerOpts, opts)
	containerOpts = addFixedPortConfig(containerOpts, opts)

	container, err := postgres.Run(ctx, "postgres:16", containerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start PostgreSQL: %w", err)
	}

	// Determine SSL mode for connection string
	sslMode := "disable"
	if opts.SSL != nil && opts.SSL.Enabled {
		sslMode = opts.SSL.Mode
		if sslMode == "" {
			sslMode = "require"
		}
	}

	connStr, err := container.ConnectionString(ctx, "sslmode="+sslMode)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get PostgreSQL connection string: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get PostgreSQL host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get PostgreSQL port: %w", err)
	}

	internal, sslConnInfo := buildInternalConnectionInfo(opts)

	return &PostgresContainer{
		Container:    container,
		URL:          connStr,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
		Internal:     internal,
		SSL:          sslConnInfo,
	}, nil
}

func addInitScriptConfig(containerOpts []testcontainers.ContainerCustomizer, opts PostgresOptions) []testcontainers.ContainerCustomizer {
	if opts.InitScript != "" {
		return append(containerOpts,
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

	return containerOpts
}

func addSSLConfig(containerOpts []testcontainers.ContainerCustomizer, opts PostgresOptions) ([]testcontainers.ContainerCustomizer, error) {
	if opts.SSL != nil && opts.SSL.Enabled && opts.SSL.CertBundle != nil {
		if opts.SSL.CertBundle.CACertPath == "" ||
			opts.SSL.CertBundle.ServerCertPath == "" ||
			opts.SSL.CertBundle.ServerKeyPath == "" {
			return nil, fmt.Errorf("SSL certificates must be written to files before starting container; call CertBundle.WriteToDir() first")
		}

		if opts.SSL.PostgresConfigPath == "" {
			return nil, fmt.Errorf("PostgresConfigPath must be set for PostgreSQL SSL containers")
		}

		return append(containerOpts,
			postgres.WithConfigFile(opts.SSL.PostgresConfigPath),
			postgres.WithSSLCert(
				opts.SSL.CertBundle.CACertPath,
				opts.SSL.CertBundle.ServerCertPath,
				opts.SSL.CertBundle.ServerKeyPath,
			),
		), nil
	}

	return containerOpts, nil
}

func addNetworkConfig(containerOpts []testcontainers.ContainerCustomizer, opts PostgresOptions) []testcontainers.ContainerCustomizer {
	if opts.NetworkName != "" {
		return append(containerOpts,
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{opts.NetworkName},
					NetworkAliases: map[string][]string{opts.NetworkName: {opts.NetworkAlias}},
				},
			}),
		)
	}

	return containerOpts
}

func addFixedPortConfig(containerOpts []testcontainers.ContainerCustomizer, opts PostgresOptions) []testcontainers.ContainerCustomizer {
	if opts.FixedHostPort != "" {
		return append(containerOpts, WithFixedPort("5432/tcp", opts.FixedHostPort))
	}

	return containerOpts
}

func buildInternalConnectionInfo(opts PostgresOptions) (config.InternalDBConnection, *SSLConnectionInfo) {
	internal := config.InternalDBConnection{
		Host:     opts.NetworkAlias,
		Port:     5432,
		Username: opts.Username,
		Password: opts.Password,
		Database: opts.Database,
	}

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

	return internal, sslConnInfo
}

// Stop terminates the PostgreSQL container.
func (p *PostgresContainer) Stop(ctx context.Context) error {
	if p.Container != nil {
		return p.Container.Terminate(ctx)
	}

	return nil
}

// GetHost returns the container's host address.
func (p *PostgresContainer) GetHost() string { return p.Host }

// GetPort returns the container's mapped port.
func (p *PostgresContainer) GetPort() string { return p.Port }

// GetURI returns the container's connection URI.
func (p *PostgresContainer) GetURI() string { return p.URL }
