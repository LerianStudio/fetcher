// Package setup provides test infrastructure for integration tests.
// It wraps the shared setup package to provide a consistent API.
package setup

import (
	"context"

	"github.com/LerianStudio/fetcher/tests/shared/config"
	sharedsetup "github.com/LerianStudio/fetcher/tests/shared/setup"
	"github.com/LerianStudio/fetcher/tests/shared/topology"
)

// InfrastructureContainers is an alias for SharedInfrastructure.
type InfrastructureContainers = sharedsetup.SharedInfrastructure

// Re-export only what's actually used by integration tests.
var (
	RemoveInfraConfig = config.RemoveInfraConfig
	InfraConfigPath   = config.InfraConfigPath
)

// Timeout constants used by integration tests.
const (
	SuiteTimeout             = config.SuiteTimeout
	ManagerReadyTimeout      = config.ManagerReadyTimeout
	ManagerReadyPollInterval = config.ManagerReadyPollInterval
	JobCompletionTimeout     = config.JobCompletionTimeout
	JobCompletionTimeoutSlow = config.JobCompletionTimeoutSlow
	SeaweedFSFileTimeout     = config.SeaweedFSFileTimeout
)

// InfrastructureOptions controls how infrastructure is started.
type InfrastructureOptions struct {
	UseFixedPorts bool
	ReuseExisting bool
	EnableSSL     bool
}

// DefaultInfrastructureOptions returns options for normal test execution.
func DefaultInfrastructureOptions() InfrastructureOptions {
	return InfrastructureOptions{UseFixedPorts: false, ReuseExisting: false}
}

// DebugInfrastructureOptions returns options for debug mode with fixed ports.
func DebugInfrastructureOptions() InfrastructureOptions {
	return InfrastructureOptions{UseFixedPorts: true, ReuseExisting: false}
}

// ReuseInfrastructureOptions returns options for reusing existing infrastructure.
func ReuseInfrastructureOptions() InfrastructureOptions {
	return InfrastructureOptions{UseFixedPorts: true, ReuseExisting: true}
}

// SSLInfrastructureOptions returns options for testing with SSL-enabled databases.
func SSLInfrastructureOptions() InfrastructureOptions {
	return InfrastructureOptions{UseFixedPorts: true, ReuseExisting: false, EnableSSL: true}
}

// StartInfrastructure starts all infrastructure containers with default options.
func StartInfrastructure(ctx context.Context) (*InfrastructureContainers, error) {
	return StartInfrastructureWithOptions(ctx, DefaultInfrastructureOptions())
}

// StartInfrastructureWithOptions starts infrastructure with specified options.
func StartInfrastructureWithOptions(ctx context.Context, opts InfrastructureOptions) (*InfrastructureContainers, error) {
	return sharedsetup.StartWithOptions(ctx, sharedsetup.InfrastructureOptions{
		UseFixedPorts:   opts.UseFixedPorts,
		ReuseExisting:   opts.ReuseExisting,
		SkipExternalDBs: false,
		InitScripts:     true,
		EnableSSL:       opts.EnableSSL,
	})
}

// SetupRabbitMQTopology creates the required exchanges and queues.
func SetupRabbitMQTopology(ctx context.Context, amqpURL string) error {
	return topology.SetupRabbitMQTopology(ctx, amqpURL)
}

// PurgeTestQueue purges the test.job.events queue to remove stale events.
func PurgeTestQueue(ctx context.Context, amqpURL string) (int, error) {
	return topology.PurgeTestQueue(ctx, amqpURL)
}
