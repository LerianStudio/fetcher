package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeriveKeyIntegration tests the CLI tool end-to-end by running it as a subprocess.
func TestDeriveKeyIntegration(t *testing.T) {
	// Skip if not in integration test mode (optional, remove if you want it to always run)
	if os.Getenv("RUN_CLI_INTEGRATION") == "" {
		t.Skip("Skipping CLI integration test. Set RUN_CLI_INTEGRATION=1 to run.")
	}

	t.Run("valid key produces correct output", func(t *testing.T) {
		// Known test key
		rawKey := []byte("this-is-a-32-byte-master-key1234")
		encodedKey := base64.StdEncoding.EncodeToString(rawKey)

		// Calculate expected output using the crypto package directly
		deriver, err := crypto.NewHKDFKeyDeriver(rawKey)
		require.NoError(t, err)
		expectedHex := hex.EncodeToString(deriver.GetExternalHMACKey())

		// Run the CLI tool
		cmd := exec.Command("go", "run", ".", "-key", encodedKey)
		cmd.Dir = "."
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()
		require.NoError(t, err, "stderr: %s", stderr.String())

		output := stdout.String()
		assert.Contains(t, output, expectedHex, "output should contain the expected hex key")
		assert.Contains(t, output, "External HMAC Key (hex):", "output should have the correct prefix")
	})

	t.Run("missing key flag returns error", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".")
		cmd.Dir = "."
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		assert.Error(t, err, "should fail without key flag")
		assert.Contains(t, stderr.String(), "Error: -key flag is required")
	})

	t.Run("invalid base64 returns error", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".", "-key", "not-valid-base64!!!")
		cmd.Dir = "."
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		assert.Error(t, err, "should fail with invalid base64")
		assert.Contains(t, stderr.String(), "invalid master key (base64)")
	})

	t.Run("short key returns error", func(t *testing.T) {
		shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
		cmd := exec.Command("go", "run", ".", "-key", shortKey)
		cmd.Dir = "."
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		assert.Error(t, err, "should fail with short key")
		assert.Contains(t, stderr.String(), "master key too short")
	})

	t.Run("help flag shows usage", func(t *testing.T) {
		cmd := exec.Command("go", "run", ".", "-help")
		cmd.Dir = "."
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		assert.NoError(t, err, "help should exit successfully")
		assert.Contains(t, stderr.String(), "Derive External HMAC Key from Master Key")
		assert.Contains(t, stderr.String(), "Usage:")
	})
}

// TestDeriveKeyDeterminism verifies that the same input always produces the same output.
func TestDeriveKeyDeterminism(t *testing.T) {
	rawKey := []byte("this-is-a-32-byte-master-key1234")

	// Derive the key multiple times
	var results []string
	for i := 0; i < 5; i++ {
		deriver, err := crypto.NewHKDFKeyDeriver(rawKey)
		require.NoError(t, err)
		results = append(results, hex.EncodeToString(deriver.GetExternalHMACKey()))
	}

	// All results should be identical
	for i := 1; i < len(results); i++ {
		assert.Equal(t, results[0], results[i], "derivation should be deterministic")
	}
}

// TestDeriveKeyOutputFormat verifies the output format is correct.
func TestDeriveKeyOutputFormat(t *testing.T) {
	rawKey := []byte("this-is-a-32-byte-master-key1234")
	deriver, err := crypto.NewHKDFKeyDeriver(rawKey)
	require.NoError(t, err)

	hexKey := hex.EncodeToString(deriver.GetExternalHMACKey())

	// Verify hex format
	assert.Len(t, hexKey, 64, "32 bytes should produce 64 hex characters")
	assert.True(t, isValidHex(hexKey), "output should be valid hexadecimal")
}

func isValidHex(s string) bool {
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
