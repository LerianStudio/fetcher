package containers

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// RabbitMQContainer wraps a RabbitMQ testcontainer with connection info.
type RabbitMQContainer struct {
	Container    *rabbitmq.RabbitMQContainer
	URI          string
	Host         string
	Port         string
	InternalHost string
}

// RabbitMQOptions configures RabbitMQ container startup.
type RabbitMQOptions struct {
	NetworkName   string
	NetworkAlias  string
	FixedHostPort string
	Username      string
	Password      string
}

// DefaultRabbitMQOptions returns default RabbitMQ options.
func DefaultRabbitMQOptions(networkName string) RabbitMQOptions {
	return RabbitMQOptions{
		NetworkName:  networkName,
		NetworkAlias: "fetcher-rabbitmq",
		Username:     "guest",
		Password:     "guest",
	}
}

// StartRabbitMQ starts a RabbitMQ container with the given options.
func StartRabbitMQ(ctx context.Context, opts RabbitMQOptions) (*RabbitMQContainer, error) {
	containerOpts := []testcontainers.ContainerCustomizer{
		rabbitmq.WithAdminUsername(opts.Username),
		rabbitmq.WithAdminPassword(opts.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Server startup complete").WithStartupTimeout(config.RabbitMQStartupTimeout),
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
		containerOpts = append(containerOpts, WithFixedPort("5672/tcp", opts.FixedHostPort))
	}

	container, err := rabbitmq.Run(ctx, "rabbitmq:3-management", containerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start RabbitMQ: %w", err)
	}

	uri, err := container.AmqpURL(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get RabbitMQ URI: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get RabbitMQ host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5672")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get RabbitMQ port: %w", err)
	}

	return &RabbitMQContainer{
		Container:    container,
		URI:          uri,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
	}, nil
}

// Stop terminates the RabbitMQ container.
func (r *RabbitMQContainer) Stop(ctx context.Context) error {
	if r.Container != nil {
		return r.Container.Terminate(ctx)
	}
	return nil
}
