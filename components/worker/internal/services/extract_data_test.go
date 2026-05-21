package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// TestParseMessage_ValidMessage tests successful message parsing.
func TestParseMessage_ValidMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
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

func TestParseMessage_JSONNull(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	result, err := uc.parseMessage(ctx, []byte(`null`), nil, nil, logger)
	if err == nil {
		t.Fatal("expected error for null JSON payload, got nil")
	}

	if result != nil {
		t.Fatalf("expected nil result for null JSON payload, got %+v", result)
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

	invalidBody := []byte(`{"invalid": json`)
	headers := map[string]any{
		"jobId": jobID.String(),
	}

	// Expect job status update to failed due to parse error
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, "", "", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error {
			if resultPath != "" {
				t.Errorf("expected empty resultPath, got %q", resultPath)
			}
			if resultHMAC != "" {
				t.Errorf("expected empty resultHMAC, got %q", resultHMAC)
			}

			errValue, ok := metadata["error"].(string)
			if !ok {
				t.Fatalf("expected metadata.error as string, got %T", metadata["error"])
			}
			if !strings.Contains(errValue, "Failed to parse message") {
				t.Fatalf("expected metadata.error to contain parse message details, got %q", errValue)
			}

			return nil
		})

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

	headers := map[string]any{
		"jobId": jobID.String(),
	}

	resultJobID := uc.extractJobIDFromMultipleSources(nil, headers, logger)

	if resultJobID != jobID {
		t.Fatalf("expected jobID %s, got %s", jobID, resultJobID)
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

	// Partial JSON with valid jobId and organizationId
	body := []byte(`{"jobId": "` + jobID.String() + `", "organizationId": "` + uuid.New().String() + `", "invalid": }`)

	resultJobID := uc.extractJobIDFromMultipleSources(body, nil, logger)

	if resultJobID != jobID {
		t.Fatalf("expected jobID %s, got %s", jobID, resultJobID)
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

	resultJobID := uc.extractJobIDFromMultipleSources(body, nil, logger)

	if resultJobID != uuid.Nil {
		t.Fatalf("expected nil jobID, got %s", resultJobID)
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
		setupMocks func(*testMocks, uuid.UUID)
		wantSkip   bool
	}{
		{
			name: "job already completed - should skip",
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusCompleted,
					}, nil)
			},
			wantSkip: true,
		},
		{
			name: "job pending - should not skip",
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusPending,
					}, nil)
			},
			wantSkip: false,
		},
		{
			name: "job not found - should not skip",
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(nil, nil)
			},
			wantSkip: false,
		},
		{
			name: "repository error - should not skip",
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(nil, errors.New("database error"))
			},
			wantSkip: false,
		},
		{
			name: "job already failed - should not skip (allows retry)",
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusFailed,
					}, nil)
			},
			wantSkip: false,
		},
		{
			name: "job currently processing - should skip",
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(&model.Job{
						ID:     jobID,
						Status: model.JobStatusProcessing,
					}, nil)
			},
			wantSkip: true,
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

			tt.setupMocks(mocks, jobID)

			logger := testLogger()
			got := uc.shouldSkipProcessing(ctx, jobID, logger)
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

			mocks.jobRepo.EXPECT().
				UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ uuid.UUID, _ model.JobStatus, _, _ string, metadata map[string]any) error {
					if metadata["error"] != tt.errorMessage {
						t.Errorf("expected error message %q in metadata, got %q", tt.errorMessage, metadata["error"])
					}
					return tt.updateErr
				})

			err := uc.updateJobWithErrors(ctx, jobID, tt.errorMessage)

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

			jobID := uc.extractJobIDFromPartialJSON(tt.body, logger)

			if tt.wantJobID && jobID == uuid.Nil {
				t.Errorf("expected non-nil jobID, got nil - %s", tt.description)
			}
			if !tt.wantJobID && jobID != uuid.Nil {
				t.Errorf("expected nil jobID, got %s - %s", jobID, tt.description)
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

			jobID := uc.extractJobIDFromMultipleSources(tt.body, tt.headers, logger)

			if tt.wantJobID && jobID == uuid.Nil {
				t.Errorf("expected non-nil jobID, got nil")
			}
			if !tt.wantJobID && jobID != uuid.Nil {
				t.Errorf("expected nil jobID, got %s", jobID)
			}

		})
	}
}

