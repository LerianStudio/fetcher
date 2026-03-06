# Lerian Fetcher: Enterprise-Grade Data Extraction Adapter

Lerian Fetcher is a centralized data extraction microservice designed to abstract and unify access to external data sources. It provides a secure, reliable, and scalable interface for Lerian products to connect, validate, and extract data from multiple database types.

## Why Fetcher?

- **Unified Data Access**: Single interface for extracting data from PostgreSQL, MySQL, Oracle, SQL Server, and MongoDB
- **Enterprise Security**: Password encryption, SSL/TLS support per database, message signing, and replay attack prevention
- **Developer-Friendly**: Clean REST API with comprehensive OpenAPI documentation and advanced filtering capabilities
- **Battle-Tested Reliability**: Circuit breaker pattern, connection pooling, and graceful error handling for production workloads

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

## Core Architecture

Lerian Fetcher is built as a cloud-native platform following Hexagonal Architecture and CQRS patterns:

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
| `GET` | `/health` | Health check |
| `GET` | `/version` | Version info |
| `GET` | `/swagger/*` | Swagger UI |

### API Reference & Testing

For hands-on API exploration and testing scenarios, the following resources are available:

- **[`components/manager/api/requests.http`](components/manager/api/requests.http)**: Ready-to-use HTTP request examples covering all API endpoints — useful for quick testing with VS Code REST Client, IntelliJ, or similar tools.
- **[`components/manager/api/swagger.yaml`](components/manager/api/swagger.yaml)**: Full OpenAPI specification for the Manager API, which can be imported into Postman, Insomnia, or any OpenAPI-compatible tool.
- **[`tests/e2e/`](tests/e2e/)**: End-to-end test suite covering connection management, data extraction across all supported databases, filtering, multi-datasource/multi-schema scenarios, schema validation, and error handling. These tests serve as practical usage examples and can be referenced to understand expected behaviors and edge cases.

### Technical Highlights

- **Hexagonal Architecture**: Clear separation between domain logic and external dependencies
- **CQRS Pattern**: Separate command and query responsibilities for optimized performance
- **Event-Driven Design**: RabbitMQ-based async processing for scalable job handling
- **Circuit Breaker**: Prevents cascading failures with configurable thresholds
- **Advanced Filtering**: 10 operators (eq, gt, gte, lt, lte, between, in, nin, ne, like)
- **Schema Discovery**: Automatic table/column detection across all database types
- **Message Signing**: HMAC-SHA256 signing with replay attack prevention
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

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.25.6+ (for development)
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

## About Lerian

Fetcher is developed by Lerian, a tech company founded in 2024, led by a team with a track record in developing ledger and core banking solutions. For any inquiries or support, please reach out to us at [contact@lerian.studio](mailto:contact@lerian.studio) or simply open a Discussion in our [GitHub repository](https://github.com/LerianStudio/fetcher/discussions).
