package seaweedfs

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

// SeaweedFSOption is a functional option for configuring SeaweedFS.
type SeaweedFSOption func(*seaweedfsOptions)

type seaweedfsOptions struct {
	hostConfigModifiers []func(*container.HostConfig)
}

func defaultSeaweedFSOptions() *seaweedfsOptions {
	return &seaweedfsOptions{}
}

func WithSeaweedFSFixedPort(hostPort string) SeaweedFSOption {
	return func(o *seaweedfsOptions) {
		o.hostConfigModifiers = append(o.hostConfigModifiers, func(hc *container.HostConfig) {
			if hc.PortBindings == nil {
				hc.PortBindings = nat.PortMap{}
			}

			hc.PortBindings[nat.Port("8888/tcp")] = []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: hostPort},
			}
		})
	}
}
