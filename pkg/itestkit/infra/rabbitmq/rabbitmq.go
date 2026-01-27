package rabbitmq

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	rmq "github.com/testcontainers/testcontainers-go/modules/rabbitmq"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
)

type RabbitConfig struct {
	Name string

	Image       string
	Username    string
	Password    string
	VHost       string
	EnableProxy bool
	ProxyName   string

	Options []RabbitOption
}

type RabbitEndpoint struct {
	AMQPURL     string
	Upstream    string
	ProxyListen string
}

type RabbitInfra struct {
	cfg       RabbitConfig
	container *rmq.RabbitMQContainer
	endpoint  *RabbitEndpoint
}

func NewRabbitInfra(cfg RabbitConfig) *RabbitInfra {
	if cfg.Image == "" {
		cfg.Image = "rabbitmq:3.13-management-alpine"
	}
	if cfg.Username == "" {
		cfg.Username = "guest"
	}
	if cfg.Password == "" {
		cfg.Password = "guest"
	}
	if cfg.VHost == "" {
		cfg.VHost = "/"
	}
	if cfg.Name == "" {
		cfg.Name = "default"
	}
	if cfg.ProxyName == "" {
		cfg.ProxyName = "amqp-" + cfg.Name
	}
	return &RabbitInfra{cfg: cfg}
}

func (r *RabbitInfra) Start(ctx context.Context, env *itestkit.Env) error {
	opts := defaultRabbitOptions()
	for _, opt := range r.cfg.Options {
		if opt != nil {
			opt(opts)
		}
	}

	runOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithImage(r.cfg.Image),
		rmq.WithAdminUsername(r.cfg.Username),
		rmq.WithAdminPassword(r.cfg.Password),
	}
	runOpts = append(runOpts, opts.runOpts...)

	c, err := rmq.RunContainer(ctx, runOpts...)
	if err != nil {
		return err
	}
	r.container = c

	host, err := c.Host(ctx)
	if err != nil {
		return err
	}
	amqpPort, err := c.MappedPort(ctx, "5672/tcp")
	if err != nil {
		return err
	}

	upstream := fmt.Sprintf("%s:%s", host, amqpPort.Port())
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

	endpoint := RabbitEndpoint{
		Upstream:    upstream,
		ProxyListen: proxyListen,
		AMQPURL:     fmt.Sprintf("amqp://%s:%s@%s%s", r.cfg.Username, r.cfg.Password, finalAddr, r.cfg.VHost),
	}
	r.endpoint = &endpoint
	return nil
}

func (r *RabbitInfra) Endpoint() (RabbitEndpoint, error) {
	if r.endpoint == nil {
		return RabbitEndpoint{}, fmt.Errorf("rabbitmq endpoint not ready")
	}
	return *r.endpoint, nil
}

func (r *RabbitInfra) AMQPURL() (string, error) {
	endpoint, err := r.Endpoint()
	if err != nil {
		return "", err
	}
	return endpoint.AMQPURL, nil
}

// HostPort returns the host and port as separate values.
// If a proxy is configured, returns the proxy address; otherwise returns the upstream address.
// The host is automatically normalized so containers can reach it (localhost is replaced with
// the Docker gateway IP).
func (r *RabbitInfra) HostPort() (host string, port int, err error) {
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

func (r *RabbitInfra) Terminate(ctx context.Context) error {
	if r.container != nil {
		return r.container.Terminate(ctx)
	}
	return nil
}

func (r *RabbitInfra) InfraKind() string { return "rabbitmq" }
func (r *RabbitInfra) InfraName() string { return r.cfg.Name }
