package setup

import (
	"context"
	"fmt"
	"sync"
)

// ContainerConfig defines configuration for a container.
type ContainerConfig struct {
	// Name is the human-readable name of the container (e.g., "mongodb", "postgres")
	Name string

	// Required indicates if this container must be started
	Required bool

	// NetworkAlias is the Docker network alias for the container
	NetworkAlias string

	// FixedPort is the fixed host port (if UseFixedPorts is enabled)
	FixedPort string

	// StartFunc is the function that starts the container
	// It receives context, network name, and options
	StartFunc func(ctx context.Context, networkName string, opts ContainerStartOptions) (any, error)

	// SSLStartFunc is the function that starts the SSL variant (optional)
	SSLStartFunc func(ctx context.Context, networkName string, opts ContainerStartOptions) (any, error)
}

// ContainerStartOptions provides common options for container startup.
type ContainerStartOptions struct {
	UseFixedPorts bool
	EnableSSL     bool
	InitScript    string
	SSLBundle     any // *ssl.CertificateBundle
}

// ContainerRegistry manages container configurations.
// Uses pointer-based storage to avoid slice reallocation issues.
type ContainerRegistry struct {
	mu      sync.RWMutex
	configs []*ContainerConfig
	byName  map[string]*ContainerConfig
}

// NewContainerRegistry creates a new container registry.
func NewContainerRegistry() *ContainerRegistry {
	return &ContainerRegistry{
		configs: make([]*ContainerConfig, 0),
		byName:  make(map[string]*ContainerConfig),
	}
}

// Register adds a container configuration to the registry.
func (r *ContainerRegistry) Register(config ContainerConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Allocate config on heap to avoid slice reallocation issues
	cfg := &config
	r.configs = append(r.configs, cfg)
	r.byName[config.Name] = cfg
}

// GetAll returns all registered container configurations.
func (r *ContainerRegistry) GetAll() []ContainerConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ContainerConfig, len(r.configs))
	for i, cfg := range r.configs {
		result[i] = *cfg
	}

	return result
}

// GetRequired returns only required container configurations.
func (r *ContainerRegistry) GetRequired() []ContainerConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ContainerConfig

	for _, cfg := range r.configs {
		if cfg.Required {
			result = append(result, *cfg)
		}
	}

	return result
}

// GetOptional returns only optional container configurations.
func (r *ContainerRegistry) GetOptional() []ContainerConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ContainerConfig

	for _, cfg := range r.configs {
		if !cfg.Required {
			result = append(result, *cfg)
		}
	}

	return result
}

// Get returns a specific container configuration by name.
func (r *ContainerRegistry) Get(name string) (*ContainerConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, ok := r.byName[name]

	return cfg, ok
}

// StartResult holds the result of starting a container.
type StartResult struct {
	Name      string
	Container any
	Error     error
}

// StartContainersParallel starts multiple containers in parallel.
func (r *ContainerRegistry) StartContainersParallel(
	ctx context.Context,
	networkName string,
	opts ContainerStartOptions,
	configs []ContainerConfig,
) []StartResult {
	var wg sync.WaitGroup

	results := make([]StartResult, len(configs))

	for i, cfg := range configs {
		wg.Add(1)

		go func(idx int, config ContainerConfig) {
			defer wg.Done()

			results[idx].Name = config.Name

			if config.StartFunc == nil {
				results[idx].Error = fmt.Errorf("no start function for container %s", config.Name)
				return
			}

			container, err := config.StartFunc(ctx, networkName, opts)
			results[idx].Container = container
			results[idx].Error = err
		}(i, cfg)
	}

	wg.Wait()

	return results
}
