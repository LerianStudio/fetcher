package main

import (
	"log"

	"github.com/LerianStudio/fetcher/components/worker/internal/bootstrap"
	"github.com/LerianStudio/fetcher/pkg/startup"
	libCommons "github.com/LerianStudio/lib-commons/v3/commons"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("worker service terminated due to unexpected panic: %v", r)
		}
	}()

	libCommons.InitLocalEnvConfig()

	app, err := bootstrap.InitWorker()
	if err != nil {
		log.Fatalf("failed to initialize worker service: %s", startup.SanitizeError(err))
	}

	
	app.Run()
}
