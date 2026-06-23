# Lerian Fetcher: Enterprise-Grade Data Extraction Adapter

Lerian Fetcher is a centralized data extraction platform designed to abstract and unify access to external data sources. It provides a secure, reliable, and scalable interface for Lerian products to connect, validate, and extract data from multiple database types — available both as standalone services and as an **embedded runtime engine** that host applications import in-process.

## Why Fetcher?

- **Unified Data Access**: Single interface for extracting data from PostgreSQL, MySQL, Oracle, SQL Server, and MongoDB
- **Run It Your Way**: Deploy as standalone Manager + Worker services, or embed the **Fetcher Engine** (`pkg/engine`) directly in your application — no separate service, queue, or storage stack required
- **Enterprise Security**: Password encryption, SSL/TLS support per database, message signing with replay protection, and SSRF host validation
- **Developer-Friendly**: Clean REST API with comprehensive OpenAPI documentation and advanced filtering capabilities
- **Battle-Tested Reliability**: Circuit breaker pattern, connection pooling, readiness probing, and graceful error handling for production workloads

## Problem Statement

Development teams building Lerian products (Reports, Matcher, and other services) were implementing isolated, redundant data access logic for each product. This led to:

- Duplicated code across multiple codebases
- Inconsistent security practices
- Increased maintenance overhead
- Fragmented monitoring and observability

## Solution

Fetcher centralizes all external data source access into a single, secure service that provides:

1. **Connection Management**: Store, validate, and test database connections with encrypted credentials
2. **Schema Discovery**: Automatic detection of tables, columns, and data types across all supported databases
3. **Data Extraction**: Unified query interface with advanced filtering, pagination, and multi-table support
4. **Job Orchestration**: Async processing with deduplication, status tracking, and event notifications

## Embedded Runtime Engine

Fetcher is more than a service — it is an importable **Engine** that host applications such as Matcher and Reporter run in-process, rather than operating a separate Fetcher deployment. The Engine moves the canonical capabilities of data extraction — connection lifecycle, schema discovery and validation, query planning, extraction execution, result handling, error taxonomy, limits, and tenant-safety — behind a single composition root at `pkg/engine`, without the service shell.

The design follows a three-layer model:

| Layer | Package | Owns |
|-------|---------|------|
| **Engine core** | `pkg/engine` | The *rules* of extraction. Infrastructure-free: depends only on host-provided port interfaces. A build-enforced boundary test forbids it from importing HTTP frameworks, message brokers, database drivers, object storage SDKs, or tenant-runtime middleware. |
| **Compatibility adapters** | `pkg/enginecompat/*` | Bridges between the Engine's ports and Fetcher's real infrastructure (MongoDB, Redis, datasource drivers), preserving the standalone services' exact behavior. |
| **Host applications** | Manager, Worker, or external products | The operational *shell*: auth, license enforcement, HTTP routes, queues, state stores, storage, telemetry, process lifecycle. |

The guiding principle is **the Engine owns what makes Fetcher *Fetcher*; host applications own *how* Fetcher runs**. The standalone Manager and Worker are themselves hosts that run over the Engine — so every product, internal or external, shares one canonical owner of datasource and extraction behavior.

### Embedding the Engine in Your Application

Embedding the Engine is three steps: **import it, provide the ports it needs, construct it with `engine.New`.** No infrastructure ships with the import — you wire the parts your host actually uses.

#### 1. Install

```bash
go get github.com/LerianStudio/fetcher/pkg/engine
```

> The engine is a **distinct Go module** from the Fetcher services (`github.com/LerianStudio/fetcher/v2`).
> It carries **zero third-party dependencies** and is versioned on its own `v1` line with
> path-prefixed tags (`pkg/engine/vX.Y.Z`). Importing it inherits none of Fetcher's service
> dependencies. See [`docs/RELEASING.md`](docs/RELEASING.md) for the dual-module release scheme.

#### 2. Provide the ports

