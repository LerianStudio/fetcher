package setup

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/LerianStudio/fetcher/tests/integration/containers/fixtures"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/mssql"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// InfrastructureContainers holds all infrastructure containers.
type InfrastructureContainers struct {
	// Main MongoDB for fetcher-db (jobs, connections)
	MongoMain *mongodb.MongoDBContainer

	// RabbitMQ for message queue
	RabbitMQ *rabbitmq.RabbitMQContainer

	// SeaweedFS for file storage
	SeaweedFS testcontainers.Container

	// Valkey/Redis for caching
	Redis *redis.RedisContainer

	// External databases for test data
	PostgresExternal  *postgres.PostgresContainer
	MySQLExternal     *mysql.MySQLContainer
	SQLServerExternal *mssql.MSSQLServerContainer
	OracleExternal    testcontainers.Container // Generic container for Oracle XE
	MongoExternal     *mongodb.MongoDBContainer

	// Connection strings (host machine perspective)
	MongoMainURI     string
	MongoExternalURI string
	RabbitMQURI      string
	SeaweedFSURL     string
	RedisURL         string
	PostgresURL      string
	MySQLURL         string
	SQLServerURL     string
	OracleURL        string

	// Connection info using Docker network hostnames.
	// Works for both containers (Docker DNS) and host (via /etc/hosts mapping).
	PostgresInternal      InternalDBConnection
	MySQLInternal         InternalDBConnection
	SQLServerInternal     InternalDBConnection
	OracleInternal        InternalDBConnection
	MongoExternalInternal InternalDBConnection

	// Network for container communication
	//nolint:staticcheck // SA1019: Using deprecated Network type for named network support
	Network testcontainers.Network
}

// InternalDBConnection holds connection info accessible within the Docker network.
type InternalDBConnection struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

// InfrastructureOptions controls how infrastructure is started.
type InfrastructureOptions struct {
	// UseFixedPorts uses fixed host ports instead of random ports.
	// This enables infrastructure reuse and consistent debug configurations.
	UseFixedPorts bool

	// ReuseExisting attempts to connect to existing infrastructure
	// instead of starting new containers.
	ReuseExisting bool
}

// DefaultInfrastructureOptions returns options for normal test execution.
func DefaultInfrastructureOptions() InfrastructureOptions {
	return InfrastructureOptions{
		UseFixedPorts: false,
		ReuseExisting: false,
	}
}

// DebugInfrastructureOptions returns options for debug mode with fixed ports.
func DebugInfrastructureOptions() InfrastructureOptions {
	return InfrastructureOptions{
		UseFixedPorts: true,
		ReuseExisting: false,
	}
}

// ReuseInfrastructureOptions returns options for reusing existing infrastructure.
func ReuseInfrastructureOptions() InfrastructureOptions {
	return InfrastructureOptions{
		UseFixedPorts: true,
		ReuseExisting: true,
	}
}

