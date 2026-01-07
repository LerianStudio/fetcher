package containers

import (
	"context"
	"fmt"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// ToxiproxyContainer wraps a Toxiproxy testcontainer with proxy management.
type ToxiproxyContainer struct {
	Container    testcontainers.Container
	Client       *toxiproxy.Client
	Host         string
	APIPort      string
	InternalHost string
	ProxyPorts   map[string]string // Map of proxy name to host port
}

// ToxiproxyOptions configures Toxiproxy container startup.
type ToxiproxyOptions struct {
	NetworkName   string
	NetworkAlias  string
	FixedHostPort string // For API port (8474)
}

// DefaultToxiproxyOptions returns default Toxiproxy options.
func DefaultToxiproxyOptions(networkName string) ToxiproxyOptions {
	return ToxiproxyOptions{
		NetworkName:  networkName,
		NetworkAlias: "toxiproxy",
	}
}

// StartToxiproxy starts a Toxiproxy container with the given options.
func StartToxiproxy(ctx context.Context, opts ToxiproxyOptions) (*ToxiproxyContainer, error) {
	req := testcontainers.ContainerRequest{
		Image: "ghcr.io/shopify/toxiproxy:2.9.0",
		ExposedPorts: []string{
			"8474/tcp", // API port
			// Reserve ports for proxies
			"27100/tcp", // MongoDB Main proxy
			"27101/tcp", // MongoDB External proxy
			"5673/tcp",  // RabbitMQ proxy
			"8889/tcp",  // SeaweedFS proxy
			"6380/tcp",  // Redis proxy
			"5433/tcp",  // PostgreSQL proxy
			"3307/tcp",  // MySQL proxy
			"1434/tcp",  // SQL Server proxy
			"1522/tcp",  // Oracle proxy
			"4007/tcp",  // Manager proxy
		},
		WaitingFor: wait.ForHTTP("/version").WithPort("8474/tcp").WithStartupTimeout(config.ToxiproxyStartupTimeout),
	}

	if opts.NetworkName != "" {
		req.Networks = []string{opts.NetworkName}
		req.NetworkAliases = map[string][]string{
			opts.NetworkName: {opts.NetworkAlias},
		}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Toxiproxy: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Toxiproxy host: %w", err)
	}

	apiPort, err := container.MappedPort(ctx, "8474")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Toxiproxy API port: %w", err)
	}

	// Create Toxiproxy client
	client := toxiproxy.NewClient(fmt.Sprintf("http://%s:%s", host, apiPort.Port()))

	return &ToxiproxyContainer{
		Container:    container,
		Client:       client,
		Host:         host,
		APIPort:      apiPort.Port(),
		InternalHost: opts.NetworkAlias,
		ProxyPorts:   make(map[string]string),
	}, nil
}

// Stop terminates the Toxiproxy container.
func (t *ToxiproxyContainer) Stop(ctx context.Context) error {
	if t.Container != nil {
		return t.Container.Terminate(ctx)
	}

	return nil
}

// ProxyConfig defines configuration for creating a proxy.
type ProxyConfig struct {
	Name     string
	Listen   string // Format: "host:port" for Toxiproxy to listen on
	Upstream string // Format: "host:port" of the upstream service
}

// CreateProxy creates a new proxy with the given configuration.
func (t *ToxiproxyContainer) CreateProxy(cfg ProxyConfig) (*toxiproxy.Proxy, error) {
	proxy, err := t.Client.CreateProxy(cfg.Name, cfg.Listen, cfg.Upstream)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy %s: %w", cfg.Name, err)
	}

	return proxy, nil
}

// GetProxyHostPort returns the host:port for accessing a proxy from the host machine.
func (t *ToxiproxyContainer) GetProxyHostPort(ctx context.Context, containerPort string) (string, error) {
	port, err := t.Container.MappedPort(ctx, nat.Port(containerPort))
	if err != nil {
		return "", fmt.Errorf("failed to get mapped port %s: %w", containerPort, err)
	}

	return fmt.Sprintf("%s:%s", t.Host, port.Port()), nil
}

