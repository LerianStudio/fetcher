package itestkit

import (
	"context"
	"testing"
)

type Suite struct {
	t     *testing.T
	infra []Infra
	chaos ChaosInterface
	env   *Env
}

type Env struct {
	Containers map[string]ContainerEndpoint
	Chaos      ChaosInterface
}

type Builder struct {
	t         *testing.T
	infra     []Infra
	chaosConf ChaosConfig
}

func New(t *testing.T) *Builder {
	if t != nil {
		t.Helper()
	}
	return &Builder{
		t:     t,
		infra: make([]Infra, 0, 4),
		chaosConf: ChaosConfig{
			Enabled: false,
		},
	}
}

func (b *Builder) WithInfra(infra Infra) *Builder {
	if infra != nil {
		b.infra = append(b.infra, infra)
	}
	return b
}

func (b *Builder) WithInfras(infras ...Infra) *Builder {
	for _, infra := range infras {
		if infra == nil {
			continue
		}
		b.infra = append(b.infra, infra)
	}
	return b
}

func (b *Builder) WithChaos(cfg ChaosConfig) *Builder {
	b.chaosConf = cfg
	return b
}

func (b *Builder) Build(ctx context.Context) (*Suite, error) {
	if b.t != nil {
		b.t.Helper()
	}

	var chaos ChaosInterface
	if b.chaosConf.Enabled {
		tc, err := NewToxiproxyChaos(ctx, b.chaosConf)
		if err != nil {
			return nil, err
		}
		chaos = tc
	}

	s := &Suite{
		t:     b.t,
		infra: b.infra,
		chaos: chaos,
		env: &Env{
			Containers: map[string]ContainerEndpoint{},
			Chaos:      chaos,
		},
	}

	if err := validateUniqueInfraNames(s.infra); err != nil {
		_ = s.Terminate(ctx)
		return nil, err
	}

	for _, inf := range s.infra {
		if err := inf.Start(ctx, s.env); err != nil {
			_ = s.Terminate(ctx)
			return nil, err
		}
	}

	return s, nil
}

func (s *Suite) Env() *Env { return s.env }

func (s *Suite) Chaos() ChaosInterface { return s.chaos }

func (s *Suite) Terminate(ctx context.Context) error {
	for i := len(s.infra) - 1; i >= 0; i-- {
		_ = s.infra[i].Terminate(ctx)
	}
	if s.chaos != nil {
		_ = s.chaos.Close(ctx)
	}
	return nil
}
