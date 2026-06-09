// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	workerCrypto "github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// integrityDirectRunner is an EngineRunner that returns a canned DirectResult, stamping
// the engine's canonical SHA-256 digest over the payload exactly as the real engine
// does. It lets the worker fast-path tests exercise reuse without a live engine.
type integrityDirectRunner struct {
	payload  []byte
	rowCount int64
	// digestOverride, when non-empty, replaces the computed SHA-256 digest so an
	// integrity-mismatch can be forced.
	digestOverride string
	// omitIntegrity drops the integrity arm entirely.
	omitIntegrity bool
}

func (r integrityDirectRunner) RunExtraction(
	context.Context,
	engine.TenantContext,
	string,
	engine.ExtractionRequest,
) (engine.ExtractionResult, error) {
	direct := &engine.DirectResult{
		Data:          r.payload,
		Format:        "json",
		RowCount:      r.rowCount,
		PlaintextSize: int64(len(r.payload)),
	}

	if !r.omitIntegrity {
		digest := r.digestOverride
		if digest == "" {
			sum := sha256.Sum256(r.payload)
			digest = hex.EncodeToString(sum[:])
		}

		direct.Integrity = &engine.ResultIntegrity{Algorithm: "SHA-256", Digest: digest}
	}

	return engine.ExtractionResult{Direct: direct}, nil
}

// genericOnlyMessage + connections describe a generic-only (no CRM) job so the split
// routes everything through the engine and the fast path is eligible.
func genericOnlyMessage() (ExtractExternalDataMessage, []*model.Connection) {
	msg := ExtractExternalDataMessage{
		JobID:        newTestJobID(),
		MappedFields: map[string]map[string][]string{"pg": {"users": {"id"}}},
		Metadata:     map[string]any{"source": "test"},
	}
	conns := []*model.Connection{{ConfigName: "pg", Type: model.TypePostgreSQL}}

	return msg, conns
}

// engineArtifact is the indented JSON shape the engine now emits — identical to
// json.MarshalIndent(result, "", "  ") for the same data.
func engineArtifact(t *testing.T) []byte {
	t.Helper()

	rows := map[string]map[string][]map[string]any{
		"pg": {"users": {{"id": float64(1), "name": "Ada"}, {"id": float64(2), "name": "Linus"}}},
	}

	b, err := json.MarshalIndent(rows, "", "  ")
	require.NoError(t, err)

	return b
}

// TestSaveExternalData_GenericOnlyReusesEngineBytes proves the fast path: when reuse
// is supplied, saveExternalData persists the ENGINE's bytes verbatim (no re-marshal)
// and reports the ENGINE's RowCount. The stored ciphertext decrypts to exactly the
// engine payload, and the HMAC signs exactly those bytes.
func TestSaveExternalData_GenericOnlyReusesEngineBytes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.SetStorageEncryptDerivedKey(storageKey)

	signer, err := workerCrypto.NewHMACSigner(hmacKey, "v1")
	require.NoError(t, err)
	uc.DocumentSigner = signer

	payload := engineArtifact(t)
	sum := sha256.Sum256(payload)
	reuse := &directReuse{
		plaintext: payload,
		rowCount:  42, // deliberately != the actual row count to prove RowCount comes from reuse
		integrity: &engine.ResultIntegrity{Algorithm: "SHA-256", Digest: hex.EncodeToString(sum[:])},
	}

	var stored []byte
	mocks.seaweedFS.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, data []byte) error { stored = data; return nil })

	msg := ExtractExternalDataMessage{JobID: newTestJobID(), Metadata: map[string]any{"source": "test"}}
	// The result map is EMPTY: the fast path must not consult it.
	resultData, err := uc.saveExternalData(testContext(), testTracer(), msg, map[string]map[string][]map[string]any{}, reuse, nil, testLogger())
	require.NoError(t, err)

	// RowCount comes from the engine, not from counting the (empty) map.
	assert.Equal(t, int64(42), resultData.RowCount, "RowCount must be the engine's authoritative count")
	assert.Equal(t, int64(len(payload)), resultData.SizeBytes, "SizeBytes must be the engine byte length")

	// Stored ciphertext decrypts to the EXACT engine bytes (no re-marshal).
	assert.Equal(t, payload, decryptStored(t, stored), "stored plaintext must be the engine bytes verbatim")

	// HMAC signs the engine bytes.
	expectedHMAC, err := signer.SignReader(bytes.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, expectedHMAC, resultData.HMAC)
}

