//go:build !chaos

package setup

import "time"

// =============================================================================
// CHAOS TEST TIMEOUT CONSTANTS
// =============================================================================
// Timeouts specific to chaos testing scenarios.
// =============================================================================

const (
	// ChaosTestTimeout is the maximum duration for a single chaos test.
	ChaosTestTimeout = 5 * time.Minute

	// ChaosInfraStartupTimeout is the timeout for starting chaos infrastructure.
	ChaosInfraStartupTimeout = 10 * time.Minute

	// BaselineCollectionTime is how long to collect baseline metrics.
	BaselineCollectionTime = 10 * time.Second

	// ChaosInjectionDuration is the default duration for chaos injection.
	ChaosInjectionDuration = 30 * time.Second

	// RecoveryObservationTime is how long to observe after removing chaos.
	RecoveryObservationTime = 15 * time.Second

	// MaxRecoveryWait is the maximum time to wait for system recovery.
	MaxRecoveryWait = 60 * time.Second

	// StabilizationDelay is delay after chaos action before measurements.
	StabilizationDelay = 2 * time.Second

	// RequestInterval is the interval between test requests during chaos.
	RequestInterval = 500 * time.Millisecond

	// HealthCheckInterval is the interval for health checks during chaos.
	HealthCheckInterval = 1 * time.Second
)

// ChaosLatencyValues contains standard latency values for testing.
var ChaosLatencyValues = struct {
	Low    time.Duration // Noticeable but not disruptive
	Medium time.Duration // Significant delay
	High   time.Duration // Severe latency
	Jitter time.Duration // Standard jitter
}{
	Low:    1 * time.Second,
	Medium: 3 * time.Second,
	High:   5 * time.Second,
	Jitter: 500 * time.Millisecond,
}

// ChaosTimeoutValues contains standard timeout values for testing.
var ChaosTimeoutValues = struct {
	Short  time.Duration // Quick timeout
	Medium time.Duration // Standard timeout
	Long   time.Duration // Extended timeout
}{
	Short:  5 * time.Second,
	Medium: 10 * time.Second,
	Long:   30 * time.Second,
}

// ChaosBandwidthValues contains standard bandwidth limits for testing.
var ChaosBandwidthValues = struct {
	Slow   int // Very slow connection (1KB/s)
	Medium int // Moderate speed (10KB/s)
	Fast   int // Light throttling (100KB/s)
}{
	Slow:   1024,   // 1 KB/s
	Medium: 10240,  // 10 KB/s
	Fast:   102400, // 100 KB/s
}
