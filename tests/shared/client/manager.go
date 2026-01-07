package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LerianStudio/fetcher/tests/shared/config"
)

// ManagerClient provides HTTP access to the Manager API.
// This is the ONLY interface tests use to interact with the system.
type ManagerClient struct {
	baseURL        string
	httpClient     *http.Client
	organizationID string
}

// NewManagerClient creates a new Manager API client.
func NewManagerClient(baseURL, organizationID string) *ManagerClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &ManagerClient{
		baseURL:        baseURL,
		organizationID: organizationID,
		httpClient: &http.Client{
			Timeout:   config.HTTPClientTimeout,
			Transport: transport,
		},
	}
}

// SSLInput contains SSL configuration for connection creation.
type SSLInput struct {
	Mode string `json:"mode"`           // SSL mode (e.g., "require", "verify-full")
	CA   string `json:"ca,omitempty"`   // CA certificate (PEM format)
	Cert string `json:"cert,omitempty"` // Client certificate (PEM format)
	Key  string `json:"key,omitempty"`  // Client private key (PEM format)
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
	SSL          *SSLInput      `json:"ssl,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// ConnectionPartialUpdateInput represents the request body for partial updates (PATCH).
// All fields are pointers to distinguish "not provided" from "provided with value".
type ConnectionPartialUpdateInput struct {
	ConfigName   *string                `json:"configName,omitempty"`
	Type         *string                `json:"type,omitempty"`
	Host         *string                `json:"host,omitempty"`
	Port         *int                   `json:"port,omitempty"`
	DatabaseName *string                `json:"databaseName,omitempty"`
	Username     *string                `json:"username,omitempty"`
	Password     *string                `json:"password,omitempty"`
	SSL          *SSLPartialUpdateInput `json:"ssl,omitempty"`
	Metadata     map[string]any         `json:"metadata,omitempty"`
}

// SSLPartialUpdateInput represents partial SSL configuration for updates.
type SSLPartialUpdateInput struct {
	Mode *string `json:"mode,omitempty"`
	CA   *string `json:"ca,omitempty"`
	Cert *string `json:"cert,omitempty"`
	Key  *string `json:"key,omitempty"`
}

// NewSSLInput creates an SSLInput from SSL connection info.
func NewSSLInput(mode, caCert, clientCert, clientKey string) *SSLInput {
	return &SSLInput{
		Mode: mode,
		CA:   caCert,
		Cert: clientCert,
		Key:  clientKey,
	}
}

// WithSSL returns a copy of ConnectionInput with SSL configuration.
func (c ConnectionInput) WithSSL(ssl *SSLInput) ConnectionInput {
	c.SSL = ssl
	return c
}

// ConnectionInputFromInternalDB creates a ConnectionInput from InternalDBConnection.
// This is useful for creating connections from container info.
func ConnectionInputFromInternalDB(configName, dbType string, conn config.InternalDBConnection) ConnectionInput {
	input := ConnectionInput{
		ConfigName:   configName,
		Type:         dbType,
		Host:         conn.Host,
		Port:         conn.Port,
		DatabaseName: conn.Database,
		Username:     conn.Username,
		Password:     conn.Password,
	}

	// Add SSL if enabled
	if conn.SSLEnabled {
		input.SSL = &SSLInput{
			Mode: conn.SSLMode,
			CA:   conn.SSLCACert,
			Cert: conn.SSLClientCert,
			Key:  conn.SSLClientKey,
		}
	}

	return input
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
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
}

// FetcherRequest represents the request body for creating a fetcher job.
type FetcherRequest struct {
	DataRequest DataRequest    `json:"dataRequest"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// DataRequest encapsulates field mappings and optional filters.
type DataRequest struct {
	MappedFields map[string]map[string][]string `json:"mappedFields"`
	Filters      []FilterRequest                `json:"filters,omitempty"`
}

// FilterRequest defines a filter condition.
type FilterRequest struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    []any  `json:"value"`
}

