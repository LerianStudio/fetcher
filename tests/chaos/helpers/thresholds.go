package helpers

import "time"

// SLAThresholds defines Service Level Agreement thresholds for chaos tests.
// These thresholds represent the minimum acceptable performance during chaos conditions.
//
// Note: Some fields are validated in ValidateAgainstSLA(), others are reserved for future phases:
//   - MinSuccessRate, MaxP50/P95/P99/P999Latency, MaxRecoveryTime: Validated in Phase 2
//   - RecoverySuccessRate: Use AssertRecoverySuccessRate() separately after chaos removal
//   - StabilityDuration, StabilityCheckCount, MaxConsecutiveFailures: Phase 3 (Recovery Stability)
//   - MinThroughputRPS: Phase 5 (Throughput Metrics)
type SLAThresholds struct {
	// Success rate thresholds (percentage 0-100)
	MinSuccessRate      float64 // Minimum acceptable success rate during chaos
	RecoverySuccessRate float64 // Required success rate after recovery (use AssertRecoverySuccessRate)

	// Latency thresholds
	MaxP50Latency  time.Duration // Maximum acceptable P50 latency
	MaxP95Latency  time.Duration // Maximum acceptable P95 latency
	MaxP99Latency  time.Duration // Maximum acceptable P99 latency
	MaxP999Latency time.Duration // Maximum acceptable P99.9 latency (warning only, see ValidateAgainstSLA)

	// Recovery thresholds
	MaxRecoveryTime        time.Duration // Maximum time to recover after chaos removal
	StabilityDuration      time.Duration // Duration system must remain stable after recovery (Phase 3)
	StabilityCheckCount    int           // Number of stability checks to perform (Phase 3)
	MaxConsecutiveFailures int           // Max consecutive failures during stability check (Phase 3)

	// Throughput thresholds
	MinThroughputRPS float64 // Minimum requests per second during chaos (Phase 5)
}

// DefaultSLAThresholds returns conservative default thresholds for chaos testing.
func DefaultSLAThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates
		MinSuccessRate:      50.0, // At least 50% success during chaos
		RecoverySuccessRate: 99.0, // 99% after recovery

		// Latency (generous for chaos conditions)
		MaxP50Latency:  5 * time.Second,
		MaxP95Latency:  10 * time.Second,
		MaxP99Latency:  15 * time.Second,
		MaxP999Latency: 30 * time.Second,

		// Recovery
		MaxRecoveryTime:        30 * time.Second,
		StabilityDuration:      10 * time.Second,
		StabilityCheckCount:    5,
		MaxConsecutiveFailures: 1,

		// Throughput
		MinThroughputRPS: 0.1, // At least 0.1 RPS during chaos
	}
}

// StrictSLAThresholds returns strict thresholds for production-like requirements.
func StrictSLAThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates
		MinSuccessRate:      80.0, // At least 80% success during chaos
		RecoverySuccessRate: 99.9, // 99.9% after recovery

		// Latency
		MaxP50Latency:  1 * time.Second,
		MaxP95Latency:  3 * time.Second,
		MaxP99Latency:  5 * time.Second,
		MaxP999Latency: 10 * time.Second,

		// Recovery
		MaxRecoveryTime:        15 * time.Second,
		StabilityDuration:      30 * time.Second,
		StabilityCheckCount:    10,
		MaxConsecutiveFailures: 0,

		// Throughput
		MinThroughputRPS: 1.0, // At least 1 RPS during chaos
	}
}

// LatencyChaosThresholds returns thresholds appropriate for latency injection tests.
// These are more lenient on latency since we're intentionally injecting delays.
func LatencyChaosThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates (should remain high even with latency)
		MinSuccessRate:      90.0,
		RecoverySuccessRate: 99.0,

		// Latency (very lenient - we're injecting latency)
		MaxP50Latency:  30 * time.Second,
		MaxP95Latency:  45 * time.Second,
		MaxP99Latency:  60 * time.Second,
		MaxP999Latency: 90 * time.Second,

		// Recovery (should be fast once latency removed)
		MaxRecoveryTime:        10 * time.Second,
		StabilityDuration:      10 * time.Second,
		StabilityCheckCount:    5,
		MaxConsecutiveFailures: 1,

		// Throughput (reduced due to latency)
		MinThroughputRPS: 0.05,
	}
}

// TimeoutChaosThresholds returns thresholds for timeout injection tests.
// These expect failures since timeouts cause request failures.
func TimeoutChaosThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates (expected to be low during timeout chaos)
		MinSuccessRate:      0.0,  // Timeouts may cause 100% failure
		RecoverySuccessRate: 99.0, // Must recover after timeout removed

		// Latency thresholds
		MaxP50Latency:  10 * time.Second,
		MaxP95Latency:  15 * time.Second,
		MaxP99Latency:  20 * time.Second,
		MaxP999Latency: 30 * time.Second,

		// Recovery
		MaxRecoveryTime:        30 * time.Second,
		StabilityDuration:      10 * time.Second,
		StabilityCheckCount:    5,
		MaxConsecutiveFailures: 1,

		// Throughput
		MinThroughputRPS: 0.0, // May be zero during timeout
	}
}

// BandwidthChaosThresholds returns thresholds for bandwidth limiting tests.
func BandwidthChaosThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates (should eventually succeed despite slow transfer)
		MinSuccessRate:      70.0,
		RecoverySuccessRate: 99.0,

		// Latency (very lenient due to slow bandwidth)
		MaxP50Latency:  60 * time.Second,
		MaxP95Latency:  120 * time.Second,
		MaxP99Latency:  180 * time.Second,
		MaxP999Latency: 300 * time.Second,

		// Recovery
		MaxRecoveryTime:        15 * time.Second,
		StabilityDuration:      10 * time.Second,
		StabilityCheckCount:    5,
		MaxConsecutiveFailures: 1,

		// Throughput (very low due to bandwidth limits)
		MinThroughputRPS: 0.01,
	}
}
