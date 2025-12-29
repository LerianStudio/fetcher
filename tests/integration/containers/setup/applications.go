package setup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// buildImageWithDocker builds a Docker image using Docker CLI with BuildKit support.
// This is needed because testcontainers-go doesn't properly support BuildKit sessions
// required for Dockerfiles with `# syntax=docker/dockerfile:1.4` directive.
func buildImageWithDocker(ctx context.Context, projectRoot, dockerfile, imageName, githubToken string) error {
	args := []string{"build", "-f", dockerfile, "-t", imageName}

	// Add GitHub token as build arg if provided
	if githubToken != "" {
		args = append(args, "--build-arg", "GITHUB_TOKEN="+githubToken)
	}

	args = append(args, projectRoot)

	// #nosec G204 -- args are constructed from controlled test inputs, not user input
	cmd := exec.CommandContext(ctx, "docker", args...)

	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// ApplicationContainers holds Manager and Worker containers.
type ApplicationContainers struct {
	Manager testcontainers.Container
	Worker  testcontainers.Container

	ManagerURL string
}

// ApplicationConfig holds configuration for starting applications.
type ApplicationConfig struct {
	Network       string
	MongoHost     string
	MongoPort     string
	RabbitMQHost  string
	RabbitMQPort  string
	SeaweedFSHost string
	SeaweedFSPort string
	RedisHost     string
	RedisPort     string
	// EncryptionKeyBase64 is the Base64-encoded 32-byte key for Manager's APP_ENC_KEY
	EncryptionKeyBase64 string
	// EncryptionKeyHex is the Hex-encoded 32-byte key for Worker's CRYPTO_*_SEAWEEDFS keys
	EncryptionKeyHex string
}

// ApplicationOptions controls how applications are started.
type ApplicationOptions struct {
	// SkipManager skips starting Manager container (for Manager debug mode).
	// When true, ExternalManagerURL must be set.
	SkipManager bool

	// SkipWorker skips starting Worker container (for Worker debug mode).
	SkipWorker bool

	// ExternalManagerURL is the URL of an externally running Manager.
	// Used when SkipManager is true.
	ExternalManagerURL string

	// ExternalWorkerHost is the hostname where Worker is running externally.
	// Used when SkipWorker is true and Manager needs to know Worker's address.
	// For Linux, this can be empty as containers can reach host directly.
	ExternalWorkerHost string
}

// DefaultApplicationOptions returns options for normal test execution.
func DefaultApplicationOptions() ApplicationOptions {
	return ApplicationOptions{
		SkipManager: false,
		SkipWorker:  false,
	}
}

// ManagerDebugOptions returns options for debugging Manager locally.
func ManagerDebugOptions(managerURL string) ApplicationOptions {
	return ApplicationOptions{
		SkipManager:        true,
		SkipWorker:         false,
		ExternalManagerURL: managerURL,
	}
}

// WorkerDebugOptions returns options for debugging Worker locally.
func WorkerDebugOptions() ApplicationOptions {
	return ApplicationOptions{
		SkipManager: false,
		SkipWorker:  true,
	}
}

// FullDebugOptions returns options for debugging both Manager and Worker locally.
func FullDebugOptions(managerURL string) ApplicationOptions {
	return ApplicationOptions{
		SkipManager:        true,
		SkipWorker:         true,
		ExternalManagerURL: managerURL,
	}
}

// StartApplications starts Manager and Worker containers.
// This is the original interface for backward compatibility.
func StartApplications(ctx context.Context, infra *InfrastructureContainers, cfg ApplicationConfig) (*ApplicationContainers, error) {
	// Check for external Manager URL (backward compatibility with existing debug mode)
	externalManagerURL := os.Getenv("EXTERNAL_MANAGER_URL")

	opts := DefaultApplicationOptions()
	if externalManagerURL != "" {
		opts = ManagerDebugOptions(externalManagerURL)
	}

	// Check for skip Worker (new feature)
	if os.Getenv("SKIP_WORKER") == "true" {
		opts.SkipWorker = true
	}

	return StartApplicationsWithOptions(ctx, infra, cfg, opts)
}

// Stop stops all application containers.
func (a *ApplicationContainers) Stop(ctx context.Context) error {
	var errs []error

	if a.Worker != nil {
		if err := a.Worker.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if a.Manager != nil {
		if err := a.Manager.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping containers: %v", errs)
	}

	return nil
}

// findProjectRoot finds the project root directory.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root")
		}

		dir = parent
	}
}

// mergeEnv merges two environment variable maps.
func mergeEnv(base, override map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}

	for k, v := range override {
		result[k] = v
	}

	return result
}

