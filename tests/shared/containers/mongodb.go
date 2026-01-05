package containers

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// MongoDBContainer wraps a MongoDB testcontainer with connection info.
type MongoDBContainer struct {
	Container    *mongodb.MongoDBContainer
	URI          string
	Host         string
	Port         string
	InternalHost string // Docker network hostname
}

// MongoDBOptions configures MongoDB container startup.
type MongoDBOptions struct {
	NetworkName   string
	NetworkAlias  string // e.g., "fetcher-mongodb" or "fetcher-mongodb-external"
	FixedHostPort string // Empty for random port, "27017" for fixed
	Username      string
	Password      string
	Database      string
}

// DefaultMongoDBMainOptions returns options for the main MongoDB (fetcher-db).
func DefaultMongoDBMainOptions(networkName string) MongoDBOptions {
	return MongoDBOptions{
		NetworkName:  networkName,
		NetworkAlias: "fetcher-mongodb",
		Username:     "root",
		Password:     "password",
		Database:     "fetcher_test",
	}
}

// DefaultMongoDBExternalOptions returns options for external MongoDB (test data).
func DefaultMongoDBExternalOptions(networkName string) MongoDBOptions {
	return MongoDBOptions{
		NetworkName:  networkName,
		NetworkAlias: "fetcher-mongodb-external",
		Username:     "root",
		Password:     "password",
		Database:     "external_transactions",
	}
}

// StartMongoDB starts a MongoDB container with the given options.
func StartMongoDB(ctx context.Context, opts MongoDBOptions) (*MongoDBContainer, error) {
	containerOpts := []testcontainers.ContainerCustomizer{
		mongodb.WithUsername(opts.Username),
		mongodb.WithPassword(opts.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Waiting for connections").WithStartupTimeout(config.MongoDBStartupTimeout),
		),
	}

	// Add network configuration
	if opts.NetworkName != "" {
		containerOpts = append(containerOpts,
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{opts.NetworkName},
					NetworkAliases: map[string][]string{opts.NetworkName: {opts.NetworkAlias}},
				},
			}),
		)
	}

	// Add fixed port if specified
	if opts.FixedHostPort != "" {
		containerOpts = append(containerOpts, WithFixedPort("27017/tcp", opts.FixedHostPort))
	}

	container, err := mongodb.Run(ctx, "mongo:7", containerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start MongoDB: %w", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MongoDB URI: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MongoDB host: %w", err)
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MongoDB port: %w", err)
	}

	return &MongoDBContainer{
		Container:    container,
		URI:          uri,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
	}, nil
}

// Stop terminates the MongoDB container.
func (m *MongoDBContainer) Stop(ctx context.Context) error {
	if m.Container != nil {
		return m.Container.Terminate(ctx)
	}
	return nil
}
