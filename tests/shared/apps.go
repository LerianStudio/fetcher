package shared

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/itestkit/addons/e2ekit"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/mongodb"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/redis"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/seaweedfs"
)

// TestOrganizationID is the default organization UUID used in all E2E test requests.
// It provides a consistent tenant identifier across test scenarios.
const TestOrganizationID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

// AppStartConfig configures how to start a Fetcher application container (Manager or Worker).
type AppStartConfig struct {
	// Image is the Docker image name/tag to use when SkipBuild is true.
	Image string
	// SkipBuild determines whether to use a pre-built image (true) or build from Dockerfile (false).
	SkipBuild bool
}

// AppEnv holds the environment configuration needed to start Manager and Worker containers.
// It contains connection details for all infrastructure services. The application components
// expect individual host/port values rather than full connection URIs.
type AppEnv struct {
	// Network is the Docker network name that enables container-to-container communication.
	// When set, containers can reach each other using their network aliases.
	Network string

	// MongoDB connection configuration.
	MongoHost     string
	MongoPort     string
	MongoUser     string
	MongoPassword string

	// RabbitMQ connection configuration.
	RabbitHost     string
	RabbitPort     string
	RabbitUser     string
	RabbitPassword string

	// Redis connection configuration.
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// SeaweedFS connection configuration.
	SeaweedFSHost string
	SeaweedFSPort string
}

// BuildAppEnv constructs environment variables from infrastructure endpoints.
// The infra HostPort() methods return network aliases when running in a shared Docker network,
// enabling direct container-to-container communication.
// The network parameter should come from itestkit.Suite.Network().
func BuildAppEnv(network string, mongo *mongodb.MongoDBInfra, rabbit *rabbitmq.RabbitInfra, redisInfra *redis.RedisInfra, seaweed *seaweedfs.SeaweedFSInfra) (*AppEnv, error) {
	mongoHost, mongoPort, err := mongo.HostPort()
	if err != nil {
		return nil, fmt.Errorf("mongo host/port: %w", err)
	}

	rabbitHost, rabbitPort, err := rabbit.HostPort()
	if err != nil {
		return nil, fmt.Errorf("rabbit host/port: %w", err)
	}

	redisHost, redisPort, err := redisInfra.HostPort()
	if err != nil {
		return nil, fmt.Errorf("redis host/port: %w", err)
	}

	seaweedHost, seaweedPort, err := seaweed.HostPort()
	if err != nil {
		return nil, fmt.Errorf("seaweedfs host/port: %w", err)
	}

	return &AppEnv{
		Network:        network,
		MongoHost:      mongoHost,
		MongoPort:      strconv.Itoa(mongoPort),
		MongoUser:      CoreInfraUsername,
		MongoPassword:  CoreInfraPassword,
		RabbitHost:     rabbitHost,
		RabbitPort:     strconv.Itoa(rabbitPort),
		RabbitUser:     CoreInfraUsername,
		RabbitPassword: CoreInfraPassword,
		RedisHost:      redisHost,
		RedisPort:      strconv.Itoa(redisPort),
		RedisPassword:  CoreInfraPassword,
		SeaweedFSHost:  seaweedHost,
		SeaweedFSPort:  strconv.Itoa(seaweedPort),
	}, nil
}

