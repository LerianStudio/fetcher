package shared

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/itestkit"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/mongodb"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/rabbitmq"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/redis"
	"github.com/LerianStudio/fetcher/pkg/itestkit/infra/seaweedfs"
)

// CoreInfra holds the core infrastructure components required for E2E tests.
// It provides pre-configured instances of MongoDB, RabbitMQ, Redis, and SeaweedFS
// that mirror the production Fetcher architecture.
type CoreInfra struct {
	// MongoDB is the primary data store for connections and jobs.
	MongoDB *mongodb.MongoDBInfra
	// RabbitMQ is the message broker for job queue and event notifications.
	RabbitMQ *rabbitmq.RabbitInfra
	// Redis is used for caching and rate limiting.
	Redis *redis.RedisInfra
	// SeaweedFS is the distributed file system for storing extraction results.
	SeaweedFS *seaweedfs.SeaweedFSInfra
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

	return &CoreInfra{
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
}

// definitionsPath returns the absolute path to the RabbitMQ topology definitions file.
// The path is resolved relative to this source file to ensure correct resolution
// regardless of the working directory.
func definitionsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "definitions.json")
}

// Infras returns all infrastructure components as a slice of itestkit.Infra interfaces.
// This is used by the itestkit.Suite builder to start all infrastructure in the correct order.
func (c *CoreInfra) Infras() []itestkit.Infra {
	return []itestkit.Infra{c.MongoDB, c.RabbitMQ, c.Redis, c.SeaweedFS}
}
