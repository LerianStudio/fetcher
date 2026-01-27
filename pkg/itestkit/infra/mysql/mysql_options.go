package mysql

import "github.com/testcontainers/testcontainers-go"

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
					FileMode:          0755,
				},
			),
		)
	}
}
