package crypto

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/hkdf"
)

func TestNewHKDFKeyDeriver(t *testing.T) {
	t.Run("success with 32 byte key", func(t *testing.T) {
		masterKey := make([]byte, 32)
		for i := range masterKey {
			masterKey[i] = byte(i)
		}

		deriver, err := NewHKDFKeyDeriver(masterKey)

		require.NoError(t, err)
		assert.NotNil(t, deriver)
		assert.Len(t, deriver.GetCredentialKey(), 32)
		assert.Len(t, deriver.GetInternalHMACKey(), 32)
		assert.Len(t, deriver.GetExternalHMACKey(), 32)
	})

	t.Run("success with longer key", func(t *testing.T) {
		masterKey := make([]byte, 64)

		deriver, err := NewHKDFKeyDeriver(masterKey)

		require.NoError(t, err)
		assert.NotNil(t, deriver)
	})

	t.Run("error with short key", func(t *testing.T) {
		masterKey := make([]byte, 16)

		deriver, err := NewHKDFKeyDeriver(masterKey)

		require.Error(t, err)
		assert.Nil(t, deriver)
		assert.Contains(t, err.Error(), "master key too short")
	})

	t.Run("error with empty key", func(t *testing.T) {
		deriver, err := NewHKDFKeyDeriver([]byte{})

		require.Error(t, err)
		assert.Nil(t, deriver)
	})
}

func TestHKDFKeyDeriver_DeriveKey(t *testing.T) {
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	deriver, err := NewHKDFKeyDeriver(masterKey)
	require.NoError(t, err)

	t.Run("deterministic derivation", func(t *testing.T) {
		key1, err := deriver.DeriveKey("test-context", 32)
		require.NoError(t, err)

		key2, err := deriver.DeriveKey("test-context", 32)
		require.NoError(t, err)

		assert.Equal(t, key1, key2, "same context should produce same key")
	})

	t.Run("different contexts produce different keys", func(t *testing.T) {
		key1, err := deriver.DeriveKey("context-a", 32)
		require.NoError(t, err)

		key2, err := deriver.DeriveKey("context-b", 32)
		require.NoError(t, err)

		assert.NotEqual(t, key1, key2, "different contexts should produce different keys")
	})

	t.Run("different lengths", func(t *testing.T) {
		key16, err := deriver.DeriveKey("test", 16)
		require.NoError(t, err)
		assert.Len(t, key16, 16)

		key64, err := deriver.DeriveKey("test", 64)
		require.NoError(t, err)
		assert.Len(t, key64, 64)

		// First 16 bytes should match (HKDF property)
		assert.Equal(t, key16, key64[:16])
	})

	t.Run("error with zero length", func(t *testing.T) {
		_, err := deriver.DeriveKey("test", 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key length must be positive")
	})

	t.Run("error with negative length", func(t *testing.T) {
		_, err := deriver.DeriveKey("test", -1)
		require.Error(t, err)
	})

	t.Run("error with empty context", func(t *testing.T) {
		_, err := deriver.DeriveKey("", 32)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context cannot be empty")
	})
}

func TestHKDFKeyDeriver_PreDerivedKeys(t *testing.T) {
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	deriver, err := NewHKDFKeyDeriver(masterKey)
	require.NoError(t, err)

	t.Run("credential key matches manual derivation", func(t *testing.T) {
		expected, err := deriver.DeriveKey(ContextCredentials, DefaultKeyLength)
		require.NoError(t, err)

		assert.Equal(t, expected, deriver.GetCredentialKey())
	})

	t.Run("internal HMAC key matches manual derivation", func(t *testing.T) {
		expected, err := deriver.DeriveKey(ContextInternalHMAC, DefaultKeyLength)
		require.NoError(t, err)

		assert.Equal(t, expected, deriver.GetInternalHMACKey())
	})

	t.Run("external HMAC key matches manual derivation", func(t *testing.T) {
		expected, err := deriver.DeriveKey(ContextExternalHMAC, DefaultKeyLength)
		require.NoError(t, err)

		assert.Equal(t, expected, deriver.GetExternalHMACKey())
	})

	t.Run("all pre-derived keys are different", func(t *testing.T) {
		credKey := deriver.GetCredentialKey()
		intKey := deriver.GetInternalHMACKey()
		extKey := deriver.GetExternalHMACKey()

		assert.NotEqual(t, credKey, intKey, "credential and internal HMAC keys should differ")
		assert.NotEqual(t, credKey, extKey, "credential and external HMAC keys should differ")
		assert.NotEqual(t, intKey, extKey, "internal and external HMAC keys should differ")
	})
}

func TestHKDFKeyDeriver_KeyIndependence(t *testing.T) {
	// Test that derived keys are cryptographically independent
	// (no correlation between keys derived from different contexts)
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	deriver, err := NewHKDFKeyDeriver(masterKey)
	require.NoError(t, err)

	t.Run("keys have no byte-level correlation", func(t *testing.T) {
		key1 := deriver.GetCredentialKey()
		key2 := deriver.GetInternalHMACKey()
		key3 := deriver.GetExternalHMACKey()

		// Count matching bytes (should be near zero for independent keys)
		matchCount12 := countMatchingBytes(key1, key2)
		matchCount13 := countMatchingBytes(key1, key3)
		matchCount23 := countMatchingBytes(key2, key3)

		// With 32 random bytes, expected matches ~32/256 ≈ 0.125 per position
		// So expected total ~4 matches. Allow up to 8 for statistical variance.
		assert.Less(t, matchCount12, 8, "too many matching bytes between credential and internal keys")
		assert.Less(t, matchCount13, 8, "too many matching bytes between credential and external keys")
		assert.Less(t, matchCount23, 8, "too many matching bytes between internal and external keys")
	})
}

func countMatchingBytes(a, b []byte) int {
	count := 0
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			count++
		}
	}

	return count
}

