package helpers

import (
	"sort"
	"sync"
	"time"
)

// ChaosMetrics collects metrics during chaos tests.
type ChaosMetrics struct {
	mu sync.Mutex

	// Request metrics
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	TimeoutRequests    int

	// Timing metrics
	TotalLatency time.Duration
	MinLatency   time.Duration
	MaxLatency   time.Duration
	Latencies    []time.Duration

	// Percentile cache (calculated on demand)
	percentileCache map[float64]time.Duration
	percentileDirty bool

	// Recovery metrics
	RecoveryTime  time.Duration
	RecoveryStart time.Time
	RecoveryEnd   time.Time

	// State tracking
	ChaosStartTime time.Time
	ChaosEndTime   time.Time
	TestStartTime  time.Time
	TestEndTime    time.Time

	// Stability tracking
	StabilityChecks        []StabilityCheck
	StabilityStartTime     time.Time
	StabilityEndTime       time.Time
	ConsecutiveFailures    int
	MaxConsecutiveFailures int

	// Error classification
	ErrorClassifier *ErrorClassifier
}

// StabilityCheck represents a single stability check result.
type StabilityCheck struct {
	Timestamp   time.Time
	Success     bool
	SuccessRate float64
	Latency     time.Duration
	Error       string
}

// NewChaosMetrics creates a new ChaosMetrics instance.
func NewChaosMetrics() *ChaosMetrics {
	return &ChaosMetrics{
		MinLatency:      time.Hour, // Will be updated with actual min
		Latencies:       make([]time.Duration, 0),
		ErrorClassifier: NewErrorClassifier(),
	}
}

// GetMinLatency returns the minimum latency, or 0 if no requests recorded.
func (m *ChaosMetrics) GetMinLatency() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TotalRequests == 0 {
		return 0
	}

	return m.MinLatency
}

// StartTest records the test start time.
func (m *ChaosMetrics) StartTest() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TestStartTime = time.Now()
}

// EndTest records the test end time.
func (m *ChaosMetrics) EndTest() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TestEndTime = time.Now()
}

// StartChaos records when chaos injection started.
func (m *ChaosMetrics) StartChaos() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ChaosStartTime = time.Now()
}

// EndChaos records when chaos injection ended.
func (m *ChaosMetrics) EndChaos() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ChaosEndTime = time.Now()
}

// RecordRequest records a request result.
func (m *ChaosMetrics) RecordRequest(success bool, timeout bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.TotalLatency += latency
	m.Latencies = append(m.Latencies, latency)
	m.percentileDirty = true

	if latency < m.MinLatency {
		m.MinLatency = latency
	}

	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}

	if success {
		m.SuccessfulRequests++
	} else {
		m.FailedRequests++
	}

	if timeout {
		m.TimeoutRequests++
	}
}

// RecordError records and classifies an error message.
func (m *ChaosMetrics) RecordError(errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ErrorClassifier != nil {
		m.ErrorClassifier.RecordError(errMsg)
	}
}

// GetErrorCounts returns error counts by category.
func (m *ChaosMetrics) GetErrorCounts() map[ErrorCategory]int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ErrorClassifier == nil {
		return nil
	}

	return m.ErrorClassifier.GetCategoryCounts()
}

// ThroughputRPS returns the requests per second during the test.
func (m *ChaosMetrics) ThroughputRPS() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	duration := m.TestEndTime.Sub(m.TestStartTime)
	if duration <= 0 {
		duration = time.Since(m.TestStartTime)
	}

	if duration <= 0 {
		return 0
	}

	return float64(m.TotalRequests) / duration.Seconds()
}

// ChaosThroughputRPS returns the requests per second during the chaos period.
// WARNING: This method divides TotalRequests by chaos duration. It produces
// INACCURATE results when significant requests occur outside the chaos period.
// Use only when chaos period is the majority of test duration.
// For accurate chaos-period throughput, track request counts separately.
func (m *ChaosMetrics) ChaosThroughputRPS() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	duration := m.ChaosEndTime.Sub(m.ChaosStartTime)
	if duration <= 0 {
		// If chaos started but not ended, calculate from start to now
		if !m.ChaosStartTime.IsZero() {
			duration = time.Since(m.ChaosStartTime)
		}
	}

	if duration <= 0 {
		return 0
	}

	return float64(m.TotalRequests) / duration.Seconds()
}

