package external

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/seaweedfs"
	"github.com/LerianStudio/fetcher/v2/pkg/testutil"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSimpleRepository(t *testing.T) {
	client := seaweedfs.NewSeaweedFSClient("http://localhost:8080")
	bucket := "test-bucket"

	repo := NewSimpleRepository(client, bucket)

	assert.NotNil(t, repo)
	assert.Equal(t, bucket, repo.bucket)
	assert.Equal(t, client, repo.client)
}

func TestSimpleRepository_Get(t *testing.T) {
	tests := []struct {
		name           string
		objectName     string
		serverResponse string
		serverStatus   int
		wantErr        bool
		wantData       []byte
	}{
		{
			name:           "success - get json file",
			objectName:     "test-object",
			serverResponse: `{"key": "value"}`,
			serverStatus:   http.StatusOK,
			wantErr:        false,
			wantData:       []byte(`{"key": "value"}`),
		},
		{
			name:           "success - get empty json",
			objectName:     "empty-object",
			serverResponse: `{}`,
			serverStatus:   http.StatusOK,
			wantErr:        false,
			wantData:       []byte(`{}`),
		},
		{
			name:           "success - get json array",
			objectName:     "array-object",
			serverResponse: `[{"id": 1}, {"id": 2}]`,
			serverStatus:   http.StatusOK,
			wantErr:        false,
			wantData:       []byte(`[{"id": 1}, {"id": 2}]`),
		},
		{
			name:           "error - file not found",
			objectName:     "not-found",
			serverResponse: "not found",
			serverStatus:   http.StatusNotFound,
			wantErr:        true,
			wantData:       nil,
		},
		{
			name:           "error - server error",
			objectName:     "server-error",
			serverResponse: "internal server error",
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
			wantData:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/test-bucket/" + tt.objectName + ".json"
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)

				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := seaweedfs.NewSeaweedFSClient(server.URL)
			repo := NewSimpleRepository(client, "test-bucket")

			data, err := repo.Get(context.Background(), tt.objectName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantData, data)
			}
		})
	}
}

func TestSimpleRepository_Get_WithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name       string
		objectName string
		wantPath   string
	}{
		{
			name:       "object with uuid",
			objectName: "550e8400-e29b-41d4-a716-446655440000",
			wantPath:   "/bucket/550e8400-e29b-41d4-a716-446655440000.json",
		},
		{
			name:       "object with underscores",
			objectName: "my_test_object",
			wantPath:   "/bucket/my_test_object.json",
		},
		{
			name:       "object with nested path",
			objectName: "folder/subfolder/file",
			wantPath:   "/bucket/folder/subfolder/file.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.wantPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			}))
			defer server.Close()

			client := seaweedfs.NewSeaweedFSClient(server.URL)
			repo := NewSimpleRepository(client, "bucket")

			_, err := repo.Get(context.Background(), tt.objectName)
			require.NoError(t, err)
		})
	}
}

