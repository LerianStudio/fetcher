// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package engine_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/engine"
)

// completedAt is a stable timestamp the result tests share.
func completedAt(t *testing.T) time.Time {
	t.Helper()

	ts, err := time.Parse(time.RFC3339, "2026-06-01T12:00:00Z")
	if err != nil {
		t.Fatalf("parse fixture timestamp: %v", err)
	}

	return ts
}

func TestDirectResult_CarriesDataAndNoRequiredReference(t *testing.T) {
	t.Parallel()

	data := []byte(`[{"id":1},{"id":2}]`)
	direct := engine.DirectResult{
		Data:          data,
		Format:        "json",
		RowCount:      2,
		PlaintextSize: int64(len(data)),
		CompletedAt:   completedAt(t),
	}

	if string(direct.Data) != string(data) {
		t.Fatalf("DirectResult.Data = %q, want %q", direct.Data, data)
	}
	if direct.Format != "json" {
		t.Fatalf("DirectResult.Format = %q, want json", direct.Format)
	}
	if direct.RowCount != 2 {
		t.Fatalf("DirectResult.RowCount = %d, want 2", direct.RowCount)
	}
	if direct.PlaintextSize != int64(len(data)) {
		t.Fatalf("DirectResult.PlaintextSize = %d, want %d", direct.PlaintextSize, len(data))
	}
	if direct.CompletedAt.IsZero() {
		t.Fatalf("DirectResult.CompletedAt must be set")
	}

	// Direct mode carries no sink reference: the result is whole on its own.
	// Reference is a nil-discriminated pointer, so direct mode leaves it nil.
	result := engine.ExtractionResult{Direct: &direct}
	if result.Direct == nil {
		t.Fatalf("ExtractionResult.Direct must be retained")
	}
	if result.Reference != nil {
		t.Fatalf("direct-mode result must carry no populated sink reference, got %+v", result.Reference)
	}
}

func TestDirectResult_CarriesCanonicalIntegrityAndProtection(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte("rows"))
	direct := engine.DirectResult{
		Data:        []byte("rows"),
		Format:      "json",
		CompletedAt: completedAt(t),
		Integrity: &engine.ResultIntegrity{
			Algorithm: "SHA-256",
			Digest:    hex.EncodeToString(sum[:]),
		},
		Protection: &engine.ResultProtection{
			Encrypted:  true,
			KeyVersion: 3,
			Mode:       "AES-256-GCM",
			AppliedBy:  engine.ProtectionAppliedByEngine,
		},
	}

	if direct.Integrity == nil || direct.Integrity.Algorithm != "SHA-256" {
		t.Fatalf("DirectResult.Integrity.Algorithm = %+v, want SHA-256", direct.Integrity)
	}
	if direct.Integrity.Digest == "" {
		t.Fatalf("DirectResult.Integrity.Digest must be set when integrity applies")
	}
	if !direct.Protection.Encrypted {
		t.Fatalf("DirectResult.Protection.Encrypted must be true")
	}
	if direct.Protection.AppliedBy != engine.ProtectionAppliedByEngine {
		t.Fatalf("DirectResult.Protection.AppliedBy = %q, want engine", direct.Protection.AppliedBy)
	}
}

func TestResultReference_CarriesLogicalReferenceNoPhysicalBackend(t *testing.T) {
	t.Parallel()

	ref := engine.ResultReference{
		Path:        "memory://tenant-a/abc123-1",
		Format:      "json",
		RowCount:    7,
		SizeBytes:   42,
		CompletedAt: completedAt(t),
		Integrity: &engine.ResultIntegrity{
			Algorithm: "HMAC-SHA-256",
			Signature: "deadbeef",
		},
		Protection: &engine.ResultProtection{
			Encrypted: true,
			AppliedBy: engine.ProtectionAppliedByAdapter,
		},
	}

	if ref.Path == "" {
		t.Fatalf("ResultReference.Path must carry the logical reference")
	}
	if ref.Format != "json" || ref.RowCount != 7 || ref.SizeBytes != 42 {
		t.Fatalf("ResultReference lost format/rowCount/sizeBytes: %+v", ref)
	}
	if ref.Integrity == nil || ref.Integrity.Signature == "" {
		t.Fatalf("store-mode reference must carry integrity for the bytes written")
	}
	if ref.Protection == nil || ref.Protection.AppliedBy != engine.ProtectionAppliedByAdapter {
		t.Fatalf("store-mode reference must carry protection metadata")
	}

	// The reference must NOT expose a physical backend type. A field literally
	// named for S3/SeaweedFS/MongoDB has no place on the logical reference.
	refType := reflect.TypeOf(ref)
	for i := 0; i < refType.NumField(); i++ {
		switch refType.Field(i).Name {
		case "Backend", "BackendType", "S3", "SeaweedFS", "Bucket", "Mongo":
			t.Fatalf("ResultReference must not expose physical backend field %q", refType.Field(i).Name)
		}
	}
}