// SuccessfulThroughputRPS returns the successful requests per second.
func (m *ChaosMetrics) SuccessfulThroughputRPS() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	duration := m.TestEndTime.Sub(m.TestStartTime)
	if duration <= 0 {
		duration = time.Since(m.TestStartTime)
	}

	if duration <= 0 {
		return 0
	}

	return float64(m.SuccessfulRequests) / duration.Seconds()
}

// StartRecovery records when recovery monitoring started.
func (m *ChaosMetrics) StartRecovery() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecoveryStart = time.Now()
}

// EndRecovery records when recovery completed.
func (m *ChaosMetrics) EndRecovery() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecoveryEnd = time.Now()
	m.RecoveryTime = m.RecoveryEnd.Sub(m.RecoveryStart)
}

// GetTotalRequests returns the total number of requests recorded.
func (m *ChaosMetrics) GetTotalRequests() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.TotalRequests
}

// GetFailedRequests returns the number of failed requests.
func (m *ChaosMetrics) GetFailedRequests() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.FailedRequests
}

// GetTimeoutRequests returns the number of timeout requests.
func (m *ChaosMetrics) GetTimeoutRequests() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.TimeoutRequests
}

// GetRecoveryTime returns the recovery time duration.
func (m *ChaosMetrics) GetRecoveryTime() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.RecoveryTime
}

// SuccessRate returns the success rate as a percentage.
func (m *ChaosMetrics) SuccessRate() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TotalRequests == 0 {
		return 0
	}

	return float64(m.SuccessfulRequests) / float64(m.TotalRequests) * 100
}

// AverageLatency returns the average latency.
func (m *ChaosMetrics) AverageLatency() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TotalRequests == 0 {
		return 0
	}

	return m.TotalLatency / time.Duration(m.TotalRequests)
}

// ChaosDuration returns how long chaos was active.
func (m *ChaosMetrics) ChaosDuration() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ChaosEndTime.IsZero() {
		return time.Since(m.ChaosStartTime)
	}

	return m.ChaosEndTime.Sub(m.ChaosStartTime)
}

// TestDuration returns the total test duration.
func (m *ChaosMetrics) TestDuration() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TestEndTime.IsZero() {
		return time.Since(m.TestStartTime)
	}

	return m.TestEndTime.Sub(m.TestStartTime)
}

// Percentile returns the latency at the given percentile (0-100).
// Uses nearest-rank method. For accurate percentiles, ensure sufficient sample size
// (recommended: 100+ samples for P99, 1000+ samples for P99.9).
// Invalid percentile values (< 0 or > 100) return 0.
func (m *ChaosMetrics) Percentile(p float64) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate percentile input
	if p < 0 || p > 100 {
		return 0
	}

	if len(m.Latencies) == 0 {
		return 0
	}

	// Initialize or clear cache if dirty
	if m.percentileCache == nil {
		m.percentileCache = make(map[float64]time.Duration)
	}

	if m.percentileDirty {
		// Clear ALL cached values when data has changed
		m.percentileCache = make(map[float64]time.Duration)
		m.percentileDirty = false
	}

	// Check cache after clearing stale entries
	if cached, ok := m.percentileCache[p]; ok {
		return cached
	}

	sorted := make([]time.Duration, len(m.Latencies))
	copy(sorted, m.Latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(float64(len(sorted)-1) * p / 100.0)
	if index < 0 {
		index = 0
	}

	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	result := sorted[index]
	m.percentileCache[p] = result

	return result
}

// P50 returns the 50th percentile (median) latency.
func (m *ChaosMetrics) P50() time.Duration {
	return m.Percentile(50)
}

