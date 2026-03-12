package redis

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
)

type RedisConfig struct {
	Name string

	Image    string
	Password string

	EnableProxy bool
	ProxyName   string

	Options []RedisOption
}

type RedisEndpoint struct {
	URL                  string
	Upstream             string
	ProxyListen          string
	ProxyListenInNetwork string
}

type RedisInfra struct {
	cfg          RedisConfig
	container    *tcredis.RedisContainer
	endpoint     *RedisEndpoint
	networkAlias string // alias for internal network communication
}

func NewRedisInfra(cfg RedisConfig) *RedisInfra {
	if cfg.Image == "" {
		cfg.Image = "redis:7-alpine"
	}

	if cfg.Name == "" {
		cfg.Name = "default"
	}

	if cfg.ProxyName == "" {
		cfg.ProxyName = "redis-" + cfg.Name
	}

	return &RedisInfra{cfg: cfg}
}

func (r *RedisInfra) Start(ctx context.Context, env *itestkit.Env) error {
	opts := defaultRedisOptions()

	for _, opt := range r.cfg.Options {
		if opt != nil {
			opt(opts)
		}
	}

	// Build network alias based on infra name
	alias := fmt.Sprintf("redis-%s", r.cfg.Name)

	runOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithImage(r.cfg.Image),
	}

	// Add to shared network if available
	if env != nil && env.Network != "" {
		runOpts = append(runOpts,
			itestkit.CNetworks(env.Network),
			itestkit.CNetworkAliases(env.Network, alias),
		)
		r.networkAlias = alias
	}

	runOpts = append(runOpts, opts.runOpts...)

	c, err := tcredis.Run(ctx, r.cfg.Image, runOpts...)
	if err != nil {
		return err
	}

	r.container = c

	host, err := c.Host(ctx)
	if err != nil {
		return err
	}

	port, err := c.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return err
	}

	upstream := fmt.Sprintf("%s:%s", host, port.Port())
	hostAddr := upstream
	proxyListen := ""
	proxyListenInNetwork := ""

	if r.cfg.EnableProxy && env != nil && env.Chaos != nil {
		// Use the container's network alias for proxy upstream when in shared network
		var proxyUpstream string
		if r.networkAlias != "" {
			proxyUpstream = fmt.Sprintf("%s:6379", r.networkAlias)
		} else {
			// Fallback to host.docker.internal for backward compatibility
			proxyUpstream = fmt.Sprintf("host.docker.internal:%s", port.Port())
		}

		ref, err := env.Chaos.CreateProxy(ctx, r.cfg.ProxyName, proxyUpstream)
		if err != nil {
			return err
		}

		hostAddr = ref.ListenAddr
		proxyListen = ref.ListenAddr
		proxyListenInNetwork = ref.InNetworkListenAddr
	}

	url := fmt.Sprintf("redis://%s", hostAddr)
	if r.cfg.Password != "" {
		url = fmt.Sprintf("redis://:%s@%s", r.cfg.Password, hostAddr)
	}

	endpoint := RedisEndpoint{
		Upstream:             upstream,
		ProxyListen:          proxyListen,
		ProxyListenInNetwork: proxyListenInNetwork,
		URL:                  url,
	}
	r.endpoint = &endpoint

	return nil
}

func (r *RedisInfra) Endpoint() (RedisEndpoint, error) {
	if r.endpoint == nil {
		return RedisEndpoint{}, fmt.Errorf("redis endpoint not ready")
	}

	return *r.endpoint, nil
}

func (r *RedisInfra) URL() (string, error) {
	endpoint, err := r.Endpoint()
	if err != nil {
		return "", err
	}

	return endpoint.URL, nil
}

func (r *RedisInfra) Addr() (string, error) {
	endpoint, err := r.Endpoint()
	if err != nil {
		return "", err
	}

	if endpoint.ProxyListen != "" {
		return endpoint.ProxyListen, nil
	}

	return endpoint.Upstream, nil
}

// HostPort returns the public, host-usable endpoint.
// Use ContainerHostPort when wiring app containers into a shared Docker network.
func (r *RedisInfra) HostPort() (host string, port int, err error) {
	endpoint, err := r.Endpoint()
	if err != nil {
		return "", 0, err
	}

	return itestkit.ResolveHostHostPort(endpoint.ProxyListen, endpoint.Upstream)
}

// ContainerHostPort returns the endpoint that app containers should use.
func (r *RedisInfra) ContainerHostPort() (host string, port int, err error) {
	endpoint, err := r.Endpoint()
	if err != nil {
		return "", 0, err
	}

	return itestkit.ResolveContainerHostPort(endpoint.ProxyListenInNetwork, r.networkAlias, 6379, endpoint.Upstream)
}

func (r *RedisInfra) Terminate(ctx context.Context) error {
	if r.container != nil {
		return r.container.Terminate(ctx)
	}

	return nil
}

func (r *RedisInfra) InfraKind() string { return "redis" }
func (r *RedisInfra) InfraName() string { return r.cfg.Name }
