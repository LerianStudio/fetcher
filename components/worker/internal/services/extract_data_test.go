package services

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// TestParseMessage_ValidMessage tests successful message parsing.
func TestParseMessage_ValidMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()
	orgID := newTestOrgID()

	validMessage := ExtractExternalDataMessage{
		JobID:          jobID,
		OrganizationID: orgID,
		MappedFields: map[string]map[string][]string{
			"datasource1": {"table1": {"field1", "field2"}},
		},
		Metadata: map[string]any{"source": "test"},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	// No mock expectations needed for JobRepository since message is valid
	logger := testLogger()
	result, err := uc.parseMessage(ctx, body, nil, nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.JobID != jobID {
		t.Fatalf("expected jobID %s, got %s", jobID, result.JobID)
	}

	if result.OrganizationID != orgID {
		t.Fatalf("expected orgID %s, got %s", orgID, result.OrganizationID)
	}
}

// TestParseMessage_InvalidJSON tests error handling for invalid JSON.
func TestParseMessage_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	invalidBody := []byte(`{"jobId": "not-valid-json`)

	// Parsing will fail, but we can't update job status without valid IDs
	result, err := uc.parseMessage(ctx, invalidBody, nil, nil, logger)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}

	if result != nil {
		t.Fatalf("expected nil result for invalid JSON, got %+v", result)
	}
}

// TestParseMessage_InvalidJSONWithJobIDInHeaders tests that jobID can be extracted from headers.
func TestParseMessage_InvalidJSONWithJobIDInHeaders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()
	orgID := newTestOrgID()

	invalidBody := []byte(`{"invalid": json`)
	headers := map[string]any{
		"jobId":          jobID.String(),
		"organizationId": orgID.String(),
	}

	// Expect job status update to failed due to parse error
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, gomock.Any()).
		Return(nil)

	result, err := uc.parseMessage(ctx, invalidBody, headers, nil, logger)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}
}

// TestExtractJobIDFromMultipleSources_FromHeaders tests extracting IDs from headers.
func TestExtractJobIDFromMultipleSources_FromHeaders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	jobID := newTestJobID()
	orgID := newTestOrgID()

	headers := map[string]any{
		"jobId":          jobID.String(),
		"organizationId": orgID.String(),
	}

	resultJobID, resultOrgID := uc.extractJobIDFromMultipleSources(nil, headers, logger)

	if resultJobID != jobID {
		t.Fatalf("expected jobID %s, got %s", jobID, resultJobID)
	}

	if resultOrgID != orgID {
		t.Fatalf("expected orgID %s, got %s", orgID, resultOrgID)
	}
}

// TestExtractJobIDFromMultipleSources_FromPartialJSON tests extracting IDs from partial JSON.
func TestExtractJobIDFromMultipleSources_FromPartialJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	jobID := newTestJobID()
	orgID := newTestOrgID()

	// Partial JSON with valid jobId and organizationId
	body := []byte(`{"jobId": "` + jobID.String() + `", "organizationId": "` + orgID.String() + `", "invalid": }`)

	resultJobID, resultOrgID := uc.extractJobIDFromMultipleSources(body, nil, logger)

	if resultJobID != jobID {
		t.Fatalf("expected jobID %s, got %s", jobID, resultJobID)
	}

	if resultOrgID != orgID {
		t.Fatalf("expected orgID %s, got %s", orgID, resultOrgID)
	}
}

// TestExtractJobIDFromMultipleSources_NoIDs tests when no IDs can be extracted.
func TestExtractJobIDFromMultipleSources_NoIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	body := []byte(`{"invalid": "data"}`)

	resultJobID, resultOrgID := uc.extractJobIDFromMultipleSources(body, nil, logger)

	if resultJobID != uuid.Nil {
		t.Fatalf("expected nil jobID, got %s", resultJobID)
	}

	if resultOrgID != uuid.Nil {
		t.Fatalf("expected nil orgID, got %s", resultOrgID)
	}
}

