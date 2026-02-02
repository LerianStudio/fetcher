//go:build e2e

// Package extraction contains end-to-end tests for the Fetcher data extraction system.
// These tests verify the complete flow from API request through job processing to result storage.
//
// The tests require Docker to be running and use testcontainers to spin up:
//   - Core infrastructure: MongoDB, RabbitMQ, Redis, SeaweedFS
//   - Source databases: PostgreSQL (always), MySQL, Oracle, SQL Server (optional)
//   - Application containers: Manager API and Worker
//
// Environment variables:
//   - MANAGER_IMAGE: Docker image for Manager (default: fetcher-manager:latest)
//   - WORKER_IMAGE: Docker image for Worker (default: fetcher-worker:latest)
//   - E2E_SKIP_BUILD: If "true", use pre-built images instead of building (default: true)
//   - E2E_ENABLE_MYSQL: If "true", include MySQL in test infrastructure
//   - E2E_ENABLE_ORACLE: If "true", include Oracle in test infrastructure
//   - E2E_ENABLE_MSSQL: If "true", include SQL Server in test infrastructure
//   - E2E_ENABLE_MONGODB: If "true", include MongoDB source in test infrastructure
//   - E2E_INFRA_ONLY: If "true", start infrastructure and block (for debugging)
//   - E2E_REUSE_INFRA: If "true", skip container creation and use existing infrastructure
//   - E2E_SKIP_MANAGER: If "true", skip Manager container (use with E2E_MANAGER_URL)
//   - E2E_SKIP_WORKER: If "true", skip Worker container (use when debugging Worker)
//   - E2E_MANAGER_URL: URL of external Manager (default: http://localhost:4006)
//   - FIXED_PORT: If "true", use fixed ports for containers (required for local debugging)
package extraction

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
	"github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/mongodb"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/mssql"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/mysql"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/oracle"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/postgres"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
)

// Package-level variables hold the test infrastructure.
// These are initialized in TestMain and shared across all test functions.
var (
	// suite manages the lifecycle of all infrastructure containers.
	suite *itestkit.Suite
	// coreInfra holds MongoDB, RabbitMQ, Redis, and SeaweedFS.
	coreInfra *e2eshared.CoreInfra
	// postgresInfra is the PostgreSQL source database (always present).
	postgresInfra *postgres.PostgresInfra
	// mysqlInfra is the MySQL source database (optional, enabled via E2E_ENABLE_MYSQL).
	mysqlInfra *mysql.MySQLInfra
	// oracleInfra is the Oracle source database (optional, enabled via E2E_ENABLE_ORACLE).
	oracleInfra *oracle.OracleInfra
	// mssqlInfra is the SQL Server source database (optional, enabled via E2E_ENABLE_MSSQL).
	mssqlInfra *mssql.MSSQLInfra
	// mongodbSourceInfra is the MongoDB source database (optional, enabled via E2E_ENABLE_MONGODB).
	// Note: This is separate from coreInfra.MongoDB which is used for Fetcher's internal storage.
	mongodbSourceInfra *mongodb.MongoDBInfra
	// managerApp is the running Manager API container.
	managerApp *e2ekit.RunningApp
	// workerApp is the running Worker container.
	workerApp *e2ekit.RunningApp
	// apiClient is the HTTP client for the Manager API.
	apiClient *e2eshared.ManagerClient
	// amqpURL is the RabbitMQ connection URL for queue operations.
	amqpURL string
)

// skipManager returns whether to skip starting the Manager container.
// When true, tests expect an external Manager at E2E_MANAGER_URL.
func skipManager() bool {
	return os.Getenv("E2E_SKIP_MANAGER") == "true"
}

// skipWorker returns whether to skip starting the Worker container.
// When true, tests expect an external Worker (e.g., running in debugger).
func skipWorker() bool {
	return os.Getenv("E2E_SKIP_WORKER") == "true"
}

// externalManagerURL returns the URL of an external Manager service.
// Used when E2E_SKIP_MANAGER=true to connect to a locally running Manager.
func externalManagerURL() string {
	return os.Getenv("E2E_MANAGER_URL")
}

// isFixedPortEnabled returns whether fixed ports are enabled.
func isFixedPortEnabled() bool {
	return os.Getenv("FIXED_PORT") == "true"
}

