// Package chaos provides a generic chaos testing framework with metrics collection,
// SLA validation, and error classification. It is designed to be framework-agnostic
// and usable by any project.
//
// # Component Tiers
//
// The package provides two tiers of components:
//
// Core Components (basic chaos testing):
//   - ChaosMetrics: Thread-safe metrics collection with percentile calculations
//   - ChaosAssertions: Test assertions for SLA validation
//   - SLAThresholds: Predefined threshold presets for different chaos scenarios
//   - ErrorClassifier: Error categorization (timeout, connection, network, application)
//   - ChaosInjectionConfig: Configuration builders for Toxiproxy toxics
//
// Extended Components (adds recovery/stability tracking):
//   - ExtendedMetrics: ChaosMetrics + recovery time and stability check tracking
//   - ExtendedAssertions: ChaosAssertions + recovery/stability validation
//   - ExtendedSLAThresholds: SLAThresholds + stability requirements
//
// # Usage - Core Only
//
// For basic chaos testing without recovery tracking:
//
//	metrics := chaos.NewChaosMetrics()
//	metrics.StartTest()
//	metrics.RecordRequest(true, false, 100*time.Millisecond)
//	assertions := chaos.NewChaosAssertions(t, metrics)
//	assertions.AssertSuccessRateAbove(90.0)
//
// # Usage - Extended (with recovery/stability)
//
// For full chaos testing with recovery and stability tracking:
//
//	metrics := chaos.NewExtendedMetrics()
//	metrics.StartTest()
//	// ... inject chaos, record requests ...
//	metrics.StartRecovery()
//	// ... wait for recovery ...
//	metrics.EndRecovery()
//	assertions := chaos.NewExtendedAssertions(t, metrics)
//	assertions.AssertRecoveryWithin(30 * time.Second)
//	assertions.AssertStabilityMaintained(95.0, 3)
package chaos
