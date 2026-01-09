package containers

import (
	"context"
	"fmt"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// MongoDBContainer wraps a MongoDB testcontainer with connection info.
type MongoDBContainer struct {
	Container    *mongodb.MongoDBContainer
	URI          string
	Host         string
	Port         string
	InternalHost string // Docker network hostname
	Internal     config.InternalDBConnection
	SSL          *SSLConnectionInfo
}

// MongoDBOptions configures MongoDB container startup.
type MongoDBOptions struct {
	NetworkName   string
	NetworkAlias  string // e.g., "fetcher-mongodb" or "fetcher-mongodb-external"
	FixedHostPort string // Empty for random port, "27017" for fixed
	Username      string
	Password      string
	Database      string
	SSL           *SSLConfig
}

// DefaultMongoDBMainOptions returns options for the main MongoDB (fetcher-db).
func DefaultMongoDBMainOptions(networkName string) MongoDBOptions {
	return MongoDBOptions{
		NetworkName:  networkName,
		NetworkAlias: "fetcher-mongodb",
		Username:     "root",
		Password:     "password",
		Database:     "fetcher_test",
	}
}

// DefaultMongoDBExternalOptions returns options for external MongoDB (test data).
func DefaultMongoDBExternalOptions(networkName string) MongoDBOptions {
	return MongoDBOptions{
		NetworkName:  networkName,
		NetworkAlias: "fetcher-mongodb-external",
		Username:     "root",
		Password:     "password",
		Database:     "external_transactions",
	}
}

// DefaultMongoDBSSLOptions returns MongoDB options with SSL enabled.
func DefaultMongoDBSSLOptions(networkName string) MongoDBOptions {
	opts := DefaultMongoDBExternalOptions(networkName)
	opts.NetworkAlias = "fetcher-mongodb-external-ssl"
	opts.SSL = &SSLConfig{
		Enabled: true,
		Mode:    "true",
	}

	return opts
}

// StartMongoDB starts a MongoDB container with the given options.
func StartMongoDB(ctx context.Context, opts MongoDBOptions) (*MongoDBContainer, error) {
	containerOpts := []testcontainers.ContainerCustomizer{
		mongodb.WithUsername(opts.Username),
		mongodb.WithPassword(opts.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Waiting for connections").WithStartupTimeout(config.MongoDBStartupTimeout),
		),
	}

	// Add SSL configuration if enabled
	// MongoDB 7 in testcontainers requires careful TLS setup to preserve
	// the original entrypoint that handles authentication initialization.
	// We use a wrapper script that sets up certs then calls the original entrypoint.
	if opts.SSL != nil && opts.SSL.Enabled && opts.SSL.CertBundle != nil {
		// MongoDB requires a combined PEM file with cert + key
		combinedPEM := opts.SSL.CertBundle.ServerCertPEM + opts.SSL.CertBundle.ServerKeyPEM
		// Wrapper script that sets up certs and then calls original entrypoint
		// with TLS options appended to the command.
		// Note: MongoDB runs as user 'mongodb' (uid 999), so we need to ensure
		// the certificate files are readable by that user.
		sslWrapperScript := `#!/bin/bash
set -e
# Setup SSL certificates before MongoDB starts
mkdir -p /etc/ssl/mongodb
echo "$SSL_COMBINED_PEM" > /etc/ssl/mongodb/server.pem
echo "$SSL_CA_CERT" > /etc/ssl/mongodb/ca.pem
# Set permissions - mongodb user (uid 999) needs to read these files
# The key file needs 600 for security, but must be owned by mongodb user
chmod 600 /etc/ssl/mongodb/server.pem
chmod 644 /etc/ssl/mongodb/ca.pem
chown -R mongodb:mongodb /etc/ssl/mongodb
# Call original docker-entrypoint.sh with TLS options
# Using preferTLS mode allows both TLS and non-TLS connections
exec /usr/local/bin/docker-entrypoint.sh mongod \
  --tlsMode preferTLS \
  --tlsCertificateKeyFile /etc/ssl/mongodb/server.pem \
  --tlsCAFile /etc/ssl/mongodb/ca.pem \
  --tlsAllowConnectionsWithoutCertificates
`
		containerOpts = append(containerOpts,
			testcontainers.WithEnv(map[string]string{
				"SSL_COMBINED_PEM": combinedPEM,
				"SSL_CA_CERT":      opts.SSL.CertBundle.CACertPEM,
			}),
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Files: []testcontainers.ContainerFile{
						{
							ContainerFilePath: "/ssl-wrapper.sh",
							Reader:            strings.NewReader(sslWrapperScript),
							FileMode:          0755,
						},
					},
					Entrypoint: []string{"/ssl-wrapper.sh"},
					Cmd:        []string{},
				},
			}),
		)
	}

	// Add network configuration
	if opts.NetworkName != "" {
		containerOpts = append(containerOpts,
			testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
				ContainerRequest: testcontainers.ContainerRequest{
					Networks:       []string{opts.NetworkName},
					NetworkAliases: map[string][]string{opts.NetworkName: {opts.NetworkAlias}},
				},
			}),
		)
	}

	// Add fixed port if specified
	if opts.FixedHostPort != "" {
		containerOpts = append(containerOpts, WithFixedPort("27017/tcp", opts.FixedHostPort))
	}

	container, err := mongodb.Run(ctx, "mongo:7", containerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start MongoDB: %w", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MongoDB URI: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MongoDB host: %w", err)
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get MongoDB port: %w", err)
	}

	// Build internal connection with SSL info
	internal := config.InternalDBConnection{
		Host:     opts.NetworkAlias,
		Port:     27017,
		Username: opts.Username,
		Password: opts.Password,
		Database: opts.Database,
	}

	// Populate SSL connection info
	var sslConnInfo *SSLConnectionInfo

	if opts.SSL != nil && opts.SSL.Enabled {
		internal.SSLEnabled = true

		internal.SSLMode = opts.SSL.Mode
		if opts.SSL.CertBundle != nil {
			internal.SSLCACert = opts.SSL.CertBundle.CACertPEM
			internal.SSLClientCert = opts.SSL.CertBundle.ClientCertPEM
			internal.SSLClientKey = opts.SSL.CertBundle.ClientKeyPEM
		}

		sslConnInfo = opts.SSL.ToConnectionInfo()
	}

	return &MongoDBContainer{
		Container:    container,
		URI:          uri,
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.NetworkAlias,
		Internal:     internal,
		SSL:          sslConnInfo,
	}, nil
}

// Stop terminates the MongoDB container.
func (m *MongoDBContainer) Stop(ctx context.Context) error {
	if m.Container != nil {
		return m.Container.Terminate(ctx)
	}

	return nil
}
