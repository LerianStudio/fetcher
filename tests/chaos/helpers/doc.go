// Package helpers provides utilities for chaos testing including:
//   - metrics.go: Collection and analysis of chaos test metrics
//   - assertions.go: Custom assertions for validating chaos test outcomes
//
// Example usage:
//
//	// Record metrics during chaos test
//	metrics := helpers.NewChaosMetrics()
//	metrics.StartTest()
//	metrics.StartChaos()
//	metrics.RecordRequest(success, timeout, latency)
//	metrics.EndChaos()
//	metrics.EndTest()
//
//	// Assert results
//	assertions := helpers.NewChaosAssertions(t, metrics)
//	assertions.AssertSuccessRateAbove(80.0)
//	assertions.AssertRecoveryWithin(30 * time.Second)
//
// For Toxiproxy chaos injection, use the functions in:
//
//	github.com/LerianStudio/fetcher/tests/shared/containers
//
// Example:
//
//	// Inject latency chaos
//	toxic, err := containers.AddLatency(proxy, "latency-test", 5000, 1000)
//
//	// Remove chaos
//	err = containers.RemoveToxic(proxy, "latency-test")
package helpers
