//go:build unit

package storage_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/storage"
	"github.com/LerianStudio/fetcher/v2/pkg/testutil"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewS3Repository_ValidConfig tests that NewS3Repository returns a valid
// repository when given a complete, well-formed S3Config. This is the happy-path
// constructor test verifying that the returned repository is non-nil and satisfies
// the storage.Repository port interface.
func TestNewS3Repository_ValidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  storage.S3Config
	}{
		{
			name: "complete_config_with_path_style",
			cfg: storage.S3Config{
				Endpoint:        "http://localhost:9000",
				Region:          "us-east-1",
				Bucket:          "test-bucket",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UsePathStyle:    true,
			},
		},
		{
			name: "complete_config_without_path_style",
			cfg: storage.S3Config{
				Endpoint:        "https://s3.amazonaws.com",
				Region:          "eu-west-1",
				Bucket:          "production-bucket",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
		},
		{
			name: "config_with_empty_region_uses_default",
			cfg: storage.S3Config{
				Endpoint:        "http://localhost:9000",
				Region:          "",
				Bucket:          "test-bucket",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
			},
		},
		{
			name: "config_with_empty_endpoint_uses_aws_defaults",
			cfg: storage.S3Config{
				Endpoint:        "",
				Region:          "us-east-1",
				Bucket:          "test-bucket",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, err := storage.NewS3Repository(context.Background(), tt.cfg)
			require.NoError(t, err)
			require.NotNil(t, repo)
		})
	}
}

// TestNewS3Repository_InvalidConfig tests that NewS3Repository returns an error
// for configurations that violate mandatory field constraints. An empty bucket
// name is the primary validation boundary since bucket is required for all S3
// operations.
func TestNewS3Repository_InvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     storage.S3Config
		wantErr string
	}{
		{
			name: "empty_bucket_returns_error",
			cfg: storage.S3Config{
				Endpoint:        "http://localhost:9000",
				Region:          "us-east-1",
				Bucket:          "",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
			},
			wantErr: "bucket",
		},
		{
			name: "partial_credentials_only_access_key",
			cfg: storage.S3Config{
				Endpoint:    "http://localhost:9000",
				Bucket:      "test-bucket",
				AccessKeyID: "minioadmin",
			},
			wantErr: "partial credentials",
		},
		{
			name: "partial_credentials_only_secret_key",
			cfg: storage.S3Config{
				Endpoint:        "http://localhost:9000",
				Bucket:          "test-bucket",
				SecretAccessKey: "minioadmin",
			},
			wantErr: "partial credentials",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, err := storage.NewS3Repository(context.Background(), tt.cfg)
			require.Error(t, err)
			assert.Nil(t, repo)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestS3Config_Fields verifies that the S3Config struct exposes the expected
// fields with correct types. This is a compile-time contract test: if any
// field is renamed, removed, or changes type, this test will fail to compile.
func TestS3Config_Fields(t *testing.T) {
	t.Parallel()

	cfg := storage.S3Config{
		Endpoint:        "http://localhost:9000",
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		KeyPrefix:       "data/",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UsePathStyle:    true,
	}

	assert.Equal(t, "http://localhost:9000", cfg.Endpoint)
	assert.Equal(t, "us-east-1", cfg.Region)
	assert.Equal(t, "test-bucket", cfg.Bucket)
	assert.Equal(t, "data/", cfg.KeyPrefix)
	assert.Equal(t, "minioadmin", cfg.AccessKeyID)
	assert.Equal(t, "minioadmin", cfg.SecretAccessKey)
	assert.True(t, cfg.UsePathStyle)
}

// createTestClientWithServer creates an S3Repository backed by an httptest.Server.
// The handler controls what the fake S3 endpoint returns, allowing unit tests to
// exercise Get/Put logic without a real S3 service. The caller must defer
// server.Close() after calling this helper.
func createTestClientWithServer(t *testing.T, handler http.HandlerFunc, cfg ...storage.S3Config) (*storage.S3Repository, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)

	baseCfg := storage.S3Config{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UsePathStyle:    true,
	}

	if len(cfg) > 0 {
		baseCfg.KeyPrefix = cfg[0].KeyPrefix

		if cfg[0].Bucket != "" {
			baseCfg.Bucket = cfg[0].Bucket
		}
	}

	ctx := testutil.TestContext()

	repo, err := storage.NewS3Repository(ctx, baseCfg)
	require.NoError(t, err)
	require.NotNil(t, repo)

	return repo, server
}

// TestS3Repository_Get_Success verifies that Get returns the bytes from the S3
// object body when the download succeeds (AC-4). The httptest server responds
// with 200 OK and the expected payload.
func TestS3Repository_Get_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		objectName string
		wantData   string
	}{
		{
			name:       "returns_bytes_from_response_body",
			objectName: "test-object.json",
			wantData:   `{"key":"value"}`,
		},
		{
			name:       "returns_binary_data",
			objectName: "encrypted-blob.bin",
			wantData:   "encrypted-binary-data-here",
		},
		{
			name:       "returns_empty_body",
			objectName: "empty.dat",
			wantData:   "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.wantData))
			}

			repo, server := createTestClientWithServer(t, handler)
			defer server.Close()

			ctx := testutil.TestContext()

			// Act
			data, err := repo.Get(ctx, tt.objectName)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, []byte(tt.wantData), data)
		})
	}
}

