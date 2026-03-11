# MongoDB Datasource

MongoDB datasource implementation for the Fetcher service, providing data extraction and schema inference capabilities.

## Overview

### Purpose
This datasource enables connection, querying, and schema discovery for MongoDB databases. It implements the `DataSource` interface and provides advanced filtering with aggregation pipelines, intelligent schema inference through sampling, and automatic BSON type conversion.

### Supported Versions
- MongoDB 4.4+
- MongoDB 5.0+
- MongoDB 6.0+
- MongoDB 7.0+
- MongoDB Atlas
- Amazon DocumentDB (with MongoDB compatibility)

### External Dependencies
| Dependency | Version | Purpose |
|------------|---------|---------|
| `go.mongodb.org/mongo-driver` | v1.17.6 | Official MongoDB Go driver |

## Architecture

### Component Diagram

```mermaid
graph TB
    subgraph "pkg/model/datasource/mongodb"
        DSConfig[DataSourceConfigMongoDB]
    end

    subgraph "pkg/mongodb"
        Repo[ExternalDataSource]
        RepoInterface[Repository Interface]
        ConnMongo[connection/ConnectionMongoDBRepository]
        JobMongo[job/JobMongoDBRepository]
    end

    subgraph "pkg/datasource"
        Factory[datasource-factory.go]
        SSLMode[sslmode/mongodb.go]
    end

    subgraph "External"
        Driver[mongo-driver]
        DB[(MongoDB)]
    end

    subgraph "lib-commons"
        MongoClient[lib-commons/v4 mongo.Client]
    end

    Factory -->|creates| DSConfig
    Factory -->|validates| SSLMode
    DSConfig -->|contains| Repo
    Repo -->|implements| RepoInterface
    Repo -->|uses| MongoClient
    MongoClient -->|uses| Driver
    Driver -->|connects| DB
    ConnMongo -->|stores| DB
    JobMongo -->|stores| DB
```

### Data Flow

```mermaid
sequenceDiagram
    participant Client
    participant DSConfig as DataSourceConfigMongoDB
    participant Repo as ExternalDataSource
    participant Driver as mongo-driver
    participant DB as MongoDB

    Client->>DSConfig: Query(ctx, collections, filters, logger)
    DSConfig->>Repo: GetDatabaseSchema(ctx)
    Repo->>DB: ListCollections

    loop For each collection
        Repo->>DB: Count documents
        alt Small collection (≤10k)
            Repo->>DB: Aggregation pipeline (field discovery)
        else Large collection (>10k)
            Repo->>DB: $sample aggregation (statistical sampling)
        end
        Repo->>DB: Sample documents (type inference)
    end

    DB-->>Repo: Schema with field types

    loop For each collection
        DSConfig->>Repo: QueryWithAdvancedFilters(ctx, collection, fields, filter)
        Repo->>Repo: buildMongoFilter()
        Repo->>DB: Find with projection
        DB-->>Repo: Cursor
        Repo->>Repo: convertBsonToMap()
    end

    Repo-->>DSConfig: map[collection][]map[string]any
    DSConfig-->>Client: Query results
```

### Design Patterns
- **Factory Pattern**: `NewDataSourceFromConnection()` creates configured datasources
- **Repository Pattern**: `ExternalDataSource` abstracts database operations
- **Strategy Pattern**: Adaptive schema discovery (aggregation vs. sampling)
- **Embedding**: `DataSourceConfigMongoDB` embeds base `DataSourceConfig`

## Components

### ExternalDataSource

**Location:** `pkg/mongodb/datasource.mongodb.go`

**Responsibility:** Core MongoDB repository for executing queries and discovering schemas.

```go
type ExternalDataSource struct {
    connection *libMongo.Client  // lib-commons/v4 mongo client
    Database   string            // Database name
}
```

#### Constructor

```go
func NewDataSourceRepository(mongoURI string, dbName string, logger log.Logger) (*ExternalDataSource, error)
```

**Behavior:**
- Creates `libMongo.Client` with `maxPoolSize=100`
- Validates connection immediately
- Returns error if connection fails

### Datasource Interface

**Location:** `pkg/mongodb/datasource.mongodb.go`

```go
type Datasource interface {
    Query(ctx context.Context, collection string, fields []string,
          filter map[string][]any) ([]map[string]any, error)
    QueryWithAdvancedFilters(ctx context.Context, collection string,
          fields []string, filter map[string]job.FilterCondition) ([]map[string]any, error)
    GetDatabaseSchema(ctx context.Context) ([]CollectionSchema, error)
    CloseConnection(ctx context.Context) error
}
```

### Query()

