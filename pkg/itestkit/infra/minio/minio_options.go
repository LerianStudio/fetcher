package minio

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

// MinioOption is a functional option for configuring MinIO infrastructure.
type MinioOption func(*minioOptions)

type minioOptions struct {
	image               string
	hostConfigModifiers []func(*container.HostConfig)
}

func defaultMinioOptions(image string) *minioOptions {
	return &minioOptions{
		image:               image,
		hostConfigModifiers: []func(*container.HostConfig){},
	}
}

// WithMinioImage overrides the Docker image used for the MinIO container.
func WithMinioImage(image string) MinioOption {
	return func(o *minioOptions) {
		o.image = image
	}
}

// WithMinioFixedPort binds the MinIO S3 API port (9000) to a specific host port.
// Useful for local debugging and infrastructure-only mode.
func WithMinioFixedPort(hostPort string) MinioOption {
	return func(o *minioOptions) {
		o.hostConfigModifiers = append(o.hostConfigModifiers, func(hc *container.HostConfig) {
			if hc.PortBindings == nil {
				hc.PortBindings = nat.PortMap{}
			}

			hc.PortBindings[nat.Port("9000/tcp")] = []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: hostPort},
			}
		})
	}
}
