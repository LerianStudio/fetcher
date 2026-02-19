package mysql

import (
	"testing"
)

func TestValidateFieldsInSchemaMySQL_CaseInsensitivity(t *testing.T) {
	tests := []struct {
		name            string
		expectedFields  []string
		schema          TableSchema
		wantMissing     []string
		wantCount       int32
	}{
		{
			name:           "all fields present exact case",
			expectedFields: []string{"id", "name", "email"},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
					{Name: "name"},
					{Name: "email"},
				},
			},
			wantMissing: nil,
			wantCount:   3,
		},
		{
			name:           "some fields missing",
			expectedFields: []string{"id", "name", "phone"},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
					{Name: "name"},
					{Name: "email"},
				},
			},
			wantMissing: []string{"phone"},
			wantCount:   3,
		},
		{
			name:           "all fields missing with empty columns",
			expectedFields: []string{"id", "name"},
			schema: TableSchema{
				TableName: "users",
				Columns:   []ColumnInformation{},
			},
			wantMissing: []string{"id", "name"},
			wantCount:   2,
		},
		{
			name:           "empty expected fields",
			expectedFields: []string{},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
				},
			},
			wantMissing: nil,
			wantCount:   0,
		},
		{
			name:           "uppercase expected matches lowercase schema",
			expectedFields: []string{"ID", "NAME"},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
					{Name: "name"},
				},
			},
			wantMissing: nil,
			wantCount:   2,
		},
		{
			name:           "mixed case columns and fields",
			expectedFields: []string{"user_id", "UserName"},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "USER_ID"},
					{Name: "username"},
				},
			},
			wantMissing: nil,
			wantCount:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int32
			missing := ValidateFieldsInSchemaMySQL(tt.expectedFields, tt.schema, &count)

			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}

			if len(missing) != len(tt.wantMissing) {
				t.Errorf("missing = %v (len %d), want %v (len %d)", missing, len(missing), tt.wantMissing, len(tt.wantMissing))
				return
			}

			for i, m := range missing {
				if m != tt.wantMissing[i] {
					t.Errorf("missing[%d] = %q, want %q", i, m, tt.wantMissing[i])
				}
			}
		})
	}
}

func TestValidateFieldsInSchemaMySQL_CounterIncrement(t *testing.T) {
	// Verify the counter accumulates correctly across multiple calls.
	schema := TableSchema{
		TableName: "users",
		Columns: []ColumnInformation{
			{Name: "id"},
			{Name: "name"},
		},
	}

	var count int32

	// First call: 2 fields
	ValidateFieldsInSchemaMySQL([]string{"id", "name"}, schema, &count)
	if count != 2 {
		t.Errorf("count after first call = %d, want 2", count)
	}

	// Second call: 3 more fields (counter should accumulate)
	ValidateFieldsInSchemaMySQL([]string{"id", "name", "email"}, schema, &count)
	if count != 5 {
		t.Errorf("count after second call = %d, want 5", count)
	}
}
