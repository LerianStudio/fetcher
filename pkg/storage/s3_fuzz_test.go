//go:build unit

package storage_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/storage"
	"github.com/LerianStudio/fetcher/pkg/testutil"
)

// FuzzS3ObjectName verifies that S3Repository.Get does not panic with arbitrary object names.
// The S3 API accepts any UTF-8 string as an object key, so we verify no crashes occur.
// Corpus covers: normal keys, empty keys, path traversal attempts, nested paths, spaces, control characters, unicode.
func FuzzS3ObjectName(f *testing.F) {
	// Seed corpus: 5+ entries covering important edge cases
	f.Add("normal-key.json")
	f.Add("")                          // empty key
	f.Add("../traversal")              // path traversal attempt
	f.Add("path/with/slashes/key.json") // nested path
	f.Add("key with spaces")            // spaces
	f.Add(string([]byte{0x00, 0x01, 0x02})) // control chars
	f.Add("ünïcödé-key.json")          // unicode
	f.Add("../../etc/passwd")          // deeper traversal
	f.Add("/absolute/path/key")        // absolute path

	// Create a test server that returns 404 NoSuchKey for any request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`)
	}))
	defer server.Close()

	f.Fuzz(func(t *testing.T, objectName string) {
		ctx := testutil.TestContext()

		repo, err := storage.NewS3Repository(ctx, storage.S3Config{
			Endpoint:        server.URL,
			Region:          "us-east-1",
			Bucket:          "test-bucket",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			UsePathStyle:    true,
		})
		if err != nil {
			return // Skip invalid configs
		}

		// Should not panic regardless of objectName value
		result, err := repo.Get(context.Background(), objectName)
		// We don't care about the error (NoSuchKey expected), only that no panic occurs
		_ = result
		_ = err
	})
}

// FuzzS3Config verifies NewS3Repository handles arbitrary bucket names without panicking.
// Corpus covers: valid buckets, empty buckets, uppercase, long names, invalid characters.
func FuzzS3Config(f *testing.F) {
	// Seed corpus: 5+ bucket name patterns
	f.Add("valid-bucket")
	f.Add("")                    // empty bucket — should return error, not panic
	f.Add("UPPER-CASE")          // uppercase (invalid, but should not panic)
	f.Add("very-long-bucket-name-that-is-quite-long-but-still-valid-name-here")
	f.Add("bucket/with/slashes") // invalid bucket name
	f.Add("bucket name")         // space in bucket name
	f.Add("bucket_underscore")   // underscore (invalid, but should not panic)
	f.Add("a")                   // single character

	f.Fuzz(func(t *testing.T, bucket string) {
		ctx := testutil.TestContext()

		// Should never panic regardless of input
		repo, err := storage.NewS3Repository(ctx, storage.S3Config{
			Endpoint:        "http://localhost:9999", // non-existent, we won't connect
			Region:          "us-east-1",
			Bucket:          bucket,
			UsePathStyle:    true,
		})
		_ = repo
		_ = err
		// Only requirement: no panic
	})
}

// FuzzS3KeyPrefixCombination verifies that arbitrary KeyPrefix and objectName combinations
// do not cause panics when constructing S3 keys.
// Corpus covers: various prefix/name combinations including edge cases.
func FuzzS3KeyPrefixCombination(f *testing.F) {
	// Seed corpus: prefix + objectName combinations
	f.Add("prefix/", "objectname")
	f.Add("", "objectname")
	f.Add("prefix/", "")
	f.Add("", "")
	f.Add("prefix/with/multiple/levels/", "deep/object/path")
	f.Add("../traversal/", "file.json")
	f.Add("prefix with spaces/", "object with spaces")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`)
	}))
	defer server.Close()

	f.Fuzz(func(t *testing.T, keyPrefix string, objectName string) {
		ctx := testutil.TestContext()

		repo, err := storage.NewS3Repository(ctx, storage.S3Config{
			Endpoint:        server.URL,
			Region:          "us-east-1",
			Bucket:          "test-bucket",
			KeyPrefix:       keyPrefix,
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			UsePathStyle:    true,
		})
		if err != nil {
			return // Skip invalid configs
		}

		// Should not panic when constructing key from prefix + objectName
		result, err := repo.Get(context.Background(), objectName)
		_ = result
		_ = err
	})
}
