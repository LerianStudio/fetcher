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
}

// Run starts the application.
// This is the only necessary code to run an app in main.go
func (app *Service) Run() {
	runLauncher(app.Logger, app.MultiQueueConsumer)

	// Graceful shutdown
	app.Log(context.Background(), libLog.LevelInfo,

		// After all consumers are done, shutdown license
		"Starting graceful shutdown...")

	if app.licenseShutdown != nil {
		app.licenseShutdown.Terminate("Consumers are done.")
	}

	app.Log(context.Background(), libLog.LevelInfo, "Graceful shutdown complete")
}
