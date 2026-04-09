package seaweedfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"go.opentelemetry.io/otel/attribute"
)

// Client defines the interface for SeaweedFS operations
//
//go:generate mockgen --destination=seaweedfs.mock.go --package=seaweedfs -mock_names Client=MockSeaweedFSClient . Client
type Client interface {
	UploadFile(ctx context.Context, path string, data []byte) error
	UploadFileWithTTL(ctx context.Context, path string, data []byte, ttl string) error
	DownloadFile(ctx context.Context, path string) ([]byte, error)
	DownloadFileWithStream(ctx context.Context, path string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, path string) error
	HealthCheck(ctx context.Context) error
}

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
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "seaweedfs.file.upload")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("seaweedfs.path", path),
		attribute.Int("seaweedfs.data_size", len(data)),
	)

	path, err := pathIsValid(path)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Invalid path", err)
		return err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)
	if ttl != "" {
		reqURL = fmt.Sprintf("%s?ttl=%s", reqURL, ttl)
		span.SetAttributes(attribute.String("seaweedfs.ttl", ttl))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(data))
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to create upload request", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to upload file", err)
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		uploadErr := fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Upload failed with client error", uploadErr)
		} else {
			libOpentelemetry.HandleSpanError(span, "Upload failed with server error", uploadErr)
		}

		return uploadErr
	}

	logger.Log(ctx, libLog.LevelDebug, fmt.Sprintf("SeaweedFS upload completed: %s", path))

	return nil
}

// DownloadFile downloads a file from SeaweedFS
func (c *SeaweedFSClient) DownloadFile(ctx context.Context, path string) ([]byte, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "seaweedfs.file.download")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("seaweedfs.path", path),
	)

	path, err := pathIsValid(path)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Invalid path", err)
		return nil, err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to create download request", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to download file", err)
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		downloadErr := fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Download failed with client error", downloadErr)
		} else {
			libOpentelemetry.HandleSpanError(span, "Download failed with server error", downloadErr)
		}

		return nil, downloadErr
	}

	reader := io.Reader(resp.Body)
	if c.maxDownloadSize > 0 {
		reader = io.LimitReader(resp.Body, c.maxDownloadSize)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to read response body", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	span.SetAttributes(attribute.Int("seaweedfs.response_size", len(data)))
	logger.Log(ctx, libLog.LevelDebug, fmt.Sprintf("SeaweedFS download completed: %s (%d bytes)", path, len(data)))

	return data, nil
}

// DownloadFileWithStream downloads a file returning a streaming reader instead of eagerly reading the body.
func (c *SeaweedFSClient) DownloadFileWithStream(ctx context.Context, path string) (io.ReadCloser, error) {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "seaweedfs.file.download_stream")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("seaweedfs.path", path),
	)

	path, err := pathIsValid(path)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Invalid path", err)
		return nil, err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to create download request", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to download file", err)
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if closeErr := resp.Body.Close(); closeErr != nil {
			closeErrWrapped := fmt.Errorf("download failed with status %d and error closing response body: %w", resp.StatusCode, closeErr)
			libOpentelemetry.HandleSpanError(span, "Download failed and error closing body", closeErrWrapped)

			return nil, closeErrWrapped
		}

		downloadErr := fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Download stream failed with client error", downloadErr)
		} else {
			libOpentelemetry.HandleSpanError(span, "Download stream failed with server error", downloadErr)
		}

		return nil, downloadErr
	}

	return resp.Body, nil
}

// DeleteFile deletes a file from SeaweedFS
func (c *SeaweedFSClient) DeleteFile(ctx context.Context, path string) error {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "seaweedfs.file.delete")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("seaweedfs.path", path),
	)

	path, err := pathIsValid(path)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Invalid path", err)
		return err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to create delete request", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to delete file", err)
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		deleteErr := fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Delete failed with client error", deleteErr)
		} else {
			libOpentelemetry.HandleSpanError(span, "Delete failed with server error", deleteErr)
		}

		return deleteErr
	}

	logger.Log(ctx, libLog.LevelDebug, fmt.Sprintf("SeaweedFS delete completed: %s", path))

	return nil
}

// HealthCheck checks if SeaweedFS is accessible
func (c *SeaweedFSClient) HealthCheck(ctx context.Context) error {
	_, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "seaweedfs.service.health_check")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	url := fmt.Sprintf("%s/status", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to create health check request", err)
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Health check failed", err)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		healthErr := fmt.Errorf("health check failed with status %d", resp.StatusCode)
		libOpentelemetry.HandleSpanError(span, "Health check returned non-OK status", healthErr)

		return healthErr
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
