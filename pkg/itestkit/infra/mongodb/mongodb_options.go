package mongodb

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

type MongoDBOption func(*mongodbOptions)

type mongodbOptions struct {
	runOpts []testcontainers.ContainerCustomizer
}

func defaultMongoDBOptions() *mongodbOptions {
	return &mongodbOptions{runOpts: []testcontainers.ContainerCustomizer{}}
}

func WithMongoDBImage(image string) MongoDBOption {
	return func(o *mongodbOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithImage(image))
	}
}

func WithMongoDBEnv(key, value string) MongoDBOption {
	return func(o *mongodbOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithEnv(map[string]string{key: value}))
	}
}

func WithMongoDBCommand(cmd ...string) MongoDBOption {
	return func(o *mongodbOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithCmd(cmd...))
	}
}

func WithMongoDBFixedPort(hostPort string) MongoDBOption {
	return func(o *mongodbOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithHostConfigModifier(
			func(hc *container.HostConfig) {
				if hc.PortBindings == nil {
					hc.PortBindings = nat.PortMap{}
				}

				hc.PortBindings[nat.Port("27017/tcp")] = []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: hostPort},
				}
			},
		))
	}
}