// StartApplicationsWithOptions starts Manager and/or Worker containers based on options.
func StartApplicationsWithOptions(ctx context.Context, infra *InfrastructureContainers, cfg ApplicationConfig, opts ApplicationOptions) (*ApplicationContainers, error) {
	apps := &ApplicationContainers{}

	// Get project root for Dockerfile context
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	networkName := "fetcher-test-network"

	// Common environment variables
	commonEnv := map[string]string{
		"ENV_NAME":                            "test",
		"LOG_LEVEL":                           "info",
		"MONGO_URI":                           "mongodb",
		"MONGO_HOST":                          "fetcher-mongodb",
		"MONGO_PORT":                          "27017",
		"MONGO_NAME":                          "fetcher_test",
		"MONGO_USER":                          "root",
		"MONGO_PASSWORD":                      "password",
		"RABBITMQ_URI":                        "amqp",
		"RABBITMQ_HOST":                       "fetcher-rabbitmq",
		"RABBITMQ_PORT_AMQP":                  "5672",
		"RABBITMQ_DEFAULT_USER":               "guest",
		"RABBITMQ_DEFAULT_PASS":               "guest",
		"SEAWEEDFS_HOST":                      "fetcher-seaweedfs-filer",
		"SEAWEEDFS_FILER_PORT":                "8888",
		"REDIS_HOST":                          "fetcher-valkey",
		"REDIS_PORT":                          "6379",
		"REDIS_PASSWORD":                      "",
		"APP_ENC_KEY":                         cfg.EncryptionKeyBase64,
		"APP_ENC_KEY_VERSION":                 "1",
		"ENABLE_TELEMETRY":                    "false",
		"PLUGIN_AUTH_ENABLED":                 "false",
		"CRYPTO_ENCRYPT_SECRET_KEY_SEAWEEDFS": cfg.EncryptionKeyHex,
		"CRYPTO_HASH_SECRET_KEY_SEAWEEDFS":    cfg.EncryptionKeyHex,
		"ORGANIZATION_IDS":                    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"LICENSE_KEY":                         "test-license-key",
	}

	// Check for GitHub token for private dependencies
	githubToken := os.Getenv("GITHUB_TOKEN")

	// Start Manager if not skipped
	if !opts.SkipManager {
		managerContainer, managerURL, err := startManagerContainer(ctx, projectRoot, networkName, commonEnv, githubToken)
		if err != nil {
			return nil, err
		}

		apps.Manager = managerContainer
		apps.ManagerURL = managerURL
	} else {
		// Use external Manager URL
		if opts.ExternalManagerURL == "" {
			return nil, fmt.Errorf("ExternalManagerURL must be set when SkipManager is true")
		}

		apps.ManagerURL = opts.ExternalManagerURL
	}

	// Start Worker if not skipped
	if !opts.SkipWorker {
		workerContainer, err := startWorkerContainer(ctx, projectRoot, networkName, commonEnv, githubToken)
		if err != nil {
			if apps.Manager != nil {
				_ = apps.Manager.Terminate(ctx)
			}

			return nil, err
		}

		apps.Worker = workerContainer
	}
	// If Worker is skipped, nothing to do - Worker runs externally

	return apps, nil
}

// startManagerContainer starts the Manager container.
func startManagerContainer(ctx context.Context, projectRoot, networkName string, commonEnv map[string]string, githubToken string) (testcontainers.Container, string, error) {
	var managerImage string

	if githubToken != "" {
		// Build the image using Docker CLI with BuildKit support
		managerImage = "fetcher-manager:test-build"
		if err := buildImageWithDocker(ctx, projectRoot, "components/manager/Dockerfile", managerImage, githubToken); err != nil {
			return nil, "", fmt.Errorf("failed to build manager image: %w", err)
		}
	} else {
		// Use pre-built image
		managerImage = os.Getenv("MANAGER_IMAGE")
		if managerImage == "" {
			return nil, "", fmt.Errorf("GITHUB_TOKEN or MANAGER_IMAGE environment variable required")
		}
	}

	managerReq := testcontainers.ContainerRequest{
		Image:        managerImage,
		ExposedPorts: []string{"4006/tcp"},
		Networks:     []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"manager"},
		},
		Env: mergeEnv(commonEnv, map[string]string{
			"SERVER_ADDRESS": ":4006",
		}),
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/health").WithPort("4006/tcp").WithStartupTimeout(ManagerStartupTimeout),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: managerReq,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to start manager: %w", err)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "4006")
	managerURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	return container, managerURL, nil
}

// startWorkerContainer starts the Worker container.
func startWorkerContainer(ctx context.Context, projectRoot, networkName string, commonEnv map[string]string, githubToken string) (testcontainers.Container, error) {
	var workerImage string

	if githubToken != "" {
		// Build the image using Docker CLI with BuildKit support
		workerImage = "fetcher-worker:test-build"
		if err := buildImageWithDocker(ctx, projectRoot, "components/worker/Dockerfile", workerImage, githubToken); err != nil {
			return nil, fmt.Errorf("failed to build worker image: %w", err)
		}
	} else {
		// Use pre-built image
		workerImage = os.Getenv("WORKER_IMAGE")
		if workerImage == "" {
			return nil, fmt.Errorf("GITHUB_TOKEN or WORKER_IMAGE environment variable required")
		}
	}

	workerReq := testcontainers.ContainerRequest{
		Image:    workerImage,
		Networks: []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"worker"},
		},
		Env: mergeEnv(commonEnv, map[string]string{
			"RABBITMQ_GENERATE_REPORT_QUEUE": "extract-external-data-queue",
			"RABBITMQ_JOB_EVENTS_EXCHANGE":   "fetcher.job.events",
			"RABBITMQ_NUMBERS_OF_WORKERS":    "2",
			"SEAWEEDFS_TTL":                  "",
		}),
		WaitingFor: wait.ForLog("Starting consumer").WithStartupTimeout(WorkerStartupTimeout),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: workerReq,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	return container, nil
}
