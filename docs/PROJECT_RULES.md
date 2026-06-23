# Architecture Documentation - Fetcher

## Overview

Fetcher is a **data extraction platform** built with Go following **Hexagonal Architecture** and **CQRS (Command Query Responsibility Segregation)** patterns. It extracts data from multiple external databases, encrypts the results, and stores them in a distributed file system. It ships in two forms: the standalone **Manager + Worker** services, and an **embedded runtime engine** (`pkg/engine`) that other Lerian products (Matcher, Reporter) import in-process. The Manager and Worker themselves run *over* the engine — the engine owns the extraction rules, the services own the operational shell.

### Technology Stack

| Category | Technology | Version |
|----------|------------|---------|
| **Language** | Go | Source of truth: `go.mod` |
| **Web Framework** | Fiber | v2.52.10 |
| **Message Queue** | RabbitMQ | v1.10.0 |
| **Event Streaming** | lib-streaming (Kafka/Redpanda) | v1.3.1 — mandatory for Worker job events |
| **Primary Database** | MongoDB | Latest |
| **File Storage** | SeaweedFS (default) / S3-compatible | SeaweedFS 3.97 / AWS SDK v2 |
| **Observability** | OpenTelemetry | v1.39.0 |
| **Auth** | lib-auth | v2.8.0 |
| **API Documentation** | Swagger/Swaggo | v1.16.6 |

### Supported External Databases

- MongoDB
- PostgreSQL
- MySQL
- Oracle
- SQL Server

---

## Project Structure

```
fetcher/
├── components/                    # Independent service components
│   ├── infra/                     # Infrastructure services (Docker Compose)
│   ├── manager/                   # HTTP API server
│   └── worker/                    # Async job processor
├── pkg/                           # Shared packages
│   ├── engine/                    # Embedded runtime core (infra-free extraction rules + ports)
│   │   └── memory/                # In-memory port impls (TEST/EMBEDDED harness only)
│   ├── enginecompat/              # Adapters bridging engine ports to Fetcher infrastructure
│   │   ├── connectioncompat/      # Mongo connection repo + job repo → engine ports
│   │   ├── schemacompat/          # Datasource factory + crypto + Redis → engine ports
│   │   ├── datasource/            # Worker extraction ConnectorFactory
│   │   ├── plugincrm/             # CRM detection helpers
│   │   └── tablenorm/             # Table/field key normalization
│   ├── model/                     # Domain models (entities, DTOs)
│   │   └── datasource/            # Per-DB config models (postgres, mysql, oracle, sqlserver, mongodb)
│   ├── ports/                     # Repository/port interfaces (connection, job, cache, publisher, storage, messaging, datasource)
│   ├── resolver/                  # Internal datasource env loading + multi-tenant resolution
│   ├── mongodb/                   # MongoDB repositories (connection, job)
│   ├── postgres/                  # PostgreSQL adapter
│   ├── mysql/                     # MySQL adapter
│   ├── oracle/                    # Oracle adapter
│   ├── sqlserver/                 # SQL Server adapter
│   ├── rabbitmq/                  # RabbitMQ adapter (incl. security_envelope.go)
│   ├── seaweedfs/                 # SeaweedFS client
│   ├── storage/                   # Storage factory (SeaweedFS / S3 abstraction)
│   ├── crypto/                    # Encryption service
│   ├── datasource/                # DataSource factory + SSL mode utils
│   │   ├── hostsafety/            # SSRF host-safety guard (adapter over libSSRF)
│   │   └── sslmode/               # SSL mode validation/injection
│   ├── schemautil/                # Schema operation utilities
│   ├── redis/                     # Redis/Valkey client adapter
│   ├── ratelimit/                 # Rate limiter (Redis-backed)
│   ├── metrics/                   # Multi-tenant OTel metrics (no-op when disabled)
│   ├── multitenant/               # Multi-tenant cross-cutting test suite
│   ├── bootstrap/readyz/          # Canonical /readyz readiness probe wiring + SaaS TLS enforcement
│   ├── startup/                   # Bootstrap error sanitization (redacts credentials from stderr)
│   ├── net/http/                  # HTTP utilities (incl. with_body.go safe_host validator)
│   ├── itestkit/                  # Integration/E2E test kit (testcontainers, chaos, queue/metrics addons)
│   ├── testutil/                  # Unit-test helpers (context, mocks)
│   ├── shell/                     # Makefile color/template/ASCII assets (not Go)
│   └── constant/                  # Application constants
├── tests/                         # Black-box test suites (build-tagged)
│   ├── shared/                    # Shared E2E/chaos helpers (package e2eshared) + fixtures, testdata
│   ├── e2e/                       # End-to-end tests (//go:build e2e)
│   ├── chaos/                     # Chaos tests with Toxiproxy (//go:build chaos)
│   └── fuzz/                      # Fuzz tests
├── scripts/                       # Build and validation scripts
├── .github/                       # CI/CD workflows
└── .githooks/                     # Git hooks for code quality
```

---

## Components

### 1. Infra Component

**Location:** `components/infra/`

**Purpose:** Infrastructure provisioning via Docker Compose. Not a Go application - purely configuration.

**Services Provided:**

| Service | Image | Purpose | Ports |
|---------|-------|---------|-------|
| `fetcher-mongodb` | `mongo:latest` | Primary data store | Variable |
| `fetcher-rabbitmq` | `rabbitmq:4.0-management-alpine` | Message broker | AMQP + Management UI |
| `fetcher-seaweedfs-master` | `chrislusf/seaweedfs:3.97` | Distributed file storage (master) | 9335, 9336 |
| `fetcher-seaweedfs-volume` | `chrislusf/seaweedfs:3.97` | Distributed file storage (volume) | 9081 |
| `fetcher-seaweedfs-filer` | `chrislusf/seaweedfs:3.97` | Distributed file storage (filer) | 8889, 8334 |
| `fetcher-keda` | `ghcr.io/kedacore/keda:2.16.0` | Kubernetes event-driven autoscaler | 8000 |

**RabbitMQ Topology:**

| Type | Name | Purpose |
|------|------|---------|
| **Queue** | `fetcher.extract-external-data.queue` | Main job processing (with DLQ support) |
| **Queue** | `fetcher.dlq` | Dead letter queue (7-day TTL, max 10,000 messages) |
| **Exchange** | `fetcher.extract-external-data.exchange` | Direct exchange for job routing |
| **Exchange** | `fetcher.dlx` | Dead letter exchange |
| **Exchange** | `fetcher.job.events` | Topic exchange for job notifications |

---

### 2. Manager Component

**Location:** `components/manager/`

**Purpose:** HTTP API server that handles connection management and job orchestration.

**Responsibilities:**
- Manage database connection configurations
- Create and track data extraction jobs
- Validate connections before job execution
- Publish jobs to RabbitMQ for async processing

**Entry Point:** `components/manager/cmd/app/main.go`

#### Internal Structure (Hexagonal Architecture)

```
components/manager/
├── cmd/app/
│   └── main.go                     # Application entry point
├── api/
│   └── docs.go                     # Swagger documentation
└── internal/
    ├── adapters/
    │   └── http/in/
    │       ├── routes.go           # HTTP route definitions (Primary Adapter). Accepts optional tenantMiddleware for multi-tenant DB resolution.
    │       ├── middlewares.go      # HTTP middleware
    │       ├── connection.go       # Connection HTTP handlers
    │       ├── fetcher.go          # Fetcher job HTTP handlers
    │       ├── migration.go        # Migration HTTP handlers (assign/unassigned)
    │       └── swagger.go          # Swagger configuration
    ├── adapters/
    │   └── cache/
    │       ├── schema_cache_interface.go  # SchemaCacheRepository interface
    │       └── schema_cache.go           # Redis-backed schema cache implementation
    ├── bootstrap/
    │   ├── config.go               # Dependency injection + license/SSRF/TLS gating
    │   ├── server.go               # HTTP server wrapper (drain-aware)
    │   ├── service.go              # Application service wrapper
    │   ├── connection_engine.go    # Wires the connection engine (engine.New + connectioncompat)
    │   ├── schema_engine.go        # Wires the schema engine (engine.New + schemacompat)
    │   └── readyz_adapters.go      # /readyz dependency checker wiring
    └── services/
        ├── command/                # CQRS Commands (Write operations)
        │   ├── create_connection.go
        │   ├── update_connection.go
        │   ├── delete_connection.go
        │   ├── create_fetcher_job.go
        │   ├── assign_connection.go
        │   ├── connection_engine_ops.go    # Update/Delete delegated to the connection engine
        │   ├── active_jobs_gate.go          # Active-execution conflict gate
        │   └── extraction_request_mapper.go # Maps FetcherRequest → engine.ExtractionRequest
        └── query/                  # CQRS Queries (Read operations)
            ├── get_connection.go
            ├── list_connections.go
            ├── test_connection.go               # NOT delegated to the engine (direct via factory)
            ├── get_job.go
            ├── get_connection_schema.go          # Always-fresh discovery via schema engine
            ├── validate_schema.go
            ├── connection_engine_access.go       # Tenant-scope reads via the connection engine
            ├── schema_engine_access.go           # Schema validation via the schema engine
            └── list_unassigned_connections.go
```

