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
}

// SQLServerOptions configures SQL Server container startup.
type SQLServerOptions struct {
	NetworkName   string
	NetworkAlias  string
	FixedHostPort string
	Password      string
	Database      string
	InitScript    string
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

	return &SQLServerContainer{
		Container:    container,
		URL:          connStr,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
		Internal: config.InternalDBConnection{
			Host:     opts.NetworkAlias,
			Port:     1433,
			Username: "sa",
			Password: opts.Password,
			Database: opts.Database,
		},
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