func TestHKDFKeyDeriver_RFC5869TestVectors(t *testing.T) {
	// Test Case 1 from RFC 5869 Appendix A
	// https://datatracker.ietf.org/doc/html/rfc5869#appendix-A
	t.Run("RFC 5869 Test Case 1", func(t *testing.T) {
		ikm, _ := hex.DecodeString("0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b")
		salt, _ := hex.DecodeString("000102030405060708090a0b0c")
		info, _ := hex.DecodeString("f0f1f2f3f4f5f6f7f8f9")
		expectedOKM, _ := hex.DecodeString("3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865")

		// Use raw HKDF to verify our understanding matches RFC
		reader := hkdf.New(sha256.New, ikm, salt, info)
		okm := make([]byte, 42)
		_, err := io.ReadFull(reader, okm)
		require.NoError(t, err)

		assert.Equal(t, expectedOKM, okm, "HKDF output should match RFC 5869 test vector")
	})

	// Test Case 2 from RFC 5869
	t.Run("RFC 5869 Test Case 2", func(t *testing.T) {
		ikm, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f")
		salt, _ := hex.DecodeString("606162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0a1a2a3a4a5a6a7a8a9aaabacadaeaf")
		info, _ := hex.DecodeString("b0b1b2b3b4b5b6b7b8b9babbbcbdbebfc0c1c2c3c4c5c6c7c8c9cacbcccdcecfd0d1d2d3d4d5d6d7d8d9dadbdcdddedfe0e1e2e3e4e5e6e7e8e9eaebecedeeeff0f1f2f3f4f5f6f7f8f9fafbfcfdfeff")
		expectedOKM, _ := hex.DecodeString("b11e398dc80327a1c8e7f78c596a49344f012eda2d4efad8a050cc4c19afa97c59045a99cac7827271cb41c65e590e09da3275600c2f09b8367793a9aca3db71cc30c58179ec3e87c14c01d5c1f3434f1d87")

		reader := hkdf.New(sha256.New, ikm, salt, info)
		okm := make([]byte, 82)
		_, err := io.ReadFull(reader, okm)
		require.NoError(t, err)

		assert.Equal(t, expectedOKM, okm, "HKDF output should match RFC 5869 test vector")
	})

	// Test Case 3 from RFC 5869 - No salt, no info
	t.Run("RFC 5869 Test Case 3 (no salt, no info)", func(t *testing.T) {
		ikm, _ := hex.DecodeString("0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b")
		expectedOKM, _ := hex.DecodeString("8da4e775a563c18f715f802a063c5a31b8a11f5c5ee1879ec3454e5f3c738d2d9d201395faa4b61a96c8")

		// No salt (nil), no info (empty)
		reader := hkdf.New(sha256.New, ikm, nil, nil)
		okm := make([]byte, 42)
		_, err := io.ReadFull(reader, okm)
		require.NoError(t, err)

		assert.Equal(t, expectedOKM, okm, "HKDF output should match RFC 5869 test vector (no salt, no info)")
	})
}