// StandardProxies defines the standard proxy configurations for Fetcher infrastructure.
type StandardProxies struct {
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

// CreateStandardProxies creates proxies for all standard infrastructure components.
// The upstream addresses should be the Docker network hostnames (e.g., "fetcher-mongodb:27017").
func (t *ToxiproxyContainer) CreateStandardProxies(upstreams StandardUpstreams) (*StandardProxies, error) {
	proxies := &StandardProxies{}
	proxyConfigs := []struct {
		name     string
		listen   string
		upstream string
		target   **toxiproxy.Proxy
	}{
		{"mongo-main", "0.0.0.0:27100", upstreams.MongoMain, &proxies.MongoMain},
		{"mongo-external", "0.0.0.0:27101", upstreams.MongoExternal, &proxies.MongoExternal},
		{"rabbitmq", "0.0.0.0:5673", upstreams.RabbitMQ, &proxies.RabbitMQ},
		{"seaweedfs", "0.0.0.0:8889", upstreams.SeaweedFS, &proxies.SeaweedFS},
		{"redis", "0.0.0.0:6380", upstreams.Redis, &proxies.Redis},
		{"postgres", "0.0.0.0:5433", upstreams.Postgres, &proxies.Postgres},
		{"mysql", "0.0.0.0:3307", upstreams.MySQL, &proxies.MySQL},
		{"sqlserver", "0.0.0.0:1434", upstreams.SQLServer, &proxies.SQLServer},
		{"oracle", "0.0.0.0:1522", upstreams.Oracle, &proxies.Oracle},
		{"manager", "0.0.0.0:4007", upstreams.Manager, &proxies.Manager},
	}

	for _, cfg := range proxyConfigs {
		if cfg.upstream != "" {
			var err error

			*cfg.target, err = t.CreateProxy(ProxyConfig{
				Name:     cfg.name,
				Listen:   cfg.listen,
				Upstream: cfg.upstream,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return proxies, nil
}

// StandardUpstreams defines upstream addresses for standard infrastructure.
type StandardUpstreams struct {
	MongoMain     string // e.g., "fetcher-mongodb:27017"
	MongoExternal string // e.g., "fetcher-mongodb-external:27017"
	RabbitMQ      string // e.g., "fetcher-rabbitmq:5672"
	SeaweedFS     string // e.g., "fetcher-seaweedfs-filer:8888"
	Redis         string // e.g., "fetcher-valkey:6379"
	Postgres      string // e.g., "postgres-external:5432"
	MySQL         string // e.g., "mysql-external:3306"
	SQLServer     string // e.g., "sqlserver-external:1433"
	Oracle        string // e.g., "oracle-external:1521"
	Manager       string // e.g., "manager:4006"
}

// DefaultStandardUpstreams returns the default upstream addresses using Docker network hostnames.
func DefaultStandardUpstreams() StandardUpstreams {
	return StandardUpstreams{
		MongoMain:     "fetcher-mongodb:27017",
		MongoExternal: "fetcher-mongodb-external:27017",
		RabbitMQ:      "fetcher-rabbitmq:5672",
		SeaweedFS:     "fetcher-seaweedfs-filer:8888",
		Redis:         "fetcher-valkey:6379",
		Postgres:      "postgres-external:5432",
		MySQL:         "mysql-external:3306",
		SQLServer:     "sqlserver-external:1433",
		Oracle:        "oracle-external:1521",
		Manager:       "manager:4006",
	}
}

// DisableProxy disables a proxy, preventing all connections.
func DisableProxy(proxy *toxiproxy.Proxy) error {
	if proxy == nil {
		return nil
	}

	proxy.Enabled = false

	return proxy.Save()
}

// EnableProxy enables a proxy, allowing connections.
func EnableProxy(proxy *toxiproxy.Proxy) error {
	if proxy == nil {
		return nil
	}

	proxy.Enabled = true

	return proxy.Save()
}

// AddLatency adds latency to a proxy.
func AddLatency(proxy *toxiproxy.Proxy, name string, latencyMS int, jitterMS int) (*toxiproxy.Toxic, error) {
	if proxy == nil {
		return nil, fmt.Errorf("proxy is nil")
	}

	return proxy.AddToxic(name, "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": latencyMS,
		"jitter":  jitterMS,
	})
}

// AddTimeout adds a timeout toxic that stops data from flowing.
func AddTimeout(proxy *toxiproxy.Proxy, name string, timeoutMS int) (*toxiproxy.Toxic, error) {
	if proxy == nil {
		return nil, fmt.Errorf("proxy is nil")
	}

	return proxy.AddToxic(name, "timeout", "downstream", 1.0, toxiproxy.Attributes{
		"timeout": timeoutMS,
	})
}

// AddBandwidth limits the bandwidth of a proxy.
func AddBandwidth(proxy *toxiproxy.Proxy, name string, rateBytesPerSec int) (*toxiproxy.Toxic, error) {
	if proxy == nil {
		return nil, fmt.Errorf("proxy is nil")
	}

	return proxy.AddToxic(name, "bandwidth", "downstream", 1.0, toxiproxy.Attributes{
		"rate": rateBytesPerSec,
	})
}

// AddSlowClose delays the closing of connections.
func AddSlowClose(proxy *toxiproxy.Proxy, name string, delayMS int) (*toxiproxy.Toxic, error) {
	if proxy == nil {
		return nil, fmt.Errorf("proxy is nil")
	}

	return proxy.AddToxic(name, "slow_close", "downstream", 1.0, toxiproxy.Attributes{
		"delay": delayMS,
	})
}

// AddResetPeer resets connections after a delay.
func AddResetPeer(proxy *toxiproxy.Proxy, name string, timeoutMS int) (*toxiproxy.Toxic, error) {
	if proxy == nil {
		return nil, fmt.Errorf("proxy is nil")
	}

	return proxy.AddToxic(name, "reset_peer", "downstream", 1.0, toxiproxy.Attributes{
		"timeout": timeoutMS,
	})
}

// RemoveToxic removes a toxic from a proxy by name.
func RemoveToxic(proxy *toxiproxy.Proxy, name string) error {
	if proxy == nil {
		return nil
	}

	return proxy.RemoveToxic(name)
}

// RemoveAllToxics removes all toxics from a proxy.
func RemoveAllToxics(proxy *toxiproxy.Proxy) error {
	if proxy == nil {
		return nil
	}

	toxics, err := proxy.Toxics()
	if err != nil {
		return fmt.Errorf("failed to get toxics: %w", err)
	}

	for _, toxic := range toxics {
		if err := proxy.RemoveToxic(toxic.Name); err != nil {
			return fmt.Errorf("failed to remove toxic %s: %w", toxic.Name, err)
		}
	}

	return nil
}

// AddLimitData limits the amount of data that can be transmitted.
// After the specified number of bytes have been transmitted, the connection is closed.
func AddLimitData(proxy *toxiproxy.Proxy, name string, bytes int) (*toxiproxy.Toxic, error) {
	if proxy == nil {
		return nil, fmt.Errorf("proxy is nil")
	}

	return proxy.AddToxic(name, "limit_data", "downstream", 1.0, toxiproxy.Attributes{
		"bytes": bytes,
	})
}

// AddSlicer slices data into smaller bits, optionally adding delay between each slice.
// avgSize: average size of each slice in bytes
// sizeVariation: variation in slice size (bytes)
// delay: delay in microseconds between each slice
func AddSlicer(proxy *toxiproxy.Proxy, name string, avgSize, sizeVariation, delay int) (*toxiproxy.Toxic, error) {
	if proxy == nil {
		return nil, fmt.Errorf("proxy is nil")
	}

	return proxy.AddToxic(name, "slicer", "downstream", 1.0, toxiproxy.Attributes{
		"average_size":   avgSize,
		"size_variation": sizeVariation,
		"delay":          delay,
	})
}