// TestCheckReportStatus tests the job status checking function.
func TestCheckReportStatus(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*testMocks, uuid.UUID)
		wantStatus model.JobStatus
		wantErr    bool
	}{
		{
			name: "job found with completed status",
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
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
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
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
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(nil, nil)
			},
			wantStatus: "",
			wantErr:    true,
		},
		{
			name: "repository error",
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
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
			logger := testLogger()

			tt.setupMocks(mocks, jobID)

			status, err := uc.checkReportStatus(ctx, jobID, logger)

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
		setupMocks   func(*testMocks, uuid.UUID)
		testErr      error
		wantErr      bool
		wantUpdateOk bool
	}{
		{
			name: "successful update",
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
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
			setupMocks: func(mocks *testMocks, jobID uuid.UUID) {
				mocks.jobRepo.EXPECT().
					UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
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
			logger := testLogger()

			tt.setupMocks(mocks, jobID)

			message := ExtractExternalDataMessage{
				JobID:    jobID,
				Metadata: map[string]any{"source": "test"},
			}

			err := uc.handleErrorWithUpdate(ctx, jobID, message, nil, "test error message", tt.testErr, logger)

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

	// Valid message should not call UpdateStatus
	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"datasource1": {"table1": {"field1"}},
		},
	}

	body, marshalErr := json.Marshal(validMessage)
	if marshalErr != nil {
		t.Fatalf("failed to marshal test message: %v", marshalErr)
	}

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

	jobID := uc.extractJobIDFromPartialJSON(body, logger)

	if jobID == uuid.Nil {
		t.Error("expected non-nil jobID from regex extraction")
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

	invalidBody := []byte(`{"invalid": json`)
	headers := map[string]any{
		"jobId": jobID.String(),
	}

	// UpdateStatus fails
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
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

// TestEncryptData tests the encryption function for storage.
func TestEncryptData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	logger := testLogger()

	t.Run("missing derived key returns error", func(t *testing.T) {
		uc := newTestUseCase(mocks)
		// storageEncryptDerivedKey is nil by default

		_, err := uc.encryptData([]byte(`{"test": "data"}`), logger)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("encrypts data successfully with valid key", func(t *testing.T) {
		uc := newTestUseCase(mocks)
		uc.SetStorageEncryptDerivedKey([]byte("01234567890123456789012345678901")) // 32 bytes

		result, err := uc.encryptData([]byte(`{"test": "data"}`), logger)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})
}

// TestQueryExternalData_EmptyMappedFields tests queryExternalData with empty mapped fields.
func TestQueryExternalData_EmptyMappedFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()

	message := ExtractExternalDataMessage{
		JobID:        newTestJobID(),
		MappedFields: map[string]map[string][]string{}, // empty
	}

	result := make(map[string]map[string][]map[string]any)

	err := uc.queryExternalData(ctx, message, nil, result)
	if err != nil {
		t.Fatalf("expected no error for empty mapped fields, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

// TestExtractExternalData_JobRepositoryFindError tests error handling when job lookup fails.
func TestExtractExternalData_JobRepositoryFindError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"datasource1": {"table1": {"field1"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	// First check for shouldSkipProcessing - returns error
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(nil, errors.New("database error"))

	// Second check for job validation - also returns error
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(nil, errors.New("database error"))

	// Expect job status to be updated to failed
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, "", "", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error {
			if resultPath != "" {
				t.Errorf("expected empty resultPath, got %q", resultPath)
			}
			if resultHMAC != "" {
				t.Errorf("expected empty resultHMAC, got %q", resultHMAC)
			}

			errValue, ok := metadata["error"].(string)
			if !ok {
				t.Fatalf("expected metadata.error as string, got %T", metadata["error"])
			}
			if !strings.Contains(errValue, "database error") {
				t.Fatalf("expected metadata.error to contain repository failure details, got %q", errValue)
			}

			return nil
		})

	// Expect failure notification
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err = uc.ExtractExternalData(ctx, body, nil)
	if err == nil {
		t.Fatal("expected error when job repository fails, got nil")
	}
}

// TestExtractExternalData_ParseErrorNoJobID tests parse error when jobID cannot be extracted.
func TestExtractExternalData_ParseErrorNoJobID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()

	// Completely invalid body with no extractable jobID
	invalidBody := []byte(`not json at all`)

	// No mocks expected because jobID cannot be extracted for status update

	err := uc.ExtractExternalData(ctx, invalidBody, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestCheckReportStatus_JobDataNil tests checkReportStatus when job data is nil.
func TestCheckReportStatus_JobDataNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()
	logger := testLogger()

	// Repository returns nil job (not found)
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(nil, nil)

	status, err := uc.checkReportStatus(ctx, jobID, logger)

	if err == nil {
		t.Fatal("expected error when job data is nil")
	}

	if status != "" {
		t.Errorf("expected empty status, got %s", status)
	}
}

// TestExtractJobIDFromPartialJSON_ValidJobIDInvalidOrgID tests partial JSON extraction.
func TestExtractJobIDFromPartialJSON_ValidJobIDInvalidOrgID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Valid jobId with invalid organizationId format
	body := []byte(`{"jobId": "550e8400-e29b-41d4-a716-446655440000", "organizationId": "not-a-uuid"}`)

	jobID := uc.extractJobIDFromPartialJSON(body, logger)

	if jobID == uuid.Nil {
		t.Error("expected valid jobID")
	}

}

// TestParseMessage_EmptyBody tests parseMessage with empty body.
func TestParseMessage_EmptyBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	emptyBody := []byte{}

	result, err := uc.parseMessage(ctx, emptyBody, nil, nil, logger)
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}

	if result != nil {
		t.Fatalf("expected nil result for empty body, got %+v", result)
	}
}

// TestParseMessage_NullPayload tests parseMessage with JSON null payload.
func TestParseMessage_NullPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	headers := map[string]any{
		"jobId": jobID.String(),
	}

	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, "", "", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error {
			if resultPath != "" {
				t.Errorf("expected empty resultPath, got %q", resultPath)
			}
			if resultHMAC != "" {
				t.Errorf("expected empty resultHMAC, got %q", resultHMAC)
			}

			errValue, ok := metadata["error"].(string)
			if !ok {
				t.Fatalf("expected metadata.error as string, got %T", metadata["error"])
			}
			if !strings.Contains(errValue, "empty message payload") {
				t.Fatalf("expected metadata.error to contain empty payload message, got %q", errValue)
			}

			return nil
		})

	result, err := uc.parseMessage(ctx, []byte(`null`), headers, nil, logger)
	if err == nil {
		t.Fatal("expected error for null payload, got nil")
	}

	if !strings.Contains(err.Error(), "empty message payload") {
		t.Fatalf("expected empty payload error, got: %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil result for null payload, got %+v", result)
	}
}

