package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libCommonsLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libCommonsOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libCommonsServer "github.com/LerianStudio/lib-commons/v4/commons/server"
	libLicense "github.com/LerianStudio/lib-license-go/v2/middleware"
	"github.com/gofiber/fiber/v2"
)

// serverNotifySignals is overridable so tests can drive a synthetic signal
// channel without racing the real signal handler.
var serverNotifySignals = signal.Notify

type licenseTerminator interface {
	Terminate(msg string)
}

// Server represents the http server for Ledger services.
type Server struct {
	app           *fiber.App
	serverAddress string
	license       licenseTerminator
	logger        libCommonsLog.Logger
	telemetry     libCommonsOtel.Telemetry
	shutdownHooks []func(context.Context) error
	drainDelay    time.Duration
}

// ServerAddress returns is a convenience method to return the server address.
func (s *Server) ServerAddress() string {
	return s.serverAddress
}

// NewServer creates an instance of Server.
// Optional shutdownHooks are registered with the server manager and executed during graceful shutdown.
func NewServer(cfg *Config, app *fiber.App, logger libCommonsLog.Logger, telemetry *libCommonsOtel.Telemetry, licenseClient *libLicense.LicenseClient, shutdownHooks ...func(context.Context) error) *Server {
	return &Server{
		app:           app,
		serverAddress: cfg.ServerAddress,
		license:       licenseClient.GetLicenseManagerShutdown(),
		logger:        logger,
		telemetry:     *telemetry,
		shutdownHooks: shutdownHooks,
		drainDelay:    resolveServerDrainDelay(cfg),
	}
}

// resolveServerDrainDelay mirrors readyz.LoadConfig's clamps: zero → 12s,
// negative → 1s. Defined locally so the package doesn't import readyz for
// a single helper.
func resolveServerDrainDelay(cfg *Config) time.Duration {
	switch {
	case cfg == nil || cfg.ReadyzDrainDelaySec == 0:
		return 12 * time.Second
	case cfg.ReadyzDrainDelaySec < 0:
		return time.Second
	default:
		return time.Duration(cfg.ReadyzDrainDelaySec) * time.Second
	}
}

// Run drives the graceful drain sequence: SetDraining → grace period →
// HTTP shutdown → hooks. lib-commons' ServerManager runs hooks after the
// HTTP listener closes, which is too late for the drain flag — kube-proxy
// must see /readyz=503 before connections are drained, otherwise in-flight
// requests are dropped. We intercept SIGTERM ourselves, flip the flag,
// sleep, and only then close ServerManager's injected shutdown channel.
func (s *Server) Run(l *libCommons.Launcher) error {
	_ = l

	shutdownCh := make(chan struct{})

	sig := make(chan os.Signal, 1)
	serverNotifySignals(sig, os.Interrupt, syscall.SIGTERM)

	drainCtx, drainCancel := context.WithCancel(context.Background())
	defer drainCancel()

	go s.drainLoop(drainCtx, sig, shutdownCh)

	manager := libCommonsServer.NewServerManager(nil, &s.telemetry, s.logger).
		WithHTTPServer(s.app, s.serverAddress).
		WithShutdownChannel(shutdownCh)

	if s.license != nil {
		manager = manager.WithShutdownHook(func(context.Context) error {
			s.license.Terminate("shutdown")

			return nil
		})
	}

	for _, hook := range s.shutdownHooks {
		manager = manager.WithShutdownHook(hook)
	}

	manager.StartWithGracefulShutdown()

	signal.Stop(sig)
	drainCancel()

	return nil
}

// drainLoop waits for SIGTERM/SIGINT, flips the readyz drain flag, sleeps
// the grace window, then closes shutdownCh to unblock ServerManager.
// Extracted so tests can drive it without the full Run wiring.
func (s *Server) drainLoop(ctx context.Context, sig <-chan os.Signal, shutdownCh chan<- struct{}) {
	select {
	case <-sig:
	case <-ctx.Done():
		close(shutdownCh)
		return
	}

	readyz.SetDraining(true)

	if s.logger != nil {
		s.logger.Log(ctx, libCommonsLog.LevelInfo,
			"SIGTERM received; readyz draining flag set, sleeping drain grace period")
	}

	if s.drainDelay > 0 {
		select {
		case <-time.After(s.drainDelay):
		case <-ctx.Done():
		}
	}

	close(shutdownCh)
}
