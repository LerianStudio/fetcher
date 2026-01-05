// Package setup provides orchestration for shared test infrastructure.
// It uses the container wrappers from tests/shared/containers to start
// all required infrastructure in parallel with proper configuration.
//
// This package is designed to be used by both integration tests and chaos tests.
// Chaos tests can compose SharedInfrastructure with Toxiproxy for network chaos injection.
package setup

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"

	"github.com/LerianStudio/fetcher/tests/shared/config"
	"github.com/LerianStudio/fetcher/tests/shared/containers"
	"github.com/LerianStudio/fetcher/tests/shared/fixtures"
	"github.com/LerianStudio/fetcher/tests/shared/network"
	"github.com/LerianStudio/fetcher/tests/shared/topology"
)

// SharedInfrastructure holds all infrastructure containers and connection info.
// This struct is the single source of truth for test infrastructure.
type SharedInfrastructure struct {
	// Network for container communication
	//nolint:staticcheck // SA1019: Using deprecated Network type for named network support
	Network testcontainers.Network

	// Core infrastructure containers
	MongoMain *containers.MongoDBContainer
	RabbitMQ  *containers.RabbitMQContainer
	SeaweedFS *containers.SeaweedFSContainers
	Redis     *containers.RedisContainer

	// External database containers (for test data extraction)
	MongoExternal     *containers.MongoDBContainer
	PostgresExternal  *containers.PostgresContainer
	MySQLExternal     *containers.MySQLContainer
	SQLServerExternal *containers.SQLServerContainer
	OracleExternal    *containers.OracleContainer
}

// InfrastructureOptions controls how infrastructure is started.
type InfrastructureOptions struct {
	// UseFixedPorts uses fixed host ports instead of random ports.
	// This enables infrastructure reuse and consistent debug configurations.
	UseFixedPorts bool

	// ReuseExisting attempts to connect to existing infrastructure
	// instead of starting new containers.
	ReuseExisting bool

	// SkipExternalDBs skips starting external databases (Postgres, MySQL, SQLServer, Oracle).
	// Useful for tests that only need core infrastructure.
	SkipExternalDBs bool

	// InitScripts controls whether to run init scripts for databases.
	// Set to false for chaos tests that need empty databases.
	InitScripts bool
}

// DefaultOptions returns options for normal test execution with all containers.
func DefaultOptions() InfrastructureOptions {
	return InfrastructureOptions{
		UseFixedPorts:   false,
		ReuseExisting:   false,
		SkipExternalDBs: false,
		InitScripts:     true,
	}
}

// DebugOptions returns options for debug mode with fixed ports.
func DebugOptions() InfrastructureOptions {
	return InfrastructureOptions{
		UseFixedPorts:   true,
		ReuseExisting:   false,
		SkipExternalDBs: false,
		InitScripts:     true,
	}
}

// ReuseOptions returns options for reusing existing infrastructure.
func ReuseOptions() InfrastructureOptions {
	return InfrastructureOptions{
		UseFixedPorts:   true,
		ReuseExisting:   true,
		SkipExternalDBs: false,
		InitScripts:     true,
	}
}

// CoreOnlyOptions returns options for starting only core infrastructure.
func CoreOnlyOptions() InfrastructureOptions {
	return InfrastructureOptions{
		UseFixedPorts:   false,
		ReuseExisting:   false,
		SkipExternalDBs: true,
		InitScripts:     false,
	}
}

// Start starts all infrastructure containers with default options.
func Start(ctx context.Context) (*SharedInfrastructure, error) {
	return StartWithOptions(ctx, DefaultOptions())
}

// StartWithOptions starts infrastructure with specified options.
// If ReuseExisting is true and infrastructure is already running, it connects
// to the existing containers instead of starting new ones.
func StartWithOptions(ctx context.Context, opts InfrastructureOptions) (*SharedInfrastructure, error) {
	// Check for existing infrastructure if reuse is enabled
	if opts.ReuseExisting && config.InfraConfigExists() {
		infra, err := connectToExisting(ctx)
		if err == nil {
			return infra, nil
		}
		// If connection fails, fall through to start new infrastructure
		fmt.Printf("Warning: Could not connect to existing infrastructure: %v\n", err)
		fmt.Println("Starting new infrastructure...")
	}

	return startNew(ctx, opts)
}