// TestParseMessage_MissingMappedFields tests parseMessage with missing mappedFields.
func TestParseMessage_MissingMappedFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	body := []byte(`{"jobId":"` + jobID.String() + `","organizationId":"` + uuid.New().String() + `"}`)

	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, "", "", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error {
			if resultPath != "" {
				t.Errorf("expected empty resultPath, got %q", resultPath)
			}
			if resultHMAC != "" {
				t.Errorf("expected empty resultHMAC, got %q", resultHMAC)
			}

			errValue, ok := metadata["error"].(string)
			if !ok {
				t.Fatalf("expected metadata.error as string, got %T", metadata["error"])
			}
			if !strings.Contains(errValue, "mappedFields is required") {
				t.Fatalf("expected metadata.error to contain mappedFields message, got %q", errValue)
			}

			return nil
		})

	result, err := uc.parseMessage(ctx, body, nil, nil, logger)
	if err == nil {
		t.Fatal("expected error for missing mappedFields, got nil")
	}

	if !strings.Contains(err.Error(), "mappedFields is required") {
		t.Fatalf("expected mappedFields validation error, got: %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil result for invalid payload, got %+v", result)
	}
}

// TestParseMessage_MissingJobID tests parseMessage with missing jobId.
func TestParseMessage_MissingJobID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	body := []byte(`{"organizationId":"` + uuid.New().String() + `","mappedFields":{"datasource1":{"table1":["field1"]}}}`)
	headers := map[string]any{
		"jobId": jobID.String(),
	}

	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, "", "", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error {
			if resultPath != "" {
				t.Errorf("expected empty resultPath, got %q", resultPath)
			}
			if resultHMAC != "" {
				t.Errorf("expected empty resultHMAC, got %q", resultHMAC)
			}

			errValue, ok := metadata["error"].(string)
			if !ok {
				t.Fatalf("expected metadata.error as string, got %T", metadata["error"])
			}
			if !strings.Contains(errValue, "jobId is required") {
				t.Fatalf("expected metadata.error to contain jobId message, got %q", errValue)
			}

			return nil
		})

	result, err := uc.parseMessage(ctx, body, headers, nil, logger)
	if err == nil {
		t.Fatal("expected error for missing jobId, got nil")
	}

	if !strings.Contains(err.Error(), "jobId is required") {
		t.Fatalf("expected jobId validation error, got: %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil result for invalid payload, got %+v", result)
	}
}

