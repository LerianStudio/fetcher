# SQL Server Datasource

Microsoft SQL Server datasource implementation for the Fetcher service, providing data extraction and schema discovery capabilities.

## Overview

### Purpose
This datasource enables connection, querying, and schema discovery for SQL Server databases. It implements the `DataSource` interface and provides advanced filtering, field validation, and automatic JSON field parsing.

### Supported Versions
- SQL Server 2016+
- SQL Server 2019+
- SQL Server 2022+ (TDS 8.0 strict encryption)
- Azure SQL Database
- Azure SQL Managed Instance
- Amazon RDS for SQL Server

### External Dependencies
| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/microsoft/go-mssqldb` | v1.9.5 | Official Microsoft SQL Server driver |
| `github.com/Masterminds/squirrel` | v1.5.4 | SQL query builder |

## Architecture

### Component Diagram

```mermaid
graph TB
    subgraph "pkg/model/datasource/sqlserver"
        DSConfig[DataSourceConfigSQLServer]
    end

    subgraph "pkg/sqlserver"
        Conn[Connection]
        Repo[ExternalDataSource]
        RepoInterface[Repository Interface]
        Placeholder[sqlServerPlaceholder]
    end

    subgraph "pkg/datasource"
        Factory[datasource-factory.go]
        SSLMode[sslmode/sqlserver.go]
    end

    subgraph "External"
        Driver[go-mssqldb Driver]
        DB[(SQL Server)]
    end

    Factory -->|creates| DSConfig
    Factory -->|validates| SSLMode
    DSConfig -->|contains| Conn
    DSConfig -->|contains| Repo
    Repo -->|implements| RepoInterface
    Repo -->|uses| Conn
    Repo -->|uses| Placeholder
    Conn -->|uses| Driver
    Driver -->|connects| DB
```

### Data Flow

```mermaid
sequenceDiagram
    participant Client
    participant DSConfig as DataSourceConfigSQLServer
    participant Repo as ExternalDataSource
    participant Conn as Connection
    participant DB as SQL Server

    Client->>DSConfig: Query(ctx, tables, filters, logger)
    DSConfig->>Repo: GetDatabaseSchema(ctx, schemas)
    Repo->>Conn: GetDB()
    Conn->>DB: INFORMATION_SCHEMA queries
    DB-->>Repo: Schema metadata

    loop For each table
        DSConfig->>Repo: QueryWithAdvancedFilters(ctx, schema, table, fields, filter)
        Repo->>Repo: ValidateTableAndFields()
        Repo->>Repo: buildAdvancedFilters()
        Repo->>Repo: ReplacePlaceholders (? → @p1, @p2)
        Repo->>DB: SELECT query with @p placeholders
        DB-->>Repo: Result rows
        Repo->>Repo: parseJSONField()
    end

    Repo-->>DSConfig: map[table][]map[string]any
    DSConfig-->>Client: Query results