// TestS3Repository_Get_NoSuchKey verifies that Get returns an error containing
// "object not found" when S3 responds with a NoSuchKey error (AC-3). The AWS SDK
// parses the S3 XML error response and produces a *types.NoSuchKey which the
// repository translates into a domain-friendly message.
func TestS3Repository_Get_NoSuchKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		objectName string
	}{
		{
			name:       "missing_object_returns_not_found_error",
			objectName: "missing-key.json",
		},
		{
			name:       "nonexistent_path_returns_not_found_error",
			objectName: "deep/nested/path/file.bin",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<Error>
    <Code>NoSuchKey</Code>
    <Message>The specified key does not exist.</Message>
    <Key>%s</Key>
    <RequestId>test-request-id</RequestId>
</Error>`, tt.objectName)
			}

			repo, server := createTestClientWithServer(t, handler)
			defer server.Close()

			ctx := testutil.TestContext()

			// Act
			data, err := repo.Get(ctx, tt.objectName)

			// Assert
			require.Error(t, err)
			assert.Nil(t, data)
			assert.Contains(t, err.Error(), "object not found")
			assert.Contains(t, err.Error(), tt.objectName)
		})
	}
}

// TestS3Repository_Get_Error verifies that Get returns a wrapped error when S3
// returns a non-NoSuchKey error (e.g., 500 Internal Server Error). This ensures
// the general error path is covered beyond the specific NoSuchKey handling.
func TestS3Repository_Get_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		objectName string
		status     int
		xmlCode    string
	}{
		{
			name:       "internal_server_error_returns_download_failed",
			objectName: "error-object.json",
			status:     http.StatusInternalServerError,
			xmlCode:    "InternalError",
		},
		{
			name:       "access_denied_returns_download_failed",
			objectName: "forbidden-object.json",
			status:     http.StatusForbidden,
			xmlCode:    "AccessDenied",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<Error>
    <Code>%s</Code>
    <Message>Server error</Message>
    <RequestId>test-request-id</RequestId>
</Error>`, tt.xmlCode)
			}

			repo, server := createTestClientWithServer(t, handler)
			defer server.Close()

			ctx := testutil.TestContext()

			// Act
			data, err := repo.Get(ctx, tt.objectName)

			// Assert
			require.Error(t, err)
			assert.Nil(t, data)
			assert.Contains(t, err.Error(), "s3 download failed")
			assert.Contains(t, err.Error(), tt.objectName)
		})
	}
}

// TestS3Repository_Put_Success verifies that Put returns nil error when S3
// accepts the upload (AC-6). The httptest server responds with 200 OK and an
// ETag, which is the standard S3 success response for PutObject.
func TestS3Repository_Put_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		objectName string
		data       []byte
	}{
		{
			name:       "uploads_json_data_successfully",
			objectName: "output.json",
			data:       []byte(`{"result":"success"}`),
		},
		{
			name:       "uploads_binary_data_successfully",
			objectName: "encrypted.bin",
			data:       []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
		},
		{
			name:       "uploads_empty_data_successfully",
			objectName: "empty.dat",
			data:       []byte{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			handler := func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPut, r.Method)
				w.Header().Set("ETag", `"abc123"`)
				w.WriteHeader(http.StatusOK)
			}

			repo, server := createTestClientWithServer(t, handler)
			defer server.Close()

			ctx := testutil.TestContext()

			// Act
			err := repo.Put(ctx, tt.objectName, tt.data)

			// Assert
			require.NoError(t, err)
		})
	}
}