// TestExtractJobIDFromMultipleSources_HeaderPrecedence tests that headers take precedence over body.
func TestExtractJobIDFromMultipleSources_HeaderPrecedence(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	headerJobID := uuid.New()
	bodyJobID := uuid.New()

	// Headers and body have different jobIDs - headers should win
	body := []byte(`{"jobId": "` + bodyJobID.String() + `"}`)
	headers := map[string]any{
		"jobId": headerJobID.String(),
	}

	resultJobID := uc.extractJobIDFromMultipleSources(body, headers, logger)

	if resultJobID != headerJobID {
		t.Errorf("expected header jobID %s to take precedence, got %s", headerJobID, resultJobID)
	}
}

// TestCountTotalRows_LargeDataset tests row counting with a large dataset.
func TestCountTotalRows_LargeDataset(t *testing.T) {
	// Create a large result set
	result := make(map[string]map[string][]map[string]any)

	// Add multiple databases with multiple tables
	for dbIdx := 0; dbIdx < 5; dbIdx++ {
		dbName := "db_" + string(rune('A'+dbIdx))
		result[dbName] = make(map[string][]map[string]any)

		for tableIdx := 0; tableIdx < 10; tableIdx++ {
			tableName := "table_" + string(rune('0'+tableIdx))
			rows := make([]map[string]any, 100) // 100 rows per table
			for i := range rows {
				rows[i] = map[string]any{"id": i, "data": "test"}
			}
			result[dbName][tableName] = rows
		}
	}

	// Expected: 5 databases * 10 tables * 100 rows = 5000 rows
	expectedCount := int64(5 * 10 * 100)

	count := countTotalRows(result)
	if count != expectedCount {
		t.Fatalf("expected count %d, got %d", expectedCount, count)
	}
}

// TestUpdateJobWithErrors_EmptyErrorMessage tests updating job with empty error message.
func TestUpdateJobWithErrors_EmptyErrorMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ model.JobStatus, _, _ string, metadata map[string]any) error {
			if metadata["error"] != "" {
				t.Errorf("expected empty error message in metadata, got %q", metadata["error"])
			}
			return nil
		})

	err := uc.updateJobWithErrors(ctx, jobID, "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestQueryDatabase_ConnectionNotFound tests queryDatabase when connection is not found.
func TestQueryDatabase_ConnectionNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	// Empty connections list - connection not found for database
	connections := []*model.Connection{}

	tables := map[string][]string{
		"users": {"id", "name", "email"},
	}

	result := make(map[string]map[string][]map[string]any)

	err := uc.queryDatabase(
		ctx,
		"postgres_db",
		tables,
		connections,
		nil,
		result,
		logger,
		testTracer(),
	)

	if err == nil {
		t.Fatal("expected error when connection not found")
	}

	if !strings.Contains(err.Error(), "connection not found for database: postgres_db") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestQueryDatabase_ConnectionFoundButDifferentConfigName tests queryDatabase with multiple connections.
func TestQueryDatabase_ConnectionFoundButDifferentConfigName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	// Connections with different config names
	connections := []*model.Connection{
		{
			ConfigName: "mysql_db",
			Type:       model.TypeMySQL,
		},
		{
			ConfigName: "oracle_db",
			Type:       model.TypeOracle,
		},
	}

	tables := map[string][]string{
		"users": {"id", "name"},
	}

	result := make(map[string]map[string][]map[string]any)

	err := uc.queryDatabase(
		ctx,
		"postgres_db", // This config name doesn't exist in connections
		tables,
		connections,
		nil,
		result,
		logger,
		testTracer(),
	)

	if err == nil {
		t.Fatal("expected error when connection not found")
	}

	if !strings.Contains(err.Error(), "connection not found for database: postgres_db") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestSaveExternalData_MarshalError tests saveExternalData with data that can't be marshaled.
func TestSaveExternalData_MarshalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test"},
	}

	// Create result with value that can't be marshaled to JSON
	// Using a channel which cannot be serialized to JSON
	result := map[string]map[string][]map[string]any{
		"db1": {
			"table1": {
				{"channel": make(chan int)}, // This will cause marshal error
			},
		},
	}

	_, err := uc.saveExternalData(ctx, testTracer(), message, result, nil, logger)
	if err == nil {
		t.Fatal("expected error when marshaling fails")
	}
}