**Parameters:**
- `ctx context.Context` - Request context with tracing
- `collection string` - Target collection name
- `fields []string` - Fields to project (`["*"]` for all)
- `filter map[string][]any` - Simple equality/IN filters

**Behavior:**
1. Builds MongoDB filter document
2. Single value → equality check
3. Multiple values → `$in` operator
4. Creates projection for specified fields
5. Executes with 10-second timeout
6. Converts BSON to Go maps

**Example:**
```go
results, err := repo.Query(ctx, "users",
    []string{"_id", "name", "email"},
    map[string][]any{"status": {"active", "pending"}})
// db.users.find({status: {$in: ["active", "pending"]}}, {_id: 1, name: 1, email: 1})
```

### QueryWithAdvancedFilters()

**Parameters:**
- Same as `Query()` but with `filter map[string]job.FilterCondition`

**Supported Operators:**

| Operator | Field | MongoDB Operator | Example |
|----------|-------|------------------|---------|
| `eq` | `Equals` | `$eq` / `$in` | `{status: {$in: ["A", "B"]}}` |
| `gt` | `GreaterThan` | `$gt` | `{amount: {$gt: 100}}` |
| `gte` | `GreaterOrEqual` | `$gte` | `{amount: {$gte: 100}}` |
| `lt` | `LessThan` | `$lt` | `{amount: {$lt: 1000}}` |
| `lte` | `LessOrEqual` | `$lte` | `{amount: {$lte: 1000}}` |
| `between` | `Between` | `$gte` + `$lte` | `{amount: {$gte: 100, $lte: 1000}}` |
| `in` | `In` | `$in` | `{status: {$in: ["A", "B"]}}` |
| `nin` | `NotIn` | `$nin` | `{status: {$nin: ["C"]}}` |
| `ne` | `NotEquals` | `$ne` / `$nin` | `{status: {$ne: "inactive"}}` |
| `like` | `Like` | `$regex` | `{name: {$regex: "^john", $options: "i"}}` |

**SQL LIKE to MongoDB Regex Conversion:**
- `%` → `.*` (any sequence)
- `_` → `.` (single character)
- Automatic anchoring (`^` / `$`)
- Case-insensitive (`$options: "i"`)

### GetDatabaseSchema()

**Parameters:**
- `ctx context.Context` - Request context

**Returns:** `[]CollectionSchema` with collection names and inferred field types

**Schema Discovery Process:**

```mermaid
graph TB
    A[ListCollections] --> B{Document Count}
    B -->|≤10k docs| C[Aggregation Pipeline]
    B -->|>10k docs| D[Statistical Sampling]
    C --> E[Field Discovery]
    D --> E
    E --> F[Sample Documents]
    F --> G[Type Inference]
    G --> H[CollectionSchema]
```

**Sampling Strategy (95% confidence, 5% margin):**

| Collection Size | Sample Size | Percentage |
|-----------------|-------------|------------|
| ≤ 1,000 | All documents | 100% |
| ≤ 10,000 | 1,000 | 10% |
| ≤ 100,000 | 2,000 | 2% |
| ≤ 1,000,000 | 5,000 | 0.5% |
| > 1,000,000 | 10,000 | 0.1% |

**Aggregation Pipelines:**

```javascript
// Field Discovery (small collections)
[
  {$limit: 1000},
  {$project: {arrayofkeyvalue: {$objectToArray: "$$ROOT"}}},
  {$unwind: "$arrayofkeyvalue"},
  {$group: {_id: null, allkeys: {$addToSet: "$arrayofkeyvalue.k"}}}
]

// Field Discovery (large collections)
[
  {$sample: {size: calculated_sample_size}},
  {$project: {arrayofkeyvalue: {$objectToArray: "$$ROOT"}}},
  {$unwind: "$arrayofkeyvalue"},
  {$group: {_id: null, allkeys: {$addToSet: "$arrayofkeyvalue.k"}}}
]
```

**Type Inference Hierarchy:**

| Type | Priority | Go Type | MongoDB Type |
|------|----------|---------|--------------|
| `objectId` | 10 | `primitive.ObjectID` | ObjectId |
| `date` | 9 | `primitive.DateTime` | Date |
| `timestamp` | 8 | `primitive.Timestamp` | Timestamp |
| `decimal` | 7 | `primitive.Decimal128` | Decimal128 |
| `binData` | 6 | `primitive.Binary` | BinData |
| `regex` | 5 | `primitive.Regex` | Regex |
| `minKey/maxKey` | 4 | `primitive.MinKey/MaxKey` | MinKey/MaxKey |
| `number` | 2 | `int/float` | Number |
| `string` | 2 | `string` | String |
| `boolean` | 2 | `bool` | Boolean |
| `array` | 2 | `bson.A` | Array |
| `object` | 2 | `bson.M/bson.D` | Object |
| `unknown` | 1 | - | Unknown |

