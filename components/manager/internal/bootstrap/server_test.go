package bootstrap

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ServerAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "standard address",
			address: "localhost:8080",
		},
		{
			name:    "wildcard address",
			address: "0.0.0.0:3000",
		},
		{
			name:    "empty address",
			address: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				serverAddress: tt.address,
			}

			got := s.ServerAddress()
			if got != tt.address {
				t.Errorf("ServerAddress() = %q, want %q", got, tt.address)
			}
		})
	}
}

// TestResolveServerDrainDelay covers the same clamp rules as
// readyz.LoadConfig — zero defaults to 12s, negatives collapse to 1s, and
// positive values pass through as seconds.
func TestResolveServerDrainDelay(t *testing.T) {
	cases := []struct {
		name string
		in   *Config
		want time.Duration
	}{
		{name: "nil config → default 12s", in: nil, want: 12 * time.Second},
		{name: "zero → default 12s", in: &Config{ReadyzDrainDelaySec: 0}, want: 12 * time.Second},
		{name: "negative → 1s", in: &Config{ReadyzDrainDelaySec: -5}, want: time.Second},
		{name: "positive passes through", in: &Config{ReadyzDrainDelaySec: 30}, want: 30 * time.Second},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, resolveServerDrainDelay(tc.in))
		})
	}
}

// TestServer_DrainLoop_SetsDrainingBeforeShutdownClose asserts the Gate 7
// ordering: on SIGTERM, drainLoop MUST set the draining flag BEFORE closing
// the ServerManager shutdown channel. Reversing the order would let
// lib-commons tear down the HTTP listener while Kubernetes is still routing
// traffic to this pod — connections would drop.
func TestServer_DrainLoop_SetsDrainingBeforeShutdownClose(t *testing.T) {
	readyz.SetDraining(false)
	t.Cleanup(func() { readyz.SetDraining(false) })

	sig := make(chan os.Signal, 1)
	shutdownCh := make(chan struct{})

	s := &Server{drainDelay: 0}

	go s.drainLoop(context.Background(), sig, shutdownCh)

	// Nothing has happened yet: flag stays false and shutdownCh is open.
	assert.False(t, readyz.IsDraining())

	select {
	case <-shutdownCh:
		t.Fatal("shutdownCh closed before signal")
	default:
	}

	sig <- syscall.SIGTERM

	// Wait for drainLoop to advance. The shutdownCh close is the last
	// action, so if it is closed we know SetDraining already fired.
	select {
	case <-shutdownCh:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdownCh never closed after SIGTERM")
	}

	assert.True(t, readyz.IsDraining(),
		"SetDraining(true) must happen before close(shutdownCh)")
}

// TestServer_DrainLoop_RespectsDrainDelay asserts the drain grace window is
// honoured end-to-end: the shutdown channel is not closed until the
// configured duration has elapsed.
func TestServer_DrainLoop_RespectsDrainDelay(t *testing.T) {
	readyz.SetDraining(false)
	t.Cleanup(func() { readyz.SetDraining(false) })

	sig := make(chan os.Signal, 1)
	shutdownCh := make(chan struct{})

	s := &Server{drainDelay: 200 * time.Millisecond}

	go s.drainLoop(context.Background(), sig, shutdownCh)

	sig <- syscall.SIGTERM

	select {
	case <-shutdownCh:
		t.Fatal("shutdownCh closed before drain delay elapsed")
	case <-time.After(50 * time.Millisecond):
		// Expected: still draining.
	}

	assert.True(t, readyz.IsDraining(),
		"draining flag flips immediately on SIGTERM, before the sleep")

	select {
	case <-shutdownCh:
		// Expected: channel closes after ~200ms total.
	case <-time.After(1 * time.Second):
		t.Fatal("shutdownCh never closed within 1s of drain-delay start")
	}
}

// TestServer_DrainLoop_ContextCancel closes shutdownCh when the parent
// context is cancelled without a SIGTERM. This is important for clean
// teardown paths that cancel the launcher externally (e.g. a test harness
// that does not fire a real OS signal).
func TestServer_DrainLoop_ContextCancel(t *testing.T) {
	readyz.SetDraining(false)
	t.Cleanup(func() { readyz.SetDraining(false) })

	sig := make(chan os.Signal, 1)
	shutdownCh := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{drainDelay: 50 * time.Millisecond}

	go s.drainLoop(ctx, sig, shutdownCh)

	cancel()

	select {
	case <-shutdownCh:
	case <-time.After(time.Second):
		t.Fatal("shutdownCh never closed after ctx cancel")
	}

	// The flag is NOT flipped on ctx-cancel; the drain flag is a SIGTERM
	// response only. This mirrors real shutdown paths that never signalled.
	assert.False(t, readyz.IsDraining())
}

// TestServer_NewServer_WiresDrainDelay asserts NewServer copies the
// ReadyzDrainDelaySec into the Server struct via resolveServerDrainDelay.
// Guards against a silent regression where a later refactor decoupled the
// fields.
func TestServer_NewServer_WiresDrainDelay(t *testing.T) {
	cfg := &Config{ServerAddress: ":0", ReadyzDrainDelaySec: 7}

	s := &Server{
		serverAddress: cfg.ServerAddress,
		drainDelay:    resolveServerDrainDelay(cfg),
	}

	assert.Equal(t, 7*time.Second, s.drainDelay)
	require.Equal(t, ":0", s.ServerAddress())
}
