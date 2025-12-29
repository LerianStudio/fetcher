package command

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	jobRepo "github.com/LerianStudio/fetcher/pkg/mongodb/job"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

// testContext creates a context with logger and tracer for testing.
func testContext() context.Context {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: "test-request-id",
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}

	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}

// newValidFetcherRequest creates a valid FetcherRequest for testing.
func newValidFetcherRequest() model.FetcherRequest {
	return model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: map[string]map[string][]string{
				"datasource1": {
					"table1": {"field1", "field2"},
				},
			},
		},
		Metadata: map[string]any{"key": "value"},
	}
}

// TestCreateFetcherJob_Execute_ValidationError tests that invalid requests return validation errors.
func TestCreateFetcherJob_Execute_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

	tests := []struct {
		name    string
		request model.FetcherRequest
		wantErr string
	}{
		{
			name: "nil mappedFields",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: nil,
				},
			},
			wantErr: "mappedFields is required",
		},
		{
			name: "empty mappedFields",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{},
				},
			},
			wantErr: "mappedFields cannot be empty",
		},
		{
			name: "datasource with empty fields",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						"datasource1": {
							"table1": {},
						},
					},
				},
			},
			wantErr: "datasource must have at least one table with fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext()
			orgID := uuid.New()

			result, err := svc.Execute(ctx, orgID, tt.request)

			if result != nil {
				t.Fatalf("expected nil result for invalid request, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error for invalid request, got nil")
			}

			var validationErr pkg.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}

			if validationErr.Message != tt.wantErr {
				t.Fatalf("expected error message %q, got %q", tt.wantErr, validationErr.Message)
			}
		})
	}
}

// TestCreateFetcherJob_Execute_DuplicateWithinWindow tests that duplicate requests return existing job.
func TestCreateFetcherJob_Execute_DuplicateWithinWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	existingJobID := uuid.New()
	existingJob := &model.Job{
		ID:             existingJobID,
		OrganizationID: orgID,
		Status:         model.JobStatusPending,
		CreatedAt:      time.Now().UTC().Add(-2 * time.Minute),
	}

	// Mock: find existing job within window
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(existingJob, nil)

	result, err := svc.Execute(ctx, orgID, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.IsDuplicate {
		t.Fatal("expected IsDuplicate to be true")
	}

	if result.IsNewCreated {
		t.Fatal("expected IsNewCreated to be false")
	}

	if result.Job.ID != existingJobID {
		t.Fatalf("expected job ID %s, got %s", existingJobID, result.Job.ID)
	}
}

// TestCreateFetcherJob_Execute_NoConnectionsFound tests that when no connections are found
// for any of the requested datasources, a validation error is returned.
func TestCreateFetcherJob_Execute_NoConnectionsFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: no connections found at all
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return([]*model.Connection{}, nil)

	result, err := svc.Execute(ctx, orgID, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for no connections found, got nil")
	}

	var validationErr pkg.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	// When NO connections are found at all, production returns ErrMissingDataSource
	if validationErr.Code != constant.ErrMissingDataSource.Error() {
		t.Fatalf("expected error code %s, got %s", constant.ErrMissingDataSource.Error(), validationErr.Code)
	}
}

// TestCreateFetcherJob_Execute_TooManyDatasources tests that too many datasources returns error.
func TestCreateFetcherJob_Execute_TooManyDatasources(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

	ctx := testContext()
	orgID := uuid.New()

	// Create request with 11 datasources (exceeds MaxDatasourcesPerJob = 10)
	mappedFields := make(map[string]map[string][]string)
	for i := 0; i < 11; i++ {
		dsName := fmt.Sprintf("datasource_%d", i)
		mappedFields[dsName] = map[string][]string{
			"table1": {"field1"},
		}
	}

	request := model.FetcherRequest{
		DataRequest: model.DataRequest{
			MappedFields: mappedFields,
		},
	}

	// No mock expectations needed - validation fails before any repository calls

	result, err := svc.Execute(ctx, orgID, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for too many datasources, got nil")
	}

	var validationErr pkg.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	if validationErr.Code != constant.ErrInvalidDataRequest.Error() {
		t.Fatalf("expected error code %s, got %s", constant.ErrInvalidDataRequest.Error(), validationErr.Code)
	}
}

// TestCreateFetcherJob_Execute_FindByRequestHashError tests internal error during hash lookup.
func TestCreateFetcherJob_Execute_FindByRequestHashError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	dbError := errors.New("database connection failed")

	// Mock: database error during duplicate check
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestCreateFetcherJob_Execute_FindByConfigNamesError tests internal error during connection lookup.
func TestCreateFetcherJob_Execute_FindByConfigNamesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

	ctx := testContext()
	orgID := uuid.New()
	request := newValidFetcherRequest()

	dbError := errors.New("database connection failed")

	// Mock: no duplicate found
	mockJobRepo.EXPECT().
		FindByRequestHashWithinWindow(gomock.Any(), orgID, gomock.Any(), DeduplicationWindowMinutes).
		Return(nil, nil)

	// Mock: database error during connection lookup
	mockConnRepo.EXPECT().
		FindByConfigNames(gomock.Any(), orgID, []string{"datasource1"}).
		Return(nil, dbError)

	result, err := svc.Execute(ctx, orgID, request)

	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var internalErr pkg.InternalServerError
	if !errors.As(err, &internalErr) {
		t.Fatalf("expected InternalServerError, got %T: %v", err, err)
	}
}

