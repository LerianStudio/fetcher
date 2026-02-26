//go:build unit

package storage_test

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/storage"
)

// FuzzNewRepository_Provider verifies that NewRepository never panics with
// arbitrary provider strings. Valid providers produce repositories; invalid
// providers produce errors. Neither case should panic.
//
// Corpus covers: both valid providers, empty string (default), common cloud
// names, path traversal attempts, control characters, and unicode strings.
func FuzzNewRepository_Provider(f *testing.F) {
	// Seed corpus: minimum 5 entries with edge cases per Ring testing-fuzz.md
	f.Add("seaweedfs")
	f.Add("s3")
	f.Add("")
	f.Add("gcs")
	f.Add("azure")
	f.Add("../traversal")
	f.Add(string([]byte{0x00, 0xFF}))
	f.Add("SEAWEEDFS")
	f.Add("S3")
	f.Add(" ")

	f.Fuzz(func(t *testing.T, provider string) {
		// Bound input length to prevent resource exhaustion
		if len(provider) > 512 {
			provider = provider[:512]
		}

		// Should never panic regardless of provider string
		_, _ = storage.NewRepository(context.Background(), storage.ProviderConfig{
			Provider:          provider,
			SeaweedFSEndpoint: "http://localhost:8889",
			S3Endpoint:        "http://localhost:9000",
			S3AccessKeyID:     "test-key",
			S3SecretAccessKey: "test-secret",
			Bucket:            "test-bucket",
			S3UsePathStyle:    true,
		})
	})
}

// FuzzNewRepository_SeaweedFSEndpoint verifies that NewRepository never panics
// with arbitrary SeaweedFS endpoint strings when using the SeaweedFS provider.
//
// Corpus covers: valid HTTP/HTTPS endpoints, empty string, malformed URLs,
// path traversal, unicode, and control characters.
func FuzzNewRepository_SeaweedFSEndpoint(f *testing.F) {
	f.Add("http://localhost:8889")
	f.Add("https://seaweedfs.example.com")
	f.Add("")
	f.Add("not-a-url")
	f.Add("http://")
	f.Add(string([]byte{0x00, 0x01, 0x02}))
	f.Add("http://日本語.example.com")

	f.Fuzz(func(t *testing.T, endpoint string) {
		if len(endpoint) > 512 {
			endpoint = endpoint[:512]
		}

		_, _ = storage.NewRepository(context.Background(), storage.ProviderConfig{
			Provider:          storage.ProviderSeaweedFS,
			SeaweedFSEndpoint: endpoint,
			Bucket:            "test-bucket",
		})
	})
}

// FuzzNewRepository_S3Bucket verifies that NewRepository never panics with
// arbitrary bucket names when using the S3 provider. The factory has a
// three-level bucket fallback (S3Bucket -> Bucket -> constant default), so
// this exercises the bucket resolution logic with random inputs.
//
// Corpus covers: valid bucket names, empty strings, uppercase, long names,
// slashes, spaces, and special characters.
func FuzzNewRepository_S3Bucket(f *testing.F) {
	f.Add("valid-bucket", "fallback-bucket")
	f.Add("", "")
	f.Add("", "fallback-only")
	f.Add("primary-only", "")
	f.Add("UPPER-CASE", "lower-case")
	f.Add("bucket/with/slashes", "bucket name with spaces")
	f.Add(string([]byte{0x00}), "normal")

	f.Fuzz(func(t *testing.T, s3Bucket, bucket string) {
		if len(s3Bucket) > 256 {
			s3Bucket = s3Bucket[:256]
		}

		if len(bucket) > 256 {
			bucket = bucket[:256]
		}

		_, _ = storage.NewRepository(context.Background(), storage.ProviderConfig{
			Provider:          storage.ProviderS3,
			S3Endpoint:        "http://localhost:9000",
			S3Bucket:          s3Bucket,
			Bucket:            bucket,
			S3AccessKeyID:     "test-key",
			S3SecretAccessKey: "test-secret",
			S3UsePathStyle:    true,
		})
	})
}