// TestSaveExternalData_MissingEnvVars tests saveExternalData with missing environment variables.
func TestSaveExternalData_MissingEnvVars(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test"},
	}

	result := map[string]map[string][]map[string]any{
		"db1": {
			"table1": {
				{"id": 1, "name": "test"},
			},
		},
	}

	// Ensure environment variables are not set
	// Note: This will fail because CRYPTO_ENCRYPT_FILE_STORAGE is not set

	_, err := uc.saveExternalData(ctx, testTracer(), message, result, nil, logger)
	if err == nil {
		t.Fatal("expected error when env vars are missing")
	}
}

// TestSaveExternalData_StoragePutError tests saveExternalData when storage put fails.
func TestSaveExternalData_StoragePutError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	// Set valid encryption keys on the UseCase
	uc.SetStorageEncryptDerivedKey([]byte("01234567890123456789012345678901"))

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test"},
	}

	result := map[string]map[string][]map[string]any{
		"db1": {
			"table1": {
				{"id": 1, "name": "test"},
			},
		},
	}

	// Mock storage to return error
	expectedObjectName := constant.ExternalDataKeyPrefix + "/" + jobID.String() + ".json"
	mocks.seaweedFS.EXPECT().
		Put(gomock.Any(), expectedObjectName, gomock.Any()).
		Return(errors.New("storage connection failed"))

	_, err := uc.saveExternalData(ctx, testTracer(), message, result, nil, logger)
	if err == nil {
		t.Fatal("expected error when storage put fails")
	}
}

// TestSaveExternalData_Success tests saveExternalData happy path.
func TestSaveExternalData_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	// Set valid encryption keys on the UseCase
	uc.SetStorageEncryptDerivedKey([]byte("01234567890123456789012345678901"))

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test"},
	}

	result := map[string]map[string][]map[string]any{
		"db1": {
			"table1": {
				{"id": 1, "name": "test1"},
				{"id": 2, "name": "test2"},
			},
			"table2": {
				{"id": 3, "data": "value"},
			},
		},
	}

	// Mock storage to succeed
	expectedObjectName := constant.ExternalDataKeyPrefix + "/" + jobID.String() + ".json"
	mocks.seaweedFS.EXPECT().
		Put(gomock.Any(), expectedObjectName, gomock.Any()).
		Return(nil)

	resultData, err := uc.saveExternalData(ctx, testTracer(), message, result, nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resultData == nil {
		t.Fatal("expected result data, got nil")
	}

	// Verify result data
	expectedPath := expectedObjectName
	if resultData.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, resultData.Path)
	}

	// Should have 3 rows total (2 from table1, 1 from table2)
	if resultData.RowCount != 3 {
		t.Errorf("expected row count 3, got %d", resultData.RowCount)
	}

	if resultData.Format != "json" {
		t.Errorf("expected format 'json', got %s", resultData.Format)
	}

	if resultData.SizeBytes <= 0 {
		t.Errorf("expected positive size, got %d", resultData.SizeBytes)
	}
}

// TestExtractExternalData_JobAlreadyCompleted tests that completed jobs are skipped.
func TestExtractExternalData_JobAlreadyCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"datasource1": {"table1": {"field1"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	// Job is already completed - should skip processing
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{
			ID:     jobID,
			Status: model.JobStatusCompleted,
		}, nil)

	err = uc.ExtractExternalData(ctx, body, nil)
	if err != nil {
		t.Fatalf("expected no error for completed job (should skip), got: %v", err)
	}
}

// TestExtractExternalData_ConnectionRepositoryError tests error handling when connection lookup fails.
func TestExtractExternalData_ConnectionRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"users": {"id", "name"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	// First call for shouldSkipProcessing - job is pending
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{
			ID:     jobID,
			Status: model.JobStatusPending,
		}, nil)

	// Second call for job validation - job exists
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{
			ID:     jobID,
			Status: model.JobStatusPending,
		}, nil)

	// Connection repository returns error
	mocks.connRepo.EXPECT().
		FindByConfigNames(gomock.Any(), []string{"postgres_db"}).
		Return(nil, errors.New("database connection failed"))

	// Expect job transition to processing before extraction
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).
		Return(nil)

	// Expect job status to be updated to failed
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	// Expect failure notification
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err = uc.ExtractExternalData(ctx, body, nil)
	if err == nil {
		t.Fatal("expected error when connection repository fails, got nil")
	}
}

// TestExtractExternalData_ProcessingStatusUpdateError tests failure when transitioning to processing.
func TestExtractExternalData_ProcessingStatusUpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"postgres_db": {"users": {"id", "name"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{ID: jobID, Status: model.JobStatusPending}, nil)

	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{ID: jobID, Status: model.JobStatusPending}, nil)

	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusProcessing, "", "", nil).
		Return(errors.New("processing update failed"))

	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err = uc.ExtractExternalData(ctx, body, nil)
	if err == nil {
		t.Fatal("expected error when processing status update fails, got nil")
	}
}

