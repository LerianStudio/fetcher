package mssql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	tcmssql "github.com/testcontainers/testcontainers-go/modules/mssql"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
)

type MSSQLConfig struct {
	Name string

	Image    string
	Password string
	Database string

	EnableProxy bool
	ProxyName   string

	Options []MSSQLOption
}

type MSSQLEndpoint struct {
	DSN         string
	Upstream    string
	ProxyListen string
}

type MSSQLInfra struct {
	cfg       MSSQLConfig
	container *tcmssql.MSSQLServerContainer
	endpoint  *MSSQLEndpoint
}

func NewMSSQLInfra(cfg MSSQLConfig) *MSSQLInfra {
	if cfg.Image == "" {
		cfg.Image = "mcr.microsoft.com/mssql/server:2022-latest"
	}
	if cfg.Password == "" {
		cfg.Password = "YourStrong@Passw0rd"
	}
	if cfg.Name == "" {
		cfg.Name = "default"
	}
	if cfg.ProxyName == "" {
		cfg.ProxyName = "mssql-" + cfg.Name
	}
	return &MSSQLInfra{cfg: cfg}
}

func (m *MSSQLInfra) Start(ctx context.Context, env *itestkit.Env) error {
	opts := defaultMSSQLOptions()
	for _, opt := range m.cfg.Options {
		if opt != nil {
			opt(opts)
		}
	}

	runOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithImage(m.cfg.Image),
		tcmssql.WithAcceptEULA(),
		tcmssql.WithPassword(m.cfg.Password),
	}
	runOpts = append(runOpts, opts.runOpts...)

	c, err := tcmssql.Run(ctx, m.cfg.Image, runOpts...)
	if err != nil {
		return err
	}
	m.container = c

	host, err := c.Host(ctx)
	if err != nil {
		return err
	}
	port, err := c.MappedPort(ctx, "1433/tcp")
	if err != nil {
		return err
	}

	upstream := fmt.Sprintf("%s:%s", host, port.Port())
	finalAddr := upstream
	proxyListen := ""

	if m.cfg.EnableProxy && env != nil && env.Chaos != nil {
		ref, err := env.Chaos.CreateProxy(ctx, m.cfg.ProxyName, upstream)
		if err != nil {
			return err
		}
		finalAddr = ref.ListenAddr
		proxyListen = ref.ListenAddr
	}

	dsn := fmt.Sprintf("sqlserver://sa:%s@%s", m.cfg.Password, finalAddr)
	if m.cfg.Database != "" {
		dsn = fmt.Sprintf("%s?database=%s", dsn, m.cfg.Database)
	}

	endpoint := MSSQLEndpoint{
		Upstream:    upstream,
		ProxyListen: proxyListen,
		DSN:         dsn,
	}
	m.endpoint = &endpoint
	return nil
}

func (m *MSSQLInfra) Endpoint() (MSSQLEndpoint, error) {
	if m.endpoint == nil {
		return MSSQLEndpoint{}, fmt.Errorf("mssql endpoint not ready")
	}
	return *m.endpoint, nil
}

func (m *MSSQLInfra) DSN() (string, error) {
	endpoint, err := m.Endpoint()
	if err != nil {
		return "", err
	}
	return endpoint.DSN, nil
}

// HostPort returns the host and port as separate values.
// If a proxy is configured, returns the proxy address; otherwise returns the upstream address.
// The host is automatically normalized so containers can reach it (localhost is replaced with
// the Docker gateway IP).
func (m *MSSQLInfra) HostPort() (host string, port int, err error) {
	endpoint, err := m.Endpoint()
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

func (m *MSSQLInfra) Terminate(ctx context.Context) error {
	if m.container != nil {
		return m.container.Terminate(ctx)
	}
	return nil
}

func (m *MSSQLInfra) InfraKind() string { return "mssql" }
func (m *MSSQLInfra) InfraName() string { return m.cfg.Name }
