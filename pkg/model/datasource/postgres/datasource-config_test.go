package postgres

import (
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model/job"
)

func TestContainsDot(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "contains dot",
			s:    "public.users",
			want: true,
		},
		{
			name: "no dot",
			s:    "users",
			want: false,
		},
		{
			name: "empty string",
			s:    "",
			want: false,
		},
		{
			name: "only dot",
			s:    ".",
			want: true,
		},
		{
			name: "multiple dots",
			s:    "db.schema.table",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsDot(tt.s); got != tt.want {
				t.Errorf("containsDot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsSchema(t *testing.T) {
	tests := []struct {
		name    string
		schemas []string
		target  string
		want    bool
	}{
		{
			name:    "schema found",
			schemas: []string{"public", "private"},
			target:  "public",
			want:    true,
		},
		{
			name:    "schema not found",
			schemas: []string{"public", "private"},
			target:  "audit",
			want:    false,
		},
		{
			name:    "empty schemas",
			schemas: []string{},
			target:  "public",
			want:    false,
		},
		{
			name:    "nil schemas",
			schemas: nil,
			target:  "public",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsSchema(tt.schemas, tt.target); got != tt.want {
				t.Errorf("containsSchema() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureDefaultSchemaIncluded(t *testing.T) {
	tests := []struct {
		name               string
		tables             map[string][]string
		schemas            []string
		wantContainsPublic bool
	}{
		{
			name: "unqualified table adds public",
			tables: map[string][]string{
				"users": {"id"},
			},
			schemas:            []string{},
			wantContainsPublic: true,
		},
		{
			name: "qualified table no change",
			tables: map[string][]string{
				"audit.logs": {"id"},
			},
			schemas:            []string{"audit"},
			wantContainsPublic: false,
		},
		{
			name: "mixed adds public",
			tables: map[string][]string{
				"audit.logs": {"id"},
				"users":      {"id"},
			},
			schemas:            []string{"audit"},
			wantContainsPublic: true,
		},
		{
			name: "public already present",
			tables: map[string][]string{
				"users": {"id"},
			},
			schemas:            []string{"public"},
			wantContainsPublic: true,
		},
		{
			name:               "empty tables",
			tables:             map[string][]string{},
			schemas:            []string{"audit"},
			wantContainsPublic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureDefaultSchemaIncluded(tt.tables, tt.schemas)
			hasPublic := containsSchema(got, "public")
			if hasPublic != tt.wantContainsPublic {
				t.Errorf("ensureDefaultSchemaIncluded() contains public = %v, want %v", hasPublic, tt.wantContainsPublic)
			}
		})
	}
}

func TestGetTableFilters_SchemaFallback(t *testing.T) {
	tests := []struct {
		name            string
		databaseFilters map[string]map[string]job.FilterCondition
		tableName       string
		wantFound       bool
		wantFieldCount  int
	}{
		{
			name: "exact match with schema",
			databaseFilters: map[string]map[string]job.FilterCondition{
				"public.transactions": {"status": {Equals: []any{"completed"}}},
			},
			tableName:      "public.transactions",
			wantFound:      true,
			wantFieldCount: 1,
		},
		{
			name: "exact match without schema",
			databaseFilters: map[string]map[string]job.FilterCondition{
				"transactions": {"status": {Equals: []any{"completed"}}},
			},
			tableName:      "transactions",
			wantFound:      true,
			wantFieldCount: 1,
		},
		{
			name: "fallback: filter without schema, table with schema",
			databaseFilters: map[string]map[string]job.FilterCondition{
				"transactions": {"status": {Equals: []any{"completed"}}},
			},
			tableName:      "public.transactions", // MappedFields has schema
			wantFound:      true,                  // Should find via fallback
			wantFieldCount: 1,
		},
		{
			name: "fallback: filter with schema, table without schema",
			databaseFilters: map[string]map[string]job.FilterCondition{
				"public.transactions": {"status": {Equals: []any{"completed"}}},
			},
			tableName:      "transactions", // MappedFields without schema
			wantFound:      true,           // Should find via default schema fallback
			wantFieldCount: 1,
		},
		{
			name: "no match - different schema",
			databaseFilters: map[string]map[string]job.FilterCondition{
				"audit.transactions": {"status": {Equals: []any{"completed"}}},
			},
			tableName: "public.transactions",
			wantFound: false,
		},
		{
			name:            "nil filters",
			databaseFilters: nil,
			tableName:       "transactions",
			wantFound:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTableFilters(tt.databaseFilters, tt.tableName)

			if tt.wantFound {
				if result == nil {
					t.Fatal("expected filters to be found, got nil")
				}
				if len(result) != tt.wantFieldCount {
					t.Fatalf("expected %d fields, got %d", tt.wantFieldCount, len(result))
				}
			} else {
				if result != nil {
					t.Fatalf("expected nil, got %+v", result)
				}
			}
		})
	}
}
