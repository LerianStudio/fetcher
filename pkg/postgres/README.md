# PostgreSQL Datasource

PostgreSQL datasource implementation for the Fetcher service, providing data extraction and schema discovery capabilities.

## Overview

### Purpose
This datasource enables connection, querying, and schema discovery for PostgreSQL databases. It implements the `DataSource` interface and provides advanced filtering, field validation, and automatic JSON field parsing.

### Supported Versions
- PostgreSQL 12+
- Compatible with PostgreSQL-compatible databases (e.g., Amazon Aurora PostgreSQL, Google Cloud SQL)

### External Dependencies
| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/jackc/pgx/v5` | v5.7.6 | PostgreSQL driver (registered as "pgx") |
| `github.com/Masterminds/squirrel` | v1.5.4 | SQL query builder |
| `github.com/lib/pq` | - | Array type handling for queries |

## Architecture

### Component Diagram

```mermaid
graph TB
    subgraph "pkg/model/datasource/postgres"
        DSConfig[DataSourceConfigPostgres]
    end

    subgraph "pkg/postgres"
        Conn[Connection]
        Repo[ExternalDataSource]
        RepoInterface[Repository Interface]
    end

    subgraph "pkg/datasource"
        Factory[datasource-factory.go]
        SSLMode[sslmode/postgresql.go]
    end

    subgraph "External"
        PGX[pgx/v5 Driver]
        DB[(PostgreSQL)]
    end

    Factory -->|creates| DSConfig
    Factory -->|validates| SSLMode
    DSConfig -->|contains| Conn
    DSConfig -->|contains| Repo
    Repo -->|implements| RepoInterface
    Repo -->|uses| Conn
    Conn -->|uses| PGX
    PGX -->|connects| DB
```

### Data Flow

```mermaid
sequenceDiagram
    participant Client
    participant DSConfig as DataSourceConfigPostgres
    participant Repo as ExternalDataSource
    participant Conn as Connection
    participant DB as PostgreSQL

    Client->>DSConfig: Query(ctx, tables, filters, logger)
    DSConfig->>Repo: GetDatabaseSchema(ctx, schemas)
    Repo->>Conn: GetDB()
    Conn->>DB: INFORMATION_SCHEMA queries
    DB-->>Repo: Schema metadata

    loop For each table
        DSConfig->>Repo: QueryWithAdvancedFilters(ctx, schema, table, fields, filter)
        Repo->>Repo: ValidateTableAndFields()
        Repo->>Repo: buildAdvancedFilters()
        Repo->>DB: SELECT query with placeholders
        DB-->>Repo: Result rows
        Repo->>Repo: parseJSONBField()
    end

    Repo-->>DSConfig: map[table][]map[string]any
    DSConfig-->>Client: Query results
```

### Design Patterns
- **Factory Pattern**: `NewDataSourceFromConnection()` creates configured datasources
- **Repository Pattern**: `ExternalDataSource` abstracts database operations
- **Interface Segregation**: `Repository` interface defines minimal contract
- **Embedding**: `DataSourceConfigPostgres` embeds base `DataSourceConfig`

## Components

### Connection

**Location:** `pkg/postgres/postgres.go`

**Responsibility:** Manages PostgreSQL database connections with connection pooling.

```go
type Connection struct {
    ConnectionString   string     // postgres://user:pass@host:port/db?sslmode=...
    DBName             string     // Database name
    ConnectionDB       *sql.DB    // Connection pool
    Connected          bool       // Connection state
    Logger             log.Logger
    MaxOpenConnections int        // Default: 25
    MaxIdleConnections int        // Default: 10
}
```

#### Methods

| Method | Parameters | Returns | Description |
|--------|------------|---------|-------------|
| `Connect()` | - | `error` | Opens connection, pings DB, configures pool |
| `GetDB()` | - | `(*sql.DB, error)` | Lazy-loads connection if nil |

**Connection Pool Settings:**
- `MaxOpenConns`: 25
- `MaxIdleConns`: 10
- `MaxLifetime`: 5 minutes
- `MaxIdleTime`: 1 minute

### Datasource Interface

**Location:** `pkg/postgres/datasource.postgres.go`

```go
type Datasource interface {
    Query(ctx context.Context, schema []TableSchema, table string,
          fields []string, filter map[string][]any) ([]map[string]any, error)
    QueryWithAdvancedFilters(ctx context.Context, schema []TableSchema, table string,
                            fields []string, filter map[string]job.FilterCondition) ([]map[string]any, error)
    GetDatabaseSchema(ctx context.Context, schemas []string) ([]TableSchema, error)
    CloseConnection() error
}
```

### ExternalDataSource

**Location:** `pkg/postgres/datasource.postgres.go`

**Responsibility:** Implements `Datasource` interface for query execution and schema discovery.

#### Query()

**Parameters:**
- `ctx context.Context` - Request context with tracing
- `schema []TableSchema` - Pre-fetched schema for validation
- `table string` - Target table (supports `"schema.table"` format)
- `fields []string` - Columns to select (`["*"]` for all)
- `filter map[string][]any` - Simple IN-clause filters

**Behavior:**
1. Validates table and fields against schema
2. Builds parameterized SELECT with squirrel (`$1, $2, ...` placeholders)
3. Executes with 10-second timeout
4. Parses JSONB fields automatically

**Example:**
```go
results, err := repo.Query(ctx, schema, "public.users",
    []string{"id", "name", "metadata"},
    map[string][]any{"status": {"active", "pending"}})