// FetcherResponse represents the response from creating a fetcher job.
type FetcherResponse struct {
	JobID     string `json:"jobId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

// JobResponse represents the response from getting a job.
type JobResponse struct {
	ID             string                         `json:"id"`
	OrganizationID string                         `json:"organizationId"`
	Status         string                         `json:"status"`
	MappedFields   map[string]map[string][]string `json:"mappedFields"`
	Filters        []FilterRequest                `json:"filters,omitempty"`
	ResultPath     string                         `json:"resultPath,omitempty"`
	Metadata       map[string]any                 `json:"metadata,omitempty"`
	RequestHash    string                         `json:"requestHash"`
	CreatedAt      string                         `json:"createdAt"`
	CompletedAt    *string                        `json:"completedAt,omitempty"`
}

// CreateConnection creates a new database connection via the Manager API.
func (c *ManagerClient) CreateConnection(ctx context.Context, input ConnectionInput) (*ConnectionResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connection input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/management/connections", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create connection failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// The API returns {"id": "uuid"} on creation
	var connectionResponse ConnectionResponse
	if err := json.Unmarshal(respBody, &connectionResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &connectionResponse, nil
}

// CreateFetcherJob creates a new data extraction job via the Manager API.
func (c *ManagerClient) CreateFetcherJob(ctx context.Context, request FetcherRequest) (*FetcherResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fetcher request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/fetcher", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create fetcher job failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result FetcherResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetJob retrieves job details by ID via the Manager API.
func (c *ManagerClient) GetJob(ctx context.Context, jobID string) (*JobResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/fetcher/"+jobID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get job failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result JobResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// WaitForJobCompletion polls the job status until it's completed or failed.
func (c *ManagerClient) WaitForJobCompletion(ctx context.Context, jobID string, timeout time.Duration) (*JobResponse, error) {
	deadline := time.Now().Add(timeout)

	var lastErr error

	consecutiveErrors := 0

	const maxConsecutiveErrors = 5

	ticker := time.NewTicker(config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				if lastErr != nil {
					return nil, fmt.Errorf("timeout waiting for job %s to complete (last error: %w)", jobID, lastErr)
				}

				return nil, fmt.Errorf("timeout waiting for job %s to complete", jobID)
			}

			job, err := c.GetJob(ctx, jobID)
			if err != nil {
				lastErr = err

				consecutiveErrors++
				if consecutiveErrors >= maxConsecutiveErrors {
					return nil, fmt.Errorf("job %s: too many consecutive errors: %w", jobID, err)
				}

				continue
			}

			consecutiveErrors = 0
			lastErr = nil

			switch job.Status {
			case "completed":
				return job, nil
			case "failed":
				return job, fmt.Errorf("job %s failed", jobID)
			}
			// Status is "pending" or "processing" - continue polling
		}
	}
}

// HealthCheck checks if the Manager API is accessible.
func (c *ManagerClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
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

// ListConnectionsResponse represents the response from listing connections.
type ListConnectionsResponse struct {
	Items []ConnectionResponse `json:"items"`
}

// ListConnections retrieves all connections via the Manager API.
func (c *ManagerClient) ListConnections(ctx context.Context) ([]ConnectionResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/management/connections", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list connections failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ListConnectionsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Items, nil
}

// DeleteConnection deletes a connection by ID via the Manager API.
func (c *ManagerClient) DeleteConnection(ctx context.Context, connectionID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/management/connections/"+connectionID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete connection failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteConnectionByConfigName finds and deletes a connection by its config name.
func (c *ManagerClient) DeleteConnectionByConfigName(ctx context.Context, configName string) error {
	connections, err := c.ListConnections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	for _, conn := range connections {
		if conn.ConfigName == configName {
			return c.DeleteConnection(ctx, conn.ID)
		}
	}

	// Connection not found - nothing to delete
	return nil
}

// GetConnection retrieves a connection by ID via the Manager API.
func (c *ManagerClient) GetConnection(ctx context.Context, connectionID string) (*ConnectionResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/management/connections/"+connectionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get connection failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ConnectionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// UpdateConnection updates a connection by ID via the Manager API.
func (c *ManagerClient) UpdateConnection(ctx context.Context, connectionID string, input ConnectionInput) (*ConnectionResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connection input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/v1/management/connections/"+connectionID, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update connection failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ConnectionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// PartialUpdateConnection updates specific fields of a connection via PATCH.
// Only the provided fields (non-nil) will be updated.
func (c *ManagerClient) PartialUpdateConnection(ctx context.Context, connectionID string, input ConnectionPartialUpdateInput) (*ConnectionResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal partial update input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/v1/management/connections/"+connectionID, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("partial update connection failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ConnectionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// StringPtr is a helper to create a string pointer (useful for partial updates).
func StringPtr(s string) *string {
	return &s
}

// IntPtr is a helper to create an int pointer (useful for partial updates).
func IntPtr(i int) *int {
	return &i
}

// ConnectionTestResponse represents the response from testing a connection.
type ConnectionTestResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latencyMs"`
}

// TestConnectionEndpoint tests a connection by ID via the Manager API.
func (c *ManagerClient) TestConnectionEndpoint(ctx context.Context, connectionID string) (*ConnectionTestResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/management/connections/"+connectionID+"/test", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited: %s", string(respBody))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("test connection failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ConnectionTestResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// SchemaValidationRequest represents the request for schema validation.
type SchemaValidationRequest struct {
	MappedFields map[string]map[string][]string `json:"mappedFields"`
}

// SchemaValidationError represents a single validation error.
type SchemaValidationError struct {
	Type         string `json:"type"`
	DataSourceID string `json:"dataSourceId"`
	Table        string `json:"table,omitempty"`
	Field        string `json:"field,omitempty"`
}

// SchemaValidationResponse represents the response from schema validation.
type SchemaValidationResponse struct {
	Status  string                  `json:"status"`
	Message string                  `json:"message"`
	Errors  []SchemaValidationError `json:"errors,omitempty"`
}

// ValidateSchema validates schema references via the Manager API.
func (c *ManagerClient) ValidateSchema(ctx context.Context, request SchemaValidationRequest) (*SchemaValidationResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/management/connections/validate-schema", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("validate schema failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result SchemaValidationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// PaginatedConnectionsResponse represents paginated connections response.
type PaginatedConnectionsResponse struct {
	Items []ConnectionResponse `json:"items"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
	Total int                  `json:"total"`
}

// ListConnectionsWithParams retrieves connections with query parameters.
func (c *ManagerClient) ListConnectionsWithParams(ctx context.Context, params map[string]string) (*PaginatedConnectionsResponse, error) {
	url := c.baseURL + "/v1/management/connections"

	if len(params) > 0 {
		query := "?"
		for k, v := range params {
			query += k + "=" + v + "&"
		}

		url += query[:len(query)-1]
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Organization-Id", c.organizationID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list connections failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result PaginatedConnectionsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}
