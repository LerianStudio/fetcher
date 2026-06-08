# Oracle Datasource

Oracle Database datasource implementation for the Fetcher service, providing data extraction and schema discovery capabilities.

## Overview

### Purpose
This datasource enables connection, querying, and schema discovery for Oracle databases. It implements the `DataSource` interface and provides advanced filtering, field validation, and automatic JSON field parsing.

### Supported Versions
- Oracle Database 12c+
- Oracle Database 19c+
- Oracle Database 21c+
- Oracle Autonomous Database
- Oracle Cloud Infrastructure (OCI) Database

### External Dependencies
| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/sijms/go-ora/v2` | v2.9.0 | Oracle driver |
| `github.com/Masterminds/squirrel` | v1.5.4 | SQL query builder |

## Architecture

### Component Diagram

```mermaid
graph TB
    subgraph "pkg/model/datasource/oracle"
        DSConfig[DataSourceConfigOracle]
    end

    subgraph "pkg/oracle"
        Conn[Connection]
        Repo[ExternalDataSource]
        RepoInterface[Repository Interface]
    end

    subgraph "pkg/datasource"
        Factory[datasource-factory.go]
        SSLMode[sslmode/oracle.go]
    end

    subgraph "External"
        Driver[go-ora/v2 Driver]
        DB[(Oracle DB)]
    end

    Factory -->|creates| DSConfig
    Factory -->|validates| SSLMode
    DSConfig -->|contains| Conn
    DSConfig -->|contains| Repo
    Repo -->|implements| RepoInterface
    Repo -->|uses| Conn
    Conn -->|uses| Driver
    Driver -->|connects| DB
```

### Data Flow

```mermaid
sequenceDiagram
    participant Client
    participant DSConfig as DataSourceConfigOracle
    participant Repo as ExternalDataSource
    participant Conn as Connection
    participant DB as Oracle

    Client->>DSConfig: Query(ctx, tables, filters, logger)
    DSConfig->>Repo: GetDatabaseSchema(ctx, schemas)
    Repo->>Conn: GetDB()
    Conn->>DB: USER_TABLES / ALL_TABLES queries
    DB-->>Repo: Schema metadata

    loop For each table
        DSConfig->>Repo: QueryWithAdvancedFilters(ctx, schema, table, fields, filter)
        Repo->>Repo: ValidateTableAndFields()
        Repo->>Repo: buildAdvancedFilters()
        Repo->>DB: SELECT query with :1, :2 placeholders
        DB-->>Repo: Result rows
        Repo->>Repo: parseJSONField()
    end

    Repo-->>DSConfig: map[table][]map[string]any
    DSConfig-->>Client: Query results