```

### Design Patterns
- **Factory Pattern**: `NewDataSourceFromConnection()` creates configured datasources
- **Repository Pattern**: `ExternalDataSource` abstracts database operations
- **Adapter Pattern**: `sqlServerPlaceholder` adapts squirrel to SQL Server syntax
- **Embedding**: `DataSourceConfigSQLServer` embeds base `DataSourceConfig`

## Components

### Connection

**Location:** `pkg/sqlserver/sqlserver.go`

**Responsibility:** Manages SQL Server database connections with connection pooling.

```go
type Connection struct {
    ConnectionString   string     // ADO-style connection string
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

**Location:** `pkg/sqlserver/datasource.sqlserver.go`

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

**Location:** `pkg/sqlserver/datasource.sqlserver.go`

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
2. Builds parameterized SELECT with squirrel
3. Converts `?` placeholders to `@p1, @p2, ...` format
4. Executes with 10-second timeout
5. Parses JSON fields automatically

**Example:**
```go
results, err := repo.Query(ctx, schema, "dbo.users",
    []string{"id", "name", "metadata"},
    map[string][]any{"status": {"active", "pending"}})
// SELECT id, name, metadata FROM dbo.users WHERE status IN (@p1, @p2)
```

#### QueryWithAdvancedFilters()

**Parameters:**
- Same as `Query()` but with `filter map[string]job.FilterCondition`

**Supported Operators:**

| Operator | Field | Example | SQL Generated |
|----------|-------|---------|---------------|
| `eq` | `Equals` | `[1, 2]` | `WHERE id IN (@p1, @p2)` |
| `gt` | `GreaterThan` | `[100]` | `WHERE amount > @p1` |
| `gte` | `GreaterOrEqual` | `[100]` | `WHERE amount >= @p1` |
| `lt` | `LessThan` | `[1000]` | `WHERE amount < @p1` |
| `lte` | `LessOrEqual` | `[1000]` | `WHERE amount <= @p1` |
| `between` | `Between` | `[100, 1000]` | `WHERE amount >= @p1 AND amount <= @p2` |
| `in` | `In` | `["a", "b"]` | `WHERE status IN (@p1, @p2)` |
| `nin` | `NotIn` | `["c"]` | `WHERE status NOT IN (@p1)` |
| `ne` | `NotEquals` | `["inactive"]` | `WHERE status != @p1` |
| `like` | `Like` | `["%active%"]` | `WHERE name LIKE @p1` |

**Special Behaviors:**
- **Date fields**: End date adjusted to `T23:59:59.999Z` for `between` operator
- **UUID fields**: Validates UUID format for fields containing "id", "uuid", etc.
- **Timeout**: 15 seconds (vs 10 for simple queries)

#### GetDatabaseSchema()

**Parameters:**
- `ctx context.Context` - Request context
- `schemas []string` - Schema names (defaults to `["dbo"]`)

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
WHERE table_type = 'BASE TABLE'
  AND table_schema IN (@p1, @p2, ...) -- or = 'dbo' if empty

-- Primary Keys
SELECT tc.table_schema, tc.table_name, kc.column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kc ...
WHERE tc.constraint_type = 'PRIMARY KEY'
  AND tc.table_schema IN (@p1, @p2, ...) -- or = 'dbo'

-- Columns
SELECT column_name, data_type,
       CASE WHEN is_nullable = 'YES' THEN 1 ELSE 0 END AS is_nullable
FROM information_schema.columns
WHERE table_name = @p1
  AND table_schema IN (@p2, @p3, ...) -- or = 'dbo'
ORDER BY ordinal_position
```

### SQL Server Placeholder Adapter

**Location:** `pkg/sqlserver/datasource.sqlserver.go`

```go
type sqlServerPlaceholder struct{}

func (p sqlServerPlaceholder) ReplacePlaceholders(sqlStr string) (string, error)
```

**Responsibility:** Converts squirrel's generic `?` placeholders to SQL Server's `@p1, @p2, ...` format.

**Example:**
```
Input:  "SELECT * FROM users WHERE id = ? AND name = ?"
Output: "SELECT * FROM users WHERE id = @p1 AND name = @p2"
```

### DataSourceConfigSQLServer

**Location:** `pkg/model/datasource/sqlserver/datasource-config.go`

**Responsibility:** High-level datasource wrapper implementing `DataSource` interface.

```go
type DataSourceConfigSQLServer struct {
    datasource.DataSourceConfig           // Base config (ID, Host, Port, etc.)
    SQLServerConnection *sqlserver.Connection
    SQLServerRepository sqlserver.Repository
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

#### Table Filter Matching

The `getTableFilters()` function uses three-phase matching:

```mermaid
graph TD
    A[Input: dbo.users] --> B{Exact match?}
    B -->|Yes| C[Return filters]
    B -->|No| D{Table has schema?}
    D -->|Yes| E{Match without schema?}
    D -->|No| F{Match with dbo prefix?}
    E -->|Yes| C
    E -->|No| G[Return nil]
    F -->|Yes| C
    F -->|No| G
```

## Integrations and Dependencies

### Dependency Diagram

```mermaid
graph TB
    subgraph "Consumers"
        Worker[components/worker UseCase<br/>extractViaEngine]
        TestConn[components/manager/services/query/test_connection.go]
        ValidateSchema[components/manager/services/query/validate_schema.go]
    end

    subgraph "Runtime Engine"
        Engine[pkg/engine<br/>PlanExtraction + ExecuteExtraction]
        EngineCompat[pkg/enginecompat/datasource<br/>ConnectorFactory]
        SchemaCompat[pkg/enginecompat/schemacompat]
    end

    subgraph "SQL Server Package"
        SQLSrvPkg[pkg/sqlserver]
        ModelSQLSrv[pkg/model/datasource/sqlserver]
    end

    subgraph "Core Dependencies"
        DSFactory[pkg/datasource/datasource-factory.go]
        DSConfig[pkg/model/datasource/datasource-config.go]
        SSLMode[pkg/datasource/sslmode/sqlserver.go]
        Model[pkg/model/connection.go]
    end

    subgraph "Shared Libraries"
        LibCommons[lib-commons/v2]
        Crypto[pkg/crypto]
    end

    Worker -->|RunExtraction| Engine
    TestConn -->|uses| DSFactory
    ValidateSchema -->|via| SchemaCompat
    Engine -->|Connector port| EngineCompat
    SchemaCompat -->|schema engine| Engine
    EngineCompat -->|uses| DSFactory

    DSFactory -->|creates| ModelSQLSrv
    DSFactory -->|validates| SSLMode
    DSFactory -->|decrypts| Crypto

    ModelSQLSrv -->|embeds| DSConfig
    ModelSQLSrv -->|uses| SQLSrvPkg

    SQLSrvPkg -->|logging| LibCommons
    SQLSrvPkg -->|tracing| LibCommons
```

### Interfaces Implemented
- `datasource.DataSource` - Core datasource interface
- `sqlserver.Repository` - SQL Server-specific repository interface

### Packages That Depend on This Datasource
| Package | File | Usage |
|---------|------|-------|
| `pkg/engine` (via `pkg/enginecompat/datasource`) | `adapter.go` | Generic data extraction jobs (worker `UseCase.extractViaEngine` → `EngineRunner.RunExtraction` → engine `Connector` port → factory) |
| `components/manager` | `test_connection.go:113` | Connection testing |
| `components/manager` (via `pkg/enginecompat/schemacompat`) | `validate_schema.go:198` | Schema validation / discovery |

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
return nil, fmt.Errorf("failed to connect to SQL Server: %w", errConnect)

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
- `WARN`: JSON parsing failures

**OpenTelemetry Spans:**

| Span Name | Attributes |
|-----------|------------|
| `sqlserver.data_source.query` | `request_id`, `repository_filter` |
| `sqlserver.data_source.query_with_advanced_filters` | `request_id`, `repository_filter` |
| `sqlserver.data_source.validate_table_and_fields` | `request_id` |
| `sqlserver.data_source.get_database_schema` | `request_id` |
| `datasource.sqlserver.get_schema_info` | `config_name`, `type`, `tables_count` |

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
        "dbo.users": {"id", "name", "email"},
    },
    map[string]map[string]job.FilterCondition{
        "sqlserver": {
            "dbo.users": {
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
        "dbo.orders": {"id", "customer_id", "total", "created_at"},
    },
    map[string]map[string]job.FilterCondition{
        "sqlserver": {
            "dbo.orders": {
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
schema, err := ds.GetSchemaInfo(ctx, []string{"dbo", "sales"})
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
conn := &sqlserver.Connection{
    ConnectionString: "sqlserver://user:password@localhost:1433?database=mydb&encrypt=disable",
    Logger:           logger,
}

if err := conn.Connect(); err != nil {
    return fmt.Errorf("connection test failed: %w", err)
}
defer conn.ConnectionDB.Close()
```

## Connection String Format

```
sqlserver://[username]:[password]@[host]:[port]?database=[dbname]&encrypt=[mode][&TrustServerCertificate=true]
```

**Examples:**
```go
// Basic connection
"sqlserver://sa:Password123@localhost:1433?database=mydb&encrypt=disable"

// With encryption
"sqlserver://sa:Password123@localhost:1433?database=mydb&encrypt=true&TrustServerCertificate=true"

// Strict encryption (SQL Server 2022+)
"sqlserver://sa:Password123@localhost:1433?database=mydb&encrypt=strict"

// Azure SQL Database
"sqlserver://user@server:Password123@server.database.windows.net:1433?database=mydb&encrypt=true"
```

**Components:**
| Component | Description | Example |
|-----------|-------------|---------|
| `username` | Database user | `sa` |
| `password` | URL-encoded password | `P%40ssw0rd` |
| `host` | Server hostname/IP | `localhost` |
| `port` | Server port | `1433` |
| `database` | Target database | `mydb` |
| `encrypt` | Encryption mode | `true`, `strict` |

**Encryption Modes:**
| Mode | Description |
|------|-------------|
| `disable` | No encryption (default) |
| `false` | Encryption disabled (explicit) |
| `true` | Encryption with TrustServerCertificate=true |
| `strict` | TDS 8.0 strict encryption (SQL Server 2022+) |

**Mode Handling Logic:**
```
disable → encrypt=false
false   → encrypt=false
true    → encrypt=true + TrustServerCertificate=true
strict  → encrypt=strict (certificate validation required)
```

## Query Timeouts

| Operation | Timeout | Constant |
|-----------|---------|----------|
| Simple queries | 10 seconds | `QueryTimeoutMedium` |
| Advanced filter queries | 15 seconds | `QueryTimeoutSlow` |
| Schema discovery | 30 seconds | `SchemaDiscoveryTimeout` |
| Connection establishment | 5 seconds | `ConnectionTimeout` |

## SQL Server-Specific Considerations

### Schema Handling
- Default schema: `dbo`
- Supports `"schema.table"` format
- Three-phase filter matching for flexibility

### Placeholder Format
- SQL Server uses `@p1, @p2, @p3, ...` named parameters
- Squirrel's `?` placeholders automatically converted

### Data Types
Common SQL Server data types and their mapping:

| SQL Server Type | Description |
|-----------------|-------------|
| `int` | 32-bit integer |
| `bigint` | 64-bit integer |
| `varchar(n)` | Variable-length string |
| `nvarchar(n)` | Unicode variable-length string |
| `datetime2` | Date and time with precision |
| `decimal(p,s)` | Exact numeric |
| `uniqueidentifier` | UUID/GUID |
| `nvarchar(max)` | Large text (often JSON) |

### JSON Field Handling

SQL Server JSON columns (stored as `nvarchar(max)`) are automatically parsed:

```go
// SQL Server table with JSON column
result, err := repo.Query(ctx, schema, "dbo.products", []string{"id", "metadata"}, nil)

// Input from SQL Server: {"metadata": []uint8(`{"color":"red","size":"large"}`)}
// Parsed result:         {"metadata": map[string]any{"color": "red", "size": "large"}}
```

**Parsing Order:**
1. Try unmarshal as `map[string]any` (JSON object)
2. Try unmarshal as `[]any` (JSON array)
3. Try unmarshal as `string` (JSON string literal)
4. Return raw bytes with warning log if all fail

## Key Characteristics

| Aspect | Detail |
|--------|--------|
| **Schema handling** | Defaults to `dbo`; supports multi-schema |
| **Field validation** | Case-sensitive matching |
| **Filter combination** | Multiple filters: OR within field, AND between fields |
| **JSON support** | Automatic parsing of JSON stored as nvarchar |
| **NULL handling** | Preserved in results as nil |
| **Wildcard support** | `"*"` expands to all columns |
| **Transaction support** | None - read-only queries |
| **Prepared statements** | Via squirrel with `@p1, @p2` parameters |
| **TDS Protocol** | Supports TDS 7.x and TDS 8.0 (strict mode) |
