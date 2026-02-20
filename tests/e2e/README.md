# E2E Tests

End-to-end tests for the Fetcher application using the `itestkit` framework. These tests validate the complete flow from API requests through job processing to result storage.

## Table of Contents

- [Quick Start](#quick-start)
- [Environment Variables](#environment-variables)
- [Prerequisites](#prerequisites)
- [Test Architecture](#test-architecture)
- [Project Structure](#project-structure)
- [Important Patterns](#important-patterns)
  - [Product-Based Isolation](#product-based-isolation)
  - [Metadata with Product Code](#metadata-with-product-code)
  - [Job Completion Polling](#job-completion-polling)
  - [Resource Cleanup](#resource-cleanup)
- [Creating New Tests](#creating-new-tests)
- [Infrastructure Components](#infrastructure-components)
- [e2ekit: Application Container Builder](#e2ekit-application-container-builder)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [Test Catalog](#test-catalog)
  - [Product CRUD](#product-crud)
  - [Product Lifecycle](#product-lifecycle)
  - [Connection Management](#connection-management)
  - [Connection Validation](#connection-validation)
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
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FIXED_PORT` | `false` | Use fixed ports for infrastructure (27017, 5672, 6379, 8333) |
| `MANAGER_IMAGE` | `fetcher-manager:latest` | Docker image for Manager |
| `WORKER_IMAGE` | `fetcher-worker:latest` | Docker image for Worker |
| `E2E_SKIP_BUILD` | `true` | Skip Docker build, use pre-built images |
| `GITHUB_TOKEN` | `""` | GitHub token for fetching and worker images |
| `E2E_ENABLE_MYSQL` | `false` | Enable MySQL infrastructure for multi-datasource tests |
| `E2E_ENABLE_ORACLE` | `false` | Enable Oracle infrastructure for Oracle-specific tests |
| `E2E_ENABLE_MSSQL` | `false` | Enable SQL Server infrastructure for MSSQL-specific tests |
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
│  │   Oracle   │                                                  │
│  │   MSSQL    │       ┌───────────┐                             │
│  └────────────┘       │ SeaweedFS │  (Result storage)           │
│   (Source DBs)        └───────────┘                             │
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
├── connection_product_filter_test.go   # Connection filtering by product
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
├── product_create_test.go              # Product creation and validation
├── product_delete_test.go              # Product deletion
├── product_get_test.go                 # Product retrieval
├── product_lifecycle_test.go           # Full CRUD lifecycle with connections
├── product_list_test.go                # Product listing and pagination
├── product_update_test.go              # Product updates (PATCH)
├── schema_validation_test.go           # Schema validation endpoint
└── README.md                           # This file

tests/shared/
├── assertions.go    # AssertJobCompleted, AssertAPIError, AssertValidUUID
├── client.go        # ManagerClient API wrapper
└── helpers.go       # CreateTestProduct, CreateTestProductAndConnection

pkg/itestkit/                 # Test infrastructure framework
├── suite.go                  # Suite builder
├── infra.go                  # Infra interface
├── hostport.go               # Host normalization utilities
├── infra/                    # Infrastructure components
│   ├── mongodb/
│   ├── postgres/
│   ├── rabbitmq/
│   ├── redis/
│   └── seaweedfs/
└── addons/
    ├── e2ekit/               # App container builder
    ├── queuekit/             # Queue consumer/assertions
    └── metricskit/           # Metrics assertions
```

## Important Patterns

### Product-Based Isolation

Every test that creates connections or jobs **must** create its own product for isolation. This prevents interference between parallel tests.

```go
func TestMyFeature(t *testing.T) {
    t.Parallel()

    ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
    defer cancel()

    // Create an isolated product for this test
    product := e2eshared.CreateTestProduct(t, apiClient, ctx)

    // Create connections scoped to this product
    conn, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
        ProductID:  product.ID,
        ConfigName: fmt.Sprintf("e2e-mytest-%s", uuid.New().String()[:8]),
        // ...
    })
}
```

When listing connections, always use `ListConnectionsWithProduct` to scope results:

```go
// Correct: scoped to product
result, err := apiClient.ListConnectionsWithProduct(ctx, product.ID, e2eshared.ListConnectionsParams{})

// Incorrect: returns ALL connections in the organization (no isolation)
result, err := apiClient.ListConnections(ctx, e2eshared.ListConnectionsParams{})
```

### Metadata with Product Code

When creating fetcher jobs, `metadata.source` **must** contain a valid product code. The API validates this field via `validateProductOwnership`.

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
        "source": product.Code,  // Must be a valid product code
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

Always use `t.Cleanup` for resource cleanup. Delete connections before products (due to foreign key constraints).

```go
product := e2eshared.CreateTestProduct(t, apiClient, ctx) // auto-cleanup registered

conn, err := apiClient.CreateConnection(ctx, connInput)
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

    // 1. Create product for isolation
    product := e2eshared.CreateTestProduct(t, apiClient, ctx)

    // 2. Create connection scoped to product
    uniqueName := fmt.Sprintf("e2e-mytest-%s", uuid.New().String()[:8])
    conn, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
        ProductID:    product.ID,
        ConfigName:   uniqueName,
        Type:         e2eshared.DBTypePostgreSQL,
        Host:         pgHost,
        Port:         pgPort,
        DatabaseName: "testdb",
        Username:     "testuser",
        Password:     "testpass",
    })
    require.NoError(t, err)

    t.Cleanup(func() {
        _ = apiClient.DeleteConnection(context.Background(), conn.ID)
    })

    // 3. Submit job with product.Code in metadata
    fetcherReq := model.FetcherRequest{
        DataRequest: model.DataRequest{
            MappedFields: map[string]map[string][]string{
                uniqueName: {
                    "transactions": {"id", "amount"},
                },
            },
        },
        Metadata: map[string]any{
            "source": product.Code,
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
// Create a product
product := e2eshared.CreateTestProduct(t, apiClient, ctx)

// Create a connection
conn, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
    ProductID:    product.ID,
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
        "source": product.Code,
    },
})
require.NoError(t, err)
```

### 5. Raw Responses for Error Testing

Use `*Raw` methods to test error scenarios (status codes, error bodies):

```go
resp, err := apiClient.CreateProductRaw(ctx, invalidInput)
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

app, err := e2ekit.New().
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
app, err := e2ekit.New().
    WithContext(ctx).
    WithDockerfile(e2ekit.DockerfileBuild{
        Context:    e2ekit.ProjectRoot(),
        Dockerfile: "Dockerfile",
        Target:     "production",
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
- MongoDB: 27017
- RabbitMQ: 5672
- Redis: 6379
- SeaweedFS: 8333

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

### "Product Not Found" errors in job creation

Ensure `metadata.source` uses a valid product code (from `CreateTestProduct`), not a hardcoded value.

### Cleanup: Remove orphan containers

```bash
docker ps -a | grep -E "(mongo|rabbit|redis|seaweed|postgres)" | awk '{print $1}' | xargs docker rm -f
```

## Best Practices

1. **Always use `t.Parallel()`** as the first line in every test
2. **Create a product per test** using `CreateTestProduct` for isolation
3. **Use `ListConnectionsWithProduct`** instead of `ListConnections` to scope results
4. **Set `metadata.source` to `product.Code`** when creating fetcher jobs
5. **Use `AssertJobCompleted` for polling** instead of AMQP message consumption
6. **Use `t.Cleanup()`** for resource cleanup to ensure cleanup runs even on failure
7. **Use unique names** with `uuid.New().String()[:8]` suffix to avoid conflicts
8. **Prefer random ports** (default) for CI/CD environments
9. **Use `*Raw` methods** for error scenario tests (to inspect status codes directly)
10. **Keep tests independent** - each test should set up its own state

## Test Catalog

### Product CRUD

| File | Test | Description |
|------|------|-------------|
| `product_create_test.go` | `TestCreateProduct_Success` | Product creation with unique code, name, description |
| `product_create_test.go` | `TestCreateProduct_WithMetadata_Success` | Product creation with metadata preserved |
| `product_create_test.go` | `TestCreateProduct_DuplicateCode_Conflict` | Duplicate code returns 409 |
| `product_create_test.go` | `TestCreateProduct_MissingRequiredFields_BadRequest` | Missing code/name/body returns 400 |
| `product_create_test.go` | `TestCreateProduct_InvalidCode_BadRequest` | Invalid codes (spaces, special chars, length) return 400 |
| `product_get_test.go` | `TestGetProduct_Exists_Success` | Retrieve existing product with all fields |
| `product_get_test.go` | `TestGetProduct_NotFound_404` | Non-existent product returns 404 |
| `product_get_test.go` | `TestGetProduct_InvalidID_BadRequest` | Invalid UUID format returns 400 |
| `product_list_test.go` | `TestListProducts_WithResults_Success` | List returns created products |
| `product_list_test.go` | `TestListProducts_Pagination_Success` | Pagination mechanics (limit, pages) |
| `product_list_test.go` | `TestListProducts_StructureValid_Success` | Response structure (Items, Page, Limit, Total) |
| `product_update_test.go` | `TestUpdateProduct_Name_Success` | Update product name via PATCH |
| `product_update_test.go` | `TestUpdateProduct_Description_Success` | Update product description |
| `product_update_test.go` | `TestUpdateProduct_Metadata_Success` | Update product metadata |
| `product_update_test.go` | `TestUpdateProduct_MultipleFields_Success` | Update multiple fields in single PATCH |
| `product_update_test.go` | `TestUpdateProduct_NotFound_404` | Update non-existent product returns 404 |
| `product_update_test.go` | `TestUpdateProduct_EmptyBody_BadRequest` | Empty update body returns 400 |
| `product_delete_test.go` | `TestDeleteProduct_Success` | Delete product, verify not retrievable |
| `product_delete_test.go` | `TestDeleteProduct_NotFound_404` | Delete non-existent product returns 404 |
| `product_delete_test.go` | `TestDeleteProduct_Idempotent` | First delete succeeds, second returns 404 |
| `product_delete_test.go` | `TestDeleteProduct_WithActiveConnections_Conflict` | Delete product with connections returns 409 |

### Product Lifecycle

| File | Test | Description |
|------|------|-------------|
| `product_lifecycle_test.go` | `TestProductLifecycle_FullCRUD` | Complete lifecycle: create, read, update, connection, job, cleanup |
| `product_lifecycle_test.go` | `TestProductLifecycle_DeleteBlockedByConnections` | Cannot delete product with active connections |

### Connection Management

| File | Test | Description |
|------|------|-------------|
| `connection_create_test.go` | `TestCreateConnection_WithProductID_Success` | Create connection with product ID |
| `connection_create_test.go` | `TestCreateConnection_WithInvalidProductID_BadRequest` | Non-UUID product ID returns 400 |
| `connection_create_test.go` | `TestCreateConnection_WithNonexistentProductID_NotFound` | Non-existent product returns 404 |
| `connection_create_test.go` | `TestCreateConnection_PostgreSQL_Success` | PostgreSQL connection creation |
| `connection_create_test.go` | `TestCreateConnection_MySQL_Success` | MySQL connection creation |
| `connection_create_test.go` | `TestCreateConnection_MongoDB_Success` | MongoDB connection creation |
| `connection_create_test.go` | `TestCreateConnection_Oracle_Success` | Oracle connection creation (skipped if unavailable) |
| `connection_create_test.go` | `TestCreateConnection_SQLServer_Success` | SQL Server connection creation (skipped if unavailable) |
| `connection_create_test.go` | `TestCreateConnection_DuplicateConfigName_Conflict` | Duplicate config name returns 409 |
| `connection_create_test.go` | `TestCreateConnection_ConcurrentDuplicateName` | Concurrent duplicates: one succeeds, one gets 409 |
| `connection_create_test.go` | `TestCreateConnection_MissingRequiredFields_BadRequest` | Missing config_name/type/host returns 400 |
| `connection_get_test.go` | `TestGetConnection_Exists_Success` | Retrieve existing connection with all fields |
| `connection_get_test.go` | `TestGetConnection_NotFound_404` | Non-existent connection returns 404 |
| `connection_get_test.go` | `TestGetConnection_InvalidID_BadRequest` | Invalid UUID format returns 400 |
| `connection_list_test.go` | `TestListConnections_Empty_Success` | Empty result for product with no connections |
| `connection_list_test.go` | `TestListConnections_WithResults_Success` | List returns exact count of created connections |
| `connection_list_test.go` | `TestListConnections_FilterByType_Success` | Product-scoped list with client-side type verification |
| `connection_list_test.go` | `TestListConnections_CombinedFilters_Success` | Multiple types/hosts with client-side filtering |
| `connection_list_test.go` | `TestListConnections_Pagination_Success` | Pagination with product-based isolation (pages of 2) |
| `connection_product_filter_test.go` | `TestListConnections_FilterByProduct_Success` | Filter connections by product ID |
| `connection_product_filter_test.go` | `TestListConnections_FilterByProduct_NoResults` | Product with no connections returns empty |
| `connection_product_filter_test.go` | `TestListConnections_FilterByProduct_InvalidProductID` | Invalid product ID returns 400 |
| `connection_product_filter_test.go` | `TestListConnections_FilterByProduct_NonexistentProduct_EmptyList` | Non-existent product returns empty or 404 |
| `connection_update_test.go` | `TestUpdateConnection_Success` | Update connection fields via PATCH |
| `connection_update_test.go` | `TestUpdateConnection_PartialUpdate_Success` | Update only specific fields |
| `connection_update_test.go` | `TestUpdateConnection_NotFound_404` | Update non-existent connection returns 404 |
| `connection_update_test.go` | `TestUpdateConnection_EmptyUpdate_BadRequest` | Empty update returns 400 |
| `connection_update_test.go` | `TestUpdateConnection_InvalidValues_BadRequest` | Invalid values (empty host, port 0, port too high) |
| `connection_delete_test.go` | `TestDeleteConnection_Success` | Delete existing connection |
| `connection_delete_test.go` | `TestDeleteConnection_NotFound_404` | Delete non-existent connection returns 404 |
| `connection_delete_test.go` | `TestDeleteConnection_Idempotent` | First delete succeeds, second returns 404 |
| `connection_delete_test.go` | `TestDeleteConnection_InvalidID_BadRequest` | Invalid ID format returns 400 |
| `connection_assign_test.go` | `TestAssignConnection_Success` | Assign unassigned connection to product |
| `connection_assign_test.go` | `TestAssignConnection_AlreadyAssigned_Conflict` | Re-assign returns 409 |
| `connection_assign_test.go` | `TestAssignConnection_ProductNotFound_404` | Assign to non-existent product returns 404 |
| `connection_assign_test.go` | `TestAssignConnection_ConnectionNotFound_404` | Assign non-existent connection returns 404 |
| `connection_assign_test.go` | `TestAssignConnection_InvalidProductID_BadRequest` | Invalid product ID returns 400 |
| `connection_assign_test.go` | `TestListUnassignedConnections_Success` | List unassigned connections returns valid structure |
| `connection_assign_test.go` | `TestListUnassignedConnections_Pagination_Success` | Pagination for unassigned connections |
| `connection_assign_test.go` | `TestListUnassignedConnections_AfterAssignment_Removed` | Connection disappears from unassigned list after assignment |
| `connection_schema_test.go` | `TestGetConnectionSchema_PostgreSQL_Success` | Schema introspection returns tables/fields for PostgreSQL |
| `connection_schema_test.go` | `TestGetConnectionSchema_NotFound_404` | Schema for non-existent connection returns 404 |
| `connection_schema_test.go` | `TestGetConnectionSchema_MySQL_Success` | Schema introspection for MySQL |
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
| `extraction_mongodb_test.go` | `TestMongoDBExtraction_WithAggregation_Success` | MongoDB aggregation and metadata |

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
| `job_status_test.go` | `TestGetJob_InvalidID_BadRequest` | Invalid job ID format returns 400 |

### Schema Validation

| File | Test | Description |
|------|------|-------------|
| `schema_validation_test.go` | `TestValidateSchema_ValidTables_Success` | Validation passes for existing tables/fields |
| `schema_validation_test.go` | `TestValidateSchema_InvalidTable_Failure` | Non-existent table returns 422 |
| `schema_validation_test.go` | `TestValidateSchema_InvalidField_Failure` | Non-existent field returns 422 |
| `schema_validation_test.go` | `TestValidateSchema_UnknownDatasource_Error` | Non-existent datasource returns error |
| `schema_validation_test.go` | `TestValidateSchema_EmptyRequest_BadRequest` | Empty request returns 400 |

### Error Scenarios

| File | Test | Description |
|------|------|-------------|
| `error_scenarios_test.go` | `TestExtraction_InvalidConnectionName_Error` | Non-existent connection name fails |
| `error_scenarios_test.go` | `TestExtraction_InvalidTableName_Error` | Non-existent table name fails |
| `error_scenarios_test.go` | `TestExtraction_EmptyMappedFields_BadRequest` | Empty mapped fields returns 400 |
| `error_scenarios_test.go` | `TestExtraction_MissingFields_BadRequest` | Table without fields returns 400 |
| `error_scenarios_test.go` | `TestExtraction_TooManyDatasources_BadRequest` | Exceeding datasource limit returns 400 |
| `error_scenarios_test.go` | `TestExtraction_ConnectionWithWrongCredentials_Rejected` | Wrong credentials fails |
| `error_scenarios_test.go` | `TestCreateJob_InvalidMetadata_BadRequest` | Missing/empty metadata.source returns 400 |

## Related Documentation

- [itestkit Package](../../pkg/itestkit/README.md) - Integration test infrastructure
- [queuekit Addon](../../pkg/itestkit/addons/queuekit/) - Queue consumer and assertions
- [e2ekit Addon](../../pkg/itestkit/addons/e2ekit/) - Application container builder
