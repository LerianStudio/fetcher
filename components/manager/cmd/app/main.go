package main

import (
	"log"

	"github.com/LerianStudio/fetcher/v2/components/manager/internal/bootstrap"
	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/startup"
)

func main() {
	pkg.InitLocalEnvConfig()

	service, err := bootstrap.InitServers()
	if err != nil {
		log.Fatalf("failed to initialize manager service: %s", startup.SanitizeError(err))
	}

	service.Run()
}
