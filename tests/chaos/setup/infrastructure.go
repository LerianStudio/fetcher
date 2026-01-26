// Package setup provides infrastructure orchestration for chaos tests.
// It composes the shared test infrastructure with Toxiproxy for network chaos injection.
package setup

import (
	"context"
	"fmt"
	"strings"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"

	"github.com/LerianStudio/fetcher/tests/shared/chaos"
	"github.com/LerianStudio/fetcher/tests/shared/config"
	"github.com/LerianStudio/fetcher/tests/shared/containers"
	"github.com/LerianStudio/fetcher/tests/shared/setup"
)

// ChaosInfrastructure composes SharedInfrastructure with Toxiproxy.
// It provides access to all standard infrastructure plus chaos injection capabilities.
type ChaosInfrastructure struct {
	// Embedded shared infrastructure - provides all base containers
	*setup.SharedInfrastructure

	// Application containers (Manager and Worker)
	Applications *setup.ApplicationContainers

	// Toxiproxy for chaos injection
	Toxiproxy *containers.ToxiproxyContainer

	// Standard proxies for all services
	Proxies *containers.StandardProxies

	// ProxyRegistry provides type-safe access to proxies by service name.
	// Use this instead of direct Proxies field access for new code.
	ProxyRegistry *chaos.ProxyRegistry

	// ChaosOps provides generic chaos injection operations.
	// Use this instead of service-specific methods (DisableRabbitMQ, etc.) for new code.
	ChaosOps *chaos.ChaosOperations

	// ProxyRouter provides generic proxy connection routing.
	// Use this instead of service-specific methods (PostgresProxyInternal, etc.) for new code.
	ProxyRouter *chaos.ProxyRouter

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
	sharedOpts := setup.InfrastructureOptions{
		UseFixedPorts:   opts.UseFixedPorts,
		ReuseExisting:   opts.ReuseExisting,
		SkipExternalDBs: opts.SkipExternalDBs,
		InitScripts:     opts.InitScripts,
	}

	// Start shared infrastructure (all containers)
	shared, err := setup.StartWithOptions(ctx, sharedOpts)
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

	// Initialize ProxyRegistry from standard proxies
	registry := chaos.NewProxyRegistry()
	registry.RegisterFromStandardProxies(&chaos.StandardProxiesAdapter{
		MongoMain:     proxies.MongoMain,
		MongoExternal: proxies.MongoExternal,
		RabbitMQ:      proxies.RabbitMQ,
		SeaweedFS:     proxies.SeaweedFS,
		Redis:         proxies.Redis,
		Postgres:      proxies.Postgres,
		MySQL:         proxies.MySQL,
		SQLServer:     proxies.SQLServer,
		Oracle:        proxies.Oracle,
		Manager:       proxies.Manager,
	})

	chaosInfra := &ChaosInfrastructure{
		SharedInfrastructure: shared,
		Applications:         apps,
		Toxiproxy:            toxiContainer,
		Proxies:              proxies,
		ProxyRegistry:        registry,
		ChaosOps:             chaos.NewChaosOperations(registry),
		ProxyRouter:          chaos.NewProxyRouter(toxiContainer.InternalHost),
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
// Cleanup Methods
// =============================================================================

// RemoveAllToxics removes all toxics from all proxies.
// Delegates to ChaosOps for consistent behavior.
func (c *ChaosInfrastructure) RemoveAllToxics() error {
	return c.ChaosOps.RemoveAllToxics()
}

// EnableAllProxies enables all proxies (restores connectivity).
// Delegates to ChaosOps for consistent behavior.
func (c *ChaosInfrastructure) EnableAllProxies() error {
	return c.ChaosOps.EnableAll()
}

// ResetChaos removes all toxics and enables all proxies.
// This is useful for cleanup between test cases.
func (c *ChaosInfrastructure) ResetChaos() error {
	return c.ChaosOps.ResetAll()
}