```

### Design Patterns
- **Factory Pattern**: `NewDataSourceFromConnection()` creates configured datasources
- **Repository Pattern**: `ExternalDataSource` abstracts database operations
- **Interface Segregation**: `Repository` interface defines minimal contract
- **Embedding**: `DataSourceConfigOracle` embeds base `DataSourceConfig`

## Components

### Connection

**Location:** `pkg/oracle/oracle.go`

**Responsibility:** Manages Oracle database connections with connection pooling.

```go
type Connection struct {
    ConnectionString   string     // TNS or Easy Connect format
    DBName             string     // Service name
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

**Location:** `pkg/oracle/datasource.oracle.go`

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

**Location:** `pkg/oracle/datasource.oracle.go`

**Responsibility:** Implements `Datasource` interface for query execution and schema discovery.

#### Query()

**Parameters:**
- `ctx context.Context` - Request context with tracing
- `schema []TableSchema` - Pre-fetched schema for validation
- `table string` - Target table (supports `"SCHEMA.TABLE"` format)
- `fields []string` - Columns to select (`["*"]` for all)
- `filter map[string][]any` - Simple IN-clause filters

**Behavior:**
1. Validates table and fields against schema (case-insensitive)
2. Builds parameterized SELECT with squirrel (`:1, :2, ...` placeholders)
3. Executes with 10-second timeout
4. Parses JSON fields automatically

**Example:**
```go
results, err := repo.Query(ctx, schema, "HR.EMPLOYEES",
    []string{"EMPLOYEE_ID", "FIRST_NAME", "METADATA"},
    map[string][]any{"STATUS": {"ACTIVE", "PENDING"}})
// SELECT EMPLOYEE_ID, FIRST_NAME, METADATA FROM HR.EMPLOYEES WHERE STATUS IN (:1, :2)
```

#### QueryWithAdvancedFilters()

**Parameters:**
- Same as `Query()` but with `filter map[string]job.FilterCondition`

**Supported Operators:**

| Operator | Field | Example | SQL Generated |
|----------|-------|---------|---------------|
| `eq` | `Equals` | `[1, 2]` | `WHERE ID IN (:1, :2)` |
| `gt` | `GreaterThan` | `[100]` | `WHERE AMOUNT > :1` |
| `gte` | `GreaterOrEqual` | `[100]` | `WHERE AMOUNT >= :1` |
| `lt` | `LessThan` | `[1000]` | `WHERE AMOUNT < :1` |
| `lte` | `LessOrEqual` | `[1000]` | `WHERE AMOUNT <= :1` |
| `between` | `Between` | `[100, 1000]` | `WHERE AMOUNT >= :1 AND AMOUNT <= :2` |
| `in` | `In` | `["A", "B"]` | `WHERE STATUS IN (:1, :2)` |
| `nin` | `NotIn` | `["C"]` | `WHERE STATUS NOT IN (:1)` |
| `ne` | `NotEquals` | `["INACTIVE"]` | `WHERE STATUS != :1` |
| `like` | `Like` | `["%ACTIVE%"]` | `WHERE NAME LIKE :1` |

**Special Behaviors:**
- **Date fields**: End date adjusted to `T23:59:59.999Z` for `between` operator
- **UUID fields**: Validates UUID format for fields containing "id", "uuid", etc.
- **Timeout**: 15 seconds (vs 10 for simple queries)

#### GetDatabaseSchema()

**Parameters:**
- `ctx context.Context` - Request context
- `schemas []string` - Schema/owner names (empty = current user's tables)

**Returns:** `[]TableSchema` with tables, columns, types, nullable flags, and primary keys

**Schema Discovery Process:**

```mermaid
graph LR
    A[queryTables] --> B[queryPrimaryKeys]
    B --> C[buildSchema]
    C --> D[buildTableSchema per table]
    D --> E[scanColumns]
```

**SQL Queries Used (No Schemas Specified):**
```sql
-- Tables (USER_TABLES)
SELECT table_name FROM user_tables ORDER BY table_name

-- Primary Keys (USER_CONSTRAINTS)
SELECT table_name, column_name
FROM user_cons_columns
WHERE constraint_name IN (
    SELECT constraint_name FROM user_constraints WHERE constraint_type = 'P'
)

-- Columns (USER_TAB_COLUMNS)
SELECT column_name, data_type,
       CASE WHEN nullable = 'Y' THEN 1 ELSE 0 END AS is_nullable
FROM user_tab_columns WHERE table_name = :1 ORDER BY column_id
```

**SQL Queries Used (With Schemas):**
```sql
-- Tables (ALL_TABLES)
SELECT table_name FROM all_tables WHERE owner IN (:1, :2, ...) ORDER BY owner, table_name

-- Primary Keys (ALL_CONSTRAINTS)
SELECT acc.table_name, acc.column_name
FROM all_cons_columns acc
JOIN all_constraints ac ON ac.owner = acc.owner
  AND ac.constraint_name = acc.constraint_name
WHERE ac.constraint_type = 'P' AND ac.owner IN (:1, :2, ...)

-- Columns (ALL_TAB_COLUMNS)
SELECT column_name, data_type,
       CASE WHEN nullable = 'Y' THEN 1 ELSE 0 END AS is_nullable
FROM all_tab_columns
WHERE table_name = :1 AND owner IN (:2, :3, ...)
ORDER BY owner, column_id
```

### DataSourceConfigOracle

**Location:** `pkg/model/datasource/oracle/datasource-config.go`

**Responsibility:** High-level datasource wrapper implementing `DataSource` interface.

```go
type DataSourceConfigOracle struct {
    datasource.DataSourceConfig           // Base config (ID, Host, Port, etc.)
    OracleConnection *oracle.Connection
    OracleRepository oracle.Repository
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
        Worker[components/worker UseCase<br/>extractViaEngine]
        TestConn[components/manager/services/query/test_connection.go]
        ValidateSchema[components/manager/services/query/validate_schema.go]
    end

    subgraph "Runtime Engine"
        Engine[pkg/engine<br/>PlanExtraction + ExecuteExtraction]
        EngineCompat[pkg/enginecompat/datasource<br/>ConnectorFactory]
        SchemaCompat[pkg/enginecompat/schemacompat]
    end

    subgraph "Oracle Package"
        OraclePkg[pkg/oracle]
        ModelOracle[pkg/model/datasource/oracle]
    end

    subgraph "Core Dependencies"
        DSFactory[pkg/datasource/datasource-factory.go]
        DSConfig[pkg/model/datasource/datasource-config.go]
        SSLMode[pkg/datasource/sslmode/oracle.go]
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

    DSFactory -->|creates| ModelOracle
    DSFactory -->|validates| SSLMode
    DSFactory -->|decrypts| Crypto

    ModelOracle -->|embeds| DSConfig
    ModelOracle -->|uses| OraclePkg

    OraclePkg -->|logging| LibCommons
    OraclePkg -->|tracing| LibCommons
```

### Interfaces Implemented
- `datasource.DataSource` - Core datasource interface
- `oracle.Repository` - Oracle-specific repository interface

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
return nil, fmt.Errorf("failed to connect to Oracle: %w", errConnect)

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
| `oracle.data_source.query` | `request_id`, `repository_filter` |
| `oracle.data_source.query_with_advanced_filters` | `request_id`, `repository_filter` |
| `oracle.data_source.validate_table_and_fields` | `request_id` |
| `oracle.data_source.get_database_schema` | `request_id` |
| `datasource.oracle.get_schema_info` | `config_name`, `type`, `tables_count` |

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
        "HR.EMPLOYEES": {"EMPLOYEE_ID", "FIRST_NAME", "EMAIL"},
    },
    map[string]map[string]job.FilterCondition{
        "oracle": {
            "HR.EMPLOYEES": {
                Equals: []any{"ACTIVE"},
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
        "SALES.ORDERS": {"ORDER_ID", "CUSTOMER_ID", "TOTAL", "CREATED_AT"},
    },
    map[string]map[string]job.FilterCondition{
        "oracle": {
            "SALES.ORDERS": {
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
// Get schema for specific owners/schemas
schema, err := ds.GetSchemaInfo(ctx, []string{"HR", "SALES"})
if err != nil {
    return err
}

for _, table := range schema.Tables {
    fmt.Printf("Table: %s, Columns: %v\n", table.Name, table.Columns)
}

// Get current user's tables only
schema, err := ds.GetSchemaInfo(ctx, nil) // Uses USER_* views
```

### Connection Testing

```go
// Direct connection test (used by test_connection service)
conn := &oracle.Connection{
    ConnectionString: "oracle://user:password@localhost:1521/ORCL",
    Logger:           logger,
}

if err := conn.Connect(); err != nil {
    return fmt.Errorf("connection test failed: %w", err)
}
defer conn.ConnectionDB.Close()
```

## Connection String Format

### Easy Connect Format

```
oracle://[username]:[password]@[host]:[port]/[service_name]
```

**Example:**
```
oracle://hr_user:P%40ssw0rd@oracle.example.com:1521/ORCL
```

### TNS Format

```
user/password@(DESCRIPTION=(ADDRESS=(PROTOCOL=tcp)(HOST=hostname)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=service_name)))
```

**Components:**
| Component | Description | Example |
|-----------|-------------|---------|
| `username` | Database user | `hr_user` |
| `password` | URL-encoded password | `P%40ssw0rd` |
| `host` | Server hostname/IP | `oracle.example.com` |
| `port` | Listener port | `1521` |
| `service_name` | Oracle service name | `ORCL` |

**SSL/Wallet Modes:**
| Mode | Description |
|------|-------------|
| `disable` | No SSL (default) |
| `false` | SSL disabled (explicit) |
| `true` | SSL enabled |
| `enable` | SSL enabled |
| `verify` | SSL with certificate verification |
| `skip-verify` | SSL without certificate verification |

## Query Timeouts

| Operation | Timeout | Constant |
|-----------|---------|----------|
| Simple queries | 10 seconds | `QueryTimeoutMedium` |
| Advanced filter queries | 15 seconds | `QueryTimeoutSlow` |
| Schema discovery | 30 seconds | `SchemaDiscoveryTimeout` |
| Connection establishment | 5 seconds | `ConnectionTimeout` |

## Oracle-Specific Considerations

### Case Sensitivity
- Oracle stores identifiers in **UPPERCASE** by default
- Field names are matched **case-insensitively**
- Query results preserve original case from schema

### Schema vs. Owner
- Oracle uses "owner" concept equivalent to schema
- Empty schemas list uses `USER_*` views (current user)
- Specified schemas use `ALL_*` views (cross-schema)

### Data Types
Common Oracle data types and their mapping:

| Oracle Type | Description |
|-------------|-------------|
| `VARCHAR2` | Variable-length character |
| `NUMBER` | Numeric data |
| `DATE` | Date and time |
| `TIMESTAMP` | Timestamp with precision |
| `CLOB` | Character large object |
| `BLOB` | Binary large object |
| `RAW` | Raw binary data |

### JSON Field Handling

Oracle JSON columns (stored as CLOB/VARCHAR2) are automatically parsed:

```go
// Oracle table with JSON column
result, err := repo.Query(ctx, schema, "PRODUCTS", []string{"ID", "METADATA"}, nil)

// Input from Oracle: {"metadata": []uint8(`{"color":"red","size":"large"}`)}
// Parsed result:     {"metadata": map[string]any{"color": "red", "size": "large"}}
```

## Key Characteristics

| Aspect | Detail |
|--------|--------|
| **Schema handling** | Supports `OWNER.TABLE` format; empty uses current user |
| **Field validation** | Case-insensitive matching, returns original case |
| **Filter combination** | Multiple filters: OR within field, AND between fields |
| **JSON support** | Automatic parsing of JSON stored as CLOB/VARCHAR2 |
| **NULL handling** | Preserved in results as nil |
| **Wildcard support** | `"*"` expands to all columns |
| **Transaction support** | None - read-only queries |
| **Prepared statements** | Via squirrel with `:1, :2` colon placeholders |
