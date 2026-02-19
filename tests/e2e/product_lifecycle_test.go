//go:build e2e

package extraction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	e2eshared "github.com/LerianStudio/fetcher/tests/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProductLifecycle_FullCRUD verifies the complete product lifecycle:
// create, read, update, connection creation, fetcher job execution,
// and cleanup with proper cascading verification.
func TestProductLifecycle_FullCRUD(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Step 1: Create product with unique code
	uniqueCode := fmt.Sprintf("e2e-lifecycle-%s", uuid.New().String()[:8])
	productInput := e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Lifecycle Product %s", uniqueCode),
		Description: "E2E lifecycle test product",
	}

	product, err := apiClient.CreateProduct(ctx, productInput)
	require.NoError(t, err, "create product")
	require.NotEmpty(t, product.ID, "product ID should be set")

	t.Logf("Step 1: Created product: id=%s, code=%s", product.ID, product.Code)

	// Step 2: Get product - verify all fields
	retrieved, err := apiClient.GetProduct(ctx, product.ID)
	require.NoError(t, err, "get product")

	assert.Equal(t, product.ID, retrieved.ID, "ID should match")
	assert.Equal(t, uniqueCode, retrieved.Code, "code should match")
	assert.Equal(t, productInput.Name, retrieved.Name, "name should match")
	assert.Equal(t, productInput.Description, retrieved.Description, "description should match")
	assert.NotEmpty(t, retrieved.CreatedAt, "created_at should be set")
	assert.NotEmpty(t, retrieved.UpdatedAt, "updated_at should be set")

	t.Logf("Step 2: Retrieved product: name=%s", retrieved.Name)

	// Step 3: Update product name
	updatedName := fmt.Sprintf("Updated Lifecycle Product %s", uniqueCode)
	updated, err := apiClient.UpdateProduct(ctx, product.ID, e2eshared.ProductUpdateInput{
		Name: &updatedName,
	})
	require.NoError(t, err, "update product")

	t.Logf("Step 3: Updated product name to: %s", updated.Name)

	// Step 4: Get product - verify name changed
	retrievedAfterUpdate, err := apiClient.GetProduct(ctx, product.ID)
	require.NoError(t, err, "get product after update")

	assert.Equal(t, updatedName, retrievedAfterUpdate.Name, "name should be updated")
	assert.Equal(t, uniqueCode, retrievedAfterUpdate.Code, "code should not change")

	t.Logf("Step 4: Verified updated name: %s", retrievedAfterUpdate.Name)

	// Step 5: Create PostgreSQL connection with product ID
	connName := fmt.Sprintf("e2e-lifecycle-conn-%s", uuid.New().String()[:8])
	connInput := e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   connName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	conn, err := apiClient.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create connection")
	require.NotEmpty(t, conn.ID, "connection ID should be set")

	t.Logf("Step 5: Created connection: id=%s, name=%s", conn.ID, conn.ConfigName)

	// Step 5b: Wait for connection to be available before using it
	err = apiClient.WaitForConnectionAvailable(ctx, conn.ID, 10*time.Second)
	require.NoError(t, err, "wait for connection to be available")

	// Step 6: Create fetcher job for that connection
	fetcherReq := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				connName: {
					"transactions": {"id", "account_id", "amount", "currency", "type", "created_at"},
				},
			},
		},
		Metadata: map[string]any{
			"source": product.Code,
			"test":   "product-lifecycle-e2e",
		},
	}

	fetcherResp, err := apiClient.CreateFetcherJob(ctx, fetcherReq)
	require.NoError(t, err, "create fetcher job")
	require.NotEmpty(t, fetcherResp.JobID, "job ID should be set")

	jobID := fetcherResp.JobID.String()
	t.Logf("Step 6: Created job: %s", jobID)

	// Step 7: Wait for job completion
	jobResult := e2eshared.AssertJobCompleted(t, apiClient, jobID, e2eshared.DefaultJobTimeout)

	assert.Equal(t, "completed", jobResult.Status, "job status should be completed")
	assert.NotEmpty(t, jobResult.ResultPath, "result path should be set")

	t.Logf("Step 7: Job completed: status=%s, resultPath=%s", jobResult.Status, jobResult.ResultPath)

	// Step 8: Delete connection
	err = apiClient.DeleteConnection(ctx, conn.ID)
	require.NoError(t, err, "delete connection")

	t.Logf("Step 8: Deleted connection: %s", conn.ID)

	// Step 9: Delete product
	err = apiClient.DeleteProduct(ctx, product.ID)
	require.NoError(t, err, "delete product")

	t.Logf("Step 9: Deleted product: %s", product.ID)

	// Step 10: Verify product returns 404
	e2eshared.AssertProductNotFound(t, apiClient, product.ID)

	t.Logf("Step 10: Verified product is gone (404)")

	// Step 11: Verify connection returns 404
	e2eshared.AssertConnectionNotFound(t, apiClient, conn.ID)

	t.Logf("Step 11: Verified connection is gone (404)")
}

// TestProductLifecycle_DeleteBlockedByConnections verifies that a product cannot be
// deleted while it has active connections. The product can only be deleted after
// all its connections are removed.
func TestProductLifecycle_DeleteBlockedByConnections(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), e2eshared.DefaultTestTimeout)
	defer cancel()

	pgHost, pgPort, err := postgresInfra.HostPort()
	require.NoError(t, err, "get postgres host/port")

	// Step 1: Create product
	uniqueCode := fmt.Sprintf("e2e-del-blocked-%s", uuid.New().String()[:8])
	product, err := apiClient.CreateProduct(ctx, e2eshared.ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("Delete Blocked Product %s", uniqueCode),
		Description: "E2E delete blocked test product",
	})
	require.NoError(t, err, "create product")

	t.Logf("Step 1: Created product: id=%s", product.ID)

	// Step 2: Create PostgreSQL connection with product ID
	connName := fmt.Sprintf("e2e-del-blocked-conn-%s", uuid.New().String()[:8])
	conn, err := apiClient.CreateConnection(ctx, e2eshared.ConnectionInput{
		ProductID:    product.ID,
		ConfigName:   connName,
		Type:         e2eshared.DBTypePostgreSQL,
		Host:         pgHost,
		Port:         pgPort,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	})
	require.NoError(t, err, "create connection")

	t.Logf("Step 2: Created connection: id=%s", conn.ID)

	// Step 3: Attempt to delete product - should be blocked (409 Conflict)
	resp, err := apiClient.DeleteProductRaw(ctx, product.ID)
	require.NoError(t, err, "request should succeed")

	assert.Equal(t, 409, resp.StatusCode(), "should return 409 Conflict when product has connections")

	t.Logf("Step 3: Delete product blocked with status %d (has connections)", resp.StatusCode())

	// Step 4: Delete connection
	err = apiClient.DeleteConnection(ctx, conn.ID)
	require.NoError(t, err, "delete connection")

	t.Logf("Step 4: Deleted connection: %s", conn.ID)

	// Step 5: Delete product - should now succeed
	err = apiClient.DeleteProduct(ctx, product.ID)
	require.NoError(t, err, "delete product should succeed after removing connections")

	t.Logf("Step 5: Deleted product: %s", product.ID)

	// Step 6: Verify product returns 404
	e2eshared.AssertProductNotFound(t, apiClient, product.ID)

	t.Logf("Step 6: Verified product is gone (404)")
}
