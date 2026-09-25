package main

import (
	"log"

	"github.com/LerianStudio/fetcher/v2/components/manager/internal/bootstrap"
	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/startup"
	"github.com/LerianStudio/lib-commons/v7/commons/buildinfo"
)

// Stamped at link time with -ldflags -X main.<name>; empty in a local build,
// where buildinfo falls back to "dev" and the toolchain's VCS stamps.
var version, revision, buildTime string

func main() {
	// Before any configuration is read, so --version answers on a bare image.
	buildinfo.Set(buildinfo.Build{Version: version, Revision: revision, BuildTime: buildTime})
	buildinfo.HandleFlag()

	pkg.InitLocalEnvConfig()

	service, err := bootstrap.InitServers()
	if err != nil {
		log.Fatalf("failed to initialize manager service: %s", startup.SanitizeError(err))
	}

	service.Run()
}
