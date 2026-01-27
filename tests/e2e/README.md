# E2E Tests

End-to-end tests for the Fetcher application using the `itestkit` framework.

## Quick Start

```bash
# Run all e2e tests
go test -v -tags e2e ./tests/e2e -timeout 5m

# Run with fixed ports (useful for debugging)
FIXED_PORT=true go test -v -tags e2e ./tests/e2e -timeout 5m

# Run with Skip Docker build, use pre-built images
GITHUB_TOKEN=... E2E_SKIP_BUILD=false go test -v -tags e2e ./tests/e2e -timeout 5m

# Run a specific test
go test -v -tags e2e ./tests/e2e -run TestPostgresExtraction -timeout 5m
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FIXED_PORT` | `false` | Use fixed ports for infrastructure (27017, 5672, 6379, 8333) |
| `MANAGER_IMAGE` | `fetcher-manager:latest` | Docker image for Manager |
| `WORKER_IMAGE` | `fetcher-worker:latest` | Docker image for Worker |
| `E2E_SKIP_BUILD` | `true` | Skip Docker build, use pre-built images |
| `GITHUB_TOKEN` | `""` | GitHub token for fetching and worker images |

## Project Structure

```
tests/e2e/
├── main_test.go              # TestMain setup/teardown
├── postgres_test.go          # Postgres extraction tests
├── README.md                 # This file
└── shared/
    ├── infra.go              # CoreInfra (MongoDB, RabbitMQ, Redis, SeaweedFS)
    ├── apps.go               # StartManager, StartWorker helpers
    ├── client.go             # ManagerClient API wrapper
    └── testdata/
        └── definitions.json  # RabbitMQ queue/exchange definitions

pkg/itestkit/                 # Test infrastructure framework
├── suite.go                  # Suite builder
├── infra.go                  # Infra interface
├── hostport.go               # Host normalization utilities
├── infra/                    # Infrastructure components
│   ├── mongodb/
│   ├── postgres/
│   ├── rabbitmq/
│   ├── redis/
│   └── seaweedfs/
└── addons/
    ├── e2ekit/               # App container builder
    ├── queuekit/             # Queue consumer/assertions
    └── metricskit/           # Metrics assertions
```

## Creating New Tests

### 1. Test File Structure

Create a new file `*_test.go` with the `e2e` build tag:

```go
//go:build e2e

package extraction

import (
    "testing"
    // imports...
)

func TestMyFeature(t *testing.T) {
    t.Parallel()  // Required: enables parallel execution

    // 1. Setup: Create resources via API
    // 2. Execute: Trigger the operation
    // 3. Wait: Use queuekit to wait for events
    // 4. Assert: Verify results
    // 5. Cleanup: Delete resources (use t.Cleanup)
}
```

> **Important:** All tests must call `t.Parallel()` as the first line to enable parallel execution. This significantly reduces total test time by running tests concurrently.

### 2. Using CoreInfra

The `CoreInfra` provides shared infrastructure (MongoDB, RabbitMQ, Redis, SeaweedFS):

```go
// Access from global variables set in main_test.go
mongoURI, _ := coreInfra.MongoDB.URI()
amqpURL, _ := coreInfra.RabbitMQ.AMQPURL()
redisAddr, _ := coreInfra.Redis.Addr()
seaweedURL, _ := coreInfra.SeaweedFS.URL()
```

### 3. Adding Additional Infrastructure

Add a new database (e.g., PostgreSQL) for your test:

```go
func TestWithPostgres(t *testing.T) {
    // Create in main_test.go setup or within test
    pgInfra := postgres.NewPostgresInfra(postgres.PostgresConfig{
        Name:     "mytest",
        Database: "testdb",
        Username: "user",
        Password: "pass",
        Options: []postgres.PostgresOption{
            postgres.WithPGInitFile(fixturesPath("init.sql"), "init.sql"),
        },
    })

    // Start it
    err := pgInfra.Start(ctx, nil)
    require.NoError(t, err)
    t.Cleanup(func() { pgInfra.Terminate(ctx) })

    // Get connection info
    host, port, _ := pgInfra.HostPort()
    dsn, _ := pgInfra.DSN()
}
```

### 4. Using the API Client

The `ManagerClient` wraps HTTP calls to the Manager API:

```go
// Create a connection
connReq := shared.CreateConnectionRequest{
    Name:     "my-connection",
    Host:     host,
    Port:     port,
    Database: "testdb",
    Username: "user",
    Password: "pass",
    Type:     "postgres",
}
conn, err := apiClient.CreateConnection(ctx, connReq)
require.NoError(t, err)
t.Cleanup(func() { apiClient.DeleteConnection(ctx, conn.ID) })

// Create a job
jobReq := shared.CreateFetcherRequest{
    ConnectionID: conn.ID,
    Query:        "SELECT * FROM users",
    MappedFields: []shared.MappedField{{Field: "id"}, {Field: "name"}},
}
job, err := apiClient.CreateJob(ctx, jobReq)
require.NoError(t, err)
```

### 5. Waiting for Queue Events

Use `queuekit` to wait for job completion events:

```go
import "github.com/LerianStudio/fetcher/pkg/itestkit/addons/queuekit"

// Create consumer builder
builder := queuekit.NewAMQPConsumerBuilder(amqpURL).
    FromQueue("reporter.fetcher-notifications.queue").
    WithAutoAck(true)

// Wait for message matching job ID
result, err := builder.
    WithMatcher(queuekit.MatchJSONField("jobId", job.ID)).
    WithTimeout(90 * time.Second).
    WaitForMessage(ctx)

require.NoError(t, err)
require.False(t, result.TimedOut)

// Parse and assert
var notification shared.JobNotification
err = json.Unmarshal(result.Messages[0].Body, &notification)
require.NoError(t, err)
assert.Equal(t, "completed", notification.Status)
```

