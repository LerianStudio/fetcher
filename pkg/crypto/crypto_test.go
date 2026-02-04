package crypto

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

// generateValidKey creates a valid 32-byte key encoded in base64 for testing.
func generateValidKey() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return base64.StdEncoding.EncodeToString(key)
}

// generateInvalidLengthKey creates a key with invalid length (not 32 bytes).
func generateInvalidLengthKey(length int) string {
	key := make([]byte, length)
	for i := range key {
		key[i] = byte(i)
	}
	return base64.StdEncoding.EncodeToString(key)
}

// TestNewAESGCMServiceFromEnv tests the constructor with various inputs.
func TestNewAESGCMServiceFromEnv(t *testing.T) {
	tests := []struct {
		name       string
		keyBase64  string
		keyVersion string
		wantErr    bool
		errContain string
		wantVer    string
	}{
		{
			name:       "valid key with version",
			keyBase64:  generateValidKey(),
			keyVersion: "v1",
			wantErr:    false,
			wantVer:    "v1",
		},
		{
			name:       "valid key with empty version defaults to 1",
			keyBase64:  generateValidKey(),
			keyVersion: "",
			wantErr:    false,
			wantVer:    "1",
		},
		{
			name:       "valid key with custom version",
			keyBase64:  generateValidKey(),
			keyVersion: "2024-01-15",
			wantErr:    false,
			wantVer:    "2024-01-15",
		},
		{
			name:       "invalid base64 encoding",
			keyBase64:  "not-valid-base64!@#$",
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid encryption key (base64)",
		},
		{
			name:       "key too short (16 bytes)",
			keyBase64:  generateInvalidLengthKey(16),
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid encryption key length",
		},
		{
			name:       "key too long (64 bytes)",
			keyBase64:  generateInvalidLengthKey(64),
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid encryption key length",
		},
		{
			name:       "empty key",
			keyBase64:  "",
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid encryption key length",
		},
		{
			name:       "key exactly 31 bytes",
			keyBase64:  generateInvalidLengthKey(31),
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid encryption key length",
		},
		{
			name:       "key exactly 33 bytes",
			keyBase64:  generateInvalidLengthKey(33),
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid encryption key length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewAESGCMServiceFromEnv(tt.keyBase64, tt.keyVersion)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("expected error to contain %q, got %q", tt.errContain, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if svc == nil {
				t.Fatal("expected non-nil service")
			}

			if svc.KeyVersion() != tt.wantVer {
				t.Fatalf("expected key version %q, got %q", tt.wantVer, svc.KeyVersion())
			}
		})
	}
}

// TestNewAESGCMService tests the raw byte constructor.
func TestNewAESGCMService(t *testing.T) {
	tests := []struct {
		name       string
		key        []byte
		keyVersion string
		wantErr    bool
		errContain string
		wantVer    string
	}{
		{
			name:       "valid 32 byte key with version",
			key:        make([]byte, 32),
			keyVersion: "v1",
			wantErr:    false,
			wantVer:    "v1",
		},
		{
			name:       "valid key with empty version defaults to 1",
			key:        make([]byte, 32),
			keyVersion: "",
			wantErr:    false,
			wantVer:    "1",
		},
		{
			name:       "key too short (16 bytes)",
			key:        make([]byte, 16),
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid encryption key length",
		},
		{
			name:       "key too long (64 bytes)",
			key:        make([]byte, 64),
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid encryption key length",
		},
		{
			name:       "empty key",
			key:        []byte{},
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid encryption key length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewAESGCMService(tt.key, tt.keyVersion)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("expected error to contain %q, got %q", tt.errContain, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if svc == nil {
				t.Fatal("expected non-nil service")
			}

			if svc.KeyVersion() != tt.wantVer {
				t.Fatalf("expected key version %q, got %q", tt.wantVer, svc.KeyVersion())
			}
		})
	}
}