#### API Endpoints

##### Connections (Tag: `Connections`)

| Method | Endpoint | Handler | Description |
|--------|----------|---------|-------------|
| `POST` | `/v1/management/connections` | `CreateConnection` | Create new database connection (requires `X-Product-Name` header, encrypted password) |
| `GET` | `/v1/management/connections` | `ListConnections` | List connections with pagination/filters (optional `X-Product-Name` header to filter by product) |
| `POST` | `/v1/management/connections/validate-schema` | `ValidateSchema` | Validate tables/fields against datasources (200 OK / 422 on failure) |
| `GET` | `/v1/management/connections/{id}` | `GetConnection` | Get connection details by ID |
| `POST` | `/v1/management/connections/{id}/test` | `TestConnection` | Test connection (rate-limited: 10/min) |
| `GET` | `/v1/management/connections/{id}/schema` | `GetConnectionSchema` | Retrieve database schema for a connection |
| `PATCH` | `/v1/management/connections/{id}` | `UpdateConnection` | Partial update (409 if active jobs) |
| `DELETE` | `/v1/management/connections/{id}` | `DeleteConnection` | Soft delete (409 if active jobs) |

##### Migration (Tag: `Migration`)

| Method | Endpoint | Handler | Description |
|--------|----------|---------|-------------|
| `GET` | `/v1/management/connections/unassigned` | `ListUnassignedConnections` | List connections not assigned to any product |
| `POST` | `/v1/management/connections/{id}/assign` | `AssignConnectionToProduct` | Assign connection to a product (one-time, irreversible, requires `X-Product-Name` header) |

##### Fetcher Jobs (Tag: `Fetcher`)

| Method | Endpoint | Handler | Description |
|--------|----------|---------|-------------|
| `POST` | `/v1/fetcher` | `CreateJob` | Create data extraction job (202 Accepted / 200 if duplicate) |
| `GET` | `/v1/fetcher/{id}` | `GetJob` | Get job status and details |

##### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/version` | Version info |
| `GET` | `/swagger/*` | Swagger UI |

---

### 3. Worker Component

**Location:** `components/worker/`

**Purpose:** Asynchronous job processor that extracts data from external databases.

**Responsibilities:**
- Consume jobs from RabbitMQ queue
- Extract data from configured external databases
- Encrypt and store results in configurable object storage (SeaweedFS or S3-compatible)
- Publish job completion/failure notifications

**Important:** This component has **NO HTTP routes** - it operates purely as a message consumer.

**Entry Point:** `components/worker/cmd/app/main.go`

#### Internal Structure

```
components/worker/
├── cmd/app/
│   └── main.go                      # Application entry point
└── internal/
    ├── adapters/
    │   └── rabbitmq/
    │       ├── consumer.rabbitmq.go # RabbitMQ consumer adapter (Primary Adapter)
    │       └── publisher.rabbitmq.go # RabbitMQ publisher adapter (Secondary Adapter)
    ├── bootstrap/
    │   ├── config.go                # Dependency injection + license/TLS gating
    │   ├── service.go               # Application service wrapper
    │   ├── consumer.go              # Multi-queue consumer orchestration with tenant ID extraction from AMQP headers and tenant MongoDB resolution
    │   ├── worker_multi_tenant_consumer.go # Per-tenant consumer fan-out (multi-tenant mode)
    │   ├── extraction_engine.go     # Wires the mandatory EngineRunner (nil runner is startup-fatal)
    │   ├── health_server.go         # Lightweight HTTP server for /health + /readyz
    │   ├── readyz_adapters.go       # /readyz dependency checker wiring (adds S3 HeadBucket)
    │   └── retry_guard.go           # Redelivery/retry guard for poison messages
    └── services/
        ├── service.go               # UseCase struct definition
        ├── extract_data.go          # Generic extraction entry point + result encrypt/store/HMAC
        ├── extract_engine.go        # extractViaEngine: routes generic extraction through the engine
        ├── extract_crm_data.go      # Plugin CRM extraction (EXCEPTION: bypasses the engine)
        ├── job_notification.go      # Job event notification publishing
        └── terminal_event_repairer.go # Repairs/re-emits missed terminal job events
```

---

## Packages (pkg/)

### Root Package (`pkg/`)

| File | Purpose |
|------|---------|
| `context.go` | Context utilities and propagation |
| `errors.go` | Custom error types and handling |
| `os.go` | OS-level utilities |
| `utils.go` | General utility functions |
| `time_utils.go` | Time manipulation utilities |

### engine (`pkg/engine/`) — Embedded Runtime Core

The **embedded runtime engine** is the importable, infrastructure-free core that owns extraction **rules**: connection lifecycle, schema discovery/validation, query planning, extraction execution, canonical result/error contracts, limits, and tenant-safety. It is designed to embed in any host (the Fetcher Manager and Worker today; Matcher and Reporter tomorrow) without dragging infrastructure with it.

