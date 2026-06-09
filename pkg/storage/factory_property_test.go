//go:build unit

package storage_test

import (
	"context"
	"testing"
	"testing/quick"

	"github.com/LerianStudio/fetcher/v2/pkg/storage"
	"github.com/stretchr/testify/require"
)

// TestProperty_NewRepository_InvalidProvider_AlwaysErrors verifies that any
// provider string other than "seaweedfs", "s3", or "" always returns an error.
// This is a critical invariant: the factory must reject unknown providers
// regardless of the rest of the configuration.
func TestProperty_NewRepository_InvalidProvider_AlwaysErrors(t *testing.T) {
	t.Parallel()

	validProviders := map[string]bool{
		"seaweedfs": true,
		"s3":        true,
		"":          true,
	}

	property := func(provider string) bool {
		if validProviders[provider] {
			return true // Skip valid providers; tested separately
		}

		_, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
			Provider:          provider,
			SeaweedFSEndpoint: "http://localhost:8889",
			S3Endpoint:        "http://localhost:9000",
			S3AccessKeyID:     "test-key",
			S3SecretAccessKey: "test-secret",
			Bucket:            "test-bucket",
			S3UsePathStyle:    true,
		})

		return err != nil // Invalid providers must always error
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 100}))
}

// TestProperty_NewRepository_SeaweedFS_NoEndpoint_AlwaysErrors verifies that
// the SeaweedFS provider always errors when the endpoint is empty, regardless
// of any other configuration values. This protects against silent misconfiguration.
func TestProperty_NewRepository_SeaweedFS_NoEndpoint_AlwaysErrors(t *testing.T) {
	t.Parallel()

	property := func(bucket string) bool {
		_, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
			Provider:          storage.ProviderSeaweedFS,
			SeaweedFSEndpoint: "", // always empty
			Bucket:            bucket,
		})

		return err != nil // Must always error
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 50}))
}

// TestProperty_NewRepository_SeaweedFS_WithEndpoint_AlwaysSucceeds verifies
// that when SeaweedFS has a non-empty endpoint, the factory always succeeds.
// The SeaweedFS constructor (NewSeaweedFSClient + NewSimpleRepository) is pure
// with no network validation, so any non-empty endpoint should work.
func TestProperty_NewRepository_SeaweedFS_WithEndpoint_AlwaysSucceeds(t *testing.T) {
	t.Parallel()

	property := func(endpoint, bucket string) bool {
		if endpoint == "" {
			return true // Skip empty endpoints; tested separately
		}

		repo, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
			Provider:          storage.ProviderSeaweedFS,
			SeaweedFSEndpoint: endpoint,
			Bucket:            bucket,
		})

		return err == nil && repo != nil
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 50}))
}

// TestProperty_NewRepository_S3_AlwaysSucceeds verifies that the S3 provider
// factory path always succeeds when given valid configuration. The S3 constructor
// does not connect to the endpoint, so any non-empty bucket should produce a
// valid repository. This property holds because the factory ensures a bucket
// is always set (via S3Bucket -> Bucket -> constant fallback).
func TestProperty_NewRepository_S3_AlwaysSucceeds(t *testing.T) {
	t.Parallel()

	property := func(endpoint, region, keyPrefix string) bool {
		repo, err := storage.NewRepository(context.Background(), storage.ProviderConfig{
			Provider:          storage.ProviderS3,
			S3Endpoint:        endpoint,
			S3Region:          region,
			S3KeyPrefix:       keyPrefix,
			S3AccessKeyID:     "test-key",
			S3SecretAccessKey: "test-secret",
			S3UsePathStyle:    true,
			// No S3Bucket or Bucket => falls back to constant.ExternalDataBucketName
		})

		return err == nil && repo != nil
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 50}))
}

// TestProperty_NewRepository_ConstructorNeverPanics verifies that NewRepository
// never panics regardless of input configuration. It either returns a valid
// repository or an error, but never a panic.
func TestProperty_NewRepository_ConstructorNeverPanics(t *testing.T) {
	t.Parallel()

	property := func(provider, endpoint, bucket, s3Endpoint, s3Bucket, s3Key, s3Secret string, pathStyle bool) bool {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Constructor panicked with provider=%q endpoint=%q", provider, endpoint)
			}
		}()

		_, _ = storage.NewRepository(context.Background(), storage.ProviderConfig{
			Provider:          provider,
			SeaweedFSEndpoint: endpoint,
			Bucket:            bucket,
			S3Endpoint:        s3Endpoint,
			S3Bucket:          s3Bucket,
			S3AccessKeyID:     s3Key,
			S3SecretAccessKey: s3Secret,
			S3UsePathStyle:    pathStyle,
		})

		return true // Survived without panic
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 100}))
}
