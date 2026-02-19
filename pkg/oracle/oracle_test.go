package oracle

import (
	"testing"
)

func TestValidateFieldsInSchemaOracle_CaseInsensitivity(t *testing.T) {
	tests := []struct {
		name            string
		expectedFields  []string
		schema          TableSchema
		wantMissing     []string
		wantCount       int32
	}{
		{
			name:           "all fields present uppercase",
			expectedFields: []string{"ID", "NAME", "EMAIL"},
			schema: TableSchema{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID"},
					{Name: "NAME"},
					{Name: "EMAIL"},
				},
			},
			wantMissing: nil,
			wantCount:   3,
		},
		{
			name:           "some fields missing",
			expectedFields: []string{"ID", "NAME", "PHONE"},
			schema: TableSchema{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID"},
					{Name: "NAME"},
					{Name: "EMAIL"},
				},
			},
			wantMissing: []string{"PHONE"},
			wantCount:   3,
		},
		{
			name:           "all fields missing with empty columns",
			expectedFields: []string{"ID", "NAME"},
			schema: TableSchema{
				TableName: "USERS",
				Columns:   []ColumnInformation{},
			},
			wantMissing: []string{"ID", "NAME"},
			wantCount:   2,
		},
		{
			name:           "empty expected fields",
			expectedFields: []string{},
			schema: TableSchema{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID"},
				},
			},
			wantMissing: nil,
			wantCount:   0,
		},
		{
			name:           "lowercase expected matches uppercase schema",
			expectedFields: []string{"id", "name"},
			schema: TableSchema{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID"},
					{Name: "NAME"},
				},
			},
			wantMissing: nil,
			wantCount:   2,
		},
		{
			name:           "mixed case columns and fields",
			expectedFields: []string{"User_Id", "Account_Name"},
			schema: TableSchema{
				TableName: "ACCOUNTS",
				Columns: []ColumnInformation{
					{Name: "USER_ID"},
					{Name: "ACCOUNT_NAME"},
				},
			},
			wantMissing: nil,
			wantCount:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int32
			missing := ValidateFieldsInSchemaOracle(tt.expectedFields, tt.schema, &count)

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

func TestValidateFieldsInSchemaOracle_CounterIncrement(t *testing.T) {
	// Verify the counter accumulates correctly across multiple calls.
	schema := TableSchema{
		TableName: "USERS",
		Columns: []ColumnInformation{
			{Name: "ID"},
			{Name: "NAME"},
		},
	}

	var count int32

	// First call: 2 fields
	ValidateFieldsInSchemaOracle([]string{"ID", "NAME"}, schema, &count)
	if count != 2 {
		t.Errorf("count after first call = %d, want 2", count)
	}

	// Second call: 3 more fields (counter should accumulate)
	ValidateFieldsInSchemaOracle([]string{"ID", "NAME", "EMAIL"}, schema, &count)
	if count != 5 {
		t.Errorf("count after second call = %d, want 5", count)
	}
}

func TestDefaultSchema_Oracle(t *testing.T) {
	// Oracle uses dynamic schema (empty string), unlike PostgreSQL or SQL Server
	if DefaultSchema != "" {
		t.Errorf("DefaultSchema = %q, want empty string (Oracle uses dynamic schema)", DefaultSchema)
	}
}
