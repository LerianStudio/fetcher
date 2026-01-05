package containers

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

// WithFixedPort creates a port binding modifier for fixed host ports.
// This is used in debug mode to ensure consistent ports for VS Code debugging.
func WithFixedPort(containerPort, hostPort string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		port := nat.Port(containerPort)
		if req.HostConfigModifier == nil {
			req.HostConfigModifier = func(hostConfig *container.HostConfig) {
				if hostConfig.PortBindings == nil {
					hostConfig.PortBindings = nat.PortMap{}
				}
				hostConfig.PortBindings[port] = []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: hostPort},
				}
			}
		} else {
			original := req.HostConfigModifier
			req.HostConfigModifier = func(hostConfig *container.HostConfig) {
				original(hostConfig)
				if hostConfig.PortBindings == nil {
					hostConfig.PortBindings = nat.PortMap{}
				}
				hostConfig.PortBindings[port] = []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: hostPort},
				}
			}
		}
		return nil
	}
}
