# Fetcher - Architecture Documentation

## Overview

**Fetcher** is a multi-database data extraction microservice designed to connect to various database types (MongoDB, PostgreSQL, Oracle, MySQL, SQL Server), extract data based on configurable mappings, and store results in distributed file storage. The system uses a distributed architecture with two main components: **Manager** (HTTP API for configuration) and **Worker** (queue-based job processor).

### Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23 |
| HTTP Framework | Fiber v2 |
| Database | MongoDB (metadata), External DBs (data) |
| Message Queue | RabbitMQ |
| File Storage | SeaweedFS |
| Observability | OpenTelemetry |
| Logging | Structured logging (lib-commons) |

### Main Features

1. **Database Connection Management** - Create, update, delete, and test database connections with SSL/TLS support and password encryption (AES-GCM)
2. **Job-Based Data Extraction** - Asynchronous job processing via RabbitMQ with advanced filtering
3. **Multi-Database Support** - MongoDB, PostgreSQL, Oracle, MySQL, SQL Server
4. **Multi-Tenancy** - Organization-based data isolation

---

## Architecture Pattern

### Clean Architecture with CQRS

The codebase follows **Clean Architecture** principles combined with **CQRS** (Command Query Responsibility Segregation).

```
┌─────────────────────────────────────────────────────────────────┐
│                     External Layer                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │  HTTP/REST  │  │  RabbitMQ   │  │      SeaweedFS          │ │
│  │  (Fiber)    │  │  Consumer   │  │   (File Storage)        │ │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘ │
└─────────┼────────────────┼─────────────────────┼───────────────┘
          │                │                     │
          ▼                ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Adapters Layer                               │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ HTTP Handlers │ RabbitMQ Adapters │ Repository Implementations││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
          │                │                     │
          ▼                ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Application Layer (Services)                   │
│  ┌───────────────────────┐  ┌─────────────────────────────────┐│
│  │    Command Services   │  │       Query Services            ││
│  │  - CreateConnection   │  │  - GetConnection                ││
│  │  - UpdateConnection   │  │  - ListConnections              ││
│  │  - DeleteConnection   │  │  - TestConnection               ││
│  │  - CreateJob          │  │  - GetJob, ListJobs             ││
│  └───────────────────────┘  └─────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
          │                                      │
          ▼                                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Domain Layer                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │     Models: Connection, Job, DataSource, SSLConfig          ││
│  │     Interfaces: Repository, DataSource, Cryptor             ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### CQRS Implementation

- **Command Services** (`/internal/services/command/`): Handle write operations (Create, Update, Delete)
- **Query Services** (`/internal/services/query/`): Handle read operations (Get, List, Test)

### Key Architectural Characteristics

1. **Dependency Inversion**: All dependencies point inward; outer layers depend on inner layers
2. **Interface Segregation**: Small, focused interfaces for repositories
3. **Domain Isolation**: Business logic isolated from infrastructure concerns
4. **Testability**: Extensive use of interfaces enables mocking

---

## System Architecture Diagram

```
                              ┌─────────────────────────────────────┐
                              │            Load Balancer            │
                              └──────────────────┬──────────────────┘
                                                 │
                                                 ▼
┌────────────────────────────────────────────────────────────────────────────────┐
│                                   MANAGER                                       │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                            HTTP Layer (Fiber)                            │  │
│  │  /v1/connections  │  /v1/jobs  │  /health  │  Middleware (CORS, Auth)   │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                        │
│                                        ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                         Service Layer (CQRS)                             │  │
│  │  ┌───────────────────────┐      ┌───────────────────────────────────┐   │  │
│  │  │    Command Services   │      │        Query Services              │   │  │
│  │  │  - CreateConnection   │      │  - GetConnection                   │   │  │
│  │  │  - UpdateConnection   │      │  - ListConnections                 │   │  │
│  │  │  - DeleteConnection   │      │  - TestConnection                  │   │  │
│  │  │  - CreateJob          │      │  - GetConnectionSchema             │   │  │
│  │  └───────────────────────┘      └───────────────────────────────────┘   │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                        │
└────────────────────────────────────────┼────────────────────────────────────────┘
                                         │
         ┌───────────────────────────────┼───────────────────────────────┐
         │                               │                               │
         ▼                               ▼                               ▼
