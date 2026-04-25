package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/LerianStudio/fetcher/pkg/bootstrap/readyz"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libCommonsServer "github.com/LerianStudio/lib-commons/v4/commons/server"
	"github.com/gofiber/fiber/v2"
)

// defaultHealthPort matches readyz.defaultHealthPort / resolveHealthPort.
const defaultHealthPort = 4007

// defaultReadyzDrainDelay is the fallback drain window when
// READYZ_DRAIN_DELAY_SEC is unset or non-positive. 12s mirrors the default
// kube-proxy sync window — see readyz.defaultDrainDelay for the rationale.
const defaultReadyzDrainDelay = 12 * time.Second

// defaultDrain applies the clamp rules for READYZ_DRAIN_DELAY_SEC. Zero or
// negative values collapse to 1s, empty values to 12s — matching the
// readyz.LoadConfig behaviour so the manager and worker agree.
func defaultDrain(sec int) time.Duration {
	switch {
	case sec < 0:
		return time.Second
	case sec == 0:
		return defaultReadyzDrainDelay
	default:
		return time.Duration(sec) * time.Second
	}
}

// HealthServer is the worker-side micro-HTTP-server that exposes /health,
// /readyz, /readyz/tenant/:id and /metrics. The worker itself has no primary
// HTTP server (it is a RabbitMQ consumer), but Kubernetes still requires a
// readiness endpoint to decide whether to route tenant events and to reap
// dead pods — this server fills that gap.
//
// The server is started by Service.Run via a second commons.RunApp, which
// means it participates in the same Launcher lifecycle as the consumer:
// a single SIGTERM tears both down, and the graceful-drain flag (set in
// Gate 7) flips /readyz to 503 before Kubernetes removes the pod from the
// Service endpoints.
type HealthServer struct {
	app       *fiber.App
	addr      string
	logger    libLog.Logger
	telemetry *libOtel.Telemetry
}

// NewHealthServer constructs the worker's /readyz micro-server. It mounts
//   - GET /health              → lib-commons Ping (Gate 7 will wrap with self-probe)
//   - GET /readyz              → the canonical /readyz handler
//   - GET /readyz/tenant/:id   → real per-tenant handler (Gate 6); disabled
//     variant returning 400 when MT is off or any MT prerequisite missing
//   - GET /metrics             → Prometheus exposition (Gate 5)
//
// Gate 6 of ring:dev-readyz wires the real checkers. deps may be nil —
// NewHealthServer still produces a working server (with an empty checker
// set) so misconfigured bootstraps are not fatal at startup.
func NewHealthServer(
	cfg *Config,
	logger libLog.Logger,
	telemetry *libOtel.Telemetry,
	deps *workerReadyzDeps,
) *HealthServer {
	readyzCfg := newWorkerReadyzConfig(cfg)
	checkers := buildWorkerReadyzCheckers(deps)
	handler := readyz.NewHandler(readyzCfg, checkers...)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Mount global endpoints first (these already come from readyz.Register),
	// then override /readyz/tenant/:id with the real Gate 6 handler.
	// Gate 7 of ring:dev-readyz: /health is gated on the startup self-probe
	// so K8s' livenessProbe restarts the pod when a dep was unreachable at
	// boot.
	app.Get("/health", readyz.HealthHandler())
	app.Get("/readyz", handler.Fiber())
	app.Get("/readyz/tenant/:id", buildWorkerTenantHandler(readyzCfg, deps))
	app.Get("/metrics", readyz.NewMetricsHandler())

	// Use the normalised port from readyzCfg so misconfigured values
	// (0, negative, >65535) collapse to the safe default rather than
	// producing an invalid listen address like ":0".
	return &HealthServer{
		app:       app,
		addr:      fmt.Sprintf(":%d", readyzCfg.HealthPort),
		logger:    logger,
		telemetry: telemetry,
	}
}

// App exposes the underlying fiber.App for testing and for the manager's
// lib-commons ServerManager wiring.
func (s *HealthServer) App() *fiber.App { return s.app }

// Address returns the listen address (e.g. ":4007").
func (s *HealthServer) Address() string { return s.addr }

// Run starts the server under a lib-commons Launcher. Matching the existing
// worker Service pattern, it accepts *libCommons.Launcher but does not use it
// directly — the ServerManager owns the listener lifecycle including
// graceful shutdown and telemetry flushing.
func (s *HealthServer) Run(_ *libCommons.Launcher) error {
	if s.logger != nil {
		s.logger.Log(context.Background(), libLog.LevelInfo,
			fmt.Sprintf("worker health server listening on %s", s.addr))
	}

	manager := libCommonsServer.NewServerManager(nil, s.telemetry, s.logger).
		WithHTTPServer(s.app, s.addr)

	manager.StartWithGracefulShutdown()

	return nil
}

// newWorkerReadyzConfig bridges the worker Config to the readyz package's
// self-contained Config struct. Missing / invalid values fall back to sane
// defaults — /readyz must remain bootable on a misconfigured worker so
// operators can see the error in the response body instead of a crashed pod.
func newWorkerReadyzConfig(cfg *Config) *readyz.Config {
	// Mirror readyz.NewHandler's nil-tolerance: a nil worker Config falls
	// through to readyz.LoadConfig which reads env directly. Prevents panic
	// under test seams or misconfigured callers.
	if cfg == nil {
		return readyz.LoadConfig()
	}

	mode := cfg.DeploymentMode
	if mode == "" {
		mode = readyz.DeploymentModeLocal
	}

	drain := defaultDrain(cfg.ReadyzDrainDelaySec)

	version := cfg.OtelServiceVersion
	if version == "" {
		version = "unknown"
	}

	port := cfg.HealthPort
	if port <= 0 || port > 65535 {
		port = defaultHealthPort
	}

	return &readyz.Config{
		DeploymentMode: mode,
		HealthPort:     port,
		DrainDelay:     drain,
		Version:        version,
	}
}
