package bootstrap

import (
	"context"

	"github.com/LerianStudio/lib-commons/v5/commons"
	libOutbox "github.com/LerianStudio/lib-commons/v5/commons/outbox"
	libLog "github.com/LerianStudio/lib-observability/log"
)

// runLauncher launches the consumer and (when present) the health-port
// micro-server under a single Launcher so one SIGTERM tears both down.
// healthServer is nil under narrow unit tests that opt out of the server.
var runLauncher = func(logger libLog.Logger, consumer *MultiQueueConsumer, healthServer *HealthServer, outboxDispatcher *libOutbox.Dispatcher) {
	opts := []commons.LauncherOption{
		commons.WithLogger(logger),
		commons.RunApp("RabbitMQ Consumer", consumer),
	}

	if healthServer != nil {
		opts = append(opts, commons.RunApp("Health Server", healthServer))
	}

	if outboxDispatcher != nil {
		opts = append(opts, commons.RunApp("Streaming Outbox Dispatcher", outboxDispatcher))
	}

	commons.NewLauncher(opts...).Run()
}

type licenseTerminator interface {
	Terminate(msg string)
}

// Service is the application glue where we put all top level components to be used.
type Service struct {
	*MultiQueueConsumer
	libLog.
		Logger
	licenseShutdown licenseTerminator
	// mtCleanup is the cleanup function for multi-tenant resources (Redis, etc.)
	mtCleanup func()
	// healthServer exposes /health, /readyz and /metrics on HEALTH_PORT.
	// runLauncher skips it when nil.
	healthServer *HealthServer
	// readyzCloser releases resources owned exclusively by the readyz
	// wiring (e.g. a dedicated MT-Redis probe client). Nil-safe.
	readyzCloser func()
	// streamingCloser closes the lib-streaming producer after consumers stop.
	streamingCloser func() error
	// outboxDispatcher replays durable lib-streaming terminal job events.
	outboxDispatcher *libOutbox.Dispatcher
}

// Run starts the application.
// This is the only necessary code to run an app in main.go
func (app *Service) Run() {
	runLauncher(app.Logger, app.MultiQueueConsumer, app.healthServer, app.outboxDispatcher)

	// Graceful shutdown
	app.Log(context.Background(), libLog.LevelInfo, "Starting graceful shutdown...")

	// Close multi-tenant resources (Redis) if present.
	// mtConsumer.Close() is handled by MultiQueueConsumer.Run() on context cancellation.
	// mtCleanup only closes Redis connection.
	if app.mtCleanup != nil {
		app.Log(context.Background(), libLog.LevelInfo, "Closing multi-tenant resources (Redis)...")
		app.mtCleanup()
		app.Log(context.Background(), libLog.LevelInfo, "Multi-tenant resources closed")
	}

	// Close readyz-owned resources (e.g. dedicated MT-Redis probe client).
	if app.readyzCloser != nil {
		app.readyzCloser()
	}

	if app.streamingCloser != nil {
		if err := app.streamingCloser(); err != nil {
			app.Log(context.Background(), libLog.LevelError, "failed to close lib-streaming producer", libLog.Err(err))
		}
	}

	// After all consumers are done, shutdown license
	if app.licenseShutdown != nil {
		app.licenseShutdown.Terminate("Consumers are done.")
	}

	app.Log(context.Background(), libLog.LevelInfo, "Graceful shutdown complete")
}
