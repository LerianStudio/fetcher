package chaos

import (
	"fmt"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"

	"github.com/LerianStudio/fetcher/tests/shared/containers"
)

// getIntAttr extracts an int attribute from the config, returning an error if missing or wrong type.
func getIntAttr(attrs map[string]any, key string) (int, error) {
	val, ok := attrs[key].(int)
	if !ok {
		return 0, fmt.Errorf("%s attribute missing or invalid type", key)
	}

	return val, nil
}

// InjectChaos injects a toxic into the proxy based on the provided configuration.
// Returns the created toxic and any error encountered.
func InjectChaos(proxy *toxiproxy.Proxy, cfg ChaosInjectionConfig) (*toxiproxy.Toxic, error) {
	a := cfg.Attributes

	switch cfg.Type {
	case "latency":
		latency, err := getIntAttr(a, "latency")
		if err != nil {
			return nil, err
		}

		jitter, err := getIntAttr(a, "jitter")
		if err != nil {
			return nil, err
		}

		return containers.AddLatency(proxy, cfg.Name, latency, jitter)

	case "timeout":
		timeout, err := getIntAttr(a, "timeout")
		if err != nil {
			return nil, err
		}

		return containers.AddTimeout(proxy, cfg.Name, timeout)

	case "bandwidth":
		rate, err := getIntAttr(a, "rate")
		if err != nil {
			return nil, err
		}

		return containers.AddBandwidth(proxy, cfg.Name, rate)

	case "reset_peer":
		timeout, err := getIntAttr(a, "timeout")
		if err != nil {
			return nil, err
		}

		return containers.AddResetPeer(proxy, cfg.Name, timeout)

	case "slow_close":
		delay, err := getIntAttr(a, "delay")
		if err != nil {
			return nil, err
		}

		return containers.AddSlowClose(proxy, cfg.Name, delay)

	case "limit_data":
		bytes, err := getIntAttr(a, "bytes")
		if err != nil {
			return nil, err
		}

		return containers.AddLimitData(proxy, cfg.Name, bytes)

	case "slicer":
		avgSize, err := getIntAttr(a, "average_size")
		if err != nil {
			return nil, err
		}

		sizeVariation, err := getIntAttr(a, "size_variation")
		if err != nil {
			return nil, err
		}

		delay, err := getIntAttr(a, "delay")
		if err != nil {
			return nil, err
		}

		return containers.AddSlicer(proxy, cfg.Name, avgSize, sizeVariation, delay)

	default:
		return proxy.AddToxic(cfg.Name, cfg.Type, cfg.Direction, cfg.Toxicity, toxiproxy.Attributes(cfg.Attributes))
	}
}

// RemoveChaos removes a specific toxic from the proxy by name.
func RemoveChaos(proxy *toxiproxy.Proxy, name string) error {
	return containers.RemoveToxic(proxy, name)
}

// RemoveAllChaos removes all toxics from the proxy.
func RemoveAllChaos(proxy *toxiproxy.Proxy) error {
	return containers.RemoveAllToxics(proxy)
}

// DisableProxy disables a proxy, preventing all connections.
func DisableProxy(proxy *toxiproxy.Proxy) error {
	return containers.DisableProxy(proxy)
}

// EnableProxy enables a proxy, allowing connections.
func EnableProxy(proxy *toxiproxy.Proxy) error {
	return containers.EnableProxy(proxy)
}
