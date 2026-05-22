# E2E Tests

End-to-end tests for the Fetcher application using the `itestkit` framework. These tests validate the complete flow from API requests through job processing to result storage.

## Table of Contents

- [Quick Start](#quick-start)
- [Environment Variables](#environment-variables)
- [Prerequisites](#prerequisites)
- [Test Architecture](#test-architecture)
- [Project Structure](#project-structure)
- [Important Patterns](#important-patterns)
  - [Product Name Isolation](#product-name-isolation)
  - [Metadata with Product Name](#metadata-with-product-name)
  - [Job Completion Polling](#job-completion-polling)
  - [Resource Cleanup](#resource-cleanup)
- [Creating New Tests](#creating-new-tests)
- [Infrastructure Components](#infrastructure-components)
- [e2ekit: Application Container Builder](#e2ekit-application-container-builder)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [Test Catalog](#test-catalog)
  - [Connection Management](#connection-management)
  - [Connection Validation](#connection-validation)
  - [Internal Datasource SSL/TLS](#internal-datasource-ssltls)
  - [Data Extraction - PostgreSQL](#data-extraction---postgresql)
  - [Data Extraction - MySQL](#data-extraction---mysql)
  - [Data Extraction - MongoDB](#data-extraction---mongodb)
  - [Data Extraction - Oracle](#data-extraction---oracle)
  - [Data Extraction - SQL Server](#data-extraction---sql-server)
  - [Data Extraction - Multi-Datasource](#data-extraction---multi-datasource)
  - [Data Extraction - Multi-Schema](#data-extraction---multi-schema)
  - [Data Extraction - Filters](#data-extraction---filters)
  - [Data Extraction - Edge Cases](#data-extraction---edge-cases)
  - [Job Management](#job-management)
  - [Schema Validation](#schema-validation)
  - [Error Scenarios](#error-scenarios)

## Quick Start

```bash
# Run all e2e tests
GITHUB_TOKEN=`cat .secrets/github_token.txt` E2E_SKIP_BUILD=false go test -v -tags e2e ./tests/e2e -timeout 10m

# Run a specific test
go test -v -tags e2e ./tests/e2e -run TestPostgresExtraction -timeout 10m

# Run with pre-built images (skip Docker build)
go test -v -tags e2e ./tests/e2e -timeout 10m

# Run with fixed ports (useful for debugging)
FIXED_PORT=true go test -v -tags e2e ./tests/e2e -timeout 10m

# Run with S3 storage (MinIO) instead of SeaweedFS
E2E_ENABLE_S3=true go test -v -tags e2e ./tests/e2e -run TestS3Storage -timeout 10m

# Infrastructure-only mode (start infra, then debug Manager/Worker in IDE)
FIXED_PORT=true E2E_INFRA_ONLY=true go test -v -tags e2e ./tests/e2e -timeout 30m

# Reuse existing infrastructure (after infra-only mode)
E2E_REUSE_INFRA=true E2E_MANAGER_URL=http://localhost:4006 go test -v -tags e2e ./tests/e2e -timeout 10m
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FIXED_PORT` | `false` | Use fixed ports for infrastructure (MongoDB 5709, RabbitMQ 3008, Redis 5707, SeaweedFS 8889) |
| `MANAGER_IMAGE` | `fetcher-manager:latest` | Docker image for Manager |
| `WORKER_IMAGE` | `fetcher-worker:latest` | Docker image for Worker |
| `E2E_SKIP_BUILD` | `true` | Skip Docker build, use pre-built images |
| `GITHUB_TOKEN` | `""` | GitHub token for fetching and worker images |
| `E2E_ENABLE_MYSQL` | `false` | Enable MySQL infrastructure for multi-datasource tests |
| `E2E_ENABLE_ORACLE` | `false` | Enable Oracle infrastructure for Oracle-specific tests |
| `E2E_ENABLE_MSSQL` | `false` | Enable SQL Server infrastructure for MSSQL-specific tests |
| `E2E_ENABLE_MONGODB` | `false` | Enable MongoDB source infrastructure for MongoDB extraction tests |
| `E2E_ENABLE_S3` | `false` | Start MinIO and configure Worker with `STORAGE_PROVIDER=s3` to validate S3 object storage |
| `E2E_INFRA_ONLY` | `false` | Start infrastructure only and block (for debugging Manager/Worker in IDE) |
| `E2E_REUSE_INFRA` | `false` | Skip container creation, connect to already-running infrastructure |
| `E2E_SKIP_MANAGER` | `false` | Skip Manager container, use external Manager at `E2E_MANAGER_URL` |
| `E2E_SKIP_WORKER` | `false` | Skip Worker container (for debugging Worker locally) |
| `E2E_MANAGER_URL` | `""` | URL of external Manager (default: `http://localhost:4006` when `E2E_SKIP_MANAGER=true`) |
| `E2E_DEBUG_LOG` | `false` | Enable debug logging for HTTP requests/responses |

## Prerequisites

- Go 1.21+
- Docker with Docker Compose
- Pre-built Fetcher images (`fetcher-manager:latest`, `fetcher-worker:latest`)

To build the images before running tests:

```bash
make build-manager
make build-worker
```

## Test Architecture

The E2E tests spin up a complete test environment using [testcontainers-go](https://golang.testcontainers.org/):

```
┌─────────────────────────────────────────────────────────────────┐
│                        Test Environment                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐         ┌──────────┐                              │
│  │ Manager  │◄───────►│ MongoDB  │  (Connection storage)        │
│  └──────────┘         └──────────┘                              │
│       │                                                          │
│       │ HTTP API                                                 │
│       ▼                                                          │
│  ┌──────────┐         ┌──────────┐                              │
│  │  Worker  │◄───────►│ RabbitMQ │  (Job queue)                 │
│  └──────────┘         └──────────┘                              │
│       │                                                          │
│       │ Extract                                                  │
│       ▼                                                          │
│  ┌────────────┐       ┌──────────┐                              │
│  │ PostgreSQL │       │  Redis   │  (Cache/Locking)             │
│  │   MySQL    │       └──────────┘                              │
│  │  MongoDB   │                                                  │
│  │   Oracle   │       ┌───────────┐                             │
│  │   MSSQL    │       │ SeaweedFS │  (Result storage, default)  │
│  └────────────┘       └───────────┘                             │
│   (Source DBs)        ┌──────────┐                              │
│                       │  MinIO   │  (S3, E2E_ENABLE_S3=true)    │
│                       └──────────┘                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
tests/e2e/
├── main_test.go                        # TestMain setup/teardown
├── connection_assign_test.go           # Connection assignment to products
├── connection_create_test.go           # Connection creation (all DB types)
├── connection_delete_test.go           # Connection deletion and idempotency
├── connection_get_test.go              # Connection retrieval
├── connection_list_test.go             # Connection listing with product isolation
├── connection_product_filter_test.go   # Connection filtering by product name
├── connection_schema_test.go           # Database schema introspection
├── connection_test_test.go             # Connection health testing
├── connection_update_test.go           # Connection updates (PATCH)
├── connection_validation_test.go       # Input validation boundaries
├── error_scenarios_test.go             # Error handling and edge cases
├── extraction_edge_cases_test.go       # Edge cases (empty results, large data)
├── extraction_filters_test.go          # Filter operators (eq, gt, lt, etc.)
├── extraction_mongodb_test.go          # MongoDB-specific extraction
├── extraction_multi_datasource_test.go # Multi-datasource extraction
├── extraction_multi_schema_test.go     # Multi-schema PostgreSQL extraction
├── extraction_mysql_test.go            # MySQL-specific extraction
├── extraction_oracle_test.go           # Oracle-specific extraction
├── extraction_postgres_test.go         # PostgreSQL extraction
├── extraction_sqlserver_test.go        # SQL Server extraction
├── job_status_test.go                  # Job lifecycle and status tracking
├── schema_validation_test.go           # Schema validation endpoint
└── README.md                           # This file

tests/shared/
├── apps.go               # StartManager, StartWorker, AppEnv, NewClientFromApp
├── assertions.go          # AssertJobCompleted, AssertJobFailed, AssertAPIError, AssertValidUUID
├── client.go              # ManagerClient API wrapper (connections, jobs, schema, assignment)
├── constants.go           # DB types, credentials, timeouts, test account IDs
├── helpers.go             # GenerateProductName, CreateTestConnection
├── infra.go               # CoreInfra (MongoDB, RabbitMQ, Redis, SeaweedFS)
├── testdata/
│   └── definitions.json   # RabbitMQ topology definitions
└── fixtures/
    ├── loader.go           # Fixture file loading utilities
    ├── postgres_init.sql   # PostgreSQL seed data
    ├── mysql_init.sql      # MySQL seed data
    ├── oracle_init.sql     # Oracle seed data
    ├── sqlserver_init.sql  # SQL Server seed data
    └── ssl/
        ├── generate.go     # SSL certificate generation
        ├── generate_test.go
        └── postgres.conf   # PostgreSQL SSL configuration

pkg/itestkit/                     # Test infrastructure framework
├── suite.go                      # Suite builder
├── infra.go                      # Infra interface
├── hostport.go                   # Host normalization utilities
├── chaos.go                      # Chaos testing support
├── chaos_toxiproxy.go            # Toxiproxy integration
├── container_generic.go          # Generic container utilities
├── customizer.go                 # Container customization
├── customizer_options.go         # Customizer options
├── infra/                        # Infrastructure components
│   ├── mongodb/
│   ├── postgres/
│   ├── mysql/
│   ├── mssql/
│   ├── oracle/
│   ├── rabbitmq/
│   ├── redis/
│   └── seaweedfs/
└── addons/
    ├── e2ekit/                   # App container builder
    ├── queuekit/                 # Queue consumer/assertions
    └── metricskit/               # Metrics assertions
```

## Important Patterns

### Product Name Isolation

Connections are associated with products via the `X-Product-Name` header. Product names are simple string labels - there is no separate product entity or API. Every test that creates connections or jobs **must** generate a unique product name for isolation. This prevents interference between parallel tests.

```go
func TestMyFeature(t *testing.T) {
    t.Parallel()

    ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
    defer cancel()

    // Generate a unique product name for this test (no API call needed)
    productName := e2eshared.GenerateProductName()

    // Create connections scoped to this product name
    conn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
        ConfigName:   fmt.Sprintf("e2e-mytest-%s", uuid.New().String()[:8]),
        Type:         e2eshared.DBTypePostgreSQL,
        Host:         pgHost,
        Port:         pgPort,
        DatabaseName: "testdb",
        Username:     "testuser",
        Password:     "testpass",
    })
}
```

When listing connections, always use `ListConnectionsWithProductName` to scope results:

```go
// Correct: scoped to product name
result, err := apiClient.ListConnectionsWithProductName(ctx, productName, e2eshared.ListConnectionsParams{})

// Incorrect: returns ALL connections in the organization (no isolation)
result, err := apiClient.ListConnections(ctx, e2eshared.ListConnectionsParams{})
```

### Metadata with Product Name

When creating fetcher jobs, `metadata.source` **must** contain the product name associated with the connections being queried. The API validates this field via product ownership checks.

```go
fetcherReq := model.FetcherRequest{
    DataRequest: model.DataRequest{
        MappedFields: map[string]map[string][]string{
            uniqueName: {
                "transactions": {"id", "amount", "status"},
            },
        },
    },
    Metadata: map[string]any{
        "source": productName,  // Must match the product name used when creating the connection
        "test":   "my-test-name",
    },
}
```

### Job Completion Polling

Use `AssertJobCompleted` to wait for job completion via API polling. This is more reliable than AMQP message consumption when tests run in parallel.

```go
// Preferred: polling via API
jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)
assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
assert.NotEmpty(t, jobResult.ResultPath)
```

### Resource Cleanup

Always use `t.Cleanup` for resource cleanup. Use `CreateTestConnection` which registers cleanup automatically, or register cleanup manually:

```go
// Option 1: Use CreateTestConnection (auto-cleanup)
conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, connInput)

// Option 2: Manual cleanup
conn, err := apiClient.CreateConnection(ctx, productName, connInput)
require.NoError(t, err)

t.Cleanup(func() {
    _ = apiClient.DeleteConnection(context.Background(), conn.ID)
})
```

## Creating New Tests

### 1. Test File Structure

Create a new file `*_test.go` with the `e2e` build tag:

```go
//go:build e2e

package extraction

import (
    "context"
    "fmt"
    "testing"

    "github.com/LerianStudio/fetcher/pkg/model"
    e2eshared "github.com/LerianStudio/fetcher/tests/shared"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyFeature(t *testing.T) {
    t.Parallel()

    ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
    defer cancel()

    // 1. Generate unique product name for isolation
    productName := e2eshared.GenerateProductName()

    // 2. Create connection scoped to product name
    uniqueName := fmt.Sprintf("e2e-mytest-%s", uuid.New().String()[:8])
    conn := e2eshared.CreateTestConnection(t, apiClient, ctx, productName, e2eshared.ConnectionInput{
        ConfigName:   uniqueName,
        Type:         e2eshared.DBTypePostgreSQL,
        Host:         pgHost,
        Port:         pgPort,
        DatabaseName: "testdb",
        Username:     "testuser",
        Password:     "testpass",
    })

    // 3. Submit job with productName in metadata
    fetcherReq := model.FetcherRequest{
        DataRequest: model.DataRequest{
            MappedFields: map[string]map[string][]string{
                uniqueName: {
                    "transactions": {"id", "amount"},
                },
            },
        },
        Metadata: map[string]any{
            "source": productName,
            "test":   "my-test",
        },
    }
    fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
    require.NoError(t, err)

    // 4. Wait for completion via polling
    jobResult := e2eshared.AssertJobCompleted(t, apiClient, fetcherResp.JobID.String(), e2eshared.DefaultJobTimeout)
    assert.Equal(t, e2eshared.JobStatusCompleted, jobResult.Status)
}
```

> **Important:** All tests must call `t.Parallel()` as the first line to enable parallel execution.

### 2. Using CoreInfra

The `CoreInfra` provides shared infrastructure (MongoDB, RabbitMQ, Redis, SeaweedFS):

```go
// Access from global variables set in main_test.go
mongoURI, _ := coreInfra.MongoDB.URI()
amqpURL, _ := coreInfra.RabbitMQ.AMQPURL()
redisAddr, _ := coreInfra.Redis.Addr()
seaweedURL, _ := coreInfra.SeaweedFS.URL()
```

### 3. Adding Additional Infrastructure

Add a new database (e.g., PostgreSQL) for your test:

```go
func TestWithPostgres(t *testing.T) {
    pgInfra := postgres.NewPostgresInfra(postgres.PostgresConfig{
        Name:     "mytest",
        Database: "testdb",
        Username: "user",
        Password: "pass",
        Options: []postgres.PostgresOption{
            postgres.WithPGInitFile(fixturesPath("init.sql"), "init.sql"),
        },
    })

    err := pgInfra.Start(ctx, nil)
    require.NoError(t, err)
    t.Cleanup(func() { pgInfra.Terminate(ctx) })

    host, port, _ := pgInfra.HostPort()
    dsn, _ := pgInfra.DSN()
}
```

### 4. Using the API Client

The `ManagerClient` wraps HTTP calls to the Manager API:

```go
// Generate product name for isolation
productName := e2eshared.GenerateProductName()

// Create a connection (product name sent via X-Product-Name header)
conn, err := apiClient.CreateConnection(ctx, productName, e2eshared.ConnectionInput{
    ConfigName:   "my-connection",
    Type:         e2eshared.DBTypePostgreSQL,
    Host:         host,
    Port:         port,
    DatabaseName: "testdb",
    Username:     "user",
    Password:     "pass",
})
require.NoError(t, err)
t.Cleanup(func() { apiClient.DeleteConnection(ctx, conn.ID) })

// Create a job
fetcherResp, err := apiClient.CreateFetcherJob(ctx, model.FetcherRequest{
    DataRequest: model.DataRequest{
        MappedFields: map[string]map[string][]string{
            "my-connection": {
                "users": {"id", "name"},
            },
        },
    },
    Metadata: map[string]any{
        "source": productName,
    },
})
require.NoError(t, err)
```

### 5. Raw Responses for Error Testing

Use `*Raw` methods to test error scenarios (status codes, error bodies):

```go
resp, err := apiClient.CreateConnectionRaw(ctx, productName, invalidInput)
require.NoError(t, err)
e2eshared.AssertAPIError(t, resp, 400, "")

resp, err = apiClient.ValidateSchemaRaw(ctx, invalidSchema)
require.NoError(t, err)
assert.Equal(t, 422, resp.StatusCode())
```

## Infrastructure Components

### Available Infra Types

| Component | Package | Default Port | Config Struct |
|-----------|---------|--------------|---------------|
| MongoDB | `infra/mongodb` | 27017 | `MongoDBConfig` |
| PostgreSQL | `infra/postgres` | 5432 | `PostgresConfig` |
| MySQL | `infra/mysql` | 3306 | `MySQLConfig` |
| MSSQL | `infra/mssql` | 1433 | `MSSQLConfig` |
| Oracle | `infra/oracle` | 1521 | `OracleConfig` |
| RabbitMQ | `infra/rabbitmq` | 5672 | `RabbitConfig` |
| Redis | `infra/redis` | 6379 | `RedisConfig` |
| SeaweedFS | `infra/seaweedfs` | 8888 | `SeaweedFSConfig` |

### Common Options

Each infrastructure supports functional options:

```go
mongodb.WithMongoDBImage("mongo:6")
mongodb.WithMongoDBEnv("MONGO_INITDB_ROOT_USERNAME", "admin")
mongodb.WithMongoDBFixedPort("27017")
```

## e2ekit: Application Container Builder

Build and run application containers:

```go
import "github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit"

app, err := e2ekit.New(nil).
    WithContext(ctx).
    WithImage("my-app:latest").
    WithEnv(map[string]string{
        "DATABASE_URL": dsn,
        "LOG_LEVEL":    "debug",
    }).
    ExposePort(8080).
    WithWait(e2ekit.WaitHTTP(8080, "/health", 60*time.Second)).
    Run()

baseURL := app.BaseURL
```

### Wait Strategies

```go
e2ekit.WaitHTTP(port, "/health", timeout)
e2ekit.WaitLog("Server started", timeout)
e2ekit.WaitPort(port, timeout)
e2ekit.WaitRunning(timeout)
```

### Building from Dockerfile

```go
app, err := e2ekit.New(nil).
    WithContext(ctx).
    WithDockerfile(e2ekit.BuildConfig{
        ContextDir: e2ekit.ProjectRoot(),
        Dockerfile: "Dockerfile",
        Tag:        "my-app:latest",
        Secrets: []e2ekit.BuildSecret{
            {ID: "github_token", Env: "GITHUB_TOKEN"},
        },
    }).
    WithEnv(env).
    ExposePort(4006).
    WithWait(e2ekit.WaitHTTP(4006, "/health", 60*time.Second)).
    Run()
```

## Troubleshooting

### Tests fail with "port already in use"

When using `FIXED_PORT=true`, ensure no other services are using the ports:
- MongoDB: 5709
- RabbitMQ: 3008
- Redis: 5707
- SeaweedFS: 8889
- PostgreSQL: 5432

### Container can't connect to host services

The framework automatically rewrites `localhost` to `host.docker.internal` for container environments. If issues persist, check:

```go
host := itestkit.NormalizeHost("localhost")
```

### Tests timeout waiting for events

1. Increase timeout in `WithTimeout()`
2. Check if the worker is processing messages (look at logs)
3. Verify queue names match between publisher and consumer

### Build fails with "GITHUB_TOKEN not set"

Set `E2E_SKIP_BUILD=true` to use pre-built images, or export `GITHUB_TOKEN`:

```bash
export GITHUB_TOKEN=your_token
go test -v -tags e2e ./tests/e2e -timeout 10m
```

### Debugging with infrastructure-only mode

Start infrastructure only, then run Manager/Worker in your IDE:

```bash
# Terminal 1: Start infrastructure
FIXED_PORT=true E2E_INFRA_ONLY=true go test -v -tags e2e ./tests/e2e -timeout 30m

# Terminal 2: Run tests against local Manager/Worker
E2E_REUSE_INFRA=true E2E_MANAGER_URL=http://localhost:4006 go test -v -tags e2e ./tests/e2e -run TestMyTest -timeout 10m
```

### Cleanup: Remove orphan containers

```bash
docker ps -a | grep -E "(mongo|rabbit|redis|seaweed|postgres)" | awk '{print $1}' | xargs docker rm -f
```

## Best Practices

1. **Always use `t.Parallel()`** as the first line in every test
2. **Generate a unique product name per test** using `GenerateProductName` for isolation
3. **Use `ListConnectionsWithProductName`** instead of `ListConnections` to scope results
4. **Set `metadata.source` to the product name** when creating fetcher jobs
5. **Use `AssertJobCompleted` for polling** instead of AMQP message consumption
6. **Use `t.Cleanup()`** for resource cleanup to ensure cleanup runs even on failure
7. **Use unique names** with `uuid.New().String()[:8]` suffix to avoid conflicts
8. **Prefer random ports** (default) for CI/CD environments
9. **Use `*Raw` methods** for error scenario tests (to inspect status codes directly)
10. **Keep tests independent** - each test should set up its own state

## Test Catalog

### Connection Management

| File | Test | Description |
|------|------|-------------|
| `connection_create_test.go` | `TestCreateConnection_WithProductName_Success` | Create connection with valid X-Product-Name header |
| `connection_create_test.go` | `TestCreateConnection_WithoutProductName_BadRequest` | Missing X-Product-Name header returns 400 |
| `connection_create_test.go` | `TestCreateConnection_PostgreSQL_Success` | PostgreSQL connection creation |
| `connection_create_test.go` | `TestCreateConnection_MySQL_Success` | MySQL connection creation (skipped if unavailable) |
| `connection_create_test.go` | `TestCreateConnection_MongoDB_Success` | MongoDB connection creation |
| `connection_create_test.go` | `TestCreateConnection_Oracle_Success` | Oracle connection creation (skipped if unavailable) |
| `connection_create_test.go` | `TestCreateConnection_SQLServer_Success` | SQL Server connection creation (skipped if unavailable) |
| `connection_create_test.go` | `TestCreateConnection_DuplicateConfigName_Conflict` | Duplicate config name returns 409 |
| `connection_create_test.go` | `TestCreateConnection_ConcurrentDuplicateName` | Concurrent duplicates: one succeeds, one gets 409 |
| `connection_create_test.go` | `TestCreateConnection_MissingRequiredFields_BadRequest` | Missing config_name/type/host returns 400 (subtests: `missing_config_name`, `missing_type`, `missing_host`) |
| `connection_create_test.go` | `TestCreateConnection_InvalidProductNameHeader` | Invalid product name header returns 400 (subtests: `whitespace_only`, `special_characters`, `too_long_101_chars`) |
| `connection_create_test.go` | `TestCreateConnection_ProductNameMaxLength_Success` | Product name of exactly 100 characters (maximum) accepted |
| `connection_get_test.go` | `TestGetConnection_Exists_Success` | Retrieve existing connection with all fields and metadata |
| `connection_get_test.go` | `TestGetConnection_NotFound_404` | Non-existent connection returns 404 |
| `connection_get_test.go` | `TestGetConnection_InvalidID_BadRequest` | Invalid UUID format returns 400 (subtests by invalid ID value) |
| `connection_list_test.go` | `TestListConnections_Empty_Success` | Empty result for product with no connections |
| `connection_list_test.go` | `TestListConnections_WithResults_Success` | List returns exact count of created connections |
| `connection_list_test.go` | `TestListConnections_FilterByType_Success` | Product-scoped list with client-side type verification |
| `connection_list_test.go` | `TestListConnections_CombinedFilters_Success` | Multiple types/hosts with client-side filtering |
| `connection_list_test.go` | `TestListConnections_Pagination_Success` | Pagination with product-based isolation (pages of 2) |
| `connection_product_filter_test.go` | `TestListConnections_FilterByProduct_Success` | Filter connections by product name |
| `connection_product_filter_test.go` | `TestListConnections_FilterByProduct_NoResults` | Product with no connections returns empty |
| `connection_product_filter_test.go` | `TestListConnections_FilterByProduct_NonexistentProduct_EmptyList` | Non-existent product name returns empty list |
| `connection_update_test.go` | `TestUpdateConnection_Success` | Update connection fields via PATCH |
| `connection_update_test.go` | `TestUpdateConnection_PartialUpdate_Success` | Update only specific fields |
| `connection_update_test.go` | `TestUpdateConnection_NotFound_404` | Update non-existent connection returns 404 |
| `connection_update_test.go` | `TestUpdateConnection_EmptyUpdate_BadRequest` | Empty update returns 400 |
| `connection_update_test.go` | `TestUpdateConnection_InvalidValues_BadRequest` | Invalid values (subtests: `empty_host`, `port_zero`, `port_too_high`, `empty_database`) |
| `connection_update_test.go` | `TestUpdateConnection_Metadata_Success` | Update connection metadata via PATCH and verify persistence |
| `connection_delete_test.go` | `TestDeleteConnection_Success` | Delete existing connection |
| `connection_delete_test.go` | `TestDeleteConnection_NotFound_404` | Delete non-existent connection returns 404 |
| `connection_delete_test.go` | `TestDeleteConnection_Idempotent` | First delete succeeds, second returns 404 |
| `connection_delete_test.go` | `TestDeleteConnection_InvalidID_BadRequest` | Invalid ID format returns 400 or 404 |
| `connection_assign_test.go` | `TestListUnassignedConnections_Success` | List unassigned connections returns valid paginated structure |
| `connection_assign_test.go` | `TestAssignConnection_AlreadyAssigned_Conflict` | Re-assign to different product returns 409 |
| `connection_assign_test.go` | `TestAssignConnection_ConnectionNotFound_404` | Assign non-existent connection returns 404 |
| `connection_assign_test.go` | `TestAssignConnection_EmptyProductName_BadRequest` | Empty product name string returns 400 |
| `connection_assign_test.go` | `TestAssignConnection_InvalidProductName_BadRequest` | Invalid product names return 400 (subtests: `whitespace_only`, `special_characters`, `too_long`) |
| `connection_schema_test.go` | `TestGetConnectionSchema_PostgreSQL_Success` | Schema introspection returns tables/fields for PostgreSQL |
| `connection_schema_test.go` | `TestGetConnectionSchema_NotFound_404` | Schema for non-existent connection returns 404 |
| `connection_schema_test.go` | `TestGetConnectionSchema_MySQL_Success` | Schema introspection for MySQL (skipped if unavailable) |
| `connection_schema_test.go` | `TestGetConnectionSchema_Oracle_Success` | Schema introspection for Oracle (skipped if unavailable) |
| `connection_schema_test.go` | `TestGetConnectionSchema_SQLServer_Success` | Schema introspection for SQL Server (skipped if unavailable) |
| `connection_schema_test.go` | `TestGetConnectionSchema_MongoDB_Success` | Schema introspection for MongoDB (skipped if unavailable) |
| `connection_test_test.go` | `TestConnectionTest_PostgreSQL_Success` | Test valid PostgreSQL connection returns success |
| `connection_test_test.go` | `TestConnectionTest_MySQL_Success` | Test valid MySQL connection (skipped if unavailable) |
| `connection_test_test.go` | `TestConnectionTest_UnreachableHost_Error` | Unreachable host returns error |
| `connection_test_test.go` | `TestConnectionTest_WrongCredentials_Error` | Wrong credentials returns error |
| `connection_test_test.go` | `TestConnectionTest_NotFound_404` | Test non-existent connection returns 404 |

### Connection Validation

| File | Test | Description |
|------|------|-------------|
| `connection_validation_test.go` | `TestConnection_ConfigName_MinLength` | 3-char config name (minimum) accepted |
| `connection_validation_test.go` | `TestConnection_ConfigName_BelowMin` | 2-char config name rejected |
| `connection_validation_test.go` | `TestConnection_ConfigName_MaxLength` | 100-char config name (maximum) accepted |
| `connection_validation_test.go` | `TestConnection_ConfigName_AboveMax` | 101-char config name rejected |
| `connection_validation_test.go` | `TestConnection_Port_MinBoundary` | Port 1 (minimum) accepted |
| `connection_validation_test.go` | `TestConnection_Port_BelowMin` | Port 0 rejected |
| `connection_validation_test.go` | `TestConnection_Port_MaxBoundary` | Port 65535 (maximum) accepted |
| `connection_validation_test.go` | `TestConnection_Port_AboveMax` | Port 65536 rejected |
| `connection_validation_test.go` | `TestConnection_Port_Negative` | Negative port rejected |
| `connection_validation_test.go` | `TestConnection_MissingPassword` | Missing password rejected |
| `connection_validation_test.go` | `TestConnection_MissingDatabaseName` | Missing database name rejected |
| `connection_validation_test.go` | `TestConnection_MissingUsername` | Missing username rejected |
| `connection_validation_test.go` | `TestConnection_InvalidDatabaseType` | Unsupported database type rejected |
| `connection_validation_test.go` | `TestConnection_MetadataPreserved` | Custom metadata preserved on create/retrieve |

### Internal Datasource SSL/TLS

Coverage for `fetcher-012` (env_loader.go SSL/TLS support). Validates the `DATASOURCE_{NAME}_SSLMODE` env-var path consumed by `pkg/resolver/env_loader.go::LoadInternalConnectionsFromEnv` and exercised through `POST /v1/management/connections/{id}/test` + `GET /v1/management/connections/{id}/schema` against the deterministic UUIDs from `pkg/resolver/registry.go::GetDeterministicID`. Custom CA / client cert plumbing (`_SSL_CA`, `_SSL_CERT`, `_SSL_KEY`) is not yet wired into any driver — those suffixes are parsed but ignored; tracked as a follow-up.

Each test spawns a **dedicated Manager** (via `e2eshared.StartManager` + `AppStartConfig.ExtraEnv`) so per-test `DATASOURCE_*` overrides don't mutate the shared `managerApp`. TLS-required datasources are spun up per test using the existing SSL fixtures in `tests/shared/fixtures/ssl/` (`generate.go` cert bundle, `postgres.conf`, `pg_hba_ssl_only.conf`). Backward-compat tests reuse the shared `postgresInfra` / `coreInfra.MongoDB`.

| File | Test | Description |
|------|------|-------------|
| `internal_datasource_ssl_postgres_test.go` | `TestInternalDatasourceSSL_Postgres_NoSSLMode_AgainstHostssl_Reproduces28000` | **Reproduces LaFinteca bug**: Manager without `DATASOURCE_*_SSLMODE` against a hostssl-only postgres returns HTTP 500 + `"Database Connection Error"` on both `/test` and `/schema`. Pins the pre-fetcher-012 failure mode. |
| `internal_datasource_ssl_postgres_test.go` | `TestInternalDatasourceSSL_Postgres_SSLModeRequire_AgainstHostssl_Success` | **Validates fetcher-012 fix**: Manager with `DATASOURCE_*_SSLMODE=require` against the same hostssl postgres succeeds — `/test` returns 200 + non-zero latency, `/schema` returns a non-nil schema response (TLS handshake negotiated by pgx). |
| `internal_datasource_ssl_postgres_test.go` | `TestInternalDatasourceSSL_Postgres_NoSSL_AgainstPlainPostgres_BackwardCompat` | **Protects existing deployments**: Manager without `_SSLMODE` against a plain postgres (no TLS required) still succeeds. Reuses the shared `postgresInfra` to avoid spinning a fourth postgres container. |
| `internal_datasource_ssl_mongodb_test.go` | `TestInternalDatasourceSSL_Mongo_SSLModeInsecure_AgainstRequireTLS_Success` | Manager with `DATASOURCE_PLUGIN_CRM_SSLMODE=insecure` against a `--tlsMode requireTLS` mongo with self-signed cert succeeds. `insecure` is the mongo-driver vocabulary that triggers `tls=true&tlsInsecure=true` on the URI (verified in `pkg/datasource/datasource_factory.go::appendMongoDBSSLParams`). |
| `internal_datasource_ssl_mongodb_test.go` | `TestInternalDatasourceSSL_Mongo_NoSSLMode_AgainstRequireTLS_Failure` | Manager without `_SSLMODE` against the same TLS-required mongo returns HTTP 500 on both `/test` and `/schema` (driver-level TLS handshake error stays in logs; handler maps to the static `"Database Connection Error"` body). |
| `internal_datasource_ssl_mongodb_test.go` | `TestInternalDatasourceSSL_Mongo_NoSSL_AgainstPlainMongo_BackwardCompat` | Manager without `_SSLMODE` against the shared `coreInfra.MongoDB` (no TLS) still succeeds. Mirrors the postgres backward-compat shape. |

**SSL fixtures used:**
- `tests/shared/fixtures/ssl/generate.go` — CA + server + client cert bundle generator (`ssl.GenerateCertificates(ssl.DefaultGenerateOptions())`). Used by both postgres and mongo TLS containers.
- `tests/shared/fixtures/ssl/postgres.conf` — postgres SSL server config (mounted via `pg.WithConfigFile`).
- `tests/shared/fixtures/ssl/pg_hba_ssl_only.conf` — pg_hba.conf with `hostssl ... scram-sha-256` (rejects plaintext, matches postgres 16 default auth).

**Testcontainer caveats discovered:**
- Postgres pg_hba.conf MUST be mounted with mode `0o644` (not `0o600`): testcontainers mounts files as root:root inside the container, but postmaster runs as UID 70 and aborts with `could not load /etc/postgresql/pg_hba.conf` if the file is unreadable.
- Postgres pg_hba auth method MUST match the bootstrap user's password hash. The `postgres:16-alpine` image defaults to `scram-sha-256` for `password_encryption`; using `md5` in pg_hba causes hostssl auth to fail with `password authentication failed`.
- Mongo 7 enforces [SERVER-72839](https://jira.mongodb.org/browse/SERVER-72839): `--tlsCertificateKeyFile` REQUIRES a paired `--tlsCAFile` for chain-of-trust validation, otherwise mongod aborts at global init.
- Mongo certificate PEM mounts MUST be mode `0o644`: mongod inside the container runs as UID 999 and cannot read a `0o600 root:root` file.
- Mongo expects a SINGLE PEM with cert + key concatenated for `--tlsCertificateKeyFile` (unlike postgres which takes separate files).

**Infra prerequisites:**
- `tests/shared/apps.go::AppStartConfig.ExtraEnv map[string]string` — added in fetcher-012; per-test `DATASOURCE_*` env-var overrides chained AFTER `ManagerEnv()`/`WorkerEnv()` (last-write-wins).
- `pkg/itestkit/infra/postgres/postgres_options.go::WithPGCustomizers(...)` — escape hatch to forward arbitrary `testcontainers.ContainerCustomizer` (e.g. `pg.WithSSLCert`, `pg.WithConfigFile`, `itestkit.CCopyFile`) into postgres container run options.
- `pkg/itestkit/infra/postgres/postgres.go::PostgresConfig.SSLMode` — parameterized DSN sslmode field; empty string defaults to `disable` for backward-compat with existing callers.
- `pkg/itestkit/infra/mongodb/mongodb_options.go::WithMongoDBFile(...)` — mounts a host file into the mongo container; mirrors the postgres `CCopyFile` pattern.

### Data Extraction - PostgreSQL

| File | Test | Description |
|------|------|-------------|
| `extraction_postgres_test.go` | `TestPostgresExtraction_TransactionsTable` | Complete extraction flow from PostgreSQL |

### Data Extraction - MySQL

| File | Test | Description |
|------|------|-------------|
| `extraction_mysql_test.go` | `TestMySQLExtraction_TransactionsTable_Success` | Complete extraction from MySQL |
| `extraction_mysql_test.go` | `TestMySQLExtraction_WithFilters_Success` | MySQL extraction with filter conditions |

### Data Extraction - MongoDB

| File | Test | Description |
|------|------|-------------|
| `extraction_mongodb_test.go` | `TestMongoDBExtraction_Collection_Success` | Complete extraction from MongoDB collection |
| `extraction_mongodb_test.go` | `TestMongoDBExtraction_WithAggregation_Success` | MongoDB connection with aggregation metadata |

### Data Extraction - Oracle

| File | Test | Description |
|------|------|-------------|
| `extraction_oracle_test.go` | `TestOracleExtraction_Table_Success` | Oracle extraction (skipped if unavailable) |
| `extraction_oracle_test.go` | `TestOracleExtraction_MultiSchema_Success` | Oracle multi-schema extraction |

### Data Extraction - SQL Server

| File | Test | Description |
|------|------|-------------|
| `extraction_sqlserver_test.go` | `TestSQLServerExtraction_Table_Success` | SQL Server extraction (skipped if unavailable) |
| `extraction_sqlserver_test.go` | `TestSQLServerExtraction_MultiSchema_Success` | SQL Server multi-schema extraction |
| `extraction_sqlserver_test.go` | `TestSQLServerExtraction_WithDateFilters_Success` | SQL Server with date-based filters |

### Data Extraction - Multi-Datasource

| File | Test | Description |
|------|------|-------------|
| `extraction_multi_datasource_test.go` | `TestMultiDatasourceExtraction` | Extraction from PostgreSQL + MySQL in single job |
| `extraction_multi_datasource_test.go` | `TestMultiDatasourceExtraction_WithFilters` | Multi-datasource with different filters per source |
| `extraction_multi_datasource_test.go` | `TestMultiDatasourceValidation` | Schema validation across multiple datasources |
| `extraction_multi_datasource_test.go` | `TestMultiDatasourceValidation_PartialFailure` | Validation reports errors when one datasource invalid |

### Data Extraction - Multi-Schema

| File | Test | Description |
|------|------|-------------|
| `extraction_multi_schema_test.go` | `TestPostgreSQLMultiSchemaExtraction` | Extraction from public, accounting, reporting schemas |
| `extraction_multi_schema_test.go` | `TestPostgreSQLMultiSchemaWithFilters` | Multi-schema with different filters per table |
| `extraction_multi_schema_test.go` | `TestPostgreSQLMultiSchemaValidation` | Schema validation for schema-qualified table names |
| `extraction_multi_schema_test.go` | `TestPostgreSQLMultiSchemaValidation_InvalidSchema` | Non-existent schema rejected with 422 |

### Data Extraction - Filters

| File | Test | Description |
|------|------|-------------|
| `extraction_filters_test.go` | `TestExtraction_AllFilterOperators/equals_single` | `status = 'completed'` |
| `extraction_filters_test.go` | `TestExtraction_AllFilterOperators/equals_multiple_OR` | `category IN ('salary', 'groceries')` |
| `extraction_filters_test.go` | `TestExtraction_AllFilterOperators/greater_than` | `amount > 1000` |
| `extraction_filters_test.go` | `TestExtraction_AllFilterOperators/greater_or_equal` | `amount >= 1500` |
| `extraction_filters_test.go` | `TestExtraction_AllFilterOperators/less_than` | `amount < 100` |
| `extraction_filters_test.go` | `TestExtraction_AllFilterOperators/less_or_equal` | `amount <= 89.99` |
| `extraction_filters_test.go` | `TestExtraction_SelectiveFilters` | Filters on some tables, all data on others |
| `extraction_filters_test.go` | `TestExtraction_DateRangeFilter` | Date range with `gte` and `lt` operators |
| `extraction_filters_test.go` | `TestExtraction_CombinedFilters` | Multiple conditions AND logic (status + type + amount) |
| `extraction_filters_test.go` | `TestExtraction_AccountIdFilter` | Filter by specific account UUID |

### Data Extraction - Edge Cases

| File | Test | Description |
|------|------|-------------|
| `extraction_edge_cases_test.go` | `TestExtraction_EmptyResultSet` | Job completes with zero matching records |
| `extraction_edge_cases_test.go` | `TestExtraction_EmptyResultSet_MultipleFilters` | Conflicting filters guarantee no results |
| `extraction_edge_cases_test.go` | `TestExtraction_LargeAmountFilter` | Very large numeric filter values |
| `extraction_edge_cases_test.go` | `TestExtraction_SpecialCharactersInFilter` | Special characters in filter values |
| `extraction_edge_cases_test.go` | `TestExtraction_ManyTablesPerDatasource` | Large number of tables per request |
| `extraction_edge_cases_test.go` | `TestExtraction_ManyFieldsPerTable` | Large number of fields per table |
| `extraction_edge_cases_test.go` | `TestExtraction_AllFieldsFromTable` | All commonly used fields from single table |

### Job Management

| File | Test | Description |
|------|------|-------------|
| `job_status_test.go` | `TestGetJob_AfterCreation_Success` | Newly created job has valid structure |
| `job_status_test.go` | `TestGetJob_Completed_Success` | Completed job shows correct status and result path |
| `job_status_test.go` | `TestGetJob_NotFound_404` | Non-existent job returns 404 |
| `job_status_test.go` | `TestCreateJob_DuplicateRequest_ReturnsExisting` | Duplicate request returns existing job (idempotency) |
| `job_status_test.go` | `TestGetJob_InvalidID_BadRequest` | Invalid job ID format returns 400 (subtests by invalid ID value) |

### Schema Validation

| File | Test | Description |
|------|------|-------------|
| `schema_validation_test.go` | `TestValidateSchema_ValidTables_Success` | Validation passes for existing tables/fields |
| `schema_validation_test.go` | `TestValidateSchema_InvalidTable_Failure` | Non-existent table returns 422 |
| `schema_validation_test.go` | `TestValidateSchema_InvalidField_Failure` | Non-existent field returns 422 |
| `schema_validation_test.go` | `TestValidateSchema_UnknownDatasource_Error` | Non-existent datasource returns error |
| `schema_validation_test.go` | `TestValidateSchema_EmptyRequest_BadRequest` | Empty request returns 400 |

### S3 Storage Verification (requires `E2E_ENABLE_S3=true`)

These tests verify that extraction results are correctly persisted to S3-compatible object storage (MinIO). All tests in this section are **skipped** unless `E2E_ENABLE_S3=true`.

| File | Test | Description |
|------|------|-------------|
| `extraction_s3_storage_test.go` | `TestS3Storage_JobCompletesSuccessfully` | Smoke test: full extraction with S3 provider — validates job status "completed" and result_path prefix |
| `extraction_s3_storage_test.go` | `TestS3Storage_ObjectExistsInBucket` | After job completion, verifies the S3 object exists via `HeadObject` |
| `extraction_s3_storage_test.go` | `TestS3Storage_ObjectHasContent` | Verifies the S3 object is non-empty (data is encrypted, content not validated) |
| `extraction_s3_storage_test.go` | `TestS3Storage_ObjectKeyMatchesResultPath` | Confirms the S3 key (`{jobID}.json`) matches the result_path returned by the API |
| `extraction_s3_storage_test.go` | `TestS3Storage_MultipleJobsStoredSeparately` | Two concurrent jobs produce distinct S3 objects with unique keys |

**Run S3 tests:**
```bash
E2E_ENABLE_S3=true go test -v -tags e2e ./tests/e2e -run TestS3Storage -timeout 10m
```

**Debug with infrastructure-only mode + S3:**
```bash
E2E_ENABLE_S3=true FIXED_PORT=true E2E_INFRA_ONLY=true go test -v -tags e2e ./tests/e2e -timeout 30m
```
MinIO S3 API will be available at `http://localhost:9000` (access key: `minioadmin`, secret: `minioadmin`).

### Error Scenarios

| File | Test | Description |
|------|------|-------------|
| `error_scenarios_test.go` | `TestExtraction_InvalidConnectionName_Error` | Non-existent connection name fails |
| `error_scenarios_test.go` | `TestExtraction_InvalidTableName_Error` | Non-existent table name fails |
| `error_scenarios_test.go` | `TestExtraction_EmptyMappedFields_BadRequest` | Empty mapped fields returns 400 |
| `error_scenarios_test.go` | `TestExtraction_MissingFields_BadRequest` | Table without fields returns 400 |
| `error_scenarios_test.go` | `TestExtraction_TooManyDatasources_BadRequest` | Exceeding datasource limit returns 400 |
| `error_scenarios_test.go` | `TestExtraction_ConnectionWithWrongCredentials_Rejected` | Wrong credentials fails |
| `error_scenarios_test.go` | `TestCreateJob_InvalidMetadata_BadRequest` | Missing/empty metadata.source returns 400 (subtests: `missing_metadata`, `missing_source`, `empty_source`) |
| `error_scenarios_test.go` | `TestCreateFetcherJob_ProductMismatch_Rejected` | Product name mismatch between metadata and connection returns 400/409 |
| `error_scenarios_test.go` | `TestCreateFetcherJob_InvalidFilterReferences_BadRequest` | Filters reference datasource not in mappedFields returns 400 |

## Related Documentation

- [itestkit Package](../../pkg/itestkit/README.md) - Integration test infrastructure
- [queuekit Addon](../../pkg/itestkit/addons/queuekit/) - Queue consumer and assertions
- [e2ekit Addon](../../pkg/itestkit/addons/e2ekit/) - Application container builder
