package chaos

import "time"

// ExtendedSLAThresholds extends SLAThresholds with stability tracking fields.
// Use this when you need recovery and stability validation thresholds.
type ExtendedSLAThresholds struct {
	SLAThresholds // Embed base thresholds

	// Extended recovery thresholds
	RecoverySuccessRate float64 // Required success rate after recovery

	// Stability thresholds
	StabilityDuration      time.Duration // Duration system must remain stable after recovery
	StabilityCheckCount    int           // Number of stability checks to perform
	MaxConsecutiveFailures int           // Max consecutive failures during stability check
}

// DefaultExtendedSLAThresholds returns conservative default thresholds including stability.
func DefaultExtendedSLAThresholds() ExtendedSLAThresholds {
	return ExtendedSLAThresholds{
		SLAThresholds:          DefaultSLAThresholds(),
		RecoverySuccessRate:    99.0,
		StabilityDuration:      10 * time.Second,
		StabilityCheckCount:    5,
		MaxConsecutiveFailures: 1,
	}
}

// StrictExtendedSLAThresholds returns strict thresholds for production-like requirements.
func StrictExtendedSLAThresholds() ExtendedSLAThresholds {
	return ExtendedSLAThresholds{
		SLAThresholds:          StrictSLAThresholds(),
		RecoverySuccessRate:    99.9,
		StabilityDuration:      30 * time.Second,
		StabilityCheckCount:    10,
		MaxConsecutiveFailures: 0,
	}
}

// LatencyExtendedSLAThresholds returns thresholds appropriate for latency injection tests.
func LatencyExtendedSLAThresholds() ExtendedSLAThresholds {
	return ExtendedSLAThresholds{
		SLAThresholds:          LatencyChaosThresholds(),
		RecoverySuccessRate:    99.0,
		StabilityDuration:      10 * time.Second,
		StabilityCheckCount:    5,
		MaxConsecutiveFailures: 1,
	}
}

// TimeoutExtendedSLAThresholds returns thresholds for timeout injection tests.
func TimeoutExtendedSLAThresholds() ExtendedSLAThresholds {
	return ExtendedSLAThresholds{
		SLAThresholds:          TimeoutChaosThresholds(),
		RecoverySuccessRate:    99.0,
		StabilityDuration:      10 * time.Second,
		StabilityCheckCount:    5,
		MaxConsecutiveFailures: 1,
	}
}

// BandwidthExtendedSLAThresholds returns thresholds for bandwidth limiting tests.
func BandwidthExtendedSLAThresholds() ExtendedSLAThresholds {
	return ExtendedSLAThresholds{
		SLAThresholds:          BandwidthChaosThresholds(),
		RecoverySuccessRate:    99.0,
		StabilityDuration:      10 * time.Second,
		StabilityCheckCount:    5,
		MaxConsecutiveFailures: 1,
	}
}
