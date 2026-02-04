# Document HMAC Verification Guide

This guide explains how external consumers can verify the integrity of data extracted by Fetcher using HMAC signatures.

## Overview

When Fetcher extracts data from external sources, it computes an HMAC-SHA256 signature of the plaintext JSON data **before encryption**. This signature is included in:

1. **Job completion notifications** (RabbitMQ) - `result.hmac` field
2. **Job API responses** - `resultHmac` field

Consumers can use this signature to verify that the decrypted data has not been tampered with.

## Key Derivation

Fetcher uses HKDF (RFC 5869) to derive purpose-specific keys from a master key. The external HMAC key used for document signatures is derived as follows:

| Parameter | Value |
|-----------|-------|
| Algorithm | HKDF-SHA256 |
| Master Key | Your `APP_ENC_KEY` (base64-decoded, min 32 bytes) |
| Salt | None (nil) |
| Info/Context | `fetcher-external-hmac-v1` |
| Output Length | 32 bytes |

### Deriving the Verification Key

#### Using the CLI Tool

```bash
# From the fetcher repository
go run scripts/crypto/derive-key/main.go -key "YOUR_BASE64_MASTER_KEY"
```

#### Go Implementation

```go
package main

import (
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "fmt"
    "io"

    "golang.org/x/crypto/hkdf"
)

func deriveExternalHMACKey(masterKeyBase64 string) ([]byte, error) {
    masterKey, err := base64.StdEncoding.DecodeString(masterKeyBase64)
    if err != nil {
        return nil, fmt.Errorf("invalid base64: %w", err)
    }

    if len(masterKey) < 32 {
        return nil, fmt.Errorf("master key too short: %d bytes", len(masterKey))
    }

    // HKDF with SHA-256, no salt, context as info
    context := "fetcher-external-hmac-v1"
    reader := hkdf.New(sha256.New, masterKey, nil, []byte(context))

    key := make([]byte, 32)
    if _, err := io.ReadFull(reader, key); err != nil {
        return nil, fmt.Errorf("key derivation failed: %w", err)
    }

    return key, nil
}
```

#### Python Implementation

```python
import base64
import hashlib
from cryptography.hazmat.primitives.kdf.hkdf import HKDF
from cryptography.hazmat.primitives import hashes

def derive_external_hmac_key(master_key_base64: str) -> bytes:
    """Derive the external HMAC key from the master key."""
    master_key = base64.b64decode(master_key_base64)

    if len(master_key) < 32:
        raise ValueError(f"Master key too short: {len(master_key)} bytes")

    hkdf = HKDF(
        algorithm=hashes.SHA256(),
        length=32,
        salt=None,
        info=b"fetcher-external-hmac-v1",
    )

    return hkdf.derive(master_key)
```

#### Node.js Implementation

```javascript
const crypto = require('crypto');

function deriveExternalHMACKey(masterKeyBase64) {
    const masterKey = Buffer.from(masterKeyBase64, 'base64');

    if (masterKey.length < 32) {
        throw new Error(`Master key too short: ${masterKey.length} bytes`);
    }

    // HKDF with SHA-256
    const context = Buffer.from('fetcher-external-hmac-v1');
    return crypto.hkdfSync('sha256', masterKey, Buffer.alloc(0), context, 32);
}
```

## Verifying Document HMAC

Once you have the external HMAC key, you can verify the document signature.

### Signature Format

The HMAC signature is a **hex-encoded** HMAC-SHA256 hash of the **plaintext JSON data** (before encryption).

```
signature = hex(HMAC-SHA256(key, plaintext_json_bytes))
```

### Verification Steps

1. **Receive the notification** with `result.hmac` field
2. **Download and decrypt** the data from SeaweedFS
3. **Compute HMAC** of the decrypted JSON bytes
4. **Compare** with the received signature (constant-time comparison)

### Go Verification Example

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func verifyDocumentHMAC(hmacKey, decryptedData []byte, expectedSignature string) bool {
    mac := hmac.New(sha256.New, hmacKey)
    mac.Write(decryptedData)
    computedSignature := hex.EncodeToString(mac.Sum(nil))

    // Constant-time comparison to prevent timing attacks
    return hmac.Equal([]byte(computedSignature), []byte(expectedSignature))
}
```

### Python Verification Example

```python
import hmac
import hashlib

def verify_document_hmac(hmac_key: bytes, decrypted_data: bytes, expected_signature: str) -> bool:
    """Verify the HMAC signature of decrypted data."""
    computed = hmac.new(hmac_key, decrypted_data, hashlib.sha256).hexdigest()
    return hmac.compare_digest(computed, expected_signature)
```

### Node.js Verification Example

```javascript
const crypto = require('crypto');

function verifyDocumentHMAC(hmacKey, decryptedData, expectedSignature) {
    const hmac = crypto.createHmac('sha256', hmacKey);
    hmac.update(decryptedData);
    const computedSignature = hmac.digest('hex');

    // Constant-time comparison
    return crypto.timingSafeEqual(
        Buffer.from(computedSignature),
        Buffer.from(expectedSignature)
    );
}
```

## Complete Verification Flow

```
┌─────────────────┐
│  Job Completed  │
│  Notification   │
│  (RabbitMQ)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Extract result  │
│ path and HMAC   │
│ from message    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Download file   │
│ from SeaweedFS  │
│ (encrypted)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Decrypt file    │
│ using your      │
│ encryption key  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Derive HMAC key │
│ from master key │
│ (HKDF)          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Compute HMAC of │
│ decrypted data  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Compare with    │◄── Match? ✓ Data is authentic
│ received HMAC   │◄── Mismatch? ✗ Data tampered
└─────────────────┘
```

## Notification Message Format

```json
{
  "jobId": "550e8400-e29b-41d4-a716-446655440000",
  "organizationId": "123e4567-e89b-12d3-a456-426614174000",
  "status": "completed",
  "result": {
    "path": "/external-data/550e8400-e29b-41d4-a716-446655440000.json",
    "sizeBytes": 15234,
    "rowCount": 150,
    "format": "json",
    "hmac": "a1b2c3d4e5f6...64_hex_characters..."
  },
  "executionTimeMs": 1523,
  "completedAt": "2025-01-15T10:30:00Z",
  "metadata": {
    "source": "your-service-name"
  }
}
```

## Security Considerations

1. **Store keys securely**: Never commit master keys to version control
2. **Use constant-time comparison**: Prevents timing attacks when comparing signatures
3. **Verify before processing**: Always verify HMAC before trusting decrypted data
4. **Key rotation**: When rotating the master key, all derived keys change automatically

## Troubleshooting

### Signature Mismatch

If verification fails, check:

1. **Correct master key**: Ensure you're using the same `APP_ENC_KEY` as Fetcher
2. **Correct context**: The info string must be exactly `fetcher-external-hmac-v1`
3. **Decryption**: Verify the data was decrypted correctly
4. **Encoding**: The signature is hex-encoded (64 characters for SHA-256)

### Key Derivation Issues

1. **Base64 decoding**: Ensure proper padding in the base64 string
2. **Key length**: Master key must be at least 32 bytes after decoding
3. **HKDF parameters**: No salt, SHA-256, 32-byte output
