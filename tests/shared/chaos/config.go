package chaos

import "time"

// LatencyMs converts a time.Duration to milliseconds as int.
// Useful for passing durations to Toxiproxy toxic configurations.
func LatencyMs(d time.Duration) int {
	return int(d.Milliseconds())
}

// TimeoutMs converts a time.Duration to milliseconds as int.
// Alias for LatencyMs for semantic clarity when configuring timeout toxics.
func TimeoutMs(d time.Duration) int {
	return int(d.Milliseconds())
}

// ChaosInjectionConfig defines configuration for chaos injection via Toxiproxy.
// This struct provides all fields needed for Toxiproxy toxic creation.
type ChaosInjectionConfig struct {
	Name       string         // Unique name for the toxic
	Type       string         // Toxic type: latency, timeout, bandwidth, reset_peer, slow_close
	Direction  string         // Direction: downstream, upstream
	Toxicity   float32        // Probability of toxic being applied (0.0 to 1.0)
	Attributes map[string]any // Toxic-specific attributes
}

// newChaosConfig creates a ChaosInjectionConfig with standard defaults.
// All toxic types use downstream direction with 100% toxicity.
func newChaosConfig(chaosType string, attrs map[string]any) ChaosInjectionConfig {
	return ChaosInjectionConfig{
		Name:       chaosType + "-chaos",
		Type:       chaosType,
		Direction:  "downstream",
		Toxicity:   1.0,
		Attributes: attrs,
	}
}

// DefaultLatencyConfig creates a config for latency injection.
func DefaultLatencyConfig(latencyMs, jitterMs int) ChaosInjectionConfig {
	return newChaosConfig("latency", map[string]any{"latency": latencyMs, "jitter": jitterMs})
}

// DefaultTimeoutConfig creates a config for timeout injection.
func DefaultTimeoutConfig(timeoutMs int) ChaosInjectionConfig {
	return newChaosConfig("timeout", map[string]any{"timeout": timeoutMs})
}

// DefaultBandwidthConfig creates a config for bandwidth limiting.
func DefaultBandwidthConfig(rateBytesPerSec int) ChaosInjectionConfig {
	return newChaosConfig("bandwidth", map[string]any{"rate": rateBytesPerSec})
}

// DefaultResetPeerConfig creates a config for connection reset.
func DefaultResetPeerConfig(timeoutMs int) ChaosInjectionConfig {
	return newChaosConfig("reset_peer", map[string]any{"timeout": timeoutMs})
}

// DefaultSlowCloseConfig creates a config for slow connection close.
func DefaultSlowCloseConfig(delayMs int) ChaosInjectionConfig {
	return newChaosConfig("slow_close", map[string]any{"delay": delayMs})
}

// DefaultLimitDataConfig creates a config for limiting data transmission.
func DefaultLimitDataConfig(bytes int) ChaosInjectionConfig {
	return newChaosConfig("limit_data", map[string]any{"bytes": bytes})
}

// DefaultSlicerConfig creates a config for slicing data into smaller packets.
func DefaultSlicerConfig(avgSize, sizeVariation, delayMicros int) ChaosInjectionConfig {
	return newChaosConfig("slicer", map[string]any{
		"average_size":   avgSize,
		"size_variation": sizeVariation,
		"delay":          delayMicros,
	})
}