// P90 returns the 90th percentile latency.
func (m *ChaosMetrics) P90() time.Duration {
	return m.Percentile(90)
}

// P95 returns the 95th percentile latency.
func (m *ChaosMetrics) P95() time.Duration {
	return m.Percentile(95)
}

// P99 returns the 99th percentile latency.
func (m *ChaosMetrics) P99() time.Duration {
	return m.Percentile(99)
}

// P999 returns the 99.9th percentile latency.
func (m *ChaosMetrics) P999() time.Duration {
	return m.Percentile(99.9)
}

// Snapshot returns a copy of the current metrics state.
// The returned struct has its own mutex and can be used independently.
func (m *ChaosMetrics) Snapshot() *ChaosMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()

	latencies := make([]time.Duration, len(m.Latencies))
	copy(latencies, m.Latencies)

	var percentileCache map[float64]time.Duration
	if m.percentileCache != nil {
		percentileCache = make(map[float64]time.Duration)
		for k, v := range m.percentileCache {
			percentileCache[k] = v
		}
	}

	// Copy stability checks
	stabilityChecks := make([]StabilityCheck, len(m.StabilityChecks))
	copy(stabilityChecks, m.StabilityChecks)

	return &ChaosMetrics{
		TotalRequests:          m.TotalRequests,
		SuccessfulRequests:     m.SuccessfulRequests,
		FailedRequests:         m.FailedRequests,
		TimeoutRequests:        m.TimeoutRequests,
		TotalLatency:           m.TotalLatency,
		MinLatency:             m.MinLatency,
		MaxLatency:             m.MaxLatency,
		Latencies:              latencies,
		percentileCache:        percentileCache,
		percentileDirty:        m.percentileDirty,
		RecoveryTime:           m.RecoveryTime,
		RecoveryStart:          m.RecoveryStart,
		RecoveryEnd:            m.RecoveryEnd,
		ChaosStartTime:         m.ChaosStartTime,
		ChaosEndTime:           m.ChaosEndTime,
		TestStartTime:          m.TestStartTime,
		TestEndTime:            m.TestEndTime,
		StabilityChecks:        stabilityChecks,
		StabilityStartTime:     m.StabilityStartTime,
		StabilityEndTime:       m.StabilityEndTime,
		ConsecutiveFailures:    m.ConsecutiveFailures,
		MaxConsecutiveFailures: m.MaxConsecutiveFailures,
		ErrorClassifier:        m.ErrorClassifier.Clone(), // Deep copy for snapshot isolation
	}
}

// StartStabilityCheck begins stability monitoring after recovery.
func (m *ChaosMetrics) StartStabilityCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StabilityStartTime = time.Now()
	m.StabilityChecks = make([]StabilityCheck, 0)
	m.ConsecutiveFailures = 0
	m.MaxConsecutiveFailures = 0
}

// RecordStabilityCheck records a stability check result.
func (m *ChaosMetrics) RecordStabilityCheck(success bool, successRate float64, latency time.Duration, err string) {
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
func (m *ChaosMetrics) EndStabilityCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StabilityEndTime = time.Now()
}

// StabilityDuration returns the duration of stability checking.
func (m *ChaosMetrics) StabilityDuration() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.StabilityEndTime.IsZero() {
		return time.Since(m.StabilityStartTime)
	}

	return m.StabilityEndTime.Sub(m.StabilityStartTime)
}

// StabilityPassRate returns the percentage of stability checks that passed.
func (m *ChaosMetrics) StabilityPassRate() float64 {
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
func (m *ChaosMetrics) GetStabilityChecks() []StabilityCheck {
	m.mu.Lock()
	defer m.mu.Unlock()

	checks := make([]StabilityCheck, len(m.StabilityChecks))
	copy(checks, m.StabilityChecks)

	return checks
}

// GetMaxConsecutiveFailures returns the maximum consecutive failures during stability checks.
func (m *ChaosMetrics) GetMaxConsecutiveFailures() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.MaxConsecutiveFailures
}
