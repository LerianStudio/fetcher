package chaos

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ExtendedAssertions extends ChaosAssertions with recovery and stability assertions.
// Use this when you need to validate recovery time and post-recovery stability.
type ExtendedAssertions struct {
	*ChaosAssertions
	t       *testing.T
	metrics *ExtendedMetrics
}

// NewExtendedAssertions creates a new ExtendedAssertions instance.
func NewExtendedAssertions(t *testing.T, metrics *ExtendedMetrics) *ExtendedAssertions {
	t.Helper()

	return &ExtendedAssertions{
		ChaosAssertions: NewChaosAssertions(t, metrics.ChaosMetrics),
		t:               t,
		metrics:         metrics,
	}
}

// AssertRecoveryWithin asserts that system recovered within timeout.
func (a *ExtendedAssertions) AssertRecoveryWithin(timeout time.Duration) {
	a.t.Helper()
	recoveryTime := a.metrics.GetRecoveryTime()
	assert.LessOrEqual(a.t, recoveryTime, timeout,
		"Recovery time %v should be within %v", recoveryTime, timeout)
}

// AssertRecoverySuccessRate asserts success rate meets recovery threshold.
func (a *ExtendedAssertions) AssertRecoverySuccessRate(minRate float64) {
	a.t.Helper()
	rate := a.metrics.SuccessRate()
	assert.GreaterOrEqual(a.t, rate, minRate,
		"Recovery success rate %.2f%% should be at least %.2f%%", rate, minRate)
}

// AssertStabilityMaintained asserts that the system remained stable during the stability check period.
func (a *ExtendedAssertions) AssertStabilityMaintained(minPassRate float64, maxConsecutiveFailures int) {
	a.t.Helper()

	passRate := a.metrics.StabilityPassRate()
	assert.GreaterOrEqual(a.t, passRate, minPassRate,
		"Stability pass rate %.2f%% should be at least %.2f%%", passRate, minPassRate)

	maxFails := a.metrics.GetMaxConsecutiveFailures()
	assert.LessOrEqual(a.t, maxFails, maxConsecutiveFailures,
		"Max consecutive failures %d should not exceed %d", maxFails, maxConsecutiveFailures)
}

// AssertStabilityDuration asserts that stability was maintained for at least the given duration.
func (a *ExtendedAssertions) AssertStabilityDuration(minDuration time.Duration) {
	a.t.Helper()
	duration := a.metrics.StabilityDuration()
	assert.GreaterOrEqual(a.t, duration, minDuration,
		"Stability duration %v should be at least %v", duration, minDuration)
}

// ValidateAgainstSLA checks metrics against extended SLA thresholds.
// This overrides the base ChaosAssertions.ValidateAgainstSLA to accept ExtendedSLAThresholds.
func (a *ExtendedAssertions) ValidateAgainstSLA(thresholds ExtendedSLAThresholds) SLAValidationResult {
	a.t.Helper()
	return a.ChaosAssertions.ValidateAgainstSLA(thresholds.SLAThresholds)
}

// AssertSLAMet asserts that all extended SLA thresholds are met.
// This overrides the base ChaosAssertions.AssertSLAMet to accept ExtendedSLAThresholds.
func (a *ExtendedAssertions) AssertSLAMet(thresholds ExtendedSLAThresholds) {
	a.t.Helper()
	a.ChaosAssertions.AssertSLAMet(thresholds.SLAThresholds)
}

// AssertRecoveryWithStability asserts recovery within time AND sustained stability.
func (a *ExtendedAssertions) AssertRecoveryWithStability(maxRecoveryTime time.Duration, thresholds ExtendedSLAThresholds) {
	a.t.Helper()

	// Assert recovery time
	recoveryTime := a.metrics.GetRecoveryTime()
	assert.LessOrEqual(a.t, recoveryTime, maxRecoveryTime,
		"Recovery time %v should be within %v", recoveryTime, maxRecoveryTime)

	// Assert stability maintained
	minPassRate := thresholds.MinSuccessRate
	if minPassRate == 0 {
		minPassRate = 100.0 - float64(thresholds.MaxConsecutiveFailures)*10
	}

	a.AssertStabilityMaintained(minPassRate, thresholds.MaxConsecutiveFailures)

	// Assert stability duration
	a.AssertStabilityDuration(thresholds.StabilityDuration)
}

// DocumentResultExtended logs the test result with extended metrics including recovery.
func DocumentResultExtended(t *testing.T, metrics *ExtendedMetrics, result string) {
	t.Helper()

	// Use base documentation
	DocumentResult(t, metrics.ChaosMetrics, result)

	// Add extended metrics
	t.Logf("  Recovery Time: %v", metrics.GetRecoveryTime())
	t.Logf("  Stability Duration: %v", metrics.StabilityDuration())
	t.Logf("  Stability Pass Rate: %.2f%%", metrics.StabilityPassRate())
	t.Logf("  Max Consecutive Failures: %d", metrics.GetMaxConsecutiveFailures())
}
