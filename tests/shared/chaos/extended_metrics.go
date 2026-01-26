package chaos

import (
	"sync"
	"time"
)

// StabilityCheck represents a single stability check result.
type StabilityCheck struct {
	Timestamp   time.Time
	Success     bool
	SuccessRate float64
	Latency     time.Duration
	Error       string
}

// ExtendedMetrics extends ChaosMetrics with recovery and stability tracking.
// Use this when you need to track recovery time and post-recovery stability.
type ExtendedMetrics struct {
	*ChaosMetrics // Embed base metrics

	mu sync.Mutex // Separate mutex for extended fields

	// Recovery metrics
	RecoveryTime  time.Duration
	RecoveryStart time.Time
	RecoveryEnd   time.Time

	// Stability tracking
	StabilityChecks        []StabilityCheck
	StabilityStartTime     time.Time
	StabilityEndTime       time.Time
	ConsecutiveFailures    int
	MaxConsecutiveFailures int
}

// NewExtendedMetrics creates a new ExtendedMetrics instance.
func NewExtendedMetrics() *ExtendedMetrics {
	return &ExtendedMetrics{
		ChaosMetrics:    NewChaosMetrics(),
		StabilityChecks: make([]StabilityCheck, 0),
	}
}

// StartRecovery records when recovery monitoring started.
func (m *ExtendedMetrics) StartRecovery() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecoveryStart = time.Now()
}

// EndRecovery records when recovery completed.
func (m *ExtendedMetrics) EndRecovery() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecoveryEnd = time.Now()
	m.RecoveryTime = m.RecoveryEnd.Sub(m.RecoveryStart)
}

// GetRecoveryTime returns the recovery time duration.
func (m *ExtendedMetrics) GetRecoveryTime() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.RecoveryTime
}

// StartStabilityCheck begins stability monitoring after recovery.
func (m *ExtendedMetrics) StartStabilityCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StabilityStartTime = time.Now()
	m.StabilityChecks = make([]StabilityCheck, 0)
	m.ConsecutiveFailures = 0
	m.MaxConsecutiveFailures = 0
}

// RecordStabilityCheck records a stability check result.
func (m *ExtendedMetrics) RecordStabilityCheck(success bool, successRate float64, latency time.Duration, err string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	check := StabilityCheck{
		Timestamp:   time.Now(),
		Success:     success,
		SuccessRate: successRate,
		Latency:     latency,
		Error:       err,
	}
	m.StabilityChecks = append(m.StabilityChecks, check)

	if success {
		m.ConsecutiveFailures = 0
	} else {
		m.ConsecutiveFailures++
		if m.ConsecutiveFailures > m.MaxConsecutiveFailures {
			m.MaxConsecutiveFailures = m.ConsecutiveFailures
		}
	}
}

// EndStabilityCheck ends stability monitoring.
func (m *ExtendedMetrics) EndStabilityCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StabilityEndTime = time.Now()
}

// StabilityDuration returns the duration of stability checking.
func (m *ExtendedMetrics) StabilityDuration() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.StabilityEndTime.IsZero() {
		if m.StabilityStartTime.IsZero() {
			return 0
		}

		return time.Since(m.StabilityStartTime)
	}

	return m.StabilityEndTime.Sub(m.StabilityStartTime)
}

// StabilityPassRate returns the percentage of stability checks that passed.
func (m *ExtendedMetrics) StabilityPassRate() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.StabilityChecks) == 0 {
		return 0
	}

	passed := 0

	for _, check := range m.StabilityChecks {
		if check.Success {
			passed++
		}
	}

	return float64(passed) / float64(len(m.StabilityChecks)) * 100
}

// GetStabilityChecks returns a copy of stability checks.
func (m *ExtendedMetrics) GetStabilityChecks() []StabilityCheck {
	m.mu.Lock()
	defer m.mu.Unlock()

	checks := make([]StabilityCheck, len(m.StabilityChecks))
	copy(checks, m.StabilityChecks)

	return checks
}

// GetMaxConsecutiveFailures returns the maximum consecutive failures during stability checks.
func (m *ExtendedMetrics) GetMaxConsecutiveFailures() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.MaxConsecutiveFailures
}

// Snapshot returns a copy of the current extended metrics state.
// Implements MetricsSnapshot interface for use with ChaosAssertions.
func (m *ExtendedMetrics) Snapshot() MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	stabilityChecks := make([]StabilityCheck, len(m.StabilityChecks))
	copy(stabilityChecks, m.StabilityChecks)

	// Get base metrics snapshot and type assert to *ChaosMetrics
	baseSnapshot := m.ChaosMetrics.Snapshot().(*ChaosMetrics)

	return &ExtendedMetrics{
		ChaosMetrics:           baseSnapshot,
		RecoveryTime:           m.RecoveryTime,
		RecoveryStart:          m.RecoveryStart,
		RecoveryEnd:            m.RecoveryEnd,
		StabilityChecks:        stabilityChecks,
		StabilityStartTime:     m.StabilityStartTime,
		StabilityEndTime:       m.StabilityEndTime,
		ConsecutiveFailures:    m.ConsecutiveFailures,
		MaxConsecutiveFailures: m.MaxConsecutiveFailures,
	}
}

// Verify ExtendedMetrics implements RecoveryMetricsProvider interface.
var _ RecoveryMetricsProvider = (*ExtendedMetrics)(nil)

// Verify ExtendedMetrics implements StabilityMetricsProvider interface.
var _ StabilityMetricsProvider = (*ExtendedMetrics)(nil)
