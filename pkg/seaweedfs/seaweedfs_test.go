package seaweedfs

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newClient(t *testing.T, handler http.HandlerFunc, opts ...Option) *SeaweedFSClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewSeaweedFSClient(srv.URL, opts...)
}

func TestNewSeaweedFSClientWithMaxDownloadSize(t *testing.T) {
	client := NewSeaweedFSClient("http://example.com", WithMaxDownloadSize(256))
	if client.httpClient == nil {
		t.Fatalf("expected httpClient to be initialized")
	}
	if client.maxDownloadSize != 256 {
		t.Fatalf("expected maxDownloadSize to be 256, got %d", client.maxDownloadSize)
	}
}

func TestUploadFileSuccess(t *testing.T) {
	var method, path, body string
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.WriteHeader(http.StatusCreated)
	})

	err := client.UploadFile(context.Background(), "/file.txt", []byte("content"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if method != http.MethodPut {
		t.Fatalf("expected PUT method, got %s", method)
	}
	if path != "/file.txt" {
		t.Fatalf("unexpected path: %s", path)
	}
	if body != "content" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestUploadFileWithTTL(t *testing.T) {
	var ttl string
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		ttl = r.URL.Query().Get("ttl")
		w.WriteHeader(http.StatusOK)
	})

	err := client.UploadFileWithTTL(context.Background(), "/file.txt", []byte("data"), "1m")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ttl != "1m" {
		t.Fatalf("expected ttl to be 1m, got %s", ttl)
	}
}

func TestUploadFileErrorStatus(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("fail"))
	})

	err := client.UploadFile(context.Background(), "/fail", []byte("data"))
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected upload error with status, got %v", err)
	}
}

func TestDownloadFileSuccess(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("download-data"))
	})

	data, err := client.DownloadFile(context.Background(), "/download")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(data) != "download-data" {
		t.Fatalf("unexpected data: %s", string(data))
	}
}

func TestDownloadFileWithLimit(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("123456789"))
	}, WithMaxDownloadSize(4))

	data, err := client.DownloadFile(context.Background(), "/limited")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(data) != "1234" {
		t.Fatalf("expected limited data, got %s", string(data))
	}
}

func TestDownloadFileErrorStatus(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("missing"))
	})

	_, err := client.DownloadFile(context.Background(), "/missing")
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected download error with status, got %v", err)
	}
}

func TestDownloadFileWithStreamSuccess(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("stream-data"))
	})

	rc, err := client.DownloadFileWithStream(context.Background(), "/stream")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("expected to read stream, got %v", err)
	}
	if string(data) != "stream-data" {
		t.Fatalf("unexpected stream data: %s", string(data))
	}
}

func TestDownloadFileWithStreamErrorStatus(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(strings.Repeat("x", 1500)))
	})

	_, err := client.DownloadFileWithStream(context.Background(), "/stream-fail")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Count(err.Error(), "x") != 1024 {
		t.Fatalf("expected error message to include limited body, got %d chars", strings.Count(err.Error(), "x"))
	}
}

func TestDeleteFileSuccess(t *testing.T) {
	var method, path string
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteFile(context.Background(), "/delete-me")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if method != http.MethodDelete {
		t.Fatalf("expected DELETE method, got %s", method)
	}
	if path != "/delete-me" {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestDeleteFileErrorStatus(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("cannot delete"))
	})

	err := client.DeleteFile(context.Background(), "/delete-fail")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected delete error with status, got %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			t.Fatalf("unexpected health path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := client.HealthCheck(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHealthCheckError(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	if err := client.HealthCheck(context.Background()); err == nil {
		t.Fatalf("expected health check error")
	}
}

func TestPathIsValid(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "adds leading slash", input: "file", want: "/file"},
		{name: "keeps leading slash", input: "/file", want: "/file"},
		{name: "rejects traversal", input: "../file", wantErr: true},
		{name: "rejects http", input: "http://evil", wantErr: true},
		{name: "rejects https", input: "https://evil", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pathIsValid(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %s", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}
