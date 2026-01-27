package mongodb

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	tcmongo "github.com/testcontainers/testcontainers-go/modules/mongodb"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
)

type MongoDBConfig struct {
	Name string

	Image    string
	Username string
	Password string

	EnableProxy bool
	ProxyName   string

	Options []MongoDBOption
}

type MongoDBEndpoint struct {
	URI         string
	Upstream    string
	ProxyListen string
}

type MongoDBInfra struct {
	cfg       MongoDBConfig
	container *tcmongo.MongoDBContainer
	endpoint  *MongoDBEndpoint
}

func NewMongoDBInfra(cfg MongoDBConfig) *MongoDBInfra {
	if cfg.Image == "" {
		cfg.Image = "mongo:7"
	}
	if cfg.Name == "" {
		cfg.Name = "default"
	}
	if cfg.ProxyName == "" {
		cfg.ProxyName = "mongo-" + cfg.Name
	}
	return &MongoDBInfra{cfg: cfg}
}

func (m *MongoDBInfra) Start(ctx context.Context, env *itestkit.Env) error {
	opts := defaultMongoDBOptions()
	for _, opt := range m.cfg.Options {
		if opt != nil {
			opt(opts)
		}
	}

	runOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithImage(m.cfg.Image),
	}

	if m.cfg.Username != "" && m.cfg.Password != "" {
		runOpts = append(runOpts,
			tcmongo.WithUsername(m.cfg.Username),
			tcmongo.WithPassword(m.cfg.Password),
		)
	}

	runOpts = append(runOpts, opts.runOpts...)

	c, err := tcmongo.Run(ctx, m.cfg.Image, runOpts...)
	if err != nil {
		return err
	}
	m.container = c

	host, err := c.Host(ctx)
	if err != nil {
		return err
	}
	port, err := c.MappedPort(ctx, "27017/tcp")
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

	var uri string
	if m.cfg.Username != "" && m.cfg.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s", m.cfg.Username, m.cfg.Password, finalAddr)
	} else {
		uri = fmt.Sprintf("mongodb://%s", finalAddr)
	}

	endpoint := MongoDBEndpoint{
		Upstream:    upstream,
		ProxyListen: proxyListen,
		URI:         uri,
	}
	m.endpoint = &endpoint
	return nil
}

func (m *MongoDBInfra) Endpoint() (MongoDBEndpoint, error) {
	if m.endpoint == nil {
		return MongoDBEndpoint{}, fmt.Errorf("mongodb endpoint not ready")
	}
	return *m.endpoint, nil
}

func (m *MongoDBInfra) URI() (string, error) {
	endpoint, err := m.Endpoint()
	if err != nil {
		return "", err
	}
	return endpoint.URI, nil
}

// HostPort returns the host and port as separate values.
// If a proxy is configured, returns the proxy address; otherwise returns the upstream address.
// The host is automatically normalized so containers can reach it (localhost is replaced with
// the Docker gateway IP).
func (m *MongoDBInfra) HostPort() (host string, port int, err error) {
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

func (m *MongoDBInfra) Terminate(ctx context.Context) error {
	if m.container != nil {
		return m.container.Terminate(ctx)
	}
	return nil
}

func (m *MongoDBInfra) InfraKind() string { return "mongodb" }
func (m *MongoDBInfra) InfraName() string { return m.cfg.Name }
