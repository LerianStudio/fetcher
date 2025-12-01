package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Cryptor exposes encryption and decryption helpers backed by AES-GCM.
// It keeps track of the key version to support rotations.
type Cryptor interface {
	Encrypt(ctx context.Context, plain string) (cipherTextBase64 string, keyVersion string, err error)
	Decrypt(ctx context.Context, cipherTextBase64, keyVersion string) (plain string, err error)
	KeyVersion() string
}

// AESGCMService implements Service using a single 32-byte AES key and a fixed key version label.
type AESGCMService struct {
	key        []byte
	keyVersion string
}

// NewAESGCMServiceFromEnv builds a service from a Base64-encoded 32-byte key.
// keyVersion can be empty; in that case it defaults to "1" to avoid blank metadata.
func NewAESGCMServiceFromEnv(keyBase64, keyVersion string) (*AESGCMService, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key (base64): %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("invalid encryption key length: got %d, expected 32 bytes", len(key))
	}

	if keyVersion == "" {
		keyVersion = "1"
	}

	return &AESGCMService{
		key:        key,
		keyVersion: keyVersion,
	}, nil
}

// Encrypt returns a Base64 string containing nonce + ciphertext.
func (s *AESGCMService) Encrypt(_ context.Context, plain string) (string, string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	// Prepend nonce to ciphertext so we only need to store one blob.
	cipherText := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(cipherText), s.keyVersion, nil
}

// Decrypt converts a Base64 nonce+ciphertext back into plaintext.
func (s *AESGCMService) Decrypt(_ context.Context, cipherTextBase64, keyVersion string) (string, error) {
	if keyVersion != s.keyVersion {
		return "", fmt.Errorf("unsupported key version: %s", keyVersion)
	}
	raw, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext (base64): %w", err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce := raw[:gcm.NonceSize()]
	cipherText := raw[gcm.NonceSize():]

	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

// KeyVersion returns the configured version label.
func (s *AESGCMService) KeyVersion() string {
	return s.keyVersion
}
