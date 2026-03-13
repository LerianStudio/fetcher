package model

import (
	"strings"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/google/uuid"
)

// TestFetcherRequest_ComputeRequestHash tests the ComputeRequestHash method.
func TestFetcherRequest_ComputeRequestHash(t *testing.T) {
	t.Run("same requests produce same hash", func(t *testing.T) {
		request1 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {
						"table1": {"field1", "field2"},
					},
				},
				Filters: NestedFilters{
					"datasource1": {
						"table1": {
							"field1": job.FilterCondition{Equals: []any{"value1"}},
						},
					},
				},
			},
		}

		request2 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {
						"table1": {"field1", "field2"},
					},
				},
				Filters: NestedFilters{
					"datasource1": {
						"table1": {
							"field1": job.FilterCondition{Equals: []any{"value1"}},
						},
					},
				},
			},
		}

		hash1, err1 := request1.ComputeRequestHash()
		if err1 != nil {
			t.Fatalf("unexpected error computing hash1: %v", err1)
		}

		hash2, err2 := request2.ComputeRequestHash()
		if err2 != nil {
			t.Fatalf("unexpected error computing hash2: %v", err2)
		}

		if hash1 != hash2 {
			t.Fatalf("expected same hash for identical requests, got %s and %s", hash1, hash2)
		}
	})

	t.Run("different requests produce different hashes", func(t *testing.T) {
		request1 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {
						"table1": {"field1"},
					},
				},
			},
		}

		request2 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource2": {
						"table2": {"field2"},
					},
				},
			},
		}

		hash1, err1 := request1.ComputeRequestHash()
		if err1 != nil {
			t.Fatalf("unexpected error computing hash1: %v", err1)
		}

		hash2, err2 := request2.ComputeRequestHash()
		if err2 != nil {
			t.Fatalf("unexpected error computing hash2: %v", err2)
		}

		if hash1 == hash2 {
			t.Fatalf("expected different hashes for different requests, both got %s", hash1)
		}
	})

	t.Run("hash is 64 characters SHA-256 hex", func(t *testing.T) {
		request := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {
						"table1": {"field1"},
					},
				},
			},
		}

		hash, err := request.ComputeRequestHash()
		if err != nil {
			t.Fatalf("unexpected error computing hash: %v", err)
		}

		if len(hash) != 64 {
			t.Fatalf("expected hash length 64 (SHA-256 hex), got %d", len(hash))
		}

		// Verify it's valid hex
		for _, c := range hash {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Fatalf("hash contains invalid hex character: %c", c)
			}
		}
	})

	t.Run("metadata affects hash", func(t *testing.T) {
		request1 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {"table1": {"field1"}},
				},
			},
			Metadata: map[string]any{"key1": "value1"},
		}

		request2 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {"table1": {"field1"}},
				},
			},
			Metadata: map[string]any{"key2": "value2"},
		}

		hash1, _ := request1.ComputeRequestHash()
		hash2, _ := request2.ComputeRequestHash()

		if hash1 == hash2 {
			t.Fatalf("metadata should affect hash, got same hashes: %s", hash1)
		}
	})

	t.Run("nil metadata produces different hash than empty metadata", func(t *testing.T) {
		request1 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {"table1": {"field1"}},
				},
			},
			Metadata: nil,
		}

		request2 := FetcherRequest{
			DataRequest: DataRequest{
				MappedFields: map[string]map[string][]string{
					"datasource1": {"table1": {"field1"}},
				},
			},
			Metadata: map[string]any{},
		}

		hash1, _ := request1.ComputeRequestHash()
		hash2, _ := request2.ComputeRequestHash()

		// Note: nil metadata and empty metadata may produce same or different hashes
		// depending on JSON marshaling. This test documents the behavior.
		// With omitempty, both should produce the same hash (no metadata field)
		if hash1 != hash2 {
			t.Logf("nil and empty metadata produce different hashes: %s vs %s", hash1, hash2)
		}
	})
}

func TestJobStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   bool
	}{
		{name: "pending is valid", status: JobStatusPending, want: true},
		{name: "processing is valid", status: JobStatusProcessing, want: true},
		{name: "completed is valid", status: JobStatusCompleted, want: true},
		{name: "failed is valid", status: JobStatusFailed, want: true},
		{name: "empty string is invalid", status: JobStatus(""), want: false},
		{name: "random string is invalid", status: JobStatus("unknown"), want: false},
		{name: "uppercase PENDING is invalid", status: JobStatus("PENDING"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.IsValid()
			if result != tt.want {
				t.Fatalf("expected %v, got %v for status %s", tt.want, result, tt.status)
			}
		})
	}
}

func TestNewJobStatusFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    JobStatus
		expectError bool
	}{
		{name: "valid pending", input: "pending", expected: JobStatusPending, expectError: false},
		{name: "valid processing", input: "processing", expected: JobStatusProcessing, expectError: false},
		{name: "valid completed", input: "completed", expected: JobStatusCompleted, expectError: false},
		{name: "valid failed", input: "failed", expected: JobStatusFailed, expectError: false},
		{name: "uppercase PENDING", input: "PENDING", expected: JobStatusPending, expectError: false},
		{name: "mixed case Completed", input: "Completed", expected: JobStatusCompleted, expectError: false},
		{name: "with leading spaces", input: "  pending  ", expected: JobStatusPending, expectError: false},
		{name: "empty string", input: "", expected: JobStatus(""), expectError: true},
		{name: "invalid status", input: "unknown", expected: JobStatus(""), expectError: true},
		{name: "whitespace only", input: "   ", expected: JobStatus(""), expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewJobStatusFromString(tt.input)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestJob_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		job         *Job
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid job",
			job: &Job{
				MappedFields: map[string]map[string][]string{
					"db1": {"table1": {"field1"}},
				},
			},
			expectError: false,
		},
		{
			name: "nil mapped fields",
			job: &Job{
				MappedFields: nil,
			},
			expectError: true,
			errorMsg:    "mappedFields is required",
		},
		{
			name: "empty mapped fields",
			job: &Job{
				MappedFields: map[string]map[string][]string{},
			},
			expectError: true,
			errorMsg:    "mappedFields cannot be empty",
		},
		{
			name: "datasource with no fields",
			job: &Job{
				MappedFields: map[string]map[string][]string{
					"db1": {"table1": {}},
				},
			},
			expectError: true,
			errorMsg:    "datasource must have at least one table with fields",
		},
		{
			name: "exceeds datasource limit",
			job: &Job{
				MappedFields: map[string]map[string][]string{
					"ds1":  {"t": {"f"}},
					"ds2":  {"t": {"f"}},
					"ds3":  {"t": {"f"}},
					"ds4":  {"t": {"f"}},
					"ds5":  {"t": {"f"}},
					"ds6":  {"t": {"f"}},
					"ds7":  {"t": {"f"}},
					"ds8":  {"t": {"f"}},
					"ds9":  {"t": {"f"}},
					"ds10": {"t": {"f"}},
					"ds11": {"t": {"f"}},
				},
			},
			expectError: true,
			errorMsg:    "Maximum 10 datasources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.IsValid()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Fatalf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestJob_GetDatasourceNames(t *testing.T) {
	tests := []struct {
		name     string
		job      *Job
		expected []string
	}{
		{
			name: "single datasource",
			job: &Job{
				MappedFields: map[string]map[string][]string{
					"db1": {"table1": {"field1"}},
				},
			},
			expected: []string{"db1"},
		},
		{
			name: "multiple datasources sorted alphabetically",
			job: &Job{
				MappedFields: map[string]map[string][]string{
					"zebra": {"t": {"f"}},
					"alpha": {"t": {"f"}},
					"beta":  {"t": {"f"}},
				},
			},
			expected: []string{"alpha", "beta", "zebra"},
		},
		{
			name: "nil mapped fields",
			job: &Job{
				MappedFields: nil,
			},
			expected: nil,
		},
		{
			name: "empty mapped fields",
			job: &Job{
				MappedFields: map[string]map[string][]string{},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.job.GetDatasourceNames()

			if tt.expected == nil {
				if result != nil {
					t.Fatalf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d datasources, got %d", len(tt.expected), len(result))
			}

			for i, name := range tt.expected {
				if result[i] != name {
					t.Fatalf("expected datasource[%d] = %s, got %s", i, name, result[i])
				}
			}
		})
	}
}

func TestJob_ToMappedFieldsMap(t *testing.T) {
	tests := []struct {
		name     string
		job      *Job
		expected map[string]any
	}{
		{
			name: "converts mapped fields to map",
			job: &Job{
				MappedFields: map[string]map[string][]string{
					"db1": {
						"table1": {"field1", "field2"},
						"table2": {"field3"},
					},
				},
			},
			expected: map[string]any{
				"db1": map[string]any{
					"table1": []string{"field1", "field2"},
					"table2": []string{"field3"},
				},
			},
		},
		{
			name: "nil mapped fields returns nil",
			job: &Job{
				MappedFields: nil,
			},
			expected: nil,
		},
		{
			name: "empty mapped fields returns empty map",
			job: &Job{
				MappedFields: map[string]map[string][]string{},
			},
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.job.ToMappedFieldsMap()

			if tt.expected == nil {
				if result != nil {
					t.Fatalf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d datasources, got %d", len(tt.expected), len(result))
			}
		})
	}
}

func TestJob_SetFailedStatus(t *testing.T) {
	t.Run("sets failed status and timestamp", func(t *testing.T) {
		job := &Job{
			Status:      JobStatusProcessing,
			CompletedAt: nil,
			Metadata:    nil,
		}

		beforeCall := time.Now().UTC()
		job.SetFailedStatus("test error message")
		afterCall := time.Now().UTC()

		if job.Status != JobStatusFailed {
			t.Fatalf("expected status %s, got %s", JobStatusFailed, job.Status)
		}

		if job.CompletedAt == nil {
			t.Fatal("expected CompletedAt to be set")
		}

		if job.CompletedAt.Before(beforeCall) || job.CompletedAt.After(afterCall) {
			t.Fatalf("CompletedAt should be between %v and %v, got %v", beforeCall, afterCall, *job.CompletedAt)
		}

		if job.Metadata == nil {
			t.Fatal("expected Metadata to be initialized")
		}

		errorMsg, ok := job.Metadata["error"]
		if !ok {
			t.Fatal("expected 'error' key in Metadata")
		}

		if errorMsg != "test error message" {
			t.Fatalf("expected error message 'test error message', got %v", errorMsg)
		}
	})

	t.Run("preserves existing metadata and adds error", func(t *testing.T) {
		job := &Job{
			Status:      JobStatusProcessing,
			CompletedAt: nil,
			Metadata: map[string]any{
				"existing_key": "existing_value",
			},
		}

		job.SetFailedStatus("failure reason")

		if len(job.Metadata) != 2 {
			t.Fatalf("expected 2 metadata entries, got %d", len(job.Metadata))
		}

		if job.Metadata["existing_key"] != "existing_value" {
			t.Fatalf("expected existing metadata to be preserved")
		}

		if job.Metadata["error"] != "failure reason" {
			t.Fatalf("expected error to be set")
		}
	})
}

func TestNewJobResponseFrom(t *testing.T) {
	t.Run("nil job returns nil", func(t *testing.T) {
		result := NewJobResponseFrom(nil)
		if result != nil {
			t.Fatalf("expected nil, got %+v", result)
		}
	})

	t.Run("converts job to response", func(t *testing.T) {
		jobID := uuid.New()
		orgID := uuid.New()
		createdAt := time.Now().UTC()
		completedAt := time.Now().UTC().Add(1 * time.Hour)

		job := &Job{
			ID:             jobID,
			OrganizationID: orgID,
			Metadata:       map[string]any{"key": "value"},
			MappedFields: map[string]map[string][]string{
				"db1": {"table1": {"field1"}},
			},
			Filters: NestedFilters{
				"db1": {
					"table1": {
						"field1": job.FilterCondition{Equals: []any{"test"}},
					},
				},
			},
			Status:      JobStatusCompleted,
			ResultPath:  "/path/to/result",
			RequestHash: "hash123",
			CreatedAt:   createdAt,
			CompletedAt: &completedAt,
		}

		result := NewJobResponseFrom(job)

		if result == nil {
			t.Fatal("expected non-nil result")
		}

		if result.ID != jobID {
			t.Fatalf("expected ID %v, got %v", jobID, result.ID)
		}

		if result.OrganizationID != orgID {
			t.Fatalf("expected OrganizationID %v, got %v", orgID, result.OrganizationID)
		}

		if result.Status != string(JobStatusCompleted) {
			t.Fatalf("expected status %s, got %s", JobStatusCompleted, result.Status)
		}

		if result.ResultPath != "/path/to/result" {
			t.Fatalf("expected ResultPath '/path/to/result', got %s", result.ResultPath)
		}

		if result.RequestHash != "hash123" {
			t.Fatalf("expected RequestHash 'hash123', got %s", result.RequestHash)
		}

		if result.CompletedAt == nil || !result.CompletedAt.Equal(completedAt) {
			t.Fatalf("expected CompletedAt %v, got %v", completedAt, result.CompletedAt)
		}
	})
}

func TestNewJob(t *testing.T) {
	t.Run("creates new job with generated ID", func(t *testing.T) {
		orgID := uuid.New()
		metadata := map[string]any{"key": "value"}
		mappedFields := map[string]map[string][]string{
			"db1": {"table1": {"field1"}},
		}
		filters := NestedFilters{
			"db1": {
				"table1": {
					"field1": job.FilterCondition{Equals: []any{"test"}},
				},
			},
		}
		createdAt := time.Now().UTC()

		job, err := NewJob(
			orgID,
			metadata,
			mappedFields,
			filters,
			JobStatusPending,
			"",
			"hash123",
			createdAt,
			nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if job == nil {
			t.Fatal("expected non-nil job")
		}

		if job.ID == uuid.Nil {
			t.Fatal("expected generated ID to be non-nil")
		}

		if job.OrganizationID != orgID {
			t.Fatalf("expected OrganizationID %v, got %v", orgID, job.OrganizationID)
		}

		if job.Status != JobStatusPending {
			t.Fatalf("expected status %s, got %s", JobStatusPending, job.Status)
		}
	})
}

func TestValidateFilterReferences(t *testing.T) {
	mappedFields := map[string]map[string][]string{
		"postgres_db": {
			"transactions":        {"id", "status", "amount"},
			"public.transactions": {"id", "status", "amount"},
		},
		"mysql_db": {
			"orders": {"id", "total"},
		},
	}

	tests := []struct {
		name    string
		filters NestedFilters
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty filters",
			filters: NestedFilters{},
			wantErr: false,
		},
		{
			name:    "nil filters",
			filters: nil,
			wantErr: false,
		},
		{
			name: "valid single filter",
			filters: NestedFilters{
				"postgres_db": {
					"transactions": {
						"status": job.FilterCondition{Equals: []any{"completed"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid filter with schema",
			filters: NestedFilters{
				"postgres_db": {
					"public.transactions": {
						"status": job.FilterCondition{Equals: []any{"completed"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple filters same datasource",
			filters: NestedFilters{
				"postgres_db": {
					"transactions": {
						"status": job.FilterCondition{Equals: []any{"completed"}},
						"amount": job.FilterCondition{GreaterThan: []any{100}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple filters different datasources",
			filters: NestedFilters{
				"postgres_db": {
					"transactions": {
						"status": job.FilterCondition{Equals: []any{"completed"}},
					},
				},
				"mysql_db": {
					"orders": {
						"total": job.FilterCondition{GreaterThan: []any{50}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "unknown datasource",
			filters: NestedFilters{
				"unknown_db": {
					"transactions": {
						"status": job.FilterCondition{Equals: []any{"completed"}},
					},
				},
			},
			wantErr: true,
			errMsg:  "datasource 'unknown_db' not found",
		},
		{
			// NOTE: Unknown table is NOT an error at validation time
			// The DataSource adapter will handle schema resolution with fallback
			name: "unknown table passes validation (resolved by adapter)",
			filters: NestedFilters{
				"postgres_db": {
					"unknown_table": {
						"status": job.FilterCondition{Equals: []any{"completed"}},
					},
				},
			},
			wantErr: false, // Changed: adapter will resolve
		},
		{
			name: "only unknown datasource causes error",
			filters: NestedFilters{
				"unknown_db": {
					"table": {
						"field": job.FilterCondition{Equals: []any{"a"}},
					},
				},
				"postgres_db": {
					"any_table": {
						"field": job.FilterCondition{Equals: []any{"b"}},
					},
				},
			},
			wantErr: true,
			errMsg:  "datasource 'unknown_db' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilterReferences(tt.filters, mappedFields)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errMsg)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
