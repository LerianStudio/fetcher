// Package setup provides infrastructure orchestration for chaos tests.
// It composes the shared test infrastructure with Toxiproxy for network chaos injection.
package setup

import (
	"context"
	"fmt"
	"strings"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"

	"github.com/LerianStudio/fetcher/tests/shared/config"
	"github.com/LerianStudio/fetcher/tests/shared/containers"
	sharedsetup "github.com/LerianStudio/fetcher/tests/shared/setup"
)

// ChaosInfrastructure composes SharedInfrastructure with Toxiproxy.
// It provides access to all standard infrastructure plus chaos injection capabilities.
type ChaosInfrastructure struct {
	// Embedded shared infrastructure - provides all base containers
	*sharedsetup.SharedInfrastructure

	// Application containers (Manager and Worker)
	Applications *sharedsetup.ApplicationContainers

	// Toxiproxy for chaos injection
	Toxiproxy *containers.ToxiproxyContainer

	// Standard proxies for all services
	Proxies *containers.StandardProxies

	// Proxy URLs for client connections (traffic goes through Toxiproxy)
	// Use these URLs instead of direct service URLs for chaos injection to work.
	ManagerProxyURL   string // HTTP URL for Manager API via proxy
	SeaweedFSProxyURL string // HTTP URL for SeaweedFS via proxy
	RabbitMQProxyURI  string // AMQP URI for RabbitMQ via proxy
}

// ChaosOptions controls how chaos infrastructure is started.
type ChaosOptions struct {
	// UseFixedPorts uses fixed host ports instead of random ports.
	UseFixedPorts bool

	// ReuseExisting attempts to connect to existing infrastructure.
	ReuseExisting bool

	// SkipExternalDBs skips starting external databases.
	SkipExternalDBs bool

	// InitScripts controls whether to run init scripts for databases.
	InitScripts bool
}

// DefaultChaosOptions returns default options for chaos test execution.
func DefaultChaosOptions() ChaosOptions {
	return ChaosOptions{
		UseFixedPorts:   false,
		ReuseExisting:   false,
		SkipExternalDBs: false,
		InitScripts:     true,
	}
}

// DebugChaosOptions returns options for debug mode with fixed ports.
func DebugChaosOptions() ChaosOptions {
	return ChaosOptions{
		UseFixedPorts:   true,
		ReuseExisting:   false,
		SkipExternalDBs: false,
		InitScripts:     true,
	}
}

// StartChaosInfrastructure starts shared infrastructure plus Toxiproxy with default options.
func StartChaosInfrastructure(ctx context.Context) (*ChaosInfrastructure, error) {
	return StartChaosInfrastructureWithOptions(ctx, DefaultChaosOptions())
}

