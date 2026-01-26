package chaos

import "time"

// MetricsProvider defines the interface for chaos metrics collection.
// This interface enables ChaosAssertions to work with both the base ChaosMetrics
// and any extended implementations.
type MetricsProvider interface {
	// Request metrics
	GetTotalRequests() int
	GetFailedRequests() int
	GetTimeoutRequests() int
	SuccessRate() float64

	// Latency metrics
	AverageLatency() time.Duration
	Percentile(p float64) time.Duration
	P50() time.Duration
	P90() time.Duration
	P95() time.Duration
	P99() time.Duration
	P999() time.Duration

	// Throughput metrics
	ThroughputRPS() float64
	SuccessfulThroughputRPS() float64

	// Duration metrics
	ChaosDuration() time.Duration
	TestDuration() time.Duration

	// Error classification
	GetErrorCounts() map[ErrorCategory]int

	// Snapshot for atomic reads
	Snapshot() MetricsSnapshot
}

// MetricsSnapshot represents an immutable snapshot of metrics at a point in time.
// It implements MetricsProvider for use in assertions.
type MetricsSnapshot interface {
	MetricsProvider
}

// RecoveryMetricsProvider extends MetricsProvider with recovery tracking.
// Implement this interface for projects that need recovery time validation.
type RecoveryMetricsProvider interface {
	MetricsProvider
	GetRecoveryTime() time.Duration
}

// StabilityMetricsProvider extends MetricsProvider with stability tracking.
// Implement this interface for projects that need post-recovery stability validation.
type StabilityMetricsProvider interface {
	MetricsProvider
	StabilityPassRate() float64
	StabilityDuration() time.Duration
	GetMaxConsecutiveFailures() int
}
