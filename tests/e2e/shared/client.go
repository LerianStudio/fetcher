package shared

import (
	"context"
	"fmt"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/go-resty/resty/v2"
)

const (
	httpClientTimeout = 30 * time.Second

	// Queue names used in E2E tests (must match definitions.json)
	NotificationsQueue = "reporter.fetcher-notifications.queue"
)

// JobNotification represents the event published when a job status changes.
// This is the message format consumed from the notifications queue.
type JobNotification struct {
	JobID     string `json:"jobId"`
	Status    string `json:"status"`
	EventType string `json:"eventType"`
}

// ManagerClient provides HTTP access to the Manager API for E2E tests.
type ManagerClient struct {
	client *resty.Client
}

// NewManagerClient creates a new Manager API client.
func NewManagerClient(baseURL, organizationID string) *ManagerClient {
	client := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(httpClientTimeout).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-Organization-Id", organizationID)

	return &ManagerClient{client: client}
}

// ConnectionInput represents the request body for creating a connection.
type ConnectionInput struct {
	ConfigName   string         `json:"configName"`
	Type         string         `json:"type"`
	Host         string         `json:"host"`
	Port         int            `json:"port"`
	DatabaseName string         `json:"databaseName"`
	Username     string         `json:"userName"`
	Password     string         `json:"password"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// ConnectionResponse represents the response from creating/getting a connection.
type ConnectionResponse struct {
	ID           string         `json:"id"`
	ConfigName   string         `json:"configName"`
	Type         string         `json:"type"`
	Host         string         `json:"host"`
	Port         int            `json:"port"`
	DatabaseName string         `json:"databaseName"`
	Username     string         `json:"userName"`
	Status       string         `json:"status,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
}

// CreateConnection creates a new database connection via the Manager API.
func (c *ManagerClient) CreateConnection(ctx context.Context, input ConnectionInput) (*ConnectionResponse, error) {
	var result ConnectionResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(input).
		SetResult(&result).
		Post("/v1/management/connections")
	if err != nil {
		return nil, fmt.Errorf("CreateConnection: %w", err)
	}

	if resp.StatusCode() != 201 && resp.StatusCode() != 200 {
		return nil, fmt.Errorf("CreateConnection: unexpected status %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// GetConnection retrieves a connection by ID via the Manager API.
func (c *ManagerClient) GetConnection(ctx context.Context, connectionID string) (*ConnectionResponse, error) {
	var result ConnectionResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get("/v1/management/connections/" + connectionID)
	if err != nil {
		return nil, fmt.Errorf("GetConnection: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("GetConnection: unexpected status %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// DeleteConnection deletes a connection by ID via the Manager API.
func (c *ManagerClient) DeleteConnection(ctx context.Context, connectionID string) error {
	resp, err := c.client.R().
		SetContext(ctx).
		Delete("/v1/management/connections/" + connectionID)
	if err != nil {
		return fmt.Errorf("DeleteConnection: %w", err)
	}

	if resp.StatusCode() != 204 && resp.StatusCode() != 200 && resp.StatusCode() != 202 {
		return fmt.Errorf("DeleteConnection: unexpected status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

// CreateFetcherJob creates a new data extraction job via the Manager API.
func (c *ManagerClient) CreateFetcherJob(ctx context.Context, request model.FetcherRequest) (*model.FetcherResponse, error) {
	var result model.FetcherResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(request).
		SetResult(&result).
		Post("/v1/fetcher")
	if err != nil {
		return nil, fmt.Errorf("CreateFetcherJob: %w", err)
	}

	if resp.StatusCode() != 202 && resp.StatusCode() != 200 {
		return nil, fmt.Errorf("CreateFetcherJob: unexpected status %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// GetJob retrieves job details by ID via the Manager API.
func (c *ManagerClient) GetJob(ctx context.Context, jobID string) (*model.JobResponse, error) {
	var result model.JobResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get("/v1/fetcher/" + jobID)
	if err != nil {
		return nil, fmt.Errorf("GetJob: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("GetJob: unexpected status %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}
