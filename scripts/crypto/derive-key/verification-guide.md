# External Document Verification Guide

This guide explains how external consumers can verify the authenticity and integrity of documents (extracted data) produced by Fetcher.

## Overview

When the Worker extracts data and stores results in SeaweedFS, it signs each document using HMAC-SHA256 with the **external HMAC key**. Consumers can independently verify these signatures to confirm that:

1. The document was produced by a trusted Fetcher instance.
2. The document has not been modified since extraction.

## Key Derivation

Fetcher derives the external HMAC key from the master key (`APP_ENC_KEY`) using HKDF (RFC 5869) with SHA-256 and the context string `fetcher-external-hmac-v1`. You do not need to know the master key derivation internals — just use the provided tool.

### Obtaining the External HMAC Key

Run the following command with the same `APP_ENC_KEY` configured in your Fetcher deployment:

```bash
make derive-key KEY="<your-base64-master-key>"
```

Or using the environment variable directly:

```bash
APP_ENC_KEY="<your-base64-master-key>" make derive-key
```

The output is a 64-character hex-encoded key:

```
External HMAC Key (hex): a1b2c3d4e5f6...
```

Store this key securely in your consumer application. It does **not** grant the ability to decrypt credentials or forge internal messages — it is scoped exclusively to document signature verification.

## Signature Format

Documents are signed using the following protocol:

- **Algorithm**: HMAC-SHA256
- **Signature version**: `v1`
- **Payload format**: `<unix-timestamp>.<document-body>`
- **Signature encoding**: Hexadecimal (64 characters)

The timestamp is a Unix epoch in seconds, concatenated with a dot separator and the raw document body.

## Verification Steps

### 1. Extract Signature Components

From the job completion event or document metadata, extract:

- `signature` — the hex-encoded HMAC signature
- `signature_version` — must be `v1`
- `timestamp` — the Unix timestamp used during signing
- `body` — the raw document content

### 2. Reconstruct the Payload

Concatenate the timestamp and body with a dot separator:

```
payload = "<timestamp>.<body>"
```

### 3. Compute the Expected Signature

Using your external HMAC key (hex-decoded to bytes), compute HMAC-SHA256 over the payload:

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

func VerifySignature(hexKey, signature string, timestamp int64, body []byte) error {
    key, err := hex.DecodeString(hexKey)
    if err != nil {
        return fmt.Errorf("invalid HMAC key: %w", err)
    }

    payload := fmt.Appendf(nil, "%d.", timestamp)
    payload = append(payload, body...)

    mac := hmac.New(sha256.New, key)
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))

    if !hmac.Equal([]byte(expected), []byte(signature)) {
        return fmt.Errorf("signature mismatch")
    }

    return nil
}
```

```python
import hmac
import hashlib

def verify_signature(hex_key: str, signature: str, timestamp: int, body: bytes) -> bool:
    key = bytes.fromhex(hex_key)
    payload = f"{timestamp}.".encode() + body
    expected = hmac.new(key, payload, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature)
```

### 4. Validate the Timestamp

To prevent replay attacks, verify that the timestamp is within an acceptable window (e.g., 5 minutes) of the current time:

```go
if abs(time.Now().Unix() - timestamp) > 300 {
    return fmt.Errorf("signature timestamp too old or too far in the future")
}
```

## Security Considerations

- **Key scope**: The external HMAC key can only verify signatures. It cannot decrypt database credentials (which use a separate AES-256-GCM key) or forge internal Manager-Worker messages (which use a separate internal HMAC key). All three keys are derived from the same master key but are cryptographically independent.
- **Key rotation**: When rotating `APP_ENC_KEY`, generate a new external HMAC key and distribute it to all consumers. Documents signed with the previous key will no longer verify against the new key.
- **Transport security**: Signature verification proves integrity and authenticity, but does not provide confidentiality. Use TLS for transport encryption.
- **Timing attacks**: Always use constant-time comparison (e.g., `hmac.Equal` in Go, `hmac.compare_digest` in Python) when comparing signatures.