// TestExtractConfigNamesFromMappedFields tests the extraction of config names.
func TestExtractConfigNamesFromMappedFields(t *testing.T) {
	tests := []struct {
		name         string
		mappedFields map[string]map[string][]string
		wantCount    int
		wantContains []string
	}{
		{
			name:         "empty map",
			mappedFields: map[string]map[string][]string{},
			wantCount:    0,
			wantContains: []string{},
		},
		{
			name:         "nil map",
			mappedFields: nil,
			wantCount:    0,
			wantContains: []string{},
		},
		{
			name: "single datasource",
			mappedFields: map[string]map[string][]string{
				"postgres_db": {"table1": {"field1"}},
			},
			wantCount:    1,
			wantContains: []string{"postgres_db"},
		},
		{
			name: "multiple datasources",
			mappedFields: map[string]map[string][]string{
				"postgres_db": {"table1": {"field1"}},
				"mysql_db":    {"table2": {"field2"}},
				"plugin_crm":  {"collection1": {"field3"}},
			},
			wantCount:    3,
			wantContains: []string{"postgres_db", "mysql_db", "plugin_crm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractConfigNamesFromMappedFields(tt.mappedFields)

			if len(result) != tt.wantCount {
				t.Fatalf("expected %d config names, got %d", tt.wantCount, len(result))
			}

			for _, want := range tt.wantContains {
				found := false
				for _, got := range result {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected config name %s not found in result %v", want, result)
				}
			}
		})
	}
}

// TestShouldSkipProcessing tests the idempotency check.
func TestShouldSkipProcessing(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*testMocks, uuid.UUID, uuid.UUID)
		wantSkip   bool
	}{
		{
			name: "job already completed - should skip",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusCompleted,
					}, nil)
			},
			wantSkip: true,
		},
		{
			name: "job pending - should not skip",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusPending,
					}, nil)
			},
			wantSkip: false,
		},
		{
			name: "job not found - should not skip",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(nil, nil)
			},
			wantSkip: false,
		},
		{
			name: "repository error - should not skip",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(nil, errors.New("database error"))
			},
			wantSkip: false,
		},
		{
			name: "job already failed - should not skip (allows retry)",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusFailed,
					}, nil)
			},
			wantSkip: false,
		},
		{
			name: "job currently processing - should not skip",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusProcessing,
					}, nil)
			},
			wantSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := newTestMocks(ctrl)
			uc := newTestUseCase(mocks)

			ctx := testContext()
			jobID := newTestJobID()
			orgID := newTestOrgID()

			tt.setupMocks(mocks, jobID, orgID)

			logger := testLogger()
			got := uc.shouldSkipProcessing(ctx, jobID, orgID, logger)
			if got != tt.wantSkip {
				t.Fatalf("expected skip=%v, got skip=%v", tt.wantSkip, got)
			}
		})
	}
}

// TestUpdateJobWithErrors tests job status update on error.
func TestUpdateJobWithErrors(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		updateErr    error
		wantErr      bool
	}{
		{
			name:         "successful update",
			errorMessage: "Test error message",
			updateErr:    nil,
			wantErr:      false,
		},
		{
			name:         "update fails",
			errorMessage: "Test error message",
			updateErr:    errors.New("database connection failed"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := newTestMocks(ctrl)
			uc := newTestUseCase(mocks)

			ctx := testContext()
			jobID := newTestJobID()
			orgID := newTestOrgID()

			mocks.jobRepo.EXPECT().
				UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, gomock.Any()).
				DoAndReturn(func(_ interface{}, _, _ uuid.UUID, _ model.JobStatus, metadata map[string]any) error {
					if metadata["error"] != tt.errorMessage {
						t.Errorf("expected error message %q in metadata, got %q", tt.errorMessage, metadata["error"])
					}
					return tt.updateErr
				})

			err := uc.updateJobWithErrors(ctx, jobID, orgID, tt.errorMessage)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

// TestGetTableFilters tests filter extraction for specific tables.
func TestGetTableFilters(t *testing.T) {
	tests := []struct {
		name       string
		dbFilters  map[string]map[string]modelJob.FilterCondition
		tableName  string
		wantNil    bool
		wantFields []string
	}{
		{
			name:      "nil filters",
			dbFilters: nil,
			tableName: "table1",
			wantNil:   true,
		},
		{
			name:      "empty filters",
			dbFilters: map[string]map[string]modelJob.FilterCondition{},
			tableName: "table1",
			wantNil:   true,
		},
		{
			name: "table not in filters",
			dbFilters: map[string]map[string]modelJob.FilterCondition{
				"other_table": {"field1": {Equals: []any{"value"}}},
			},
			tableName: "table1",
			wantNil:   true,
		},
		{
			name: "table found in filters",
			dbFilters: map[string]map[string]modelJob.FilterCondition{
				"table1": {
					"field1": {Equals: []any{"value1"}},
					"field2": {GreaterThan: []any{100}},
				},
			},
			tableName:  "table1",
			wantNil:    false,
			wantFields: []string{"field1", "field2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTableFilters(tt.dbFilters, tt.tableName)

			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			for _, field := range tt.wantFields {
				if _, ok := result[field]; !ok {
					t.Fatalf("expected field %s in result", field)
				}
			}
		})
	}
}
