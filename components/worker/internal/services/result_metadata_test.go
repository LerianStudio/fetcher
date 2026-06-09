// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package services

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	workerCrypto "github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// storageKey is a fixed 32-byte AES key used across the result-metadata tests so
// the encrypt/decrypt round-trip is deterministic in plaintext terms.
var storageKey = []byte("01234567890123456789012345678901")

// hmacKey is a fixed >=32-byte HMAC key for the real signer used in parity tests.
var hmacKey = []byte("hmac-key-hmac-key-hmac-key-hmac!!")

// decryptStored reverses encryptData: Base64 decode, AES-256-GCM open with the
// fixed storageKey, returning the recovered plaintext bytes.
func decryptStored(t *testing.T, stored []byte) []byte {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(string(stored))
	require.NoError(t, err)

	block, err := aes.NewCipher(storageKey)
	require.NoError(t, err)

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(raw), gcm.NonceSize())
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]

	plain, err := gcm.Open(nil, nonce, ct, nil)
	require.NoError(t, err)

	return plain
}

func sampleResult() map[string]map[string][]map[string]any {
	return map[string]map[string][]map[string]any{
		"pg": {
			"users": {
				{"id": float64(1), "name": "Ada"},
				{"id": float64(2), "name": "Linus"},
			},
		},
	}
}

// TestSaveExternalData_DeclaresCanonicalIntegrityAndProtection is the ST-02 RED
// contract: the result the Worker attaches to a completed job carries the canonical
// T-007 integrity + protection metadata derived from the EXISTING encrypt/store/HMAC
// behavior — without changing the stored bytes, the HMAC, or the path.
func TestSaveExternalData_DeclaresCanonicalIntegrityAndProtection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.SetStorageEncryptDerivedKey(storageKey)

	signer, err := workerCrypto.NewHMACSigner(hmacKey, "v1")
	require.NoError(t, err)
	uc.DocumentSigner = signer

	ctx := testContext()
	jobID := newTestJobID()
	message := ExtractExternalDataMessage{JobID: jobID, Metadata: map[string]any{"source": "test"}}
	result := sampleResult()

	mocks.seaweedFS.EXPECT().
		Put(gomock.Any(), constant.ExternalDataKeyPrefix+"/"+jobID.String()+".json", gomock.Any()).
		Return(nil)

	resultData, err := uc.saveExternalData(ctx, testTracer(), message, result, nil, nil, testLogger())
	require.NoError(t, err)
	require.NotNil(t, resultData)

	// Canonical integrity: declares the EXACT current HMAC, truthfully labeled.
	require.NotNil(t, resultData.Integrity, "expected canonical integrity metadata")
	assert.Equal(t, "HMAC-SHA256", resultData.Integrity.Algorithm)
	assert.Equal(t, resultData.HMAC, resultData.Integrity.Signature,
		"declared integrity signature must equal the existing result HMAC field")
	assert.Empty(t, resultData.Integrity.Digest, "HMAC is a keyed signature, not an unkeyed digest")

	// Canonical protection: describes the STORED RESULT (not credentials).
	require.NotNil(t, resultData.Protection, "expected canonical protection metadata")
	assert.True(t, resultData.Protection.Encrypted)
	assert.Equal(t, engine.ProtectionAppliedByAdapter, resultData.Protection.AppliedBy)
	assert.Equal(t, "adapter-managed", resultData.Protection.Mode)
}

// TestSaveExternalData_HMACSignsPlaintextByteExact pins the HMAC semantics: it is
// computed over the json.MarshalIndent(result) PLAINTEXT bytes (before encryption),
// HMAC-SHA256. This is the byte-exact preservation gate.
func TestSaveExternalData_HMACSignsPlaintextByteExact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.SetStorageEncryptDerivedKey(storageKey)

	signer, err := workerCrypto.NewHMACSigner(hmacKey, "v1")
	require.NoError(t, err)
	uc.DocumentSigner = signer

	jobID := newTestJobID()
	message := ExtractExternalDataMessage{JobID: jobID, Metadata: map[string]any{"source": "test"}}
	result := sampleResult()

	var stored []byte
	mocks.seaweedFS.EXPECT().
		Put(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, data []byte) error {
			stored = data
			return nil
		})

	resultData, err := uc.saveExternalData(testContext(), testTracer(), message, result, nil, nil, testLogger())
	require.NoError(t, err)

	// Independent expectation: HMAC-SHA256 over the EXACT MarshalIndent plaintext.
	plaintext, err := json.MarshalIndent(result, "", "  ")
	require.NoError(t, err)

	expectedHMAC, err := signer.SignReader(bytes.NewReader(plaintext))
	require.NoError(t, err)

	assert.Equal(t, expectedHMAC, resultData.HMAC, "HMAC must sign the plaintext MarshalIndent bytes")
	assert.Equal(t, expectedHMAC, resultData.Integrity.Signature)

	// The stored bytes are the ENCRYPTION of that exact plaintext (decrypt to verify).
	assert.NotEqual(t, plaintext, stored, "stored bytes must be encrypted, not plaintext")
	assert.Equal(t, plaintext, decryptStored(t, stored), "stored ciphertext must decrypt to the exact plaintext")
}