// TestCreateFetcherJob_Constants verifies the service constants are as expected.
func TestCreateFetcherJob_Constants(t *testing.T) {
	if DeduplicationWindowMinutes != 5 {
		t.Fatalf("expected DeduplicationWindowMinutes to be 5, got %d", DeduplicationWindowMinutes)
	}

	if model.MaxDatasourcesPerJob != 10 {
		t.Fatalf("expected MaxDatasourcesPerJob to be 10, got %d", model.MaxDatasourcesPerJob)
	}

	if ExtractExternalDataQueue != "extract-external-data-queue" {
		t.Fatalf("expected ExtractExternalDataQueue to be 'extract-external-data-queue', got %s", ExtractExternalDataQueue)
	}
}

// TestTransformFiltersForWorker tests the filter transformation logic.
func TestTransformFiltersForWorker(t *testing.T) {
	svc := &CreateFetcherJob{}

	tests := []struct {
		name         string
		filters      []model.Filter
		mappedFields map[string]map[string][]string
		wantNil      bool
		checkFunc    func(t *testing.T, result map[string]map[string]map[string]job.FilterCondition)
	}{
		{
			name:         "empty filters",
			filters:      []model.Filter{},
			mappedFields: map[string]map[string][]string{"ds": {"table": {"field"}}},
			wantNil:      true,
		},
		{
			name:         "empty mappedFields",
			filters:      []model.Filter{{Field: "ds.table.field", Operator: "eq", Value: []any{"val"}}},
			mappedFields: map[string]map[string][]string{},
			wantNil:      true,
		},
		{
			name: "single filter applied to specific table",
			filters: []model.Filter{
				{Field: "postgres_db.transactions.status", Operator: "eq", Value: []any{"completed"}},
			},
			mappedFields: map[string]map[string][]string{
				"postgres_db": {
					"transactions": {"id", "status"},
					"accounts":     {"id", "name"},
				},
			},
			checkFunc: func(t *testing.T, result map[string]map[string]map[string]job.FilterCondition) {
				// Filter should be on transactions table
				if _, ok := result["postgres_db"]["transactions"]["status"]; !ok {
					t.Fatal("expected filter on postgres_db.transactions.status")
				}
				if len(result["postgres_db"]["transactions"]["status"].Equals) != 1 {
					t.Fatalf("expected 1 Equals value, got %d", len(result["postgres_db"]["transactions"]["status"].Equals))
				}
				// accounts table should NOT have this filter
				if _, ok := result["postgres_db"]["accounts"]; ok {
					t.Fatal("accounts table should not have any filters")
				}
			},
		},
		{
			name: "filter with schema-qualified table",
			filters: []model.Filter{
				{Field: "postgres_db.public.transactions.status", Operator: "in", Value: []any{"completed", "pending"}},
			},
			mappedFields: map[string]map[string][]string{
				"postgres_db": {
					"public.transactions": {"id", "status"},
				},
			},
			checkFunc: func(t *testing.T, result map[string]map[string]map[string]job.FilterCondition) {
				if _, ok := result["postgres_db"]["public.transactions"]["status"]; !ok {
					t.Fatal("expected filter on postgres_db.public.transactions.status")
				}
				if len(result["postgres_db"]["public.transactions"]["status"].In) != 2 {
					t.Fatalf("expected 2 In values, got %d", len(result["postgres_db"]["public.transactions"]["status"].In))
				}
			},
		},
		{
			name: "multiple filters on different datasources",
			filters: []model.Filter{
				{Field: "postgres_db.transactions.status", Operator: "eq", Value: []any{"completed"}},
				{Field: "mysql_db.orders.total", Operator: "gt", Value: []any{100}},
			},
			mappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id", "status"}},
				"mysql_db":    {"orders": {"id", "total"}},
			},
			checkFunc: func(t *testing.T, result map[string]map[string]map[string]job.FilterCondition) {
				// Check postgres filter
				if _, ok := result["postgres_db"]["transactions"]["status"]; !ok {
					t.Fatal("expected filter on postgres_db.transactions.status")
				}
				// Check mysql filter
				if _, ok := result["mysql_db"]["orders"]["total"]; !ok {
					t.Fatal("expected filter on mysql_db.orders.total")
				}
				if len(result["mysql_db"]["orders"]["total"].GreaterThan) != 1 {
					t.Fatal("expected GreaterThan filter on mysql_db.orders.total")
				}
			},
		},
		{
			name: "all operators",
			filters: []model.Filter{
				{Field: "ds.tbl.f1", Operator: "eq", Value: []any{"a"}},
				{Field: "ds.tbl.f2", Operator: "gt", Value: []any{1}},
				{Field: "ds.tbl.f3", Operator: "gte", Value: []any{2}},
				{Field: "ds.tbl.f4", Operator: "lt", Value: []any{3}},
				{Field: "ds.tbl.f5", Operator: "lte", Value: []any{4}},
				{Field: "ds.tbl.f6", Operator: "ne", Value: []any{"b"}},
				{Field: "ds.tbl.f7", Operator: "in", Value: []any{"x", "y"}},
				{Field: "ds.tbl.f8", Operator: "nin", Value: []any{"z"}},
				{Field: "ds.tbl.f9", Operator: "like", Value: []any{"%test%"}},
			},
			mappedFields: map[string]map[string][]string{
				"ds": {"tbl": {"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9"}},
			},
			checkFunc: func(t *testing.T, result map[string]map[string]map[string]job.FilterCondition) {
				tbl := result["ds"]["tbl"]
				if len(tbl["f1"].Equals) != 1 {
					t.Error("f1 should have Equals")
				}
				if len(tbl["f2"].GreaterThan) != 1 {
					t.Error("f2 should have GreaterThan")
				}
				if len(tbl["f3"].GreaterOrEqual) != 1 {
					t.Error("f3 should have GreaterOrEqual")
				}
				if len(tbl["f4"].LessThan) != 1 {
					t.Error("f4 should have LessThan")
				}
				if len(tbl["f5"].LessOrEqual) != 1 {
					t.Error("f5 should have LessOrEqual")
				}
				if len(tbl["f6"].NotEquals) != 1 {
					t.Error("f6 should have NotEquals")
				}
				if len(tbl["f7"].In) != 2 {
					t.Error("f7 should have 2 In values")
				}
				if len(tbl["f8"].NotIn) != 1 {
					t.Error("f8 should have NotIn")
				}
				if len(tbl["f9"].Like) != 1 {
					t.Error("f9 should have Like")
				}
			},
		},
		{
			name: "filter with unknown datasource is skipped",
			filters: []model.Filter{
				{Field: "unknown_db.table.field", Operator: "eq", Value: []any{"val"}},
			},
			mappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id"}},
			},
			wantNil: true, // No valid filters, result should be nil
		},
		{
			name: "filter with unknown table is skipped",
			filters: []model.Filter{
				{Field: "postgres_db.unknown_table.field", Operator: "eq", Value: []any{"val"}},
			},
			mappedFields: map[string]map[string][]string{
				"postgres_db": {"transactions": {"id"}},
			},
			wantNil: true, // No valid filters, result should be nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.transformFiltersForWorker(tt.filters, tt.mappedFields)

			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

// TestCreateFetcherJob_Execute_InvalidFilterReferences tests that invalid filter references return validation error.
// NOTE: Table name validation is intentionally NOT done at this stage - the DataSource adapter handles
// schema resolution with fallback logic (e.g., trying "public.table" if "table" not found).
// Therefore, only datasource names and filter format are validated here.
func TestCreateFetcherJob_Execute_InvalidFilterReferences(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConnRepo := connRepo.NewMockRepository(ctrl)
	mockJobRepo := jobRepo.NewMockRepository(ctrl)

	svc := NewCreateFetcherJob(mockConnRepo, mockJobRepo, nil, nil)

	tests := []struct {
		name    string
		request model.FetcherRequest
		wantErr string
	}{
		{
			name: "filter references unknown datasource",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						"postgres_db": {"transactions": {"id", "status"}},
					},
					Filters: []model.FilterRequest{
						{Field: "unknown_db.transactions.status", Operator: "eq", Value: []any{"completed"}},
					},
				},
			},
			wantErr: "datasource 'unknown_db' not found",
		},
		{
			name: "filter with invalid format",
			request: model.FetcherRequest{
				DataRequest: model.DataRequest{
					MappedFields: map[string]map[string][]string{
						"postgres_db": {"transactions": {"id", "status"}},
					},
					Filters: []model.FilterRequest{
						{Field: "status", Operator: "eq", Value: []any{"completed"}},
					},
				},
			},
			wantErr: "invalid filter field format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext()
			orgID := uuid.New()

			result, err := svc.Execute(ctx, orgID, tt.request)

			if result != nil {
				t.Fatalf("expected nil result for invalid filter, got %+v", result)
			}

			if err == nil {
				t.Fatal("expected error for invalid filter, got nil")
			}

			var validationErr pkg.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}

			if !strings.Contains(validationErr.Message, tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, validationErr.Message)
			}
		})
	}
}