// StartChaosInfrastructureWithOptions starts chaos infrastructure with specified options.
func StartChaosInfrastructureWithOptions(ctx context.Context, opts ChaosOptions) (*ChaosInfrastructure, error) {
	// Convert chaos options to shared infrastructure options
	sharedOpts := sharedsetup.InfrastructureOptions{
		UseFixedPorts:   opts.UseFixedPorts,
		ReuseExisting:   opts.ReuseExisting,
		SkipExternalDBs: opts.SkipExternalDBs,
		InitScripts:     opts.InitScripts,
	}

	// Start shared infrastructure (all containers)
	shared, err := sharedsetup.StartWithOptions(ctx, sharedOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to start shared infrastructure: %w", err)
	}

	// Setup RabbitMQ topology for event consumption
	if err := shared.SetupRabbitMQTopology(ctx); err != nil {
		_ = shared.Stop(ctx)
		return nil, fmt.Errorf("failed to setup rabbitmq topology: %w", err)
	}

	// Start Toxiproxy
	toxiOpts := containers.DefaultToxiproxyOptions(config.NetworkName)

	toxiContainer, err := containers.StartToxiproxy(ctx, toxiOpts)
	if err != nil {
		_ = shared.Stop(ctx)
		return nil, fmt.Errorf("failed to start toxiproxy: %w", err)
	}

	// Create standard proxies for all services
	upstreams := containers.DefaultStandardUpstreams()

	proxies, err := toxiContainer.CreateStandardProxies(upstreams)
	if err != nil {
		_ = toxiContainer.Stop(ctx)
		_ = shared.Stop(ctx)

		return nil, fmt.Errorf("failed to create proxies: %w", err)
	}

	// Start application containers (Manager and Worker)
	// Test-only encryption keys: 32 bytes for AES-256 (key: "test-encryption-key-32bytes-ok!!")
	encryptionKeyBase64 := "dGVzdC1lbmNyeXB0aW9uLWtleS0zMmJ5dGVzLW9rISE="
	encryptionKeyHex := "746573742d656e6372797074696f6e2d6b65792d333262797465732d6f6b2121"
	appConfig := shared.DefaultApplicationConfig(encryptionKeyBase64, encryptionKeyHex)

	apps, err := shared.StartApplications(ctx, appConfig)
	if err != nil {
		_ = toxiContainer.Stop(ctx)
		_ = shared.Stop(ctx)

		return nil, fmt.Errorf("failed to start applications: %w", err)
	}

	chaosInfra := &ChaosInfrastructure{
		SharedInfrastructure: shared,
		Applications:         apps,
		Toxiproxy:            toxiContainer,
		Proxies:              proxies,
	}

	// Build proxy URLs for client connections
	if err := chaosInfra.buildProxyURLs(ctx); err != nil {
		_ = apps.Stop(ctx)
		_ = toxiContainer.Stop(ctx)
		_ = shared.Stop(ctx)

		return nil, fmt.Errorf("failed to build proxy URLs: %w", err)
	}

	return chaosInfra, nil
}

// Stop terminates all containers including Toxiproxy.
func (c *ChaosInfrastructure) Stop(ctx context.Context) error {
	var errs []error

	if c.Applications != nil {
		if err := c.Applications.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("applications: %w", err))
		}
	}

	if c.Toxiproxy != nil {
		if err := c.Toxiproxy.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("toxiproxy: %w", err))
		}
	}

	if c.SharedInfrastructure != nil {
		if err := c.SharedInfrastructure.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shared infrastructure: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping chaos infrastructure: %v", errs)
	}

	return nil
}

// buildProxyURLs constructs URLs that route traffic through Toxiproxy proxies.
// These URLs should be used by test clients to enable chaos injection.
func (c *ChaosInfrastructure) buildProxyURLs(ctx context.Context) error {
	if c.Toxiproxy == nil {
		return fmt.Errorf("cannot build proxy URLs: Toxiproxy container is nil")
	}

	if c.Proxies == nil {
		return fmt.Errorf("cannot build proxy URLs: StandardProxies is nil")
	}

	// Helper to extract container port from proxy Listen address (e.g., "0.0.0.0:5433" -> "5433/tcp")
	// Handles IPv6 format: [::]:port has multiple colons, port is always last
	extractContainerPort := func(proxy *toxiproxy.Proxy) (string, error) {
		if proxy == nil {
			return "", nil
		}
		// Listen format is "host:port", extract port
		parts := strings.Split(proxy.Listen, ":")
		if len(parts) < 2 {
			return "", fmt.Errorf("unexpected proxy Listen format: %s", proxy.Listen)
		}
		// Port is always the last part (handles IPv6 [::]:port format)
		return parts[len(parts)-1] + "/tcp", nil
	}

	// Manager proxy URL
	if c.Proxies.Manager != nil {
		containerPort, err := extractContainerPort(c.Proxies.Manager)
		if err != nil {
			return fmt.Errorf("failed to extract manager proxy port: %w", err)
		}

		if containerPort != "" {
			hostPort, err := c.Toxiproxy.GetProxyHostPort(ctx, containerPort)
			if err != nil {
				return fmt.Errorf("failed to get manager proxy host port: %w", err)
			}

			c.ManagerProxyURL = fmt.Sprintf("http://%s", hostPort)
		}
	}

	// SeaweedFS proxy URL
	if c.Proxies.SeaweedFS != nil {
		containerPort, err := extractContainerPort(c.Proxies.SeaweedFS)
		if err != nil {
			return fmt.Errorf("failed to extract seaweedfs proxy port: %w", err)
		}

		if containerPort != "" {
			hostPort, err := c.Toxiproxy.GetProxyHostPort(ctx, containerPort)
			if err != nil {
				return fmt.Errorf("failed to get seaweedfs proxy host port: %w", err)
			}

			c.SeaweedFSProxyURL = fmt.Sprintf("http://%s", hostPort)
		}
	}

	// RabbitMQ proxy URI
	// Note: Credentials are hardcoded for test containers only. These are the default
	// RabbitMQ credentials used by ephemeral test containers and should NOT be used
	// in production code.
	if c.Proxies.RabbitMQ != nil {
		containerPort, err := extractContainerPort(c.Proxies.RabbitMQ)
		if err != nil {
			return fmt.Errorf("failed to extract rabbitmq proxy port: %w", err)
		}

		if containerPort != "" {
			hostPort, err := c.Toxiproxy.GetProxyHostPort(ctx, containerPort)
			if err != nil {
				return fmt.Errorf("failed to get rabbitmq proxy host port: %w", err)
			}

			c.RabbitMQProxyURI = fmt.Sprintf("amqp://guest:guest@%s/", hostPort)
		}
	}

	return nil
}