func TestResultIntegrity_DigestOrSignature(t *testing.T) {
	t.Parallel()

	// HMAC is ONE possible integrity signature, not the result identity itself.
	key := []byte("external-hmac-key")
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("payload"))
	signature := hex.EncodeToString(mac.Sum(nil))

	hmacIntegrity := engine.ResultIntegrity{Algorithm: "HMAC-SHA-256", Signature: signature}
	if !hmacIntegrity.IsPresent() {
		t.Fatalf("integrity with a signature must report present")
	}
	if hmacIntegrity.Signature == "" || hmacIntegrity.Digest != "" {
		t.Fatalf("HMAC integrity must be a signature, not a digest: %+v", hmacIntegrity)
	}

	sum := sha256.Sum256([]byte("payload"))
	digestIntegrity := engine.ResultIntegrity{Algorithm: "SHA-256", Digest: hex.EncodeToString(sum[:])}
	if !digestIntegrity.IsPresent() {
		t.Fatalf("integrity with a digest must report present")
	}

	empty := engine.ResultIntegrity{}
	if empty.IsPresent() {
		t.Fatalf("empty integrity must report not present")
	}
}

func TestResultProtection_AppliedByValidation(t *testing.T) {
	t.Parallel()

	for _, applier := range []engine.ProtectionApplier{
		engine.ProtectionAppliedByEngine,
		engine.ProtectionAppliedByAdapter,
		engine.ProtectionAppliedByHost,
	} {
		if !applier.IsValid() {
			t.Fatalf("ProtectionApplier %q must be valid", applier)
		}
	}

	for _, bad := range []engine.ProtectionApplier{"", "system", "credential", "user", "ENGINE"} {
		if bad.IsValid() {
			t.Fatalf("ProtectionApplier %q must be rejected", bad)
		}
	}
}

// TestResultProtection_NotReusingCredentialProtection guards the critical design
// boundary: result-byte protection must be its OWN canonical type and must NOT
// reuse the credential-encryption sidecar (ProtectedCredential) or its field
// names/terminology.
func TestResultProtection_NotReusingCredentialProtection(t *testing.T) {
	t.Parallel()

	protectionType := reflect.TypeOf(engine.ResultProtection{})
	credentialType := reflect.TypeOf(engine.ProtectedCredential{})

	if protectionType == credentialType {
		t.Fatalf("ResultProtection must not be the credential ProtectedCredential type")
	}

	// Credential machinery carries Ciphertext bytes. Result protection metadata
	// describes state and must never embed credential ciphertext.
	for i := 0; i < protectionType.NumField(); i++ {
		if protectionType.Field(i).Name == "Ciphertext" {
			t.Fatalf("ResultProtection must not reuse credential 'Ciphertext' field")
		}
	}

	// Sanity: the credential type is the one carrying Ciphertext, so the two are
	// genuinely distinct models, not aliases.
	if _, ok := credentialType.FieldByName("Ciphertext"); !ok {
		t.Fatalf("test premise broken: ProtectedCredential should carry Ciphertext")
	}
}

// TestExtractionResult_NoSensitivePayloadInReference asserts the reference and
// errors describe the result without embedding the payload bytes.
func TestExtractionResult_NoSensitivePayloadInReference(t *testing.T) {
	t.Parallel()

	refType := reflect.TypeOf(engine.ResultReference{})
	for i := 0; i < refType.NumField(); i++ {
		if refType.Field(i).Type.Kind() == reflect.Slice &&
			refType.Field(i).Type.Elem().Kind() == reflect.Uint8 {
			t.Fatalf("ResultReference must not carry a raw byte payload field %q", refType.Field(i).Name)
		}
		switch refType.Field(i).Name {
		case "Data", "Payload", "Rows", "Body":
			t.Fatalf("ResultReference must not embed payload field %q", refType.Field(i).Name)
		}
	}
}
