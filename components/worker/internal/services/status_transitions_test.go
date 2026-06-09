// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	workerCrypto "github.com/LerianStudio/fetcher/v2/pkg/crypto"
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// enginePathUseCase builds a UseCase wired with a capturing engine runner that
// returns the supplied direct rows, plus a real signer + valid storage key so the
// full success path (engine -> save -> sign -> complete) runs.
func engineSuccessUseCase(t *testing.T, mocks *testMocks, rows map[string]map[string][]map[string]any) *UseCase {
	t.Helper()

	uc := newTestUseCase(mocks)
	uc.SetStorageEncryptDerivedKey(storageKey)

	signer, err := workerCrypto.NewHMACSigner(hmacKey, "v1")
	require.NoError(t, err)
	uc.DocumentSigner = signer

	payload, err := json.Marshal(rows)
	require.NoError(t, err)

	uc.EngineRunner = &capturingEngineRunner{
		result: engine.ExtractionResult{
			Direct: &engine.DirectResult{Data: payload, Format: "json", RowCount: countTotalRows(rows)},
		},
	}

	return uc
}

// TestExtractExternalData_EnginePath_TransitionOrder_PendingProcessingCompleted
// locks the SUCCESS transition ORDER on the engine path: the job is moved
// PENDING -> PROCESSING (repo) BEFORE the engine runs, and PROCESSING -> COMPLETED
// (repo) is persisted BEFORE the job.completed event is published. Order is
// asserted with gomock.InOrder, not just final state.
func TestExtractExternalData_EnginePath_TransitionOrder_PendingProcessingCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	rows := map[string]map[string][]map[string]any{"pg": {"users": {{"id": float64(1)}}}}
	uc := engineSuccessUseCase(t, mocks, rows)

	jobID := newTestJobID()
	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID:        jobID,
		MappedFields: map[string]map[string][]string{"pg": {"users": {"id"}}},
		Metadata:     map[string]any{"source": "test"},
	})

	pendingJob := &model.Job{ID: jobID, Status: model.JobStatusPending}
	connection := &model.Connection{ConfigName: "pg", Type: model.TypePostgreSQL}

	// shouldSkipProcessing + job validation both read the pending job.
	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(pendingJob, nil).Times(2)
	mocks.connRepo.EXPECT().FindByConfigNames(gomock.Any(), []string{"pg"}).Return([]*model.Connection{connection}, nil)

	processing := mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).
		Return(nil)

	storePut := mocks.seaweedFS.EXPECT().
		Put(gomock.Any(), constant.ExternalDataKeyPrefix+"/"+jobID.String()+".json", gomock.Any()).
		Return(nil).
		After(processing)

	completed := mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusCompleted, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		After(storePut)

	// The completed-event publish MUST happen AFTER the completed status is persisted.
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		Return(nil).
		After(completed)

	require.NoError(t, uc.ExtractExternalData(testContext(), body, nil))
}

// TestExtractExternalData_EnginePath_TransitionOrder_PendingProcessingFailed locks
// the FAILURE transition ORDER on the engine path: PENDING -> PROCESSING before the
// engine runs, then an engine error drives PROCESSING -> FAILED (repo) BEFORE the
// job.failed event publishes. The engine error is a CategoryValidation error — the
// ST-01 Option-2 surviving delta (missing/malformed table fails at plan time) — and
// it must still map to the existing FAILED status + failed event.
func TestExtractExternalData_EnginePath_TransitionOrder_PendingProcessingFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.EngineRunner = erroringEngineRunner{
		err: engine.NewEngineError(engine.CategoryValidation, "extraction references an unknown datasource connection"),
	}

	jobID := newTestJobID()
	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID:        jobID,
		MappedFields: map[string]map[string][]string{"pg": {"ghost": {"id"}}},
		Metadata:     map[string]any{"source": "test"},
	})

	pendingJob := &model.Job{ID: jobID, Status: model.JobStatusPending}
	connection := &model.Connection{ConfigName: "pg", Type: model.TypePostgreSQL}

	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(pendingJob, nil).Times(2)
	mocks.connRepo.EXPECT().FindByConfigNames(gomock.Any(), []string{"pg"}).Return([]*model.Connection{connection}, nil)

	processing := mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).
		Return(nil)

	failed := mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, "", "", gomock.Any()).
		Return(nil).
		After(processing)

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil).
		After(failed)

	err := uc.ExtractExternalData(testContext(), body, nil)
	require.Error(t, err)
}

