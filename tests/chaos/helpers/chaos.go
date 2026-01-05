//go:build chaos

package helpers

import (
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"

	"github.com/LerianStudio/fetcher/tests/shared/containers"
)

// =============================================================================
// TIME DURATION HELPERS
// =============================================================================

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

// =============================================================================
// CHAOS INJECTION CONFIG
// =============================================================================

// ChaosInjectionConfig defines configuration for chaos injection via Toxiproxy.
// This struct provides all fields needed for Toxiproxy toxic creation.
type ChaosInjectionConfig struct {
	Name       string                 // Unique name for the toxic
	Type       string                 // Toxic type: latency, timeout, bandwidth, reset_peer, slow_close
	Direction  string                 // Direction: downstream, upstream
	Toxicity   float32                // Probability of toxic being applied (0.0 to 1.0)
	Attributes map[string]interface{} // Toxic-specific attributes
}

// =============================================================================
// DEFAULT CONFIG CREATORS
// =============================================================================

// DefaultLatencyConfig creates a ChaosInjectionConfig for latency injection.
// latencyMs: base latency in milliseconds
// jitterMs: random variance in milliseconds added to latency
func DefaultLatencyConfig(latencyMs, jitterMs int) ChaosInjectionConfig {
	return ChaosInjectionConfig{
		Name:      "latency-chaos",
		Type:      "latency",
		Direction: "downstream",
		Toxicity:  1.0,
		Attributes: map[string]interface{}{
			"latency": latencyMs,
			"jitter":  jitterMs,
		},
	}
}

// DefaultTimeoutConfig creates a ChaosInjectionConfig for timeout injection.
// timeoutMs: timeout in milliseconds after which connection is closed
func DefaultTimeoutConfig(timeoutMs int) ChaosInjectionConfig {
	return ChaosInjectionConfig{
		Name:      "timeout-chaos",
		Type:      "timeout",
		Direction: "downstream",
		Toxicity:  1.0,
		Attributes: map[string]interface{}{
			"timeout": timeoutMs,
		},
	}
}

// DefaultBandwidthConfig creates a ChaosInjectionConfig for bandwidth limiting.
// rateBytesPerSec: bandwidth limit in bytes per second
func DefaultBandwidthConfig(rateBytesPerSec int) ChaosInjectionConfig {
	return ChaosInjectionConfig{
		Name:      "bandwidth-chaos",
		Type:      "bandwidth",
		Direction: "downstream",
		Toxicity:  1.0,
		Attributes: map[string]interface{}{
			"rate": rateBytesPerSec,
		},
	}
}

// DefaultResetPeerConfig creates a ChaosInjectionConfig for connection reset.
// timeoutMs: timeout in milliseconds after which the connection is reset
func DefaultResetPeerConfig(timeoutMs int) ChaosInjectionConfig {
	return ChaosInjectionConfig{
		Name:      "reset-peer-chaos",
		Type:      "reset_peer",
		Direction: "downstream",
		Toxicity:  1.0,
		Attributes: map[string]interface{}{
			"timeout": timeoutMs,
		},
	}
}

// DefaultSlowCloseConfig creates a ChaosInjectionConfig for slow connection close.
// delayMs: delay in milliseconds before connection close completes
func DefaultSlowCloseConfig(delayMs int) ChaosInjectionConfig {
	return ChaosInjectionConfig{
		Name:      "slow-close-chaos",
		Type:      "slow_close",
		Direction: "downstream",
		Toxicity:  1.0,
		Attributes: map[string]interface{}{
			"delay": delayMs,
		},
	}
}

// DefaultLimitDataConfig creates a ChaosInjectionConfig for limiting data transmission.
// After the specified number of bytes have been transmitted, the connection is closed.
// bytes: maximum number of bytes to allow before closing the connection
func DefaultLimitDataConfig(bytes int) ChaosInjectionConfig {
	return ChaosInjectionConfig{
		Name:      "limit-data-chaos",
		Type:      "limit_data",
		Direction: "downstream",
		Toxicity:  1.0,
		Attributes: map[string]interface{}{
			"bytes": bytes,
		},
	}
}