// StartInfrastructure starts all infrastructure containers.
//
//nolint:gocognit,gocyclo // Complexity is inherent to coordinated container startup; splitting would reduce cohesion
func StartInfrastructure(ctx context.Context) (*InfrastructureContainers, error) {
	infra := &InfrastructureContainers{}

	// Create a shared network with a fixed name for application container compatibility
	// Note: Using deprecated GenericNetwork API because network.New() doesn't support custom names,
	// but applications.go requires the hardcoded "fetcher-test-network" name
	//nolint:staticcheck // SA1019: Using deprecated API for named network support
	net, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name:   "fetcher-test-network",
			Driver: "bridge",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	infra.Network = net

	networkName := "fetcher-test-network"

	// Start containers in parallel
	errChan := make(chan error, 9) // 7 original + 2 for SQL Server and Oracle

	// MongoDB Main
	go func() {
		container, err := mongodb.Run(ctx, "mongo:7.0",
			mongodb.WithUsername("root"),
			mongodb.WithPassword("password"),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"fetcher-mongodb"}},
				},
			}),
		)
		if err != nil {
			errChan <- fmt.Errorf("mongodb-main: %w", err)
			return
		}

		infra.MongoMain = container
		uri, _ := container.ConnectionString(ctx)
		infra.MongoMainURI = uri

		errChan <- nil
	}()

	// MongoDB External
	go func() {
		container, err := mongodb.Run(ctx, "mongo:7.0",
			mongodb.WithUsername("root"),
			mongodb.WithPassword("password"),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"mongodb-external"}},
				},
			}),
		)
		if err != nil {
			errChan <- fmt.Errorf("mongodb-external: %w", err)
			return
		}

		infra.MongoExternal = container
		uri, _ := container.ConnectionString(ctx)
		infra.MongoExternalURI = uri
		infra.MongoExternalInternal = InternalDBConnection{
			Host:     "mongodb-external",
			Port:     27017,
			Database: "testdb",
			Username: "root",
			Password: "password",
		}

		errChan <- nil
	}()

	// RabbitMQ
	go func() {
		container, err := rabbitmq.Run(ctx, "rabbitmq:4.0-management-alpine",
			rabbitmq.WithAdminUsername("guest"),
			rabbitmq.WithAdminPassword("guest"),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"fetcher-rabbitmq"}},
				},
			}),
			testcontainers.WithWaitStrategy(
				wait.ForAll(
					wait.ForLog("Server startup complete").WithStartupTimeout(RabbitMQStartupTimeout),
					wait.ForListeningPort("5672/tcp"),
				),
			),
		)
		if err != nil {
			errChan <- fmt.Errorf("rabbitmq: %w", err)
			return
		}
		// Note: RabbitMQ topology (exchanges, queues, bindings) is setup by
		// SetupRabbitMQTopology() called in SetupSuite after infrastructure starts

		infra.RabbitMQ = container
		uri, _ := container.AmqpURL(ctx)
		infra.RabbitMQURI = uri

		errChan <- nil
	}()

	// SeaweedFS
	go func() {
		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "chrislusf/seaweedfs:latest",
				ExposedPorts: []string{"9333/tcp", "8888/tcp", "8080/tcp"},
				Cmd:          []string{"server", "-dir=/data", "-master.port=9333", "-filer", "-filer.port=8888", "-volume.port=8080"},
				Networks:     []string{networkName},
				NetworkAliases: map[string][]string{
					networkName: {"fetcher-seaweedfs-filer"},
				},
				WaitingFor: wait.ForAll(
					wait.ForHTTP("/cluster/status").WithPort("9333/tcp").WithStartupTimeout(SeaweedFSStartupTimeout),
					wait.ForListeningPort("8888/tcp"),
				),
			},
			Started: true,
		})
		if err != nil {
			errChan <- fmt.Errorf("seaweedfs: %w", err)
			return
		}

		infra.SeaweedFS = container
		host, _ := container.Host(ctx)
		port, _ := container.MappedPort(ctx, "8888")
		infra.SeaweedFSURL = fmt.Sprintf("http://%s:%s", host, port.Port())

		errChan <- nil
	}()

	// Redis/Valkey
	go func() {
		container, err := redis.Run(ctx, "valkey/valkey:latest",
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"fetcher-valkey"}},
				},
			}),
		)
		if err != nil {
			errChan <- fmt.Errorf("redis: %w", err)
			return
		}

		infra.Redis = container
		uri, _ := container.ConnectionString(ctx)
		infra.RedisURL = uri

		errChan <- nil
	}()

	// PostgreSQL External (with init script for test data)
	go func() {
		initScript, err := fixtures.GetPostgresInitSQL()
		if err != nil {
			errChan <- fmt.Errorf("postgres init script: %w", err)
			return
		}

		// Write init script to temp file
		container, err := postgres.Run(ctx, "postgres:15-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpassword"),
			postgres.WithInitScripts(createTempFile(initScript, "postgres_init.sql")),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"postgres-external"}},
				},
			}),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(PostgresStartupTimeout),
			),
		)
		if err != nil {
			errChan <- fmt.Errorf("postgres: %w", err)
			return
		}

		infra.PostgresExternal = container
		uri, _ := container.ConnectionString(ctx, "sslmode=disable")
		infra.PostgresURL = uri
		infra.PostgresInternal = InternalDBConnection{
			Host:     "postgres-external",
			Port:     5432,
			Database: "testdb",
			Username: "testuser",
			Password: "testpassword",
		}

		errChan <- nil
	}()

	// MySQL External (with init script for test data)
	go func() {
		initScript, err := fixtures.GetMySQLInitSQL()
		if err != nil {
			errChan <- fmt.Errorf("mysql init script: %w", err)
			return
		}

		container, err := mysql.Run(ctx, "mysql:8.0",
			mysql.WithDatabase("testdb"),
			mysql.WithUsername("testuser"),
			mysql.WithPassword("testpassword"),
			mysql.WithScripts(createTempFile(initScript, "mysql_init.sql")),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"mysql-external"}},
				},
			}),
			testcontainers.WithWaitStrategy(
				wait.ForLog("ready for connections").
					WithOccurrence(2).
					WithStartupTimeout(MySQLStartupTimeout),
			),
		)
		if err != nil {
			errChan <- fmt.Errorf("mysql: %w", err)
			return
		}

		infra.MySQLExternal = container
		uri, _ := container.ConnectionString(ctx)
		infra.MySQLURL = uri
		infra.MySQLInternal = InternalDBConnection{
			Host:     "mysql-external",
			Port:     3306,
			Database: "testdb",
			Username: "testuser",
			Password: "testpassword",
		}

		errChan <- nil
	}()

	// SQL Server External (Q3 2024 data) - run init script manually for better error handling
	go func() {
		container, err := mssql.Run(ctx, "mcr.microsoft.com/mssql/server:2022-latest",
			mssql.WithAcceptEULA(),
			mssql.WithPassword("TestPassword123!"),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"mssql-external"}},
				},
			}),
			testcontainers.WithWaitStrategy(
				wait.ForLog("SQL Server is now ready for client connections").
					WithStartupTimeout(SQLServerStartupTimeout),
			),
		)
		if err != nil {
			errChan <- fmt.Errorf("mssql: %w", err)
			return
		}

		// Give SQL Server a moment to fully initialize
		time.Sleep(SQLServerInitDelay)

		// Execute init script via sqlcmd inside container
		initScript, err := fixtures.GetSQLServerInitSQL()
		if err != nil {
			errChan <- fmt.Errorf("mssql init script read: %w", err)
			return
		}

		// Copy script to container
		scriptPath := "/tmp/init.sql"
		if err := container.CopyToContainer(ctx, []byte(initScript), scriptPath, 0644); err != nil {
			errChan <- fmt.Errorf("mssql copy init script: %w", err)
			return
		}

		// Execute init script with sqlcmd
		exitCode, output, err := container.Exec(ctx, []string{
			"/opt/mssql-tools18/bin/sqlcmd",
			"-S", "localhost",
			"-U", "sa",
			"-P", "TestPassword123!",
			"-No",
			"-i", scriptPath,
		})
		if err != nil {
			errChan <- fmt.Errorf("mssql init script exec error: %w", err)
			return
		}

		if exitCode != 0 {
			// Read output for debugging
			outputBytes := make([]byte, 4096)

			n, _ := output.Read(outputBytes)
			errChan <- fmt.Errorf("mssql init script failed: exit=%d output=%s", exitCode, string(outputBytes[:n]))

			return
		}

		infra.SQLServerExternal = container
		uri, _ := container.ConnectionString(ctx)
		infra.SQLServerURL = uri
		infra.SQLServerInternal = InternalDBConnection{
			Host:     "mssql-external",
			Port:     1433,
			Database: "testdb",
			Username: "sa",
			Password: "TestPassword123!",
		}

		errChan <- nil
	}()

	// Oracle XE External (Special quarter data) - init script applied after startup
	go func() {
		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "gvenzl/oracle-xe:21-slim",
				ExposedPorts: []string{"1521/tcp"},
				Env: map[string]string{
					"ORACLE_PASSWORD":   "testpassword",
					"APP_USER":          "testuser",
					"APP_USER_PASSWORD": "testpassword",
				},
				Networks: []string{networkName},
				NetworkAliases: map[string][]string{
					networkName: {"oracle-external"},
				},
				WaitingFor: wait.ForAll(
					wait.ForLog("DATABASE IS READY TO USE!").WithStartupTimeout(OracleStartupTimeout),
					wait.ForListeningPort("1521/tcp"),
				),
			},
			Started: true,
		})
		if err != nil {
			errChan <- fmt.Errorf("oracle: %w", err)
			return
		}

		// Execute init script via sqlplus inside container
		initScript, err := fixtures.GetOracleInitSQL()
		if err != nil {
			errChan <- fmt.Errorf("oracle init script read: %w", err)
			return
		}

		// Write script to oracle's home directory (owned by oracle user) and execute
		// The container runs as 'oracle' user, so we need to write to a directory it owns
		oracleHome := "/opt/oracle"
		scriptPath := oracleHome + "/init.sql"

		err = container.CopyToContainer(ctx, []byte(initScript), scriptPath, 0644)
		if err != nil {
			errChan <- fmt.Errorf("oracle copy init script: %w", err)
			return
		}

		// Execute init script as testuser in XEPDB1 database
		exitCode, _, err := container.Exec(ctx, []string{
			"sqlplus", "-S",
			"testuser/testpassword@//localhost:1521/XEPDB1",
			"@" + scriptPath,
		})
		if err != nil || exitCode != 0 {
			errChan <- fmt.Errorf("oracle init script execution failed: exit=%d err=%v", exitCode, err)
			return
		}

		infra.OracleExternal = container
		host, _ := container.Host(ctx)
		port, _ := container.MappedPort(ctx, "1521")
		// Oracle connection string format for go-ora
		infra.OracleURL = fmt.Sprintf("oracle://testuser:testpassword@%s:%s/XEPDB1", host, port.Port())
		infra.OracleInternal = InternalDBConnection{
			Host:     "oracle-external",
			Port:     1521,
			Database: "XEPDB1", // Oracle service name
			Username: "testuser",
			Password: "testpassword",
		}

		errChan <- nil
	}()

	// Wait for all containers
	var errs []error

	for i := 0; i < 9; i++ {
		if err := <-errChan; err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		_ = infra.Stop(ctx)
		return nil, fmt.Errorf("failed to start infrastructure: %v", errs)
	}

	return infra, nil
}

