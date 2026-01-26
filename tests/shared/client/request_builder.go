package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RequestBuilder provides a fluent interface for building HTTP requests.
// It replaces the duplicated request construction logic across all client methods.
type RequestBuilder struct {
	client         *http.Client
	method         string
	url            string
	body           []byte
	headers        map[string]string
	expectedStatus []int
	err            error // Stores errors from builder methods to return during Execute
}

// NewRequestBuilder creates a new request builder with the given HTTP client.
func NewRequestBuilder(client *http.Client) *RequestBuilder {
	return &RequestBuilder{
		client:  client,
		headers: make(map[string]string),
	}
}

// Get sets up a GET request to the given URL.
func (b *RequestBuilder) Get(url string) *RequestBuilder {
	b.method = http.MethodGet
	b.url = url

	return b
}

// Post sets up a POST request to the given URL.
func (b *RequestBuilder) Post(url string) *RequestBuilder {
	b.method = http.MethodPost
	b.url = url

	return b
}

// Put sets up a PUT request to the given URL.
func (b *RequestBuilder) Put(url string) *RequestBuilder {
	b.method = http.MethodPut
	b.url = url

	return b
}

// Patch sets up a PATCH request to the given URL.
func (b *RequestBuilder) Patch(url string) *RequestBuilder {
	b.method = http.MethodPatch
	b.url = url

	return b
}

// Delete sets up a DELETE request to the given URL.
func (b *RequestBuilder) Delete(url string) *RequestBuilder {
	b.method = http.MethodDelete
	b.url = url

	return b
}

// WithJSONBody sets the request body as JSON.
// Marshal errors are stored and returned during Execute.
func (b *RequestBuilder) WithJSONBody(body any) *RequestBuilder {
	data, err := json.Marshal(body)
	if err != nil {
		b.err = fmt.Errorf("failed to marshal request body: %w", err)
		return b
	}

	b.body = data
	b.headers["Content-Type"] = "application/json"

	return b
}

// WithHeader sets a custom header.
func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	b.headers[key] = value
	return b
}

// WithOrganizationID sets the X-Organization-Id header.
func (b *RequestBuilder) WithOrganizationID(orgID string) *RequestBuilder {
	b.headers["X-Organization-Id"] = orgID
	return b
}

// ExpectStatus sets the expected HTTP status code(s).
func (b *RequestBuilder) ExpectStatus(codes ...int) *RequestBuilder {
	b.expectedStatus = codes
	return b
}

// Execute sends the request and decodes the response into result.
// If result is nil, the response body is not decoded.
// Returns any errors from builder methods (e.g., marshal failures).
func (b *RequestBuilder) Execute(ctx context.Context, result any) error {
	// Check for errors from builder methods
	if b.err != nil {
		return b.err
	}

	var bodyReader io.Reader
	if b.body != nil {
		bodyReader = bytes.NewReader(b.body)
	}

	req, err := http.NewRequestWithContext(ctx, b.method, b.url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range b.headers {
		req.Header.Set(key, value)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if !b.isExpectedStatus(resp.StatusCode) {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	// Decode response if result is provided
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// isExpectedStatus checks if the status code matches any expected code.
func (b *RequestBuilder) isExpectedStatus(code int) bool {
	if len(b.expectedStatus) == 0 {
		// Default: accept 2xx
		return code >= 200 && code < 300
	}

	for _, expected := range b.expectedStatus {
		if code == expected {
			return true
		}
	}

	return false
}

// Reset clears the builder for reuse.
func (b *RequestBuilder) Reset() *RequestBuilder {
	b.method = ""
	b.url = ""
	b.body = nil
	b.headers = make(map[string]string)
	b.expectedStatus = nil
	b.err = nil

	return b
}
