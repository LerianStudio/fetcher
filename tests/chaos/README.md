# Chaos E2E Tests

This directory contains chaos engineering tests for validating the Fetcher system's resilience under controlled failure scenarios using [Toxiproxy](https://github.com/Shopify/toxiproxy).

## Key Principle: API-First Testing

**All tests interact with the system through the Manager API**, exactly like production clients. This ensures we're testing real system behavior, not mocked scenarios.

```
Test Flow:
1. Create connection via API -> 2. Create job via API ->
3. [INJECT CHAOS] -> 4. Wait for job event ->
5. [REMOVE CHAOS] -> 6. Verify recovery via API
```

## Architecture

```
tests/chaos/
├── setup/
│   ├── infrastructure.go    # ChaosInfrastructure with Toxiproxy
│   ├── constants.go         # Chaos values, timing constants
│   └── timeouts.go          # Infrastructure & chaos timing
├── helpers/
│   ├── doc.go               # Package documentation
│   ├── metrics.go           # ChaosMetrics - thread-safe metrics collection
│   ├── assertions.go        # ChaosAssertions - SLA validation
│   ├── chaos.go             # Chaos injection helpers
│   ├── errors.go            # ErrorClassifier - error categorization
│   ├── thresholds.go        # SLAThresholds - SLA definitions
│   ├── metrics_test.go      # 58 unit tests
│   └── errors_test.go       # Error classifier tests
└── e2e/
    ├── suite_test.go              # ChaosTestSuite base class
    ├── manager_rabbitmq_test.go   # RabbitMQ event broker chaos
    ├── manager_mongodb_test.go    # MongoDB fallback storage chaos
    ├── manager_redis_test.go      # Redis rate limiting fallback
    ├── worker_postgres_test.go    # PostgreSQL extraction chaos
    ├── worker_mysql_test.go       # MySQL extraction chaos
    ├── worker_sqlserver_test.go   # SQL Server extraction chaos
    ├── worker_oracle_test.go      # Oracle extraction chaos
    ├── worker_mongodb_test.go     # MongoDB external extraction chaos
    ├── worker_seaweedfs_test.go   # SeaweedFS storage chaos
    └── full_flow_test.go          # Multi-point chaos scenarios
```

## Key Abstractions

### ChaosMetrics (`helpers/metrics.go`)

Thread-safe metrics collection with automatic percentile caching:

```go
// Recording metrics
s.metrics.RecordRequest(success, timeout, latency)
s.metrics.RecordError(errMsg)

// Lifecycle tracking
s.metrics.StartChaos() / EndChaos()
s.metrics.StartRecovery() / EndRecovery()
s.metrics.StartStabilityCheck() / EndStabilityCheck()

// Query metrics
s.metrics.SuccessRate()           // Percentage
s.metrics.Percentile(99)          // P99 latency
s.metrics.ThroughputRPS()         // Requests per second
s.metrics.GetRecoveryTime()       // Recovery duration
```

### ChaosAssertions (`helpers/assertions.go`)

Custom assertions for chaos testing:

```go
assertions := helpers.NewChaosAssertions(t, s.metrics)

// Success rate assertions
assertions.AssertSuccessRateAbove(90.0)
assertions.AssertNoFailures()

// Latency assertions (percentiles)
assertions.AssertP95Within(500 * time.Millisecond)
assertions.AssertP99Within(1 * time.Second)

// Recovery assertions
assertions.AssertRecoveryWithin(30 * time.Second)
assertions.AssertSteadyStateRestored(baseline, 5.0)

// SLA validation
result := assertions.ValidateAgainstSLA(helpers.DefaultSLAThresholds())
assertions.AssertSLAMet(helpers.StrictSLAThresholds())
```

### SLAThresholds (`helpers/thresholds.go`)

Predefined SLA configurations:

| Preset | During Chaos | After Recovery | Use Case |
|--------|-------------|----------------|----------|
| `DefaultSLAThresholds()` | 50% success | 99% success | General chaos |
| `StrictSLAThresholds()` | 80% success | 99.9% success | Production-like |
| `LatencyChaosThresholds()` | 90% success | 99% success | Latency injection |
| `TimeoutChaosThresholds()` | 0% success | 99% success | Expected failures |
| `BandwidthChaosThresholds()` | 70% success | 99% success | Bandwidth limiting |