// TestExtractExternalData_EnginePath_ValidationError_FailureEventPayloadCompatible
// proves the engine CategoryValidation error maps to a failure-event payload that is
// SHAPE-compatible with the current contract: same jobId, status "failed", the
// free-form sanitized metadata.error.message, and metadata.source preserved. Only
// the opaque error string text differs (the surviving ST-01 delta), which the
// free-form error field already accommodates.
func TestExtractExternalData_EnginePath_ValidationError_FailureEventPayloadCompatible(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.EngineRunner = erroringEngineRunner{
		err: engine.NewEngineError(engine.CategoryValidation, "extraction datasource selects no tables"),
	}

	jobID := newTestJobID()
	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID:        jobID,
		MappedFields: map[string]map[string][]string{"pg": {"users": {"id"}}},
		Metadata:     map[string]any{"source": "plugin_crm"},
	})

	pendingJob := &model.Job{ID: jobID, Status: model.JobStatusPending}
	connection := &model.Connection{ConfigName: "pg", Type: model.TypePostgreSQL}

	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(pendingJob, nil).Times(2)
	mocks.connRepo.EXPECT().FindByConfigNames(gomock.Any(), []string{"pg"}).Return([]*model.Connection{connection}, nil)
	mocks.jobRepo.EXPECT().UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).Return(nil)
	mocks.jobRepo.EXPECT().UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, "", "", gomock.Any()).Return(nil)

	var failedPayload []byte
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, payload []byte) error {
			failedPayload = payload
			return nil
		})

	require.Error(t, uc.ExtractExternalData(testContext(), body, nil))

	var notification struct {
		JobID    string         `json:"jobId"`
		Status   string         `json:"status"`
		Metadata map[string]any `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal(failedPayload, &notification))

	// Shape-compatible with the current failure-notification contract.
	assert.Equal(t, jobID.String(), notification.JobID)
	assert.Equal(t, "failed", notification.Status)
	assert.Equal(t, "plugin_crm", notification.Metadata["source"], "metadata.source preserved on failure")

	errMeta, ok := notification.Metadata["error"].(map[string]any)
	require.True(t, ok, "failure payload must carry metadata.error object, got %T", notification.Metadata["error"])
	msg, ok := errMeta["message"].(string)
	require.True(t, ok, "metadata.error.message must be a string")
	assert.NotEmpty(t, msg, "failure error message must be populated")
	// The engine error is credential-free and carries no URI, so it survives
	// sanitization intact and is not redacted away.
	assert.NotContains(t, msg, "[redacted]")
}

// TestExtractExternalData_EnginePath_StreamingDisabled_FailsClosed locks the CURRENT
// hardened semantics: event emission is MANDATORY. When streaming is disabled, the
// completed status is still persisted (repo update happens), but the overall
// operation FAILS because the required completion event cannot be emitted — the
// message is not silently ACKed without its event. This is the hardened behavior in
// this codebase, NOT a pre-hardening "nil publisher = no-op".
func TestExtractExternalData_EnginePath_StreamingDisabled_FailsClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	rows := map[string]map[string][]map[string]any{"pg": {"users": {{"id": float64(1)}}}}
	uc := engineSuccessUseCase(t, mocks, rows)
	uc.JobEventStreamingEnabled = false // streaming disabled

	jobID := newTestJobID()
	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID:        jobID,
		MappedFields: map[string]map[string][]string{"pg": {"users": {"id"}}},
		Metadata:     map[string]any{"source": "test"},
	})

	pendingJob := &model.Job{ID: jobID, Status: model.JobStatusPending}
	connection := &model.Connection{ConfigName: "pg", Type: model.TypePostgreSQL}

	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(pendingJob, nil).Times(2)
	mocks.connRepo.EXPECT().FindByConfigNames(gomock.Any(), []string{"pg"}).Return([]*model.Connection{connection}, nil)
	mocks.jobRepo.EXPECT().UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).Return(nil)
	mocks.seaweedFS.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	// Completed status IS persisted first.
	mocks.jobRepo.EXPECT().UpdateStatus(gomock.Any(), jobID, model.JobStatusCompleted, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	// NO Publish call expected: streaming disabled. The operation fails closed.

	err := uc.ExtractExternalData(testContext(), body, nil)
	require.Error(t, err, "mandatory event emission disabled must fail the operation, not silently ACK")
}