### DataSourceConfigMongoDB

**Location:** `pkg/model/datasource/mongodb/datasource-config.go`

**Responsibility:** High-level datasource wrapper implementing `DataSource` interface.

```go
type DataSourceConfigMongoDB struct {
    datasource.DataSourceConfig           // Base config (ID, Host, Port, etc.)
    MongoDBRepository *mongodb.ExternalDataSource
    MongoURI          string
    Options           string
}
```

#### Methods

| Method | Description |
|--------|-------------|
| `GetConfig()` | Returns embedded base configuration |
| `GetType()` | Returns database type string |
| `Connect(ctx, logger)` | Sets status to available (connection pre-established) |
| `Close(ctx)` | Closes repository connection |
| `Query(ctx, tables, filters, logger)` | Multi-collection query orchestration |
| `GetSchemaInfo(ctx, schemas)` | Returns `*model.DataSourceSchema` |

## BSON Type Conversion

### convertBsonToMap()

Recursively converts MongoDB BSON documents to Go maps:

| BSON Type | Go Type | Example |
|-----------|---------|---------|
| `bson.M` | `map[string]any` | `{"key": "value"}` |
| `bson.A` | `[]any` | `["a", "b", "c"]` |
| `bson.D` | `map[string]any` | Ordered document |
| `primitive.DateTime` | `time.Time` | `2024-01-15T10:30:00Z` |
| `primitive.ObjectID` | `string` (hex) | `"507f1f77bcf86cd799439011"` |
| `primitive.Binary` (16 bytes) | `string` (UUID) | `"550e8400-e29b-41d4-a716-446655440000"` |
| `primitive.Binary` (other) | `string` (hex) | `"0x1234abcd"` |
| Other primitives | As-is | - |

## Integrations and Dependencies

### Dependency Diagram

```mermaid
graph TB
    subgraph "Consumers"
        Worker[components/worker/services/extract-data.go]
        TestConn[components/manager/services/query/test_connection.go]
        ValidateSchema[components/manager/services/query/validate_schema.go]
    end

    subgraph "MongoDB Package"
        MongoDBPkg[pkg/mongodb]
        ModelMongo[pkg/model/datasource/mongodb]
        ConnRepo[pkg/mongodb/connection]
        JobRepo[pkg/mongodb/job]
    end

    subgraph "Core Dependencies"
        DSFactory[pkg/datasource/datasource-factory.go]
        DSConfig[pkg/model/datasource/datasource-config.go]
        SSLMode[pkg/datasource/sslmode/mongodb.go]
        Model[pkg/model/connection.go]
    end

    subgraph "Shared Libraries"
        LibCommons[lib-commons/v2]
        Crypto[pkg/crypto]
    end

    Worker -->|uses| DSFactory
    TestConn -->|uses| DSFactory
    ValidateSchema -->|uses| DSFactory

    DSFactory -->|creates| ModelMongo
    DSFactory -->|validates| SSLMode
    DSFactory -->|decrypts| Crypto

    ModelMongo -->|embeds| DSConfig
    ModelMongo -->|uses| MongoDBPkg

    ConnRepo -->|stores connections| LibCommons
    JobRepo -->|stores jobs| LibCommons

    MongoDBPkg -->|logging| LibCommons
    MongoDBPkg -->|tracing| LibCommons
```

### Interfaces Implemented
- `datasource.DataSource` - Core datasource interface
- `mongodb.Repository` - MongoDB-specific repository interface

### Additional Repositories

#### ConnectionMongoDBRepository
**Location:** `pkg/mongodb/connection/connection.mongodb.go`

Stores and retrieves connection metadata:
- `Create()`, `Update()`, `Delete()` - CRUD operations
- `FindByID()`, `FindByConfigNames()` - Lookups
- `List()` - Paginated listing with filters

#### JobMongoDBRepository
**Location:** `pkg/mongodb/job/job.go`

Manages job records:
- `Create()`, `Update()`, `UpdateStatus()` - Job lifecycle
- `FindByID()`, `FindByRequestHashWithinWindow()` - Deduplication
- `ExistsRunningByMappedFieldKey()` - Conflict detection
- `List()` - Paginated job listing

## Error Handling

### Error Mapping

`MapMongoErrorToResponse()` maps MongoDB errors to application errors:

| MongoDB Error | Application Error |
|---------------|-------------------|
| `context.Canceled` | `ErrServiceUnavailable` |
| `context.DeadlineExceeded` | `ErrServiceUnavailable` |
| `mongo.IsTimeout()` | `ErrServiceUnavailable` |
| `mongo.ErrNoDocuments` | `ErrNotFound` |
| `mongo.IsDuplicateKeyError()` | `ErrConflict` |
| `mongo.IsNetworkError()` | `ErrServiceUnavailable` |
| Auth errors (13, 18) | `ErrInternalServer` |
| Unavailable (6, 7, 89, 91) | `ErrServiceUnavailable` |
| Default | `ErrInternalServer` |