// Stop stops all infrastructure containers.
//
//nolint:gocognit,gocyclo // Complexity from exhaustive nil checks for each container type
func (i *InfrastructureContainers) Stop(ctx context.Context) error {
	var errs []error

	if i.MongoMain != nil {
		if err := i.MongoMain.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.MongoExternal != nil {
		if err := i.MongoExternal.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.RabbitMQ != nil {
		if err := i.RabbitMQ.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.SeaweedFS != nil {
		if err := i.SeaweedFS.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Redis != nil {
		if err := i.Redis.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.PostgresExternal != nil {
		if err := i.PostgresExternal.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.MySQLExternal != nil {
		if err := i.MySQLExternal.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.SQLServerExternal != nil {
		if err := i.SQLServerExternal.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if i.OracleExternal != nil {
		if err := i.OracleExternal.Terminate(ctx); err != nil {
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

// createTempFile creates a temporary file with the given content and returns the path.
func createTempFile(content, name string) string {
	tmpDir := "/tmp"
	path := fmt.Sprintf("%s/%s", tmpDir, name)
	// Write content to file
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return ""
	}

	return path
}

// StartInfrastructureWithOptions starts infrastructure with specified options.
// If ReuseExisting is true and infrastructure is already running, it connects
// to the existing containers instead of starting new ones.
func StartInfrastructureWithOptions(ctx context.Context, opts InfrastructureOptions) (*InfrastructureContainers, error) {
	// Check for existing infrastructure if reuse is enabled
	if opts.ReuseExisting && InfraConfigExists() {
		infra, err := connectToExistingInfrastructure(ctx)
		if err == nil {
			return infra, nil
		}
		// If connection fails, fall through to start new infrastructure
		fmt.Printf("Warning: Could not connect to existing infrastructure: %v\n", err)
		fmt.Println("Starting new infrastructure...")
	}

	// Start new infrastructure
	return startNewInfrastructure(ctx, opts)
}

// connectToExistingInfrastructure connects to already-running infrastructure.
func connectToExistingInfrastructure(_ context.Context) (*InfrastructureContainers, error) {
	config, err := LoadInfraConfig(InfraConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load infrastructure config: %w", err)
	}

	// Build InfrastructureContainers from saved config
	// Note: Container handles are nil since we're connecting to existing containers
	infra := &InfrastructureContainers{
		MongoMainURI:     config.MongoMainURI,
		MongoExternalURI: config.MongoExternalURI,
		RabbitMQURI:      config.RabbitMQURI,
		SeaweedFSURL:     config.SeaweedFSURL,
		RedisURL:         config.RedisURL,
		PostgresURL:      config.PostgresURL,
		MySQLURL:         config.MySQLURL,
		SQLServerURL:     config.SQLServerURL,
		OracleURL:        config.OracleURL,

		PostgresInternal:      config.PostgresInternal,
		MySQLInternal:         config.MySQLInternal,
		SQLServerInternal:     config.SQLServerInternal,
		OracleInternal:        config.OracleInternal,
		MongoExternalInternal: config.MongoExternalInternal,
	}

	// Verify connectivity by pinging RabbitMQ
	// This ensures the infrastructure is actually running
	conn, err := amqp.Dial(config.RabbitMQURI)
	if err != nil {
		return nil, fmt.Errorf("infrastructure appears down, cannot connect to RabbitMQ: %w", err)
	}

	_ = conn.Close()

	return infra, nil
}

// Fixed port binding helper functions
// These use HostConfigModifier to bind to specific host ports when UseFixedPorts is true.
// For random ports, they return nil to let testcontainers modules use their defaults.

// portBindingModifier creates a HostConfigModifier that binds container ports to specific host ports.
func portBindingModifier(bindings map[string]string) func(hostConfig *container.HostConfig) {
	return func(hostConfig *container.HostConfig) {
		if hostConfig.PortBindings == nil {
			hostConfig.PortBindings = nat.PortMap{}
		}

		for containerPort, hostPort := range bindings {
			port := nat.Port(containerPort)
			hostConfig.PortBindings[port] = []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: hostPort},
			}
		}
	}
}

func getMongoHostConfigModifier(opts InfrastructureOptions, isExternal bool) func(*container.HostConfig) {
	if !opts.UseFixedPorts {
		return nil
	}

	hostPort := FixedPorts.MongoMain
	if isExternal {
		hostPort = FixedPorts.MongoExternal
	}

	return portBindingModifier(map[string]string{"27017/tcp": hostPort})
}

func getRabbitMQHostConfigModifier(opts InfrastructureOptions) func(*container.HostConfig) {
	if !opts.UseFixedPorts {
		return nil
	}

	return portBindingModifier(map[string]string{"5672/tcp": FixedPorts.RabbitMQ})
}

func getSeaweedFSExposedPorts() []string {
	// SeaweedFS is a GenericContainer, so we always need to specify ports
	return []string{"9333/tcp", "8888/tcp", "8080/tcp"}
}

func getSeaweedFSHostConfigModifier(opts InfrastructureOptions) func(*container.HostConfig) {
	if !opts.UseFixedPorts {
		return nil
	}

	return portBindingModifier(map[string]string{"8888/tcp": FixedPorts.SeaweedFSFiler})
}

func getRedisHostConfigModifier(opts InfrastructureOptions) func(*container.HostConfig) {
	if !opts.UseFixedPorts {
		return nil
	}

	return portBindingModifier(map[string]string{"6379/tcp": FixedPorts.Redis})
}

func getPostgresHostConfigModifier(opts InfrastructureOptions) func(*container.HostConfig) {
	if !opts.UseFixedPorts {
		return nil
	}

	return portBindingModifier(map[string]string{"5432/tcp": FixedPorts.Postgres})
}

func getMySQLHostConfigModifier(opts InfrastructureOptions) func(*container.HostConfig) {
	if !opts.UseFixedPorts {
		return nil
	}

	return portBindingModifier(map[string]string{"3306/tcp": FixedPorts.MySQL})
}

func getSQLServerHostConfigModifier(opts InfrastructureOptions) func(*container.HostConfig) {
	if !opts.UseFixedPorts {
		return nil
	}

	return portBindingModifier(map[string]string{"1433/tcp": FixedPorts.SQLServer})
}

func getOracleExposedPorts() []string {
	// Oracle is a GenericContainer, so we always need to specify ports
	return []string{"1521/tcp"}
}

func getOracleHostConfigModifier(opts InfrastructureOptions) func(*container.HostConfig) {
	if !opts.UseFixedPorts {
		return nil
	}

	return portBindingModifier(map[string]string{"1521/tcp": FixedPorts.Oracle})
}

// startNewInfrastructure starts new infrastructure containers with options.
//
//nolint:gocognit,gocyclo // Complexity is inherent to parallel container startup with configuration options
func startNewInfrastructure(ctx context.Context, opts InfrastructureOptions) (*InfrastructureContainers, error) {
	infra := &InfrastructureContainers{}

	// Create a shared network with a fixed name for application container compatibility
	// Note: Using deprecated GenericNetwork API because network.New() doesn't support custom names,
	// but applications.go requires the hardcoded "fetcher-test-network" name
	//nolint:staticcheck // SA1019: Using deprecated API for named network support
	net, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name:   "fetcher-test-network",
			Driver: "bridge",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	infra.Network = net

	networkName := "fetcher-test-network"

	// Start containers in parallel
	errChan := make(chan error, 9)

	// MongoDB Main
	go func() {
		reqOpts := []testcontainers.ContainerCustomizer{
			mongodb.WithUsername("root"),
			mongodb.WithPassword("password"),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"fetcher-mongodb"}},
				},
			}),
		}
		if modifier := getMongoHostConfigModifier(opts, false); modifier != nil {
			reqOpts = append(reqOpts, testcontainers.WithHostConfigModifier(modifier))
		}

		container, err := mongodb.Run(ctx, "mongo:7.0", reqOpts...)
		if err != nil {
			errChan <- fmt.Errorf("mongodb-main: %w", err)
			return
		}

		infra.MongoMain = container
		uri, _ := container.ConnectionString(ctx)
		infra.MongoMainURI = uri

		errChan <- nil
	}()

	// MongoDB External
	go func() {
		reqOpts := []testcontainers.ContainerCustomizer{
			mongodb.WithUsername("root"),
			mongodb.WithPassword("password"),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"mongodb-external"}},
				},
			}),
		}
		if modifier := getMongoHostConfigModifier(opts, true); modifier != nil {
			reqOpts = append(reqOpts, testcontainers.WithHostConfigModifier(modifier))
		}

		container, err := mongodb.Run(ctx, "mongo:7.0", reqOpts...)
		if err != nil {
			errChan <- fmt.Errorf("mongodb-external: %w", err)
			return
		}

		infra.MongoExternal = container
		uri, _ := container.ConnectionString(ctx)
		infra.MongoExternalURI = uri
		infra.MongoExternalInternal = InternalDBConnection{
			Host:     "mongodb-external",
			Port:     27017,
			Database: "testdb",
			Username: "root",
			Password: "password",
		}

		errChan <- nil
	}()

	// RabbitMQ
	go func() {
		reqOpts := []testcontainers.ContainerCustomizer{
			rabbitmq.WithAdminUsername("guest"),
			rabbitmq.WithAdminPassword("guest"),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"fetcher-rabbitmq"}},
				},
			}),
			testcontainers.WithWaitStrategy(
				wait.ForAll(
					wait.ForLog("Server startup complete").WithStartupTimeout(RabbitMQStartupTimeout),
					wait.ForListeningPort("5672/tcp"),
				),
			),
		}
		if modifier := getRabbitMQHostConfigModifier(opts); modifier != nil {
			reqOpts = append(reqOpts, testcontainers.WithHostConfigModifier(modifier))
		}

		container, err := rabbitmq.Run(ctx, "rabbitmq:4.0-management-alpine", reqOpts...)
		if err != nil {
			errChan <- fmt.Errorf("rabbitmq: %w", err)
			return
		}

		infra.RabbitMQ = container
		uri, _ := container.AmqpURL(ctx)
		infra.RabbitMQURI = uri

		errChan <- nil
	}()

	// SeaweedFS
	go func() {
		req := testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "chrislusf/seaweedfs:latest",
				ExposedPorts: getSeaweedFSExposedPorts(),
				Cmd:          []string{"server", "-dir=/data", "-master.port=9333", "-filer", "-filer.port=8888", "-volume.port=8080"},
				Networks:     []string{networkName},
				NetworkAliases: map[string][]string{
					networkName: {"fetcher-seaweedfs-filer"},
				},
				WaitingFor: wait.ForAll(
					wait.ForHTTP("/cluster/status").WithPort("9333/tcp").WithStartupTimeout(SeaweedFSStartupTimeout),
					wait.ForListeningPort("8888/tcp"),
				),
			},
			Started: true,
		}
		if modifier := getSeaweedFSHostConfigModifier(opts); modifier != nil {
			req.HostConfigModifier = modifier
		}

		ctr, err := testcontainers.GenericContainer(ctx, req)
		if err != nil {
			errChan <- fmt.Errorf("seaweedfs: %w", err)
			return
		}

		infra.SeaweedFS = ctr
		host, _ := ctr.Host(ctx)
		port, _ := ctr.MappedPort(ctx, "8888")
		infra.SeaweedFSURL = fmt.Sprintf("http://%s:%s", host, port.Port())

		errChan <- nil
	}()

	// Redis/Valkey
	go func() {
		reqOpts := []testcontainers.ContainerCustomizer{
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"fetcher-valkey"}},
				},
			}),
		}
		if modifier := getRedisHostConfigModifier(opts); modifier != nil {
			reqOpts = append(reqOpts, testcontainers.WithHostConfigModifier(modifier))
		}

		container, err := redis.Run(ctx, "valkey/valkey:latest", reqOpts...)
		if err != nil {
			errChan <- fmt.Errorf("redis: %w", err)
			return
		}

		infra.Redis = container
		uri, _ := container.ConnectionString(ctx)
		infra.RedisURL = uri

		errChan <- nil
	}()

	// PostgreSQL External
	go func() {
		initScript, err := fixtures.GetPostgresInitSQL()
		if err != nil {
			errChan <- fmt.Errorf("postgres init script: %w", err)
			return
		}

		reqOpts := []testcontainers.ContainerCustomizer{
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpassword"),
			postgres.WithInitScripts(createTempFile(initScript, "postgres_init.sql")),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"postgres-external"}},
				},
			}),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(PostgresStartupTimeout),
			),
		}
		if modifier := getPostgresHostConfigModifier(opts); modifier != nil {
			reqOpts = append(reqOpts, testcontainers.WithHostConfigModifier(modifier))
		}

		container, err := postgres.Run(ctx, "postgres:15-alpine", reqOpts...)
		if err != nil {
			errChan <- fmt.Errorf("postgres: %w", err)
			return
		}

		infra.PostgresExternal = container
		uri, _ := container.ConnectionString(ctx, "sslmode=disable")
		infra.PostgresURL = uri
		infra.PostgresInternal = InternalDBConnection{
			Host:     "postgres-external",
			Port:     5432,
			Database: "testdb",
			Username: "testuser",
			Password: "testpassword",
		}

		errChan <- nil
	}()

	// MySQL External
	go func() {
		initScript, err := fixtures.GetMySQLInitSQL()
		if err != nil {
			errChan <- fmt.Errorf("mysql init script: %w", err)
			return
		}

		reqOpts := []testcontainers.ContainerCustomizer{
			mysql.WithDatabase("testdb"),
			mysql.WithUsername("testuser"),
			mysql.WithPassword("testpassword"),
			mysql.WithScripts(createTempFile(initScript, "mysql_init.sql")),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"mysql-external"}},
				},
			}),
			testcontainers.WithWaitStrategy(
				wait.ForLog("ready for connections").
					WithOccurrence(2).
					WithStartupTimeout(MySQLStartupTimeout),
			),
		}
		if modifier := getMySQLHostConfigModifier(opts); modifier != nil {
			reqOpts = append(reqOpts, testcontainers.WithHostConfigModifier(modifier))
		}

		container, err := mysql.Run(ctx, "mysql:8.0", reqOpts...)
		if err != nil {
			errChan <- fmt.Errorf("mysql: %w", err)
			return
		}

		infra.MySQLExternal = container
		uri, _ := container.ConnectionString(ctx)
		infra.MySQLURL = uri
		infra.MySQLInternal = InternalDBConnection{
			Host:     "mysql-external",
			Port:     3306,
			Database: "testdb",
			Username: "testuser",
			Password: "testpassword",
		}

		errChan <- nil
	}()

	// SQL Server External
	go func() {
		reqOpts := []testcontainers.ContainerCustomizer{
			mssql.WithAcceptEULA(),
			mssql.WithPassword("TestPassword123!"),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{networkName},
					NetworkAliases: map[string][]string{networkName: {"mssql-external"}},
				},
			}),
			testcontainers.WithWaitStrategy(
				wait.ForLog("SQL Server is now ready for client connections").
					WithStartupTimeout(SQLServerStartupTimeout),
			),
		}
		if modifier := getSQLServerHostConfigModifier(opts); modifier != nil {
			reqOpts = append(reqOpts, testcontainers.WithHostConfigModifier(modifier))
		}

		ctr, err := mssql.Run(ctx, "mcr.microsoft.com/mssql/server:2022-latest", reqOpts...)
		if err != nil {
			errChan <- fmt.Errorf("mssql: %w", err)
			return
		}

		time.Sleep(SQLServerInitDelay)

		initScript, err := fixtures.GetSQLServerInitSQL()
		if err != nil {
			errChan <- fmt.Errorf("mssql init script read: %w", err)
			return
		}

		scriptPath := "/tmp/init.sql"
		if err := ctr.CopyToContainer(ctx, []byte(initScript), scriptPath, 0644); err != nil {
			errChan <- fmt.Errorf("mssql copy init script: %w", err)
			return
		}

		exitCode, output, err := ctr.Exec(ctx, []string{
			"/opt/mssql-tools18/bin/sqlcmd",
			"-S", "localhost",
			"-U", "sa",
			"-P", "TestPassword123!",
			"-No",
			"-i", scriptPath,
		})
		if err != nil {
			errChan <- fmt.Errorf("mssql init script exec error: %w", err)
			return
		}

		if exitCode != 0 {
			outputBytes := make([]byte, 4096)

			n, _ := output.Read(outputBytes)
			errChan <- fmt.Errorf("mssql init script failed: exit=%d output=%s", exitCode, string(outputBytes[:n]))

			return
		}

		infra.SQLServerExternal = ctr
		uri, _ := ctr.ConnectionString(ctx)
		infra.SQLServerURL = uri
		infra.SQLServerInternal = InternalDBConnection{
			Host:     "mssql-external",
			Port:     1433,
			Database: "testdb",
			Username: "sa",
			Password: "TestPassword123!",
		}

		errChan <- nil
	}()

	// Oracle XE External
	go func() {
		req := testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "gvenzl/oracle-xe:21-slim",
				ExposedPorts: getOracleExposedPorts(),
				Env: map[string]string{
					"ORACLE_PASSWORD":   "testpassword",
					"APP_USER":          "testuser",
					"APP_USER_PASSWORD": "testpassword",
				},
				Networks: []string{networkName},
				NetworkAliases: map[string][]string{
					networkName: {"oracle-external"},
				},
				WaitingFor: wait.ForAll(
					wait.ForLog("DATABASE IS READY TO USE!").WithStartupTimeout(OracleStartupTimeout),
					wait.ForListeningPort("1521/tcp"),
				),
			},
			Started: true,
		}
		if modifier := getOracleHostConfigModifier(opts); modifier != nil {
			req.HostConfigModifier = modifier
		}

		oracleCtr, err := testcontainers.GenericContainer(ctx, req)
		if err != nil {
			errChan <- fmt.Errorf("oracle: %w", err)
			return
		}

		initScript, err := fixtures.GetOracleInitSQL()
		if err != nil {
			errChan <- fmt.Errorf("oracle init script read: %w", err)
			return
		}

		oracleHome := "/opt/oracle"
		scriptPath := oracleHome + "/init.sql"

		err = oracleCtr.CopyToContainer(ctx, []byte(initScript), scriptPath, 0644)
		if err != nil {
			errChan <- fmt.Errorf("oracle copy init script: %w", err)
			return
		}

		exitCode, _, err := oracleCtr.Exec(ctx, []string{
			"sqlplus", "-S",
			"testuser/testpassword@//localhost:1521/XEPDB1",
			"@" + scriptPath,
		})
		if err != nil || exitCode != 0 {
			errChan <- fmt.Errorf("oracle init script execution failed: exit=%d err=%v", exitCode, err)
			return
		}

		infra.OracleExternal = oracleCtr
		host, _ := oracleCtr.Host(ctx)
		port, _ := oracleCtr.MappedPort(ctx, "1521")
		infra.OracleURL = fmt.Sprintf("oracle://testuser:testpassword@%s:%s/XEPDB1", host, port.Port())
		infra.OracleInternal = InternalDBConnection{
			Host:     "oracle-external",
			Port:     1521,
			Database: "XEPDB1",
			Username: "testuser",
			Password: "testpassword",
		}

		errChan <- nil
	}()

	// Wait for all containers
	var errs []error

	for i := 0; i < 9; i++ {
		if err := <-errChan; err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		_ = infra.Stop(ctx)
		return nil, fmt.Errorf("failed to start infrastructure: %v", errs)
	}

	// Save config if using fixed ports (for reuse support)
	if opts.UseFixedPorts {
		if err := saveInfraConfig(ctx, infra); err != nil {
			fmt.Printf("Warning: Could not save infrastructure config: %v\n", err)
		}
	}

	return infra, nil
}

