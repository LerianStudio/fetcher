package containers

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// RedisContainer wraps a Redis testcontainer with connection info.
type RedisContainer struct {
	Container    *redis.RedisContainer
	URL          string
	Host         string
	Port         string
	InternalHost string
}

// RedisOptions configures Redis container startup.
type RedisOptions struct {
	NetworkName   string
	NetworkAlias  string
	FixedHostPort string
}

// DefaultRedisOptions returns default Redis options.
func DefaultRedisOptions(networkName string) RedisOptions {
	return RedisOptions{
		NetworkName:  networkName,
		NetworkAlias: "fetcher-valkey",
	}
}

// StartRedis starts a Redis/Valkey container with the given options.
func StartRedis(ctx context.Context, opts RedisOptions) (*RedisContainer, error) {
	containerOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").WithStartupTimeout(config.RedisStartupTimeout),
		),
	}

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

	if opts.FixedHostPort != "" {
		containerOpts = append(containerOpts, WithFixedPort("6379/tcp", opts.FixedHostPort))
	}

	container, err := redis.Run(ctx, "valkey/valkey:8", containerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start Redis: %w", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Redis URI: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Redis host: %w", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get Redis port: %w", err)
	}

	return &RedisContainer{
		Container:    container,
		URL:          uri,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
	}, nil
}

// Stop terminates the Redis container.
func (r *RedisContainer) Stop(ctx context.Context) error {
	if r.Container != nil {
		return r.Container.Terminate(ctx)
	}

	return nil
}

// GetHost returns the container's host address.
func (r *RedisContainer) GetHost() string { return r.Host }

// GetPort returns the container's mapped port.
func (r *RedisContainer) GetPort() string { return r.Port }

// GetURI returns the container's connection URL.
func (r *RedisContainer) GetURI() string { return r.URL }
