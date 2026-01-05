package datasource

import (
	"reflect"
	"sort"
	"testing"
)

func TestGetUniqueSchemas(t *testing.T) {
	tests := []struct {
		name   string
		tables map[string][]string
		want   []string
	}{
		{
			name:   "nil tables",
			tables: nil,
			want:   nil,
		},
		{
			name:   "empty tables",
			tables: map[string][]string{},
			want:   nil,
		},
		{
			name: "single schema qualified table",
			tables: map[string][]string{
				"public.users": {"id", "name"},
			},
			want: []string{"public"},
		},
		{
			name: "multiple tables same schema",
			tables: map[string][]string{
				"public.users":  {"id", "name"},
				"public.orders": {"id", "total"},
			},
			want: []string{"public"},
		},
		{
			name: "multiple schemas",
			tables: map[string][]string{
				"public.users":    {"id"},
				"private.secrets": {"key"},
				"audit.logs":      {"timestamp"},
			},
			want: []string{"audit", "private", "public"},
		},
		{
			name: "unqualified table names",
			tables: map[string][]string{
				"users":  {"id"},
				"orders": {"id"},
			},
			want: nil,
		},
		{
			name: "mixed qualified and unqualified",
			tables: map[string][]string{
				"public.users": {"id"},
				"orders":       {"id"},
			},
			want: []string{"public"},
		},
		{
			name: "empty table name",
			tables: map[string][]string{
				"": {"id"},
			},
			want: nil,
		},
		{
			name: "whitespace table name",
			tables: map[string][]string{
				"   ": {"id"},
			},
			want: nil,
		},
		{
			name: "dot only table name",
			tables: map[string][]string{
				".": {"id"},
			},
			want: nil,
		},
		{
			name: "schema.table with empty schema",
			tables: map[string][]string{
				".users": {"id"},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetUniqueSchemas(tt.tables)

			// Sort both for comparison
			if got != nil {
				sort.Strings(got)
			}
			if tt.want != nil {
				sort.Strings(tt.want)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUniqueSchemas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitSchemaTable(t *testing.T) {
	tests := []struct {
		name       string
		qualified  string
		wantSchema string
		wantTable  string
	}{
		{
			name:       "valid schema.table",
			qualified:  "public.users",
			wantSchema: "public",
			wantTable:  "users",
		},
		{
			name:       "unqualified table",
			qualified:  "users",
			wantSchema: "",
			wantTable:  "users",
		},
		{
			name:       "empty string",
			qualified:  "",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "whitespace only",
			qualified:  "   ",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "schema with whitespace",
			qualified:  "  public  .  users  ",
			wantSchema: "public",
			wantTable:  "users",
		},
		{
			name:       "dot only",
			qualified:  ".",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "empty schema with table",
			qualified:  ".users",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "schema with empty table",
			qualified:  "public.",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "multiple dots",
			qualified:  "db.schema.table",
			wantSchema: "db",
			wantTable:  "schema.table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSchema, gotTable := SplitSchemaTable(tt.qualified)
			if gotSchema != tt.wantSchema {
				t.Errorf("SplitSchemaTable() schema = %v, want %v", gotSchema, tt.wantSchema)
			}
			if gotTable != tt.wantTable {
				t.Errorf("SplitSchemaTable() table = %v, want %v", gotTable, tt.wantTable)
			}
		})
	}
}
