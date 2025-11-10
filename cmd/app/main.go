package main

import (
	"golang-plugin-boilerplate/internal/bootstrap"
	"golang-plugin-boilerplate/pkg"
)

// @title					Golang plugin boilerplate
// @version					1.0.0
// @description				This is a swagger documentation for Golang plugin boilerplate
// @termsOfService			http://swagger.io/terms/
// @host					localhost:4000
// @BasePath					/
func main() {
	pkg.InitLocalEnvConfig()
	bootstrap.InitServers().Run()
}
