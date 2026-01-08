# Container Integration Tests

End-to-end integration tests for the Fetcher project (Manager + Worker), validating data extraction pipelines across multiple database types (PostgreSQL, MySQL, SQL Server, Oracle, MongoDB).

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Execution Modes](#execution-modes)
  - [Normal Mode](#1-normal-mode-default)
  - [Manager Debug Mode](#2-manager-debug-mode)
  - [Worker Debug Mode](#3-worker-debug-mode)
  - [Full Debug Mode](#4-full-debug-mode)
  - [Debug Integration Test in VS Code](#5-debug-integration-test-in-vs-code)
- [Test Architecture](#test-architecture)
- [Test Catalog](#test-catalog)
- [Configuration Reference](#configuration-reference)
- [Infrastructure Management](#infrastructure-management)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

## Overview

These integration tests verify the complete data extraction flow:

```
Test → Manager API → RabbitMQ → Worker → External DB → SeaweedFS → Event → Test Validation
```

**What's tested:**
- Connection management via Manager API
- Job creation and validation
- Message publishing/consuming via RabbitMQ
- Data extraction from external databases
- File upload to SeaweedFS
- Event-driven job completion notifications

## Prerequisites

### Required Software

| Software | Version | Purpose |
|----------|---------|---------|
| Docker | 20.10+ | Container runtime |
| Go | 1.21+ | Test execution |
| Make | Any | Task automation |

### Host Configuration

> **Required for Debug Modes only.** Skip this if running in Normal mode.

Add these entries to `/etc/hosts` to enable hostname resolution when running Manager or Worker locally:

```bash
# Fetcher integration test hostnames
127.0.0.1       fetcher-mongodb
127.0.0.1       fetcher-rabbitmq
127.0.0.1       fetcher-seaweedfs-master
127.0.0.1       fetcher-seaweedfs-volume
127.0.0.1       fetcher-seaweedfs-filer
127.0.0.1       fetcher-keda
127.0.0.1       fetcher-valkey
127.0.0.1       fetcher-worker
127.0.0.1       fetcher-manager
127.0.0.1       mongodb-external
127.0.0.1       postgres-external
127.0.0.1       mysql-external
127.0.0.1       oracle-external
127.0.0.1       mssql-external
```

**Why this works:** Docker containers resolve these hostnames via Docker DNS, while local processes resolve them via `/etc/hosts` to localhost.

## Quick Start

Run all tests with Manager and Worker as containers:

```bash
# Option 1: Using pre-built images (recommended for local dev)
MANAGER_IMAGE=fetcher-manager:local \
WORKER_IMAGE=fetcher-worker:local \
  make test-integration-container

# Option 2: Build from Dockerfile (CI/CD or fresh build)
# Uses Docker CLI with BuildKit for proper syntax directive support
export GITHUB_TOKEN=your_token
make test-integration-container
```

> **Note:** When `GITHUB_TOKEN` is set, images are built using Docker CLI with BuildKit enabled. This is required because the Dockerfiles use `# syntax=docker/dockerfile:1.4` directive which needs proper BuildKit session support.

## Execution Modes

| Mode | Manager | Worker | Use Case |
|------|---------|--------|----------|
| **Normal** | Container | Container | CI/CD, automated testing |
| **Manager Debug** | Local (IDE) | Container | Debug API logic, job creation |
| **Worker Debug** | Container | Local (IDE) | Debug extraction logic |
| **Full Debug** | Local | Local | Debug end-to-end flow |

### 1. Normal Mode (Default)

Both Manager and Worker run as containers. Ideal for CI/CD pipelines.

```bash
MANAGER_IMAGE=fetcher-manager:local \
WORKER_IMAGE=fetcher-worker:local \
  make test-integration-container
```

### 2. Manager Debug Mode

Debug Manager in VS Code while Worker runs as container.

**Step 1:** Start infrastructure

```bash
make test-integration-infra
```

**Step 2:** Launch Manager in VS Code

Press `F5` and select **"Manager (Test Infra)"**

<details>
<summary>VS Code launch.json configuration</summary>

```json
{
    "name": "Manager (Test Infra)",
    "type": "go",
    "request": "launch",
    "mode": "auto",
    "program": "${workspaceFolder}/components/manager/cmd/app",
    "cwd": "${workspaceFolder}/components/manager",
    "env": {
        "ENV_NAME": "test",
        "LOG_LEVEL": "debug",
        "SERVER_ADDRESS": ":4006",
        "MONGO_URI": "mongodb",
        "MONGO_HOST": "fetcher-mongodb",
        "MONGO_PORT": "27017",
        "MONGO_NAME": "fetcher_test",
        "MONGO_USER": "root",
        "MONGO_PASSWORD": "password",
        "RABBITMQ_URI": "amqp",
        "RABBITMQ_HOST": "fetcher-rabbitmq",
        "RABBITMQ_PORT_AMQP": "5672",
        "RABBITMQ_DEFAULT_USER": "guest",
        "RABBITMQ_DEFAULT_PASS": "guest",
        "SEAWEEDFS_HOST": "fetcher-seaweedfs-filer",
        "SEAWEEDFS_FILER_PORT": "8888",
        "REDIS_HOST": "fetcher-valkey",
        "REDIS_PORT": "6379",
        "REDIS_PASSWORD": "",
        "APP_ENC_KEY": "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=",
        "APP_ENC_KEY_VERSION": "1",
        "ENABLE_TELEMETRY": "false",
        "PLUGIN_AUTH_ENABLED": "false",
        "LICENSE_KEY": "test-license-key"
    }
}
```

</details>

**Step 3:** Run tests

```bash
# Run all tests
make test-integration-debug-manager

# Run specific test
make test-integration-debug-manager TEST=TestSingleDatasourcePostgreSQL
```

### 3. Worker Debug Mode

Debug Worker in VS Code while Manager runs as container.

**Step 1:** Start infrastructure

```bash
make test-integration-infra
```

**Step 2:** Launch Worker in VS Code

Press `F5` and select **"Worker (Test Infra)"**

<details>
<summary>VS Code launch.json configuration</summary>

```json
{
    "name": "Worker (Test Infra)",
    "type": "go",
    "request": "launch",
    "mode": "auto",
    "program": "${workspaceFolder}/components/worker/cmd/app",
    "cwd": "${workspaceFolder}/components/worker",
    "env": {
        "ENV_NAME": "test",
        "LOG_LEVEL": "debug",
        "MONGO_URI": "mongodb",
        "MONGO_HOST": "fetcher-mongodb",
        "MONGO_PORT": "27017",
        "MONGO_NAME": "fetcher_test",
        "MONGO_USER": "root",
        "MONGO_PASSWORD": "password",
        "RABBITMQ_URI": "amqp",
        "RABBITMQ_HOST": "fetcher-rabbitmq",
        "RABBITMQ_PORT_AMQP": "5672",
        "RABBITMQ_DEFAULT_USER": "guest",
        "RABBITMQ_DEFAULT_PASS": "guest",
        "RABBITMQ_FETCHER_WORK_QUEUE": "fetcher.extract-external-data.queue",
        "RABBITMQ_JOB_EVENTS_EXCHANGE": "fetcher.job.events",
        "RABBITMQ_NUMBERS_OF_WORKERS": "2",
        "SEAWEEDFS_HOST": "fetcher-seaweedfs-filer",
        "SEAWEEDFS_FILER_PORT": "8888",
        "SEAWEEDFS_TTL": "",
        "REDIS_HOST": "fetcher-valkey",
        "REDIS_PORT": "6379",
        "REDIS_PASSWORD": "",
        "APP_ENC_KEY": "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=",
        "APP_ENC_KEY_VERSION": "1",
        "CRYPTO_ENCRYPT_SECRET_KEY_SEAWEEDFS": "3132333435363738393031323334353637383930313233343536373839303132",
        "CRYPTO_HASH_SECRET_KEY_SEAWEEDFS": "3132333435363738393031323334353637383930313233343536373839303132",
        "ENABLE_TELEMETRY": "false",
        "LICENSE_KEY": "test-license-key",
        "ORGANIZATION_IDS": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
    }
}
```

</details>

**Step 3:** Run tests

```bash
# Run all tests
make test-integration-debug-worker

# Run specific test
make test-integration-debug-worker TEST=TestSingleDatasourceMySQL
```

### 4. Full Debug Mode

Debug both Manager and Worker simultaneously in VS Code.

**Step 1:** Start infrastructure

```bash
make test-integration-infra
```

**Step 2:** Launch both applications

1. Press `F5` → Select **"Manager (Test Infra)"**
2. Open new VS Code window, press `F5` → Select **"Worker (Test Infra)"**

**Step 3:** Run tests

```bash
# Run all tests
make test-integration-debug-full

# Run specific test
make test-integration-debug-full TEST=TestMultiDatasourceExtraction
```

### Debug Tips

**Useful breakpoints:**

| Component | File | Purpose |
|-----------|------|---------|
| Manager | `components/manager/internal/services/command/create_fetcher_job.go:71` | Job creation entry |
| Manager | `components/manager/internal/adapters/cache/schema_cache.go` | Schema caching |
| Worker | `components/worker/internal/services/extract-data.go` | Data extraction |
| Worker | `components/worker/internal/bootstrap/worker.go` | Initialization |

### 5. Debug Integration Test in VS Code

Debug integration tests directly in VS Code with breakpoints. Three configurations are available:

| Configuration | Manager | Worker | Prerequisites |
|---------------|---------|--------|---------------|
| **Integration Test (Manager Local)** | Local (IDE) | Container | `make test-integration-infra` + F5 "Manager (Test Infra)" |
| **Integration Test (Worker Local)** | Container | Local (IDE) | `make test-integration-infra` + F5 "Worker (Test Infra)" |
| **Integration Test (Full Container)** | Container | Container | None (images prompted at runtime) |

**How to use:**

1. Press `F5` and select the desired configuration
2. VS Code will prompt for the GitHub token:
   - **Enter your token** → Builds fresh images from Dockerfile (captures latest code changes)
   - **Press ENTER (empty)** → Uses pre-built `fetcher-*:local` images (faster startup)
3. A picker will appear to select which test to run
4. Set breakpoints in test code or application code as needed

> **Note:** "Manager Local" and "Worker Local" modes require starting the respective application in debug mode first.

<details>
<summary>VS Code launch.json configurations</summary>

```json
{
    "name": "Integration Test (Manager Local)",
    "type": "go",
    "request": "launch",
    "mode": "test",
    "program": "${workspaceFolder}/tests/integration/containers",
    "buildFlags": "-tags=integration",
    "env": {
        "REUSE_INFRA": "true",
        "EXTERNAL_MANAGER_URL": "http://localhost:4006",
        "GITHUB_TOKEN": "${input:githubToken}",
        "MANAGER_IMAGE": "fetcher-manager:local",
        "WORKER_IMAGE": "fetcher-worker:local"
    },
    "args": ["-test.run", "${input:integrationTestName}", "-test.v", "-test.timeout=10m", "-test.count=1"]
},
{
    "name": "Integration Test (Worker Local)",
    "type": "go",
    "request": "launch",
    "mode": "test",
    "program": "${workspaceFolder}/tests/integration/containers",
    "buildFlags": "-tags=integration",
    "env": {
        "REUSE_INFRA": "true",
        "SKIP_WORKER": "true",
        "GITHUB_TOKEN": "${input:githubToken}",
        "MANAGER_IMAGE": "fetcher-manager:local",
        "WORKER_IMAGE": "fetcher-worker:local"
    },
    "args": ["-test.run", "${input:integrationTestName}", "-test.v", "-test.timeout=10m", "-test.count=1"]
},
{
    "name": "Integration Test (Full Container)",
    "type": "go",
    "request": "launch",
    "mode": "test",
    "program": "${workspaceFolder}/tests/integration/containers",
    "buildFlags": "-tags=integration",
    "env": {
        "GITHUB_TOKEN": "${input:githubToken}",
        "MANAGER_IMAGE": "fetcher-manager:local",
        "WORKER_IMAGE": "fetcher-worker:local"
    },
    "args": ["-test.run", "${input:integrationTestName}", "-test.v", "-test.timeout=10m", "-test.count=1"]
}
```

> **Note:** `-test.count=1` disables Go's test caching, ensuring tests always run (important for debugging).

> **Image source options:**
> - **Enter GitHub token** → Builds fresh images from Dockerfile (slower, but captures latest code)
> - **Press ENTER (empty)** → Uses pre-built `fetcher-*:local` images (faster startup)

Add this `inputs` section to enable test picker and image source prompt:

```json
"inputs": [
    {
        "id": "githubToken",
        "type": "promptString",
        "description": "GitHub token to build fresh images (or ENTER to use fetcher-*:local)",
        "password": true
    },
    {
        "id": "integrationTestName",
        "type": "pickString",
        "description": "Select integration test to run",
        "options": [
            "TestWorkerIntegrationSuite",
            "--- Core Data Extraction (P0) ---",
            "TestWorkerIntegrationSuite/TestSingleDatasourcePostgreSQL",
            "TestWorkerIntegrationSuite/TestSingleDatasourceMySQL",
            "TestWorkerIntegrationSuite/TestSingleDatasourceSQLServer",
            "TestWorkerIntegrationSuite/TestSingleDatasourceOracle",
            "TestWorkerIntegrationSuite/TestSingleDatasourceMongoDB",
            "--- Advanced Data Extraction (P1) ---",
            "TestWorkerIntegrationSuite/TestMultiDatasourceExtraction",
            "TestWorkerIntegrationSuite/TestJobWithFilters",
            "TestWorkerIntegrationSuite/TestJobIdempotency",
            "TestWorkerIntegrationSuite/TestSeaweedFSFileValidation",
            "TestWorkerIntegrationSuite/TestMetadataPassthrough",
            "--- Connection Management (P1) ---",
            "TestWorkerIntegrationSuite/TestConnection_GetByID",
            "TestWorkerIntegrationSuite/TestConnection_GetByID_NotFound",
            "TestWorkerIntegrationSuite/TestConnection_Update",
            "TestWorkerIntegrationSuite/TestConnection_Update_NotFound",
            "TestWorkerIntegrationSuite/TestConnection_Delete",
            "TestWorkerIntegrationSuite/TestConnection_Delete_NotFound",
            "TestWorkerIntegrationSuite/TestConnection_ListWithPagination",
            "TestWorkerIntegrationSuite/TestConnection_ListWithTypeFilter",
            "--- Connection Test & Schema Validation (P1) ---",
            "TestWorkerIntegrationSuite/TestConnection_TestEndpoint",
            "TestWorkerIntegrationSuite/TestConnection_TestEndpoint_NotFound",
            "TestWorkerIntegrationSuite/TestConnection_TestEndpoint_InvalidCredentials",
            "TestWorkerIntegrationSuite/TestValidateSchema_Success",
            "TestWorkerIntegrationSuite/TestValidateSchema_TableNotFound",
            "TestWorkerIntegrationSuite/TestValidateSchema_FieldNotFound",
            "TestWorkerIntegrationSuite/TestValidateSchema_DatasourceNotFound",
            "TestWorkerIntegrationSuite/TestConnection_DeleteWithActiveJob",
            "TestWorkerIntegrationSuite/TestConnection_UpdateWithActiveJob",
            "--- Error Scenarios (P2) ---",
            "TestWorkerIntegrationSuite/TestErrorScenario_ConnectionDown",
            "TestWorkerIntegrationSuite/TestErrorScenario_InvalidCredentials",
            "TestWorkerIntegrationSuite/TestErrorScenario_MissingDatasource",
            "TestWorkerIntegrationSuite/TestErrorScenario_NonExistentTable",
            "--- Validation & Edge Cases (P2) ---",
            "TestWorkerIntegrationSuite/TestConnection_DuplicateConfigName",
            "TestWorkerIntegrationSuite/TestJob_InvalidInput",
            "TestWorkerIntegrationSuite/TestJob_AllFilterOperators",
            "TestWorkerIntegrationSuite/TestConnection_Metadata"
        ],
        "default": "TestWorkerIntegrationSuite/TestSingleDatasourcePostgreSQL"
    }
]
```

</details>

## Test Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                       fetcher-test-network                          │
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ MongoDB  │  │ RabbitMQ │  │SeaweedFS │  │  Valkey  │            │
│  │  :27017  │  │  :5672   │  │  :8888   │  │  :6379   │            │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘            │
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │PostgreSQL│  │  MySQL   │  │SQL Server│  │  Oracle  │            │
│  │  :5432   │  │  :3306   │  │  :1433   │  │  :1521   │            │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘            │
│                                                                     │
│  ┌───────────────────────┐  ┌───────────────────────┐              │
│  │   fetcher-manager     │  │    fetcher-worker     │              │
│  │        :4006          │  │                       │              │
│  └───────────────────────┘  └───────────────────────┘              │
└─────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
1. Test ──POST /v1/management/connections──► Manager (create connection)
2. Test ──POST /v1/fetcher────────────────► Manager (create job)
3. Manager ──validate & publish───────────► RabbitMQ
4. Worker ◄──consume message──────────────► RabbitMQ
5. Worker ──extract data──────────────────► External DB
6. Worker ──upload file───────────────────► SeaweedFS
7. Worker ──publish completion────────────► RabbitMQ (fetcher.job.events)
8. Test ◄──receive event──────────────────► RabbitMQ (test.job.events queue)
```

## Test Catalog

### Core Data Extraction Tests (P0)

| Test | Database | Description |
|------|----------|-------------|
| `TestSingleDatasourcePostgreSQL` | PostgreSQL | Complete extraction pipeline validation |
| `TestSingleDatasourceMySQL` | MySQL | MySQL-specific connection and query execution |
| `TestSingleDatasourceSQLServer` | SQL Server | T-SQL extraction with Windows SQL Server |
| `TestSingleDatasourceOracle` | Oracle | Oracle DB with service name metadata |
| `TestSingleDatasourceMongoDB` | MongoDB | NoSQL document extraction with field projection |

### Multi-Schema Data Extraction Tests (P0)

| Test | Expected Rows | Description |
|------|---------------|-------------|
| `TestPostgreSQLMultiSchemaExtraction` | 48 | Extract from multiple PostgreSQL schemas (public, accounting, reporting) |
| `TestPostgreSQLMultiSchemaWithFilters` | 29 | Filtered extraction across multiple PostgreSQL schemas |
| `TestMultiDatasourceMultiSchemaExtraction` | 82 | Complex extraction: 3 datasources × 2 schemas each with filters |
| `TestValidateSchema_MultiSchema` | N/A | Schema validation with schema-qualified table names |

### Advanced Data Extraction Tests (P1)

| Test | Description |
|------|-------------|
| `TestMultiDatasourceExtraction` | Multi-source extraction (PostgreSQL + MySQL) in single job |
| `TestJobWithFilters` | Filter conditions (eq, status='completed') |
| `TestJobWithSelectiveFilters` | Filter conditions (eq, status='completed') |
| `TestJobIdempotency` | Request deduplication behavior (same hash → same job) |
| `TestSeaweedFSFileValidation` | Output file exists and has content in SeaweedFS |
| `TestMetadataPassthrough` | Metadata preservation through extraction pipeline |

### Connection Management Tests (P1)

| Test | Description |
|------|-------------|
| `TestConnection_GetByID` | Retrieve connection by UUID |
| `TestConnection_GetByID_NotFound` | 404 for non-existent connection |
| `TestConnection_Update` | Update connection fields via PATCH |
| `TestConnection_Update_NotFound` | 404 when updating non-existent connection |
| `TestConnection_Delete` | Delete connection and verify 404 on GET |
| `TestConnection_Delete_NotFound` | 404 when deleting non-existent connection |
| `TestConnection_ListWithPagination` | Paginated connection listing (limit, page) |
| `TestConnection_ListWithTypeFilter` | Filter connections by database type |

### Connection Test & Schema Validation Tests (P1)

| Test | Description |
|------|-------------|
| `TestConnection_TestEndpoint` | Test connection endpoint with valid credentials |
| `TestConnection_TestEndpoint_NotFound` | 404 when testing non-existent connection |
| `TestConnection_TestEndpoint_InvalidCredentials` | Error response for bad credentials |
| `TestValidateSchema_Success` | Schema validation with existing table/fields |
| `TestValidateSchema_TableNotFound` | TABLE_NOT_FOUND error for non-existent table |
| `TestValidateSchema_FieldNotFound` | FIELD_NOT_FOUND error for non-existent column |
| `TestValidateSchema_DatasourceNotFound` | HTTP 400 for unknown datasource |
| `TestConnection_DeleteWithActiveJob` | 409 Conflict when deleting with active job |
| `TestConnection_UpdateWithActiveJob` | 409 Conflict when updating with active job |

### Error Scenario Tests (P2)

| Test | Description |
|------|-------------|
| `TestErrorScenario_ConnectionDown` | Graceful handling of unavailable datasource |
| `TestErrorScenario_InvalidCredentials` | Authentication failure handling |
| `TestErrorScenario_MissingDatasource` | Missing datasource in job request |
| `TestErrorScenario_NonExistentTable` | Invalid table reference handling |

### Validation & Edge Case Tests (P2)

| Test | Description |
|------|-------------|
| `TestConnection_DuplicateConfigName` | 409 Conflict for duplicate configName |
| `TestJob_InvalidInput` | 400 Bad Request for empty mappedFields |
| `TestJob_AllFilterOperators` | All filter operators (eq, ne, gt, gte, lt, lte, in, nin) |
| `TestConnection_Metadata` | Custom metadata persistence on connection creation |

### SSL Connection Tests (P2)

These tests validate SSL/TLS connections to databases. Run with `ENABLE_SSL=true`.

| Test | Description |
|------|-------------|
| `TestSSLConnectionValidation` | Validates SSL connections to PostgreSQL, MySQL, MongoDB, and SQL Server |
| `TestMultiDatasourceSSLConnections` | Multi-datasource extraction using SSL connections |

**Running SSL tests:**

```bash
# Run all tests including SSL
ENABLE_SSL=true \
MANAGER_IMAGE=fetcher-manager:local \
WORKER_IMAGE=fetcher-worker:local \
  make test-integration-container

# Run only SSL tests
ENABLE_SSL=true \
MANAGER_IMAGE=fetcher-manager:local \
WORKER_IMAGE=fetcher-worker:local \
TEST=TestSSLConnectionValidation \
  make test-integration-container
```

> **Note:** SSL tests are skipped when `ENABLE_SSL` is not set or set to `false`. The SSL infrastructure starts additional database containers on separate ports (PostgreSQL:5433, MySQL:3307, etc.) with self-signed certificates.

## Configuration Reference

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `GITHUB_TOKEN` | GitHub token for building images from Dockerfile (uses Docker CLI with BuildKit) | CI/CD mode |
| `MANAGER_IMAGE` | Docker image for Manager (used when `GITHUB_TOKEN` is not set) | Local dev |
| `WORKER_IMAGE` | Docker image for Worker (used when `GITHUB_TOKEN` is not set) | Local dev |
| `EXTERNAL_MANAGER_URL` | URL of external Manager | Debug modes |
| `SKIP_WORKER` | Skip Worker container (`true`/`false`) | No |
| `REUSE_INFRA` | Reuse existing infrastructure (`true`/`false`) | No |
| `USE_FIXED_PORTS` | Use fixed port mapping (`true`/`false`) | No |
| `TEST` | Run specific test by name | No |
| `TEST_ENCRYPTION_KEY_BASE64` | Base64 encryption key for Manager | No |
| `TEST_ENCRYPTION_KEY_HEX` | Hex encryption key for Worker | No |
| `ENABLE_SSL` | Enable SSL database containers and tests (`true`/`false`) | No |

**Image resolution priority:**
1. If `GITHUB_TOKEN` is set → Build image from Dockerfile using Docker CLI with BuildKit
2. If `MANAGER_IMAGE`/`WORKER_IMAGE` is set → Use pre-built image
3. Otherwise → Error (one of the above is required)

### Fixed Ports (Debug Mode)

| Service | Port | Hostname |
|---------|------|----------|
| MongoDB (main) | 27017 | `fetcher-mongodb` |
| MongoDB (external) | 27018 | `mongodb-external` |
| RabbitMQ | 5672 | `fetcher-rabbitmq` |
| SeaweedFS Filer | 8888 | `fetcher-seaweedfs-filer` |
| Valkey/Redis | 6379 | `fetcher-valkey` |
| PostgreSQL | 5432 | `postgres-external` |
| MySQL | 3306 | `mysql-external` |
| SQL Server | 1433 | `mssql-external` |
| Oracle | 1521 | `oracle-external` |
| Manager API | 4006 | `localhost` |

### SSL Ports (ENABLE_SSL=true)

| Service | Port | Hostname |
|---------|------|----------|
| PostgreSQL SSL | 5433 | `postgres-external-ssl` |
| MySQL SSL | 3307 | `mysql-external-ssl` |
| SQL Server SSL | 1434 | `sqlserver-external-ssl` |
| Oracle SSL | 1522 | `oracle-external-ssl` |
| MongoDB SSL | 27019 | `fetcher-mongodb-external-ssl` |

## Infrastructure Management

### Infrastructure Reuse

`REUSE_INFRA=true` keeps containers running between test runs, reducing startup time from 3-5 minutes to seconds.

| Scenario | Setting | Reason |
|----------|---------|--------|
| Local development | `REUSE_INFRA=true` | Fast iteration |
| CI/CD pipelines | `REUSE_INFRA=false` | Clean state |
| Debugging flaky tests | `REUSE_INFRA=false` | Eliminate state pollution |

**Automatic cleanup with `REUSE_INFRA=true`:**

1. **RabbitMQ queue purge** - Removes stale events from previous runs (`SetupSuite`)
2. **MongoDB cleanup** - Deletes test connections before each test (`SetupTest`)
3. **Unique request hashes** - Each test uses timestamped metadata

### Manual Control

```bash
# Start infrastructure (keeps running)
make test-integration-infra

# Check port availability
make test-integration-check

# Stop and clean everything
make test-integration-clean
```

### Cleanup

```bash
# Recommended: automated cleanup
make test-integration-clean

# Manual cleanup
docker stop $(docker ps -q --filter "name=fetcher")
docker network rm fetcher-test-network
docker container prune -f
rm -f /tmp/fetcher-test-infra.json
```

## Troubleshooting

### Quick Fix

Most issues resolve with:

```bash
make test-integration-clean
make test-integration-check
```

### Common Errors

<details>
<summary><strong>"network with name fetcher-test-network already exists"</strong></summary>

```bash
make test-integration-clean
# Or manually:
docker network rm fetcher-test-network
```

</details>

<details>
<summary><strong>"address already in use" (port conflict)</strong></summary>

```bash
# Check which ports have conflicts
make test-integration-check

# Find what's using the port
lsof -i :27017

# Clean up leftover containers
make test-integration-clean
```

</details>

<details>
<summary><strong>"infrastructure appears down, cannot connect to RabbitMQ"</strong></summary>

The config file exists but containers are not running:

```bash
rm /tmp/fetcher-test-infra.json
make test-integration-infra
```

</details>

<details>
<summary><strong>"timeout waiting for job event"</strong></summary>

**Possible causes:**
1. Stale events in queue (fixed by automatic purge)
2. Worker not processing (check logs)
3. Connection test failing

**Debug:**
```bash
# Check RabbitMQ UI: http://localhost:15672 (guest/guest)
# Check Worker logs in VS Code debug console
```

</details>

<details>
<summary><strong>"expected pending, got completed" (deduplication issue)</strong></summary>

1. Restart Manager after code changes to `ComputeRequestHash()`
2. Verify tests use unique metadata (not sharing request objects)

</details>

<details>
<summary><strong>"dial tcp: lookup postgres-external: no such host"</strong></summary>

Missing `/etc/hosts` entries. See [Host Configuration](#host-configuration).

</details>

### Slow Tests

Tests take 60-70 seconds due to container startup (Oracle ~3min, SQL Server ~2min).

**Speed up iteration:**

```bash
# Keep infrastructure running
make test-integration-infra

# Run specific test with reuse
REUSE_INFRA=true \
EXTERNAL_MANAGER_URL=http://localhost:4006 \
  go test -tags=integration -v \
  -run "TestSingleDatasourcePostgreSQL" \
  ./tests/integration/containers/...
```

## Development

### Adding New Tests

1. **Create test function** in `integration_test.go`:

   ```go
   func (s *WorkerIntegrationTestSuite) TestYourNewTest() {
       t := s.T()

       // Use testMetadata for unique request hash
       metadata := s.testMetadata("TestYourNewTest")

       // Test implementation...
   }
   ```

2. **Register cleanup** if using new connection names:

   Add connection name to `SetupTest()` cleanup list.

3. **Update documentation**:

   Add test to the [Test Catalog](#test-catalog) section.

### Project Structure

```
tests/integration/containers/
├── cmd/                 # start-infra command for persistent infrastructure
├── setup/               # Application containers (Manager, Worker)
├── testdata/            # Test data files
├── integration_test.go  # Test suite
└── README.md            # This file

tests/shared/            # Shared test infrastructure library
├── config/              # Configuration, ports, timeouts
├── client/              # API clients (Manager, RabbitMQ, SeaweedFS)
├── containers/          # Docker container orchestration
├── network/             # Docker network management
├── fixtures/            # Database init scripts (embedded SQL)
├── topology/            # RabbitMQ exchange/queue setup
└── README.md            # Detailed API documentation
```

> **See [`tests/shared/README.md`](../../shared/README.md)** for comprehensive API documentation of the shared infrastructure library.
