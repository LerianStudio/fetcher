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
│   ├── infrastructure.go         # ChaosInfrastructure with Toxiproxy
│   └── timeouts.go               # Infrastructure & chaos timing
├── suite_test.go                 # ChaosTestSuite base class
├── manager_rabbitmq_test.go      # RabbitMQ event broker chaos
├── manager_mongodb_test.go       # MongoDB fallback storage chaos
├── manager_redis_test.go         # Redis rate limiting fallback
├── worker_postgres_test.go       # PostgreSQL extraction chaos
├── worker_mysql_test.go          # MySQL extraction chaos
├── worker_sqlserver_test.go      # SQL Server extraction chaos
├── worker_oracle_test.go         # Oracle extraction chaos
├── worker_mongodb_test.go        # MongoDB external extraction chaos
├── worker_seaweedfs_test.go      # SeaweedFS storage chaos
└── full_flow_test.go             # Multi-point chaos scenarios

tests/shared/chaos/               # Shared chaos utilities
├── doc.go                        # Package documentation
├── constants.go                  # Chaos values, timing constants
├── config.go                     # ChaosInjectionConfig builders
├── injection.go                  # Chaos injection helpers
├── operations.go                 # ChaosOperations facade
├── service_registry.go           # ProxyRegistry for type-safe access
├── proxy_router.go               # ProxyRouter for connection routing
├── metrics.go                    # ChaosMetrics - thread-safe collection
├── assertions.go                 # ChaosAssertions - SLA validation
├── errors.go                     # ErrorClassifier - error categorization
├── thresholds.go                 # SLAThresholds - SLA definitions
└── *_test.go                     # Unit tests for all components
```

## Key Abstractions

### ChaosMetrics (`tests/shared/chaos/metrics.go`)

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

### ChaosAssertions (`tests/shared/chaos/assertions.go`)

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

### SLAThresholds (`tests/shared/chaos/thresholds.go`)

Predefined SLA configurations:

| Preset | During Chaos | After Recovery | Use Case |
|--------|-------------|----------------|----------|
| `DefaultSLAThresholds()` | 50% success | 99% success | General chaos |
| `StrictSLAThresholds()` | 80% success | 99.9% success | Production-like |
| `LatencyChaosThresholds()` | 90% success | 99% success | Latency injection |
| `TimeoutChaosThresholds()` | 0% success | 99% success | Expected failures |
| `BandwidthChaosThresholds()` | 70% success | 99% success | Bandwidth limiting |

### ErrorClassifier (`tests/shared/chaos/errors.go`)

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
go test -v ./tests/shared/chaos/...
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
    pg := s.chaosInfra.ProxyRouter.GetProxyConnection(chaos.ServicePostgres, s.chaosInfra.PostgresInternal())
    _, err := s.managerClient.CreateConnection(s.ctx, client.ConnectionInput{...})

    // Phase 2: Inject chaos
    s.metrics.StartChaos()
    chaosConfig := chaos.DefaultLatencyConfig(500, 100)
    toxic, err := s.chaosInfra.ChaosOps.AddChaos(chaos.ServicePostgres, chaosConfig)
    defer s.chaosInfra.ChaosOps.RemoveChaos(chaos.ServicePostgres, chaosConfig.Name)
    time.Sleep(chaos.StabilizationDelay)

    // Phase 3: Test under chaos
    jobResp, err := s.managerClient.CreateFetcherJob(s.ctx, ...)
    notification, err := s.eventConsumer.WaitForJobEvent(...)
    s.metrics.RecordRequest(true, false, duration)
    s.metrics.EndChaos()

    // Phase 4: Remove chaos & verify recovery
    s.chaosInfra.ChaosOps.RemoveChaos(chaos.ServicePostgres, chaosConfig.Name)
    s.metrics.StartRecovery()
    time.Sleep(chaos.RecoveryObservationTime)
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

Use **ProxyRouter** for connections through Toxiproxy:

```go
// Proxied connections (through Toxiproxy for chaos injection)
pg := s.chaosInfra.ProxyRouter.GetProxyConnection(chaos.ServicePostgres, s.chaosInfra.PostgresInternal())
mysql := s.chaosInfra.ProxyRouter.GetProxyConnection(chaos.ServiceMySQL, s.chaosInfra.MySQLInternal())
mssql := s.chaosInfra.ProxyRouter.GetProxyConnection(chaos.ServiceSQLServer, s.chaosInfra.SQLServerInternal())
oracle := s.chaosInfra.ProxyRouter.GetProxyConnection(chaos.ServiceOracle, s.chaosInfra.OracleInternal())
mongo := s.chaosInfra.ProxyRouter.GetProxyConnection(chaos.ServiceMongoExternal, s.chaosInfra.MongoExternalInternal())

// Direct connections (bypass proxy - use for baseline)
pg := s.chaosInfra.PostgresInternal()
```

### Proxy Access

Use **ProxyRegistry** for type-safe proxy access:

```go
// Database proxies
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServicePostgres)
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServiceMySQL)
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServiceSQLServer)
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServiceOracle)

// Infrastructure proxies
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServiceRabbitMQ)
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServiceRedis)
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServiceMongoMain)      // Manager state
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServiceMongoExternal)  // External extraction
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServiceSeaweedFS)
proxy := s.chaosInfra.ProxyRegistry.GetProxy(chaos.ServiceManager)
```

### Chaos Operations (via ChaosOps)

```go
// Enable/disable entire proxy
s.chaosInfra.ChaosOps.DisableService(chaos.ServicePostgres)
s.chaosInfra.ChaosOps.EnableService(chaos.ServicePostgres)

// Add chaos with config builders
latencyConfig := chaos.DefaultLatencyConfig(500, 100)
s.chaosInfra.ChaosOps.AddChaos(chaos.ServicePostgres, latencyConfig)

timeoutConfig := chaos.DefaultTimeoutConfig(5000)
s.chaosInfra.ChaosOps.AddChaos(chaos.ServiceSeaweedFS, timeoutConfig)

bandwidthConfig := chaos.DefaultBandwidthConfig(10240)
s.chaosInfra.ChaosOps.AddChaos(chaos.ServiceSeaweedFS, bandwidthConfig)

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
chaos.ChaosLatencyValues.Low      // 500ms
chaos.ChaosLatencyValues.Medium   // 3s
chaos.ChaosLatencyValues.High     // 5s
chaos.ChaosLatencyValues.Jitter   // 500ms

chaos.ChaosTimeoutValues.Short    // 5s
chaos.ChaosTimeoutValues.Medium   // 15s
chaos.ChaosTimeoutValues.Long     // 30s

chaos.ChaosBandwidthValues.Low    // 1 KB/s
chaos.ChaosBandwidthValues.Medium // 10 KB/s
chaos.ChaosBandwidthValues.High   // 100 KB/s
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
