package shared

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// GenerateProductName generates a unique product name for test isolation.
// Product names are simple string labels (no API call needed).
func GenerateProductName() string {
	return fmt.Sprintf("e2e-%s", uuid.New().String()[:8])
}

// CreateTestConnection creates a connection with a given product name and registers cleanup.
// The connection is automatically deleted when the test completes.
func CreateTestConnection(t *testing.T, client *ManagerClient, ctx context.Context, productName string, connInput ConnectionInput) *ConnectionResponse {
	t.Helper()

	conn, err := client.CreateConnection(ctx, productName, connInput)
	require.NoError(t, err, "create test connection")
	require.NotNil(t, conn, "connection should not be nil")

	t.Cleanup(func() {
		_ = client.DeleteConnection(context.Background(), conn.ID)
	})

	return conn
}
