package services

import (
	"context"
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
		UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, gomock.Any(), gomock.Any()).
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
				UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _ uuid.UUID, _ model.JobStatus, _ string, metadata map[string]any) error {
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

// TestExtractJobIDFromPartialJSON tests regex-based extraction from malformed JSON.
func TestExtractJobIDFromPartialJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        []byte
		wantJobID   bool
		wantOrgID   bool
		description string
	}{
		{
			name:        "valid partial JSON with both IDs",
			body:        []byte(`{"jobId": "550e8400-e29b-41d4-a716-446655440000", "organizationId": "650e8400-e29b-41d4-a716-446655440000", "invalid": }`),
			wantJobID:   true,
			wantOrgID:   true,
			description: "should extract both IDs from partial JSON",
		},
		{
			name:        "regex extraction with spaces",
			body:        []byte(`{ "jobId" : "550e8400-e29b-41d4-a716-446655440000" , "organizationId" : "650e8400-e29b-41d4-a716-446655440000" }`),
			wantJobID:   true,
			wantOrgID:   true,
			description: "should handle extra whitespace",
		},
		{
			name:        "only jobId present",
			body:        []byte(`{"jobId": "550e8400-e29b-41d4-a716-446655440000"}`),
			wantJobID:   true,
			wantOrgID:   false,
			description: "should extract only jobId",
		},
		{
			name:        "invalid UUID format",
			body:        []byte(`{"jobId": "not-a-uuid", "organizationId": "also-not-uuid"}`),
			wantJobID:   false,
			wantOrgID:   false,
			description: "should fail on invalid UUID format",
		},
		{
			name:        "empty body",
			body:        []byte(``),
			wantJobID:   false,
			wantOrgID:   false,
			description: "should return nil UUIDs for empty body",
		},
		{
			name:        "completely invalid JSON",
			body:        []byte(`not json at all`),
			wantJobID:   false,
			wantOrgID:   false,
			description: "should return nil UUIDs for non-JSON",
		},
		{
			name:        "jobId in middle of text",
			body:        []byte(`some text before "jobId": "550e8400-e29b-41d4-a716-446655440000" some text after`),
			wantJobID:   true,
			wantOrgID:   false,
			description: "should extract jobId from middle of text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := newTestMocks(ctrl)
			uc := newTestUseCase(mocks)
			logger := testLogger()

			jobID, orgID := uc.extractJobIDFromPartialJSON(tt.body, logger)

			if tt.wantJobID && jobID == uuid.Nil {
				t.Errorf("expected non-nil jobID, got nil - %s", tt.description)
			}
			if !tt.wantJobID && jobID != uuid.Nil {
				t.Errorf("expected nil jobID, got %s - %s", jobID, tt.description)
			}

			if tt.wantOrgID && orgID == uuid.Nil {
				t.Errorf("expected non-nil orgID, got nil - %s", tt.description)
			}
			if !tt.wantOrgID && orgID != uuid.Nil {
				t.Errorf("expected nil orgID, got %s - %s", orgID, tt.description)
			}
		})
	}
}

