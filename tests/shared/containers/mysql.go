package containers

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// mysqlStringReader creates an io.Reader from a string.
func mysqlStringReader(s string) io.Reader {
	return strings.NewReader(s)
}

// MySQLContainer wraps a MySQL testcontainer with connection info.
type MySQLContainer struct {
	Container    *mysql.MySQLContainer
	URL          string
	Host         string
	Port         string
	InternalHost string
	Internal     config.InternalDBConnection
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
							Reader:            mysqlStringReader(opts.InitScript),
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

	return &MySQLContainer{
		Container:    container,
		URL:          connStr,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
		Internal: config.InternalDBConnection{
			Host:     opts.NetworkAlias,
			Port:     3306,
			Username: opts.Username,
			Password: opts.Password,
			Database: opts.Database,
		},
	}, nil
}

// Stop terminates the MySQL container.
func (m *MySQLContainer) Stop(ctx context.Context) error {
	if m.Container != nil {
		return m.Container.Terminate(ctx)
	}
	return nil
}
