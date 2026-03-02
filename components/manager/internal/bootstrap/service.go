package bootstrap

import (
	"github.com/LerianStudio/lib-commons/v3/commons"
	"github.com/LerianStudio/lib-commons/v3/commons/log"
)

// Service is the application glue where we put all top-level components to be used.
type Service struct {
	*Server
	log.Logger
	cleanups []func()
}

// Run starts the application.
// This is the only necessary code to run an app in the main.go
func (app *Service) Run() {
	defer app.runCleanups()

	commons.NewLauncher(
		commons.WithLogger(app.Logger),
		commons.RunApp("HTTP Service", app.Server),
	).Run()
}

// runCleanups executes all registered cleanup functions in reverse order.
func (app *Service) runCleanups() {
	for i := len(app.cleanups) - 1; i >= 0; i-- {
		app.cleanups[i]()
	}
}