┌─────────────────┐           ┌─────────────────┐           ┌─────────────────┐
│     MongoDB     │           │    RabbitMQ     │           │    SeaweedFS    │
│   (Metadata)    │           │    (Queues)     │           │  (File Store)   │
│  - connections  │           │  - job_queue    │           │  - JSON results │
│  - jobs         │           │  - notify_queue │           │                 │
└─────────────────┘           └────────┬────────┘           └─────────────────┘
                                       │                               ▲
                                       ▼                               │
┌────────────────────────────────────────────────────────────────────────────────┐
│                                   WORKER                                        │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                      RabbitMQ Consumer Layer                             │  │
│  │              Competing Consumers (configurable concurrency)              │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                        │
│                                        ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                          Service Layer                                   │  │
│  │  - ExtractData (orchestrates extraction)                                 │  │
│  │  - JobNotification (sends completion notifications)                      │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                        │
└────────────────────────────────────────┼────────────────────────────────────────┘
                                         │
         ┌───────────────────────────────┴───────────────────────────────┐
         │                               │                               │
         ▼                               ▼                               ▼
┌─────────────────┐           ┌─────────────────┐           ┌─────────────────┐
│   PostgreSQL    │           │     MongoDB     │           │     Oracle      │
│   DataSource    │           │   DataSource    │           │   DataSource    │
└─────────────────┘           └─────────────────┘           └─────────────────┘
         │                               │                               │
         ▼                               ▼                               ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           Customer Databases                                     │
│                 (External PostgreSQL, MongoDB, Oracle, MySQL, SQL Server)        │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## Package Structure

### Component: Manager (`/components/manager/`)

```
components/manager/
├── cmd/app/main.go                 # Application entry point
└── internal/
    ├── adapters/
    │   └── http/
    │       └── in/
    │           ├── connection.go   # Connection HTTP handlers
    │           ├── job.go          # Job HTTP handlers
    │           ├── middlewares.go  # CORS, auth, logging
    │           └── routes.go       # Route registration
    ├── bootstrap/
    │   ├── config.go              # Environment configuration
    │   ├── server.go              # HTTP server setup
    │   └── service.go             # Dependency injection
    └── services/
        ├── command/
        │   ├── create_connection.go
        │   ├── update_connection.go
        │   ├── delete_connection.go
        │   └── create_job.go
        └── query/
            ├── get_connection.go
            ├── list_connections.go
            ├── test_connection.go
            ├── get_job.go
            ├── list_jobs.go
            └── get_connection_schema.go
```

| Package | Responsibility |
|---------|----------------|
| `cmd/app` | Application bootstrap, graceful shutdown |
| `adapters/http/in` | HTTP request handling, validation, response formatting |
| `bootstrap` | Configuration loading, dependency wiring, server setup |
| `services/command` | Write operations with business logic |
| `services/query` | Read operations with pagination, filtering |

### Component: Worker (`/components/worker/`)

```
components/worker/
├── cmd/app/main.go                 # Worker entry point
└── internal/
    ├── adapters/
    │   └── rabbitmq/
    │       ├── consumer.rabbitmq.go  # Queue consumer
    │       └── publisher.rabbitmq.go # Queue publisher
    ├── bootstrap/
    │   ├── config.go               # Environment configuration
    │   ├── consumer.go             # Consumer setup
    │   └── service.go              # Dependency injection
    └── services/
        ├── service.go              # Service interface
        ├── extract-data.go         # Data extraction logic
        ├── extract_crm_data.go     # CRM-specific extraction
        └── job_notification.go     # Job status notifications
```

| Package | Responsibility |
|---------|----------------|
| `cmd/app` | Worker bootstrap, signal handling |
| `adapters/rabbitmq` | RabbitMQ consumer/publisher wrappers |
| `bootstrap` | Worker configuration, consumer wiring |
| `services` | Data extraction business logic |

### Shared Package (`/pkg/`)