// =============================================================================
// Proxy Accessor Methods
// =============================================================================
// These methods provide convenient access to individual proxies for chaos injection.

// GetRabbitMQProxy returns the RabbitMQ proxy for chaos injection.
func (c *ChaosInfrastructure) GetRabbitMQProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.RabbitMQ
}

// GetMongoMainProxy returns the MongoDB main proxy for chaos injection.
func (c *ChaosInfrastructure) GetMongoMainProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.MongoMain
}

// GetMongoExternalProxy returns the MongoDB external proxy for chaos injection.
func (c *ChaosInfrastructure) GetMongoExternalProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.MongoExternal
}

// GetRedisProxy returns the Redis proxy for chaos injection.
func (c *ChaosInfrastructure) GetRedisProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.Redis
}

// GetPostgresProxy returns the PostgreSQL proxy for chaos injection.
func (c *ChaosInfrastructure) GetPostgresProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.Postgres
}

// GetMySQLProxy returns the MySQL proxy for chaos injection.
func (c *ChaosInfrastructure) GetMySQLProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.MySQL
}

// GetSQLServerProxy returns the SQL Server proxy for chaos injection.
func (c *ChaosInfrastructure) GetSQLServerProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.SQLServer
}

// GetOracleProxy returns the Oracle proxy for chaos injection.
func (c *ChaosInfrastructure) GetOracleProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.Oracle
}

// GetSeaweedFSProxy returns the SeaweedFS proxy for chaos injection.
func (c *ChaosInfrastructure) GetSeaweedFSProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.SeaweedFS
}

// GetManagerProxy returns the Manager proxy for chaos injection.
func (c *ChaosInfrastructure) GetManagerProxy() *toxiproxy.Proxy {
	if c.Proxies == nil {
		return nil
	}

	return c.Proxies.Manager
}

// =============================================================================
// Chaos Injection Convenience Methods
// =============================================================================
// These methods provide shortcuts for common chaos injection patterns.

// DisableRabbitMQ disables the RabbitMQ proxy, simulating a complete outage.
func (c *ChaosInfrastructure) DisableRabbitMQ() error {
	return containers.DisableProxy(c.GetRabbitMQProxy())
}

// EnableRabbitMQ enables the RabbitMQ proxy, restoring connectivity.
func (c *ChaosInfrastructure) EnableRabbitMQ() error {
	return containers.EnableProxy(c.GetRabbitMQProxy())
}

// AddRabbitMQLatency adds latency to RabbitMQ connections.
func (c *ChaosInfrastructure) AddRabbitMQLatency(name string, latencyMS, jitterMS int) (*toxiproxy.Toxic, error) {
	return containers.AddLatency(c.GetRabbitMQProxy(), name, latencyMS, jitterMS)
}

// DisableMongoMain disables the MongoDB main proxy.
func (c *ChaosInfrastructure) DisableMongoMain() error {
	return containers.DisableProxy(c.GetMongoMainProxy())
}

