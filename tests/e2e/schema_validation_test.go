//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"
	"time"

	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateSchema_ValidTables_Success verifies that schema validation
// passes when all referenced tables and fields exist.
func TestValidateSchema_ValidTables_Success(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create product and connection
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-schema-valid-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Validate schema with valid tables and fields
	request := e2eshared.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			uniqueName: {
				"transactions": {"id", "account_id", "amount", "currency", "type", "created_at"},
			},
		},
	}

	result, err := apiClient.ValidateSchema(ctx, request)
	require.NoError(t, err, "validate schema")

	assert.Equal(t, "success", result.Status, "validation should succeed")
	assert.Empty(t, result.Errors, "should have no errors")

	t.Logf("Schema validation successful: %s", result.Message)
}

// TestValidateSchema_InvalidTable_Failure verifies that schema validation
// fails when a referenced table does not exist.
func TestValidateSchema_InvalidTable_Failure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create product and connection
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-schema-invalid-table-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Validate schema with non-existent table
	request := e2eshared.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			uniqueName: {
				"nonexistent_table": {"id", "name"},
			},
		},
	}

	resp, err := apiClient.ValidateSchemaRaw(ctx, request)
	require.NoError(t, err, "validate schema request should succeed")

	// API returns 422 Unprocessable Entity for schema validation failures
	assert.Equal(t, 422, resp.StatusCode(), "should return 422 for invalid table")

	// Parse the error response body
	body := string(resp.Body())
	assert.Contains(t, body, "nonexistent_table", "error should reference the invalid table")

	t.Logf("Schema validation correctly returned 422: %s", body)
}

// TestValidateSchema_InvalidField_Failure verifies that schema validation
// fails when a referenced field does not exist in the table.
func TestValidateSchema_InvalidField_Failure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Create product and connection
	product := e2eshared.CreateTestProduct(t, apiClient, ctx)

	uniqueName := fmt.Sprintf("e2e-schema-invalid-field-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   uniqueName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")

	t.Cleanup(func() {
		_ = apiClient.DeleteConnection(context.Background(), conn.ID)
	})

	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Validate schema with non-existent field
	request := e2eshared.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			uniqueName: {
				"transactions": {"id", "nonexistent_field", "amount"},
			},
		},
	}

	resp, err := apiClient.ValidateSchemaRaw(ctx, request)
	require.NoError(t, err, "validate schema request should succeed")

	// API returns 422 Unprocessable Entity for schema validation failures
	assert.Equal(t, 422, resp.StatusCode(), "should return 422 for invalid field")

	// Parse the error response body
	body := string(resp.Body())
	assert.Contains(t, body, "nonexistent_field", "error should reference the invalid field")

	t.Logf("Schema validation correctly returned 422: %s", body)
}

// TestValidateSchema_UnknownDatasource_Error verifies that schema validation
// fails when a referenced datasource does not exist.
func TestValidateSchema_UnknownDatasource_Error(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	// Validate schema with non-existent datasource
	request := e2eshared.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{
			"nonexistent-datasource": {
				"transactions": {"id", "amount"},
			},
		},
	}

	resp, err := apiClient.ValidateSchemaRaw(ctx, request)
	require.NoError(t, err, "request should succeed")

	// Unknown datasource should return an error (400 bad request, 404 not found, or 422 validation failure)
	assert.True(t, resp.StatusCode() >= 400,
		"unknown datasource should return error status, got %d", resp.StatusCode())

	t.Logf("Unknown datasource returned status %d: %s", resp.StatusCode(), string(resp.Body()))
}

// TestValidateSchema_EmptyRequest_BadRequest verifies that schema validation
// with an empty request returns a bad request error.
func TestValidateSchema_EmptyRequest_BadRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	request := e2eshared.SchemaValidationRequest{
		MappedFields: map[string]map[string][]string{},
	}

	resp, err := apiClient.ValidateSchemaRaw(ctx, request)
	require.NoError(t, err, "request should succeed")

	// Empty request should be rejected
	assert.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 422,
		"should return 400 or 422, got %d", resp.StatusCode())

	t.Logf("Empty schema validation request returned status %d", resp.StatusCode())
}
