package main

import (
	"log"

	"github.com/LerianStudio/fetcher/v2/components/worker/internal/bootstrap"
	"github.com/LerianStudio/fetcher/v2/pkg/startup"
	libCommons "github.com/LerianStudio/lib-commons/v7/commons"
	"github.com/LerianStudio/lib-commons/v7/commons/buildinfo"
)

// Stamped at link time with -ldflags -X main.<name>; empty in a local build,
// where buildinfo falls back to "dev" and the toolchain's VCS stamps.
var version, revision, buildTime string

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("worker service terminated due to unexpected panic: %v", r)
		}
	}()

	// Before any configuration is read, so --version answers on a bare image.
	buildinfo.Set(buildinfo.Build{Version: version, Revision: revision, BuildTime: buildTime})
	buildinfo.HandleFlag()

	libCommons.InitLocalEnvConfig()

	app, err := bootstrap.InitWorker()
	if err != nil {
		log.Fatalf("failed to initialize worker service: %s", startup.SanitizeError(err))
	}

	app.Run()
}