// EnableMongoMain enables the MongoDB main proxy.
func (c *ChaosInfrastructure) EnableMongoMain() error {
	return containers.EnableProxy(c.GetMongoMainProxy())
}

// AddMongoMainLatency adds latency to MongoDB main connections.
func (c *ChaosInfrastructure) AddMongoMainLatency(name string, latencyMS, jitterMS int) (*toxiproxy.Toxic, error) {
	return containers.AddLatency(c.GetMongoMainProxy(), name, latencyMS, jitterMS)
}

// DisableRedis disables the Redis proxy.
func (c *ChaosInfrastructure) DisableRedis() error {
	return containers.DisableProxy(c.GetRedisProxy())
}

// EnableRedis enables the Redis proxy.
func (c *ChaosInfrastructure) EnableRedis() error {
	return containers.EnableProxy(c.GetRedisProxy())
}

// AddRedisLatency adds latency to Redis connections.
func (c *ChaosInfrastructure) AddRedisLatency(name string, latencyMS, jitterMS int) (*toxiproxy.Toxic, error) {
	return containers.AddLatency(c.GetRedisProxy(), name, latencyMS, jitterMS)
}

// DisablePostgres disables the PostgreSQL proxy.
func (c *ChaosInfrastructure) DisablePostgres() error {
	return containers.DisableProxy(c.GetPostgresProxy())
}

// EnablePostgres enables the PostgreSQL proxy.
func (c *ChaosInfrastructure) EnablePostgres() error {
	return containers.EnableProxy(c.GetPostgresProxy())
}

// AddPostgresLatency adds latency to PostgreSQL connections.
func (c *ChaosInfrastructure) AddPostgresLatency(name string, latencyMS, jitterMS int) (*toxiproxy.Toxic, error) {
	return containers.AddLatency(c.GetPostgresProxy(), name, latencyMS, jitterMS)
}

// DisableSeaweedFS disables the SeaweedFS proxy, simulating a complete storage outage.
func (c *ChaosInfrastructure) DisableSeaweedFS() error {
	return containers.DisableProxy(c.GetSeaweedFSProxy())
}

// EnableSeaweedFS enables the SeaweedFS proxy, restoring storage connectivity.
func (c *ChaosInfrastructure) EnableSeaweedFS() error {
	return containers.EnableProxy(c.GetSeaweedFSProxy())
}

// AddSeaweedFSLatency adds latency to SeaweedFS connections.
func (c *ChaosInfrastructure) AddSeaweedFSLatency(name string, latencyMS, jitterMS int) (*toxiproxy.Toxic, error) {
	return containers.AddLatency(c.GetSeaweedFSProxy(), name, latencyMS, jitterMS)
}

// AddSeaweedFSTimeout adds a timeout toxic to SeaweedFS connections.
func (c *ChaosInfrastructure) AddSeaweedFSTimeout(name string, timeoutMS int) (*toxiproxy.Toxic, error) {
	return containers.AddTimeout(c.GetSeaweedFSProxy(), name, timeoutMS)
}

// AddSeaweedFSBandwidth limits the bandwidth of SeaweedFS connections.
func (c *ChaosInfrastructure) AddSeaweedFSBandwidth(name string, rateBytesPerSec int) (*toxiproxy.Toxic, error) {
	return containers.AddBandwidth(c.GetSeaweedFSProxy(), name, rateBytesPerSec)
}

// =============================================================================
// Cleanup Methods
// =============================================================================