// TestCompleteJob_CompletedStatusUpdateError tests completed status persistence failure path.
func TestCompleteJob_CompletedStatusUpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	tracer := testTracer()
	ctx, span := tracer.Start(ctx, "test.complete_job")
	defer span.End()

	jobID := newTestJobID()
	logger := testLogger()

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test-service"},
	}

	resultData := &JobResultData{
		Path:      "external-data/result.json",
		SizeBytes: 10,
		RowCount:  1,
		Format:    "json",
		HMAC:      "hmac",
	}

	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusCompleted, resultData.Path, resultData.HMAC, nil).
		Return(errors.New("completed update failed"))

	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err := uc.completeJob(ctx, tracer, message, resultData, time.Now().Add(-time.Second), span, logger)
	if err == nil {
		t.Fatal("expected error when completed status update fails, got nil")
	}
}

// TestExtractExternalData_ParseErrorWithJobIDInHeaders tests parse error path with notification.
func TestExtractExternalData_ParseErrorWithJobIDInHeaders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	// Invalid JSON body
	invalidBody := []byte(`{"invalid": json`)

	headers := map[string]any{
		"jobId": jobID.String(),
	}

	// Expect job status update to failed
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	// Expect failure notification due to parse error
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err := uc.ExtractExternalData(ctx, invalidBody, headers)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestQueryExternalData_WithConnections tests queryExternalData with connections but connection not found for specific database.
func TestQueryExternalData_WithConnections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()

	message := ExtractExternalDataMessage{
		JobID: newTestJobID(),
		MappedFields: map[string]map[string][]string{
			"missing_db": {"table1": {"field1"}},
		},
	}

	// Connections don't include the requested database
	connections := []*model.Connection{
		{
			ConfigName: "other_db",
			Type:       model.TypePostgreSQL,
		},
	}

	result := make(map[string]map[string][]map[string]any)

	err := uc.queryExternalData(ctx, message, connections, result)
	if err == nil {
		t.Fatal("expected error when connection not found for database")
	}
}

// TestEncryptData_NilKeyReturnsError tests that nil derived key returns error.
func TestEncryptData_NilKeyReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// storageEncryptDerivedKey is nil by default
	data := []byte(`{"test": "data"}`)

	_, err := uc.encryptData(data, logger)
	if err == nil {
		t.Error("expected error with nil key")
	}
}

// TestEncryptData_Success tests successful encryption.
func TestEncryptData_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Set valid 32-byte hex keys on the UseCase struct
	uc.SetStorageEncryptDerivedKey([]byte("01234567890123456789012345678901"))

	data := []byte(`{"test": "data"}`)

	result, err := uc.encryptData(data, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) == 0 {
		t.Error("expected non-empty encrypted result")
	}

	// Encrypted result should be different from original
	if string(result) == string(data) {
		t.Error("encrypted data should differ from original")
	}
}

// TestQueryExternalData_MultipleDatabase tests queryExternalData with multiple databases.
func TestQueryExternalData_MultipleDatabase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()

	message := ExtractExternalDataMessage{
		JobID: newTestJobID(),
		MappedFields: map[string]map[string][]string{
			"db1": {"table1": {"field1"}},
			"db2": {"table2": {"field2"}},
		},
	}

	// Neither connection exists - both db1 and db2 will fail with "connection not found"
	// We don't include any connections to ensure the error is hit before
	// any DataSource creation (which would require mocking Decrypt)
	connections := []*model.Connection{}

	result := make(map[string]map[string][]map[string]any)

	// This should fail because neither db1 nor db2 connection is found
	err := uc.queryExternalData(ctx, message, connections, result)
	if err == nil {
		t.Fatal("expected error when database connection not found")
	}

	// Verify error is about connection not found
	if !strings.Contains(err.Error(), "connection not found") {
		t.Errorf("expected 'connection not found' error, got: %v", err)
	}
}