// ManagerEnv returns the complete set of environment variables required by the Manager container.
// It includes MongoDB, RabbitMQ, and Redis configuration, plus encryption keys and logging settings.
// The returned map can be passed directly to e2ekit.Builder.WithEnv().
func (e *AppEnv) ManagerEnv() map[string]string {
	portStr := strconv.Itoa(ManagerAPIPort)

	return map[string]string{
		"ENV_NAME":                             "test",
		"SERVER_PORT":                          portStr,
		"SERVER_ADDRESS":                       ":" + portStr,
		"APP_ENC_KEY":                          "kV2RgskAt2gr+rtJmldM0gVEQNXduXXp3Le8VFCQKj8=",
		"APP_ENC_KEY_VERSION":                  "1",
		"MONGO_URI":                            "mongodb",
		"MONGO_HOST":                           e.MongoHost,
		"MONGO_PORT":                           e.MongoPort,
		"MONGO_USER":                           e.MongoUser,
		"MONGO_PASSWORD":                       e.MongoPassword,
		"MONGO_NAME":                           "fetcher-db",
		"MONGO_MAX_POOL_SIZE":                  "100",
		"RABBITMQ_URI":                         "amqp",
		"RABBITMQ_HOST":                        e.RabbitHost,
		"RABBITMQ_PORT_AMQP":                   e.RabbitPort,
		"RABBITMQ_DEFAULT_USER":                e.RabbitUser,
		"RABBITMQ_DEFAULT_PASS":                e.RabbitPassword,
		"RABBITMQ_FETCHER_WORK_QUEUE":          "fetcher.extract-external-data.queue",
		"REDIS_HOST":                           e.RedisHost,
		"REDIS_PORT":                           e.RedisPort,
		"REDIS_PASSWORD":                       e.RedisPassword,
		"REDIS_DB":                             "0",
		"LOG_LEVEL":                            "debug",
		"ENABLE_TELEMETRY":                     "true",
		"OTEL_RESOURCE_SERVICE_NAME":           "fetcher",
		"OTEL_LIBRARY_NAME":                    "github.com/LerianStudio/fetcher",
		"OTEL_RESOURCE_SERVICE_VERSION":        "v1.0.0",
		"OTEL_RESOURCE_DEPLOYMENT_ENVIRONMENT": "development",
		"OTEL_EXPORTER_OTLP_ENDPOINT_PORT":     "4317",
		"OTEL_EXPORTER_OTLP_ENDPOINT":          "host.docker.internal:4317",
	}
}

// WorkerEnv returns the complete set of environment variables required by the Worker container.
// It includes MongoDB, RabbitMQ, and SeaweedFS configuration, plus encryption keys for data storage.
// The returned map can be passed directly to e2ekit.Builder.WithEnv().
func (e *AppEnv) WorkerEnv() map[string]string {
	return map[string]string{
		"ENV_NAME":                             "test",
		"APP_ENC_KEY":                          "kV2RgskAt2gr+rtJmldM0gVEQNXduXXp3Le8VFCQKj8=",
		"APP_ENC_KEY_VERSION":                  "1",
		"MONGO_URI":                            "mongodb",
		"MONGO_HOST":                           e.MongoHost,
		"MONGO_PORT":                           e.MongoPort,
		"MONGO_USER":                           e.MongoUser,
		"MONGO_PASSWORD":                       e.MongoPassword,
		"MONGO_NAME":                           "fetcher-db",
		"MONGO_MAX_POOL_SIZE":                  "100",
		"RABBITMQ_URI":                         "amqp",
		"RABBITMQ_HOST":                        e.RabbitHost,
		"RABBITMQ_PORT_AMQP":                   e.RabbitPort,
		"RABBITMQ_DEFAULT_USER":                e.RabbitUser,
		"RABBITMQ_DEFAULT_PASS":                e.RabbitPassword,
		"RABBITMQ_FETCHER_WORK_QUEUE":          "fetcher.extract-external-data.queue",
		"RABBITMQ_JOB_EVENTS_EXCHANGE":         "fetcher.job.events",
		"RABBITMQ_NUMBERS_OF_WORKERS":          "1",
		"SEAWEEDFS_HOST":                       e.SeaweedFSHost,
		"SEAWEEDFS_FILER_PORT":                 e.SeaweedFSPort,
		"SEAWEEDFS_TTL":                        "24h",
		"LOG_LEVEL":                            "debug",
		"ENABLE_TELEMETRY":                     "true",
		"CRYPTO_ENCRYPT_SECRET_KEY_SEAWEEDFS":  "3132333435363738393031323334353637383930313233343536373839303132",
		"CRYPTO_HASH_SECRET_KEY_SEAWEEDFS":     "3132333435363738393031323334353637383930313233343536373839303132",
		"OTEL_RESOURCE_SERVICE_NAME":           "fetcher",
		"OTEL_LIBRARY_NAME":                    "github.com/LerianStudio/fetcher",
		"OTEL_RESOURCE_SERVICE_VERSION":        "v1.0.0",
		"OTEL_RESOURCE_DEPLOYMENT_ENVIRONMENT": "development",
		"OTEL_EXPORTER_OTLP_ENDPOINT_PORT":     "4317",
		"OTEL_EXPORTER_OTLP_ENDPOINT":          "host.docker.internal:4317",
	}
}

