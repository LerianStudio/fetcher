package containers

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// stringReader creates an io.Reader from a string.
func stringReader(s string) io.Reader {
	return strings.NewReader(s)
}

// PostgresContainer wraps a PostgreSQL testcontainer with connection info.
type PostgresContainer struct {
	Container    *postgres.PostgresContainer
	URL          string
	Host         string
	Port         string
	InternalHost string
	Internal     config.InternalDBConnection
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

	// Add init script if provided
	if opts.InitScript != "" {
		containerOpts = append(containerOpts,
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Files: []testcontainers.ContainerFile{
						{
							ContainerFilePath: "/docker-entrypoint-initdb.d/init.sql",
							Reader:            stringReader(opts.InitScript),
							FileMode:          0644,
						},
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
		containerOpts = append(containerOpts, WithFixedPort("5432/tcp", opts.FixedHostPort))
	}

	container, err := postgres.Run(ctx, "postgres:16", containerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start PostgreSQL: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
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

	return &PostgresContainer{
		Container:    container,
		URL:          connStr,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
		Internal: config.InternalDBConnection{
			Host:     opts.NetworkAlias,
			Port:     5432,
			Username: opts.Username,
			Password: opts.Password,
			Database: opts.Database,
		},
	}, nil
}

// Stop terminates the PostgreSQL container.
func (p *PostgresContainer) Stop(ctx context.Context) error {
	if p.Container != nil {
		return p.Container.Terminate(ctx)
	}
	return nil
}
