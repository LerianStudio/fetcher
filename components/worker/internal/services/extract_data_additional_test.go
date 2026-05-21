package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/constant"
	workerCrypto "github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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
	mockDataSource := modelDatasource.NewMockDataSource(ctrl)

	uc.SetDataSourceFactory(func(_ context.Context, _ *model.Connection, _ workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
		return mockDataSource, nil
	})

	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID).Return(pendingJob, nil).Times(2)
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).
		Return(nil)
	mocks.connRepo.EXPECT().FindByConfigNames(gomock.Any(), []string{"postgres_db"}).Return([]*model.Connection{connection}, nil)
	mockDataSource.EXPECT().Connect(gomock.Any(), gomock.Any()).Return(nil)
	mockDataSource.EXPECT().Query(gomock.Any(), map[string][]string{"users": {"id", "name"}}, nil, gomock.Any()).
		Return(map[string][]map[string]any{"users": {{"id": 1, "name": "Ada"}}}, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)
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

func TestQueryDatabase_DataSourceFactoryAndLifecycleErrors(t *testing.T) {
	tests := []struct {
		name          string
		factoryErr    error
		connectErr    error
		queryErr      error
		closeErr      error
		wantErrSubstr string
	}{
		{name: "factory error", factoryErr: errors.New("factory failed"), wantErrSubstr: "failed to create data source"},
		{name: "connect error", connectErr: errors.New("connect failed"), wantErrSubstr: "failed to connect to postgres_db"},
		{name: "query error", queryErr: errors.New("query failed"), wantErrSubstr: "failed to query postgres_db"},
		{name: "close error is logged as warning but does not fail the query", closeErr: errors.New("close failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := newTestMocks(ctrl)
			uc := newTestUseCase(mocks)
			ctx := testContext()
			logger := testLogger()

			connection := &model.Connection{ConfigName: "postgres_db", Type: model.TypePostgreSQL}
			mockDataSource := modelDatasource.NewMockDataSource(ctrl)

			uc.SetDataSourceFactory(func(_ context.Context, _ *model.Connection, _ workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
				if tt.factoryErr != nil {
					return nil, tt.factoryErr
				}
				return mockDataSource, nil
			})

			if tt.factoryErr == nil {
				mockDataSource.EXPECT().Connect(gomock.Any(), gomock.Any()).Return(tt.connectErr)
			}

			if tt.factoryErr == nil && tt.connectErr == nil {
				mockDataSource.EXPECT().Query(gomock.Any(), map[string][]string{"users": {"id"}}, nil, gomock.Any()).
					Return(map[string][]map[string]any{"users": {{"id": 1}}}, tt.queryErr)
				mockDataSource.EXPECT().Close(gomock.Any()).Return(tt.closeErr)
			}

			result := make(map[string]map[string][]map[string]any)
			err := uc.queryDatabase(ctx, "postgres_db", map[string][]string{"users": {"id"}}, []*model.Connection{connection}, nil, result, logger, testTracer())

			if tt.wantErrSubstr == "" {
				require.NoError(t, err)
				assert.Len(t, result["postgres_db"]["users"], 1)
				return
			}

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

	resultData, err := uc.saveExternalData(ctx, testTracer(), message, result, nil, logger)
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