// TestNewAESGCMService_WithKeyDeriver tests integration with KeyDeriver.
func TestNewAESGCMService_WithKeyDeriver(t *testing.T) {
	// Create a key deriver with a test master key
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	keyDeriver, err := NewHKDFKeyDeriver(masterKey)
	if err != nil {
		t.Fatalf("failed to create key deriver: %v", err)
	}

	// Create AESGCMService with derived credential key
	svc, err := NewAESGCMService(keyDeriver.GetCredentialKey(), "v1")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Test encrypt/decrypt round trip
	ctx := context.Background()
	plaintext := "test secret data"

	ciphertext, version, err := svc.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	decrypted, err := svc.Decrypt(ctx, ciphertext, version)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

// TestEncrypt tests the Encrypt method with various inputs.
func TestEncrypt(t *testing.T) {
	validKey := generateValidKey()
	svc, err := NewAESGCMServiceFromEnv(validKey, "v1")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "encrypt simple string",
			plaintext: "hello world",
			wantErr:   false,
		},
		{
			name:      "encrypt empty string",
			plaintext: "",
			wantErr:   false,
		},
		{
			name:      "encrypt unicode characters",
			plaintext: "Hello, World!",
			wantErr:   false,
		},
		{
			name:      "encrypt special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr:   false,
		},
		{
			name:      "encrypt newlines and tabs",
			plaintext: "line1\nline2\ttabbed",
			wantErr:   false,
		},
		{
			name:      "encrypt long string",
			plaintext: strings.Repeat("a", 10000),
			wantErr:   false,
		},
		{
			name:      "encrypt JSON-like content",
			plaintext: `{"password":"secret123","api_key":"abc-xyz"}`,
			wantErr:   false,
		},
		{
			name:      "encrypt database connection string",
			plaintext: "postgresql://user:p@ssw0rd@localhost:5432/db?sslmode=require",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cipherText, keyVersion, err := svc.Encrypt(ctx, tt.plaintext)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cipherText == "" {
				t.Fatal("expected non-empty ciphertext")
			}

			if keyVersion != "v1" {
				t.Fatalf("expected key version v1, got %s", keyVersion)
			}

			// Verify ciphertext is valid base64
			_, err = base64.StdEncoding.DecodeString(cipherText)
			if err != nil {
				t.Fatalf("ciphertext is not valid base64: %v", err)
			}

			// Ciphertext should be different from plaintext
			if cipherText == tt.plaintext && tt.plaintext != "" {
				t.Fatal("ciphertext should not equal plaintext")
			}
		})
	}
}

// TestDecrypt tests the Decrypt method with various inputs.
func TestDecrypt(t *testing.T) {
	validKey := generateValidKey()
	svc, err := NewAESGCMServiceFromEnv(validKey, "v1")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()

	// First encrypt a known value
	originalPlaintext := "secret password 123"
	cipherText, keyVersion, err := svc.Encrypt(ctx, originalPlaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	tests := []struct {
		name       string
		cipherText string
		keyVersion string
		wantErr    bool
		errContain string
	}{
		{
			name:       "decrypt valid ciphertext",
			cipherText: cipherText,
			keyVersion: keyVersion,
			wantErr:    false,
		},
		{
			name:       "wrong key version",
			cipherText: cipherText,
			keyVersion: "v2",
			wantErr:    true,
			errContain: "unsupported key version",
		},
		{
			name:       "invalid base64 ciphertext",
			cipherText: "not-valid-base64!@#$",
			keyVersion: "v1",
			wantErr:    true,
			errContain: "invalid ciphertext (base64)",
		},
		{
			name:       "ciphertext too short",
			cipherText: base64.StdEncoding.EncodeToString([]byte("short")),
			keyVersion: "v1",
			wantErr:    true,
			errContain: "ciphertext too short",
		},
		{
			name:       "empty ciphertext",
			cipherText: "",
			keyVersion: "v1",
			wantErr:    true,
			errContain: "ciphertext too short",
		},
		{
			name:       "tampered ciphertext",
			cipherText: base64.StdEncoding.EncodeToString(make([]byte, 50)),
			keyVersion: "v1",
			wantErr:    true,
		},
		{
			name:       "empty key version",
			cipherText: cipherText,
			keyVersion: "",
			wantErr:    true,
			errContain: "unsupported key version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plaintext, err := svc.Decrypt(ctx, tt.cipherText, tt.keyVersion)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("expected error to contain %q, got %q", tt.errContain, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if plaintext != originalPlaintext {
				t.Fatalf("expected plaintext %q, got %q", originalPlaintext, plaintext)
			}
		})
	}
}

// TestEncryptDecryptRoundTrip tests that encrypt followed by decrypt returns the original value.
func TestEncryptDecryptRoundTrip(t *testing.T) {
	validKey := generateValidKey()
	svc, err := NewAESGCMServiceFromEnv(validKey, "v1")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "hello world",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "unicode characters",
			plaintext: "Hello, World!",
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:      "database password",
			plaintext: "Sup3r$ecret!P@ssw0rd",
		},
		{
			name:      "connection string",
			plaintext: "mongodb://admin:password123@cluster0.example.mongodb.net/mydb",
		},
		{
			name:      "multiline content",
			plaintext: "line1\nline2\nline3",
		},
		{
			name:      "json payload",
			plaintext: `{"username":"admin","password":"secret123","token":"eyJhbGciOiJIUzI1NiJ9"}`,
		},
		{
			name:      "very long string",
			plaintext: strings.Repeat("LongPassword123!", 1000),
		},
		{
			name:      "binary-like content",
			plaintext: string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Encrypt
			cipherText, keyVersion, err := svc.Encrypt(ctx, tt.plaintext)
			if err != nil {
				t.Fatalf("failed to encrypt: %v", err)
			}

			// Decrypt
			decrypted, err := svc.Decrypt(ctx, cipherText, keyVersion)
			if err != nil {
				t.Fatalf("failed to decrypt: %v", err)
			}

			// Verify round-trip
			if decrypted != tt.plaintext {
				t.Fatalf("round-trip failed: expected %q, got %q", tt.plaintext, decrypted)
			}
		})
	}
}