// startNew starts new infrastructure containers.
//
//nolint:gocognit,gocyclo // Complexity is inherent to parallel container startup
func startNew(ctx context.Context, opts InfrastructureOptions) (*SharedInfrastructure, error) {
	infra := &SharedInfrastructure{}

	// Create network
	net, err := network.CreateNetwork(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	infra.Network = net

	networkName := config.NetworkName

	// Determine number of containers to start
	containerCount := 5 // MongoMain, MongoExternal, RabbitMQ, SeaweedFS, Redis
	if !opts.SkipExternalDBs {
		containerCount += 4 // Postgres, MySQL, SQLServer, Oracle
	}

	errChan := make(chan error, containerCount)

	// Get fixed ports if needed
	var fixedPorts config.PortsConfig
	if opts.UseFixedPorts {
		fixedPorts = config.FixedPorts
	}

	// Start MongoDB Main
	go func() {
		mongoOpts := containers.DefaultMongoDBMainOptions(networkName)
		if opts.UseFixedPorts {
			mongoOpts.FixedHostPort = fixedPorts.MongoMain
		}

		ctr, err := containers.StartMongoDB(ctx, mongoOpts)
		if err != nil {
			errChan <- fmt.Errorf("mongodb-main: %w", err)
			return
		}
		infra.MongoMain = ctr
		errChan <- nil
	}()

	// Start MongoDB External
	go func() {
		mongoOpts := containers.DefaultMongoDBExternalOptions(networkName)
		if opts.UseFixedPorts {
			mongoOpts.FixedHostPort = fixedPorts.MongoExternal
		}

		ctr, err := containers.StartMongoDB(ctx, mongoOpts)
		if err != nil {
			errChan <- fmt.Errorf("mongodb-external: %w", err)
			return
		}
		infra.MongoExternal = ctr
		errChan <- nil
	}()

	// Start RabbitMQ
	go func() {
		rabbitOpts := containers.DefaultRabbitMQOptions(networkName)
		if opts.UseFixedPorts {
			rabbitOpts.FixedHostPort = fixedPorts.RabbitMQ
		}

		ctr, err := containers.StartRabbitMQ(ctx, rabbitOpts)
		if err != nil {
			errChan <- fmt.Errorf("rabbitmq: %w", err)
			return
		}
		infra.RabbitMQ = ctr
		errChan <- nil
	}()

	// Start SeaweedFS
	go func() {
		seaweedOpts := containers.DefaultSeaweedFSOptions(networkName)
		if opts.UseFixedPorts {
			seaweedOpts.FixedHostPort = fixedPorts.SeaweedFSFiler
		}

		ctr, err := containers.StartSeaweedFS(ctx, seaweedOpts)
		if err != nil {
			errChan <- fmt.Errorf("seaweedfs: %w", err)
			return
		}
		infra.SeaweedFS = ctr
		errChan <- nil
	}()

	// Start Redis
	go func() {
		redisOpts := containers.DefaultRedisOptions(networkName)
		if opts.UseFixedPorts {
			redisOpts.FixedHostPort = fixedPorts.Redis
		}

		ctr, err := containers.StartRedis(ctx, redisOpts)
		if err != nil {
			errChan <- fmt.Errorf("redis: %w", err)
			return
		}
		infra.Redis = ctr
		errChan <- nil
	}()

	// Start external databases if not skipped
	if !opts.SkipExternalDBs {
		// Start PostgreSQL
		go func() {
			postgresOpts := containers.DefaultPostgresOptions(networkName)
			if opts.UseFixedPorts {
				postgresOpts.FixedHostPort = fixedPorts.Postgres
			}
			if opts.InitScripts {
				initSQL, err := fixtures.GetPostgresInitSQL()
				if err != nil {
					errChan <- fmt.Errorf("postgres init script: %w", err)
					return
				}
				postgresOpts.InitScript = initSQL
			}

			ctr, err := containers.StartPostgres(ctx, postgresOpts)
			if err != nil {
				errChan <- fmt.Errorf("postgres: %w", err)
				return
			}
			infra.PostgresExternal = ctr
			errChan <- nil
		}()

		// Start MySQL
		go func() {
			mysqlOpts := containers.DefaultMySQLOptions(networkName)
			if opts.UseFixedPorts {
				mysqlOpts.FixedHostPort = fixedPorts.MySQL
			}
			if opts.InitScripts {
				initSQL, err := fixtures.GetMySQLInitSQL()
				if err != nil {
					errChan <- fmt.Errorf("mysql init script: %w", err)
					return
				}
				mysqlOpts.InitScript = initSQL
			}

			ctr, err := containers.StartMySQL(ctx, mysqlOpts)
			if err != nil {
				errChan <- fmt.Errorf("mysql: %w", err)
				return
			}
			infra.MySQLExternal = ctr
			errChan <- nil
		}()

		// Start SQL Server
		go func() {
			sqlserverOpts := containers.DefaultSQLServerOptions(networkName)
			if opts.UseFixedPorts {
				sqlserverOpts.FixedHostPort = fixedPorts.SQLServer
			}
			if opts.InitScripts {
				initSQL, err := fixtures.GetSQLServerInitSQL()
				if err != nil {
					errChan <- fmt.Errorf("sqlserver init script: %w", err)
					return
				}
				sqlserverOpts.InitScript = initSQL
			}

			ctr, err := containers.StartSQLServer(ctx, sqlserverOpts)
			if err != nil {
				errChan <- fmt.Errorf("sqlserver: %w", err)
				return
			}
			infra.SQLServerExternal = ctr
			errChan <- nil
		}()

		// Start Oracle
		go func() {
			oracleOpts := containers.DefaultOracleOptions(networkName)
			if opts.UseFixedPorts {
				oracleOpts.FixedHostPort = fixedPorts.Oracle
			}
			if opts.InitScripts {
				initSQL, err := fixtures.GetOracleInitSQL()
				if err != nil {
					errChan <- fmt.Errorf("oracle init script: %w", err)
					return
				}
				oracleOpts.InitScript = initSQL
			}

			ctr, err := containers.StartOracle(ctx, oracleOpts)
			if err != nil {
				errChan <- fmt.Errorf("oracle: %w", err)
				return
			}
			infra.OracleExternal = ctr
			errChan <- nil
		}()
	}

	// Wait for all containers
	var errs []error
	for i := 0; i < containerCount; i++ {
		if err := <-errChan; err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		_ = infra.Stop(ctx)
		return nil, fmt.Errorf("failed to start infrastructure: %v", errs)
	}

	// Save config for reuse if using fixed ports
	if opts.UseFixedPorts {
		if err := infra.SaveConfig(); err != nil {
			fmt.Printf("Warning: Could not save infrastructure config: %v\n", err)
		}
	}

	return infra, nil
}

// connectToExisting connects to already-running infrastructure.
func connectToExisting(ctx context.Context) (*SharedInfrastructure, error) {
	cfg, err := config.LoadInfraConfig(config.InfraConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load infrastructure config: %w", err)
	}

	// Verify connectivity by setting up RabbitMQ topology
	// This ensures the infrastructure is actually running
	if err := topology.SetupRabbitMQTopology(ctx, cfg.RabbitMQURI); err != nil {
		return nil, fmt.Errorf("infrastructure appears down: %w", err)
	}

	// Build SharedInfrastructure from saved config
	// Note: Container handles are nil since we're connecting to existing containers
	infra := &SharedInfrastructure{
		MongoMain: &containers.MongoDBContainer{
			URI:          cfg.MongoMainURI,
			InternalHost: "fetcher-mongodb",
		},
		MongoExternal: &containers.MongoDBContainer{
			URI:          cfg.MongoExternalURI,
			InternalHost: "fetcher-mongodb-external",
		},
		RabbitMQ: &containers.RabbitMQContainer{
			URI:          cfg.RabbitMQURI,
			InternalHost: "fetcher-rabbitmq",
		},
		SeaweedFS: &containers.SeaweedFSContainers{
			URL:          cfg.SeaweedFSURL,
			InternalHost: "fetcher-seaweedfs-filer",
		},
		Redis: &containers.RedisContainer{
			URL:          cfg.RedisURL,
			InternalHost: "fetcher-valkey",
		},
		PostgresExternal: &containers.PostgresContainer{
			URL:          cfg.PostgresURL,
			InternalHost: "postgres-external",
			Internal:     cfg.PostgresInternal,
		},
		MySQLExternal: &containers.MySQLContainer{
			URL:          cfg.MySQLURL,
			InternalHost: "mysql-external",
			Internal:     cfg.MySQLInternal,
		},
		SQLServerExternal: &containers.SQLServerContainer{
			URL:          cfg.SQLServerURL,
			InternalHost: "sqlserver-external",
			Internal:     cfg.SQLServerInternal,
		},
		OracleExternal: &containers.OracleContainer{
			URL:          cfg.OracleURL,
			InternalHost: "oracle-external",
			Internal:     cfg.OracleInternal,
		},
	}

	return infra, nil
}

// Stop terminates all infrastructure containers.
func (i *SharedInfrastructure) Stop(ctx context.Context) error {
	var errs []error

	if i.MongoMain != nil {
		if err := i.MongoMain.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.MongoExternal != nil {
		if err := i.MongoExternal.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.RabbitMQ != nil {
		if err := i.RabbitMQ.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.SeaweedFS != nil {
		if err := i.SeaweedFS.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Redis != nil {
		if err := i.Redis.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.PostgresExternal != nil {
		if err := i.PostgresExternal.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.MySQLExternal != nil {
		if err := i.MySQLExternal.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.SQLServerExternal != nil {
		if err := i.SQLServerExternal.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.OracleExternal != nil {
		if err := i.OracleExternal.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Network != nil {
		if err := i.Network.Remove(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping containers: %v", errs)
	}

	return nil
}

// SetupRabbitMQTopology creates the required exchanges and queues.
func (i *SharedInfrastructure) SetupRabbitMQTopology(ctx context.Context) error {
	if i.RabbitMQ == nil {
		return fmt.Errorf("RabbitMQ container not available")
	}
	return topology.SetupRabbitMQTopology(ctx, i.RabbitMQ.URI)
}

// PurgeTestQueue purges the test.job.events queue.
func (i *SharedInfrastructure) PurgeTestQueue(ctx context.Context) (int, error) {
	if i.RabbitMQ == nil {
		return 0, fmt.Errorf("RabbitMQ container not available")
	}
	return topology.PurgeTestQueue(ctx, i.RabbitMQ.URI)
}

// SaveConfig saves the infrastructure configuration for reuse.
func (i *SharedInfrastructure) SaveConfig() error {
	cfg := &config.InfraConfig{
		NetworkName: config.NetworkName,
	}

	// Set URIs from containers
	if i.MongoMain != nil {
		cfg.MongoMainURI = i.MongoMain.URI
	}
	if i.MongoExternal != nil {
		cfg.MongoExternalURI = i.MongoExternal.URI
	}
	if i.RabbitMQ != nil {
		cfg.RabbitMQURI = i.RabbitMQ.URI
	}
	if i.SeaweedFS != nil {
		cfg.SeaweedFSURL = i.SeaweedFS.URL
	}
	if i.Redis != nil {
		cfg.RedisURL = i.Redis.URL
	}
	if i.PostgresExternal != nil {
		cfg.PostgresURL = i.PostgresExternal.URL
		cfg.PostgresInternal = i.PostgresExternal.Internal
	}
	if i.MySQLExternal != nil {
		cfg.MySQLURL = i.MySQLExternal.URL
		cfg.MySQLInternal = i.MySQLExternal.Internal
	}
	if i.SQLServerExternal != nil {
		cfg.SQLServerURL = i.SQLServerExternal.URL
		cfg.SQLServerInternal = i.SQLServerExternal.Internal
	}
	if i.OracleExternal != nil {
		cfg.OracleURL = i.OracleExternal.URL
		cfg.OracleInternal = i.OracleExternal.Internal
	}

	// Set ports from containers
	cfg.Ports = config.InfraPorts{
		MongoMain:      getPort(i.MongoMain),
		MongoExternal:  getPort(i.MongoExternal),
		RabbitMQ:       getRabbitMQPort(i.RabbitMQ),
		SeaweedFSFiler: getSeaweedFSPort(i.SeaweedFS),
		Redis:          getRedisPort(i.Redis),
		Postgres:       getPostgresPort(i.PostgresExternal),
		MySQL:          getMySQLPort(i.MySQLExternal),
		SQLServer:      getSQLServerPort(i.SQLServerExternal),
		Oracle:         getOraclePort(i.OracleExternal),
	}

	return cfg.Save(config.InfraConfigPath)
}

// Helper functions to get ports safely
func getPort(m *containers.MongoDBContainer) string {
	if m != nil {
		return m.Port
	}
	return ""
}

func getRabbitMQPort(r *containers.RabbitMQContainer) string {
	if r != nil {
		return r.Port
	}
	return ""
}

func getSeaweedFSPort(s *containers.SeaweedFSContainers) string {
	if s != nil {
		return s.Port
	}
	return ""
}

func getRedisPort(r *containers.RedisContainer) string {
	if r != nil {
		return r.Port
	}
	return ""
}

func getPostgresPort(p *containers.PostgresContainer) string {
	if p != nil {
		return p.Port
	}
	return ""
}

func getMySQLPort(m *containers.MySQLContainer) string {
	if m != nil {
		return m.Port
	}
	return ""
}

func getSQLServerPort(s *containers.SQLServerContainer) string {
	if s != nil {
		return s.Port
	}
	return ""
}

func getOraclePort(o *containers.OracleContainer) string {
	if o != nil {
		return o.Port
	}
	return ""
}

// GetMongoExternalInternal returns the internal connection info for MongoDB External.
// This is used by integration tests to configure connections for the fetcher application.
func (i *SharedInfrastructure) GetMongoExternalInternal() config.InternalDBConnection {
	if i.MongoExternal == nil {
		return config.InternalDBConnection{}
	}
	return config.InternalDBConnection{
		Host:     i.MongoExternal.InternalHost,
		Port:     27017,
		Username: "root",
		Password: "password",
		Database: "external_transactions",
	}
}

// =============================================================================
// Convenience Accessor Methods
// =============================================================================
// These methods provide a flatter API for accessing common URIs and connection info.
// They are used by integration tests and other consumers that need simple access.

// RabbitMQURI returns the RabbitMQ AMQP URI.
func (i *SharedInfrastructure) RabbitMQURI() string {
	if i.RabbitMQ == nil {
		return ""
	}
	return i.RabbitMQ.URI
}

// SeaweedFSURL returns the SeaweedFS filer URL.
func (i *SharedInfrastructure) SeaweedFSURL() string {
	if i.SeaweedFS == nil {
		return ""
	}
	return i.SeaweedFS.URL
}

// RedisURL returns the Redis URL.
func (i *SharedInfrastructure) RedisURL() string {
	if i.Redis == nil {
		return ""
	}
	return i.Redis.URL
}

// MongoMainURI returns the main MongoDB URI.
func (i *SharedInfrastructure) MongoMainURI() string {
	if i.MongoMain == nil {
		return ""
	}
	return i.MongoMain.URI
}

// MongoExternalURI returns the external MongoDB URI.
func (i *SharedInfrastructure) MongoExternalURI() string {
	if i.MongoExternal == nil {
		return ""
	}
	return i.MongoExternal.URI
}

// PostgresInternal returns the internal connection info for PostgreSQL.
func (i *SharedInfrastructure) PostgresInternal() config.InternalDBConnection {
	if i.PostgresExternal == nil {
		return config.InternalDBConnection{}
	}
	return i.PostgresExternal.Internal
}

// MySQLInternal returns the internal connection info for MySQL.
func (i *SharedInfrastructure) MySQLInternal() config.InternalDBConnection {
	if i.MySQLExternal == nil {
		return config.InternalDBConnection{}
	}
	return i.MySQLExternal.Internal
}

// SQLServerInternal returns the internal connection info for SQL Server.
func (i *SharedInfrastructure) SQLServerInternal() config.InternalDBConnection {
	if i.SQLServerExternal == nil {
		return config.InternalDBConnection{}
	}
	return i.SQLServerExternal.Internal
}

// OracleInternal returns the internal connection info for Oracle.
func (i *SharedInfrastructure) OracleInternal() config.InternalDBConnection {
	if i.OracleExternal == nil {
		return config.InternalDBConnection{}
	}
	return i.OracleExternal.Internal
}

// MongoExternalInternal returns the internal connection info for external MongoDB.
func (i *SharedInfrastructure) MongoExternalInternal() config.InternalDBConnection {
	return i.GetMongoExternalInternal()
}

// PostgresURL returns the external PostgreSQL URL.
func (i *SharedInfrastructure) PostgresURL() string {
	if i.PostgresExternal == nil {
		return ""
	}
	return i.PostgresExternal.URL
}

// MySQLURL returns the external MySQL URL.
func (i *SharedInfrastructure) MySQLURL() string {
	if i.MySQLExternal == nil {
		return ""
	}
	return i.MySQLExternal.URL
}

// SQLServerURL returns the external SQL Server URL.
func (i *SharedInfrastructure) SQLServerURL() string {
	if i.SQLServerExternal == nil {
		return ""
	}
	return i.SQLServerExternal.URL
}

// OracleURL returns the external Oracle URL.
func (i *SharedInfrastructure) OracleURL() string {
	if i.OracleExternal == nil {
		return ""
	}
	return i.OracleExternal.URL
}
