package oracle

import "github.com/testcontainers/testcontainers-go"

type OracleOption func(*oracleOptions)

type oracleOptions struct {
	runOpts []testcontainers.ContainerCustomizer
}

func defaultOracleOptions() *oracleOptions {
	return &oracleOptions{runOpts: []testcontainers.ContainerCustomizer{}}
}

func WithOracleImage(image string) OracleOption {
	return func(o *oracleOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithImage(image))
	}
}

func WithOracleEnv(key, value string) OracleOption {
	return func(o *oracleOptions) {
		o.runOpts = append(o.runOpts, testcontainers.WithEnv(map[string]string{key: value}))
	}
}

func WithOracleInitScript(hostPath, containerFileName string) OracleOption {
	return func(o *oracleOptions) {
		if containerFileName == "" {
			containerFileName = "init.sql"
		}
		o.runOpts = append(o.runOpts,
			testcontainers.WithFiles(
				testcontainers.ContainerFile{
					HostFilePath:      hostPath,
					ContainerFilePath: "/container-entrypoint-initdb.d/" + containerFileName,
					FileMode:          0755,
				},
			),
		)
	}
}