// TestSaveExternalData_EmptyResult tests saveExternalData with empty result.
func TestSaveExternalData_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()
	jobID := newTestJobID()

	// Set valid encryption keys on the UseCase
	uc.SetStorageEncryptDerivedKey([]byte("01234567890123456789012345678901"))

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test"},
	}

	// Empty result
	result := map[string]map[string][]map[string]any{}

	// Mock storage to succeed
	expectedObjectName := constant.ExternalDataKeyPrefix + "/" + jobID.String() + ".json"
	mocks.seaweedFS.EXPECT().
		Put(gomock.Any(), expectedObjectName, gomock.Any()).
		Return(nil)

	resultData, err := uc.saveExternalData(ctx, testTracer(), message, result, nil, logger)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resultData == nil {
		t.Fatal("expected result data, got nil")
	}

	// Should have 0 rows
	if resultData.RowCount != 0 {
		t.Errorf("expected row count 0 for empty result, got %d", resultData.RowCount)
	}
}

// TestQueryDatabase_WithFilters tests queryDatabase with database filters.
func TestQueryDatabase_WithFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	logger := testLogger()

	connections := []*model.Connection{}
	tables := map[string][]string{
		"users": {"id", "name"},
	}

	allFilters := map[string]map[string]map[string]modelJob.FilterCondition{
		"postgres_db": {
			"users": {
				"status": {Equals: []any{"active"}},
			},
		},
	}

	result := make(map[string]map[string][]map[string]any)

	// Should fail because connection not found
	err := uc.queryDatabase(
		ctx,
		"postgres_db",
		tables,
		connections,
		allFilters,
		result,
		logger,
		testTracer(),
	)

	if err == nil {
		t.Fatal("expected error when connection not found")
	}
}

// TestQueryExternalData_NilConnections tests queryExternalData with nil connections.
func TestQueryExternalData_NilConnections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()

	message := ExtractExternalDataMessage{
		JobID: newTestJobID(),
		MappedFields: map[string]map[string][]string{
			"db1": {"table1": {"field1"}},
		},
	}

	result := make(map[string]map[string][]map[string]any)

	// nil connections should result in connection not found error
	err := uc.queryExternalData(ctx, message, nil, result)
	if err == nil {
		t.Fatal("expected error with nil connections")
	}
}

// TestCompleteJob_NotificationFailure_StillReturnsNil verifies the persist-before-notify
// contract: when the DB update succeeds but notification publish fails, completeJob
// returns nil (success) because the DB is the source of truth.
func TestCompleteJob_NotificationFailure_StillReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	tracer := testTracer()
	ctx, span := tracer.Start(ctx, "test.complete_job_notify_fail")
	defer span.End()

	jobID := newTestJobID()
	logger := testLogger()

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test-service"},
	}

	resultData := &JobResultData{
		Path:      "external-data/result.json",
		SizeBytes: 1024,
		RowCount:  50,
		Format:    "json",
		HMAC:      "test-hmac",
	}

	// DB update succeeds
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusCompleted, resultData.Path, resultData.HMAC, nil).
		Return(nil)

	// Notification publish fails
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.completed", gomock.Any()).
		Return(errors.New("connection refused"))

	err := uc.completeJob(ctx, tracer, message, resultData, time.Now().Add(-time.Second), span, logger)
	if err != nil {
		t.Fatalf("expected nil error (DB is source of truth), got: %v", err)
	}
}

// TestCompleteJob_NilResultData verifies that completeJob handles nil resultData
// by failing the job instead of panicking.
func TestCompleteJob_NilResultData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	tracer := testTracer()
	ctx, span := tracer.Start(ctx, "test.complete_job_nil_result")
	defer span.End()

	jobID := newTestJobID()
	logger := testLogger()

	message := ExtractExternalDataMessage{
		JobID:    jobID,
		Metadata: map[string]any{"source": "test-service"},
	}

	// Expect job to be marked as failed due to nil result data
	mocks.jobRepo.EXPECT().
		UpdateStatus(gomock.Any(), jobID, model.JobStatusFailed, "", "", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ model.JobStatus, resultPath, resultHMAC string, metadata map[string]any) error {
			errValue, ok := metadata["error"].(string)
			if !ok {
				t.Fatalf("expected metadata.error as string, got %T", metadata["error"])
			}
			if !strings.Contains(errValue, "result data is nil") {
				t.Fatalf("expected error about nil result data, got %q", errValue)
			}

			return nil
		})

	// Expect failure notification
	mocks.rabbitPublisher.EXPECT().
		Publish(gomock.Any(), "test-exchange", "job.failed", gomock.Any()).
		Return(nil)

	err := uc.completeJob(ctx, tracer, message, nil, time.Now().Add(-time.Second), span, logger)
	if err == nil {
		t.Fatal("expected error for nil resultData, got nil")
	}

	if !strings.Contains(err.Error(), "result data is nil") {
		t.Fatalf("expected nil result data error, got: %v", err)
	}
}