```
pkg/
├── constant/
│   ├── app.go                # Application constants
│   ├── errors.go             # Error codes (FET-0001 to FET-0035)
│   ├── mongo.go              # MongoDB collection names
│   ├── pagination.go         # Pagination constants
│   └── datasource-config.go  # DB connection constants
├── crypto/
│   └── crypto.go             # AES-GCM encryption service
├── datasource/
│   └── datasource-factory.go # DataSource factory pattern
├── model/
│   ├── connection.go         # Connection domain entity
│   ├── pagination.go         # Pagination models
│   ├── job/
│   │   └── job_queue.go      # Job queue message model
│   └── datasource/
│       ├── datasource-config.go    # Base DataSource interface
│       ├── mongodb/                # MongoDB-specific config
│       ├── postgres/               # PostgreSQL-specific config
│       ├── oracle/                 # Oracle-specific config
│       ├── mysql/                  # MySQL-specific config
│       └── sqlserver/              # SQL Server-specific config
├── mongodb/
│   ├── connection/
│   │   ├── connection.mongodb.go   # Connection repository
│   │   └── connection.go           # MongoDB model mapping
│   ├── job/
│   │   ├── job.mongodb.go          # Job repository
│   │   └── job.go                  # Job model/entity
│   └── datasource.mongodb.go       # MongoDB DataSource impl
├── postgres/
│   ├── postgres.go                 # Connection management
│   └── datasource.postgres.go      # PostgreSQL DataSource impl
├── oracle/
│   ├── oracle.go                   # Connection management
│   └── datasource.oracle.go        # Oracle DataSource impl
├── mysql/
│   ├── mysql.go                    # Connection management
│   └── datasource.mysql.go         # MySQL DataSource impl
├── sqlserver/
│   ├── sqlserver.go                # Connection management
│   └── datasource.sqlserver.go     # SQL Server DataSource impl
├── rabbitmq/
│   └── rabbitmq.go                 # RabbitMQ adapter
├── seaweedfs/
│   ├── seaweedfs.go                # SeaweedFS client
│   └── external/
│       └── external-data.go        # External data repository
├── net/http/
│   ├── response.go                 # HTTP response utilities
│   ├── errors.go                   # HTTP error handling
│   └── http-utils.go               # Request parsing
├── errors.go                       # Custom error types
├── context.go                      # Context utilities
└── utils.go                        # General utilities
```

---

## Core Domain

### Domain Entities

#### Connection

Represents a database connection configuration with encryption support.

**Key Fields:**
- `ID`, `OrganizationID` - Unique identifiers
- `ConfigName` - Unique name within organization
- `Type` - Database type (mongodb, postgresql, oracle, mysql, sqlserver)
- `Host`, `Port`, `DatabaseName`, `Username` - Connection details
- `PasswordEncrypted` - AES-GCM encrypted password
- `SSL` - Optional SSL configuration
- Timestamps: `CreatedAt`, `UpdatedAt`, `DeletedAt` (soft delete)

#### Job

Represents a data extraction job with status tracking.

**Key Fields:**
- `ID`, `OrganizationID` - Unique identifiers
- `MappedFields` - Map of table/collection to fields
- `Filters` - Query filters
- `Status` - Job status (pending, processing, completed, failed)
- `ResultPath` - SeaweedFS path for results
- Timestamps: `CreatedAt`, `CompletedAt`

**Job Statuses:**
- `pending` - Job created, waiting for processing
- `processing` - Worker actively processing
- `completed` - Successfully finished
- `failed` - Processing failed

#### DataSource Interface

Common interface for all database connections.

```go
type DataSource interface {
    GetConfig() DataSourceConfig
    Connect(ctx context.Context, logger log.Logger) error
    Close(ctx context.Context) error
    GetType() string
    Query(ctx context.Context, tables map[string][]string,
          filters map[string]map[string]FilterCondition,
          logger log.Logger) (map[string][]map[string]any, error)
}
```

---

## External Integrations

### MongoDB (Internal)

**Purpose:** Stores application metadata (connections, jobs)

**Collections:**
- `connection` - Database connection configurations
- `job` - Extraction job records

### RabbitMQ

**Purpose:** Asynchronous job processing

**Queues:**
- Job creation queue (Manager -> Worker)
- Job notification queue (Worker -> External)

**Features:** Automatic reconnection, prefetch count, message headers for tracing

### SeaweedFS

**Purpose:** Distributed file storage for extraction results

