package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// SignatureVersion represents the version of the signing algorithm.
const SignatureVersion = "v1"

// ErrInvalidSignature is returned when signature verification fails.
var ErrInvalidSignature = errors.New("invalid message signature")

// ErrMissingSignature is returned when a required signature is missing.
var ErrMissingSignature = errors.New("missing message signature")

// ErrUnsupportedSignatureVersion is returned when the signature version is not supported.
var ErrUnsupportedSignatureVersion = errors.New("unsupported signature version")

// Signer provides message signing and verification using HMAC-SHA256.
//
//go:generate mockgen --destination=signer.mock.go --package=crypto . Signer
type Signer interface {
	// Sign creates an HMAC-SHA256 signature for the given payload.
	// The payload should be: timestamp + "." + message_body
	Sign(payload []byte) string

	// SignReader creates an HMAC-SHA256 signature by streaming data from a reader.
	// This is useful for signing large files without loading them entirely into memory.
	// Memory usage is O(1) regardless of input size.
	SignReader(r io.Reader) (string, error)

	// Verify checks if the signature is valid for the given payload.
	// Returns nil if valid, ErrInvalidSignature if invalid.
	Verify(payload []byte, signature string) error

	// SignatureVersion returns the current signature version (e.g., "v1").
	SignatureVersion() string
}

// HMACSigner implements Signer using HMAC-SHA256.
type HMACSigner struct {
	key     []byte
	version string
}

// NewHMACSigner creates a new HMAC-SHA256 signer with the given key.
// The key should be at least 32 bytes for security.
// Version defaults to SignatureVersion ("v1") if empty.
func NewHMACSigner(key []byte, version string) (*HMACSigner, error) {
	if len(key) < 32 {
		return nil, fmt.Errorf("signing key too short: got %d bytes, minimum 32 required", len(key))
	}

	if version == "" {
		version = SignatureVersion
	}

	return &HMACSigner{
		key:     key,
		version: version,
	}, nil
}

// Sign creates an HMAC-SHA256 signature for the given payload.
func (s *HMACSigner) Sign(payload []byte) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write(payload)

	return hex.EncodeToString(mac.Sum(nil))
}

// SignReader creates an HMAC-SHA256 signature by streaming data from a reader.
// This is useful for signing large files without loading them entirely into memory.
// Memory usage is O(1) regardless of input size (uses 32KB buffer).
func (s *HMACSigner) SignReader(r io.Reader) (string, error) {
	mac := hmac.New(sha256.New, s.key)

	if _, err := io.Copy(mac, r); err != nil {
		return "", fmt.Errorf("failed to read data for signing: %w", err)
	}

	return hex.EncodeToString(mac.Sum(nil)), nil
}

// Verify checks if the signature is valid for the given payload.
func (s *HMACSigner) Verify(payload []byte, signature string) error {
	expected := s.Sign(payload)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}

	return nil
}

// SignatureVersion returns the current signature version.
func (s *HMACSigner) SignatureVersion() string {
	return s.version
}

// BuildSignaturePayload constructs the payload to sign: timestamp.body
func BuildSignaturePayload(timestamp int64, body []byte) []byte {
	payload := make([]byte, 0, 20+1+len(body))
	payload = fmt.Appendf(payload, "%d.", timestamp)
	payload = append(payload, body...)

	return payload
}
