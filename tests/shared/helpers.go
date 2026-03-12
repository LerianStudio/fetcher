package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	libCrypto "github.com/LerianStudio/lib-commons/v3/commons/crypto"
	libLog "github.com/LerianStudio/lib-commons/v3/commons/log"
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

// SeaweedFS encryption keys used in the test Worker environment.
// These match the values in WorkerEnv() (apps.go).
const (
	seaweedFSEncryptKey = "3132333435363738393031323334353637383930313233343536373839303132"
	seaweedFSHashKey    = "3132333435363738393031323334353637383930313233343536373839303132"
)

// ExtractionResult is the parsed structure of a job result file.
// The keys are: datasource name -> table name -> rows (each row is a map of field -> value).
type ExtractionResult = map[string]map[string][]map[string]any

// DownloadAndDecryptResult downloads a job result file from SeaweedFS, decrypts it,
// and returns the parsed extraction result. It fails the test if any step fails.
//
// Parameters:
//   - seaweedFSURL: the SeaweedFS filer HTTP URL (e.g., "http://localhost:8888")
//   - resultPath: the ResultPath from JobResponse (e.g., "/external-data/<jobID>.json")
func DownloadAndDecryptResult(t *testing.T, ctx context.Context, seaweedFSURL, resultPath string) ExtractionResult {
	t.Helper()

	// Download encrypted file from SeaweedFS
	reqURL := seaweedFSURL + resultPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	require.NoError(t, err, "create download request")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "download result from SeaweedFS")

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "SeaweedFS download should return 200")

	rawBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read SeaweedFS response body")
	require.NotEmpty(t, rawBytes, "result file should not be empty")

	// Decrypt using the same keys as the Worker
	crypto := &libCrypto.Crypto{
		EncryptSecretKey: seaweedFSEncryptKey,
		HashSecretKey:    seaweedFSHashKey,
		Logger:           &libLog.NoneLogger{},
	}

	err = crypto.InitializeCipher()
	require.NoError(t, err, "initialize crypto cipher")

	encryptedStr := string(rawBytes)

	plaintext, err := crypto.Decrypt(&encryptedStr)
	require.NoError(t, err, "decrypt result file")
	require.NotNil(t, plaintext, "decrypted result should not be nil")

	// Parse JSON
	var result ExtractionResult

	err = json.Unmarshal([]byte(*plaintext), &result)
	require.NoError(t, err, "unmarshal result JSON")

	return result
}

// CountResultRows returns the total number of rows across all datasources and tables
// in an extraction result.
func CountResultRows(result ExtractionResult) int {
	count := 0

	for _, tables := range result {
		for _, rows := range tables {
			count += len(rows)
		}
	}

	return count
}
