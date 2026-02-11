package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Key derivation contexts for different purposes.
// Each context produces cryptographically independent keys from the same master key.
const (
	// ContextCredentialsLabel is a non-secret context identifier used for key derivation domain separation.
	// It is NOT a password, token, or credential.
	// #nosec G101 -- not a hardcoded credential; this is a non-secret context string used for key derivation.
	ContextCredentialsLabel = "fetcher-credentials-v1"

	// ContextInternalHMAC is used to derive the key for signing internal messages (Manager↔Worker).
	ContextInternalHMAC = "fetcher-internal-hmac-v1"

	// ContextExternalHMAC is used to derive the key for signing external messages and document HMACs.
	ContextExternalHMAC = "fetcher-external-hmac-v1"
)

// DefaultKeyLength is the standard key length for AES-256 and HMAC-SHA256.
const DefaultKeyLength = 32

// KeyDeriver provides deterministic key derivation from a master key.
// Different contexts produce cryptographically independent keys.
//
//go:generate mockgen --destination=key_deriver.mock.go --package=crypto . KeyDeriver
type KeyDeriver interface {
	// DeriveKey derives a key of the specified length for the given context.
	// The same master key and context always produce the same derived key.
	DeriveKey(context string, length int) ([]byte, error)

	// GetCredentialKey returns the pre-derived key for credential encryption (AES-256).
	GetCredentialKey() []byte

	// GetInternalHMACKey returns the pre-derived key for internal message signing.
	GetInternalHMACKey() []byte

	// GetExternalHMACKey returns the pre-derived key for external message signing and document HMACs.
	GetExternalHMACKey() []byte
}

// HKDFKeyDeriver implements KeyDeriver using HKDF (RFC 5869) with SHA-256.
// Keys are derived once at construction time and cached for performance.
type HKDFKeyDeriver struct {
	masterKey       []byte
	credentialKey   []byte
	internalHMACKey []byte
	externalHMACKey []byte
}

// NewHKDFKeyDeriver creates a new key deriver from a master key.
// The master key should be at least 32 bytes for security.
// All standard keys (credentials, internal HMAC, external HMAC) are derived at construction time.
func NewHKDFKeyDeriver(masterKey []byte) (*HKDFKeyDeriver, error) {
	if len(masterKey) < 32 {
		return nil, fmt.Errorf("master key too short: got %d bytes, minimum 32 required", len(masterKey))
	}

	deriver := &HKDFKeyDeriver{
		masterKey: masterKey,
	}

	// Pre-derive all standard keys
	var err error

	deriver.credentialKey, err = deriver.DeriveKey(ContextCredentialsLabel, DefaultKeyLength)
	if err != nil {
		return nil, fmt.Errorf("failed to derive credential key: %w", err)
	}

	deriver.internalHMACKey, err = deriver.DeriveKey(ContextInternalHMAC, DefaultKeyLength)
	if err != nil {
		return nil, fmt.Errorf("failed to derive internal HMAC key: %w", err)
	}

	deriver.externalHMACKey, err = deriver.DeriveKey(ContextExternalHMAC, DefaultKeyLength)
	if err != nil {
		return nil, fmt.Errorf("failed to derive external HMAC key: %w", err)
	}

	return deriver, nil
}

// DeriveKey derives a key of the specified length for the given context using HKDF.
// HKDF uses SHA-256 as the hash function.
// The context string is used as the "info" parameter in HKDF.
// No salt is used (nil salt is acceptable per RFC 5869).
func (d *HKDFKeyDeriver) DeriveKey(context string, length int) ([]byte, error) {
	if length <= 0 {
		return nil, errors.New("key length must be positive")
	}

	if context == "" {
		return nil, errors.New("context cannot be empty")
	}

	// HKDF with SHA-256, no salt, context as info
	reader := hkdf.New(sha256.New, d.masterKey, nil, []byte(context))

	key := make([]byte, length)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return key, nil
}

// GetCredentialKey returns a copy of the pre-derived key for credential encryption.
// The returned slice is a copy; modifications do not affect the original key.
func (d *HKDFKeyDeriver) GetCredentialKey() []byte {
	keyCopy := make([]byte, len(d.credentialKey))
	copy(keyCopy, d.credentialKey)

	return keyCopy
}

// GetInternalHMACKey returns a copy of the pre-derived key for internal message signing.
// The returned slice is a copy; modifications do not affect the original key.
func (d *HKDFKeyDeriver) GetInternalHMACKey() []byte {
	keyCopy := make([]byte, len(d.internalHMACKey))
	copy(keyCopy, d.internalHMACKey)

	return keyCopy
}

// GetExternalHMACKey returns a copy of the pre-derived key for external message signing and document HMACs.
// The returned slice is a copy; modifications do not affect the original key.
func (d *HKDFKeyDeriver) GetExternalHMACKey() []byte {
	keyCopy := make([]byte, len(d.externalHMACKey))
	copy(keyCopy, d.externalHMACKey)

	return keyCopy
}

// DecodeMasterKey decodes a Base64-encoded master key.
// The key must be at least 32 bytes after decoding for use with NewHKDFKeyDeriver.
func DecodeMasterKey(keyBase64 string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid master key (base64): %w", err)
	}

	if len(key) < 32 {
		return nil, fmt.Errorf("master key too short: got %d bytes, minimum 32 required", len(key))
	}

	return key, nil
}
