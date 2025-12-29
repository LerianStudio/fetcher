package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LerianStudio/fetcher/tests/integration/containers/setup"
)

// SeaweedFSClient provides HTTP access to SeaweedFS for result validation.
type SeaweedFSClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewSeaweedFSClient creates a new SeaweedFS client.
func NewSeaweedFSClient(baseURL string) *SeaweedFSClient {
	return &SeaweedFSClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: setup.HTTPClientTimeout,
		},
	}
}

// GetFile downloads a file from SeaweedFS.
func (c *SeaweedFSClient) GetFile(ctx context.Context, path string) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

// FileExists checks if a file exists in SeaweedFS.
func (c *SeaweedFSClient) FileExists(ctx context.Context, path string) (bool, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check file: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// WaitForFile waits for a file to appear in SeaweedFS.
func (c *SeaweedFSClient) WaitForFile(ctx context.Context, path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)

	ticker := time.NewTicker(setup.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for file %s", path)
			}

			data, err := c.GetFile(ctx, path)
			if err == nil {
				return data, nil
			}
			// File not found yet - continue waiting
		}
	}
}

// HealthCheck checks if SeaweedFS is accessible.
func (c *SeaweedFSClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/status", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
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