### ErrorClassifier (`helpers/errors.go`)

Categorizes errors during chaos:

```go
// Categories: Timeout, Connection, Network, Application, Unknown
category := helpers.ClassifyError(errMsg)
s.metrics.RecordError(errMsg)  // Auto-classifies

// Query error breakdown
counts := s.metrics.ErrorClassifier.GetCategoryCounts()
assertions.AssertConnectionErrorsExpected()
```

## Running Tests

**Prerequisites:** Either set `MANAGER_IMAGE` (local build) or `GITHUB_TOKEN` (pull from registry):

```bash
# Option A: Use pre-built local image
export MANAGER_IMAGE=fetcher-manager:local

# Option B: Use GitHub token to pull image
export GITHUB_TOKEN=<your_token>
```

```bash
# Full suite (~45 minutes)
make test-chaos

# Verbose output
make test-chaos-verbose

# Quick tests - latency only (~20 minutes)
make test-chaos-quick

# Single test (uses testify/suite - prefix with TestChaosE2E/)
go test -v -tags=chaos -run "TestChaosE2E/TestPostgresLatency" ./tests/chaos/e2e/...
# or
export GITHUB_TOKEN=<your_token> && go test -v -tags=chaos -run "TestChaosE2E/TestPostgresLatency" ./tests/chaos/e2e/...

# Multiple tests matching pattern
go test -v -tags=chaos -run "TestChaosE2E/TestPostgres" ./tests/chaos/e2e/...
# or
export GITHUB_TOKEN=<your_token> && go test -v -tags=chaos -run "TestChaosE2E/TestPostgres" ./tests/chaos/e2e/...

# Unit tests only (no Docker required)
go test -v ./tests/chaos/helpers/...
```

## Chaos Types

| Type | Function | Description |
|------|----------|-------------|
| `latency` | `DefaultLatencyConfig(ms, jitter)` | Add network delay with jitter |
| `timeout` | `DefaultTimeoutConfig(ms)` | Close connection after delay |
| `bandwidth` | `DefaultBandwidthConfig(bytesPerSec)` | Limit throughput |
| `reset_peer` | `DefaultResetPeerConfig(ms)` | Reset TCP connection |
| `slow_close` | `DefaultSlowCloseConfig(ms)` | Delayed connection closure |
| `limit_data` | `DefaultLimitDataConfig(bytes)` | Limit bytes before close |
| `slicer` | `DefaultSlicerConfig(size, var, delay)` | Packet fragmentation |

## Test Structure Pattern

All tests follow a 5-phase pattern:

```go
func (s *ChaosTestSuite) TestComponent_Scenario() {
    t := s.T()

    // Document hypothesis
    helpers.DocumentHypothesis(t, helpers.FormatHypothesis(
        "complete job with increased latency",
        "PostgreSQL has 500ms network delay",
    ))

    // Phase 1: Setup (create connections before chaos)
    configName := s.uniqueConfigName("chaos_postgres")
    pg := s.chaosInfra.PostgresProxyInternal()
    _, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{...})

    // Phase 2: Inject chaos
    s.metrics.StartChaos()
    proxy := s.chaosInfra.GetPostgresProxy()
    chaosConfig := helpers.DefaultLatencyConfig(500, 100)
    toxic, err := helpers.InjectChaos(proxy, chaosConfig)
    defer helpers.RemoveChaos(proxy, chaosConfig.Name)
    time.Sleep(setup.StabilizationDelay)

    // Phase 3: Test under chaos
    jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, ...)
    notification, err := s.eventConsumer.WaitForJobEvent(...)
    s.metrics.RecordRequest(true, false, duration)
    s.metrics.EndChaos()

    // Phase 4: Remove chaos & verify recovery
    helpers.RemoveChaos(proxy, chaosConfig.Name)
    s.metrics.StartRecovery()
    time.Sleep(setup.RecoveryObservationTime)
    // ... verify recovery job succeeds ...
    s.metrics.EndRecovery()

    // Phase 5: Assert & document
    assertions := helpers.NewChaosAssertions(t, s.metrics)
    assertions.AssertRecoveryWithin(30 * time.Second)
    helpers.DocumentResult(t, s.metrics, "Job completed despite latency")
}
```

## API Usage

### Database Connection Info

Use **method calls**, not field access:

