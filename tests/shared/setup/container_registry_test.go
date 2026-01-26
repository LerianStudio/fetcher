package setup

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerRegistry_Register(t *testing.T) {
	registry := NewContainerRegistry()

	registry.Register(ContainerConfig{
		Name:     "test-container",
		Required: true,
	})

	configs := registry.GetAll()
	assert.Len(t, configs, 1)
	assert.Equal(t, "test-container", configs[0].Name)
}

func TestContainerRegistry_GetRequired(t *testing.T) {
	registry := NewContainerRegistry()

	registry.Register(ContainerConfig{Name: "required", Required: true})
	registry.Register(ContainerConfig{Name: "optional", Required: false})

	required := registry.GetRequired()
	assert.Len(t, required, 1)
	assert.Equal(t, "required", required[0].Name)
}

func TestContainerRegistry_GetOptional(t *testing.T) {
	registry := NewContainerRegistry()

	registry.Register(ContainerConfig{Name: "required", Required: true})
	registry.Register(ContainerConfig{Name: "optional", Required: false})

	optional := registry.GetOptional()
	assert.Len(t, optional, 1)
	assert.Equal(t, "optional", optional[0].Name)
}

func TestContainerRegistry_Get(t *testing.T) {
	registry := NewContainerRegistry()

	registry.Register(ContainerConfig{Name: "postgres", Required: true})

	cfg, ok := registry.Get("postgres")
	require.True(t, ok)
	assert.Equal(t, "postgres", cfg.Name)

	_, ok = registry.Get("nonexistent")
	assert.False(t, ok)
}

func TestContainerRegistry_StartContainersParallel(t *testing.T) {
	registry := NewContainerRegistry()

	// Create configs with start functions
	successConfig := ContainerConfig{
		Name:     "success",
		Required: true,
		StartFunc: func(ctx context.Context, networkName string, opts ContainerStartOptions) (any, error) {
			return "success-container", nil
		},
	}
	failConfig := ContainerConfig{
		Name:     "fail",
		Required: true,
		StartFunc: func(ctx context.Context, networkName string, opts ContainerStartOptions) (any, error) {
			return nil, errors.New("start failed")
		},
	}
	noStartFunc := ContainerConfig{
		Name:     "no-start",
		Required: true,
	}

	registry.Register(successConfig)
	registry.Register(failConfig)
	registry.Register(noStartFunc)

	results := registry.StartContainersParallel(
		context.Background(),
		"test-network",
		ContainerStartOptions{},
		registry.GetAll(),
	)

	require.Len(t, results, 3)

	// Find results by name
	resultMap := make(map[string]StartResult)
	for _, r := range results {
		resultMap[r.Name] = r
	}

	// Success config should succeed
	assert.NoError(t, resultMap["success"].Error)
	assert.Equal(t, "success-container", resultMap["success"].Container)

	// Fail config should have error
	assert.Error(t, resultMap["fail"].Error)
	assert.Contains(t, resultMap["fail"].Error.Error(), "start failed")

	// No start func should have error
	assert.Error(t, resultMap["no-start"].Error)
	assert.Contains(t, resultMap["no-start"].Error.Error(), "no start function")
}