### Logging and Observability

**Log Levels:**
- `INFO`: Connection status, collection discovery
- `DEBUG`: Query filters, aggregation pipelines
- `ERROR`: Connection failures, query errors
- `WARN`: Document decode errors (continues processing)

**OpenTelemetry Spans:**

| Span Name | Attributes |
|-----------|------------|
| `mongodb.data_source.query` | `request_id`, `repository_filter` |
| `mongodb.data_source.query_with_advanced_filters` | `request_id`, `repository_filter` |
| `mongodb.data_source.get_database_schema` | `request_id` |
| `datasource.mongodb.get_schema_info` | `config_name`, `type`, `tables_count` |

## Usage Examples

### Basic Query

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
        "users": {"_id", "name", "email"},
    },
    map[string]map[string]job.FilterCondition{
        "mongodb": {
            "users": {
                Equals: []any{"active"},
            },
        },
    },
    logger,
)
```

### Advanced Filtering

```go
// Complex query with multiple operators
results, err := ds.Query(ctx,
    map[string][]string{
        "orders": {"_id", "customer_id", "total", "created_at"},
    },
    map[string]map[string]job.FilterCondition{
        "mongodb": {
            "orders": {
                Between: []any{"2024-01-01", "2024-12-31"},
                GreaterThan: []any{100.00},
                Like: []any{"%premium%"},  // Converted to regex
            },
        },
    },
    logger,
)
```

### Schema Discovery

```go
// Discover schema with intelligent sampling
schema, err := ds.GetSchemaInfo(ctx, nil)
if err != nil {
    return err
}

for _, collection := range schema.Tables {
    fmt.Printf("Collection: %s\n", collection.Name)
    for _, field := range collection.Columns {
        fmt.Printf("  - %s: %s\n", field.Name, field.Type)
    }
}
```

### Health Check

```go
// Kubernetes readiness probe
// mongoProvider must implement Client(ctx) (*mongo.Client, error)
err := mongodb.PingMongo(ctx, mongoProvider, 5*time.Second)
if err != nil {
    return fmt.Errorf("MongoDB health check failed: %w", err)
}
```

## Connection URI Format

```
mongodb://[username:password@]host[:port][/database][?options]
```

**Examples:**
```go
// Standard connection
"mongodb://localhost:27017/mydb"

// With authentication
"mongodb://user:password@localhost:27017/mydb?authSource=admin"

// With SSL/TLS
"mongodb://user:password@localhost:27017/mydb?tls=true"

// Replica set
"mongodb://host1:27017,host2:27017,host3:27017/mydb?replicaSet=rs0"

// Atlas connection
"mongodb+srv://user:password@cluster.mongodb.net/mydb"
```

**Common Options:**
| Option | Description | Example |
|--------|-------------|---------|
| `authSource` | Authentication database | `admin` |
| `tls` | Enable TLS | `true` |
| `tlsInsecure` | Skip certificate verification | `true` |
| `replicaSet` | Replica set name | `rs0` |
| `readPreference` | Read preference | `secondary` |
| `maxPoolSize` | Connection pool size | `100` (hardcoded) |

**SSL/TLS Modes:**
| Mode | Description |
|------|-------------|
| `disable` | No TLS (default) |
| `false` | TLS disabled (explicit) |
| `true` | TLS enabled |
| `enable` | TLS enabled |
| `insecure` | TLS without certificate verification |

## Query Timeouts

| Operation | Timeout | Constant |
|-----------|---------|----------|
| Simple queries | 10 seconds | `QueryTimeoutMedium` |
| Advanced filter queries | 15 seconds | `QueryTimeoutSlow` |
| Schema discovery | 30 seconds | `SchemaDiscoveryTimeout` |
| Health check ping | 5 seconds | `DefaultPingTimeout` |

## Key Characteristics

| Aspect | Detail |
|--------|--------|
| **Schema handling** | Schemaless; infers structure via sampling |
| **Field projection** | `["*"]` returns all fields |
| **Filter combination** | MongoDB `$and` semantics |
| **Type inference** | Statistical sampling with priority hierarchy |
| **NULL handling** | BSON `null` preserved as Go `nil` |
| **ObjectID conversion** | Converted to hex string |
| **Binary/UUID detection** | 16-byte binaries formatted as UUID |
| **Connection pool** | Fixed at 100 connections |
| **Transaction support** | None - read-only queries |