// saveInfraConfig saves the infrastructure configuration for reuse.
func saveInfraConfig(ctx context.Context, infra *InfrastructureContainers) error {
	// Get mapped ports
	mongoMainPort, _ := infra.MongoMain.MappedPort(ctx, "27017")
	mongoExternalPort, _ := infra.MongoExternal.MappedPort(ctx, "27017")
	rabbitmqPort, _ := infra.RabbitMQ.MappedPort(ctx, "5672")
	seaweedfsPort, _ := infra.SeaweedFS.MappedPort(ctx, "8888")
	redisPort, _ := infra.Redis.MappedPort(ctx, "6379")
	postgresPort, _ := infra.PostgresExternal.MappedPort(ctx, "5432")
	mysqlPort, _ := infra.MySQLExternal.MappedPort(ctx, "3306")
	sqlserverPort, _ := infra.SQLServerExternal.MappedPort(ctx, "1433")
	oraclePort, _ := infra.OracleExternal.MappedPort(ctx, "1521")

	config := &InfraConfig{
		NetworkName:      "fetcher-test-network",
		MongoMainURI:     infra.MongoMainURI,
		MongoExternalURI: infra.MongoExternalURI,
		RabbitMQURI:      infra.RabbitMQURI,
		SeaweedFSURL:     infra.SeaweedFSURL,
		RedisURL:         infra.RedisURL,
		PostgresURL:      infra.PostgresURL,
		MySQLURL:         infra.MySQLURL,
		SQLServerURL:     infra.SQLServerURL,
		OracleURL:        infra.OracleURL,
		Ports: InfraPorts{
			MongoMain:      mongoMainPort.Port(),
			MongoExternal:  mongoExternalPort.Port(),
			RabbitMQ:       rabbitmqPort.Port(),
			SeaweedFSFiler: seaweedfsPort.Port(),
			Redis:          redisPort.Port(),
			Postgres:       postgresPort.Port(),
			MySQL:          mysqlPort.Port(),
			SQLServer:      sqlserverPort.Port(),
			Oracle:         oraclePort.Port(),
		},
		PostgresInternal:      infra.PostgresInternal,
		MySQLInternal:         infra.MySQLInternal,
		SQLServerInternal:     infra.SQLServerInternal,
		OracleInternal:        infra.OracleInternal,
		MongoExternalInternal: infra.MongoExternalInternal,
	}

	return config.Save(InfraConfigPath)
}