// infraOnly returns whether to run in infrastructure-only mode.
// When true, only starts infrastructure and blocks until interrupted.
// Used for debugging Manager/Worker locally in VS Code.
func infraOnly() bool {
	return os.Getenv("E2E_INFRA_ONLY") == "true"
}

// reuseInfra returns whether to reuse existing infrastructure.
// When true, skips container creation and connects to already-running containers.
// Use this when running tests against infra started by E2E_INFRA_ONLY=true.
func reuseInfra() bool {
	return os.Getenv("E2E_REUSE_INFRA") == "true"
}

// TestMain is the entry point for E2E tests. It sets up all infrastructure before
// running tests and tears everything down afterward, regardless of test outcomes.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var code int
	defer func() {
		teardown(ctx)
		os.Exit(code)
	}()

	if err := setup(ctx); err != nil {
		log.Fatalf("Setup failed: %v", err)
	}

	code = m.Run()
}

// setup initializes all test infrastructure in the correct order.
// It creates containers for databases, message broker, cache, and file storage,
// then starts the Manager and Worker application containers.
func setup(ctx context.Context) error {
	// If reusing existing infrastructure, skip container creation
	if reuseInfra() {
		return setupReuseInfra()
	}

	log.Println("Starting E2E test infrastructure...")

	// 1. Create core infrastructure (MongoDB, RabbitMQ, Redis, SeaweedFS)
	coreInfra = e2eshared.NewCoreInfra()

	// 2. Create PostgreSQL source database (always required for basic tests)
	pgOpts := []postgres.PostgresOption{
		postgres.WithPGInitFile(fixturesPath("postgres_init.sql"), "init.sql"),
	}
	if isFixedPortEnabled() {
		pgOpts = append(pgOpts, postgres.WithPGFixedPort("5432"))
	}

	postgresInfra = postgres.NewPostgresInfra(postgres.PostgresConfig{
		Name:     "source",
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
		Options:  pgOpts,
	})

	// 3. Build itestkit suite with core infrastructure and PostgreSQL
	suiteBuilder := itestkit.New(nil).
		WithInfras(coreInfra.Infras()...).
		WithInfra(postgresInfra)

	// 4. Optionally add MySQL if enabled via E2E_ENABLE_MYSQL=true
	if isInfraEnabled("MYSQL") {
		log.Println("MySQL infrastructure enabled")
		mysqlOpts := []mysql.MySQLOption{
			mysql.WithMySQLInitScript(fixturesPath("mysql_init.sql"), "init.sql"),
		}
		if isFixedPortEnabled() {
			mysqlOpts = append(mysqlOpts, mysql.WithMySQLFixedPort("3306"))
		}

		mysqlInfra = mysql.NewMySQLInfra(mysql.MySQLConfig{
			Name:     "source-mysql",
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
			Options:  mysqlOpts,
		})
		suiteBuilder = suiteBuilder.WithInfra(mysqlInfra)
	}

	// 5. Optionally add Oracle if enabled via E2E_ENABLE_ORACLE=true
	if isInfraEnabled("ORACLE") {
		log.Println("Oracle infrastructure enabled")
		oracleOpts := []oracle.OracleOption{
			oracle.WithOracleInitScript(fixturesPath("oracle_init.sql"), "init.sql"),
		}
		if isFixedPortEnabled() {
			oracleOpts = append(oracleOpts, oracle.WithOracleFixedPort("1521"))
		}

		oracleInfra = oracle.NewOracleInfra(oracle.OracleConfig{
			Name:     "source-oracle",
			Password: "testpass",
			SID:      "XE",
			Options:  oracleOpts,
		})
		suiteBuilder = suiteBuilder.WithInfra(oracleInfra)
	}

	// 6. Optionally add SQL Server if enabled via E2E_ENABLE_MSSQL=true
	if isInfraEnabled("MSSQL") {
		log.Println("SQL Server infrastructure enabled")
		mssqlOpts := []mssql.MSSQLOption{}
		if isFixedPortEnabled() {
			mssqlOpts = append(mssqlOpts, mssql.WithMSSQLFixedPort("1433"))
		}

		mssqlInfra = mssql.NewMSSQLInfra(mssql.MSSQLConfig{
			Name:     "source-mssql",
			Database: "testdb",
			// SQL Server requires complex passwords (uppercase, lowercase, number, special char)
			Password: "YourStrong@Passw0rd",
			Options:  mssqlOpts,
		})
		suiteBuilder = suiteBuilder.WithInfra(mssqlInfra)
	}

	// 7. Optionally add MongoDB source if enabled via E2E_ENABLE_MONGODB=true
	// Note: This is a separate MongoDB instance for data extraction, NOT the CoreInfra MongoDB.
	if isInfraEnabled("MONGODB") {
		log.Println("MongoDB source infrastructure enabled")
		mongoOpts := []mongodb.MongoDBOption{}
		if isFixedPortEnabled() {
			mongoOpts = append(mongoOpts, mongodb.WithMongoDBFixedPort("27017"))
		}

		mongodbSourceInfra = mongodb.NewMongoDBInfra(mongodb.MongoDBConfig{
			Name:     "source-mongodb",
			Username: "testuser",
			Password: "testpass",
			Options:  mongoOpts,
		})
		suiteBuilder = suiteBuilder.WithInfra(mongodbSourceInfra)
	}

	// 8. Build and start all infrastructure containers
	var err error
	suite, err = suiteBuilder.Build(ctx)
	if err != nil {
		return fmt.Errorf("build suite: %w", err)
	}

	// 9. Build app environment with connection details for container-to-container communication
	appEnv, err := e2eshared.BuildAppEnv(suite.Network(), coreInfra.MongoDB, coreInfra.RabbitMQ, coreInfra.Redis, coreInfra.SeaweedFS)
	if err != nil {
		return fmt.Errorf("build app env: %w", err)
	}

	// Store AMQP URL for direct queue operations in tests
	amqpURL, _ = coreInfra.RabbitMQ.AMQPURL()

	// 10. If infra-only mode, block and wait for interrupt
	if infraOnly() {
		log.Println("")
		log.Println("═══════════════════════════════════════════════════════════════")
		log.Println("  INFRA-ONLY MODE - Infrastructure is ready for debugging")
		log.Println("═══════════════════════════════════════════════════════════════")
		log.Println("")
		log.Println("  MongoDB:    localhost:5709")
		log.Println("  RabbitMQ:   localhost:3008")
		log.Println("  Redis:      localhost:5707")
		log.Println("  SeaweedFS:  localhost:8889")
		log.Println("  PostgreSQL: localhost:5432")
		log.Println("")
		log.Println("  Start Manager/Worker in VS Code, then press Ctrl+C to exit.")
		log.Println("═══════════════════════════════════════════════════════════════")

		// Block until interrupt
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("\nShutting down infrastructure...")
		return fmt.Errorf("infra-only mode: interrupted by user")
	}

	// 11. Start Manager HTTP API container (unless skipped for debugging)
	if !skipManager() {
		log.Println("Starting Manager container...")
		managerApp, err = e2eshared.StartManager(nil, ctx, appEnv, e2eshared.AppStartConfig{
			Image:     managerImage(),
			SkipBuild: skipBuild(),
		})
		if err != nil {
			return fmt.Errorf("start manager: %w", err)
		}
	} else {
		log.Println("Skipping Manager container (E2E_SKIP_MANAGER=true)")
	}

	// 12. Start Worker message consumer container (unless skipped for debugging)
	if !skipWorker() {
		log.Println("Starting Worker container...")
		workerApp, err = e2eshared.StartWorker(nil, ctx, appEnv, e2eshared.AppStartConfig{
			Image:     workerImage(),
			SkipBuild: skipBuild(),
		})
		if err != nil {
			return fmt.Errorf("start worker: %w", err)
		}
	} else {
		log.Println("Skipping Worker container (E2E_SKIP_WORKER=true)")
	}

	// 13. Create API client configured for the running Manager
	if url := externalManagerURL(); url != "" {
		log.Printf("Using external Manager at: %s", url)
		apiClient = e2eshared.NewManagerClient(url, e2eshared.TestOrganizationID)
	} else if managerApp != nil {
		apiClient = e2eshared.NewClientFromApp(managerApp)
	} else {
		return fmt.Errorf("no Manager available: set E2E_MANAGER_URL or disable E2E_SKIP_MANAGER")
	}

	log.Println("E2E infrastructure ready")
	return nil
}