// TestS3Repository_Put_Error verifies that Put returns a wrapped error when S3
// rejects the upload (AC-5). For example, an AccessDenied (403) response should
// produce an error containing "s3 upload failed".
func TestS3Repository_Put_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		objectName string
		status     int
		xmlCode    string
	}{
		{
			name:       "access_denied_returns_upload_failed",
			objectName: "restricted.json",
			status:     http.StatusForbidden,
			xmlCode:    "AccessDenied",
		},
		{
			name:       "internal_error_returns_upload_failed",
			objectName: "error-upload.json",
			status:     http.StatusInternalServerError,
			xmlCode:    "InternalError",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<Error>
    <Code>%s</Code>
    <Message>Access Denied</Message>
    <RequestId>test-request-id</RequestId>
</Error>`, tt.xmlCode)
			}

			repo, server := createTestClientWithServer(t, handler)
			defer server.Close()

			ctx := testutil.TestContext()

			// Act
			err := repo.Put(ctx, tt.objectName, []byte("test-data"))

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "s3 upload failed")
			assert.Contains(t, err.Error(), tt.objectName)
		})
	}
}

// TestS3Repository_KeyPrefix verifies that KeyPrefix from S3Config is prepended
// to the objectName in both Get and Put operations (AC-7). The httptest handler
// inspects the incoming request URL path to confirm the key prefix is present.
func TestS3Repository_KeyPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		keyPrefix   string
		objectName  string
		expectedKey string
	}{
		{
			name:        "prefix_prepended_to_object_name",
			keyPrefix:   "data/",
			objectName:  "file.json",
			expectedKey: "data/file.json",
		},
		{
			name:        "nested_prefix_prepended_to_object_name",
			keyPrefix:   "org/tenant/results/",
			objectName:  "output.bin",
			expectedKey: "org/tenant/results/output.bin",
		},
		{
			name:        "empty_prefix_uses_plain_object_name",
			keyPrefix:   "",
			objectName:  "plain.json",
			expectedKey: "plain.json",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name+"_get", func(t *testing.T) {
			t.Parallel()

			// Arrange
			var capturedPath string

			handler := func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.Header().Set("Content-Type", "application/octet-stream")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("test-content"))
			}

			repo, server := createTestClientWithServer(t, handler, storage.S3Config{
				KeyPrefix: tt.keyPrefix,
			})
			defer server.Close()

			ctx := testutil.TestContext()

			// Act
			_, err := repo.Get(ctx, tt.objectName)

			// Assert
			require.NoError(t, err)
			assert.True(t, strings.Contains(capturedPath, tt.expectedKey),
				"expected path to contain %q, got %q", tt.expectedKey, capturedPath)
		})

		t.Run(tt.name+"_put", func(t *testing.T) {
			t.Parallel()

			// Arrange
			var capturedPath string

			handler := func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.Header().Set("ETag", `"abc123"`)
				w.WriteHeader(http.StatusOK)
			}

			repo, server := createTestClientWithServer(t, handler, storage.S3Config{
				KeyPrefix: tt.keyPrefix,
			})
			defer server.Close()

			ctx := testutil.TestContext()

			// Act
			err := repo.Put(ctx, tt.objectName, []byte("test-data"))

			// Assert
			require.NoError(t, err)
			assert.True(t, strings.Contains(capturedPath, tt.expectedKey),
				"expected path to contain %q, got %q", tt.expectedKey, capturedPath)
		})
	}
}

// TestS3Repository_TenantKeyPrefix verifies that when tenant context is set,
// the objectName is prefixed with the tenant ID in both Get and Put operations.
// In single-tenant mode (no tenant in context), the key is used unchanged.
func TestS3Repository_TenantKeyPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tenantID      string
		keyPrefix     string
		objectName    string
		expectedInKey string
	}{
		{
			name:          "tenant_prefix_added_to_get_put",
			tenantID:      "org_abc123",
			keyPrefix:     "data/",
			objectName:    "job-id.json",
			expectedInKey: "data/org_abc123/job-id.json",
		},
		{
			name:          "no_tenant_uses_plain_key",
			tenantID:      "",
			keyPrefix:     "data/",
			objectName:    "job-id.json",
			expectedInKey: "data/job-id.json",
		},
		{
			name:          "tenant_with_empty_prefix",
			tenantID:      "org_xyz",
			keyPrefix:     "",
			objectName:    "result.json",
			expectedInKey: "org_xyz/result.json",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name+"_get", func(t *testing.T) {
			t.Parallel()

			var capturedPath string

			handler := func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.Header().Set("Content-Type", "application/octet-stream")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("test-content"))
			}

			repo, server := createTestClientWithServer(t, handler, storage.S3Config{
				KeyPrefix: tt.keyPrefix,
			})
			defer server.Close()

			ctx := testutil.TestContext()
			if tt.tenantID != "" {
				ctx = tmcore.ContextWithTenantID(ctx, tt.tenantID)
			}

			_, err := repo.Get(ctx, tt.objectName)

			require.NoError(t, err)
			assert.True(t, strings.Contains(capturedPath, tt.expectedInKey),
				"expected path to contain %q, got %q", tt.expectedInKey, capturedPath)
		})

		t.Run(tt.name+"_put", func(t *testing.T) {
			t.Parallel()

			var capturedPath string

			handler := func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.Header().Set("ETag", `"abc123"`)
				w.WriteHeader(http.StatusOK)
			}

			repo, server := createTestClientWithServer(t, handler, storage.S3Config{
				KeyPrefix: tt.keyPrefix,
			})
			defer server.Close()

			ctx := testutil.TestContext()
			if tt.tenantID != "" {
				ctx = tmcore.ContextWithTenantID(ctx, tt.tenantID)
			}

			err := repo.Put(ctx, tt.objectName, []byte("test-data"))

			require.NoError(t, err)
			assert.True(t, strings.Contains(capturedPath, tt.expectedInKey),
				"expected path to contain %q, got %q", tt.expectedInKey, capturedPath)
		})
	}
}
