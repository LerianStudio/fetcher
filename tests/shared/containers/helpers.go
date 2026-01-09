package containers

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

// WithFixedPort creates a port binding modifier for fixed host ports.
// This is used in debug mode to ensure consistent ports for VS Code debugging.
//
// applyPortBinding is a helper to set up fixed port bindings
func applyPortBinding(hostConfig *container.HostConfig, port nat.Port, hostPort string) {
	if hostConfig.PortBindings == nil {
		hostConfig.PortBindings = nat.PortMap{}
	}

	hostConfig.PortBindings[port] = []nat.PortBinding{
		{HostIP: "0.0.0.0", HostPort: hostPort},
	}
}

func WithFixedPort(containerPort, hostPort string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		port := nat.Port(containerPort)

		if req.HostConfigModifier == nil {
			req.HostConfigModifier = func(hostConfig *container.HostConfig) {
				applyPortBinding(hostConfig, port, hostPort)
			}
		} else {
			original := req.HostConfigModifier
			req.HostConfigModifier = func(hostConfig *container.HostConfig) {
				original(hostConfig)
				applyPortBinding(hostConfig, port, hostPort)
			}
		}

		return nil
	}
}
