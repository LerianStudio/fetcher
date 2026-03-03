package main

import (
	"log"

	"github.com/LerianStudio/fetcher/components/manager/internal/bootstrap"
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/startup"
)

// @title					Fetcher Manager API
// @version					1.0.0
// @description				API documentation for the Fetcher Manager component
// @termsOfService			http://swagger.io/terms/
// @host					localhost:4006
// @BasePath					/
func main() {
	pkg.InitLocalEnvConfig()

	app, err := bootstrap.InitServers()
	if err != nil {
		log.Fatalf("failed to initialize manager service: %s", startup.SanitizeError(err))
	}

	
	app.Run()
}
