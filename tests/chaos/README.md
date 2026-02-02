# Chaos Tests

This directory contains chaos engineering tests for the Fetcher system. These tests validate system resilience, fault tolerance, and recovery behavior under adverse conditions using [Toxiproxy](https://github.com/Shopify/toxiproxy) for network fault injection.

## Quick Start

### Running All Chaos Tests

```bash
# Run all chaos tests (requires Docker)
go test -v -tags=chaos ./tests/chaos/... -timeout 30m

# Run with Skip Docker build, use pre-built images
GITHUB_TOKEN=`cat .secrets/github_token.txt` E2E_SKIP_BUILD=false go test -v -tags=chaos ./tests/chaos/... -timeout 30m
```

### Running Specific Test Categories

```bash
# Manager + MongoDB tests
go test -v -tags=chaos ./tests/chaos/... -run "TestManager_MongoDB" -timeout 10m

# Worker + Database tests
go test -v -tags=chaos ./tests/chaos/... -run "TestWorker_PostgreSQL" -timeout 10m

# SLO validation tests
go test -v -tags=chaos ./tests/chaos/... -run "TestSLO" -timeout 15m

# Circuit breaker tests
go test -v -tags=chaos ./tests/chaos/... -run "TestWorker_CircuitBreaker" -timeout 15m
```

### Running a Single Test

```bash
go test -v -tags=chaos ./tests/chaos/... -run "TestManager_MongoDB_HighLatency" -timeout 5m
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MANAGER_IMAGE` | `fetcher-manager:latest` | Docker image for Manager |
| `WORKER_IMAGE` | `fetcher-worker:latest` | Docker image for Worker |
| `E2E_SKIP_BUILD` | `true` | Skip Docker build, use pre-built images |
| `GITHUB_TOKEN` | `""` | GitHub token for fetching and worker images |

## Prerequisites

- Go 1.21+
- Docker with Docker Compose
- Pre-built Fetcher images (`fetcher-manager:latest`, `fetcher-worker:latest`)

To build the images before running tests:

```bash
make build-manager
make build-worker
```

## Test Architecture

### Infrastructure Components

The chaos tests spin up a complete test environment using [testcontainers-go](https://golang.testcontainers.org/):

```
┌─────────────────────────────────────────────────────────────────┐
│                        Test Environment                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐    ┌───────────┐    ┌──────────┐                 │
│  │ Manager  │◄──►│ Toxiproxy │◄──►│ MongoDB  │                 │
│  └──────────┘    └───────────┘    └──────────┘                 │
│       │              │                                          │
│       │         ┌────┴────┐                                     │
│       │         │         │                                     │
│       ▼         ▼         ▼                                     │
│  ┌──────────┐  ┌─────────┐  ┌─────────┐  ┌───────────┐        │
│  │  Worker  │  │RabbitMQ │  │  Redis  │  │ SeaweedFS │        │
│  └──────────┘  └─────────┘  └─────────┘  └───────────┘        │
│       │                                                         │
│       ▼                                                         │
│  ┌────────────┐                                                 │
│  │ PostgreSQL │ (Source Database)                               │
│  └────────────┘                                                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Chaos Injection via Toxiproxy

All infrastructure components have proxies enabled (`EnableProxy: true`) that route traffic through Toxiproxy. This allows injection of:

| Toxic Type | Description | Use Case |
|------------|-------------|----------|
| **Latency** | Adds delay to requests | Simulates network latency, slow services |
| **Bandwidth** | Limits throughput (KB/s) | Simulates congested networks |
| **Timeout** | Delays then drops connections | Simulates service unavailability |
| **Reset** | Immediately closes connections | Simulates connection failures |

## Test Categories

### 1. Manager + MongoDB Tests (`manager_mongodb_test.go`)

Tests Manager API resilience when MongoDB experiences issues.

| Test | Chaos Condition | Expected Behavior |
|------|-----------------|-------------------|
| `TestManager_MongoDB_HighLatency` | 500ms latency | Success rate >= 95%, P99 < 2s |
| `TestManager_MongoDB_Timeout` | 5s connection timeout | Graceful error handling |
| `TestManager_MongoDB_Intermittent` | Connection cut/restore | System recovery |
| `TestManager_MongoDB_Bandwidth` | 128 KB/s bandwidth | Degraded but functional |
| `TestManager_CRUD_UnderLatency` | 300ms latency | All CRUD operations complete |

### 2. Manager + RabbitMQ Tests (`manager_rabbitmq_test.go`)

Tests job creation and message delivery when RabbitMQ has issues.

| Test | Chaos Condition | Expected Behavior |
|------|-----------------|-------------------|
| `TestManager_RabbitMQ_Unavailable` | Connection cut | Graceful failure, other APIs work |
| `TestManager_RabbitMQ_HighLatency` | 1s latency | Jobs eventually complete |
| `TestManager_RabbitMQ_Recovery` | Cut/restore cycle | System recovers, jobs process |
| `TestManager_RabbitMQ_SlowConsumer` | 64 KB/s bandwidth | Jobs complete with delay |

### 3. Worker + Database Tests (`worker_database_test.go`)

Tests Worker behavior when source databases have issues.

| Test | Chaos Condition | Expected Behavior |
|------|-----------------|-------------------|
| `TestWorker_PostgreSQL_HighLatency` | 1s latency | Jobs complete within SLO |
| `TestWorker_PostgreSQL_Timeout` | 5s timeout | Graceful failure or retry |
| `TestWorker_PostgreSQL_PartialFailure` | Mid-extraction cut | Clean failure, no corruption |
| `TestWorker_Database_Recovery` | Cut/restore cycle | Jobs succeed after recovery |
| `TestWorker_Database_SlowQuery` | 64 KB/s bandwidth | Jobs complete with delay |

### 4. Worker + SeaweedFS Tests (`worker_seaweedfs_test.go`)

Tests Worker behavior when object storage has issues.

| Test | Chaos Condition | Expected Behavior |
|------|-----------------|-------------------|
| `TestWorker_SeaweedFS_Unavailable` | Connection cut | Job fails gracefully |
| `TestWorker_SeaweedFS_SlowUpload` | 56 KB/s bandwidth | Jobs complete with delay |
| `TestWorker_SeaweedFS_LatencySpike` | 2s latency | Jobs complete within extended time |
| `TestWorker_SeaweedFS_Recovery` | Cut/restore cycle | Jobs succeed after recovery |

### 5. Circuit Breaker Tests (`worker_circuitbreaker_test.go`)

Tests the circuit breaker pattern implementation.

| Test | Scenario | Expected Behavior |
|------|----------|-------------------|
| `TestWorker_CircuitBreaker_Opens` | 5+ consecutive failures | Circuit opens, rejects requests |
| `TestWorker_CircuitBreaker_HalfOpen` | After 30s cooldown | Allows probe request |
| `TestWorker_CircuitBreaker_Recovery` | Successful probe | Circuit closes, normal operation |
| `TestWorker_ExponentialBackoff` | Transient failures | Retries with backoff |

### 6. SLO Validation Tests (`slo_validation_test.go`)

Validates that the system meets defined Service Level Objectives.

| Test | Validation |
|------|------------|
| `TestSLO_ManagerAPI_UnderLatencyChaos` | Success rate >= 95%, P99 < 2s |
| `TestSLO_WorkerProcessing_UnderChaos` | Job success rate >= 90%, P99 < 60s |
| `TestSLO_SystemRecovery_AfterChaos` | Recovery time < 35s |
| `TestSLO_ErrorClassification` | Errors properly categorized |

## SLO Thresholds

The chaos tests validate the following Service Level Objectives:

### Manager API (Under Chaos)

| Metric | Threshold |
|--------|-----------|
| Success Rate | >= 95% |
| P99 Latency | < 2 seconds |

### Worker Processing (Under Chaos)

| Metric | Threshold |
|--------|-----------|
| Job Success Rate | >= 90% |
| P99 Duration | < 60 seconds |

### Recovery

| Metric | Threshold |
|--------|-----------|
| Recovery Time | < 35 seconds |

### Circuit Breaker Configuration

| Parameter | Value |
|-----------|-------|
| Failure Threshold | 5 consecutive failures |
| Cooldown Period | 30 seconds |

## Test Output

Each test produces a chaos report with metrics:

```
╔══════════════════════════════════════════════════════════════╗
║                    CHAOS TEST REPORT                         ║
╠══════════════════════════════════════════════════════════════╣

  REQUEST METRICS
     Total Requests:      100
     Successful:          100
     Failed:              0
     Timeouts:            0
     Success Rate:        100.00%

  LATENCY METRICS
     Average:             408.46ms
     Min:                 355.62ms
     P50 (median):        408.96ms
     P90:                 448.75ms
     P95:                 450.96ms
     P99:                 452.75ms

  THROUGHPUT
     Overall:             2.18 req/s
     Successful:          2.18 req/s
     During Chaos:        2.18 req/s

  DURATION
     Test Duration:       45.88s
     Chaos Duration:      45.88s

╚══════════════════════════════════════════════════════════════╝
```

## Writing New Chaos Tests

### Basic Test Structure

```go
//go:build chaos

package chaos

func TestMyComponent_ChaosCondition(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), DefaultChaosTestTimeout)
    defer cancel()

    metrics := metricskit.NewChaosMetrics()
    metrics.StartTest()

    // Phase 1: Baseline (optional)
    t.Log("Phase 1: Recording baseline...")
    verifyAPIHealthy(t, ctx, metrics, 10)

    // Phase 2: Inject chaos
    t.Log("Phase 2: Injecting chaos...")
    cleanup := injectLatency(t, ctx, mongoProxy, 500*time.Millisecond, 50*time.Millisecond)
    defer cleanup()

    metrics.StartChaos()

    // Phase 3: Run operations under chaos
    t.Log("Phase 3: Running under chaos...")
    runRequestsUnderChaos(t, ctx, metrics, 50, 100*time.Millisecond)

    metrics.EndChaos()
    metrics.EndTest()

    // Phase 4: Validate SLOs
    t.Log("Phase 4: Validating SLOs...")
    logChaosReport(t, metrics)
    assertSLOs(t, metrics, SLOManagerSuccessRate, SLOManagerP99Latency)
}
```

### Available Chaos Injection Helpers

```go
// Add latency with jitter
cleanup := injectLatency(t, ctx, proxyName, 500*time.Millisecond, 50*time.Millisecond)
defer cleanup()

