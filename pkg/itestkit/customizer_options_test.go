package itestkit

import (
	"reflect"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

func applyCustomizer(t *testing.T, req *testcontainers.GenericContainerRequest, c Customizer) {
	t.Helper()

	if c == nil {
		return
	}

	if err := c.Customize(req); err != nil {
		t.Fatalf("customize request: %v", err)
	}
}

func TestCustomizerOptions_MutateContainerRequest(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "basic request customizers mutate expected fields",
			run: func(t *testing.T) {
				t.Parallel()

				req := testcontainers.GenericContainerRequest{}
				envs := map[string]string{"APP_MODE": "test"}
				customizer := CEnvs(envs)
				envs["APP_MODE"] = "mutated"

				applyCustomizer(t, &req, CImage("alpine:3.20"))
				applyCustomizer(t, &req, customizer)
				applyCustomizer(t, &req, CCmd("sh", "-c", "echo ok"))
				applyCustomizer(t, &req, CLabel("team", "platform"))
				applyCustomizer(t, &req, CName("itest-container"))

				if req.Image != "alpine:3.20" {
					t.Fatalf("expected image to be customized, got %q", req.Image)
				}

				if got := req.Env["APP_MODE"]; got != "test" {
					t.Fatalf("expected copied env value, got %q", got)
				}

				if !reflect.DeepEqual(req.Cmd, []string{"sh", "-c", "echo ok"}) {
					t.Fatalf("unexpected command: %#v", req.Cmd)
				}

				if got := req.Labels["team"]; got != "platform" {
					t.Fatalf("expected label to be set, got %q", got)
				}

				if req.Name != "itest-container" {
					t.Fatalf("expected name to be set, got %q", req.Name)
				}
			},
		},
		{
			name: "port network host and bind customizers deduplicate entries",
			run: func(t *testing.T) {
				t.Parallel()

				req := testcontainers.GenericContainerRequest{}

				applyCustomizer(t, &req, CExposedPorts("8080/tcp", "8080/tcp", "", "9090/tcp"))
				applyCustomizer(t, &req, CNetworks("shared", "shared", "secondary"))
				applyCustomizer(t, &req, CHostDockerInternal())
				applyCustomizer(t, &req, CHostDockerInternal())
				applyCustomizer(t, &req, CBindMount("/tmp/host", "/tmp/container", "ro"))
				applyCustomizer(t, &req, CBindMount("/tmp/host", "/tmp/container", "ro"))

				if !reflect.DeepEqual(req.ExposedPorts, []string{"8080/tcp", "9090/tcp"}) {
					t.Fatalf("unexpected exposed ports: %#v", req.ExposedPorts)
				}

				if !reflect.DeepEqual(req.Networks, []string{"shared", "secondary"}) {
					t.Fatalf("unexpected networks: %#v", req.Networks)
				}

				if !reflect.DeepEqual(req.ExtraHosts, []string{"host.docker.internal:host-gateway"}) {
					t.Fatalf("unexpected extra hosts: %#v", req.ExtraHosts)
				}

				if !reflect.DeepEqual(req.Binds, []string{"/tmp/host:/tmp/container:ro"}) {
					t.Fatalf("unexpected bind mounts: %#v", req.Binds)
				}
			},
		},
		{
			name: "network aliases initialize and ignore empty inputs",
			run: func(t *testing.T) {
				t.Parallel()

				req := testcontainers.GenericContainerRequest{}
				applyCustomizer(t, &req, CNetworkAliases("shared", "mongo", "mongo", "redis"))
				applyCustomizer(t, &req, CNetworkAliases("", "ignored"))
				applyCustomizer(t, &req, CNetworkAliases("shared"))

				got := req.NetworkAliases["shared"]
				if !reflect.DeepEqual(got, []string{"mongo", "redis"}) {
					t.Fatalf("unexpected aliases: %#v", got)
				}

				if len(req.NetworkAliases) != 1 {
					t.Fatalf("expected only one network alias entry, got %#v", req.NetworkAliases)
				}
			},
		},
		{
			name: "file copy helpers preserve destination metadata",
			run: func(t *testing.T) {
				t.Parallel()

				req := testcontainers.GenericContainerRequest{}
				applyCustomizer(t, &req, CCopyFile("/tmp/local.env", "/app/.env", 0o600))
				applyCustomizer(t, &req, CCopyDir("/tmp/scripts", "/docker-entrypoint-initdb.d", 0o755))
				applyCustomizer(t, &req, CInitScriptDirEntryPoint("/tmp/sql/01-init.sql", "/docker-entrypoint-initdb.d/", 0o644))

				if len(req.Files) != 3 {
					t.Fatalf("expected three copied entries, got %d", len(req.Files))
				}

				if req.Files[0].ContainerFilePath != "/app/.env" {
					t.Fatalf("unexpected file target: %q", req.Files[0].ContainerFilePath)
				}

				if req.Files[1].ContainerFilePath != "/docker-entrypoint-initdb.d" {
					t.Fatalf("unexpected dir target: %q", req.Files[1].ContainerFilePath)
				}

				if req.Files[2].ContainerFilePath != "/docker-entrypoint-initdb.d/01-init.sql" {
					t.Fatalf("unexpected init entrypoint target: %q", req.Files[2].ContainerFilePath)
				}
			},
		},
		{
			name: "env from os is no-op when unset and applied when present",
			run: func(t *testing.T) {
				t.Setenv("ITESTKIT_PRESENT", "configured")

				presentReq := testcontainers.GenericContainerRequest{}
				applyCustomizer(t, &presentReq, CEnvFromOS("ITESTKIT_PRESENT"))
				if got := presentReq.Env["ITESTKIT_PRESENT"]; got != "configured" {
					t.Fatalf("expected env from OS to be propagated, got %q", got)
				}

				unsetReq := testcontainers.GenericContainerRequest{}
				applyCustomizer(t, &unsetReq, CEnvFromOS("ITESTKIT_ABSENT"))
				if len(unsetReq.Env) != 0 {
					t.Fatalf("expected unset env customizer to be a no-op, got %#v", unsetReq.Env)
				}
			},
		},
		{
			name: "all and merge helpers filter nil entries",
			run: func(t *testing.T) {
				t.Parallel()

				var nilCustomizer Customizer
				filtered := CAll(nilCustomizer, CName("one"), nilCustomizer, CLabel("k", "v"))
				if len(filtered) != 2 {
					t.Fatalf("expected nil customizers to be filtered, got %d entries", len(filtered))
				}

				merged := MergeCustomizers(nil, filtered[:1], nil, filtered[1:])
				if len(merged) != 2 {
					t.Fatalf("expected merged customizers to preserve non-empty lists, got %d entries", len(merged))
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
