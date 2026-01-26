package setup

import (
	"context"
	"os"

	"github.com/LerianStudio/fetcher/tests/shared/setup"
)

// Re-export types from shared setup for backward compatibility.
type (
	ApplicationContainers = setup.ApplicationContainers
	ApplicationConfig     = setup.ApplicationConfig
	ApplicationOptions    = setup.ApplicationOptions
)

// Re-export functions from shared setup for backward compatibility.
var (
	DefaultApplicationOptions = setup.DefaultApplicationOptions
	ManagerDebugOptions       = setup.ManagerDebugOptions
	WorkerDebugOptions        = setup.WorkerDebugOptions
	FullDebugOptions          = setup.FullDebugOptions
)

// StartApplications starts Manager and Worker containers.
func StartApplications(ctx context.Context, infra *InfrastructureContainers, cfg ApplicationConfig) (*ApplicationContainers, error) {
	externalManagerURL := os.Getenv("EXTERNAL_MANAGER_URL")

	opts := DefaultApplicationOptions()
	if externalManagerURL != "" {
		opts = ManagerDebugOptions(externalManagerURL)
	}

	if os.Getenv("SKIP_WORKER") == "true" {
		opts.SkipWorker = true
	}

	return infra.StartApplicationsWithOptions(ctx, cfg, opts)
}

// StartApplicationsWithOptions starts Manager and/or Worker containers based on options.
func StartApplicationsWithOptions(ctx context.Context, infra *InfrastructureContainers, cfg ApplicationConfig, opts ApplicationOptions) (*ApplicationContainers, error) {
	return infra.StartApplicationsWithOptions(ctx, cfg, opts)
}