// TestEncryptData_InvalidKeyLength verifies that cipher initialization
// fails with invalid (not 16, 24, or 32 bytes) encryption keys.
func TestEncryptData_InvalidKeyLength(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)
	logger := testLogger()

	// Override with a 31-byte key (invalid for AES)
	uc.SetStorageEncryptDerivedKey([]byte("0123456789012345678901234567890"))

	_, err := uc.encryptData([]byte(`{"test": "data"}`), logger)
	if err == nil {
		t.Fatal("expected error for invalid key length")
	}

	if !strings.Contains(err.Error(), "create AES cipher") {
		t.Fatalf("expected AES cipher error, got: %v", err)
	}
}

// TestSanitizeErrorForNotification verifies that URI patterns are redacted
// in error messages before they are published to notification consumers.
func TestSanitizeErrorForNotification(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redacts mongodb connection string",
			input:    "failed to connect: mongodb://admin:secret@db.internal:27017/prod",
			expected: "failed to connect: [redacted]",
		},
		{
			name:     "redacts amqp connection string",
			input:    "dial: amqp://guest:guest@rabbitmq.svc:5672/",
			expected: "dial: [redacted]",
		},
		{
			name:     "preserves message without URIs",
			input:    "connection not found for database: postgres_db",
			expected: "connection not found for database: postgres_db",
		},
		{
			name:     "redacts multiple URIs",
			input:    "sources: mongodb://u:p@h1 and amqp://u:p@h2",
			expected: "sources: [redacted] and [redacted]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeErrorForNotification(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeErrorForNotification() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestExtractExternalData_NonPendingJobSkipsProcessing verifies the CAS-style
// guard: if the job has moved past PENDING status (e.g. another worker already
// picked it up), processing is skipped instead of re-processing.
func TestExtractExternalData_NonPendingJobSkipsProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	ctx := testContext()
	jobID := newTestJobID()

	validMessage := ExtractExternalDataMessage{
		JobID: jobID,
		MappedFields: map[string]map[string][]string{
			"datasource1": {"table1": {"field1"}},
		},
		Metadata: map[string]any{"source": "test-service"},
	}

	body, err := json.Marshal(validMessage)
	if err != nil {
		t.Fatalf("failed to marshal test message: %v", err)
	}

	// shouldSkipProcessing: job is FAILED (not completed/processing) -> don't skip
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{
			ID:     jobID,
			Status: model.JobStatusFailed,
		}, nil)

	// FindByID for job validation: returns job in FAILED status
	mocks.jobRepo.EXPECT().
		FindByID(gomock.Any(), jobID).
		Return(&model.Job{
			ID:     jobID,
			Status: model.JobStatusFailed,
		}, nil)

	// CAS guard: job.Status != PENDING -> skip (no UpdateStatus, no further processing)
	// No further mock expectations needed -- the function returns nil early.

	err = uc.ExtractExternalData(ctx, body, nil)
	if err != nil {
		t.Fatalf("expected nil (skip), got error: %v", err)
	}
}

// TestValidateExtractExternalDataMessage_EmptyTables verifies that mappedFields
// entries with present database keys but empty table maps are rejected.
// This catches the silent no-op where a db entry exists but has no tables to query.
func TestValidateExtractExternalDataMessage_EmptyTables(t *testing.T) {
	tests := []struct {
		name        string
		message     *ExtractExternalDataMessage
		wantErr     bool
		errContains string
	}{
		{
			name: "db entry with empty tables map is rejected",
			message: &ExtractExternalDataMessage{
				JobID: newTestJobID(),
				MappedFields: map[string]map[string][]string{
					"mydb": {}, // empty inner map
				},
			},
			wantErr:     true,
			errContains: `mappedFields["mydb"] has no tables`,
		},
		{
			name: "one valid db and one empty db is rejected",
			message: &ExtractExternalDataMessage{
				JobID: newTestJobID(),
				MappedFields: map[string]map[string][]string{
					"gooddb":  {"users": {"id", "name"}},
					"emptydb": {},
				},
			},
			wantErr:     true,
			errContains: `mappedFields["emptydb"] has no tables`,
		},
		{
			name: "all dbs with tables is accepted",
			message: &ExtractExternalDataMessage{
				JobID: newTestJobID(),
				MappedFields: map[string]map[string][]string{
					"db1": {"t1": {"col1"}},
					"db2": {"t2": {"col2"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExtractExternalDataMessage(tt.message)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q, got: %v", tt.errContains, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