// RemoveAllToxics removes all toxics from all proxies.
func (c *ChaosInfrastructure) RemoveAllToxics() error {
	var errs []error

	proxies := []*toxiproxy.Proxy{
		c.GetRabbitMQProxy(),
		c.GetMongoMainProxy(),
		c.GetMongoExternalProxy(),
		c.GetRedisProxy(),
		c.GetPostgresProxy(),
		c.GetMySQLProxy(),
		c.GetSQLServerProxy(),
		c.GetOracleProxy(),
		c.GetSeaweedFSProxy(),
		c.GetManagerProxy(),
	}

	for _, proxy := range proxies {
		if proxy != nil {
			if err := containers.RemoveAllToxics(proxy); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors removing toxics: %v", errs)
	}

	return nil
}

// EnableAllProxies enables all proxies (restores connectivity).
func (c *ChaosInfrastructure) EnableAllProxies() error {
	var errs []error

	proxies := []*toxiproxy.Proxy{
		c.GetRabbitMQProxy(),
		c.GetMongoMainProxy(),
		c.GetMongoExternalProxy(),
		c.GetRedisProxy(),
		c.GetPostgresProxy(),
		c.GetMySQLProxy(),
		c.GetSQLServerProxy(),
		c.GetOracleProxy(),
		c.GetSeaweedFSProxy(),
		c.GetManagerProxy(),
	}

	for _, proxy := range proxies {
		if proxy != nil {
			if err := containers.EnableProxy(proxy); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors enabling proxies: %v", errs)
	}

	return nil
}

// ResetChaos removes all toxics and enables all proxies.
// This is useful for cleanup between test cases.
func (c *ChaosInfrastructure) ResetChaos() error {
	if err := c.RemoveAllToxics(); err != nil {
		return fmt.Errorf("failed to remove toxics: %w", err)
	}

	if err := c.EnableAllProxies(); err != nil {
		return fmt.Errorf("failed to enable proxies: %w", err)
	}

	return nil
}

// =============================================================================
// Proxy Connection Methods
// =============================================================================
// These methods return connection info that routes through Toxiproxy.
// Use these instead of direct connection methods when you want chaos injection
// to affect the traffic.

// PostgresProxyInternal returns PostgreSQL connection info routed through Toxiproxy.
// Traffic flows: Worker -> Toxiproxy:5433 -> postgres-external:5432
func (c *ChaosInfrastructure) PostgresProxyInternal() config.InternalDBConnection {
	direct := c.PostgresInternal()

	return config.InternalDBConnection{
		Host:     c.Toxiproxy.InternalHost, // "toxiproxy"
		Port:     5433,                     // Toxiproxy listen port for postgres
		Username: direct.Username,
		Password: direct.Password,
		Database: direct.Database,
	}
}

// MySQLProxyInternal returns MySQL connection info routed through Toxiproxy.
// Traffic flows: Worker -> Toxiproxy:3307 -> mysql-external:3306
func (c *ChaosInfrastructure) MySQLProxyInternal() config.InternalDBConnection {
	direct := c.MySQLInternal()

	return config.InternalDBConnection{
		Host:     c.Toxiproxy.InternalHost,
		Port:     3307,
		Username: direct.Username,
		Password: direct.Password,
		Database: direct.Database,
	}
}

// SQLServerProxyInternal returns SQL Server connection info routed through Toxiproxy.
// Traffic flows: Worker -> Toxiproxy:1434 -> sqlserver-external:1433
func (c *ChaosInfrastructure) SQLServerProxyInternal() config.InternalDBConnection {
	direct := c.SQLServerInternal()

	return config.InternalDBConnection{
		Host:     c.Toxiproxy.InternalHost,
		Port:     1434,
		Username: direct.Username,
		Password: direct.Password,
		Database: direct.Database,
	}
}

// OracleProxyInternal returns Oracle connection info routed through Toxiproxy.
// Traffic flows: Worker -> Toxiproxy:1522 -> oracle-external:1521
func (c *ChaosInfrastructure) OracleProxyInternal() config.InternalDBConnection {
	direct := c.OracleInternal()

	return config.InternalDBConnection{
		Host:     c.Toxiproxy.InternalHost,
		Port:     1522,
		Username: direct.Username,
		Password: direct.Password,
		Database: direct.Database,
	}
}

// MongoExternalProxyInternal returns external MongoDB connection info routed through Toxiproxy.
// Traffic flows: Worker -> Toxiproxy:27101 -> fetcher-mongodb-external:27017
func (c *ChaosInfrastructure) MongoExternalProxyInternal() config.InternalDBConnection {
	direct := c.MongoExternalInternal()

	return config.InternalDBConnection{
		Host:     c.Toxiproxy.InternalHost,
		Port:     27101,
		Username: direct.Username,
		Password: direct.Password,
		Database: direct.Database,
	}
}
