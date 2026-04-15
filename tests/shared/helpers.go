package shared

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	fetcherCrypto "github.com/LerianStudio/fetcher/pkg/crypto"
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

// testAppEncKey is the APP_ENC_KEY used in the test Worker environment.
// It matches the value in WorkerEnv() (apps.go).
const testAppEncKey = "kV2RgskAt2gr+rtJmldM0gVEQNXduXXp3Le8VFCQKj8="

// ExtractionResult is the parsed structure of a job result file.
// The keys are: datasource name -> table name -> rows (each row is a map of field -> value).
type ExtractionResult = map[string]map[string][]map[string]any

// DownloadAndDecryptResult downloads a job result file from SeaweedFS, decrypts it,
// and returns the parsed extraction result. It fails the test if any step fails.
//
// Parameters:
//   - seaweedFSURL: the SeaweedFS filer HTTP URL (e.g., "http://localhost:8888")
//   - bucket: the storage bucket name (maps to a filer directory, e.g., "fetcher-storage")
//   - resultPath: the S3 key from JobResponse (e.g., "external-data/<jobID>.json")
func DownloadAndDecryptResult(t *testing.T, ctx context.Context, seaweedFSURL, bucket, resultPath string) ExtractionResult {
	t.Helper()

	// Download encrypted file from SeaweedFS filer — bucket maps to a filer directory
	reqURL := fmt.Sprintf("%s/%s/%s", seaweedFSURL, bucket, resultPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	require.NoError(t, err, "create download request")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "download result from SeaweedFS")

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "SeaweedFS download should return 200")

	rawBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read SeaweedFS response body")
	require.NotEmpty(t, rawBytes, "result file should not be empty")

	// Derive the same storage encryption key the Worker uses via HKDF
	masterKey, err := base64.StdEncoding.DecodeString(testAppEncKey)
	require.NoError(t, err, "decode APP_ENC_KEY")

	keyDeriver, err := fetcherCrypto.NewHKDFKeyDeriver(masterKey)
	require.NoError(t, err, "create HKDF key deriver")

	storageKey := keyDeriver.GetStorageEncryptKey()

	// Decrypt AES-GCM: the encrypted file is Base64(nonce[12] || ciphertext + auth_tag)
	ciphertextWithNonce, err := base64.StdEncoding.DecodeString(string(rawBytes))
	require.NoError(t, err, "base64-decode encrypted data")

	block, err := aes.NewCipher(storageKey)
	require.NoError(t, err, "create AES cipher")

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err, "create GCM")

	nonceSize := gcm.NonceSize()
	require.True(t, len(ciphertextWithNonce) > nonceSize, "ciphertext too short")

	nonce, ciphertext := ciphertextWithNonce[:nonceSize], ciphertextWithNonce[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	require.NoError(t, err, "decrypt result file")

	// Parse JSON
	var result ExtractionResult

	err = json.Unmarshal(plaintext, &result)
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