// teardown stops and removes all test containers in reverse order.
// It's called from TestMain's defer to ensure cleanup happens even if tests panic.
// When E2E_REUSE_INFRA=true, teardown is skipped to keep containers running.
func teardown(ctx context.Context) {
	// Don't teardown if we're reusing existing infrastructure
	if reuseInfra() {
		log.Println("Skipping teardown (E2E_REUSE_INFRA=true)")
		return
	}

	log.Println("Tearing down E2E infrastructure...")

	if workerApp != nil {
		_ = workerApp.Container.Terminate(ctx)
	}
	if managerApp != nil {
		_ = managerApp.Container.Terminate(ctx)
	}
	if suite != nil {
		_ = suite.Terminate(ctx)
	}
}

// managerImage returns the Docker image to use for the Manager container.
// Reads from MANAGER_IMAGE env var, defaulting to "fetcher-manager:latest".
func managerImage() string {
	if img := os.Getenv("MANAGER_IMAGE"); img != "" {
		return img
	}
	return "fetcher-manager:latest"
}

// workerImage returns the Docker image to use for the Worker container.
// Reads from WORKER_IMAGE env var, defaulting to "fetcher-worker:latest".
func workerImage() string {
	if img := os.Getenv("WORKER_IMAGE"); img != "" {
		return img
	}
	return "fetcher-worker:latest"
}

