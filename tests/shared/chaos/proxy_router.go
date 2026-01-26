package chaos

import (
	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// proxyPortMap defines the Toxiproxy listen ports for each service.
// These must match the ports exposed in ToxiproxyContainer configuration.
var proxyPortMap = map[ServiceName]int{
	ServicePostgres:      5433,
	ServiceMySQL:         3307,
	ServiceSQLServer:     1434,
	ServiceOracle:        1522,
	ServiceMongoMain:     27100,
	ServiceMongoExternal: 27101,
	ServiceRabbitMQ:      5673,
	ServiceSeaweedFS:     8889,
	ServiceRedis:         6380,
	ServiceManager:       4007,
}

// ProxyRouter routes connections through Toxiproxy.
// It replaces the 5 service-specific methods (PostgresProxyInternal, etc.)
// with a single generic method.
type ProxyRouter struct {
	toxiproxyHost string
}

// NewProxyRouter creates a ProxyRouter that routes through the given Toxiproxy host.
// The host is typically "toxiproxy" (Docker network alias).
func NewProxyRouter(toxiproxyHost string) *ProxyRouter {
	return &ProxyRouter{toxiproxyHost: toxiproxyHost}
}

// GetProxyConnection returns connection info routed through Toxiproxy.
// The returned connection uses Toxiproxy's host and the mapped proxy port,
// preserving credentials and database name from the direct connection.
//
// If the service is not in the proxy port map, returns the direct connection unchanged.
func (r *ProxyRouter) GetProxyConnection(service ServiceName, direct config.InternalDBConnection) config.InternalDBConnection {
	port, ok := proxyPortMap[service]
	if !ok {
		// Unknown service - return direct connection
		return direct
	}

	direct.Host = r.toxiproxyHost
	direct.Port = port

	return direct
}

// GetProxyPort returns the Toxiproxy listen port for a service.
// Returns 0 if the service is not configured.
func (r *ProxyRouter) GetProxyPort(service ServiceName) int {
	return proxyPortMap[service]
}