// SELECT id, name, metadata FROM public.users WHERE status IN ($1, $2)
```

#### QueryWithAdvancedFilters()

**Parameters:**
- Same as `Query()` but with `filter map[string]job.FilterCondition`

**Supported Operators:**

| Operator | Field | Example | SQL Generated |
|----------|-------|---------|---------------|
| `eq` | `Equals` | `[1, 2]` | `WHERE id IN ($1, $2)` |
| `gt` | `GreaterThan` | `[100]` | `WHERE amount > $1` |
| `gte` | `GreaterOrEqual` | `[100]` | `WHERE amount >= $1` |
| `lt` | `LessThan` | `[1000]` | `WHERE amount < $1` |
| `lte` | `LessOrEqual` | `[1000]` | `WHERE amount <= $1` |
| `between` | `Between` | `[100, 1000]` | `WHERE amount >= $1 AND amount <= $2` |
| `in` | `In` | `["a", "b"]` | `WHERE status IN ($1, $2)` |
| `nin` | `NotIn` | `["c"]` | `WHERE status NOT IN ($1)` |
| `ne` | `NotEquals` | `["inactive"]` | `WHERE status != $1` |
| `like` | `Like` | `["%active%"]` | `WHERE name LIKE $1` |

**Special Behaviors:**
- **Date fields**: End date adjusted to `T23:59:59.999Z` for `between` operator
- **UUID fields**: Validates UUID format for fields containing "id", "uuid", etc.
- **Timeout**: 15 seconds (vs 10 for simple queries)

#### GetDatabaseSchema()

**Parameters:**
- `ctx context.Context` - Request context
- `schemas []string` - Schema names (defaults to `["public"]`)

**Returns:** `[]TableSchema` with tables, columns, types, nullable flags, and primary keys

**Schema Discovery Process:**

```mermaid
graph LR
    A[queryTables] --> B[queryPrimaryKeys]
    B --> C[buildSchema]
    C --> D[buildTableSchema per table]
    D --> E[scanColumns]
```

**SQL Queries Used:**
```sql
-- Tables
SELECT table_name FROM information_schema.tables
WHERE table_schema = ANY($1) AND table_type = 'BASE TABLE'

-- Primary Keys
SELECT tc.table_name, kc.column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kc ...
WHERE tc.constraint_type = 'PRIMARY KEY'

-- Columns
SELECT column_name, data_type,
       CASE WHEN is_nullable = 'YES' THEN true ELSE false END
FROM information_schema.columns
WHERE table_schema = ANY($1) AND table_name = $2
```

### DataSourceConfigPostgres

**Location:** `pkg/model/datasource/postgres/datasource-config.go`

**Responsibility:** High-level datasource wrapper implementing `DataSource` interface.

```go
type DataSourceConfigPostgres struct {
    datasource.DataSourceConfig           // Base config (ID, Host, Port, etc.)
    PostgresConnection *postgres.Connection
    PostgresRepository *postgres.ExternalDataSource
}
```

#### Methods

| Method | Description |
|--------|-------------|
| `GetConfig()` | Returns embedded base configuration |
| `GetType()` | Returns database type string |
| `Connect(ctx, logger)` | Sets status to available (connection pre-established) |
| `Close(ctx)` | Closes repository connection |
| `Query(ctx, tables, filters, logger)` | Multi-table query orchestration |
| `GetSchemaInfo(ctx, schemas)` | Returns `*model.DataSourceSchema` |

## Integrations and Dependencies

### Dependency Diagram

```mermaid
graph TB
    subgraph "Consumers"
        Worker[components/worker/services/extract-data.go]
        TestConn[components/manager/services/query/test_connection.go]
        ValidateSchema[components/manager/services/query/validate_schema.go]
    end

    subgraph "PostgreSQL Package"
        PGPkg[pkg/postgres]
        ModelPG[pkg/model/datasource/postgres]
    end

    subgraph "Core Dependencies"
        DSFactory[pkg/datasource/datasource-factory.go]
        DSConfig[pkg/model/datasource/datasource-config.go]
        SSLMode[pkg/datasource/sslmode/postgresql.go]
        Model[pkg/model/connection.go]
    end

    subgraph "Shared Libraries"
        LibCommons[lib-commons/v2]
        Crypto[pkg/crypto]
    end

    Worker -->|uses| DSFactory
    TestConn -->|uses| DSFactory
    ValidateSchema -->|uses| DSFactory

    DSFactory -->|creates| ModelPG
    DSFactory -->|validates| SSLMode
    DSFactory -->|decrypts| Crypto

    ModelPG -->|embeds| DSConfig
    ModelPG -->|uses| PGPkg

    PGPkg -->|logging| LibCommons
    PGPkg -->|tracing| LibCommons
