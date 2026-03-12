//go:build unit

package storage_test

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRepository_SeaweedFS(t *testing.T) {
	t.Parallel()

	repo, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider:          storage.ProviderSeaweedFS,
		SeaweedFSEndpoint: "http://localhost:8889",
		Bucket:            "external-data",
	})
	require.NoError(t, err)
	require.NotNil(t, repo)
}

func TestNewRepository_S3(t *testing.T) {
	t.Parallel()

	repo, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider:          storage.ProviderS3,
		S3Endpoint:        "http://localhost:9000",
		S3Bucket:          "external-data",
		S3AccessKeyID:     "minioadmin",
		S3SecretAccessKey: "minioadmin",
		S3UsePathStyle:    true,
	})
	require.NoError(t, err)
	require.NotNil(t, repo)
}

func TestNewRepository_UnsupportedProvider(t *testing.T) {
	t.Parallel()

	_, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider: "gcs",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported storage provider")
}

// TestNewRepository_SeaweedFS_DefaultEmpty verifies that an empty provider string
// defaults to the SeaweedFS path, since "" is treated as a synonym for "seaweedfs".
func TestNewRepository_SeaweedFS_DefaultEmpty(t *testing.T) {
	t.Parallel()

	repo, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider:          "",
		SeaweedFSEndpoint: "http://localhost:8889",
	})
	require.NoError(t, err)
	require.NotNil(t, repo)
}

// TestNewRepository_SeaweedFS_MissingEndpoint verifies that the SeaweedFS path
// returns a descriptive error when the endpoint is empty. This is the primary
// validation boundary for the SeaweedFS provider.
func TestNewRepository_SeaweedFS_MissingEndpoint(t *testing.T) {
	t.Parallel()

	_, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider:          storage.ProviderSeaweedFS,
		SeaweedFSEndpoint: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "seaweedfs endpoint is required")
}

// TestNewRepository_SeaweedFS_DefaultBucket verifies that when Bucket is empty,
// the factory uses the constant.ExternalDataBucketName default. The returned
// repository must be non-nil (a valid SimpleRepository).
func TestNewRepository_SeaweedFS_DefaultBucket(t *testing.T) {
	t.Parallel()

	repo, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider:          storage.ProviderSeaweedFS,
		SeaweedFSEndpoint: "http://localhost:8889",
		Bucket:            "", // empty => uses constant.ExternalDataBucketName
	})
	require.NoError(t, err)
	require.NotNil(t, repo)
}

// TestNewRepository_S3_UsesDefaultBucket verifies that when both S3Bucket and
// Bucket are empty, the factory falls back to constant.ExternalDataBucketName.
func TestNewRepository_S3_UsesDefaultBucket(t *testing.T) {
	t.Parallel()

	repo, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider:          storage.ProviderS3,
		S3Endpoint:        "http://localhost:9000",
		S3AccessKeyID:     "key",
		S3SecretAccessKey: "secret",
		S3UsePathStyle:    true,
		// S3Bucket and Bucket are both empty => should use default
	})
	require.NoError(t, err)
	require.NotNil(t, repo)
}

// TestNewRepository_S3_UsesBucketFallback verifies that when S3Bucket is empty
// but Bucket is set, the factory uses Bucket as the S3 bucket name.
func TestNewRepository_S3_UsesBucketFallback(t *testing.T) {
	t.Parallel()

	repo, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider:          storage.ProviderS3,
		S3Endpoint:        "http://localhost:9000",
		Bucket:            "my-bucket",
		S3AccessKeyID:     "key",
		S3SecretAccessKey: "secret",
		S3UsePathStyle:    true,
	})
	require.NoError(t, err)
	require.NotNil(t, repo)
}

// TestNewRepository_S3_PreferS3Bucket verifies that when both S3Bucket and
// Bucket are set, the factory prefers S3Bucket over the generic Bucket field.
func TestNewRepository_S3_PreferS3Bucket(t *testing.T) {
	t.Parallel()

	repo, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider:          storage.ProviderS3,
		S3Endpoint:        "http://localhost:9000",
		Bucket:            "generic-bucket",
		S3Bucket:          "s3-specific-bucket",
		S3AccessKeyID:     "key",
		S3SecretAccessKey: "secret",
		S3UsePathStyle:    true,
	})
	require.NoError(t, err)
	require.NotNil(t, repo)
}

// TestNewRepository_EmptyProvider_MissingEndpoint verifies that even when
// provider is "" (defaults to SeaweedFS), the endpoint validation still fires.
func TestNewRepository_EmptyProvider_MissingEndpoint(t *testing.T) {
	t.Parallel()

	_, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
		Provider:          "",
		SeaweedFSEndpoint: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "seaweedfs endpoint is required")
}

// TestNewRepository_Constants verifies that the exported provider constants
// have the expected string values, acting as a compile-time contract test.
func TestNewRepository_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "seaweedfs", storage.ProviderSeaweedFS)
	assert.Equal(t, "s3", storage.ProviderS3)
}

// TestNewRepository_UnsupportedProviders verifies that various invalid provider
// strings all return errors containing "unsupported storage provider". This is
// a table-driven extension of the basic unsupported provider test.
func TestNewRepository_UnsupportedProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider string
	}{
		{name: "gcs", provider: "gcs"},
		{name: "azure", provider: "azure"},
		{name: "minio", provider: "minio"},
		{name: "uppercase_S3", provider: "S3"},
		{name: "uppercase_SEAWEEDFS", provider: "SEAWEEDFS"},
		{name: "whitespace", provider: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
				Provider: tt.provider,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported storage provider")
		})
	}
}
