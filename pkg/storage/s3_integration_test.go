//go:build integration

package storage_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/itestkit"
	"github.com/LerianStudio/fetcher/v2/pkg/itestkit/infra/minio"
	"github.com/LerianStudio/fetcher/v2/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBucketName = "test-external-data"

// startMinIOInfra starts a MinIO container via itestkit and registers cleanup with t.Cleanup.
// The returned *minio.MinioInfra is ready to use: the container is healthy and the bucket
// testBucketName already exists.
//
// Each test calling this function gets its own isolated container, preventing state leakage
// between test cases.
func startMinIOInfra(t *testing.T) *minio.MinioInfra {
	t.Helper()

	ctx := context.Background()

	minioInfra := minio.NewMinioInfra(minio.MinioConfig{
		Bucket: testBucketName,
	})

	suite, err := itestkit.New(t).
		WithInfra(minioInfra).
		Build(ctx)
	require.NoError(t, err, "failed to start MinIO infrastructure")

	t.Cleanup(func() {
		_ = suite.Terminate(context.Background())
	})

	return minioInfra
}

// newTestRepository creates an S3Repository configured to talk to the given MinIO instance.
// Connection details (endpoint, credentials, region, bucket) are read directly from minioInfra,
// eliminating the need to repeat configuration values across tests.
func newTestRepository(t *testing.T, minioInfra *minio.MinioInfra) *storage.S3Repository {
	t.Helper()

	endpointURL, err := minioInfra.URL()
	require.NoError(t, err, "failed to get MinIO endpoint URL")

	repo, err := storage.NewS3Repository(context.Background(), storage.S3Config{
		Endpoint:        endpointURL,
		Region:          minioInfra.Region(),
		Bucket:          minioInfra.Bucket(),
		AccessKeyID:     minioInfra.AccessKeyID(),
		SecretAccessKey: minioInfra.SecretAccessKey(),
		UsePathStyle:    true,
	})
	require.NoError(t, err, "failed to create S3Repository")

	return repo
}

// TestIntegration_S3Repository_PutAndGetRoundTrip verifies that data
// uploaded to S3 via Put can be retrieved exactly via Get, with matching byte content.
// IS-1: Put data then Get same data → bytes match (round-trip)
func TestIntegration_S3Repository_PutAndGetRoundTrip(t *testing.T) {
	repo := newTestRepository(t, startMinIOInfra(t))

	ctx := context.Background()
	objectName := "test-job-123.json"
	data := []byte(`{"jobID":"123","result":"encrypted-data"}`)

	// Act: Put the object
	err := repo.Put(ctx, objectName, data)
	require.NoError(t, err, "Put operation failed")

	// Assert: Get the object and verify bytes match
	retrieved, err := repo.Get(ctx, objectName)
	require.NoError(t, err, "Get operation failed")

	assert.True(t, bytes.Equal(data, retrieved),
		"round-trip data mismatch: expected %q, got %q", string(data), string(retrieved))
}

// TestIntegration_S3Repository_GetNonExistentReturnsNotFound verifies that
// attempting to retrieve a non-existent object returns an error with "object not found" message.
// IS-2: Get non-existent object → returns "object not found" error
func TestIntegration_S3Repository_GetNonExistentReturnsNotFound(t *testing.T) {
	repo := newTestRepository(t, startMinIOInfra(t))

	ctx := context.Background()

	// Act: Attempt to get non-existent object
	_, err := repo.Get(ctx, "non-existent-key.json")

	// Assert: Error is returned with appropriate message
	require.Error(t, err, "Get should return error for non-existent object")
	assert.Contains(t, err.Error(), "object not found",
		"error message should indicate object was not found")
}

// TestIntegration_S3Repository_KeyPrefixPutAndGet verifies that when
// S3Repository is configured with a KeyPrefix, Put and Get operations
// correctly prepend the prefix to object keys, and data round-trips correctly.
// IS-3: Put with KeyPrefix → Get with same prefix returns data
func TestIntegration_S3Repository_KeyPrefixPutAndGet(t *testing.T) {
	minioInfra := startMinIOInfra(t)

	endpointURL, err := minioInfra.URL()
	require.NoError(t, err, "failed to get MinIO endpoint URL")

	// Create repository with KeyPrefix (variant not covered by the default helper)
	repo, err := storage.NewS3Repository(context.Background(), storage.S3Config{
		Endpoint:        endpointURL,
		Region:          minioInfra.Region(),
		Bucket:          minioInfra.Bucket(),
		KeyPrefix:       "external-data/",
		AccessKeyID:     minioInfra.AccessKeyID(),
		SecretAccessKey: minioInfra.SecretAccessKey(),
		UsePathStyle:    true,
	})
	require.NoError(t, err, "failed to create S3Repository with KeyPrefix")

	ctx := context.Background()
	objectName := "job-456.json"
	data := []byte(`{"jobID":"456","result":"data-with-prefix"}`)

	// Act: Put and Get with KeyPrefix
	err = repo.Put(ctx, objectName, data)
	require.NoError(t, err, "Put operation failed")

	retrieved, err := repo.Get(ctx, objectName)
	require.NoError(t, err, "Get operation failed")

	// Assert: Data round-trips correctly with prefix applied
	assert.True(t, bytes.Equal(data, retrieved),
		"round-trip data with prefix mismatch: expected %q, got %q", string(data), string(retrieved))
}

// TestIntegration_S3Repository_PutEmptyData verifies that S3Repository
// correctly handles empty byte slices, storing and retrieving them without error.
// IS-4: Put empty data → Get returns empty bytes
func TestIntegration_S3Repository_PutEmptyData(t *testing.T) {
	repo := newTestRepository(t, startMinIOInfra(t))

	ctx := context.Background()
	objectName := "empty-job.json"
	data := []byte{}

	// Act: Put empty data
	err := repo.Put(ctx, objectName, data)
	require.NoError(t, err, "Put empty data failed")

	// Assert: Get returns empty bytes
	retrieved, err := repo.Get(ctx, objectName)
	require.NoError(t, err, "Get empty data failed")

	assert.Empty(t, retrieved, "Get should return empty bytes for empty data")
	assert.True(t, bytes.Equal(data, retrieved),
		"empty data round-trip failed: expected %q, got %q", string(data), string(retrieved))
}