The Engine depends only on host-provided interfaces. Only one is always required; the rest are opt-in and degrade gracefully when absent.

| Port | Required? | Without it |
|------|-----------|------------|
| `ConnectorRegistry` | **Always** | `engine.New` fails — extraction is impossible |
| `CredentialProtector` | Only with `WithEncryptedPersistence(true)` | Credentials are not encrypted at rest |
| `ConnectionStore` | Optional | Connection CRUD returns a "not configured" error |
| `SchemaCache` | Optional | Schema is always discovered live |
| `ResultSink` | Optional | Store mode unavailable; extraction runs in Direct mode |
| `ExecutionStore` | Optional | No durable execution-state tracking |
| `ActiveExecutionChecker`, `Observability` | Optional | Conflict-gating/tracing become no-ops |

For tests and quick starts, the **`pkg/engine/memory`** harness satisfies every port in-memory — no MongoDB, Redis, RabbitMQ, or storage required.

#### 3. Construct, plan, execute

This example is fully self-contained (it uses the in-memory harness) and mirrors the real engine tests:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/engine/memory"
)

func main() {
	ctx := context.Background()

	// Provide ports. In production these are your real adapters (see pkg/enginecompat);
	// here the in-memory harness stands in so the example runs with zero infrastructure.
	store := memory.NewConnectionStore()
	registry := memory.NewConnectorRegistry()

	// Construct the Engine. WithConnectorRegistry is the only required option.
	eng, err := engine.New(
		engine.WithConnectorRegistry(registry),
		engine.WithConnectionStore(store),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Every operation is scoped to a tenant — the sole isolation dimension.
	tenant, err := engine.NewTenantContext("tenant-123")
	if err != nil {
		log.Fatal(err)
	}

	// Register a connector for the datasource type, then persist a connection.
	conn := memory.NewTemplateConnector(memory.ConnectorBehavior{
		Schema: engine.SchemaSnapshot{
			ConfigName: "pg-main",
			Tables:     []engine.TableSnapshot{{Name: "public.users", Fields: []string{"id", "email"}}},
		},
		Rows: map[string][]map[string]any{
			"public.users": {{"id": 1, "email": "a@example.com"}},
		},
	})
	registry.Register("postgres", memory.NewConnectorFactory(conn))

	if _, err = eng.CreateConnection(ctx, tenant, engine.NewConnectionInput(engine.ConnectionInputParams{
		ConfigName: "pg-main",
		Type:       "postgres",
		Host:       "localhost",
		Port:       5432,
	})); err != nil {
		log.Fatal(err)
	}

	// Plan validates the request against the live schema and enforces limits.
	plan, err := eng.PlanExtraction(ctx, tenant, engine.ExtractionRequest{
		MappedFields: map[string]engine.FieldSelection{
			"pg-main": {"public.users": {"id", "email"}},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Execute. With no ResultSink wired, the Engine runs in Direct mode and returns
	// inline JSON bytes plus a SHA-256 integrity digest.
	result, err := eng.ExecuteExtraction(ctx, plan)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("rows=%d bytes=%d\n", result.Direct.RowCount, len(result.Direct.Data))
}
```

#### Moving to production

Swap the memory harness for real adapters. Fetcher's own services are the reference implementation — read how they wire the Engine via `pkg/enginecompat`:

- **Connection CRUD** — [`components/manager/internal/bootstrap/connection_engine.go`](components/manager/internal/bootstrap/connection_engine.go)
- **Schema discovery + caching** — [`components/manager/internal/bootstrap/schema_engine.go`](components/manager/internal/bootstrap/schema_engine.go)
- **Plan + execute extraction** — [`components/worker/internal/bootstrap/extraction_engine.go`](components/worker/internal/bootstrap/extraction_engine.go)

For the full port reference and the build-enforced import boundary, see [`docs/PROJECT_RULES.md`](docs/PROJECT_RULES.md) § *engine (`pkg/engine/`)*.

## Core Architecture

Lerian Fetcher is built as a cloud-native platform following Hexagonal Architecture and CQRS patterns. The standalone Manager and Worker services run *over* the embedded Engine described above:

### Services

1. **Manager Service**: HTTP API server for connection and job management.

   - Implements hexagonal architecture with CQRS pattern
   - RESTful API with OpenAPI documentation
   - MongoDB for connection and job metadata storage
   - RabbitMQ for async job dispatch
   - Connection testing and validation before job execution

2. **Worker Service**: Asynchronous job processor for data extraction.

   - Consumes jobs from RabbitMQ queue
   - Extracts data from configured external databases
   - Encrypts and stores results in configurable object storage (SeaweedFS or S3-compatible)
   - Publishes job completion/failure notifications
   - Configurable worker concurrency (default: 5)

3. **Infrastructure Layer**: Containerized infrastructure services.

   - MongoDB for primary metadata storage
   - RabbitMQ for message queuing with DLQ support
   - SeaweedFS for distributed file storage (default) or any S3-compatible service (AWS S3, MinIO)
   - Valkey/Redis for caching
   - KEDA for Kubernetes event-driven autoscaling

### Supported Databases

| Database | Versions | Key Features |
|----------|----------|--------------|
| **PostgreSQL** | 12+ | Multi-schema support, JSONB auto-parsing, connection pooling |
| **MySQL** | 8.0+ | JSON field auto-parsing, multi-table extraction, connection pooling |
| **Oracle** | 19c+ | Owner/schema namespaces, multi-table extraction, connection pooling |
| **SQL Server** | 2019+ | Multi-schema support, table filter matching, connection pooling |
| **MongoDB** | 7.0+ | Schemaless inference, statistical sampling, aggregation pipelines |

### API Endpoints

#### Connection Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/management/connections` | Create new database connection (encrypted password) |
| `GET` | `/v1/management/connections` | List connections with pagination/filters |
| `GET` | `/v1/management/connections/{id}` | Get connection details by ID |
| `POST` | `/v1/management/connections/{id}/test` | Test connection and return latency metrics |
| `PATCH` | `/v1/management/connections/{id}` | Partial update (409 if active jobs) |
| `DELETE` | `/v1/management/connections/{id}` | Soft delete (409 if active jobs) |
| `POST` | `/v1/management/connections/validate-schema` | Validate tables/fields exist in datasources |

#### Fetcher  Jobs

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/fetcher` | Create data extraction job (202 Accepted / 200 if duplicate) |
| `GET` | `/v1/fetcher/{id}` | Get job status and details |

#### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Liveness check |
| `GET` | `/readyz` | Readiness check — parallel dependency probes (MongoDB, RabbitMQ, Redis; S3 on Worker), returns 503 while draining on SIGTERM |
| `GET` | `/version` | Version info |
| `GET` | `/swagger/*` | Swagger UI |

### API Reference & Testing

For hands-on API exploration and testing scenarios, the following resources are available:

- **[`components/manager/api/requests.http`](components/manager/api/requests.http)**: Ready-to-use HTTP request examples covering all API endpoints — useful for quick testing with VS Code REST Client, IntelliJ, or similar tools.
- **[`components/manager/api/swagger.yaml`](components/manager/api/swagger.yaml)**: Full OpenAPI specification for the Manager API, which can be imported into Postman, Insomnia, or any OpenAPI-compatible tool.
- **[`tests/e2e/`](tests/e2e/)**: End-to-end test suite covering connection management, data extraction across all supported databases, filtering, multi-datasource/multi-schema scenarios, schema validation, and error handling. These tests serve as practical usage examples and can be referenced to understand expected behaviors and edge cases.

### Technical Highlights

- **Embedded Runtime Engine**: Infrastructure-free, importable extraction core (`pkg/engine`) with a build-enforced dependency boundary
- **Hexagonal Architecture**: Clear separation between domain logic and external dependencies
- **CQRS Pattern**: Separate command and query responsibilities for optimized performance
- **Event-Driven Design**: RabbitMQ-based async processing for scalable job handling
- **Circuit Breaker**: Prevents cascading failures with configurable thresholds
- **Advanced Filtering**: 10 operators (eq, gt, gte, lt, lte, between, in, nin, ne, like)
- **Schema Discovery**: Automatic table/column detection across all database types
- **Message Signing**: HMAC-SHA256 signing bound to tenant + route (exchange/routing key) to prevent cross-tenant and cross-route replay
- **SSRF Protection**: Connection host validation against private/metadata address ranges (multi-tenant deployments)
- **Readiness Probing**: `/readyz` with parallel dependency checks, SaaS TLS enforcement, and SIGTERM drain coupling
- **Multi-Tenant Support**: Database-per-tenant isolation via JWT-based tenant context, with zero overhead when disabled
- **OpenTelemetry**: Distributed tracing and metrics for comprehensive observability

### Data Extraction Capabilities

- **Multi-Datasource**: Extract from multiple databases in a single job
- **Multi-Table**: Query multiple tables/collections per datasource
- **Multi-Schema**: Support for PostgreSQL schemas, Oracle owners, SQL Server schemas
- **Field Projection**: Select specific fields or use `["*"]` for all fields
- **JSON/BSON Parsing**: Automatic parsing of JSON fields in relational databases
- **Deduplication**: 5-minute window for duplicate job detection
- **Result Storage**: Encrypted results stored in pluggable object storage (SeaweedFS or S3-compatible) with configurable TTL

### Worker Job Event Streaming

Worker startup fails closed unless lib-streaming is enabled for mandatory `job.completed` and `job.failed` notifications:

| Variable | Description | Default |
|----------|-------------|---------|
| `STREAMING_ENABLED` | Must be `true` for Worker job notifications | `false` in lib-streaming; Fetcher Worker requires `true` |
| `STREAMING_BROKERS` | Kafka/Redpanda bootstrap servers used by lib-streaming config validation | `localhost:9092` |
| `STREAMING_CLOUDEVENTS_SOURCE` | CloudEvents source for Fetcher Worker events | - |
| `RABBITMQ_JOB_EVENTS_EXCHANGE` | RabbitMQ exchange used by the streaming RabbitMQ route target | `fetcher.job.events` |
| `RABBITMQ_ALLOW_LEGACY_BODY_SIGNATURE_FALLBACK` | Temporary migration flag for pre-envelope body-only HMAC signatures | `false` |

Single-tenant deployments emit with stable tenant ID `single-tenant`; multi-tenant deployments require tenant context from the consumer before emitting.

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go toolchain declared in `go.mod` (for development)
- Make

### Quick Start

1. **Clone the Repository:**
   ```bash
   git clone https://github.com/LerianStudio/fetcher.git
   cd fetcher
   ```

2. **Setup environment variables:**
   ```bash
   make set-env
   ```

3. **Generate the master encryption key:**
   ```bash
   make generate-master-key
   ```
   Copy the generated key and set it as `APP_ENC_KEY` in both `components/manager/.env` and `components/worker/.env`. This key is **required** — the services will not start without it. See [Security](#security) for details.

4. **Start all services:**
   ```bash
   make up
   ```

5. **Access the API:**
   - REST API: `http://localhost:4006`
   - Swagger UI: `http://localhost:4006/swagger/index.html`
   - RabbitMQ Management: `http://localhost:3008`

### Security

Fetcher uses a single master key (`APP_ENC_KEY`) to derive three cryptographically independent keys via HKDF (RFC 5869). This means you only need to manage one secret, but the system internally separates concerns:

| Derived Key | Purpose |
|-------------|---------|
| **Credential Key** | AES-256-GCM encryption of database passwords stored in MongoDB |
| **Internal HMAC Key** | HMAC-SHA256 signing of RabbitMQ messages between Manager and Worker, preventing message tampering |
| **External HMAC Key** | HMAC-SHA256 signing of extracted data documents, enabling consumers to verify authenticity |

#### Generating the Master Key

```bash
make generate-master-key
```

This produces a cryptographically secure 32-byte key encoded in base64. Set it as `APP_ENC_KEY` in the `.env` files of both Manager and Worker. Both services **must** use the same key — the Worker needs it to decrypt connection credentials and verify internal message signatures.

The `APP_ENC_KEY_VERSION` variable (default: `1`) tracks key rotations. Increment it when rotating keys; the system uses it to identify which key version encrypted each credential.

#### Deriving the External HMAC Key

External consumers that need to verify document signatures can derive the external HMAC key from the master key:

```bash
make derive-key KEY="<your-base64-master-key>"
```

This outputs a hex-encoded HMAC key that consumers use to verify HMAC-SHA256 signatures on extracted data. See [scripts/crypto/derive-key/verification-guide.md](scripts/crypto/derive-key/verification-guide.md) for the full verification protocol.

### Multi-Tenant Support

Fetcher supports multi-tenant deployments with database-per-tenant isolation. When enabled, each tenant gets isolated MongoDB databases, Redis key namespaces, and S3 object paths.

| Variable | Description | Default |
|----------|-------------|---------|
| `MULTI_TENANT_ENABLED` | Enable multi-tenant mode | `false` |
| `MULTI_TENANT_URL` | Tenant Manager service URL | - |
| `MULTI_TENANT_MAX_TENANT_POOLS` | Max concurrent tenant connection pools | `100` |
| `MULTI_TENANT_IDLE_TIMEOUT_SEC` | Idle tenant connection timeout | `300` |
| `MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD` | Circuit breaker failure threshold | `5` |
| `MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC` | Circuit breaker reset timeout | `30` |

When `MULTI_TENANT_ENABLED=false` (default), all multi-tenant code paths are bypassed with zero performance impact. See `docs/multi-tenant-guide.md` for activation instructions.

### Internal Datasource env vars

`DATASOURCE_{NAME}_*` env vars configure internal datasources (e.g. `midaz_onboarding`) loaded by `pkg/resolver/env_loader.go`.

| Var                          | Required | Example                                                          |
|------------------------------|----------|------------------------------------------------------------------|
| `DATASOURCE_{N}_CONFIG_NAME` | yes      | `midaz_onboarding`                                               |
| `DATASOURCE_{N}_TYPE`        | yes      | `postgresql` / `mysql` / `oracle` / `mongodb` / `sql_server`     |
| `DATASOURCE_{N}_HOST`        | yes      | `db.internal.example.com`                                        |
| `DATASOURCE_{N}_PORT`        | yes      | `5432`                                                           |
| `DATASOURCE_{N}_DATABASE`    | yes      | `onboarding`                                                     |
| `DATASOURCE_{N}_USER`        | yes      | `midaz`                                                          |
| `DATASOURCE_{N}_PASSWORD`    | yes      | from secret manager                                              |
| `DATASOURCE_{N}_OPTIONS`     | no       | `authSource=admin&directConnection=true` (mongodb only)          |
| `DATASOURCE_{N}_SSLMODE`     | no       | `require` (postgres); see PROJECT_RULES                          |

See `docs/PROJECT_RULES.md` § Internal datasource SSL/TLS env vars for behavior. For TLS to a managed database, use `_SSLMODE` — that is the validated path. Custom CA / client certificate plumbing for the internal-datasource path is not implemented yet (the driver consumes only `_SSLMODE`); track follow-up in `tasks/fetcher.md`.

## About Lerian

Fetcher is developed by Lerian, a tech company founded in 2024, led by a team with a track record in developing ledger and core banking solutions. For any inquiries or support, please reach out to us at [contact@lerian.studio](mailto:contact@lerian.studio) or simply open a Discussion in our [GitHub repository](https://github.com/LerianStudio/fetcher/discussions).
