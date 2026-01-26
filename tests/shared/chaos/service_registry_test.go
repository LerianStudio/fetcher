package chaos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceName_AllServicesAreDefined(t *testing.T) {
	// Verify all expected services are defined as constants
	services := []ServiceName{
		ServiceRabbitMQ,
		ServiceMongoMain,
		ServiceMongoExternal,
		ServiceRedis,
		ServicePostgres,
		ServiceMySQL,
		ServiceSQLServer,
		ServiceOracle,
		ServiceSeaweedFS,
		ServiceManager,
	}

	assert.Len(t, services, 10, "Should have exactly 10 service constants")

	// Verify service names are non-empty strings
	for _, svc := range services {
		assert.NotEmpty(t, string(svc), "Service name should not be empty")
	}
}

func TestServiceName_String(t *testing.T) {
	assert.Equal(t, "rabbitmq", string(ServiceRabbitMQ))
	assert.Equal(t, "postgres", string(ServicePostgres))
	assert.Equal(t, "manager", string(ServiceManager))
}

func TestAllServices_ReturnsAllServiceNames(t *testing.T) {
	services := AllServices()
	assert.Len(t, services, 10, "AllServices should return 10 services")
}

func TestProxyRegistry_GetProxy(t *testing.T) {
	// Create a mock proxy for testing
	registry := NewProxyRegistry()

	// Initially empty
	proxy := registry.GetProxy(ServicePostgres)
	assert.Nil(t, proxy, "Should return nil for unregistered service")

	// After registration, should return the proxy
	// Note: We can't test with real proxies without Toxiproxy running
	// This test verifies the registry mechanics work
}

func TestProxyRegistry_RegisterAndGet(t *testing.T) {
	registry := NewProxyRegistry()

	// Register should not panic with nil (graceful handling)
	registry.Register(ServicePostgres, nil)

	// Get should return nil for nil registration
	proxy := registry.GetProxy(ServicePostgres)
	assert.Nil(t, proxy)
}

func TestProxyRegistry_AllProxies(t *testing.T) {
	registry := NewProxyRegistry()

	// Initially empty
	all := registry.AllProxies()
	assert.Empty(t, all)
}

func TestAllServices_SynchronizedWithProxyPortMap(t *testing.T) {
	// This test ensures AllServices() stays in sync with proxyPortMap.
	// If a new service is added to proxyPortMap but not to AllServices(),
	// bulk operations (RemoveAllToxics, EnableAll) would miss that service.

	services := AllServices()

	// Verify each service in AllServices has a port mapping
	for _, svc := range services {
		port := proxyPortMap[svc]
		assert.NotZero(t, port, "Service %s should have a port mapping in proxyPortMap", svc)
	}

	// Verify counts match - catches services in proxyPortMap but not in AllServices
	assert.Equal(t, len(services), len(proxyPortMap),
		"AllServices() count (%d) should match proxyPortMap count (%d)",
		len(services), len(proxyPortMap))
}
