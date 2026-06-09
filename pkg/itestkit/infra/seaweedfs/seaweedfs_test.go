package seaweedfs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/itestkit"
	"github.com/moby/moby/api/types/container"
	mobyNetwork "github.com/moby/moby/api/types/network"
)

func TestSeaweedFSHelpersWithoutDocker(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "new infra applies stable defaults",
			run: func(t *testing.T) {
				t.Parallel()

				infra := NewSeaweedFSInfra(SeaweedFSConfig{})
				if infra.cfg.Image != defaultImage {
					t.Fatalf("expected default image %q, got %q", defaultImage, infra.cfg.Image)
				}

				if infra.cfg.Name != "default" {
					t.Fatalf("expected default infra name, got %q", infra.cfg.Name)
				}

				if infra.cfg.StartupTimeout != defaultStartupTimeout {
					t.Fatalf("expected default startup timeout %v, got %v", defaultStartupTimeout, infra.cfg.StartupTimeout)
				}

				if infra.cfg.ProxyName != "seaweed-default" {
					t.Fatalf("expected default proxy name, got %q", infra.cfg.ProxyName)
				}
			},
		},
		{
			name: "endpoint and url require start state",
			run: func(t *testing.T) {
				t.Parallel()

				infra := NewSeaweedFSInfra(SeaweedFSConfig{Name: "files"})
				if _, err := infra.Endpoint(); err == nil || !strings.Contains(err.Error(), "endpoint not ready") {
					t.Fatalf("expected endpoint readiness error, got %v", err)
				}

				if _, err := infra.URL(); err == nil || !strings.Contains(err.Error(), "endpoint not ready") {
					t.Fatalf("expected url readiness error, got %v", err)
				}
			},
		},
		{
			name: "host port helpers reuse stored endpoint data",
			run: func(t *testing.T) {
				t.Parallel()

				infra := &SeaweedFSInfra{endpoint: &SeaweedFSEndpoint{
					URL:         "http://127.0.0.1:18888",
					Host:        "127.0.0.1",
					Port:        "18888",
					Upstream:    "127.0.0.1:18888",
					ProxyListen: "toxiproxy:10001",
				}}

				host, port, err := infra.HostPort()
				if err != nil {
					t.Fatalf("host port should parse: %v", err)
				}

				// HostPort normalizes loopback addresses for Docker connectivity
				expectedHost := itestkit.NormalizeHost("127.0.0.1")
				if host != expectedHost || port != 18888 {
					t.Fatalf("unexpected host port: %s %d", host, port)
				}

				// Verify proxy listen is stored
				ep, err := infra.Endpoint()
				if err != nil {
					t.Fatalf("endpoint should resolve: %v", err)
				}
				if ep.ProxyListen != "toxiproxy:10001" {
					t.Fatalf("unexpected proxy listen: %s", ep.ProxyListen)
				}
			},
		},
		{
			name: "host port surfaces invalid stored ports",
			run: func(t *testing.T) {
				t.Parallel()

				infra := &SeaweedFSInfra{endpoint: &SeaweedFSEndpoint{Host: "127.0.0.1", Port: "not-a-port"}}
				if _, _, err := infra.HostPort(); err == nil || !strings.Contains(err.Error(), "invalid port") {
					t.Fatalf("expected invalid port error, got %v", err)
				}
			},
		},
		{
			name: "terminate is nil safe when docker resources were never started",
			run: func(t *testing.T) {
				t.Parallel()

				infra := &SeaweedFSInfra{}
				if err := infra.Terminate(context.Background()); err != nil {
					t.Fatalf("expected nil-safe terminate, got %v", err)
				}
			},
		},
		{
			name: "fixed port option mutates host config bindings",
			run: func(t *testing.T) {
				t.Parallel()

				opts := defaultSeaweedFSOptions()
				WithSeaweedFSFixedPort("18888")(opts)

				if len(opts.hostConfigModifiers) != 1 {
					t.Fatalf("expected one host config modifier, got %d", len(opts.hostConfigModifiers))
				}

				hc := &container.HostConfig{}
				opts.hostConfigModifiers[0](hc)

				binding := hc.PortBindings[mobyNetwork.MustParsePort("8888/tcp")]
				if len(binding) != 1 || binding[0].HostPort != "18888" {
					t.Fatalf("unexpected port binding: %#v", binding)
				}
			},
		},
		{
			name: "custom config keeps explicit values",
			run: func(t *testing.T) {
				t.Parallel()

				infra := NewSeaweedFSInfra(SeaweedFSConfig{
					Name:           "tenant-a",
					Image:          "chrislusf/seaweedfs:3.70",
					StartupTimeout: 45 * time.Second,
					ProxyName:      "seaweed-a",
				})

				if infra.cfg.Name != "tenant-a" || infra.cfg.Image != "chrislusf/seaweedfs:3.70" {
					t.Fatalf("unexpected explicit config: %#v", infra.cfg)
				}

				if infra.cfg.StartupTimeout != 45*time.Second || infra.cfg.ProxyName != "seaweed-a" {
					t.Fatalf("unexpected explicit timeout/proxy: %#v", infra.cfg)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
