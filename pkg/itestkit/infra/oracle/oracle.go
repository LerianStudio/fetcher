package oracle

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
)

type OracleConfig struct {
	Name string

	Image    string
	Password string
	SID      string

	EnableProxy bool
	ProxyName   string

	Options []OracleOption
}

type OracleEndpoint struct {
	DSN         string
	Upstream    string
	ProxyListen string
}

type OracleInfra struct {
	cfg       OracleConfig
	container testcontainers.Container
	endpoint  *OracleEndpoint
}

func NewOracleInfra(cfg OracleConfig) *OracleInfra {
	if cfg.Image == "" {
		cfg.Image = "gvenzl/oracle-xe:21-slim"
	}
	if cfg.Password == "" {
		cfg.Password = "testpass"
	}
	if cfg.SID == "" {
		cfg.SID = "XE"
	}
	if cfg.Name == "" {
		cfg.Name = "default"
	}
	if cfg.ProxyName == "" {
		cfg.ProxyName = "oracle-" + cfg.Name
	}
	return &OracleInfra{cfg: cfg}
}

func (o *OracleInfra) Start(ctx context.Context, env *itestkit.Env) error {
	opts := defaultOracleOptions()
	for _, opt := range o.cfg.Options {
		if opt != nil {
			opt(opts)
		}
	}

	req := testcontainers.ContainerRequest{
		Image:        o.cfg.Image,
		ExposedPorts: []string{"1521/tcp"},
		Env: map[string]string{
			"ORACLE_PASSWORD": o.cfg.Password,
		},
		WaitingFor: wait.ForLog("DATABASE IS READY TO USE!").WithStartupTimeout(5 * time.Minute),
	}

	genericReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}

	for _, runOpt := range opts.runOpts {
		if err := runOpt.Customize(&genericReq); err != nil {
			return err
		}
	}

	c, err := testcontainers.GenericContainer(ctx, genericReq)
	if err != nil {
		return err
	}
	o.container = c

	host, err := c.Host(ctx)
	if err != nil {
		return err
	}
	port, err := c.MappedPort(ctx, "1521/tcp")
	if err != nil {
		return err
	}

	upstream := fmt.Sprintf("%s:%s", host, port.Port())
	finalAddr := upstream
	proxyListen := ""

	if o.cfg.EnableProxy && env != nil && env.Chaos != nil {
		ref, err := env.Chaos.CreateProxy(ctx, o.cfg.ProxyName, upstream)
		if err != nil {
			return err
		}
		finalAddr = ref.ListenAddr
		proxyListen = ref.ListenAddr
	}

	endpoint := OracleEndpoint{
		Upstream:    upstream,
		ProxyListen: proxyListen,
		DSN:         fmt.Sprintf("oracle://system:%s@%s/%s", o.cfg.Password, finalAddr, o.cfg.SID),
	}
	o.endpoint = &endpoint
	return nil
}

func (o *OracleInfra) Endpoint() (OracleEndpoint, error) {
	if o.endpoint == nil {
		return OracleEndpoint{}, fmt.Errorf("oracle endpoint not ready")
	}
	return *o.endpoint, nil
}

func (o *OracleInfra) DSN() (string, error) {
	endpoint, err := o.Endpoint()
	if err != nil {
		return "", err
	}
	return endpoint.DSN, nil
}

func (o *OracleInfra) GoDRORDSN() (string, error) {
	endpoint, err := o.Endpoint()
	if err != nil {
		return "", err
	}
	addr := endpoint.Upstream
	if endpoint.ProxyListen != "" {
		addr = endpoint.ProxyListen
	}
	return fmt.Sprintf("system/%s@%s/%s", o.cfg.Password, addr, o.cfg.SID), nil
}

// HostPort returns the host and port as separate values.
// If a proxy is configured, returns the proxy address; otherwise returns the upstream address.
// The host is automatically normalized so containers can reach it (localhost is replaced with
// the Docker gateway IP).
func (o *OracleInfra) HostPort() (host string, port int, err error) {
	endpoint, err := o.Endpoint()
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

func (o *OracleInfra) Terminate(ctx context.Context) error {
	if o.container != nil {
		return o.container.Terminate(ctx)
	}
	return nil
}

func (o *OracleInfra) InfraKind() string { return "oracle" }
func (o *OracleInfra) InfraName() string { return o.cfg.Name }
