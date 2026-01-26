package chaos

import (
	"sync"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
)

// ServiceName represents a named service in the chaos testing infrastructure.
// Using a typed string enables compile-time safety when accessing proxies.
type ServiceName string

// Service name constants for all infrastructure services.
// These match the proxy names used in Toxiproxy configuration.
const (
	ServiceRabbitMQ      ServiceName = "rabbitmq"
	ServiceMongoMain     ServiceName = "mongo-main"
	ServiceMongoExternal ServiceName = "mongo-external"
	ServiceRedis         ServiceName = "redis"
	ServicePostgres      ServiceName = "postgres"
	ServiceMySQL         ServiceName = "mysql"
	ServiceSQLServer     ServiceName = "sqlserver"
	ServiceOracle        ServiceName = "oracle"
	ServiceSeaweedFS     ServiceName = "seaweedfs"
	ServiceManager       ServiceName = "manager"
)

// AllServices returns a slice of all defined service names.
// Useful for iterating over all proxies (e.g., cleanup, status checks).
// IMPORTANT: Keep this in sync with proxyPortMap in proxy_router.go.
// See TestAllServices_SynchronizedWithProxyPortMap for validation.
func AllServices() []ServiceName {
	return []ServiceName{
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
}

// String returns the string representation of the service name.
// Implements fmt.Stringer for use in log messages and error formatting.
func (s ServiceName) String() string {
	return string(s)
}

// ProxyRegistry provides type-safe access to Toxiproxy proxies by service name.
// It replaces multiple getter methods (GetPostgresProxy, GetMySQLProxy, etc.)
// with a single generic GetProxy(service) method.
type ProxyRegistry struct {
	mu      sync.RWMutex
	proxies map[ServiceName]*toxiproxy.Proxy
}

// NewProxyRegistry creates an empty proxy registry.
func NewProxyRegistry() *ProxyRegistry {
	return &ProxyRegistry{
		proxies: make(map[ServiceName]*toxiproxy.Proxy),
	}
}

// Register adds a proxy to the registry for the given service.
// If proxy is nil, the service is registered but GetProxy will return nil.
func (r *ProxyRegistry) Register(service ServiceName, proxy *toxiproxy.Proxy) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.proxies[service] = proxy
}

// GetProxy returns the proxy for the given service, or nil if not registered.
// Thread-safe for concurrent access.
func (r *ProxyRegistry) GetProxy(service ServiceName) *toxiproxy.Proxy {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.proxies[service]
}

// AllProxies returns all registered proxies as a slice.
// Useful for bulk operations like RemoveAllToxics or EnableAllProxies.
func (r *ProxyRegistry) AllProxies() []*toxiproxy.Proxy {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*toxiproxy.Proxy, 0, len(r.proxies))
	for _, proxy := range r.proxies {
		if proxy != nil {
			result = append(result, proxy)
		}
	}

	return result
}

// RegisterFromStandardProxies populates the registry from containers.StandardProxies.
// This bridges the existing infrastructure with the new registry pattern.
func (r *ProxyRegistry) RegisterFromStandardProxies(sp *StandardProxiesAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sp == nil {
		return
	}

	r.proxies[ServiceRabbitMQ] = sp.RabbitMQ
	r.proxies[ServiceMongoMain] = sp.MongoMain
	r.proxies[ServiceMongoExternal] = sp.MongoExternal
	r.proxies[ServiceRedis] = sp.Redis
	r.proxies[ServicePostgres] = sp.Postgres
	r.proxies[ServiceMySQL] = sp.MySQL
	r.proxies[ServiceSQLServer] = sp.SQLServer
	r.proxies[ServiceOracle] = sp.Oracle
	r.proxies[ServiceSeaweedFS] = sp.SeaweedFS
	r.proxies[ServiceManager] = sp.Manager
}

// StandardProxiesAdapter mirrors containers.StandardProxies for decoupling.
// This allows the chaos package to work without importing containers directly.
type StandardProxiesAdapter struct {
	MongoMain     *toxiproxy.Proxy
	MongoExternal *toxiproxy.Proxy
	RabbitMQ      *toxiproxy.Proxy
	SeaweedFS     *toxiproxy.Proxy
	Redis         *toxiproxy.Proxy
	Postgres      *toxiproxy.Proxy
	MySQL         *toxiproxy.Proxy
	SQLServer     *toxiproxy.Proxy
	Oracle        *toxiproxy.Proxy
	Manager       *toxiproxy.Proxy
}
