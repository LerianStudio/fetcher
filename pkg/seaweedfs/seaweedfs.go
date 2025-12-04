package seaweedfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SeaweedFSClient provides direct HTTP access to SeaweedFS
type SeaweedFSClient struct {
	baseURL         string
	httpClient      *http.Client
	maxDownloadSize int64
}

// Option configures the SeaweedFSClient.
type Option func(*SeaweedFSClient)

// NewSeaweedFSClient creates a new simple HTTP client for SeaweedFS
func NewSeaweedFSClient(baseURL string, opts ...Option) *SeaweedFSClient {
	client := &SeaweedFSClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// WithMaxDownloadSize limits the number of bytes read in DownloadFile. Values <= 0 disable the limit.
func WithMaxDownloadSize(limit int64) Option {
	return func(c *SeaweedFSClient) {
		if limit > 0 {
			c.maxDownloadSize = limit
		}
	}
}

// UploadFile uploads a file to SeaweedFS
func (c *SeaweedFSClient) UploadFile(ctx context.Context, path string, data []byte) error {
	path, err := pathIsValid(path)
	if err != nil {
		return err
	}

	return c.UploadFileWithTTL(ctx, path, data, "")
}

// UploadFileWithTTL uploads a file to SeaweedFS with optional TTL
func (c *SeaweedFSClient) UploadFileWithTTL(ctx context.Context, path string, data []byte, ttl string) error {
	path, err := pathIsValid(path)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)
	if ttl != "" {
		reqURL = fmt.Sprintf("%s?ttl=%s", reqURL, ttl)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DownloadFile downloads a file from SeaweedFS
func (c *SeaweedFSClient) DownloadFile(ctx context.Context, path string) ([]byte, error) {
	path, err := pathIsValid(path)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	reader := io.Reader(resp.Body)
	if c.maxDownloadSize > 0 {
		reader = io.LimitReader(resp.Body, c.maxDownloadSize)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

// DownloadFileWithStream downloads a file returning a streaming reader instead of eagerly reading the body.
func (c *SeaweedFSClient) DownloadFileWithStream(ctx context.Context, path string) (io.ReadCloser, error) {
	path, err := pathIsValid(path)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if closeErr := resp.Body.Close(); closeErr != nil {
			return nil, fmt.Errorf("download failed with status %d and error closing response body: %w", resp.StatusCode, closeErr)
		}

		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

// DeleteFile deletes a file from SeaweedFS
func (c *SeaweedFSClient) DeleteFile(ctx context.Context, path string) error {
	path, err := pathIsValid(path)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// HealthCheck checks if SeaweedFS is accessible
func (c *SeaweedFSClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/status", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// pathIsValid checks for potential security issues in the path
func pathIsValid(path string) (string, error) {
	if strings.Contains(path, "..") || strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "", fmt.Errorf("invalid path: potential security issue")
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path, nil
}