// Add timeout (connection will be dropped after timeout)
cleanup := injectTimeout(t, ctx, proxyName, 5*time.Second)
defer cleanup()

// Limit bandwidth
cleanup := injectBandwidthLimit(t, ctx, proxyName, 128) // 128 KB/s
defer cleanup()

// Cut connection completely
cleanup := cutConnection(t, ctx, proxyName)
defer cleanup()

// Restore connection (remove all toxics)
restoreConnection(t, ctx, proxyName)
```

### Available Proxy Names

| Proxy | Target |
|-------|--------|
| `mongoProxy` | MongoDB (`mongo-fetcher-chaos`) |
| `rabbitProxy` | RabbitMQ (`amqp-fetcher-chaos`) |
| `redisProxy` | Redis (`redis-fetcher-chaos`) |
| `seaweedProxy` | SeaweedFS (`seaweed-fetcher-chaos`) |
| `postgresProxy` | PostgreSQL (`pg-source`) |

## Troubleshooting

### Tests Timeout

- Increase the timeout: `-timeout 30m`
- Check Docker resources (memory, CPU)
- Verify Docker images are built and available

### Connection Refused Errors

- Ensure Docker daemon is running
- Check that required ports are available
- Verify no conflicting containers are running: `docker ps`

### Flaky Tests

- Chaos tests are inherently non-deterministic
- Run multiple times to verify consistency
- Check system resources during test execution

### Viewing Container Logs

```bash
# List running containers
docker ps

# View logs for a specific container
docker logs <container_id>

# Follow logs in real-time
docker logs -f <container_id>
```

## Best Practices

1. **Use unique names** for resources (connections, jobs) to avoid conflicts between tests
2. **Use `t.Cleanup()`** for resource cleanup to ensure cleanup runs even on test failure
3. **Follow the 4-phase pattern**: Baseline → Inject Chaos → Run Under Chaos → Validate SLOs
4. **Always defer cleanup functions** returned by chaos injection helpers
5. **Use polling for job completion** (`waitForJobCompletionPolling`) when RabbitMQ is under chaos
6. **Log chaos reports** at the end of each test for debugging
7. **Remove existing toxics** before injecting new ones to avoid conflicts
8. **Set appropriate timeouts** - chaos tests take longer than normal tests

## Related Documentation

- [E2E Tests](../e2e/README.md) - End-to-end functional tests
- [itestkit Package](../../pkg/itestkit/README.md) - Integration test infrastructure
- [metricskit Addon](../../pkg/itestkit/addons/metricskit/) - Chaos metrics collection