**Operations:** UploadFile, DownloadFile, DeleteFile, HealthCheck

### External Data Sources

**Factory Pattern:** Creates appropriate DataSource based on connection type

**Supported Databases:**

| Database | Query Builder |
|----------|---------------|
| MongoDB | Native driver + aggregation |
| PostgreSQL | Squirrel |
| Oracle | Native driver |
| MySQL | Native driver |
| SQL Server | Native driver |

---

## Data Flow

### Create and Process Job

```
Client        Manager         RabbitMQ        Worker         DataSource    SeaweedFS
  │              │               │               │               │              │
  │ POST /jobs   │               │               │               │              │
  │─────────────>│               │               │               │              │
  │              │               │               │               │              │
  │              │ Create job    │               │               │              │
  │              │ (pending)     │               │               │              │
  │              │               │               │               │              │
  │              │ Publish msg   │               │               │              │
  │              │──────────────>│               │               │              │
  │              │               │               │               │              │
  │ 201 (jobId)  │               │               │               │              │
  │<─────────────│               │               │               │              │
  │              │               │               │               │              │
  │              │               │ Consume msg   │               │              │
  │              │               │──────────────>│               │              │
  │              │               │               │               │              │
  │              │               │               │ Connect       │              │
  │              │               │               │──────────────>│              │
  │              │               │               │               │              │
  │              │               │               │ Query data    │              │
  │              │               │               │──────────────>│              │
  │              │               │               │               │              │
  │              │               │               │    Results    │              │
  │              │               │               │<──────────────│              │
  │              │               │               │               │              │
  │              │               │               │         Upload results       │
  │              │               │               │─────────────────────────────>│
  │              │               │               │               │              │
  │              │               │               │ Update job    │              │
  │              │               │               │ (completed)   │              │
  │              │               │               │               │              │
  │              │               │ Publish notification          │              │
  │              │               │<──────────────│               │              │
```

### Data Extraction Detail

```
Worker Service
       │
       ▼
┌──────────────────┐
│ Get Connection   │ ──> MongoDB (fetch connection config)
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Decrypt Password │ ──> Crypto Service (AES-GCM)
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Create DataSource│ ──> DataSource Factory
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Get Schema       │ ──> External DB (discover tables/columns)
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Validate Fields  │ ──> Check requested fields exist
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Execute Query    │ ──> External DB (with filters)
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Transform Results│ ──> Convert to JSON
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Upload to Storage│ ──> SeaweedFS
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Update Job Status│ ──> MongoDB (completed)
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Send Notification│ ──> RabbitMQ (notify downstream)
└──────────────────┘
```

---

## Configuration

### Manager Environment Variables

| Variable | Description |
|----------|-------------|
| `HTTP_PORT` | HTTP server port |
| `MONGO_URI` | MongoDB connection string |
| `MONGO_DATABASE` | MongoDB database name |
| `RABBITMQ_URI` | RabbitMQ connection string |
| `ENCRYPTION_KEY` | Base64-encoded AES key (32 bytes) |
| `ENCRYPTION_KEY_VERSION` | Key version for rotation |

### Worker Environment Variables

| Variable | Description |
|----------|-------------|
| `RABBITMQ_URI` | RabbitMQ connection string |
| `RABBITMQ_QUEUE_NAME` | Queue to consume from |
| `RABBITMQ_NUM_WORKERS` | Consumer concurrency |
| `SEAWEEDFS_URL` | SeaweedFS filer URL |
| `SEAWEEDFS_BUCKET` | Storage bucket name |

---

## HTTP API Routes

```
GET    /v1/connections                    # List connections
POST   /v1/connections                    # Create connection
GET    /v1/connections/:id                # Get connection
PUT    /v1/connections/:id                # Update connection
DELETE /v1/connections/:id                # Delete connection
GET    /v1/connections/:id/test           # Test connection
GET    /v1/connections/:config_name/schema # Get schema

GET    /v1/jobs                           # List jobs
POST   /v1/jobs                           # Create job
GET    /v1/jobs/:id                       # Get job
```

---

## Key Design Decisions

### 1. CQRS Pattern

**Decision:** Separate command and query services

**Rationale:**
- Clear separation of concerns
- Different scaling needs for reads vs writes
- Easier to maintain and test

