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

// CoreInfra holds infrastructure components shared across E2E tests.
type CoreInfra struct {
	MongoDB   *mongodb.MongoDBInfra
	RabbitMQ  *rabbitmq.RabbitInfra
	Redis     *redis.RedisInfra
	SeaweedFS *seaweedfs.SeaweedFSInfra
}

// isFixedPortEnabled returns true if FIXED_PORT env var is "true".
func isFixedPortEnabled() bool {
	if val := os.Getenv("FIXED_PORT"); val != "" {
		return strings.EqualFold(val, "true")
	}
	return false
}

// NewCoreInfra creates the core infrastructure configuration.
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
			Username: "plugin",
			Password: "Lerian@123",
			Options:  mongoOpts,
		}),
		RabbitMQ: rabbitmq.NewRabbitInfra(rabbitmq.RabbitConfig{
			Name:     "fetcher",
			Username: "plugin",
			Password: "Lerian@123",
			Options:  rabbitOpts,
		}),
		Redis: redis.NewRedisInfra(redis.RedisConfig{
			Name:     "fetcher",
			Password: "Lerian@123",
			Options:  redisOpts,
		}),
		SeaweedFS: seaweedfs.NewSeaweedFSInfra(seaweedfs.SeaweedFSConfig{
			Name:    "fetcher",
			Options: seaweedOpts,
		}),
	}
}

func definitionsPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "definitions.json")
}

// Infras returns all infrastructure components as a slice.
func (c *CoreInfra) Infras() []itestkit.Infra {
	return []itestkit.Infra{c.MongoDB, c.RabbitMQ, c.Redis, c.SeaweedFS}
}
