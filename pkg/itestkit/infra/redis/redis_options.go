package redis

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

type RedisOption func(*redisOptions)

type redisOptions struct {
	runOpts []testcontainers.ContainerCustomizer
}

func defaultRedisOptions() *redisOptions {
	return &redisOptions{runOpts: []testcontainers.ContainerCustomizer{}}
}

func WithRedisImage(image string) RedisOption {
	return func(o *redisOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithImage(image))
	}
}

func WithRedisEnv(key, value string) RedisOption {
	return func(o *redisOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithEnv(map[string]string{key: value}))
	}
}

func WithRedisCommand(cmd ...string) RedisOption {
	return func(o *redisOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithCmd(cmd...))
	}
}

func WithRedisFixedPort(hostPort string) RedisOption {
	return func(o *redisOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithHostConfigModifier(
			func(hc *container.HostConfig) {
				if hc.PortBindings == nil {
					hc.PortBindings = nat.PortMap{}
				}

				hc.PortBindings[nat.Port("6379/tcp")] = []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: hostPort},
				}
			},
		))
	}
}
