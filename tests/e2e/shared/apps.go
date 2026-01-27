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

const TestOrganizationID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

// AppStartConfig configures how to start an application container.
type AppStartConfig struct {
	Image     string // Docker image name (used when SkipBuild is true)
	SkipBuild bool   // If true, use pre-built image; if false, build from Dockerfile
}

// AppEnv holds environment configuration for Manager/Worker containers.
// The application expects individual components rather than full URIs.
type AppEnv struct {
	// MongoDB
	MongoHost     string
	MongoPort     string
	MongoUser     string
	MongoPassword string

	// RabbitMQ
	RabbitHost     string
	RabbitPort     string
	RabbitUser     string
	RabbitPassword string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// SeaweedFS
	SeaweedFSHost string
	SeaweedFSPort string
}

// BuildAppEnv constructs environment variables from infrastructure endpoints.
// The infra HostPort() methods automatically normalize localhost to the Docker gateway IP,
// so containers can reach the infrastructure services.
func BuildAppEnv(mongo *mongodb.MongoDBInfra, rabbit *rabbitmq.RabbitInfra, redisInfra *redis.RedisInfra, seaweed *seaweedfs.SeaweedFSInfra) (*AppEnv, error) {
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
		MongoHost:      mongoHost,
		MongoPort:      strconv.Itoa(mongoPort),
		MongoUser:      "plugin",
		MongoPassword:  "Lerian@123",
		RabbitHost:     rabbitHost,
		RabbitPort:     strconv.Itoa(rabbitPort),
		RabbitUser:     "plugin",
		RabbitPassword: "Lerian@123",
		RedisHost:      redisHost,
		RedisPort:      strconv.Itoa(redisPort),
		RedisPassword:  "Lerian@123",
		SeaweedFSHost:  seaweedHost,
		SeaweedFSPort:  strconv.Itoa(seaweedPort),
	}, nil
}

// ManagerEnv returns environment variables for the Manager container.
func (e *AppEnv) ManagerEnv() map[string]string {
	return map[string]string{
		"ENV_NAME":                    "test",
		"SERVER_PORT":                 "4006",
		"SERVER_ADDRESS":              ":4006",
		"APP_ENC_KEY":                 "kV2RgskAt2gr+rtJmldM0gVEQNXduXXp3Le8VFCQKj8=",
		"APP_ENC_KEY_VERSION":         "1",
		"MONGO_URI":                   "mongodb",
		"MONGO_HOST":                  e.MongoHost,
		"MONGO_PORT":                  e.MongoPort,
		"MONGO_USER":                  e.MongoUser,
		"MONGO_PASSWORD":              e.MongoPassword,
		"MONGO_NAME":                  "fetcher-db",
		"MONGO_MAX_POOL_SIZE":         "100",
		"RABBITMQ_URI":                "amqp",
		"RABBITMQ_HOST":               e.RabbitHost,
		"RABBITMQ_PORT_AMQP":          e.RabbitPort,
		"RABBITMQ_DEFAULT_USER":       e.RabbitUser,
		"RABBITMQ_DEFAULT_PASS":       e.RabbitPassword,
		"RABBITMQ_FETCHER_WORK_QUEUE": "fetcher.extract-external-data.queue",
		"REDIS_HOST":                  e.RedisHost,
		"REDIS_PORT":                  e.RedisPort,
		"REDIS_PASSWORD":              e.RedisPassword,
		"REDIS_DB":                    "0",
		"LOG_LEVEL":                   "debug",
		"ENABLE_TELEMETRY":            "false",
	}
}

// WorkerEnv returns environment variables for the Worker container.
func (e *AppEnv) WorkerEnv() map[string]string {
	return map[string]string{
		"ENV_NAME":                            "test",
		"APP_ENC_KEY":                         "kV2RgskAt2gr+rtJmldM0gVEQNXduXXp3Le8VFCQKj8=",
		"APP_ENC_KEY_VERSION":                 "1",
		"MONGO_URI":                           "mongodb",
		"MONGO_HOST":                          e.MongoHost,
		"MONGO_PORT":                          e.MongoPort,
		"MONGO_USER":                          e.MongoUser,
		"MONGO_PASSWORD":                      e.MongoPassword,
		"MONGO_NAME":                          "fetcher-db",
		"MONGO_MAX_POOL_SIZE":                 "100",
		"RABBITMQ_URI":                        "amqp",
		"RABBITMQ_HOST":                       e.RabbitHost,
		"RABBITMQ_PORT_AMQP":                  e.RabbitPort,
		"RABBITMQ_DEFAULT_USER":               e.RabbitUser,
		"RABBITMQ_DEFAULT_PASS":               e.RabbitPassword,
		"RABBITMQ_FETCHER_WORK_QUEUE":         "fetcher.extract-external-data.queue",
		"RABBITMQ_JOB_EVENTS_EXCHANGE":        "fetcher.job.events",
		"RABBITMQ_NUMBERS_OF_WORKERS":         "1",
		"SEAWEEDFS_HOST":                      e.SeaweedFSHost,
		"SEAWEEDFS_FILER_PORT":                e.SeaweedFSPort,
		"SEAWEEDFS_TTL":                       "24h",
		"LOG_LEVEL":                           "debug",
		"ENABLE_TELEMETRY":                    "false",
		"CRYPTO_ENCRYPT_SECRET_KEY_SEAWEEDFS": "3132333435363738393031323334353637383930313233343536373839303132",
		"CRYPTO_HASH_SECRET_KEY_SEAWEEDFS":    "3132333435363738393031323334353637383930313233343536373839303132",
	}
}

// StartManager starts the Manager container using e2ekit.
// When SkipBuild is false, builds the image using Dockerfile with BuildKit secrets.
func StartManager(t *testing.T, ctx context.Context, env *AppEnv, cfg AppStartConfig) (*e2ekit.RunningApp, error) {
	builder := e2ekit.New(t).
		WithContext(ctx).
		ExposePort(4006).
		WithEnv(env.ManagerEnv()).
		WithWait(e2ekit.WaitHTTP(4006, "/health", 60*time.Second))

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

// StartWorker starts the Worker container using e2ekit.
// When SkipBuild is false, builds the image using Dockerfile with BuildKit secrets.
func StartWorker(t *testing.T, ctx context.Context, env *AppEnv, cfg AppStartConfig) (*e2ekit.RunningApp, error) {
	builder := e2ekit.New(t).
		WithContext(ctx).
		WithEnv(env.WorkerEnv()).
		WithWait(e2ekit.WaitLog("Starting consumer for queue", 60*time.Second))

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

// NewClientFromApp creates a ManagerClient from a running e2ekit app.
func NewClientFromApp(app *e2ekit.RunningApp) *ManagerClient {
	return NewManagerClient(app.BaseURL, TestOrganizationID)
}
