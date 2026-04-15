package e2ekit

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

type stubRewriter struct{ suffix string }

func (s stubRewriter) Rewrite(env map[string]string) map[string]string {
	out := cloneMap(env)
	for k, v := range out {
		out[k] = v + s.suffix
	}

	return out
}

func TestBuilderHelpersAndProjectRoot(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "builder fluent helpers mutate cheap branches deterministically",
			run: func(t *testing.T) {
				t.Parallel()

				cmd := []string{"serve", "--debug"}
				builder := New(t).
					WithImage("ghcr.io/example/app:1.0.0").
					WithDockerfile(BuildConfig{ContextDir: "/tmp/project", Dockerfile: "Dockerfile.test"}).
					WithImage("ghcr.io/example/app:2.0.0").
					WithEnv(map[string]string{"A": "one"}).
					WithEnvVar("B", "two").
					WithCmd(cmd...).
					ExposePort(8080).
					ExposePort(8080).
					WithNetworks("shared", "extra").
					WithExtraHosts("db.internal:host-gateway").
					WithWait(nil).
					WithWait(WaitPort(9090, 0)).
					WithRewriter(nil).
					WithRewriter(stubRewriter{suffix: "-rewritten"}).
					DisableDefaultLocalhostRewrite().
					WithLogsOnFailure(false).
					WithLogsOnFailureMaxBytes(0).
					WithLogsOnFailureMaxBytes(2048)

				cmd[0] = "mutated"

				if builder.build != nil {
					t.Fatalf("expected WithImage to clear dockerfile build config")
				}

				if builder.image != "ghcr.io/example/app:2.0.0" {
					t.Fatalf("unexpected image: %q", builder.image)
				}

				if !reflect.DeepEqual(builder.env, map[string]string{"A": "one", "B": "two"}) {
					t.Fatalf("unexpected env map: %#v", builder.env)
				}

				if !reflect.DeepEqual(builder.cmd, []string{"serve", "--debug"}) {
					t.Fatalf("expected command slice to be copied, got %#v", builder.cmd)
				}

				if !reflect.DeepEqual(builder.ports, []string{"8080/tcp"}) {
					t.Fatalf("unexpected exposed ports: %#v", builder.ports)
				}

				if !reflect.DeepEqual(builder.networks, []string{"shared", "extra"}) {
					t.Fatalf("unexpected networks: %#v", builder.networks)
				}

				if builder.logOnFail || builder.logOnFailMaxBytes != 2048 {
					t.Fatalf("unexpected failure log configuration: enabled=%v max=%d", builder.logOnFail, builder.logOnFailMaxBytes)
				}

				if len(builder.rewriters) != 1 {
					t.Fatalf("expected default localhost rewriter to be removed, got %d rewriters", len(builder.rewriters))
				}
			},
		},
		{
			name: "project root helpers find go.mod from directories and files",
			run: func(t *testing.T) {
				t.Parallel()

				root := t.TempDir()
				if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
					t.Fatalf("write go.mod: %v", err)
				}

				nestedDir := filepath.Join(root, "internal", "service")
				if err := os.MkdirAll(nestedDir, 0o755); err != nil {
					t.Fatalf("mkdir nested dir: %v", err)
				}

				filePath := filepath.Join(nestedDir, "service.go")
				if err := os.WriteFile(filePath, []byte("package service\n"), 0o644); err != nil {
					t.Fatalf("write nested file: %v", err)
				}

				if got := ProjectRootFrom(nestedDir); got != root {
					t.Fatalf("expected directory root %q, got %q", root, got)
				}

				if got := ProjectRootFrom(filePath); got != root {
					t.Fatalf("expected file root %q, got %q", root, got)
				}

				if got := ProjectRootFrom(t.TempDir()); got != "" {
					t.Fatalf("expected missing go.mod to return empty string, got %q", got)
				}

				if got := ProjectRoot(); got == "" {
					t.Fatalf("expected repository ProjectRoot to be discovered")
				}
			},
		},
		{
			name: "wait strategy helpers configure exposed ports and timeouts",
			run: func(t *testing.T) {
				t.Parallel()

				httpReq := testcontainers.ContainerRequest{}
				WaitHTTP(8080, "", 0).Configure(&httpReq)
				if !reflect.DeepEqual(httpReq.ExposedPorts, []string{"8080/tcp"}) {
					t.Fatalf("unexpected http wait ports: %#v", httpReq.ExposedPorts)
				}

				portReq := testcontainers.ContainerRequest{}
				WaitPort(9090, 0).Configure(&portReq)
				if !reflect.DeepEqual(portReq.ExposedPorts, []string{"9090/tcp"}) {
					t.Fatalf("unexpected port wait ports: %#v", portReq.ExposedPorts)
				}

				logReq := testcontainers.ContainerRequest{}
				WaitLog("boot complete", time.Second).Configure(&logReq)
				if logReq.WaitingFor == nil {
					t.Fatalf("expected log wait strategy to populate WaitingFor")
				}

				runningReq := testcontainers.ContainerRequest{WaitingFor: logReq.WaitingFor}
				WaitRunning(0).Configure(&runningReq)
				if runningReq.WaitingFor != nil {
					t.Fatalf("expected running wait to clear WaitingFor")
				}
			},
		},
		{
			name: "localhost rewriter handles urls hosts and dsn fragments",
			run: func(t *testing.T) {
				t.Setenv("TESTCONTAINERS_HOST_OVERRIDE", "10.20.30.40")

				tests := []struct {
					input string
					want  string
				}{
					{input: "http://localhost:8080/health", want: "http://10.20.30.40:8080/health"},
					{input: "https://127.0.0.1", want: "https://10.20.30.40"},
					{input: "localhost", want: "10.20.30.40"},
					{input: "127.0.0.1:6379", want: "10.20.30.40:6379"},
					{input: "mongodb://user:pass@localhost:27017/db", want: "mongodb://user:pass@10.20.30.40:27017/db"},
					{input: "host=localhost user=test", want: "host=10.20.30.40 user=test"},
					{input: "postgres://db.internal:5432/app", want: "postgres://db.internal:5432/app"},
				}

				for _, tt := range tests {
					tt := tt
					t.Run(tt.input, func(t *testing.T) {
						got := rewriteLocalhostForContainer(tt.input)
						if got != tt.want {
							t.Fatalf("rewrite mismatch: want %q, got %q", tt.want, got)
						}
					})
				}
			},
		},
		{
			name: "cheap helpers keep maps and slices independent",
			run: func(t *testing.T) {
				t.Parallel()

				list := uniqueAppend([]string{"8080/tcp"}, "8080/tcp")
				list = uniqueAppend(list, "9090/tcp")
				if !reflect.DeepEqual(list, []string{"8080/tcp", "9090/tcp"}) {
					t.Fatalf("unexpected unique append result: %#v", list)
				}

				env := map[string]string{"HOST": "localhost"}
				cloned := cloneMap(env)
				cloned["HOST"] = "mutated"
				if env["HOST"] != "localhost" {
					t.Fatalf("expected cloneMap to isolate source map")
				}

				rewriter := RewriteLocalhostToHostGateway()
				rewritten := rewriter.Rewrite(map[string]string{"URL": "db.internal:5432"})
				if rewritten["URL"] != "db.internal:5432" {
					t.Fatalf("expected non-localhost values to stay unchanged, got %q", rewritten["URL"])
				}
			},
		},
		{
			name: "run returns validation error before container bootstrap when config is incomplete",
			run: func(t *testing.T) {
				t.Parallel()

				_, err := New(t).Run()
				if err == nil || !strings.Contains(err.Error(), "missing image or dockerfile build config") {
					t.Fatalf("expected missing image/build error, got %v", err)
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