// StartManager starts the Manager HTTP API container using e2ekit.
//
// The function supports two modes based on AppStartConfig.SkipBuild:
//   - SkipBuild=true: Uses the pre-built image specified in cfg.Image
//   - SkipBuild=false: Builds the image from components/manager/Dockerfile using BuildKit secrets
//
// The container is configured with:
//   - Health check endpoint at /health for readiness detection
//   - Shared Docker network for container-to-container communication
//   - All required environment variables from AppEnv
//
// Parameters:
//   - t: Testing instance (can be nil when called from TestMain)
//   - ctx: Context for timeout control
//   - env: Environment configuration from BuildAppEnv
//   - cfg: Container start configuration
//
// Returns the running application with its base URL for API calls.
func StartManager(t *testing.T, ctx context.Context, env *AppEnv, cfg AppStartConfig) (*e2ekit.RunningApp, error) { //nolint:thelper // t can be nil when called from TestMain
	if t != nil {
		t.Helper()
	}

	builder := e2ekit.New(t).
		WithContext(ctx).
		ExposePort(ManagerAPIPort).
		WithEnv(env.ManagerEnv()).
		WithWait(e2ekit.WaitHTTP(ManagerAPIPort, "/health", 60*time.Second))

	// Add to shared network for container-to-container communication
	if env.Network != "" {
		builder = builder.WithNetworks(env.Network)
	}

	if cfg.SkipBuild {
		builder = builder.WithImage(cfg.Image)
	} else {
		builder = builder.WithDockerfile(e2ekit.BuildConfig{
			ContextDir: e2ekit.ProjectRoot(),
			Dockerfile: "components/manager/Dockerfile",
			Tag:        cfg.Image,
			Secrets: []e2ekit.BuildSecret{
				{ID: "github_token", Env: "GITHUB_TOKEN"},
			},
		})
	}

	return builder.Run()
}

// StartWorker starts the Worker message consumer container using e2ekit.
//
// The function supports two modes based on AppStartConfig.SkipBuild:
//   - SkipBuild=true: Uses the pre-built image specified in cfg.Image
//   - SkipBuild=false: Builds the image from components/worker/Dockerfile using BuildKit secrets
//
// The container is configured with:
//   - Log-based readiness detection (waits for "Starting consumer for queue" message)
//   - Shared Docker network for container-to-container communication
//   - All required environment variables from AppEnv
//
// Parameters:
//   - t: Testing instance (can be nil when called from TestMain)
//   - ctx: Context for timeout control
//   - env: Environment configuration from BuildAppEnv
//   - cfg: Container start configuration
//
// Returns the running application container.
func StartWorker(t *testing.T, ctx context.Context, env *AppEnv, cfg AppStartConfig) (*e2ekit.RunningApp, error) { //nolint:thelper // t can be nil when called from TestMain
	if t != nil {
		t.Helper()
	}

	builder := e2ekit.New(t).
		WithContext(ctx).
		WithEnv(env.WorkerEnv()).
		WithWait(e2ekit.WaitLog("Starting consumer for queue", 60*time.Second))

	// Add to shared network for container-to-container communication
	if env.Network != "" {
		builder = builder.WithNetworks(env.Network)
	}

	if cfg.SkipBuild {
		builder = builder.WithImage(cfg.Image)
	} else {
		builder = builder.WithDockerfile(e2ekit.BuildConfig{
			ContextDir: e2ekit.ProjectRoot(),
			Dockerfile: "components/worker/Dockerfile",
			Tag:        cfg.Image,
			Secrets: []e2ekit.BuildSecret{
				{ID: "github_token", Env: "GITHUB_TOKEN"},
			},
		})
	}

	return builder.Run()
}

// NewClientFromApp creates a ManagerClient configured to communicate with a running Manager container.
// It extracts the base URL from the running app and uses the default TestOrganizationID.
// This is the preferred way to create a client after starting the Manager with StartManager.
func NewClientFromApp(app *e2ekit.RunningApp) *ManagerClient {
	return NewManagerClient(app.BaseURL, TestOrganizationID)
}