// TestExtractJobIDFromMultipleSources_EdgeCases tests edge cases for ID extraction.
func TestExtractJobIDFromMultipleSources_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		body      []byte
		headers   map[string]any
		wantJobID bool
		wantOrgID bool
	}{
		{
			name:      "nil body and nil headers",
			body:      nil,
			headers:   nil,
			wantJobID: false,
			wantOrgID: false,
		},
		{
			name:      "empty body and empty headers",
			body:      []byte{},
			headers:   map[string]any{},
			wantJobID: false,
			wantOrgID: false,
		},
		{
			name: "header with non-string jobId",
			body: nil,
			headers: map[string]any{
				"jobId": 12345, // not a string
			},
			wantJobID: false,
			wantOrgID: false,
		},
		{
			name: "header with invalid UUID string",
			body: nil,
			headers: map[string]any{
				"jobId": "not-a-valid-uuid",
			},
			wantJobID: false,
			wantOrgID: false,
		},
		{
			name: "header with jobId only, no orgId",
			body: nil,
			headers: map[string]any{
				"jobId": "550e8400-e29b-41d4-a716-446655440000",
			},
			wantJobID: true,
			wantOrgID: false,
		},
		{
			name: "header with orgId as non-string",
			body: nil,
			headers: map[string]any{
				"jobId":          "550e8400-e29b-41d4-a716-446655440000",
				"organizationId": 123, // not a string
			},
			wantJobID: true,
			wantOrgID: false,
		},
		{
			name: "fallback to body when header invalid",
			body: []byte(`{"jobId": "550e8400-e29b-41d4-a716-446655440000"}`),
			headers: map[string]any{
				"jobId": "invalid-uuid",
			},
			wantJobID: true,
			wantOrgID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mocks := newTestMocks(ctrl)
			uc := newTestUseCase(mocks)
			logger := testLogger()

			jobID, orgID := uc.extractJobIDFromMultipleSources(tt.body, tt.headers, logger)

			if tt.wantJobID && jobID == uuid.Nil {
				t.Errorf("expected non-nil jobID, got nil")
			}
			if !tt.wantJobID && jobID != uuid.Nil {
				t.Errorf("expected nil jobID, got %s", jobID)
			}

			if tt.wantOrgID && orgID == uuid.Nil {
				t.Errorf("expected non-nil orgID, got nil")
			}
			if !tt.wantOrgID && orgID != uuid.Nil {
				t.Errorf("expected nil orgID, got %s", orgID)
			}
		})
	}
}

// TestCheckReportStatus tests the job status checking function.
func TestCheckReportStatus(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*testMocks, uuid.UUID, uuid.UUID)
		wantStatus model.JobStatus
		wantErr    bool
	}{
		{
			name: "job found with completed status",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusCompleted,
					}, nil)
			},
			wantStatus: model.JobStatusCompleted,
			wantErr:    false,
		},
		{
			name: "job found with processing status",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusProcessing,
					}, nil)
			},
			wantStatus: model.JobStatusProcessing,
			wantErr:    false,
		},
		{
			name: "job not found - returns nil",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(nil, nil)
			},
			wantStatus: "",
			wantErr:    true,
		},
		{
			name: "repository error",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID, orgID).
					Return(nil, errors.New("database connection failed"))
			},
			wantStatus: "",
			wantErr:    true,
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
			logger := testLogger()

			tt.setupMocks(mocks, jobID, orgID)

			status, err := uc.checkReportStatus(ctx, jobID, orgID, logger)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if status != tt.wantStatus {
				t.Fatalf("expected status %s, got %s", tt.wantStatus, status)
			}
		})
	}
}

// TestCountTotalRows tests the row counting function.
func TestCountTotalRows(t *testing.T) {
	tests := []struct {
		name      string
		result    map[string]map[string][]map[string]any
		wantCount int64
	}{
		{
			name:      "empty result",
			result:    map[string]map[string][]map[string]any{},
			wantCount: 0,
		},
		{
			name:      "nil result",
			result:    nil,
			wantCount: 0,
		},
		{
			name: "single database, single table, single row",
			result: map[string]map[string][]map[string]any{
				"db1": {
					"table1": {{"id": 1, "name": "test"}},
				},
			},
			wantCount: 1,
		},
		{
			name: "single database, single table, multiple rows",
			result: map[string]map[string][]map[string]any{
				"db1": {
					"table1": {
						{"id": 1, "name": "test1"},
						{"id": 2, "name": "test2"},
						{"id": 3, "name": "test3"},
					},
				},
			},
			wantCount: 3,
		},
		{
			name: "single database, multiple tables",
			result: map[string]map[string][]map[string]any{
				"db1": {
					"table1": {
						{"id": 1},
						{"id": 2},
					},
					"table2": {
						{"id": 3},
						{"id": 4},
						{"id": 5},
					},
				},
			},
			wantCount: 5,
		},
		{
			name: "multiple databases, multiple tables",
			result: map[string]map[string][]map[string]any{
				"db1": {
					"table1": {
						{"id": 1},
						{"id": 2},
					},
					"table2": {
						{"id": 3},
					},
				},
				"db2": {
					"table3": {
						{"id": 4},
						{"id": 5},
						{"id": 6},
					},
				},
			},
			wantCount: 6,
		},
		{
			name: "database with empty table",
			result: map[string]map[string][]map[string]any{
				"db1": {
					"table1": {},
					"table2": {
						{"id": 1},
					},
				},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countTotalRows(tt.result)
			if count != tt.wantCount {
				t.Fatalf("expected count %d, got %d", tt.wantCount, count)
			}
		})
	}
}

