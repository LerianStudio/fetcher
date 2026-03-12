package shared

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/go-resty/resty/v2"
)

// HTTP client configuration constants.
const (
	// httpClientTimeout is the default timeout for all HTTP operations.
	httpClientTimeout = 30 * time.Second
)

// RabbitMQ queue names used in E2E tests. These must match the topology defined in
// testdata/definitions.json.
const (
	// NotificationsQueue is the queue where job completion notifications are published.
	NotificationsQueue = "reporter.fetcher-notifications.queue"
)

// JobNotification represents the event payload published to RabbitMQ when a job status changes.
// Tests can consume this from NotificationsQueue to verify job completion asynchronously.
type JobNotification struct {
	// JobID is the unique identifier of the job.
	JobID string `json:"jobId"`
	// Status is the new status of the job (e.g., "completed", "failed").
	Status string `json:"status"`
	// EventType describes the type of event (e.g., "job.completed").
	EventType string `json:"eventType"`
}

// ManagerClient provides a typed HTTP client for the Manager API.
// It wraps resty.Client with Fetcher-specific operations and automatic JSON serialization.
// All methods include both a typed version (returns parsed response) and a Raw version
// (returns *resty.Response for testing error scenarios).
type ManagerClient struct {
	client *resty.Client
}

// NewManagerClient creates a new Manager API client configured with the given base URL
// and organization ID. The organization ID is sent in the X-Organization-Id header
// with every request, as required by the Manager API for multi-tenancy.
//
// Debug logging can be enabled by setting E2E_DEBUG_LOG=true in the environment.
// When enabled, all HTTP requests and responses (including headers and bodies)
// are printed to stderr, which is useful for diagnosing test failures.
func NewManagerClient(baseURL, organizationID string) *ManagerClient {
	client := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(httpClientTimeout).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-Organization-Id", organizationID)

	if os.Getenv("E2E_DEBUG_LOG") == "true" {
		client.SetDebug(true)
	}

	return &ManagerClient{client: client}
}

