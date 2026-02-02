package mssql

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

type MSSQLOption func(*mssqlOptions)

type mssqlOptions struct {
	runOpts []testcontainers.ContainerCustomizer
}

func defaultMSSQLOptions() *mssqlOptions {
	return &mssqlOptions{runOpts: []testcontainers.ContainerCustomizer{}}
}

func WithMSSQLImage(image string) MSSQLOption {
	return func(o *mssqlOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithImage(image))
	}
}

func WithMSSQLEnv(key, value string) MSSQLOption {
	return func(o *mssqlOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithEnv(map[string]string{key: value}))
	}
}

// WithMSSQLFixedPort binds the SQL Server container to a specific host port.
// Use this for debugging scenarios where the local app needs to connect
// to the containerized database on a predictable port.
func WithMSSQLFixedPort(hostPort string) MSSQLOption {
	return func(o *mssqlOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithHostConfigModifier(
			func(hc *container.HostConfig) {
				if hc.PortBindings == nil {
					hc.PortBindings = nat.PortMap{}
				}
				hc.PortBindings["1433/tcp"] = []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: hostPort},
				}
			},
		))
	}
}
