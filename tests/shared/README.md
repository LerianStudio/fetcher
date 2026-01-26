# tests/shared - Shared Test Infrastructure

Centralized testing infrastructure library for integration tests, chaos tests, and direct datasource testing.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Quick Start](#quick-start)
  - [Option A: Direct Datasource Testing (Lightweight)](#option-a-direct-datasource-testing-lightweight)
  - [Option B: Full Stack Testing (End-to-End)](#option-b-full-stack-testing-end-to-end)
- [Package Reference](#package-reference)
  - [config](#config)
  - [client](#client)
  - [containers](#containers)
  - [network](#network)
  - [fixtures](#fixtures)
  - [topology](#topology)
- [Complete Integration Example](#complete-integration-example)
- [Datasource Integration Test Example](#datasource-integration-test-example)
- [Dependencies](#dependencies)
- [Key Design Decisions](#key-design-decisions)
- [setup](#setup)
- [Toxiproxy (containers)](#toxiproxy-containers)
- [Chaos Testing Framework (tests/shared/chaos)](#chaos-testing-framework-testssharedchaos)

---

## Architecture Overview

```
tests/shared/
├── config/       # Configuration, constants, timeouts
├── client/       # HTTP clients for API interactions
├── containers/   # Docker container orchestration (individual containers + Toxiproxy)
├── network/      # Docker network management
├── fixtures/     # Test data and SQL initialization
├── setup/        # Infrastructure orchestration (parallel startup, reuse)
└── topology/     # RabbitMQ exchange/queue setup
```

**Design Patterns:**
- **Single Responsibility**: Each package has one focused purpose
- **Factory Pattern**: `Default*Options()` functions + `Start*()` functions
- **Wrapper Pattern**: Container structs encapsulate testcontainers with connection info
- **Options Pattern**: Flexible configuration with sensible defaults
- **Embedded Resources**: SQL fixtures embedded in binary via `//go:embed`

---

## Quick Start

This library supports two main usage patterns:

### Option A: Direct Datasource Testing (Lightweight)

Test datasources directly without Manager API, RabbitMQ, or SeaweedFS. Ideal for unit/component tests of datasource packages.

```go
import (
    "github.com/LerianStudio/fetcher/pkg/postgres"
    "github.com/LerianStudio/fetcher/tests/shared/config"
    "github.com/LerianStudio/fetcher/tests/shared/containers"
    "github.com/LerianStudio/fetcher/tests/shared/fixtures"
    "github.com/LerianStudio/fetcher/tests/shared/network"
    libCommons "github.com/LerianStudio/lib-commons/v2/commons"
)

// 1. Create network
net, _ := network.CreateNetwork(ctx)

// 2. Start database container
pgOpts := containers.DefaultPostgresOptions(config.NetworkName)
pgOpts.InitScript, _ = fixtures.GetPostgresInitSQL()
pg, _ := containers.StartPostgres(ctx, pgOpts)

// 3. Create datasource directly (no Manager API needed!)
logger := libCommons.NewLoggerFromContext(ctx)
conn := &postgres.Connection{
    ConnectionString:   pg.URL,  // Container provides ready-to-use connection string
    DBName:             pg.Internal.Database,
    Logger:             logger,
    MaxOpenConnections: 25,
    MaxIdleConnections: 5,
}
ds, _ := postgres.NewDataSourceRepository(conn)
defer ds.CloseConnection()

// 4. Test datasource directly
schema, _ := ds.GetDatabaseSchema(ctx, []string{"public"})
results, _ := ds.Query(ctx, schema, "transactions", []string{"id", "amount"}, nil)

// OR

// 1. Start database container
accountsInitSQL = `
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    alias VARCHAR(32) NOT NULL UNIQUE,
    balance DECIMAL(19,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_accounts_alias ON accounts(alias);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
`

pgOpts := containers.PostgresOptions{
	NetworkAlias: "test-postgres",
	Username:     "testuser",
	Password:     "testpass",
	Database:     "testdb",
	InitScript:   accountsInitSQL,
}

testPostgresContainer, err := containers.StartPostgres(ctx, pgOpts)
if err != nil {
	fmt.Fprintf(os.Stderr, "failed to start PostgreSQL container: %v\n", err)
	os.Exit(1)
}

// 2. Create datasource directly (no Manager API needed!)
logger := libCommons.NewLoggerFromContext(ctx)
conn := &postgres.Connection{
    ConnectionString:   pg.URL,  // Container provides ready-to-use connection string
    DBName:             pg.Internal.Database,
    Logger:             logger,
    MaxOpenConnections: 25,
    MaxIdleConnections: 5,
}
ds, _ := postgres.NewDataSourceRepository(conn)
defer ds.CloseConnection()

// 3. Test datasource directly
schema, _ := ds.GetDatabaseSchema(ctx, []string{"public"})
results, _ := ds.Query(ctx, schema, "transactions", []string{"id", "amount"}, nil)

```

### Option B: Full Stack Testing (End-to-End)

Test complete flow through Manager API with all infrastructure. Ideal for E2E and integration tests.

```go
import (
    "github.com/LerianStudio/fetcher/tests/shared/client"
    "github.com/LerianStudio/fetcher/tests/shared/config"
    "github.com/LerianStudio/fetcher/tests/shared/containers"
    "github.com/LerianStudio/fetcher/tests/shared/fixtures"
    "github.com/LerianStudio/fetcher/tests/shared/network"
    "github.com/LerianStudio/fetcher/tests/shared/topology"
)

// 1. Create network
net, _ := network.CreateNetwork(ctx)

// 2. Start containers
pgOpts := containers.DefaultPostgresOptions(config.NetworkName)
pgOpts.InitScript, _ = fixtures.GetPostgresInitSQL()
pg, _ := containers.StartPostgres(ctx, pgOpts)

// 3. Use Manager API client
mgr := client.NewManagerClient("http://localhost:4006", fixtures.TestOrganizationID)
conn, _ := mgr.CreateConnection(ctx, client.ConnectionInput{
    ConfigName:   "my-postgres",
    Type:         "POSTGRESQL",
    Host:         pg.Internal.Host,
    Port:         pg.Internal.Port,
    Username:     pg.Internal.Username,
    Password:     pg.Internal.Password,
    DatabaseName: pg.Internal.Database,
})
```

---

## Package Reference

### config

Configuration constants, timeouts, and infrastructure state management.

#### Types

```go
// Port configuration for all services
type PortsConfig struct {
    MongoMain      string  // "27017" - Main MongoDB
    MongoExternal  string  // "27018" - External MongoDB for test data
    RabbitMQ       string  // "5672"  - RabbitMQ AMQP
    SeaweedFSFiler string  // "8888"  - SeaweedFS Filer HTTP
    Redis          string  // "6379"  - Redis/Valkey
    Postgres       string  // "5432"  - PostgreSQL
    MySQL          string  // "3306"  - MySQL
    SQLServer      string  // "1433"  - SQL Server
    Oracle         string  // "1521"  - Oracle
    Manager        string  // "4006"  - Manager API
}

// Database connection info for Docker network hostnames
type InternalDBConnection struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`      // Note: int, not string
    Username string `json:"userName"`  // Note: Username, not User
    Password string `json:"password"`
    Database string `json:"database"`
}

// Infrastructure state persisted to file
type InfraConfig struct {
    NetworkName           string               `json:"networkName"`
    MongoMainURI          string               `json:"mongoMainUri"`
    MongoExternalURI      string               `json:"mongoExternalUri"`
    RabbitMQURI           string               `json:"rabbitmqUri"`
    SeaweedFSURL          string               `json:"seaweedfsUrl"`
    RedisURL              string               `json:"redisUrl"`
    PostgresURL           string               `json:"postgresUrl"`
    MySQLURL              string               `json:"mysqlUrl"`
    SQLServerURL          string               `json:"sqlserverUrl"`
    OracleURL             string               `json:"oracleUrl"`
    Ports                 InfraPorts           `json:"ports"`
    PostgresInternal      InternalDBConnection `json:"postgresInternal"`
    MySQLInternal         InternalDBConnection `json:"mysqlInternal"`
    SQLServerInternal     InternalDBConnection `json:"sqlserverInternal"`
    OracleInternal        InternalDBConnection `json:"oracleInternal"`
    MongoExternalInternal InternalDBConnection `json:"mongoExternalInternal"`
}
```

#### Constants

```go
// Fixed ports for infrastructure reuse and debugging
var FixedPorts = PortsConfig{...}

// Infrastructure config file path
const InfraConfigPath = "/tmp/fetcher-test-infra.json"

// Docker network name
const NetworkName = "fetcher-test-network"

// Timeouts
const (
    SuiteTimeout             = 15 * time.Minute
    ManagerReadyTimeout      = 60 * time.Second
    ManagerReadyPollInterval = 1 * time.Second
    JobCompletionTimeout     = 6000 * time.Second
    JobCompletionTimeoutSlow = 9000 * time.Second
    SeaweedFSFileTimeout     = 30 * time.Second
    HTTPClientTimeout        = 30 * time.Second
    PollingInterval          = 500 * time.Millisecond

    // Container startup timeouts
    RabbitMQStartupTimeout  = 120 * time.Second
    PostgresStartupTimeout  = 60 * time.Second
    MySQLStartupTimeout     = 120 * time.Second
    SQLServerStartupTimeout = 180 * time.Second
    OracleStartupTimeout    = 300 * time.Second
    MongoDBStartupTimeout   = 60 * time.Second
    RedisStartupTimeout     = 30 * time.Second
    SeaweedFSStartupTimeout = 60 * time.Second
    ManagerStartupTimeout   = 120 * time.Second
    WorkerStartupTimeout    = 120 * time.Second
)
```

#### Functions

```go
// Save infrastructure config to file
func (c *InfraConfig) Save(path string) error

// Load infrastructure config from file
func LoadInfraConfig(path string) (*InfraConfig, error)

// Check if infrastructure config exists
func InfraConfigExists() bool

// Remove infrastructure config file
func RemoveInfraConfig() error
```

#### Usage Contract

```go
// Check for existing infrastructure
if config.InfraConfigExists() {
    cfg, _ := config.LoadInfraConfig(config.InfraConfigPath)
    // Reuse existing infrastructure
}

// Save infrastructure state for reuse
cfg := &config.InfraConfig{...}
cfg.Save(config.InfraConfigPath)
```

---

### client

HTTP clients for Manager API, RabbitMQ events, and SeaweedFS file access.

#### ManagerClient

```go
// Create client
func NewManagerClient(baseURL, organizationID string) *ManagerClient

// Connection management
func (c *ManagerClient) CreateConnection(ctx context.Context, input ConnectionInput) (*ConnectionResponse, error)
func (c *ManagerClient) GetConnection(ctx context.Context, connectionID string) (*ConnectionResponse, error)
func (c *ManagerClient) UpdateConnection(ctx context.Context, connectionID string, input ConnectionInput) (*ConnectionResponse, error)
func (c *ManagerClient) DeleteConnection(ctx context.Context, connectionID string) error
func (c *ManagerClient) DeleteConnectionByConfigName(ctx context.Context, configName string) error
func (c *ManagerClient) ListConnections(ctx context.Context) ([]ConnectionResponse, error)
func (c *ManagerClient) TestConnectionEndpoint(ctx context.Context, connectionID string) (*ConnectionTestResponse, error)

// Job management
func (c *ManagerClient) CreateFetcherJob(ctx context.Context, request FetcherRequest) (*FetcherResponse, error)
func (c *ManagerClient) GetJob(ctx context.Context, jobID string) (*JobResponse, error)
func (c *ManagerClient) WaitForJobCompletion(ctx context.Context, jobID string, timeout time.Duration) (*JobResponse, error)

// Schema validation
func (c *ManagerClient) ValidateSchema(ctx context.Context, request SchemaValidationRequest) (*SchemaValidationResponse, error)

// Health
func (c *ManagerClient) HealthCheck(ctx context.Context) error
```

#### Request/Response Types

```go
type ConnectionInput struct {
    ConfigName   string         `json:"configName"`
    Type         string         `json:"type"`         // POSTGRESQL, MYSQL, SQL_SERVER, ORACLE, MONGODB
    Host         string         `json:"host"`
    Port         int            `json:"port"`         // Note: int
    DatabaseName string         `json:"databaseName"`
    Username     string         `json:"userName"`
    Password     string         `json:"password"`
    Metadata     map[string]any `json:"metadata,omitempty"`
}

type FetcherRequest struct {
    DataRequest DataRequest    `json:"dataRequest"`
    Metadata    map[string]any `json:"metadata,omitempty"`
}

type DataRequest struct {
    MappedFields map[string]map[string][]string `json:"mappedFields"`
    Filters      []FilterRequest                `json:"filters,omitempty"`
}

type FilterRequest struct {
    Field    string `json:"field"`
    Operator string `json:"operator"`  // eq, neq, gt, gte, lt, lte, in, nin
    Value    []any  `json:"value"`
}
```

#### RabbitMQEventConsumer

```go
// Create event consumer
func NewRabbitMQEventConsumer(amqpURL string) (*RabbitMQEventConsumer, error)

// Wait for job completion/failure event
func (c *RabbitMQEventConsumer) WaitForJobEvent(ctx context.Context, jobID string, timeout time.Duration) (*JobNotification, error)

// Cleanup (no-op, connections are per-call)
func (c *RabbitMQEventConsumer) Close() error
```

```go
type JobNotification struct {
    JobID           string         `json:"jobId"`
    OrganizationID  string         `json:"organizationId"`
    Status          string         `json:"status"`  // "completed" or "failed"
    Metadata        map[string]any `json:"metadata,omitempty"`
    Result          *JobResultData `json:"result,omitempty"`
    ExecutionTimeMs int64          `json:"executionTimeMs,omitempty"`
    CompletedAt     *time.Time     `json:"completedAt,omitempty"`
}
```

#### SeaweedFSClient

```go
// Create client
func NewSeaweedFSClient(baseURL string) *SeaweedFSClient

// File operations
func (c *SeaweedFSClient) GetFile(ctx context.Context, path string) ([]byte, error)
func (c *SeaweedFSClient) FileExists(ctx context.Context, path string) (bool, error)
func (c *SeaweedFSClient) WaitForFile(ctx context.Context, path string, timeout time.Duration) ([]byte, error)

// Health
func (c *SeaweedFSClient) HealthCheck(ctx context.Context) error
```

#### Usage Contract

```go
// Create clients
mgr := client.NewManagerClient(managerURL, fixtures.TestOrganizationID)
events, _ := client.NewRabbitMQEventConsumer(rabbitmqURI)
seaweed := client.NewSeaweedFSClient(seaweedURL)

// Create connection
conn, _ := mgr.CreateConnection(ctx, client.ConnectionInput{
    ConfigName:   "test-postgres",
    Type:         "POSTGRESQL",
    Host:         "postgres-external",
    Port:         5432,
    DatabaseName: "testdb",
    Username:     "testuser",
    Password:     "testpass",
})

// Create and monitor job
job, _ := mgr.CreateFetcherJob(ctx, model.FetcherRequest{
    DataRequest: model.DataRequest{
        MappedFields: map[string]map[string][]string{
            "test-postgres": {
                "transactions": {"id", "account_id", "amount"},
            },
        },
    },
})

// Wait for completion via RabbitMQ event
notification, _ := events.WaitForJobEvent(ctx, job.JobID, config.JobCompletionTimeout)

// Get result file
if notification.Status == "completed" {
    data, _ := seaweed.WaitForFile(ctx, notification.Result.Path, config.SeaweedFSFileTimeout)
}
```

---

### containers

Docker container orchestration using testcontainers-go.

#### Common Pattern

All containers follow the same pattern:

```go
// Options struct for configuration
type <Service>Options struct {
    NetworkName   string  // Docker network to join
    NetworkAlias  string  // Hostname within network
    FixedHostPort string  // Optional fixed port on host
    // ... service-specific options
}

// Default options function
func Default<Service>Options(networkName string) <Service>Options

// Start function
func Start<Service>(ctx context.Context, opts <Service>Options) (*<Service>Container, error)

// Container wrapper with connection info
type <Service>Container struct {
    Container    *<module>.<Service>Container  // Underlying testcontainer
    URL          string                         // Connection string
    Host         string                         // Host for external access
    Port         string                         // Port for external access
    InternalHost string                         // Docker network hostname
    Internal     config.InternalDBConnection    // For SQL databases
}

// Stop function
func (c *<Service>Container) Stop(ctx context.Context) error
```

#### Available Containers

| Service | Image | Default Alias | Default Credentials |
|---------|-------|---------------|---------------------|
| PostgreSQL | `postgres:16` | `postgres-external` | testuser/testpass, testdb |
| MySQL | `mysql:8` | `mysql-external` | testuser/testpass, testdb |
| SQL Server | `mcr.microsoft.com/mssql/server:2022-latest` | `mssql-external` | sa/TestPass123!, testdb |
| Oracle | `gvenzl/oracle-xe:21-slim-faststart` | `oracle-external` | TESTUSER/TestPass123 |
| MongoDB Main | `mongo:7` | `fetcher-mongodb` | root/password, fetcher_test |
| MongoDB External | `mongo:7` | `fetcher-mongodb-external` | root/password, external_transactions |
| RabbitMQ | `rabbitmq:3-management` | `fetcher-rabbitmq` | guest/guest |
| Redis | `valkey/valkey:8` | `fetcher-valkey` | (no auth) |
| SeaweedFS | `chrislusf/seaweedfs:*` | `fetcher-seaweedfs-*` | (no auth) |

#### SQL Database Functions

```go
// PostgreSQL
func DefaultPostgresOptions(networkName string) PostgresOptions
func StartPostgres(ctx context.Context, opts PostgresOptions) (*PostgresContainer, error)

// MySQL
func DefaultMySQLOptions(networkName string) MySQLOptions
func StartMySQL(ctx context.Context, opts MySQLOptions) (*MySQLContainer, error)

// SQL Server
func DefaultSQLServerOptions(networkName string) SQLServerOptions
func StartSQLServer(ctx context.Context, opts SQLServerOptions) (*SQLServerContainer, error)

// Oracle
func DefaultOracleOptions(networkName string) OracleOptions
func StartOracle(ctx context.Context, opts OracleOptions) (*OracleContainer, error)
```

#### Infrastructure Functions

```go
// MongoDB (two instances available)
func DefaultMongoDBMainOptions(networkName string) MongoDBOptions      // Internal fetcher DB
func DefaultMongoDBExternalOptions(networkName string) MongoDBOptions  // Test data source
func StartMongoDB(ctx context.Context, opts MongoDBOptions) (*MongoDBContainer, error)

// RabbitMQ
func DefaultRabbitMQOptions(networkName string) RabbitMQOptions
func StartRabbitMQ(ctx context.Context, opts RabbitMQOptions) (*RabbitMQContainer, error)

// Redis/Valkey
func DefaultRedisOptions(networkName string) RedisOptions
func StartRedis(ctx context.Context, opts RedisOptions) (*RedisContainer, error)

// SeaweedFS (multi-container: master, volume, filer)
func DefaultSeaweedFSOptions(networkName string) SeaweedFSOptions
func StartSeaweedFS(ctx context.Context, opts SeaweedFSOptions) (*SeaweedFSContainers, error)
```

#### Helper Functions

```go
// Create fixed port binding for debugging
func WithFixedPort(containerPort, hostPort string) testcontainers.CustomizeRequestOption
```

#### Usage Contract

```go
// Start PostgreSQL with init script
pgOpts := containers.DefaultPostgresOptions(config.NetworkName)
pgOpts.FixedHostPort = config.FixedPorts.Postgres
pgOpts.InitScript, _ = fixtures.GetPostgresInitSQL()
pg, _ := containers.StartPostgres(ctx, pgOpts)
defer pg.Stop(ctx)

// Access connection info
fmt.Println(pg.URL)           // Full connection string
fmt.Println(pg.Host)          // Host for external access
fmt.Println(pg.Port)          // Mapped port
fmt.Println(pg.InternalHost)  // Docker network hostname
fmt.Println(pg.Internal.Port) // Internal port (int): 5432
```

#### Direct Datasource Connection

Containers provide all information needed to create datasources directly, without using the Manager API:

```go
// PostgreSQL
pg, _ := containers.StartPostgres(ctx, pgOpts)
conn := &postgres.Connection{
    ConnectionString:   pg.URL,
    DBName:             pg.Internal.Database,
    Logger:             logger,
    MaxOpenConnections: 25,
    MaxIdleConnections: 5,
}
ds, _ := postgres.NewDataSourceRepository(conn)

// MySQL
my, _ := containers.StartMySQL(ctx, myOpts)
conn := &mysql.Connection{
    ConnectionString:   my.URL,
    DBName:             my.Internal.Database,
    Logger:             logger,
    MaxOpenConnections: 25,
    MaxIdleConnections: 5,
}
ds, _ := mysql.NewDataSourceRepository(conn)

// MongoDB
mongo, _ := containers.StartMongoDB(ctx, mongoOpts)
ds, _ := mongodb.NewDataSourceRepository(mongo.URI, mongo.Internal.Database, logger)

// Oracle
ora, _ := containers.StartOracle(ctx, oraOpts)
conn := &oracle.Connection{
    ConnectionString:   ora.URL,
    DBName:             ora.Internal.Database,
    Logger:             logger,
    MaxOpenConnections: 25,
    MaxIdleConnections: 5,
}
ds, _ := oracle.NewDataSourceRepository(conn)

// SQL Server
sql, _ := containers.StartSQLServer(ctx, sqlOpts)
conn := &sqlserver.Connection{
    ConnectionString:   sql.URL,
    DBName:             sql.Internal.Database,
    Logger:             logger,
    MaxOpenConnections: 25,
    MaxIdleConnections: 5,
}
ds, _ := sqlserver.NewDataSourceRepository(conn)
```

---

### network

Docker network creation for container communication.

#### Functions

```go
// Create Docker network "fetcher-test-network"
func CreateNetwork(ctx context.Context) (testcontainers.Network, error)

// Get network name constant
func GetNetworkName() string  // Returns "fetcher-test-network"
```

#### Usage Contract

```go
// Create network before starting containers
net, err := network.CreateNetwork(ctx)
if err != nil {
    return err
}
defer net.Remove(ctx)

// Use config.NetworkName when configuring containers
opts := containers.DefaultPostgresOptions(config.NetworkName)
```

---

### fixtures

Test data and SQL initialization scripts.

#### Constants

```go
// Consistent test account IDs across all databases
var TestAccountIDs = []string{
    "11111111-1111-1111-1111-111111111111",
    "22222222-2222-2222-2222-222222222222",
    "33333333-3333-3333-3333-333333333333",
}

// Organization ID for all test operations
const TestOrganizationID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

// Transaction categories
var Categories = []string{"salary", "groceries", "utilities", "entertainment", "travel"}
```

#### SQL Init Functions

```go
// Get embedded SQL init scripts
func GetPostgresInitSQL() (string, error)
func GetMySQLInitSQL() (string, error)
func GetSQLServerInitSQL() (string, error)
func GetOracleInitSQL() (string, error)
```

#### MongoDB Seeding

```go
// Seed MongoDB with Q4 2024 test transactions
func InitMongoDBExternal(ctx context.Context, connectionString, database string) error
```

#### Expected Record Counts

```go
func ExpectedPostgresRecordCount() int   // 26 (Q1 2024)
func ExpectedMySQLRecordCount() int      // 20 (Q2 2024)
func ExpectedSQLServerRecordCount() int  // 26 (Q3 2024)
func ExpectedOracleRecordCount() int     // 26 (Q4 2023-Q1 2024)
func ExpectedMongoDBRecordCount() int    // 20 (Q4 2024)
```

#### Test Data Schema

All databases contain a `transactions` table with:

| Column | Type | Description |
|--------|------|-------------|
| id | UUID/VARCHAR | Primary key |
| account_id | UUID/VARCHAR | FK to test accounts |
| amount | DECIMAL/FLOAT | Transaction amount |
| currency | VARCHAR | USD, EUR, GBP |
| type | VARCHAR | credit, debit |
| description | VARCHAR | Transaction description |
| category | VARCHAR | salary, groceries, utilities, entertainment, travel |
| status | VARCHAR | completed, pending |
| created_at | TIMESTAMP | Transaction date |
| updated_at | TIMESTAMP | Last update |

#### Usage Contract

```go
// Use init script when starting container
initSQL, _ := fixtures.GetPostgresInitSQL()
pgOpts.InitScript = initSQL

// Seed MongoDB after container starts
mongoConn, _ := containers.StartMongoDB(ctx, mongoOpts)
fixtures.InitMongoDBExternal(ctx, mongoConn.URI, "external_transactions")

// Use test constants in assertions
assert.Equal(t, fixtures.ExpectedPostgresRecordCount(), len(results))
```

---

### topology

RabbitMQ exchange and queue configuration.

#### Functions

```go
// Setup complete RabbitMQ topology for Fetcher
func SetupRabbitMQTopology(ctx context.Context, amqpURL string) error

// Purge test queue (for infrastructure reuse)
func PurgeTestQueue(ctx context.Context, amqpURL string) (int, error)
```

#### Created Resources

**Exchanges:**
- `fetcher.extract-external-data.exchange` (direct) - Job queue exchange
- `fetcher.job.events` (topic) - Job event notifications
- `fetcher.dlx` (direct) - Dead letter exchange

**Queues:**
- `fetcher.extract-external-data.queue` - Main job queue (TTL: 7 days, max: 10k messages)
- `fetcher.dlq` - Dead letter queue
- `test.job.events` - Test event capture queue

**Bindings:**
- `fetcher.extract-external-data.queue` ← `fetcher.extract-external-data.exchange` (key: `fetcher.job.key`)
- `fetcher.dlq` ← `fetcher.dlx` (key: `fetcher.dlq.key`)
- `test.job.events` ← `fetcher.job.events` (keys: `job.completed.*`, `job.failed.*`)

#### Usage Contract

```go
// Setup after RabbitMQ container starts
rabbitmq, _ := containers.StartRabbitMQ(ctx, rabbitmqOpts)
topology.SetupRabbitMQTopology(ctx, rabbitmq.URI)

// Purge before test run (when reusing infrastructure)
purged, _ := topology.PurgeTestQueue(ctx, rabbitmq.URI)
```

---

## Complete Integration Example

```go
func TestIntegration(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), config.SuiteTimeout)
    defer cancel()

    // 1. Create network
    net, err := network.CreateNetwork(ctx)
    require.NoError(t, err)
    defer net.Remove(ctx)

    // 2. Start infrastructure
    mongoMain, _ := containers.StartMongoDB(ctx, containers.DefaultMongoDBMainOptions(config.NetworkName))
    defer mongoMain.Stop(ctx)

    rabbitmq, _ := containers.StartRabbitMQ(ctx, containers.DefaultRabbitMQOptions(config.NetworkName))
    defer rabbitmq.Stop(ctx)

    redis, _ := containers.StartRedis(ctx, containers.DefaultRedisOptions(config.NetworkName))
    defer redis.Stop(ctx)

    seaweed, _ := containers.StartSeaweedFS(ctx, containers.DefaultSeaweedFSOptions(config.NetworkName))
    defer seaweed.Stop(ctx)

    // 3. Setup RabbitMQ topology
    topology.SetupRabbitMQTopology(ctx, rabbitmq.URI)

    // 4. Start external database with test data
    pgOpts := containers.DefaultPostgresOptions(config.NetworkName)
    pgOpts.InitScript, _ = fixtures.GetPostgresInitSQL()
    pg, _ := containers.StartPostgres(ctx, pgOpts)
    defer pg.Stop(ctx)

    // 5. Start application containers (Manager + Worker)
    // ... application startup code ...

    // 6. Create clients
    mgr := client.NewManagerClient(managerURL, fixtures.TestOrganizationID)
    events, _ := client.NewRabbitMQEventConsumer(rabbitmq.URI)
    seaweedClient := client.NewSeaweedFSClient(seaweed.URL)

    // 7. Run test
    conn, _ := mgr.CreateConnection(ctx, client.ConnectionInput{
        ConfigName:   "test-postgres",
        Type:         "POSTGRESQL",
        Host:         pg.Internal.Host,
        Port:         pg.Internal.Port,
        DatabaseName: pg.Internal.Database,
        Username:     pg.Internal.Username,
        Password:     pg.Internal.Password,
    })

    job, _ := mgr.CreateFetcherJob(ctx, model.FetcherRequest{
        DataRequest: model.DataRequest{
            MappedFields: map[string]map[string][]string{
                "test-postgres": {"transactions": {"id", "account_id", "amount"}},
            },
        },
    })

    notification, _ := events.WaitForJobEvent(ctx, job.JobID, config.JobCompletionTimeout)
    assert.Equal(t, "completed", notification.Status)

    data, _ := seaweedClient.WaitForFile(ctx, notification.Result.Path, config.SeaweedFSFileTimeout)
    assert.NotEmpty(t, data)
}
```

---

## Datasource Integration Test Example

Test datasources directly without the full Fetcher stack. This approach is ideal for:
- Testing datasource query logic in isolation
- Faster feedback cycles (no Manager/Worker/RabbitMQ overhead)
- Debugging datasource-specific issues

```go
//go:build integration
// +build integration

package postgres_test

import (
    "context"
    "testing"
    "time"

    libCommons "github.com/LerianStudio/lib-commons/v2/commons"
    "github.com/LerianStudio/fetcher/pkg/postgres"
    "github.com/LerianStudio/fetcher/tests/shared/config"
    "github.com/LerianStudio/fetcher/tests/shared/containers"
    "github.com/LerianStudio/fetcher/tests/shared/fixtures"
    "github.com/LerianStudio/fetcher/tests/shared/network"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPostgresDataSource_Integration(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // 1. Create Docker network
    net, err := network.CreateNetwork(ctx)
    require.NoError(t, err)
    defer net.Remove(ctx)

    // 2. Start PostgreSQL container with test data
    pgOpts := containers.DefaultPostgresOptions(config.NetworkName)
    pgOpts.InitScript, err = fixtures.GetPostgresInitSQL()
    require.NoError(t, err)

    pg, err := containers.StartPostgres(ctx, pgOpts)
    require.NoError(t, err)
    defer pg.Stop(ctx)

    // 3. Create connection from container info
    logger := libCommons.NewLoggerFromContext(ctx)
    conn := &postgres.Connection{
        ConnectionString:   pg.URL,
        DBName:             pg.Internal.Database,
        Logger:             logger,
        MaxOpenConnections: 25,
        MaxIdleConnections: 5,
    }

    // 4. Create datasource (injects connection)
    ds, err := postgres.NewDataSourceRepository(conn)
    require.NoError(t, err)
    defer ds.CloseConnection()

    // 5. Run tests
    t.Run("GetDatabaseSchema returns table structure", func(t *testing.T) {
        schema, err := ds.GetDatabaseSchema(ctx, []string{"public"})
        require.NoError(t, err)
        assert.NotEmpty(t, schema)

        // Find transactions table
        var transactionsSchema *postgres.TableSchema
        for i := range schema {
            if schema[i].TableName == "transactions" {
                transactionsSchema = &schema[i]
                break
            }
        }

        require.NotNil(t, transactionsSchema, "transactions table should exist")
        assert.NotEmpty(t, transactionsSchema.Columns)

        // Verify expected columns exist
        columnNames := make(map[string]bool)
        for _, col := range transactionsSchema.Columns {
            columnNames[col.Name] = true
        }

        assert.True(t, columnNames["id"], "should have id column")
        assert.True(t, columnNames["account_id"], "should have account_id column")
        assert.True(t, columnNames["amount"], "should have amount column")
    })

    t.Run("Query returns all records with wildcard fields", func(t *testing.T) {
        schema, err := ds.GetDatabaseSchema(ctx, []string{"public"})
        require.NoError(t, err)

        results, err := ds.Query(ctx, schema, "transactions", []string{"*"}, nil)
        require.NoError(t, err)

        expectedCount := fixtures.ExpectedPostgresRecordCount()
        assert.Len(t, results, expectedCount)
    })

    t.Run("Query returns specific fields", func(t *testing.T) {
        schema, err := ds.GetDatabaseSchema(ctx, []string{"public"})
        require.NoError(t, err)

        results, err := ds.Query(ctx, schema, "transactions", []string{"id", "amount", "currency"}, nil)
        require.NoError(t, err)
        assert.NotEmpty(t, results)

        // Verify only requested fields are present
        for _, row := range results {
            assert.Contains(t, row, "id")
            assert.Contains(t, row, "amount")
            assert.Contains(t, row, "currency")
        }
    })

    t.Run("Query with equality filter", func(t *testing.T) {
        schema, err := ds.GetDatabaseSchema(ctx, []string{"public"})
        require.NoError(t, err)

        filter := map[string][]any{
            "account_id": {fixtures.TestAccountIDs[0]},
        }

        results, err := ds.Query(ctx, schema, "transactions", []string{"*"}, filter)
        require.NoError(t, err)

        for _, row := range results {
            assert.Equal(t, fixtures.TestAccountIDs[0], row["account_id"])
        }
    })

    t.Run("Query with multiple value filter (IN)", func(t *testing.T) {
        schema, err := ds.GetDatabaseSchema(ctx, []string{"public"})
        require.NoError(t, err)

        filter := map[string][]any{
            "account_id": {fixtures.TestAccountIDs[0], fixtures.TestAccountIDs[1]},
        }

        results, err := ds.Query(ctx, schema, "transactions", []string{"*"}, filter)
        require.NoError(t, err)

        validIDs := map[string]bool{
            fixtures.TestAccountIDs[0]: true,
            fixtures.TestAccountIDs[1]: true,
        }
        for _, row := range results {
            accountID := row["account_id"].(string)
            assert.True(t, validIDs[accountID], "account_id should be in filter list")
        }
    })

    t.Run("Query non-existent table returns error", func(t *testing.T) {
        schema, err := ds.GetDatabaseSchema(ctx, []string{"public"})
        require.NoError(t, err)

        _, err = ds.Query(ctx, schema, "non_existent_table", []string{"*"}, nil)
        assert.Error(t, err)
    })

    t.Run("Query with no matching filter returns empty", func(t *testing.T) {
        schema, err := ds.GetDatabaseSchema(ctx, []string{"public"})
        require.NoError(t, err)

        filter := map[string][]any{
            "account_id": {"00000000-0000-0000-0000-000000000000"},
        }

        results, err := ds.Query(ctx, schema, "transactions", []string{"*"}, filter)
        require.NoError(t, err)
        assert.Empty(t, results)
    })
}
```

**Run with:**
```bash
go test -tags=integration -v ./pkg/postgres/...
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/testcontainers/testcontainers-go` | Container orchestration |
| `github.com/testcontainers/testcontainers-go/modules/*` | Service-specific modules |
| `github.com/rabbitmq/amqp091-go` | RabbitMQ client |
| `go.mongodb.org/mongo-driver` | MongoDB driver |
| `github.com/microsoft/go-mssqldb` | SQL Server driver |
| `github.com/sijms/go-ora/v2` | Oracle driver |

---

## Key Design Decisions

1. **Centralized Timeouts**: All timeout values in `config/timeouts.go` for easy tuning
2. **Fixed Ports Strategy**: Enables VS Code debugging and infrastructure reuse
3. **InfraConfig Persistence**: Allows tests to detect and reuse running infrastructure
4. **Embedded SQL Fixtures**: Init scripts always available in test binary
5. **Options Pattern**: Flexible container configuration with sensible defaults
6. **Dual MongoDB**: Main (internal) and External (test data source) instances
7. **Event-Driven Testing**: RabbitMQ events for async job completion notification
8. **Shared Setup Package**: Orchestrates all containers in parallel, supports reuse
9. **Toxiproxy Integration**: Network chaos injection for resilience testing
10. **Direct Datasource Testing**: Containers expose `.URL` and `.Internal` for creating datasources directly, enabling lightweight integration tests without the full Fetcher stack (Manager, Worker, RabbitMQ, SeaweedFS)

---

## setup

Infrastructure orchestration for starting all containers in parallel with proper configuration.
This package is designed to be used by both integration tests and chaos tests.

### Types

```go
// SharedInfrastructure holds all infrastructure containers and connection info.
type SharedInfrastructure struct {
    Network           testcontainers.Network

    // Core infrastructure
    MongoMain         *containers.MongoDBContainer
    RabbitMQ          *containers.RabbitMQContainer
    SeaweedFS         *containers.SeaweedFSContainers
    Redis             *containers.RedisContainer

    // External databases
    MongoExternal     *containers.MongoDBContainer
    PostgresExternal  *containers.PostgresContainer
    MySQLExternal     *containers.MySQLContainer
    SQLServerExternal *containers.SQLServerContainer
    OracleExternal    *containers.OracleContainer
}

// InfrastructureOptions controls how infrastructure is started.
type InfrastructureOptions struct {
    UseFixedPorts   bool  // Use fixed ports for debugging/reuse
    ReuseExisting   bool  // Connect to existing infrastructure
    SkipExternalDBs bool  // Skip Postgres, MySQL, SQLServer, Oracle
    InitScripts     bool  // Run SQL init scripts
}
```

### Functions

```go
// Option presets
func DefaultOptions() InfrastructureOptions      // All containers, random ports
func DebugOptions() InfrastructureOptions        // All containers, fixed ports
func ReuseOptions() InfrastructureOptions        // Reuse existing, fixed ports
func CoreOnlyOptions() InfrastructureOptions     // Core only, skip external DBs

// Start infrastructure
func Start(ctx context.Context) (*SharedInfrastructure, error)
func StartWithOptions(ctx context.Context, opts InfrastructureOptions) (*SharedInfrastructure, error)

// Infrastructure methods
func (i *SharedInfrastructure) Stop(ctx context.Context) error
func (i *SharedInfrastructure) SetupRabbitMQTopology(ctx context.Context) error
func (i *SharedInfrastructure) PurgeTestQueue(ctx context.Context) (int, error)
func (i *SharedInfrastructure) SaveConfig() error

// Convenience accessors for URIs and URLs
func (i *SharedInfrastructure) RabbitMQURI() string
func (i *SharedInfrastructure) SeaweedFSURL() string
func (i *SharedInfrastructure) RedisURL() string
func (i *SharedInfrastructure) MongoMainURI() string
func (i *SharedInfrastructure) MongoExternalURI() string
func (i *SharedInfrastructure) PostgresURL() string
func (i *SharedInfrastructure) MySQLURL() string
func (i *SharedInfrastructure) SQLServerURL() string
func (i *SharedInfrastructure) OracleURL() string

// Internal connection info for database containers
func (i *SharedInfrastructure) PostgresInternal() config.InternalDBConnection
func (i *SharedInfrastructure) MySQLInternal() config.InternalDBConnection
func (i *SharedInfrastructure) SQLServerInternal() config.InternalDBConnection
func (i *SharedInfrastructure) OracleInternal() config.InternalDBConnection
func (i *SharedInfrastructure) MongoExternalInternal() config.InternalDBConnection
```

### Usage

```go
import "github.com/LerianStudio/fetcher/tests/shared/setup"

// Simple start with defaults
infra, err := chaos.Start(ctx)
defer infra.Stop(ctx)

// Start with options
infra, err := chaos.StartWithOptions(ctx, chaos.DebugOptions())

// Setup RabbitMQ topology after start
infra.SetupRabbitMQTopology(ctx)

// Access URIs/URLs via convenience methods
fmt.Println(infra.RabbitMQURI())
fmt.Println(infra.PostgresURL())
fmt.Println(infra.MongoMainURI())

// Or access containers directly
fmt.Println(infra.RabbitMQ.URI)
fmt.Println(infra.PostgresExternal.URL)
fmt.Println(infra.MongoMain.URI)

// Get internal connection info for database configuration
pg := infra.PostgresInternal()
fmt.Println(pg.Host, pg.Port, pg.Database)
```

### Integration vs Chaos Usage

```go
// Integration tests - use setup directly
func TestIntegration(t *testing.T) {
    infra, _ := chaos.Start(ctx)
    defer infra.Stop(ctx)
    infra.SetupRabbitMQTopology(ctx)
    // ... run tests using infra.* containers
}

// Chaos tests - compose with Toxiproxy
func TestChaos(t *testing.T) {
    infra, _ := chaos.Start(ctx)
    defer infra.Stop(ctx)

    // Add Toxiproxy layer
    toxiproxy, _ := containers.StartToxiproxy(ctx, containers.DefaultToxiproxyOptions(config.NetworkName))
    defer toxiproxy.Stop(ctx)

    // Create proxies to infrastructure
    proxies, _ := toxiproxy.CreateStandardProxies(containers.DefaultStandardUpstreams())

    // Inject chaos
    containers.AddLatency(proxies.RabbitMQ, "high-latency", 500, 100)

    // Run tests through proxies
}
```

---

## Toxiproxy (containers)

Network chaos injection for resilience testing.

### Types

```go
// ToxiproxyContainer wraps a Toxiproxy testcontainer with proxy management.
type ToxiproxyContainer struct {
    Container    testcontainers.Container
    Client       *toxiproxy.Client
    Host         string
    APIPort      string
    InternalHost string
    ProxyPorts   map[string]string
}

// StandardProxies holds proxies for all infrastructure components.
type StandardProxies struct {
    MongoMain     *toxiproxy.Proxy
    MongoExternal *toxiproxy.Proxy
    RabbitMQ      *toxiproxy.Proxy
    SeaweedFS     *toxiproxy.Proxy
    Redis         *toxiproxy.Proxy
    Postgres      *toxiproxy.Proxy
    MySQL         *toxiproxy.Proxy
    SQLServer     *toxiproxy.Proxy
    Oracle        *toxiproxy.Proxy
    Manager       *toxiproxy.Proxy
}

// StandardUpstreams defines upstream addresses for proxies.
type StandardUpstreams struct {
    MongoMain     string  // "fetcher-mongodb:27017"
    MongoExternal string  // "fetcher-mongodb-external:27017"
    RabbitMQ      string  // "fetcher-rabbitmq:5672"
    SeaweedFS     string  // "fetcher-seaweedfs-filer:8888"
    Redis         string  // "fetcher-valkey:6379"
    Postgres      string  // "postgres-external:5432"
    MySQL         string  // "mysql-external:3306"
    SQLServer     string  // "sqlserver-external:1433"
    Oracle        string  // "oracle-external:1521"
    Manager       string  // "manager:4006"
}
```

### Functions

```go
// Start Toxiproxy
func DefaultToxiproxyOptions(networkName string) ToxiproxyOptions
func StartToxiproxy(ctx context.Context, opts ToxiproxyOptions) (*ToxiproxyContainer, error)
func (t *ToxiproxyContainer) Stop(ctx context.Context) error

// Create proxies
func DefaultStandardUpstreams() StandardUpstreams
func (t *ToxiproxyContainer) CreateProxy(cfg ProxyConfig) (*toxiproxy.Proxy, error)
func (t *ToxiproxyContainer) CreateStandardProxies(upstreams StandardUpstreams) (*StandardProxies, error)
func (t *ToxiproxyContainer) GetProxyHostPort(ctx context.Context, containerPort string) (string, error)

// Proxy control
func DisableProxy(proxy *toxiproxy.Proxy) error
func EnableProxy(proxy *toxiproxy.Proxy) error

// Chaos injection (toxics)
func AddLatency(proxy *toxiproxy.Proxy, name string, latencyMS, jitterMS int) (*toxiproxy.Toxic, error)
func AddTimeout(proxy *toxiproxy.Proxy, name string, timeoutMS int) (*toxiproxy.Toxic, error)
func AddBandwidth(proxy *toxiproxy.Proxy, name string, rateBytesPerSec int) (*toxiproxy.Toxic, error)
func AddSlowClose(proxy *toxiproxy.Proxy, name string, delayMS int) (*toxiproxy.Toxic, error)
func AddResetPeer(proxy *toxiproxy.Proxy, name string, timeoutMS int) (*toxiproxy.Toxic, error)
func RemoveToxic(proxy *toxiproxy.Proxy, name string) error
func RemoveAllToxics(proxy *toxiproxy.Proxy) error
```

### Proxy Ports

| Service | Proxy Listen Port | Upstream |
|---------|-------------------|----------|
| MongoDB Main | 27100 | fetcher-mongodb:27017 |
| MongoDB External | 27101 | fetcher-mongodb-external:27017 |
| RabbitMQ | 5673 | fetcher-rabbitmq:5672 |
| SeaweedFS | 8889 | fetcher-seaweedfs-filer:8888 |
| Redis | 6380 | fetcher-valkey:6379 |
| PostgreSQL | 5433 | postgres-external:5432 |
| MySQL | 3307 | mysql-external:3306 |
| SQL Server | 1434 | sqlserver-external:1433 |
| Oracle | 1522 | oracle-external:1521 |
| Manager | 4007 | manager:4006 |

### Usage

```go
import "github.com/LerianStudio/fetcher/tests/shared/containers"

// Start Toxiproxy
toxiproxy, _ := containers.StartToxiproxy(ctx, containers.DefaultToxiproxyOptions(config.NetworkName))
defer toxiproxy.Stop(ctx)

// Create all standard proxies
proxies, _ := toxiproxy.CreateStandardProxies(containers.DefaultStandardUpstreams())

// Add network latency to RabbitMQ
containers.AddLatency(proxies.RabbitMQ, "high-latency", 500, 100)

// Disable PostgreSQL connection entirely
containers.DisableProxy(proxies.Postgres)

// Add bandwidth limit to SeaweedFS
containers.AddBandwidth(proxies.SeaweedFS, "slow-storage", 10240)  // 10KB/s

// Reset connections after timeout
containers.AddResetPeer(proxies.Redis, "connection-reset", 5000)

// Remove toxic after test
containers.RemoveToxic(proxies.RabbitMQ, "high-latency")

// Remove all toxics from proxy
containers.RemoveAllToxics(proxies.SeaweedFS)

// Re-enable disabled proxy
containers.EnableProxy(proxies.Postgres)
```

### Chaos Test Example

```go
func TestRabbitMQNetworkPartition(t *testing.T) {
    ctx := context.Background()

    // Start base infrastructure
    infra, _ := chaos.Start(ctx)
    defer infra.Stop(ctx)
    infra.SetupRabbitMQTopology(ctx)

    // Add chaos layer
    toxiproxy, _ := containers.StartToxiproxy(ctx, containers.DefaultToxiproxyOptions(config.NetworkName))
    defer toxiproxy.Stop(ctx)

    proxies, _ := toxiproxy.CreateStandardProxies(containers.DefaultStandardUpstreams())

    // Configure Manager/Worker to use proxy endpoints instead of direct
    // ... application configuration ...

    // Test 1: Normal operation
    job := createJob(ctx)
    notification := waitForJobEvent(ctx, job.JobID)
    assert.Equal(t, "completed", notification.Status)

    // Test 2: Add latency, verify graceful handling
    containers.AddLatency(proxies.RabbitMQ, "queue-latency", 1000, 200)
    job2 := createJob(ctx)
    notification2 := waitForJobEvent(ctx, job2.JobID)
    assert.Equal(t, "completed", notification2.Status)

    // Test 3: Complete partition, verify recovery
    containers.DisableProxy(proxies.RabbitMQ)
    time.Sleep(5 * time.Second)
    containers.EnableProxy(proxies.RabbitMQ)

    job3 := createJob(ctx)
    notification3 := waitForJobEvent(ctx, job3.JobID)
    assert.Equal(t, "completed", notification3.Status)
}
```

---

## Chaos Testing Framework (tests/shared/chaos)

A project-agnostic chaos testing framework for measuring system resilience under network fault injection. This package provides metrics collection, error classification, SLA validation, and assertion helpers that work with any project using Toxiproxy.

### Architecture

```
tests/shared/chaos/
├── doc.go              # Package documentation
├── interfaces.go       # MetricsProvider, MetricsSnapshot interfaces
├── errors.go           # ErrorClassifier for categorizing failures
├── metrics.go          # ChaosMetrics - core metrics collection
├── thresholds.go       # SLAThresholds - validation thresholds
├── assertions.go       # ChaosAssertions - test assertions
├── config.go           # ChaosInjectionConfig - Toxiproxy configuration
├── injection.go        # InjectChaos, RemoveChaos wrappers
├── constants.go        # Standard chaos values (latency, timeout, bandwidth)
├── extended_metrics.go     # ExtendedMetrics - adds recovery/stability tracking
├── extended_thresholds.go  # ExtendedSLAThresholds - extended validation
└── extended_assertions.go  # ExtendedAssertions - recovery/stability assertions
```

**Design Patterns:**
- **Interface-based**: `MetricsProvider` interface allows custom metrics implementations
- **Composition**: Extended types embed core types for backward compatibility
- **Thread-safe**: All metrics operations protected by mutex
- **Percentile caching**: Efficient P50/P90/P99/P99.9 calculations with cache invalidation

### Core Types

#### ChaosMetrics

Collects request metrics, latency measurements, and error classification during chaos tests.

```go
import "github.com/LerianStudio/fetcher/tests/shared/chaos"

// Create metrics collector
m := chaos.NewChaosMetrics()

// Lifecycle methods
m.StartTest()
m.StartChaos()      // Mark chaos injection start
m.EndChaos()        // Mark chaos injection end
m.EndTest()

// Record requests (thread-safe)
m.RecordRequest(success bool, timeout bool, latency time.Duration)
m.RecordRequestWithError(success bool, timeout bool, latency time.Duration, err error)

// Query metrics
m.SuccessRate()           // Returns percentage (0-100)
m.ThroughputRPS()         // Requests per second over test duration
m.SuccessfulThroughputRPS() // Successful requests per second
m.ChaosThroughputRPS()    // Requests per second during chaos period
m.AverageLatency()        // Mean latency

// Percentile latencies
m.P50()                   // 50th percentile
m.P90()                   // 90th percentile
m.P99()                   // 99th percentile
m.P999()                  // 99.9th percentile
m.Percentile(p float64)   // Any percentile (0-100)

// Duration tracking
m.ChaosDuration()         // Time chaos was active
m.TestDuration()          // Total test time

// Error classification
counts := m.GetErrorCounts()  // map[ErrorCategory]int
```

#### ErrorClassifier

Classifies errors into categories for analysis.

```go
// Error categories
const (
    ErrorCategoryTimeout     // Connection/context timeouts
    ErrorCategoryConnection  // Connection refused, reset, EOF
    ErrorCategoryNetwork     // DNS failures, no route to host
    ErrorCategoryApplication // HTTP errors, business logic
    ErrorCategoryUnknown     // Unclassified errors
)

// Classify an error
category := chaos.ClassifyError(err)

// Or use the classifier in metrics
m.RecordRequestWithError(false, true, latency, err)
breakdown := m.ErrorClassifier.GetErrorsByCategory()
```

#### SLAThresholds

Define success criteria for chaos tests.

```go
// Default thresholds
thresholds := chaos.DefaultSLAThresholds()
// MinSuccessRate: 95.0%
// MaxP99Latency: 5 seconds
// MaxErrorRate: 5.0%
// MinThroughputRPS: 1.0

// Strict production thresholds
thresholds := chaos.StrictSLAThresholds()
// MinSuccessRate: 99.9%
// MaxP99Latency: 1 second
// MaxErrorRate: 0.1%
// MinThroughputRPS: 10.0

// Scenario-specific thresholds
thresholds := chaos.LatencyChaosThresholds()   // Higher latency tolerance
thresholds := chaos.TimeoutChaosThresholds()   // Accepts failures during timeout
thresholds := chaos.BandwidthChaosThresholds() // Accepts 50% failures
```

#### ChaosAssertions

Test assertions for validating metrics against SLA thresholds.

```go
import "github.com/LerianStudio/fetcher/tests/shared/chaos"

func TestChaosResilience(t *testing.T) {
    m := chaos.NewChaosMetrics()
    a := chaos.NewChaosAssertions(t, m)

    // ... run chaos test ...

    // Individual assertions
    a.AssertSuccessRate(95.0)          // Minimum 95% success
    a.AssertLatencyP99(5 * time.Second) // Max P99 latency
    a.AssertErrorRate(5.0)             // Maximum 5% errors
    a.AssertThroughput(10.0)           // Minimum 10 RPS

    // SLA validation (all at once)
    result := a.ValidateAgainstSLA(chaos.DefaultSLAThresholds())
    if !result.Passed {
        t.Logf("SLA violations: %v", result.Violations)
    }

    // Assert SLA is met (fails test if not)
    a.AssertSLAMet(chaos.DefaultSLAThresholds())
}
```

#### ChaosInjectionConfig

Configuration helpers for Toxiproxy toxic injection.

```go
// Pre-built configurations
cfg := chaos.DefaultLatencyConfig()   // 500ms latency, 100ms jitter
cfg := chaos.DefaultTimeoutConfig()   // 10s timeout
cfg := chaos.DefaultBandwidthConfig() // 10KB/s limit
cfg := chaos.DefaultResetPeerConfig() // Connection reset after 5s
cfg := chaos.DefaultSlowCloseConfig() // 3s close delay
cfg := chaos.DefaultLimitDataConfig() // 1KB data limit
cfg := chaos.DefaultSlicerConfig()    // 1KB chunks, 10ms delay

// Custom configuration
cfg := chaos.ChaosInjectionConfig{
    Type:       "latency",
    Name:       "my-latency",
    Attributes: map[string]interface{}{
        "latency": chaos.LatencyMs(1000), // Helper: converts to ms
        "jitter":  chaos.LatencyMs(200),
    },
    Stream:     "downstream",
    Toxicity:   1.0,
}

// Inject into proxy
toxic, err := chaos.InjectChaos(proxy, cfg)

// Remove chaos
chaos.RemoveChaos(proxy, "my-latency")
chaos.RemoveAllChaos(proxy)

// Disable/enable proxy
chaos.DisableProxy(proxy)
chaos.EnableProxy(proxy)
```

### Extended Types (Recovery & Stability)

For tests that need to validate recovery time and post-recovery stability.

#### ExtendedMetrics

Extends ChaosMetrics with recovery and stability tracking.

```go
m := chaos.NewExtendedMetrics()

// Base metrics (inherited from ChaosMetrics)
m.StartTest()
m.RecordRequest(true, false, 100*time.Millisecond)

// Recovery tracking
m.StartRecovery()
// ... wait for system to recover ...
m.EndRecovery()
recoveryTime := m.GetRecoveryTime()

// Stability tracking
m.StartStabilityCheck()
m.RecordStabilityCheck(success bool, successRate float64, latency time.Duration, errMsg string)
m.EndStabilityCheck()

// Stability metrics
m.StabilityDuration()         // How long stability was monitored
m.StabilityPassRate()         // Percentage of checks that passed
m.GetMaxConsecutiveFailures() // Worst failure streak
m.GetStabilityChecks()        // All recorded checks
```

#### ExtendedSLAThresholds

Adds recovery and stability thresholds.

```go
thresholds := chaos.DefaultExtendedSLAThresholds()
// Inherits base SLAThresholds plus:
// RecoverySuccessRate: 99.0%
// StabilityDuration: 10 seconds
// StabilityCheckCount: 5
// MaxConsecutiveFailures: 1
```

#### ExtendedAssertions

Adds recovery and stability assertions.

```go
m := chaos.NewExtendedMetrics()
a := chaos.NewExtendedAssertions(t, m)

// Base assertions (inherited)
a.AssertSuccessRate(95.0)
a.AssertSLAMet(chaos.DefaultExtendedSLAThresholds())

// Recovery assertions
a.AssertRecoveryWithin(30 * time.Second)
a.AssertRecoverySuccessRate(99.0)

// Stability assertions
a.AssertStabilityMaintained(95.0, 2) // 95% pass rate, max 2 consecutive failures
a.AssertStabilityDuration(10 * time.Second)

// Combined recovery + stability
a.AssertRecoveryWithStability(30*time.Second, chaos.DefaultExtendedSLAThresholds())
```

### Constants

Standard chaos values for consistent testing.

```go
// Latency values
chaos.ChaosLatencyValues.Low    // 500ms - noticeable but not disruptive
chaos.ChaosLatencyValues.Medium // 3s - significant delay
chaos.ChaosLatencyValues.High   // 5s - severe latency
chaos.ChaosLatencyValues.Jitter // 500ms - standard jitter

// Timeout values
chaos.ChaosTimeoutValues.Short  // 5s - quick operations
chaos.ChaosTimeoutValues.Medium // 15s - normal operations
chaos.ChaosTimeoutValues.Long   // 30s - slow operations

// Bandwidth limits (bytes per second)
chaos.ChaosBandwidthValues.Low    // 1KB/s - very slow
chaos.ChaosBandwidthValues.Medium // 10KB/s - moderate
chaos.ChaosBandwidthValues.High   // 100KB/s - light throttle

// Success rate thresholds
chaos.ChaosSuccessRateThresholds.DuringChaos        // 50%
chaos.ChaosSuccessRateThresholds.AfterRecovery      // 99%
chaos.ChaosSuccessRateThresholds.DuringLatencyChaos // 90%
chaos.ChaosSuccessRateThresholds.DuringTimeoutChaos // 0%

// Timing constants
chaos.StabilizationDelay        // 2s - wait after injecting chaos
chaos.RecoveryObservationTime   // 5s - wait after removing chaos
chaos.ChaosInjectionDuration    // 10s - default chaos duration
```

### Documentation Helpers

```go
// Document test hypothesis
chaos.DocumentHypothesis(t, "System should maintain 90% success rate under 500ms latency")

// Or format hypothesis
hypothesis := chaos.FormatHypothesis(
    "maintain 90% success rate",
    "latency injection of 500ms",
)
chaos.DocumentHypothesis(t, hypothesis)

// Capture baseline before chaos
baseline := chaos.MeasureSteadyState(m)

// Document test results
chaos.DocumentResult(t, m, "PASS")
// Outputs: success rate, latency percentiles, throughput, error breakdown

// Document error breakdown
chaos.DocumentErrorBreakdown(t, m)
```

### Complete Chaos Test Example

```go
//go:build chaos

package chaos_test

import (
    "context"
    "testing"
    "time"

    "github.com/LerianStudio/fetcher/tests/shared/chaos"
    "github.com/LerianStudio/fetcher/tests/shared/config"
    "github.com/LerianStudio/fetcher/tests/shared/containers"
    "github.com/LerianStudio/fetcher/tests/shared/setup"
    "github.com/stretchr/testify/require"
)

func TestDatabaseLatencyResilience(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // 1. Setup infrastructure with Toxiproxy
    infra, err := chaos.Start(ctx)
    require.NoError(t, err)
    defer infra.Stop(ctx)

    toxiproxy, err := containers.StartToxiproxy(ctx, containers.DefaultToxiproxyOptions(config.NetworkName))
    require.NoError(t, err)
    defer toxiproxy.Stop(ctx)

    proxies, err := toxiproxy.CreateStandardProxies(containers.DefaultStandardUpstreams())
    require.NoError(t, err)

    // 2. Setup metrics and document hypothesis
    metrics := chaos.NewExtendedMetrics()
    chaos.DocumentHypothesis(t, chaos.FormatHypothesis(
        "maintain 90% success rate",
        "PostgreSQL latency injection of 500ms",
    ))

    // 3. Capture steady state baseline
    metrics.StartTest()
    baseline := chaos.MeasureSteadyState(metrics)
    t.Logf("Baseline success rate: %.2f%%", baseline.SuccessRate)

    // 4. Inject chaos
    metrics.StartChaos()
    _, err = chaos.InjectChaos(proxies.Postgres, chaos.DefaultLatencyConfig())
    require.NoError(t, err)
    time.Sleep(chaos.StabilizationDelay)

    // 5. Run operations under chaos
    for i := 0; i < 100; i++ {
        start := time.Now()
        err := performDatabaseOperation(ctx)
        latency := time.Since(start)
        metrics.RecordRequestWithError(err == nil, false, latency, err)
    }

    // 6. Remove chaos and track recovery
    chaos.RemoveAllChaos(proxies.Postgres)
    metrics.EndChaos()
    metrics.StartRecovery()
    time.Sleep(chaos.RecoveryObservationTime)
    metrics.EndRecovery()

    // 7. Monitor stability
    metrics.StartStabilityCheck()
    for i := 0; i < 5; i++ {
        start := time.Now()
        err := performDatabaseOperation(ctx)
        latency := time.Since(start)
        metrics.RecordStabilityCheck(err == nil, metrics.SuccessRate(), latency, "")
        time.Sleep(time.Second)
    }
    metrics.EndStabilityCheck()
    metrics.EndTest()

    // 8. Validate and document results
    assertions := chaos.NewExtendedAssertions(t, metrics)
    thresholds := chaos.LatencyExtendedSLAThresholds()

    assertions.AssertSLAMet(thresholds)
    assertions.AssertRecoveryWithStability(30*time.Second, thresholds)

    chaos.DocumentResultExtended(t, metrics, "PASS")
}
```

### Running Chaos Tests

```bash
# Run all chaos tests
go test -tags=chaos -v ./tests/chaos/...

# Run specific test
go test -tags=chaos -v -run TestDatabaseLatencyResilience ./tests/chaos/e2e/

# Run with short flag (skip long-running tests)
go test -tags=chaos -v -short ./tests/chaos/...
```