```go
// Proxied connections (through Toxiproxy)
pg := s.chaosInfra.PostgresProxyInternal()
mysql := s.chaosInfra.MySQLProxyInternal()
mssql := s.chaosInfra.SQLServerProxyInternal()
oracle := s.chaosInfra.OracleProxyInternal()
mongo := s.chaosInfra.MongoExternalProxyInternal()

// Direct connections (bypass proxy - use for baseline)
pg := s.chaosInfra.PostgresInternal()
```

### Proxy Access

Use **specific getter methods**:

```go
// Database proxies
proxy := s.chaosInfra.GetPostgresProxy()
proxy := s.chaosInfra.GetMySQLProxy()
proxy := s.chaosInfra.GetSQLServerProxy()
proxy := s.chaosInfra.GetOracleProxy()

// Infrastructure proxies
proxy := s.chaosInfra.GetRabbitMQProxy()
proxy := s.chaosInfra.GetRedisProxy()
proxy := s.chaosInfra.GetMongoMainProxy()      // Manager state
proxy := s.chaosInfra.GetMongoExternalProxy()  // External extraction
proxy := s.chaosInfra.GetSeaweedFSProxy()
proxy := s.chaosInfra.GetManagerProxy()
```

### Convenience Methods

```go
// Enable/disable entire proxy
s.chaosInfra.DisablePostgres()
s.chaosInfra.EnablePostgres()

// Add chaos directly
s.chaosInfra.AddPostgresLatency("latency", 500, 100)
s.chaosInfra.AddSeaweedFSTimeout("timeout", 5000)
s.chaosInfra.AddSeaweedFSBandwidth("bandwidth", 10240)

// Cleanup
s.chaosInfra.RemoveAllToxics()    // Clear all chaos
s.chaosInfra.EnableAllProxies()   // Restore connectivity
s.chaosInfra.ResetChaos()         // Both
```

## Constants Reference

### Timing Constants (`setup/constants.go`)

| Constant | Value | Purpose |
|----------|-------|---------|
| `ChaosInfraStartupTimeout` | 10 min | Infrastructure startup |
| `ManagerReadyTimeout` | 2 min | Manager API availability |
| `JobCompletionTimeout` | 2 min | Normal job completion |
| `JobCompletionTimeoutSlow` | 5 min | Bandwidth-limited jobs |
| `StabilizationDelay` | 2 sec | Wait after chaos injection |
| `RecoveryObservationTime` | 5 sec | Wait after chaos removal |

### Chaos Values

```go
setup.ChaosLatencyValues.Low      // 500ms
setup.ChaosLatencyValues.Medium   // 3s
setup.ChaosLatencyValues.High     // 5s
setup.ChaosLatencyValues.Jitter   // 500ms

setup.ChaosTimeoutValues.Short    // 5s
setup.ChaosTimeoutValues.Medium   // 15s
setup.ChaosTimeoutValues.Long     // 30s

setup.ChaosBandwidthValues.Low    // 1 KB/s
setup.ChaosBandwidthValues.Medium // 10 KB/s
setup.ChaosBandwidthValues.High   // 100 KB/s
```

## Prerequisites

- Docker daemon running
- 8GB+ RAM available
- `tests/shared/` infrastructure complete
- Build with `-tags=chaos` flag

## Differences from Integration Tests

| Aspect | Integration Tests | Chaos Tests |
|--------|------------------|-------------|
| Focus | Correct behavior | Resilience under failure |
| Infrastructure | Direct connections | Via Toxiproxy proxies |
| Execution time | ~5-10 min | ~30-45 min |
| Build tag | `integration` | `chaos` |
| Package | `containers` | `e2e` |
| Metrics | Basic pass/fail | Percentiles, SLA validation |

## Test Coverage

| Component | Scenarios Tested |
|-----------|-----------------|
| **RabbitMQ** | Latency, timeout, circuit breaker recovery |
| **MongoDB** | Main (Manager state), External (extraction) |
| **Redis** | Rate-limiting fallback, health check recovery |
| **PostgreSQL** | Latency, timeout, connection reset |
| **MySQL** | Timeout handling, recovery |
| **SQL Server** | Latency injection, extraction under chaos |
| **Oracle** | Connection resilience, timeout handling |
| **SeaweedFS** | File upload latency, timeout, bandwidth, reset_peer |
| **Full Flow** | Multi-point chaos, concurrent failures |
