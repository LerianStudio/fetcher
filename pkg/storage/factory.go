package storage

import (
	"context"
	"fmt"

	portStorage "github.com/LerianStudio/fetcher/pkg/ports/storage"
	simpleClient "github.com/LerianStudio/fetcher/pkg/seaweedfs"
	"github.com/LerianStudio/fetcher/pkg/seaweedfs/external"
)

const (
	// ProviderSeaweedFS is the provider identifier for SeaweedFS HTTP storage.
	ProviderSeaweedFS = "seaweedfs"

	// ProviderS3 is the provider identifier for S3-compatible storage.
	ProviderS3 = "s3"
)

// ProviderConfig aggregates all configuration needed for any storage provider.
type ProviderConfig struct {
	// Provider selects the storage backend: "seaweedfs" (default) or "s3".
	Provider string

	// SeaweedFSEndpoint is the HTTP endpoint for SeaweedFS Filer (e.g., "http://seaweedfs:8889").
	SeaweedFSEndpoint string

	// Bucket is the bucket name for SeaweedFS storage.
	Bucket string

	// S3 configuration fields (used when Provider="s3").
	// SSL is controlled by the URL scheme of S3Endpoint: "http://" disables SSL.
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3KeyPrefix       string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3UsePathStyle    bool
}

// NewRepository creates the appropriate storage.Repository based on the provider configuration.
//   - "seaweedfs" or "" -> SimpleRepository (SeaweedFS HTTP, current behavior)
//   - "s3"              -> S3Repository (AWS SDK v2; compatible with MinIO and SeaweedFS S3 mode)
//   - any other value   -> error
//
// Returns (nil, error) if validation fails, the provider is unsupported, or client creation fails.
// Callers MUST check the error before using the repository.
func NewRepository(ctx context.Context, cfg ProviderConfig) (portStorage.Repository, error) {
	switch cfg.Provider {
	case ProviderSeaweedFS, "":
		if cfg.SeaweedFSEndpoint == "" {
			return nil, fmt.Errorf("seaweedfs endpoint is required when provider is %q", ProviderSeaweedFS)
		}

		bucket := cfg.Bucket
		if bucket == "" {
			return nil, fmt.Errorf("bucket name is required (set OBJECT_STORAGE_BUCKET)")
		}

		client := simpleClient.NewSeaweedFSClient(cfg.SeaweedFSEndpoint)
		if client == nil {
			return nil, fmt.Errorf("failed to create seaweedfs client for endpoint %q", cfg.SeaweedFSEndpoint)
		}

		return external.NewSimpleRepository(client, bucket), nil

	case ProviderS3:
		s3Bucket := cfg.S3Bucket
		if s3Bucket == "" {
			s3Bucket = cfg.Bucket
		}

		if s3Bucket == "" {
			return nil, fmt.Errorf("bucket name is required (set OBJECT_STORAGE_BUCKET)")
		}

		return NewS3Repository(ctx, S3Config{
			Endpoint:        cfg.S3Endpoint,
			Region:          cfg.S3Region,
			Bucket:          s3Bucket,
			KeyPrefix:       cfg.S3KeyPrefix,
			AccessKeyID:     cfg.S3AccessKeyID,
			SecretAccessKey: cfg.S3SecretAccessKey,
			UsePathStyle:    cfg.S3UsePathStyle,
		})

	default:
		return nil, fmt.Errorf("unsupported storage provider: %q (supported: %q, %q)", cfg.Provider, ProviderSeaweedFS, ProviderS3)
	}
}