```

### Interfaces Implemented
- `datasource.DataSource` - Core datasource interface
- `postgres.Repository` - PostgreSQL-specific repository interface

### Packages That Depend on This Datasource
| Package | File | Usage |
|---------|------|-------|
| `components/worker` | `extract-data.go:354` | Data extraction jobs |
| `components/manager` | `test_connection.go:113` | Connection testing |
| `components/manager` | `validate_schema.go:198` | Schema validation |

## Error Handling

### Custom Error Types

Errors use the standardized `FET-XXXX` code format:

| Code | Constant | Description |
|------|----------|-------------|
| `FET-0413` | `ErrInvalidSSLMode` | Invalid SSL mode value |
| `FET-1040` | `ErrConnectionDown` | Database connection failed |
| `FET-1060` | `ErrSchemaValidationFailed` | Schema validation error |

### Error Wrapping Pattern

```go
// Connection errors
return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", errConnect)

// Query errors
return nil, fmt.Errorf("error executing query: %w", err)

// Timeout detection
if queryCtx.Err() == context.DeadlineExceeded {
    return nil, fmt.Errorf("query execution timeout after %v: %w", timeout, err)
}
```

### Retry Strategy
- **No built-in retry**: Relies on connection pooling for resilience
- **Connection pool**: Automatically manages connection lifecycle
- **Caller responsibility**: Services implement retry logic as needed

### Logging and Observability

**Log Levels:**
- `INFO`: Connection status, query execution starts
- `DEBUG`: SQL generation, connection strings (password masked)
- `ERROR`: Connection failures, query errors
- `WARN`: JSONB parsing failures

**OpenTelemetry Spans:**

| Span Name | Attributes |
|-----------|------------|
| `postgres.data_source.query` | `request_id`, `repository_filter` |
| `postgres.data_source.query_with_advanced_filters` | `request_id`, `repository_filter` |
| `postgres.data_source.validate_table_and_fields` | `request_id` |
| `postgres.data_source.get_database_schema` | `request_id` |
| `datasource.postgres.get_schema_info` | `config_name`, `type`, `tables_count` |

## Usage Examples

### Basic CRUD Operations

#### Simple Query

```go
// Create datasource via factory
ds, err := datasource.NewDataSourceFromConnection(ctx, conn, cryptor, logger)
if err != nil {
    return err
}
defer ds.Close(ctx)

// Query with simple filter
results, err := ds.Query(ctx,
    map[string][]string{
        "public.users": {"id", "name", "email"},
    },
    map[string]map[string]job.FilterCondition{
        "postgres": {
            "public.users": {
                Equals: []any{"active"},
            },
        },
    },
    logger,
)
```

#### Advanced Filtering

```go
// Date range query with multiple conditions
results, err := ds.Query(ctx,
    map[string][]string{
        "public.orders": {"id", "customer_id", "total", "created_at"},
    },
    map[string]map[string]job.FilterCondition{
        "postgres": {
            "public.orders": {
                Between: []any{"2024-01-01", "2024-12-31"},  // Auto-adjusted end date
                GreaterThan: []any{100.00},
            },
        },
    },
    logger,
)
```

### Schema Discovery

```go
// Get schema for specific schemas
schema, err := ds.GetSchemaInfo(ctx, []string{"public", "audit"})
if err != nil {
    return err
}

for _, table := range schema.Tables {
    fmt.Printf("Table: %s, Columns: %v\n", table.Name, table.Columns)
}
```

### Connection Testing

```go
// Direct connection test (used by test_connection service)
conn := &postgres.Connection{
    ConnectionString: "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
    Logger:           logger,
}

if err := conn.Connect(); err != nil {
    return fmt.Errorf("connection test failed: %w", err)
}
defer conn.ConnectionDB.Close()
```

## Connection String Format

```
postgres://[username]:[password]@[host]:[port]/[database]?sslmode=[mode]
```

**Components:**
| Component | Description | Example |
|-----------|-------------|---------|
| `username` | Database user | `myuser` |
| `password` | URL-encoded password | `P%40ssw0rd` |
| `host` | Server hostname/IP | `localhost` |
| `port` | Server port | `5432` |
| `database` | Target database | `mydb` |
| `sslmode` | SSL connection mode | `require` |

**SSL Modes:**
| Mode | Description |
|------|-------------|
| `disable` | No SSL (default) |
| `allow` | Try non-SSL first, then SSL |
| `prefer` | Try SSL first, fallback to non-SSL |
| `require` | Require SSL, don't verify certificate |
| `verify-ca` | Require SSL, verify CA |
| `verify-full` | Require SSL, verify CA and hostname |

## Query Timeouts

| Operation | Timeout | Constant |
|-----------|---------|----------|
| Simple queries | 10 seconds | `QueryTimeoutMedium` |
| Advanced filter queries | 15 seconds | `QueryTimeoutSlow` |
| Schema discovery | 30 seconds | `SchemaDiscoveryTimeout` |
| Connection establishment | 5 seconds | `ConnectionTimeout` |
