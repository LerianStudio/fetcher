package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
)

type PostgresConfig struct {
	Name string

	Image    string
	Database string
	Username string
	Password string

	EnableProxy bool
	ProxyName   string

	Options []PostgresOption
}

type PostgresEndpoint struct {
	DSN         string
	Upstream    string
	ProxyListen string
}

type PostgresInfra struct {
	cfg       PostgresConfig
	container *pg.PostgresContainer
	endpoint  *PostgresEndpoint
}

func NewPostgresInfra(cfg PostgresConfig) *PostgresInfra {
	if cfg.Image == "" {
		cfg.Image = "postgres:16-alpine"
	}
	if cfg.Database == "" {
		cfg.Database = "app"
	}
	if cfg.Username == "" {
		cfg.Username = "app"
	}
	if cfg.Password == "" {
		cfg.Password = "app"
	}
	if cfg.Name == "" {
		cfg.Name = "default"
	}
	if cfg.ProxyName == "" {
		cfg.ProxyName = "pg-" + cfg.Name
	}
	return &PostgresInfra{cfg: cfg}
}

func (p *PostgresInfra) Start(ctx context.Context, env *itestkit.Env) error {
	opts := defaultPostgresOptions()
	for _, opt := range p.cfg.Options {
		if opt != nil {
			opt(opts)
		}
	}

	runOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithImage(p.cfg.Image),
		pg.WithDatabase(p.cfg.Database),
		pg.WithUsername(p.cfg.Username),
		pg.WithPassword(p.cfg.Password),
	}
	runOpts = append(runOpts, opts.runOpts...)

	c, err := pg.Run(ctx, p.cfg.Image, runOpts...)
	if err != nil {
		return err
	}
	p.container = c

	host, err := c.Host(ctx)
	if err != nil {
		return err
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return err
	}

	upstream := fmt.Sprintf("%s:%s", host, port.Port())
	finalAddr := upstream
	proxyListen := ""

	if p.cfg.EnableProxy && env != nil && env.Chaos != nil {
		ref, err := env.Chaos.CreateProxy(ctx, p.cfg.ProxyName, upstream)
		if err != nil {
			return err
		}
		finalAddr = ref.ListenAddr
		proxyListen = ref.ListenAddr
	}

	endpoint := PostgresEndpoint{
		Upstream:    upstream,
		ProxyListen: proxyListen,
		DSN:         fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", p.cfg.Username, p.cfg.Password, finalAddr, p.cfg.Database),
	}
	p.endpoint = &endpoint
	return nil
}

func (p *PostgresInfra) Endpoint() (PostgresEndpoint, error) {
	if p.endpoint == nil {
		return PostgresEndpoint{}, fmt.Errorf("postgres endpoint not ready")
	}
	return *p.endpoint, nil
}

func (p *PostgresInfra) DSN() (string, error) {
	endpoint, err := p.Endpoint()
	if err != nil {
		return "", err
	}
	return endpoint.DSN, nil
}

// HostPort returns the host and port as separate values.
// If a proxy is configured, returns the proxy address; otherwise returns the upstream address.
// The host is automatically normalized so containers can reach it (localhost is replaced with
// the Docker gateway IP).
func (p *PostgresInfra) HostPort() (host string, port int, err error) {
	endpoint, err := p.Endpoint()
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

func (p *PostgresInfra) Terminate(ctx context.Context) error {
	if p.container != nil {
		return p.container.Terminate(ctx)
	}
	return nil
}

func (p *PostgresInfra) InfraKind() string { return "postgres" }
func (p *PostgresInfra) InfraName() string { return p.cfg.Name }
