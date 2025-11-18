package main

import (
	"github.com/LerianStudio/fetcher/components/worker/internal/bootstrap"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
)

func main() {
	libCommons.InitLocalEnvConfig()
	bootstrap.InitWorker().Run()
}