// DefaultSlicerConfig creates a ChaosInjectionConfig for slicing data into smaller packets.
// This simulates packet fragmentation and can be combined with delay to simulate slow networks.
// avgSize: average size of each slice in bytes
// sizeVariation: variation in slice size (bytes)
// delayMicros: delay in microseconds between each slice
func DefaultSlicerConfig(avgSize, sizeVariation, delayMicros int) ChaosInjectionConfig {
	return ChaosInjectionConfig{
		Name:      "slicer-chaos",
		Type:      "slicer",
		Direction: "downstream",
		Toxicity:  1.0,
		Attributes: map[string]interface{}{
			"average_size":   avgSize,
			"size_variation": sizeVariation,
			"delay":          delayMicros,
		},
	}
}

// =============================================================================
// CHAOS INJECTION FUNCTIONS
// =============================================================================

// InjectChaos injects a toxic into the proxy based on the provided configuration.
// Returns the created toxic and any error encountered.
func InjectChaos(proxy *toxiproxy.Proxy, cfg ChaosInjectionConfig) (*toxiproxy.Toxic, error) {
	switch cfg.Type {
	case "latency":
		latency, _ := cfg.Attributes["latency"].(int)
		jitter, _ := cfg.Attributes["jitter"].(int)
		return containers.AddLatency(proxy, cfg.Name, latency, jitter)

	case "timeout":
		timeout, _ := cfg.Attributes["timeout"].(int)
		return containers.AddTimeout(proxy, cfg.Name, timeout)

	case "bandwidth":
		rate, _ := cfg.Attributes["rate"].(int)
		return containers.AddBandwidth(proxy, cfg.Name, rate)

	case "reset_peer":
		timeout, _ := cfg.Attributes["timeout"].(int)
		return containers.AddResetPeer(proxy, cfg.Name, timeout)

	case "slow_close":
		delay, _ := cfg.Attributes["delay"].(int)
		return containers.AddSlowClose(proxy, cfg.Name, delay)

	case "limit_data":
		bytes, _ := cfg.Attributes["bytes"].(int)
		return containers.AddLimitData(proxy, cfg.Name, bytes)

	case "slicer":
		avgSize, _ := cfg.Attributes["average_size"].(int)
		sizeVariation, _ := cfg.Attributes["size_variation"].(int)
		delay, _ := cfg.Attributes["delay"].(int)
		return containers.AddSlicer(proxy, cfg.Name, avgSize, sizeVariation, delay)

	default:
		// Fall back to generic toxic creation for custom types
		return proxy.AddToxic(cfg.Name, cfg.Type, cfg.Direction, cfg.Toxicity, toxiproxy.Attributes(cfg.Attributes))
	}
}

// =============================================================================
// CHAOS REMOVAL FUNCTIONS
// =============================================================================

// RemoveChaos removes a specific toxic from the proxy by name.
func RemoveChaos(proxy *toxiproxy.Proxy, name string) error {
	return containers.RemoveToxic(proxy, name)
}

// RemoveAllChaos removes all toxics from the proxy.
// Wraps containers.RemoveAllToxics for convenience.
func RemoveAllChaos(proxy *toxiproxy.Proxy) error {
	return containers.RemoveAllToxics(proxy)
}

// =============================================================================
// PROXY CONTROL FUNCTIONS
// =============================================================================

// DisableProxy disables a proxy, preventing all connections.
// Wraps containers.DisableProxy for convenience.
func DisableProxy(proxy *toxiproxy.Proxy) error {
	return containers.DisableProxy(proxy)
}

// EnableProxy enables a proxy, allowing connections.
// Wraps containers.EnableProxy for convenience.
func EnableProxy(proxy *toxiproxy.Proxy) error {
	return containers.EnableProxy(proxy)
}
