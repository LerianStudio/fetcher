//go:build integration
// +build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LerianStudio/fetcher/tests/integration/containers/fixtures"
	"github.com/LerianStudio/fetcher/tests/integration/containers/setup"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fmt.Println("Starting infrastructure containers with fixed ports...")
	fmt.Println("This enables VS Code debugging and infrastructure reuse.")
	fmt.Println("")

	// Use fixed ports for debug mode
	opts := setup.DebugInfrastructureOptions()

	infra, err := setup.StartInfrastructureWithOptions(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start infrastructure: %v\n", err)
		os.Exit(1)
	}

	// Setup RabbitMQ topology
	err = setup.SetupRabbitMQTopology(ctx, infra.RabbitMQURI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup RabbitMQ topology: %v\n", err)
		infra.Stop(ctx)
		os.Exit(1)
	}

	// Seed MongoDB External
	err = fixtures.InitMongoDBExternal(ctx, infra.MongoExternalURI, "external_transactions")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to seed MongoDB: %v\n", err)
		infra.Stop(ctx)
		os.Exit(1)
	}

	// Get mapped ports for display
	mongoMainPort, _ := infra.MongoMain.MappedPort(ctx, "27017")
	rabbitmqPort, _ := infra.RabbitMQ.MappedPort(ctx, "5672")
	seaweedfsPort, _ := infra.SeaweedFS.MappedPort(ctx, "8888")
	redisPort, _ := infra.Redis.MappedPort(ctx, "6379")
	postgresPort, _ := infra.PostgresExternal.MappedPort(ctx, "5432")
	mysqlPort, _ := infra.MySQLExternal.MappedPort(ctx, "3306")
	sqlserverPort, _ := infra.SQLServerExternal.MappedPort(ctx, "1433")
	oraclePort, _ := infra.OracleExternal.MappedPort(ctx, "1521")

	// Build config for JSON output
	config := map[string]interface{}{
		"mongoMainUri":       infra.MongoMainURI,
		"mongoExternalUri":   infra.MongoExternalURI,
		"rabbitmqUri":        infra.RabbitMQURI,
		"seaweedfsUrl":       infra.SeaweedFSURL,
		"redisUrl":           infra.RedisURL,
		"postgresUrl":        infra.PostgresURL,
		"mysqlUrl":           infra.MySQLURL,
		"sqlserverUrl":       infra.SQLServerURL,
		"oracleUrl":          infra.OracleURL,
		"mongoMainPort":      mongoMainPort.Port(),
		"rabbitmqPort":       rabbitmqPort.Port(),
		"seaweedfsFilerPort": seaweedfsPort.Port(),
		"redisPort":          redisPort.Port(),
		"postgresPort":       postgresPort.Port(),
		"mysqlPort":          mysqlPort.Port(),
		"sqlserverPort":      sqlserverPort.Port(),
		"oraclePort":         oraclePort.Port(),
	}

	configJSON, _ := json.MarshalIndent(config, "", "  ")

	fmt.Println("\n" + string(repeatChar('=', 70)))
	fmt.Println("INFRASTRUCTURE READY")
	fmt.Println(string(repeatChar('=', 70)))
	fmt.Println("\nConnection details (JSON):")
	fmt.Println(string(configJSON))

	fmt.Printf("\nConfig saved to: %s\n", setup.InfraConfigPath)

	fmt.Println("\n" + string(repeatChar('-', 70)))
	fmt.Println("FIXED PORTS (for VS Code debugging):")
	fmt.Println(string(repeatChar('-', 70)))
	fmt.Printf("  MongoDB Main:     localhost:%s\n", mongoMainPort.Port())
	fmt.Printf("  RabbitMQ:         localhost:%s\n", rabbitmqPort.Port())
	fmt.Printf("  SeaweedFS Filer:  localhost:%s\n", seaweedfsPort.Port())
	fmt.Printf("  Redis:            localhost:%s\n", redisPort.Port())
	fmt.Printf("  PostgreSQL:       localhost:%s\n", postgresPort.Port())
	fmt.Printf("  MySQL:            localhost:%s\n", mysqlPort.Port())
	fmt.Printf("  SQL Server:       localhost:%s\n", sqlserverPort.Port())
	fmt.Printf("  Oracle:           localhost:%s\n", oraclePort.Port())

	fmt.Println("\n" + string(repeatChar('-', 70)))
	fmt.Println("ENVIRONMENT VARIABLES FOR MANAGER (copy/paste to terminal):")
	fmt.Println(string(repeatChar('-', 70)))
	fmt.Printf(`
export ENV_NAME=test
export LOG_LEVEL=debug
export SERVER_ADDRESS=:4006
export MONGO_URI=mongodb
export MONGO_HOST=localhost
export MONGO_PORT=%s
export MONGO_NAME=fetcher_test
export MONGO_USER=root
export MONGO_PASSWORD=password
export RABBITMQ_URI=amqp
export RABBITMQ_HOST=localhost
export RABBITMQ_PORT_AMQP=%s
export RABBITMQ_DEFAULT_USER=guest
export RABBITMQ_DEFAULT_PASS=guest
export SEAWEEDFS_HOST=localhost
export SEAWEEDFS_FILER_PORT=%s
export REDIS_HOST=localhost
export REDIS_PORT=%s
export REDIS_PASSWORD=
export APP_ENC_KEY=MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=
export APP_ENC_KEY_VERSION=1
export ENABLE_TELEMETRY=false
export PLUGIN_AUTH_ENABLED=false
export LICENSE_KEY=test-license-key
`, mongoMainPort.Port(), rabbitmqPort.Port(), seaweedfsPort.Port(), redisPort.Port())

	fmt.Println("\n" + string(repeatChar('-', 70)))
	fmt.Println("ENVIRONMENT VARIABLES FOR WORKER (copy/paste to terminal):")
	fmt.Println(string(repeatChar('-', 70)))
	fmt.Printf(`
export ENV_NAME=test
export LOG_LEVEL=debug
export MONGO_URI=mongodb
export MONGO_HOST=localhost
export MONGO_PORT=%s
export MONGO_NAME=fetcher_test
export MONGO_USER=root
export MONGO_PASSWORD=password
export RABBITMQ_URI=amqp
export RABBITMQ_HOST=localhost
export RABBITMQ_PORT_AMQP=%s
export RABBITMQ_DEFAULT_USER=guest
export RABBITMQ_DEFAULT_PASS=guest
export RABBITMQ_GENERATE_REPORT_QUEUE=extract-external-data-queue
export RABBITMQ_JOB_EVENTS_EXCHANGE=fetcher.job.events
export RABBITMQ_NUMBERS_OF_WORKERS=2
export SEAWEEDFS_HOST=localhost
export SEAWEEDFS_FILER_PORT=%s
export SEAWEEDFS_TTL=
export REDIS_HOST=localhost
export REDIS_PORT=%s
export REDIS_PASSWORD=
export APP_ENC_KEY=MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=
export APP_ENC_KEY_VERSION=1
export CRYPTO_ENCRYPT_SECRET_KEY_SEAWEEDFS=3132333435363738393031323334353637383930313233343536373839303132
export CRYPTO_HASH_SECRET_KEY_SEAWEEDFS=3132333435363738393031323334353637383930313233343536373839303132
export ENABLE_TELEMETRY=false
export LICENSE_KEY=test-license-key
export ORGANIZATION_IDS=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
`, mongoMainPort.Port(), rabbitmqPort.Port(), seaweedfsPort.Port(), redisPort.Port())

	fmt.Println("\n" + string(repeatChar('-', 70)))
	fmt.Println("USAGE INSTRUCTIONS:")
	fmt.Println(string(repeatChar('-', 70)))
	fmt.Print(`
1. DEBUG MANAGER ONLY (VS Code):
   - Open VS Code, press F5, select "Manager (Integration Test Debug)"
   - Run tests: make test-integration-debug-manager [TEST=<test_name>]

2. DEBUG WORKER ONLY (VS Code):
   - Open VS Code, press F5, select "Worker (Integration Test Debug)"
   - Run tests: make test-integration-debug-worker [TEST=<test_name>]

3. DEBUG BOTH (VS Code):
   - Start Manager in VS Code (step 1)
   - Start Worker in another VS Code window
   - Run tests: make test-integration-debug-full [TEST=<test_name>]
`)

	fmt.Println("\n" + string(repeatChar('-', 70)))
	fmt.Println("Press Ctrl+C to stop infrastructure")
	fmt.Println(string(repeatChar('-', 70)))

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nStopping infrastructure...")

	// Remove config file
	setup.RemoveInfraConfig()

	infra.Stop(context.Background())
	fmt.Println("Infrastructure stopped.")
}

func repeatChar(c byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return b
}
