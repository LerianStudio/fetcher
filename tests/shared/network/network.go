package network

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// CreateNetwork creates a Docker network for test containers.
// Note: Using deprecated GenericNetwork API because network.New() doesn't support custom names,
// but the infrastructure requires the hardcoded "fetcher-test-network" name.
//
//nolint:staticcheck // SA1019: Using deprecated API for named network support
func CreateNetwork(ctx context.Context) (testcontainers.Network, error) {
	net, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name:   config.NetworkName,
			Driver: "bridge",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	return net, nil
}

// GetNetworkName returns the standard network name.
func GetNetworkName() string {
	return config.NetworkName
}
