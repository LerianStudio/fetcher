package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	workerCrypto "github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	"go.uber.org/mock/gomock"
)

func TestExtractExternalData_JobNotFoundMarksFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()
	orgID := newTestOrgID()
	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID:          jobID,
		OrganizationID: orgID,
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"users": {"id"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	})

	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID, orgID).Return(nil, nil)
	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID, orgID).Return(nil, nil)
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed.test-service", gomock.Any()).
		Return(nil)

	err := uc.ExtractExternalData(ctx, body, nil)
	if err == nil {
		t.Fatal("expected error when job is missing")
	}
	if !strings.Contains(err.Error(), "Job not found in database") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractExternalData_NoConnectionsMarksFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()
	orgID := newTestOrgID()
	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID:          jobID,
		OrganizationID: orgID,
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"users": {"id"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	})

	pendingJob := &model.Job{ID: jobID, Status: model.JobStatusPending}
	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID, orgID).Return(pendingJob, nil)
	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID, orgID).Return(pendingJob, nil)
	mocks.connRepo.EXPECT().FindByConfigNames(gomock.Any(), orgID, []string{"postgres_db"}).Return([]*model.Connection{}, nil)
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed.test-service", gomock.Any()).
		Return(nil)

	err := uc.ExtractExternalData(ctx, body, nil)
	if err == nil {
		t.Fatal("expected error when no connections are found")
	}
	if !strings.Contains(err.Error(), "No connections found for config names") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractExternalData_CompletedStatusUpdateFailureMarksJobFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	originalFactory := newDataSourceFromConnection
	t.Cleanup(func() {
		newDataSourceFromConnection = originalFactory
	})

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	uc.DocumentSigner = workerCrypto.NewMockSigner(ctrl)

	ctx := testContext()
	jobID := newTestJobID()
	orgID := newTestOrgID()
	t.Setenv("CRYPTO_ENCRYPT_SECRET_KEY_SEAWEEDFS", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("CRYPTO_HASH_SECRET_KEY_SEAWEEDFS", "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")

	body := mustMarshalMessage(t, ExtractExternalDataMessage{
		JobID:          jobID,
		OrganizationID: orgID,
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"users": {"id", "name"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	})

	pendingJob := &model.Job{ID: jobID, Status: model.JobStatusPending}
	connection := &model.Connection{ConfigName: "postgres_db", Type: model.TypePostgreSQL}
	mockDataSource := modelDatasource.NewMockDataSource(ctrl)

	newDataSourceFromConnection = func(context.Context, *model.Connection, workerCrypto.Cryptor, libLog.Logger) (modelDatasource.DataSource, error) {
		return mockDataSource, nil
	}

	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID, orgID).Return(pendingJob, nil)
	mocks.jobRepo.EXPECT().FindByID(gomock.Any(), jobID, orgID).Return(pendingJob, nil)
	mocks.connRepo.EXPECT().FindByConfigNames(gomock.Any(), orgID, []string{"postgres_db"}).Return([]*model.Connection{connection}, nil)
	mockDataSource.EXPECT().Connect(gomock.Any(), gomock.Any()).Return(nil)
	mockDataSource.EXPECT().Query(gomock.Any(), map[string][]string{"users": {"id", "name"}}, nil, gomock.Any()).
		Return(map[string][]map[string]any{"users": {{"id": 1, "name": "Ada"}}}, nil)
	mockDataSource.EXPECT().Close(gomock.Any()).Return(nil)
	signer := uc.DocumentSigner.(*workerCrypto.MockSigner)
	signer.EXPECT().SignReader(gomock.Any()).Return("test-hmac", nil)
	mocks.seaweedFS.EXPECT().Put(gomock.Any(), jobID.String()+".json", gomock.Any()).Return(nil)
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusCompleted, "/external-data/"+jobID.String()+".json", "test-hmac", nil).
		Return(errors.New("status update failed"))
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, "", "", gomock.Any()).
		Return(nil)
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed.test-service", gomock.Any()).
		Return(nil)

	err := uc.ExtractExternalData(ctx, body, nil)
	if err == nil {
		t.Fatal("expected error when completed status update fails")
	}
	if !strings.Contains(err.Error(), "status update failed") {
		t.Fatalf("unexpected error: %v", err)
	}
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
		{name: "close error is ignored on success", closeErr: errors.New("close failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			originalFactory := newDataSourceFromConnection
			t.Cleanup(func() {
				newDataSourceFromConnection = originalFactory
			})

			mocks := newTestMocks(ctrl)
			uc := newTestUseCase(mocks)
			ctx := testContext()
			logger := testLogger()

			connection := &model.Connection{ConfigName: "postgres_db", Type: model.TypePostgreSQL}
			mockDataSource := modelDatasource.NewMockDataSource(ctrl)

			newDataSourceFromConnection = func(context.Context, *model.Connection, workerCrypto.Cryptor, libLog.Logger) (modelDatasource.DataSource, error) {
				if tt.factoryErr != nil {
					return nil, tt.factoryErr
				}
				return mockDataSource, nil
			}

			if tt.factoryErr == nil {
				mockDataSource.EXPECT().Connect(gomock.Any(), gomock.Any()).Return(tt.connectErr)
			}

			if tt.factoryErr == nil && tt.connectErr == nil {
				mockDataSource.EXPECT().Query(gomock.Any(), map[string][]string{"users": {"id"}}, nil, gomock.Any()).
					Return(map[string][]map[string]any{"users": {{"id": 1}}}, tt.queryErr)
				mockDataSource.EXPECT().Close(gomock.Any()).Return(tt.closeErr)
			}

			result := make(map[string]map[string][]map[string]any)
			err := uc.queryDatabase(ctx, "postgres_db", map[string][]string{"users": {"id"}}, []*model.Connection{connection}, nil, newTestOrgID(), result, logger, testTracer())

			if tt.wantErrSubstr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if got := result["postgres_db"]["users"]; len(got) != 1 {
					t.Fatalf("expected merged query result, got %+v", result)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErrSubstr, err)
			}
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
	message := ExtractExternalDataMessage{JobID: newTestJobID(), OrganizationID: newTestOrgID()}
	result := map[string]map[string][]map[string]any{
		"db1": {"table1": {{"id": 1}}},
	}

	signer := uc.DocumentSigner.(*workerCrypto.MockSigner)
	signer.EXPECT().SignReader(gomock.Any()).Return("", errors.New("sign failed"))

	resultData, err := uc.saveExternalDataToSeaweedFS(ctx, testTracer(), message, result, nil, logger)
	if err == nil {
		t.Fatal("expected error when document signing fails")
	}
	if resultData != nil {
		t.Fatalf("expected nil result data, got %+v", resultData)
	}
	if !strings.Contains(err.Error(), "computing document HMAC") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustMarshalMessage(t *testing.T, message ExtractExternalDataMessage) []byte {
	t.Helper()

	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}

	return body
}
