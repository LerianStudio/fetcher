package main

import (
	"fmt"
	"os"

	"github.com/LerianStudio/fetcher/components/worker/internal/bootstrap"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
)

func main() {
	libCommons.InitLocalEnvConfig()

	service, err := bootstrap.InitWorker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize worker: %v\n", err)
		os.Exit(1)
	}

	service.Run()
}
