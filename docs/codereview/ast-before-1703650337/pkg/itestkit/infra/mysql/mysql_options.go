package mysql

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

type MySQLOption func(*mysqlOptions)

type mysqlOptions struct {
	runOpts []testcontainers.ContainerCustomizer
}

func defaultMySQLOptions() *mysqlOptions {
	return &mysqlOptions{runOpts: []testcontainers.ContainerCustomizer{}}
}

func WithMySQLImage(image string) MySQLOption {
	return func(o *mysqlOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithImage(image))
	}
}

func WithMySQLEnv(key, value string) MySQLOption {
	return func(o *mysqlOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithEnv(map[string]string{key: value}))
	}
}

func WithMySQLInitScript(hostPath, containerFileName string) MySQLOption {
	return func(o *mysqlOptions) {
		if containerFileName == "" {
			containerFileName = "init.sql"
		}
		o.runOpts = append(o.runOpts,
			testcontainers.WithFiles(
				testcontainers.ContainerFile{
					HostFilePath:      hostPath,
					ContainerFilePath: "/docker-entrypoint-initdb.d/" + containerFileName,
					FileMode:          0o755,
				},
			),
		)
	}
}

// WithMySQLFixedPort binds the MySQL container to a specific host port.
// Use this for debugging scenarios where the local app needs to connect
// to the containerized database on a predictable port.
func WithMySQLFixedPort(hostPort string) MySQLOption {
	return func(o *mysqlOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithHostConfigModifier(
			func(hc *container.HostConfig) {
				if hc.PortBindings == nil {
					hc.PortBindings = nat.PortMap{}
				}
				hc.PortBindings["3306/tcp"] = []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: hostPort},
				}
			},
		))
	}
}