### 6. Available Matchers

```go
// JSON field matching (supports dot notation)
queuekit.MatchJSONField("jobId", "123")
queuekit.MatchJSONField("user.id", 42)

// Routing key
queuekit.MatchRoutingKey("job.completed")
queuekit.MatchRoutingKeyPrefix("job.")
queuekit.MatchRoutingKeyPattern(`job\.\w+\.done`)

// Headers
queuekit.MatchHeader("x-type", "notification")
queuekit.MatchHeaderExists("x-correlation-id")

// Body content
queuekit.MatchBodyContains("success")
queuekit.MatchBodyPattern(`"status":\s*"completed"`)

// Combinators
queuekit.MatchAll(matcher1, matcher2)  // AND
queuekit.MatchAny(matcher1, matcher2)  // OR
queuekit.MatchNone(matcher)            // NOT
```

## Infrastructure Components

### Available Infra Types

| Component | Package | Default Port | Config Struct |
|-----------|---------|--------------|---------------|
| MongoDB | `infra/mongodb` | 27017 | `MongoDBConfig` |
| PostgreSQL | `infra/postgres` | 5432 | `PostgresConfig` |
| MySQL | `infra/mysql` | 3306 | `MySQLConfig` |
| MSSQL | `infra/mssql` | 1433 | `MSSQLConfig` |
| Oracle | `infra/oracle` | 1521 | `OracleConfig` |
| RabbitMQ | `infra/rabbitmq` | 5672 | `RabbitConfig` |
| Redis | `infra/redis` | 6379 | `RedisConfig` |
| SeaweedFS | `infra/seaweedfs` | 8888 | `SeaweedFSConfig` |

### Common Options

Each infrastructure supports functional options:

```go
// Custom image
mongodb.WithMongoDBImage("mongo:6")

// Environment variables
mongodb.WithMongoDBEnv("MONGO_INITDB_ROOT_USERNAME", "admin")

// Fixed port (when FIXED_PORT=true is set)
mongodb.WithMongoDBFixedPort("27017")
```

## e2ekit: Application Container Builder

Build and run application containers:

```go
import "github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit"

app, err := e2ekit.New().
    WithContext(ctx).
    WithImage("my-app:latest").
    WithEnv(map[string]string{
        "DATABASE_URL": dsn,
        "LOG_LEVEL":    "debug",
    }).
    ExposePort(8080).
    WithWait(e2ekit.WaitHTTP(8080, "/health", 60*time.Second)).
    Run()

// Access the running app
baseURL := app.BaseURL  // http://host:port
```

### Wait Strategies

```go
// Wait for HTTP endpoint
e2ekit.WaitHTTP(port, "/health", timeout)

// Wait for log message
e2ekit.WaitLog("Server started", timeout)

// Wait for port to be listening
e2ekit.WaitPort(port, timeout)

// Minimal wait (just container running)
e2ekit.WaitRunning(timeout)
```

### Building from Dockerfile

```go
app, err := e2ekit.New().
    WithContext(ctx).
    WithDockerfile(e2ekit.DockerfileBuild{
        Context:    e2ekit.ProjectRoot(),
        Dockerfile: "Dockerfile",
        Target:     "production",
        Secrets: []e2ekit.BuildSecret{
            {ID: "github_token", Env: "GITHUB_TOKEN"},
        },
    }).
    WithEnv(env).
    ExposePort(4006).
    WithWait(e2ekit.WaitHTTP(4006, "/health", 60*time.Second)).
    Run()
```

## Troubleshooting

### Tests fail with "port already in use"

When using `FIXED_PORT=true`, ensure no other services are using the ports:
- MongoDB: 27017
- RabbitMQ: 5672
- Redis: 6379
- SeaweedFS: 8333

### Container can't connect to host services

The framework automatically rewrites `localhost` to `host.docker.internal` for container environments. If issues persist, check:

```go
// Get normalized host
host := itestkit.NormalizeHost("localhost")
```

### Tests timeout waiting for events

1. Increase timeout in `WithTimeout()`
2. Check if the worker is processing messages (look at logs)
3. Verify queue names match between publisher and consumer

### Build fails with "GITHUB_TOKEN not set"

Set `E2E_SKIP_BUILD=true` to use pre-built images, or export `GITHUB_TOKEN`:

```bash
export GITHUB_TOKEN=your_token
go test -v -tags e2e ./tests/e2e -timeout 5m
```

### Cleanup: Remove orphan containers

```bash
docker ps -a | grep -E "(mongo|rabbit|redis|seaweed|postgres)" | awk '{print $1}' | xargs docker rm -f
```

## Best Practices

1. **Always use `t.Parallel()`** as the first line in every test to enable concurrent execution
2. **Use `t.Cleanup()`** for resource cleanup to ensure cleanup runs even on test failure
3. **Use unique names** for resources (connections, jobs) to avoid conflicts in parallel tests
4. **Prefer random ports** (default) for CI/CD environments
5. **Use fixed ports** only for local debugging
6. **Keep tests independent** - each test should set up its own state
7. **Use meaningful assertions** - verify both success conditions and error messages
8. **Use matchers with unique IDs** - filter queue messages by job ID to receive only your test's events
