package chaos

import "time"

// ChaosLatencyValues contains standard latency values for chaos testing.
var ChaosLatencyValues = struct {
	Low    time.Duration // Noticeable but not disruptive
	Medium time.Duration // Significant delay
	High   time.Duration // Severe latency
	Jitter time.Duration // Standard jitter for randomization
}{
	Low:    500 * time.Millisecond,
	Medium: 3 * time.Second,
	High:   5 * time.Second,
	Jitter: 500 * time.Millisecond,
}

// ChaosTimeoutValues contains standard timeout values for chaos testing.
var ChaosTimeoutValues = struct {
	Short  time.Duration // Quick timeout for fast operations
	Medium time.Duration // Standard timeout for normal operations
	Long   time.Duration // Extended timeout for slow operations
}{
	Short:  5 * time.Second,
	Medium: 15 * time.Second,
	Long:   30 * time.Second,
}

// ChaosBandwidthValues contains standard bandwidth limits for chaos testing in bytes per second.
var ChaosBandwidthValues = struct {
	Low    int // Very slow connection (1 KB/s)
	Medium int // Moderate speed (10 KB/s)
	High   int // Light throttling (100 KB/s)
}{
	Low:    1024,   // 1 KB/s
	Medium: 10240,  // 10 KB/s
	High:   102400, // 100 KB/s
}

// ChaosSuccessRateThresholds contains standard success rate thresholds for chaos testing.
var ChaosSuccessRateThresholds = struct {
	DuringChaos        float64 // Minimum success rate during chaos
	AfterRecovery      float64 // Minimum success rate after recovery
	DuringLatencyChaos float64 // Higher threshold for latency (shouldn't cause failures)
	DuringTimeoutChaos float64 // Lower threshold for timeout (expected failures)
}{
	DuringChaos:        50.0, // At least 50% during general chaos
	AfterRecovery:      99.0, // At least 99% after recovery
	DuringLatencyChaos: 90.0, // At least 90% during latency injection
	DuringTimeoutChaos: 0.0,  // May be 0% during timeout injection
}

// Timing constants for chaos injection and recovery observation.
const (
	// StabilizationDelay is the delay after injecting chaos before testing.
	StabilizationDelay = 2 * time.Second

	// RecoveryObservationTime is the delay after removing chaos before testing recovery.
	RecoveryObservationTime = 5 * time.Second

	// ChaosInjectionDuration is the default duration for chaos injection.
	ChaosInjectionDuration = 10 * time.Second
)
