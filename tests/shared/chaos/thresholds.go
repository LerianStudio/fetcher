package chaos

import "time"

// SLAThresholds defines Service Level Agreement thresholds for chaos tests.
// These thresholds represent the minimum acceptable performance during chaos conditions.
//
// This is the CORE version without stability fields.
// For extended thresholds with stability tracking, see the helpers package.
type SLAThresholds struct {
	// Success rate thresholds (percentage 0-100)
	MinSuccessRate float64 // Minimum acceptable success rate during chaos

	// Latency thresholds
	MaxP50Latency  time.Duration // Maximum acceptable P50 latency
	MaxP95Latency  time.Duration // Maximum acceptable P95 latency
	MaxP99Latency  time.Duration // Maximum acceptable P99 latency
	MaxP999Latency time.Duration // Maximum acceptable P99.9 latency (warning only)

	// Recovery thresholds
	MaxRecoveryTime time.Duration // Maximum time to recover after chaos removal

	// Throughput thresholds
	MinThroughputRPS float64 // Minimum requests per second during chaos
}

// DefaultSLAThresholds returns conservative default thresholds for chaos testing.
func DefaultSLAThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates
		MinSuccessRate: 50.0, // At least 50% success during chaos

		// Latency (generous for chaos conditions)
		MaxP50Latency:  5 * time.Second,
		MaxP95Latency:  10 * time.Second,
		MaxP99Latency:  15 * time.Second,
		MaxP999Latency: 30 * time.Second,

		// Recovery
		MaxRecoveryTime: 30 * time.Second,

		// Throughput
		MinThroughputRPS: 0.1, // At least 0.1 RPS during chaos
	}
}

// StrictSLAThresholds returns strict thresholds for production-like requirements.
func StrictSLAThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates
		MinSuccessRate: 80.0, // At least 80% success during chaos

		// Latency
		MaxP50Latency:  1 * time.Second,
		MaxP95Latency:  3 * time.Second,
		MaxP99Latency:  5 * time.Second,
		MaxP999Latency: 10 * time.Second,

		// Recovery
		MaxRecoveryTime: 15 * time.Second,

		// Throughput
		MinThroughputRPS: 1.0, // At least 1 RPS during chaos
	}
}

// LatencyChaosThresholds returns thresholds appropriate for latency injection tests.
// These are more lenient on latency since we're intentionally injecting delays.
func LatencyChaosThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates (should remain high even with latency)
		MinSuccessRate: 90.0,

		// Latency (very lenient - we're injecting latency)
		MaxP50Latency:  30 * time.Second,
		MaxP95Latency:  45 * time.Second,
		MaxP99Latency:  60 * time.Second,
		MaxP999Latency: 90 * time.Second,

		// Recovery (should be fast once latency removed)
		MaxRecoveryTime: 10 * time.Second,

		// Throughput (reduced due to latency)
		MinThroughputRPS: 0.05,
	}
}

// TimeoutChaosThresholds returns thresholds for timeout injection tests.
// These expect failures since timeouts cause request failures.
func TimeoutChaosThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates (expected to be low during timeout chaos)
		MinSuccessRate: 0.0, // Timeouts may cause 100% failure

		// Latency thresholds
		MaxP50Latency:  10 * time.Second,
		MaxP95Latency:  15 * time.Second,
		MaxP99Latency:  20 * time.Second,
		MaxP999Latency: 30 * time.Second,

		// Recovery
		MaxRecoveryTime: 30 * time.Second,

		// Throughput
		MinThroughputRPS: 0.0, // May be zero during timeout
	}
}

// BandwidthChaosThresholds returns thresholds for bandwidth limiting tests.
func BandwidthChaosThresholds() SLAThresholds {
	return SLAThresholds{
		// Success rates (should eventually succeed despite slow transfer)
		MinSuccessRate: 70.0,

		// Latency (very lenient due to slow bandwidth)
		MaxP50Latency:  60 * time.Second,
		MaxP95Latency:  120 * time.Second,
		MaxP99Latency:  180 * time.Second,
		MaxP999Latency: 300 * time.Second,

		// Recovery
		MaxRecoveryTime: 15 * time.Second,

		// Throughput (very low due to bandwidth limits)
		MinThroughputRPS: 0.01,
	}
}
