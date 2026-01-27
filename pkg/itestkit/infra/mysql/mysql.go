package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
)

type MySQLConfig struct {
	Name string

	Image        string
	Database     string
	Username     string
	Password     string
	RootPassword string

	EnableProxy bool
	ProxyName   string

	Options []MySQLOption
}

type MySQLEndpoint struct {
	DSN         string
	Upstream    string
	ProxyListen string
}

type MySQLInfra struct {
	cfg       MySQLConfig
	container *tcmysql.MySQLContainer
	endpoint  *MySQLEndpoint
}

func NewMySQLInfra(cfg MySQLConfig) *MySQLInfra {
	if cfg.Image == "" {
		cfg.Image = "mysql:8.0"
	}
	if cfg.Database == "" {
		cfg.Database = "testdb"
	}
	if cfg.Username == "" {
		cfg.Username = "testuser"
	}
	if cfg.Password == "" {
		cfg.Password = "testpass"
	}
	if cfg.RootPassword == "" {
		cfg.RootPassword = cfg.Password
	}
	if cfg.Name == "" {
		cfg.Name = "default"
	}
	if cfg.ProxyName == "" {
		cfg.ProxyName = "mysql-" + cfg.Name
	}
	return &MySQLInfra{cfg: cfg}
}

func (m *MySQLInfra) Start(ctx context.Context, env *itestkit.Env) error {
	opts := defaultMySQLOptions()
	for _, opt := range m.cfg.Options {
		if opt != nil {
			opt(opts)
		}
	}

	runOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithImage(m.cfg.Image),
		tcmysql.WithDatabase(m.cfg.Database),
		tcmysql.WithUsername(m.cfg.Username),
		tcmysql.WithPassword(m.cfg.Password),
	}
	runOpts = append(runOpts, opts.runOpts...)

	c, err := tcmysql.Run(ctx, m.cfg.Image, runOpts...)
	if err != nil {
		return err
	}
	m.container = c

	host, err := c.Host(ctx)
	if err != nil {
		return err
	}
	port, err := c.MappedPort(ctx, "3306/tcp")
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

	endpoint := MySQLEndpoint{
		Upstream:    upstream,
		ProxyListen: proxyListen,
		DSN:         fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", m.cfg.Username, m.cfg.Password, finalAddr, m.cfg.Database),
	}
	m.endpoint = &endpoint
	return nil
}

func (m *MySQLInfra) Endpoint() (MySQLEndpoint, error) {
	if m.endpoint == nil {
		return MySQLEndpoint{}, fmt.Errorf("mysql endpoint not ready")
	}
	return *m.endpoint, nil
}

func (m *MySQLInfra) DSN() (string, error) {
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
func (m *MySQLInfra) HostPort() (host string, port int, err error) {
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

func (m *MySQLInfra) Terminate(ctx context.Context) error {
	if m.container != nil {
		return m.container.Terminate(ctx)
	}
	return nil
}

func (m *MySQLInfra) InfraKind() string { return "mysql" }
func (m *MySQLInfra) InfraName() string { return m.cfg.Name }
