package containers

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// SeaweedFSContainers holds all SeaweedFS components.
type SeaweedFSContainers struct {
	Master       testcontainers.Container
	Volume       testcontainers.Container
	Filer        testcontainers.Container
	URL          string
	Host         string
	Port         string
	InternalHost string
}

// SeaweedFSOptions configures SeaweedFS container startup.
type SeaweedFSOptions struct {
	NetworkName   string
	MasterAlias   string
	VolumeAlias   string
	FilerAlias    string
	FixedHostPort string
}

// DefaultSeaweedFSOptions returns default SeaweedFS options.
func DefaultSeaweedFSOptions(networkName string) SeaweedFSOptions {
	return SeaweedFSOptions{
		NetworkName: networkName,
		MasterAlias: "fetcher-seaweedfs-master",
		VolumeAlias: "fetcher-seaweedfs-volume",
		FilerAlias:  "fetcher-seaweedfs-filer",
	}
}

// StartSeaweedFS starts all SeaweedFS components.
func StartSeaweedFS(ctx context.Context, opts SeaweedFSOptions) (*SeaweedFSContainers, error) {
	// Start Master
	masterReq := testcontainers.ContainerRequest{
		Image:        "chrislusf/seaweedfs:latest",
		ExposedPorts: []string{"9333/tcp"},
		Cmd:          []string{"master"},
		WaitingFor:   wait.ForHTTP("/cluster/status").WithPort("9333/tcp").WithStartupTimeout(config.SeaweedFSStartupTimeout),
	}

	if opts.NetworkName != "" {
		masterReq.Networks = []string{opts.NetworkName}
		masterReq.NetworkAliases = map[string][]string{
			opts.NetworkName: {opts.MasterAlias},
		}
	}

	master, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: masterReq,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start SeaweedFS master: %w", err)
	}

	// Start Volume
	volumeReq := testcontainers.ContainerRequest{
		Image:        "chrislusf/seaweedfs:latest",
		ExposedPorts: []string{"8080/tcp"},
		Cmd:          []string{"volume", "-mserver=" + opts.MasterAlias + ":9333", "-port=8080"},
		WaitingFor:   wait.ForHTTP("/status").WithPort("8080/tcp").WithStartupTimeout(config.SeaweedFSStartupTimeout),
	}

	if opts.NetworkName != "" {
		volumeReq.Networks = []string{opts.NetworkName}
		volumeReq.NetworkAliases = map[string][]string{
			opts.NetworkName: {opts.VolumeAlias},
		}
	}

	volume, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: volumeReq,
		Started:          true,
	})
	if err != nil {
		_ = master.Terminate(ctx)
		return nil, fmt.Errorf("failed to start SeaweedFS volume: %w", err)
	}

	// Start Filer
	filerReq := testcontainers.ContainerRequest{
		Image:        "chrislusf/seaweedfs:latest",
		ExposedPorts: []string{"8888/tcp"},
		Cmd:          []string{"filer", "-master=" + opts.MasterAlias + ":9333"},
		WaitingFor:   wait.ForHTTP("/").WithPort("8888/tcp").WithStartupTimeout(config.SeaweedFSStartupTimeout),
	}

	if opts.NetworkName != "" {
		filerReq.Networks = []string{opts.NetworkName}
		filerReq.NetworkAliases = map[string][]string{
			opts.NetworkName: {opts.FilerAlias},
		}
	}

	if opts.FixedHostPort != "" {
		filerReq.ExposedPorts = []string{opts.FixedHostPort + ":8888/tcp"}
	}

	filer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: filerReq,
		Started:          true,
	})
	if err != nil {
		_ = volume.Terminate(ctx)
		_ = master.Terminate(ctx)

		return nil, fmt.Errorf("failed to start SeaweedFS filer: %w", err)
	}

	host, err := filer.Host(ctx)
	if err != nil {
		_ = filer.Terminate(ctx)
		_ = volume.Terminate(ctx)
		_ = master.Terminate(ctx)

		return nil, fmt.Errorf("failed to get SeaweedFS host: %w", err)
	}

	port, err := filer.MappedPort(ctx, "8888")
	if err != nil {
		_ = filer.Terminate(ctx)
		_ = volume.Terminate(ctx)
		_ = master.Terminate(ctx)

		return nil, fmt.Errorf("failed to get SeaweedFS port: %w", err)
	}

	return &SeaweedFSContainers{
		Master:       master,
		Volume:       volume,
		Filer:        filer,
		URL:          fmt.Sprintf("http://%s:%s", host, port.Port()),
		Host:         host,
		Port:         port.Port(),
		InternalHost: opts.FilerAlias,
	}, nil
}

// Stop terminates all SeaweedFS containers.
func (s *SeaweedFSContainers) Stop(ctx context.Context) error {
	var errs []error

	if s.Filer != nil {
		if err := s.Filer.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if s.Volume != nil {
		if err := s.Volume.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if s.Master != nil {
		if err := s.Master.Terminate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping SeaweedFS: %v", errs)
	}

	return nil
}

// GetHost returns the container's host address.
func (s *SeaweedFSContainers) GetHost() string { return s.Host }

// GetPort returns the container's mapped port.
func (s *SeaweedFSContainers) GetPort() string { return s.Port }

// GetURI returns the container's URL.
func (s *SeaweedFSContainers) GetURI() string { return s.URL }
