package itestkit

import (
	"context"
	"fmt"
	"time"

	toxiclient "github.com/Shopify/toxiproxy/v2/client"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type toxiproxyChaos struct {
	container testcontainers.Container
	client    *toxiclient.Client
	proxies   map[string]*toxiclient.Proxy
}

func NewToxiproxyChaos(ctx context.Context, cfg ChaosConfig) (ChaosInterface, error) {
	image := cfg.Image
	if image == "" {
		image = "ghcr.io/shopify/toxiproxy:2.9.0"
	}

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"8474/tcp"},
		WaitingFor:   wait.ForListeningPort("8474/tcp").WithStartupTimeout(30 * time.Second),
		ExtraHosts:   []string{"host.docker.internal:host-gateway"},
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, err
	}
	port, err := c.MappedPort(ctx, "8474/tcp")
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, err
	}

	api := fmt.Sprintf("http://%s:%s", host, port.Port())

	return &toxiproxyChaos{
		container: c,
		client:    toxiclient.NewClient(api),
		proxies:   make(map[string]*toxiclient.Proxy),
	}, nil
}

func (t *toxiproxyChaos) CreateProxy(ctx context.Context, name string, upstream string) (ProxyRef, error) {
	p, err := t.client.CreateProxy(name, "0.0.0.0:0", upstream)
	if err != nil {
		return ProxyRef{}, err
	}
	t.proxies[name] = p
	return ProxyRef{Name: name, ListenAddr: p.Listen, Upstream: upstream}, nil
}

func (t *toxiproxyChaos) AddLatency(ctx context.Context, proxyName string, latency, jitter time.Duration) error {
	p := t.proxies[proxyName]
	if p == nil {
		return fmt.Errorf("proxy not found: %s", proxyName)
	}

	_, err := p.AddToxic(
		"latency",
		"latency",
		"downstream",
		1.0,
		toxiclient.Attributes{
			"latency": int(latency / time.Millisecond),
			"jitter":  int(jitter / time.Millisecond),
		},
	)
	return err
}

func (t *toxiproxyChaos) CutConnection(ctx context.Context, proxyName string) error {
	p := t.proxies[proxyName]
	if p == nil {
		return fmt.Errorf("proxy not found: %s", proxyName)
	}

	_, err := p.AddToxic(
		"cut",
		"timeout",
		"downstream",
		1.0,
		toxiclient.Attributes{
			"timeout": 1,
		},
	)
	return err
}

func (t *toxiproxyChaos) AddTimeout(ctx context.Context, proxyName string, timeout time.Duration) error {
	p := t.proxies[proxyName]
	if p == nil {
		return fmt.Errorf("proxy not found: %s", proxyName)
	}

	_, err := p.AddToxic(
		"timeout",
		"timeout",
		"downstream",
		1.0,
		toxiclient.Attributes{
			"timeout": int(timeout / time.Millisecond),
		},
	)
	return err
}

func (t *toxiproxyChaos) AddBandwidth(ctx context.Context, proxyName string, rateKBps int64) error {
	p := t.proxies[proxyName]
	if p == nil {
		return fmt.Errorf("proxy not found: %s", proxyName)
	}

	_, err := p.AddToxic(
		"bandwidth",
		"bandwidth",
		"downstream",
		1.0,
		toxiclient.Attributes{
			"rate": rateKBps,
		},
	)
	return err
}

func (t *toxiproxyChaos) RemoveToxic(ctx context.Context, proxyName, toxicName string) error {
	p := t.proxies[proxyName]
	if p == nil {
		return fmt.Errorf("proxy not found: %s", proxyName)
	}
	return p.RemoveToxic(toxicName)
}

func (t *toxiproxyChaos) RemoveAllToxics(ctx context.Context, proxyName string) error {
	p := t.proxies[proxyName]
	if p == nil {
		return fmt.Errorf("proxy not found: %s", proxyName)
	}

	toxicsAny, err := p.Toxics()
	if err != nil {
		return err
	}

	switch toxics := any(toxicsAny).(type) {

	case []toxiclient.Toxic:
		for _, toxic := range toxics {
			if toxic.Name == "" {
				continue
			}
			if err := p.RemoveToxic(toxic.Name); err != nil {
				return err
			}
		}
		return nil

	case []*toxiclient.Toxic:
		for _, toxic := range toxics {
			if toxic == nil || toxic.Name == "" {
				continue
			}
			if err := p.RemoveToxic(toxic.Name); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unexpected toxics type from toxiproxy client: %T", toxicsAny)
	}
}

func (t *toxiproxyChaos) Close(ctx context.Context) error {
	for _, p := range t.proxies {
		_ = p.Delete()
	}
	if t.container != nil {
		return t.container.Terminate(ctx)
	}
	return nil
}
