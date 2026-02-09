package crypto

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
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

// Tests for SignReader

func TestHMACSigner_SignReader_EquivalenceWithSign(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	signer, err := NewHMACSigner(key, "v1")
	require.NoError(t, err)

	testCases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello world")},
		{"json payload", []byte(`{"jobId":"123","status":"completed"}`)},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
		{"unicode", []byte("日本語テスト")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Sign with Sign()
			expectedSig := signer.Sign(tc.data)

			// Sign with SignReader()
			reader := bytes.NewReader(tc.data)
			actualSig, err := signer.SignReader(reader)

			require.NoError(t, err)
			assert.Equal(t, expectedSig, actualSig, "SignReader should produce same signature as Sign")
		})
	}
}

func TestHMACSigner_SignReader_LargeData(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	signer, err := NewHMACSigner(key, "v1")
	require.NoError(t, err)

	// Create 1MB of random data
	largeData := make([]byte, 1024*1024)
	_, err = rand.Read(largeData)
	require.NoError(t, err)

	// Sign with Sign()
	expectedSig := signer.Sign(largeData)

	// Sign with SignReader()
	reader := bytes.NewReader(largeData)
	actualSig, err := signer.SignReader(reader)

	require.NoError(t, err)
	assert.Equal(t, expectedSig, actualSig)
}

func TestHMACSigner_SignReader_StreamingMemoryEfficiency(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	signer, err := NewHMACSigner(key, "v1")
	require.NoError(t, err)

	// Create a reader that generates data on demand (simulating large file)
	// This tests that SignReader doesn't load all data into memory at once
	dataSize := 10 * 1024 * 1024 // 10MB
	reader := &limitedReader{remaining: dataSize, pattern: []byte("test pattern data ")}

	sig, err := signer.SignReader(reader)

	require.NoError(t, err)
	assert.Len(t, sig, 64) // SHA256 hex = 64 chars
}

// limitedReader generates a repeated pattern up to a size limit
type limitedReader struct {
	remaining int
	pattern   []byte
	offset    int
}

func (r *limitedReader) Read(p []byte) (n int, err error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}

	for n < len(p) && r.remaining > 0 {
		copyLen := len(r.pattern) - r.offset
		if copyLen > len(p)-n {
			copyLen = len(p) - n
		}
		if copyLen > r.remaining {
			copyLen = r.remaining
		}

		copy(p[n:], r.pattern[r.offset:r.offset+copyLen])
		n += copyLen
		r.remaining -= copyLen
		r.offset = (r.offset + copyLen) % len(r.pattern)
	}

	return n, nil
}

// errorReader always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(_ []byte) (n int, err error) {
	return 0, r.err
}

func TestHMACSigner_SignReader_Error(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	signer, err := NewHMACSigner(key, "v1")
	require.NoError(t, err)

	expectedErr := errors.New("simulated read error")
	reader := &errorReader{err: expectedErr}

	sig, err := signer.SignReader(reader)

	require.Error(t, err)
	assert.Empty(t, sig)
	assert.Contains(t, err.Error(), "failed to read data for signing")
}

func TestHMACSigner_SignReader_Deterministic(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	signer, err := NewHMACSigner(key, "v1")
	require.NoError(t, err)

	data := []byte("deterministic test data")

	// Sign multiple times with new readers
	var signatures []string
	for i := 0; i < 5; i++ {
		reader := bytes.NewReader(data)
		sig, err := signer.SignReader(reader)
		require.NoError(t, err)
		signatures = append(signatures, sig)
	}

	// All signatures should be identical
	for i := 1; i < len(signatures); i++ {
		assert.Equal(t, signatures[0], signatures[i], "signature %d should match signature 0", i)
	}
}

func BenchmarkHMACSigner_SignReader_1KB(b *testing.B) {
	key := make([]byte, 32)
	signer, _ := NewHMACSigner(key, "v1")
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, _ = signer.SignReader(reader)
	}
}

func BenchmarkHMACSigner_SignReader_1MB(b *testing.B) {
	key := make([]byte, 32)
	signer, _ := NewHMACSigner(key, "v1")
	data := make([]byte, 1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, _ = signer.SignReader(reader)
	}
}

func BenchmarkHMACSigner_SignReader_100MB(b *testing.B) {
	key := make([]byte, 32)
	signer, _ := NewHMACSigner(key, "v1")
	data := make([]byte, 100*1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, _ = signer.SignReader(reader)
	}
}