// TestKeyVersion tests the KeyVersion method.
func TestKeyVersion(t *testing.T) {
	tests := []struct {
		name       string
		keyVersion string
		wantVer    string
	}{
		{
			name:       "explicit version v1",
			keyVersion: "v1",
			wantVer:    "v1",
		},
		{
			name:       "explicit version v2",
			keyVersion: "v2",
			wantVer:    "v2",
		},
		{
			name:       "date-based version",
			keyVersion: "2024-01-15",
			wantVer:    "2024-01-15",
		},
		{
			name:       "empty version defaults to 1",
			keyVersion: "",
			wantVer:    "1",
		},
		{
			name:       "numeric version",
			keyVersion: "42",
			wantVer:    "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewAESGCMServiceFromEnv(generateValidKey(), tt.keyVersion)
			if err != nil {
				t.Fatalf("failed to create service: %v", err)
			}

			if svc.KeyVersion() != tt.wantVer {
				t.Fatalf("expected key version %q, got %q", tt.wantVer, svc.KeyVersion())
			}
		})
	}
}

// TestNonceUniqueness tests that each encryption produces a unique output.
func TestNonceUniqueness(t *testing.T) {
	validKey := generateValidKey()
	svc, err := NewAESGCMServiceFromEnv(validKey, "v1")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	plaintext := "same plaintext for all"

	// Encrypt the same plaintext multiple times
	const iterations = 100
	ciphertexts := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		cipherText, _, err := svc.Encrypt(ctx, plaintext)
		if err != nil {
			t.Fatalf("encryption failed at iteration %d: %v", i, err)
		}

		if ciphertexts[cipherText] {
			t.Fatalf("duplicate ciphertext found at iteration %d: nonce is not unique", i)
		}
		ciphertexts[cipherText] = true
	}

	// Verify we have unique ciphertexts
	if len(ciphertexts) != iterations {
		t.Fatalf("expected %d unique ciphertexts, got %d", iterations, len(ciphertexts))
	}
}

// TestDifferentKeysProduceDifferentCiphertexts tests that different keys produce different outputs.
func TestDifferentKeysProduceDifferentCiphertexts(t *testing.T) {
	// Create two different 32-byte keys
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(255 - i)
	}

	svc1, err := NewAESGCMServiceFromEnv(base64.StdEncoding.EncodeToString(key1), "v1")
	if err != nil {
		t.Fatalf("failed to create service 1: %v", err)
	}

	svc2, err := NewAESGCMServiceFromEnv(base64.StdEncoding.EncodeToString(key2), "v1")
	if err != nil {
		t.Fatalf("failed to create service 2: %v", err)
	}

	ctx := context.Background()
	plaintext := "same plaintext"

	cipher1, _, err := svc1.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt with key 1: %v", err)
	}

	cipher2, _, err := svc2.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt with key 2: %v", err)
	}

	// Decode both ciphertexts to compare the actual encrypted content (not just random nonces)
	// Since nonces are random, we cannot directly compare ciphertexts
	// But we can verify that one key cannot decrypt the other's ciphertext

	// Try to decrypt cipher1 with svc2 (should fail)
	_, err = svc2.Decrypt(ctx, cipher1, "v1")
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key")
	}

	// Try to decrypt cipher2 with svc1 (should fail)
	_, err = svc1.Decrypt(ctx, cipher2, "v1")
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key")
	}
}

// TestCryptorInterface verifies AESGCMService implements Cryptor interface.
func TestCryptorInterface(t *testing.T) {
	var _ Cryptor = (*AESGCMService)(nil)

	svc, err := NewAESGCMServiceFromEnv(generateValidKey(), "v1")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Verify the service can be used as Cryptor interface
	var cryptor Cryptor = svc

	ctx := context.Background()
	plaintext := "interface test"

	cipherText, keyVersion, err := cryptor.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("interface Encrypt failed: %v", err)
	}

	decrypted, err := cryptor.Decrypt(ctx, cipherText, keyVersion)
	if err != nil {
		t.Fatalf("interface Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("interface round-trip failed: expected %q, got %q", plaintext, decrypted)
	}

	if cryptor.KeyVersion() != "v1" {
		t.Fatalf("interface KeyVersion failed: expected v1, got %s", cryptor.KeyVersion())
	}
}

