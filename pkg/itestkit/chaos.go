package itestkit

import (
	"context"
	"time"
)

type ChaosConfig struct {
	Enabled bool
	Image   string // opcional
}

type ProxyRef struct {
	Name                string
	ListenAddr          string // host-usable host:port for test code running outside Docker
	InNetworkListenAddr string // optional shared-network host:port for app containers inside Docker
	Upstream            string // host:port real
}

type ChaosInterface interface {
	CreateProxy(ctx context.Context, name string, upstream string) (ProxyRef, error)
	AddLatency(ctx context.Context, proxyName string, latency, jitter time.Duration) error
	AddTimeout(ctx context.Context, proxyName string, timeout time.Duration) error
	AddBandwidth(ctx context.Context, proxyName string, rateKBps int64) error
	CutConnection(ctx context.Context, proxyName string) error
	RemoveToxic(ctx context.Context, proxyName, toxicName string) error
	RemoveAllToxics(ctx context.Context, proxyName string) error
	Close(ctx context.Context) error
}