// TestHandleErrorWithUpdate tests the error handling with job update.
func TestHandleErrorWithUpdate(t *testing.T) {
	tests := []struct {
		name         string
		setupMocks   func(*testMocks, uuid.UUID, uuid.UUID)
		testErr      error
		wantErr      bool
		wantUpdateOk bool
	}{
		{
			name: "successful update",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, gomock.Any(), gomock.Any()).
					Return(nil)
				// handleErrorWithUpdate also publishes a notification
				mocks.rabbitPublisher.EXPECT().
					Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			testErr:      errors.New("test error"),
			wantErr:      true,
			wantUpdateOk: true,
		},
		{
			name: "update fails",
			setupMocks: func(mocks *testMocks, jobID, orgID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			testErr:      errors.New("test error"),
			wantErr:      true,
			wantUpdateOk: false,
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
			logger := testLogger()

			tt.setupMocks(mocks, jobID, orgID)

			message := ExtractExternalDataMessage{
				JobID:          jobID,
				OrganizationID: orgID,
				Metadata:       map[string]any{"source": "test"},
			}

			err := uc.handleErrorWithUpdate(ctx, jobID, orgID, message, nil, "test error message", tt.testErr, logger)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

// TestParseMessage_WithNilError tests parse message when error is nil.
func TestParseMessage_WithNilError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()
	orgID := newTestOrgID()

	// Valid message should not call UpdateStatus
	validMessage := ExtractExternalDataMessage{
		JobID:          jobID,
		OrganizationID: orgID,
		MappedFields: map[string]map[string][]string{
			"datasource1": {"table1": {"field1"}},
		},
	}

	body, _ := json.Marshal(validMessage)
	result, err := uc.parseMessage(ctx, body, nil, nil, logger)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
}

// TestExtractJobIDFromPartialJSON_RegexFallback tests regex fallback path.
func TestExtractJobIDFromPartialJSON_RegexFallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Malformed JSON that will fail decoder but has valid UUID in regex pattern
	body := []byte(`{bad json but "jobId":"550e8400-e29b-41d4-a716-446655440000"}`)

	jobID, orgID := uc.extractJobIDFromPartialJSON(body, logger)

	if jobID == uuid.Nil {
		t.Error("expected non-nil jobID from regex extraction")
	}
	if orgID != uuid.Nil {
		t.Error("expected nil orgID when not in body")
	}
}

// TestParseMessage_UpdateStatusError tests when UpdateStatus fails during parse error.
func TestParseMessage_UpdateStatusError(t *testing.T) {
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

	// UpdateStatus fails
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, orgID, model.JobStatusFailed, gomock.Any(), gomock.Any()).
		Return(errors.New("update failed"))

	result, err := uc.parseMessage(ctx, invalidBody, headers, nil, logger)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}
}

// TestExtractConfigNamesFromMappedFields_WithComplexStructure tests complex mapped fields.
func TestExtractConfigNamesFromMappedFields_WithComplexStructure(t *testing.T) {
	mappedFields := map[string]map[string][]string{
		"postgres_prod": {
			"users":    {"id", "email", "name"},
			"orders":   {"order_id", "user_id", "total"},
			"products": {"product_id", "name", "price"},
		},
		"mysql_analytics": {
			"events": {"event_id", "timestamp", "user_id"},
		},
		"mongodb_logs": {
			"application_logs": {"level", "message", "timestamp"},
		},
	}

	result := extractConfigNamesFromMappedFields(mappedFields)

	if len(result) != 3 {
		t.Fatalf("expected 3 config names, got %d", len(result))
	}

	expectedConfigs := map[string]bool{
		"postgres_prod":   false,
		"mysql_analytics": false,
		"mongodb_logs":    false,
	}

	for _, config := range result {
		if _, exists := expectedConfigs[config]; exists {
			expectedConfigs[config] = true
		}
	}

	for config, found := range expectedConfigs {
		if !found {
			t.Errorf("expected config %s not found in result", config)
		}
	}
}
