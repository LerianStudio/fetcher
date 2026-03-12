package itestkit

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeInfra struct {
	name  string
	order *[]string
	err   error
}

func (f *fakeInfra) Start(context.Context, *Env) error { return nil }

func (f *fakeInfra) Terminate(context.Context) error {
	if f.order != nil {
		*f.order = append(*f.order, f.name)
	}

	return f.err
}

type fakeNamedInfra struct {
	kind string
	name string
}

func (f fakeNamedInfra) Start(context.Context, *Env) error { return nil }
func (f fakeNamedInfra) Terminate(context.Context) error   { return nil }
func (f fakeNamedInfra) InfraKind() string                 { return f.kind }
func (f fakeNamedInfra) InfraName() string                 { return f.name }

type fakeChaos struct {
	closed bool
	err    error
}

func (f *fakeChaos) CreateProxy(context.Context, string, string) (ProxyRef, error) {
	return ProxyRef{}, nil
}
func (f *fakeChaos) AddLatency(context.Context, string, time.Duration, time.Duration) error {
	return nil
}
func (f *fakeChaos) AddTimeout(context.Context, string, time.Duration) error { return nil }
func (f *fakeChaos) AddBandwidth(context.Context, string, int64) error       { return nil }
func (f *fakeChaos) CutConnection(context.Context, string) error             { return nil }
func (f *fakeChaos) RemoveToxic(context.Context, string, string) error       { return nil }
func (f *fakeChaos) RemoveAllToxics(context.Context, string) error           { return nil }
func (f *fakeChaos) Close(context.Context) error {
	f.closed = true
	return f.err
}

func TestBuilderConfigurationAndSuiteLifecycle(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "new builder uses stable defaults even with nil testing instance",
			run: func(t *testing.T) {
				t.Parallel()

				builder := New(nil)
				if builder.t != nil {
					t.Fatalf("expected nil testing instance to be preserved")
				}

				if len(builder.infra) != 0 {
					t.Fatalf("expected builder infra to start empty, got %d entries", len(builder.infra))
				}

				if builder.chaosConf.Enabled {
					t.Fatalf("expected chaos to be disabled by default")
				}
			},
		},
		{
			name: "with infra helpers keep insertion order and ignore nil values",
			run: func(t *testing.T) {
				t.Parallel()

				first := &fakeInfra{name: "first"}
				second := &fakeInfra{name: "second"}
				third := &fakeInfra{name: "third"}

				builder := New(t).
					WithInfra(nil).
					WithInfra(first).
					WithInfras(nil, second, third, nil)

				if got := len(builder.infra); got != 3 {
					t.Fatalf("expected three infras, got %d", got)
				}

				if !reflect.DeepEqual(builder.infra, []Infra{first, second, third}) {
					t.Fatalf("unexpected infra order: %#v", builder.infra)
				}
			},
		},
		{
			name: "with chaos stores configuration and accessors expose suite state",
			run: func(t *testing.T) {
				t.Parallel()

				chaos := &fakeChaos{}
				env := &Env{Containers: map[string]ContainerEndpoint{}, Chaos: chaos, Network: "shared-net"}
				suite := &Suite{env: env, chaos: chaos}

				builder := New(t).WithChaos(ChaosConfig{Enabled: true, Image: "toxiproxy:test"})
				if !builder.chaosConf.Enabled || builder.chaosConf.Image != "toxiproxy:test" {
					t.Fatalf("unexpected chaos config: %#v", builder.chaosConf)
				}

				if suite.Env() != env {
					t.Fatalf("expected Env accessor to return suite env")
				}

				gotChaos, ok := suite.Chaos().(*fakeChaos)
				if !ok || gotChaos != chaos {
					t.Fatalf("expected Chaos accessor to return suite chaos")
				}

				if got := suite.Network(); got != "shared-net" {
					t.Fatalf("expected network from env, got %q", got)
				}

				if got := (&Suite{}).Network(); got != "" {
					t.Fatalf("expected empty network when env is nil, got %q", got)
				}
			},
		},
		{
			name: "terminate tears down infra in reverse order and ignores cleanup errors",
			run: func(t *testing.T) {
				t.Parallel()

				order := make([]string, 0, 3)
				chaos := &fakeChaos{err: errors.New("close failure")}
				suite := &Suite{
					infra: []Infra{
						&fakeInfra{name: "first", order: &order},
						&fakeInfra{name: "second", order: &order, err: errors.New("ignored")},
						&fakeInfra{name: "third", order: &order},
					},
					chaos: chaos,
				}

				if err := suite.Terminate(context.Background()); err != nil {
					t.Fatalf("expected terminate to swallow cleanup errors, got %v", err)
				}

				if !reflect.DeepEqual(order, []string{"third", "second", "first"}) {
					t.Fatalf("expected reverse termination order, got %#v", order)
				}

				if !chaos.closed {
					t.Fatalf("expected chaos cleaner to be closed")
				}
			},
		},
		{
			name: "unique infra names reject duplicate kind and normalized name",
			run: func(t *testing.T) {
				t.Parallel()

				duplicate := validateUniqueInfraNames([]Infra{
					fakeNamedInfra{kind: "postgres", name: ""},
					fakeNamedInfra{kind: "postgres", name: "default"},
				})
				if duplicate == nil {
					t.Fatalf("expected duplicate default name to fail")
				}

				if err := validateUniqueInfraNames([]Infra{
					fakeNamedInfra{kind: "postgres", name: "primary"},
					fakeNamedInfra{kind: "postgres", name: "replica"},
					&fakeInfra{name: "unnamed-non-named-interface"},
				}); err != nil {
					t.Fatalf("expected distinct names to pass, got %v", err)
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
