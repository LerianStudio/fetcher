package shared

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/LerianStudio/fetcher/v2/pkg/itestkit"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/minio"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/mongodb"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/rabbitmq"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/redis"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/seaweedfs"
)

// CoreInfra holds the core infrastructure components required for E2E tests.
// It provides pre-configured instances of MongoDB, RabbitMQ, Redis, and SeaweedFS
// that mirror the production Fetcher architecture.
//
// When E2E_ENABLE_S3=true, Minio is also populated and the Worker is configured
// to use the S3 storage provider instead of SeaweedFS.
type CoreInfra struct {
	// MongoDB is the primary data store for connections and jobs.
	MongoDB *mongodb.MongoDBInfra
	// RabbitMQ is the message broker for job queue and event notifications.
	RabbitMQ *rabbitmq.RabbitInfra
	// Redis is used for caching and rate limiting.
	Redis *redis.RedisInfra
	// SeaweedFS is the distributed file system for storing extraction results.
	// Used when E2E_ENABLE_S3 is not set.
	SeaweedFS *seaweedfs.SeaweedFSInfra
	// Minio is the S3-compatible object storage for extraction results.
	// Non-nil only when E2E_ENABLE_S3=true.
	Minio *minio.MinioInfra
}

// isFixedPortEnabled checks if fixed port mode is enabled via the FIXED_PORT environment variable.
// When enabled, infrastructure containers use predetermined ports instead of dynamic allocation.
// This is useful for local development and debugging with external tools.
func isFixedPortEnabled() bool {
	if val := os.Getenv("FIXED_PORT"); val != "" {
		return strings.EqualFold(val, "true")
	}

	return false
}

// isS3Enabled checks if S3 storage testing is enabled via the E2E_ENABLE_S3 environment variable.
// When enabled, a MinIO container is started and the Worker is configured to use
// the S3 storage provider instead of SeaweedFS.
func isS3Enabled() bool {
	return strings.EqualFold(os.Getenv("E2E_ENABLE_S3"), "true")
}

// NewCoreInfra creates and returns a new CoreInfra with all infrastructure components configured.
// It uses the shared credentials from constants and optionally applies fixed ports if FIXED_PORT=true.
// The RabbitMQ instance is pre-configured with the topology defined in testdata/definitions.json.
func NewCoreInfra() *CoreInfra {
	mongoOpts := []mongodb.MongoDBOption{}
	rabbitOpts := []rabbitmq.RabbitOption{
		rabbitmq.WithRabbitDefinitions(definitionsPath()),
	}
	redisOpts := []redis.RedisOption{}
	seaweedOpts := []seaweedfs.SeaweedFSOption{}

	if isFixedPortEnabled() {
		mongoOpts = append(mongoOpts, mongodb.WithMongoDBFixedPort("5709"))
		rabbitOpts = append(rabbitOpts, rabbitmq.WithRabbitFixedPort("3008"))
		redisOpts = append(redisOpts, redis.WithRedisFixedPort("5707"))
		seaweedOpts = append(seaweedOpts, seaweedfs.WithSeaweedFSFixedPort("8889"))
	}

	infra := &CoreInfra{
		MongoDB: mongodb.NewMongoDBInfra(mongodb.MongoDBConfig{
			Name:     "fetcher",
			Username: CoreInfraUsername,
			Password: CoreInfraPassword,
			Options:  mongoOpts,
		}),
		RabbitMQ: rabbitmq.NewRabbitInfra(rabbitmq.RabbitConfig{
			Name:     "fetcher",
			Username: CoreInfraUsername,
			Password: CoreInfraPassword,
			Options:  rabbitOpts,
		}),
		Redis: redis.NewRedisInfra(redis.RedisConfig{
			Name:     "fetcher",
			Password: CoreInfraPassword,
			Options:  redisOpts,
		}),
		SeaweedFS: seaweedfs.NewSeaweedFSInfra(seaweedfs.SeaweedFSConfig{
			Name:    "fetcher",
			Options: seaweedOpts,
		}),
	}

	if isS3Enabled() {
		minioOpts := []minio.MinioOption{}
		if isFixedPortEnabled() {
			minioOpts = append(minioOpts, minio.WithMinioFixedPort("9000"))
		}

		infra.Minio = minio.NewMinioInfra(minio.MinioConfig{
			Name:    "fetcher",
			Options: minioOpts,
		})
	}

	return infra
}

// callerDir returns the directory of the calling source file.
// It wraps runtime.Caller to avoid triggering the dogsled linter
// for multiple blank identifiers.
//
// IMPORTANT: This function assumes exactly 1 frame of indirection (called
// from definitionsPath only). If the call chain changes, the skip=1 value
// must be updated accordingly.
func callerDir() string {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "."
	}

	return filepath.Dir(file)
}

// definitionsPath returns the absolute path to the RabbitMQ topology definitions file.
// The path is resolved relative to this source file to ensure correct resolution
// regardless of the working directory.
func definitionsPath() string {
	return filepath.Join(callerDir(), "testdata", "definitions.json")
}

// Infras returns all infrastructure components as a slice of itestkit.Infra interfaces.
// This is used by the itestkit.Suite builder to start all infrastructure in the correct order.
// MinIO is included when E2E_ENABLE_S3=true (i.e. when c.Minio is non-nil).
func (c *CoreInfra) Infras() []itestkit.Infra {
	infras := []itestkit.Infra{c.MongoDB, c.RabbitMQ, c.Redis, c.SeaweedFS}
	if c.Minio != nil {
		infras = append(infras, c.Minio)
	}

	return infras
}
