package mssql

import "github.com/testcontainers/testcontainers-go"

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
