package bootstrap

import (
	"context"

	"github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
)

type licenseTerminator interface {
	Terminate(msg string)
}

// Service is the application glue where we put all top level components to be used.
type Service struct {
	*MultiQueueConsumer
	libLog.
		Logger
	licenseShutdown licenseTerminator
}

// Run starts the application.
// This is the only necessary code to run an app in main.go
func (app *Service) Run() {
	commons.NewLauncher(
		commons.WithLogger(app.Logger),
		commons.RunApp("RabbitMQ Consumer", app.MultiQueueConsumer),
	).Run()

	// Graceful shutdown
	app.Log(context.Background(), libLog.LevelInfo,

		// After all consumers are done, shutdown license
		"Starting graceful shutdown...")

	if app.licenseShutdown != nil {
		app.licenseShutdown.Terminate("Consumers are done.")
	}

	app.Log(context.Background(), libLog.LevelInfo, "Graceful shutdown complete")
}
