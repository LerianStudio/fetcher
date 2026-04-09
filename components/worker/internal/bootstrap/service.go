package bootstrap

import (
	"context"

	"github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
)

var runLauncher = func(logger libLog.Logger, consumer *MultiQueueConsumer) {
	commons.NewLauncher(
		commons.WithLogger(logger),
		commons.RunApp("RabbitMQ Consumer", consumer),
	).Run()
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
}

// Run starts the application.
// This is the only necessary code to run an app in main.go
func (app *Service) Run() {
	runLauncher(app.Logger, app.MultiQueueConsumer)

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

	// After all consumers are done, shutdown license
	if app.licenseShutdown != nil {
		app.licenseShutdown.Terminate("Consumers are done.")
	}

	app.Log(context.Background(), libLog.LevelInfo, "Graceful shutdown complete")
}