// skipBuild returns whether to skip building Docker images.
// If true, uses pre-built images; if false, builds from Dockerfile.
// Reads from E2E_SKIP_BUILD env var, defaulting to true.
func skipBuild() bool {
	if value := os.Getenv("E2E_SKIP_BUILD"); value != "" {
		return value == "true"
	}
	return true
}

// fixturesPath returns the absolute path to a fixture file in tests/shared/fixtures.
func fixturesPath(name string) string {
	return filepath.Join(e2ekit.ProjectRoot(), "tests", "shared", "fixtures", name)
}

// isInfraEnabled checks if an optional infrastructure type is enabled via environment variable.
// The environment variable format is E2E_ENABLE_{infraType} (e.g., E2E_ENABLE_MYSQL=true).
func isInfraEnabled(infraType string) bool {
	envVar := fmt.Sprintf("E2E_ENABLE_%s", infraType)
	return os.Getenv(envVar) == "true"
}

// setupReuseInfra configures the test to use already-running infrastructure.
// This is used when containers were started via E2E_INFRA_ONLY=true and are still running.
// It skips container creation and creates stub infra objects with fixed port endpoints.
func setupReuseInfra() error {
	log.Println("Reusing existing E2E infrastructure (E2E_REUSE_INFRA=true)...")

	// Create PostgreSQL stub with fixed port endpoint
	postgresInfra = postgres.NewPostgresInfraStub(postgres.PostgresConfig{
		Name:     "source",
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
	}, "localhost", 5432)

	// Create MySQL stub if enabled
	if isInfraEnabled("MYSQL") {
		mysqlInfra = mysql.NewMySQLInfraStub(mysql.MySQLConfig{
			Name:     "source-mysql",
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
		}, "localhost", 3306)
	}

	// Create Oracle stub if enabled
	if isInfraEnabled("ORACLE") {
		oracleInfra = oracle.NewOracleInfraStub(oracle.OracleConfig{
			Name:     "source-oracle",
			Password: "testpass",
			SID:      "XE",
		}, "localhost", 1521)
	}

	// Create MSSQL stub if enabled
	if isInfraEnabled("MSSQL") {
		mssqlInfra = mssql.NewMSSQLInfraStub(mssql.MSSQLConfig{
			Name:     "source-mssql",
			Database: "testdb",
			Password: "YourStrong@Passw0rd",
		}, "localhost", 1433)
	}

	// Create MongoDB source stub if enabled
	if isInfraEnabled("MONGODB") {
		mongodbSourceInfra = mongodb.NewMongoDBInfraStub(mongodb.MongoDBConfig{
			Name:     "source-mongodb",
			Username: "testuser",
			Password: "testpass",
		}, "localhost", 27017)
	}

	// Configure API client to connect to external Manager
	managerURL := externalManagerURL()
	if managerURL == "" {
		managerURL = "http://localhost:4006" // Default when using fixed ports
	}

	log.Printf("Using Manager at: %s", managerURL)
	apiClient = e2eshared.NewManagerClient(managerURL, e2eshared.TestOrganizationID)

	// Configure AMQP URL for direct queue operations (using fixed port)
	amqpURL = "amqp://plugin:Lerian%40123@localhost:3008"

	log.Println("E2E infrastructure ready (reusing existing containers)")
	return nil
}
