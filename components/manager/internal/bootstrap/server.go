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

// serverNotifySignals is the signal.Notify indirection used by Server.Run so
// tests can inject a synthetic signal channel without racing against
// lib-commons' own signal handler. In production it points at signal.Notify.
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
	// drainDelay controls how long Run blocks after SIGTERM before allowing
	// the lib-commons ServerManager to tear down the HTTP listener. Copied
	// from Config.ReadyzDrainDelaySec so the Server does not need a back-ref
	// to the whole config struct.
	drainDelay time.Duration
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

// resolveServerDrainDelay converts Config.ReadyzDrainDelaySec into a
// time.Duration, applying the same clamps as readyz.LoadConfig: zero falls
// back to 12s, negative values to 1s. Kept local so the server package does
// not import readyz for a single helper.
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

// Run runs the server.
//
// Gate 7 of ring:dev-readyz — graceful drain:
//
// lib-commons' ServerManager.WithShutdownHook invokes hooks AFTER the HTTP
// listener is closed (see lib-commons v4.6.0 commons/server/shutdown.go,
// executeShutdown, line ~397). That ordering is too late for our drain
// contract — we need readyz.SetDraining(true) to fire BEFORE connections
// are drained so Kubernetes' readinessProbe flips to 503 and the kube-proxy
// stops routing new traffic to this pod. Otherwise in-flight requests would
// be dropped as the listener closes.
//
// Solution: intercept SIGTERM ourselves via signal.Notify, flip the drain
// flag, sleep for the grace period, and THEN close the ServerManager's
// injected shutdown channel via WithShutdownChannel. ServerManager blocks on
// that channel instead of installing its own signal handler, so the drain
// sequence is strictly: SetDraining → grace → HTTP shutdown → hooks.
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

	// Stop listening for OS signals; tears down the goroutine above if it
	// has not already returned.
	signal.Stop(sig)
	drainCancel()

	return nil
}

// drainLoop is the sidecar shutdown coordinator invoked from Run. It waits
// for SIGTERM/SIGINT, flips the readyz drain flag, sleeps for the grace
// window and finally closes shutdownCh to unblock the lib-commons
// ServerManager. Extracted as a method so tests can drive it deterministically
// without the full Run wiring.
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
