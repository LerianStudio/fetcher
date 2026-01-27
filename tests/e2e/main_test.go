//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
	"github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/postgres"
	e2eshared "github.com/LerianStudio/fetcher/tests/e2e/shared"
)

var (
	suite         *itestkit.Suite
	coreInfra     *e2eshared.CoreInfra
	postgresInfra *postgres.PostgresInfra
	managerApp    *e2ekit.RunningApp
	workerApp     *e2ekit.RunningApp
	apiClient     *e2eshared.ManagerClient
	amqpURL       string
)

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

func setup(ctx context.Context) error {
	log.Println("Starting E2E test infrastructure...")

	// 1. Create core infrastructure
	coreInfra = e2eshared.NewCoreInfra()

	// 2. Create PostgreSQL (source database)
	postgresInfra = postgres.NewPostgresInfra(postgres.PostgresConfig{
		Name:     "source",
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
		Options: []postgres.PostgresOption{
			postgres.WithPGInitFile(fixturesPath("postgres_init.sql"), "init.sql"),
		},
	})

	// 3. Build the itestkit suite
	var err error
	suite, err = itestkit.New(nil).
		WithInfras(coreInfra.Infras()...).
		WithInfra(postgresInfra).
		Build(ctx)
	if err != nil {
		return fmt.Errorf("build suite: %w", err)
	}

	// 4. Build app environment
	appEnv, err := e2eshared.BuildAppEnv(coreInfra.MongoDB, coreInfra.RabbitMQ, coreInfra.Redis, coreInfra.SeaweedFS)
	if err != nil {
		return fmt.Errorf("build app env: %w", err)
	}

	// Store AMQP URL for queuekit
	amqpURL, _ = coreInfra.RabbitMQ.AMQPURL()

	// 5. Start Manager container
	log.Println("Starting Manager container...")
	managerApp, err = e2eshared.StartManager(nil, ctx, appEnv, e2eshared.AppStartConfig{
		Image:     managerImage(),
		SkipBuild: skipBuild(),
	})
	if err != nil {
		return fmt.Errorf("start manager: %w", err)
	}

	// 6. Start Worker container
	log.Println("Starting Worker container...")
	workerApp, err = e2eshared.StartWorker(nil, ctx, appEnv, e2eshared.AppStartConfig{
		Image:     workerImage(),
		SkipBuild: skipBuild(),
	})
	if err != nil {
		return fmt.Errorf("start worker: %w", err)
	}

	// 7. Create API client
	apiClient = e2eshared.NewClientFromApp(managerApp)

	log.Println("E2E infrastructure ready")
	return nil
}

func teardown(ctx context.Context) {
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

func managerImage() string {
	if img := os.Getenv("MANAGER_IMAGE"); img != "" {
		return img
	}
	return "fetcher-manager:latest"
}

func workerImage() string {
	if img := os.Getenv("WORKER_IMAGE"); img != "" {
		return img
	}
	return "fetcher-worker:latest"
}

func skipBuild() bool {
	if value := os.Getenv("E2E_SKIP_BUILD"); value != "" {
		return value == "true"
	}
	return true
}

func fixturesPath(name string) string {
	return filepath.Join(e2ekit.ProjectRoot(), "tests", "shared", "fixtures", name)
}
