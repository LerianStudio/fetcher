package crypto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHMACSigner_ValidKey(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	signer, err := NewHMACSigner(key, "")
	require.NoError(t, err)
	require.NotNil(t, signer)
	assert.Equal(t, SignatureVersion, signer.SignatureVersion())
}

func TestNewHMACSigner_CustomVersion(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	signer, err := NewHMACSigner(key, "v2")
	require.NoError(t, err)
	assert.Equal(t, "v2", signer.SignatureVersion())
}

func TestNewHMACSigner_KeyTooShort(t *testing.T) {
	t.Parallel()

	key := make([]byte, 16)
	signer, err := NewHMACSigner(key, "")
	require.Error(t, err)
	require.Nil(t, signer)
	assert.Contains(t, err.Error(), "signing key too short")
}

func TestHMACSigner_SignAndVerify(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	signer, err := NewHMACSigner(key, "v1")
	require.NoError(t, err)

	payload := []byte("1704067200.{\"jobId\":\"123\"}")

	// Sign
	signature := signer.Sign(payload)
	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 64) // SHA256 hex = 64 chars

	// Verify valid signature
	err = signer.Verify(payload, signature)
	assert.NoError(t, err)

	// Verify invalid signature
	err = signer.Verify(payload, "invalid-signature")
	assert.ErrorIs(t, err, ErrInvalidSignature)

	// Verify tampered payload
	tamperedPayload := []byte("1704067200.{\"jobId\":\"456\"}")
	err = signer.Verify(tamperedPayload, signature)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestHMACSigner_DifferentKeysDifferentSignatures(t *testing.T) {
	t.Parallel()

	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(255 - i)
	}

	signer1, _ := NewHMACSigner(key1, "v1")
	signer2, _ := NewHMACSigner(key2, "v1")

	payload := []byte("test payload")

	sig1 := signer1.Sign(payload)
	sig2 := signer2.Sign(payload)

	assert.NotEqual(t, sig1, sig2)

	// Cross-verification should fail
	err := signer1.Verify(payload, sig2)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestBuildSignaturePayload(t *testing.T) {
	t.Parallel()

	timestamp := int64(1704067200)
	body := []byte(`{"jobId":"123"}`)

	payload := BuildSignaturePayload(timestamp, body)

	expected := []byte(`1704067200.{"jobId":"123"}`)
	assert.Equal(t, expected, payload)
}

func TestHMACSigner_ConsistentSignatures(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	signer, _ := NewHMACSigner(key, "v1")
	payload := []byte("consistent test")

	// Sign multiple times
	sig1 := signer.Sign(payload)
	sig2 := signer.Sign(payload)
	sig3 := signer.Sign(payload)

	// Should be identical (deterministic)
	assert.Equal(t, sig1, sig2)
	assert.Equal(t, sig2, sig3)
}

func TestNewHMACSignerFromCryptor(t *testing.T) {
	t.Parallel()

	// Create a valid AESGCMService
	keyBase64 := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="

	cryptor, err := NewAESGCMServiceFromEnv(keyBase64, "v1")
	require.NoError(t, err)

	signer, err := NewHMACSignerFromCryptor(cryptor)
	require.NoError(t, err)
	require.NotNil(t, signer)

	// Verify it works
	payload := []byte("test")
	signature := signer.Sign(payload)
	err = signer.Verify(payload, signature)
	assert.NoError(t, err)
}

func TestHMACSigner_ConcurrentSigning(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	signer, _ := NewHMACSigner(key, "v1")
	payload := []byte("concurrent test")

	const goroutines = 100
	results := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			sig := signer.Sign(payload)
			results <- sig
		}()
	}

	// Collect results
	firstSig := <-results
	for i := 1; i < goroutines; i++ {
		sig := <-results
		assert.Equal(t, firstSig, sig, "all signatures should be identical")
	}
}

func BenchmarkHMACSigner_Sign(b *testing.B) {
	key := make([]byte, 32)
	signer, _ := NewHMACSigner(key, "v1")
	payload := BuildSignaturePayload(time.Now().Unix(), []byte(`{"jobId":"123","data":"test"}`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		signer.Sign(payload)
	}
}

func BenchmarkHMACSigner_Verify(b *testing.B) {
	key := make([]byte, 32)
	signer, _ := NewHMACSigner(key, "v1")
	payload := BuildSignaturePayload(time.Now().Unix(), []byte(`{"jobId":"123","data":"test"}`))
	signature := signer.Sign(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = signer.Verify(payload, signature)
	}
}