// ConnectionInput represents the request body for creating a new database connection.
// All fields except Metadata are required. The product name is sent via the X-Product-Name
// header, not in the request body.
type ConnectionInput struct {
	// ConfigName is a unique identifier for this connection within the organization.
	ConfigName string `json:"configName"`
	// Type is the database type (use DBType* constants from constants.go).
	Type string `json:"type"`
	// Host is the database server hostname or IP address.
	Host string `json:"host"`
	// Port is the database server port number.
	Port int `json:"port"`
	// DatabaseName is the name of the database/schema to connect to.
	DatabaseName string `json:"databaseName"`
	// Username is the database authentication username.
	Username string `json:"userName"`
	// Password is the database authentication password (encrypted at rest).
	Password string `json:"password"`
	// Metadata contains optional key-value pairs for custom attributes.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ConnectionResponse represents the API response for connection operations.
// It includes all input fields plus server-generated fields like ID and timestamps.
type ConnectionResponse struct {
	// ID is the server-generated unique identifier (UUID).
	ID string `json:"id"`
	// ProductName is the product name this connection belongs to.
	ProductName string `json:"productName,omitempty"`
	// ConfigName is the unique connection name within the organization.
	ConfigName string `json:"configName"`
	// Type is the database type.
	Type string `json:"type"`
	// Host is the database server hostname.
	Host string `json:"host"`
	// Port is the database server port.
	Port int `json:"port"`
	// DatabaseName is the database/schema name.
	DatabaseName string `json:"databaseName"`
	// Username is the database username.
	Username string `json:"userName"`
	// Status indicates the connection validation status.
	Status string `json:"status,omitempty"`
	// Metadata contains custom key-value pairs.
	Metadata map[string]any `json:"metadata,omitempty"`
	// CreatedAt is the ISO 8601 timestamp when the connection was created.
	CreatedAt string `json:"createdAt"`
	// UpdatedAt is the ISO 8601 timestamp when the connection was last modified.
	UpdatedAt string `json:"updatedAt"`
}

// checkStatus validates the HTTP response status code against a list of expected values.
// Returns nil if the response status matches any expected code, otherwise returns an error
// with the method name, actual status code, and response body for debugging.
func checkStatus(resp *resty.Response, method string, expected ...int) error {
	for _, code := range expected {
		if resp.StatusCode() == code {
			return nil
		}
	}

	return fmt.Errorf("%s: unexpected status %d: %s", method, resp.StatusCode(), resp.String())
}

// CreateConnection creates a new database connection via POST /v1/management/connections.
// The productName is sent via the X-Product-Name header.
// Returns the created connection with its server-generated ID on success (201 or 200).
func (c *ManagerClient) CreateConnection(ctx context.Context, productName string, input ConnectionInput) (*ConnectionResponse, error) {
	var result ConnectionResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-Product-Name", productName).
		SetBody(input).
		SetResult(&result).
		Post("/v1/management/connections")
	if err != nil {
		return nil, fmt.Errorf("CreateConnection: %w", err)
	}

	if err := checkStatus(resp, "CreateConnection", 200, 201); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetConnection retrieves a connection by ID via GET /v1/management/connections/{id}.
// Returns the connection details if found (200), or an error if not found (404) or other failure.
func (c *ManagerClient) GetConnection(ctx context.Context, connectionID string) (*ConnectionResponse, error) {
	var result ConnectionResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get("/v1/management/connections/" + connectionID)
	if err != nil {
		return nil, fmt.Errorf("GetConnection: %w", err)
	}

	if err := checkStatus(resp, "GetConnection", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteConnection performs a soft delete on a connection via DELETE /v1/management/connections/{id}.
// Returns nil on success (200, 202, or 204). The connection cannot be deleted if it has active jobs.
func (c *ManagerClient) DeleteConnection(ctx context.Context, connectionID string) error {
	resp, err := c.client.R().
		SetContext(ctx).
		Delete("/v1/management/connections/" + connectionID)
	if err != nil {
		return fmt.Errorf("DeleteConnection: %w", err)
	}

	return checkStatus(resp, "DeleteConnection", 200, 202, 204)
}

// CreateFetcherJob creates a new data extraction job via POST /v1/fetcher.
// The job is queued for async processing by the Worker. Returns 202 (Accepted) for new jobs
// or 200 if a duplicate job already exists within the idempotency window.
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

	if err := checkStatus(resp, "CreateFetcherJob", 200, 202); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetJob retrieves job details and status via GET /v1/fetcher/{id}.
// Use this to poll for job completion or check job results.
func (c *ManagerClient) GetJob(ctx context.Context, jobID string) (*model.JobResponse, error) {
	var result model.JobResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get("/v1/fetcher/" + jobID)
	if err != nil {
		return nil, fmt.Errorf("GetJob: %w", err)
	}

	if err := checkStatus(resp, "GetJob", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetJobRaw retrieves job details and returns the raw *resty.Response without parsing.
// Use this for testing error scenarios where you need access to the status code and raw body.
func (c *ManagerClient) GetJobRaw(ctx context.Context, jobID string) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		Get("/v1/fetcher/" + jobID)
}

// ListConnectionsParams holds query parameters for the list connections endpoint.
// All fields are optional; zero values are omitted from the query string.
//
// Note: The API does not support filtering by host or type via query parameters.
// Use ListConnectionsWithProductName to filter by product (via X-Product-Name header).
type ListConnectionsParams struct {
	// Page is the page number for pagination (1-based).
	Page int
	// Limit is the maximum number of items per page.
	Limit int
	// SortOrder specifies the sort direction: "asc" or "desc".
	SortOrder string
}

// toQueryString builds the URL query string from the non-zero parameter values.
// Returns an empty string if all parameters are zero/empty.
func (p ListConnectionsParams) toQueryString() string {
	query := url.Values{}
	if p.Page > 0 {
		query.Set("page", strconv.Itoa(p.Page))
	}

	if p.Limit > 0 {
		query.Set("limit", strconv.Itoa(p.Limit))
	}

	if p.SortOrder != "" {
		query.Set("sortOrder", p.SortOrder)
	}

	if len(query) == 0 {
		return ""
	}

	return "?" + query.Encode()
}

// ListConnectionsResponse represents the paginated response from GET /v1/management/connections.
type ListConnectionsResponse struct {
	// Items contains the connections for the current page.
	Items []ConnectionResponse `json:"items"`
	// Page is the current page number.
	Page int `json:"page"`
	// Limit is the maximum items per page.
	Limit int `json:"limit"`
	// Total is the total count of connections matching the filters.
	Total int `json:"total"`
}

// ListConnections retrieves connections via GET /v1/management/connections with optional filters.
func (c *ManagerClient) ListConnections(ctx context.Context, params ListConnectionsParams) (*ListConnectionsResponse, error) {
	var result ListConnectionsResponse

	path := "/v1/management/connections" + params.toQueryString()

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get(path)
	if err != nil {
		return nil, fmt.Errorf("ListConnections: %w", err)
	}

	if err := checkStatus(resp, "ListConnections", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListConnectionsRaw retrieves connections and returns the raw *resty.Response for testing error scenarios.
func (c *ManagerClient) ListConnectionsRaw(ctx context.Context, params ListConnectionsParams) (*resty.Response, error) {
	path := "/v1/management/connections" + params.toQueryString()

	return c.client.R().
		SetContext(ctx).
		Get(path)
}

// ConnectionUpdateInput represents the request body for PATCH /v1/management/connections/{id}.
// All fields are pointers to support partial updates - nil values are omitted from the request,
// allowing you to update only specific fields without affecting others.
type ConnectionUpdateInput struct {
	// ConfigName updates the connection's unique identifier.
	ConfigName *string `json:"configName,omitempty"`
	// Type updates the database type.
	Type *string `json:"type,omitempty"`
	// Host updates the database server hostname.
	Host *string `json:"host,omitempty"`
	// Port updates the database server port.
	Port *int `json:"port,omitempty"`
	// DatabaseName updates the database/schema name.
	DatabaseName *string `json:"databaseName,omitempty"`
	// Username updates the database username.
	Username *string `json:"userName,omitempty"`
	// Password updates the database password (will be re-encrypted).
	Password *string `json:"password,omitempty"`
	// Metadata updates the custom key-value pairs.
	Metadata *map[string]any `json:"metadata,omitempty"`
}

// UpdateConnection performs a partial update via PATCH /v1/management/connections/{id}.
// Only non-nil fields in the input are updated. Returns 409 Conflict if the connection has active jobs.
func (c *ManagerClient) UpdateConnection(ctx context.Context, connectionID string, input ConnectionUpdateInput) (*ConnectionResponse, error) {
	var result ConnectionResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(input).
		SetResult(&result).
		Patch("/v1/management/connections/" + connectionID)
	if err != nil {
		return nil, fmt.Errorf("UpdateConnection: %w", err)
	}

	if err := checkStatus(resp, "UpdateConnection", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateConnectionRaw performs a partial update and returns the raw *resty.Response for testing error scenarios.
func (c *ManagerClient) UpdateConnectionRaw(ctx context.Context, connectionID string, input ConnectionUpdateInput) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		SetBody(input).
		Patch("/v1/management/connections/" + connectionID)
}

// TestConnectionResult represents the response from POST /v1/management/connections/{id}/test.
type TestConnectionResult struct {
	// Status is "success" if the connection test passed, or "failure" otherwise.
	Status string `json:"status"`
	// Message provides details about the test result or error.
	Message string `json:"message"`
	// LatencyMs is the connection latency in milliseconds.
	LatencyMs int64 `json:"latencyMs"`
}

// TestConnection tests database connectivity via POST /v1/management/connections/{id}/test.
// This endpoint is rate-limited (10 requests/minute per connection).
func (c *ManagerClient) TestConnection(ctx context.Context, connectionID string) (*TestConnectionResult, error) {
	var result TestConnectionResult

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Post("/v1/management/connections/" + connectionID + "/test")
	if err != nil {
		return nil, fmt.Errorf("TestConnection: %w", err)
	}

	if err := checkStatus(resp, "TestConnection", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// TestConnectionRaw tests connectivity and returns the raw *resty.Response for testing error scenarios.
func (c *ManagerClient) TestConnectionRaw(ctx context.Context, connectionID string) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		Post("/v1/management/connections/" + connectionID + "/test")
}

// WaitForConnectionAvailable polls the connection test endpoint until the connection is available
// or the timeout is reached. This is useful when creating a new connection and needing to wait
// for the background validation to complete before using it for extraction jobs.
func (c *ManagerClient) WaitForConnectionAvailable(ctx context.Context, connectionID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 200 * time.Millisecond

	for time.Now().Before(deadline) {
		result, err := c.TestConnection(ctx, connectionID)
		if err == nil && result.Status == "success" {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			// Continue polling
		}
	}

	return fmt.Errorf("connection %s did not become available within %v", connectionID, timeout)
}

// SchemaValidationRequest represents the request body for POST /v1/management/connections/validate-schema.
// The MappedFields map is structured as: dataSourceID -> tableName -> []fieldNames
type SchemaValidationRequest struct {
	// MappedFields maps datasource IDs to tables and their fields to validate.
	MappedFields map[string]map[string][]string `json:"mappedFields"`
}

// SchemaValidationError represents a single validation error when a table or field is not found.
type SchemaValidationError struct {
	// Type describes the error type (e.g., "table_not_found", "field_not_found").
	Type string `json:"type"`
	// DataSourceID identifies which datasource had the validation error.
	DataSourceID string `json:"dataSourceId"`
	// Table is the table name that was not found (for table_not_found errors).
	Table string `json:"table,omitempty"`
	// Field is the field name that was not found (for field_not_found errors).
	Field string `json:"field,omitempty"`
}

// SchemaValidationResponse represents the response from schema validation.
type SchemaValidationResponse struct {
	// Status is "success" if all tables and fields exist, or "failure" otherwise.
	Status string `json:"status"`
	// Message provides a summary of the validation result.
	Message string `json:"message"`
	// Errors contains details about each validation failure.
	Errors []SchemaValidationError `json:"errors,omitempty"`
}

// ValidateSchema verifies that the specified tables and fields exist in the configured datasources
// via POST /v1/management/connections/validate-schema. Use this before creating extraction jobs
// to ensure the schema mapping is valid.
func (c *ManagerClient) ValidateSchema(ctx context.Context, request SchemaValidationRequest) (*SchemaValidationResponse, error) {
	var result SchemaValidationResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(request).
		SetResult(&result).
		Post("/v1/management/connections/validate-schema")
	if err != nil {
		return nil, fmt.Errorf("ValidateSchema: %w", err)
	}

	if err := checkStatus(resp, "ValidateSchema", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// ValidateSchemaRaw validates schema and returns the raw response for testing error scenarios.
func (c *ManagerClient) ValidateSchemaRaw(ctx context.Context, request SchemaValidationRequest) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		SetBody(request).
		Post("/v1/management/connections/validate-schema")
}

// CreateConnectionRaw creates a connection and returns the raw response for testing error scenarios.
// The productName is sent via the X-Product-Name header.
func (c *ManagerClient) CreateConnectionRaw(ctx context.Context, productName string, input ConnectionInput) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		SetHeader("X-Product-Name", productName).
		SetBody(input).
		Post("/v1/management/connections")
}

// GetConnectionRaw retrieves a connection and returns the raw response for testing error scenarios.
func (c *ManagerClient) GetConnectionRaw(ctx context.Context, connectionID string) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		Get("/v1/management/connections/" + connectionID)
}

// DeleteConnectionRaw deletes a connection and returns the raw response for testing error scenarios.
func (c *ManagerClient) DeleteConnectionRaw(ctx context.Context, connectionID string) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		Delete("/v1/management/connections/" + connectionID)
}

// CreateFetcherJobRaw creates a job and returns the raw response for testing error scenarios.
func (c *ManagerClient) CreateFetcherJobRaw(ctx context.Context, request model.FetcherRequest) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		SetBody(request).
		Post("/v1/fetcher")
}

// ############################################################################
// Connection Assignment and Schema Types and Methods
// ############################################################################

// AssignConnection assigns a connection to a product via POST /v1/management/connections/{id}/assign.
// The productName is sent via the X-Product-Name header.
func (c *ManagerClient) AssignConnection(ctx context.Context, connectionID string, productName string) (*ConnectionResponse, error) {
	var result ConnectionResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-Product-Name", productName).
		SetResult(&result).
		Post("/v1/management/connections/" + connectionID + "/assign")
	if err != nil {
		return nil, fmt.Errorf("AssignConnection: %w", err)
	}

	if err := checkStatus(resp, "AssignConnection", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// AssignConnectionRaw assigns a connection and returns the raw response for testing error scenarios.
// The productName is sent via the X-Product-Name header.
func (c *ManagerClient) AssignConnectionRaw(ctx context.Context, connectionID string, productName string) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		SetHeader("X-Product-Name", productName).
		Post("/v1/management/connections/" + connectionID + "/assign")
}

// ListUnassignedConnections retrieves connections without a product assignment
// via GET /v1/management/connections/unassigned.
func (c *ManagerClient) ListUnassignedConnections(ctx context.Context, params ListConnectionsParams) (*ListConnectionsResponse, error) {
	var result ListConnectionsResponse

	path := "/v1/management/connections/unassigned" + params.toQueryString()

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get(path)
	if err != nil {
		return nil, fmt.Errorf("ListUnassignedConnections: %w", err)
	}

	if err := checkStatus(resp, "ListUnassignedConnections", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListUnassignedConnectionsRaw retrieves unassigned connections and returns the raw response.
func (c *ManagerClient) ListUnassignedConnectionsRaw(ctx context.Context, params ListConnectionsParams) (*resty.Response, error) {
	path := "/v1/management/connections/unassigned" + params.toQueryString()

	return c.client.R().
		SetContext(ctx).
		Get(path)
}

// ConnectionSchemaTable represents a table/collection in the schema response.
type ConnectionSchemaTable struct {
	// Name is the table or collection name.
	Name string `json:"name"`
	// Fields contains the column/field names.
	Fields []string `json:"fields"`
}

// ConnectionSchemaResponse represents the response from GET /v1/management/connections/{id}/schema.
type ConnectionSchemaResponse struct {
	// Tables contains the database tables/collections and their fields.
	Tables []ConnectionSchemaTable `json:"tables"`
}

// GetConnectionSchema retrieves the database schema for a connection
// via GET /v1/management/connections/{id}/schema.
func (c *ManagerClient) GetConnectionSchema(ctx context.Context, connectionID string) (*ConnectionSchemaResponse, error) {
	var result ConnectionSchemaResponse

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get("/v1/management/connections/" + connectionID + "/schema")
	if err != nil {
		return nil, fmt.Errorf("GetConnectionSchema: %w", err)
	}

	if err := checkStatus(resp, "GetConnectionSchema", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetConnectionSchemaRaw retrieves schema and returns the raw response for testing error scenarios.
func (c *ManagerClient) GetConnectionSchemaRaw(ctx context.Context, connectionID string) (*resty.Response, error) {
	return c.client.R().
		SetContext(ctx).
		Get("/v1/management/connections/" + connectionID + "/schema")
}

// ListConnectionsWithProductName retrieves connections filtered by product name using the X-Product-Name header.
func (c *ManagerClient) ListConnectionsWithProductName(ctx context.Context, productName string, params ListConnectionsParams) (*ListConnectionsResponse, error) {
	var result ListConnectionsResponse

	path := "/v1/management/connections" + params.toQueryString()

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-Product-Name", productName).
		SetResult(&result).
		Get(path)
	if err != nil {
		return nil, fmt.Errorf("ListConnectionsWithProductName: %w", err)
	}

	if err := checkStatus(resp, "ListConnectionsWithProductName", 200); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListConnectionsWithProductNameRaw retrieves connections by product name and returns the raw response.
func (c *ManagerClient) ListConnectionsWithProductNameRaw(ctx context.Context, productName string, params ListConnectionsParams) (*resty.Response, error) {
	path := "/v1/management/connections" + params.toQueryString()

	return c.client.R().
		SetContext(ctx).
		SetHeader("X-Product-Name", productName).
		Get(path)
}