| File | Purpose |
|------|---------|
| `engine.go` | Facade — `engine.New(opts ...Option) (*Engine, error)` |
| `options.go` | Functional options wiring host-provided ports |
| `ports.go` | Host-provided port interfaces (the engine's only dependencies) |
| `connection.go`, `connection_ops.go` | Connection lifecycle rules |
| `schema.go`, `schema_ops.go` | Schema discovery/validation rules |
| `planner.go`, `extraction.go`, `runner.go` | Query planning + extraction execution |
| `connector.go`, `result_test.go` (contract) | Canonical result/error contracts |
| `limits.go` | Limit enforcement |
| `test_connection.go` | Connection test |
| `dependency_test.go` | **Build-enforced boundary guard** (see below) |
| `memory/` | In-memory port implementations for TEST/EMBEDDED harness only — **not production persistence** |

**Ports (`pkg/engine/ports.go`).** The engine depends ONLY on host-provided interfaces:

- **Required:** `ConnectorRegistry`, `CredentialProtector`
- **Optional:** `ConnectionStore`, `ExecutionStore`, `ResultSink`, `SchemaCache`, `ActiveExecutionChecker`, `Observability`

**Boundary contract (build-enforced).** `dependency_test.go` runs `go list -deps` and **fails the build** if the engine transitively imports any infrastructure. Forbidden imports include: Fiber, swag, amqp091-go, lib-streaming, mongo-driver, go-redis, SQL drivers (pgx/mysql/mssqldb/go-ora/lib-pq), `database/sql`, `os/exec`, `plugin`, `net/http`, `net/rpc`, aws-sdk-go-v2, `pkg/seaweedfs`, the local infra packages (`pkg/rabbitmq`, `pkg/storage`, `pkg/mongodb`, `pkg/redis`, `pkg/net/http`, `pkg/postgres`, `pkg/mysql`, `pkg/oracle`, `pkg/sqlserver`, `pkg/datasource`, `pkg/ratelimit`, `pkg/bootstrap/readyz`), lib-auth, lib-license-go, `components/*/internal`, and `deployments/helm/infra`. Rationale: the engine must embed in any host without pulling in infrastructure.

**Execution modes:**

| Mode | Behavior |
|------|----------|
| **Direct** | Inline bytes + SHA-256 integrity |
| **Store** | Persisted via `ResultSink`, returns a reference |
| **Auto** | Store if a sink is present, else Direct |

**Limits enforced:** `timeout` (via `context.WithTimeout`), `MaxResultBytes` (post-serialization), `MaxDatasources` / `MaxTablesPerDatasource` / `MaxFieldsPerTable` (at planning). Per-request `Overrides` may only **lower** limits, never raise them.

**Isolation:** `TenantID` is the **sole** isolation dimension. The engine core has no org/product concept — those belong to the host.

### enginecompat (`pkg/enginecompat/`) — Compatibility Adapters

Bridges the engine's ports to Fetcher's concrete infrastructure. The engine stays infra-free; these adapters do the wiring.

| Subpackage | Purpose |
|------------|---------|
| `connectioncompat/` | Mongo connection repo → `ConnectionStore`; job repo → `ActiveExecutionChecker`; tenant bridge |
| `schemacompat/` | Datasource factory + crypto → `ConnectorFactory`; Redis → `SchemaCache`; request-scoped `ConnectionStore` via `WithResolvedConnections` |
| `datasource/` | Worker extraction `ConnectorFactory` (wraps the existing `pkg/datasource` factory) |
| `plugincrm/` | CRM detection helpers |
| `tablenorm/` | Table/field key normalization |

### constant (`pkg/constant/`)

Application-wide constants.

| File | Purpose |
|------|---------|
| `app.go` | Application constants (includes `ApplicationName`, `ModuleManager`, `ModuleWorker` for multi-tenant service identification) |
| `errors.go` | Error code constants |
| `mongo.go` | MongoDB-specific constants |
| `pagination.go` | Pagination defaults |
| `datasource-config.go` | DataSource type constants |
| `seaweedfs.go` | SeaweedFS configuration constants |

### model (`pkg/model/`)

Domain models including entities, DTOs, requests, and responses.

| File | Purpose |
|------|---------|
| `connection.go` | Connection domain entity + DTOs (ConnectionInput, ConnectionUpdateInput, ConnectionResponse). Includes `ProductName` field for product isolation. |
| `job.go` | Job domain entity + DTOs (FetcherRequest, FetcherResponse, JobResponse, JobStatus enum) |
| `schema.go` | Schema validation models (SchemaValidationRequest, SchemaValidationResponse, DataSourceSchema) |
| `pagination.go` | Pagination models and utilities |

**Subpackages:**

- `model/job/` - Job queue message types (`job_queue.go`)
- `model/datasource/` - DataSource interface + per-DB config models: `postgres/`, `mysql/`, `oracle/`, `sqlserver/`, `mongodb/`

### ports (`pkg/ports/`)

Repository and port interfaces (the hexagonal "ports"), grouped by subdirectory. Each has a `*.mock.go` generated alongside via `mockgen`.

| Subpackage | Interface |
|------------|-----------|
| `connection/` | `Repository` — connection persistence |
| `job/` | `Repository` — job persistence |
| `cache/` | `SchemaCacheRepository` — Redis schema cache |
| `publisher/` | `Repository` — job event publishing |
| `storage/` | `Repository` — object storage |
| `messaging/` | `Publisher` — message dispatch |
| `datasource/` | `CRM` — plugin CRM datasource contract |

### resolver (`pkg/resolver/`)

Resolves which datasource/database a request targets, and loads operator-defined internal datasources.

| File | Purpose |
|------|---------|
| `env_loader.go` | Loads internal datasources from `DATASOURCE_{NAME}_*` env vars (incl. SSL mode) |
| `registry.go` | Internal datasource registry |
| `resolver.go`, `single_tenant.go`, `multi_tenant.go` | Tenant→database resolution (single vs multi-tenant) |
| `tenant_config_adapter.go` | Adapts tenant config provider to the resolver |

### mongodb (`pkg/mongodb/`)

MongoDB connection and repository implementations.

| File | Purpose |
|------|---------|
| `mongo.go` | MongoDB connection management |
| `tenant.go` | Shared tenant-aware database resolution (ResolveDatabase) |

**Subpackages:**

- `mongodb/connection/` - Connection repository
  - `connection.go` - MongoDB model mapping
  - `connection.mongodb.go` - Repository implementation (CRUD operations)
  - `indexes.go` - Index definitions

- `mongodb/job/` - Job repository
  - `job.go` - MongoDB model mapping
  - `job.mongodb.go` - Repository implementation (CRUD operations)
  - `indexes.go` - Index definitions

### redis (`pkg/redis/`)

Redis/Valkey client adapter used for rate limiting and schema caching.

| File | Purpose |
|------|---------|
| `redis.go` | Redis connection management and client wrapper |

### ratelimit (`pkg/ratelimit/`)

Rate limiter implementation backed by Redis.

| File | Purpose |
|------|---------|
| `ratelimit.go` | Token-bucket rate limiter using Redis store |

### datasource/sslmode (`pkg/datasource/sslmode/`)

SSL mode configuration utilities for database connections.

| File | Purpose |
|------|---------|
| `sslmode.go` | SSL mode validation and injection for datasource configs |

### Database Adapters

Each database type has its own package implementing the DataSource interface:

| Package | Files | Purpose |
|---------|-------|---------|
| `postgres/` | `postgres.go`, `datasource.postgres.go` | PostgreSQL connection and data extraction |
| `mysql/` | `mysql.go`, `datasource.mysql.go` | MySQL connection and data extraction |
| `oracle/` | `oracle.go`, `datasource.oracle.go` | Oracle connection and data extraction |
| `sqlserver/` | `sqlserver.go`, `datasource.sqlserver.go` | SQL Server connection and data extraction |

### datasource (`pkg/datasource/`)

DataSource factory that creates the appropriate database adapter based on connection type. Also home to the SSRF guard (`hostsafety/`) and SSL mode utilities (`sslmode/`).

| File | Purpose |
|------|---------|
| `datasource_factory.go` | Factory pattern implementation + pre-dial SSRF host check |

```go
func NewDataSourceFromConnection(ctx context.Context, conn *model.Connection, cryptor crypto.Cryptor, logger log.Logger) (datasource.DataSource, error) {
    switch conn.Type {
    case model.TypeMongoDB:     return newDataSourceConfigMongoDB(...)
    case model.TypePostgreSQL:  return newDataSourceConfigPostgres(...)
    case model.TypeOracle:      return newDataSourceConfigOracle(...)
    case model.TypeMySQL:       return newDataSourceConfigMySQL(...)
    case model.TypeSQLServer:   return newDataSourceConfigSQLServer(...)
    }
}
```

### rabbitmq (`pkg/rabbitmq/`)

Resilient RabbitMQ adapter with connection management and message publishing.

| File | Purpose |
|------|---------|
| `rabbitmq.go` | RabbitMQ connection, channel management, publishing |

### storage (`pkg/storage/`)

Provider-agnostic storage factory that selects between SeaweedFS and S3-compatible backends at startup.

| File | Purpose |
|------|---------|
| `factory.go` | `NewRepository()` factory — selects backend from `STORAGE_PROVIDER` env var (`"seaweedfs"` or `"s3"`) |
| `s3.go` | `S3Repository` — AWS SDK v2 implementation; supports AWS S3, MinIO, SeaweedFS S3, and any S3-compatible service |

SSL for S3 is controlled by the URL scheme of `OBJECT_STORAGE_ENDPOINT` (`http://` → no SSL, `https://` → SSL). The factory defaults to SeaweedFS when `STORAGE_PROVIDER` is empty.

### seaweedfs (`pkg/seaweedfs/`)

SeaweedFS HTTP client. Used directly by `pkg/storage` when `STORAGE_PROVIDER=seaweedfs`.

| File | Purpose |
|------|---------|
| `seaweedfs.go` | SeaweedFS HTTP client operations |
| `external/external_data.go` | `SimpleRepository` — implements `storage.Repository` against SeaweedFS Filer |

### crypto (`pkg/crypto/`)

Encryption service using AES-GCM.

| File | Purpose |
|------|---------|
| `crypto.go` | AES-GCM encryption/decryption service |

### net/http (`pkg/net/http/`)

HTTP utilities for the Fiber framework.

| File | Purpose |
|------|---------|
| `errors.go` | HTTP error handling and responses |
| `response.go` | Standard response formatting |
| `cursor.go` | Cursor-based pagination utilities |
| `with_body.go` | Request body handling + the `safe_host` SSRF validator tag |
| `with_recover.go` | Panic-recovery middleware |
| `request_id.go` | Request ID propagation |
| `http-utils.go` | General HTTP utilities |

### schemautil (`pkg/schemautil/`)

Utility functions for handling database schema operations (shared between the schema engine path and adapters).

### startup (`pkg/startup/`)

Bootstrap-time error sanitization. `SanitizeError` redacts connection strings and credential fragments (e.g. `amqp://user:pass@host`) from errors before they are printed to stderr during startup.

### metrics (`pkg/metrics/`)

Multi-tenant metrics instrumentation using OpenTelemetry.

| File | Purpose |
|------|---------|
| `tenant_metrics.go` | 4 canonical tenant metrics with no-op when disabled |

### multitenant (`pkg/multitenant/`)

Cross-cutting multi-tenant test suite (test-only package).

| File | Purpose |
|------|---------|
| `multitenant_test.go` | Tenant isolation, backward compatibility, and context propagation tests |

---

## Architecture Layers

### Hexagonal Architecture

The project follows hexagonal architecture (ports and adapters):

```
                    ┌─────────────────────────────────────────┐
                    │              PRIMARY ADAPTERS           │
                    │         (Inbound - Driving Side)        │
                    │                                         │
                    │  ┌─────────────┐    ┌─────────────────┐ │
                    │  │ HTTP/Fiber  │    │ RabbitMQ        │ │
                    │  │ (Manager)   │    │ Consumer        │ │
                    │  │             │    │ (Worker)        │ │
                    │  └──────┬──────┘    └────────┬────────┘ │
                    └─────────┼───────────────────┼───────────┘
                              │                    │
                              ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         APPLICATION CORE                                │
│                                                                         │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                    APPLICATION SERVICES                            │ │
│  │                                                                    │ │
│  │   ┌─────────────────────┐      ┌─────────────────────┐             │ │
│  │   │      COMMANDS       │      │       QUERIES       │             │ │
│  │   │  (Write Operations) │      │  (Read Operations)  │             │ │
│  │   │                     │      │                     │             │ │
│  │   │ - CreateConnection  │      │ - GetConnection     │             │ │
│  │   │ - UpdateConnection  │      │ - ListConnections   │             │ │
│  │   │ - DeleteConnection  │      │ - TestConnection    │             │ │
│  │   │ - CreateFetcherJob  │      │ - GetJob            │             │ │
│  │   │ - AssignConnection  │      │ - ListUnassigned    │             │ │
│  │   │                     │      │                     │             │ │
│  │   └─────────────────────┘      └─────────────────────┘             │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                         DOMAIN                                     │ │
│  │                                                                    │ │
│  │   ┌─────────────┐    ┌─────────────┐                              │ │
│  │   │ Connection  │    │    Job      │                              │ │
│  │   │   Entity    │    │   Entity    │                              │ │
│  │   └─────────────┘    └─────────────┘                              │ │
│  │   ┌─────────────┐                                                 │ │
│  │   │ DataSource  │                                                 │ │
│  │   │  Interface  │                                                 │ │
│  │   └─────────────┘                                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                     PORTS (Interfaces)                             │ │
│  │                                                                    │ │
│  │   - ConnectionRepository    - RabbitMQ Publisher                   │ │
│  │   - JobRepository           - Storage Repository                    │ │
│  │   - SchemaCacheRepository   - Cryptor                              │ │
│  │   - DataSource                                                     │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                              │                    │
                              ▼                    ▼
                    ┌─────────────────────────────────────────┐
                    │            SECONDARY ADAPTERS           │
                    │        (Outbound - Driven Side)         │
                    │                                         │
                    │  ┌───────────┐  ┌───────────┐  ┌──────┐ │
                    │  │  MongoDB  │  │ RabbitMQ  │  │ Etc. │ │
                    │  │   Repo    │  │ Publisher │  │      │ │
                    │  └───────────┘  └───────────┘  └──────┘ │
                    │                                         │
                    │  ┌───────────┐  ┌───────────┐           │
                    │  │  Storage  │  │ Database  │           │
                    │  │   Repo    │  │ Adapters  │           │
                    │  └───────────┘  └───────────┘           │
                    └─────────────────────────────────────────┘
```

### CQRS Pattern

Commands and Queries are separated:

- **Commands** (Write operations): `CreateConnection`, `UpdateConnection`, `DeleteConnection`, `CreateFetcherJob`, `AssignConnection`
- **Queries** (Read operations): `GetConnection`, `ListConnections`, `TestConnection`, `GetJob`, `GetConnectionSchema`, `ValidateSchema`, `ListUnassignedConnections`

**Worker Exception:** The Worker component does **not** follow CQRS. Instead, it uses a single `UseCase` struct (`components/worker/internal/services/service.go`) that holds all dependencies and exposes methods like `ExtractExternalData()` and `SendJobNotification()`. This is intentional because the Worker has no HTTP API and processes messages from a single queue, making CQRS separation unnecessary.

### Embedded Runtime Engine

Both components run their extraction logic over the embedded runtime engine (`pkg/engine`, see [Packages](#engine-pkgengine--embedded-runtime-core)). The engine owns the *rules*; the host owns the *infrastructure*. Wiring happens in each component's `bootstrap/` package via `pkg/enginecompat` adapters.

**Manager** wires **two** engine instances in `components/manager/internal/bootstrap/`:

| File | Engine | Delegated operations |
|------|--------|----------------------|
| `connection_engine.go` | Connection engine | Connection CRUD |
| `schema_engine.go` | Schema engine | Schema discovery / validation |

The Manager retains ownership of auth, license, HTTP, rate-limiting, idempotency, RabbitMQ dispatch, and `/readyz`. **Test-connection does NOT go through the engine** — it still dials directly via the datasource factory.

**Worker** wires a **mandatory** `EngineRunner` in `components/worker/internal/bootstrap/extraction_engine.go`; a `nil` runner is **startup-fatal**. Generic extraction flows: Worker `UseCase` → `extract_engine.go` (`extractViaEngine`) → `EngineRunner.RunExtraction` → `engine.PlanExtraction` + `engine.ExecuteExtraction` (DIRECT mode). The Worker owns encrypt/store/HMAC, job status, and event publishing **outside** the engine. `plugin_crm` is the documented **exception** that bypasses the engine entirely (legacy `extract_crm_data.go`: collection-prefix fan-out, filter-field hashing, PII decryption).

> **Note:** The Worker's legacy direct extraction path has been **removed** — generic extraction can only run through the engine.

---

## Sequence Diagrams

### POST /v1/management/connections - Create Connection

```mermaid
sequenceDiagram
    participant Client
    participant Manager as Manager Component
    participant Handler as HTTP Handler
    participant Command as CreateConnectionCommand
    participant Cryptor as Crypto Service
    participant Repo as ConnectionRepository
    participant MongoDB

    Client->>Manager: POST /v1/management/connections
    Manager->>Handler: CreateConnection()
    Handler->>Handler: Parse & Validate Request
    Handler->>Command: Execute(input)
    Command->>Cryptor: Encrypt(password)
    Cryptor-->>Command: encryptedPassword
    Command->>Command: Generate UUID, Set timestamps
    Command->>Repo: Create(connection)
    Repo->>MongoDB: InsertOne()
    MongoDB-->>Repo: Result
    Repo-->>Command: connection
    Command-->>Handler: connection
    Handler-->>Manager: 201 Created + ConnectionResponse
    Manager-->>Client: HTTP 201
```

### GET /v1/management/connections - List Connections

```mermaid
sequenceDiagram
    participant Client
    participant Manager as Manager Component
    participant Handler as HTTP Handler
    participant Query as ListConnectionsQuery
    participant Repo as ConnectionRepository
    participant MongoDB

    Client->>Manager: GET /v1/management/connections
    Manager->>Handler: ListConnections()
    Handler->>Handler: Parse pagination params
    Handler->>Query: Execute(orgId, params)
    Query->>Repo: List(orgId, params)
    Repo->>MongoDB: Find() with filters
    MongoDB-->>Repo: []Connection
    Repo-->>Query: []Connection
    Query-->>Handler: []Connection
    Handler->>Handler: Map to []ConnectionResponse
    Handler-->>Manager: 200 OK + Response
    Manager-->>Client: HTTP 200
```

### GET /v1/management/connections/{id} - Get Connection

```mermaid
sequenceDiagram
    participant Client
    participant Manager as Manager Component
    participant Handler as HTTP Handler
    participant Query as GetConnectionQuery
    participant Repo as ConnectionRepository
    participant MongoDB

    Client->>Manager: GET /v1/management/connections/{id}
    Manager->>Handler: GetConnection()
    Handler->>Handler: Extract ID from path
    Handler->>Query: Execute(id, orgId)
    Query->>Repo: FindByID(id, orgId)
    Repo->>MongoDB: FindOne()
    MongoDB-->>Repo: Connection
    Repo-->>Query: Connection
    Query-->>Handler: Connection
    Handler->>Handler: Map to ConnectionResponse
    Handler-->>Manager: 200 OK + Response
    Manager-->>Client: HTTP 200
```

### POST /v1/management/connections/{id}/test - Test Connection

```mermaid
sequenceDiagram
    participant Client
    participant Manager as Manager Component
    participant Handler as HTTP Handler
    participant RateLimiter as Rate Limiter
    participant Query as TestConnectionQuery
    participant Repo as ConnectionRepository
    participant Cryptor as Crypto Service
    participant Factory as DataSourceFactory
    participant ExternalDB as External Database

    Client->>Manager: POST /v1/management/connections/{id}/test
    Manager->>Handler: TestConnection()
    Handler->>RateLimiter: Check rate limit (10/min)
    RateLimiter-->>Handler: OK
    Handler->>Query: Execute(id, orgId)
    Query->>Repo: FindByID(id, orgId)
    Repo-->>Query: Connection
    Query->>Cryptor: Decrypt(password)
    Cryptor-->>Query: decryptedPassword
    Query->>Factory: NewDataSourceFromConnection()
    Factory-->>Query: DataSource
    Query->>ExternalDB: Connect()
    ExternalDB-->>Query: Success/Error
    Query->>ExternalDB: Close()
    Query-->>Handler: TestResult
    Handler-->>Manager: 200 OK + TestResponse
    Manager-->>Client: HTTP 200
```

### PATCH /v1/management/connections/{id} - Update Connection

```mermaid
sequenceDiagram
    participant Client
    participant Manager as Manager Component
    participant Handler as HTTP Handler
    participant Command as UpdateConnectionCommand
    participant Repo as ConnectionRepository
    participant JobRepo as JobRepository
    participant Cryptor as Crypto Service
    participant MongoDB

    Client->>Manager: PATCH /v1/management/connections/{id}
    Manager->>Handler: UpdateConnection()
    Handler->>Handler: Parse & Validate Request
    Handler->>Command: Execute(id, orgId, input)
    Command->>JobRepo: HasActiveJobs(connectionId)
    JobRepo-->>Command: hasActive
    alt Has Active Jobs
        Command-->>Handler: 409 Conflict
    else No Active Jobs
        Command->>Repo: FindByID(id, orgId)
        Repo-->>Command: connection
        opt Password changed
            Command->>Cryptor: Encrypt(newPassword)
            Cryptor-->>Command: encryptedPassword
        end
        Command->>Command: Apply updates, set UpdatedAt
        Command->>Repo: Update(connection)
        Repo->>MongoDB: UpdateOne()
        MongoDB-->>Repo: Result
        Repo-->>Command: connection
        Command-->>Handler: connection
        Handler-->>Manager: 200 OK + ConnectionResponse
    end
    Manager-->>Client: HTTP Response
```

### DELETE /v1/management/connections/{id} - Delete Connection

```mermaid
sequenceDiagram
    participant Client
    participant Manager as Manager Component
    participant Handler as HTTP Handler
    participant Command as DeleteConnectionCommand
    participant JobRepo as JobRepository
    participant Repo as ConnectionRepository
    participant MongoDB

    Client->>Manager: DELETE /v1/management/connections/{id}
    Manager->>Handler: DeleteConnection()
    Handler->>Command: Execute(id, orgId)
    Command->>JobRepo: HasActiveJobs(connectionId)
    JobRepo-->>Command: hasActive
    alt Has Active Jobs
        Command-->>Handler: 409 Conflict
    else No Active Jobs
        Command->>Repo: FindByID(id, orgId)
        Repo-->>Command: connection
        Command->>Repo: Delete(id, orgId)
        Repo->>MongoDB: UpdateOne(deletedAt)
        MongoDB-->>Repo: Result
        Repo-->>Command: OK
        Command-->>Handler: OK
        Handler-->>Manager: 204 No Content
    end
    Manager-->>Client: HTTP Response
```

### POST /v1/fetcher - Create Job

```mermaid
sequenceDiagram
    participant Client
    participant Manager as Manager Component
    participant Handler as HTTP Handler
    participant Command as CreateFetcherJobCommand
    participant ConnRepo as ConnectionRepository
    participant TestQuery as TestConnectionQuery
    participant JobRepo as JobRepository
    participant RabbitMQ
    participant MongoDB

    Client->>Manager: POST /v1/fetcher
    Manager->>Handler: CreateJob()
    Handler->>Handler: Parse & Validate Request
    Handler->>Command: Execute(input)
    Command->>Command: Compute request hash
    Command->>JobRepo: FindByHash(hash, 5min window)
    JobRepo-->>Command: existingJob?
    alt Duplicate found
        Command-->>Handler: 200 OK (existing job)
    else New job
        Command->>ConnRepo: FindByConfigNames(names)
        ConnRepo-->>Command: []Connection
        loop For each connection
            Command->>TestQuery: TestConnection(conn)
            TestQuery-->>Command: Success/Error
        end
        alt Any connection failed
            Command-->>Handler: 400 Bad Request
        else All connections OK
            Command->>Command: Generate UUID, Set status=PENDING
            Command->>JobRepo: Create(job)
            JobRepo->>MongoDB: InsertOne()
            MongoDB-->>JobRepo: Result
            JobRepo-->>Command: job
            Command->>RabbitMQ: Publish(job message)
            RabbitMQ-->>Command: ACK
            Command-->>Handler: job
            Handler-->>Manager: 202 Accepted + JobResponse
        end
    end
    Manager-->>Client: HTTP Response
```

### GET /v1/fetcher/{id} - Get Job

```mermaid
sequenceDiagram
    participant Client
    participant Manager as Manager Component
    participant Handler as HTTP Handler
    participant Query as GetJobQuery
    participant Repo as JobRepository
    participant MongoDB

    Client->>Manager: GET /v1/fetcher/{id}
    Manager->>Handler: GetJob()
    Handler->>Handler: Extract ID from path
    Handler->>Query: Execute(id, orgId)
    Query->>Repo: FindByID(id, orgId)
    Repo->>MongoDB: FindOne()
    MongoDB-->>Repo: Job
    Repo-->>Query: Job
    Query-->>Handler: Job
    Handler->>Handler: Map to JobResponse
    Handler-->>Manager: 200 OK + Response
    Manager-->>Client: HTTP 200
```

### Worker Component - Extract Data Flow

Generic extraction now runs **through the embedded engine** (DIRECT mode). The Worker owns everything *outside* the engine: decrypting nothing inside the rules (the engine's `CredentialProtector` port handles that), but encrypt/store/HMAC of results, job status, and event publishing. `plugin_crm` is the documented **exception** — it bypasses the engine entirely and uses the legacy `extract_crm_data.go` path.

```mermaid
sequenceDiagram
    participant RabbitMQ as RabbitMQ Queue
    participant Consumer as MultiQueueConsumer
    participant Routes as ConsumerRoutes
    participant Handler as handlerGenerateReport
    participant UseCase as ExtractExternalData
    participant JobRepo as JobRepository
    participant Engine as Embedded Engine
    participant Connector as Engine Connector port
    participant Factory as DataSourceFactory
    participant ExternalDB as External Database
    participant Cryptor as Crypto Service
    participant Storage as Object Storage
    participant Publisher as RabbitMQ Publisher
    participant MongoDB

    RabbitMQ->>Consumer: Message (job payload)
    Consumer->>Routes: Dispatch message
    Routes->>Handler: Handle(message)
    Handler->>Handler: Parse message body
    Handler->>UseCase: ExtractExternalData(ctx, body)

    UseCase->>UseCase: Extract jobId from message
    UseCase->>JobRepo: FindByID(jobId)
    JobRepo->>MongoDB: FindOne()
    MongoDB-->>JobRepo: Job
    JobRepo-->>UseCase: Job

    alt Job already completed
        UseCase-->>Handler: Skip (idempotent)
    else Job pending/in_progress
        UseCase->>JobRepo: UpdateStatus(IN_PROGRESS)

        alt plugin_crm source (engine bypass)
            UseCase->>UseCase: extractCRMData (collection-prefix fan-out,<br/>filter-field hashing, PII decryption)
            UseCase->>Factory: NewDataSourceFromConnection()
            Factory-->>UseCase: DataSource
            UseCase->>ExternalDB: Query()
            ExternalDB-->>UseCase: []map[string]any
        else Generic source (via engine)
            UseCase->>UseCase: extractViaEngine(ctx, job)
            UseCase->>Engine: PlanExtraction(plan)
            Engine-->>UseCase: validated plan (limits enforced)
            UseCase->>Engine: ExecuteExtraction(plan) [DIRECT mode]
            Engine->>Connector: Open + Query (engine port)
            Connector->>Factory: NewDataSourceFromConnection()
            Factory-->>Connector: DataSource
            Connector->>ExternalDB: Connect / Query / Close
            ExternalDB-->>Connector: rows
            Connector-->>Engine: rows
            Engine-->>UseCase: Result (inline bytes + SHA-256)
        end

        UseCase->>Cryptor: Encrypt(results) + HMAC
        Cryptor-->>UseCase: encryptedData
        UseCase->>Storage: Store(encryptedData, TTL)
        Storage-->>UseCase: fileId

        UseCase->>JobRepo: UpdateStatus(COMPLETED, resultUrl)
        JobRepo->>MongoDB: UpdateOne()
        MongoDB-->>JobRepo: OK

        UseCase->>Publisher: Publish(job.completed event)
        Publisher->>RabbitMQ: Publish to topic exchange
        RabbitMQ-->>Publisher: ACK

        UseCase-->>Handler: Success
    end

    Handler-->>Routes: ACK message
    Routes-->>Consumer: ACK
    Consumer-->>RabbitMQ: ACK
```

---

## Inter-Component Communication

### Communication Flow

```
                                   Client
                                     │
                                     │ HTTP
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           MANAGER COMPONENT                              │
│                                                                          │
│   POST /v1/fetcher                                                       │
│      1. Validate request                                                 │
│      2. Check idempotency (5-min window)                                 │
│      3. Validate connections exist                                       │
│      4. Test each connection                                             │
│      5. Create job in MongoDB                                            │
│      6. Publish to RabbitMQ                                              │
│      7. Return 202 Accepted                                              │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                                     │ RabbitMQ
                                     │ fetcher.extract-external-data.queue
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           WORKER COMPONENT                               │
│                                                                          │
│   Consumer Loop                                                          │
│      1. Consume message                                                  │
│      2. Parse job details                                                │
│      3. Find connections by config name                                  │
│      4. Query each external database                                     │
│      5. Encrypt & store results in object storage (SeaweedFS or S3)      │
│      6. Update job status in MongoDB                                     │
│      7. Publish notification to RabbitMQ topic                           │
│      8. ACK message                                                      │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    │                │                │
                    ▼                ▼                ▼
               MongoDB        Object Storage    RabbitMQ
           (shared data)    (extracted data)  (notifications)
```

### Shared Resources

| Resource | Used By | Purpose |
|----------|---------|---------|
| MongoDB `connections` collection | Manager, Worker | Store connection configs (includes `productName` for product isolation) |
| MongoDB `jobs` collection | Manager, Worker | Store job state |
| Redis/Valkey | Manager | Rate limiting (connection tests), schema caching |
| RabbitMQ `fetcher.extract-external-data.queue` | Manager (publish), Worker (consume) | Job dispatch |
| RabbitMQ `fetcher.job.events` exchange | Worker (publish) | Job status notifications |
| Object storage `external_data` bucket (SeaweedFS or S3) | Worker | Store encrypted extracted data |
| Encryption keys (env vars) | Manager, Worker | Encrypt/decrypt passwords & data |

---

## Evolution Guidelines

### Adding a New Database Type

1. **Create adapter package** in `pkg/<dbtype>/`:
   - `<dbtype>.go` - Connection management
   - `datasource.<dbtype>.go` - DataSource interface implementation

2. **Create model** in `pkg/model/datasource/<dbtype>/`:
   - `datasource-config.go` - Configuration struct

3. **Update factory** in `pkg/datasource/datasource_factory.go`:
   - Add new case in switch statement
   - Add helper function for creating the adapter

4. **Add constant** in `pkg/constant/datasource-config.go`:
   - Add new type constant

5. **Update model** in `pkg/model/connection.go`:
   - Add new type to ConnectionType enum if needed

6. **Reachable through the engine:** extraction now flows through the embedded engine, so the new type must be reachable via the engine's connector registry. In practice the `pkg/enginecompat/datasource` `ConnectorFactory` **wraps** the existing `pkg/datasource` factory, so adding the factory case in step 3 is sufficient — no separate engine registration is required. Add an engine-path extraction test to confirm the new type resolves through `enginecompat`.

### Adding a New API Endpoint

1. **Create handler** in `components/manager/internal/adapters/http/in/`:
   - Add handler function with Swagger annotations

2. **Add route** in `components/manager/internal/adapters/http/in/routes.go`:
   - Register the new endpoint

3. **Create service** in `components/manager/internal/services/`:
   - If write operation: `command/<operation>.go`
   - If read operation: `query/<operation>.go`

4. **Update bootstrap** in `components/manager/internal/bootstrap/config.go`:
   - Wire dependencies if needed

5. **Generate docs**:
   - Run `make generate-docs`

### Adding a New Worker Queue

1. **Create handler** in `components/worker/internal/services/`:
   - Implement business logic

2. **Create adapter** in `components/worker/internal/adapters/rabbitmq/`:
   - Implement consumer if needed

3. **Register queue** in `components/worker/internal/bootstrap/consumer.go`:
   - Add queue configuration and handler mapping

4. **Update RabbitMQ topology** in `components/infra/rabbitmq/etc/definitions.json`:
   - Add queue, exchange, and binding definitions

### Internal datasource SSL/TLS env vars

Internal datasources (`midaz_onboarding`, `midaz_transaction`, `plugin_crm`) are loaded by `pkg/resolver/env_loader.go` from `DATASOURCE_{NAME}_*` env vars. SSL/TLS is opt-in via the following suffix:

| Suffix      | Required when SSL is desired | Notes |
|-------------|------------------------------|-------|
| `_SSLMODE`  | yes — gate                   | Allowed values are per-driver; see `pkg/datasource/sslmode/`. Per-type strings (`postgresql`, `mysql`, `oracle`, `sql_server`, `mongodb`) gate the allowlist. |

These variables are read once at process start (Manager and Worker). Changes require a pod restart — there is no hot reload.

Behavior:

- `_SSLMODE` absent → `conn.SSL` stays `nil`; the datasource factory falls back to its driver default (PostgreSQL = `disable`, MongoDB = `disable`, MySQL = `disable`, etc.).
- `_SSLMODE` set but **invalid** for the connection's `_TYPE` → the connection is **skipped** at load time with a `WARN` log; misconfiguration surfaces loudly instead of silently downgrading.
- `_SSLMODE` set and valid → `conn.SSL = &SSLConfig{Mode}` is populated and the datasource factory interpolates it into the driver's DSN/URI (`?sslmode=<value>` for PostgreSQL, `&tls=true[&tlsInsecure=true]` for MongoDB).

Required for managed databases that reject plaintext: AWS RDS PostgreSQL, GCP Cloud SQL, Azure Database for PostgreSQL all require `sslmode=require` or stronger. Without trust pinning, fetcher relies on the driver's default certificate validation (PostgreSQL `verify-full` consults the system CA store).

**Not implemented yet — follow-up:** Custom CA / client certificate plumbing (`_SSL_CA`, `_SSL_CERT`, `_SSL_KEY`) for the internal-datasource path is not wired into any driver today. Operators who require CA pinning or mTLS should track the follow-up task in `tasks/fetcher.md`. The multi-tenant resolver also propagates only `SSLMode` (not the cert bundle) and inherits the same limitation.

Operator guidance for the MongoDB allowlist: `_SSLMODE=insecure` (and `skip-verify`) explicitly disables hostname / chain validation by emitting `tlsInsecure=true`. Use these values only for local/test environments. For production MongoDB use `_SSLMODE=true` (TLS with full driver-side validation against the system trust store).

### Layer Responsibilities

| Layer | Location | Responsibility | Dependencies |
|-------|----------|----------------|--------------|
| **Primary Adapters** | `internal/adapters/http/in/` or `internal/adapters/rabbitmq/` | Receive external input, transform to domain | Application Services |
| **Application Services** | `internal/services/command/` or `internal/services/query/` | Orchestrate business logic, coordinate domain operations | Domain, Ports |
| **Domain** | `pkg/model/` | Business entities, value objects, domain logic | None (pure) |
| **Ports** | Interfaces in `pkg/` packages | Define contracts for external dependencies | None (interfaces) |
| **Secondary Adapters** | `pkg/mongodb/`, `pkg/rabbitmq/`, `pkg/storage/`, `pkg/seaweedfs/` | Implement ports, integrate with external systems | External libraries |

### Best Practices for Evolution

1. **Always use interfaces (ports)** for external dependencies
2. **Keep domain models pure** - no infrastructure concerns
3. **Separate commands from queries** - follow CQRS strictly
4. **Use the factory pattern** for creating adapters dynamically
5. **Test at service level** - mock ports, not implementations
6. **Keep handlers thin** - delegate to services immediately
7. **Use dependency injection** - wire dependencies in bootstrap

---

## Test Infrastructure

### Layout

| Suite | Location | Build tag | Notes |
|-------|----------|-----------|-------|
| **Unit** | `*_test.go` alongside source | none | `make test-unit` |
| **E2E** | `tests/e2e/` | `e2e` | `make test-e2e` (`make test-integration` is an alias) |
| **Chaos** | `tests/chaos/` | `chaos` | `make test-chaos` (Toxiproxy fault injection) |
| **Fuzz** | `tests/fuzz/` | none | `make test-fuzzy` |

The container/infrastructure orchestration that used to live under `tests/shared/{containers,network,...}` has moved to the reusable **`pkg/itestkit`** library. `tests/shared/` is now a flat set of E2E/chaos helpers (package `shared`, conventionally imported as `e2eshared`): `client.go` (Manager API client), `apps.go` (app container startup), `assertions.go` + `helpers.go` (job polling, product-name generation), `readyz.go` (`/readyz` probes), `infra.go`, `storage_bucket.go`, plus `fixtures/` and `testdata/`.

### itestkit (`pkg/itestkit/`)

Infrastructure-agnostic test kit built on testcontainers-go. See [`pkg/itestkit/README.md`](../pkg/itestkit/README.md) for the full builder/suite API.

| Path | Purpose |
|------|---------|
| `itestkit/infra/{postgres,mysql,oracle,mssql,mongodb,rabbitmq,redis,seaweedfs,minio}` | Pre-built container providers |
| `itestkit/addons/e2ekit` | Build app images from Dockerfile (BuildKit secrets, wait strategies, env rewriting) |
| `itestkit/addons/metricskit` | Chaos SLO assertions and reporting |
| `itestkit/addons/queuekit` | AMQP message-queue testing (matchers, consumer builder, assertions) |

`minio` is the S3-compatible provider for S3 integration/E2E tests (activated via `E2E_ENABLE_S3=true`).

### Running Tests

```bash
make test-unit                         # Unit tests (all components)
make test-e2e                          # E2E suite (requires Docker; -tags=e2e)
make test-integration                  # Alias for the E2E suite
make test-chaos                        # Chaos suite (Toxiproxy; -tags=chaos)
make test-fuzzy                        # Fuzz tests
make test-all                          # All suites sequentially

# E2E with pre-built images (skip in-test build)
E2E_SKIP_BUILD=true \
MANAGER_IMAGE=fetcher-manager:latest \
WORKER_IMAGE=fetcher-worker:latest \
  make test-e2e
```

> **See [`tests/e2e/README.md`](../tests/e2e/README.md)** for E2E conventions and the test catalog, and [`pkg/itestkit/README.md`](../pkg/itestkit/README.md) for the container kit API.

---

## Build and Development

### Quick Start

```bash
# Setup development environment
make dev-setup

# Copy environment files
make set-env

# Start all services
make up

# Run tests
make test

# Generate Swagger docs
make generate-docs
```

### Available Make Commands

| Command | Description |
|---------|-------------|
| `make help` | Display all available commands |
| `make dev-setup` | Complete development environment setup |
| `make set-env` | Copy .env.example to .env for all components |
| `make up` | Start all services (infra first, then backends) |
| `make down` | Stop all services |
| `make rebuild-up` | Rebuild and restart all services |
| `make test` | Run all tests |
| `make lint` | Run golangci-lint with auto-fix |
| `make sec` | Run gosec security analysis |
| `make generate-docs` | Generate Swagger documentation |

### CI/CD Workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `go-combined-analysis.yml` | PR to develop/main | Runs Go analysis pipeline |
| `release.yml` | Push to develop/main/hotfix | Semantic release + AI changelog |
| `build.yml` | Tag push | Build and publish Docker images |
| `check-branch.yml` | PR to main | Enforce PR only from develop/hotfix |

---

## Security: Host Safety / SSRF Guard

The Fetcher Manager exposes endpoints that, given any authenticated tenant, must accept a database `Host` value and establish a real network connection to it (CREATE/UPDATE connection, POST /test, POST /validate-schema, POST /v1/fetcher). Without a guard, these endpoints become a reconnaissance and SSRF vector against the platform's own internal network (loopback, RFC1918, cloud metadata IMDS at `169.254.169.254`, internal Kubernetes hostnames). The SSRF host safety guard closes this vector for multi-tenant deployments while preserving full backward compatibility for single-tenant operators who legitimately point at internal hosts.

### Behavior by Deployment Mode

| Mode | `MULTI_TENANT_ENABLED` | Guard active? | Behavior |
|------|-----------------------|---------------|----------|
| **Single-tenant** (default) | `false` | ❌ No | All hosts accepted, including loopback / RFC1918. The operator owns the deployment and may legitimately point at internal hosts. Zero regression vs. pre-fetcher-013 behavior. |
| **Multi-tenant** | `true` | ✅ Yes | Tenant-supplied hosts are filtered against a denylist; internal datasources loaded from the operator's environment variables remain exempt by construction. |

Activation is automatic: `InitServers` calls `hostsafety.SetHostSafetyEnabled(cfg.MultiTenantEnabled)` once at bootstrap, before any service or route is built. Single-tenant deployments require no opt-out.

### Defense-in-Depth: Two Layers

```
POST /v1/management/connections                   ┐
POST /v1/management/connections (PATCH)           │
POST /v1/management/connections/:id/test          ├─► Validator layer (DTO)
POST /v1/management/connections/validate-schema   │   `safe_host` tag in pkg/net/http/with_body.go
                                                  │   Rejects IP literals at request parse time.
                                                  │
                                                  └─► Factory layer (pre-dial)
                                                      hostsafety.ValidateHostForConnection
                                                      in pkg/datasource/datasource_factory.go.
                                                      Resolves hostnames via DNS and checks
                                                      every returned IP against the denylist.
```

| Layer | Where | What it catches | What it does NOT catch |
|-------|-------|-----------------|------------------------|
| **DTO `safe_host` validator** | `pkg/net/http/with_body.go` (`validateSafeHost`) | IP literals in the request body (`"host":"127.0.0.1"`) | Hostnames — passes them through to factory layer |
| **Factory guard** | `pkg/datasource/datasource_factory.go` (call to `hostsafety.ValidateHostForConnection`) | Hostnames that resolve to denylisted IPs; hostname literals matched by `libSSRF.IsBlockedHostname` (case-insensitive, root-label normalized) | DNS rebinding (resolve-time TOCTOU); covered by future `fetcher-014` |

Both layers read the same package-level flag (`hostsafety.SetHostSafetyEnabled`) and call into the same `libSSRF` predicates, so they cannot fall out of sync.

### Denylist — Source of Truth

The denylist of blocked IP networks and hostname literals lives in [`github.com/LerianStudio/lib-commons/v5/commons/security/ssrf`](https://github.com/LerianStudio/lib-commons/tree/main/commons/security/ssrf). Fetcher inherits updates on `go mod tidy`; the local adapter at [`pkg/datasource/hostsafety/hostsafety.go`](../pkg/datasource/hostsafety/hostsafety.go) only owns:

1. The bootstrap-time gate (`SetHostSafetyEnabled` / `IsEnabled`).
2. The internal-datasource bypass (`EncryptionKeyVersion == ""` short-circuit).

All blocklist semantics — CIDR matching, hostname literal matching, IPv4-mapped IPv6 unmap, fail-closed on unparseable IPs — are owned by libSSRF and exercised by `libSSRF.IsBlockedIP(net.IP)` and `libSSRF.IsBlockedHostname(string)`. Updating the denylist means bumping `lib-commons/v5` in `go.mod`, NOT editing the adapter.

Coverage categories enforced by libSSRF (non-exhaustive, see the upstream source for the canonical list):

| Category | Notes |
|----------|-------|
| IPv4 loopback | `127.0.0.0/8` |
| IPv4 link-local | `169.254.0.0/16` — covers AWS/GCP/Azure IMDS at `169.254.169.254` |
| IPv4 private (RFC1918) | `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16` |
| IPv4 reserved / non-routable | CGNAT (`100.64.0.0/10`), multicast, `0.0.0.0/8`, TEST-NETs (`192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`), benchmarking (`198.18.0.0/15`), IETF protocol assignments (`192.0.0.0/24`), reserved (`240.0.0.0/4`) |
| IPv6 equivalents | `::1/128` (loopback), `fe80::/10` (link-local), `fc00::/7` (ULA), `::/128` (unspecified), `2001:db8::/32` (documentation), `100::/64` (discard-only) |
| IPv4-mapped IPv6 | Handled via `addr.Unmap()` normalization — `::ffff:127.0.0.1` is treated identically to `127.0.0.1` |
| Hostname literals | `localhost`, `metadata.google.internal`, `metadata.gcp.internal`, `metadata.azure.internal`, `169.254.169.254`, plus suffixes `.local` (mDNS), `.internal` (cloud metadata + internal DNS), `.cluster.local` (Kubernetes) |

Matching is case-insensitive and strips the optional trailing root label (`localhost.` is treated as `localhost`). Hostnames that fail to resolve (NXDOMAIN, timeout) are deliberately NOT blocked at the factory layer — forcing a 400 there would leak a different error than for "doesn't exist", recreating the same reconnaissance oracle the guard is meant to close. The driver dial that follows surfaces its own connect error.

### Internal Datasource Bypass

The Fetcher bootstrap loads "internal" datasources (e.g., for self-hosted reporting) from environment variables via `pkg/resolver/env_loader.go`. These connections are created in-memory with `EncryptionKeyVersion == ""` (the in-memory plaintext-password contract — see `pkg/datasource/datasource_factory.go` resolvePassword).

`ValidateHostForConnection` short-circuits to `nil` when `EncryptionKeyVersion == ""`, exempting these connections from the guard. Rationale: they are defined by the platform operator (not by a tenant), so they fall outside the threat model — and they routinely legitimately point at internal addresses.

### Trust Chain Invariant — Manager-Only Enforcement

**The guard is enforced ONLY in the Manager HTTP API.** The Worker has no guard of its own. This is correct given the following invariant the design assumes:

> The only path for a `Connection` to enter MongoDB is via the Manager HTTP API. The Worker reads `Connection` documents from MongoDB by ID at job-processing time and trusts them to have been validated upstream.

**Any future writer of `Connection` documents** (e.g., a migration script, an import job, a separate service consuming a Kafka topic, an admin CLI) **MUST** either:

1. Invoke `hostsafety.ValidateHostForConnection(ctx, conn)` before persisting, OR
2. Be explicitly exempted by setting `EncryptionKeyVersion == ""` (only valid for trusted operator-loaded datasources).

Breaking this invariant silently re-opens the SSRF vector. If you change the persistence boundary, update both this section and `hostsafety.ValidateHostForConnection`.

### Error Surface

The guard adds one new error code to the existing schema:

| Code | Constant | HTTP | Title | Message |
|------|----------|------|-------|---------|
| `FET-0414` | `constant.ErrForbiddenHost` | 400 | Forbidden Host | `Host is not a valid external database endpoint` |

The message is intentionally generic. It does NOT echo the host, identify which CIDR matched, or distinguish DTO-layer rejection from factory-layer rejection. Surfacing those details would recreate the reconnaissance oracle the guard is built to close.

### Out of Scope (Explicit Non-Goals)

The guard is scoped to **reconnaissance by authenticated tenants** against internal infrastructure. The following adjacent threats are documented as follow-ups:

| Concern | Follow-up task | Notes |
|---------|---------------|-------|
| DNS rebinding (TOCTOU between resolve and dial) | `fetcher-014` | Requires a `SafeDialer` that pins the resolved IP for the lifetime of the connection. Activate if the threat model evolves to include hostile domain owners. |
| Worker-side guard (in case the Mongo write invariant changes) | `fetcher-015` | Only required if a second writer of `Connection` documents is introduced. |
| Per-tenant allowlist (explicit "this tenant may reach host X") | `fetcher-016` | Today the denylist is global. An allowlist interface lets specific tenants opt into reaching a curated internal host (e.g., a regulator-mandated regulator IMDS proxy). |
| E2E activation (SSRF tests in `tests/e2e/connection_ssrf_test.go` run as `t.Skip` until MT-enabled E2E lands) | `fetcher-017` | Requires `EnvOverride` in `e2eshared.AppStartConfig` + a tenant-manager fixture (Opção A from fetcher-013 M4 gap report). |

---

## Key Files Reference

| Component | File | Purpose |
|-----------|------|---------|
| Manager | `components/manager/cmd/app/main.go` | Entry point |
| Manager | `components/manager/internal/bootstrap/config.go` | DI container + multi-tenant middleware init |
| Manager | `components/manager/internal/adapters/http/in/routes.go` | Route definitions |
| Manager | `components/manager/internal/services/command/create_fetcher_job.go` | Job creation logic |
| Worker | `components/worker/cmd/app/main.go` | Entry point |
| Worker | `components/worker/internal/bootstrap/config.go` | DI container |
| Worker | `components/worker/internal/bootstrap/consumer.go` | Consumer orchestration + tenant DB resolution |
| Worker | `components/worker/internal/services/extract_data.go` | Generic extraction entry + result encrypt/store/HMAC |
| Worker | `components/worker/internal/services/extract_engine.go` | Routes generic extraction through the embedded engine |
| Worker | `components/worker/internal/bootstrap/extraction_engine.go` | Wires the mandatory EngineRunner |
| Shared | `pkg/engine/engine.go` | Embedded runtime engine facade (`engine.New`) |
| Shared | `pkg/engine/ports.go` | Engine host-provided port interfaces |
| Shared | `pkg/engine/dependency_test.go` | Build-enforced engine boundary guard |
| Shared | `pkg/model/connection.go` | Connection domain |
| Shared | `pkg/model/job.go` | Job domain |
| Shared | `pkg/datasource/datasource_factory.go` | DataSource factory + pre-dial SSRF check |
| Shared | `pkg/mongodb/tenant.go` | Shared tenant-aware database resolution |
| Shared | `pkg/metrics/tenant_metrics.go` | Multi-tenant OTel metrics |
| Shared | `pkg/constant/app.go` | Service and module constants |
| Shared | `pkg/rabbitmq/rabbitmq.go` | RabbitMQ adapter |
| Infra | `components/infra/docker-compose.yml` | Infrastructure |
| Infra | `components/infra/rabbitmq/etc/definitions.json` | RabbitMQ topology |
| Tests | `tests/shared/client.go` | Manager API test client (package `shared`, imported as `e2eshared`) |
| Tests | `tests/shared/helpers.go` | Product-name generation, job polling helpers |
| Tests | `tests/shared/readyz.go` | `/readyz` probe helpers |
| Tests | `pkg/itestkit/` | Container orchestration kit (infra providers + e2ekit/metricskit/queuekit addons) |
| Tests | `tests/e2e/` | End-to-end suite (`//go:build e2e`) |
| Tests | `tests/chaos/` | Chaos suite (`//go:build chaos`, Toxiproxy) |