// TestDecryptWithDifferentKeyVersions tests handling of key version mismatches.
func TestDecryptWithDifferentKeyVersions(t *testing.T) {
	ctx := context.Background()

	// Service with version v1
	svcV1, err := NewAESGCMServiceFromEnv(generateValidKey(), "v1")
	if err != nil {
		t.Fatalf("failed to create v1 service: %v", err)
	}

	// Service with version v2 (same key, different version label)
	svcV2, err := NewAESGCMServiceFromEnv(generateValidKey(), "v2")
	if err != nil {
		t.Fatalf("failed to create v2 service: %v", err)
	}

	plaintext := "secret data"

	// Encrypt with v1
	cipherText, keyVersion, err := svcV1.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	if keyVersion != "v1" {
		t.Fatalf("expected key version v1, got %s", keyVersion)
	}

	// Decrypt with v1 should succeed
	decrypted, err := svcV1.Decrypt(ctx, cipherText, "v1")
	if err != nil {
		t.Fatalf("v1 decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}

	// Decrypt with v2 service using v1 version should fail
	_, err = svcV2.Decrypt(ctx, cipherText, "v1")
	if err == nil {
		t.Fatal("expected error when decrypting with v2 service using v1 version")
	}
	if !strings.Contains(err.Error(), "unsupported key version") {
		t.Fatalf("expected unsupported key version error, got: %v", err)
	}

	// Decrypt with v1 service using v2 version should also fail
	_, err = svcV1.Decrypt(ctx, cipherText, "v2")
	if err == nil {
		t.Fatal("expected error when decrypting with wrong version")
	}
	if !strings.Contains(err.Error(), "unsupported key version") {
		t.Fatalf("expected unsupported key version error, got: %v", err)
	}
}

// TestCiphertextStructure tests that the ciphertext has the expected structure.
func TestCiphertextStructure(t *testing.T) {
	validKey := generateValidKey()
	svc, err := NewAESGCMServiceFromEnv(validKey, "v1")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	plaintext := "test message"

	cipherText, _, err := svc.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Decode the ciphertext
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		t.Fatalf("failed to decode ciphertext: %v", err)
	}

	// AES-GCM nonce size is 12 bytes
	// The ciphertext should be at least nonce (12) + tag (16) = 28 bytes
	minSize := 12 + 16
	if len(raw) < minSize {
		t.Fatalf("ciphertext too short: expected at least %d bytes, got %d", minSize, len(raw))
	}

	// Ciphertext should include nonce (12) + plaintext length + tag (16)
	expectedMinSize := 12 + len(plaintext) + 16
	if len(raw) < expectedMinSize {
		t.Fatalf("ciphertext size incorrect: expected at least %d bytes, got %d", expectedMinSize, len(raw))
	}
}

// TestContextCancellation tests that context cancellation is properly handled.
// Note: Current implementation does not use context for cancellation,
// but this test documents the expected behavior.
func TestContextCancellation(t *testing.T) {
	validKey := generateValidKey()
	svc, err := NewAESGCMServiceFromEnv(validKey, "v1")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Current implementation does not check context cancellation
	// These operations should still succeed
	plaintext := "test"
	cipherText, keyVersion, err := svc.Encrypt(ctx, plaintext)
	if err != nil {
		// If implementation changes to check context, this would be expected
		t.Logf("Encrypt respects context cancellation: %v", err)
		return
	}

	decrypted, err := svc.Decrypt(ctx, cipherText, keyVersion)
	if err != nil {
		t.Logf("Decrypt respects context cancellation: %v", err)
		return
	}

	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

// TestConcurrentEncryption tests that the service is safe for concurrent use.
func TestConcurrentEncryption(t *testing.T) {
	validKey := generateValidKey()
	svc, err := NewAESGCMServiceFromEnv(validKey, "v1")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	plaintext := "concurrent test"
	const goroutines = 100

	results := make(chan string, goroutines)
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			cipherText, keyVersion, err := svc.Encrypt(ctx, plaintext)
			if err != nil {
				errors <- err
				return
			}

			decrypted, err := svc.Decrypt(ctx, cipherText, keyVersion)
			if err != nil {
				errors <- err
				return
			}

			results <- decrypted
		}()
	}

	// Collect results
	for i := 0; i < goroutines; i++ {
		select {
		case err := <-errors:
			t.Fatalf("concurrent operation failed: %v", err)
		case result := <-results:
			if result != plaintext {
				t.Fatalf("concurrent round-trip failed: expected %q, got %q", plaintext, result)
			}
		}
	}
}