func TestHKDFKeyDeriver_ConsistencyAcrossInstances(t *testing.T) {
	// Two derivers with the same master key should produce identical keys
	masterKey := []byte("this-is-a-32-byte-master-key12")
	masterKey = append(masterKey, []byte("34")...) // 32 bytes total

	deriver1, err := NewHKDFKeyDeriver(masterKey)
	require.NoError(t, err)

	deriver2, err := NewHKDFKeyDeriver(masterKey)
	require.NoError(t, err)

	t.Run("credential keys match across instances", func(t *testing.T) {
		assert.Equal(t, deriver1.GetCredentialKey(), deriver2.GetCredentialKey())
	})

	t.Run("internal HMAC keys match across instances", func(t *testing.T) {
		assert.Equal(t, deriver1.GetInternalHMACKey(), deriver2.GetInternalHMACKey())
	})

	t.Run("external HMAC keys match across instances", func(t *testing.T) {
		assert.Equal(t, deriver1.GetExternalHMACKey(), deriver2.GetExternalHMACKey())
	})

	t.Run("custom derivations match across instances", func(t *testing.T) {
		key1, err := deriver1.DeriveKey("custom-context", 64)
		require.NoError(t, err)

		key2, err := deriver2.DeriveKey("custom-context", 64)
		require.NoError(t, err)

		assert.Equal(t, key1, key2)
	})
}

func TestHKDFKeyDeriver_DifferentMasterKeys(t *testing.T) {
	masterKey1 := []byte("this-is-a-32-byte-master-key1234") // 32 bytes
	masterKey2 := []byte("another-32-byte-master-key-12345") // 32 bytes

	deriver1, err := NewHKDFKeyDeriver(masterKey1)
	require.NoError(t, err)

	deriver2, err := NewHKDFKeyDeriver(masterKey2)
	require.NoError(t, err)

	t.Run("different master keys produce different credential keys", func(t *testing.T) {
		assert.NotEqual(t, deriver1.GetCredentialKey(), deriver2.GetCredentialKey())
	})

	t.Run("different master keys produce different internal HMAC keys", func(t *testing.T) {
		assert.NotEqual(t, deriver1.GetInternalHMACKey(), deriver2.GetInternalHMACKey())
	})

	t.Run("different master keys produce different external HMAC keys", func(t *testing.T) {
		assert.NotEqual(t, deriver1.GetExternalHMACKey(), deriver2.GetExternalHMACKey())
	})
}

func TestHKDFKeyDeriver_KeysAreNotMasterKey(t *testing.T) {
	masterKey := []byte("this-is-a-32-byte-master-key1234") // 32 bytes
	deriver, err := NewHKDFKeyDeriver(masterKey)
	require.NoError(t, err)

	t.Run("credential key differs from master key", func(t *testing.T) {
		assert.False(t, bytes.Equal(masterKey, deriver.GetCredentialKey()))
	})

	t.Run("internal HMAC key differs from master key", func(t *testing.T) {
		assert.False(t, bytes.Equal(masterKey, deriver.GetInternalHMACKey()))
	})

	t.Run("external HMAC key differs from master key", func(t *testing.T) {
		assert.False(t, bytes.Equal(masterKey, deriver.GetExternalHMACKey()))
	})
}

