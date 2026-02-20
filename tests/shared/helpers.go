package shared

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// CreateTestProduct creates a product with a unique code and registers cleanup.
// The product is automatically deleted when the test completes.
func CreateTestProduct(t *testing.T, client *ManagerClient, ctx context.Context) *ProductResponse {
	t.Helper()

	uniqueCode := fmt.Sprintf("e2e-%s", uuid.New().String()[:8])
	product, err := client.CreateProduct(ctx, ProductInput{
		Code:        uniqueCode,
		Name:        fmt.Sprintf("E2E Test Product %s", uniqueCode),
		Description: "Auto-created for E2E test",
	})
	require.NoError(t, err, "create test product")
	require.NotNil(t, product, "product should not be nil")

	t.Cleanup(func() {
		_ = client.DeleteProduct(context.Background(), product.ID)
	})

	return product
}

// CreateTestProductAndConnection creates a product and a connection assigned to it.
// Both are automatically cleaned up when the test completes.
func CreateTestProductAndConnection(t *testing.T, client *ManagerClient, ctx context.Context, connInput ConnectionInput) (*ProductResponse, *ConnectionResponse) {
	t.Helper()

	product := CreateTestProduct(t, client, ctx)
	connInput.ProductID = product.ID

	conn, err := client.CreateConnection(ctx, connInput)
	require.NoError(t, err, "create test connection")
	require.NotNil(t, conn, "connection should not be nil")

	t.Cleanup(func() {
		_ = client.DeleteConnection(context.Background(), conn.ID)
	})

	return product, conn
}