// TestSaveExternalData_EngineParity proves the T-007 byte-compatibility claim: the
// Engine's DirectResult.Data is the SAME map[config]map[table][]rows JSON shape, so
// feeding the decoded engine result through saveExternalData yields a HMAC and
// stored plaintext byte-identical to feeding the legacy result map directly.
func TestSaveExternalData_EngineParity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	signer, err := workerCrypto.NewHMACSigner(hmacKey, "v1")
	require.NoError(t, err)

	rows := sampleResult()

	// Legacy path: rows go straight into saveExternalData.
	legacyMocks := newTestMocks(ctrl)
	legacyUC := newTestUseCase(legacyMocks)
	legacyUC.SetStorageEncryptDerivedKey(storageKey)
	legacyUC.DocumentSigner = signer

	var legacyStored []byte
	legacyMocks.seaweedFS.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, data []byte) error { legacyStored = data; return nil })

	jobID := newTestJobID()
	msg := ExtractExternalDataMessage{JobID: jobID, Metadata: map[string]any{"source": "test"}}
	legacyResult, err := legacyUC.saveExternalData(testContext(), testTracer(), msg, rows, nil, nil, testLogger())
	require.NoError(t, err)

	// Engine path: the SAME rows, serialized by the engine as DirectResult.Data
	// (indented, matching the engine's MarshalIndent contract), then decoded back
	// through decodeDirectResult into the worker result map.
	enginePayload, err := json.MarshalIndent(rows, "", "  ")
	require.NoError(t, err)

	decoded := make(map[string]map[string][]map[string]any)
	require.NoError(t, decodeDirectResult(
		testContext(),
		&engine.DirectResult{Data: enginePayload, Format: "json"},
		decoded,
		testLogger(),
	))

	engineMocks := newTestMocks(ctrl)
	engineUC := newTestUseCase(engineMocks)
	engineUC.SetStorageEncryptDerivedKey(storageKey)
	engineUC.DocumentSigner = signer

	var engineStored []byte
	engineMocks.seaweedFS.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, data []byte) error { engineStored = data; return nil })

	engineResult, err := engineUC.saveExternalData(testContext(), testTracer(), msg, decoded, nil, nil, testLogger())
	require.NoError(t, err)

	// HMAC byte-identical (the whole parity claim).
	assert.Equal(t, legacyResult.HMAC, engineResult.HMAC, "engine-path HMAC must equal legacy-path HMAC for identical rows")

	// Stored plaintext byte-identical (ciphertext differs by nonce; decrypt to compare).
	assert.Equal(t, decryptStored(t, legacyStored), decryptStored(t, engineStored),
		"engine-path stored plaintext must equal legacy-path stored plaintext")

	// Path + size + rowCount byte-identical.
	assert.Equal(t, legacyResult.Path, engineResult.Path)
	assert.Equal(t, legacyResult.SizeBytes, engineResult.SizeBytes)
	assert.Equal(t, legacyResult.RowCount, engineResult.RowCount)
}

// TestCompleteJob_PublishesCanonicalMetadataInNotification proves the declared
// integrity + protection metadata reaches the CONTRACT surface: the published
// job.completed notification payload carries the result's integrity (HMAC-SHA256 +
// signature) and protection (encrypted, adapter, adapter-managed), alongside the
// unchanged path + hmac fields.
func TestCompleteJob_PublishesCanonicalMetadataInNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	jobID := newTestJobID()
	message := ExtractExternalDataMessage{JobID: jobID, Metadata: map[string]any{"source": "test"}}

	resultData := &JobResultData{
		Path:   "tenant/results/job.json",
		HMAC:   "abc123",
		Format: "json",
		Integrity: &engine.ResultIntegrity{
			Algorithm: "HMAC-SHA256",
			Signature: "abc123",
		},
		Protection: &engine.ResultProtection{
			Encrypted: true,
			AppliedBy: engine.ProtectionAppliedByAdapter,
			Mode:      "adapter-managed",
		},
	}

	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusCompleted, resultData.Path, resultData.HMAC, gomock.Any()).
		Return(nil)

	var publishedPayload []byte
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, payload []byte) error {
			publishedPayload = payload
			return nil
		})

	require.NoError(t, uc.completeJob(testContext(), message, resultData, time.Now().Add(-time.Second), nil, testLogger()))

	var notification struct {
		Result struct {
			HMAC       string                   `json:"hmac"`
			Integrity  *engine.ResultIntegrity  `json:"integrity"`
			Protection *engine.ResultProtection `json:"protection"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(publishedPayload, &notification))

	assert.Equal(t, "abc123", notification.Result.HMAC)
	require.NotNil(t, notification.Result.Integrity)
	assert.Equal(t, "HMAC-SHA256", notification.Result.Integrity.Algorithm)
	assert.Equal(t, "abc123", notification.Result.Integrity.Signature)
	require.NotNil(t, notification.Result.Protection)
	assert.True(t, notification.Result.Protection.Encrypted)
	assert.Equal(t, engine.ProtectionAppliedByAdapter, notification.Result.Protection.AppliedBy)
	assert.Equal(t, "adapter-managed", notification.Result.Protection.Mode)
}