// TestSaveExternalData_FastPathByteIdenticalToMapMarshal is the byte-identity gate:
// for the SAME generic-only data, the fast path (reuse engine bytes) and the fallback
// (json.MarshalIndent the result map) produce byte-identical stored plaintext, HMAC,
// size, and row count. This is the load-bearing claim that reuse is safe.
func TestSaveExternalData_FastPathByteIdenticalToMapMarshal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	signer, err := workerCrypto.NewHMACSigner(hmacKey, "v1")
	require.NoError(t, err)

	rows := map[string]map[string][]map[string]any{
		"pg": {"users": {{"id": float64(1), "name": "Ada"}, {"id": float64(2), "name": "Linus"}}},
	}

	// The engine's serialized form is MarshalIndent — exactly what the fallback produces.
	enginePayload, err := json.MarshalIndent(rows, "", "  ")
	require.NoError(t, err)
	sum := sha256.Sum256(enginePayload)

	msg := ExtractExternalDataMessage{JobID: newTestJobID(), Metadata: map[string]any{"source": "test"}}

	// Fallback path: result map serialized worker-side (reuse nil).
	fbMocks := newTestMocks(ctrl)
	fbUC := newTestUseCase(fbMocks)
	fbUC.SetStorageEncryptDerivedKey(storageKey)
	fbUC.DocumentSigner = signer

	var fbStored []byte
	fbMocks.seaweedFS.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, data []byte) error { fbStored = data; return nil })

	fbResult, err := fbUC.saveExternalData(testContext(), testTracer(), msg, rows, nil, nil, testLogger())
	require.NoError(t, err)

	// Fast path: same data via engine bytes (reuse non-nil).
	fpMocks := newTestMocks(ctrl)
	fpUC := newTestUseCase(fpMocks)
	fpUC.SetStorageEncryptDerivedKey(storageKey)
	fpUC.DocumentSigner = signer

	var fpStored []byte
	fpMocks.seaweedFS.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, data []byte) error { fpStored = data; return nil })

	reuse := &directReuse{
		plaintext: enginePayload,
		rowCount:  countTotalRows(rows),
		integrity: &engine.ResultIntegrity{Algorithm: "SHA-256", Digest: hex.EncodeToString(sum[:])},
	}
	fpResult, err := fpUC.saveExternalData(testContext(), testTracer(), msg, map[string]map[string][]map[string]any{}, reuse, nil, testLogger())
	require.NoError(t, err)

	// Byte-identical stored plaintext, HMAC, size, and row count.
	assert.Equal(t, decryptStored(t, fbStored), decryptStored(t, fpStored), "stored plaintext must be byte-identical")
	assert.Equal(t, fbResult.HMAC, fpResult.HMAC, "HMAC must be byte-identical")
	assert.Equal(t, fbResult.SizeBytes, fpResult.SizeBytes, "size must be identical")
	assert.Equal(t, fbResult.RowCount, fpResult.RowCount, "row count must be identical")
}

// TestSaveExternalData_FastPathIntegrityMismatchFails proves the reused digest is
// finally USED: when the engine's SHA-256 does not match the reused bytes (corruption
// between engine and save), the save fails with a clear integrity error and never
// persists the artifact.
func TestSaveExternalData_FastPathIntegrityMismatchFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.SetStorageEncryptDerivedKey(storageKey)

	signer, err := workerCrypto.NewHMACSigner(hmacKey, "v1")
	require.NoError(t, err)
	uc.DocumentSigner = signer

	payload := engineArtifact(t)
	reuse := &directReuse{
		plaintext: payload,
		rowCount:  2,
		// A wrong digest: the bytes do not hash to this value.
		integrity: &engine.ResultIntegrity{Algorithm: "SHA-256", Digest: "deadbeef"},
	}

	// Put must NEVER be called: the mismatch aborts before any persistence.
	msg := ExtractExternalDataMessage{JobID: newTestJobID(), Metadata: map[string]any{"source": "test"}}
	_, err = uc.saveExternalData(testContext(), testTracer(), msg, map[string]map[string][]map[string]any{}, reuse, nil, testLogger())
	require.Error(t, err)
	require.ErrorContains(t, err, "integrity mismatch")
}

// TestExtractInto_GenericOnlyReturnsReuse proves the wiring: a generic-only job routes
// through the engine and extractInto returns a non-nil directReuse carrying the
// engine's bytes and row count. A subsequent integrity check passes for honest bytes.
func TestExtractInto_GenericOnlyReturnsReuse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	payload := engineArtifact(t)
	uc.EngineRunner = integrityDirectRunner{payload: payload, rowCount: 2}

	msg, conns := genericOnlyMessage()
	result := make(map[string]map[string][]map[string]any)

	reuse, err := uc.extractInto(testContext(), msg, conns, result)
	require.NoError(t, err)
	require.NotNil(t, reuse, "a generic-only job must yield a reusable engine payload")
	assert.Equal(t, payload, reuse.plaintext)
	assert.Equal(t, int64(2), reuse.rowCount)
	require.NoError(t, reuse.verifyIntegrity(), "engine-stamped digest must verify for honest bytes")

	// The result map is NOT populated on the fast path: the bytes are reused directly.
	assert.Empty(t, result, "generic-only fast path must not rehydrate the result map")
}
