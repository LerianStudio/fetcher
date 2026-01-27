package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	URL         string
	Upstream    string
	ProxyListen string
}

type RedisInfra struct {
	cfg       RedisConfig
	container *tcredis.RedisContainer
	endpoint  *RedisEndpoint
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

	runOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithImage(r.cfg.Image),
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
	finalAddr := upstream
	proxyListen := ""

	if r.cfg.EnableProxy && env != nil && env.Chaos != nil {
		ref, err := env.Chaos.CreateProxy(ctx, r.cfg.ProxyName, upstream)
		if err != nil {
			return err
		}
		finalAddr = ref.ListenAddr
		proxyListen = ref.ListenAddr
	}

	url := fmt.Sprintf("redis://%s", finalAddr)
	if r.cfg.Password != "" {
		url = fmt.Sprintf("redis://:%s@%s", r.cfg.Password, finalAddr)
	}

	endpoint := RedisEndpoint{
		Upstream:    upstream,
		ProxyListen: proxyListen,
		URL:         url,
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

// HostPort returns the host and port as separate values.
// If a proxy is configured, returns the proxy address; otherwise returns the upstream address.
// The host is automatically normalized so containers can reach it (localhost is replaced with
// the Docker gateway IP).
func (r *RedisInfra) HostPort() (host string, port int, err error) {
	endpoint, err := r.Endpoint()
	if err != nil {
		return "", 0, err
	}

	addr := endpoint.Upstream
	if endpoint.ProxyListen != "" {
		addr = endpoint.ProxyListen
	}

	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format: %s", addr)
	}

	portNum, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", parts[1])
	}

	return itestkit.NormalizeHost(parts[0]), portNum, nil
}

func (r *RedisInfra) Terminate(ctx context.Context) error {
	if r.container != nil {
		return r.container.Terminate(ctx)
	}
	return nil
}

func (r *RedisInfra) InfraKind() string { return "redis" }
func (r *RedisInfra) InfraName() string { return r.cfg.Name }
