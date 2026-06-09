package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/bootstrap/readyz"
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libCommonsServer "github.com/LerianStudio/lib-commons/v5/commons/server"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	"github.com/gofiber/fiber/v2"
)

const defaultHealthPort = 4007

// defaultReadyzDrainDelay matches the default kube-proxy sync window.
const defaultReadyzDrainDelay = 12 * time.Second

// defaultDrain mirrors readyz.LoadConfig's clamps so worker and manager
// agree: 0 → 12s, negative → 1s.
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

// HealthServer is a worker-side micro-HTTP server that exposes /health,
// /readyz, /readyz/tenant/:id and /metrics. The worker is a RabbitMQ
// consumer with no primary HTTP server, but Kubernetes still needs a
// readiness endpoint to schedule tenant events and reap dead pods.
// It runs under the same Launcher lifecycle as the consumer, so a single
// SIGTERM tears both down via the shared drain flag.
//
// Deliberately not mounted here: lib-streaming's manifest handler. This server
// is unauthenticated kube-health surface, not a public/admin API; exposing the
// manifest here would leak event topology through a probe port. The follow-up is
// to add an authenticated worker admin surface and mount streaming.NewStreamingHandler
// there, not to bolt it onto /health by stealth. Subtle difference, large blast radius.
type HealthServer struct {
	app       *fiber.App
	addr      string
	logger    libLog.Logger
	telemetry *libOtel.Telemetry
}

// NewHealthServer mounts /health, /readyz, /readyz/tenant/:id and /metrics.
// /readyz/tenant/:id falls back to a 400 disabled handler when MT is off or
// any MT prerequisite is missing. deps may be nil — the server still works
// with an empty checker set, so a misconfigured bootstrap is not fatal.
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

	app.Get("/health", readyz.HealthHandler())
	app.Get("/readyz", handler.Fiber())
	app.Get("/readyz/tenant/:id", buildWorkerTenantHandler(readyzCfg, deps))
	app.Get("/metrics", readyz.NewMetricsHandler())

	// readyzCfg.HealthPort is already normalised so misconfigured values
	// (0, negative, >65535) collapse to the safe default rather than
	// producing an invalid listen address like ":0".
	return &HealthServer{
		app:       app,
		addr:      fmt.Sprintf(":%d", readyzCfg.HealthPort),
		logger:    logger,
		telemetry: telemetry,
	}
}

func (s *HealthServer) App() *fiber.App { return s.app }

func (s *HealthServer) Address() string { return s.addr }

// Run accepts *libCommons.Launcher for lifecycle parity with the worker
// Service but delegates listener ownership to ServerManager.
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

// newWorkerReadyzConfig forwards the worker's Config into readyz.Config,
// keeping /readyz bootable on misconfigured input so operators see the
// error in the response body rather than a crashed pod. nil cfg falls
// through to readyz.LoadConfig.
func newWorkerReadyzConfig(cfg *Config) *readyz.Config {
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
