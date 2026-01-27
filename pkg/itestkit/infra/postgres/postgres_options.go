package postgres

import "github.com/testcontainers/testcontainers-go"

type PostgresOption func(*postgresOptions)

type postgresOptions struct {
	runOpts []testcontainers.ContainerCustomizer
}

func defaultPostgresOptions() *postgresOptions {
	return &postgresOptions{runOpts: []testcontainers.ContainerCustomizer{}}
}

func WithPGImage(image string) PostgresOption {
	return func(o *postgresOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithImage(image))
	}
}

func WithPGEnv(key, value string) PostgresOption {
	return func(o *postgresOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithEnv(map[string]string{key: value}))
	}
}

func WithPGCommand(cmd ...string) PostgresOption {
	return func(o *postgresOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithCmd(cmd...))
	}
}

func WithPGInitFile(hostPath string, containerFileName string) PostgresOption {
	return func(o *postgresOptions) {
		if containerFileName == "" {
			containerFileName = "init.sql"
		}
		o.runOpts = append(o.runOpts,
			testcontainers.WithFiles(
				testcontainers.ContainerFile{
					HostFilePath:      hostPath,
					ContainerFilePath: "/docker-entrypoint-initdb.d/" + containerFileName,
					FileMode:          0755,
				},
			),
		)
	}
}