### 2. Password Encryption (AES-256-GCM)

**Decision:** AES-256-GCM with key versioning

**Rationale:**
- Industry-standard encryption
- Key rotation support without re-encrypting existing data
- Nonce prepended to ciphertext for storage simplicity

### 3. DataSource Factory Pattern

**Decision:** Factory for creating database connections

**Rationale:**
- Single point of database type detection
- Consistent connection handling across types
- Easy to add new database types

### 4. Job-Based Async Processing

**Decision:** Asynchronous job processing via RabbitMQ

**Rationale:**
- Long-running queries don't block HTTP responses
- Horizontal scalability of workers
- Reliable message delivery with acknowledgments

### 5. Soft Delete Pattern

**Decision:** Use `deleted_at` timestamp instead of physical deletion

**Rationale:**
- Audit trail preservation
- Accidental deletion recovery
- Referential integrity preservation

### 6. Multi-Tenancy via Organization ID

**Decision:** Organization ID in all entities and queries

**Rationale:**
- Data isolation between organizations
- Shared infrastructure
- Simplified access control

### 7. Repository Interface Pattern

**Decision:** Interface-based repositories with mock generation

**Rationale:**
- Testability without real databases
- Swappable implementations
- Clear contracts

### 8. Context-Based Tracing

**Decision:** OpenTelemetry tracing through context

**Rationale:**
- Distributed tracing across services
- Request correlation
- Performance monitoring

### 9. Timeout Management

**Decision:** Context-based timeouts for all external operations

**Constants:**
- `QueryTimeoutMedium` = 30 seconds
- `QueryTimeoutSlow` = 60 seconds
- `SchemaDiscoveryTimeout` = 45 seconds
- `ConnectionTimeout` = 10 seconds

### 10. Standardized Error Codes

**Decision:** Application-specific error codes (FET-0001 to FET-0035)

**Rationale:**
- Consistent error handling
- Client-friendly error messages
- Debugging assistance

---

## Testing Strategy

### Test Organization

Tests are co-located with source files using the `*_test.go` convention.

### Testing Patterns

1. **In-Memory MongoDB (memongo)** - Repository tests with isolated environment
2. **Mock Generation (mockgen)** - Interface-based mocking
3. **Table-Driven Tests** - Comprehensive edge case coverage
4. **Test Fixtures** - Reusable test data factories

### Test Categories

| Category | Files | Description |
|----------|-------|-------------|
| Unit | `*_test.go` | Isolated component tests |
| Integration | `*_integration_test.go` | External service integration |
| Repository | `*.mongodb_test.go` | Database operation tests |

---

## Future Development Guidelines

### Adding a New Database Type

1. Create new package under `/pkg/{database}/`
2. Implement `DataSource` interface
3. Add configuration struct in `/pkg/model/datasource/{database}/`
4. Update factory in `/pkg/datasource/datasource-factory.go`
5. Add new type constant in `/pkg/model/connection.go`

### Adding a New API Endpoint

1. Add handler in `/components/manager/internal/adapters/http/in/`
2. Register route in `routes.go`
3. Create service in `/services/command/` or `/services/query/`
4. Add necessary repository methods if needed

### Adding a New Worker Task

1. Create new service method in `/components/worker/internal/services/`
2. Register handler in consumer setup
3. Define message structure in `/pkg/model/`

---

## Key Files Reference

| Category | Key Files |
|----------|-----------|
| **Entry Points** | `/components/manager/cmd/app/main.go`, `/components/worker/cmd/app/main.go` |
| **HTTP Layer** | `/components/manager/internal/adapters/http/in/*.go` |
| **Services** | `/components/*/internal/services/**/*.go` |
| **Domain Models** | `/pkg/model/*.go`, `/pkg/mongodb/job/job.go` |
| **Repositories** | `/pkg/mongodb/**/*.mongodb.go` |
| **DataSources** | `/pkg/*/datasource.*.go` |
| **Infrastructure** | `/pkg/rabbitmq/*.go`, `/pkg/seaweedfs/*.go` |
| **Configuration** | `/components/*/internal/bootstrap/config.go` |
| **Error Codes** | `/pkg/constant/errors.go` |
