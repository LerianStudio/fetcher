package rabbitmq

import (
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

type RabbitOption func(*rabbitOptions)

type rabbitOptions struct {
	runOpts []testcontainers.ContainerCustomizer
}

func defaultRabbitOptions() *rabbitOptions {
	return &rabbitOptions{runOpts: []testcontainers.ContainerCustomizer{}}
}

func WithRabbitImage(image string) RabbitOption {
	return func(o *rabbitOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithImage(image))
	}
}

func WithRabbitEnv(key, value string) RabbitOption {
	return func(o *rabbitOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithEnv(map[string]string{key: value}))
	}
}

func WithRabbitCommand(cmd ...string) RabbitOption {
	return func(o *rabbitOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithCmd(cmd...))
	}
}

func WithRabbitFixedPort(hostPort string) RabbitOption {
	return func(o *rabbitOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithHostConfigModifier(
			func(hc *container.HostConfig) {
				if hc.PortBindings == nil {
					hc.PortBindings = nat.PortMap{}
				}

				hc.PortBindings[nat.Port("5672/tcp")] = []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: hostPort},
				}
			},
		))
	}
}

func WithRabbitDefinitions(hostPath string) RabbitOption {
	return func(o *rabbitOptions) {
		o.runOpts = append(o.runOpts,
			testcontainers.WithFiles(
				testcontainers.ContainerFile{
					HostFilePath:      hostPath,
					ContainerFilePath: "/etc/rabbitmq/definitions.json",
					FileMode:          0644,
				},
			),
			testcontainers.WithFiles(
				testcontainers.ContainerFile{
					Reader:            rabbitConfReader(),
					ContainerFilePath: "/etc/rabbitmq/conf.d/20-definitions.conf",
					FileMode:          0644,
				},
			),
		)
	}
}

func rabbitConfReader() *configReader {
	return &configReader{content: "management.load_definitions = /etc/rabbitmq/definitions.json\n"}
}

type configReader struct {
	content string
	offset  int
}

func (r *configReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.content) {
		return 0, io.EOF
	}

	n = copy(p, r.content[r.offset:])
	r.offset += n

	return n, nil
}