func TestHKDFKeyDeriver_GettersReturnCopies(t *testing.T) {
	masterKey := []byte("this-is-a-32-byte-master-key1234") // 32 bytes
	deriver, err := NewHKDFKeyDeriver(masterKey)
	require.NoError(t, err)

	t.Run("GetCredentialKey returns independent copy", func(t *testing.T) {
		key1 := deriver.GetCredentialKey()
		key2 := deriver.GetCredentialKey()

		// Both should be equal initially
		assert.Equal(t, key1, key2)

		// Modify key1
		originalFirstByte := key1[0]
		key1[0] = ^key1[0] // Flip all bits

		// key2 should be unchanged (proves they are independent copies)
		assert.Equal(t, originalFirstByte, key2[0], "modifying one copy should not affect another")

		// Get a fresh copy and verify it's unchanged
		key3 := deriver.GetCredentialKey()
		assert.Equal(t, originalFirstByte, key3[0], "internal state should be unchanged")
	})

	t.Run("GetInternalHMACKey returns independent copy", func(t *testing.T) {
		key1 := deriver.GetInternalHMACKey()
		key2 := deriver.GetInternalHMACKey()

		assert.Equal(t, key1, key2)

		originalFirstByte := key1[0]
		key1[0] = ^key1[0]

		assert.Equal(t, originalFirstByte, key2[0], "modifying one copy should not affect another")

		key3 := deriver.GetInternalHMACKey()
		assert.Equal(t, originalFirstByte, key3[0], "internal state should be unchanged")
	})

	t.Run("GetExternalHMACKey returns independent copy", func(t *testing.T) {
		key1 := deriver.GetExternalHMACKey()
		key2 := deriver.GetExternalHMACKey()

		assert.Equal(t, key1, key2)

		originalFirstByte := key1[0]
		key1[0] = ^key1[0]

		assert.Equal(t, originalFirstByte, key2[0], "modifying one copy should not affect another")

		key3 := deriver.GetExternalHMACKey()
		assert.Equal(t, originalFirstByte, key3[0], "internal state should be unchanged")
	})
}

func TestDecodeMasterKey(t *testing.T) {
	t.Run("success with valid 32-byte base64 key", func(t *testing.T) {
		// Create a 32-byte key and encode it
		rawKey := make([]byte, 32)
		for i := range rawKey {
			rawKey[i] = byte(i)
		}
		encodedKey := base64.StdEncoding.EncodeToString(rawKey)

		decoded, err := DecodeMasterKey(encodedKey)

		require.NoError(t, err)
		assert.Equal(t, rawKey, decoded)
		assert.Len(t, decoded, 32)
	})

	t.Run("success with valid 64-byte base64 key", func(t *testing.T) {
		rawKey := make([]byte, 64)
		for i := range rawKey {
			rawKey[i] = byte(i)
		}
		encodedKey := base64.StdEncoding.EncodeToString(rawKey)

		decoded, err := DecodeMasterKey(encodedKey)

		require.NoError(t, err)
		assert.Equal(t, rawKey, decoded)
		assert.Len(t, decoded, 64)
	})

	t.Run("error with key shorter than 32 bytes", func(t *testing.T) {
		rawKey := make([]byte, 16) // Too short
		encodedKey := base64.StdEncoding.EncodeToString(rawKey)

		decoded, err := DecodeMasterKey(encodedKey)

		require.Error(t, err)
		assert.Nil(t, decoded)
		assert.Contains(t, err.Error(), "master key too short")
		assert.Contains(t, err.Error(), "got 16 bytes")
		assert.Contains(t, err.Error(), "minimum 32 required")
	})

	t.Run("error with invalid base64", func(t *testing.T) {
		invalidBase64 := "not-valid-base64!!!"

		decoded, err := DecodeMasterKey(invalidBase64)

		require.Error(t, err)
		assert.Nil(t, decoded)
		assert.Contains(t, err.Error(), "invalid master key (base64)")
	})

	t.Run("error with empty string", func(t *testing.T) {
		decoded, err := DecodeMasterKey("")

		require.Error(t, err)
		assert.Nil(t, decoded)
		assert.Contains(t, err.Error(), "master key too short")
	})

	t.Run("decoded key can be used with NewHKDFKeyDeriver", func(t *testing.T) {
		// Integration test: decode and use
		rawKey := []byte("this-is-a-32-byte-master-key1234")
		encodedKey := base64.StdEncoding.EncodeToString(rawKey)

		decoded, err := DecodeMasterKey(encodedKey)
		require.NoError(t, err)

		deriver, err := NewHKDFKeyDeriver(decoded)
		require.NoError(t, err)
		assert.NotNil(t, deriver)
		assert.Len(t, deriver.GetCredentialKey(), 32)
	})
}
