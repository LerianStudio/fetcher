package chaos

import (
	"fmt"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
)

// ChaosOperations provides a facade for chaos injection operations.
// It replaces service-specific methods (DisablePostgres, AddPostgresLatency, etc.)
// with generic methods that accept a ServiceName parameter.
type ChaosOperations struct {
	registry *ProxyRegistry
}

// NewChaosOperations creates a new ChaosOperations facade with the given registry.
// Panics if registry is nil to fail fast during test setup rather than during cleanup.
func NewChaosOperations(registry *ProxyRegistry) *ChaosOperations {
	if registry == nil {
		panic("chaos: NewChaosOperations requires non-nil registry")
	}

	return &ChaosOperations{registry: registry}
}

// DisableService disables the proxy for the given service, preventing all connections.
// Returns nil if the service is not registered (graceful degradation for cleanup operations).
// NOTE: This differs from AddChaos which returns an error for unregistered services
// because AddChaos has a return value that callers typically use.
func (c *ChaosOperations) DisableService(service ServiceName) error {
	proxy := c.registry.GetProxy(service)
	if proxy == nil {
		return nil // Graceful handling of unregistered service
	}

	return DisableProxy(proxy)
}

// EnableService enables the proxy for the given service, restoring connectivity.
// Returns nil if the service is not registered (graceful degradation).
func (c *ChaosOperations) EnableService(service ServiceName) error {
	proxy := c.registry.GetProxy(service)
	if proxy == nil {
		return nil
	}

	return EnableProxy(proxy)
}

// AddChaos injects chaos (toxic) into the proxy for the given service.
// Returns error if the service is not registered.
func (c *ChaosOperations) AddChaos(service ServiceName, config ChaosInjectionConfig) (*toxiproxy.Toxic, error) {
	proxy := c.registry.GetProxy(service)
	if proxy == nil {
		return nil, fmt.Errorf("service %s not registered in proxy registry", service)
	}

	return InjectChaos(proxy, config)
}

// RemoveChaos removes a specific toxic from the service's proxy.
// Returns nil if the service is not registered (graceful degradation).
func (c *ChaosOperations) RemoveChaos(service ServiceName, toxicName string) error {
	proxy := c.registry.GetProxy(service)
	if proxy == nil {
		return nil
	}

	return RemoveChaos(proxy, toxicName)
}

// RemoveAllChaosFromService removes all toxics from the service's proxy.
// Returns nil if the service is not registered.
func (c *ChaosOperations) RemoveAllChaosFromService(service ServiceName) error {
	proxy := c.registry.GetProxy(service)
	if proxy == nil {
		return nil
	}

	return RemoveAllChaos(proxy)
}

// forEachProxy applies an operation to all proxies, collecting errors.
// Returns aggregated error if any operations failed.
func (c *ChaosOperations) forEachProxy(op func(*toxiproxy.Proxy) error, errMsg string) error {
	var errs []error

	for _, proxy := range c.registry.AllProxies() {
		if err := op(proxy); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s: %v", errMsg, errs)
	}

	return nil
}

// EnableAll enables all registered proxies.
func (c *ChaosOperations) EnableAll() error {
	return c.forEachProxy(EnableProxy, "errors enabling proxies")
}

// RemoveAllToxics removes all toxics from all registered proxies.
func (c *ChaosOperations) RemoveAllToxics() error {
	return c.forEachProxy(RemoveAllChaos, "errors removing toxics")
}

// ResetAll removes all toxics and enables all proxies.
// This is useful for cleanup between test cases.
func (c *ChaosOperations) ResetAll() error {
	if err := c.RemoveAllToxics(); err != nil {
		return fmt.Errorf("failed to remove toxics: %w", err)
	}

	if err := c.EnableAll(); err != nil {
		return fmt.Errorf("failed to enable proxies: %w", err)
	}

	return nil
}