func TestSimpleRepository_Put(t *testing.T) {
	tests := []struct {
		name         string
		objectName   string
		data         []byte
		serverStatus int
		wantErr      bool
	}{
		{
			name:         "success - put json data",
			objectName:   "test-object.json",
			data:         []byte(`{"key": "value"}`),
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "success - put with status created",
			objectName:   "new-object.json",
			data:         []byte(`{"new": true}`),
			serverStatus: http.StatusCreated,
			wantErr:      false,
		},
		{
			name:         "success - put empty data",
			objectName:   "empty.json",
			data:         []byte{},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "success - put large data",
			objectName:   "large.json",
			data:         make([]byte, 10000),
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "error - server error",
			objectName:   "error.json",
			data:         []byte(`{"key": "value"}`),
			serverStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name:         "error - forbidden",
			objectName:   "forbidden.json",
			data:         []byte(`{"key": "value"}`),
			serverStatus: http.StatusForbidden,
			wantErr:      true,
		},
		{
			name:         "error - bad request",
			objectName:   "bad.json",
			data:         []byte(`{"key": "value"}`),
			serverStatus: http.StatusBadRequest,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/test-bucket/" + tt.objectName
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, http.MethodPut, r.Method)

				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			client := seaweedfs.NewSeaweedFSClient(server.URL)
			repo := NewSimpleRepository(client, "test-bucket")

			err := repo.Put(context.Background(), tt.objectName, tt.data)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSimpleRepository_Put_VerifiesDataSent(t *testing.T) {
	expectedData := []byte(`{"important": "data", "count": 42}`)
	var receivedData []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedData, err = readAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := seaweedfs.NewSeaweedFSClient(server.URL)
	repo := NewSimpleRepository(client, "bucket")

	err := repo.Put(context.Background(), "verify.json", expectedData)

	require.NoError(t, err)
	assert.Equal(t, expectedData, receivedData)
}

func TestSimpleRepository_Put_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := seaweedfs.NewSeaweedFSClient(server.URL)
	repo := NewSimpleRepository(client, "bucket")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := repo.Put(ctx, "test.json", []byte("data"))

	assert.Error(t, err)
}

func TestSimpleRepository_Get_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := seaweedfs.NewSeaweedFSClient(server.URL)
	repo := NewSimpleRepository(client, "bucket")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := repo.Get(ctx, "test")

	assert.Error(t, err)
}

func TestSimpleRepository_TenantKeyPrefixing(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		objectName string
		operation  string // "get" or "put"
		wantPath   string
	}{
		{
			name:       "get with tenant context prefixes path with tenant ID",
			tenantID:   "org_abc123",
			objectName: "job-result",
			operation:  "get",
			wantPath:   "/test-bucket/org_abc123/job-result.json",
		},
		{
			name:       "get without tenant context uses plain path",
			tenantID:   "",
			objectName: "job-result",
			operation:  "get",
			wantPath:   "/test-bucket/job-result.json",
		},
		{
			name:       "put with tenant context prefixes path with tenant ID",
			tenantID:   "org_xyz789",
			objectName: "output.json",
			operation:  "put",
			wantPath:   "/test-bucket/org_xyz789/output.json",
		},
		{
			name:       "put without tenant context uses plain path",
			tenantID:   "",
			objectName: "output.json",
			operation:  "put",
			wantPath:   "/test-bucket/output.json",
		},
		{
			name:       "get with tenant and nested object name",
			tenantID:   "tenant_42",
			objectName: "sub/folder/data",
			operation:  "get",
			wantPath:   "/test-bucket/tenant_42/sub/folder/data.json",
		},
		{
			name:       "put with tenant and nested object name",
			tenantID:   "tenant_42",
			objectName: "sub/folder/data.json",
			operation:  "put",
			wantPath:   "/test-bucket/tenant_42/sub/folder/data.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				if tt.operation == "get" {
					_, _ = w.Write([]byte(`{"data": "test"}`))
				}
			}))
			defer server.Close()

			client := seaweedfs.NewSeaweedFSClient(server.URL)
			repo := NewSimpleRepository(client, "test-bucket")

			ctx := testutil.TestContext()
			if tt.tenantID != "" {
				ctx = tmcore.ContextWithTenantID(ctx, tt.tenantID)
			}

			switch tt.operation {
			case "get":
				_, err := repo.Get(ctx, tt.objectName)
				require.NoError(t, err)
			case "put":
				err := repo.Put(ctx, tt.objectName, []byte(`{"data": "test"}`))
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantPath, capturedPath,
				"expected path %q, got %q", tt.wantPath, capturedPath)
		})
	}
}

// readAll is a helper to read all data from a reader
func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var result []byte
	buf := make([]byte, 1024)

	for {
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}

		if err != nil {
			if err.Error() == "EOF" {
				break
			}

			return nil, err
		}
	}

	return result, nil
}
