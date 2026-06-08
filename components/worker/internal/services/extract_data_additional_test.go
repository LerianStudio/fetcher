package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/constant"
	workerCrypto "github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fixedDirectRunner is an EngineRunner returning fixed DIRECT-mode bytes, used by
// generic-extraction tests after the legacy datasource path was removed.
type fixedDirectRunner struct{ data []byte }

func (r *fixedDirectRunner) RunExtraction(context.Context, engine.TenantContext, string, engine.ExtractionRequest) (engine.ExtractionResult, error) {
	return engine.ExtractionResult{Direct: &engine.DirectResult{Data: r.data, Format: "json"}}, nil
}

func TestExtractExternalData_JobNotFoundMarksFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"users": {"id"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	})

	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(nil, nil).Times(2)
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err := uc.ExtractExternalData(ctx, body, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "Job not found in database")
}

func TestExtractExternalData_NoConnectionsMarksFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"users": {"id"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	})

	pendingJob := &model.Job{ID: jobID, Status: model.JobStatusPending}
	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(pendingJob, nil).Times(2)
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).
		Return(nil)
	mocks.connRepo.EXPECT().FindByConfigNames(gomock.Any(), []string{"postgres_db"}).Return([]*model.Connection{}, nil)
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err := uc.ExtractExternalData(ctx, body, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "No connections found for config names")
}

func TestExtractExternalData_CompletedStatusUpdateFailureMarksJobFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.DocumentSigner = workerCrypto.NewMockSigner(ctrl)
	// Override with valid 32-byte derived key for storage encryption.
	uc.SetStorageEncryptDerivedKey([]byte("01234567890123456789012345678901"))

	// Generic extraction now flows through the Engine runner (strangler complete).
	uc.EngineRunner = &fixedDirectRunner{data: []byte(`{"postgres_db":{"users":[{"id":1,"name":"Ada"}]}}`)}

	ctx := testContext()
	jobID := newTestJobID()

	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"users": {"id", "name"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	})

	pendingJob := &model.Job{ID: jobID, Status: model.JobStatusPending}
	connection := &model.Connection{ConfigName: "postgres_db", Type: model.TypePostgreSQL}

	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(pendingJob, nil).Times(2)
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).
		Return(nil)
	mocks.connRepo.EXPECT().FindByConfigNames(gomock.Any(), []string{"postgres_db"}).Return([]*model.Connection{connection}, nil)
	signer := uc.DocumentSigner.(*workerCrypto.MockSigner)
	signer.EXPECT().SignReader(gomock.Any()).Return("test-hmac", nil)
	mocks.seaweedFS.EXPECT().Put(gomock.Any(), constant.ExternalDataKeyPrefix+"/"+jobID.String()+".json", gomock.Any()).Return(nil)
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusCompleted, constant.ExternalDataKeyPrefix+"/"+jobID.String()+".json", "test-hmac", gomock.Any()).
		Return(errors.New("status update failed"))
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, "", "", gomock.Any()).
		Return(nil)
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err := uc.ExtractExternalData(ctx, body, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "status update failed")
}

// TestQueryPluginCRMDatabase_DataSourceFactoryAndLifecycleErrors exercises the
// connection lifecycle of the plugin_crm extraction path (the only remaining
// direct-datasource path after the strangler completion): factory error, connect
// error, and the connection-not-found case. The generic-datasource Query lifecycle
// is no longer a Worker concern — generic extraction runs through the Engine.
func TestQueryPluginCRMDatabase_DataSourceFactoryAndLifecycleErrors(t *testing.T) {
	tests := []struct {
		name          string
		noConnection  bool
		factoryErr    error
		connectErr    error
		wantErrSubstr string
	}{
		{name: "connection not found", noConnection: true, wantErrSubstr: "connection not found for database: plugin_crm"},
		{name: "factory error", factoryErr: errors.New("factory failed"), wantErrSubstr: "failed to create data source"},
		{name: "connect error", connectErr: errors.New("connect failed"), wantErrSubstr: "failed to connect"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := newTestMocks(ctrl)
			uc := newTestUseCase(mocks)
			ctx := testContext()

			connections := []*model.Connection{{ConfigName: "plugin_crm", Type: model.TypeMongoDB}}
			if tt.noConnection {
				connections = nil
			}

			mockDataSource := modelDatasource.NewMockDataSource(ctrl)
			uc.SetDataSourceFactory(func(_ context.Context, _ *model.Connection, _ workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
				if tt.factoryErr != nil {
					return nil, tt.factoryErr
				}
				return mockDataSource, nil
			})

			if !tt.noConnection && tt.factoryErr == nil {
				mockDataSource.EXPECT().Connect(gomock.Any(), gomock.Any()).Return(tt.connectErr)
				if tt.connectErr != nil {
					// connect failed before the deferred Close is registered; no Close.
				}
			}

			message := ExtractExternalDataMessage{
				JobID:        newTestJobID(),
				MappedFields: map[string]map[string][]string{"plugin_crm": {"holders": {"id"}}},
			}
			result := make(map[string]map[string][]map[string]any)
			err := uc.queryPluginCRMDatabase(ctx, message, connections, result)

			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErrSubstr)
		})
	}
}

func TestSaveExternalDataToSeaweedFS_DocumentSignerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.DocumentSigner = workerCrypto.NewMockSigner(ctrl)

	ctx := testContext()
	logger := testLogger()
	message := ExtractExternalDataMessage{JobID: newTestJobID()}
	result := map[string]map[string][]map[string]any{
		"db1": {"table1": {{"id": 1}}},
	}

	signer := uc.DocumentSigner.(*workerCrypto.MockSigner)
	signer.EXPECT().SignReader(gomock.Any()).Return("", errors.New("sign failed"))

	resultData, err := uc.saveExternalData(ctx, testTracer(), message, result, nil, nil, logger)
	require.Error(t, err)
	require.Nil(t, resultData)
	require.ErrorContains(t, err, "computing document HMAC")
}

func mustMarshalMessage(t *testing.T, message ExtractExternalDataMessage) []byte {
	t.Helper()

	body, err := json.Marshal(message)
	require.NoError(t, err)

	return body
}
