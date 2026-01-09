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
   - Encrypts and stores results in SeaweedFS
   - Publishes job completion/failure notifications
   - Configurable worker concurrency (default: 5)

3. **Infrastructure Layer**: Containerized infrastructure services.

   - MongoDB for primary metadata storage
   - RabbitMQ for message queuing with DLQ support
   - SeaweedFS for distributed file storage
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

#### Fetcher Jobs

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

### Technical Highlights

- **Hexagonal Architecture**: Clear separation between domain logic and external dependencies
- **CQRS Pattern**: Separate command and query responsibilities for optimized performance
- **Event-Driven Design**: RabbitMQ-based async processing for scalable job handling
- **Circuit Breaker**: Prevents cascading failures with configurable thresholds
- **Advanced Filtering**: 10 operators (eq, gt, gte, lt, lte, between, in, nin, ne, like)
- **Schema Discovery**: Automatic table/column detection across all database types
- **Message Signing**: HMAC-SHA256 signing with replay attack prevention
- **OpenTelemetry**: Distributed tracing and metrics for comprehensive observability

### Data Extraction Capabilities

- **Multi-Datasource**: Extract from multiple databases in a single job
- **Multi-Table**: Query multiple tables/collections per datasource
- **Multi-Schema**: Support for PostgreSQL schemas, Oracle owners, SQL Server schemas
- **Field Projection**: Select specific fields or use `["*"]` for all fields
- **JSON/BSON Parsing**: Automatic parsing of JSON fields in relational databases
- **Deduplication**: 5-minute window for duplicate job detection
- **Result Storage**: Encrypted results stored in SeaweedFS with configurable TTL

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.25.3+ (for development)
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

3. **Start all services:**
   ```bash
   make up
   ```

4. **Access the API:**
   - REST API: `http://localhost:4006`
   - Swagger UI: `http://localhost:4006/swagger/index.html`
   - RabbitMQ Management: `http://localhost:3008`

### Development Commands

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

## About Lerian

Fetcher is developed by Lerian, a tech company founded in 2024, led by a team with a track record in developing ledger and core banking solutions. For any inquiries or support, please reach out to us at [contact@lerian.studio](mailto:contact@lerian.studio) or simply open a Discussion in our [GitHub repository](https://github.com/LerianStudio/fetcher/discussions).
