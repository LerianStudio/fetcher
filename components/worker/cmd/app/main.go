package main

import (
	"log"

	"github.com/LerianStudio/fetcher/components/worker/internal/bootstrap"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
)

func main() {
	libCommons.InitLocalEnvConfig()

	service, err := bootstrap.InitWorker()
	if err != nil {
		log.Fatal(err)
	}

	service.Run()
}
